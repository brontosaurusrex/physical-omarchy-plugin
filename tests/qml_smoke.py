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
    property int phase: 0
    property int rounds: 0
    property int initialFrames: 0
    property bool capturing: false
    property double startTime: 0
    property int checkedLevel: 1
    Port.Physical { id: game }
    Component.onCompleted: game.open("{}")
    function capture(name) {
        var dir = Quickshell.env("PHYSICAL_TEST_ARTIFACTS");
        if (dir) { capturing = true; game.contentView.grabToImage(function(result) { result.saveToFile(dir + "/" + name + ".png"); capturing = false; }); }
    }
    function findItem(item, name) {
        if (!item) return null;
        if (item.objectName === name) return item;
        var children = item.children || [];
        for (var i = 0; i < children.length; ++i) { var found = findItem(children[i], name); if (found) return found; }
        return null;
    }
    Timer {
        interval: 100; running: true; repeat: true
        onTriggered: {
            root.rounds++;
            if (game.errorText || game.renderError) { console.error("SMOKE FAILED", game.errorText, game.renderError); Qt.exit(1); }
            if (root.rounds > 150) { console.error("SMOKE TIMEOUT", root.phase); Qt.exit(1); }
            if (!game.engineReady || game.paintedFrames < 2) return;
            if (root.capturing) return;
            if (root.phase === 0) {
                game.shortcut(" "); game.shortcut("o"); root.phase = 1; root.initialFrames = game.paintedFrames; root.startTime = Date.now();
            } else if (root.phase === 1 && Date.now() - root.startTime > 3000) {
                console.log("SMOKE rendered", game.paintedFrames - root.initialFrames, "frames", (game.paintedFrames - root.initialFrames) * 1000 / (Date.now() - root.startTime), "FPS");
                if (Quickshell.env("PHYSICAL_TEST_WAYLAND")) {
                    var audio = findItem(game, "physicalAudio"), wall = findItem(game, "sound-wall-0"), paddle = findItem(game, "sound-paddle-0");
                    if (!audio || !wall || !paddle || !wall.ready || !paddle.ready) { console.error("SMOKE contact clips not loaded"); Qt.exit(1); }
                    audio.play([{kind:"wall"}, {kind:"paddle"}]);
                    if (!(wall.playing || findItem(game, "sound-wall-1").playing) || !(paddle.playing || findItem(game, "sound-paddle-1").playing)) { console.error("SMOKE contact clips not playing"); Qt.exit(1); }
                }
                var volume = findItem(game.contentView, "audio-volume");
                if (!volume || volume.width < 50 || volume.height < 12 || volume.value !== 0.5) { console.error("SMOKE unusable volume slider"); Qt.exit(1); }
                volume.value = 0.25; volume.moved();
                capture("physical-dark"); root.phase = 11;
            } else if (root.phase === 11 && game.state.volume === 0.25) {
                game.shortcut("k"); root.phase = 2;
            } else if (root.phase === 2 && game.state.replay) {
                game.shortcut("Escape"); game.shortcut("e"); root.phase = 3;
            } else if (root.phase === 3 && game.state.editor) {
                var gravity = findItem(game.contentView, "physics-gravity");
                if (!gravity || gravity.height < 12 || gravity.width < 100) { console.error("SMOKE unusable physics slider"); Qt.exit(1); }
                Color.shellValues = ({});
                Color.background = "#efefef"; Color.foreground = "#111111"; Color.accent = "#2266aa"; Color.urgent = "#cc2233"; Color.muted = "#777777";
                game.send({ action:"setting", group:"physics", key:"gravity", value:300 });
                root.phase = 4;
            } else if (root.phase === 4) {
                capture("physical-light-lab"); root.phase = 14;
            } else if (root.phase === 14) {
                game.shortcut("e"); game.close(); root.initialFrames = game.paintedFrames; root.phase = 5; root.rounds = 0;
            } else if (root.phase === 5 && root.rounds >= 5) {
                if (game.paintedFrames - root.initialFrames > 2) { console.error("SMOKE hidden kept rendering"); Qt.exit(1); }
                game.open("{}"); root.phase = 6;
            } else if (root.phase === 6 && game.state.paused && game.opened) {
                game.shortcut("u"); root.phase = 7;
            } else if (root.phase === 7 && game.state.level === root.checkedLevel) {
                if (root.checkedLevel === 33) { console.log("SMOKE PASS: native QML rendering of 33 levels, playback, replay, tuner, live theme, volume, hidden pause, reopen"); Qt.exit(0); }
                root.checkedLevel++; game.shortcut("PageDown");
            }
        }
    }
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
    run = subprocess.run(["quickshell", "--no-color", "-p", str(folder / "shell.qml")], env=env, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=25)
    bad = [line for line in run.stdout.splitlines() if "WARN scene:" in line or "ERROR" in line]
    if run.returncode or "SMOKE PASS:" not in run.stdout or bad:
        print(run.stdout)
        raise SystemExit(run.returncode or 1)
    print("\n".join(line.strip() for line in run.stdout.splitlines() if "SMOKE rendered" in line))
    print(next(line.strip() for line in run.stdout.splitlines() if "SMOKE PASS:" in line))
