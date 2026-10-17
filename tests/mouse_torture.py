#!/usr/bin/env python3
"""Sustained real-Wayland pointer lifecycle test with an external stall watchdog.

PHYSICAL_MOUSE_SECONDS=300 python3 tests/mouse_torture.py
PHYSICAL_MOUSE_INJECT_FREEZE=ui|engine verifies the watchdog, then exits successfully.
Uses private progress, a temporary collision-free level, and restores desktop focus.
"""
import json
import os
import pathlib
import queue
import shutil
import signal
import select
import shlex
import subprocess
import tempfile
import threading
import time

project = pathlib.Path(__file__).resolve().parents[1]
shell = pathlib.Path(os.environ.get('OMARCHY_SHELL_DIR', '/usr/share/omarchy/shell'))
seconds = int(os.environ.get('PHYSICAL_MOUSE_SECONDS', '300'))
freeze = os.environ.get('PHYSICAL_MOUSE_INJECT_FREEZE', '')
if not 10 <= seconds <= 3600 or freeze not in ('', 'ui', 'engine'):
    raise SystemExit('Duration must be 10–3600 seconds; freeze must be ui or engine.')
if not os.environ.get('WAYLAND_DISPLAY'):
    raise SystemExit('This test needs the actual Wayland/Hyprland desktop.')
