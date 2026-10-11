#pragma once
#include <QAudioSink>
#include <QFile>
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

// One persistent output stream mixes all collision voices. Rapid hits never
// create, stop or restart backend audio streams.
class AudioMixer : public QObject {
    Q_OBJECT
    Q_PROPERTY(QUrl directory READ directory WRITE setDirectory NOTIFY directoryChanged)
    Q_PROPERTY(qreal volume READ volume WRITE setVolume NOTIFY volumeChanged)
    Q_PROPERTY(quint64 revision READ revision NOTIFY revisionChanged)
    struct Clip { QVector<float> samples; int rate=44100; };
    struct Voice { QString name; int index; double position=0; };
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
    QAudioSink *sink=nullptr;
    Output *output=nullptr;
    QTimer status;
    bool started=false;
    qint64 mix(char *data,qint64 length) {
        QMutexLocker lock(&mutex);
        const int bytes=format.bytesPerSample(), channels=format.channelCount();
        if (!bytes || !channels) return 0;
        const qint64 frames=length/(bytes*channels);
        for (qint64 frame=0;frame<frames;++frame) {
            float sample=0;
            for (auto &voice:voices) {
                const auto &clip=clips[voice.name];
                const int index=int(voice.position);
                if (index<clip.samples.size()) {
                    const float fraction=float(voice.position-index);
                    const float a=clip.samples[index],b=clip.samples[qMin(index+1,clip.samples.size()-1)];
                    sample+=(a+(b-a)*fraction)*float(gain)*.18f;
                    voice.position+=double(clip.rate)/format.sampleRate();
                }
            }
            sample=std::clamp(sample,-1.f,1.f);
            for (int channel=0;channel<channels;++channel) {
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
        voices.erase(std::remove_if(voices.begin(),voices.end(),[this](const Voice &v){ return v.position>=clips[v.name].samples.size(); }),voices.end());
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
        const QStringList names={"wall","paddle","brick0","brick1","brick2","brick3","magic0","magic1","magic2","magic3","tone120","tone240","tone360","tone480","tone720","tone960","tone1440"};
        for(const auto &name:names) {
            QFile file(folder.resolved(QUrl(name+".wav")).toLocalFile());
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
            if(encoding!=1 || channels!=1 || bits!=16 || clip.rate<=0 || pcm.isEmpty()) continue;
            clip.samples.reserve(pcm.size()/2);
            for(qsizetype i=0;i+1<pcm.size();i+=2) clip.samples.append(qFromLittleEndian<qint16>(pcm.constData()+i)/32768.f);
            loaded.insert(name,std::move(clip));
        }
        { QMutexLocker lock(&mutex); clips=std::move(loaded); }
        emit directoryChanged(); ++version; emit revisionChanged();
    }
    Q_INVOKABLE bool ready(const QString &name) { QMutexLocker lock(&mutex); return clips.contains(name); }
    Q_INVOKABLE bool playing(const QString &name,int index) {
        QMutexLocker lock(&mutex);
        for(const auto &v:voices) if(v.name==name && v.index==index) return true;
        return false;
    }
    Q_INVOKABLE void play(const QString &name,int index) {
        if(!sink) return;
        { QMutexLocker lock(&mutex);
          if(gain<=0 || !clips.contains(name)) return;
          for(const auto &v:voices) if(v.name==name && v.index==index) return;
          if(voices.size()>=34) return;
          voices.append({name,index,0}); }
        if(!started) { sink->start(output); started=true; }
        status.start(); ++version; emit revisionChanged();
    }
    Q_INVOKABLE void stop() { { QMutexLocker lock(&mutex); voices.clear(); } status.stop(); ++version; emit revisionChanged(); }
signals:
    void directoryChanged(); void volumeChanged(); void revisionChanged();
};
