# Physical for Omarchy

An Omarchy shell plugin and standalone Qt Linux app. Go runs Physical's physics engine as a separate
native process; Qt Quick/QML draws the game and provides the controls. No browser,
webview, WebAssembly, or web server is needed at runtime.

The controls follow Omarchy's active colors, fonts, spacing, and window rounding.
The playfield, effects, replay, and SVG exports retain Physical's original
per-level palettes and game font. Pause, fullscreen, and menu buttons sit above
the playfield; settings and a keyboard controls table are shown immediately in the menu.
The right-click mouse capture hint appears near the top, using the normal text color. Audio defaults to 25%
and saves your preference between sessions. A single native audio stream mixes
preloaded samples and generated collision tones, avoiding stream restarts during dense effect bursts.
The initial window is limited to 90% of the display width and height, keeps the
game aspect ratio, and opens centered horizontally directly beneath the top bar.
Qt uses the monitor scale for high-DPI rendering; the 1280-pixel window cap is in logical pixels.
The first level's Ready screen is titled **P H Y S I C A L**.

## Install

Requires current Omarchy with its Quickshell plugin API and Qt 6.9+ for accelerated OpenGL drawing.
On Omarchy/Arch, `./install` offers to install any missing build and audio packages
(`gcc`, `go`, `pkgconf`, `qt6-base`, `qt6-declarative`, `qt6-multimedia`, `qt6-wayland`, `wayland`, and `wayland-protocols`)
before building. The build has no external Go dependencies.

```bash
cd ~/source/physical-omarchy-plugin
./install
```

This installs a copy into `~/.config/omarchy/plugins/b.physical`, restarts the shell,
waits for plugin discovery, and enables a solid foreground-colored ball button
in the center section of the Omarchy bar. Re-run `./install` after editing the
source. The ball uses a smooth circle path and Omarchy's active color while
the game window is open. Restarting loads the new native module and resets an
active game and RAM replay. Omarchy does not accept symlinked plugin packages.

```bash
omarchy-shell shell toggle b.physical
omarchy plugin disable b.physical
```

Closing the panel retains the game and RAM replay, pauses it, and stops the frame
clock and sounds. Switching to another window pauses gameplay. Reopening requires
an explicit resume. A shell restart resets the live game and RAM replay.

## Standalone app and private-test bundles

After `./build`, run `./bin/physical` for a native Qt window without Omarchy or
Quickshell. The standalone controls use Qt's desktop palette; game colors stay
with the original level palettes. Your window manager handles placement and tiling.

`./package` creates two archives in `dist/`: an Omarchy plugin with compiled
runtime files and source, and a standalone Linux app. Extract the Omarchy bundle
and run `./install --prebuilt` to install without Go or a C++ compiler. Extract
the standalone bundle and run `./bin/physical` or `./install` for a user install
with a white-ball launcher icon. These bundles need compatible Qt/system libraries.

See [PACKAGING.md](PACKAGING.md) for transfer, installation, compatibility,
uninstall and the later marketplace publication steps. Packaging does not change
repository visibility or publish anything; keep the repository private until
another machine has been tested.

## Controls

| Input | Action |
| --- | --- |
| Mouse / arrows / A D | Move paddle |
| Right click | Capture/release mouse on Wayland |
| Space + F / fullscreen button | Toggle fullscreen |
| Space / Enter / click | Start, retry after game over, or advance after a clear |
| Space | Pause/resume |
| Escape | Exit replay; otherwise close window, release mouse and pause game |
| Page Up / Page Down | Navigate unlocked levels |
| O | Automatic paddle; marks the run assisted |
| R / S | Toggle debris / sound |
| M / Z / 2 / T | Assisted magnets / zapper / extra ball / rescue |
| P or I / F | Cycle diagnostics / engine FPS |
| Shift + F | Reset the lowest FPS measurement |
| U | Temporarily unlock all levels; does not save the unlock |
| E | Native physics/debris lab |
| J / K / L | Reverse / pause / forward replay |
| Left/right arrows / Home / End in replay | Scrub |
| Escape / Shift + K / Exit replay menu button | Return from replay to the game |
| V / B | Export SVG / SVG motion trails |
| ? / Q | Controls in menu / close plugin (Q is ignored in the physics lab) |

