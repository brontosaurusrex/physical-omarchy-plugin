#pragma once
#include <QObject>
#include <QProcess>
#include <QGuiApplication>
#include <QClipboard>
#include <QSignalBlocker>

// Shared process transport for the shell plugin and regular Qt application.
class EngineProcess : public QObject {
    Q_OBJECT
    Q_PROPERTY(QStringList command READ command WRITE setCommand NOTIFY commandChanged)
    Q_PROPERTY(bool running READ running WRITE setRunning NOTIFY runningChanged)
    QProcess process;
    QStringList arguments;
    QByteArray output, errors;
    void readLines(QByteArray &pending, const QByteArray &bytes, bool standardError) {
        pending += bytes;
        if (pending.size() > 4 * 1024 * 1024) {
            emit errorLine("fatal: engine output exceeded the frame limit");
            process.kill(); return;
        }
        int newline;
        while ((newline = pending.indexOf('\n')) >= 0) {
            const auto line = QString::fromUtf8(pending.left(newline));
            pending.remove(0, newline + 1);
            if (standardError) emit errorLine(line); else emit lineReceived(line);
        }
    }
public:
    explicit EngineProcess(QObject *parent = nullptr): QObject(parent) {
        connect(&process, &QProcess::stateChanged, this, [this] { emit runningChanged(); });
        connect(&process, &QProcess::readyReadStandardOutput, this, [this] { readLines(output, process.readAllStandardOutput(), false); });
        connect(&process, &QProcess::readyReadStandardError, this, [this] { readLines(errors, process.readAllStandardError(), true); });
        connect(&process, &QProcess::finished, this, [this](int code, QProcess::ExitStatus) { emit exited(code); });
        connect(&process, &QProcess::errorOccurred, this, [this](QProcess::ProcessError error) {
            if (error == QProcess::FailedToStart) { emit errorLine("fatal: " + process.errorString()); emit exited(-1); }
        });
    }
    ~EngineProcess() override {
        QSignalBlocker blocker(&process);
        if (running()) {
            process.write("{\"action\":\"quit\"}\n");
            process.closeWriteChannel();
            if (!process.waitForFinished(300)) { process.kill(); process.waitForFinished(100); }
        }
    }
    QStringList command() const { return arguments; }
    void setCommand(const QStringList &value) { if (arguments == value) return; arguments = value; emit commandChanged(); }
    bool running() const { return process.state() != QProcess::NotRunning; }
    void setRunning(bool value) {
        if (value == running()) return;
        if (!value) { process.terminate(); return; }
        if (arguments.isEmpty()) { emit errorLine("fatal: no engine command"); emit exited(-1); return; }
        output.clear(); errors.clear();
        process.start(arguments.first(), arguments.mid(1));
    }
    Q_INVOKABLE void write(const QString &line) { if (running()) process.write(line.toUtf8()); }
    Q_INVOKABLE void copyText(const QString &text) { QGuiApplication::clipboard()->setText(text); }
signals:
    void commandChanged();
    void runningChanged();
    void lineReceived(const QString &line);
    void errorLine(const QString &line);
    void exited(int exitCode);
};
