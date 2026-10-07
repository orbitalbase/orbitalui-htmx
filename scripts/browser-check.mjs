// Browser check for the interactive components: drives ModalShell,
// ConfirmButton, Drawer and EntityPicker through orbital-ui.js, Alpine and
// htmx in headless Chrome with real pointer and keyboard input.
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
    await send('Page.enable');
    await send('Emulation.setDeviceMetricsOverride', {width: 1280, height: 900, deviceScaleFactor: 1, mobile: false});
    const close = async () => { try { ws.close(); } catch {} chrome.kill(); await rm(profile, {recursive: true, force: true}).catch(() => {}); };
    return {send, errors, close};
}

const {stop, base} = await startServer();
let b;
try {
    b = await startChrome();
} catch (e) {
    await stop();
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
const drawerSettled = () => eventually(() => js('getComputedStyle(document.querySelector("#drw .drawer-panel")).transform.replace(/\\s/g, "") === "matrix(1,0,0,1,0,0)"'), 'drawer to settle');
const checks = [];
const check = async (name, fn) => {
    try { await fn(); checks.push(['ok', name]); console.log(`ok   ${name}`); }
    catch (e) { checks.push(['FAIL', name]); console.log(`FAIL ${name}\n     ${e.message}`); }
};
const expect = (cond, msg) => { if (!cond) throw new Error(msg); };

try {
    await b.send('Page.navigate', {url: base});
    await eventually(() => js('!!(window.Alpine && window.htmx && document.querySelector(".entity-picker")?._x_dataStack)'), 'Alpine, htmx and the picker to start');

    await check('runtime installs OrbitalUI and the LedgerUI alias', async () => {
        expect(await js('typeof OrbitalUI.openDialog === "function" && window.LedgerUI === window.OrbitalUI'), 'runtime not installed');
    });

    await check('dialog opens modally and focuses its first field', async () => {
        await click('#open-dialog');
        await eventually(() => js('document.getElementById("dlg").matches(":modal")'), 'modal dialog');
        expect(await focused() === 'dlg-name', `focus is on ${await focused()}`);
    });

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

    await check('a backdrop click closes the dialog; a click inside does not', async () => {
        await click('#open-dialog');
        await eventually(() => js('document.getElementById("dlg").open'), 'dialog to open');
        await click('#dlg .ui-dialog__body');
        expect(await js('document.getElementById("dlg").open'), 'inside click closed the dialog');
        await clickAt(8, 8);
        await eventually(() => js('!document.getElementById("dlg").open'), 'dialog to close');
    });

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

    await check('fullscreen toggle updates class, pressed state and label', async () => {
        await drawerSettled();
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

    await check('picker lists options on focus and picks with the keyboard', async () => {
        await click('#customer');
        await eventually(() => js('document.querySelectorAll("#customer-list [role=option]").length === 3'), 'three options')
            .catch(async e => { throw new Error(`${e.message}; state=${await js('JSON.stringify((({open, busy, failed, items, query}) => ({open, busy, failed, items: items.length, query}))(document.querySelector("#picker-form .entity-picker")._x_dataStack[0])) + " focus=" + (document.activeElement && document.activeElement.id)')}`); });
        expect(await js('document.getElementById("customer").getAttribute("aria-expanded")') === 'true', 'aria-expanded not true');
        await type('nor');
        await eventually(() => js('document.querySelector("#customer-list [role=option]")?.textContent.includes("Northstar")'), 'filtered options');
        await key('Enter', 'Enter', 13);
        await eventually(() => js('document.querySelector("#picker-form input[name=customer]").value === "c2"'), 'picked value c2');
        expect(await js('document.getElementById("customer").value') === 'Northstar Technologies', 'input label not set');
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

    await check('no script errors on the page', async () => {
        expect(b.errors.length === 0, b.errors.join('\n'));
    });
} finally {
    await b.close();
    await stop();
}

const failed = checks.filter(([s]) => s === 'FAIL').length;
console.log(`\n${checks.length - failed}/${checks.length} browser checks passed`);
process.exit(failed ? 1 : 0);
