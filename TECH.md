# How Physical for Omarchy works

Physical is a native Linux game packaged as an Omarchy shell plugin and a standalone Qt app.

| Technology | What it does |
| --- | --- |
| **Go** (`engine/`) | Runs physics, collisions, levels, power-ups, replay, progress and SVG export in a separate process. Physics uses a fixed 240 Hz step; that is separate from display FPS. |
| **Qt Quick / QML** (`GameCore.qml`, windows, menus and widgets) | Defines the window, controls, input handlers and bar button. QML describes the interface; small JavaScript functions connect its actions. |
| **C++ / Qt** (`native/renderer.cpp`) | Reads the engine's drawing packets, prepares geometry, caches brick shapes and draws the game through Qt's scene graph and QPainter. |
| **GPU / OpenGL** | Draws batches of triangles for bricks, debris and zapper glow, blends their edges, and displays textures and the interface. |
| **Qt Multimedia** (`native/audiomixer.h`, `native/audioroom.h`, `AudioPool.qml`) | Mixes stereo PCM samples and generated oscillator tones into one persistent output stream. Ordinary bricks pick random samples; power-ups have individual cues. Pitch, filters, gain, stereo pan and per-level room echoes run on the CPU. |
| **Wayland** (`native/pointerlock.cpp`) | Talks to the compositor to capture the mouse and receive relative movement. Right click toggles capture; Esc exits replay, or closes the window and releases capture during live play. |
| **Omarchy / Quickshell** | Loads the plugin, supplies the bar and themed controls, and hosts its Qt window. The standalone app uses a regular Qt window and needs no Quickshell. |
| **C++ / QProcess** (`native/engineprocess.h`) | Starts the Go engine and moves drawing/state packets over process pipes. Both hosts share this transport. |

The bar ball in `BallWidget.qml` uses a circular path with Qt Quick Shapes'
curve renderer for smooth edges at its small icon size. Its color follows the
bar foreground when closed and the bar's active color while the game window is
open. The scoped shell's `isPluginOpen` API supplies that state; a paused engine
retained after closing does not keep the icon highlighted.

## CPU versus GPU

The **CPU** computes ball movement, collisions, effects and game state. It also
mixes audio, parses drawing packets, builds triangle geometry and paints some
remaining content, including game text, into textures.

The **GPU** turns geometry and textures into screen pixels. It does not run the
game's physics. Cached brick meshes and batched effects reduce repeated painting
and draw overhead. Large triangle meshes are split into smaller chunks so dense
levels do not lose bricks at an index limit. Replay uses this same antialiased
brick-mesh path with its own cache, rebuilt only when recorded brick visibility,
tilt or styling changes. Replay transport commands draw the selected recorded
frame even while playback is paused; exiting restores the live cache.

Each frame follows this path:

```text
Input → Go simulation → JSON drawing packets over process pipes
      → C++ renderer → Qt scene graph / OpenGL → display
```

Magnetic debris queries a direct array of brick-grid cells instead of hashing
each cell repeatedly. The cell contents and traversal order match the original
grid, so this reduces CPU lookup work without changing which brick attracts a
fragment. Regression checks compare 3,000 queries, including destroyed bricks.

The renderer retains a painted foreground texture when its recorded picture is
byte-for-byte unchanged. Static completion text can stay cached while separate
GPU layers animate debris; changes to text, colors or other foreground content
invalidate it, and resizing still repaints at the new display size.

Particle contours travel to Qt only when first used or changed. Each depth layer
then sends a compact batch of full-precision positions, rotations and colors.
The renderer reuses the same triangulated outlines and caches resolved colors;
unused contours are discarded on both sides. Returning particles resend their
definitions, including after replay or adaptive rendering. This preserves the
particles, drawing order and physics while reducing packet and allocation work.

