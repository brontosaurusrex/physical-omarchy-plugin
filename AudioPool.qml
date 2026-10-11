import QtQuick
import "native" as Native

Item {
    id: root
    objectName: "physicalAudio"
    property bool enabled: true
    property real volume: 0.25
    property int variation: 0
    Native.AudioMixer { id: mixer; directory: Qt.resolvedUrl("sounds/"); volume: root.volume }
    readonly property var clips: ["wall", "paddle", "brick0", "brick1", "brick2", "brick3", "magic0", "magic1", "magic2", "magic3", "tone120", "tone240", "tone360", "tone480", "tone720", "tone960", "tone1440"]

    function play(events) {
        if (!enabled || volume <= 0) return;
        for (var i = 0; i < events.length; ++i) {
            var event = events[i], name = "";
            if (event.kind === "wall" || event.kind === "paddle") name = event.kind;
            else if (event.kind === "brick") name = "brick" + (++variation % 4);
            else if (event.kind === "magic") name = "magic" + (++variation % 4);
            else {
                var frequencies = [120, 240, 360, 480, 720, 960, 1440];
                var frequency = (event.args || [480])[0], nearest = 0;
                for (var n = 1; n < frequencies.length; ++n) if (Math.abs(frequencies[n] - frequency) < Math.abs(frequencies[nearest] - frequency)) nearest = n;
                name = "tone" + frequencies[nearest];
            }
            var index = clips.indexOf(name);
            if (index >= 0) bank.itemAt(index).playSound();
        }
    }
    function stop() { mixer.stop(); }
    onEnabledChanged: if (!enabled) stop()

    // Two voices per clip share one persistent native audio output.
    Repeater {
        id: bank
        model: root.clips
        delegate: Item {
            id: sampleGroup
            required property string modelData
            property int cursor: 0
            function playSound() {
                // Use a free voice instead of stopping an active audio stream
                // for every hit in a dense brick/zapper burst.
                for (var i = 0; i < voices.count; ++i) {
                    var index = (cursor + i) % voices.count;
                    var voice = voices.itemAt(index);
                    if (voice && !voice.playing && !voice.pending) {
                        cursor = (index + 1) % voices.count;
                        voice.playSound();
                        return;
                    }
                }
            }
            function stop() { for (var i = 0; i < voices.count; ++i) voices.itemAt(i).stop(); }
            Repeater {
                id: voices
                model: 2
                delegate: Item {
                    id: voice
                    required property int index
                    objectName: "sound-" + sampleGroup.modelData + "-" + index
                    readonly property bool ready: { mixer.revision; return mixer.ready(sampleGroup.modelData); }
                    readonly property bool playing: { mixer.revision; return mixer.playing(sampleGroup.modelData, index); }
                    readonly property bool pending: false
                    function playSound() { mixer.play(sampleGroup.modelData, index); }
                    function stop() { mixer.stop(); }
                }
            }
        }
    }
}
