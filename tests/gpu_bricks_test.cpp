#include "../native/renderer.cpp"
#include <QGuiApplication>
#include <QTimer>
#include <cstdio>

int main(int argc,char **argv) {
    QGuiApplication app(argc,argv);
    QQuickWindow window; window.resize(900,450); window.setTitle("Physical brick rendering test");
    GameRenderer renderer; renderer.setParentItem(window.contentItem()); renderer.setWidth(900); renderer.setHeight(450);
    auto resize=[&] { renderer.setWidth(window.width()); renderer.setHeight(window.height()); };
    QObject::connect(&window,&QQuickWindow::widthChanged,&app,resize);
    QObject::connect(&window,&QQuickWindow::heightChanged,&app,resize);
    window.show();
    QTimer::singleShot(500,&app,[&] {
        if(window.rendererInterface()->graphicsApi()!=QSGRendererInterface::OpenGL) { std::fprintf(stderr,"FAIL: test requires actual OpenGL rendering\n"); app.exit(1); return; }
        QJsonArray bricks;
        for(int row=0;row<15;++row) for(int col=0;col<44;++col) {
            for(qreal v:{qreal(col*40+20),qreal(row*40+20),34.,34.,0.,qreal((row+col)%3)}) bricks.append(v);
        }
        QJsonArray batch{"bricks",bricks,false,5,"#22aa44","#99ddaa","#aa22cc","#dd99ee","#2255cc","#99aadd"};
        QJsonObject packet{{"type","frame"},{"width",1800},{"height",900},{"state",QJsonObject{}},{"caches",QJsonObject{{"1",QJsonArray{batch}}}},{"ops",QJsonArray{QJsonArray{"set","fillStyle","#000000"},QJsonArray{"fillRect",0,0,1800,900},QJsonArray{"cache",1}}}};
        renderer.acceptLine(QString::fromUtf8(QJsonDocument(packet).toJson(QJsonDocument::Compact)));
        QTimer::singleShot(500,&app,[&] {
            const auto image=window.grabWindow();
            if(image.isNull()) { std::fprintf(stderr,"FAIL: could not capture GPU window\n"); app.exit(1); return; }
            int missing=0;
            for(int row=0;row<15;++row) for(int col=0;col<44;++col) {
                const int x=qRound((col*40+37)*image.width()/1800.);
                const int y=qRound((row*40+37)*image.height()/900.);
                const QColor expected=QColor(QStringList{"#22aa44","#aa22cc","#2255cc"}[(row+col)%3]);
                if(image.pixelColor(x,y)!=expected) ++missing;
            }
            if(qEnvironmentVariableIsSet("PHYSICAL_TEST_ARTIFACTS")) image.save(qEnvironmentVariable("PHYSICAL_TEST_ARTIFACTS")+"/physical-all-bricks.png");
            std::printf("GPU brick check: %d / 660 visible\n",660-missing);
            app.exit(missing ? 1 : 0);
        });
    });
    QTimer::singleShot(5000,&app,[&] { std::fprintf(stderr,"FAIL: GPU test timed out\n"); app.exit(1); });
    return app.exec();
}
