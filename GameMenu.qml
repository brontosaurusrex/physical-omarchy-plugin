import QtQuick
import QtQuick.Controls as QQC
import QtQuick.Layouts
import qs.Commons

Rectangle {
    id: root
    required property var controller
    implicitHeight: contents.implicitHeight + 32
    color: Color.menu.background
    border.color: Color.menu.border
    border.width: 1
    radius: Style.cornerRadius
    function key(key, code) { controller.closeMenu(); controller.shortcut(key, code); }
    Keys.onPressed: function(event) {
        if (event.key === Qt.Key_Escape) { root.controller.closeMenu(); event.accepted = true; }
    }
    MouseArea { anchors.fill: parent; acceptedButtons: Qt.AllButtons; }
    QQC.ScrollView {
        anchors.fill: parent
        anchors.margins: 16
        clip: true
        contentWidth: availableWidth
        ColumnLayout {
            id: contents
            width: parent.width
            spacing: Style.space(10)
            RowLayout {
                Layout.fillWidth: true
                Text { text: "PHYSICAL"; color: Color.accent; font.family: Style.fontFamily; font.pixelSize: Style.font.heading; Layout.fillWidth: true; }
                GameButton { text: "×"; onClicked: root.controller.closeMenu(); }
            }
            Text { text: "Level " + root.controller.state.level + " / " + root.controller.state.levels; color: Color.muted; font.family: Style.fontFamily; font.pixelSize: Style.font.body; }
            RowLayout {
                GameButton { text: root.controller.state.ready ? "Start" : "Resume"; onClicked: root.key(" ", "Space"); }
                GameButton { text: "Retry"; onClicked: { root.controller.closeMenu(); root.controller.send({action: "restart"}); } }
                GameButton { text: "Fullscreen"; onClicked: { root.controller.closeMenu(); root.controller.toggleFullscreen(); } }
            }
            RowLayout {
                Layout.fillWidth: true
                GameButton { text: root.controller.state.sound ? "Sound on" : "Muted"; selected: root.controller.state.sound === true; onClicked: root.controller.shortcut("s", "KeyS"); }
                Text { text: Math.round(volume.value * 100) + "%"; color: Color.foreground; font.family: Style.fontFamily; font.pixelSize: Style.font.body; }
                QQC.Slider {
                    id: volume
                    objectName: "audio-volume"
                    Layout.fillWidth: true
                    Layout.minimumWidth: 60
                    implicitHeight: Style.space(24)
                    from: 0; to: 1; stepSize: 0.01
                    value: root.controller.state.volume === undefined ? 0.25 : root.controller.state.volume
                    onMoved: root.controller.send({action: "volume", value: value})
                    background: Rectangle {
                        x: volume.leftPadding
                        y: volume.topPadding + volume.availableHeight / 2 - height / 2
                        width: volume.availableWidth; height: 3; color: Color.muted
                        Rectangle { width: volume.visualPosition * parent.width; height: parent.height; color: Color.accent; }
                    }
                    handle: Rectangle {
                        x: volume.leftPadding + volume.visualPosition * (volume.availableWidth - width)
                        y: volume.topPadding + volume.availableHeight / 2 - height / 2
                        width: Style.space(12); height: width; radius: width / 2
                        color: volume.pressed ? Color.accent : Color.foreground
                    }
                }
            }
            RowLayout {
                GameButton { text: "Debris"; selected: root.controller.state.debris === true; onClicked: root.controller.shortcut("r", "KeyR"); }
                GameButton { text: "Auto"; selected: root.controller.state.auto === true; onClicked: root.controller.shortcut("o", "KeyO"); }
                GameButton { text: "Lab"; selected: root.controller.state.editor === true; onClicked: root.key("e", "KeyE"); }
            }
            RowLayout {
                GameButton { text: "Replay"; onClicked: root.key("k", "KeyK"); }
                GameButton { text: "FPS"; onClicked: root.controller.shortcut("f", "KeyF"); }
                GameButton { text: "Controls"; selected: root.controller.helpVisible; onClicked: root.controller.helpVisible = !root.controller.helpVisible; }
            }
            Text {
                Layout.fillWidth: true
                visible: root.controller.helpVisible
                text: "Mouse / ← → / A D   Move paddle\nRight click   Capture / release mouse\nSpace / Enter / click   Start or continue\nSpace / Esc   Pause\nSpace + F   Fullscreen\nPage Up / Down   Unlocked levels\nO   Automatic paddle (assisted)\nR   Debris    S   Sound\nM / Z   Magnets / zapper    2   Extra ball\nP   Diagnostics    F   Engine FPS\nE   Physics lab\n\nJ / K / L   Reverse / pause / forward replay\n← → / Home / End   Scrub replay\nEsc   Release mouse / exit fullscreen / replay\nV / B   Export SVG / SVG with trails\n\n?   Controls    Q   Close plugin\nSwitching windows pauses the game."
                color: Color.foreground
                font.family: Style.fontFamily
                font.pixelSize: Style.font.bodySmall
                lineHeight: 1.3
                wrapMode: Text.WordWrap
            }
            Text {
                Layout.fillWidth: true
                text: root.controller.state.replay ? "Replay " + (root.controller.state.replayFrame + 1) + " / " + root.controller.state.replayFrames + " · " + root.controller.state.replaySpeed + "×" : "Right click captures the mouse · ? shows controls"
                color: Color.muted
                font.family: Style.fontFamily
                font.pixelSize: Style.font.bodySmall
                wrapMode: Text.WordWrap
            }
            GameButton { text: "Close game"; onClicked: root.controller.dismiss(); }
        }
    }
}