The game window's `QQuickWindow::afterAnimating` preparation signal requests
frames, with at most two requests outstanding. This gives the separate engine
time to prepare the next scene and avoids depending on Quickshell's shared
`FrameAnimation` clock, which Qt can drive with timers in multi-window setups.
Each animated request carries the measured window-frame duration; elapsed time from
skipped requests accumulates into the next one. Physics and render interpolation
use that duration rather than uneven pipe-arrival times in the Go process.
Pause, menu and hidden transitions discard unsent animation time.
Window changes and resumes reset the clock to the current screen's refresh
period, excluding time spent idle. It does not change the 240 Hz physics steps.
Hidden windows stop requesting frames. A faster GPU can improve drawing, but
CPU work, packet processing, driver overhead and display refresh also limit FPS.

## GPU compatibility

There is no Intel-specific code: Intel, AMD and NVIDIA GPUs should work when
their drivers support the OpenGL backend used by Qt/Quickshell. This build has
been tested on Intel HD Graphics 520; other GPUs are expected to work but have
not been verified here. “Any GPU” is not a guarantee for unsupported hardware
or broken drivers.

The game's accelerated path currently selects **OpenGL**. Qt itself supports
other graphics backends, but this game's custom fast path has not been enabled
or validated for them. A QPainter fallback also exists and is tested with Qt's
software backend; it can be substantially slower during dense effects. It is
not an automatic recovery mechanism for a failing GPU driver.

Qt applies monitor scaling for high-DPI displays. Larger resolutions require
more pixels and larger textures, so 4K/8K layout support does not guarantee the
same FPS as 1080p.

