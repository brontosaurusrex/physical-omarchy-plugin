import QtQuick
import QtQuick.Window
import "theme"
import "native" as Native

Window {
    id: window
    objectName: "physicalWindow"
    title: "Physical"
    property bool fullscreen: false
    width: game.defaultWindowWidth
    height: game.defaultWindowHeight
    minimumWidth: Math.min(640, game.defaultWindowWidth)
    minimumHeight: Math.min(360, game.defaultWindowHeight)
    visibility: fullscreen ? Window.FullScreen : (game.opened ? Window.Windowed : Window.Hidden)
    color: game.backdropColor
    onActiveChanged: if (!active && game.opened) game.suspend()
    onClosing: game.close()
    Rectangle {
        id: frame
        anchors.fill: parent
        color: window.color
        radius: window.fullscreen ? 0 : Style.cornerRadius
        border.width: window.fullscreen ? 0 : Style.normalBorderWidth
        border.color: Color.menu.border
    }
    GameCore {
        id: game
        objectName: "physicalGame"
        windowView: window
        surface: frame
        onDismissed: Qt.quit()
        onClipboardRequested: function(text) { desktop.copyText(text); }
        Component.onCompleted: open("{}")
    }
    Native.EngineProcess { id: desktop }
}
