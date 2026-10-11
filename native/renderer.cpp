#include <QQuickPaintedItem>
#include <QQuickWindow>
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
#include <QMutexLocker>
#include <atomic>
#include <cmath>
#include <memory>
#include "pointerlock.h"

struct Vertex { float x,y; unsigned char r,g,b,a; };

class DebrisLayer : public QQuickItem {
    QVector<Vertex> vertices;
    qreal gw=1800,gh=900;
public:
    DebrisLayer(QQuickItem *parent):QQuickItem(parent) { setFlag(ItemHasContents,true); }
    void setFrame(const QVector<Vertex> &next,qreal w,qreal h) { vertices=next; gw=w; gh=h; update(); }
protected:
    QSGNode *updatePaintNode(QSGNode *old,UpdatePaintNodeData *) override {
        auto node=static_cast<QSGGeometryNode *>(old);
        if(vertices.isEmpty()) { delete node; return nullptr; }
        if(!node) {
            node=new QSGGeometryNode;
            auto geometry=new QSGGeometry(QSGGeometry::defaultAttributes_ColoredPoint2D(),vertices.size());
            geometry->setDrawingMode(QSGGeometry::DrawTriangles);
            node->setGeometry(geometry); node->setFlag(QSGNode::OwnsGeometry);
            node->setMaterial(new QSGVertexColorMaterial); node->setFlag(QSGNode::OwnsMaterial);
        } else node->geometry()->allocate(vertices.size());
        auto output=node->geometry()->vertexDataAsColoredPoint2D();
        for(int i=0;i<vertices.size();++i) {
            auto &v=vertices[i]; output[i].set(v.x*width()/gw,v.y*height()/gh,v.r,v.g,v.b,v.a);
        }
        node->markDirty(QSGNode::DirtyGeometry);
        return node;
    }
    void geometryChange(const QRectF &next,const QRectF &previous) override { QQuickItem::geometryChange(next,previous); update(); }
};

