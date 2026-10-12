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
            id: ball
            anchors.centerIn: parent
            width: Math.max(1, Math.min(radioIconMetrics.tightBoundingRect.width, radioIconMetrics.tightBoundingRect.height) - 2 / Screen.devicePixelRatio)
            height: width
            // Preserve curvature instead of the rounded Rectangle's coarse
            // corner segments. The curve renderer includes edge antialiasing.
            preferredRendererType: Shape.CurveRenderer
            ShapePath {
                strokeWidth: -1
                fillColor: button.active && button.useActiveColor ? button.activeColor : button.foreground
                PathAngleArc {
                    centerX: ball.width / 2
                    centerY: ball.height / 2
                    radiusX: ball.width / 2
                    radiusY: ball.height / 2
                    startAngle: 0
                    sweepAngle: 360
                }
                Behavior on fillColor {
                    enabled: !root.bar || root.bar.foregroundAnimationEnabled
                    ColorAnimation { duration: 160 }
                }
            }
        }
        tooltipText: "Physical · native physics game"
        onPressed: function(mouseButton) {
            if (root.bar) root.bar.run("omarchy-shell shell toggle b.physical");
        }
    }
}
