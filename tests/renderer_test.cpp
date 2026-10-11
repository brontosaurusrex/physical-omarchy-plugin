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
int main(int argc,char **argv) {
    QGuiApplication application(argc,argv);
    QPainterPath circle; circle.addEllipse(QPointF(0,0),12,12); checkOutline(circle);
    QPainterPath rounded; rounded.addRoundedRect(QRectF(-12,-5,24,10),3,3); checkOutline(rounded);
    QPainterPath star;
    for(int i=0;i<16;++i) { qreal radius=i%2 ? 4 : 12, angle=i*std::acos(-1.)/8; QPointF point(radius*std::cos(angle),radius*std::sin(angle)); if(i)star.lineTo(point);else star.moveTo(point); }
    star.closeSubpath(); checkOutline(star);
    QPainterPath reversed;
    const auto polygon=star.toFillPolygon(); reversed.moveTo(polygon.last()); for(int i=polygon.size()-2;i>=0;--i)reversed.lineTo(polygon[i]); reversed.closeSubpath(); checkOutline(reversed);

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
    std::puts("PASS: native outline triangulation, game colors, and debris depth ordering");
}
