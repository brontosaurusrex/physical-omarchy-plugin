#include "../native/renderer.cpp"
#include <QGuiApplication>
#include <cstdio>
#include <cstdlib>

static void require(bool value,const char *message) {
    if(!value) { std::fprintf(stderr,"FAIL: %s\n",message); std::exit(1); }
}
static qreal area(const QPolygonF &points) {
    qreal sum=0;
    for(int i=0;i<points.size();++i) { auto a=points[i],b=points[(i+1)%points.size()]; sum+=a.x()*b.y()-b.x()*a.y(); }
    return std::abs(sum)/2;
}
static void checkOutline(const QPainterPath &path) {
    auto triangles=triangulate(path); qreal total=0;
    for(int i=0;i<triangles.size();i+=3) total+=area(QPolygonF{triangles[i],triangles[i+1],triangles[i+2]});
    require(!triangles.empty() && triangles.size()%3==0,"outline produced no complete triangles");
    require(std::abs(total-area(path.toFillPolygon()))<0.00001,"triangulation changed the outline area");
}
class TestGeometryLayer : public DebrisLayer {
public:
    TestGeometryLayer():DebrisLayer(nullptr) {}
    using DebrisLayer::updatePaintNode;
};
int main(int argc,char **argv) {
    QGuiApplication application(argc,argv);
    QPainterPath circle; circle.addEllipse(QPointF(0,0),12,12); checkOutline(circle);
    QPainterPath rounded; rounded.addRoundedRect(QRectF(-12,-5,24,10),3,3); checkOutline(rounded);
    QPainterPath star;
    for(int i=0;i<16;++i) { qreal radius=i%2 ? 4 : 12, angle=i*std::acos(-1.)/8; QPointF point(radius*std::cos(angle),radius*std::sin(angle)); if(i)star.lineTo(point);else star.moveTo(point); }
    star.closeSubpath(); checkOutline(star);
    QPainterPath reversed;
    const auto polygon=star.toFillPolygon(); reversed.moveTo(polygon.last()); for(int i=polygon.size()-2;i>=0;--i)reversed.lineTo(polygon[i]); reversed.closeSubpath(); checkOutline(reversed);

    QPainterPath bolt; bolt.moveTo(10,10); bolt.lineTo(40,20); bolt.lineTo(55,5);
    QVector<Vertex> glow;
    appendStroke(glow,bolt,QTransform(),QColor(255,128,64,128),4,0.5,1);
    require(!glow.isEmpty() && glow.size()%3==0,"glow produced incomplete triangles");
    bool transparentEdge=false,visibleCore=false;
    for(const auto &v:glow) {
        require(v.r<=v.a && v.g<=v.a && v.b<=v.a,"glow colors are not premultiplied");
        if(v.a==0)transparentEdge=true;
        if(v.a>0)visibleCore=true;
    }
    require(transparentEdge && visibleCore,"glow lost antialiased edges or core");

    QVector<Vertex> brickMesh;
    appendBrickMesh(brickMesh,10,20,34,38,.2,5,QColor("#22aa44"),QColor("#99ddaa"),1,1);
    require(!brickMesh.isEmpty() && brickMesh.size()%3==0,"GPU brick mesh is incomplete");
    bool fillVisible=false,strokeVisible=false,edgeTransparent=false;
    for(const auto &v:brickMesh) {
        require(std::isfinite(v.x) && std::isfinite(v.y),"GPU brick produced invalid geometry");
        require(v.r<=v.a && v.g<=v.a && v.b<=v.a,"GPU brick colors are not premultiplied");
        if(v.a==0) edgeTransparent=true;
        if(v.a==255 && v.r==34 && v.g==170 && v.b==68) fillVisible=true;
        if(v.a==255 && v.r==153 && v.g==221 && v.b==170) strokeVisible=true;
    }
    require(fillVisible && strokeVisible && edgeTransparent,"GPU brick lost fill, stroke or antialiasing fringe");
    // Full dense levels exceed 16-bit vertex ranges. Every triangle must survive
    // chunking, including updates that shrink or empty the layer.
    TestGeometryLayer layer; layer.setWidth(1800); layer.setHeight(900);
    QVector<Vertex> large;
    for(int i=0;i<660;++i) appendBrickMesh(large,i%44*40,i/44*40,34,38,0,5,QColor("#22aa44"),QColor("#99ddaa"),1,1);
    require(large.size()>65535,"large brick test did not exceed index range");
    layer.setFrame(large,1800,900);
    auto nodes=layer.updatePaintNode(nullptr,nullptr);
    int count=0,children=0;
    for(auto child=nodes->firstChild();child;child=child->nextSibling()) {
        auto geometry=static_cast<QSGGeometryNode *>(child)->geometry();
        require(geometry->vertexCount()<=61440 && geometry->vertexCount()%3==0,"unsafe GPU triangle chunk");
        auto points=geometry->vertexDataAsColoredPoint2D();
        for(int i=0;i<geometry->vertexCount();++i,++count) {
            require(points[i].x==large[count].x && points[i].y==large[count].y && points[i].a==large[count].a,"GPU chunk lost or reordered brick vertices");
        }
        ++children;
    }
    require(count==large.size() && children>1,"dense brick field was truncated");
    layer.setFrame(brickMesh,1800,900); nodes=layer.updatePaintNode(nodes,nullptr);
    require(nodes->childCount()==1,"shrinking GPU frame retained stale geometry");
    layer.setFrame({},1800,900); require(layer.updatePaintNode(nodes,nullptr)==nullptr,"empty GPU frame retained bricks");
    GameRenderer renderer; renderer.setWidth(80); renderer.setHeight(80);
    const QString packet=R"({"type":"frame","width":80,"height":80,"state":{},"caches":{"1":[["set","fillStyle","#00ff00"],["fillRect",32,32,16,16]]},"ops":[["set","fillStyle","#0000ff"],["fillRect",0,0,80,80],["debris",1,30,40,1,0,"#ff0000",10,true,0,[]],["cache",1],["debris",2,50,40,1,0,"#ff8800",10,true,0,[]]]})";
    renderer.acceptLine(packet);
    require(renderer.renderError().isEmpty(),"native drawing packet failed");
    QImage image(80,80,QImage::Format_ARGB32_Premultiplied); image.fill(Qt::transparent);
    QPainter painter(&image); renderer.paint(&painter); painter.end();
    require(image.pixelColor(10,10)==QColor("#0000ff"),"game background changed");
    require(image.pixelColor(25,40)==QColor("#ff0000"),"slow debris disappeared");
    require(image.pixelColor(35,40)==QColor("#00ff00"),"slow debris covered living bricks");
    require(image.pixelColor(45,40)==QColor("#ff8800"),"fast debris was hidden behind living bricks");
    // The compact batch must rasterize the same rounded/tilted brick contours
    // as the original individual canvas commands, including each palette kind.
    auto renderBricks=[&](bool batch) {
        GameRenderer r; r.setWidth(100); r.setHeight(100);
        QJsonArray cache;
        QJsonArray values;
        for(int kind=0;kind<3;++kind) {
            qreal x=10+kind*28,y=20+kind*18,angle=.2;
            values.append(x); values.append(y); values.append(20); values.append(12); values.append(angle); values.append(kind);
            if(!batch) {
                cache.append(QJsonArray{"set","fillStyle",QStringList{"#11aa22","#aa33bb","#4455dd"}[kind]});
                cache.append(QJsonArray{"set","strokeStyle",QStringList{"#99dd99","#ee99ff","#aabbff"}[kind]});
                cache.append(QJsonArray{"set","lineWidth",kind==0 ? 1 : 2});
                cache.append(QJsonArray{"save"}); cache.append(QJsonArray{"translate",x+10,y+6});
                cache.append(QJsonArray{"rotate",angle}); cache.append(QJsonArray{"beginPath"});
                cache.append(QJsonArray{"roundRect",-10,-6,20,12,3});
                cache.append(QJsonArray{"fill"}); cache.append(QJsonArray{"stroke"}); cache.append(QJsonArray{"restore"});
            }
        }
        if(batch) cache.append(QJsonArray{"bricks",values,true,3,"#11aa22","#99dd99","#aa33bb","#ee99ff","#4455dd","#aabbff"});
        QJsonObject packet{{"type","frame"},{"width",100},{"height",100},{"caches",QJsonObject{{"1",cache}}},{"ops",QJsonArray{QJsonArray{"cache",1}}}};
        r.acceptLine(QString::fromUtf8(QJsonDocument(packet).toJson(QJsonDocument::Compact)));
        require(r.renderError().isEmpty(),"brick batch failed");
        QImage result(100,100,QImage::Format_ARGB32_Premultiplied); result.fill(Qt::transparent);
        QPainter painter(&result); r.paint(&painter); painter.end(); return result;
    };
    const auto original=renderBricks(false),batched=renderBricks(true);
    int difference=0;
    for(int y=0;y<100;++y) for(int x=0;x<100;++x) {
        const auto a=original.pixelColor(x,y),b=batched.pixelColor(x,y);
        if(std::abs(a.red()-b.red())>3 || std::abs(a.green()-b.green())>3 || std::abs(a.blue()-b.blue())>3 || std::abs(a.alpha()-b.alpha())>3) ++difference;
    }
    require(difference<10,"brick batch changed rounded corners, tilt or game colors");
    // The new transport must produce identical raster output, including moving
    // cached contours, a shape change and particles crossing the brick layer.
    GameRenderer oldParticles,newParticles;
    oldParticles.setWidth(80); oldParticles.setHeight(80);
    newParticles.setWidth(80); newParticles.setHeight(80);
    auto raster=[&](GameRenderer &renderer,const QJsonObject &packet) {
        renderer.acceptLine(QString::fromUtf8(QJsonDocument(packet).toJson(QJsonDocument::Compact)));
        require(renderer.renderError().isEmpty(),"particle packet failed");
        QImage result(80,80,QImage::Format_ARGB32_Premultiplied); result.fill(Qt::transparent);
        QPainter painter(&result); renderer.paint(&painter); painter.end(); return result;
    };
    const QJsonArray points{-8,-4,3,-7,9,2,-3,8};
    for(int frame=0;frame<4;++frame) {
        const qreal x=30.25+frame*.6,radius=frame>=2 ? 8 : 10;
        QJsonObject reference=QJsonDocument::fromJson(packet.toUtf8()).object();
        reference["ops"]=QJsonArray{QJsonArray{"set","fillStyle","#0000ff"},QJsonArray{"fillRect",0,0,80,80},
            QJsonArray{"debris",1,x,40,1,0,"rgb(255,0,0)",radius,true,0,QJsonArray{}},QJsonArray{"cache",1},
            QJsonArray{"debris",2,50,40,.8,.6,"#ff8800",10,false,.4,points}};
        QJsonObject batched=reference;
        batched["ops"]=QJsonArray{QJsonArray{"set","fillStyle","#0000ff"},QJsonArray{"fillRect",0,0,80,80},
            QJsonArray{"debris-batch",QJsonObject{{"instances",QJsonArray{1,x,40,1,0}},{"colors",QJsonArray{"rgb(255,0,0)"}}}},
            QJsonArray{"cache",1},QJsonArray{"debris-batch",QJsonObject{{"instances",QJsonArray{2,50,40,.8,.6}},{"colors",QJsonArray{"#ff8800"}}}}};
        QJsonObject shapes;
        if(frame==0 || frame==2) shapes["1"]=QJsonObject{{"radius",radius},{"circle",true},{"roundness",0},{"points",QJsonArray{}}};
        if(frame==0) shapes["2"]=QJsonObject{{"radius",10},{"circle",false},{"roundness",.4},{"points",points}};
        batched["debrisShapes"]=shapes;
        require(raster(oldParticles,reference)==raster(newParticles,batched),"particle batching changed pixels or depth");
    }
    std::puts("PASS: native outline triangulation, game colors, particle transport pixel equivalence and debris depth ordering");
}
