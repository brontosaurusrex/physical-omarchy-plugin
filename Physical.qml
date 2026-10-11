import QtQuick
import QtQuick.Controls as QQC
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import Quickshell.Hyprland
import qs.Commons
import qs.Ui
import "native" as Native

Item {
    id: root
    property var shell: null
    property var manifest: null
    property bool opened: false
    property bool engineReady: false
    // Replies can arrive after the next display callback. A two-frame bound
    // avoids halving the display rate while keeping the request queue short.
    property int ticksPending: 0
    property bool helpVisible: false
    property bool menuVisible: false
    property bool spaceHeld: false
    property bool fullscreenChord: false
    readonly property bool fullscreen: panel.fullscreen
    readonly property bool mouseLocked: pointerLock.locked
    readonly property var launchScreen: {
        var screens = Quickshell.screens;
        var monitor = Hyprland.focusedMonitor;
        for (var i = 0; i < screens.length; ++i) if (monitor && screens[i].name === monitor.name) return screens[i];
        return screens.length ? screens[0] : null;
    }
    readonly property int defaultWindowWidth: Math.min(1280, Math.floor((panel.screen ? panel.screen.width : 1280) * 0.9))
    readonly property int defaultWindowHeight: Math.min(760, Math.floor((panel.screen ? panel.screen.height : 760) * 0.9))
    readonly property color hudColor: state.palette && state.palette.length > 4 ? state.palette[4] : Color.foreground
    function focusGame() { Qt.callLater(function() { gameInput.forceActiveFocus(); }); }
    function toggleMenu() {
        menuVisible = !menuVisible;
        if (menuVisible) { pointerLock.release(); send({action: "pause"}); }
        else helpVisible = false;
        focusGame();
    }
    function closeMenu() { menuVisible = false; helpVisible = false; focusGame(); }
    function toggleFullscreen() { pointerLock.release(); panel.fullscreen = !panel.fullscreen; focusGame(); }
    function pressKey(event) {
        if (event.key === Qt.Key_Space) { if (!event.isAutoRepeat) { spaceHeld = true; fullscreenChord = false; } return; }
        if (event.key === Qt.Key_F && spaceHeld) { if (!event.isAutoRepeat) { toggleFullscreen(); fullscreenChord = true; } return; }
        if (event.key === Qt.Key_Question) { if (!menuVisible) toggleMenu(); helpVisible = !helpVisible; return; }
        if (event.key === Qt.Key_Escape) {
            if (pointerLock.requested) { pointerLock.release(); return; }
            if (menuVisible) { closeMenu(); return; }
            if (panel.fullscreen) { toggleFullscreen(); return; }
        }
        if (event.key === Qt.Key_Q && !state.editor) { dismiss(); return; }
        if (menuVisible) return;
        send({ action: "key", key: keyName(event), code: keyName(event), down: true, repeat: event.isAutoRepeat, shift: (event.modifiers & Qt.ShiftModifier) !== 0 });
    }
    function releaseKey(event) {
        if (event.isAutoRepeat) return;
        if (event.key === Qt.Key_Space) {
            if (spaceHeld && !fullscreenChord && !menuVisible) shortcut(" ", "Space");
            spaceHeld = false; fullscreenChord = false; return;
        }
        send({ action: "key", key: keyName(event), code: keyName(event), down: false });
    }
    property var state: ({})
    property var sliders: []
    property string errorText: ""
    property string notice: ""
    property real pointerX: 0
    property real pointerY: 0
    property bool pointerDirty: false
    readonly property int paintedFrames: canvas.paintedFrames
    readonly property string renderError: canvas.renderError
    readonly property var mouseCapture: pointerLock
    readonly property var windowView: panel
    readonly property var gameView: canvas
    readonly property var contentView: card
    readonly property string windowPath: Qt.resolvedUrl("physical-window").toString().replace(/^file:\/\//, "")
    property bool windowSetupReady: false
    readonly property string enginePath: Qt.resolvedUrl("bin/physical-engine").toString().replace(/^file:\/\//, "")
    readonly property var theme: ({ background: String(Color.background), foreground: String(Color.foreground), accent: String(Color.accent), urgent: String(Color.urgent), muted: String(Color.muted) })
    readonly property bool animating: opened && engineReady && !menuVisible && ((!state.paused && !state.ready && !state.gameOver && !state.complete && (!state.editor || state.live)) || (state.replay && state.replayDirection !== 0))

    function send(data) {
        if (!engineReady || !engine.running) return;
        engine.write(JSON.stringify(data) + "\n");
    }
    function open(payloadJson) {
        panel.screen = launchScreen;
        opened = true;
        if (windowSetupReady || Quickshell.env("PHYSICAL_TESTING") === "1") panel.visible = true;
        else if (!windowSetup.running) windowSetup.running = true;
        if (!engine.running) { errorText = ""; engine.running = true; }
        else if (!engineReady) engine.write('{"action":"hello"}\n');
        else { send({ action: "visibility", down: true }); }
        Qt.callLater(function() { gameInput.forceActiveFocus(); });
    }
    function close() {
        pointerLock.release();
        spaceHeld = false; fullscreenChord = false; menuVisible = false; panel.fullscreen = false;
        send({ action: "visibility", down: false });
        audio.stop();
        opened = false;
        panel.visible = false;
        helpVisible = false;
        pointerDirty = false;
    }
    function dismiss() {
        close();
        if (shell && typeof shell.hide === "function") shell.hide("b.physical");
    }
    function shortcut(key, code) {
        send({ action: "key", key: key, code: code || key, down: true });
        send({ action: "key", key: key, code: code || key, down: false });
        gameInput.forceActiveFocus();
    }
    function keyName(event) {
        var special = ({});
        special[Qt.Key_Left] = "ArrowLeft"; special[Qt.Key_Right] = "ArrowRight";
        special[Qt.Key_Up] = "ArrowUp"; special[Qt.Key_Down] = "ArrowDown";
        special[Qt.Key_Space] = " "; special[Qt.Key_Return] = "Enter"; special[Qt.Key_Enter] = "Enter";
        special[Qt.Key_Escape] = "Escape"; special[Qt.Key_PageUp] = "PageUp"; special[Qt.Key_PageDown] = "PageDown";
        special[Qt.Key_Home] = "Home"; special[Qt.Key_End] = "End";
        return special[event.key] || event.text || "";
    }
    function receive(line) {
        try {
            var packet = canvas.acceptLine(line);
            if (packet.type !== "frame") return;
            var first = !engineReady;
            engineReady = true;
            if (packet.action === "tick") ticksPending = Math.max(0, ticksPending - 1);
            state = packet.state;
            if (packet.sliders) sliders = packet.sliders;
            if (packet.sounds && opened) audio.play(packet.sounds);
            if (packet.clipboard) { Quickshell.clipboardText = packet.clipboard; notice = "Settings copied"; noticeTimer.restart(); }
            if (packet.export) { notice = "Saved " + packet.export; noticeTimer.restart(); }
            if (packet.error) errorText = packet.error;
            if (first) { send({ action: "visibility", down: opened }); }
        } catch (error) { errorText = "Native engine: " + String(error); }
    }
    function tick() {
        if (ticksPending >= 2 || !engineReady || !opened) return;
        if (pointerDirty) { pointerDirty = false; send({ action: "pointer", x: pointerX, y: pointerY }); }
        ticksPending++;
        send({ action: "tick" });
    }

    FontLoader { id: gameFont; source: "fonts/EnvyCodeRNerdFontMono-Bold.ttf"; }
    Process {
        id: engine
        command: [root.enginePath]
        stdinEnabled: true
        stdout: SplitParser { onRead: function(line) { root.receive(line); } }
        stderr: SplitParser { onRead: function(line) { if (/panic|fatal|Invalid input/.test(line)) root.errorText = line; } }
        onExited: function(exitCode) {
            root.engineReady = false;
            root.ticksPending = 0;
            if (root.opened) root.errorText = "Physical engine stopped (" + exitCode + "). Reopen Physical to retry.";
        }
    }
    Process {
        id: windowSetup
        command: [root.windowPath, String(root.defaultWindowWidth), String(root.defaultWindowHeight)]
        onExited: function(exitCode) {
            root.windowSetupReady = true;
            if (root.opened) { panel.visible = true; Qt.callLater(function() { gameInput.forceActiveFocus(); }); }
        }
    }
    Timer { interval: 100; repeat: true; running: root.opened && root.engineReady && !root.animating; onTriggered: root.tick(); }
    Timer { id: noticeTimer; interval: 6000; onTriggered: root.notice = ""; }
    AudioPool { id: audio; enabled: root.state.sound === true; volume: root.state.volume === undefined ? 0.25 : root.state.volume; }
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
                if (String(active[1]) !== "Physical") { pointerLock.release(); root.spaceHeld = false; root.send({ action: "pause" }); audio.stop(); }
            }
        }
    }

    FloatingWindow {
        id: panel
        visible: false
        title: "Physical"
        color: root.state.palette ? root.state.palette[0] : Color.background
        implicitWidth: root.defaultWindowWidth
        implicitHeight: root.defaultWindowHeight
        minimumSize: Qt.size(Math.min(640, root.defaultWindowWidth), Math.min(360, root.defaultWindowHeight))
        onVisibleChanged: if (!visible && root.opened) root.dismiss()

        BorderSurface {
            id: card
            anchors.fill: parent
            color: panel.color
            borderSpec: panel.fullscreen ? Border.none() : Border.surfaceSpec("menu", "border", Color.menu.border, Math.max(1, Style.normalBorderWidth))
            radius: panel.fullscreen ? 0 : Style.cornerRadius
            FrameAnimation { running: root.animating; onTriggered: root.tick(); }
            Item {
                id: playArea
                anchors.fill: parent
                anchors.margins: panel.fullscreen ? 0 : 2
                Item {
                    id: gameInput
                    anchors.centerIn: parent
                    width: Math.min(parent.width, parent.height * canvas.gameWidth / canvas.gameHeight)
                    height: width * canvas.gameHeight / canvas.gameWidth
                    focus: true
                    onWidthChanged: pointerLock.release()
                    onHeightChanged: pointerLock.release()
                    GameCanvas { id: canvas; objectName: "physicalCanvas"; anchors.fill: parent; theme: root.theme; fontFamily: gameFont.status === FontLoader.Ready ? gameFont.name : Style.resolvedFontFamily; }
                    Native.PointerLock {
                        id: pointerLock
                        anchors.fill: parent
                        onLockedChanged: { root.pointerDirty = false; root.send({action: "pointer-lock", down: locked}); }
                        onRelativeMotion: function(dx, dy) { root.send({action: "relative-pointer", x: dx * canvas.gameWidth / gameInput.width}); }
                    }
                    MouseArea {
                        anchors.fill: parent
                        hoverEnabled: true
                        acceptedButtons: Qt.LeftButton | Qt.RightButton
                        cursorShape: pointerLock.locked ? Qt.BlankCursor : Qt.ArrowCursor
                        onPositionChanged: function(mouse) {
                            if (pointerLock.requested) return;
                            root.pointerX = mouse.x * canvas.gameWidth / width;
                            root.pointerY = mouse.y * canvas.gameHeight / height;
                            root.pointerDirty = true;
                        }
                        onPressed: function(mouse) {
                            gameInput.forceActiveFocus();
                            if (mouse.button === Qt.RightButton) {
                                if (pointerLock.requested) pointerLock.release();
                                else if (!pointerLock.lock()) { root.notice = "Mouse capture unavailable"; noticeTimer.restart(); }
                            } else root.send({ action: "pointer", down: true, x: mouse.x * canvas.gameWidth / width, y: mouse.y * canvas.gameHeight / height });
                        }
                    }
                    Keys.onPressed: function(event) { root.pressKey(event); event.accepted = true; }
                    Keys.onReleased: function(event) { root.releaseKey(event); event.accepted = true; }
                    Row {
                        anchors.top: parent.top
                        anchors.right: parent.right
                        anchors.margins: 14
                        spacing: 8
                        HudButton { icon: root.state.paused || root.state.ready ? "play" : "pause"; foreground: root.hudColor; onClicked: { pointerLock.release(); root.shortcut(" ", "Space"); } }
                        HudButton { icon: "fullscreen"; foreground: root.hudColor; onClicked: root.toggleFullscreen(); }
                        HudButton { icon: "menu"; foreground: root.hudColor; onClicked: root.toggleMenu(); }
                    }
                }
                Loader {
                    anchors.top: parent.top
                    anchors.bottom: parent.bottom
                    anchors.right: parent.right
                    width: Math.min(Style.space(420), parent.width * 0.46)
                    active: root.state.editor === true && !root.menuVisible
                    sourceComponent: Tuner {
                        state: root.state
                        sliders: root.sliders
                        onCommand: function(data) { root.send(data); }
                        onDismissed: root.shortcut("e", "KeyE")
                    }
                }
                MouseArea { anchors.fill: parent; visible: root.menuVisible; acceptedButtons: Qt.AllButtons; onClicked: root.closeMenu(); }
                GameMenu {
                    anchors.top: parent.top
                    anchors.right: parent.right
                    anchors.margins: 12
                    width: Math.min(Style.space(360), parent.width - 24)
                    height: Math.min(implicitHeight, parent.height - 24)
                    visible: root.menuVisible
                    controller: root
                }
                Rectangle {
                    anchors.horizontalCenter: parent.horizontalCenter
                    anchors.bottom: parent.bottom
                    anchors.bottomMargin: 16
                    width: Math.min(parent.width - 32, toast.implicitWidth + 24)
                    height: toast.implicitHeight + 16
                    visible: !!toast.text
                    color: Color.menu.background
                    radius: Style.cornerRadius
                    Text {
                        id: toast
                        anchors.centerIn: parent
                        width: Math.min(implicitWidth, parent.width - 24)
                        text: root.errorText || canvas.renderError || root.notice
                        color: root.errorText || canvas.renderError ? Color.urgent : Color.foreground
                        font.family: Style.fontFamily
                        font.pixelSize: Style.font.bodySmall
                        wrapMode: Text.Wrap
                    }
                }
            }
        }
    }
}
