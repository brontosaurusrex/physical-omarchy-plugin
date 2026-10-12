# How Physical for Omarchy works

Physical is a native Linux game packaged as an Omarchy shell plugin.

| Technology | What it does |
| --- | --- |
| **Go** (`engine/`) | Runs physics, collisions, levels, power-ups, replay, progress and SVG export in a separate process. Physics uses a fixed 240 Hz step; that is separate from display FPS. |
| **Qt Quick / QML** (`Physical.qml`, menus and widgets) | Defines the window, controls, input handlers and bar button. QML describes the interface; small JavaScript functions connect its actions. |
| **C++ / Qt** (`native/renderer.cpp`) | Reads the engine's drawing packets, prepares geometry, caches brick shapes and draws the game through Qt's scene graph and QPainter. |
| **GPU / OpenGL** | Draws batches of triangles for bricks, debris and zapper glow, blends their edges, and displays textures and the interface. |
| **Qt Multimedia** (`native/audiomixer.h`, `AudioPool.qml`) | Mixes preloaded WAV samples into one audio output stream. Ordinary bricks pick random samples; power-ups have individual cues. |
| **Wayland** (`native/pointerlock.cpp`) | Talks to the compositor to capture the mouse and receive relative movement. Right click toggles capture; Esc closes the window and releases it. |
| **Omarchy / Quickshell** | Loads the plugin, supplies the bar and themed controls, and hosts the Qt interface. |

## CPU versus GPU

The **CPU** computes ball movement, collisions, effects and game state. It also
mixes audio, parses drawing packets, builds triangle geometry and paints some
remaining content, including game text, into textures.

The **GPU** turns geometry and textures into screen pixels. It does not run the
game's physics. Cached brick meshes and batched effects reduce repeated painting
and draw overhead. Large triangle meshes are split into smaller chunks so dense
levels do not lose bricks at an index limit.

Each frame follows this path:

```text
Input → Go simulation → JSON drawing packets over process pipes
      → C++ renderer → Qt scene graph / OpenGL → display
```

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

`./build` compiles the Go executable and C++ Qt module. `./install` installs a
copy as `~/.config/omarchy/plugins/b.physical`; its installer offers missing
Arch build/runtime packages. Qt, Quickshell and the graphics/audio drivers are
needed at runtime. Go and the C++ compiler are needed to build from source,
but not to execute an already-built game.

The shell controls follow Omarchy's theme. The playfield keeps the game's
original level palettes. Progress is stored separately from the web version.
No browser, web server or WebAssembly runs the game.