level = '''gravity=0
airDrag=0
stuckSpeedThreshold=0
ballRescueEnabled=false
ballX=900
ballY=450
ballVx=60
ballVy=0
brickWidth=100
brickHeight=40
brickOffsetTop=50
title="MOUSE TEST"
---
#
'''
harness = r'''
import QtQuick
import QtQuick.Window
import QtTest
import Quickshell
import Quickshell.Wayland
import "Port" as Port
ShellRoot {
    id: root
    property int phase: 0
    property int cycles: 0
    onCyclesChanged: {
        if (cycles >= 1 && Quickshell.env("PHYSICAL_MOUSE_INJECT_FREEZE"))
            game.windowView.title = Quickshell.env("PHYSICAL_MOUSE_DIALOG_TITLE");
    }
    property real relativeStart: 0
    property int motionEvents: 0
    property int assertions: 0
    property bool wasLocked: false
    property int unlocks: 0
    property int recoveries: 0
    property double started: 0
    property double phaseStarted: Date.now()
    property bool popup: false
    Port.Physical { id: game }
    TestCase { id: input; parent: game.contentView; when: false }
    // Match Omarchy's notification surface: a full-screen passive overlay with
    // an input region only over the top-right card, in the SAME Qt process.
    PanelWindow {
        visible: root.popup
        anchors { top: true; bottom: true; left: true; right: true }
        exclusionMode: ExclusionMode.Ignore
        WlrLayershell.namespace: "physical-mouse-torture"
        WlrLayershell.layer: WlrLayer.Overlay
        WlrLayershell.keyboardFocus: WlrKeyboardFocus.None
        color: "transparent"
        mask: Region { item: toast }
        Rectangle {
            id: toast
            anchors.top: parent.top; anchors.right: parent.right; anchors.margins: 30
            width: 360; height: 110; color: "#23344a"
            Text { anchors.centerIn: parent; text: "Physical mouse test"; color: "white" }
        }
    }
    Connections {
        target: game.mouseCapture
        function onLockedChanged() {
            if (game.mouseLocked && !root.wasLocked) root.recoveries++;
            else if (!game.mouseLocked && root.wasLocked) root.unlocks++;
            root.wasLocked = game.mouseLocked;
        }
    }
    function fail(message) {
        console.error("MOUSE FAILED", message, "phase",phase,"cycle",cycles,
                      "requested",game.mouseCapture.requested,"locked",game.mouseLocked);
        Qt.exit(1);
    }
    function require(condition, message) {
        assertions++;
        if (!condition) { fail(message); return false; }
        return true;
    }
    function next(value) { phase = value; phaseStarted = Date.now(); }
    function find(item) {
        if (item.objectName === "physicalPointerInput") return item;
        for (var child of item.children || []) { var found = find(child); if (found) return found; }
        return null;
    }
    function click() {
        var area = find(game.contentView);
        input.mouseClick(area, area.width/2, area.height/2, Qt.RightButton);
    }
    function locked() {
        if (game.mouseLocked && game.mouseCapture.requested) { assertions++; return true; }
        if (Date.now()-phaseStarted > 1500) fail("capture did not recover");
        return false;
    }
    function moved(condition, message) {
        if (condition) { assertions++; return true; }
        if (Date.now()-phaseStarted > 1500) fail(message);
        return false;
    }
    function free() { return require(!game.mouseCapture.requested && !game.mouseLocked, "unexpected capture"); }
    function target(x) {
        return require(Math.abs(game.state.probeTarget-x) < 0.1 && Math.abs(game.state.paddleX-x) < 0.1,
                       "absolute input stalled: target " + game.state.probeTarget + ", paddle " + game.state.paddleX + " expected " + x);
    }
    function move(x,y) {
        // Burst through the window and its margins before the final target.
        // The UI should coalesce motion without losing the last position.
        for (var i=0; i<64; ++i) {
            input.mouseMove(game.contentView, 1+(game.contentView.width-2)*i/63,
                            game.contentView.height/2, 0);
            motionEvents++;
        }
        input.mouseMove(game.contentView, x, y, 0); motionEvents++;
    }
    function moveInGame(fraction) {
        var p = game.gameView.mapToItem(game.contentView, game.gameView.width*fraction, game.gameView.height/2);
        move(p.x,p.y);
    }
    Component.onCompleted: game.open("{}")
    Timer { interval: 1000; repeat: true; running: true; onTriggered: {
        console.log("MOUSE HEARTBEAT", JSON.stringify({phase:root.phase, cycles:root.cycles,
                    replies:game.frameReplies, frames:game.paintedFrames, assertions:root.assertions}));
    } }
    Timer { interval: 250; repeat: true; running: true; onTriggered: {
        if (game.errorText || game.renderError) { fail(game.errorText || game.renderError); return; }
        if (!game.engineReady || game.paintedFrames < 2) return;
        if (cycles >= 1 && Quickshell.env("PHYSICAL_MOUSE_INJECT_FREEZE")) return;
        if (Date.now()-phaseStarted > 4000) { fail("phase timed out"); return; }
        switch (root.phase) {
        case 0:
            started = Date.now(); game.send({action:"volume",value:0});
            console.log("MOUSE READY"); next(1); break;
        case 1:
            if (game.state.ready || game.state.paused) game.shortcut(" ");
            moveInGame(0.25); next(2); break;
        case 2:
            if (!free() || !target(game.gameView.gameWidth*0.25-game.state.probePaddleWidth/2)) return;
            move(1,game.contentView.height/2); next(3); break;
        case 3:
            if (!free() || !target(0)) return;
            move(game.contentView.width-1,game.contentView.height/2); next(4); break;
        case 4:
            if (!free() || !target(game.gameView.gameWidth-game.state.probePaddleWidth)) return;
            console.log("MOUSE OUTSIDE"); next(5); break;
        case 5:
            if (!free() || !target(game.gameView.gameWidth-game.state.probePaddleWidth)) return;
            console.log("MOUSE ENTER"); next(6); break;
        case 6:
            if (!free()) return;
            if (game.state.paused) game.shortcut(" ");
            popup = true; next(25); break;
        case 25:
            if (!free()) return;
            popup = false; next(26); break;
        case 26:
            if (!free()) return;
            click(); next(7); break;
        case 7:
            if (!locked()) return;
            relativeStart = game.state.paddleX;
            console.log("MOUSE RELATIVE -80"); next(27); break;
        case 27:
            if (!locked() || !moved(game.state.paddleX < relativeStart-1,"captured relative motion did not move paddle left")) return;
            relativeStart = game.state.paddleX;
            console.log("MOUSE RELATIVE 40"); next(28); break;
        case 28:
            if (!locked() || !moved(game.state.paddleX > relativeStart+1,"captured relative motion did not move paddle right: start="+relativeStart+" now="+game.state.paddleX+" paused="+game.state.paused+" ready="+game.state.ready)) return;
            popup = true; next(8); break;
        case 8:
            if (!locked()) return;
            if (!require(game.contentView.Window.window.active && !game.state.paused,"passive popup stole keyboard focus or paused game")) return;
            relativeStart = game.state.paddleX;
            console.log("MOUSE RELATIVE -20"); next(29); break;
        case 29:
            if (!locked() || !moved(game.state.paddleX < relativeStart-1,"relative motion stopped after notification recovery")) return;
            popup = false; next(9); break;
        case 9:
            if (!locked()) return;
            if (cycles%5 === 0) console.log("MOUSE NOTIFY");
            game.toggleFullscreen(); next(10); break;
        case 10:
            if (!locked()) return;
            game.toggleFullscreen(); next(11); break;
        case 11:
            if (!locked()) return;
            console.log("MOUSE COMPOSITOR fullscreen"); next(12); break;
        case 12:
            if (!locked()) return;
            console.log("MOUSE COMPOSITOR fullscreen"); next(13); break;
        case 13:
            if (!locked()) return;
            console.log("MOUSE COMPOSITOR maximized"); next(14); break;
        case 14:
            if (!locked()) return;
            console.log("MOUSE COMPOSITOR maximized"); next(15); break;
        case 15:
            if (!locked()) return;
            click(); next(16); break;
        case 16:
            if (!free()) return;
            click(); next(17); break;
        case 17:
            if (!locked()) return;
            console.log("MOUSE FOCUS LOSS"); next(18); break;
        case 18:
            if (!free()) return;
            console.log("MOUSE FOCUS RETURN"); next(19); break;
        case 19:
            if (!free()) return;
            click(); next(20); break;
        case 20:
            if (!locked()) return;
            game.toggleMenu(); if (!free()) return; next(21); break;
        case 21:
            if (!free()) return;
            game.closeMenu(); click(); next(22); break;
        case 22:
            if (!locked()) return;
            input.keyClick(Qt.Key_Escape); next(23); break;
        case 23:
            if (!free() || !require(!game.opened && !game.windowView.visible,"Escape did not close")) return;
            game.open("{}"); next(24); break;
        case 24:
            if (!free()) return;
            cycles++;
            if (Date.now()-started >= __SECONDS__*1000) {
                console.log("MOUSE PASS", JSON.stringify({seconds:(Date.now()-started)/1000,
                            cycles:cycles, assertions:assertions, motionEvents:motionEvents, lockActivations:recoveries, unlockEvents:unlocks}));
                Qt.exit(0); return;
            }
            next(1); break;
        }
    } }
}
'''.replace('__SECONDS__', str(seconds))


