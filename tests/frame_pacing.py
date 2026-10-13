#!/usr/bin/env python3
"""Check real Qt/engine motion under deliberately uneven pipe processing."""
import os
import pathlib
import shutil
import subprocess
import tempfile

project = pathlib.Path(__file__).resolve().parents[1]
shell = pathlib.Path(os.environ.get("OMARCHY_SHELL_DIR", "/usr/share/omarchy/shell"))
# No collisions, acceleration or completion during this test. Physics and the
# renderer stay unchanged; only a temporary level and transport diagnostics differ.
level = """gravity=0
airDrag=0
stuckSpeedThreshold=0
ballRescueEnabled=false
ballX=300
ballY=450
ballVx=60
ballVy=0
brickWidth=100
brickHeight=40
brickOffsetTop=50
title="FRAME PACING TEST"
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
    property int rounds: 0
    property double started: 0
    property int samples: 0
    property int longFrames: 0
    property double lastTick: 0
    property double lastClock: 0
    property double lastX: 0
    property double worstMotionError: 0
    property double pausedX: 0
    Port.Physical { id: game }
    function fail(message) { console.error("PACING FAILED", message); Qt.exit(1); }
    Component.onCompleted: {
        game.open("{}");
        game.stateChanged.connect(function() {
            var s = game.state;
            if (root.phase !== 1 || !s.probeTick || s.probeTick === root.lastTick) return;
            if (s.probeTick > 15 && root.lastTick) {
                if (!(s.probeDelta > 0)) { fail("display elapsed time missing from tick"); return; }
                var clockError = Math.abs(s.probeClock - root.lastClock - s.probeDelta * 1000);
                var motionError = Math.abs(s.probeX - root.lastX - 60 * s.probeDelta);
                if (clockError > 0.001 || motionError > 0.001) {
                    fail("process delay changed visual speed: clock error " + clockError + ", motion error " + motionError + ", tick " + s.probeTick + ", dt " + s.probeDelta + ", velocity " + s.probeVX + ", x " + s.probeX + ", previous " + root.lastX + ", state " + JSON.stringify(s)); return;
                }
                root.worstMotionError = Math.max(root.worstMotionError, motionError);
                root.samples++;
                if (s.probeDelta > 0.025) root.longFrames++;
            }
            root.lastTick = s.probeTick;
            root.lastClock = s.probeClock;
            root.lastX = s.probeX;
        });
    }
    Timer { interval: 50; running: true; repeat: true; onTriggered: {
        if (++root.rounds > 240) { fail("timeout in phase " + root.phase); return; }
        if (game.errorText || game.renderError) { fail(game.errorText || game.renderError); return; }
        if (!game.engineReady || game.paintedFrames < 2) return;
        if (root.phase === 0) {
            game.shortcut(" "); root.phase = 1; root.started = Date.now();
        } else if (root.phase === 1 && Date.now() - root.started > 6500) {
            if (root.samples < 250 || !root.longFrames) { fail("insufficient delayed frames: " + root.samples + ", " + root.longFrames); return; }
            console.log("PACING motion samples", root.samples, "accumulated frames", root.longFrames, "worst position error", root.worstMotionError);
            root.phase = 2; game.shortcut(" "); root.started = Date.now();
        } else if (root.phase === 2 && game.state.paused && Date.now() - root.started > 150) {
            root.pausedX = game.state.probeX; root.phase = 3; root.started = Date.now();
        } else if (root.phase === 3 && Date.now() - root.started > 600) {
            if (game.animating || Math.abs(game.state.probeX - root.pausedX) > 0.001) { fail("paused ball moved"); return; }
            root.phase = 4; game.shortcut(" "); root.started = Date.now();
        } else if (root.phase === 4 && !game.state.paused && Date.now() - root.started > 200) {
            var movement = game.state.probeX - root.pausedX;
            if (movement < 3 || movement > 24) { fail("resume caught up paused time: " + movement); return; }
            game.close(); root.started = Date.now(); root.phase = 5;
        } else if (root.phase === 5 && Date.now() - root.started > 250) {
            if (game.animating) { fail("hidden display clock kept running"); return; }
            console.log("PACING PASS: constant visual speed under engine delays, skipped-request elapsed time, pause/resume and hidden idle"); Qt.exit(0);
        }
    } }
}
'''

with tempfile.TemporaryDirectory(prefix="physical-pacing-test-") as directory:
    folder = pathlib.Path(directory)
    for name in ["Commons", "Ui"]:
        shutil.copytree(shell / name, folder / name)
    port = folder / "Port"
    shutil.copytree(project, port, ignore=shutil.ignore_patterns(".git", "dist", "__pycache__", "audit-logs"))
    (port / "engine/levels/level1.txt").write_text(level)
    native = port / "engine/native.go"
    source = native.read_text().replace('"strings"', '"strings"\n"time"')
    source = source.replace('var nativeVolume = 0.25', 'var probeTick int\nvar probeDelta float64\nvar nativeVolume = 0.25')
    source = source.replace('case "tick":', '''case "tick":
        probeTick++
        probeDelta = c.DT
        // Delay only the temporary engine, exercising real asynchronous pipes.
        if probeTick % 90 == 0 { time.Sleep(55 * time.Millisecond) } else if probeTick % 3 == 0 { time.Sleep(8 * time.Millisecond) }
''')
    source = source.replace('"physicsSteps": physicsLastFrameSteps,', '"probeTick": probeTick, "probeClock": nativeTime, "probeDelta": probeDelta, "probeVX": ball.vx, "probeX": interpolatedRenderSnapshot(renderInterpolationAlpha).balls[0].x, "physicsSteps": physicsLastFrameSteps,')
    native.write_text(source)
    environment = dict(os.environ, GOCACHE=os.environ.get("GOCACHE", "/tmp/physical-go-cache"))
    subprocess.run(["go", "build", "-buildvcs=false", "-o", str(port / "bin/physical-engine"), "."], cwd=port / "engine", env=environment, check=True, timeout=60)
    (port / "native/qmldir").write_text("plugin physicalrenderer " + str(port / "native") + "\n")
    (folder / "shell.qml").write_text(harness)
    runtime = folder / "runtime"
    runtime.mkdir(mode=0o700)
    environment.update(PHYSICAL_STATE_PATH=str(folder / "progress.json"), PHYSICAL_TESTING="1", QT_NO_XDG_DESKTOP_PORTAL="1", QT_QPA_PLATFORMTHEME="")
    if not os.environ.get("PHYSICAL_TEST_WAYLAND"):
        environment.update(QT_QPA_PLATFORM="offscreen", QT_QUICK_BACKEND="software", XDG_RUNTIME_DIR=str(runtime), DBUS_SESSION_BUS_ADDRESS="unix:path=/dev/null")
        environment.pop("WAYLAND_DISPLAY", None)
    environment.pop("HYPRLAND_INSTANCE_SIGNATURE", None)
    result = subprocess.run(["quickshell", "--no-color", "-p", str(folder / "shell.qml")], env=environment, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=20)
    bad = any(message in result.stdout for message in ["ERROR", "PACING FAILED", "WARN scene:"])
    if result.returncode or bad or "PACING PASS:" not in result.stdout:
        print(result.stdout)
        raise SystemExit(result.returncode or 1)
    print("\n".join(line.strip() for line in result.stdout.splitlines() if "PACING" in line))
