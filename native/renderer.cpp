#include <QQuickPaintedItem>
#include <QQuickWindow>
#include <QScreen>
#include <QQmlExtensionPlugin>
#include <QPainter>
#include <QPainterPath>
#include <QPicture>
#include <QJsonDocument>
#include <QJsonArray>
#include <QJsonObject>
#include <QRegularExpression>
#include <QFontMetricsF>
#include <QSGGeometryNode>
#include <QSGVertexColorMaterial>
#include <QSGRendererInterface>
#include <QMutex>
#include <QElapsedTimer>
#include <QMutexLocker>
#include <QSet>
#include <atomic>
#include <cstring>
#include <cmath>
#include <memory>
#include "pointerlock.h"
#include "audiomixer.h"
#include "engineprocess.h"
#include "framerecorder.h"

struct Vertex { float x,y; unsigned char r,g,b,a; };

class DebrisLayer : public QQuickItem {
    QVector<Vertex> vertices;
    qreal gw=1800,gh=900;
public:
    DebrisLayer(QQuickItem *parent):QQuickItem(parent) { setFlag(ItemHasContents,true); }
    void setFrame(const QVector<Vertex> &next,qreal w,qreal h) { vertices=next; gw=w; gh=h; update(); }
protected:
    QSGNode *updatePaintNode(QSGNode *old,UpdatePaintNodeData *) override {
        // Qt's batch renderer can use 16-bit indices even for unindexed input.
        // Keep triangle chunks below that range so large brick fields render whole.
        constexpr int chunkSize=61440; // Divisible by three.
        if(vertices.isEmpty()) { delete old; return nullptr; }
        auto root=old ? old : new QSGNode;
        auto child=root->firstChild();
        for(int start=0;start<vertices.size();start+=chunkSize) {
            const int count=qMin(chunkSize,int(vertices.size())-start);
            QSGGeometryNode *node;
            if(child) {
                node=static_cast<QSGGeometryNode *>(child);
                child=child->nextSibling();
                node->geometry()->allocate(count);
            } else {
                node=new QSGGeometryNode;
                auto geometry=new QSGGeometry(QSGGeometry::defaultAttributes_ColoredPoint2D(),count);
                geometry->setDrawingMode(QSGGeometry::DrawTriangles);
                node->setGeometry(geometry); node->setFlag(QSGNode::OwnsGeometry);
                node->setMaterial(new QSGVertexColorMaterial); node->setFlag(QSGNode::OwnsMaterial);
                root->appendChildNode(node);
            }
            auto output=node->geometry()->vertexDataAsColoredPoint2D();
            for(int i=0;i<count;++i) {
                const auto &v=vertices[start+i]; output[i].set(v.x*width()/gw,v.y*height()/gh,v.r,v.g,v.b,v.a);
            }
            node->markDirty(QSGNode::DirtyGeometry);
        }
        while(child) {
            auto next=child->nextSibling(); root->removeChildNode(child); delete child; child=next;
        }
        return root;
    }
    void geometryChange(const QRectF &next,const QRectF &previous) override { QQuickItem::geometryChange(next,previous); update(); }
};

class PictureLayer : public QQuickPaintedItem {
    QPicture picture;
    qreal gw=1800,gh=900;
    std::atomic<qint64> paintNs{0};
    std::atomic<int> paints{0};
    const bool profile=qEnvironmentVariableIsSet("PHYSICAL_RENDER_TIMINGS");
public:
    qreal paintMs() const { return paintNs.load()/1e6; }
    int paintCount() const { return paints.load(); }
    PictureLayer(QQuickItem *parent):QQuickPaintedItem(parent) { setAntialiasing(true); setRenderTarget(FramebufferObject); }
    void setFrame(const QPicture &next,qreal w,qreal h) {
        // Particles live in separate GPU layers. Retain the text texture when
        // its complete recorded content is identical, especially on results.
        if(gw==w && gh==h && picture.size()==next.size() &&
           (!next.size() || std::memcmp(picture.data(),next.data(),next.size())==0)) return;
        picture=next; gw=w; gh=h; update();
    }
    void paint(QPainter *painter) override { QElapsedTimer timer; if(profile) timer.start(); painter->scale(width()/gw,height()/gh); painter->drawPicture(0,0,picture); paints.fetch_add(1); if(profile) paintNs.store(timer.nsecsElapsed()); }
};

