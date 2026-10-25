#!/usr/bin/env python3
"""Earn a brick-flip power-up through real collisions in a private level."""
import json
import math
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

project = Path(__file__).resolve().parents[1]
disabled = "\n".join("enable" + name + "=false" for name in [
    "LowGravity", "PassThrough", "Nuke", "ReverseGravity", "DualBalls", "BlackHole",
    "Magnet", "Influencer", "Zapper", "BreakUnbreakable", "BigPaddle"])
level = disabled + """
enableBrickFlip=true
powerUpDuration=0.35
gravity=0
airDrag=0
magnusCoefficient=0
ballX=900
ballY=270
ballVx=0
ballVy=-300
brickWidth=60
brickHeight=30
brickPadding=10
brickOffsetTop=80
debrisPiecesMin=0
debrisPiecesMax=0
title="MAGIC FLIP TEST"
---
# U
#
 M
"""

with tempfile.TemporaryDirectory(prefix="physical-magic-flip-") as directory:
    root = Path(directory)
    engine = root / "engine"
    shutil.copytree(project / "engine", engine)
    (engine / "levels/level1.txt").write_text(level)
    (engine / "levels/level2.txt").write_text(level.replace("enableBrickFlip=true\n", ""))
    (engine / "levels/level3.txt").write_text(level.replace("enableBrickFlip=true", "enableBrickFlip=false"))
    binary = root / "engine-test"
    environment = dict(os.environ, GOCACHE=os.environ.get("GOCACHE", "/tmp/physical-go-cache"),
                       PHYSICAL_STATE_PATH=str(root / "progress.json"))
    subprocess.run(["go", "build", "-buildvcs=false", "-o", str(binary), "."],
                   cwd=engine, env=environment, check=True, timeout=60)
    process = subprocess.Popen([str(binary)], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                               stderr=subprocess.PIPE, text=True, env=environment)
    caches = {}
    def read():
        line = process.stdout.readline()
        assert line, "Engine stopped: " + process.stderr.read()
        packet = json.loads(line)
        assert "error" not in packet, packet
        caches.update(packet["caches"])
        for op in packet["ops"]:
            if op[0] == "cache":
                batch = next(op for op in caches[str(op[1])] if op[0] == "bricks")
                packet["bricks"] = [batch[1][i:i+6] for i in range(0, len(batch[1]), 6)]
        return packet
    def send(action, **values):
        process.stdin.write(json.dumps(dict(action=action, **values)) + "\n")
        process.stdin.flush()
        return read()
    def key(value): return send("key", key=value, down=True)
    def hit_magic():
        for _ in range(90):
            packet = send("tick", dt=1/60)
            if packet["state"]["remaining"] == 2:
                return packet
        raise AssertionError(("Magic brick was not hit", packet["state"]))
    def finish_flip(packet):
        for _ in range(180):
            if not packet["state"]["flipAnimating"]:
                return packet
            packet = send("tick", dt=1/60)
        raise AssertionError("Magic wipe did not finish")
    try:
        before = read()
        mirror = min(b[1] for b in before["bricks"]) + max(b[1]+b[3] for b in before["bricks"])
        key(" ")
        activation = hit_magic()
        assert activation["state"]["flipAnimating"] and activation["state"]["status"] == "Brick Flip!"
        assert not activation["state"]["flipReverse"], "Power-up did not wipe left to right"
        assert any(s.get("feature") == "brickflip" for s in activation.get("sounds", [])), activation.get("sounds")
        flipped = finish_flip(activation)
        assert flipped["state"]["flipped"] and flipped["state"]["flipPowerActive"]
        assert flipped["state"]["recordEligible"], "Earned power-up marked a normal run assisted"
        survivors = [b for b in before["bricks"] if int(b[5]) & 1 == 0]
        assert len(survivors) == len(flipped["bricks"])
        for a, b in zip(survivors, flipped["bricks"]):
            expected = [a[0], mirror-a[1]-a[3], a[2], a[3], -a[4], a[5]]
            assert all(math.isclose(x, y, abs_tol=1e-6) for x, y in zip(expected, b)), (a, b)
        live = flipped["bricks"]
        key(" ")
        assert send("tick", dt=.1)["state"]["flipTimer"] == flipped["state"]["flipTimer"], "Pause consumed the power-up timer"
        key("k")
        replay = key("End")["bricks"]
        assert len(replay) == len(live) and all(
            math.isclose(x, y, abs_tol=1e-4) for a, b in zip(replay, live) for x, y in zip(a, b)
        ), ("Replay lost the earned flip", replay, live)
        assert send("replay-exit")["bricks"] == live
        key(" ")
        for _ in range(120):
            expired = send("tick", dt=1/60)
            if not expired["state"]["flipPowerActive"]:
                break
        else:
            raise AssertionError("Magic flip did not expire")
        assert expired["state"]["flipAnimating"], "Expiry did not use the return wipe"
        assert expired["state"]["flipReverse"], "Expiry did not wipe right to left"
        restored = finish_flip(expired)
        assert not restored["state"]["flipped"] and restored["state"]["recordEligible"]
        for a,b in zip(survivors,restored["bricks"]):
            assert all(math.isclose(x,y,abs_tol=1e-6) for x,y in zip(a,b)), "Return wipe changed original geometry"
        key(" ")
        assert not key("y")["state"]["recordEligible"], "Manual Y lost its assistance guard"
        key("u")
        assert not key("PageDown")["state"]["flipped"], "Flip leaked into the next level"
        key(" ")
        no_flip = hit_magic()
        assert not no_flip["state"]["flipped"] and not no_flip["state"]["flipAnimating"], "Default game still awards a flip"
        key("PageDown")
        key(" ")
        no_flip = hit_magic()
        assert not no_flip["state"]["flipped"] and not no_flip["state"]["flipAnimating"], "Disabled flip power-up still activated"
    finally:
        process.communicate('{"action":"quit"}\n', timeout=5)
    assert process.returncode == 0
print("PASS: earned flip/wipe, timed return, paused timer, exact geometry, normal record eligibility, sound event, replay, manual assistance, next-level reset and disabled flag")
