import QtQuick

Item {
    id: root
    property string icon: "menu"
    property color foreground: "white"
    signal clicked()
    width: 28
    height: 28
    onIconChanged: glyph.requestPaint()
    onForegroundChanged: glyph.requestPaint()
    Rectangle { anchors.fill: parent; radius: 5; color: root.foreground; opacity: mouse.containsMouse ? 0.12 : 0; }
    Canvas {
        id: glyph
        anchors.fill: parent
        opacity: mouse.containsMouse ? 1 : 0.75
        onPaint: {
            var c = getContext("2d"); c.reset(); c.strokeStyle = root.foreground; c.fillStyle = root.foreground; c.lineWidth = 1.6;
            if (root.icon === "pause") { c.fillRect(9, 7, 3, 14); c.fillRect(16, 7, 3, 14); }
            else if (root.icon === "play") { c.beginPath(); c.moveTo(10,7); c.lineTo(21,14); c.lineTo(10,21); c.closePath(); c.fill(); }
            else if (root.icon === "fullscreen") {
                c.beginPath(); c.moveTo(12,7); c.lineTo(7,7); c.lineTo(7,12); c.moveTo(16,7); c.lineTo(21,7); c.lineTo(21,12);
                c.moveTo(7,16); c.lineTo(7,21); c.lineTo(12,21); c.moveTo(16,21); c.lineTo(21,21); c.lineTo(21,16); c.stroke();
            } else { for (var y = 7; y <= 21; y += 7) { c.beginPath(); c.arc(14,y,1.6,0,Math.PI*2); c.fill(); } }
        }
    }
    MouseArea { id: mouse; anchors.fill: parent; hoverEnabled: true; cursorShape: Qt.PointingHandCursor; onClicked: root.clicked(); }
}
