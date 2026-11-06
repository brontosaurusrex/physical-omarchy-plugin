#include "../native/audiomixer.h"
#include "../bin/audiomixer.moc"
#include <QCoreApplication>
#include <QElapsedTimer>
#include <QThread>
#include <cstdio>
#include <cstdlib>

class AudioOutputTest {
    static void require(bool condition,const char *message) {
        if(!condition) { std::fprintf(stderr,"AUDIO OUTPUT FAILED: %s\n",message); std::exit(1); }
    }
    template<class Predicate> static bool wait(Predicate ready,int timeout=5000) {
        QElapsedTimer elapsed;elapsed.start();
        while(!ready() && elapsed.elapsed()<timeout) {
            QCoreApplication::processEvents();QThread::msleep(5);
        }
        return ready();
    }
    static QAudioDevice find(const QString &name) {
        for(const auto &device:QMediaDevices::audioOutputs())
            if(device.description()==name)return device;
        return {};
    }
    static bool active(AudioMixer &mixer) {
        return mixer.sink && mixer.started && mixer.sink->error()==QtAudio::NoError &&
            (mixer.sink->state()==QtAudio::ActiveState || mixer.sink->state()==QtAudio::IdleState);
    }
public:
    static void run(const QString &first,const QString &second,const QString &sounds) {
        require(wait([&]{return !find(first).isNull() && !find(second).isNull();}),"virtual outputs unavailable");
        AudioMixer mixer;
        mixer.setVolume(0); // The test never produces audible sound.
        mixer.setDirectory(QUrl::fromLocalFile(sounds));
        require(mixer.ready("brick0"),"sample bank failed to load");
        const QVariantMap settings{{"dry",.4},{"inputType","lowpass"},{"inputHz",20000.},{"inputQ",0.},
            {"outputType","lowpass"},{"outputHz",20000.},{"outputQ",0.},{"feedbackHz",5000.},{"feedbackQ",.45},
            {"delay1",.01},{"delay2",.02},{"delay3",.04},{"gain1",.4},{"gain2",.28},{"gain3",.2},{"feedback",.5}};
        mixer.setRoom(settings);
        const auto a=find(first), b=find(second);
        mixer.configureOutput(a);
        // Muted play requests intentionally do not start a stream. Start the
        // silent output directly to exercise the actual backend lifecycle.
        mixer.startOutput();
        const bool firstReady=wait([&]{return active(mixer) && mixer.sink->processedUSecs()>50000;});
        if(!firstReady) std::fprintf(stderr,"OUTPUT DEBUG sink=%d started=%d wanted=%d state=%d error=%d device=%s default=%s open=%d rate=%d channels=%d sampleFormat=%d processed=%lld buffer=%lld\n",bool(mixer.sink),mixer.started,mixer.outputWanted,mixer.sink?int(mixer.sink->state()):-1,mixer.sink?int(mixer.sink->error()):-1,mixer.outputDeviceId.constData(),QMediaDevices::defaultAudioOutput().id().constData(),mixer.output->isOpen(),mixer.format.sampleRate(),mixer.format.channelCount(),int(mixer.format.sampleFormat()),mixer.sink?static_cast<long long>(mixer.sink->processedUSecs()):-1,mixer.sink?static_cast<long long>(mixer.sink->bufferSize()):-1);
        require(firstReady,"first output did not process audio");
        std::printf("AUDIO OUTPUT buffer_us=%lld\n",static_cast<long long>(mixer.format.durationForBytes(mixer.sink->bufferSize())));
        // Replace a live sink with a different output, retaining sample/room/gain
        // settings and discarding voices generated before the switch.
        {
            QMutexLocker lock(&mixer.mutex);
            AudioMixer::Voice stale;
            stale.name="stale-collision";stale.clip=&mixer.clips["brick0"];
            stale.position=-8*stale.clip->rate;
            mixer.voices.append(stale);
        }
        mixer.configureOutput(b);
        require(mixer.outputDeviceId==b.id() && mixer.voices.isEmpty(),"device switch retained old voices");
        require(mixer.ready("brick0") && mixer.roomSettings==settings && mixer.gain==0,"device switch lost mixer settings");
        require(wait([&]{return active(mixer) && mixer.sink->processedUSecs()>50000;}),"second output failed to resume");
        for(int i=0;i<12;++i) {
            const auto &device=i%2 ? b : a;
            mixer.configureOutput(device);
            require(wait([&]{return active(mixer) && mixer.sink->processedUSecs()>20000;}),"repeated output switching stalled");
        }
        mixer.configureOutput(a);
        require(wait([&]{return active(mixer);}),"return to first output failed");
        std::puts("AUDIO OUTPUT REMOVE FIRST");std::fflush(stdout);
        // Python removes only the test sink. The real QMediaDevices notification
        // must select the user's existing default output and restart playback.
        require(wait([&]{return find(first).isNull() && mixer.outputDeviceId==QMediaDevices::defaultAudioOutput().id() && active(mixer);}),"removed device did not recover onto the default output");
        require(mixer.ready("brick0") && mixer.voices.isEmpty() && mixer.roomSettings==settings,"removal replayed voices or lost room settings");
        const auto version=mixer.version;
        mixer.sink->stop();
        require(wait([&]{return mixer.version>version && active(mixer);}),"stopped stream did not recover without a device-list change");
        mixer.configureOutput({});
        require(!mixer.sink && !mixer.started && mixer.voices.isEmpty(),"no-output state retained a stream");
        mixer.playTone("unavailable","sine",440,440,.1,.1,0,0);
        require(mixer.voices.isEmpty(),"audio queued while no device was available");
        mixer.refreshOutput();
        require(wait([&]{return active(mixer);}),"return from no-output state failed");
        require(mixer.gain==0 && mixer.ready("brick0") && mixer.roomSettings==settings,"recovery changed volume, samples or room");
        std::puts("AUDIO OUTPUT PASS: live switch, device removal, stopped-stream retry, no-output recovery, no stale sounds and retained settings");
    }
};
int main(int argc,char **argv) {
    QCoreApplication app(argc,argv);
    QCoreApplication::setApplicationName("Physical silent output recovery test");
    if(argc!=4)return 2;
    AudioOutputTest::run(QString::fromLocal8Bit(argv[1]),QString::fromLocal8Bit(argv[2]),QString::fromLocal8Bit(argv[3]));
}
