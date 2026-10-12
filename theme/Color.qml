pragma Singleton
import QtQuick

QtObject {
    id: root
    // The plugin supplies Omarchy's live singleton; a standalone app uses Qt's palette.
    property var source: null
    property SystemPalette system: SystemPalette { colorGroup: SystemPalette.Active }
    readonly property color background: source ? source.background : system.window
    readonly property color foreground: source ? source.foreground : system.windowText
    readonly property color accent: source ? source.accent : system.highlight
    readonly property color urgent: source ? source.urgent : "#d05b5b"
    readonly property color muted: source ? source.muted : system.mid
    readonly property var menu: source ? source.menu : ({background: background, border: muted, selectedBackground: system.alternateBase})
}
