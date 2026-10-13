import QtQuick
import QtQuick.Controls as QQC
import QtQuick.Layouts
import "theme"
import "native" as Native

Item {
    id: root
    required property var windowView
    required property var surface
    readonly property var panel: windowView
    property var launchScreen: null
    property var prepareWindow: function() {}
    signal dismissed()
    signal clipboardRequested(string text)
    property bool opened: false
    property bool engineReady: false
    // Replies can arrive after the next display callback. A two-frame bound
    // avoids halving the display rate while keeping the request queue short.
    property int ticksPending: 0
    property real pendingFrameTime: 0
    property bool menuVisible: false
    property bool spaceHeld: false
    property bool fullscreenChord: false
    readonly property bool fullscreen: panel ? panel.fullscreen : false
    readonly property bool mouseLocked: pointerLock.locked
    readonly property color backdropColor: "#1a1a2e"
    readonly property real playfieldMargin: 6
    readonly property real gameAspectRatio: canvas.gameWidth / canvas.gameHeight
    function initialWindowSize(screenWidth, screenHeight) {
        // Fit the playfield after reserving its blue surround and shell frame.
        var inset = 2 * (playfieldMargin + 2);
        var maxWidth = Math.min(1280, Math.floor(screenWidth * 0.9));
        var maxHeight = Math.floor(screenHeight * 0.9);
        var contentHeight = Math.max(1, Math.floor(Math.min((maxWidth - inset) / gameAspectRatio, maxHeight - inset)));
        return Qt.size(Math.floor(contentHeight * gameAspectRatio) + inset, contentHeight + inset);
    }
    readonly property size defaultWindowSize: initialWindowSize(panel && panel.screen ? panel.screen.width : 1280, panel && panel.screen ? panel.screen.height : 760)
    readonly property int defaultWindowWidth: defaultWindowSize.width
    readonly property int defaultWindowHeight: defaultWindowSize.height
    readonly property color hudColor: state.palette && state.palette.length > 4 ? state.palette[4] : Color.foreground
    function focusGame() { Qt.callLater(function() { gameInput.forceActiveFocus(); }); }
    function toggleMenu() {
        menuVisible = !menuVisible;
        if (menuVisible) { pointerLock.release(); send({action: "pause"}); }
        focusGame();
    }
    function closeMenu() { menuVisible = false; focusGame(); }
    function toggleFullscreen() { pointerLock.release(); panel.fullscreen = !panel.fullscreen; focusGame(); }
    function handleEscape() {
        if (state.replay) {
            pointerLock.release();
            closeMenu();
            send({action:"replay-exit"});
        } else dismiss();
    }
    function pressKey(event) {
        if (event.key === Qt.Key_Space) { if (!event.isAutoRepeat) { spaceHeld = true; fullscreenChord = false; } return; }
        if (event.key === Qt.Key_F && spaceHeld) { if (!event.isAutoRepeat) { toggleFullscreen(); fullscreenChord = true; } return; }
        if (event.key === Qt.Key_Question) { if (!menuVisible && !event.isAutoRepeat) toggleMenu(); return; }
        if (event.key === Qt.Key_Escape) { if (!event.isAutoRepeat) handleEscape(); return; }
        if (event.key === Qt.Key_Q && !state.editor) { dismiss(); return; }
        if (event.key === Qt.Key_S && !event.isAutoRepeat && !menuVisible && !state.editor) { send({action:"toggle-sound"}); return; }
        if (event.key === Qt.Key_K && (event.modifiers & Qt.ShiftModifier) && state.replay) { send({action:"replay-exit"}); return; }
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
    property int frameReplies: 0
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
    readonly property var gameView: canvas
    readonly property var contentView: card
    property bool windowSetupReady: true
    readonly property string enginePath: Qt.resolvedUrl("bin/physical-engine").toString().replace(/^file:\/\//, "")
    readonly property var theme: ({ background: String(Color.background), foreground: String(Color.foreground), accent: String(Color.accent), urgent: String(Color.urgent), muted: String(Color.muted) })
    // The original engine keeps debris moving behind the completion table.
    // Use the display clock until it disappears, then return to idle polling.
    readonly property bool animating: opened && engineReady && !menuVisible && ((!state.paused && !state.ready && !state.gameOver && (!state.complete || state.debrisPieces > 0) && (!state.editor || state.live)) || (state.replay && state.replayDirection !== 0))
    onAnimatingChanged: pendingFrameTime = 0

    function send(data) {
        if (!engineReady || !engine.running) return;
        engine.write(JSON.stringify(data) + "\n");
    }
    function open(payloadJson) {
        if (launchScreen) panel.screen = launchScreen;
        opened = true;
        if (windowSetupReady) panel.visible = true;
        else prepareWindow();
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
        pointerDirty = false;
    }
    function suspend() {
        pointerLock.release(); spaceHeld = false; fullscreenChord = false;
        send({action:"pause"}); audio.stop();
    }
    function dismiss() {
        close();
        dismissed();
    }
    function shortcut(key, code) {
        send({ action: "key", key: key, code: code || key, down: true });
        send({ action: "key", key: key, code: code || key, down: false });
        gameInput.forceActiveFocus();
    }
    function togglePlayback() {
        shortcut(state.replay ? (state.replayDirection === 0 ? "l" : "k") : " ", "Space");
    }
    function keyName(event) {
        var special = ({});
        special[Qt.Key_Left] = "ArrowLeft"; special[Qt.Key_Right] = "ArrowRight";
        special[Qt.Key_Up] = "ArrowUp"; special[Qt.Key_Down] = "ArrowDown";
        special[Qt.Key_Space] = " "; special[Qt.Key_Return] = "Enter"; special[Qt.Key_Enter] = "Enter";
        special[Qt.Key_Escape] = "Escape"; special[Qt.Key_PageUp] = "PageUp"; special[Qt.Key_PageDown] = "PageDown";
        special[Qt.Key_Home] = "Home"; special[Qt.Key_End] = "End";
        if (special[event.key]) return special[event.key];
        if (event.text) return event.text;
        // Qt test/remote key events can carry a key code with empty text.
        if ((event.key >= Qt.Key_A && event.key <= Qt.Key_Z) || (event.key >= Qt.Key_0 && event.key <= Qt.Key_9)) return String.fromCharCode(event.key).toLowerCase();
        return "";
    }
    function receive(line) {
        try {
            var packet = canvas.acceptLine(line);
            if (packet.type !== "frame") return;
            frameReplies++;
            var first = !engineReady;
            engineReady = true;
            if (packet.action === "tick") ticksPending = Math.max(0, ticksPending - 1);
            state = packet.state;
            if (packet.sliders) sliders = packet.sliders;
            if (packet.room) audio.setRoom(packet.room);
            if (packet.sounds && opened) audio.play(packet.sounds);
            if (packet.clipboard) { clipboardRequested(packet.clipboard); notice = "Settings copied"; noticeTimer.restart(); }
            if (packet.export) { notice = "Saved " + packet.export; noticeTimer.restart(); }
            if (packet.error) errorText = packet.error;
            if (first) { send({ action: "visibility", down: opened }); }
        } catch (error) { errorText = "Native engine: " + String(error); }
    }
    function tick(frameDelta) {
        // Advance by the display clock, not by when the separate process reads
        // its pipe. Preserve elapsed time if our bounded queue skips a request.
        if (frameDelta !== undefined) {
            // Empty clock updates must not catch up time spent on idle screens.
            if (!(frameDelta > 0)) return;
            pendingFrameTime += frameDelta;
        }
        if (ticksPending >= 2 || !engineReady || !opened) return;
        if (pointerDirty) { pointerDirty = false; send({ action: "pointer", x: pointerX, y: pointerY }); }
        ticksPending++;
        var elapsed = pendingFrameTime;
        pendingFrameTime = 0;
        send(elapsed > 0 ? { action: "tick", dt: elapsed } : { action: "tick" });
    }

    FontLoader { id: gameFont; source: "fonts/EnvyCodeRNerdFontMono-Bold.ttf"; }
    Native.EngineProcess {
        id: engine
        command: [root.enginePath]
        onLineReceived: function(line) { root.receive(line); }
        onErrorLine: function(line) { if (/panic|fatal|Invalid input/.test(line)) root.errorText = line; }
        onExited: function(exitCode) {
            root.engineReady = false;
            root.ticksPending = 0;
            if (root.opened) root.errorText = "Physical engine stopped (" + exitCode + "). Reopen Physical to retry.";
        }
    }
    Timer { interval: 100; repeat: true; running: root.opened && root.engineReady && !root.animating; onTriggered: root.tick(); }
    Timer { id: noticeTimer; interval: 6000; onTriggered: root.notice = ""; }
    AudioPool { id: audio; enabled: root.state.sound === true; volume: root.state.volume === undefined ? 0.25 : root.state.volume; }
    Item {
        id: card
        parent: root.surface
        anchors.fill: parent
        Shortcut { sequence: "Escape"; context: Qt.WindowShortcut; enabled: root.opened; autoRepeat: false; onActivated: root.handleEscape(); }
        Item {
            id: playArea
            anchors.fill: parent
            anchors.margins: root.playfieldMargin + (panel.fullscreen ? 0 : 2)
            Item {
                id: gameInput
                anchors.centerIn: parent
                width: Math.min(parent.width, parent.height * canvas.gameWidth / canvas.gameHeight)
                height: width * canvas.gameHeight / canvas.gameWidth
                focus: true
                onWidthChanged: pointerLock.release()
                onHeightChanged: pointerLock.release()
                GameCanvas { id: canvas; objectName: "physicalCanvas"; frameClockRunning: root.animating; onFrameTick: function(delta) { root.tick(delta); } anchors.fill: parent; theme: root.theme; surroundColor: root.backdropColor; fontFamily: gameFont.status === FontLoader.Ready ? gameFont.name : Style.resolvedFontFamily; }
                Native.PointerLock {
                    id: pointerLock
                    anchors.fill: parent
                    onLockedChanged: { root.pointerDirty = false; root.send({action: "pointer-lock", down: locked}); }
                    onRelativeMotion: function(dx, dy) { root.send({action: "relative-pointer", x: dx * canvas.gameWidth / gameInput.width}); }
                }
                MouseArea {
                    objectName: "physicalPointerInput"
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
                        if (mouse.button === Qt.LeftButton) root.send({ action: "pointer", down: true, x: mouse.x * canvas.gameWidth / width, y: mouse.y * canvas.gameHeight / height });
                    }
                    onClicked: function(mouse) {
                        if (mouse.button !== Qt.RightButton) return;
                        // Finish the button grab and focus change before requesting
                        // compositor capture, especially on the first click.
                        Qt.callLater(function() {
                            if (!root.opened || root.menuVisible) return;
                            if (pointerLock.requested) pointerLock.release();
                            else if (!pointerLock.lock()) { root.notice = "Mouse capture unavailable"; noticeTimer.restart(); }
                        });
                    }
                }
                Keys.onPressed: function(event) { root.pressKey(event); event.accepted = true; }
                Keys.onReleased: function(event) { root.releaseKey(event); event.accepted = true; }
                Row {
                    readonly property real gameScale: gameInput.width / canvas.gameWidth
                    anchors.top: parent.top
                    anchors.right: parent.right
                    // Match the rendered HUD's x=22 and first baseline y=37,
                    // accounting for blank space around the button glyphs.
                    anchors.topMargin: (37 + hudTextMetrics.tightBoundingRect.y) * gameScale - playbackButton.inkBounds.y
                    anchors.rightMargin: (22 + hudTextMetrics.tightBoundingRect.x) * gameScale - (menuButton.width - menuButton.inkBounds.x - menuButton.inkBounds.width)
                    spacing: 8
                    TextMetrics { id: hudTextMetrics; font.family: canvas.fontFamily; font.pixelSize: 18; font.bold: true; text: "Lives:" }
                    HudButton { id: playbackButton; objectName: "playback-toggle"; icon: root.state.replay ? (root.state.replayDirection === 0 ? "play" : "pause") : (root.state.paused || root.state.ready ? "play" : "pause"); foreground: root.hudColor; fontFamily: canvas.fontFamily; onClicked: { pointerLock.release(); root.togglePlayback(); } }
                    HudButton { icon: "fullscreen"; foreground: root.hudColor; fontFamily: canvas.fontFamily; onClicked: root.toggleFullscreen(); }
                    HudButton { id: menuButton; icon: "menu"; foreground: root.hudColor; fontFamily: canvas.fontFamily; onClicked: root.toggleMenu(); }
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
