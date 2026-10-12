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
                if (game.windowView.width > game.defaultWindowWidth + 2 || game.windowView.height > game.defaultWindowHeight + 2) { console.error("SMOKE oversized initial window"); Qt.exit(1); return; }
                capture("physical-ready"); root.phase = 20;
            } else if (root.phase === 20) {
                game.pressKey({key: Qt.Key_Space, isAutoRepeat: false});
                game.pressKey({key: Qt.Key_F, isAutoRepeat: false});
                game.releaseKey({key: Qt.Key_Space, isAutoRepeat: false});
                if (!game.fullscreen) { console.error("SMOKE fullscreen chord failed"); Qt.exit(1); return; }
                root.startTime = Date.now(); root.phase = 21;
            } else if (root.phase === 21 && Date.now() - root.startTime > 300) {
                if (Quickshell.env("PHYSICAL_TEST_WAYLAND") && game.windowView.width < game.windowView.screen.width - 2) { console.error("SMOKE fullscreen geometry failed"); Qt.exit(1); return; }
                game.pressKey({key: Qt.Key_Escape, isAutoRepeat: false});
                if (game.opened || game.fullscreen || game.mouseCapture.requested) { console.error("SMOKE Escape did not close fullscreen game"); Qt.exit(1); return; }
                game.open("{}");
                root.startTime = Date.now(); root.phase = 22;
            } else if (root.phase === 22 && Date.now() - root.startTime > 300) {
                if (game.windowView.width > game.defaultWindowWidth + 2 || game.windowView.height > game.defaultWindowHeight + 2) { console.error("SMOKE fullscreen restore size failed"); Qt.exit(1); return; }
                if (!Quickshell.env("PHYSICAL_TEST_WAYLAND") && game.mouseCapture.lock()) { console.error("SMOKE offscreen mouse lock succeeded"); Qt.exit(1); return; }
                game.mouseCapture.release(); root.phase = 10;
            } else if (root.phase === 10) {
                game.shortcut(" "); game.shortcut("o"); root.phase = 1; root.initialFrames = game.paintedFrames; root.startTime = Date.now();
            } else if (root.phase === 1 && Date.now() - root.startTime > 3000) {
                console.log("SMOKE rendered", game.paintedFrames - root.initialFrames, "frames", (game.paintedFrames - root.initialFrames) * 1000 / (Date.now() - root.startTime), "FPS");
                if (Quickshell.env("PHYSICAL_TEST_WAYLAND")) {
                    var audio = findItem(game, "physicalAudio"), wall = findItem(game, "sound-wall-0"), paddle = findItem(game, "sound-paddle-0");
                    if (!audio || !wall || !paddle || !wall.ready || !paddle.ready) { console.error("SMOKE contact clips not loaded"); Qt.exit(1); return; }
                    if (audio.roomSettings.name !== "space" || audio.roomSettings.dry !== .5) { console.error("SMOKE level room settings did not reach audio mixer"); Qt.exit(1); return; }
                    var previous = "", selected = ({});
                    for (var sample = 0; sample < 300; ++sample) {
                        var name = audio.chooseClip({kind:"brick"});
                        if (name === previous || audio.brickClips.indexOf(name) < 0) { console.error("SMOKE random brick selection failed"); Qt.exit(1); return; }
                        selected[name] = true; previous = name;
                    }
                    if (Object.keys(selected).length < 15) { console.error("SMOKE normal sample bank not used"); Qt.exit(1); return; }
                    for (var cue = 0; cue < audio.featureClips.length; ++cue) {
                        var featureName = audio.featureClips[cue];
                        if (audio.chooseClip({kind:"magic", feature:featureName.slice(8)}) !== featureName || !findItem(game, "sound-" + featureName + "-0").ready) { console.error("SMOKE powerup cue missing", featureName); Qt.exit(1); return; }
                    }
                    audio.play([{kind:"wall",wave:"triangle",args:[230,175,.05,.12,0]}, {kind:"paddle",wave:"triangle",args:[180,320,.08,.18,0]}, {kind:"magic",feature:"blackhole",rate:1,gain:.05,cutoff:500,q:.5}]);
                    if (!audio.isPlaying("feature-blackhole")) { console.error("SMOKE powerup sample not playing"); Qt.exit(1); return; }
                    if (!(wall.playing || findItem(game, "sound-wall-1").playing) || !(paddle.playing || findItem(game, "sound-paddle-1").playing)) { console.error("SMOKE contact clips not playing"); Qt.exit(1); }
                }
                game.toggleMenu();
                var volume = findItem(game.contentView, "audio-volume");
                if (!volume || volume.width < 50 || volume.height < 12 || volume.value !== 0.25) { console.error("SMOKE unusable volume slider"); Qt.exit(1); }
                volume.value = 0.37; volume.moved();
                capture("physical-menu"); root.phase = 11;
            } else if (root.phase === 11 && game.state.volume === 0.37) {
                game.closeMenu(); game.shortcut("k"); root.phase = 2;
            } else if (root.phase === 2 && game.state.replay) {
                capture("physical-replay"); root.phase = 12;
            } else if (root.phase === 12) {
                game.send({action:"replay-exit"}); game.shortcut("e"); root.phase = 3;
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
    try:
        run = subprocess.run(["quickshell", "--no-color", "-p", str(folder / "shell.qml")], env=env, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=25)
    except subprocess.TimeoutExpired as error:
        print((error.stdout or b"" ).decode(errors="replace"))
        raise
    if os.environ.get("PHYSICAL_TEST_DEBUG"): print(run.stdout)
    bad = [line for line in run.stdout.splitlines() if "WARN scene:" in line or "ERROR" in line or "QSoundEffect: Error decoding" in line]
    if run.returncode or "SMOKE PASS:" not in run.stdout or bad:
        print(run.stdout)
        raise SystemExit(run.returncode or 1)
    print("\n".join(line.strip() for line in run.stdout.splitlines() if "SMOKE rendered" in line))
    print(next(line.strip() for line in run.stdout.splitlines() if "SMOKE PASS:" in line))
