# Physical for Omarchy

A native Omarchy shell plugin: Go runs Physical's physics engine as a separate
native process; Qt Quick/QML draws the game and provides the controls. No browser,
webview, WebAssembly, or web server is needed at runtime.

The controls follow Omarchy's active colors, fonts, spacing, and window rounding.
The playfield, effects, replay, and SVG exports retain Physical's original
per-level palettes and game font. Pause, fullscreen, and menu buttons sit above
the playfield; settings and keyboard help live in the menu. Audio defaults to 25%
and saves your preference between sessions. A single native audio stream mixes
preloaded collision sounds, avoiding stream restarts during dense effect bursts.
The initial window is limited to 90% of the display width and height, keeps the
game aspect ratio, and opens centered horizontally directly beneath the top bar.
Qt uses the monitor scale for high-DPI rendering; the 1280-pixel window cap is in logical pixels.

## Install

Requires current Omarchy with its Quickshell plugin API and Qt 6.9+ for accelerated OpenGL drawing.
On Omarchy/Arch, `./install` offers to install any missing build and audio packages
(`gcc`, `go`, `pkgconf`, `qt6-base`, `qt6-declarative`, `qt6-multimedia`, `qt6-wayland`, `wayland`, and `wayland-protocols`)
before building. The build has no external Go dependencies.

```bash
cd ~/source/physical-omarchy-plugin
./install
```

This installs a copy into `~/.config/omarchy/plugins/b.physical`, rescans plugins,
and enables a solid white ball button in the center section of the Omarchy bar. Re-run `./install`
after editing the source. After changing the compiled renderer, also run
`omarchy restart shell` so Qt loads the new native module. Omarchy does not accept
symlinked plugin packages.

```bash
omarchy-shell shell toggle b.physical
omarchy plugin disable b.physical
```

Closing the panel retains the game and RAM replay, pauses it, and stops the frame
clock and sounds. Switching to another window pauses gameplay. Reopening requires
an explicit resume. A shell restart resets the live game and RAM replay.

## Controls

| Input | Action |
| --- | --- |
| Mouse / arrows / A D | Move paddle |
| Right click | Capture/release mouse on Wayland |
| Space + F / fullscreen button | Toggle fullscreen |
| Space / Enter / click | Start, retry after game over, or advance after a clear |
| Space / Escape | Pause; Escape first releases mouse or exits fullscreen |
| Page Up / Page Down | Navigate unlocked levels |
| O | Automatic paddle; marks the run assisted |
| R / S | Toggle debris / sound |
| M / Z / 2 / T | Assisted magnets / zapper / extra ball / rescue |
| P / F | Cycle diagnostics / engine FPS |
| E | Native physics/debris lab |
| J / K / L | Reverse / pause / forward replay |
| Arrows / Home / End in replay | Scrub |
| V / B | Export SVG / SVG motion trails |
| ? / Q | Controls in menu / close plugin |

The physics lab exposes the original engine's physics, debris, and auto-paddle
sliders. Copy settings exports the original configuration text to the clipboard.

## Data

Progression, records, and debris preference are stored separately from the web game:
`~/.local/share/physical-omarchy/progress.json` (or under `XDG_DATA_HOME`).
SVG exports go to `~/Pictures/Physical/`. Replay stays in RAM.

## Architecture

- `Physical.qml`: plugin lifecycle, floating window, input, theme, native process.
- `BallWidget.qml`: Omarchy bar integration.
- `GameCanvas.qml` / `native/renderer.cpp`: a compiled Qt drawing module. It parses
  drawing packets in C++, batches debris into scene graph triangles, records
  other native QPainter commands, caches the static
  brick layer, and paints through an OpenGL framebuffer on the normal Omarchy
  renderer. Rounded bricks, debris and zapper glow use GPU triangles; brick
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

Audio uses one native output stream. Normal brick hits choose from all 21
original samples randomly without immediately repeating a sample. Each activated
power-up uses its own original cue; generic magic-brick hits keep a separate cue
bank. Power-up cues take priority during dense collision bursts. Playback keeps
the engine's pitch scale, including zapper hits. The web version's randomized
pitch/filter, stereo positioning, and room-reverb graph are not reproduced. Mobile/browser-specific
features (touch-mode selector, device tilt,
and the browser Gamepad API) are not part of this desktop port. Standard Omarchy
window controls handle moving and resizing. Fullscreen is available in the game;
right click captures the mouse through Wayland, and Escape releases it.

## Validation

```bash
./check
```

Checks include native triangulation and depth-order regressions, the original
collision/buffer regressions, all 33 levels, live
level palettes, saved volume, native physics, replay and SVG export, the native tuner,
hidden-window pause, progression, and a headless real-Quickshell UI smoke test.
For a real display FPS measurement, run
`PHYSICAL_TEST_WAYLAND=1 python3 tests/qml_smoke.py`; this opens a temporary test
window for a few seconds and prints its painted frame rate.
`PHYSICAL_TEST_WAYLAND=1 ./check` also verifies all 660 brick centers in a
real GPU screenshot, covering dense fields that exceed 16-bit vertex ranges.

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
or repeated Qt socket warnings.
Set `PHYSICAL_RENDER_TIMINGS=1` to include native parsing, command compilation
and painter timings. These timings add instrumentation and should be disabled
for final FPS measurements.

The UI test runs in a temporary folder using copies of the installed Omarchy
components; it does not change the desktop theme or install the plugin.
