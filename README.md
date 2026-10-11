# Physical for Omarchy

A native Omarchy shell plugin: Go runs Physical's physics engine as a separate
native process; Qt Quick/QML draws the game and provides the controls. No browser,
webview, WebAssembly, or web server is needed at runtime.

The controls follow Omarchy's active colors, fonts, spacing, and window rounding.
The playfield, effects, replay, and SVG exports retain Physical's original
per-level palettes. The volume slider starts at 50% of the previous port's
output level and saves your preference between sessions.

## Install

Requires current Omarchy with its Quickshell plugin API and Qt 6.9+ for accelerated OpenGL drawing.
On Omarchy/Arch, `./install` offers to install any missing build and audio packages
(`gcc`, `go`, `pkgconf`, `qt6-base`, `qt6-declarative`, and `qt6-multimedia`)
before building. The build has no external Go dependencies.

```bash
cd ~/source/physical-omarchy-plugin
./install
```

This installs a copy into `~/.config/omarchy/plugins/b.physical`, rescans plugins,
and enables a solid white ball button on the left side of the Omarchy bar. Re-run `./install`
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
| Space / Enter / click | Start, retry after game over, or advance after a clear |
| Space / Escape | Pause; Escape leaves replay first |
| Page Up / Page Down | Navigate unlocked levels |
| O | Automatic paddle; marks the run assisted |
| R / S | Toggle debris / sound |
| M / Z / 2 / T | Assisted magnets / zapper / extra ball / rescue |
| P / F | Cycle diagnostics / engine FPS |
| E | Native physics/debris lab |
| J / K / L | Reverse / pause / forward replay |
| Arrows / Home / End in replay | Scrub |
| V / B | Export SVG / SVG motion trails |
| ? / Q | Help / close plugin |

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
  renderer. The software backend remains available for headless checks. Debris
  keeps its original behind/in-front ordering; the zapper halo uses layered strokes.
- `engine/`: the existing Go physics, level parser, powerups, replay, and SVG
  exporter adapted to native execution. `internal/qtbridge` records the original
  renderer's drawing commands and supplies local persistence and level access;
  it does not implement or run a browser.
- `sounds/`: WAV conversions of original samples plus native synthesized cues.

The Qt display clock requests at most two outstanding simulation frames to avoid skipping alternate display refreshes; native
physics retains its 240 Hz fixed step. Hidden panels request no frames. The
engine runs separately so a Go panic does not take down the shell process.

Audio uses a pool of Qt SoundEffects. It preserves brick/magic samples and basic
synthesized cues, but does not reproduce the web version's exact randomized
pitch/filter, stereo positioning, or room-reverb graph. Mobile/browser-specific
features (touch-mode selector, device tilt, pointer lock, browser fullscreen,
and the browser Gamepad API) are not part of this desktop port. Standard Omarchy
window controls handle moving, resizing, and fullscreen.

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

`PHYSICAL_TEST_WAYLAND=1 python3 tests/performance.py` runs a 30-second gameplay
benchmark with additional balls, magnets, and zapper effects, and reports average
and minimum painted FPS.

The UI test runs in a temporary folder using copies of the installed Omarchy
components; it does not change the desktop theme or install the plugin.
