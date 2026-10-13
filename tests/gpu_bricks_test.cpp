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
        QTimer::singleShot(500,&app,[&,packet] {
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
            if(missing) { app.exit(1); return; }
            // Compare the same moving particles through both transports on the
            // actual GPU. Their shape, colors, layering and vertices must match.
            QJsonObject reference=packet;
            reference["ops"]=QJsonArray{QJsonArray{"set","fillStyle","#000000"},QJsonArray{"fillRect",0,0,1800,900},
                QJsonArray{"debris",1,100.25,100.5,1,0,"#ff4400",30,true,0,QJsonArray{}},QJsonArray{"cache",1},
                QJsonArray{"debris",2,170,130,.8,.6,"rgb(80,190,240)",28,false,.4,QJsonArray{-30,-20,20,-30,35,15,-20,30}}};
            renderer.acceptLine(QString::fromUtf8(QJsonDocument(reference).toJson(QJsonDocument::Compact)));
            QTimer::singleShot(150,&app,[&,reference] {
                const auto original=window.grabWindow();
                QJsonObject batch=reference;
                batch["debrisShapes"]=QJsonObject{{"1",QJsonObject{{"radius",30},{"circle",true},{"roundness",0},{"points",QJsonArray{}}}},
                    {"2",QJsonObject{{"radius",28},{"circle",false},{"roundness",.4},{"points",QJsonArray{-30,-20,20,-30,35,15,-20,30}}}}};
                batch["ops"]=QJsonArray{QJsonArray{"set","fillStyle","#000000"},QJsonArray{"fillRect",0,0,1800,900},
                    QJsonArray{"debris-batch",QJsonObject{{"instances",QJsonArray{1,100.25,100.5,1,0}},{"colors",QJsonArray{"#ff4400"}}}},QJsonArray{"cache",1},
                    QJsonArray{"debris-batch",QJsonObject{{"instances",QJsonArray{2,170,130,.8,.6}},{"colors",QJsonArray{"rgb(80,190,240)"}}}}};
                renderer.acceptLine(QString::fromUtf8(QJsonDocument(batch).toJson(QJsonDocument::Compact)));
                QTimer::singleShot(150,&app,[&,original] {
                    const auto updated=window.grabWindow();
                    if(original.isNull() || original!=updated || !renderer.renderError().isEmpty()) {
                        std::fprintf(stderr,"FAIL: GPU particle transport changed the image\n"); app.exit(1); return;
                    }
                    std::puts("GPU particle check: original and batched images identical"); app.exit(0);
                });
            });
        });
    });
    QTimer::singleShot(5000,&app,[&] { std::fprintf(stderr,"FAIL: GPU test timed out\n"); app.exit(1); });
    return app.exec();
}
