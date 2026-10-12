import QtQuick
import "native" as Native

Item {
    id: root
    objectName: "physicalAudio"
    property bool enabled: true
    property real volume: 0.25
    property int lastBrickSample: -1
    property var roomSettings: ({})
    Native.AudioMixer { id: mixer; directory: Qt.resolvedUrl("sounds/"); volume: root.volume }
    readonly property var brickClips: ["brick0", "brick1", "brick2", "brick3", "brick4", "brick5", "brick6", "brick7", "brick8", "brick9", "brick10", "brick11", "brick12", "brick13", "brick14", "brick15", "brick16", "brick17", "brick18", "brick19", "brick20"]
    readonly property var featureClips: ["feature-bigpaddle", "feature-blackhole", "feature-breakunbreakable", "feature-dualballs", "feature-influencer", "feature-lowgravity", "feature-magnet", "feature-nuke", "feature-passthrough", "feature-reversegravity", "feature-speedboost", "feature-zapper"]
    readonly property var clips: ["wall", "paddle"].concat(brickClips, featureClips)

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
            return "";
        }
        return "";
    }
    function play(events) {
        for (var i=0;i<events.length;++i) {
            var event=events[i], args=event.args || [];
            if(event.kind === "stop-features") { mixer.stopFeatures(); continue; }
            if(event.kind === "stop-feature") { mixer.stopFeatures(event.feature); continue; }
            if(!enabled || volume<=0) continue;
            if(event.wave) {
                mixer.playTone(event.kind,event.wave,args[0],args[1],args[2],args[3],args[4],event.pan || 0);
            } else {
                var previousSample=lastBrickSample;
                var name=chooseClip(event);
                var accepted=mixer.playAvailable(name,event.rate || 1,event.gain === undefined ? .18 : event.gain,event.pan || 0,event.cutoff || 0,event.q || .707);
                if(!accepted && event.kind === "brick") lastBrickSample=previousSample;
                if(!accepted && event.feature) {
                    // Original fallback when all four feature voices are occupied.
                    var base=330*(.95+Math.random()*.10), pan=event.pan || 0;
                    mixer.playTone("tone","triangle",base,base*4/3,.09+Math.random()*.03,(.08+Math.random()*.03)*.8,0,pan);
                    mixer.playTone("tone","triangle",base*4/3,base*2,.10+Math.random()*.04,(.07+Math.random()*.03)*.8,.07+Math.random()*.03,pan);
                    var duration=.18+Math.random()*.06, amplitude=(.11+Math.random()*.04)*.8/3, delay=.14+Math.random()*.05;
                    var frequencies=[base*2,base*2.5,base*3];
                    for(var voice=0;voice<3;++voice) mixer.playTone("tone","sine",frequencies[voice]*(1+voice*.002),frequencies[voice]*.98,duration,amplitude,delay,pan);
                }
            }
        }
    }
    function isPlaying(name) { return mixer.playingAny(name); }
    function setRoom(settings) { roomSettings = settings; mixer.setRoom(settings); }
    function stop() { mixer.stop(); }
    onEnabledChanged: if (!enabled) stop()

    // Compatibility markers for diagnostics; voices share one native output stream.
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
                    readonly property bool ready: { mixer.directory; return sampleGroup.modelData === "wall" || sampleGroup.modelData === "paddle" || mixer.ready(sampleGroup.modelData); }
                    readonly property bool playing: {
                        if (sampleGroup.modelData !== "wall" && sampleGroup.modelData !== "paddle") return false;
                        mixer.revision; return mixer.playingAny(sampleGroup.modelData);
                    }
                    readonly property bool pending: false
                    function playSound(pitch) { mixer.playAvailable(sampleGroup.modelData, pitch); }
                    function stop() { mixer.stop(); }
                }
            }
        }
    }
}
