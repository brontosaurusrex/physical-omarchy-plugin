pragma Singleton
import QtQuick

QtObject {
    property var source: null
    readonly property string fontFamily: source ? source.fontFamily : Qt.application.font.family
    readonly property string resolvedFontFamily: source ? source.resolvedFontFamily : fontFamily
    readonly property int cornerRadius: source ? source.cornerRadius : 8
    readonly property int normalBorderWidth: source ? source.normalBorderWidth : 1
    readonly property var font: source ? source.font : ({body: 14, bodySmall: 12, heading: 18, subtitle: 15})
    function space(value) { return source ? source.space(value) : value; }
}
