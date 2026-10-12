#pragma once
#include <QAudioSink>
#include <QFile>
#include <QDir>
#include <QIODevice>
#include <QMediaDevices>
#include <QMutex>
#include <QMutexLocker>
#include <QObject>
#include <QTimer>
#include <QUrl>
#include <QtEndian>
#include <algorithm>
#include <cmath>
#include <cstring>
#include <memory>
#include "audioroom.h"

// One persistent output stream mixes all collision voices. Rapid hits never
// create, stop or restart backend audio streams.
class AudioMixer : public QObject {
    Q_OBJECT
    friend class AudioMixerTest;
    Q_PROPERTY(QUrl directory READ directory WRITE setDirectory NOTIFY directoryChanged)
    Q_PROPERTY(qreal volume READ volume WRITE setVolume NOTIFY volumeChanged)
    Q_PROPERTY(quint64 revision READ revision NOTIFY revisionChanged)
    struct Clip { QVector<float> samples; int rate=48000; int channels=1; };
    struct Voice {
        QString name; int index=0; double position=0, pitch=1; const Clip *clip=nullptr;
        std::shared_ptr<const Clip> owned;
        double amplitude=1, pan=0, panLeft=1,panRight=1,panCross=0, b0=1,b1=0,b2=0,a1=0,a2=0,z1[2]={0,0},z2[2]={0,0};
        int fadeLeft=-1, fadeTotal=1;
    };
    class Output : public QIODevice {
        AudioMixer *mixer;
    public:
        explicit Output(AudioMixer *owner):QIODevice(owner),mixer(owner) { open(ReadOnly); }
        bool isSequential() const override { return true; }
        qint64 bytesAvailable() const override { return 65536 + QIODevice::bytesAvailable(); }
        qint64 readData(char *data,qint64 length) override { return mixer->mix(data,length); }
        qint64 writeData(const char *,qint64) override { return -1; }
    };
    QUrl folder;
    qreal gain=.25;
    quint64 version=0;
    QHash<QString,Clip> clips;
    QVector<Voice> voices;
    QMutex mutex;
    QAudioFormat format;
    AudioRoom room;
    QAudioSink *sink=nullptr;
    Output *output=nullptr;
    QTimer status;
    bool started=false;
    int nextVoice=0;
    void startOutput() {
        if(!started) { sink->start(output); started=true; }
        status.start(); ++version; emit revisionChanged();
    }
    static void setPan(Voice &v,double pan) {
        v.pan=std::clamp(pan,-1.,1.);
        const double angle=(v.clip->channels==1 ? (v.pan+1)/4 : std::abs(v.pan)/2)*std::acos(-1.);
        if(v.clip->channels==1) { v.panLeft=std::cos(angle); v.panRight=std::sin(angle); }
        else { v.panLeft=v.pan>0 ? std::cos(angle) : 1; v.panRight=v.pan<0 ? std::cos(angle) : 1; v.panCross=std::sin(angle); }
    }
    void filter(Voice &voice,double hz,double q) {
        if(hz<=0) return;
        AudioBiquad filter;
        filter.configure("lowpass",hz,q,format.sampleRate());
        voice.b0=filter.b0; voice.b1=filter.b1; voice.b2=filter.b2;
        voice.a1=filter.a1; voice.a2=filter.a2;
    }
    static double blep(double phase,double step) {
        if(phase<step) { double t=phase/step; return t+t-t*t-1; }
        if(phase>1-step) { double t=(phase-1)/step; return t*t+t+t+1; }
        return 0;
    }
    qint64 mix(char *data,qint64 length) {
        QMutexLocker lock(&mutex);
        const int bytes=format.bytesPerSample(), channels=format.channelCount();
        if (!bytes || !channels) return 0;
        const qint64 frames=length/(bytes*channels);
        for (qint64 frame=0;frame<frames;++frame) {
            double stereo[2]={0,0};
            for (auto &voice:voices) {
                const auto &clip=*voice.clip;
                const int count=clip.samples.size()/clip.channels;
                if(voice.position<0) { voice.position+=double(clip.rate)/format.sampleRate(); continue; }
                const int index=int(voice.position);
                if(index>=count || voice.fadeLeft==0) continue;
                double value[2]={0,0};
                for(int channel=0;channel<clip.channels;++channel) {
                    const double fraction=voice.position-index;
                    const double a=clip.samples[index*clip.channels+channel], b=clip.samples[qMin(index+1,count-1)*clip.channels+channel];
                    double x=a+(b-a)*fraction;
                    value[channel]=voice.b0*x+voice.z1[channel];
                    voice.z1[channel]=voice.b1*x-voice.a1*value[channel]+voice.z2[channel];
                    voice.z2[channel]=voice.b2*x-voice.a2*value[channel];
                }
                if(clip.channels==1) {
                    value[1]=value[0]*voice.panRight; value[0]*=voice.panLeft;
                } else if(voice.pan<0) {
                    value[0]+=value[1]*voice.panCross; value[1]*=voice.panRight;
                } else {
                    value[1]+=value[0]*voice.panCross; value[0]*=voice.panLeft;
                }
                const double fade=voice.fadeLeft<0 ? 1 : double(voice.fadeLeft--)/voice.fadeTotal;
                for(int channel=0;channel<2;++channel) stereo[channel]+=value[channel]*voice.amplitude*fade;
                voice.position+=double(clip.rate)/format.sampleRate()*voice.pitch;
            }
            room.process(stereo);
            stereo[0]*=gain; stereo[1]*=gain;
            for (int channel=0;channel<channels;++channel) {
                float sample=std::clamp(float(channels==1 ? (stereo[0]+stereo[1])*.70710678 : stereo[channel%2]),-1.f,1.f);
                char *target=data+(frame*channels+channel)*bytes;
                switch(format.sampleFormat()) {
                case QAudioFormat::Float: std::memcpy(target,&sample,4); break;
                case QAudioFormat::Int16: { qint16 value=qint16(sample*32767); std::memcpy(target,&value,2); break; }
                case QAudioFormat::Int32: { qint32 value=qint32(double(sample)*2147483647); std::memcpy(target,&value,4); break; }
                case QAudioFormat::UInt8: *target=char(qRound((sample+1)*127.5f)); break;
                default: break;
                }
            }
        }
        voices.erase(std::remove_if(voices.begin(),voices.end(),[this](const Voice &v){ return v.fadeLeft==0 || v.position>=v.clip->samples.size()/v.clip->channels; }),voices.end());
        return frames*bytes*channels;
    }
public:
    explicit AudioMixer(QObject *parent=nullptr):QObject(parent) {
        const auto device=QMediaDevices::defaultAudioOutput();
        if (!device.isNull()) {
            format=device.preferredFormat();
            sink=new QAudioSink(device,format,this);
            sink->setBufferSize(format.bytesForDuration(40000));
            output=new Output(this);
        }
        status.setInterval(30);
        connect(&status,&QTimer::timeout,this,[this]{
            bool empty; { QMutexLocker lock(&mutex); empty=voices.isEmpty(); }
            ++version; emit revisionChanged();
            if(empty) status.stop();
        });
    }
    ~AudioMixer() override { if(sink) sink->stop(); }
    QUrl directory() const { return folder; }
    qreal volume() const { return gain; }
    quint64 revision() const { return version; }
    void setVolume(qreal value) { { QMutexLocker lock(&mutex); gain=qBound(0.,value,1.); } emit volumeChanged(); }
    void setDirectory(const QUrl &value) {
        if(folder==value) return;
        stop(); folder=value;
        QHash<QString,Clip> loaded;
        const QDir directory(folder.toLocalFile());
        const auto files=directory.entryList({"*.wav"},QDir::Files,QDir::Name);
        for(const auto &filename:files) {
            const QString name=filename.chopped(4);
            QFile file(directory.filePath(filename));
            if(!file.open(QIODevice::ReadOnly)) continue;
            const QByteArray wav=file.readAll();
            if(wav.size()<12 || wav.left(4)!="RIFF" || wav.mid(8,4)!="WAVE") continue;
            Clip clip; QByteArray pcm; int encoding=0,channels=0,bits=0;
            for(qsizetype offset=12;offset+8<=wav.size();) {
                const quint32 size=qFromLittleEndian<quint32>(wav.constData()+offset+4);
                if(size>quint32(wav.size()-offset-8)) break;
                const auto id=wav.mid(offset,4); const char *chunk=wav.constData()+offset+8;
                if(id=="fmt " && size>=16) {
                    encoding=qFromLittleEndian<quint16>(chunk); channels=qFromLittleEndian<quint16>(chunk+2);
                    clip.rate=qFromLittleEndian<quint32>(chunk+4); bits=qFromLittleEndian<quint16>(chunk+14);
                } else if(id=="data") pcm=wav.mid(offset+8,size);
                offset+=8+qsizetype(size)+(size&1);
            }
            if(encoding!=1 || (channels!=1 && channels!=2) || bits!=16 || clip.rate<=0 || pcm.isEmpty()) continue;
            clip.channels=channels;
            clip.samples.reserve(pcm.size()/2);
            for(qsizetype i=0;i+1<pcm.size();i+=2) clip.samples.append(qFromLittleEndian<qint16>(pcm.constData()+i)/32768.f);
            loaded.insert(name,std::move(clip));
        }
        { QMutexLocker lock(&mutex); clips=std::move(loaded); }
        emit directoryChanged(); ++version; emit revisionChanged();
    }
    Q_INVOKABLE bool ready(const QString &name) { QMutexLocker lock(&mutex); return clips.contains(name); }
    Q_INVOKABLE void setRoom(const QVariantMap &settings) {
        QMutexLocker lock(&mutex); room.configure(settings,format.sampleRate());
    }
    Q_INVOKABLE bool playing(const QString &name,int index) {
        QMutexLocker lock(&mutex);
        for(const auto &v:voices) if(v.name==name && v.index==index) return true;
        return false;
    }
    Q_INVOKABLE bool playingAny(const QString &name) {
        QMutexLocker lock(&mutex);
        for(const auto &v:voices) if(v.name==name && v.fadeLeft!=0) return true;
        return false;
    }
    Q_INVOKABLE void playTone(const QString &name,const QString &wave,double start,double end,double duration,double amplitude,double delay,double pan) {
        if(!sink || !std::isfinite(start+end+duration+amplitude+delay+pan)) return;
        duration=std::clamp(duration,.009,4.); delay=std::clamp(delay,0.,8.);
        start=std::clamp(start,1.,20000.); end=std::clamp(end,1.,20000.);
        auto clip=std::make_shared<Clip>(); clip->rate=48000;
        const int count=qRound((duration+.02)*clip->rate);
        clip->samples.reserve(count);
        double phase=0;
        const double peak=std::max(amplitude,.0001), durationFrames=duration*clip->rate;
        const double frequencyStep=std::exp(std::log(end/start)/durationFrames);
        const double attackStep=std::pow(peak/.0001,1./384);
        const double decayStep=std::pow(.0001/peak,1./(durationFrames-384));
        double frequency=start, envelope=.0001;
        for(int i=0;i<count;++i) {
            const double step=std::min(.45,frequency/clip->rate);
            double sample;
            if(wave=="triangle") sample=1-4*std::abs(phase-.5);
            else if(wave=="sawtooth") sample=2*phase-1-blep(phase,step);
            else if(wave=="square") sample=(phase<.5 ? 1 : -1)+blep(phase,step)-blep(std::fmod(phase+.5,1.),step);
            else sample=std::sin(phase*2*std::acos(-1.));
            if(i==384) envelope=peak;
            clip->samples.append(float(sample*envelope));
            envelope=i<384 ? envelope*attackStep : (i<durationFrames ? envelope*decayStep : .0001);
            frequency=i<durationFrames ? frequency*frequencyStep : end;
            phase=std::fmod(phase+step,1.);
        }
        { QMutexLocker lock(&mutex);
          if(gain<=0 || voices.size()>=64) return;
          Voice v; v.name=name; v.index=nextVoice++; v.position=-delay*clip->rate; v.clip=clip.get(); v.owned=clip; setPan(v,pan);
          voices.append(std::move(v)); }
        startOutput();
    }
    Q_INVOKABLE bool playAvailable(const QString &name,double pitch=1,double amplitude=.18,double pan=0,double cutoff=0,double q=.707) {
        if(!sink || !std::isfinite(pitch+amplitude+pan+cutoff+q)) return false;
        { QMutexLocker lock(&mutex);
          const auto clip=clips.constFind(name);
          if(gain<=0 || clip==clips.cend() || voices.size()>=64) return false;
          const bool feature=name.startsWith("feature-");
          int active=0; bool retrigger=false;
          for(auto &v:voices) {
              if(feature && v.name==name && v.fadeLeft<0) { v.fadeTotal=v.fadeLeft=qRound(format.sampleRate()*.018); retrigger=true; }
              if(v.fadeLeft<0 && (feature ? v.name.startsWith("feature-") : v.name.startsWith("brick"))) ++active;
          }
          if(active>=(feature ? 4 : 8)) return false;
          Voice v; v.name=name; v.index=nextVoice++; v.pitch=std::clamp(pitch,.1,4.); v.clip=&clip.value();
          v.amplitude=std::clamp(amplitude,0.,1.); setPan(v,pan);
          if(retrigger) v.position=-.018*.55*v.clip->rate;
          filter(v,cutoff,q); voices.append(std::move(v)); }
        startOutput(); return true;
    }
    Q_INVOKABLE void stopFeatures(const QString &name=QString()) {
        QMutexLocker lock(&mutex);
        for(auto &v:voices) if(v.name.startsWith("feature-") && (name.isEmpty() || v.name=="feature-"+name)) {
            v.fadeTotal=v.fadeLeft=qRound(format.sampleRate()*.018);
        }
    }
    Q_INVOKABLE void stop() { { QMutexLocker lock(&mutex); voices.clear(); room.clear(); } status.stop(); ++version; emit revisionChanged(); }
signals:
    void directoryChanged(); void volumeChanged(); void revisionChanged();
};
