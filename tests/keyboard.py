#!/usr/bin/env python3
"""Exercise keyboard delivery through Qt, with isolated progress and SVG exports."""
import os, pathlib, shutil, subprocess, tempfile

project = pathlib.Path(__file__).resolve().parents[1]
shell_dir = pathlib.Path(os.environ.get("OMARCHY_SHELL_DIR", "/usr/share/omarchy/shell"))
harness = r'''
import QtQuick
import QtTest
import Quickshell
import qs.Commons
import "Port" as Port
ShellRoot {
    id: root
    Port.Physical { id: game }
    TestCase { id: input; parent: game.contentView; when: false }
    property int step: 0
    property int attempts: 0
    property var before: ({})
    property bool sent: false
    property var cases: []
    function find(item,name) {
        if(item.objectName===name) return item;
        for(var child of item.children || []) { var found=find(child,name); if(found) return found; }
        return null;
    }
    function click(name) { var button=find(game.contentView,name); input.mouseClick(button,button.width/2,button.height/2); }
    function tap(key, modifiers) { input.keyClick(key, modifiers || Qt.NoModifier); }
    function same(field) { return game.state[field] === before[field]; }
    function changed(field) { return !same(field); }
    function buildCases() { cases = [
        {name:"Ready sound", run:()=>tap(Qt.Key_S), check:()=>changed("sound")},
        {name:"Ready debris", run:()=>tap(Qt.Key_R), check:()=>changed("debrisMaster")},
        {name:"Enter start", run:()=>tap(Qt.Key_Return), check:()=>!game.state.ready && !game.state.paused},
        {name:"A paddle", run:()=>input.keyPress(Qt.Key_A), check:()=>game.state.paddleX < before.paddleX, finish:()=>input.keyRelease(Qt.Key_A)},
        {name:"D paddle", run:()=>input.keyPress(Qt.Key_D), check:()=>game.state.paddleX > before.paddleX, finish:()=>input.keyRelease(Qt.Key_D)},
        {name:"Left paddle", run:()=>input.keyPress(Qt.Key_Left), check:()=>game.state.paddleX < before.paddleX, finish:()=>input.keyRelease(Qt.Key_Left)},
        {name:"Right paddle", run:()=>input.keyPress(Qt.Key_Right), check:()=>game.state.paddleX > before.paddleX, finish:()=>input.keyRelease(Qt.Key_Right)},
        {name:"Pause", run:()=>tap(Qt.Key_Space), check:()=>game.state.paused},
        {name:"Resume", run:()=>tap(Qt.Key_Space), check:()=>!game.state.paused},
        {name:"Sound", run:()=>tap(Qt.Key_S), check:()=>changed("sound")},
        {name:"Magnet", run:()=>tap(Qt.Key_M), check:()=>changed("magnet")},
        {name:"Zapper", run:()=>tap(Qt.Key_Z), check:()=>changed("zapper")},
        {name:"Extra ball", run:()=>tap(Qt.Key_2), check:()=>game.state.balls > before.balls},
        {name:"Rescue", run:()=>tap(Qt.Key_T), check:()=>game.state.status==="Ball teleported (T)"},
        {name:"Auto", run:()=>tap(Qt.Key_O), check:()=>changed("auto")},
        {name:"Diagnostics P", run:()=>tap(Qt.Key_P), check:()=>JSON.stringify(game.state.diagnostics)!==JSON.stringify(before.diagnostics)},
        {name:"Diagnostics I", run:()=>tap(Qt.Key_I), check:()=>JSON.stringify(game.state.diagnostics)!==JSON.stringify(before.diagnostics)},
        {name:"FPS", run:()=>tap(Qt.Key_F), check:()=>changed("fpsVisible") && !game.fullscreen},
        {name:"Lowest FPS", run:()=>tap(Qt.Key_F,Qt.ShiftModifier), check:()=>same("fpsVisible") && game.state.fpsLowest===0},
        {name:"Replay pause", run:()=>tap(Qt.Key_K), check:()=>game.state.replay && game.state.replayDirection===0},
        {name:"Replay toolbar play",run:()=>click("playback-toggle"),check:()=>game.state.replayDirection===1},
        {name:"Replay toolbar pause",run:()=>click("playback-toggle"),check:()=>game.state.replayDirection===0},
        {name:"Replay home", run:()=>tap(Qt.Key_Home), check:()=>game.state.replayFrame===0},
        {name:"Replay step right", run:()=>tap(Qt.Key_Right), check:()=>game.state.replayFrame===1},
        {name:"Replay step left", run:()=>tap(Qt.Key_Left), check:()=>game.state.replayFrame===0},
        {name:"Replay end", run:()=>tap(Qt.Key_End), check:()=>game.state.replayFrame===game.state.replayFrames-1},
        {name:"Replay reverse", run:()=>tap(Qt.Key_J), check:()=>game.state.replayDirection===-1},
        {name:"Replay forward", run:()=>tap(Qt.Key_L), check:()=>game.state.replayDirection===1},
        {name:"Replay pause again", run:()=>tap(Qt.Key_K), check:()=>game.state.replayDirection===0},
        {name:"SVG", run:()=>tap(Qt.Key_V), check:()=>game.notice.indexOf("Saved ")===0},
        {name:"SVG trails", run:()=>{ game.notice=""; tap(Qt.Key_B); }, check:()=>game.notice.indexOf("Saved ")===0},
        {name:"Replay exit Shift K", run:()=>tap(Qt.Key_K,Qt.ShiftModifier), check:()=>!game.state.replay},
        {name:"Replay menu enter", run:()=>tap(Qt.Key_K), check:()=>game.state.replay},
        {name:"Replay menu show", run:()=>game.toggleMenu(), check:()=>game.menuVisible},
        {name:"Replay menu guards",run:()=>{},check:()=>["debris","auto","lab","fps"].every(name=>!find(game.contentView,"toggle-"+name).enabled)},
        {name:"Replay menu playback",run:()=>click("menu-playback"),check:()=>!game.menuVisible && game.state.replayDirection===1},
        {name:"Replay menu reopen",run:()=>game.toggleMenu(),check:()=>game.menuVisible},
        {name:"Replay menu exit", run:()=>{ var button=find(game.contentView,"replay-toggle"); input.mouseClick(button,button.width/2,button.height/2); }, check:()=>!game.state.replay && !game.menuVisible},
        {name:"Replay Escape enter", run:()=>tap(Qt.Key_K), check:()=>game.state.replay},
        {name:"Replay Escape return", run:()=>tap(Qt.Key_Escape), check:()=>!game.state.replay && game.opened && game.windowView.visible},
        {name:"Replay menu Escape enter", run:()=>tap(Qt.Key_K), check:()=>game.state.replay},
        {name:"Replay menu Escape show", run:()=>{ game.toggleMenu(); find(game.contentView,"replay-toggle").forceActiveFocus(); }, check:()=>game.menuVisible},
        {name:"Replay menu Escape return", run:()=>tap(Qt.Key_Escape), check:()=>!game.state.replay && !game.menuVisible && game.opened && !game.mouseCapture.requested},
        {name:"Replay fullscreen enter", run:()=>tap(Qt.Key_K), check:()=>game.state.replay},
        {name:"Replay fullscreen on", run:()=>{ input.keyPress(Qt.Key_Space); tap(Qt.Key_F); input.keyRelease(Qt.Key_Space); }, check:()=>game.fullscreen && game.state.replay},
        {name:"Replay fullscreen Escape return", run:()=>tap(Qt.Key_Escape), check:()=>!game.state.replay && game.fullscreen && game.opened},
        {name:"Replay fullscreen back", run:()=>{ input.keyPress(Qt.Key_Space); tap(Qt.Key_F); input.keyRelease(Qt.Key_Space); }, check:()=>!game.fullscreen && game.opened},
        {name:"Lab", run:()=>tap(Qt.Key_E), check:()=>game.state.editor},
        {name:"Lab Q guard", run:()=>tap(Qt.Key_Q), check:()=>game.opened && game.state.editor},
        {name:"Lab exit", run:()=>tap(Qt.Key_E), check:()=>!game.state.editor},
        {name:"Unlock", run:()=>tap(Qt.Key_U), check:()=>game.state.unlocked},
        {name:"Page down", run:()=>tap(Qt.Key_PageDown), check:()=>game.state.level===2},
        {name:"Page up", run:()=>tap(Qt.Key_PageUp), check:()=>game.state.level===1},
        {name:"Help", run:()=>tap(Qt.Key_Question), check:()=>game.menuVisible && game.helpVisible},
        {name:"Menu sound", run:()=>game.send({action:"toggle-sound"}), check:()=>changed("sound")},
        {name:"Menu Escape", run:()=>tap(Qt.Key_Escape), check:()=>!game.opened, finish:()=>game.open("{}")},
        {name:"Reopened Space", run:()=>tap(Qt.Key_Space), check:()=>!game.state.ready && !game.state.paused},
        {name:"Fullscreen chord", run:()=>{ input.keyPress(Qt.Key_Space); tap(Qt.Key_F); input.keyRelease(Qt.Key_Space); }, check:()=>game.fullscreen && !game.state.paused},
        {name:"Fullscreen chord back", run:()=>{ input.keyPress(Qt.Key_Space); tap(Qt.Key_F); input.keyRelease(Qt.Key_Space); }, check:()=>!game.fullscreen && !game.state.paused},
        {name:"Close Q", run:()=>tap(Qt.Key_Q), check:()=>!game.opened}
    ]; }
    Component.onCompleted: { buildCases(); game.open("{}"); }
    Timer { interval: 250; repeat: true; running: true; onTriggered: {
        if(game.errorText || game.renderError || ++root.attempts>160) { console.error("KEYBOARD FAILED",root.step,game.errorText,game.renderError); Qt.exit(1); return; }
        if(!game.engineReady || game.paintedFrames<2) return;
        if(root.step>=root.cases.length) { console.log("KEYBOARD PASS:",root.cases.length,"keyboard checks through Qt input"); Qt.exit(0); return; }
        var test=root.cases[root.step];
        if(!root.sent) { root.before=JSON.parse(JSON.stringify(game.state)); test.run(); root.sent=true; return; }
        if(!test.check()) { console.error("KEYBOARD FAILED",test.name,JSON.stringify(game.state)); Qt.exit(1); return; }
        if(test.finish) test.finish();
        console.log("KEYBOARD OK",test.name);
        root.step++; root.sent=false;
    } }
}
'''

