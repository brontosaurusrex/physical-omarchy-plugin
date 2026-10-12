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

Qt's display clock requests frames, with at most two requests outstanding.
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

Qt references: [scene graph rendering](https://doc.qt.io/qt-6/qtquick-visualcanvas-scenegraph-renderer.html)
and [graphics backends](https://doc.qt.io/qt-6/qtquick-visualcanvas-adaptations.html).

## Build and runtime

`./build` compiles the Go executable, C++ Qt module and standalone Qt executable. `./install` installs a
copy as `~/.config/omarchy/plugins/b.physical`; its installer offers missing
Arch build/runtime packages. Qt, Quickshell and the graphics/audio drivers are
needed at runtime. Go and the C++ compiler are needed to build from source,
but not to execute an already-built game. `./package` creates prebuilt bundles;
`./install --prebuilt` installs a plugin bundle without compiling. The installer
restarts the shell after copying the native module.

The plugin's `Plugin.qml` uses a short Python setup process to update its local
`native/qmldir` library path before loading `Physical.qml`. Quickshell cannot
resolve this native library through its rewritten relative QML URLs. This step
only writes the module path; it neither compiles nor downloads anything. The
standalone app uses a normal relative Qt module definition. Bundles require
compatible Qt and system libraries; see [PACKAGING.md](PACKAGING.md).

The shell controls follow Omarchy's theme. The playfield keeps the game's
original level palettes. Standalone menus use Qt's desktop palette. Both hosts
share `GameCore.qml`, render/audio modules and controls. Plugin and standalone
progress files are separate from each other and from the web version.
No browser, web server or WebAssembly runs the game.

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
`tests/pointer_lock.py` verifies Wayland capture through the QML mouse handler.
`tests/performance.py` runs real gameplay with private progress, tracks painted
frames and engine replies, and uses an external heartbeat watchdog. Deliberate
UI/engine freeze injection verifies the detector; longer runs can rotate dense
levels 23, 26 and 27. Tests cannot guarantee that every driver or future run is
crash-free. See [the current audit](docs/audit-2026-10-12.md) for measured results.

`BenchmarkNativeDenseFrames` profiles simulation, drawing-packet preparation and
JSON encoding without a window or audio device. Its timings are engine CPU
costs, not display FPS. Use `GODEBUG=randseednop=0` for reproducible rounds, and
`PHYSICAL_BENCH_HASH_GRID=1` to compare the original hash-grid lookup path.
