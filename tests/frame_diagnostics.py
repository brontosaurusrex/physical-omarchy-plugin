#!/usr/bin/env python3
"""Capture an actual slow engine, with isolated assets/state and no input injection."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

project = Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory(prefix="physical-frame-diagnostics-") as directory:
    root = Path(directory)
    data = root / "assets"
    data.mkdir()
    for source in project.glob("*.qml"):
        shutil.copy2(source, data / source.name)
    for name in ("theme", "sounds", "fonts", "engine", "packaging"):
        shutil.copytree(project / name, data / name)
    (data / "native").mkdir()
    shutil.copy2(project / "native/libphysicalrenderer.so", data / "native/libphysicalrenderer.so")
    (data / "native/qmldir").write_text("plugin physicalrenderer .\n")
    (data / "bin").mkdir()
    source = data / "engine/native.go"
    text = source.read_text().replace('"strings"', '"strings"\n"time"', 1)
    assert 'case "tick":' in text
    source.write_text(text.replace('case "tick":', 'case "tick":\n        time.Sleep(55 * time.Millisecond)', 1))
    environment = dict(os.environ, GOCACHE=os.environ.get("GOCACHE", "/tmp/physical-go-cache"))
    subprocess.run(["go", "build", "-buildvcs=false", "-o", str(data / "bin/physical-engine"), "."],
                   cwd=data / "engine", env=environment, check=True, timeout=60)
    qml = data / "Standalone.qml"
    checks = '''
    property double diagnosticStarted: Date.now()
    Timer { interval: 200; repeat: true; running: true; onTriggered: {
        if (game.errorText || game.renderError) { console.error("DIAGNOSTICS FAILED", game.errorText, game.renderError); Qt.exit(1); return; }
        if (!game.engineReady) return;
        if (!game.state.auto) game.shortcut("o");
        if (game.state.ready || game.state.paused) game.shortcut(" ");
        if (Date.now() - window.diagnosticStarted > 18500) { game.close(); Qt.quit(); }
    } }
'''
    qml.write_text(qml.read_text().rstrip()[:-1] + checks + "\n}\n")
    runtime = root / "runtime"
    runtime.mkdir(mode=0o700)
    environment.update(QT_QPA_PLATFORM="offscreen", QT_QUICK_BACKEND="software",
                       QT_QPA_PLATFORMTHEME="", QT_FORCE_STDERR_LOGGING="1",
                       XDG_RUNTIME_DIR=str(runtime), PHYSICAL_STATE_PATH=str(root / "progress.json"),
                       PHYSICAL_FRAME_DIAGNOSTICS="1", PHYSICAL_DIAGNOSTICS_DIR=str(root / "diagnostics"))
    run = subprocess.run([str(project / "bin/physical"), "--data-dir", str(data)],
                         env=environment, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=25)
    if run.returncode or not (root / "diagnostics/latest.json").exists():
        print(run.stdout)
        raise SystemExit(run.returncode or "No slow-frame capture")
    report = json.loads((root / "diagnostics/latest.json").read_text())
    samples = report["samples"]
    active = [s for s in samples if s["active_gameplay"]]
    assert len(samples) >= 14, report
    assert report["post_samples_remaining"] == 0, report
    assert any(s["painted_fps"] < 40 and s["tick_latency_max_ms"] >= 50 for s in active), active
    assert any(s.get("engine_work_mean_ms", 0) >= 50 and s.get("transport_timed_ticks", 0) > 0 for s in active), active
    assert all("commands_peak_unacknowledged" in s and "qt_write_peak_pending_bytes" in s for s in samples)
    assert all("submitted_fps" in s and "accept_frame_mean_ms" in s and "mouse_locked" in s for s in samples)
    assert (root / "diagnostics/last-session.json").exists()
    assert "DIAGNOSTICS FAILED" not in run.stdout, run.stdout
    # This test checks the recorder. The separate UI smoke test owns theme/menu
    # compatibility checks; successful capture must not conceal its warnings.
    ui_errors = sorted({line for line in run.stdout.splitlines() if "TypeError" in line or "ReferenceError" in line})
    if ui_errors:
        print("Existing UI warnings during the recorder test:")
        print("\n".join(ui_errors))
    print("PASS: real QML/engine slowdown captured with timing counters, ten post-trigger samples and close flush")
