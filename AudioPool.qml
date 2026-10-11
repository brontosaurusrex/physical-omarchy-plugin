import QtQuick
import QtMultimedia

Item {
    id: root
    property bool enabled: true
    property real volume: 0.5
    property int cursor: 0
    property int variation: 0
    function play(events) {
        if (!enabled) return;
        for (var i = 0; i < events.length; ++i) {
            var event = events[i], name = "";
            if (event.kind === "brick") name = "brick" + (++variation % 4);
            else if (event.kind === "magic") name = "magic" + (++variation % 4);
            else {
                var frequencies = [120, 240, 360, 480, 720, 960, 1440];
                var frequency = (event.args || [480])[0];
                var nearest = 0;
                for (var n = 1; n < frequencies.length; ++n) if (Math.abs(frequencies[n] - frequency) < Math.abs(frequencies[nearest] - frequency)) nearest = n;
                name = "tone" + frequencies[nearest];
            }
            var player = players.itemAt(cursor++ % players.count);
            if (player) player.playSound(Qt.resolvedUrl("sounds/" + name + ".wav"));
        }
    }
    function stop() { for (var i = 0; i < players.count; ++i) players.itemAt(i).stop(); }
    Repeater {
        id: players
        model: 12
        delegate: Item {
            function playSound(url) { effect.stop(); effect.source = url; effect.play(); }
            function stop() { effect.stop(); }
            SoundEffect { id: effect; volume: 0.18 * root.volume }
        }
    }
}
