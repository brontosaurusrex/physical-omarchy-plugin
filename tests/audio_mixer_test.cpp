#include "../native/audiomixer.h"
#include "../bin/audiomixer.moc"
#include <QCoreApplication>
#include <cstdio>
#include <cstdlib>
#include <complex>

class AudioMixerTest {
    static void require(bool condition,const char *message) {
        if(!condition) { std::fprintf(stderr,"FAIL: %s\n",message); std::exit(1); }
    }
public:
    static void testRoom() {
        // A low/high-pass filter's gain at cutoff is 10^(Q/20), per Web Audio.
        for(const auto &type:{QString("lowpass"),QString("highpass")}) {
            AudioBiquad filter; filter.configure(type,2400,.4,48000);
            const auto z=std::polar(1.,-2*std::acos(-1.)*2400/48000);
            const double response=std::abs((filter.b0+filter.b1*z+filter.b2*z*z)/(1.+filter.a1*z+filter.a2*z*z));
            require(std::abs(response-std::pow(10.,.4/20))<1e-10,"filter resonance uses linear Q instead of Web Audio dB");
        }
        AudioBiquad band; band.configure("bandpass",1800,2.8,48000);
        const auto z=std::polar(1.,-2*std::acos(-1.)*1800/48000);
        require(std::abs(std::abs((band.b0+band.b1*z+band.b2*z*z)/(1.+band.a1*z+band.a2*z*z))-1)<1e-10,"bandpass center gain changed");
        QVariantMap settings{{"dry",.4},{"inputType","lowpass"},{"inputHz",20000.},{"inputQ",0.},
                             {"outputType","lowpass"},{"outputHz",20000.},{"outputQ",0.},
                             {"feedbackHz",5000.},{"feedbackQ",.45},
                             {"delay1",.01},{"delay2",.02},{"delay3",.04},
                             {"gain1",.4},{"gain2",.28},{"gain3",.2},{"feedback",.5}};
        AudioRoom room; room.configure(settings,48000);
        for(int i=0;i<8000;++i) { double stereo[2]={0,0}; room.process(stereo); }
        double energy[4]={0,0,0,0};
        for(int i=0;i<4800;++i) {
            double stereo[2]={i==0 ? 1. : 0.,0}; room.process(stereo);
            require(std::isfinite(stereo[0]) && std::isfinite(stereo[1]),"room emitted nonfinite impulse response");
            require(stereo[1]==0,"room leaked left input into right output");
            if(i==0) require(std::abs(stereo[0]-.4)<1e-10,"dry branch gain incorrect");
            else if(i<480) require(stereo[0]==0,"reflection arrived before first delay");
            const int taps[]={480,960,1920,3840};
            for(int tap=0;tap<4;++tap) if(i>=taps[tap] && i<taps[tap]+100) energy[tap]+=stereo[0]*stereo[0];
        }
        for(double value:energy) require(value>1e-5,"reflection tap or damped feedback tail missing");
        room.clear();
        for(int i=0;i<4800;++i) { double stereo[2]={0,0}; room.process(stereo); require(stereo[0]==0 && stereo[1]==0,"stop left stale room echoes"); }
        // Exercise level changes, fractional delays and all three filter types.
        for(const auto &type:{QString("highpass"),QString("bandpass"),QString("lowpass")}) {
            settings["inputType"]=type; settings["inputHz"]=type=="bandpass" ? 1800. : 600.;
            settings["inputQ"]=type=="bandpass" ? 2.8 : .8;
            settings["outputHz"]=750.; settings["dry"]=0.; settings["delay1"]=.00811;
            room.configure(settings,48000);
            for(int i=0;i<48000;++i) {
                double stereo[2]={std::sin(i*.05)*.1,std::sin(i*.09)*.1}; room.process(stereo);
                for(double value:stereo) require(std::isfinite(value) && std::abs(value)<2,"room transition became unstable");
            }
        }
        settings["dry"]=1.; room.configure(settings,48000);
        for(int i=0;i<8000;++i) { double stereo[2]={0,0}; room.process(stereo); }
        double dry[2]={.1,-.2}; room.process(dry);
        require(dry[0]==.1 && dry[1]==-.2,"dry room did not bypass wet graph");
        settings["dry"]=.4; room.configure(settings,48000);
        for(int i=0;i<48000;++i) { double stereo[2]={0,0}; room.process(stereo); require(stereo[0]==0 && stereo[1]==0,"returning from dry room resurrected stale echoes"); }
    }
    static void run() {
        testRoom();
        AudioMixer mixer;
        // Never start hardware output: deterministic pulls exercise the same mixer.
        mixer.format.setSampleRate(48000);
        mixer.format.setChannelCount(2);
        mixer.format.setSampleFormat(QAudioFormat::Float);
        if(!mixer.sink) mixer.sink=new QAudioSink(mixer.format,&mixer);
        mixer.started=true;
        mixer.setVolume(.25);
        mixer.playTone("wall","triangle",230,175,.1,.2,.02,-1);
        require(mixer.voices.size()==1,"generated tone was not queued");
        const auto &voice=mixer.voices.first();
        // Compare the recurrent sweep/envelope with the original closed formula.
        double phase=0;
        for(int i=0;i<5760;++i) {
            double t=double(i)/48000;
            double step=230*std::exp(std::log(175./230)*std::min(t/.1,1.))/48000;
            double envelope=t<.008 ? .0001*std::pow(.2/.0001,t/.008) : .2*std::pow(.0001/.2,std::clamp((t-.008)/(.1-.008),0.,1.));
            double expected=(1-4*std::abs(phase-.5))*envelope;
            require(std::abs(voice.clip->samples[i]-expected)<1e-6,"optimized sweep/envelope changed PCM");
            phase=std::fmod(phase+step,1.);
        }
        require(voice.owned && voice.clip->samples.size()==5760,"generated duration/lifetime incorrect");
        QVector<float> data(960*2);
        require(mixer.mix(reinterpret_cast<char *>(data.data()),data.size()*sizeof(float))==data.size()*sizeof(float),"audio read length incorrect");
        for(float value:data) require(value==0,"delayed tone started too early");
        data.resize(5760*2);
        mixer.mix(reinterpret_cast<char *>(data.data()),data.size()*sizeof(float));
        float peak=0;
        for(int i=0;i<data.size();i+=2) {
            require(std::isfinite(data[i]) && std::isfinite(data[i+1]),"nonfinite audio sample");
            peak=std::max(peak,std::abs(data[i]));
            require(std::abs(data[i+1])<1e-6,"full-left mono pan leaked to right channel");
        }
        require(peak>.01 && peak<=.051,"volume/envelope gain incorrect");
        require(mixer.voices.isEmpty(),"completed tone retained voice memory");

        AudioMixer::Clip clip; clip.rate=48000; clip.channels=2;
        clip.samples.fill(.25f,24000);
        for(int i=0;i<9;++i) mixer.clips.insert(QString("brick%1").arg(i),clip);
        for(int i=0;i<8;++i) require(mixer.playAvailable(QString("brick%1").arg(i),1,.05,.75,2400,.4),"normal sample rejected before eight-voice cap");
        require(!mixer.playAvailable("brick8"),"normal sample cap exceeded");
        data.resize(480*2);
        mixer.mix(reinterpret_cast<char *>(data.data()),data.size()*sizeof(float));
        for(float value:data) require(std::isfinite(value) && std::abs(value)<=1,"filtered sample overflow");
        mixer.stop();
        for(int i=0;i<5;++i) mixer.clips.insert(QString("feature-%1").arg(i),clip);
        for(int i=0;i<4;++i) require(mixer.playAvailable(QString("feature-%1").arg(i)),"feature rejected before four-voice cap");
        require(!mixer.playAvailable("feature-4"),"feature cap exceeded");
        require(mixer.playAvailable("feature-0"),"same feature failed to retrigger at capacity");
        int active=0,fading=0;
        for(const auto &v:mixer.voices) { if(v.fadeLeft<0) ++active; else ++fading; }
        require(active==4 && fading==1,"feature retrigger layered active copies");
        require(mixer.voices.first().fadeLeft==864,"retrigger fade was not 18 ms");
        require(std::abs(mixer.voices.last().position+475.2)<1e-6,"replacement delay was not 9.9 ms");
        data.resize(864*2);
        mixer.mix(reinterpret_cast<char *>(data.data()),data.size()*sizeof(float));
        require(mixer.voices.size()==4,"retired feature voice did not finish fading");
        mixer.stopFeatures();
        mixer.mix(reinterpret_cast<char *>(data.data()),data.size()*sizeof(float));
        require(mixer.voices.isEmpty(),"stopFeatures retained feature voices");
        mixer.setVolume(0);
        require(!mixer.playAvailable("brick0"),"muted sample was queued");
        mixer.playTone("tone","sine",440,220,.1,.2,0,0);
        require(mixer.voices.isEmpty(),"muted tone was queued");
        std::puts("PASS: native tone duration/delay/gain/pan, Web Audio filter response, room reflections/feedback/transitions, voice caps, feature retrigger/fade and mute");
    }
};
int main(int argc,char **argv) { QCoreApplication app(argc,argv); AudioMixerTest::run(); }
