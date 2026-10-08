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
    const port = await eventually(async () => (await readFile(join(profile, 'DevToolsActivePort'), 'utf8')).split('\n')[0], 'Chrome DevTools port', 15000);
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
        b = await startChrome();
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
                await click('#load-swap');
                await eventually(() => js('!!document.querySelector("#swapped .entity-picker")?._x_dataStack'), 'Alpine on swapped picker');
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
            expect(await js('document.querySelector("[data-oui-theme-switch] input:checked").value') === 'system', 'switch does not show system');
            await b.send('Emulation.setEmulatedMedia', {features: [{name: 'prefers-color-scheme', value: 'light'}]});
            await eventually(() => js('document.documentElement.dataset.theme === "light"'), 'live follow of the system change');
        });

        await check('the theme switch applies, persists and survives a reload without a flash', async () => {
            await click('[data-oui-theme-switch] input[value=dark]');
            await eventually(() => js('document.documentElement.dataset.theme === "dark"'), 'dark theme applied');
            expect(await js('localStorage.getItem("oui-theme")') === 'dark', 'preference not persisted');
            expect(await js('getComputedStyle(document.querySelector(".ui-shell")).backgroundColor') !== await js('(() => { const el = document.createElement("div"); el.style.background = "#f6f7f9"; document.body.append(el); const c = getComputedStyle(el).backgroundColor; el.remove(); return c; })()'), 'the canvas did not change colour');
            await openShell();
            expect(await js('window.__themeAtBody') === 'dark', `reload painted ${await js('window.__themeAtBody')} first`);
            expect(await js('document.querySelector("[data-oui-theme-switch] input:checked").value') === 'dark', 'switch does not reflect the stored preference');
            await click('[data-oui-theme-switch] input[value=light]');
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
