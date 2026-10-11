#!/usr/bin/env python3
"""Smoke-test the actual plugin in a headless Quickshell with private state."""
import os, pathlib, shutil, subprocess, tempfile

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
    property int minimumFrames: 10000
    property int maxDebris: 0
    Port.Physical { id: game }
    Component.onCompleted: game.open("{}")
    Timer { interval: 1000; running: true; repeat: true; onTriggered: {
        if (!game.engineReady) return;
        if (!root.started) { game.shortcut(" "); game.shortcut("o"); root.started = true; return; }
        root.totalFrames += game.paintedFrames - root.frames;
        root.minimumFrames = Math.min(root.minimumFrames,game.paintedFrames - root.frames);
        root.maxDebris = Math.max(root.maxDebris,game.state.debrisPieces || 0);
        console.log("PERF", ++root.seconds, "frames", game.paintedFrames - root.frames, "engineFPS", game.state.fps, "debris", game.state.debrisPieces, "remaining", game.state.remaining);
        root.frames = game.paintedFrames;
        if (root.seconds === 10 && Quickshell.env("PHYSICAL_TEST_ARTIFACTS")) game.contentView.grabToImage(function(result) { result.saveToFile(Quickshell.env("PHYSICAL_TEST_ARTIFACTS") + "/physical-gpu-stress.png"); });
        if (root.seconds === 5) { game.shortcut("z"); game.shortcut("m"); game.shortcut("2"); game.shortcut("2"); }
        if (game.state.complete) game.shortcut(" ");
        else if (game.state.gameOver) { game.send({action:"restart"}); game.shortcut("o"); }
        if (root.seconds >= 30) { console.log("SMOKE PASS: sustained gameplay", "average",root.totalFrames/root.seconds,"minimum",root.minimumFrames,"max debris",root.maxDebris); Qt.exit(0); }
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
    env = dict(os.environ, QT_QPA_PLATFORM="offscreen", QT_QUICK_BACKEND="software", QT_NO_XDG_DESKTOP_PORTAL="1", QT_QPA_PLATFORMTHEME="", XDG_RUNTIME_DIR=str(runtime), PHYSICAL_STATE_PATH=str(folder / "progress.json"), PHYSICAL_TESTING="1", DBUS_SESSION_BUS_ADDRESS="unix:path=/dev/null")
    if os.environ.get("PHYSICAL_TEST_WAYLAND"):
        for key in ["QT_QUICK_BACKEND", "QT_QPA_PLATFORM", "XDG_RUNTIME_DIR", "DBUS_SESSION_BUS_ADDRESS"]:
            env.pop(key, None)
            if key in os.environ: env[key] = os.environ[key]
    else:
        env.pop("WAYLAND_DISPLAY", None)
    env.pop("HYPRLAND_INSTANCE_SIGNATURE", None)
    run = subprocess.run(["quickshell", "--no-color", "-p", str(folder / "shell.qml")], env=env, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=45)
    bad = [line for line in run.stdout.splitlines() if "WARN scene:" in line or "ERROR" in line]
    if run.returncode or "SMOKE PASS:" not in run.stdout or bad:
        print(run.stdout)
        raise SystemExit(run.returncode or 1)
    print("\n".join(line.strip() for line in run.stdout.splitlines() if "PERF" in line))
    print(next(line.strip() for line in run.stdout.splitlines() if "SMOKE PASS:" in line))
