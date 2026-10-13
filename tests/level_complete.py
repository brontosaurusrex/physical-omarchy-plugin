#!/usr/bin/env python3
"""Complete a tiny test level through real physics and check its Qt frame clock."""
import os
import pathlib
import shutil
import subprocess
import tempfile

project = pathlib.Path(__file__).resolve().parents[1]
shell = pathlib.Path(os.environ.get("OMARCHY_SHELL_DIR", "/usr/share/omarchy/shell"))
# Only the temporary engine's embedded level changes. Its physics, completion
# handling and drawing code are the production code, without test-only commands.
level = """gravity=0
ballX=900
ballY=170
ballVx=0
ballVy=-300
brickWidth=100
brickHeight=40
brickPadding=10
brickOffsetTop=50
debrisPiecesMin=32
debrisPiecesMax=32
debrisLifetime=4
debrisLifetimeVariationPercent=0
debrisGravityScale=0
title="TRANSITION TEST"
---
#
"""
harness = r'''
import QtQuick
import Quickshell
import "Port" as Port
ShellRoot {
    id: root
    property int phase: 0
    property double started: 0
    property int frames: 0
    property int cornerPaints: 0
    property int initialCornerPaints: 0
    property int initialFrontPaints: 0
    property int rounds: 0
    Port.Physical { id: game }
    Component.onCompleted: game.open("{}")
    function fail(message) { console.error("TRANSITION FAILED", message); Qt.exit(1); }
    function find(item, name) {
        if (item.objectName === name) return item;
        for (var child of item.children || []) { var found = find(child, name); if (found) return found; }
        return null;
    }
    Timer { interval: 50; running: true; repeat: true; onTriggered: {
        if (++root.rounds > 220) { fail("timeout in phase " + root.phase); return; }
        if (game.errorText || game.renderError) { fail(game.errorText || game.renderError); return; }
        if (!game.engineReady || game.paintedFrames < 2) return;
        if (root.phase === 0) {
            for (var i = 0; i < 4; ++i) {
                var corner = find(game.gameView, "physical-corner-" + i);
                if (!corner) { fail("missing corner image"); return; }
                corner.painted.connect(function() { root.cornerPaints++; });
            }
            game.shortcut(" "); root.phase = 1;
        } else if (root.phase === 1 && game.state.complete) {
            if (game.state.debrisPieces < 1) { fail("fixture produced no debris"); return; }
            root.started = Date.now(); root.frames = game.paintedFrames;
            root.initialCornerPaints = root.cornerPaints; root.phase = 2;
            root.initialFrontPaints = game.state.qtFrontPaintCount;
        } else if (root.phase === 2 && Date.now() - root.started >= 1800) {
            var fps = (game.paintedFrames - root.frames) * 1000 / (Date.now() - root.started);
            console.log("TRANSITION completion FPS", fps, "debris", game.state.debrisPieces);
            if (fps < 30 || !game.state.complete || game.state.debrisPieces < 1) { fail("completion animation slowed or vanished"); return; }
            if (root.cornerPaints !== root.initialCornerPaints) { fail("static corners repainted during animation"); return; }
            var repaints = game.state.qtFrontPaintCount - root.initialFrontPaints;
            if (repaints > 6) { fail("static completion text repainted with every particle frame"); return; }
            console.log("TRANSITION cached foreground repaints", repaints);
            root.phase = 3;
        } else if (root.phase === 3 && game.state.debrisPieces === 0) {
            if (game.animating) { fail("empty completion screen kept the display clock running"); return; }
            game.send({action:"pause"}); root.started = Date.now(); root.phase = 4;
        } else if (root.phase === 4 && game.state.paused && Date.now() - root.started > 250) {
            if (game.animating) { fail("paused completion kept animating"); return; }
            game.shortcut(" "); root.phase = 5;
        } else if (root.phase === 5 && !game.state.complete && game.state.level === 2) {
            if (!game.state.ready) { fail("next level did not reach READY"); return; }
            game.close(); root.started = Date.now(); root.frames = game.paintedFrames; root.phase = 6;
        } else if (root.phase === 6 && Date.now() - root.started > 400) {
            if (game.animating || game.paintedFrames - root.frames > 2) { fail("hidden game kept drawing"); return; }
            console.log("TRANSITION PASS: completion debris, cached corners, empty-screen idle, pause, continue and hidden idle"); Qt.exit(0);
        }
    } }
}
'''

with tempfile.TemporaryDirectory(prefix="physical-transition-test-") as directory:
    folder = pathlib.Path(directory)
    for name in ["Commons", "Ui"]:
        shutil.copytree(shell / name, folder / name)
    port = folder / "Port"
    shutil.copytree(project, port, ignore=shutil.ignore_patterns(".git", "dist", "__pycache__", "audit-logs"))
    (port / "engine/levels/level1.txt").write_text(level)
    environment = dict(os.environ, GOCACHE=os.environ.get("GOCACHE", "/tmp/physical-go-cache"))
    subprocess.run(["go", "build", "-buildvcs=false", "-o", str(port / "bin/physical-engine"), "."], cwd=port / "engine", env=environment, check=True, timeout=60)
    (port / "native/qmldir").write_text("plugin physicalrenderer " + str(port / "native") + "\n")
    (folder / "shell.qml").write_text(harness)
    runtime = folder / "runtime"
    runtime.mkdir(mode=0o700)
    environment.update(PHYSICAL_STATE_PATH=str(folder / "progress.json"), PHYSICAL_TESTING="1", PHYSICAL_RENDER_TIMINGS="1", QT_NO_XDG_DESKTOP_PORTAL="1", QT_QPA_PLATFORMTHEME="")
    if not os.environ.get("PHYSICAL_TEST_WAYLAND"):
        environment.update(QT_QPA_PLATFORM="offscreen", QT_QUICK_BACKEND="software", XDG_RUNTIME_DIR=str(runtime), DBUS_SESSION_BUS_ADDRESS="unix:path=/dev/null")
        environment.pop("WAYLAND_DISPLAY", None)
    environment.pop("HYPRLAND_INSTANCE_SIGNATURE", None)
    result = subprocess.run(["quickshell", "--no-color", "-p", str(folder / "shell.qml")], env=environment, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=20)
    bad = any(message in result.stdout for message in ["ERROR", "TRANSITION FAILED", "WARN scene:"])
    if result.returncode or bad or "TRANSITION PASS:" not in result.stdout:
        print(result.stdout)
        raise SystemExit(result.returncode or 1)
    print("\n".join(line.strip() for line in result.stdout.splitlines() if "TRANSITION" in line))
