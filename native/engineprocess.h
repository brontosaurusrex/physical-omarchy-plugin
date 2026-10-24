#pragma once
#include <QObject>
#include <QProcess>
#include <QGuiApplication>
#include <QClipboard>
#include <QSignalBlocker>
#include "framerecorder.h"

// Shared process transport for the shell plugin and regular Qt application.
class EngineProcess : public QObject {
    Q_OBJECT
    Q_PROPERTY(QStringList command READ command WRITE setCommand NOTIFY commandChanged)
    Q_PROPERTY(bool running READ running WRITE setRunning NOTIFY runningChanged)
    QProcess process;
    QStringList arguments;
    QByteArray output, errors;
    const bool diagnostics=FrameRecorder::enabledAt(FrameRecorder::defaultDirectory());
    qint64 sequence=0,acknowledged=0,receivedUs=0,peakBytes=0,peakCommands=0,peakMouse=0;
    std::deque<qint64> mouseCommands;
    qint64 mouseOverflow=0;
    void writeCommand(const QByteArray &bytes) {
        process.write(bytes);
        if(diagnostics)peakBytes=qMax(peakBytes,process.bytesToWrite());
        // QProcess otherwise leaves this to a writable-socket event. A tick
        // sent just before Qt's render synchronization would wait behind it.
        // Flush ready pipe data now, without waiting if the engine is busy.
        if(process.bytesToWrite()>0)process.waitForBytesWritten(0);
    }
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
            if (standardError) emit errorLine(line); else {if(diagnostics)receivedUs=frameDiagnosticNowUs();emit lineReceived(line);}
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
        sequence=acknowledged=peakBytes=peakCommands=peakMouse=mouseOverflow=0;mouseCommands.clear();
        process.start(arguments.first(), arguments.mid(1));
    }
    Q_INVOKABLE void write(const QString &line) {
        if(!running())return;
        if(!diagnostics){writeCommand(line.toUtf8());return;}
        auto command=QJsonDocument::fromJson(line.toUtf8()).object();
        command["_diag_seq"]=++sequence;command["_diag_sent_us"]=frameDiagnosticNowUs();
        const auto action=command["action"].toString();
        if(action=="relative-pointer"||action=="pointer") {
            if(mouseCommands.size()<4096)mouseCommands.push_back(sequence);else ++mouseOverflow;
        }
        writeCommand(QJsonDocument(command).toJson(QJsonDocument::Compact)+'\n');
        peakCommands=qMax(peakCommands,sequence-acknowledged);
        peakMouse=qMax(peakMouse,qint64(mouseCommands.size()));
    }
    Q_INVOKABLE qint64 diagnosticReceivedUs()const{return receivedUs;}
    Q_INVOKABLE void diagnosticAcknowledge(qint64 value) {
        if(value<acknowledged||value>sequence)return;
        acknowledged=value;
        while(!mouseCommands.empty()&&mouseCommands.front()<=value)mouseCommands.pop_front();
    }
    Q_INVOKABLE QVariantMap diagnosticTransportState() {
        QVariantMap state{{"commands_unacknowledged",sequence-acknowledged},{"commands_peak_unacknowledged",peakCommands},
            {"mouse_commands_unacknowledged",qint64(mouseCommands.size())},{"mouse_commands_peak_unacknowledged",peakMouse},
            {"mouse_counter_overflow",mouseOverflow},{"qt_write_pending_bytes",process.bytesToWrite()},{"qt_write_peak_pending_bytes",peakBytes}};
        peakBytes=process.bytesToWrite();peakCommands=sequence-acknowledged;peakMouse=mouseCommands.size();mouseOverflow=0;
        return state;
    }
    Q_INVOKABLE void copyText(const QString &text) { QGuiApplication::clipboard()->setText(text); }
signals:
    void commandChanged();
    void runningChanged();
    void lineReceived(const QString &line);
    void errorLine(const QString &line);
    void exited(int exitCode);
};