Replay uses the same antialiased brick meshes as live play, with a separate cache
for recorded brick visibility. Returning from replay preserves the live game and
keeps the window open, including fullscreen.

The physics lab exposes the original engine's physics, debris, and auto-paddle
sliders. Copy settings exports the original configuration text to the clipboard.

## Data

Progression, records, and debris preference are stored separately from the web game:
`~/.local/share/physical-omarchy/progress.json` for the plugin, and
`~/.local/share/physical/progress.json` for the standalone app (or under
`XDG_DATA_HOME`). `PHYSICAL_STATE_PATH` overrides either path.
SVG exports go to `~/Pictures/Physical/`. Replay stays in RAM.

## Architecture

See [TECH.md](TECH.md) for a short explanation of the stack, CPU/GPU work and
graphics compatibility.

- `Plugin.qml`: prepares the local native-library path before loading the plugin.
- `Physical.qml`: Omarchy lifecycle, floating window and live theme integration.
- `Standalone.qml` / `standalone/main.cpp`: independent Qt window and executable.
- `GameCore.qml`: shared gameplay UI, input, menus and engine communication.
- `native/engineprocess.h`: shared native process transport and clipboard helper.
- `BallWidget.qml`: Omarchy bar integration.
- `GameCanvas.qml` / `native/renderer.cpp`: a compiled Qt drawing module. It parses
  drawing packets in C++ and draws rounded bricks, debris and zapper glow
  through GPU triangle batches. The background uses an OpenGL framebuffer;
  remaining moving objects and game text are painted by QPainter into an image
  texture for the GPU to display. Brick
  contours are cached and their edge coverage follows display scale. Large
  meshes are split into triangle chunks to preserve every brick. The
  software painter remains available for headless checks. Debris keeps its
  original behind/in-front ordering.
- `native/pointerlock.cpp`: compositor pointer constraints and relative mouse input
  on the existing Qt Wayland connection, released on focus loss, resize, or menu.
- `engine/`: the existing Go physics, level parser, powerups, replay, and SVG
  exporter adapted to native execution. `internal/qtbridge` records the original
  renderer's drawing commands and supplies local persistence and level access;
  it does not implement or run a browser.
- `sounds/`: WAV conversions of original samples plus native synthesized cues.

The Qt display clock requests at most two outstanding simulation frames to avoid skipping alternate display refreshes; native
physics retains its 240 Hz fixed step. Hidden panels request no frames. The
engine runs separately so a Go panic does not take down the shell process.

Audio uses one persistent native output stream. Normal brick hits choose from
all 21 original **runtime Opus** samples, decoded to stereo PCM for Qt, without
immediate repeats. Each activated power-up uses its own runtime cue. Generic
magic hits, wall/paddle contact, life loss, completion, zapper and other effects
use generated tones with the original oscillator parameters and timing.
Random pitch/filter/gain, stereo position, the original brick burst budget,
eight active normal sample voices and four active feature cues are retained;
repeated feature cues restart with an 18 ms fade. Source/output hashes are in
`sounds/samples.json`. Audio is **not bit-identical** to the browser: the native
oscillators, resampling and output timing differ. Per-level room presets now
retain the original dry/wet mix, filters, three reflection taps and damped
feedback, with a 160 ms transition. Low/high-pass resonance follows Web Audio's
dB interpretation; filter coefficients during room transitions update every 32
samples. See the
[audio and control audit](docs/audit-2026-10-12.md).

Magnetic debris uses direct brick-grid cell lookup to reduce engine CPU work.
Regression tests compare its results with the original map lookup; the current
audit records a controlled comparison on levels 23, 26 and 27.

The build audited on 2026-10-12 completed a **15-minute** stress test on Intel HD
Graphics 520, cycling levels 23, 26 and 27 with sound and dense effects enabled.
It averaged **59.6 painted FPS**, with a lowest gameplay sample of **53.8 FPS**
and no detected crash or freeze. The direct-grid optimization reduced engine
CPU time by **30–33%** in a separate controlled benchmark. These are native
measurements; the browser version has not been benchmarked directly against this
build. The [audit](docs/audit-2026-10-12.md) includes raw logs, startup samples
and test limits.

