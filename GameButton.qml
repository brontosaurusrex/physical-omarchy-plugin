import QtQuick
import "theme" as Theme

Rectangle {
    id: root
    property string text: ""
    property bool selected: false
    signal clicked()
    implicitWidth: label.implicitWidth + Theme.Style.space(24)
    implicitHeight: Theme.Style.space(34)
    radius: Math.min(Theme.Style.cornerRadius, Theme.Style.space(6))
    color: mouse.containsMouse || selected ? Theme.Color.menu.selectedBackground : "transparent"
    border.width: 1
    border.color: mouse.containsMouse || selected ? Theme.Color.accent : Theme.Color.muted
    opacity: enabled ? 1 : 0.45
    Text { id: label; anchors.centerIn: parent; text: root.text; color: root.selected ? Theme.Color.accent : Theme.Color.foreground; font.family: Theme.Style.fontFamily; font.pixelSize: Theme.Style.font.body; }
    MouseArea { id: mouse; anchors.fill: parent; hoverEnabled: true; cursorShape: Qt.PointingHandCursor; onClicked: root.clicked(); }
}
