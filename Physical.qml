import QtQuick
import QtQuick.Controls as QQC
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import Quickshell.Hyprland
import qs.Commons
import qs.Ui

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
    property var state: ({})
    property var sliders: []
    property string errorText: ""
    property string notice: ""
    property real pointerX: 0
    property real pointerY: 0
    property bool pointerDirty: false
    readonly property int paintedFrames: canvas.paintedFrames
    readonly property string renderError: canvas.renderError
    readonly property var gameView: canvas
    readonly property var contentView: card
    readonly property string windowPath: Qt.resolvedUrl("physical-window").toString().replace(/^file:\/\//, "")
    property bool windowSetupReady: false
    readonly property string enginePath: Qt.resolvedUrl("bin/physical-engine").toString().replace(/^file:\/\//, "")
    readonly property var theme: ({ background: String(Color.background), foreground: String(Color.foreground), accent: String(Color.accent), urgent: String(Color.urgent), muted: String(Color.muted) })
    readonly property bool animating: opened && engineReady && !helpVisible && ((!state.paused && !state.ready && !state.gameOver && !state.complete && (!state.editor || state.live)) || (state.replay && state.replayDirection !== 0))

    function send(data) {
        if (!engineReady || !engine.running) return;
        engine.write(JSON.stringify(data) + "\n");
    }
    function open(payloadJson) {
        opened = true;
        if (windowSetupReady || Quickshell.env("PHYSICAL_TESTING") === "1") panel.visible = true;
        else if (!windowSetup.running) windowSetup.running = true;
        if (!engine.running) { errorText = ""; engine.running = true; }
        else if (!engineReady) engine.write('{"action":"hello"}\n');
        else { send({ action: "visibility", down: true }); }
        Qt.callLater(function() { gameInput.forceActiveFocus(); });
    }
    function close() {
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
        command: [root.windowPath, String(Style.space(1280)), String(Style.space(760))]
        onExited: function(exitCode) {
            root.windowSetupReady = true;
            if (root.opened) { panel.visible = true; Qt.callLater(function() { gameInput.forceActiveFocus(); }); }
        }
    }
    Timer { interval: 100; repeat: true; running: root.opened && root.engineReady && !root.animating; onTriggered: root.tick(); }
    Timer { id: noticeTimer; interval: 6000; onTriggered: root.notice = ""; }
    AudioPool { id: audio; enabled: root.state.sound === true; volume: root.state.volume === undefined ? 0.5 : root.state.volume; }
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
                if (String(active[1]) !== "Physical") { root.send({ action: "pause" }); audio.stop(); }
            }
        }
    }

    FloatingWindow {
        id: panel
        visible: false
        title: "Physical"
        color: Color.background
        implicitWidth: Style.space(1280)
        implicitHeight: Style.space(760)
        minimumSize: Qt.size(Style.space(760), Style.space(460))
        onVisibleChanged: if (!visible && root.opened) root.dismiss()

        BorderSurface {
            id: card
            anchors.fill: parent
            color: Color.background
            borderSpec: Border.surfaceSpec("menu", "border", Color.menu.border, Math.max(1, Style.normalBorderWidth))
            radius: Style.cornerRadius
            FrameAnimation { running: root.animating; onTriggered: root.tick(); }

            ColumnLayout {
                anchors.fill: parent
                anchors.margins: Style.space(14)
                spacing: Style.space(10)
                RowLayout {
                    Layout.fillWidth: true
                    spacing: Style.space(8)
                    Text { text: "PHYSICAL"; color: Color.accent; font.family: Style.fontFamily; font.pixelSize: Style.font.heading; }
                    Text { text: root.engineReady ? "Level " + root.state.level + " / " + root.state.levels : "Starting…"; color: Color.muted; font.family: Style.fontFamily; font.pixelSize: Style.font.body; Layout.fillWidth: true; }
                    GameButton { text: root.state.ready ? "Start" : root.state.paused ? "Resume" : "Pause"; enabled: root.engineReady && !root.state.editor; onClicked: root.shortcut(" ", "Space"); }
                    GameButton { text: "Retry"; enabled: root.engineReady; onClicked: { root.send({ action: "restart" }); gameInput.forceActiveFocus(); } }
                    GameButton { text: "Lab"; selected: root.state.editor === true; enabled: root.engineReady; onClicked: root.shortcut("e", "KeyE"); }
                    GameButton { text: "?"; selected: root.helpVisible; onClicked: { root.helpVisible = !root.helpVisible; if (root.helpVisible) root.send({ action: "pause" }); gameInput.forceActiveFocus(); } }
                    GameButton { text: "×"; onClicked: root.dismiss(); }
                }
                Item {
                    id: playArea
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    Item {
                        id: gameInput
                        anchors.centerIn: parent
                        width: Math.min(parent.width, parent.height * canvas.gameWidth / canvas.gameHeight)
                        height: width * canvas.gameHeight / canvas.gameWidth
                        focus: true
                        GameCanvas { id: canvas; objectName: "physicalCanvas"; anchors.fill: parent; theme: root.theme; fontFamily: Style.resolvedFontFamily; }
                        MouseArea {
                            anchors.fill: parent
                            hoverEnabled: true
                            acceptedButtons: Qt.LeftButton
                            onPositionChanged: function(mouse) { root.pointerX = mouse.x * canvas.gameWidth / width; root.pointerY = mouse.y * canvas.gameHeight / height; root.pointerDirty = true; }
                            onPressed: function(mouse) { gameInput.forceActiveFocus(); root.send({ action: "pointer", down: true, x: mouse.x * canvas.gameWidth / width, y: mouse.y * canvas.gameHeight / height }); }
                        }
                        Keys.onPressed: function(event) {
                            if (event.key === Qt.Key_Q && !root.state.editor) { root.dismiss(); event.accepted = true; return; }
                            if (event.key === Qt.Key_Question) { root.helpVisible = !root.helpVisible; if (root.helpVisible) root.send({ action: "pause" }); event.accepted = true; return; }
                            if (root.helpVisible && event.key === Qt.Key_Escape) { root.helpVisible = false; event.accepted = true; return; }
                            if (root.helpVisible) return;
                            root.send({ action: "key", key: root.keyName(event), code: root.keyName(event), down: true, repeat: event.isAutoRepeat, shift: (event.modifiers & Qt.ShiftModifier) !== 0 });
                            event.accepted = true;
                        }
                        Keys.onReleased: function(event) {
                            if (event.isAutoRepeat) return;
                            root.send({ action: "key", key: root.keyName(event), code: root.keyName(event), down: false });
                            event.accepted = true;
                        }
                    }
                    Rectangle {
                        anchors.centerIn: parent
                        width: Math.min(parent.width - Style.space(24), Style.space(740))
                        height: help.implicitHeight + Style.space(48)
                        visible: root.helpVisible
                        color: Color.menu.background
                        border.color: Color.menu.border
                        border.width: 1
                        radius: Style.cornerRadius
                        MouseArea { anchors.fill: parent; }
                        Text {
                            id: help
                            anchors.fill: parent
                            anchors.margins: Style.space(24)
                            text: "PHYSICAL\n\nMouse / ← → / A D   Move paddle\nSpace / Enter / click   Start or continue\nSpace / Esc   Pause\nPage Up / Down   Unlocked levels\nO   Automatic paddle (assisted)\nR   Debris on / off     S   Sound on / off\nM / Z   Magnets / zapper     2   Extra ball\nP   Diagnostics     F   Engine FPS\nE   Physics lab\n\nJ / K / L   Reverse / pause / forward replay\n← → / Home / End   Scrub replay\nEsc   Leave replay\nV / B   Export SVG / SVG with motion trails\n\n?   Help     Q   Close plugin\n\nClosing or switching windows pauses the game."
                            color: Color.foreground
                            font.family: Style.fontFamily
                            font.pixelSize: Style.font.subtitle
                            lineHeight: 1.25
                            wrapMode: Text.WordWrap
                        }
                    }
                    Loader {
                        anchors.top: parent.top
                        anchors.bottom: parent.bottom
                        anchors.right: parent.right
                        width: Math.min(Style.space(420), parent.width * 0.46)
                        active: root.state.editor === true && !root.helpVisible
                        sourceComponent: Tuner {
                            state: root.state
                            sliders: root.sliders
                            onCommand: function(data) { root.send(data); }
                            onDismissed: root.shortcut("e", "KeyE")
                        }
                    }
                }
                RowLayout {
                    Layout.fillWidth: true
                    Text {
                        Layout.fillWidth: true
                        text: root.errorText || canvas.renderError || root.notice || (root.state.replay ? "Replay · " + (root.state.replayFrame + 1) + " / " + root.state.replayFrames + " · " + root.state.replaySpeed + "×" : "Mouse / ← →   ·   Space pause   ·   J K L replay   ·   ? help")
                        color: root.errorText || canvas.renderError ? Color.urgent : Color.muted
                        font.family: Style.fontFamily
                        font.pixelSize: Style.font.bodySmall
                        elide: Text.ElideMiddle
                    }
                    Text { text: "Volume " + Math.round(volumeSlider.value * 100) + "%"; color: Color.muted; font.family: Style.fontFamily; font.pixelSize: Style.font.bodySmall; }
                    QQC.Slider {
                        id: volumeSlider
                        objectName: "audio-volume"
                        Layout.preferredWidth: Style.space(110)
                        implicitHeight: Style.space(24)
                        from: 0; to: 1; stepSize: 0.01
                        value: root.state.volume === undefined ? 0.5 : root.state.volume
                        onMoved: root.send({ action: "volume", value: value })
                        background: Rectangle {
                            x: volumeSlider.leftPadding
                            y: volumeSlider.topPadding + volumeSlider.availableHeight / 2 - height / 2
                            width: volumeSlider.availableWidth; height: 3; color: Color.muted
                            Rectangle { width: volumeSlider.visualPosition * parent.width; height: parent.height; color: Color.accent; }
                        }
                        handle: Rectangle {
                            x: volumeSlider.leftPadding + volumeSlider.visualPosition * (volumeSlider.availableWidth - width)
                            y: volumeSlider.topPadding + volumeSlider.availableHeight / 2 - height / 2
                            width: Style.space(12); height: width; radius: width / 2
                            color: volumeSlider.pressed ? Color.accent : Color.foreground
                        }
                    }
                    Text { text: (root.state.auto ? "AUTO · " : "") + (root.state.sound ? "SOUND ON" : "MUTED"); color: Color.muted; font.family: Style.fontFamily; font.pixelSize: Style.font.bodySmall; }
                }
            }
        }
    }
}