with tempfile.TemporaryDirectory(prefix="physical-qml-test-") as directory:
    folder = pathlib.Path(directory)
    for name in ["Commons", "Ui"]:
        shutil.copytree(shell_dir / name, folder / name)
    shutil.copytree(project, folder / "Port", ignore=shutil.ignore_patterns(".git", "__pycache__"))
    (folder / "Port/native/qmldir").write_text("plugin physicalrenderer " + str(folder / "Port/native") + "\n")
    (folder / "shell.qml").write_text(harness)
    runtime = folder / "runtime"
    runtime.mkdir(mode=0o700)
    env = dict(os.environ, QT_QPA_PLATFORM="offscreen", QT_QUICK_BACKEND="software", QT_NO_XDG_DESKTOP_PORTAL="1", QT_QPA_PLATFORMTHEME="", XDG_RUNTIME_DIR=str(runtime), PHYSICAL_STATE_PATH=str(folder / "progress.json"), PHYSICAL_TESTING="1", PHYSICAL_EXPORT_DIR=str(folder / "exports"), DBUS_SESSION_BUS_ADDRESS="unix:path=/dev/null")
    if os.environ.get("PHYSICAL_TEST_WAYLAND"):
        for key in ["QT_QUICK_BACKEND", "QT_QPA_PLATFORM", "XDG_RUNTIME_DIR", "DBUS_SESSION_BUS_ADDRESS"]:
            env.pop(key, None)
            if key in os.environ: env[key] = os.environ[key]
    else:
        env.pop("WAYLAND_DISPLAY", None)
    env.pop("HYPRLAND_INSTANCE_SIGNATURE", None)
    try:
        run = subprocess.run(["quickshell", "--no-color", "-p", str(folder / "shell.qml")], env=env, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=50)
    except subprocess.TimeoutExpired as error:
        print((error.stdout or b"" ).decode(errors="replace"))
        raise
    if os.environ.get("PHYSICAL_TEST_DEBUG"): print(run.stdout)
    bad = [line for line in run.stdout.splitlines() if "WARN scene:" in line or "ERROR" in line or "QSoundEffect: Error decoding" in line]
    if run.returncode or "KEYBOARD PASS:" not in run.stdout or bad:
        print(run.stdout)
        raise SystemExit(run.returncode or 1)
    print("\n".join(line.strip() for line in run.stdout.splitlines() if "KEYBOARD OK" in line))
    print(next(line.strip() for line in run.stdout.splitlines() if "KEYBOARD PASS:" in line))
