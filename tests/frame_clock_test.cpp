#include "../native/renderer.cpp"
#include <QGuiApplication>
#include <QTimer>
#include <cstdio>

int main(int argc,char **argv) {
    QGuiApplication app(argc,argv);
    QQuickWindow window;
    window.setTitle("Physical frame-clock regression");window.resize(640,320);
    GameRenderer renderer;
    renderer.setParentItem(window.contentItem());renderer.setWidth(640);renderer.setHeight(320);
    const qreal refresh=window.screen()?window.screen()->refreshRate():60;
    int ticks=0,replies=0,pending=0,phase=0,delay=1,paintedAtBlock=0;
    bool replyEnabled=true,expectFirst=false;
    qreal firstDelta=0;
    const QString packet=R"({"type":"frame","state":{},"width":640,"height":320,"ops":[["set","fillStyle","#182332"],["fillRect",0,0,640,320]]})";
    auto fail=[&](const char *message){std::fprintf(stderr,"FAIL: %s (phase %d, ticks %d, replies %d)\n",message,phase,ticks,replies);app.exit(1);};
    QObject::connect(&renderer,&GameRenderer::frameTick,&app,[&](qreal dt){
        ++ticks;
        if(expectFirst){firstDelta=dt;expectFirst=false;}
        if(pending>=2||!replyEnabled)return;
        ++pending;
        QTimer::singleShot(delay,&app,[&]{--pending;++replies;renderer.acceptLine(packet);});
    });
    QTimer sample;
    QObject::connect(&sample,&QTimer::timeout,&app,[&]{
        ++phase;
        std::printf("CLOCK phase %d ticks %d replies %d pending %d\n",phase,ticks,replies,pending);std::fflush(stdout);
        switch(phase) {
        case 1:delay=qRound(1400/refresh);break;
        case 3:
            if(ticks<refresh*.8||replies<refresh*.8){fail("reply delay reduced display/engine cadence");return;}
            replyEnabled=false;break;
        case 4:paintedAtBlock=renderer.paintedFrames();break;
        case 5:
            if(ticks<refresh*.8){fail("frame clock stopped waiting for engine replies");return;}
            if(renderer.paintedFrames()!=paintedAtBlock){fail("clock redraws repainted the unchanged game texture");return;}
            replyEnabled=true;delay=1;break;
        case 6:
            if(ticks<refresh*.8||replies<refresh*.8){fail("clock did not recover after engine replies resumed");return;}
            renderer.setFrameClockRunning(false);break;
        case 7:
            if(ticks!=0){fail("paused frame clock kept ticking");return;}
            expectFirst=true;renderer.setFrameClockRunning(true);break;
        case 8:
            if(expectFirst||firstDelta<=0||firstDelta>2/refresh||ticks<refresh*.8){fail("resume included paused time or lost frame cadence");return;}
            renderer.setFrameClockRunning(false);
            std::puts("PASS: asynchronous replies cannot throttle the display clock; cached textures, bounded requests, pause and resume preserved");
            app.quit();return;
        }
        ticks=replies=0;
    });
    window.show();renderer.setFrameClockRunning(true);sample.start(1000);
    QTimer::singleShot(12000,&app,[&]{fail("test timeout");});
    return app.exec();
}
