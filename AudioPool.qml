import QtQuick
import QtMultimedia

Item {
    id: root
    objectName: "physicalAudio"
    property bool enabled: true
    property real volume: 0.5
    property int variation: 0
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
    function stop() { for (var i = 0; i < bank.count; ++i) { var clip = bank.itemAt(i); if (clip) clip.stop(); } }
    onEnabledChanged: if (!enabled) stop()

    // Sources never change during playback. Two preloaded voices per clip keep
    // brick activity from cancelling or reloading the collision sounds.
    Repeater {
        id: bank
        model: root.clips
        delegate: Item {
            id: sampleGroup
            required property string modelData
            property int cursor: 0
            function playSound() { voices.itemAt(cursor++ % voices.count).playSound(); }
            function stop() { for (var i = 0; i < voices.count; ++i) voices.itemAt(i).stop(); }
            Repeater {
                id: voices
                model: 2
                delegate: Item {
                    id: voice
                    required property int index
                    objectName: "sound-" + sampleGroup.modelData + "-" + index
                    readonly property bool ready: effect.status === SoundEffect.Ready
                    readonly property bool playing: effect.playing
                    property bool pending: false
                    function playSound() {
                        effect.stop();
                        pending = effect.status !== SoundEffect.Ready;
                        if (!pending) effect.play();
                    }
                    function stop() { pending = false; effect.stop(); }
                    SoundEffect {
                        id: effect
                        source: Qt.resolvedUrl("sounds/" + sampleGroup.modelData + ".wav")
                        volume: 0.18 * root.volume
                        onStatusChanged: {
                            if (status === SoundEffect.Ready && voice.pending) {
                                voice.pending = false;
                                if (root.enabled && root.volume > 0) play();
                            }
                        }
                    }
                }
            }
        }
    }
}
