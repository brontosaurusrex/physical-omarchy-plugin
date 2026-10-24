#!/usr/bin/env python3
"""Check vertical brick flips and replay caches through the real engine."""
import json
import math
import os
from pathlib import Path
import subprocess
import tempfile
import xml.etree.ElementTree as ET

project = Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory(prefix="physical-brick-flip-") as directory:
    environment = dict(os.environ, PHYSICAL_STATE_PATH=directory + "/progress.json", PHYSICAL_EXPORT_DIR=directory + "/exports")
    process = subprocess.Popen([str(project / "bin/physical-engine")], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, env=environment)
    caches = {}

    def read():
        line = process.stdout.readline()
        assert line, "Engine stopped"
        packet = json.loads(line)
        assert "error" not in packet, packet
        caches.update(packet["caches"])
        references = [op[1] for op in packet["ops"] if op[0] == "cache"]
        if len(references) == 1:
            packet["_brick_batch"] = next(op for op in caches[str(references[0])] if op[0] == "bricks")
        return packet

    def send(action, **values):
        process.stdin.write(json.dumps(dict(action=action, **values)) + "\n")
        process.stdin.flush()
        return read()

    def key(value, **values):
        return send("key", key=value, down=True, **values)

    def batch(packet):
        return packet["_brick_batch"]

    unchanged = ("level", "levels", "lives", "score", "paused", "ready", "gameOver", "win", "complete", "balls", "remaining", "auto", "sound", "volume", "palette", "corner", "allCorners", "x", "y", "omega", "paddleX", "fpsVisible", "diagnostics", "unlocked")

    def check_flip(before, after):
        assert all(before["state"][k] == after["state"][k] for k in unchanged), (before["state"], after["state"])
        original, flipped = batch(before), batch(after)
        assert original[2:] == flipped[2:], "Brick styling changed"
        a, b = original[1], flipped[1]
        assert len(a) == len(b) and a
        mirror = min(a[1::6]) + max(a[i + 1] + a[i + 3] for i in range(0, len(a), 6))
        for i in range(0, len(a), 6):
            expected = [a[i], mirror - a[i + 1] - a[i + 3], a[i + 2], a[i + 3], -a[i + 4], a[i + 5]]
            assert all(math.isclose(x, y, abs_tol=1e-6) for x, y in zip(b[i:i + 6], expected)), (i, a[i:i + 6], b[i:i + 6])

    try:
        frame = read()
        assert not frame["state"]["flipped"]
        key("u")
        frame = send("hello")
        for level in range(1, frame["state"]["levels"] + 1):
            assert frame["state"]["level"] == level and not frame["state"]["flipped"]
            original = batch(frame)
            flipped = key("y")
            assert flipped["state"]["flipped"]
            check_flip(frame, flipped)
            assert key("y", repeat=True)["state"]["flipped"], "Key repeat toggled the layout"
            restored = key("Y")
            assert not restored["state"]["flipped"] and batch(restored) == original, "Two flips changed the original layout"
            if level < frame["state"]["levels"]:
                key("y")
                frame = key("PageDown")

        key("PageDown")  # Wrap to level 1.
        key(" ")
        frame = send("tick", dt=1 / 60)
        flipped = key("y")
        check_flip(frame, flipped)
        live = batch(flipped)
        assert key("k")["state"]["replay"]
        old = key("Home")
        check_flip(old, key("End"))
        assert batch(key("Home")) != batch(key("End")), "Replay cache ignored orientation changes"
        key("Home")
        key("v")
        original_svg = next(Path(directory, "exports").glob("*.svg")).read_text()
        key("End")
        key("v")
        assert len(list(Path(directory, "exports").glob("*.svg"))) == 2
        def rects(svg):
            return [node.attrib for node in ET.fromstring(svg).iter("{http://www.w3.org/2000/svg}rect")]
        assert any(rects(p.read_text()) != rects(original_svg) for p in Path(directory, "exports").glob("*.svg")), "SVG lost the flipped geometry"
        before = key("End")
        assert batch(key("y")) == batch(before), "Flip mutated replay"
        assert batch(send("replay-exit")) == live, "Replay changed the live layout"
        assert key("e")["state"]["editor"]
        assert key("y")["state"]["flipped"], "Flip ran inside the physics editor"
    finally:
        process.communicate('{"action":"quit"}\n', timeout=5)
    assert process.returncode == 0
    print("PASS: all 33 levels mirror only bricks; double flip, gameplay state, repeat guard, replay, SVG and editor guard")
