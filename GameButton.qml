import QtQuick
import "theme"

Rectangle {
    id: root
    property string text: ""
    property bool selected: false
    signal clicked()
    implicitWidth: label.implicitWidth + Style.space(24)
    implicitHeight: Style.space(34)
    radius: Math.min(Style.cornerRadius, Style.space(6))
    color: mouse.containsMouse || selected ? Color.menu.selectedBackground : "transparent"
    border.width: 1
    border.color: mouse.containsMouse || selected ? Color.accent : Color.muted
    opacity: enabled ? 1 : 0.45
    Text { id: label; anchors.centerIn: parent; text: root.text; color: root.selected ? Color.accent : Color.foreground; font.family: Style.fontFamily; font.pixelSize: Style.font.body; }
    MouseArea { id: mouse; anchors.fill: parent; hoverEnabled: true; cursorShape: Qt.PointingHandCursor; onClicked: root.clicked(); }
}
