#!/usr/bin/env python3
"""Check pace, saved preferences and speed-specific records through the engine."""
from contextlib import contextmanager
import json
import math
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

project = Path(__file__).resolve().parents[1]
linear = """gravity=0
airDrag=0
magnusCoefficient=0
stuckSpeedThreshold=0
ballRescueEnabled=false
ballX=300
ballY=450
ballVx=120
ballVy=0
brickWidth=100
brickHeight=40
brickOffsetTop=50
debrisPiecesMin=0
debrisPiecesMax=0
title="SPEED TEST"
---
#
"""
completion = linear.replace("ballX=300", "ballX=900").replace("ballY=450", "ballY=270").replace("ballVx=120", "ballVx=0").replace("ballVy=0", "ballVy=-300").replace("---\n#\n", "---\n#\n#\n")
speeds = {"6": 80, "7": 90, "8": 100, "9": 150, "0": 200}

with tempfile.TemporaryDirectory(prefix="physical-game-speed-") as directory:
    root = Path(directory)
    engine = root / "engine"
    shutil.copytree(project / "engine", engine)
    for path in (engine / "levels").glob("level*.txt"):
        path.unlink()
    binary = root / "engine-test"
    environment = dict(os.environ, GOCACHE=os.environ.get("GOCACHE", "/tmp/physical-go-cache"))

    def build(level):
        for number in (1, 2):
            (engine / "levels" / f"level{number}.txt").write_text(level)
        subprocess.run(["go", "build", "-buildvcs=false", "-o", str(binary), "."],
                       cwd=engine, env=environment, check=True, timeout=60)

    @contextmanager
    def running(name):
        state_path = root / (name + ".json")
        process = subprocess.Popen([str(binary)], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                   stderr=subprocess.PIPE, text=True,
                                   env=dict(environment, PHYSICAL_STATE_PATH=str(state_path)))
        def read():
            line = process.stdout.readline()
            assert line, "Engine stopped: " + process.stderr.read()
            frame = json.loads(line)
            assert "error" not in frame, frame
            return frame
        def send(action, **values):
            process.stdin.write(json.dumps(dict(action=action, **values)) + "\n")
            process.stdin.flush()
            return read()
        def key(value, **values):
            return send("key", key=value, down=True, **values)
        try:
            yield read(), send, key, state_path
        finally:
            process.communicate('{"action":"quit"}\n', timeout=5)
            assert process.returncode == 0

    build(linear)
    for key_name, percent in speeds.items():
        with running("pace-" + str(percent)) as (first, send, key, state_path):
            assert first["state"]["gameSpeed"] == 100
            assert key(key_name)["state"]["status"] == f"@{percent}%"
            assert key("8", repeat=True)["state"]["gameSpeed"] == percent
            assert send("key", key="8", down=False)["state"]["gameSpeed"] == percent
            first = key(" ")
            steps = 0
            for _ in range(60):
                frame = send("tick", dt=1/60)
                steps += frame["state"]["physicsSteps"]
            state = frame["state"]
            assert steps == round(240 * percent / 100), (percent, steps)
            assert math.isclose(state["x"] - first["state"]["x"], 120 * percent / 100, abs_tol=1e-6), state
            assert math.isclose(state["levelTime"], 1, abs_tol=1e-6), state
            assert math.isclose(state["fps"], 60, abs_tol=.01), state
            assert state["recordEligible"] and not state["mixedSpeed"]
            assert key(" ")["state"]["paused"]
            frozen = send("tick", dt=.1)["state"]
            assert frozen["x"] == state["x"] and frozen["levelTime"] == state["levelTime"]
            assert key("k")["state"]["replay"]
            assert key("0")["state"]["gameSpeed"] == percent, "Speed shortcut changed replay"
            send("replay-exit")
            key("u")
            assert key("PageDown")["state"]["gameSpeed"] == percent
            assert send("restart")["state"]["gameSpeed"] == percent
        with running("pace-" + str(percent)) as (first, send, key, state_path):
            assert first["state"]["gameSpeed"] == percent, "Saved pace was lost on reopen"

    for value in ("0", "91", "NaN", "999"):
        (root / "invalid.json").write_text(json.dumps({"physical.gameSpeedPercent": value}))
        with running("invalid") as (first, *_):
            assert first["state"]["gameSpeed"] == 100

    for previous, current in ((110,150),(120,200)):
        record = f"breakout.best.v1.level.1.timeMs.at{previous}"
        (root / "migrate.json").write_text(json.dumps({"physical.gameSpeedPercent":str(previous),record:"123"}))
        with running("migrate") as (first, send, key, state_path):
            assert first["state"]["gameSpeed"] == current
        saved=json.loads(state_path.read_text())
        assert saved["physical.gameSpeedPercent"] == str(current) and saved[record]=="123"

    build(completion)
    def until_complete(send):
        for _ in range(120):
            frame = send("tick", dt=1/60)
            if frame["state"]["complete"]:
                return frame
        raise AssertionError(("Tiny level never completed", frame["state"]))

    # Retain a legacy 100% best; custom speeds must write different keys.
    records = root / "records.json"
    legacy = "breakout.best.v1.level.1.timeMs"
    records.write_text(json.dumps({legacy: "1"}))
    for key_name in ("6", "7", "8", "9", "0"):
        percent = speeds[key_name]
        with running("records") as (first, send, key, state_path):
            if first["state"]["level"] != 1:
                key("PageUp")
            key(key_name)
            key(" ")
            completed = until_complete(send)
            texts = [op[1] for op in completed["ops"] if op[0] == "fillText"]
            assert f"LEVEL COMPLETE @{percent}%" in texts, texts
            state = key(" ")["state"]
            assert state["level"] == 2 and state["gameSpeed"] == percent
            key(" ")
            until_complete(send)
            win = key(" ")
            assert win["state"]["win"]
            assert any(f"@{percent}%" in op[1] for op in win["ops"] if op[0] == "fillText")
        saved = json.loads(records.read_text())
        suffix = "" if percent == 100 else f".at{percent}"
        assert legacy + suffix in saved, saved
        assert "breakout.best.v1.fullGame.2.timeMs" + suffix in saved, saved
        assert saved[legacy] == "1", "Custom pace replaced a legacy 100% record"

    with running("mixed-level") as (first, send, key, state_path):
        key("6")
        key(" ")
        send("tick", dt=.05)
        assert key("0")["state"]["mixedSpeed"]
        assert not key("6")["state"]["recordEligible"], "Restoring pace made a mixed run eligible"
        completed = until_complete(send)
        assert any("MIXED SPEED" in op[1] for op in completed["ops"] if op[0] == "fillText")
        saved = json.loads(state_path.read_text())
        assert not any(k.startswith("breakout.best.") for k in saved), saved

    with running("mixed-game") as (first, send, key, state_path):
        key("6")
        key(" ")
        until_complete(send)
        key("7")  # Select the next level's pace on the results screen.
        assert key(" ")["state"]["gameSpeed"] == 90
        key(" ")
        assert not until_complete(send)["state"]["mixedSpeed"], "Fixed-speed level was marked mixed"
        win = key(" ")
        assert any("MIXED SPEED" in op[1] for op in win["ops"] if op[0] == "fillText")
        saved = json.loads(state_path.read_text())
        assert not any("fullGame" in k for k in saved), saved
        assert "breakout.best.v1.level.1.timeMs.at80" in saved
        assert "breakout.best.v1.level.2.timeMs.at90" in saved

print("PASS: five pace ratios, fixed physics steps, real-time FPS/results, persistence, transitions, replay guards, per-speed records and mixed runs")
