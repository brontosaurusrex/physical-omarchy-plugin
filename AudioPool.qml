import QtQuick
import "native" as Native

Item {
    id: root
    objectName: "physicalAudio"
    property bool enabled: true
    property real volume: 0.25
    property int variation: 0
    property int lastBrickSample: -1
    Native.AudioMixer { id: mixer; directory: Qt.resolvedUrl("sounds/"); volume: root.volume }
    readonly property var brickClips: ["brick0", "brick1", "brick2", "brick3", "brick4", "brick5", "brick6", "brick7", "brick8", "brick9", "brick10", "brick11", "brick12", "brick13", "brick14", "brick15", "brick16", "brick17", "brick18", "brick19", "brick20"]
    readonly property var featureClips: ["feature-bigpaddle", "feature-blackhole", "feature-breakunbreakable", "feature-dualballs", "feature-influencer", "feature-lowgravity", "feature-magnet", "feature-nuke", "feature-passthrough", "feature-reversegravity", "feature-speedboost", "feature-zapper"]
    readonly property var clips: ["wall", "paddle", "magic0", "magic1", "magic2", "magic3", "tone120", "tone240", "tone360", "tone480", "tone720", "tone960", "tone1440"].concat(brickClips, featureClips)

    function chooseClip(event) {
        if (event.kind === "wall" || event.kind === "paddle") return event.kind;
        if (event.kind === "brick") {
            var count = brickClips.length;
            var index = Math.floor(Math.random() * (lastBrickSample < 0 ? count : count - 1));
            if (lastBrickSample >= 0 && index >= lastBrickSample) ++index;
            lastBrickSample = index;
            return brickClips[index];
        }
        if (event.kind === "magic") {
            var cue = "feature-" + String(event.feature || "").trim().toLowerCase();
            if (featureClips.indexOf(cue) >= 0) return cue;
            return "magic" + (++variation % 4);
        }
        var frequencies = [120, 240, 360, 480, 720, 960, 1440];
        var frequency = (event.args || [480])[0], nearest = 0;
        for (var n = 1; n < frequencies.length; ++n) if (Math.abs(frequencies[n] - frequency) < Math.abs(frequencies[nearest] - frequency)) nearest = n;
        return "tone" + frequencies[nearest];
    }
    function play(events) {
        if (!enabled || volume <= 0) return;
        for (var i = 0; i < events.length; ++i) {
            var event = events[i], index = clips.indexOf(chooseClip(event));
            var pitch = (event.kind === "brick" || event.kind === "magic") && event.args ? event.args[1] : 1;
            if (index >= 0) bank.itemAt(index).playSound(pitch > 0 ? pitch : 1);
        }
    }
    function isPlaying(name) { return mixer.playing(name, 0) || mixer.playing(name, 1); }
    function stop() { mixer.stop(); }
    onEnabledChanged: if (!enabled) stop()

    // Two voices per clip share one persistent native audio output.
    Repeater {
        id: bank
        model: root.clips
        delegate: Item {
            id: sampleGroup
            required property string modelData
            function playSound(pitch) { mixer.playAvailable(modelData, pitch); }

            Repeater {
                id: voices
                model: 2
                delegate: Item {
                    id: voice
                    required property int index
                    objectName: "sound-" + sampleGroup.modelData + "-" + index
                    readonly property bool ready: { mixer.directory; return mixer.ready(sampleGroup.modelData); }
                    readonly property bool playing: {
                        if (sampleGroup.modelData !== "wall" && sampleGroup.modelData !== "paddle") return false;
                        mixer.revision; return mixer.playing(sampleGroup.modelData, index);
                    }
                    readonly property bool pending: false
                    function playSound(pitch) { mixer.play(sampleGroup.modelData, index, pitch); }
                    function stop() { mixer.stop(); }
                }
            }
        }
    }
}