Mobile/browser-specific
features (touch-mode selector, device tilt,
and the browser Gamepad API) are not part of this desktop port. Standard Omarchy
window controls handle moving and resizing. Fullscreen is available in the game;
right click captures the mouse through Wayland, and Escape exits replay or closes the live window and releases capture.

## License

Code is [MIT](LICENSE). Sound assets, including original recordings and generated
clips, are [CC0 1.0](sounds/LICENSE). The bundled font keeps its
[SIL Open Font License](fonts/LICENCE.md).

## Validation

```bash
./check
```

Checks include native triangulation and depth-order regressions, the original
collision/buffer regressions, 3,000 magnetic-debris grid comparisons, native
audio/room PCM checks, sample-bank hashes, all 33 levels, live
level palettes, saved volume, native physics, replay and SVG export, the native tuner,
hidden-window pause, progression, and a real-Quickshell UI smoke test plus 60 control checks through Qt input.
`python3 tests/standalone.py` checks the independent Qt host with isolated progress.
`PHYSICAL_TEST_BOOTSTRAP=1 python3 tests/qml_smoke.py` checks the plugin's native-library setup after relocation.
`python3 tests/packaging.py` checks archives, checksums, prebuilt installation,
plugin removal/reinstallation, delayed discovery and its timeout, standalone
installation and progress-preserving removal.
For a real display FPS measurement, run
`PHYSICAL_TEST_WAYLAND=1 python3 tests/qml_smoke.py`; this opens a temporary test
window for a few seconds and prints its painted frame rate.
`PHYSICAL_TEST_WAYLAND=1 ./check` also verifies all 660 brick centers in a
real GPU screenshot, covering dense fields that exceed 16-bit vertex ranges,
and tests real Wayland capture through right click, release/reacquisition,
Escape closing, menus, fullscreen and reopening.

`PHYSICAL_TEST_WAYLAND=1 python3 tests/performance.py` runs a three-minute
level 26 benchmark with additional balls, magnets, and zapper effects. It reports
painted FPS every second, engine FPS, debris counts, replay history, and the
average and minimum painted FPS. Completed runs and game overs restart the same
level, and the test uses private progress. Initial loading and restart samples
are included in the overall minimum, with transitions marked in the log.
Set `PHYSICAL_PERF_STRESS=0` for normal gameplay, `PHYSICAL_PERF_LEVEL=1` to
choose another level, or `PHYSICAL_PERF_SECONDS=300` for a five-minute run.
`PHYSICAL_PERF_RESTART_SECONDS=30` starts a fresh round every 30 seconds to
repeat dense destruction instead of lingering on the last brick.
`PHYSICAL_PERF_MUTE=1` isolates rendering and physics from audio playback. The
benchmark stops and fails on engine errors, closed windows, stalled rendering,
missing engine replies or repeated Qt socket warnings. An external Python
watchdog fails if Qt stops emitting gameplay heartbeats for eight seconds
(25 seconds during startup), and terminates the isolated test process group.
It works even when the Qt UI thread is frozen. Logs include engine reply counts.

For a longer rotating torture test with repeated dense destruction:

```sh
PHYSICAL_TEST_WAYLAND=1 PHYSICAL_PERF_LEVELS=23,26,27 PHYSICAL_PERF_SECONDS=900 PHYSICAL_PERF_RESTART_SECONDS=30 GODEBUG=randseednop=0 python3 tests/performance.py
```

To test the installed copy, add
`PHYSICAL_PERF_PROJECT="$HOME/.config/omarchy/plugins/b.physical"`; otherwise the
harness copies the current project build.

`PHYSICAL_PERF_INJECT_FREEZE=ui` or `engine` deliberately stops the isolated
test process after its third sample to verify failure detection. UI injection
uses a two-second deadline to avoid a compositor warning; any warning with the
unique test-window title is cleaned up. These negative
tests should exit with status 1. They do not stop the installed shell/game.
Set `PHYSICAL_RENDER_TIMINGS=1` to include native parsing, command compilation
and painter timings. These timings add instrumentation and should be disabled
for final FPS measurements.

The UI test runs in a temporary folder using copies of the installed Omarchy
components; it does not change the desktop theme or install the plugin.
