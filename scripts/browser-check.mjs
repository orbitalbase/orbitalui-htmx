// Browser check for the interactive components: drives ModalShell,
// ConfirmButton, Drawer and EntityPicker through orbital-ui.js, Alpine and
// htmx in headless Chrome with real pointer and keyboard input. The full suite
// runs against the standard Alpine build and against Alpine's CSP build served
// under `script-src 'self'` (any CSP violation or page error fails the run); a
// third, shorter pass loads a stand-in host runtime (window.LedgerUI) before
// orbital-ui.js and checks the Alpine adapters delegate to it.
//
//   node scripts/browser-check.mjs
//
// Needs Node 22+ (global WebSocket), Go, and Chrome or Chromium (CHROME_BIN
// overrides the executable). Exits non-zero on the first failed check.
import {spawn} from 'node:child_process';
import {mkdtemp, readFile, rm} from 'node:fs/promises';
import {existsSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {createInterface} from 'node:readline';

const sleep = ms => new Promise(r => setTimeout(r, ms));
async function eventually(fn, what, timeout = 5000) {
    const end = Date.now() + timeout;
    let last;
    while (Date.now() < end) {
        try { const v = await fn(); if (v) return v; } catch (e) { last = e; }
        await sleep(40);
    }
    throw new Error(`timed out waiting for ${what}${last ? `: ${last.message}` : ''}`);
}

function chromePath() {
    if (process.env.CHROME_BIN) return process.env.CHROME_BIN;
    for (const p of ['/Applications/Google Chrome.app/Contents/MacOS/Google Chrome', '/usr/bin/google-chrome', '/usr/bin/google-chrome-stable', '/usr/bin/chromium', '/usr/bin/chromium-browser']) {
        if (existsSync(p)) return p;
    }
    return 'chromium';
}

async function startServer() {
    // Build first and run the binary itself: killing `go run` would leave its
    // child server alive.
    const dir = await mkdtemp(join(tmpdir(), 'oui-check-bin-'));
    const bin = join(dir, 'browsercheck');
    await new Promise((resolve, reject) => {
        const build = spawn('go', ['build', '-o', bin, './internal/browsercheck'], {stdio: 'inherit'});
        build.on('error', reject);
        build.on('exit', code => code === 0 ? resolve() : reject(new Error(`go build exited ${code}`)));
    });
    const server = spawn(bin, ['-addr', '127.0.0.1:0'], {stdio: ['ignore', 'pipe', 'inherit']});
    const line = await new Promise((resolve, reject) => {
        server.on('error', reject);
        server.on('exit', code => reject(new Error(`browsercheck server exited (${code})`)));
        createInterface({input: server.stdout}).once('line', resolve);
    });
    const stop = async () => { server.kill(); await rm(dir, {recursive: true, force: true}).catch(() => {}); };
    return {stop, base: line.trim()};
}

async function startChrome() {
    const profile = await mkdtemp(join(tmpdir(), 'oui-browser-'));
    const chrome = spawn(chromePath(), ['--headless=new', '--no-first-run', '--no-default-browser-check', '--disable-gpu',
        '--remote-debugging-port=0', `--user-data-dir=${profile}`, '--window-size=1280,900', 'about:blank'], {stdio: 'ignore'});
    let port;
    try {
        port = await eventually(async () => (await readFile(join(profile, 'DevToolsActivePort'), 'utf8')).split('\n')[0], 'Chrome DevTools port', 15000);
    } catch (e) {
        // A cold CI runner occasionally never writes the port file: clean up so
        // the caller can retry with a fresh process and profile.
        chrome.kill();
        await rm(profile, {recursive: true, force: true}).catch(() => {});
        throw e;
    }
    const targets = await eventually(async () => (await fetch(`http://127.0.0.1:${port}/json/list`)).json(), 'Chrome targets');
    const page = targets.find(t => t.type === 'page');
    const ws = new WebSocket(page.webSocketDebuggerUrl);
    await new Promise((resolve, reject) => { ws.onopen = resolve; ws.onerror = reject; setTimeout(() => reject(new Error('DevTools socket did not open')), 10000); });
    let id = 0;
    const pending = new Map();
    const errors = [];
    ws.onmessage = ev => {
        const msg = JSON.parse(ev.data);
        if (msg.id && pending.has(msg.id)) {
            const {resolve, reject} = pending.get(msg.id);
            pending.delete(msg.id);
            msg.error ? reject(new Error(msg.error.message)) : resolve(msg.result);
        } else if (msg.method === 'Runtime.exceptionThrown') {
            errors.push(msg.params.exceptionDetails.exception?.description || msg.params.exceptionDetails.text);
        } else if (msg.method === 'Log.entryAdded' && msg.params.entry.level === 'error') {
            const {source, text, url = ''} = msg.params.entry;
            // The 422 is the validation response the checks provoke on purpose;
            // the favicon is not part of the page under test.
            if (source === 'network' && (/\b422\b/.test(text) || url.endsWith('/favicon.ico'))) return;
            errors.push(`${source}: ${text} ${url}`);
        } else if (msg.method === 'Runtime.consoleAPICalled' && msg.params.type === 'error') {
            errors.push(msg.params.args.map(a => a.value ?? a.description).join(' '));
        }
    };
    const send = (method, params = {}) => new Promise((resolve, reject) => {
        const n = ++id;
        pending.set(n, {resolve, reject});
        ws.send(JSON.stringify({id: n, method, params}));
    });
    await send('Runtime.enable');
    await send('Log.enable');
    await send('Page.enable');
    if (process.env.OUI_CPU_THROTTLE) await send('Emulation.setCPUThrottlingRate', {rate: +process.env.OUI_CPU_THROTTLE});
    const resize = (width, height = 900) => send('Emulation.setDeviceMetricsOverride', {width, height, deviceScaleFactor: 1, mobile: width < 600});
    await resize(1280);
    const close = async () => { try { ws.close(); } catch {} chrome.kill(); await rm(profile, {recursive: true, force: true}).catch(() => {}); };
    return {send, errors, close, resize};
}


const {stop, base} = await startServer();
const results = [];

// Starts Chrome, retrying once: a cold runner occasionally never writes the port file.
async function launchChrome() {
    try {
        return await startChrome().catch(e => {
            console.error(`Chrome did not start (${e.message}); retrying once`);
            return startChrome();
        });
    } catch (e) {
        console.error(`could not start Chrome (${chromePath()}): ${e.message}`);
        process.exit(1);
    }
}

// Page evaluation, real pointer and keyboard input, and result recording for
// one browser; `name` prefixes every check label.
function drive(b, name) {
    const js = async expr => {
        const r = await b.send('Runtime.evaluate', {expression: expr, returnByValue: true, awaitPromise: true});
        if (r.exceptionDetails) throw new Error(r.exceptionDetails.exception?.description || r.exceptionDetails.text);
        return r.result.value;
    };
    const center = sel => js(`(() => { const el = document.querySelector(${JSON.stringify(sel)}); el.scrollIntoView({block: 'center', inline: 'nearest'}); const r = el.getBoundingClientRect(); return [r.left + r.width / 2, r.top + r.height / 2]; })()`);
    const clickAt = async (x, y) => {
        for (const type of ['mousePressed', 'mouseReleased']) await b.send('Input.dispatchMouseEvent', {type, x, y, button: 'left', clickCount: 1});
    };
    const click = async sel => { const [x, y] = await center(sel); await clickAt(x, y); };
    const key = async (k, code = k, keyCode = 0) => {
        for (const type of ['keyDown', 'keyUp']) await b.send('Input.dispatchKeyEvent', {type, key: k, code, windowsVirtualKeyCode: keyCode});
    };
    const type = text => b.send('Input.insertText', {text});
    const check = async (label, fn) => {
        const full = `[${name}] ${label}`;
        try { await fn(); results.push(['ok', full]); console.log(`ok   ${full}`); }
        catch (e) { results.push(['FAIL', full]); console.log(`FAIL ${full}\n     ${e.message}`); }
    };
    const expect = (cond, msg) => { if (!cond) throw new Error(msg); };
    return {js, center, clickAt, click, key, type, check, expect};
}

// Runs the checks for one page variant in a fresh Chrome. `full` selects the
// whole suite; the host variant runs only the delegation checks.
async function runVariant(name, path, {full = true, host = false} = {}) {
    const b = await launchChrome();
    const {js, center, clickAt, click, key, type, check, expect} = drive(b, name);
    const focused = () => js('document.activeElement && (document.activeElement.id || document.activeElement.className)');
    // Wait until htmx has finished swapping and settling, so freshly swapped
    // markup is processed before the next interaction.
    const htmxIdle = () => eventually(() => js('!document.querySelector(".htmx-request, .htmx-swapping, .htmx-settling, .htmx-added")'), 'htmx to settle');
    // Wait for the drawer's slide-in to finish so pointer targets stop moving.
    const drawerSettled = () => eventually(() => js('["matrix(1,0,0,1,0,0)", "none"].includes(getComputedStyle(document.querySelector("#drw .drawer-panel")).transform.replace(/\\s/g, ""))'), 'drawer to settle');

    try {
        await b.send('Page.navigate', {url: base + path});
        await eventually(() => js('!!(window.Alpine && window.htmx && document.querySelector(".entity-picker")?._x_dataStack)'), 'Alpine, htmx and the picker to start');

        await check('runtime installs OrbitalUI and the LedgerUI alias', async () => {
            expect(await js('typeof OrbitalUI.openDialog === "function" && window.LedgerUI === window.OrbitalUI'), 'runtime not installed');
            if (host) expect(await js('window.LedgerUI.host === true && window.LedgerUI.dialog.toString().includes("calls.dialog")'), 'the host runtime was replaced');
            else expect(await js('window.LedgerUI.host === undefined'), 'unexpected host runtime');
        });

        if (path.includes('csp=1')) {
            await check('the page is served under script-src \'self\' and blocks inline scripts', async () => {
                expect((await js('fetch(location.href).then(r => r.headers.get("content-security-policy"))')) === "script-src 'self'", 'CSP header missing');
                await js('(() => { const s = document.createElement("script"); s.textContent = "window.__inlineRan = true"; document.head.appendChild(s); })()');
                expect(!(await js('window.__inlineRan')), 'an inline script ran despite the policy');
                b.errors.length = 0; // the blocked inline script is reported as a violation on purpose
            });
        }

        await check('the Alpine factories ouiDialog/ouiDrawer/ouiPicker drive the markup', async () => {
            expect(await js('document.getElementById("dlg")._x_dataStack && typeof document.getElementById("dlg")._x_dataStack[0].backdrop === "function"'), 'dialog scope missing backdrop');
            expect(await js('typeof document.getElementById("drw")._x_dataStack[0].escape === "function"'), 'drawer scope missing escape');
            expect(await js('document.querySelector(".entity-picker")._x_dataStack[0].cfg.id === "customer"'), 'picker did not read data-oui-picker');
        });

        await check('a dismissible Alert hides on its close button', async () => {
            await click('.ui-alert__close');
            await eventually(() => js('getComputedStyle(document.querySelector(".ui-alert")).display === "none"'), 'alert hidden');
        });

        await check('dialog opens modally and focuses its first field', async () => {
            await click('#open-dialog');
            await eventually(() => js('document.getElementById("dlg").matches(":modal")'), 'modal dialog');
            if (!host) expect(await focused() === 'dlg-name', `focus is on ${await focused()}`);
            if (host) expect(await js('LedgerUI.calls.openDialog') === 1, 'the host openDialog did not run exactly once');
        });

        if (full) {
            await check('a 422 response swaps inline errors and keeps the dialog open', async () => {
                await click('#dlg-save');
                await eventually(() => js('!!document.querySelector("#dlg .field-error")'), 'inline error');
                await htmxIdle();
                expect(await js('document.getElementById("dlg").open'), 'dialog closed on 422');
            });

            await check('HX-Trigger closeDialog closes the dialog and restores focus', async () => {
                await click('#dlg-name');
                await type('Ada Lovelace');
                await click('#dlg-save');
                await eventually(() => js('!document.getElementById("dlg").open'), 'dialog to close');
                await eventually(async () => await focused() === 'open-dialog', 'focus back on the opener', 1000)
                    .catch(async e => { throw new Error(`${e.message}; focus is on "${await focused()}"`); });
            });

            await check('Escape closes the dialog and restores focus', async () => {
                await click('#open-dialog');
                await eventually(() => js('document.getElementById("dlg").open'), 'dialog to open');
                await key('Escape', 'Escape', 27);
                await eventually(() => js('!document.getElementById("dlg").open'), 'dialog to close');
                expect(await focused() === 'open-dialog', `focus is on ${await focused()}`);
            });
        }

        await check('a backdrop click closes the dialog; a click inside does not', async () => {
            if (!await js('document.getElementById("dlg").open')) {
                await click('#open-dialog');
                await eventually(() => js('document.getElementById("dlg").open'), 'dialog to open');
            }
            await click('#dlg .ui-dialog__body');
            expect(await js('document.getElementById("dlg").open'), 'inside click closed the dialog');
            await clickAt(8, 8);
            await eventually(() => js('!document.getElementById("dlg").open'), 'dialog to close');
            if (host) expect(await js('LedgerUI.calls.backdrop') === 1, 'the host backdrop handler did not run through the adapter');
        });

        if (full) {
            await check('ConfirmButton confirms over htmx and closes', async () => {
                await click('#cfm-form button');
                await eventually(() => js('document.getElementById("cfm").open'), 'confirm dialog');
                await click('#cfm button[type=submit]');
                await eventually(() => js('!!document.getElementById("confirm-result") && !document.getElementById("cfm").open'), 'confirmation result');
            });

            await check('drawer opens, locks the page and focuses the loaded heading', async () => {
                await click('#open-drawer');
                await eventually(() => js('document.querySelector("#drw .drawer-panel").classList.contains("is-open") && document.body.classList.contains("drawer-lock")'), 'open drawer');
                await eventually(async () => await focused() === 'drawer-heading', 'heading focus');
            });

            await check('at desktop width the drawer is a right slide-over', async () => {
                await drawerSettled();
                const box = await js('(() => { const r = document.querySelector("#drw .drawer-panel").getBoundingClientRect(); return [r.left, r.top, r.width, r.height, innerWidth, innerHeight]; })()');
                expect(Math.abs(box[0] + box[2] - box[4]) < 1 && box[2] < box[4] / 2 && Math.abs(box[3] - box[5]) < 1, `panel box ${box}`);
            });

            await check('fullscreen toggle updates class, pressed state and label', async () => {
                await click('#drw [data-drawer-fullscreen]');
                expect(await js('document.querySelector("#drw .drawer-panel").classList.contains("drawer-panel--full")'), 'not fullscreen');
                expect(await js('document.querySelector("#drw [data-drawer-fullscreen]").getAttribute("aria-pressed")') === 'true', 'aria-pressed not true');
            });

            await check('Escape closes the drawer, unlocks the page and restores focus', async () => {
                await key('Escape', 'Escape', 27);
                await eventually(() => js('!document.querySelector("#drw .drawer-panel").classList.contains("is-open") && !document.body.classList.contains("drawer-lock")'), 'closed drawer');
                expect(await focused() === 'open-drawer', `focus is on ${await focused()}`);
                expect(!(await js('document.querySelector("#drw .drawer-panel").classList.contains("drawer-panel--full")')), 'fullscreen survived close');
            });

            await check('HX-Trigger drawerClose closes the drawer after a save', async () => {
                await click('#open-drawer');
                // Reopening reloads the content; wait for the fresh form (its heading
                // takes focus on load) so the save does not race the reload.
                await eventually(async () => await focused() === 'drawer-heading', 'fresh drawer content');
                await drawerSettled();
                await click('#drawer-save');
                await eventually(() => js('!document.querySelector("#drw .drawer-panel").classList.contains("is-open")'), 'drawer to close');
            });

            await check('at phone width the drawer is a bottom sheet', async () => {
                await b.resize(390, 800);
                await click('#open-drawer');
                await eventually(() => js('document.querySelector("#drw .drawer-panel").classList.contains("is-open")'), 'open drawer');
                await eventually(async () => await focused() === 'drawer-heading', 'fresh drawer content');
                await drawerSettled();
                const box = await js('(() => { const r = document.querySelector("#drw .drawer-panel").getBoundingClientRect(); return [r.left, r.bottom, r.width, innerWidth, innerHeight]; })()');
                expect(Math.abs(box[2] - box[3]) < 1 && Math.abs(box[1] - box[4]) < 1, `panel box ${box}`);
                await key('Escape', 'Escape', 27);
                await eventually(() => js('!document.querySelector("#drw .drawer-panel").classList.contains("is-open")'), 'closed drawer');
                await b.resize(1280);
            });
        } else {
            await check('host drawer opens through the host runtime and closes on Escape via the adapter', async () => {
                await click('#open-drawer');
                await eventually(() => js('document.querySelector("#drw .drawer-panel").classList.contains("is-open")'), 'open drawer');
                expect(await js('LedgerUI.calls.openDrawer') === 1, 'the host openDrawer did not run exactly once');
                await js('document.querySelector("#drw .drawer-panel").focus()');
                await key('Escape', 'Escape', 27);
                await eventually(() => js('!document.querySelector("#drw .drawer-panel").classList.contains("is-open")'), 'closed drawer');
            });
        }

        await check('picker lists options on focus and picks with the keyboard', async () => {
            await click('#customer');
            await eventually(() => js('document.querySelectorAll("#customer-list [role=option]").length === 3'), 'three options')
                .catch(async e => { throw new Error(`${e.message}; state=${await js('JSON.stringify((({open, busy, failed, items, query}) => ({open, busy, failed, items: items.length, query}))(document.querySelector("#picker-form .entity-picker")._x_dataStack[0])) + " focus=" + (document.activeElement && document.activeElement.id)')}`); });
            expect(await js('document.getElementById("customer").getAttribute("aria-expanded")') === 'true', 'aria-expanded not true');
            if (host) {
                // The host stand-in has no keyboard model; a click on a row goes through its choose().
                await click('#customer-list li[role=option]:nth-of-type(2) button');
                await eventually(() => js('document.querySelector("#picker-form input[name=customer]").value === "c2"'), 'picked value c2');
                expect(await js('LedgerUI.calls.choose') === 1, 'the host choose did not run');
                expect(await js('LedgerUI.calls.picker') >= 1, 'the host picker factory was not used');
                return;
            }
            await type('nor');
            await eventually(() => js('document.querySelector("#customer-list [role=option]")?.textContent.includes("Northstar")'), 'filtered options');
            await key('Enter', 'Enter', 13);
            await eventually(() => js('document.querySelector("#picker-form input[name=customer]").value === "c2"'), 'picked value c2');
            expect(await js('document.getElementById("customer").value') === 'Northstar Technologies', 'input label not set');
        });

        if (full) {
            await check('picker rows can be picked with the mouse', async () => {
                await js('document.getElementById("customer").blur()');
                await eventually(() => js('document.querySelector("#picker-form .entity-picker__clear").offsetParent !== null'), 'clear button');
                await click('#picker-form .entity-picker__clear');
                await eventually(() => js('document.querySelector("#picker-form input[name=customer]").value === ""'), 'cleared value');
                await click('#customer');
                await eventually(() => js('document.querySelectorAll("#customer-list [role=option]").length === 3'), 'three options');
                await click('#customer-list li[role=option]:nth-of-type(3) button');
                await eventually(() => js('document.querySelector("#picker-form input[name=customer]").value === "c3"'), 'picked value c3');
                expect(await js('document.getElementById("customer").value') === 'Atlas Manufacturing', 'input label not set');
            });

            await check('picker creates a record when nothing matches', async () => {
                await js('document.getElementById("customer").blur()');
                await eventually(() => js('document.querySelector("#picker-form .entity-picker__clear").offsetParent !== null'), 'clear button');
                await click('#picker-form .entity-picker__clear');
                await eventually(() => js('document.querySelector("#picker-form input[name=customer]").value === ""'), 'cleared value');
                await type('Zeta Corp');
                // Enter during the search is swallowed (no form submit, no stale pick).
                await key('Enter', 'Enter', 13);
                expect(await js('location.pathname === "/" && !!document.getElementById("customer")'), 'Enter during a pending search submitted the form');
                await eventually(() => js('document.querySelector("#customer-list .entity-picker__option--active")?.textContent.includes("Create customer")'), 'highlighted create row');
                await key('Enter', 'Enter', 13);
                await eventually(() => js('document.querySelector("#picker-form input[name=customer]").value.startsWith("new-")'), 'created record id');
                expect(await js('document.getElementById("customer").value') === 'Zeta Corp', 'created label not shown');
            });

            await check('components swapped in by htmx initialise and work', async () => {
                // A programmatic click: the previous picker's list may still be
                // fading out over the button, and this check is about the swap.
                await js('document.getElementById("load-swap").click()');
                await eventually(() => js('!!document.querySelector("#swapped .entity-picker")?._x_dataStack'), 'Alpine on swapped picker')
                    .catch(async e => { throw new Error(`${e.message}; swapped=${await js('!!document.querySelector("#swapped .entity-picker")')} htmx=${await js('document.querySelectorAll(".htmx-request").length')}`); });
                await htmxIdle();
                await click('#swapped-picker');
                await eventually(() => js('document.querySelectorAll("#swapped-picker-list [role=option]").length === 3'), 'swapped picker options');
            });
        }

        // ── component runtime alone: orbital-ui.js without shell.js ────────────
        // Hosts with their own toast / palette system load only the component
        // runtime; the shell runtime must not be pulled in, and a toast trigger
        // addressed to it must be a no-op, not an error.
        await check('orbital-ui.js without shell.js: no OrbitalShell, and an HX-Trigger toast does nothing', async () => {
            await b.send('Page.navigate', {url: base + path});
            await eventually(() => js('!!(window.OrbitalUI && window.Alpine)'), 'the component page to start');
            b.errors.length = 0;
            expect(await js('typeof window.OrbitalShell') === 'undefined', 'window.OrbitalShell exists without shell.js');
            await js('htmx.ajax("POST", "/shell/toast", {target: "#confirm-out", swap: "none"})');
            await new Promise(r => setTimeout(r, 400));
            expect(await js('document.querySelector("[data-toast-stack], .ui-toast") === null'), 'a toast was rendered without shell.js');
            expect(b.errors.length === 0, b.errors.join('\n'));
        });

        // ── shell: palette, toast, theme switch, mobile sidebar ────────────────
        // Served on /shell with the same variant query (?csp=1 / ?host=1): the
        // shell runtime is plain DOM delegation, so it must behave identically
        // under standard Alpine, Alpine's CSP build and beside a host runtime.
        const shellPath = '/shell' + (path.includes('csp=1') ? '?csp=1' : host ? '?host=1' : '');
        const chord = async (k, code, keyCode, modifiers) => {
            for (const type of ['keyDown', 'keyUp']) await b.send('Input.dispatchKeyEvent', {type, key: k, code, windowsVirtualKeyCode: keyCode, modifiers});
        };
        const openShell = async () => {
            await b.send('Page.navigate', {url: base + shellPath});
            await eventually(() => js('!!(window.OrbitalShell && window.Alpine && document.querySelector("[data-oui-shell]"))'), 'the shell page to start');
        };
        const paletteOpen = () => js('document.querySelector("dialog[data-oui-palette]").open');
        // Records which theme <html> carried when <body> first appeared: what a
        // pre-paint script must have set, with no flash of the wrong theme.
        await b.send('Page.addScriptToEvaluateOnNewDocument', {source: `new MutationObserver((_, o) => { if (document.body) { window.__themeAtBody = document.documentElement.getAttribute('data-theme'); o.disconnect(); } }).observe(document, {childList: true, subtree: true});`});
        await b.resize(1280);
        await openShell();
        await js('localStorage.clear()');

        await check('shell renders the frame, sidebar items and a closed palette', async () => {
            expect(await js('!!document.querySelector(".ui-shell__sidebar .ui-nav__link[aria-current=page]") && !!document.querySelector(".ui-nav__count") && !!document.querySelector(".ui-nav__pct")'), 'sidebar items missing');
            expect(await js('getComputedStyle(document.querySelector(".ui-topbar__burger")).display === "none"'), 'burger visible at desktop width');
            expect(!(await paletteOpen()), 'palette open at load');
            expect(await js('document.querySelector("[data-oui-shell]").dataset.navOpen') === 'false', 'nav open at load');
        });

        await check('Ctrl+K opens the palette: combobox focused, static rows listed, first row active', async () => {
            await chord('k', 'KeyK', 75, 2);
            await eventually(paletteOpen, 'palette open');
            const input = '#command-palette-input';
            expect(await js('document.activeElement.id') === 'command-palette-input', `focus is on ${await focused()}`);
            expect(await js(`document.querySelector("${input}").getAttribute("role")`) === 'combobox', 'input is not a combobox');
            await eventually(() => js('document.querySelectorAll("#command-palette-list [role=option]").length === 3'), 'three suggestions');
            expect(await js(`document.querySelector("${input}").getAttribute("aria-activedescendant")`) === 'command-palette-opt-0', 'first row is not the active descendant');
            expect(await js('document.querySelector("#command-palette-opt-0").getAttribute("aria-selected")') === 'true', 'first row not selected');
        });

        await check('typing filters static rows, merges the endpoint results and arrows move the active row', async () => {
            await type('rep');
            await eventually(() => js('document.querySelectorAll("#command-palette-list [role=option]").length === 2'), 'static + endpoint rows')
                .catch(async e => { throw new Error(`${e.message}; rows=${await js('[...document.querySelectorAll("#command-palette-list [role=option]")].map(e => e.textContent).join("|")')}`); });
            expect(await js('document.querySelector("#command-palette-opt-0").textContent.includes("Reports") && document.querySelector("#command-palette-opt-0 mark").textContent.toLowerCase() === "rep"'), 'static row not first or not highlighted');
            expect(await js('document.querySelector("#command-palette-opt-1").textContent.includes("Report archive 2025")'), 'endpoint row missing');
            await key('ArrowDown', 'ArrowDown', 40);
            expect(await js('document.querySelector("#command-palette-input").getAttribute("aria-activedescendant")') === 'command-palette-opt-1', 'ArrowDown did not move the active descendant');
            expect(await js('document.querySelector("#command-palette-opt-1").getAttribute("aria-selected")') === 'true', 'row 1 not selected');
            await key('ArrowUp', 'ArrowUp', 38);
            expect(await js('document.querySelector("#command-palette-input").getAttribute("aria-activedescendant")') === 'command-palette-opt-0', 'ArrowUp did not move back');
            await key('ArrowDown', 'ArrowDown', 40);
        });

        await check('Enter follows the active endpoint row', async () => {
            await key('Enter', 'Enter', 13);
            await eventually(() => js('location.pathname === "/shell/target" && location.search === "?to=archive"'), 'navigation to the archive target');
            await eventually(() => js('!!document.getElementById("target")'), 'target page');
        });

        await check('Escape closes the palette and focus returns to the opener', async () => {
            await openShell();
            await click('.ui-topbar-search');
            await eventually(paletteOpen, 'palette open from the topbar button');
            expect(await js('document.activeElement.id') === 'command-palette-input', 'input not focused');
            await key('Escape', 'Escape', 27);
            await eventually(async () => !(await paletteOpen()), 'palette closed');
            expect(await js('document.activeElement.classList.contains("ui-topbar-search")'), `focus did not return to the search button (on ${await focused()})`);
        });

        await check('a query with no match shows the empty message; Ctrl+K toggles it shut', async () => {
            await chord('k', 'KeyK', 75, 2);
            await eventually(paletteOpen, 'palette open');
            await type('zzzz');
            await eventually(() => js('!!document.querySelector("#command-palette-list .ui-palette__none")'), 'empty message');
            expect(await js('document.querySelector("#command-palette-input").getAttribute("aria-expanded")') === 'false', 'aria-expanded stayed true with no rows');
            await chord('k', 'KeyK', 75, 2);
            await eventually(async () => !(await paletteOpen()), 'palette to close on the second Ctrl+K');
        });

        await check('a close followed by an immediate reopen keeps the new search alive', async () => {
            // dialog's close event is queued: it lands after the reopen and the
            // typed query below, and must not cancel the endpoint search.
            await js(`(() => {
                const d = document.querySelector("dialog[data-oui-palette]");
                if (!d.open) d.showModal();
                d.close();
                window.OrbitalShell.openPalette();
                const input = document.getElementById("command-palette-input");
                input.value = "rep";
                input.dispatchEvent(new Event("input", {bubbles: true}));
            })()`);
            try {
                await eventually(() => js('document.querySelectorAll("#command-palette-list [role=option]").length === 2'), 'static + endpoint rows after a fast reopen')
                    .catch(async e => { throw new Error(`${e.message}; rows=${await js('[...document.querySelectorAll("#command-palette-list li")].map(e => e.textContent).join("|")')}`); });
            } finally {
                await js('(() => { const d = document.querySelector("dialog[data-oui-palette]"); if (d.open) d.close(); })()');
                await eventually(async () => !(await paletteOpen()), 'palette closed');
            }
        });

        await check('palette results never follow javascript: hrefs or form actions', async () => {
          try {
            await chord('k', 'KeyK', 75, 2);
            await eventually(paletteOpen, 'palette open');
            await type('unsafe');
            await eventually(() => js('document.querySelectorAll("#command-palette-list [role=option]").length === 2'), 'two unsafe rows')
                .catch(async e => { throw new Error(`${e.message}; open=${await paletteOpen()} value=${await js('document.querySelector("#command-palette-input").value')} focus=${await focused()} rows=${await js('[...document.querySelectorAll("#command-palette-list li")].map(e => e.textContent).join("|")')}`); });
            expect(await js('[...document.querySelectorAll("#command-palette-list [role=option] a")].every(a => a.getAttribute("href") === "#")'), 'an unsafe href reached the DOM');
            await key('ArrowDown', 'ArrowDown', 40);
            await key('Enter', 'Enter', 13);
            await key('ArrowUp', 'ArrowUp', 38);
            await key('Enter', 'Enter', 13);
            expect(await js('window.__ouiPwned === undefined && !document.querySelector("dialog[data-oui-palette] form")'), 'an unsafe target ran or a form was built');
          } finally {
            // Leave no modal open for the checks that follow, pass or fail.
            await js('(() => { const d = document.querySelector("dialog[data-oui-palette]"); if (d.open) d.close(); })()');
            await eventually(async () => !(await paletteOpen()), 'palette closed');
          }
        });

        await check('an HX-Trigger toast appears with its icon and the right live semantics, then auto-dismisses', async () => {
            await js('document.querySelector("[data-toast-stack]").setAttribute("data-lifetime", "700")');
            await click('#fire-success');
            await eventually(() => js('!!document.querySelector(".ui-toast--success")'), 'success toast');
            expect(await js('document.querySelector(".ui-toast--success").getAttribute("role")') === 'status', 'success toast is not role=status');
            expect(await js('document.querySelector("[data-toast-stack]").getAttribute("aria-live")') === 'polite', 'stack is not aria-live=polite');
            expect(await js('document.querySelector(".ui-toast--success .ui-toast__title").textContent') === 'Saved', 'title not set');
            expect(await js('!!document.querySelector(".ui-toast--success .ui-toast__icon svg")'), 'icon missing');
            await eventually(() => js('!document.querySelector(".ui-toast--success")'), 'auto-dismiss', 3000);
        });

        await check('auto-dismiss pauses while the toast is hovered', async () => {
            await click('#fire-success');
            await eventually(() => js('!!document.querySelector(".ui-toast--success")'), 'success toast');
            const [x, y] = await center('.ui-toast--success');
            await b.send('Input.dispatchMouseEvent', {type: 'mouseMoved', x, y});
            await sleep(1300);
            expect(await js('!!document.querySelector(".ui-toast--success")'), 'toast dismissed while hovered');
            await b.send('Input.dispatchMouseEvent', {type: 'mouseMoved', x: 5, y: 5});
            await eventually(() => js('!document.querySelector(".ui-toast--success")'), 'dismissal after the pointer left', 3000);
        });

        await check('a danger toast is role=alert, stays put, and its close button removes it', async () => {
            await click('#fire-danger');
            await eventually(() => js('!!document.querySelector(".ui-toast--danger")'), 'danger toast');
            expect(await js('document.querySelector(".ui-toast--danger").getAttribute("role")') === 'alert', 'danger toast is not role=alert');
            await sleep(1200);
            expect(await js('!!document.querySelector(".ui-toast--danger")'), 'danger toast auto-dismissed');
            await click('.ui-toast--danger .ui-toast__close');
            await eventually(() => js('!document.querySelector(".ui-toast--danger")'), 'close button to remove the toast');
        });

        await check('a string payload is the message; the server-rendered toast has its own close button', async () => {
            await click('#fire-string');
            await eventually(() => js('[...document.querySelectorAll(".ui-toast__msg")].some(e => e.textContent === "Plain message")'), 'string toast');
            await click('.ui-toast:not(.ui-toast--success):not(.ui-toast--danger) .ui-toast__close');
            await eventually(() => js('![...document.querySelectorAll(".ui-toast__msg")].some(e => e.textContent === "Plain message")'), 'string toast closed');
            expect(await js('window.OrbitalShell.toast({tone: "ai", message: "From script", auto: false}) instanceof HTMLElement'), 'OrbitalShell.toast did not return the toast');
            await eventually(() => js('!!document.querySelector(".ui-toast--ai")'), 'script toast');
        });

        await check('theme.js runs before the stylesheets and the first theme follows the system', async () => {
            expect(await js('(() => { const s = document.querySelector("script[src$=\\"theme.js\\"]"); return !!s && !s.defer && !s.async && s.compareDocumentPosition(document.querySelector("link[rel=stylesheet]")) & Node.DOCUMENT_POSITION_FOLLOWING; })()'), 'theme.js is not a blocking script ahead of the stylesheets');
            await b.send('Emulation.setEmulatedMedia', {features: [{name: 'prefers-color-scheme', value: 'dark'}]});
            await js('localStorage.clear()');
            await openShell();
            expect(await js('window.__themeAtBody') === 'dark', `data-theme was ${await js('window.__themeAtBody')} when <body> appeared`);
            expect(await js('document.documentElement.dataset.themePreference') === 'system', 'preference is not system');
            expect(await js('document.querySelector(".ui-topbar [data-oui-theme-switch] input:checked").value') === 'system', 'switch does not show system');
            await b.send('Emulation.setEmulatedMedia', {features: [{name: 'prefers-color-scheme', value: 'light'}]});
            await eventually(() => js('document.documentElement.dataset.theme === "light"'), 'live follow of the system change');
        });

        await check('the theme switch applies, persists and survives a reload without a flash', async () => {
            await click('.ui-topbar [data-oui-theme-switch] input[value=dark]');
            await eventually(() => js('document.documentElement.dataset.theme === "dark"'), 'dark theme applied');
            expect(await js('localStorage.getItem("oui-theme")') === 'dark', 'preference not persisted');
            expect(await js('getComputedStyle(document.querySelector(".ui-shell")).backgroundColor') !== await js('(() => { const el = document.createElement("div"); el.style.background = "#f6f7f9"; document.body.append(el); const c = getComputedStyle(el).backgroundColor; el.remove(); return c; })()'), 'the canvas did not change colour');
            await openShell();
            expect(await js('window.__themeAtBody') === 'dark', `reload painted ${await js('window.__themeAtBody')} first`);
            expect(await js('document.querySelector(".ui-topbar [data-oui-theme-switch] input:checked").value') === 'dark', 'switch does not reflect the stored preference');
            await click('.ui-topbar [data-oui-theme-switch] input[value=light]');
            await eventually(() => js('document.documentElement.dataset.theme === "light" && localStorage.getItem("oui-theme") === "light"'), 'light pinned');
            await b.send('Emulation.setEmulatedMedia', {features: [{name: 'prefers-color-scheme', value: 'dark'}]});
            await sleep(150);
            expect(await js('document.documentElement.dataset.theme') === 'light', 'an explicit light choice followed the system');
            await b.send('Emulation.setEmulatedMedia', {features: [{name: 'prefers-color-scheme', value: 'light'}]});
        });

        await check('at phone width the burger opens the off-canvas sidebar and the scrim closes it', async () => {
            await b.resize(390, 800);
            await openShell();
            expect(await js('getComputedStyle(document.querySelector(".ui-shell__sidebar")).visibility') === 'hidden', 'sidebar visible while closed');
            expect(await js('document.querySelector(".ui-shell__sidebar").inert'), 'closed sidebar is not inert');
            const size = await js('(() => { const r = document.querySelector(".ui-topbar__burger").getBoundingClientRect(); return [r.width, r.height]; })()');
            expect(size[0] >= 44 && size[1] >= 44, `burger is ${size} (needs a 44px target)`);
            await click('.ui-topbar__burger');
            await eventually(() => js('getComputedStyle(document.querySelector(".ui-shell__sidebar")).visibility === "visible" && !document.querySelector(".ui-shell__sidebar").inert'), 'sidebar to open');
            expect(await js('document.querySelector(".ui-topbar__burger").getAttribute("aria-expanded")') === 'true', 'aria-expanded not true');
            expect(await js('document.activeElement.closest(".ui-shell__sidebar") !== null'), `focus is on ${await focused()}, not in the sidebar`);
            await eventually(() => js('document.querySelector(".ui-shell__sidebar").getBoundingClientRect().left >= -1'), 'sidebar to slide in');
            await clickAt(370, 500);
            await eventually(() => js('document.querySelector("[data-oui-shell]").dataset.navOpen === "false"'), 'scrim click to close');
            expect(await js('document.activeElement.classList.contains("ui-topbar__burger")'), `focus is on ${await focused()}, not the burger`);
            await click('.ui-topbar__burger');
            await eventually(() => js('document.querySelector("[data-oui-shell]").dataset.navOpen === "true"'), 'reopen');
            await key('Escape', 'Escape', 27);
            await eventually(() => js('document.querySelector("[data-oui-shell]").dataset.navOpen === "false"'), 'Escape to close');
            await b.resize(1280);
        });

        await check('phone topbar search and New are 36px chips with an invisible 44px hit area', async () => {
            await b.resize(390, 800);
            await openShell();
            const probe = sel => js(`(() => {
                const el = document.querySelector(${JSON.stringify(sel)});
                const r = el.getBoundingClientRect(), cx = r.left + r.width / 2, cy = r.top + r.height / 2;
                const hit = (dx, dy) => { const t = document.elementFromPoint(cx + dx, cy + dy); return !!t && (t === el || el.contains(t)); };
                const cs = getComputedStyle(el);
                return {w: r.width, h: r.height, centre: hit(0, 0), edge: hit(20, 20) && hit(-20, -20) && hit(20, -20) && hit(-20, 20), outside: hit(26, 0) || hit(-26, 0) || hit(0, 26) || hit(0, -26), bg: cs.backgroundColor, shadow: cs.boxShadow, radius: cs.borderTopLeftRadius};
            })()`);
            const search = await probe('.ui-topbar-search');
            expect(search.w === 36 && search.h === 36, `search chip is ${search.w}x${search.h}, want 36x36`);
            expect(search.centre && search.edge, 'search: a point 20px off-centre does not hit it');
            expect(!search.outside, 'search: the hit area extends beyond 44px');
            expect(search.bg !== 'rgba(0, 0, 0, 0)' && search.shadow !== 'none' && parseFloat(search.radius) > 0, `search chip lost its surface/ring/radius (${search.bg} | ${search.shadow} | ${search.radius})`);
            const add = await probe('.ui-topbar-new__btn');
            expect(add.w === 36 && add.h === 36, `New is ${add.w}x${add.h}, want 36x36`);
            expect(add.centre && add.edge, 'New: a point 20px off-centre does not hit it');
            expect(!add.outside, 'New: the hit area extends beyond 44px');
            // A host-set --oui-topbar-control resizes the chips and keeps the 44px hit area.
            await js('document.documentElement.style.setProperty("--oui-topbar-control", "2rem")');
            const small = await probe('.ui-topbar-search');
            expect(small.w === 32 && small.edge && !small.outside, `32px chip: ${JSON.stringify(small)}`);
            await js('document.documentElement.style.removeProperty("--oui-topbar-control")');
            await click('.ui-topbar-search');
            await eventually(paletteOpen, 'palette to open from a tap 0px off the chip centre');
            await key('Escape', 'Escape', 27);
            await eventually(async () => !(await paletteOpen()), 'palette to close');
        });

        await check('phone nav, menu and palette rows keep their desktop heights unless a host sets --oui-row-touch', async () => {
            const rowH = () => js('document.querySelector(".ui-nav--shell .ui-nav__link:not(.ui-nav__link--dense)").getBoundingClientRect().height');
            const heights = () => js('[...document.querySelectorAll(".ui-nav--shell .ui-nav__link")].map(a => Math.round(a.getBoundingClientRect().height))');
            await click('.ui-topbar__burger');
            await eventually(() => js('document.querySelector("[data-oui-shell]").dataset.navOpen === "true"'), 'sidebar to open');
            await eventually(() => js('document.querySelector(".ui-shell__sidebar").getBoundingClientRect().left >= -1'), 'sidebar to slide in');
            const phone = await heights();
            expect(phone[0] === 34 && phone[1] === 34, `default phone nav rows are ${phone} (want the desktop 34 for the first section)`);
            await js('document.documentElement.style.setProperty("--oui-row-touch", "2.75rem")');
            const touch = await heights();
            expect(touch.every(h => h === 44), `with --oui-row-touch every nav row should be 44: ${touch}`);
            await js('document.documentElement.style.removeProperty("--oui-row-touch")');
            await key('Escape', 'Escape', 27);
            // Shell menu rows: 32px by default on a phone, uniform with the token.
            await click('.ui-topbar-new__btn');
            await eventually(() => js('document.querySelector(".ui-topbar-new").open'), 'New menu to open');
            await sleep(400); // the panel scales in
            const menuH = () => js('[...document.querySelectorAll(".ui-topbar-new .ui-menu__item")].map(a => Math.round(a.getBoundingClientRect().height))');
            expect((await menuH()).every(h => h === 32), `default phone menu rows are ${await menuH()}`);
            await js('document.documentElement.style.setProperty("--oui-row-touch", "2.75rem")');
            expect((await menuH()).every(h => h === 44), `tokened menu rows are ${await menuH()}`);
            await key('Escape', 'Escape', 27);
            await js('document.documentElement.style.removeProperty("--oui-row-touch")');
            // Outside AppShell a phone menu row keeps the v1 44px touch height.
            const looseH = await js('(() => { const a = document.createElement("a"); a.className = "ui-menu__item"; a.href = "#"; a.textContent = "Row"; document.body.append(a); const h = Math.round(a.getBoundingClientRect().height); a.remove(); return h; })()');
            expect(looseH === 44, `phone menu row outside the shell is ${looseH}, want 44`);
            // Palette rows: the pointer: coarse rule defaults to the desktop 2.5rem and takes the token.
            const coarse = await js('[...document.styleSheets].flatMap(sh => { try { return [...sh.cssRules]; } catch (e) { return []; } }).filter(r => r.media && r.media.mediaText.includes("pointer: coarse")).map(r => r.cssText).join(" ")');
            expect(/\.ui-palette__item a \{[^}]*height: var\(--oui-row-touch, 2\.5rem\)/.test(coarse), `coarse palette rule: ${coarse}`);
            await b.resize(1280);
        });

        await check('menus: arrows, Home/End move between items; Escape closes and returns focus to the trigger', async () => {
            await openShell();
            const onItem = () => js('document.activeElement.textContent.trim()');
            await js('document.querySelector(".ui-topbar-new > summary").focus()');
            await key('ArrowDown', 'ArrowDown', 40);
            await eventually(() => js('document.querySelector(".ui-topbar-new").open'), 'ArrowDown to open the menu');
            expect(await onItem() === 'Document', `first item is ${await onItem()}`);
            await key('ArrowDown', 'ArrowDown', 40);
            expect(await onItem() === 'Folder', `next item is ${await onItem()}`);
            await key('End', 'End', 35);
            expect(await onItem() === 'Import', `End -> ${await onItem()}`);
            await key('ArrowDown', 'ArrowDown', 40);
            expect(await onItem() === 'Document', `wrap -> ${await onItem()}`);
            await key('ArrowUp', 'ArrowUp', 38);
            expect(await onItem() === 'Import', `ArrowUp wrap -> ${await onItem()}`);
            await key('Home', 'Home', 36);
            expect(await onItem() === 'Document', `Home -> ${await onItem()}`);
            await key('Escape', 'Escape', 27);
            await eventually(() => js('!document.querySelector(".ui-topbar-new").open'), 'Escape to close');
            expect(await js('document.activeElement === document.querySelector(".ui-topbar-new > summary")'), `focus is on ${await focused()}, not the trigger`);
        });

        await check('arrow keys on a ThemeSwitch inside a menu move the radio, not the menu focus', async () => {
            await openShell();
            await js('OrbitalTheme.setPreference("system")');
            await js('document.querySelector(".ui-sidebar-user-menu").setAttribute("open", "")');
            await js('document.querySelector(".ui-sidebar-user-menu input[name=account-theme][value=system]").focus()');
            await key('ArrowDown', 'ArrowDown', 40);
            await eventually(() => js('document.activeElement.name === "account-theme" && document.activeElement.value !== "system" && document.activeElement.checked'), 'the next theme radio to be focused and checked')
                .catch(async e => { throw new Error(`${e.message}; focus=${await focused()}`); });
            expect(await js('document.querySelectorAll("input[name=account-theme]").length === 3 && document.querySelectorAll("input[name=oui-theme]").length === 3'), 'the two switches do not have separate radio groups');
            await js('OrbitalTheme.setPreference("system"); document.querySelector(".ui-sidebar-user-menu").removeAttribute("open")');
        });

        await check('theme-color meta follows the theme through the data-light / data-dark attributes', async () => {
            await openShell();
            const meta = () => js('document.querySelector("meta[name=theme-color]").getAttribute("content")');
            await js('OrbitalTheme.setPreference("light")');
            expect(await meta() === '#fbfaf8', `light content is ${await meta()}`);
            await js('OrbitalTheme.setPreference("dark")');
            expect(await meta() === '#14130f', `dark content is ${await meta()}`);
            // Another tab changing the preference arrives as a storage event.
            await js('localStorage.setItem("oui-theme", "light"); window.dispatchEvent(new StorageEvent("storage", {key: "oui-theme", newValue: "light"}))');
            expect(await meta() === '#fbfaf8', `cross-tab content is ${await meta()}`);
            // No data attributes: untouched.
            await js('(() => { const m = document.createElement("meta"); m.name = "theme-color"; m.content = "#123456"; document.head.appendChild(m); window.__plainMeta = m; })()');
            await js('OrbitalTheme.setPreference("dark")');
            expect(await js('window.__plainMeta.getAttribute("content")') === '#123456', 'a theme-color meta without data attributes was changed');
            await js('localStorage.clear()');
        });

        await check('a palette form row posts the text in the box, not the last response', async () => {
            await openShell();
            await chord('k', 'KeyK', 75, 2);
            await eventually(paletteOpen, 'palette open');
            await type('ask');
            await eventually(() => js('[...document.querySelectorAll("#command-palette-list [role=option]")].some(e => e.textContent.includes("Ask agent"))'), 'the Ask agent row');
            const ask = await js('(() => { const o = [...document.querySelectorAll("#command-palette-list [role=option]")]; const row = o.find(e => e.textContent.includes("Ask agent")); const org = o.find(e => e.textContent.includes("Open org")); return {selected: row.getAttribute("aria-selected"), index: o.indexOf(row), promoted: row.classList.contains("ui-palette__item--promoted"), text: row.textContent, quoteIn: !!row.querySelector(".ui-palette__quote"), marks: row.querySelectorAll("mark").length, orgAccent: !!org.querySelector(".ui-avatar--accent.ui-avatar--square")}; })()');
            expect(ask.index === 1 && ask.selected === 'true', `Default row: ${JSON.stringify(ask)}`);
            expect(ask.promoted && ask.quoteIn && ask.marks === 0 && ask.text.includes('Ask agent “ask”'), `promoted/quote row: ${JSON.stringify(ask)}`);
            expect(ask.orgAccent, 'a square avatar row did not take the accent tone');
            // Keep typing and submit inside the debounce window, before the next response.
            await type(' the board');
            await key('Enter', 'Enter', 13);
            await eventually(() => js('!!document.querySelector("#echo")'), 'the echo page');
            expect(await js('document.querySelector("#echo").textContent') === 'ask the board', `posted ${JSON.stringify(await js('document.querySelector("#echo").textContent'))}`);
            expect(await js('document.querySelector("#echo-token").textContent') === 't0k', 'the other form fields were lost');
        });

        await check('a palette row announces oui:palette-submit (item, query, row) before it is followed, and a handler can cancel it', async () => {
            await openShell();
            await js('window.__ps = []; document.querySelector("dialog[data-oui-palette]").addEventListener("oui:palette-submit", e => { window.__ps.push({label: e.detail.item.label, query: e.detail.query, row: e.detail.row && e.detail.row.getAttribute("role"), cancelable: e.cancelable, path: location.pathname}); if (window.__psCancel) e.preventDefault(); });');
            await js('window.__psCancel = true');
            await chord('k', 'KeyK', 75, 2);
            await eventually(paletteOpen, 'palette open');
            await type('rep');
            await eventually(() => js('document.querySelectorAll("#command-palette-list [role=option]").length === 2'), 'two rows');
            await key('Enter', 'Enter', 13);
            await sleep(250);
            const cancelled = await js('({ps: window.__ps, path: location.pathname, open: document.querySelector("dialog[data-oui-palette]").open})');
            expect(cancelled.ps.length === 1 && cancelled.ps[0].label === 'Reports' && cancelled.ps[0].query === 'rep' && cancelled.ps[0].row === 'option' && cancelled.ps[0].cancelable, `event detail: ${JSON.stringify(cancelled)}`);
            expect(cancelled.path === '/shell' && cancelled.open, `a cancelled Enter still followed the row: ${JSON.stringify(cancelled)}`);
            // A pointer click is announced too, once, and cancels the same way.
            await js('window.__ps.length = 0');
            await click('#command-palette-opt-0');
            await sleep(250);
            const clicked = await js('({n: window.__ps.length, label: window.__ps[0] && window.__ps[0].label, path: location.pathname, open: document.querySelector("dialog[data-oui-palette]").open})');
            expect(clicked.n === 1 && clicked.label === 'Reports' && clicked.path === '/shell' && clicked.open, `cancelled click: ${JSON.stringify(clicked)}`);
            // Not cancelled: the event fires once and the row is followed.
            await js('window.__psCancel = false; window.__ps.length = 0; document.getElementById("command-palette-input").focus()');
            await key('ArrowDown', 'ArrowDown', 40);
            await key('Enter', 'Enter', 13);
            await eventually(() => js('location.pathname === "/shell/target" && location.search === "?to=archive"'), 'navigation after an uncancelled event');
        });

        await check('oui:palette-submit also precedes a form row, and cancelling it stops the post', async () => {
            await openShell();
            await js('window.__ps = []; window.__psCancel = true; document.querySelector("dialog[data-oui-palette]").addEventListener("oui:palette-submit", e => { window.__ps.push({label: e.detail.item.label, query: e.detail.query}); if (window.__psCancel) e.preventDefault(); });');
            await chord('k', 'KeyK', 75, 2);
            await eventually(paletteOpen, 'palette open');
            await type('ask the board');
            await eventually(() => js('[...document.querySelectorAll("#command-palette-list [role=option]")].some(e => e.textContent.includes("Ask agent"))'), 'the Ask agent row');
            await key('Enter', 'Enter', 13);
            await sleep(300);
            const held = await js('({ps: window.__ps, path: location.pathname, form: !!document.querySelector("dialog[data-oui-palette] form")})');
            expect(held.ps.length === 1 && held.ps[0].label.startsWith('Ask agent') && held.ps[0].query === 'ask the board', `form row event: ${JSON.stringify(held)}`);
            expect(held.path === '/shell' && !held.form, `a cancelled form row still submitted: ${JSON.stringify(held)}`);
            await js('window.__psCancel = false');
            await key('Enter', 'Enter', 13);
            await eventually(() => js('!!document.querySelector("#echo")'), 'the echo page after an uncancelled form row');
        });

        // ── kit parity: icon placement, avatars, orb, cards, data, ring, chips, segments, tabs, fields ──
        const kitPath = '/kit' + (path.includes('csp=1') ? '?csp=1' : host ? '?host=1' : '');
        const openKit = async () => {
            await b.resize(1280);
            await b.send('Page.navigate', {url: base + kitPath});
            await eventually(() => js('!!document.querySelector("#k-name")'), 'the kit page');
        };
        await openKit();
        const rect = sel => js(`(() => { const r = document.querySelector(${JSON.stringify(sel)}).getBoundingClientRect(); return {x: r.left, y: r.top, w: r.width, h: r.height, r: r.right, b: r.bottom}; })()`);

        await check('Button IconRight draws the icon after the label; TrailingIcon pairs with a leading icon', async () => {
            const o = await js(`(() => { const x = el => el.getBoundingClientRect().left; const f = i => { const bt = document.querySelectorAll("#k-buttons .ui-button")[i]; const svgs = [...bt.querySelectorAll("svg")]; return {svgs: svgs.map(x), label: x(bt.querySelector("span"))}; }; return [f(0), f(1), f(2)]; })()`);
            expect(o[0].svgs.length === 1 && o[0].svgs[0] > o[0].label, `IconRight icon is not right of the label: ${JSON.stringify(o[0])}`);
            expect(o[1].svgs.length === 2 && o[1].svgs[0] < o[1].label && o[1].svgs[1] > o[1].label, `leading + trailing pair: ${JSON.stringify(o[1])}`);
            expect(o[2].svgs[0] < o[2].label, `default leading icon moved: ${JSON.stringify(o[2])}`);
        });

        await check('Avatar initials follow Kindred\'s font size per avatar size', async () => {
            const o = await js('[...document.querySelectorAll("#k-avatars .ui-avatar")].map(a => { const cs = getComputedStyle(a); return [Math.round(a.getBoundingClientRect().width), parseFloat(cs.fontSize), cs.letterSpacing]; })');
            const wantFs = {16: 8, 20: 9, 24: 10.5, 32: 12, 40: 14.5, 56: 20, 72: 26};
            for (const [size, fs, ls] of o) {
                expect(wantFs[size] === fs, `avatar ${size}px has font-size ${fs}, want ${wantFs[size]}`);
                if (size === 72) expect(Math.abs(parseFloat(ls) + 0.52) < 0.02, `72px letter-spacing ${ls}, want -0.02em`);
            }
        });

        await check('the orb keeps its brand colours in dark mode (--oui-orb-a/-b/-c are not redefined)', async () => {
            const read = () => js('(() => { const cs = getComputedStyle(document.documentElement); const core = getComputedStyle(document.querySelector("#k-orb .ui-orb__core"), "::before").backgroundImage; return {a: cs.getPropertyValue("--oui-orb-a").trim(), b: cs.getPropertyValue("--oui-orb-b").trim(), c: cs.getPropertyValue("--oui-orb-c").trim(), brand: cs.getPropertyValue("--oui-brand").trim(), core}; })()');
            await js('OrbitalTheme.setPreference("light")');
            const light = await read();
            await js('OrbitalTheme.setPreference("dark")');
            const dark = await read();
            expect(dark.a === light.a && dark.b === light.b && dark.c === light.c, `orb tokens changed in dark: ${JSON.stringify([light, dark])}`);
            expect(dark.brand !== light.brand, 'dark mode did not change --oui-brand, so the check proves nothing');
            expect(dark.core.includes('rgb(68, 76, 231)') && !dark.core.includes(dark.brand), `the dark orb swirl is not drawn from the light stops: ${dark.core}`);
            await js('document.documentElement.style.setProperty("--oui-orb-a", "rgb(1, 2, 3)")');
            expect((await read()).core.includes('rgb(1, 2, 3)'), 'a host re-point of --oui-orb-a did not reach the orb');
            await js('document.documentElement.style.removeProperty("--oui-orb-a"); OrbitalTheme.setPreference("light")');
        });

        await check('Card Padded carries a 20px gutter; Bare has no body wrapper and lets host classes lay out the content', async () => {
            const o = await js(`(() => { const p = document.querySelector("#k-card-padded"), g = document.querySelector("#k-card-bare"); const ps = getComputedStyle(p), gs = getComputedStyle(g), hs = getComputedStyle(p.querySelector(".ui-card__header")); return {pad: ps.paddingLeft + "/" + ps.paddingTop, headPad: hs.paddingLeft, align: hs.alignItems, body: !!p.querySelector(".ui-card__body"), bareBody: !!g.querySelector(".ui-card__body"), bareDisplay: gs.display, barePad: gs.paddingLeft, direct: g.querySelectorAll(":scope > span").length}; })()`);
            expect(o.pad === '20px/20px' && o.headPad === '0px' && o.body, `padded card: ${JSON.stringify(o)}`);
            expect(o.align === 'flex-start', `default header align-items ${o.align}, want flex-start`);
            await js('document.querySelector("#k-card-padded").style.setProperty("--oui-card-header-align", "center")');
            const centred = await js('getComputedStyle(document.querySelector("#k-card-padded .ui-card__header")).alignItems');
            expect(centred === 'center', `--oui-card-header-align: center gave ${centred}`);
            await js('document.querySelector("#k-card-padded").style.removeProperty("--oui-card-header-align")');
            expect(!o.bareBody && o.bareDisplay === 'block' && o.barePad === '20px' && o.direct === 2, `bare card: ${JSON.stringify(o)}`);
            await js('(() => { const s = document.createElement("style"); s.id = "host-s"; s.textContent = ".host-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }"; document.head.appendChild(s); })()');
            const cols = await js('getComputedStyle(document.querySelector("#k-card-bare")).gridTemplateColumns.split(" ").length');
            expect(cols === 2, `a host layout class on a Bare card did not apply (${cols} columns)`);
            await js('document.getElementById("host-s").remove()');
        });

        await check('DataList rows keep their dt/dd together when a sibling row is taller; cols gaps are 16px / 28px', async () => {
            const o = await js(`(() => { const rows = [...document.querySelectorAll("#k-dl .ui-dl__row")]; const r0 = rows[0]; const dt = r0.querySelector("dt").getBoundingClientRect(), dd = r0.querySelector("dd").getBoundingClientRect(); const dl = getComputedStyle(document.querySelector("#k-dl dl")); return {gap: dt.bottom - dd.top, rowH: r0.getBoundingClientRect().height, dtH: dt.height, ddH: dd.height, rg: dl.rowGap, cg: dl.columnGap, align: getComputedStyle(r0).alignContent, wide: getComputedStyle(rows[2]).gridColumnStart, dd2: rows[1].querySelectorAll("dd span").length}; })()`);
            expect(o.align === 'start' || o.align === 'flex-start', `row align-content ${o.align}`);
            expect(Math.abs(o.gap) <= 4, `dt/dd spread inside a tall row (rows stretch to the tallest, content must not): ${JSON.stringify(o)}`);
            expect(o.rg === '16px' && o.cg === '28px', `gaps ${o.rg} / ${o.cg}`);
            expect(o.dd2 === 4, 'DataItem children (rich values) were not rendered inside the dd');
        });

        await check('RingMeter takes any pixel size and an optional arc glow via --oui-ring-glow', async () => {
            const a = await rect('#k-ring .ui-ring:nth-child(1)'), s = await rect('#k-ring .ui-ring:nth-child(2)'), p = await rect('#k-ring .ui-ring:nth-child(3)');
            expect(a.w === 72 && a.h === 72 && s.w === 28 && s.h === 28 && p.w === 64, `ring sizes ${a.w}, ${s.w}, ${p.w}`);
            expect(await js('getComputedStyle(document.querySelector("#k-ring .ui-ring__arc")).filter') === 'none', 'the arc has a glow by default');
            await js('document.documentElement.style.setProperty("--oui-ring-glow", "drop-shadow(0 1px 5px rgb(178 75 243 / 0.28))")');
            expect((await js('getComputedStyle(document.querySelector("#k-ring .ui-ring__arc")).filter')).startsWith('drop-shadow'), '--oui-ring-glow did not apply');
            await js('document.documentElement.style.removeProperty("--oui-ring-glow")');
        });

        await check('ChipGroup: tick first, spark last, a field-style legend, help and error, 32px chips', async () => {
            const o = await js(`(() => { const fs = document.querySelector("fieldset.ui-chip-group"); const body = fs.querySelectorAll(".ui-choice-chip__body")[1]; const x = sel => body.querySelector(sel).getBoundingClientRect().left; const lg = getComputedStyle(fs.querySelector("legend")); return {tick: x(".ui-choice-chip__tick"), label: x(".ui-choice-chip__label"), spark: x(".ui-choice-chip__spark"), sparkTitle: body.querySelector(".ui-choice-chip__spark").title, legendW: lg.fontWeight, legendSize: lg.fontSize, req: !!fs.querySelector("legend .field-label__req"), err: fs.querySelector(".field-error").getAttribute("role"), help: !!fs.querySelector(".field-help"), desc: fs.getAttribute("aria-describedby"), h: body.getBoundingClientRect().height}; })()`);
            expect(o.label < o.spark, `spark is not after the label: ${JSON.stringify(o)}`);
            const alpha = await js('(() => { const body = document.querySelectorAll("fieldset.ui-chip-group .ui-choice-chip__body")[0]; const t = body.querySelector(".ui-choice-chip__tick"); return [getComputedStyle(t).display, t.getBoundingClientRect().left < body.querySelector(".ui-choice-chip__label").getBoundingClientRect().left]; })()');
            expect((alpha[0] === 'inline-flex' || alpha[0] === 'flex') && alpha[1], `a selected chip's tick is not before its label: ${JSON.stringify(alpha)}`);
            expect(o.sparkTitle === 'Suggested' && o.legendW === '500' && o.legendSize === '13px' && o.req, `legend / spark: ${JSON.stringify(o)}`);
            expect(o.err === 'alert' && o.help && o.desc === 'k-chip-error k-chip-help', `help / error: ${JSON.stringify(o)}`);
            expect(o.h === 32, `chip height ${o.h}, want 32`);
        });

        await check('SegmentedNav slides a thumb to the active segment; TabNav counts follow the active-count tokens', async () => {
            const nav = await rect('#k-seg .ui-seg-nav');
            const thumb = () => js('(() => { const n = document.querySelector("#k-seg .ui-seg-nav"); const cs = getComputedStyle(n, "::before"); return {x: new DOMMatrix(cs.transform).m41, w: parseFloat(cs.width), r: cs.borderTopLeftRadius, pad: cs.left}; })()');
            const t1 = await thumb();
            const seg = (nav.w - 6) / 3;
            expect(Math.abs(t1.w - seg) < 1 && Math.abs(t1.x - seg) < 1 && t1.r === '8px' && t1.pad === '3px', `thumb on segment 2: ${JSON.stringify(t1)} (segment ${seg})`);
            await js('document.querySelector("#k-seg .ui-seg-nav").classList.replace("ui-seg-nav--i1", "ui-seg-nav--i2")');
            await eventually(async () => Math.abs((await thumb()).x - 2 * seg) < 1, 'the thumb to slide to segment 3');
            const active = await js('(() => { const a = document.querySelector("#k-tabs a.tab--active"); const after = getComputedStyle(a, "::after"); return {line: after.backgroundColor, ariaCurrent: a.getAttribute("aria-current"), count: getComputedStyle(a.querySelector(".tab__count")).backgroundColor}; })()');
            expect(active.line !== 'rgba(0, 0, 0, 0)' && active.ariaCurrent === 'page', `active tab: ${JSON.stringify(active)}`);
            await js('document.documentElement.style.setProperty("--oui-tab-count-active-bg", "rgb(1, 2, 3)")');
            expect(await js('getComputedStyle(document.querySelector("#k-tabs a.tab--active .tab__count")).backgroundColor') === 'rgb(1, 2, 3)', '--oui-tab-count-active-bg did not reach the active count');
            expect(await js('getComputedStyle(document.querySelector("#k-tabs a:not(.tab--active) .tab__count")).backgroundColor') !== 'rgb(1, 2, 3)', 'the token leaked to an inactive tab');
            await js('document.documentElement.style.removeProperty("--oui-tab-count-active-bg")');
        });

        await check('IconSized sets 12/16/24px; SuggestedTag has a configurable title', async () => {
            const sizes = await js('[...document.querySelectorAll("#k-icons svg")].map(s => s.getBoundingClientRect().width)');
            expect(sizes.join() === '12,16,24', `icon sizes ${sizes}`);
            expect(await js('document.querySelector("#k-field .ui-suggested").title') === 'Suggested by your agent', 'SuggestedTag title');
        });

        await check('field controls are a box-shadow ring (no layout change); focus takes the action ring, halo and --control-focus-border', async () => {
            const o = await js(`(() => { const el = document.querySelector("#k-plain"); const cs = getComputedStyle(el); return {h: el.getBoundingClientRect().height, want: parseFloat(cs.minHeight) + (cs.boxSizing === "content-box" ? 2 : 0), bw: cs.borderTopWidth, bc: cs.borderTopColor, sh: cs.boxShadow}; })()`);
            expect(o.h === o.want && o.bw === '1px' && o.bc === 'rgba(0, 0, 0, 0)', `control metrics: ${JSON.stringify(o)}`);
            expect(/0px 0px 0px 1px/.test(o.sh) && /0px 1px 1px 0px/.test(o.sh), `ring + elevation: ${o.sh}`);
            await click('#k-plain');
            await sleep(250);
            const f = await js('getComputedStyle(document.querySelector("#k-plain")).boxShadow');
            expect(/0px 0px 0px 1px/.test(f) && /0px 0px 0px 4px/.test(f), `focus ring + halo: ${f}`);
            await js('document.documentElement.style.setProperty("--control-focus-border", "rgb(255, 0, 0)")');
            await sleep(250);
            expect((await js('getComputedStyle(document.querySelector("#k-plain")).boxShadow')).includes('rgb(255, 0, 0)'), '--control-focus-border no longer drives the focus ring');
            await js('document.documentElement.style.removeProperty("--control-focus-border")');
        });

        // ── kit finish: the last small options and tokens ──
        const cs = (sel, prop, pseudo) => js(`getComputedStyle(document.querySelector(${JSON.stringify(sel)})${pseudo ? ', ' + JSON.stringify(pseudo) : ''})[${JSON.stringify(prop)}]`);
        const setVar = (sel, name, val) => js(`document.querySelector(${JSON.stringify(sel)}).style.setProperty(${JSON.stringify(name)}, ${JSON.stringify(val)})`);
        const clearVar = (sel, name) => js(`document.querySelector(${JSON.stringify(sel)}).style.removeProperty(${JSON.stringify(name)})`);

        await check('--oui-control-pad-x insets field text (default .75rem) and the shell affixes', async () => {
            expect(await cs('#kf-bare', 'paddingLeft') === '12px', `default pad ${await cs('#kf-bare', 'paddingLeft')}`);
            await setVar('#k-finish', '--oui-control-pad-x', '11px');
            const o = await js('(() => { const i = document.querySelector("#kf-bare"), a = document.querySelector("#kf-pad").parentElement.querySelector(".field-input__affix"); return [getComputedStyle(i).paddingLeft, getComputedStyle(i).paddingRight, "", getComputedStyle(a).paddingLeft]; })()');
            expect(o[0] === '11px' && o[1] === '11px' && o[3] === '11px', `pad-x token: ${JSON.stringify(o)}`);
            await clearVar('#k-finish', '--oui-control-pad-x');
        });

        await check('a suggested field with an icon or affix draws one ring (on the shell), not two', async () => {
            const o = await js('(() => { const i = document.querySelector("#kf-sug"); const shell = i.parentElement; return {shell: getComputedStyle(shell).boxShadow, inner: getComputedStyle(i).boxShadow}; })()');
            expect(o.inner === 'none' && o.shell !== 'none' && o.shell.includes('0px 0px 0px 1px'), `suggested shell rings: ${JSON.stringify(o)}`);
            const bare = await js('getComputedStyle(document.querySelector("#k-name")).boxShadow');
            expect(bare !== 'none', 'a suggested bare control lost its ring');
        });

        await check('textarea min-height, resize and numeric face come from tokens with unchanged defaults', async () => {
            expect(await cs('#kf-ta', 'minHeight') === '88px' && await cs('#kf-ta', 'resize') === 'vertical', `textarea defaults ${await cs('#kf-ta', 'minHeight')} / ${await cs('#kf-ta', 'resize')}`);
            expect(await cs('#kf-ta-grow', 'resize') === 'none', 'autogrow textarea must not resize');
            await setVar('#k-finish', '--oui-textarea-min-height', '120px');
            await setVar('#k-finish', '--oui-textarea-resize', 'none');
            expect(await cs('#kf-ta', 'minHeight') === '120px' && await cs('#kf-ta', 'resize') === 'none', 'textarea tokens did not apply');
            await clearVar('#k-finish', '--oui-textarea-min-height'); await clearVar('#k-finish', '--oui-textarea-resize');
            const num = await js('(() => { const a = getComputedStyle(document.querySelector("#kf-num")), b = getComputedStyle(document.querySelector("#kf-numtype")), p = getComputedStyle(document.querySelector("#kf-bare")); return {num: a.fontFamily, numeric: a.fontVariantNumeric, type: b.fontFamily, plain: p.fontFamily}; })()');
            expect(num.num === num.type && num.num !== num.plain && num.numeric.includes('tabular-nums'), `.num does not win over font: inherit: ${JSON.stringify(num)}`);
        });

        await check('--oui-control-ring-inset draws the field ring inside; default is outside', async () => {
            expect(!(await cs('#kf-bare', 'boxShadow')).includes('inset'), 'default ring is inset');
            await setVar('#k-finish', '--oui-control-ring-inset', 'inset');
            const o = await js('[getComputedStyle(document.querySelector("#kf-bare")).boxShadow, getComputedStyle(document.querySelector("#kf-sug").parentElement).boxShadow]');
            expect(o[0].includes('inset') && o[1].includes('inset'), `inset ring: ${JSON.stringify(o)}`);
            await clearVar('#k-finish', '--oui-control-ring-inset');
        });

        await check('ChipGroup finish: 12px tick/spark, legend flush, ai spark on a checked chip, ring tokens, spark title', async () => {
            const o = await js(`(() => { const fs = document.querySelector("fieldset.ui-chip-group[aria-describedby=undefined], fieldset.ui-chip-group:has(input[name=kf-chip])"); const body = fs.querySelector(".ui-choice-chip__body"); const w = sel => body.querySelector(sel + " svg").getBoundingClientRect().width; const checkedSpark = getComputedStyle(body.querySelector(".ui-choice-chip__spark")).color; return {tick: w(".ui-choice-chip__tick"), spark: w(".ui-choice-chip__spark"), legend: getComputedStyle(fs.querySelector("legend")).marginBottom, sparkColor: checkedSpark, chipColor: getComputedStyle(body).color, title: body.querySelector(".ui-choice-chip__spark").title}; })()`);
            expect(o.tick === 12 && o.spark === 12 && o.legend === '0px', `chip marks: ${JSON.stringify(o)}`);
            expect(o.sparkColor !== o.chipColor && o.title === 'Suggested by your agent', `checked spark inherits the chip colour / title: ${JSON.stringify(o)}`);
            const ring = sel => js(`getComputedStyle(document.querySelectorAll("fieldset:has(input[name=kf-chip]) .ui-choice-chip__body")[${sel}]).boxShadow`);
            const beta = await ring(1);
            await setVar('#k-finish', '--oui-chip-suggested-ring', 'rgb(1, 2, 3)');
            await setVar('#k-finish', '--oui-chip-suggested-halo', 'rgb(4, 5, 6)');
            await sleep(300);
            const after = [await ring(1), await ring(0)];
            expect(after[0].includes('rgb(1, 2, 3)') && after[1].includes('rgb(4, 5, 6)') && beta !== after[0], `chip ring tokens: ${beta} -> ${after}`);
            await clearVar('#k-finish', '--oui-chip-suggested-ring'); await clearVar('#k-finish', '--oui-chip-suggested-halo');
        });

        await check('DataList wide rows span all columns; DataItem Field renders data-field on the dd', async () => {
            const o = await js('(() => { const rows = [...document.querySelectorAll("#kf-dl .ui-dl__row")]; const wide = rows[2].getBoundingClientRect(), a = rows[0].getBoundingClientRect(), dl = document.querySelector("#kf-dl dl").getBoundingClientRect(); return {wideW: wide.width, dlW: dl.width, aW: a.width, field: rows[2].querySelector("dd").dataset.field, plain: rows[3].querySelector("dd").dataset.field, none: rows[0].querySelector("dd").hasAttribute("data-field")}; })()');
            expect(Math.abs(o.wideW - o.dlW) < 1 && o.aW < o.dlW, `wide row does not span the columns: ${JSON.stringify(o)}`);
            expect(o.field === 'wide' && o.plain === 'plain' && !o.none, `data-field: ${JSON.stringify(o)}`);
        });

        await check('a Bare card keeps a host display:flex class as a row; padded cards stay columns; CardHeader renders alone', async () => {
            expect(await cs('#kf-bcard', 'display') === 'flex' && await cs('#kf-bcard', 'flexDirection') === 'row', `bare flex card: ${await cs('#kf-bcard', 'display')} ${await cs('#kf-bcard', 'flexDirection')}`);
            expect(await cs('#k-card-padded', 'flexDirection') === 'column', 'a normal card lost its column');
            const h = await js('(() => { const c = document.querySelector("#kf-bcard"); return {hdr: !!c.querySelector(":scope > header.ui-card__header .ui-card__icon"), t: c.querySelector(".ui-card__title").tagName, sub: !!c.querySelector(".ui-card__subtitle"), act: !!c.querySelector(".ui-card__actions .ui-button")}; })()');
            expect(h.hdr && h.t === 'H2' && h.sub && h.act, `CardHeader: ${JSON.stringify(h)}`);
        });

        await check('--oui-badge-dot tints a badge dot separately from its text', async () => {
            expect(await cs('#kf-badge .ui-badge__dot', 'backgroundColor') === await cs('#kf-badge .ui-badge', 'color'), 'the dot no longer follows the badge colour');
            await setVar('#kf-badge', '--oui-badge-dot', 'rgb(9, 8, 7)');
            expect(await cs('#kf-badge .ui-badge__dot', 'backgroundColor') === 'rgb(9, 8, 7)', '--oui-badge-dot did not apply');
            expect(await cs('#kf-badge .ui-badge', 'color') !== 'rgb(9, 8, 7)', 'the token tinted the label');
            await clearVar('#kf-badge', '--oui-badge-dot');
        });

        await check('the shared Icon svg does not shrink in a narrow flex row', async () => {
            expect(await js('document.querySelector("#kf-iconrow svg").getBoundingClientRect().width') === 17, 'the icon shrank');
        });

        await check('StatePanel TitleTag h3, FieldSelect disabled placeholder, field help id + describedby, opt-in error alert', async () => {
            const o = await js('(() => { const sel = document.querySelector("#kf-sel"); const f = sel.closest(".field"); const calm = document.querySelector("#kf-calm").closest(".field"); return {tag: document.querySelector("#kf-state .ui-state > *:not(.ui-state__icon)").tagName, dis: sel.options[0].disabled, selected: sel.selectedIndex, helpId: f.querySelector(".field-help").id, desc: sel.getAttribute("aria-describedby"), role: f.querySelector(".field-error").getAttribute("role"), calmRole: calm.querySelector(".field-error").getAttribute("role"), calmHelp: calm.querySelector(".field-help")}; })()');
            expect(o.tag === 'H3' && o.dis && o.selected === 0, `h3 / disabled placeholder: ${JSON.stringify(o)}`);
            expect(o.helpId === 'kf-sel-help' && o.desc === 'kf-sel-help kf-sel-error', `help id / describedby: ${JSON.stringify(o)}`);
            expect(o.role === 'alert' && o.calmRole === null, `error role: ${JSON.stringify(o)}`);
        });

        await check('--oui-tab-height-phone sets the phone tab height (TabNav included)', async () => {
            await b.resize(400);
            await sleep(150);
            const d = await js('document.querySelector("#k-tabs a").getBoundingClientRect().height');
            await setVar('#k-tabs', '--oui-tab-height-phone', '3.25rem');
            const t = await js('document.querySelector("#k-tabs a").getBoundingClientRect().height');
            await clearVar('#k-tabs', '--oui-tab-height-phone');
            await b.resize(1280);
            expect(d === 44 && t === 52, `phone tab height ${d} -> ${t}`);
        });

        await check('no script errors or CSP violations on the page', async () => {
            expect(b.errors.length === 0, b.errors.join('\n'));
        });
    } finally {
        await b.close();
    }
}

// Drives the agentic chat (agentchat.Chat, assets/agent-chat.js) against the
// stub agent at /agentchat: two roots on one page, served under
// `script-src 'self'`, so the runtime must work without eval or inline script.
async function runAgentChat() {
    const b = await launchChrome();
    const {js, click, key, type, check, expect} = drive(b, 'agentchat');
    const F = '#host-full ';
    const C = '#host-compact ';
    const q = (sel, expr) => js(`(() => { const el = document.querySelector(${JSON.stringify(sel)}); return el ? ${expr} : null; })()`);
    // The last match: rows of different tags share one thread, so :last-of-type cannot pick "the newest reply".
    const ql = (sel, expr) => js(`(() => { const el = [...document.querySelectorAll(${JSON.stringify(sel)})].at(-1); return el ? ${expr} : null; })()`);
    const qt = sel => ql(sel, 'el.textContent.trim()');
    const count = sel => js(`document.querySelectorAll(${JSON.stringify(sel)}).length`);
    const text = sel => q(sel, 'el.textContent.trim()');
    const state = () => js('fetch("/agentchat/state").then(r => r.json())');
    const lastRequest = async () => (await state()).requests.at(-1);
    const send = async (root, message) => {
        await click(`${root}[data-agentchat-input]`);
        await type(message);
        await key('Enter', 'Enter', 13);
    };
    const idle = root => eventually(() => js(`document.querySelector(${JSON.stringify(root + '[data-agentchat-composer]')}).dataset.state === "idle"`), 'the composer to go idle', 10000);

    try {
        await b.send('Page.navigate', {url: base + '/agentchat'});
        await eventually(() => js('!!(window.OrbitalChat && document.querySelector("#host-full [data-agentchat-composer]"))'), 'the chat page');
        await js('(() => { window.__events = []; for (const n of ["session", "step", "done", "error", "action", "send", "summarize", "starter"]) document.addEventListener("agentchat:" + n, e => window.__events.push([n, e.detail])); })()');

        await check('both roots bind, ship the contract markup, carry no global ids and run under script-src \'self\'', async () => {
            expect((await js('fetch(location.href).then(r => r.headers.get("content-security-policy"))')) === "script-src 'self'", 'CSP header missing');
            expect(await js('[...document.querySelectorAll("[data-agentchat]")].every(r => OrbitalChat.get(r) && OrbitalChat.get(r).busy === false) && document.querySelectorAll("[data-agentchat]").length === 2'), 'a root did not bind');
            const own = ['authorize', 'refuse', 'panel-body', 'panel-a', 'panel-b', 'rail-link', 'rail-link2']; // the stub's own markup
            expect(await js(`[...document.querySelectorAll("[data-agentchat] [id]")].map(e => e.id).filter(id => !${JSON.stringify(own)}.includes(id)).length`) === 0, 'a chat element carries a global id');
            expect(await q(F + '.ui-chat-thread', 'getComputedStyle(el).overflowY') === 'auto', 'the thread does not scroll');
            expect(await q(F + '[data-agentchat-messages]', 'el.getAttribute("role") + el.getAttribute("aria-live")') === 'logpolite', 'the thread is not a polite log');
            expect(await q(F + '[data-agentchat-input]', 'el.getAttribute("aria-label")') === 'Message', 'the composer textarea has no accessible name');
            expect(await text(F + '[data-agentchat-count]') === '0 / 200', 'character counter');
        });

        await check('an action card refuses inline (422 keeps the card and re-enables the button), then swaps to the receipt and queues a note', async () => {
            await click(F + '#refuse button');
            await eventually(() => q(F + '[data-agentchat-card-error]', 'el.textContent'), 'the inline error');
            expect(await text(F + '[data-agentchat-card-error]') === 'The proposal is already closed.', 'refusal text');
            expect(await js('[...document.querySelectorAll("#host-full #authorize button, #host-full #refuse button")].every(b => !b.disabled)'), 'buttons stayed disabled after an error');
            expect(await q(F + '#refuse', 'el.hasAttribute("aria-busy")') === false, 'the form stayed busy');
            await click(F + '#authorize button');
            await eventually(() => q(F + '#receipt', 'true'), 'the receipt');
            expect(await count(F + '#authorize') === 0 && await count(F + '[data-agentchat-card]') === 1, 'the card was not replaced by the receipt');
            expect(await text(F + '.ui-chat-note [data-agentchat-note-title]') === 'Payment authorised', 'note row');
            const ev = await js('window.__events.filter(e => e[0] === "action").map(e => e[1].note)');
            expect(ev.join() === 'Payment authorised', `agentchat:action notes ${ev}`);
        });

        await check('Enter sends: the reply streams in progressively, then done.html replaces it', async () => {
            await send(F, 'hello there');
            await eventually(() => count(F + '.ui-chat-msg--user'), 'the user row');
            expect(await qt(F + '.ui-chat-msg--user .ui-chat-bubble') === 'hello there', 'user bubble text');
            expect(await q(F + '[data-agentchat-input]', 'el.value') === '', 'the composer was not cleared');
            expect(await js('document.querySelector("#host-full [data-agentchat]").classList.contains("is-busy")'), 'root is not busy');
            const seen = [];
            let busy = null;
            await eventually(async () => {
                const t = await q(F + '.ui-chat-msg.is-pending [data-agentchat-body].is-provisional', 'el.textContent') || '';
                if (t && !seen.includes(t)) {
                    seen.push(t);
                    if (!busy) busy = await js(`(() => { const r = document.querySelector("#host-full .ui-chat-msg.is-pending"); const b = r.querySelector("[data-agentchat-body]"); return {busy: r.getAttribute("aria-busy"), hidden: b.getAttribute("aria-hidden"), caret: !!b.querySelector(".ui-caret"), stop: document.querySelector("#host-full [data-agentchat-send]").getAttribute("aria-label")}; })()`);
                }
                return t.includes('answer.');
            }, 'the streamed text', 10000);
            expect(seen.length >= 2 && seen[0].length < seen.at(-1).length, `text did not stream progressively: ${JSON.stringify(seen)}`);
            expect(seen.at(-1).replace('​', '') .startsWith('Here is your provisional answer.'), `provisional text: ${seen.at(-1)}`);
            expect(busy && busy.busy === 'true' && busy.hidden === 'true' && busy.caret && busy.stop === 'Stop', `streaming row state: ${JSON.stringify(busy)}`);
            await eventually(() => count(F + '.ui-chat-msg.is-pending').then(n => n === 0), 'the reply to settle', 10000);
            const reply = F + '.ui-chat-msg--assistant ';
            expect(await qt(reply + '[data-agentchat-body]') === 'Final server answer.', `final text: ${await qt(reply + '[data-agentchat-body]')}`);
            expect(await ql(reply + '[data-agentchat-body]', 'el.querySelectorAll("strong").length === 1 && !el.classList.contains("is-provisional") && !el.querySelector(".ui-caret") && !el.hasAttribute("aria-hidden")'), 'the server html did not replace the provisional text');
            expect(await ql(F + '.ui-chat-msg--assistant', '!el.hasAttribute("aria-busy")'), 'aria-busy stayed after done');
            await idle(F);
        });

        await check('the turn request carries session, composer fields, the transcript and the queued note', async () => {
            const req = (await state()).requests[0];
            expect(req.message[0] === 'hello there' && req.session_id[0] === '' && req.scope[0] === 'all', `fields: ${JSON.stringify(req)}`);
            expect(JSON.stringify(req.notes) === '["[\\"Payment authorised\\"]"]', `notes: ${JSON.stringify(req.notes)}`);
            const transcript = JSON.parse(req.transcript[0]);
            expect(transcript.length === 2 && transcript[0].role === 'assistant' && transcript[0].text === 'Welcome back.' && transcript[1].text.includes('Authorised'), `transcript: ${JSON.stringify(transcript)}`);
            const sent = await js('window.__events.find(e => e[0] === "send")[1]');
            expect(sent.body === undefined || typeof sent.body === 'object', 'agentchat:send detail');
        });

        await check('session and done events reach the app; the session id lands in the root, the card form and the address bar', async () => {
            expect(await js('document.querySelector("#host-full [data-agentchat]").dataset.sessionId') === 's-1', 'root session id');
            expect(await js('location.search') === '?session_id=s-1', 'address bar');
            expect(await q(F + 'input[name=session_id]', 'el.value') === 's-1', 'hidden session_id');
            const session = await js('window.__events.find(e => e[0] === "session")[1]');
            expect(session.session_id === 's-1' && session.data.owner === 'me', `session detail: ${JSON.stringify(session)}`);
            const done = await js('window.__events.find(e => e[0] === "done")[1]');
            expect(done.html.includes('<strong>server</strong>') && done.card_html.includes('tok-1'), 'done detail');
        });

        await check('steps fold into "4 steps" with a ×2 count and the narration as a caption', async () => {
            const steps = await js(`(() => { const row = [...document.querySelectorAll("#host-full .ui-chat-msg--assistant")].at(-1); const list = row.querySelector(".ui-chat-steps"); return {toggle: row.querySelector(".ui-chat-steps__toggle")?.textContent, hidden: list.hidden, expanded: row.querySelector(".ui-chat-steps__toggle").getAttribute("aria-expanded"), items: [...list.children].map(li => [li.querySelector("[data-agentchat-step-label]").textContent, li.querySelector(".ui-chat-step__count")?.textContent || "", li.querySelector("[data-agentchat-step-note]").textContent, li.className]) }; })()`);
            expect(steps.toggle === '4 steps' && steps.hidden && steps.expanded === 'false', `folded: ${JSON.stringify(steps)}`);
            expect(JSON.stringify(steps.items.map(i => i.slice(0, 3))) === JSON.stringify([['Reading the profile', '', ''], ['Checking listings', '×2', ''], ['Drafting the answer', '', 'Let me look at that.']]), `steps: ${JSON.stringify(steps.items)}`);
            expect(steps.items.every(i => i[3].includes('is-done')), 'a step is still active');
            await js('[...document.querySelectorAll("#host-full .ui-chat-steps__toggle")].at(-1).scrollIntoView()');
            const [x, y] = await js('(() => { const r = [...document.querySelectorAll("#host-full .ui-chat-steps__toggle")].at(-1).getBoundingClientRect(); return [r.left + r.width / 2, r.top + r.height / 2]; })()');
            for (const t of ['mousePressed', 'mouseReleased']) await b.send('Input.dispatchMouseEvent', {type: t, x, y, button: 'left', clickCount: 1});
            expect(await js('(() => { const t = [...document.querySelectorAll("#host-full .ui-chat-steps__toggle")].at(-1); return !t.nextElementSibling.hidden && t.getAttribute("aria-expanded") === "true"; })()'), 'the toggle did not open the list');
        });

        await check('question card submit sends q_* fields as a turn and collapses into an Answered row', async () => {
            expect(await q(F + '.ui-chat-card--questions input[name=q_mode]:checked', 'el.value') === 'Hybrid', 'prefilled chip');
            const id = await q(F + '.ui-chat-card--questions input[name=q_min]', 'el.id');
            expect(/^q-tok1-min$/.test(id), `field id ${id}`);
            await click(F + '.ui-chat-card--questions input[name=q_min]');
            await type('5000');
            await click(F + '.ui-chat-card--questions .ui-chat-card__head'); // blur: digits group
            expect(await q(F + '.ui-chat-card--questions input[name=q_min]', 'el.value') === '5,000', 'digit grouping on blur');
            const before = (await state()).requests.length;
            // The stub answers at once, so by the time this script looks the reply may have cleared the card.
            // agentchat:send fires after the collapse and before the request: sample the card there.
            await js(`document.addEventListener("agentchat:send", e => { const card = e.detail.form.closest("[data-agentchat-card]"); window.__atSend = {collapsed: card.classList.contains("is-answered") && card.inert, summary: document.querySelector("#host-full .ui-chat-answered [data-agentchat-summary]")?.textContent}; }, {once: true})`);
            await click(F + '.ui-chat-card--questions button[type=submit]:not([name])');
            await eventually(() => js('!!window.__atSend'), 'the turn to be sent');
            const atSend = await js('window.__atSend');
            expect(atSend.collapsed, 'the card was not collapsed when the turn was sent');
            expect(atSend.summary === 'Work arrangement: Hybrid · Minimum pay: 5,000', `summary: ${atSend.summary}`);
            await eventually(async () => (await state()).requests.length === before + 1, 'the answers request');
            const req = await lastRequest();
            expect(req.answer_questions[0] === '1' && req.question_token[0] === 'tok-1' && req.q_mode[0] === 'Hybrid' && req.q_min[0] === '5000' && req.session_id[0] === 's-1' && !req.message && req.scope.join() === 'all', `answers request: ${JSON.stringify(req)}`);
            await eventually(() => qt(F + '.ui-chat-msg--assistant [data-agentchat-body]').then(t => t === 'Thanks, I have your answers.'), 'the confirmation', 10000);
            expect(await count(F + '[data-agentchat-card-slot] > *') === 0, 'the answered card stayed in the slot');
            expect(await count(F + '.ui-chat-answered') === 1, 'the Answered row is gone');
            await idle(F);
        });

        await check('Stop aborts the reply, marks it Stopped and tells the server', async () => {
            await send(F, 'slow');
            await eventually(() => count(F + '.ui-chat-msg.is-pending .ui-chat-step'), 'the first step');
            await eventually(() => q(F + '.ui-chat-msg.is-pending [data-agentchat-body]:not([hidden])', 'true'), 'streamed text');
            expect(await q(F + '[data-agentchat-composer]', 'el.dataset.state') === 'busy', 'composer is not busy');
            await click(F + '[data-agentchat-send]');
            await idle(F);
            expect(await count(F + '.ui-chat-msg.is-pending') === 0, 'a pending row remained');
            expect(await qt(F + '.ui-chat-msg--assistant .ui-chat-stopped') === 'Stopped', 'no Stopped mark');
            expect(await qt(F + '.ui-chat-msg--assistant [data-agentchat-body]').then(t => t.startsWith('still thinking')), 'the partial text was dropped');
            await eventually(async () => (await state()).cancels === 1, 'the cancel request');
            expect(await q(F + '[data-agentchat-send]', 'el.getAttribute("aria-label")') === 'Send message', 'the Stop button stayed');
        });

        await check('a failed turn shows an error with Retry and restores the text; Retry sends it again', async () => {
            await send(F, 'fail');
            await eventually(() => q(F + '.ui-chat-error', 'true'), 'the error row');
            expect(await text(F + '.ui-chat-error [data-agentchat-error-text]') === "The agent couldn't finish that reply.", 'error text');
            expect(await q(F + '[data-agentchat-input]', 'el.value') === 'fail', 'the text was not restored');
            expect(await count(F + '.ui-chat-bubble') === 2, 'the failed message stayed in the thread'); // hello there and slow, not fail
            const before = (await state()).requests.length;
            await click(F + '[data-agentchat-retry]');
            await eventually(() => count(F + '.ui-chat-error').then(n => n === 0), 'the error to go');
            await eventually(async () => (await state()).requests.length === before + 1, 'the retried request');
            expect((await lastRequest()).message[0] === 'fail', 'retry sent another message');
            const sent = (await state()).requests.slice(-2).map(r => r.turn_id[0]);
            expect(/^[0-9a-f-]{36}$/.test(sent[0]) && sent[0] === sent[1], `Retry must reuse the turn id: ${sent}`);
            await eventually(() => qt(F + '.ui-chat-msg--assistant [data-agentchat-body]').then(t => t === 'Final server answer.'), 'the retried reply', 10000);
            expect(await q(F + '[data-agentchat-input]', 'el.value') === '', 'the restored text stayed in the composer after Retry');
            expect(await qt(F + '.ui-chat-msg--user .ui-chat-bubble') === 'fail', 'the user row after Retry');
            await idle(F);
        });

        await check('the composer notice goes once the user edits', async () => {
            expect(await text(C + '[data-agentchat-notice]') === 'Press Enter to send this', 'the notice is missing');
            expect(await count(F + '[data-agentchat-notice]') === 0, 'a composer without a Notice rendered one');
            await click(C + '[data-agentchat-input]');
            await type('a');
            expect(await count(C + '[data-agentchat-notice]') === 0, 'the notice stayed after an edit');
        });

        await check('an action starter raises agentchat:starter with its name and sends nothing', async () => {
            const before = (await state()).requests.length;
            await js('window.__events.length = 0');
            await click(C + '[data-agentchat-action="import"]');
            const ev = await js('window.__events.filter(e => e[0] === "starter").map(e => e[1].action)');
            expect(ev.join() === 'import', `starter events: ${ev}`);
            await sleep(300);
            expect((await state()).requests.length === before, 'an action starter sent a turn');
            expect(await count(C + '.ui-chat-msg--user') === 0 && await count(C + '[data-agentchat-empty]') === 1, 'an action starter changed the thread');
        });

        await check('a second root works alongside: its own session, no transcript, starters send and hide', async () => {
            await click(C + '[data-agentchat-prompt]');
            await eventually(() => count(C + '.ui-chat-msg--user'), 'the starter prompt as a user row');
            await eventually(() => qt(C + '.ui-chat-msg--assistant [data-agentchat-body]').then(t => t === 'Final server answer.'), 'the compact reply', 10000);
            const req = await lastRequest();
            expect(req.message[0] === 'hello' && req.session_id[0] === '' && !req.transcript && !req.notes, `compact request: ${JSON.stringify(req)}`);
            const ids = (await state()).requests.map(r => r.turn_id[0]);
            expect(ids.every(id => /^[0-9a-f-]{36}$/.test(id)) && new Set(ids).size === ids.length - 1, `every new turn needs its own id, only the Retry repeats one: ${ids}`);
            expect(req.csrf && req.csrf.join() === 'c1', `a starter turn lost the composer's hidden field: ${JSON.stringify(req)}`);
            expect(await count(C + '[data-agentchat-empty]') === 0, 'the empty state stayed');
            expect(await count(F + '.ui-chat-msg--user') === 3, 'the compact chat wrote into the full chat');
            expect(await q(C + '.ui-chat-composer', 'getComputedStyle(el).maxWidth') === 'none', 'the panel variant is not compact');
            await idle(C);
        });

        await check('the user bubble keeps Kindred\'s ink shadow', async () => {
            const sh = await ql(F + '.ui-chat-bubble', 'getComputedStyle(el).boxShadow');
            expect(sh.includes('rgba(13, 16, 23, 0.12)'), `bubble shadow: ${sh}`);
        });

        await check('the controller runs app work: note handle, hold, setCard and setSession', async () => {
            const root = '#host-full [data-agentchat]';
            const ctl = expr => js(`(() => { const c = OrbitalChat.get(document.querySelector(${JSON.stringify(root)})); return ${expr}; })()`);
            // notes
            await js(`window.__note = OrbitalChat.get(document.querySelector(${JSON.stringify(root)})).note("Reading cv.pdf…", "Pulling out roles"); window.__note.working(true)`);
            expect(await ql(F + '.ui-chat-note', 'el.classList.contains("is-working")'), 'working(true) did not mark the note');
            await js('window.__note.update("Profile imported", ""); window.__note.working(false)');
            expect(await qt(F + '.ui-chat-note [data-agentchat-note-title]') === 'Profile imported' && await ql(F + '.ui-chat-note', '!el.classList.contains("is-working") && el.querySelector("[data-agentchat-note-text]").hidden'), 'update/working(false)');
            const notes = await count(F + '.ui-chat-note');
            await js('window.__note.remove()');
            expect(await count(F + '.ui-chat-note') === notes - 1, 'remove() did not remove the note');
            // hold
            const before = (await state()).requests.length;
            await js(`window.__release = OrbitalChat.get(document.querySelector(${JSON.stringify(root)})).hold("Uploading…")`);
            expect(await ctl('c.busy') === true && await q(F + '[data-agentchat-composer]', 'el.dataset.state === "held" && el.inert'), 'hold() did not make the composer inert');
            expect(await q(F + '[data-agentchat-send]', 'el.getAttribute("aria-label")') === 'Send message', 'a held chat offers Stop');
            expect(await ctl('c.send("blocked")') === undefined, 'send() ran while held');
            await eventually(async () => await qt(F + '[data-agentchat-status] .ui-chat-status__text:not(.is-out)') === 'Uploading…', 'the held status');
            await js('window.__release(); window.__release()');
            expect(await ctl('c.busy') === false && await q(F + '[data-agentchat-composer]', 'el.dataset.state === "idle" && !el.inert'), 'release() did not free the composer');
            expect((await state()).requests.length === before, 'a held chat sent a turn');
            await ctl('c.send("via controller")');
            await eventually(async () => (await state()).requests.length === before + 1, 'the controller turn');
            expect((await lastRequest()).scope.join() === 'all', `controller.send lost the composer's hidden field: ${JSON.stringify(await lastRequest())}`);
            await idle(F);
            // card and session
            await ctl('c.setCard("<p id=\\"app-card\\">Hello</p>")');
            expect(await count(F + '[data-agentchat-card-slot] #app-card') === 1, 'setCard did not fill the slot');
            await ctl('c.setCard("")');
            expect(await count(F + '[data-agentchat-card-slot] > *') === 0, 'setCard("") did not clear the slot');
            const search = await js('location.search');
            await ctl('c.setSession("s-9")');
            expect(await js(`document.querySelector(${JSON.stringify(root)}).dataset.sessionId`) === 's-9' && await js('location.search') === search, 'setSession(id) must update the root only');
            await ctl('c.setSession("s-10", "/agentchat?session_id=s-10")');
            expect(await js('location.search') === '?session_id=s-10', 'setSession(id, url) did not update the address bar');
        });

        await check('the panel docks from 1280px and slides over below with a scrim; the rail docks from 1600px', async () => {
            const panel = F + '[data-agentchat-panel]';
            const rail = F + '[data-agentchat-rail]';
            const cs = (sel, prop) => q(sel, `getComputedStyle(el)[${JSON.stringify(prop)}]`);
            await b.resize(1280);
            await sleep(100);
            expect(await cs(panel, 'position') === 'relative' && await cs(panel, 'visibility') === 'visible' && await cs(F + '.ui-chat-toggle--panel', 'display') === 'none' && await cs(F + '[data-agentchat-scrim]', 'display') === 'none', 'the panel is not docked at 1280px');
            expect(await cs(rail, 'visibility') === 'hidden' && await cs(F + '.ui-chat-toggle--rail', 'display') !== 'none', 'the rail should slide over at 1280px');
            await b.resize(1000);
            await eventually(async () => await cs(panel, 'position') === 'absolute' && await cs(panel, 'visibility') === 'hidden' && await cs(F + '.ui-chat-toggle--panel', 'display') !== 'none', 'the panel to become a slide-over');
            await click(F + '.ui-chat-toggle--panel');
            await eventually(async () => await cs(panel, 'visibility') === 'visible' && await cs(F + '[data-agentchat-scrim]', 'opacity') === '1', 'the panel and scrim to show');
            expect(await q(F + '.ui-chat-toggle--panel', 'el.getAttribute("aria-expanded")') === 'true', 'aria-expanded');
            await key('Escape', 'Escape', 27);
            await eventually(async () => await cs(panel, 'visibility') === 'hidden', 'Escape to close the panel');
            await click(F + '.ui-chat-toggle--rail');
            await eventually(async () => await cs(rail, 'visibility') === 'visible', 'the rail to open');
            await click(F + '[data-agentchat-scrim]');
            await eventually(async () => await cs(rail, 'visibility') === 'hidden', 'the scrim to close the rail');
            await b.resize(1700);
            await sleep(100);
            expect(await cs(rail, 'position') === 'relative' && await cs(rail, 'visibility') === 'visible' && await cs(F + '.ui-chat-toggle--rail', 'display') === 'none', 'the rail is not docked at 1700px');
            await b.resize(1280);
        });

        await check('a slide-over is a modal dialog: focus moves in, Tab is trapped, Escape closes and focus returns; docked it is not', async () => {
            const panel = F + '[data-agentchat-panel]';
            const rail = F + '[data-agentchat-rail]';
            const active = () => js('document.activeElement && document.activeElement.id');
            const tab = (shift) => b.send('Input.dispatchKeyEvent', {type: 'rawKeyDown', key: 'Tab', code: 'Tab', windowsVirtualKeyCode: 9, modifiers: shift ? 8 : 0});
            await b.resize(1000);
            // The panel slides out from its docked place first; wait until it is parked so it cannot cover the toggle.
            await eventually(async () => await q(panel, 'getComputedStyle(el).visibility') === 'hidden' && await q(F + '.ui-chat-toggle--panel', 'getComputedStyle(el).display !== "none"'), 'the panel to park');
            expect(await q(panel, 'el.getAttribute("role")') === null, 'a closed panel is a dialog');
            await click(F + '.ui-chat-toggle--panel');
            await eventually(async () => await active() === 'panel-b', 'focus to move to the data-agentchat-autofocus element').catch(async e => { throw new Error(`${e.message}; focus is on ${await js('document.activeElement.outerHTML.slice(0, 80)')}, panel is ${await js('getComputedStyle(document.querySelector("#host-full [data-agentchat-panel]")).visibility + " " + document.querySelector("#host-full [data-agentchat]").className')}`); });
            expect(await q(panel, 'el.getAttribute("role") + el.getAttribute("aria-modal") + el.getAttribute("aria-label")') === 'dialogtrueDetails', 'dialog semantics');
            await js('document.getElementById("panel-b").focus()');
            await tab(false);
            expect(await active() === 'panel-a', `Tab from the last control went to ${await active()}`);
            await tab(true);
            expect(await active() === 'panel-b', `Shift+Tab from the first control went to ${await active()}`);
            await js('document.getElementById("authorize")?.querySelector("button")?.focus()');
            await tab(false);
            expect(await js('!!document.activeElement.closest("[data-agentchat-panel]")'), 'Tab escaped the dialog from outside');
            await key('Escape', 'Escape', 27);
            await eventually(async () => await q(panel, 'getComputedStyle(el).visibility') === 'hidden', 'Escape to close');
            expect(await js('document.activeElement.classList.contains("ui-chat-toggle--panel")'), 'focus did not return to the toggle');
            expect(await q(panel, 'el.getAttribute("role")') === null && await q(F + '.ui-chat-toggle--panel', 'el.getAttribute("aria-expanded")') === 'false', 'a closed panel kept its dialog role');
            await click(F + '.ui-chat-toggle--rail');
            await eventually(async () => await active() === 'rail-link', 'focus to move into the rail');
            expect(await q(rail, 'el.getAttribute("role") + el.getAttribute("aria-modal")') === 'dialogtrue', 'rail dialog semantics');
            await click(F + '[data-agentchat-scrim]');
            await eventually(() => js('document.activeElement.classList.contains("ui-chat-toggle--rail")'), 'focus to return to the rail toggle');
            await b.resize(1280);
            await eventually(async () => await q(panel, 'getComputedStyle(el).position') === 'relative', 'the panel to dock');
            expect(await q(panel, 'el.getAttribute("role")') === null && await q(panel, 'el.hasAttribute("aria-modal")') === false, 'a docked panel is a dialog');
        });

        await check('chat buttons carry no browser border or grey face (no reliance on a CSS preflight)', async () => {
            const bad = await js(`[...document.querySelectorAll(".ui-chat button")].filter(b => { const cs = getComputedStyle(b); return cs.borderTopStyle === "outset" || cs.borderTopStyle === "solid" && parseFloat(cs.borderTopWidth) >= 2 || cs.backgroundColor === "rgb(239, 239, 239)" || cs.backgroundColor === "rgb(233, 233, 237)"; }).map(b => b.className || b.outerHTML.slice(0, 60))`);
            expect(bad.length === 0, `default-styled buttons: ${bad.join(', ')}`);
            const kinds = await js('new Set([...document.querySelectorAll(".ui-chat button")].map(b => b.className.split(" ")[0])).size');
            expect(kinds >= 5, `only ${kinds} kinds of button were checked`);
        });

        await check('a page rendered mid-turn polls the state endpoint and swaps bare rows into the existing log', async () => {
            await b.send('Page.navigate', {url: base + '/agentchat/resume'});
            await eventually(() => js('!!document.querySelector("[data-agentchat-composer]") && !!window.OrbitalChat'), 'the resume page');
            expect(await js('!!document.querySelector("[data-agentchat-pending]")') && await q('[data-agentchat-composer]', 'el.dataset.state') === 'busy', 'no pending row or busy composer');
            expect(await q('.ui-chat-msg.is-pending', 'el.getAttribute("aria-busy")') === 'true', 'the pending row is not aria-busy');
            await eventually(() => count('[data-agentchat-pending]').then(n => n === 0), 'the turn to land', 15000);
            expect(await count('[role=log]') === 1 && await count('[data-agentchat-messages] > .ui-chat-msg') === 2, 'the rows did not land inside the one existing log');
            expect(await qt('[data-agentchat-messages] .ui-chat-msg--assistant [data-agentchat-body]') === 'Resumed answer.', 'resumed reply');
            const ctx = await js('(() => { const chip = document.querySelector("#from-chip").getBoundingClientRect(), bubble = document.querySelector(".ui-chat-bubble").getBoundingClientRect(); return {above: chip.bottom <= bubble.top + 1, right: Math.abs(chip.right - bubble.right) < 2}; })()');
            expect(ctx.above && ctx.right, `the user context is not a right-aligned row above the bubble: ${JSON.stringify(ctx)}`);
            expect(await count('[data-agentchat-card-slot] form[data-agentchat-submit=turn]') === 1, 'the card did not land');
            expect(await q('[data-agentchat-composer]', 'el.dataset.state') === 'idle', 'the composer stayed busy');
            const polls = (await state()).resumes;
            expect(polls.length === 2 && polls.every(id => id === 's-r'), `state polls: ${JSON.stringify(polls)}`);
        });

        await check('no script errors or CSP violations on the chat page', async () => {
            expect(b.errors.length === 0, b.errors.join('\n'));
        });
    } finally {
        await b.close();
    }
}

try {
    await runVariant('standard', '/');
    await runVariant('csp', '/?csp=1');
    await runVariant('host', '/?host=1', {full: false, host: true});
    await runAgentChat();
} finally {
    await stop();
}

const failed = results.filter(([s]) => s === 'FAIL').length;
console.log(`\n${results.length - failed}/${results.length} browser checks passed`);
process.exit(failed ? 1 : 0);
