import QtQuick
import QtQuick.Window
import "native" as Native

Native.GameRenderer {
    id: canvas
    // The shell controls retain their live theme; game colors come from Go.
    property var theme: ({})
    property color surroundColor: "#1a1a2e"
    readonly property real cornerRadius: Math.min(8, width / 2, height / 2)

    // Four tiny cached images preserve circular edges with smooth coverage.
    // Only size/color changes repaint them; gameplay never redraws the masks.
    // This also avoids CurveRenderer's Qt/Mesa grabToImage crash with our FBO.
    Repeater {
        model: 4
        delegate: Item {
            required property int index
            readonly property real rasterScale: 4 * Screen.devicePixelRatio
            width: canvas.cornerRadius
            height: width
            x: index % 2 ? canvas.width - width : 0
            y: index >= 2 ? canvas.height - height : 0
            rotation: [0, 90, 270, 180][index]
            z: 6 // Above the renderer's foreground/effect layers.
            Canvas {
                objectName: "physical-corner-" + parent.index
                // Supersample at the current monitor's pixel density.
                width: parent.width * parent.rasterScale
                height: parent.height * parent.rasterScale
                transformOrigin: Item.TopLeft
                scale: 1 / parent.rasterScale
                smooth: true
                renderTarget: Canvas.Image
                property color maskColor: canvas.surroundColor
                onMaskColorChanged: requestPaint()
                onWidthChanged: requestPaint()
                onHeightChanged: requestPaint()
                onAvailableChanged: if (available) requestPaint()
                onPaint: {
                    var ctx = getContext("2d");
                    ctx.clearRect(0, 0, width, height);
                    ctx.fillStyle = maskColor;
                    ctx.beginPath();
                    ctx.moveTo(0, 0);
                    ctx.lineTo(0, height);
                    ctx.arc(width, height, width, Math.PI, 1.5 * Math.PI, false);
                    ctx.closePath();
                    ctx.fill();
                }
            }
        }
    }
}