Qt references: [scene graph rendering](https://doc.qt.io/qt-6/qtquick-visualcanvas-scenegraph-renderer.html),
[graphics backends](https://doc.qt.io/qt-6/qtquick-visualcanvas-adaptations.html),
[animation clocks](https://doc.qt.io/qt-6/qtquick-visualcanvas-scenegraph.html)
and [window frame preparation](https://doc.qt.io/qt-6/qquickwindow.html#afterAnimating).

## Build and runtime

`./build` compiles the Go executable, C++ Qt module and standalone Qt executable. `./install` installs a
copy as `~/.config/omarchy/plugins/b.physical`; its installer offers missing
Arch build/runtime packages. Qt, Quickshell and the graphics/audio drivers are
needed at runtime. Go and the C++ compiler are needed to build from source,
but not to execute an already-built game. `./package` creates prebuilt bundles;
`./install --prebuilt` installs a plugin bundle without compiling. The installer
restarts the shell after copying the native module, then waits up to 15 seconds
for asynchronous plugin discovery before enabling Physical.

The plugin's `Plugin.qml` uses a short Python setup process to update its local
`native/qmldir` library path before loading `Physical.qml`. Quickshell cannot
resolve this native library through its rewritten relative QML URLs. This step
only writes the module path; it neither compiles nor downloads anything. The
standalone app uses a normal relative Qt module definition. Bundles require
compatible Qt and system libraries; see [PACKAGING.md](PACKAGING.md).

The shell controls follow Omarchy's theme. The top-right game controls use the
bundled Envy Code R Nerd Font Mono Bold for pause/play and fullscreen, with the
web version's exact pause/fullscreen glyphs and Unicode vertical ellipsis
for the menu (with Qt font fallback where needed). Their tint follows the level
palette. The playfield keeps the game's original level palettes, with the web
version's dark blue surround (`#1a1a2e`)
and 8-pixel rounded corners in windowed and fullscreen modes. Four tiny cached
corner images use supersampled circular paths for smooth edges and repaint only
when their size or color changes. The playfield retains its 2:1 ratio and leaves
a margin around it. The display clock continues on the level-complete screen
while debris is present, then returns to idle polling when the screen is still.
Standalone menus use Qt's desktop palette. Both hosts
share `GameCore.qml`, render/audio modules and controls. Plugin and standalone
progress files are separate from each other and from the web version.
No browser, web server or WebAssembly runs the game.

Mouse capture tracks the playfield's current surface region after layout,
including letterboxing during window resizing. The Wayland constraint stays
persistent across temporary compositor unlocks. After an 80 ms pause in geometry
changes or a compositor unlock, an inactive constraint is re-requested while
the game still has focus. A committed Wayland cursor-position hint requests
the playfield center as the release position, clear of notification cards that
can cover the original capture position. This lets capture recover from passive
notifications without pausing the game.
This preserves capture through Hyprland's Super+F and Super+Alt+F transitions,
Space+F and the fullscreen button. Right click, menu, Escape, hiding and focus
loss cancel the request and its pending recovery timer; returning focus alone
does not capture the mouse again.
Uncaptured pointer movement is tracked across the whole game window, including
the blue surround and letterboxing, with paddle targets clamped by the engine.
The handler is passive; leaving the window leaves the desktop pointer free.

## Audio and test limits

The sample bank is decoded from the original game's runtime Opus files, with
stereo channels preserved. The Go engine emits the original tone waveform,
frequency sweep, duration, gain, delay and position. C++ generates and mixes
these tones with filtered sample voices. Repeated feature cues fade and restart.
The original room presets feed a dry branch and a wet branch with input/output
filters, three reflection delays and a filtered feedback loop on the third tap.
Room parameters come from the Go level config; QML forwards changes to the mixer.
Gains and delay times ramp over 160 ms; filter coefficients update every 32
samples during that ramp and remain cached afterward. Low/high-pass Q uses the
Web Audio dB convention, while band-pass Q is linear. Silent tails are cleared
after 12 seconds, and closing/muting clears them immediately. Native oscillators,
resampling and scheduling still differ from browser DSP, so this is not
bit-identical output.

`tests/keyboard.py` sends Qt keyboard events through the actual QML window.
`tests/pointer_lock.py` verifies Wayland capture through the QML mouse handler,
compositor fullscreen/maximize and native fullscreen transitions, focus loss,
passive notifications, manual reacquisition, menu, Escape and reopening.
`PHYSICAL_MOUSE_SECONDS=300 python3 tests/mouse_torture.py` repeats these paths
on the live Wayland desktop, also checking uncaptured margin movement through
to the engine and that the desktop cursor can leave the window. It uses a
temporary collision-free level and private progress, then restores the cursor
and previously focused window. Same-process notification overlays reproduce
the shell's popup behavior; periodic real desktop notifications are included.
A temporary Wayland virtual mouse also checks actual relative-motion delivery
in both directions, after notification recovery, and outside the game window.
Its [wlroots protocol](https://github.com/swaywm/wlr-protocols/blob/master/unstable/wlr-virtual-pointer-unstable-v1.xml)
is vendored with its MIT notice in `tests/protocols/`; the helper is compiled
only for this test. An external watchdog checks UI heartbeats, engine replies
and rendered frames. Set `PHYSICAL_MOUSE_INJECT_FREEZE=ui` or `engine` to
verify detection with a deliberately stopped test process. Qt input and the
virtual mouse exercise software paths; they cannot reproduce a USB device or
cable failure. Physical mouse hardware still needs manual playtesting.
The five-minute test runs separately from `./check`.
`tests/level_complete.py` embeds a tiny level in a temporary engine build and
checks actual level completion, smooth debris animation, cached text and corners
and the return to idle when debris disappears. It leaves game levels untouched.
`tests/frame_pacing.py` checks constant-speed ball rendering in a temporary level
while deliberately delaying the test engine, including skipped requests and
pause/resume. Neither delays nor test levels are installed with the game.
`tests/performance.py` runs real gameplay with private progress, tracks painted
frames and engine replies, and uses an external heartbeat watchdog. Deliberate
UI/engine freeze injection verifies the detector; longer runs can rotate dense
levels 23, 26 and 27. Tests cannot guarantee that every driver or future run is
crash-free. See the [gameplay audit](docs/audit-2026-10-12.md) and
[frame timing and particle measurements](docs/audit-2026-10-13.md).

`BenchmarkNativeDenseFrames` profiles simulation, drawing-packet preparation and
JSON encoding without a window or audio device. Its timings are engine CPU
costs, not display FPS. Use `GODEBUG=randseednop=0` for reproducible rounds, and
`PHYSICAL_BENCH_HASH_GRID=1` to compare the original hash-grid lookup path.