class PictureLayer : public QQuickPaintedItem {
    QPicture picture;
    qreal gw=1800,gh=900;
public:
    PictureLayer(QQuickItem *parent):QQuickPaintedItem(parent) { setAntialiasing(true); setRenderTarget(FramebufferObject); }
    void setFrame(const QPicture &next,qreal w,qreal h) { picture=next; gw=w; gh=h; update(); }
    void paint(QPainter *painter) override { painter->scale(width()/gw,height()/gh); painter->drawPicture(0,0,picture); }
};

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
    DebrisLayer *slowLayer=nullptr,*fastLayer=nullptr,*boltLayer=nullptr;
    PictureLayer *brickLayer=nullptr,*frontLayer=nullptr;
    quint64 childBrickRevision=0;
    QMutex mutex;
    std::shared_ptr<Scene> scene=std::make_shared<Scene>();
    QHash<int,QPicture> caches;
    QHash<int,quint64> revisions;
    quint64 nextRevision=0, renderedRevision=0;
    QImage brickImage;
    QSize brickSize;
    std::atomic<int> count{0};
    QString error, family="monospace";
    qreal gw=1800, gh=900;

    static QColor color(const QString &text) {
        if (text.startsWith("rgb(" ) || text.startsWith("rgba(")) {
            auto parts=text.mid(text.indexOf('(')+1).chopped(1).split(',');
            if (parts.size()>=3) return QColor::fromRgbF(parts[0].toDouble()/255., parts[1].toDouble()/255., parts[2].toDouble()/255., parts.size()==4 ? parts[3].toDouble() : 1.);
        }
        return QColor(text);
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
                if(outlines.size()>2000) outlines.clear();
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
                const QColor fill=color(op[6].toString());
                if(vertices) {
                    for(const auto &point:outlines[id].triangles) vertices->append(Vertex{float(n(2)+point.x()*n(4)-point.y()*n(5)),float(n(3)+point.x()*n(5)+point.y()*n(4)),static_cast<unsigned char>(fill.red()),static_cast<unsigned char>(fill.green()),static_cast<unsigned char>(fill.blue()),255});
                } else {
                    painter.save(); painter.setTransform(QTransform(n(4),n(5),-n(5),n(4),n(2),n(3)));
                    painter.fillPath(outlines[id].path,fill); painter.restore();
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
        setAntialiasing(true); setRenderTarget(QQuickPaintedItem::FramebufferObject); setOpaquePainting(true); setClip(true);
        slowLayer=new DebrisLayer(this); slowLayer->setZ(1);
        brickLayer=new PictureLayer(this); brickLayer->setZ(2);
        // Bricks are cached: rasterize their rounded edges with high-quality
        // antialiasing once per change, then composite the texture on the GPU.
        brickLayer->setRenderTarget(QQuickPaintedItem::Image);
        fastLayer=new DebrisLayer(this); fastLayer->setZ(3);
        boltLayer=new DebrisLayer(this); boltLayer->setZ(4);
        frontLayer=new PictureLayer(this); frontLayer->setZ(5);
        // Raster text and rounded moving objects with smooth coverage; particle
        // geometry remains in separate GPU batches beneath this texture.
        frontLayer->setRenderTarget(QQuickPaintedItem::Image);
    }
    qreal gameWidth()const{return gw;} qreal gameHeight()const{return gh;}
    int paintedFrames()const{return count.load();}
    QString renderError()const{return error;}
    QString fontFamily()const{return family;}
    void setFontFamily(const QString &value) { if(family==value)return; family=value; emit fontFamilyChanged(); }
    Q_INVOKABLE QVariantMap acceptLine(const QString &line) {
        QJsonParseError parse;
        QJsonObject packet=QJsonDocument::fromJson(line.toUtf8(),&parse).object();
        if(parse.error!=QJsonParseError::NoError) { error=parse.errorString(); emit renderErrorChanged(); return {}; }
        if(packet["type"].toString()!="frame") return packet.toVariantMap();
        const auto updates=packet["caches"].toObject();
        for(auto it=updates.begin();it!=updates.end();++it) { int id=it.key().toInt(); caches[id]=compile(it.value().toArray()); revisions[id]=++nextRevision; }
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
        slowLayer->setVisible(next->gpu); brickLayer->setVisible(next->gpu); fastLayer->setVisible(next->gpu); boltLayer->setVisible(next->gpu); frontLayer->setVisible(next->gpu);
        if(next->gpu) {
            slowLayer->setFrame(next->slow,next->width,next->height);
            fastLayer->setFrame(next->fast,next->width,next->height);
            boltLayer->setFrame(next->bolts,next->width,next->height);
            frontLayer->setFrame(next->front,next->width,next->height);
            if(childBrickRevision!=next->brickRevision) { brickLayer->setFrame(next->bricks,next->width,next->height); childBrickRevision=next->brickRevision; }
        }
        if(gw!=next->width || gh!=next->height) { gw=next->width; gh=next->height; emit geometryChanged(); }
        { QMutexLocker lock(&mutex); scene=next; }
        packet.remove("ops"); packet.remove("caches");
        if(!error.isEmpty()) emit renderErrorChanged();
        update();
        return packet.toVariantMap();
    }
    void paint(QPainter *painter) override {
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
        QMetaObject::invokeMethod(this,[this]{ emit paintedFramesChanged(); },Qt::QueuedConnection);
    }
protected:
    void geometryChange(const QRectF &next,const QRectF &previous) override {
        QQuickPaintedItem::geometryChange(next,previous);
        for(QQuickItem *item:{static_cast<QQuickItem*>(slowLayer),static_cast<QQuickItem*>(brickLayer),static_cast<QQuickItem*>(fastLayer),static_cast<QQuickItem*>(boltLayer),static_cast<QQuickItem*>(frontLayer)}) {
            if(item) { item->setWidth(next.width()); item->setHeight(next.height()); }
        }
    }
signals:
    void geometryChanged(); void paintedFramesChanged(); void renderErrorChanged(); void fontFamilyChanged();
};

class PhysicalRendererPlugin : public QQmlExtensionPlugin {
    Q_OBJECT
    Q_PLUGIN_METADATA(IID QQmlExtensionInterface_iid)
public:
    void registerTypes(const char *uri) override { qmlRegisterType<GameRenderer>(uri,1,0,"GameRenderer"); qmlRegisterType<PointerLock>(uri,1,0,"PointerLock"); }
};
#include "renderer.moc"
