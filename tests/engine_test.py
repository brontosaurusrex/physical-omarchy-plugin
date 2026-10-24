#!/usr/bin/env python3
"""Exercise the real native engine protocol, physics, replay, theme, and persistence."""
import json, math, os, pathlib, subprocess, tempfile, time

project = pathlib.Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory(prefix="physical-native-test-") as directory:
    environment = dict(os.environ, PHYSICAL_STATE_PATH=directory + "/progress.json", PHYSICAL_EXPORT_DIR=directory + "/exports")
    process = subprocess.Popen([str(project / "bin/physical-engine")], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, env=environment)
    caches = {}
    def read():
        line = process.stdout.readline()
        assert line, "Engine exited: " + process.stderr.read()
        packet = json.loads(line)
        assert "error" not in packet, packet.get("error")
        caches.update(packet["caches"])
        return packet
    def send(action, **values):
        process.stdin.write(json.dumps(dict(action=action, **values)) + "\n")
        process.stdin.flush()
        return read()
    def key(value): return send("key", key=value, down=True)

    first = read()
    assert first["state"]["levels"] == 33
    assert first["state"]["ready"] and first["caches"]
    assert first["state"]["volume"] == 0.25
    assert any(op[:2] == ["fillText", "P H Y S I C A L"] for op in first["ops"]), "First-level title was lost"
    assert first["room"]["name"] == "space" and first["room"]["dry"] == .5
    assert send("volume", value=0.37)["state"]["volume"] == 0.37
    assert send("volume", value=2)["state"]["volume"] == 1
    assert send("volume", value=-1)["state"]["volume"] == 0
    send("volume", value=0.37)
    original_background = first["ops"][0][2]
    colored = send("theme", theme=dict(background="#eeeeee", accent="#2266aa"))
    assert colored["ops"][0][2] == original_background, "Shell theme replaced game palette"
    backgrounds = {original_background}
    hidden_ready = send("visibility", down=False)
    assert hidden_ready["state"]["ready"] and hidden_ready["state"]["paused"]
    send("visibility", down=True)
    resumed = key(" ")["state"]
    assert not resumed["ready"] and not resumed["paused"], "Reopened READY game needs more than one start key"
    initial = send("tick", dt=1/60)
    key("o")
    start = time.monotonic()
    for _ in range(360): frame = send("tick", dt=1/60)
    assert (frame["state"]["x"], frame["state"]["y"]) != (initial["state"]["x"], initial["state"]["y"])
    assert frame["state"]["replayFrames"] > 10
    assert frame["state"]["physicsSteps"] == 4
    assert not frame["caches"], "Unchanged bricks must not be resent every frame"
    assert "room" not in frame, "Unchanged room settings must not restart ramps every frame"
    def brick_batch(packet):
        references = [op[1] for op in packet["ops"] if op[0] == "cache"]
        assert len(references) == 1, "Frame bypassed cached antialiased bricks"
        commands = caches[str(references[0])]
        batches = [op for op in commands if op[0] == "bricks"]
        assert len(batches) == 1
        return references[0], batches[0]
    live_cache, live_batch = brick_batch(frame)
    replay = key("k")
    assert replay["state"]["replay"]
    replay_cache, replay_batch = brick_batch(replay)
    assert live_cache != replay_cache, "Replay overwrote the live brick cache"
    assert replay_batch[2:] == live_batch[2:]
    assert len(replay_batch[1]) == len(live_batch[1])
    assert all(math.isclose(a, b, abs_tol=1e-4) for a, b in zip(replay_batch[1], live_batch[1])), "Replay bricks differ from the recorded live frame"
    assert not send("tick", dt=1/60)["caches"], "Paused replay rebuilt unchanged brick meshes"
    home = key("Home")
    assert home["state"]["replayFrame"] == 0
    assert brick_batch(home)[0] == replay_cache, "Scrubbing bypassed the replay cache"
    assert len(brick_batch(home)[1][1]) >= len(replay_batch[1]), "Rewind lost recorded bricks"
    assert key("End")["state"]["replayFrame"] > 0
    key("v")
    exports = list(pathlib.Path(directory, "exports").glob("*.svg"))
    assert exports and "<svg" in exports[0].read_text()
    restored = key("Escape")
    assert not restored["state"]["replay"]
    assert brick_batch(restored) == (live_cache, live_batch), "Replay exit changed live bricks"
    assert (restored["state"]["x"], restored["state"]["y"], restored["state"]["score"]) == (frame["state"]["x"], frame["state"]["y"], frame["state"]["score"]), "Replay changed the live session"
    hidden = send("visibility", down=False)
    assert hidden["state"]["paused"]
    frozen = send("tick", dt=1)
    assert (hidden["state"]["x"], hidden["state"]["y"]) == (frozen["state"]["x"], frozen["state"]["y"])
    send("visibility", down=True)
    assert key("e")["state"]["editor"]
    send("setting", group="physics", key="gravity", value=300)
    assert "gravity=300" in send("tuner-export")["clipboard"]
    assert not key("e")["state"]["editor"]
    if frame["state"]["auto"]: key("o")
    key("u")
    for expected in range(2, 34):
        selected = key("PageDown")
        assert selected["state"]["level"] == expected
        assert selected["width"] > 0 and selected["height"] > 0
        if expected in (23,26,27):
            assert selected["room"]["name"] == {23:"cave",26:"matrix",27:"space"}[expected]
            assert selected["room"]["dry"] == {23:.1,26:.6,27:.5}[expected]
        backgrounds.add(selected["ops"][0][2])
        started = key(" ")
        if started["state"]["paused"]: key(" ")
        for _ in range(5): live = send("tick", dt=1/60)
        if expected in (23, 26, 27):
            live_id, live_bricks = brick_batch(live)
            dense_replay = key("k")
            replay_id, replay_bricks = brick_batch(dense_replay)
            assert dense_replay["state"]["replay"], (expected, dense_replay["state"])
            assert live_id != replay_id and replay_bricks[2:] == live_bricks[2:], (expected, live_id, replay_id, live_bricks[2:], replay_bricks[2:])
            assert len(replay_bricks[1]) == len(live_bricks[1]), "Dense replay lost bricks"
            assert all(math.isclose(a, b, abs_tol=1e-4) for a, b in zip(replay_bricks[1], live_bricks[1]))
            assert not send("tick", dt=1/60)["caches"]
            key("Home")
            key("End")
            returned = send("replay-exit")
            assert brick_batch(returned) == (live_id, live_bricks)

    assert len(backgrounds) > 1, "Level palettes were lost"
    assert pathlib.Path(directory, "progress.json").exists()
    send("tuner-reset")
    process.stdin.write('{"action":"quit"}\n'); process.stdin.flush()
    process.wait(timeout=5)
    assert process.returncode == 0
    persisted = subprocess.Popen([str(project / "bin/physical-engine")], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, env=environment)
    assert json.loads(persisted.stdout.readline())["state"]["volume"] == 0.37
    persisted.communicate('{"action":"quit"}\n', timeout=5)
    assert persisted.returncode == 0
    log = process.stderr.read()
    assert "Panic" not in log and "panic" not in log, log
    print(f"PASS: 33 levels, native 240 Hz physics, replay, SVG, level palettes, saved volume, tuner, pause, progression ({time.monotonic()-start:.2f}s)")
