#!/usr/bin/env python3
"""Exercise real Wayland capture through the game's right-click handler."""
import json, os, pathlib, shutil, subprocess, tempfile, threading

project = pathlib.Path(__file__).resolve().parents[1]
shell = pathlib.Path('/usr/share/omarchy/shell')
harness = r'''
import QtQuick
import QtQuick.Window
import QtTest
import Quickshell
import Quickshell.Wayland
import qs.Commons
import "Port" as Port
ShellRoot {
    id: root
    property bool popup: false
    property int phase: 0
    property int attempts: 0
    property int resizeStep: 0
    property double resizeStarted: 0
    readonly property var resizeModes: ["fullscreen", "fullscreen", "maximized", "maximized", "native", "native"]
    Port.Physical { id: game }
    PanelWindow {
        visible: root.popup
        anchors { top: true; bottom: true; left: true; right: true }
        exclusionMode: ExclusionMode.Ignore
        WlrLayershell.namespace: "physical-pointer-notification-test"
        WlrLayershell.layer: WlrLayer.Overlay
        WlrLayershell.keyboardFocus: WlrKeyboardFocus.None
        color: "transparent"
        mask: Region { item: toast }
        Rectangle {
            id: toast
            anchors.top: parent.top; anchors.right: parent.right; anchors.margins: 30
            width: 360; height: 110; color: "#23344a"
        }
    }
    TestCase { id: input; parent: game.contentView; when: false }
    function find(item) {
        if (item.objectName === "physicalPointerInput") return item;
        for (var child of item.children || []) { var found = find(child); if (found) return found; }
        return null;
    }
    function click() { var area = find(game.contentView); input.mouseClick(area, area.width / 2, area.height / 2, Qt.RightButton); }
    function fail(message) { console.error("POINTER FAILED", message, "requested", game.mouseCapture.requested, "locked", game.mouseLocked, "notice", game.notice, "windowActive", game.contentView.Window.window.active); Qt.exit(1); }
    Component.onCompleted: game.open("{}")
    Timer { interval: 300; running: true; repeat: true; onTriggered: {
        if (game.errorText || game.renderError) { fail(game.errorText || game.renderError); return; }
        if (++root.attempts > 110) { fail("timeout at phase " + root.phase); return; }
        if (!game.engineReady || game.paintedFrames < 2) return;
        if (root.phase === 0) { console.log("POINTER READY"); root.phase = 1; return; }
        if (root.phase === 1) { click(); root.phase = 2; return; }
        if (root.phase === 2) {
            if (!game.mouseLocked) { console.log("POINTER waiting",game.mouseCapture.supported,game.mouseCapture.requested,game.notice); return; }
            console.log("POINTER right click captured"); root.popup = true; root.phase = 25; return;
        }
        if (root.phase === 25) {
            if (!game.mouseLocked || !game.mouseCapture.requested || game.state.paused) { fail("notification interrupted capture"); return; }
            root.popup = false; root.phase = 26; return;
        }
        if (root.phase === 26) {
            if (!game.mouseLocked) { fail("capture lost after notification disappeared"); return; }
            console.log("POINTER capture survived passive notification"); click(); root.phase = 3; return;
        }
        if (root.phase === 3) {
            if (game.mouseCapture.requested || game.mouseLocked) { fail("second right click did not release"); return; }
            click(); root.phase = 4; return;
        }
        if (root.phase === 4) {
            if (!game.mouseLocked) return;
            root.phase = 12; return;
        }
        if (root.phase === 12) {
            if (root.resizeStep === root.resizeModes.length) {
                // Resize must not leave a stale request that prevents release.
                click(); root.phase = 14; return;
            }
            var mode = root.resizeModes[root.resizeStep];
            root.resizeStarted = Date.now(); root.phase = 13;
            if (mode === "native") game.toggleFullscreen();
            else console.log("POINTER COMPOSITOR", mode);
            return;
        }
        if (root.phase === 13 && Date.now() - root.resizeStarted > 600) {
            if (!game.mouseCapture.requested || !game.mouseLocked) { fail("capture lost after " + root.resizeModes[root.resizeStep] + " transition " + root.resizeStep); return; }
            console.log("POINTER capture survived", root.resizeModes[root.resizeStep], root.resizeStep);
            ++root.resizeStep; root.phase = 12; return;
        }
        if (root.phase === 14) {
            if (game.mouseCapture.requested || game.mouseLocked) { fail("right click after resize did not release"); return; }
            click(); root.phase = 15; return;
        }
        if (root.phase === 15) {
            if (!game.mouseLocked) return;
            console.log("POINTER FOCUS LOSS"); root.phase = 16; return;
        }
        if (root.phase === 16) {
            if (game.mouseCapture.requested || game.mouseLocked) { fail("focus loss did not release"); return; }
            console.log("POINTER FOCUS RETURN"); root.phase = 17; return;
        }
        if (root.phase === 17) {
            // Returning focus must not capture again until another right click.
            if (game.mouseCapture.requested || game.mouseLocked) { fail("returning focus recaptured mouse"); return; }
            click(); root.phase = 18; return;
        }
        if (root.phase === 18) {
            if (!game.mouseLocked) return;
            input.keyClick(Qt.Key_Escape); root.phase = 5; return;
        }
        if (root.phase === 5) {
            if (game.opened || game.windowView.visible || game.mouseCapture.requested) { fail("Escape did not close and release"); return; }
            game.open("{}"); root.phase = 6; return;
        }
        if (root.phase === 6) {
            // QtTest clicks do not move the real compositor pointer or focus a
            // remapped Wayland surface. Restore those prerequisites first.
            console.log("POINTER REOPEN READY"); root.phase = 27; return;
        }
        if (root.phase === 27) {
            if (!game.contentView.Window.window.active) { fail("test window did not regain focus"); return; }
            click(); root.phase = 7; return;
        }
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
            console.log("POINTER PASS: right click lock/release, passive notification, compositor fullscreen/maximize, native fullscreen, focus loss, reacquire, Escape, menu, reopen"); Qt.exit(0);
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
    previous = json.loads(subprocess.check_output(['hyprctl', 'activewindow', '-j']))
    run = subprocess.Popen(['quickshell', '--no-color', '-p', str(folder / 'shell.qml')], env=env, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    timeout = threading.Timer(40, run.kill); timeout.start()
    lines = []
    try:
        for line in run.stdout:
            lines.append(line)
            if 'POINTER' in line or 'ERROR' in line: print(line.strip(), flush=True)
            if 'POINTER READY' in line or 'POINTER REOPEN READY' in line:
                clients = json.loads(subprocess.check_output(['hyprctl', 'clients', '-j']))
                client = next(c for c in clients if c['pid'] == run.pid and c['title'] == 'Physical')
                offset = (client['size'][0]-50, 50) if 'POINTER READY' in line else (client['size'][0]//2, client['size'][1]//2)
                x, y = [client['at'][i] + offset[i] for i in (0, 1)]
                code = 'hl.dispatch(hl.dsp.focus({window=' + json.dumps('address:' + client['address']) + '})); hl.dispatch(hl.dsp.cursor.move({x=' + str(x) + ',y=' + str(y) + '}))'
                subprocess.run(['hyprctl', 'eval', code], check=True, stdout=subprocess.DEVNULL)
            if 'POINTER FOCUS LOSS' in line:
                code = 'hl.dispatch(hl.dsp.focus({window=' + json.dumps('address:' + previous['address']) + '}))'
                subprocess.run(['hyprctl', 'eval', code], check=True, stdout=subprocess.DEVNULL)
            if 'POINTER FOCUS RETURN' in line:
                code = 'hl.dispatch(hl.dsp.focus({window=' + json.dumps('address:' + client['address']) + '}))'
                subprocess.run(['hyprctl', 'eval', code], check=True, stdout=subprocess.DEVNULL)
            if 'POINTER COMPOSITOR ' in line:
                mode = line.split('POINTER COMPOSITOR ', 1)[1].strip()
                assert mode in ('fullscreen', 'maximized'), mode
                # Use the exact dispatchers bound to Super+F and Super+Alt+F.
                code = 'hl.dispatch(hl.dsp.focus({window=' + json.dumps('address:' + client['address']) + '})); hl.dispatch(hl.dsp.window.fullscreen({mode=' + json.dumps(mode) + '}))'
                subprocess.run(['hyprctl', 'eval', code], check=True, stdout=subprocess.DEVNULL)
        run.wait()
    finally:
        timeout.cancel()
        if run.poll() is None: run.kill(); run.wait()
        subprocess.run(['hyprctl', 'eval', 'hl.dispatch(hl.dsp.cursor.move({x=' + str(cursor['x']) + ',y=' + str(cursor['y']) + '}))'], stdout=subprocess.DEVNULL)
        if previous.get('address'):
            subprocess.run(['hyprctl', 'eval', 'hl.dispatch(hl.dsp.focus({window=' + json.dumps('address:' + previous['address']) + '}))'], stdout=subprocess.DEVNULL)
    if run.returncode or not any('POINTER PASS:' in line for line in lines):
        print(''.join(lines)); raise SystemExit(run.returncode or 1)
