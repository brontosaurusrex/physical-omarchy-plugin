#!/usr/bin/env python3
"""Exercise actual Y input and the idle/paused Qt clock during brick wipes."""
import os
import pathlib
import shutil
import subprocess
import tempfile

project = pathlib.Path(__file__).resolve().parents[1]
shell = pathlib.Path(os.environ.get("OMARCHY_SHELL_DIR", "/usr/share/omarchy/shell"))
harness = r'''
import QtQuick
import QtTest
import Quickshell
import "Port" as Port
ShellRoot {
    id: root
    Port.Physical { id: game }
    TestCase { id: input; parent: game.contentView; when: false }
    property int phase: 0
    property int rounds: 0
    property int samples: 0
    property var before: ({})
    property bool early: false
    property bool late: false
    property bool capturing: false
    property bool began: false
    function fail(message) { console.error("WIPE FAILED", message); Qt.exit(1); }
    function capture(name, done) {
        var folder = Quickshell.env("PHYSICAL_WIPE_PREVIEW_DIR");
        if (!folder) { if (done) done(); return; }
        root.capturing = true;
        game.contentView.grabToImage(function(result) {
            if (!result.saveToFile(folder + "/" + name + ".png")) { fail("saving preview"); return; }
            root.capturing = false;
            if (done) done();
        });
    }
    function beginFlip() {
        root.before = JSON.parse(JSON.stringify(game.state));
        root.samples = 0; root.began = false;
        input.keyClick(Qt.Key_Y);
    }
    Component.onCompleted: game.open("{}")
    Timer { interval: 20; running: true; repeat: true; onTriggered: {
        if (++root.rounds > 600) { fail("timeout at phase " + root.phase); return; }
        if (game.errorText || game.renderError) { fail(game.errorText || game.renderError); return; }
        if (!game.engineReady || game.paintedFrames < 2) return;
        if (root.phase === 0) {
            root.phase = 1;
            capture("before", function() { root.phase = 2; beginFlip(); });
        } else if ([2, 3, 5].indexOf(root.phase) !== -1) {
            if (game.state.flipAnimating) {
                root.began = true; root.samples++;
                if (!game.animating) { fail("idle/paused wipe clock stopped"); return; }
                if (game.state.flipReverse !== (root.phase === 3)) { fail("wrong wipe direction"); return; }
                if (root.phase === 2 && !root.capturing) {
                    if (!root.early && game.state.flipPhase > .4 && game.state.flipPhase < .9) {
                        root.early = true; capture("early");
                    } else if (!root.late && game.state.flipPhase > 1.4 && game.state.flipPhase < 1.9) {
                        root.late = true; capture("late");
                    }
                }
                return;
            }
            if (!root.began || root.capturing) return;
            if (root.samples < 6 || game.animating || game.state.flipped === root.before.flipped) {
                fail("wipe failed to animate, settle or stop its clock"); return;
            }
            for (var field of ["x", "y", "paddleX", "lives", "score", "ready", "paused"]) {
                if (game.state[field] !== root.before[field]) { fail("wipe changed " + field); return; }
            }
            if (root.phase === 2) {
                if (!root.early || !root.late) { fail("wipe skipped its intermediate layouts"); return; }
                root.phase = 1;
                capture("after", function() { root.phase = 3; beginFlip(); });
            } else if (root.phase === 3) {
                game.shortcut(" "); root.phase = 4;
            } else {
                console.log("WIPE PASS: Qt Y input, opposite wipe directions, READY/paused clock, exact ball/paddle state and settled idle"); Qt.exit(0);
            }
        } else if (root.phase === 4 && !game.state.ready) {
            if (!game.state.paused) { game.shortcut(" "); return; }
            root.phase = 5; beginFlip();
        }
    } }
}
'''

with tempfile.TemporaryDirectory(prefix="physical-wipe-test-") as directory:
    folder = pathlib.Path(directory)
    for name in ["Commons", "Ui"]:
        shutil.copytree(shell / name, folder / name)
    port = folder / "Port"
    shutil.copytree(project, port, ignore=shutil.ignore_patterns(".git", "dist", "__pycache__", "audit-logs"))
    (port / "native/qmldir").write_text("plugin physicalrenderer " + str(port / "native") + "\n")
    (folder / "shell.qml").write_text(harness)
    runtime = folder / "runtime"
    runtime.mkdir(mode=0o700)
    environment = dict(os.environ, PHYSICAL_FRAME_DIAGNOSTICS="0", PHYSICAL_STATE_PATH=str(folder / "progress.json"), PHYSICAL_TESTING="1", QT_NO_XDG_DESKTOP_PORTAL="1", QT_QPA_PLATFORMTHEME="")
    preview = environment.get("PHYSICAL_WIPE_PREVIEW_DIR")
    if preview:
        pathlib.Path(preview).mkdir(parents=True, exist_ok=True)
    if not os.environ.get("PHYSICAL_TEST_WAYLAND"):
        environment.update(QT_QPA_PLATFORM="offscreen", QT_QUICK_BACKEND="software", XDG_RUNTIME_DIR=str(runtime), DBUS_SESSION_BUS_ADDRESS="unix:path=/dev/null")
        environment.pop("WAYLAND_DISPLAY", None)
    environment.pop("HYPRLAND_INSTANCE_SIGNATURE", None)
    result = subprocess.run(["quickshell", "--no-color", "-p", str(folder / "shell.qml")], env=environment, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=20)
    if result.returncode or "WIPE PASS:" not in result.stdout or "WIPE FAILED" in result.stdout:
        print(result.stdout)
        raise SystemExit(result.returncode or 1)
    print(next(line.strip() for line in result.stdout.splitlines() if "WIPE PASS:" in line))
