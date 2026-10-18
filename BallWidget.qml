import QtQuick
import QtQuick.Window
import QtQuick.Shapes
import qs.Commons
import qs.Ui as Ui

Ui.BarWidget {
    id: root
    moduleName: "b.physical"
    implicitWidth: button.implicitWidth
    implicitHeight: button.implicitHeight
    Ui.WidgetButton {
        id: button
        anchors.fill: parent
        bar: root.bar
        active: !!(root.bar && root.bar.shell && typeof root.bar.shell.isPluginOpen === "function" && root.bar.shell.isPluginOpen(root.moduleName))
        hasVisualContent: true
        labelVisible: false
        fixedWidth: vertical ? barSize : Style.space(12) + scaledHorizontalMargin * 2
        TextMetrics {
            id: radioIconMetrics
            text: "\uf0ac"
            font.family: button.fontFamily
            font.pixelSize: button.fontSize
        }
        Shape {
            id: icon
            anchors.centerIn: parent
            width: Math.max(1, Math.min(radioIconMetrics.tightBoundingRect.width, radioIconMetrics.tightBoundingRect.height) - 2 / Screen.devicePixelRatio)
            height: width
            preferredRendererType: Shape.CurveRenderer
            property color iconColor: button.active && button.useActiveColor ? button.activeColor : button.foreground
            Behavior on iconColor {
                enabled: !root.bar || root.bar.foregroundAnimationEnabled
                ColorAnimation { duration: 160 }
            }

            // Ball above the paddle; both fit within the previous icon bounds.
            ShapePath {
                strokeWidth: -1
                fillColor: icon.iconColor
                PathAngleArc {
                    centerX: icon.width * 0.70
                    centerY: icon.height * 0.23
                    radiusX: icon.width * 0.15
                    radiusY: icon.height * 0.15
                    startAngle: 0
                    sweepAngle: 360
                }
            }
            ShapePath {
                strokeWidth: -1
                fillColor: icon.iconColor
                startX: icon.width * 0.14
                startY: icon.height * 0.73
                PathLine { x: icon.width * 0.86; y: icon.height * 0.73 }
                PathArc {
                    x: icon.width * 0.86; y: icon.height * 0.91
                    radiusX: icon.width * 0.09; radiusY: icon.height * 0.09
                    direction: PathArc.Clockwise
                }
                PathLine { x: icon.width * 0.14; y: icon.height * 0.91 }
                PathArc {
                    x: icon.width * 0.14; y: icon.height * 0.73
                    radiusX: icon.width * 0.09; radiusY: icon.height * 0.09
                    direction: PathArc.Clockwise
                }
            }
        }
        tooltipText: "Physical · native physics game"
        onPressed: function(mouseButton) {
            if (root.bar) root.bar.run("omarchy-shell shell toggle b.physical");
        }
    }
}