// Quadratic rounded contours match the original canvas corners. Inner fill,
// stroke and a one-physical-pixel transparent fringe are separate GPU rings.
static void appendBrickMesh(QVector<Vertex> &out,qreal x,qreal y,qreal w,qreal h,
                            qreal angle,qreal radius,QColor fill,QColor stroke,qreal pen,qreal fringe) {
    const int segments=qBound(3,int(std::ceil(std::sqrt(radius/qMax(fringe,.001)))),12);
    auto contour=[&](qreal inset) {
        QVector<QPointF> points;
        const qreal l=inset,t=inset,r=w-inset,b=h-inset;
        const qreal corner=qBound(0.,radius-inset,qMax(0.,qMin((r-l)/2,(b-t)/2)));
        QPointF current(l+corner,t); points.append(current);
        auto line=[&](QPointF next) { points.append(next); current=next; };
        auto quad=[&](QPointF control,QPointF end) {
            const QPointF start=current;
            for(int i=1;i<=segments;++i) { qreal f=qreal(i)/segments; points.append(start*((1-f)*(1-f))+control*(2*f*(1-f))+end*(f*f)); }
            current=end;
        };
        line({r-corner,t}); quad({r,t},{r,t+corner});
        line({r,b-corner}); quad({r,b},{r-corner,b});
        line({l+corner,b}); quad({l,b},{l,b-corner});
        line({l,t+corner}); quad({l,t},{l+corner,t});
        points.removeLast(); return points;
    };
    fringe=qMin(fringe,pen*.8);
    const auto core=contour(pen/2+fringe/2),inner=contour(pen/2-fringe/2),outer=contour(-pen/2+fringe/2),edge=contour(-pen/2-fringe/2);
    const qreal c=std::cos(angle),s=std::sin(angle);
    auto vertex=[&](QPointF p,QColor color,qreal alpha=1) {
        p-=QPointF(w/2,h/2);
        const qreal coverage=color.alphaF()*alpha;
        return Vertex{float(x+w/2+p.x()*c-p.y()*s),float(y+h/2+p.x()*s+p.y()*c),
            static_cast<unsigned char>(qRound(color.red()*coverage)),static_cast<unsigned char>(qRound(color.green()*coverage)),
            static_cast<unsigned char>(qRound(color.blue()*coverage)),static_cast<unsigned char>(qRound(255*coverage))};
    };
    auto triangle=[&](Vertex a,Vertex b,Vertex c) { out.append(a);out.append(b);out.append(c); };
    const auto center=vertex({w/2,h/2},fill);
    for(int i=0;i<inner.size();++i) {
        const int j=(i+1)%inner.size();
        const auto fi=vertex(core[i],fill),fj=vertex(core[j],fill);
        triangle(center,fi,fj);
        const auto ia=vertex(inner[i],stroke),ib=vertex(inner[j],stroke);
        triangle(fi,ia,ib); triangle(fi,ib,fj);
        const auto ai=vertex(inner[i],stroke),aj=vertex(inner[j],stroke),bi=vertex(outer[i],stroke),bj=vertex(outer[j],stroke);
        triangle(ai,bi,bj); triangle(ai,bj,aj);
        const auto ci=vertex(edge[i],stroke,0),cj=vertex(edge[j],stroke,0);
        triangle(bi,ci,cj); triangle(bi,cj,bj);
    }
}

// Build glow strokes as a single GPU triangle batch. Rasterizing six wide,
// antialiased QPainter strokes per bolt was the expensive zapper path.
static void appendStroke(QVector<Vertex> &out,const QPainterPath &path,const QTransform &transform,
                         QColor color,qreal lineWidth,qreal opacity,qreal fringe) {
    const qreal radius=lineWidth/2;
    auto vertex=[&](QPointF point,qreal coverage) {
        point=transform.map(point);
        const qreal alpha=qBound(0.,color.alphaF()*opacity*coverage,1.);
        return Vertex{float(point.x()),float(point.y()),static_cast<unsigned char>(qRound(color.red()*alpha)),
            static_cast<unsigned char>(qRound(color.green()*alpha)),static_cast<unsigned char>(qRound(color.blue()*alpha)),
            static_cast<unsigned char>(qRound(255*alpha))};
    };
    auto triangle=[&](QPointF a,QPointF b,QPointF c,qreal aa=1,qreal ab=1,qreal ac=1) {
        out.append(vertex(a,aa)); out.append(vertex(b,ab)); out.append(vertex(c,ac));
    };
    for(const auto &points:path.toSubpathPolygons()) {
        QVector<QPointF> normals;
        for(int i=0;i<points.size();++i) {
            auto normal=[](QPointF delta) {
                const qreal length=std::hypot(delta.x(),delta.y());
                return length>0.001 ? QPointF(-delta.y()/length,delta.x()/length) : QPointF();
            };
            const QPointF before=normal(points[i]-points[qMax(0,i-1)]);
            const QPointF after=normal(points[qMin(i+1,points.size()-1)]-points[i]);
            if(i==0)normals.append(after);
            else if(i==points.size()-1)normals.append(before);
            else {
                const QPointF sum=before+after;
                const qreal length=std::hypot(sum.x(),sum.y());
                const QPointF direction=length>0.001 ? sum/length : after;
                const qreal projection=QPointF::dotProduct(direction,after);
                normals.append(direction/qMax(0.5,projection));
            }
        }
        for(int i=1;i<points.size();++i) {
            const QPointF a=points[i-1],b=points[i],na=normals[i-1]*radius,nb=normals[i]*radius;
            const QPointF fa=normals[i-1]*(radius+fringe),fb=normals[i]*(radius+fringe);
            triangle(a+na,a-na,b+nb); triangle(a-na,b-nb,b+nb);
            triangle(a+na,b+nb,a+fa,1,1,0); triangle(b+nb,b+fb,a+fa,1,0,0);
            triangle(a-na,a-fa,b-nb,1,0,1); triangle(b-nb,a-fa,b-fb,1,0,0);
        }
        // Caps only at the ends; the joined strip covers the interior vertices.
        for(const auto &point:{points.first(),points.last()}) for(int i=0;i<12;++i) {
            const qreal a=i*2*std::acos(-1.)/12,b=(i+1)*2*std::acos(-1.)/12;
            const QPointF na(std::cos(a),std::sin(a)),nb(std::cos(b),std::sin(b));
            triangle(point,point+na*radius,point+nb*radius);
            triangle(point+na*radius,point+na*(radius+fringe),point+nb*radius,1,0,1);
            triangle(point+nb*radius,point+na*(radius+fringe),point+nb*(radius+fringe),1,0,0);
        }
    }
}

