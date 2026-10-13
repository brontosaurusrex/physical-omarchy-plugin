import QtQuick

Item {
    id: root
    property string icon: "menu"
    property color foreground: "white"
    property string fontFamily: "EnvyCodeR Nerd Font Mono"
    // Match the web controls' font glyphs; keep a play glyph for native playback.
    readonly property string symbol: icon === "pause" ? "󰏤" : icon === "play" ? "󰐊" : icon === "fullscreen" ? "" : "⋮"
    signal clicked()
    width: 28
    height: 28
    Rectangle { anchors.fill: parent; radius: 5; color: root.foreground; opacity: mouse.containsMouse ? 0.12 : 0; }
    Text {
        id: glyph
        objectName: "hud-glyph"
        anchors.fill: parent
        text: root.symbol
        color: root.foreground
        font.family: root.fontFamily
        font.pixelSize: 24
        font.bold: true
        horizontalAlignment: Text.AlignHCenter
        verticalAlignment: Text.AlignVCenter
        opacity: mouse.containsMouse ? 1 : 0.75
    }
    MouseArea { id: mouse; anchors.fill: parent; hoverEnabled: true; cursorShape: Qt.PointingHandCursor; onClicked: root.clicked(); }
}
