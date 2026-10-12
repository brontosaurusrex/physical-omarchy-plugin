#!/usr/bin/env python3
"""Measure sustained gameplay in a real Quickshell using private progress."""
import os, pathlib, shutil, subprocess, tempfile, queue, signal, threading, time

duration = int(os.environ.get("PHYSICAL_PERF_SECONDS", "180"))
level = int(os.environ.get("PHYSICAL_PERF_LEVEL", "26"))
restart_seconds = int(os.environ.get("PHYSICAL_PERF_RESTART_SECONDS", "0"))
if restart_seconds and not 10 <= restart_seconds <= duration:
    raise SystemExit("Restart interval must be 0 or between 10 seconds and the test duration")
levels = [int(value) for value in os.environ.get("PHYSICAL_PERF_LEVELS", str(level)).split(",")]
if not levels or any(not 1 <= value <= 33 for value in levels):
    raise SystemExit("Levels must be a comma-separated list in 1–33")
level = levels[0]
stress = os.environ.get("PHYSICAL_PERF_STRESS", "1") != "0"
if not 10 <= duration <= 3600 or not 1 <= level <= 33:
    raise SystemExit("Duration must be 10–3600 seconds and level 1–33")
freeze = os.environ.get("PHYSICAL_PERF_INJECT_FREEZE", "")
if freeze not in ("", "ui", "engine"):
    raise SystemExit("Freeze injection must be ui or engine")
