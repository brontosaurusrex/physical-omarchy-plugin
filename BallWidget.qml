import QtQuick
import QtQuick.Window
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
        hasVisualContent: true
        labelVisible: false
        fixedWidth: vertical ? barSize : Style.space(12) + scaledHorizontalMargin * 2
        TextMetrics {
            id: radioIconMetrics
            text: "\uf0ac"
            font.family: button.fontFamily
            font.pixelSize: button.fontSize
        }
        Rectangle {
            anchors.centerIn: parent
            width: Math.max(1, Math.min(radioIconMetrics.tightBoundingRect.width, radioIconMetrics.tightBoundingRect.height) - 2 / Screen.devicePixelRatio)
            height: width
            radius: width / 2
            color: button.foreground
            antialiasing: true
        }
        tooltipText: "Physical · native physics game"
        onPressed: function(mouseButton) {
            if (root.bar) root.bar.run("omarchy-shell shell toggle b.physical");
        }
    }
}
