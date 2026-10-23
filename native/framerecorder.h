#pragma once
#include <QCoreApplication>
#include <QDateTime>
#include <QDir>
#include <QElapsedTimer>
#include <QFile>
#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QSaveFile>
#include <atomic>
#include <deque>
#include <memory>
#include <chrono>

inline qint64 frameDiagnosticNowUs() {
    return std::chrono::duration_cast<std::chrono::microseconds>(std::chrono::system_clock::now().time_since_epoch()).count();
}

// GUI-thread sampling; the render thread touches only the atomic counters.
// No timers, frame requests, graphics queries or per-frame file writes.
class FrameRecorder {
    QString directory, captureName;
    QJsonArray history, capture;
    QElapsedTimer clock;
    qint64 lastSample=0, activeMs=0, lowMs=0, cooldownUntil=0, triggerTime=0;
    int postSamples=0;
    quint64 clockFrames=0, longFrames=0, replies=0, latencyCount=0;
    double maxClockMs=0, latencySumMs=0, latencyMaxMs=0;
    quint64 acceptedFrames=0;
    double acceptSumMs=0, acceptMaxMs=0;
    std::deque<qint64> requests;
    QJsonObject metadata;
    struct Timing {quint64 count=0;double total=0,maximum=0;};
    QMap<QString,Timing> stages;
    quint64 invalidTimestamps=0;