// Ear clipping runs only when an outline changes. Every displayed fragment
// then needs just transformed colored vertices, and a layer draws in one batch.
static QVector<QPointF> triangulate(const QPainterPath &path) {
    QPolygonF polygon=path.toFillPolygon();
    if(polygon.size()>1 && polygon.first()==polygon.last()) polygon.removeLast();
    QVector<QPointF> result;
    if(polygon.size()<3)return result;
    auto cross=[](const QPointF&a,const QPointF&b,const QPointF&c) { return (b.x()-a.x())*(c.y()-a.y())-(b.y()-a.y())*(c.x()-a.x()); };
    qreal area=0; for(int i=0;i<polygon.size();++i) { auto a=polygon[i],b=polygon[(i+1)%polygon.size()]; area+=a.x()*b.y()-b.x()*a.y(); }
    const qreal sign=area>=0 ? 1 : -1;
    QVector<int> indices; for(int i=0;i<polygon.size();++i)indices.append(i);
    while(indices.size()>3) {
        bool found=false;
        for(int i=0;i<indices.size();++i) {
            const int a=indices[(i+indices.size()-1)%indices.size()],b=indices[i],c=indices[(i+1)%indices.size()];
            if(cross(polygon[a],polygon[b],polygon[c])*sign<=0.0000001)continue;
            bool contains=false;
            for(int j:indices) {
                if(j==a || j==b || j==c)continue;
                if(cross(polygon[a],polygon[b],polygon[j])*sign>=-0.0000001 && cross(polygon[b],polygon[c],polygon[j])*sign>=-0.0000001 && cross(polygon[c],polygon[a],polygon[j])*sign>=-0.0000001) { contains=true; break; }
            }
            if(contains)continue;
            result.append(polygon[a]); result.append(polygon[b]); result.append(polygon[c]);
            indices.removeAt(i); found=true; break;
        }
        if(!found) { // Remove duplicate/collinear vertices before retrying.
            bool removed=false;
            for(int i=0;i<indices.size();++i) {
                if(std::abs(cross(polygon[indices[(i+indices.size()-1)%indices.size()]],polygon[indices[i]],polygon[indices[(i+1)%indices.size()]]))<0.0000001) { indices.removeAt(i); removed=true; break; }
            }
            if(!removed)break;
        }
    }
    if(indices.size()==3) { for(int i:indices)result.append(polygon[i]); }
    return result;
}

// Compile the original Canvas stream once in native code, then let QPainter
// execute it without thousands of JavaScript/Canvas calls per display frame.
class GameRenderer : public QQuickPaintedItem {
    Q_OBJECT
    Q_PROPERTY(qreal gameWidth READ gameWidth NOTIFY geometryChanged)
    Q_PROPERTY(qreal gameHeight READ gameHeight NOTIFY geometryChanged)
    Q_PROPERTY(int paintedFrames READ paintedFrames NOTIFY paintedFramesChanged)
    Q_PROPERTY(QString renderError READ renderError NOTIFY renderErrorChanged)
    Q_PROPERTY(QString fontFamily READ fontFamily WRITE setFontFamily NOTIFY fontFamilyChanged)
    Q_PROPERTY(bool frameClockRunning READ frameClockRunning WRITE setFrameClockRunning NOTIFY frameClockRunningChanged)
    Q_PROPERTY(bool diagnosticsEnabled READ diagnosticsEnabled CONSTANT)
    Q_PROPERTY(QString diagnosticsDirectory READ diagnosticsDirectory CONSTANT)
    struct Scene { QPicture back, bricks, front; qreal width=1800, height=900; quint64 brickRevision=0; bool gpu=false; QVector<Vertex> slow,fast,bolts; };
    struct Style {
        QColor fill=Qt::black, stroke=Qt::black;
        qreal lineWidth=1, opacity=1, shadowBlur=0;
        QColor shadow=Qt::transparent;
        Qt::PenCapStyle cap=Qt::FlatCap;
        Qt::PenJoinStyle join=Qt::MiterJoin;
        QList<qreal> dash;
        QFont font;
        QString align="start", baseline="alphabetic";
    };
    struct Outline { QPainterPath path; QVector<QPointF> triangles; QJsonArray points; qreal radius=0, roundness=0; bool circle=false; };
    QHash<quint32,Outline> outlines;
    QSet<quint32> frameOutlines;
    QHash<QString,QColor> debrisColors;
    DebrisLayer *slowLayer=nullptr,*fastLayer=nullptr,*boltLayer=nullptr;
    PictureLayer *brickLayer=nullptr,*frontLayer=nullptr;
    DebrisLayer *brickGpuLayer=nullptr;
    QHash<int,QVector<Vertex>> brickMeshes;
    QHash<QString,QVector<Vertex>> brickTemplates;
    QHash<int,QJsonArray> brickCommands;
    int activeBrickCache=-1;
    quint64 childBrickRevision=0;
    QMutex mutex;
    std::shared_ptr<Scene> scene=std::make_shared<Scene>();
    QHash<int,QPicture> caches;
    QHash<int,quint64> revisions;
    quint64 nextRevision=0, renderedRevision=0;
    QImage brickImage;
    QSize brickSize;
    std::atomic<int> count{0};
    std::atomic<quint64> clockEpoch{0};
    QElapsedTimer frameClock;
    QMetaObject::Connection clockConnection;
    QMetaObject::Connection submissionConnection;
    FrameRecorder recorder;

