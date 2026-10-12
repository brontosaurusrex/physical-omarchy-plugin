#!/usr/bin/env python3
"""Exercise real Wayland capture through the game's right-click handler."""
import json, os, pathlib, shutil, subprocess, tempfile, threading

project = pathlib.Path(__file__).resolve().parents[1]
shell = pathlib.Path('/usr/share/omarchy/shell')
harness = r'''
import QtQuick
import QtTest
import Quickshell
import qs.Commons
import "Port" as Port
ShellRoot {
    id: root
    property int phase: 0
    property int attempts: 0
    Port.Physical { id: game }
    TestCase { id: input; parent: game.contentView; when: false }
    function find(item) {
        if (item.objectName === "physicalPointerInput") return item;
        for (var child of item.children || []) { var found = find(child); if (found) return found; }
        return null;
    }
    function click() { var area = find(game.contentView); input.mouseClick(area, area.width / 2, area.height / 2, Qt.RightButton); }
    function fail(message) { console.error("POINTER FAILED", message, "requested", game.mouseCapture.requested, "locked", game.mouseLocked, "notice", game.notice); Qt.exit(1); }
    Component.onCompleted: game.open("{}")
    Timer { interval: 300; running: true; repeat: true; onTriggered: {
        if (game.errorText || game.renderError) { fail(game.errorText || game.renderError); return; }
        if (++root.attempts > 50) { fail("timeout at phase " + root.phase); return; }
        if (!game.engineReady || game.paintedFrames < 2) return;
        if (root.phase === 0) { console.log("POINTER READY"); root.phase = 1; return; }
        if (root.phase === 1) { click(); root.phase = 2; return; }
        if (root.phase === 2) {
            if (!game.mouseLocked) { console.log("POINTER waiting",game.mouseCapture.supported,game.mouseCapture.requested,game.notice); return; }
            console.log("POINTER right click captured"); click(); root.phase = 3; return;
        }
        if (root.phase === 3) {
            if (game.mouseCapture.requested || game.mouseLocked) { fail("second right click did not release"); return; }
            click(); root.phase = 4; return;
        }
        if (root.phase === 4) {
            if (!game.mouseLocked) return;
            input.keyClick(Qt.Key_Escape); root.phase = 5; return;
        }
        if (root.phase === 5) {
            if (game.opened || game.windowView.visible || game.mouseCapture.requested) { fail("Escape did not close and release"); return; }
            game.open("{}"); root.phase = 6; return;
        }
        if (root.phase === 6) { click(); root.phase = 7; return; }
        if (root.phase === 7) {
            if (!game.mouseLocked) return;
            game.toggleMenu();
            if (game.mouseCapture.requested) { fail("menu did not release"); return; }
            input.keyClick(Qt.Key_Escape); root.phase = 8; return;
        }
        if (root.phase === 8) {
            if (game.opened) { fail("Escape in menu did not close"); return; }
            game.open("{}"); root.phase = 9; return;
        }
        if (root.phase === 9) { game.toggleFullscreen(); root.phase = 10; return; }
        if (root.phase === 10) { input.keyClick(Qt.Key_Escape); root.phase = 11; return; }
        if (root.phase === 11) {
            if (game.opened || game.fullscreen || game.mouseCapture.requested) { fail("Escape in fullscreen did not close"); return; }
            console.log("POINTER PASS: right click lock/release, reacquire, Escape, menu, fullscreen, reopen"); Qt.exit(0);
        }
    } }
}
'''
if not os.environ.get('WAYLAND_DISPLAY'):
    raise SystemExit('This test requires the actual Wayland desktop.')
with tempfile.TemporaryDirectory(prefix='physical-pointer-test-') as directory:
    folder = pathlib.Path(directory)
    for name in ['Commons', 'Ui']:
        shutil.copytree(shell / name, folder / name)
    shutil.copytree(project, folder / 'Port', ignore=shutil.ignore_patterns('.git', '__pycache__', 'dist'))
    (folder / 'Port/native/qmldir').write_text('plugin physicalrenderer ' + str(folder / 'Port/native') + '\n')
    (folder / 'shell.qml').write_text(harness)
    env = dict(os.environ, PHYSICAL_TESTING='1', PHYSICAL_STATE_PATH=str(folder / 'progress.json'))
    cursor = json.loads(subprocess.check_output(['hyprctl', 'cursorpos', '-j']))
    run = subprocess.Popen(['quickshell', '--no-color', '-p', str(folder / 'shell.qml')], env=env, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    timeout = threading.Timer(25, run.kill); timeout.start()
    lines = []
    try:
        for line in run.stdout:
            lines.append(line)
            if 'POINTER' in line or 'ERROR' in line: print(line.strip(), flush=True)
            if 'POINTER READY' in line:
                clients = json.loads(subprocess.check_output(['hyprctl', 'clients', '-j']))
                client = next(c for c in clients if c['pid'] == run.pid and c['title'] == 'Physical')
                x, y = [client['at'][i] + client['size'][i] // 2 for i in (0, 1)]
                code = 'hl.dispatch(hl.dsp.focus({window=' + json.dumps('address:' + client['address']) + '})); hl.dispatch(hl.dsp.cursor.move({x=' + str(x) + ',y=' + str(y) + '}))'
                subprocess.run(['hyprctl', 'eval', code], check=True, stdout=subprocess.DEVNULL)
        run.wait()
    finally:
        timeout.cancel()
        if run.poll() is None: run.kill(); run.wait()
        subprocess.run(['hyprctl', 'eval', 'hl.dispatch(hl.dsp.cursor.move({x=' + str(cursor['x']) + ',y=' + str(cursor['y']) + '}))'], stdout=subprocess.DEVNULL)
    if run.returncode or not any('POINTER PASS:' in line for line in lines):
        print(''.join(lines)); raise SystemExit(run.returncode or 1)