def hypr(code):
    subprocess.run(['hyprctl', 'eval', code], check=True, stdout=subprocess.DEVNULL, timeout=3)


def focus(address):
    hypr('hl.dispatch(hl.dsp.focus({window=' + json.dumps('address:'+address) + '}))')


def cursor_move(x, y):
    hypr('hl.dispatch(hl.dsp.cursor.move({x='+str(x)+',y='+str(y)+'}))')


def clients():
    return json.loads(subprocess.check_output(['hyprctl', '-j', 'clients'], timeout=3))


with tempfile.TemporaryDirectory(prefix='physical-mouse-torture-') as directory:
    folder = pathlib.Path(directory)
    for name in ('Commons', 'Ui'):
        shutil.copytree(shell/name, folder/name)
    port = folder/'Port'
    shutil.copytree(project, port, ignore=shutil.ignore_patterns('.git', 'dist', '__pycache__', 'audit-logs'))
    (port/'engine/levels/level1.txt').write_text(level)
    source = port/'engine/native.go'
    source.write_text(source.read_text().replace('"paddleX": paddle.x,', '"paddleX": paddle.x, "probeTarget": mousePaddleTargetX, "probePaddleWidth": paddle.w,'))
    environment = dict(os.environ, GOCACHE=os.environ.get('GOCACHE','/tmp/physical-go-cache'))
    subprocess.run(['go', 'build', '-buildvcs=false', '-o', str(port/'bin/physical-engine'), '.'], cwd=port/'engine', env=environment, check=True, timeout=60)
    (port/'native/qmldir').write_text('plugin physicalrenderer '+str(port/'native')+'\n')
    (folder/'shell.qml').write_text(harness)
    protocol = project/'tests/protocols/wlr-virtual-pointer-unstable-v1.xml'
    subprocess.run(['wayland-scanner','client-header',str(protocol),str(folder/'virtual-pointer-client.h')],check=True)
    subprocess.run(['wayland-scanner','private-code',str(protocol),str(folder/'virtual-pointer.c')],check=True)
    flags = shlex.split(subprocess.check_output(['pkg-config','--cflags','--libs','wayland-client'],text=True))
    subprocess.run(['cc','-O2','-Wall','-Wextra',str(project/'tests/virtual_pointer.c'),str(folder/'virtual-pointer.c'),'-I'+str(folder),'-o',str(folder/'virtual-pointer')]+flags,check=True)
    device = subprocess.Popen([str(folder/'virtual-pointer')],text=True,stdin=subprocess.PIPE,stdout=subprocess.PIPE)
    def device_reply(expected):
        if not select.select([device.stdout],[],[],3)[0] or device.stdout.readline().strip() != expected:
            raise RuntimeError('Virtual mouse helper stalled or exited')
    def device_command(command):
        device.stdin.write(command+'\n'); device.stdin.flush(); device_reply('OK')
    device_reply('READY')
    dialog_title = 'Physical mouse watchdog test '+folder.name
    environment.update(PHYSICAL_TESTING='1', PHYSICAL_STATE_PATH=str(folder/'progress.json'), PHYSICAL_MOUSE_DIALOG_TITLE=dialog_title)
    cursor = json.loads(subprocess.check_output(['hyprctl','-j','cursorpos'],timeout=3))
    previous = json.loads(subprocess.check_output(['hyprctl','-j','activewindow'],timeout=3))
    if not previous.get('address'):
        raise SystemExit('A previous desktop window is needed for focus-switch assertions.')
    run = subprocess.Popen(['quickshell','--no-color','-p',str(folder/'shell.qml')],env=environment,text=True,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,start_new_session=True)
    output = queue.Queue()
    def read():
        for line in run.stdout:
            output.put(line)
        output.put(None)
    threading.Thread(target=read,daemon=True).start()
    began = time.monotonic()
    heartbeat = began
    engine_progress = began
    frame_progress = began
    frames = -1
    replies = -1
    injected = False
    watchdog = ''
    passed = False
    lines = []
    client = None
    try:
        while time.monotonic()-began < seconds+45:
            try:
                line = output.get(timeout=0.25)
            except queue.Empty:
                line = ''
            if line is None:
                break
            if line:
                lines.append(line)
                if 'MOUSE' in line or 'ERROR' in line:
                    print(line.strip(),flush=True)
                if 'MOUSE HEARTBEAT ' in line:
                    data = json.loads(line.split('MOUSE HEARTBEAT ',1)[1])
                    heartbeat = time.monotonic()
                    if data['frames'] != frames:
                        frames = data['frames']; frame_progress = heartbeat
                    if data['replies'] != replies:
                        replies = data['replies']; engine_progress = heartbeat
                    if freeze and not injected and data['cycles'] >= 1:
                        pid = run.pid
                        if freeze == 'engine':
                            children = pathlib.Path(f'/proc/{run.pid}/task/{run.pid}/children').read_text().split()
                            pid = next(int(p) for p in children if 'physical-engine' in pathlib.Path(f'/proc/{p}/comm').read_text())
                        os.kill(pid,signal.SIGSTOP)
                        injected = True
                        print('MOUSE injected '+freeze+' freeze',flush=True)
                if 'MOUSE READY' in line or 'MOUSE ENTER' in line:
                    client = next(c for c in clients() if c['pid']==run.pid and c['title']=='Physical')
                    focus(client['address'])
                    # Begin capture under the notification's input region.
                    cursor_move(client['at'][0]+client['size'][0]-50,client['at'][1]+50)
                if 'MOUSE RELATIVE ' in line:
                    delta = float(line.split('MOUSE RELATIVE ',1)[1])
                    device_command('m '+str(delta)+' 0')
                if 'MOUSE OUTSIDE' in line:
                    client = next(c for c in clients() if c['pid']==run.pid and c['title']=='Physical')
                    x = client['at'][0]+client['size'][0]//2
                    y = client['at'][1]+client['size'][1]+12
                    cursor_move(x,y)
                    actual = json.loads(subprocess.check_output(['hyprctl','-j','cursorpos'],timeout=3))
                    if abs(actual['x']-x)>2 or abs(actual['y']-y)>2:
                        raise RuntimeError('Uncaptured desktop cursor could not leave the game: '+str(actual))
                    device_command('m 12 0')
                    moved = json.loads(subprocess.check_output(['hyprctl','-j','cursorpos'],timeout=3))
                    if moved['x'] <= actual['x']+1:
                        raise RuntimeError('Desktop cursor frozen on uncaptured relative movement: '+str(moved))
                if 'MOUSE COMPOSITOR ' in line:
                    mode = line.split('MOUSE COMPOSITOR ',1)[1].strip()
                    assert mode in ('fullscreen','maximized')
                    focus(client['address'])
                    hypr('hl.dispatch(hl.dsp.window.fullscreen({mode='+json.dumps(mode)+'}))')
                if 'MOUSE FOCUS LOSS' in line:
                    focus(previous['address'])
                if 'MOUSE FOCUS RETURN' in line:
                    focus(client['address'])
                if 'MOUSE NOTIFY' in line:
                    subprocess.run(['notify-send','--app-name=Physical mouse test','--expire-time=1200','Physical mouse test','Checking capture through a notification'],check=True,timeout=3)
                if 'MOUSE PASS ' in line:
                    passed = True
                if 'MOUSE FAILED' in line and not injected:
                    raise RuntimeError('Mouse assertion failed')
            now = time.monotonic()
            if now-heartbeat > (25 if replies<0 else 4):
                watchdog = 'UI heartbeat stalled'; break
            if replies>=0 and now-engine_progress > 4:
                watchdog = 'engine replies stalled'; break
            if frames>=0 and now-frame_progress > 4 and not freeze:
                watchdog = 'rendered frames stalled'; break
        else:
            raise RuntimeError('External test deadline exceeded')
        if freeze:
            expected = 'UI heartbeat stalled' if freeze=='ui' else 'engine replies stalled'
            if not injected or watchdog != expected:
                raise RuntimeError('Freeze detector failed: '+watchdog)
            print('MOUSE WATCHDOG PASS: '+watchdog,flush=True)
        elif watchdog or not passed:
            raise RuntimeError(watchdog or 'Game exited before success')
        else:
            run.wait(timeout=5)
            if run.returncode:
                raise RuntimeError('Game exited with status '+str(run.returncode))
    except Exception:
        print(''.join(lines[-40:]),flush=True)
        raise
    finally:
        # SIGCONT also frees a deliberately stopped engine before termination.
        try:
            os.killpg(run.pid,signal.SIGCONT)
            os.killpg(run.pid,signal.SIGTERM)
            try: run.wait(timeout=3)
            except subprocess.TimeoutExpired:
                os.killpg(run.pid,signal.SIGKILL); run.wait()
        except ProcessLookupError:
            pass
        if freeze:
            for entry in pathlib.Path('/proc').iterdir():
                if not entry.name.isdigit(): continue
                try:
                    if (entry/'comm').read_text().strip() != 'hyprland-dialog': continue
                    command = (entry/'cmdline').read_bytes().decode(errors='replace')
                    if dialog_title in command: os.kill(int(entry.name),signal.SIGTERM)
                except OSError:
                    pass
        device.stdin.close()
        try: device.wait(timeout=3)
        except subprocess.TimeoutExpired: device.kill(); device.wait()
        cursor_move(cursor['x'],cursor['y'])
        focus(previous['address'])
