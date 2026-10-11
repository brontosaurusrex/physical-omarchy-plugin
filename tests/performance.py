#!/usr/bin/env python3
"""Measure sustained gameplay in a real Quickshell using private progress."""
import os, pathlib, shutil, subprocess, tempfile

duration = int(os.environ.get("PHYSICAL_PERF_SECONDS", "180"))
level = int(os.environ.get("PHYSICAL_PERF_LEVEL", "26"))
stress = os.environ.get("PHYSICAL_PERF_STRESS", "1") != "0"
if not 10 <= duration <= 3600 or not 1 <= level <= 33:
    raise SystemExit("Duration must be 10–3600 seconds and level 1–33")
project = pathlib.Path(__file__).resolve().parents[1]
shell_dir = pathlib.Path(os.environ.get("OMARCHY_SHELL_DIR", "/usr/share/omarchy/shell"))
harness = r'''
import QtQuick
import Quickshell
import qs.Commons
import "Port" as Port
ShellRoot {
    id: root
    property int seconds: 0
    property int frames: 0
    property bool started: false
    property int totalFrames: 0
    property real minimumFrames: 10000
    property real elapsed: 0
    property double lastSample: 0
    property bool navigating: false
    property bool restarting: false
    property int maxDebris: 0
    Port.Physical { id: game }
    Component.onCompleted: game.open("{}")
    Timer { interval: 1000; running: true; repeat: true; onTriggered: {
        if (!game.engineReady) return;
        if (!root.navigating) {
            if (Quickshell.env("PHYSICAL_PERF_MUTE") === "1") game.send({action:"volume", value:0});
            game.shortcut("u");
            for (var i = 1; i < __LEVEL__; ++i) game.shortcut("PageDown");
            root.navigating = true; return;
        }
        if (game.state.level !== __LEVEL__) { console.error("PERF wrong level", game.state.level); Qt.exit(1); return; }
        if (!root.started) {
            if (!game.state.auto) game.shortcut("o");
            if (game.state.ready || game.state.paused) game.shortcut(" ");
            if (__STRESS__) { game.shortcut("z"); game.shortcut("m"); game.shortcut("2"); game.shortcut("2"); }
            root.started = true; root.frames = game.paintedFrames; root.lastSample = Date.now(); return;
        }
        var interval = (Date.now() - root.lastSample) / 1000;
        var painted = game.paintedFrames - root.frames;
        root.elapsed += interval;
        root.lastSample = Date.now();
        root.totalFrames += game.paintedFrames - root.frames;
        root.minimumFrames = Math.min(root.minimumFrames, painted / interval);
        root.maxDebris = Math.max(root.maxDebris,game.state.debrisPieces || 0);
        console.log("PERF", ++root.seconds, "paintedFPS", (painted / interval).toFixed(2), "level", game.state.level, "balls", game.state.balls, "replayFrames", game.state.replayFrames, "engineFPS", game.state.fps, "debris", game.state.debrisPieces, "remaining", game.state.remaining, "transition", game.state.complete || game.state.gameOver || game.state.ready);
        root.frames = game.paintedFrames;
        if (root.seconds === 10 && Quickshell.env("PHYSICAL_TEST_ARTIFACTS")) game.contentView.grabToImage(function(result) { result.saveToFile(Quickshell.env("PHYSICAL_TEST_ARTIFACTS") + "/physical-gpu-stress.png"); });
        if (root.restarting) {
            if (!game.state.auto) game.shortcut("o");
            if (game.state.ready || game.state.paused) game.shortcut(" ");
            if (__STRESS__) { game.shortcut("z"); game.shortcut("m"); game.shortcut("2"); game.shortcut("2"); }
            root.restarting = false;
        } else if (!root.restarting && (game.state.complete || game.state.gameOver)) {
            game.send({action:"restart"}); root.restarting = true;
        } else if (!root.restarting && (game.state.ready || game.state.paused)) game.shortcut(" ");
        if (root.elapsed >= __SECONDS__) { console.log("SMOKE PASS: sustained gameplay", "level", __LEVEL__, "stress", __STRESS__, "seconds", root.elapsed.toFixed(2), "average",root.totalFrames/root.elapsed,"minimum",root.minimumFrames,"max debris",root.maxDebris); Qt.exit(0); }
    } }
}
'''

harness = harness.replace("__LEVEL__", str(level)).replace("__SECONDS__", str(duration)).replace("__STRESS__", "true" if stress else "false")

with tempfile.TemporaryDirectory(prefix="physical-qml-test-") as directory:
    folder = pathlib.Path(directory)
    for name in ["Commons", "Ui"]:
        shutil.copytree(shell_dir / name, folder / name)
    shutil.copytree(project, folder / "Port", ignore=shutil.ignore_patterns(".git", "__pycache__"))
    (folder / "Port/native/qmldir").write_text("plugin physicalrenderer " + str(folder / "Port/native") + "\n")
    (folder / "shell.qml").write_text(harness)
    runtime = folder / "runtime"
    runtime.mkdir(mode=0o700)
    env = dict(os.environ, QT_QPA_PLATFORM="offscreen", QT_QUICK_BACKEND="software", QT_NO_XDG_DESKTOP_PORTAL="1", QT_QPA_PLATFORMTHEME="", XDG_RUNTIME_DIR=str(runtime), PHYSICAL_STATE_PATH=str(folder / "progress.json"), PHYSICAL_TESTING="1", DBUS_SESSION_BUS_ADDRESS="unix:path=/dev/null")
    if os.environ.get("PHYSICAL_TEST_WAYLAND"):
        for key in ["QT_QUICK_BACKEND", "QT_QPA_PLATFORM", "XDG_RUNTIME_DIR", "DBUS_SESSION_BUS_ADDRESS"]:
            env.pop(key, None)
            if key in os.environ: env[key] = os.environ[key]
    else:
        env.pop("WAYLAND_DISPLAY", None)
    env.pop("HYPRLAND_INSTANCE_SIGNATURE", None)
    import threading
    run = subprocess.Popen(["quickshell", "--no-color", "-p", str(folder / "shell.qml")], env=env, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    timeout = threading.Timer(duration + 60, run.kill)
    timeout.start()
    lines = []
    socketErrors = 0
    try:
        for line in run.stdout:
            lines.append(line)
            if "QSocketNotifier:" in line:
                socketErrors += 1
                if socketErrors >= 20:
                    print("PERF FAILED: repeated Qt socket errors; stopping stalled benchmark", flush=True)
                    run.kill()
                    break
            if "PERF" in line or "SMOKE PASS:" in line or os.environ.get("PHYSICAL_TEST_DEBUG"):
                print(line.strip(), flush=True)
        run.wait()
    finally:
        timeout.cancel()
        if run.poll() is None: run.kill(); run.wait()
    bad = [line for line in lines if "WARN scene:" in line or "ERROR" in line or "QSocketNotifier:" in line]
    if run.returncode or not any("SMOKE PASS:" in line for line in lines) or bad:
        print("".join(lines))
        raise SystemExit(run.returncode or 1)
