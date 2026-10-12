import QtQuick
import QtQuick.Controls as QQC
import QtQuick.Layouts
import "theme"

Rectangle {
    id: root
    required property var state
    required property var sliders
    signal command(var data)
    signal dismissed()
    color: Color.menu.background
    border.color: Color.menu.border
    border.width: 1
    radius: Style.cornerRadius

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: Style.space(16)
        spacing: Style.space(10)
        RowLayout {
            Layout.fillWidth: true
            Text { text: "PHYSICS LAB"; color: Color.accent; font.family: Style.fontFamily; font.pixelSize: Style.font.heading; Layout.fillWidth: true; }
            GameButton { text: "Close"; onClicked: root.dismissed(); }
        }
        Text { text: "Changes mark this run as assisted."; color: Color.muted; font.family: Style.fontFamily; font.pixelSize: Style.font.body; }
        Flow {
            Layout.fillWidth: true
            Layout.preferredHeight: childrenRect.height
            spacing: Style.space(8)
            GameButton { text: root.state.live ? "Live" : "Frozen"; selected: root.state.live === true; onClicked: root.command({ action: "setting", group: "toggle", key: "live", down: !root.state.live }); }
            GameButton { text: "Debris"; selected: root.state.debris === true; onClicked: root.command({ action: "setting", group: "toggle", key: "debris", down: !root.state.debris }); }
            GameButton { text: "Corners"; selected: root.state.corner === true; onClicked: root.command({ action: "setting", group: "toggle", key: "corner", down: !root.state.corner }); }
            GameButton { text: "All corners"; selected: root.state.allCorners === true; onClicked: root.command({ action: "setting", group: "toggle", key: "allCorners", down: !root.state.allCorners }); }
            GameButton { text: "Defaults"; onClicked: root.command({ action: "tuner-reset" }); }
            GameButton { text: "Copy settings"; onClicked: root.command({ action: "tuner-export" }); }
        }
        QQC.ScrollView {
            Layout.fillWidth: true
            Layout.fillHeight: true
            clip: true
            contentWidth: availableWidth
            Column {
                width: parent.width
                spacing: Style.space(10)
                Repeater {
                    model: root.sliders
                    delegate: Column {
                        id: row
                        required property var modelData
                        required property int index
                        width: parent.width
                        spacing: Style.space(4)
                        Text {
                            visible: row.index === 0 || root.sliders[row.index - 1].group !== row.modelData.group
                            height: visible ? implicitHeight + Style.space(8) : 0
                            text: row.modelData.group
                            color: Color.accent
                            font.family: Style.fontFamily
                            font.pixelSize: Style.font.subtitle
                        }
                        Row {
                            width: parent.width
                            Text { width: parent.width * 0.7; text: row.modelData.label; color: Color.foreground; font.family: Style.fontFamily; font.pixelSize: Style.font.body; }
                            Text { width: parent.width * 0.3; horizontalAlignment: Text.AlignRight; text: Number(slider.value).toFixed(row.modelData.precision); color: Color.muted; font.family: Style.fontFamily; font.pixelSize: Style.font.body; }
                        }
                        QQC.Slider {
                            id: slider
                            objectName: "physics-" + row.modelData.key
                            width: parent.width
                            implicitHeight: Style.space(24)
                            height: implicitHeight
                            from: row.modelData.min
                            to: row.modelData.max
                            stepSize: row.modelData.step
                            value: row.modelData.value
                            onMoved: root.command({ action: "setting", group: row.modelData.kind, key: row.modelData.key, value: value });
                            background: Rectangle {
                                x: slider.leftPadding
                                y: slider.topPadding + slider.availableHeight / 2 - height / 2
                                width: slider.availableWidth
                                height: 3
                                color: Color.muted
                                Rectangle { width: slider.visualPosition * parent.width; height: parent.height; color: Color.accent; }
                            }
                            handle: Rectangle {
                                x: slider.leftPadding + slider.visualPosition * (slider.availableWidth - width)
                                y: slider.topPadding + slider.availableHeight / 2 - height / 2
                                width: Style.space(12); height: width; radius: width / 2
                                color: slider.pressed ? Color.accent : Color.foreground
                            }
                        }
                    }
                }
            }
        }
    }
}