    QString error, family="monospace";
    qreal gw=1800, gh=900;

    static QColor color(const QString &text) {
        if (text.startsWith("rgb(" ) || text.startsWith("rgba(")) {
            auto parts=text.mid(text.indexOf('(')+1).chopped(1).split(',');
            if (parts.size()>=3) return QColor::fromRgbF(parts[0].toDouble()/255., parts[1].toDouble()/255., parts[2].toDouble()/255., parts.size()==4 ? parts[3].toDouble() : 1.);
        }
        return QColor(text);
    }
    void ensureOutline(quint32 id,qreal radius,bool circle,qreal roundness,const QJsonArray &points) {
        if(!outlines.contains(id) || outlines[id].radius!=radius || outlines[id].roundness!=roundness || outlines[id].circle!=circle || outlines[id].points!=points) {
            Outline outline; outline.points=points; outline.radius=radius; outline.roundness=roundness; outline.circle=circle;
            auto &shape=outline.path; shape.setFillRule(Qt::WindingFill);
            const int count=points.size()/2;
            if(circle) shape.addEllipse(QPointF(0,0),radius,radius);
            else if(count>=3 && roundness<=0.001) {
                shape.moveTo(points[0].toDouble(),points[1].toDouble());
                for(int i=1;i<count;++i) shape.lineTo(points[i*2].toDouble(),points[i*2+1].toDouble());
                shape.closeSubpath();
            } else if(count>=3) {
                const qreal corner=0.08+qBound(0.,roundness,1.)*0.34;
                QPointF last(points[(count-1)*2].toDouble(),points[(count-1)*2+1].toDouble());
                QPointF first(points[0].toDouble(),points[1].toDouble());
                shape.moveTo(last+(first-last)*(1-corner));
                for(int i=0;i<count;++i) {
                    const int next=(i+1)%count;
                    QPointF current(points[i*2].toDouble(),points[i*2+1].toDouble());
                    QPointF following(points[next*2].toDouble(),points[next*2+1].toDouble());
                    shape.quadTo(current,current+(following-current)*corner);
                }
                shape.closeSubpath();
            }
            outline.triangles=triangulate(shape);
            outlines[id]=outline;
        }
    }
    void drawDebris(QPainter &painter,QVector<Vertex> *vertices,quint32 id,qreal x,qreal y,qreal c,qreal s,const QString &text) {
        frameOutlines.insert(id);
        const auto outline=outlines.constFind(id);
        if(outline==outlines.cend()) { error="Missing particle outline: " + QString::number(id); return; }
        auto fill=debrisColors.constFind(text);
        if(fill==debrisColors.cend()) {
            if(debrisColors.size()>4096) debrisColors.clear();
            fill=debrisColors.insert(text,color(text));
        }
        if(vertices) {
            const auto red=static_cast<unsigned char>(fill->red()),green=static_cast<unsigned char>(fill->green()),blue=static_cast<unsigned char>(fill->blue());
            const auto &triangles=outline->triangles;
            vertices->reserve(vertices->size()+triangles.size());
            for(const auto &point:triangles) vertices->append(Vertex{float(x+point.x()*c-point.y()*s),float(y+point.x()*s+point.y()*c),red,green,blue,255});
        } else {
            painter.save(); painter.setTransform(QTransform(c,s,-s,c,x,y));
            painter.fillPath(outline->path,*fill); painter.restore();
        }
    }
    QPicture compile(const QJsonArray &ops,QVector<Vertex> *vertices=nullptr,QVector<Vertex> *bolts=nullptr) {
        QPicture picture;
        QPainter painter(&picture);
        painter.setRenderHint(QPainter::Antialiasing);
        painter.setRenderHint(QPainter::TextAntialiasing);
        Style style; style.font.setFamily(family); style.font.setPixelSize(16);
        QList<Style> stack;
        QPainterPath path; path.setFillRule(Qt::WindingFill);
        for (const auto &entry: ops) {
            const auto op=entry.toArray();
            const QString name=op[0].toString();
            auto n=[&](int i){ return op[i].toDouble(); };
            if (name=="set") {
                const QString key=op[1].toString(), value=op[2].toString();
                if (key=="fillStyle") style.fill=color(value);
                else if (key=="strokeStyle") style.stroke=color(value);
                else if (key=="lineWidth") style.lineWidth=n(2);
                else if (key=="shadowBlur") style.shadowBlur=n(2);
                else if (key=="shadowColor") style.shadow=color(value);
                else if (key=="globalAlpha") { style.opacity=n(2); painter.setOpacity(style.opacity); }
                else if (key=="lineCap") style.cap=value=="round" ? Qt::RoundCap : value=="square" ? Qt::SquareCap : Qt::FlatCap;
                else if (key=="lineJoin") style.join=value=="round" ? Qt::RoundJoin : value=="bevel" ? Qt::BevelJoin : Qt::MiterJoin;
                else if (key=="textAlign") style.align=value;
                else if (key=="textBaseline") style.baseline=value;
                else if (key=="font") {
                    static const QRegularExpression size("([0-9.]+)px");
                    auto match=size.match(value);
                    style.font.setPixelSize(match.hasMatch() ? qRound(match.captured(1).toDouble()) : 16);
                    style.font.setBold(value.contains("bold") || value.contains("GameFont"));
                } else if (key!="shadowBlur" && key!="shadowColor" && key!="shadowOffsetX" && key!="shadowOffsetY") {
                    error="Unsupported drawing property: " + key;
                }
            } else if (name=="save") { painter.save(); stack.append(style); }
            else if (name=="restore") { if (!stack.empty()) { painter.restore(); style=stack.takeLast(); } }
            else if (name=="beginPath") { path=QPainterPath(); path.setFillRule(Qt::WindingFill); }
            else if (name=="closePath") path.closeSubpath();
            else if (name=="moveTo") path.moveTo(n(1),n(2));
            else if (name=="lineTo") path.lineTo(n(1),n(2));
            else if (name=="quadraticCurveTo") path.quadTo(n(1),n(2),n(3),n(4));
            else if (name=="roundRect") {
                const qreal x=n(1),y=n(2),w=n(3),h=n(4),r=qMax(0.,qMin(n(5),qMin(w/2,h/2)));
                path.moveTo(x+r,y); path.lineTo(x+w-r,y); path.quadTo(x+w,y,x+w,y+r);
                path.lineTo(x+w,y+h-r); path.quadTo(x+w,y+h,x+w-r,y+h);
                path.lineTo(x+r,y+h); path.quadTo(x,y+h,x,y+h-r);
                path.lineTo(x,y+r); path.quadTo(x,y,x+r,y); path.closeSubpath();
            } else if (name=="bricks") {
                const auto values=op[1].toArray(); const bool tilted=op[2].toBool();
                QVector<QPainterPath> groups[3];
                if(vertices) vertices->reserve(vertices->size()+values.size()/6*588);
                for(int i=0;i+5<values.size();i+=6) {
                    const qreal x=values[i].toDouble(),y=values[i+1].toDouble(),w=values[i+2].toDouble(),h=values[i+3].toDouble();
                    const int kind=qBound(0,values[i+5].toInt(),2);
                    const qreal r=qMax(0.,qMin(n(3),qMin(w/2,h/2)));
                    if(vertices) {
                        const qreal dpr=window() ? window()->effectiveDevicePixelRatio() : 1;
                        const qreal fringe=gw/qMax(1.,width()*dpr);
                        const QColor fill=color(op[4+kind*2].toString()),stroke=color(op[5+kind*2].toString());
                        const QString key=QString::number(w)+":"+QString::number(h)+":"+QString::number(r)+":"+QString::number(kind)+":"+fill.name(QColor::HexArgb)+":"+stroke.name(QColor::HexArgb)+":"+QString::number(fringe);
                        if(!brickTemplates.contains(key)) {
                            if(brickTemplates.size()>256) brickTemplates.clear();
                            QVector<Vertex> mesh; appendBrickMesh(mesh,0,0,w,h,0,r,fill,stroke,kind==0 ? 1 : 2,fringe);
                            brickTemplates.insert(key,std::move(mesh));
                        }
                        const qreal angle=tilted ? values[i+4].toDouble() : 0,c=std::cos(angle),s=std::sin(angle);
                        const auto &mesh=brickTemplates[key];
                        for(auto v:mesh) {
                            const qreal px=v.x-w/2,py=v.y-h/2;
                            v.x=x+w/2+px*c-py*s; v.y=y+h/2+px*s+py*c;
                            vertices->append(v);
                        }
                        continue;
                    }
                    QPainterPath brick;
                    brick.moveTo(r,0); brick.lineTo(w-r,0); brick.quadTo(w,0,w,r);
                    brick.lineTo(w,h-r); brick.quadTo(w,h,w-r,h);
                    brick.lineTo(r,h); brick.quadTo(0,h,0,h-r);
                    brick.lineTo(0,r); brick.quadTo(0,0,r,0); brick.closeSubpath();
                    QTransform transform; transform.translate(x+w/2,y+h/2);
                    if(tilted) transform.rotateRadians(values[i+4].toDouble());
                    transform.translate(-w/2,-h/2); groups[kind].append(transform.map(brick));
                }
                for(int kind=0;kind<3;++kind) {
                    const auto fill=color(op[4+kind*2].toString());
                    const QPen pen(color(op[5+kind*2].toString()),kind==0 ? 1 : 2,Qt::SolidLine,Qt::FlatCap,Qt::MiterJoin);
                    for(const auto &brick:groups[kind]) { painter.fillPath(brick,fill); painter.strokePath(brick,pen); }
                }
            } else if (name=="arc") {
                const qreal pi=std::acos(-1.), full=2*pi, start=n(4);
                qreal delta=n(5)-start;
                const bool reverse=op.size()>6 && op[6].toBool();
                if (!reverse) { if (delta>=full) delta=full; else { delta=std::fmod(delta,full); if(delta<0) delta+=full; } }
                else { if (delta<=-full) delta=-full; else { delta=std::fmod(delta,full); if(delta>0) delta-=full; } }
                const QPointF first(n(1)+n(3)*std::cos(start),n(2)+n(3)*std::sin(start));
                if (path.isEmpty()) path.moveTo(first); else path.lineTo(first);
                path.arcTo(QRectF(n(1)-n(3),n(2)-n(3),2*n(3),2*n(3)),-start*180/pi,-delta*180/pi);
            } else if (name=="debris") {
                const quint32 id=quint32(n(1)); const qreal radius=n(7),roundness=n(9);
                const bool circle=op[8].toBool(); const auto points=op[10].toArray();
                ensureOutline(id,radius,circle,roundness,points);
                drawDebris(painter,vertices,id,n(2),n(3),n(4),n(5),op[6].toString());
            } else if (name=="debris-batch") {
                const auto batch=op[1].toObject();
                const auto instances=batch["instances"].toArray(),colors=batch["colors"].toArray();
                if(instances.size()!=colors.size()*5) { error="Invalid particle batch"; continue; }
                for(int i=0;i<colors.size();++i) {
                    const int base=i*5;
                    drawDebris(painter,vertices,quint32(instances[base].toDouble()),
                        instances[base+1].toDouble(),instances[base+2].toDouble(),
                        instances[base+3].toDouble(),instances[base+4].toDouble(),colors[i].toString());
                }
            } else if (name=="fill") painter.fillPath(path,style.fill);
            else if (name=="stroke") {
                if(bolts && style.shadowBlur>0 && style.shadow.alpha()>0) {
                    const qreal fringe=qMax(0.5,gw/qMax(width(),1.));
                    QColor glow=style.shadow; glow.setAlphaF(glow.alphaF()*0.18);
                    for(int layer=3;layer>=1;--layer) appendStroke(*bolts,path,painter.transform(),glow,style.lineWidth+style.shadowBlur*layer/1.5,style.opacity,fringe);
                    appendStroke(*bolts,path,painter.transform(),style.stroke,style.lineWidth,style.opacity,fringe);
                    continue;
                }
                // Software/replay fallback retains the same colored halo.
                if(style.shadowBlur>0 && style.shadow.alpha()>0) {
                    QColor glow=style.shadow; glow.setAlphaF(glow.alphaF()*0.09);
                    for(int layer=6;layer>=1;--layer) painter.strokePath(path,QPen(glow,style.lineWidth+style.shadowBlur*layer/3.,Qt::SolidLine,Qt::RoundCap,Qt::RoundJoin));
                }
                QPen pen(style.stroke,style.lineWidth,Qt::SolidLine,style.cap,style.join);
                if (!style.dash.empty()) { QList<qreal> scaled; for (auto d:style.dash) scaled.append(d/qMax(style.lineWidth,0.001)); pen.setDashPattern(scaled); }
                painter.strokePath(path,pen);
            } else if (name=="fillRect") painter.fillRect(QRectF(n(1),n(2),n(3),n(4)),style.fill);
            else if (name=="fillText") {
                painter.setFont(style.font); painter.setPen(style.fill);
                const QString text=op[1].toString(); const QFontMetricsF metrics(style.font);
                qreal x=n(2),y=n(3),w=metrics.horizontalAdvance(text);
                if (style.align=="center") x-=w/2; else if (style.align=="right" || style.align=="end") x-=w;
                if (style.baseline=="top" || style.baseline=="hanging") y+=metrics.ascent();
                else if (style.baseline=="middle") y+=(metrics.ascent()-metrics.descent())/2;
                else if (style.baseline=="bottom") y-=metrics.descent();
                painter.drawText(QPointF(x,y),text);
            } else if (name=="setTransform") painter.setTransform(QTransform(n(1),n(2),n(3),n(4),n(5),n(6)));
            else if (name=="translate") painter.translate(n(1),n(2));
            else if (name=="rotate") painter.rotate(n(1)*180/std::acos(-1.));
            else if (name=="setLineDash") { style.dash.clear(); for (const auto &d:op[1].toArray()) style.dash.append(d.toDouble()); }
            else if (name!="clearRect") error="Unsupported drawing command: " + name;
        }
        while (!stack.empty()) { painter.restore(); stack.removeLast(); }
        painter.end();
        return picture;
    }
public:
    GameRenderer(QQuickItem *parent=nullptr):QQuickPaintedItem(parent) {
        frameClock.start();
        connect(this,&QQuickItem::windowChanged,this,[this](QQuickWindow *view) {
            disconnect(clockConnection);
            disconnect(submissionConnection);
            clockEpoch.fetch_add(2);
            if(!view)return;
            if(recorder.enabled)submissionConnection=connect(view,&QQuickWindow::frameSwapped,this,
                [counter=recorder.submitted]{counter->fetch_add(1,std::memory_order_relaxed);},Qt::DirectConnection);
            // This window's preparation point gives the engine a full render
            // interval to return its next scene, without a shared animation timer.
            clockConnection=connect(view,&QQuickWindow::afterAnimating,this,
                [this,view,last=qint64(0),seen=quint64(0)]() mutable {
                    const quint64 epoch=clockEpoch.load();
                    if(!(epoch&1)) { last=0; seen=epoch; return; }
                    if(epoch!=seen) { last=0; seen=epoch; }
                    const qint64 now=frameClock.nsecsElapsed();
                    const qreal refresh=view->screen() ? view->screen()->refreshRate() : 60.;
                    const qreal elapsed=last ? (now-last)/1e9 : 1./qMax(1.,refresh);
                    last=now;
                    recorder.clockTick(elapsed);
                    if(clockEpoch.load()==epoch) {
                        emit frameTick(elapsed);
                        // Engine replies dirty the scene asynchronously. Keep
                        // requesting display frames while playing so their
                        // arrival time cannot become the clock's own cadence.
                        // Window updates reuse unchanged item textures.
                        if(clockEpoch.load()==epoch&&view->isVisible())view->update();
                    }
                },Qt::DirectConnection);
        });

        setAntialiasing(true); setRenderTarget(QQuickPaintedItem::FramebufferObject); setOpaquePainting(true); setClip(true);
        slowLayer=new DebrisLayer(this); slowLayer->setZ(1);
        brickLayer=new PictureLayer(this); brickLayer->setZ(2);
        brickGpuLayer=new DebrisLayer(this); brickGpuLayer->setZ(2);
        // The image path remains for software rendering and legacy packets;
        // current brick batches use cached GPU meshes with antialiased edges.
        brickLayer->setRenderTarget(QQuickPaintedItem::Image);
        fastLayer=new DebrisLayer(this); fastLayer->setZ(3);
        boltLayer=new DebrisLayer(this); boltLayer->setZ(4);
        frontLayer=new PictureLayer(this); frontLayer->setZ(5);
        // Raster text and rounded moving objects with smooth coverage; particle
        // geometry remains in separate GPU batches beneath this texture.
        frontLayer->setRenderTarget(QQuickPaintedItem::Image);
    }
    bool frameClockRunning()const{return clockEpoch.load()&1;}
    bool diagnosticsEnabled()const{return recorder.enabled;}
    QString diagnosticsDirectory()const{return recorder.path();}
    Q_INVOKABLE void diagnosticTickSent(){recorder.tickSent();}
    Q_INVOKABLE void diagnosticTickReply(const QVariantMap &timing={},qint64 receivedUs=0){recorder.tickReply(QJsonObject::fromVariantMap(timing),receivedUs);}
    Q_INVOKABLE void diagnosticResetRequests(){recorder.resetRequests();}
    Q_INVOKABLE void beginFrameDiagnostics(){recorder.begin();}
    Q_INVOKABLE void finishFrameDiagnostics(){recorder.finish("window-closed");}
    Q_INVOKABLE void sampleFrameDiagnostics(const QVariantMap &values) {
        auto state=QJsonObject::fromVariantMap(values);
        if(window()) {
            state["window_active"]=window()->isActive();
            state["window_visible"]=window()->isVisible();
            state["device_pixel_ratio"]=window()->effectiveDevicePixelRatio();
            state["graphics_api"]=int(window()->rendererInterface()->graphicsApi());
            state["screen_refresh_hz"]=window()->screen()?window()->screen()->refreshRate():0;
        }
        recorder.sample(state);
    }
    void setFrameClockRunning(bool active) {
        if(active==frameClockRunning())return;
        // A new epoch discards the idle interval and any previous window clock.
        clockEpoch.fetch_add(1); emit frameClockRunningChanged();
        if(active)update();
    }
    qreal gameWidth()const{return gw;} qreal gameHeight()const{return gh;}
    int paintedFrames()const{return count.load();}
    QString renderError()const{return error;}
    QString fontFamily()const{return family;}
    void setFontFamily(const QString &value) { if(family==value)return; family=value; emit fontFamilyChanged(); }
    Q_INVOKABLE QVariantMap acceptLine(const QString &line) {
        QElapsedTimer diagnosticTimer; if(recorder.enabled)diagnosticTimer.start();
        const bool profile=qEnvironmentVariableIsSet("PHYSICAL_RENDER_TIMINGS");
        QElapsedTimer timer; if(profile) timer.start();
        QJsonParseError parse;
        QJsonObject packet=QJsonDocument::fromJson(line.toUtf8(),&parse).object();
        if(parse.error!=QJsonParseError::NoError) { error=parse.errorString(); emit renderErrorChanged(); return {}; }
        if(packet["type"].toString()!="frame") return packet.toVariantMap();
        const qreal parseMs=profile ? timer.nsecsElapsed()/1e6 : 0;
        frameOutlines.clear();
        const auto shapes=packet["debrisShapes"].toObject();
        for(auto it=shapes.begin();it!=shapes.end();++it) {
            const auto shape=it.value().toObject();
            ensureOutline(it.key().toUInt(),shape["radius"].toDouble(),shape["circle"].toBool(),
                shape["roundness"].toDouble(),shape["points"].toArray());
        }
        const auto updates=packet["caches"].toObject();
        const bool gpu=window() && window()->rendererInterface()->graphicsApi()==QSGRendererInterface::OpenGL;
        for(auto it=updates.begin();it!=updates.end();++it) {
            int id=it.key().toInt(); QVector<Vertex> mesh; brickCommands[id]=it.value().toArray();
            caches[id]=compile(it.value().toArray(),gpu ? &mesh : nullptr);
            brickMeshes[id]=std::move(mesh); revisions[id]=++nextRevision;
        }
        const qreal cacheMs=profile ? timer.nsecsElapsed()/1e6-parseMs : 0;
        const auto ops=packet["ops"].toArray();
        auto next=std::make_shared<Scene>();
        next->width=packet["width"].toDouble(1800); next->height=packet["height"].toDouble(900);
        QJsonArray before,after; int cache=-1;
        for(const auto &op:ops) { const auto array=op.toArray(); if(array[0].toString()=="cache") cache=array[1].toInt(); else if(cache<0) before.append(op); else after.append(op); }
        if(cache<0) { next->front=compile(before); }
        else {
            next->gpu=window() && window()->rendererInterface()->graphicsApi()==QSGRendererInterface::OpenGL;
            next->back=compile(before,next->gpu ? &next->slow : nullptr);
            next->bricks=caches.value(cache); next->brickRevision=revisions.value(cache);
            next->front=compile(after,next->gpu ? &next->fast : nullptr,next->gpu ? &next->bolts : nullptr);
        }
        for(auto it=outlines.begin();it!=outlines.end();) {
            if(!frameOutlines.contains(it.key())) it=outlines.erase(it); else ++it;
        }
        activeBrickCache=cache;
        slowLayer->setVisible(next->gpu); brickLayer->setVisible(next->gpu && brickMeshes.value(cache).isEmpty()); brickGpuLayer->setVisible(next->gpu && !brickMeshes.value(cache).isEmpty()); fastLayer->setVisible(next->gpu); boltLayer->setVisible(next->gpu); frontLayer->setVisible(next->gpu);
        if(next->gpu) {
            slowLayer->setFrame(next->slow,next->width,next->height);
            fastLayer->setFrame(next->fast,next->width,next->height);
            boltLayer->setFrame(next->bolts,next->width,next->height);
            frontLayer->setFrame(next->front,next->width,next->height);
            if(childBrickRevision!=next->brickRevision) {
                const auto mesh=brickMeshes.value(cache);
                if(mesh.isEmpty()) brickLayer->setFrame(next->bricks,next->width,next->height);
                else brickGpuLayer->setFrame(mesh,next->width,next->height);
                childBrickRevision=next->brickRevision;
            }
        }
        if(gw!=next->width || gh!=next->height) { gw=next->width; gh=next->height; emit geometryChanged(); }
        { QMutexLocker lock(&mutex); scene=next; }
        if(profile) {
            auto state=packet["state"].toObject();
            state["qtParseMs"]=parseMs; state["qtCacheMs"]=cacheMs;
            state["qtCompileMs"]=timer.nsecsElapsed()/1e6-parseMs-cacheMs;
            state["qtBrickPaintMs"]=brickLayer->paintMs(); state["qtFrontPaintMs"]=frontLayer->paintMs();
            state["qtFrontPaintCount"]=frontLayer->paintCount();
            packet["state"]=state;
        }
        packet.remove("ops"); packet.remove("caches"); packet.remove("debrisShapes");
        if(!error.isEmpty()) emit renderErrorChanged();
        update();
        if(recorder.enabled)recorder.acceptedFrame(diagnosticTimer.nsecsElapsed()/1e6);
        return packet.toVariantMap();
    }
    void paint(QPainter *painter) override {
        QElapsedTimer diagnosticTimer; if(recorder.enabled)diagnosticTimer.start();
        std::shared_ptr<Scene> current;
        { QMutexLocker lock(&mutex); current=scene; }
        if(width()<=0 || height()<=0)return;
        // QPicture playback restores the caller's transform. Scale at playback,
        // and cache the static brick raster at the displayed physical size.
        const qreal ratio=window() ? window()->effectiveDevicePixelRatio() : 1;
        const QSize size(qMax(1,qCeil(width()*ratio)),qMax(1,qCeil(height()*ratio)));
        if(!current->gpu && current->brickRevision && (renderedRevision!=current->brickRevision || brickSize!=size)) {
            brickImage=QImage(size,QImage::Format_ARGB32_Premultiplied); brickImage.fill(Qt::transparent);
            QPainter cached(&brickImage); cached.scale(size.width()/current->width,size.height()/current->height);
            cached.drawPicture(0,0,current->bricks); cached.end();
            brickSize=size; renderedRevision=current->brickRevision;
        }
        painter->save(); painter->scale(width()/current->width,height()/current->height);
        painter->drawPicture(0,0,current->back); painter->restore();
        if(!current->gpu && current->brickRevision) painter->drawImage(QRectF(0,0,width(),height()),brickImage);
        painter->save(); painter->scale(width()/current->width,height()/current->height);
        if(!current->gpu) painter->drawPicture(0,0,current->front); painter->restore();
        count.fetch_add(1);
        if(recorder.enabled)recorder.paintedFrame(diagnosticTimer.nsecsElapsed());
        QMetaObject::invokeMethod(this,[this]{ emit paintedFramesChanged(); },Qt::QueuedConnection);
    }
protected:
    void geometryChange(const QRectF &next,const QRectF &previous) override {
        QQuickPaintedItem::geometryChange(next,previous);
        for(QQuickItem *item:{static_cast<QQuickItem*>(slowLayer),static_cast<QQuickItem*>(brickLayer),static_cast<QQuickItem*>(brickGpuLayer),static_cast<QQuickItem*>(fastLayer),static_cast<QQuickItem*>(boltLayer),static_cast<QQuickItem*>(frontLayer)}) {
            if(item) { item->setWidth(next.width()); item->setHeight(next.height()); }
        }
        if(activeBrickCache>=0 && window() && window()->rendererInterface()->graphicsApi()==QSGRendererInterface::OpenGL && !brickMeshes.value(activeBrickCache).isEmpty()) {
            QVector<Vertex> mesh; compile(brickCommands.value(activeBrickCache),&mesh);
            brickMeshes[activeBrickCache]=mesh; brickGpuLayer->setFrame(mesh,gw,gh);
        }
    }
signals:
    void frameTick(qreal seconds); void frameClockRunningChanged();
    void geometryChanged(); void paintedFramesChanged(); void renderErrorChanged(); void fontFamilyChanged();
};

class PhysicalRendererPlugin : public QQmlExtensionPlugin {
    Q_OBJECT
    Q_PLUGIN_METADATA(IID QQmlExtensionInterface_iid)
public:
    void registerTypes(const char *uri) override { qmlRegisterType<GameRenderer>(uri,1,0,"GameRenderer"); qmlRegisterType<PointerLock>(uri,1,0,"PointerLock"); qmlRegisterType<AudioMixer>(uri,1,0,"AudioMixer"); qmlRegisterType<EngineProcess>(uri,1,0,"EngineProcess"); }
};
#include "renderer.moc"
#include "audiomixer.moc"
#include "engineprocess.moc"