project = pathlib.Path(os.environ.get("PHYSICAL_PERF_PROJECT", str(pathlib.Path(__file__).resolve().parents[1]))).resolve()
shell_dir = pathlib.Path(os.environ.get("OMARCHY_SHELL_DIR", "/usr/share/omarchy/shell"))
harness = r'''
import QtQuick
import Quickshell
import qs.Commons
import "Port" as Port
ShellRoot {
    id: root
    property var levels: __LEVELS__
    property int levelIndex: 0
    readonly property int targetLevel: levels[levelIndex]
    property int lastReplies: 0
    property int seconds: 0
    property int frames: 0
    property bool started: false
    property int totalFrames: 0
    property real minimumFrames: 10000
    property real roundElapsed: 0
    property real elapsed: 0
    property double lastSample: 0
    property bool navigating: false
    property bool restarting: false
    property int maxDebris: 0
    Port.Physical { id: game }
    Component.onCompleted: {
        game.open("{}");
        if(Quickshell.env("PHYSICAL_PERF_DIALOG_TITLE")) game.windowView.title=Quickshell.env("PHYSICAL_PERF_DIALOG_TITLE");
    }
    Timer { interval: 1000; running: true; repeat: true; onTriggered: {
        if (game.errorText || game.renderError) { console.error("PERF FAILED", game.errorText, game.renderError); Qt.exit(1); return; }
        if (!game.opened) { console.error("PERF FAILED: game window closed"); Qt.exit(1); return; }
        if (!game.engineReady) return;
        if (!root.navigating) {
            if (Quickshell.env("PHYSICAL_PERF_MUTE") === "1") game.send({action:"volume", value:0});
            game.shortcut("u");
            for (var i = 1; i < __LEVEL__; ++i) game.shortcut("PageDown");
            root.navigating = true; return;
        }
        if (game.state.level !== root.targetLevel) { console.error("PERF wrong level", game.state.level); Qt.exit(1); return; }
        if (!root.started) {
            if (!game.state.auto) game.shortcut("o");
            if (game.state.ready || game.state.paused) game.shortcut(" ");
            if (__STRESS__) { game.shortcut("z"); game.shortcut("m"); game.shortcut("2"); game.shortcut("2"); }
            root.started = true; root.lastReplies = game.frameReplies; root.frames = game.paintedFrames; root.lastSample = Date.now(); return;
        }
        if (game.frameReplies === root.lastReplies) { console.error("PERF FAILED: engine replies stalled"); Qt.exit(1); return; }
        var replies = game.frameReplies - root.lastReplies;
        root.lastReplies = game.frameReplies;
        var interval = (Date.now() - root.lastSample) / 1000;
        var painted = game.paintedFrames - root.frames;
        if (painted === 0 && root.started) { console.error("PERF FAILED: rendering stalled"); Qt.exit(1); return; }
        root.elapsed += interval;
        root.roundElapsed += interval;
        root.lastSample = Date.now();
        root.totalFrames += game.paintedFrames - root.frames;
        root.minimumFrames = Math.min(root.minimumFrames, painted / interval);
        root.maxDebris = Math.max(root.maxDebris,game.state.debrisPieces || 0);
        console.log("PERF", ++root.seconds, "paintedFPS", (painted / interval).toFixed(2), "level", game.state.level, "replies", replies, "balls", game.state.balls, "replayFrames", game.state.replayFrames, "engineFPS", game.state.fps, "debris", game.state.debrisPieces, "remaining", game.state.remaining, "transition", game.state.complete || game.state.gameOver || game.state.ready || game.state.paused);
        if (Quickshell.env("PHYSICAL_RENDER_TIMINGS")) console.log("TIMING", root.seconds, "parse", game.state.qtParseMs, "cache", game.state.qtCacheMs, "compile", game.state.qtCompileMs, "brickPaint", game.state.qtBrickPaintMs, "frontPaint", game.state.qtFrontPaintMs);
        root.frames = game.paintedFrames;
        if (root.seconds === 10 && Quickshell.env("PHYSICAL_TEST_ARTIFACTS")) game.contentView.grabToImage(function(result) { result.saveToFile(Quickshell.env("PHYSICAL_TEST_ARTIFACTS") + "/physical-gpu-stress.png"); });
        if (root.restarting) {
            if (!game.state.auto) game.shortcut("o");
            if (game.state.ready || game.state.paused) game.shortcut(" ");
            if (__STRESS__) { game.shortcut("z"); game.shortcut("m"); game.shortcut("2"); game.shortcut("2"); }
            root.restarting = false;
        } else if (!root.restarting && (game.state.complete || game.state.gameOver || (__RESTART__ > 0 && root.roundElapsed >= __RESTART__))) {
            console.log("PERF restarting round", root.seconds, "remaining", game.state.remaining);
            root.levelIndex = (root.levelIndex + 1) % root.levels.length;
            game.send({action:"restart"});
            var difference = root.targetLevel - game.state.level;
            for(var step=0;step<Math.abs(difference);++step) game.shortcut(difference>0 ? "PageDown" : "PageUp");
            root.restarting = true; root.roundElapsed = 0;
        } else if (!root.restarting && (game.state.ready || game.state.paused)) game.shortcut(" ");
        if (root.elapsed >= __SECONDS__) { console.log("SMOKE PASS: sustained gameplay", "levels", root.levels, "stress", __STRESS__, "seconds", root.elapsed.toFixed(2), "average",root.totalFrames/root.elapsed,"minimum",root.minimumFrames,"max debris",root.maxDebris); Qt.exit(0); }
    } }
}
'''

harness = harness.replace("__LEVELS__", str(levels)).replace("__LEVEL__", str(level)).replace("__SECONDS__", str(duration)).replace("__STRESS__", "true" if stress else "false").replace("__RESTART__", str(restart_seconds))

