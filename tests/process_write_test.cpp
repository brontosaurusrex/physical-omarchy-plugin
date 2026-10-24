#include "../native/engineprocess.h"
#include <QGuiApplication>
#include <QTimer>
#include <QThread>
#include <algorithm>
#include <cstdio>
int main(int argc, char **argv) {
    QGuiApplication app(argc, argv);
    EngineProcess process, blocked;
    blocked.setCommand({"python3", "-c", "import time;time.sleep(4)"});
    blocked.setRunning(true);
    process.setCommand({"python3", "-u", "-c",
        "import sys,json,time\n"
        "for line in sys.stdin:\n"
        " d=json.loads(line);d['received_us']=time.time_ns()//1000;print(json.dumps(d),flush=True)"});
    QVector<double> waits;
    QObject::connect(&process, &EngineProcess::lineReceived, &app, [&](const QString &line) {
        const auto reply = QJsonDocument::fromJson(line.toUtf8()).object();
        if (reply.contains("sent_us")) {
            waits.append((reply["received_us"].toInteger() - reply["sent_us"].toInteger()) / 1000.);
        }
    });
    process.setRunning(true);
    QTimer frame;
    QObject::connect(&frame, &QTimer::timeout, &app, [&] {
        const auto command = QJsonObject{{"action", "probe"}, {"sent_us", frameDiagnosticNowUs()}};
        process.write(QString::fromUtf8(QJsonDocument(command).toJson(QJsonDocument::Compact)) + "\n");
        // Model the GUI's render synchronization immediately after a tick.
        QThread::msleep(12);
    });
    QTimer::singleShot(200, &app, [&] {
        QElapsedTimer timer;
        timer.start();
        for (int i = 0; i < 64; ++i) blocked.write(QString(4096, 'x'));
        if (timer.elapsed() > 150) {
            std::puts("FAIL: full engine pipe blocked GUI writes");
            app.exit(1);
            return;
        }
        frame.start(16);
    });
    QTimer::singleShot(1500, &app, [&] {
        frame.stop();
        std::sort(waits.begin(), waits.end());
        if (waits.size() < 30) {
            std::puts("FAIL: too few command timing samples");
            app.exit(1);
            return;
        }
        const double median = waits[waits.size() / 2];
        const double p95 = waits[int(waits.size() * .95)];
        std::printf("samples=%lld median_command_wait_ms=%.3f p95=%.3f\n",
                    static_cast<long long>(waits.size()), median, p95);
        if (median > 5) {
            std::puts("FAIL: command waited behind simulated render sync");
            app.exit(1);
            return;
        }
        std::puts("PASS: engine commands reach the pipe before GUI render synchronization");
        app.quit();
    });
    return app.exec();
}
#include "engineprocess.moc"
