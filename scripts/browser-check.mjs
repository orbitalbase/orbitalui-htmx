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

// Runs the checks for one page variant in a fresh Chrome. `full` selects the
// whole suite; the host variant runs only the delegation checks.
async function runVariant(name, path, {full = true, host = false} = {}) {
    let b;
    try {
        b = await startChrome().catch(e => {
            console.error(`Chrome did not start (${e.message}); retrying once`);
            return startChrome();
        });
    } catch (e) {
        console.error(`could not start Chrome (${chromePath()}): ${e.message}`);
        process.exit(1);
    }
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
    const focused = () => js('document.activeElement && (document.activeElement.id || document.activeElement.className)');
    // Wait until htmx has finished swapping and settling, so freshly swapped
    // markup is processed before the next interaction.
    const htmxIdle = () => eventually(() => js('!document.querySelector(".htmx-request, .htmx-swapping, .htmx-settling, .htmx-added")'), 'htmx to settle');
    // Wait for the drawer's slide-in to finish so pointer targets stop moving.
    const drawerSettled = () => eventually(() => js('["matrix(1,0,0,1,0,0)", "none"].includes(getComputedStyle(document.querySelector("#drw .drawer-panel")).transform.replace(/\\s/g, ""))'), 'drawer to settle');
    const check = async (label, fn) => {
        const full = `[${name}] ${label}`;
        try { await fn(); results.push(['ok', full]); console.log(`ok   ${full}`); }
        catch (e) { results.push(['FAIL', full]); console.log(`FAIL ${full}\n     ${e.message}`); }
    };
    const expect = (cond, msg) => { if (!cond) throw new Error(msg); };

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

        await check('no script errors or CSP violations on the page', async () => {
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
} finally {
    await stop();
}

const failed = results.filter(([s]) => s === 'FAIL').length;
console.log(`\n${results.length - failed}/${results.length} browser checks passed`);
process.exit(failed ? 1 : 0);