    bool write(const QString &name,const QJsonObject &object) {
        if(!QDir().mkpath(directory))return false;
        QFile::setPermissions(directory,QFile::ReadOwner|QFile::WriteOwner|QFile::ExeOwner);
        QSaveFile file(directory+"/"+name);
        if(!file.open(QIODevice::WriteOnly))return false;
        file.setPermissions(QFile::ReadOwner|QFile::WriteOwner);
        const auto bytes=QJsonDocument(object).toJson();
        if(file.write(bytes)!=bytes.size())return false;
        return file.commit();
    }
    QJsonObject document(const QString &reason,const QJsonArray &samples) const {
        auto object=metadata;
        object["reason"]=reason;
        object["samples"]=samples;
        object["saved_at_utc"]=QDateTime::currentDateTimeUtc().toString(Qt::ISODateWithMs);
        return object;
    }
    void saveCapture(const QString &reason) {
        auto object=document(reason,capture);
        object["capture_file"]=captureName;
        object["trigger_elapsed_ms"]=triggerTime;
        object["post_samples_remaining"]=postSamples;
        if(!write(captureName,object)||!write("latest.json",object)) {
            qWarning("Physical frame recorder: could not write capture");
            return;
        }
        // Retain eight incidents, plus the convenient latest/session files.
        const auto files=QDir(directory).entryList({"capture-*.json"},QDir::Files,QDir::Name);
        for(int i=0;i<files.size()-8;++i)QFile::remove(directory+"/"+files[i]);
    }
public:
    const bool enabled;
    std::atomic<quint64> painted{0}, paintNs{0}, maxPaintNs{0};
    std::shared_ptr<std::atomic<quint64>> submitted=std::make_shared<std::atomic<quint64>>(0);
    static QString defaultDirectory() {
        return qEnvironmentVariable("PHYSICAL_DIAGNOSTICS_DIR",
            qEnvironmentVariable("XDG_STATE_HOME",QDir::homePath()+"/.local/state")+"/physical/diagnostics");
    }
    static bool enabledAt(const QString &path) {
        const auto value=qgetenv("PHYSICAL_FRAME_DIAGNOSTICS");
        return value.isEmpty() ? QFile::exists(path+"/enabled") : value=="1";
    }
    FrameRecorder():FrameRecorder(defaultDirectory(),enabledAt(defaultDirectory())) {}
    FrameRecorder(const QString &path,bool active):directory(path),enabled(active) {
        clock.start();
        metadata={{"schema_version",2},{"pid",QCoreApplication::applicationPid()},
            {"qt_version",qVersion()},{"session_started_utc",QDateTime::currentDateTimeUtc().toString(Qt::ISODateWithMs)},
            {"threshold_fps",40},{"trigger_duration_ms",2000},
            {"trigger_sources",QJsonArray{"painted_fps","clock_fps","tick_replies_per_second","engine_fps"}},
            {"submission_note","Qt frameSwapped counts submissions, not physical monitor presentation"},
            {"paint_note","paint_cpu_ms measures the main item's paint call, not GPU execution or all child layers"}};
        metadata["transport_clock_note"]="Unix microseconds across processes; negative intervals rejected. Clock adjustments can affect stage estimates. Output wait includes the small timing trailer, pipe transfer and Qt delivery.";
    }
    ~FrameRecorder() { if(enabled&&postSamples)finish("recorder-destroyed"); }
    QString path()const{return directory;}
    void begin() {
        if(!enabled)return;
        lastSample=clock.elapsed(); activeMs=lowMs=0;
        painted=paintNs=maxPaintNs=0; submitted->store(0);
        clockFrames=longFrames=replies=latencyCount=0;
        maxClockMs=latencySumMs=latencyMaxMs=0;
        acceptedFrames=0;acceptSumMs=acceptMaxMs=0;
        metadata["window_opened_utc"]=QDateTime::currentDateTimeUtc().toString(Qt::ISODateWithMs);
        stages.clear();invalidTimestamps=0;
    }
    void clockTick(double seconds) {
        if(!enabled)return;
        ++clockFrames; if(seconds>.025)++longFrames;
        maxClockMs=qMax(maxClockMs,seconds*1000);
    }
    void tickSent() { if(enabled&&requests.size()<8)requests.push_back(clock.nsecsElapsed()); }
    void tickReply(const QJsonObject &timing={},qint64 receivedUs=0) {
        if(!enabled)return;
        ++replies;
        if(!timing.isEmpty()&&receivedUs>0) {
            const qint64 sent=timing["qt_sent_us"].toInteger(),read=timing["engine_read_us"].toInteger(),ready=timing["engine_ready_us"].toInteger();
            const qint64 accepted=frameDiagnosticNowUs();
            if(sent>0&&sent<=read&&read<=ready&&ready<=receivedUs&&receivedUs<=accepted) {
                for(const auto &pair:std::initializer_list<std::pair<QString,double>>{
                    {"command_wait",(read-sent)/1000.},{"engine_work",(ready-read)/1000.},
                    {"reply_wait",(receivedUs-ready)/1000.},{"qt_receive_to_accept",(accepted-receivedUs)/1000.}}) {
                    auto &stage=stages[pair.first];++stage.count;stage.total+=pair.second;stage.maximum=qMax(stage.maximum,pair.second);
                }
            } else ++invalidTimestamps;
        }
        if(requests.empty())return;
        const double ms=(clock.nsecsElapsed()-requests.front())/1e6;
        requests.pop_front(); ++latencyCount; latencySumMs+=ms; latencyMaxMs=qMax(latencyMaxMs,ms);
    }
    void resetRequests(){requests.clear();}
    void acceptedFrame(double ms) {
        if(!enabled)return;
        ++acceptedFrames;acceptSumMs+=ms;acceptMaxMs=qMax(acceptMaxMs,ms);
    }
    void paintedFrame(quint64 ns) {
        if(!enabled)return;
        painted.fetch_add(1,std::memory_order_relaxed);
        paintNs.fetch_add(ns,std::memory_order_relaxed);
        // Only the render thread writes this maximum between GUI samples.
        auto value=maxPaintNs.load(std::memory_order_relaxed);
        while(value<ns&&!maxPaintNs.compare_exchange_weak(value,ns,std::memory_order_relaxed)){}
    }
    // Explicit timestamps also make threshold/retention tests deterministic.
    void sample(QJsonObject state,qint64 now=-1) {
        if(!enabled)return;
        if(now<0)now=clock.elapsed();
        const auto elapsed=now-lastSample; lastSample=now;
        if(elapsed<=0)return;
        const auto frames=painted.exchange(0), swaps=submitted->exchange(0);
        const auto ns=paintNs.exchange(0), maximum=maxPaintNs.exchange(0);
        const double seconds=elapsed/1000.;
        state["elapsed_ms"]=now; state["interval_ms"]=elapsed;
        state["utc"]=QDateTime::currentDateTimeUtc().toString(Qt::ISODateWithMs);
        state["painted_fps"]=frames/seconds; state["submitted_fps"]=swaps/seconds;
        state["clock_fps"]=clockFrames/seconds; state["clock_gaps_over_25ms"]=qint64(longFrames);
        state["clock_max_gap_ms"]=maxClockMs;
        state["tick_replies_per_second"]=replies/seconds;
        state["tick_latency_mean_ms"]=latencyCount ? latencySumMs/latencyCount : 0;
        state["tick_latency_max_ms"]=latencyMaxMs;
        state["oldest_request_age_ms"]=requests.empty()?0:(clock.nsecsElapsed()-requests.front())/1e6;
        state["paint_cpu_mean_ms"]=frames ? ns/1e6/frames : 0;
        state["paint_cpu_max_ms"]=maximum/1e6;
        state["accept_frame_mean_ms"]=acceptedFrames ? acceptSumMs/acceptedFrames : 0;
        state["accept_frame_max_ms"]=acceptMaxMs;
        for(auto it=stages.cbegin();it!=stages.cend();++it) {
            state[it.key()+"_mean_ms"]=it->count?it->total/it->count:0;
            state[it.key()+"_max_ms"]=it->maximum;
        }
        state["transport_timed_ticks"]=qint64(stages.value("engine_work").count);
        state["transport_invalid_timestamps"]=qint64(invalidTimestamps);
        stages.clear();invalidTimestamps=0;
        QJsonArray slowSources;
        for(const auto *name:{"painted_fps","clock_fps","tick_replies_per_second"})
            if(state[name].toDouble()<40)slowSources.append(name);
        const double engineFps=state["engine_fps"].toDouble();
        if(engineFps>0&&engineFps<40)slowSources.append("engine_fps");
        state["low_fps_sources"]=slowSources;
        clockFrames=longFrames=replies=latencyCount=0;
        maxClockMs=latencySumMs=latencyMaxMs=0;
        acceptedFrames=0;acceptSumMs=acceptMaxMs=0;
        history.append(state);
        while(history.size()>30)history.removeAt(0);
        if(postSamples) {
            capture.append(state);
            if(!--postSamples)saveCapture("low-fps-complete");
        }
        if(!state["active_gameplay"].toBool()) {activeMs=lowMs=0;return;}
        activeMs+=elapsed;
        // Ignore startup/resume and the first partially active sampling interval.
        if(activeMs<3000){lowMs=0;return;}
        lowMs=!slowSources.isEmpty() ? lowMs+elapsed : 0;
        if(lowMs>=2000&&!postSamples&&now>=cooldownUntil) {
            capture=history; postSamples=10; cooldownUntil=now+60000;
            captureName="capture-"+QDateTime::currentDateTimeUtc().toString("yyyyMMdd-HHmmss-zzz")+"-"+QString::number(QCoreApplication::applicationPid())+"-"+QString::number(now)+".json";
            triggerTime=now;
            saveCapture("low-fps-triggered");
            qInfo("Physical frame recorder captured a slowdown: %s",qPrintable(directory+"/latest.json"));
        }
    }
    void finish(const QString &reason) {
        if(!enabled)return;
        if(postSamples) { saveCapture(reason); postSamples=0; }
        if(!history.isEmpty()&&!write("last-session.json",document(reason,history)))
            qWarning("Physical frame recorder: could not save session");
        history=QJsonArray(); activeMs=lowMs=0;
    }
};
