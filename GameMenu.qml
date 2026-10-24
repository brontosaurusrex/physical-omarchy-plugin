import QtQuick
import QtQuick.Controls as QQC
import QtQuick.Layouts
import "theme"

Rectangle {
    id: root
    required property var controller
    implicitHeight: contents.implicitHeight + 32
    color: Color.menu.background
    border.color: Color.menu.border
    border.width: 1
    radius: Style.cornerRadius
    onVisibleChanged: if (visible && menuScroll.contentItem) menuScroll.contentItem.contentY = 0
    readonly property var shortcutSections: [
        {title: "Game", rows: [
            ["Right click", "Capture / release mouse"],
            ["Mouse", "Move paddle"],
            ["←", "Move left"], ["→", "Move right"],
            ["A", "Move left"], ["D", "Move right"],
            ["Space", "Start, pause / resume"],
            ["Enter", "Start, retry / continue"],
            ["Left click", "Start, retry / continue"],
            ["Space + F", "Toggle fullscreen"],
            ["Page Up", "Previous unlocked level"],
            ["Page Down", "Next unlocked level"],
            ["S", "Toggle sound"],
            ["?", "Open this menu"],
            ["Esc", "Close game / exit replay"],
            ["Q", "Close game (outside lab)"]
        ]},
        {title: "Extras", rows: [
            ["O", "Continuous demo / auto paddle (assisted)"],
            ["R", "Toggle debris"],
            ["M", "Toggle magnets (assisted)"],
            ["Z", "Toggle zapper (assisted)"],
            ["2", "Extra ball (assisted)"],
            ["T", "Rescue ball (assisted)"],
            ["P", "Cycle diagnostics"],
            ["I", "Cycle diagnostics"],
            ["F", "Show engine FPS"],
            ["Shift + F", "Reset lowest FPS"],
            ["U", "Unlock levels temporarily"],
            ["E", "Open / close physics lab"]
        ]},
        {title: "Replay", rows: [
            ["J", "Play backwards"],
            ["K", "Enter / pause replay"],
            ["L", "Play forwards"],
            ["←", "Previous frame"],
            ["→", "Next frame"],
            ["Home", "First frame"],
            ["End", "Last frame"],
            ["Esc", "Return to game"],
            ["Shift + K", "Return to game"],
            ["V", "Export SVG"],
            ["B", "Export SVG with trails"]
        ]}
    ]
    function key(key, code) { controller.closeMenu(); controller.shortcut(key, code); }
    Keys.onPressed: function(event) {
        if (event.key === Qt.Key_Escape) { if (!event.isAutoRepeat) root.controller.handleEscape(); event.accepted = true; }
    }
    MouseArea { anchors.fill: parent; acceptedButtons: Qt.AllButtons; }
    QQC.ScrollView {
        id: menuScroll
        objectName: "menu-scroll"
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
            Text {
                objectName: "mouse-capture-hint"
                Layout.fillWidth: true
                text: "Right click in the game to capture the mouse.\nRight click again to release it."
                color: Color.foreground
                font.family: Style.fontFamily
                font.pixelSize: Style.font.body
                wrapMode: Text.WordWrap
            }
            RowLayout {
                GameButton { objectName: "menu-playback"; text: root.controller.state.replay ? "Play replay" : (root.controller.state.ready ? "Start" : "Resume"); onClicked: { root.controller.closeMenu(); root.controller.togglePlayback(); } }
                GameButton { text: "Retry"; onClicked: { root.controller.closeMenu(); root.controller.send({action: "restart"}); } }
                GameButton { text: "Fullscreen"; onClicked: { root.controller.closeMenu(); root.controller.toggleFullscreen(); } }
            }
            RowLayout {
                Layout.fillWidth: true
                GameButton { text: root.controller.state.sound ? "Sound on" : "Muted"; selected: root.controller.state.sound === true; onClicked: root.controller.send({action:"toggle-sound"}); }
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
                GameButton { objectName: "toggle-debris"; enabled: !root.controller.state.replay; text: "Debris"; selected: root.controller.state.debrisMaster === true; onClicked: root.controller.shortcut("r", "KeyR"); }
                GameButton { objectName: "toggle-auto"; enabled: !root.controller.state.replay; text: "Demo"; selected: root.controller.state.auto === true; onClicked: root.controller.shortcut("o", "KeyO"); }
                GameButton { objectName: "toggle-lab"; enabled: !root.controller.state.replay; text: "Lab"; selected: root.controller.state.editor === true; onClicked: root.key("e", "KeyE"); }
            }
            RowLayout {
                GameButton { objectName: "replay-toggle"; text: root.controller.state.replay ? "Exit replay" : "Replay"; onClicked: { if (root.controller.state.replay) { root.controller.closeMenu(); root.controller.send({action:"replay-exit"}); } else root.key("k", "KeyK"); } }
                GameButton { objectName: "toggle-fps"; enabled: !root.controller.state.replay; text: "FPS"; onClicked: root.controller.shortcut("f", "KeyF"); }
            }
            Rectangle { Layout.fillWidth: true; implicitHeight: 1; color: Color.menu.border; }
            ColumnLayout {
                objectName: "controls-table"
                Layout.fillWidth: true
                spacing: Style.space(12)
                Text { text: "Controls"; color: Color.foreground; font.family: Style.fontFamily; font.pixelSize: Style.font.subtitle; font.bold: true; }
                Repeater {
                    model: root.shortcutSections
                    delegate: ColumnLayout {
                        required property var modelData
                        Layout.fillWidth: true
                        spacing: Style.space(6)
                        Text { text: modelData.title; color: Color.accent; font.family: Style.fontFamily; font.pixelSize: Style.font.bodySmall; font.bold: true; }
                        Repeater {
                            model: modelData.rows
                            delegate: RowLayout {
                                required property var modelData
                                Layout.fillWidth: true
                                spacing: Style.space(12)
                                Text {
                                    text: modelData[0]
                                    Layout.preferredWidth: Math.min(Style.space(120), root.width * 0.35)
                                    Layout.alignment: Qt.AlignTop
                                    color: Color.foreground
                                    font.family: Style.fontFamily
                                    font.pixelSize: Style.font.bodySmall
                                    font.bold: true
                                    wrapMode: Text.WordWrap
                                }
                                Text {
                                    text: modelData[1]
                                    Layout.fillWidth: true
                                    color: Color.foreground
                                    font.family: Style.fontFamily
                                    font.pixelSize: Style.font.bodySmall
                                    wrapMode: Text.WordWrap
                                }
                            }
                        }
                    }
                }
            }
            Text {
                Layout.fillWidth: true
                text: root.controller.state.replay ? "Replay " + (root.controller.state.replayFrame + 1) + " / " + root.controller.state.replayFrames + " · " + root.controller.state.replaySpeed + "×" : "Switching windows pauses the game."
                color: Color.foreground
                font.family: Style.fontFamily
                font.pixelSize: Style.font.bodySmall
                wrapMode: Text.WordWrap
            }
            GameButton { text: "Close game"; onClicked: root.controller.dismiss(); }
        }
    }
}
