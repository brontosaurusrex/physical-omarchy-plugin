#!/usr/bin/env python3
"""Exercise endless demo transitions through the native engine protocol."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

project = Path(__file__).resolve().parents[1]
tiny = '''gravity=0
ballX=900
ballY=170
ballVx=0
ballVy=-300
brickWidth=100
brickHeight=40
brickPadding=10
brickOffsetTop=50
debrisPiecesMin=0
debrisPiecesMax=0
title="DEMO TEST"
---
#
'''
with tempfile.TemporaryDirectory(prefix="physical-auto-demo-") as directory:
    root = Path(directory)
    shutil.copytree(project / "engine", root / "engine")
    for level in (root / "engine/levels").glob("level*.txt"):
        level.write_text(tiny)
    # Lose all lives immediately on one private level, without test-only commands.
    (root / "engine/levels/level11.txt").write_text(tiny.replace("ballY=170", "ballY=1200").replace("ballVy=-300", "ballVy=300"))
    environment = dict(os.environ, GOCACHE=os.environ.get("GOCACHE", "/tmp/physical-go-cache"), PHYSICAL_STATE_PATH=str(root / "progress.json"))
    binary = root / "engine-test"
    subprocess.run(["go", "build", "-buildvcs=false", "-o", str(binary), "."], cwd=root / "engine", env=environment, check=True, timeout=60)
    process = subprocess.Popen([str(binary)], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, env=environment)
    def read():
        line = process.stdout.readline()
        assert line, "Engine stopped"
        packet = json.loads(line)
        assert "error" not in packet, packet
        return packet["state"]
    def send(action, **values):
        process.stdin.write(json.dumps(dict(action=action, **values)) + "\n")
        process.stdin.flush()
        return read()
    def key(value): return send("key", key=value, down=True)
    def tick(n=1):
        for _ in range(n): state = send("tick", dt=.1)
        return state
    def until(predicate, limit=80):
        for _ in range(limit):
            state = tick()
            if predicate(state): return state
        raise AssertionError(state)
    try:
        assert read()["ready"]
        key(" ")
        until(lambda s: s["complete"])
        assert tick(40)["complete"], "Manual results advanced automatically"
        send("pause")
        assert not key("o")["paused"], "Enabling demo left the results screen paused"
        assert tick(10)["level"] == 1, "Results were not briefly displayed"
        state = until(lambda s: s["level"] == 2)
        assert state["auto"] and not state["ready"]
        until(lambda s: s["complete"])
        assert key("k")["replay"]
        state = tick(30)
        assert state["level"] == 2 and state["replay"], "Demo advanced during replay"
        send("replay-exit")
        assert key("e")["editor"]
        state = tick(30)
        assert state["level"] == 2 and state["editor"], "Demo advanced in the editor"
        key("e")
        send("pause")
        state = tick(30)
        assert state["level"] == 2 and state["complete"] and state["paused"], state
        send("visibility", down=False)
        assert tick(30)["level"] == 2, "Hidden demo advanced"
        send("visibility", down=True)
        key("o")
        state = key(" ")
        assert state["level"] == 3 and state["ready"] and not state["auto"]
        key("u")
        for level in range(4, 14): assert key("PageDown")["level"] == level
        key("o")
        until(lambda s: s["complete"])
        state = until(lambda s: s["level"] != 13)
        assert state["level"] == 15 and state["auto"], state
        assert key("PageUp")["level"] == 13, "Reverse demo navigation stuck at skipped level"
        key("o")
        assert key("PageDown")["level"] == 14, "Manual play lost level 14"
        key("o")
        assert tick()["level"] == 15, "Enabling demo on level 14 failed to skip it"
        key("o")
        for level in range(14, 10, -1): assert key("PageUp")["level"] == level
        key("o")
        until(lambda s: s["gameOver"])
        state = until(lambda s: s["level"] == 1)
        assert state["auto"] and not state["gameOver"] and not state["ready"] and state["lives"] > 0, state
        key("o")
        for level in range(2, 34): assert key("PageDown")["level"] == level
        key(" ")
        until(lambda s: s["complete"])
        state = key(" ")
        assert state["win"] and tick(30)["win"], "Manual win restarted automatically"
        key("o")
        state = until(lambda s: s["level"] == 1)
        assert state["auto"] and not state["gameOver"], state
    finally:
        process.communicate('{"action":"quit"}\n', timeout=5)
    assert process.returncode == 0
    print("PASS: demo auto-advance, result delay, loss/win restart, level-14 skip, manual progression, pause and hidden guards")
