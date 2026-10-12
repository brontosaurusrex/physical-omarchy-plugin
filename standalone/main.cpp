#include <QGuiApplication>
#include <QQmlApplicationEngine>
#include <QQuickWindow>
#include <QSGRendererInterface>
#include <QCommandLineParser>
#include <QDir>
#include <QFileInfo>
#include <QIcon>
#include <QStandardPaths>
#include <cstdio>

int main(int argc, char **argv) {
    QGuiApplication::setApplicationName("Physical");
    QGuiApplication::setOrganizationName("brontosaurusrex");
    QGuiApplication::setDesktopFileName("io.github.brontosaurusrex.Physical");
    QQuickWindow::setGraphicsApi(QSGRendererInterface::OpenGL);
    QGuiApplication app(argc, argv);
    if (!qEnvironmentVariableIsSet("PHYSICAL_STATE_PATH")) {
        const QString state = QStandardPaths::writableLocation(QStandardPaths::GenericDataLocation) + "/physical/progress.json";
        qputenv("PHYSICAL_STATE_PATH", state.toUtf8());
    }
    QCommandLineParser parser;
    parser.setApplicationDescription("Physical native Linux brick breaker");
    parser.addHelpOption();
    parser.addOption({"data-dir", "Directory containing the QML and game assets.", "path"});
    parser.process(app);
    const QDir executable(QCoreApplication::applicationDirPath());
    QString directory = parser.value("data-dir");
    if (directory.isEmpty()) {
        for (const QString &candidate : {executable.absoluteFilePath("../share/physical-app"), executable.absoluteFilePath("..")}) {
            if (QFileInfo::exists(candidate + "/Standalone.qml")) { directory = candidate; break; }
        }
    }
    if (!QFileInfo::exists(directory + "/Standalone.qml")) {
        std::fprintf(stderr, "Physical assets not found. Use --data-dir PATH.\n"); return 1;
    }
    app.setWindowIcon(QIcon(directory + "/packaging/physical.svg"));
    QQmlApplicationEngine engine;
    QObject::connect(&engine, &QQmlApplicationEngine::objectCreationFailed, &app, [] { QCoreApplication::exit(1); }, Qt::QueuedConnection);
    engine.load(QUrl::fromLocalFile(QDir(directory).absoluteFilePath("Standalone.qml")));
    return app.exec();
}
