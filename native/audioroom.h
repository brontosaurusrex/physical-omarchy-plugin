#pragma once
#include <QVariantMap>
#include <QVector>
#include <array>
#include <algorithm>
#include <cmath>

// BiquadFilterNode equations: low/high-pass Q is in dB; band-pass Q is linear.
// https://www.w3.org/TR/webaudio/#filters-characteristics
struct AudioBiquad {
    double b0=1,b1=0,b2=0,a1=0,a2=0,z1[2]={0,0},z2[2]={0,0};
    void configure(const QString &type,double hz,double q,int rate) {
        const double w=2*std::acos(-1.)*std::clamp(hz,1.,rate*.499)/rate, c=std::cos(w);
        const double alpha=std::sin(w)/(2*(type=="bandpass" ? std::max(q,.0001) : std::pow(10.,q/20)));
        const double a0=1+alpha;
        if(type=="highpass") { b0=(1+c)/2/a0; b1=-(1+c)/a0; b2=b0; }
        else if(type=="bandpass") { b0=alpha/a0; b1=0; b2=-b0; }
        else { b0=(1-c)/2/a0; b1=(1-c)/a0; b2=b0; }
        a1=-2*c/a0; a2=(1-alpha)/a0;
    }
    double process(double x,int channel) {
        const double y=b0*x+z1[channel];
        z1[channel]=b1*x-a1*y+z2[channel]; z2[channel]=b2*x-a2*y;
        if(std::abs(z1[channel])<1e-20) z1[channel]=0;
        if(std::abs(z2[channel])<1e-20) z2[channel]=0;
        return y;
    }
    void clear() { z1[0]=z1[1]=z2[0]=z2[1]=0; }
};

// Same routing as the original: dry input plus three filtered reflection taps.
// Only tap 3 feeds its own damped feedback loop, before its tap gain.
class AudioRoom {
    enum Param { Dry,InputHz,InputQ,OutputHz,OutputQ,FeedbackHz,FeedbackQ,
                 Delay1,Delay2,Delay3,Gain1,Gain2,Gain3,Feedback,Count };
    std::array<double,Count> value{1,350,1,350,1,350,1,.01,.02,.04,0,0,0,0},target=value,step{};
    QString inputType="lowpass",outputType="lowpass";
    AudioBiquad inputFilter,outputFilter,feedbackFilter;
    QVector<std::array<double,2>> inputBuffer,feedbackBuffer;
    int rate=0,cursor=0,ramp=0,coefficientTick=0;
    qint64 tail=0;
    void coefficients() {
        inputFilter.configure(inputType,value[InputHz],value[InputQ],rate);
        outputFilter.configure(outputType,value[OutputHz],value[OutputQ],rate);
        feedbackFilter.configure("lowpass",value[FeedbackHz],value[FeedbackQ],rate);
    }
    double read(const QVector<std::array<double,2>> &buffer,double seconds,int channel) const {
        const double delay=std::clamp(seconds*rate,1.,double(buffer.size()-2));
        const int offset=int(delay);
        int first=cursor-offset; if(first<0) first+=buffer.size();
        int second=first-1; if(second<0) second+=buffer.size();
        return buffer[first][channel]+(buffer[second][channel]-buffer[first][channel])*(delay-offset);
    }
public:
    void configure(const QVariantMap &settings,int sampleRate) {
        if(sampleRate<=0) return;
        if(rate!=sampleRate) {
            rate=sampleRate; inputBuffer.resize(rate*2+2); feedbackBuffer.resize(rate*2+2); clear();
        }
        const char *keys[Count]={"dry","inputHz","inputQ","outputHz","outputQ","feedbackHz","feedbackQ",
                                "delay1","delay2","delay3","gain1","gain2","gain3","feedback"};
        auto next=target;
        for(int i=0;i<Count;++i) {
            if(!settings.contains(keys[i])) return;
            next[i]=settings.value(keys[i]).toDouble();
            if(!std::isfinite(next[i])) return;
        }
        next[Dry]=std::clamp(next[Dry],0.,1.);
        next[Feedback]=std::clamp(next[Feedback],0.,.75);
        for(int i=Delay1;i<=Delay3;++i) next[i]=std::clamp(next[i],.001,2.);
        const auto nextInput=settings.value("inputType").toString(),nextOutput=settings.value("outputType").toString();
        if(next==target && inputType==nextInput && outputType==nextOutput) return;
        inputType=nextInput; outputType=nextOutput; target=next;
        ramp=std::max(1,qRound(sampleRate*.16));
        for(int i=0;i<Count;++i) step[i]=(target[i]-value[i])/ramp;
        coefficients(); coefficientTick=0;
    }
    void clear() {
        std::fill(inputBuffer.begin(),inputBuffer.end(),std::array<double,2>{0,0});
        std::fill(feedbackBuffer.begin(),feedbackBuffer.end(),std::array<double,2>{0,0});
        inputFilter.clear(); outputFilter.clear(); feedbackFilter.clear(); cursor=0; tail=0;
    }
    void process(double stereo[2]) {
        if(!rate) return;
        if(ramp>0) {
            for(int i=0;i<Count;++i) value[i]+=step[i];
            if(--ramp==0) value=target;
            // Frequency/Q ramps use 32-sample coefficient steps; delay and gain
            // ramps are sample-accurate. No trig/exponentiation in steady state.
            if(++coefficientTick==32 || !ramp) { coefficients(); coefficientTick=0; }
        }
        if(value[Dry]==1 && target[Dry]==1) { if(tail) clear(); return; }
        if(stereo[0]!=0 || stereo[1]!=0) tail=qint64(rate)*12;
        else if(tail==0) return;
        else if(--tail==0) { clear(); return; }
        for(int channel=0;channel<2;++channel) {
            const double input=inputFilter.process(stereo[channel],channel);
            double taps[3];
            for(int tap=0;tap<2;++tap) taps[tap]=read(inputBuffer,value[Delay1+tap],channel);
            taps[2]=read(feedbackBuffer,value[Delay3],channel);
            inputBuffer[cursor][channel]=input;
            feedbackBuffer[cursor][channel]=input+feedbackFilter.process(taps[2],channel)*value[Feedback];
            const double wet=outputFilter.process(taps[0]*value[Gain1]+taps[1]*value[Gain2]+taps[2]*value[Gain3],channel);
            stereo[channel]=stereo[channel]*value[Dry]+wet*(1-value[Dry]);
        }
        if(++cursor==inputBuffer.size()) cursor=0;
    }
};