with tempfile.TemporaryDirectory(prefix="physical-qml-test-") as directory:
    folder = pathlib.Path(directory)
    for name in ["Commons", "Ui"]:
        shutil.copytree(shell_dir / name, folder / name)
    shutil.copytree(project, folder / "Port", ignore=shutil.ignore_patterns(".git", "__pycache__"))
    (folder / "Port/native/qmldir").write_text("plugin physicalrenderer " + str(folder / "Port/native") + "\n")
    (folder / "shell.qml").write_text(harness)
    runtime = folder / "runtime"
    runtime.mkdir(mode=0o700)
    env = dict(os.environ, QT_QPA_PLATFORM="offscreen", QT_QUICK_BACKEND="software", QT_NO_XDG_DESKTOP_PORTAL="1", QT_QPA_PLATFORMTHEME="", XDG_RUNTIME_DIR=str(runtime), PHYSICAL_STATE_PATH=str(folder / "progress.json"), PHYSICAL_TESTING="1", DBUS_SESSION_BUS_ADDRESS="unix:path=/dev/null")
    if os.environ.get("PHYSICAL_TEST_WAYLAND"):
        for key in ["QT_QUICK_BACKEND", "QT_QPA_PLATFORM", "XDG_RUNTIME_DIR", "DBUS_SESSION_BUS_ADDRESS"]:
            env.pop(key, None)
            if key in os.environ: env[key] = os.environ[key]
    else:
        env.pop("WAYLAND_DISPLAY", None)
    env.pop("HYPRLAND_INSTANCE_SIGNATURE", None)
    # A reader thread lets the watchdog run even if Qt produces no more output.
    # Give deliberate freezes a unique window title so their compositor warning
    # can be removed without touching a real application's dialog.
    dialog_title="Physical watchdog test " + folder.name
    if freeze: env["PHYSICAL_PERF_DIALOG_TITLE"]=dialog_title
    run = subprocess.Popen(["quickshell", "--no-color", "-p", str(folder / "shell.qml")], env=env, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, start_new_session=True)
    messages = queue.Queue()
    def reader():
        for line in run.stdout: messages.put(line)
        messages.put(None)
    threading.Thread(target=reader, daemon=True).start()
    lines = []
    socketErrors = 0
    started = heartbeat = time.monotonic()
    have_heartbeat = injected = failed = False
    def stop_group():
        try:
            os.killpg(run.pid, signal.SIGCONT)
            os.killpg(run.pid, signal.SIGTERM)
            run.wait(timeout=2)
        except subprocess.TimeoutExpired:
            os.killpg(run.pid, signal.SIGKILL); run.wait(timeout=2)
        except ProcessLookupError:
            pass
    try:
        while True:
            now = time.monotonic()
            heartbeat_limit = (2 if injected and freeze == "ui" else 8) if have_heartbeat else 25
            if now-started > duration+60 or now-heartbeat > heartbeat_limit:
                print("PERF FAILED: external heartbeat watchdog timed out", flush=True)
                failed = True; break
            try: line = messages.get(timeout=.25)
            except queue.Empty: continue
            if line is None: break
            lines.append(line)
            if "PERF " in line and "paintedFPS" in line:
                heartbeat = time.monotonic(); have_heartbeat = True
                if freeze and not injected and "PERF 3 paintedFPS" in line:
                    target = run.pid
                    if freeze == "engine":
                        children = pathlib.Path(f"/proc/{run.pid}/task/{run.pid}/children").read_text().split()
                        target = next(int(child) for child in children if "physical-engine" in pathlib.Path(f"/proc/{child}/cmdline").read_text())
                    os.kill(target, signal.SIGSTOP); injected = True
                    print(f"PERF INJECT: stopped {freeze} process {target}", flush=True)
            if "QSocketNotifier:" in line:
                socketErrors += 1
                if socketErrors >= 20:
                    print("PERF FAILED: repeated Qt socket errors", flush=True)
                    failed = True; break
            if "PERF" in line or "TIMING" in line or "SMOKE PASS:" in line or os.environ.get("PHYSICAL_TEST_DEBUG"):
                print(line.strip(), flush=True)
        if run.poll() is None:
            if failed: stop_group()
            else: run.wait(timeout=5)
    finally:
        # Also terminates a stopped engine child after the Qt process exits.
        stop_group()
        if freeze:
            for entry in pathlib.Path("/proc").iterdir():
                if not entry.name.isdigit(): continue
                try:
                    if (entry/"comm").read_text().strip() != "hyprland-dialog": continue
                    command=(entry/"cmdline").read_bytes().decode(errors="replace")
                    if dialog_title in command: os.kill(int(entry.name),signal.SIGTERM)
                except (OSError,ProcessLookupError): pass
    bad = [line for line in lines if "WARN scene:" in line or "ERROR" in line or "QSocketNotifier:" in line]
    if failed or run.returncode or not any("SMOKE PASS:" in line for line in lines) or bad:
        if os.environ.get("PHYSICAL_TEST_DEBUG"): print("".join(lines))
        raise SystemExit(1)
