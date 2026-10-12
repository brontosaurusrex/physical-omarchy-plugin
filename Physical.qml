import QtQuick
import Quickshell
import Quickshell.Io
import Quickshell.Hyprland
import qs.Commons
import qs.Ui
import "theme" as Theme

GameCore {
    id: root
    property var shell: null
    property var manifest: null
    windowView: hostWindow
    surface: frame
    windowSetupReady: Quickshell.env("PHYSICAL_TESTING") === "1"
    readonly property string windowPath: Qt.resolvedUrl("physical-window").toString().replace(/^file:\/\//, "")
    launchScreen: {
        var screens = Quickshell.screens;
        var monitor = Hyprland.focusedMonitor;
        for (var i = 0; i < screens.length; ++i) if (monitor && screens[i].name === monitor.name) return screens[i];
        return screens.length ? screens[0] : null;
    }
    prepareWindow: function() { if (!windowSetup.running) windowSetup.running = true; }
    onDismissed: if (shell && typeof shell.hide === "function") shell.hide("b.physical")
    onClipboardRequested: function(text) { Quickshell.clipboardText = text; }
    Component.onCompleted: { Theme.Color.source = Color; Theme.Style.source = Style; }
    Process {
        id: windowSetup
        command: [root.windowPath, String(root.defaultWindowWidth), String(root.defaultWindowHeight)]
        onExited: function(exitCode) {
            root.windowSetupReady = true;
            if (root.opened) { panel.visible = true; root.focusGame(); }
        }
    }
    Connections {
        target: Hyprland
        function onRawEvent(event) {
            if (event.name === "configreloaded") {
                root.windowSetupReady = false;
                if (root.opened && !windowSetup.running) windowSetup.running = true;
                return;
            }
            if (!root.opened) return;
            if (event.name === "openwindow") {
                var parts = event.parse(4);
                if (String(parts[2]) === "org.omarchy.screensaver") root.dismiss();
            } else if (event.name === "activewindow") {
                var active = event.parse(2);
                if (String(active[1]) !== "Physical") root.suspend();
            }
        }
    }
    FloatingWindow {
        id: hostWindow
        visible: false
        title: "Physical"
        color: root.state.palette ? root.state.palette[0] : Color.background
        implicitWidth: root.defaultWindowWidth
        implicitHeight: root.defaultWindowHeight
        minimumSize: Qt.size(Math.min(640, root.defaultWindowWidth), Math.min(360, root.defaultWindowHeight))
        onVisibleChanged: if (!visible && root.opened) root.dismiss()
        BorderSurface {
            id: frame
            anchors.fill: parent
            color: panel.color
            borderSpec: panel.fullscreen ? Border.none() : Border.surfaceSpec("menu", "border", Color.menu.border, Math.max(1, Style.normalBorderWidth))
            radius: panel.fullscreen ? 0 : Style.cornerRadius
        }
    }
}
