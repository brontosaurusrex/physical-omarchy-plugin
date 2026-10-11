#!/usr/bin/env python3
"""Exercise the real native engine protocol, physics, replay, theme, and persistence."""
import json, os, pathlib, subprocess, tempfile, time

project = pathlib.Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory(prefix="physical-native-test-") as directory:
    environment = dict(os.environ, PHYSICAL_STATE_PATH=directory + "/progress.json", PHYSICAL_EXPORT_DIR=directory + "/exports")
    process = subprocess.Popen([str(project / "bin/physical-engine")], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, env=environment)
    def read():
        line = process.stdout.readline()
        assert line, "Engine exited: " + process.stderr.read()
        packet = json.loads(line)
        assert "error" not in packet, packet.get("error")
        return packet
    def send(action, **values):
        process.stdin.write(json.dumps(dict(action=action, **values)) + "\n")
        process.stdin.flush()
        return read()
    def key(value): return send("key", key=value, down=True)

    first = read()
    assert first["state"]["levels"] == 33
    assert first["state"]["ready"] and first["caches"]
    assert first["state"]["volume"] == 0.5
    assert send("volume", value=0.25)["state"]["volume"] == 0.25
    assert send("volume", value=2)["state"]["volume"] == 1
    assert send("volume", value=-1)["state"]["volume"] == 0
    send("volume", value=0.25)
    original_background = first["ops"][0][2]
    colored = send("theme", theme=dict(background="#eeeeee", accent="#2266aa"))
    assert colored["ops"][0][2] == original_background, "Shell theme replaced game palette"
    backgrounds = {original_background}
    assert not key(" ")["state"]["ready"]
    initial = send("tick", dt=1/60)
    key("o")
    start = time.monotonic()
    for _ in range(360): frame = send("tick", dt=1/60)
    assert (frame["state"]["x"], frame["state"]["y"]) != (initial["state"]["x"], initial["state"]["y"])
    assert frame["state"]["replayFrames"] > 10
    assert frame["state"]["physicsSteps"] == 4
    assert not frame["caches"], "Unchanged bricks must not be resent every frame"
    replay = key("k")
    assert replay["state"]["replay"]
    assert key("Home")["state"]["replayFrame"] == 0
    assert key("End")["state"]["replayFrame"] > 0
    key("v")
    exports = list(pathlib.Path(directory, "exports").glob("*.svg"))
    assert exports and "<svg" in exports[0].read_text()
    assert not key("Escape")["state"]["replay"]
    hidden = send("visibility", down=False)
    assert hidden["state"]["paused"]
    frozen = send("tick", dt=1)
    assert (hidden["state"]["x"], hidden["state"]["y"]) == (frozen["state"]["x"], frozen["state"]["y"])
    send("visibility", down=True)
    assert key("e")["state"]["editor"]
    send("setting", group="physics", key="gravity", value=300)
    assert "gravity=300" in send("tuner-export")["clipboard"]
    assert not key("e")["state"]["editor"]
    key("u")
    for expected in range(2, 34):
        selected = key("PageDown")
        assert selected["state"]["level"] == expected
        assert selected["width"] > 0 and selected["height"] > 0
        backgrounds.add(selected["ops"][0][2])
        key(" ")
        for _ in range(5): send("tick", dt=1/60)
    assert len(backgrounds) > 1, "Level palettes were lost"
    assert pathlib.Path(directory, "progress.json").exists()
    send("tuner-reset")
    process.stdin.write('{"action":"quit"}\n'); process.stdin.flush()
    process.wait(timeout=5)
    assert process.returncode == 0
    persisted = subprocess.Popen([str(project / "bin/physical-engine")], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, env=environment)
    assert json.loads(persisted.stdout.readline())["state"]["volume"] == 0.25
    persisted.communicate('{"action":"quit"}\n', timeout=5)
    assert persisted.returncode == 0
    log = process.stderr.read()
    assert "Panic" not in log and "panic" not in log, log
    print(f"PASS: 33 levels, native 240 Hz physics, replay, SVG, level palettes, saved volume, tuner, pause, progression ({time.monotonic()-start:.2f}s)")
