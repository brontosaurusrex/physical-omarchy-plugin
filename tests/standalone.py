#!/usr/bin/env python3
"""Check the real standalone host with private assets, progress and Qt input."""
import json
import os
import pathlib
import shutil
import subprocess
import tempfile

project = pathlib.Path(__file__).resolve().parents[1]
assets_source = pathlib.Path(os.environ.get("PHYSICAL_STANDALONE_DATA", str(project)))
app_binary = pathlib.Path(os.environ.get("PHYSICAL_STANDALONE_BINARY", str(project / "bin/physical")))
checks = r'''
    property int testPhase: 0
    property int testRounds: 0
    TestCase { id: input; parent: game.contentView; when: false }
    function find(item, name) {
        if (item.objectName === name) return item;
        for (var child of item.children || []) { var match = find(child, name); if (match) return match; }
        return null;
    }
    function fail(message) { console.error("STANDALONE FAILED", message, testPhase); Qt.exit(1); }
    Timer { interval: 200; repeat: true; running: true; onTriggered: {
        if (game.errorText || game.renderError) { fail(game.errorText || game.renderError); return; }
        if (++window.testRounds > 80) { fail("timeout"); return; }
        if (!game.engineReady || game.paintedFrames < 2) return;
        switch (window.testPhase) {
        case 0:
            if (!window.visible || game.state.volume !== .25 || game.state.level !== 1) { fail("initial window/state"); return; }
            input.keyClick(Qt.Key_Return); break;
        case 1:
            if (game.state.ready || game.state.paused) { fail("start"); return; }
            input.keyClick(Qt.Key_Question); break;
        case 2:
            if (!game.menuVisible || !find(game.contentView, "controls-table").visible || !find(game.contentView, "mouse-capture-hint").visible) { fail("controls menu"); return; }
            game.send({action:"volume", value:.37}); game.closeMenu(); game.shortcut("o"); break;
        case 3:
            if (game.state.volume !== .37) { fail("volume"); return; }
            game.shortcut("k"); break;
        case 4:
            if (!game.state.replay) { fail("replay"); return; }
            input.keyClick(Qt.Key_Escape); break;
        case 5:
            if (game.state.replay || !game.opened || !window.visible) { fail("replay Escape"); return; }
            input.keyPress(Qt.Key_Space); input.keyClick(Qt.Key_F); input.keyRelease(Qt.Key_Space); break;
        case 6:
            if (!window.fullscreen || window.visibility !== Window.FullScreen) { fail("fullscreen"); return; }
            input.keyPress(Qt.Key_Space); input.keyClick(Qt.Key_F); input.keyRelease(Qt.Key_Space); break;
        case 7:
            if (window.fullscreen || (window.visibility !== Window.Windowed && window.visibility !== Window.Maximized)) { fail("fullscreen restore: flag=" + window.fullscreen + " visibility=" + window.visibility + " opened=" + game.opened + " focus=" + game.contentView.activeFocus); return; }
            game.shortcut("e"); break;
        case 8:
            if (!game.state.editor) { fail("lab"); return; }
            game.send({action:"tuner-export"}); break;
        case 9:
            if (game.notice !== "Settings copied") { fail("clipboard"); return; }
            game.shortcut("e"); break;
        case 10:
            if (game.state.editor) { fail("lab exit"); return; }
            console.log("STANDALONE PASS: native Qt window, controls, volume, replay Escape, fullscreen, lab, clipboard, close");
            input.keyClick(Qt.Key_Escape); return;
        }
        window.testPhase++;
    } }
'''

with tempfile.TemporaryDirectory(prefix="physical standalone test ") as directory:
    root = pathlib.Path(directory)
    data = root / "share/physical-app"
    data.mkdir(parents=True)
    for source in assets_source.glob("*.qml"):
        shutil.copy2(source, data / source.name)
    for name in ["theme", "sounds", "fonts", "packaging"]:
        shutil.copytree(assets_source / name, data / name)
    (data / "native").mkdir()
    for name in ["qmldir", "libphysicalrenderer.so"]:
        shutil.copy2(assets_source / "native" / name, data / "native" / name)
    (data / "native/qmldir").write_text("plugin physicalrenderer .\n")
    (data / "bin").mkdir()
    shutil.copy2(assets_source / "bin/physical-engine", data / "bin/physical-engine")
    (root / "bin").mkdir()
    shutil.copy2(app_binary, root / "bin/physical")
    qml = (data / "Standalone.qml").read_text().replace("import QtQuick\n", "import QtQuick\nimport QtTest\n", 1)
    (data / "Standalone.qml").write_text(qml.rstrip()[:-1] + checks + "\n}\n")
    env = dict(os.environ, QT_FORCE_STDERR_LOGGING="1", XDG_DATA_HOME=str(root / "data"), OMARCHY_SHELL_DIR=str(root / "no-omarchy"))
    env.pop("PHYSICAL_STATE_PATH", None)
    if not os.environ.get("PHYSICAL_TEST_WAYLAND"):
        env.update(QT_QPA_PLATFORM="offscreen", QT_QUICK_BACKEND="software", QT_QPA_PLATFORMTHEME="")
    try:
        run = subprocess.run([str(root / "bin/physical")], env=env, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=25)
    except subprocess.TimeoutExpired as error:
        print((error.stdout or b"").decode(errors="replace"))
        raise
    bad = [line for line in run.stdout.splitlines() if any(word in line for word in ["FAILED", "ReferenceError", "TypeError", "Binding loop", "failed to load", "is not installed", "Error:"])]
    if run.returncode or "STANDALONE PASS:" not in run.stdout or bad:
        print(run.stdout)
        raise SystemExit(run.returncode or 1)
    state = root / "data/physical/progress.json"
    assert state.is_file(), "Standalone did not use its own progress directory"
    assert json.loads(state.read_text())["physical.audioVolume"] == "0.37", state.read_text()
    print(next(line for line in run.stdout.splitlines() if "STANDALONE PASS:" in line))
