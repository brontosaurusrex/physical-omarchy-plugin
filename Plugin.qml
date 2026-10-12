import QtQuick
import Quickshell.Io

// Quickshell rewrites local QML URLs, so native libraries need a filesystem
// path. Prepare it before loading Physical.qml; no compiler/install hook runs.
Item {
    id: root
    property var shell: null
    property var manifest: null
    property bool pendingOpen: false
    property var pendingPayload: "{}"
    readonly property var game: gameLoader.item
    readonly property bool opened: game ? game.opened : pendingOpen
    readonly property string moduleDirectory: Qt.resolvedUrl("native/").toString().replace(/^file:\/\//, "").replace(/\/$/, "")
    function open(payload) {
        pendingPayload = payload || "{}";
        pendingOpen = true;
        if (game) { pendingOpen = false; game.open(pendingPayload); }
    }
    function close() { pendingOpen = false; if (game) game.close(); }
    function toggle(payload) { if (opened) close(); else open(payload); }
    onShellChanged: if (game) game.shell = shell
    onManifestChanged: if (game) game.manifest = manifest
    Process {
        id: moduleSetup
        command: ["python3", root.moduleDirectory + "/setup-module.py", root.moduleDirectory]
        onExited: function(exitCode) {
            if (exitCode === 0) gameLoader.active = true;
            else console.error("Physical: cannot prepare native module path", exitCode);
        }
    }
    Component.onCompleted: moduleSetup.running = true
    Loader {
        id: gameLoader
        active: false
        source: Qt.resolvedUrl("Physical.qml")
        onLoaded: {
            item.shell = root.shell;
            item.manifest = root.manifest;
            if (root.pendingOpen) { root.pendingOpen = false; item.open(root.pendingPayload); }
        }
    }
}
