# Port source

Imported from `/home/b/source/physical/dev` on 2026-10-11, including its current
uncommitted physics-buffer and canvas batching work. Repository HEAD at import:
`f0ec34186ea28bbff636f878995669696a55fbf4`.

The original project remains unchanged. The copied Go source keeps its physics,
collision ordering, fixed-step scheduler, level parser, powerups, scoring,
progression, debris, replay, and vector export. Browser-specific functions remain
in the source to minimize changes to the engine, but are not used by the QML
front end and are removed from the native binary by Go's linker when unreachable.

Native adaptations replace the `syscall/js` import with a local drawing/storage
adapter, change startup to a line-based native protocol, retain level colors, route sound cues to Qt, and provide clipboard/SVG file
exports. The browser physics editor is replaced with native QML controls backed
by the same slider definitions and setter functions. The accelerated renderer
uses a local C++ Qt module; fragment instances are compact native commands and
their cached outlines are triangulated for batched scene graph drawing.

No license was declared in the original source checkout. This port does not
invent a new license for the game or its assets.
