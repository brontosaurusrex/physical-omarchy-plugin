#include "../native/renderer.cpp"
#include <QGuiApplication>
#include <QThread>
#include <QTimer>
#include <algorithm>
#include <cstdio>

// Compare the game's elapsed time with independently sampled window
// submissions while the GUI thread is intermittently busy.
int main(int argc, char **argv) {
    QGuiApplication app(argc, argv);
    QQuickWindow window;
    window.setTitle("Physical submission-clock regression");
    window.resize(640, 320);
    QElapsedTimer clock;
    clock.start();
    QMutex mutex;
    qint64 previous = 0;
    double submittedInterval = 0;
    QObject::connect(&window, &QQuickWindow::frameSwapped, &window, [&] {
        const auto now = clock.nsecsElapsed();
        QMutexLocker lock(&mutex);
        if (previous) submittedInterval = (now - previous) / 1e9;
        previous = now;
    }, Qt::DirectConnection);

    GameRenderer renderer;
    renderer.setParentItem(window.contentItem());
    renderer.setWidth(640);
    renderer.setHeight(320);
    QVector<double> errors;
    int ticks = 0;
    QObject::connect(&renderer, &GameRenderer::frameTick, &app, [&](qreal dt) {
        ++ticks;
        QMutexLocker lock(&mutex);
        if (ticks > 10 && submittedInterval > 0)
            errors.append(std::abs(dt - submittedInterval) * 1000);
    });

    QTimer busy;
    busy.setTimerType(Qt::PreciseTimer);
    busy.setInterval(31);
    QObject::connect(&busy, &QTimer::timeout, &app, [] { QThread::msleep(6); });
    busy.start();
    window.show();
    renderer.setFrameClockRunning(true);
    QTimer::singleShot(2500, &app, [&] {
        renderer.setFrameClockRunning(false);
        std::sort(errors.begin(), errors.end());
        if (errors.size() < 30) {
            std::puts("FAIL: too few submission-clock samples");
            app.exit(1);
            return;
        }
        const double p95 = errors[int(errors.size() * .95)];
        std::printf("samples=%lld p95_clock_error_ms=%.3f\n",
                    static_cast<long long>(errors.size()), p95);
        if (p95 > .5) {
            std::puts("FAIL: GUI delays changed the submitted-frame time step");
            app.exit(1);
            return;
        }
        std::puts("PASS: motion clock follows submitted frames despite GUI delays");
        app.quit();
    });
    const int result = app.exec();
    renderer.setFrameClockRunning(false);
    window.hide();
    return result;
}
