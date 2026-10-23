#include "../native/framerecorder.h"
#include <QTemporaryDir>
#include <cstdio>
#include <cstdlib>

static void require(bool value,const char *message) {
    if(!value){std::fprintf(stderr,"FAIL: %s\n",message);std::exit(1);}
}
static QJsonObject read(const QString &path) {
    QFile file(path); require(file.open(QIODevice::ReadOnly),"capture readable");
    return QJsonDocument::fromJson(file.readAll()).object();
}
static void feed(FrameRecorder &recorder,int seconds,int frames,bool active=true) {
    for(int i=0;i<frames;++i){recorder.clockTick(1./qMax(1,frames));recorder.paintedFrame(1000000);recorder.tickSent();recorder.tickReply();}
    recorder.submitted->fetch_add(frames);
    recorder.sample({{"active_gameplay",active},{"ticks_pending",0},{"mouse_locked",true}},seconds*1000);
}
int main(int argc,char **argv) {
    QCoreApplication app(argc,argv);
    QTemporaryDir temporary;require(temporary.isValid(),"temporary state directory");
    const auto directory=temporary.path();
    const auto previousFlag=qgetenv("PHYSICAL_FRAME_DIAGNOSTICS");
    const auto previousDirectory=qgetenv("PHYSICAL_DIAGNOSTICS_DIR");
    qunsetenv("PHYSICAL_FRAME_DIAGNOSTICS");
    qputenv("PHYSICAL_DIAGNOSTICS_DIR",(directory+"/fresh-install").toUtf8());
    {
        FrameRecorder freshInstall;
        require(!freshInstall.enabled,"fresh installation defaults to diagnostics off");
    }
    QFile optIn(directory+"/enabled");require(optIn.open(QIODevice::WriteOnly),"create explicit opt-in");optIn.close();
    require(FrameRecorder::enabledAt(directory),"explicit marker enables recorder");
    qputenv("PHYSICAL_FRAME_DIAGNOSTICS","0");
    require(!FrameRecorder::enabledAt(directory),"disable override wins over marker");
    qputenv("PHYSICAL_FRAME_DIAGNOSTICS","1");
    require(FrameRecorder::enabledAt(directory+"/fresh-install"),"explicit environment opt-in");
    if(previousFlag.isNull())qunsetenv("PHYSICAL_FRAME_DIAGNOSTICS");else qputenv("PHYSICAL_FRAME_DIAGNOSTICS",previousFlag);
    if(previousDirectory.isNull())qunsetenv("PHYSICAL_DIAGNOSTICS_DIR");else qputenv("PHYSICAL_DIAGNOSTICS_DIR",previousDirectory);
    FrameRecorder recorder(directory,true);
    // Idle, startup, a single bad second and interrupted dips must not trigger.
    feed(recorder,1,0,false);feed(recorder,2,0,false);
    feed(recorder,3,30);feed(recorder,4,30);feed(recorder,5,60);
    feed(recorder,6,30);feed(recorder,7,60);feed(recorder,8,30);feed(recorder,9,60);
    require(!QFile::exists(directory+"/latest.json"),"ignore idle/startup/isolated dips");
    for(int t=10;t<=40;++t)feed(recorder,t,60);
    recorder.tickSent();recorder.tickReply();
    feed(recorder,41,30);feed(recorder,42,30);
    auto triggered=read(directory+"/latest.json");
    require(triggered["reason"]=="low-fps-triggered","two low seconds trigger immediately");
    require(triggered["samples"].toArray().size()==30,"thirty preceding samples retained");
    require(triggered["post_samples_remaining"].toInt()==10,"post-trigger capture armed");
    auto last=triggered["samples"].toArray().last().toObject();
    require(last["painted_fps"].toDouble()==30&&last["submitted_fps"].toDouble()==30,"paint/submission counters sampled");
    require(last["clock_fps"].toDouble()==30&&last["clock_gaps_over_25ms"].toInt()==30,"clock gaps sampled");
    require(last["paint_cpu_mean_ms"].toDouble()==1,"paint CPU duration sampled");
    const auto file=triggered["capture_file"].toString();
    for(int t=43;t<=52;++t)feed(recorder,t,60);
    auto completed=read(directory+"/latest.json");
    require(completed["reason"]=="low-fps-complete"&&completed["post_samples_remaining"].toInt()==0,"ten post-trigger seconds saved");
    require(completed["samples"].toArray().size()==40,"capture bounded to forty samples");
    for(int t=53;t<=100;++t)feed(recorder,t,30);
    require(read(directory+"/latest.json")["capture_file"]==file,"incident cooldown prevents repeated captures");
    feed(recorder,101,30);feed(recorder,102,30);
    require(read(directory+"/latest.json")["capture_file"]!=file,"new incident after cooldown");
    recorder.finish("window-closed");
    require(read(directory+"/latest.json")["reason"]=="window-closed","close flushes incomplete post-trigger capture");
    require(read(directory+"/last-session.json")["samples"].toArray().size()==30,"close saves bounded session history");
    for(int t=103;t<=1000;++t)feed(recorder,t,30);
    require(QDir(directory).entryList({"capture-*.json"},QDir::Files).size()==8,"incident retention bounded to eight");
    FrameRecorder inputMasked(directory+"/input-masked",true);
    for(int t=1;t<=8;++t) {
        // Input paints at 60 FPS while simulation ticks only arrive at 30 Hz.
        for(int i=0;i<60;++i)inputMasked.paintedFrame(1000000);
        for(int i=0;i<30;++i){inputMasked.clockTick(1./30);inputMasked.tickSent();inputMasked.tickReply();}
        inputMasked.sample({{"active_gameplay",true},{"engine_fps",30}},t*1000);
    }
    const auto masked=read(directory+"/input-masked/latest.json")["samples"].toArray().last().toObject();
    require(masked["painted_fps"].toDouble()==60&&masked["low_fps_sources"].toArray().contains("tick_replies_per_second"),"input repaints cannot conceal half-rate simulation");
    FrameRecorder disabled(directory+"/disabled",false);
    FrameRecorder transport(directory+"/transport",true);
    const auto now=frameDiagnosticNowUs();
    transport.tickSent();
    transport.tickReply({{"qt_sent_us",now-10000},{"engine_read_us",now-7000},{"engine_ready_us",now-2000}},now-1000);
    feed(transport,1,60,false);transport.finish("test");
    const auto stages=read(directory+"/transport/last-session.json")["samples"].toArray().last().toObject();
    require(stages["command_wait_mean_ms"].toDouble()==3&&stages["engine_work_mean_ms"].toDouble()==5&&stages["reply_wait_mean_ms"].toDouble()==1,"timestamps separate queueing, engine work and reply delivery");
    for(int t=1;t<=20;++t)feed(disabled,t,30);
    disabled.finish("closed");
    require(!QDir(directory+"/disabled").exists(),"disabled recorder performs no file writes");
    const auto permissions=QFile::permissions(directory+"/latest.json");
    require(!(permissions&(QFile::ReadGroup|QFile::ReadOther|QFile::WriteGroup|QFile::WriteOther)),"captures private to owner");
    std::puts("PASS: frame recorder thresholds, idle/startup guards, timing counters, pre/post history, close flush, cooldown, retention and disabled mode");
}
