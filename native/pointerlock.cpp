#include "pointerlock.h"
#include <QGuiApplication>
#include <QQuickWindow>
#include <QtGui/qguiapplication_platform.h>
#include <cstring>
#include "pointer-constraints-client.h"
#include "relative-pointer-client.h"

PointerLock::PointerLock(QQuickItem *parent):QQuickItem(parent) {
    dispatchTimer.setInterval(4);
    connect(&dispatchTimer,&QTimer::timeout,this,[this] {
        if(queue) wl_display_dispatch_queue_pending(display,queue);
    });
    connect(this,&QQuickItem::windowChanged,this,[this](QQuickWindow *next) {
        release();
        if(next) {
            connect(next,&QWindow::activeChanged,this,[this,next] { if(!next->isActive()) release(); });
            connect(next,&QWindow::visibleChanged,this,[this](bool visible) { if(!visible)release(); });
        }
    });
}
PointerLock::~PointerLock() {
    release();
    if(pointer) wl_pointer_release(pointer);
    if(seat) wl_seat_release(seat);
    if(relativeManager) zwp_relative_pointer_manager_v1_destroy(relativeManager);
    if(constraints) zwp_pointer_constraints_v1_destroy(constraints);
    if(registry) wl_registry_destroy(registry);
    if(display) wl_display_flush(display);
    if(queue) wl_event_queue_destroy(queue);
}
void PointerLock::setLocked(bool value) { if(active==value)return; active=value; emit lockedChanged(); }
void PointerLock::global(void *data,wl_registry *registry,uint32_t name,const char *interface,uint32_t version) {
    auto self=static_cast<PointerLock *>(data);
    if(std::strcmp(interface,zwp_pointer_constraints_v1_interface.name)==0)
        self->constraints=static_cast<zwp_pointer_constraints_v1 *>(wl_registry_bind(registry,name,&zwp_pointer_constraints_v1_interface,1));
    else if(std::strcmp(interface,zwp_relative_pointer_manager_v1_interface.name)==0)
        self->relativeManager=static_cast<zwp_relative_pointer_manager_v1 *>(wl_registry_bind(registry,name,&zwp_relative_pointer_manager_v1_interface,1));
    else if(std::strcmp(interface,wl_seat_interface.name)==0 && version>=5 && !self->seat)
        self->seat=static_cast<wl_seat *>(wl_registry_bind(registry,name,&wl_seat_interface,5));
}
void PointerLock::globalRemoved(void *,wl_registry *,uint32_t) {}
bool PointerLock::initialize() {
    if(registry)return supported();
#if QT_CONFIG(wayland)
    if(!QGuiApplication::platformName().startsWith("wayland"))return false;
    auto native=qGuiApp->nativeInterface<QNativeInterface::QWaylandApplication>();
    if(!native)return false;
    display=native->display();
    queue=wl_display_create_queue(display);
    registry=wl_display_get_registry(display);
    wl_proxy_set_queue(reinterpret_cast<wl_proxy *>(registry),queue);
    static const wl_registry_listener listener={global,globalRemoved};
    wl_registry_add_listener(registry,&listener,this);
    if(wl_display_roundtrip_queue(display,queue)<0)return false;
    emit supportedChanged();
    return supported();
#else
    return false;
#endif
}
bool PointerLock::lock() {
    if(lockedPointer)return true;
    if(!window() || !window()->isVisible() || !initialize())return false;
    // Qt 6.9+ returns the wl_surface as the Wayland window's native winId.
    auto surface=reinterpret_cast<wl_surface *>(window()->winId());
    if(!surface)return false;
    pointer=wl_seat_get_pointer(seat);
    auto native=qGuiApp->nativeInterface<QNativeInterface::QWaylandApplication>();
    auto region=wl_compositor_create_region(native->compositor());
    const QPointF origin=mapToScene(QPointF(0,0));
    wl_region_add(region,qRound(origin.x()),qRound(origin.y()),qRound(width()),qRound(height()));
    relativePointer=zwp_relative_pointer_manager_v1_get_relative_pointer(relativeManager,pointer);
    static const zwp_relative_pointer_v1_listener motion={moved};
    zwp_relative_pointer_v1_add_listener(relativePointer,&motion,this);
    lockedPointer=zwp_pointer_constraints_v1_lock_pointer(constraints,surface,pointer,region,ZWP_POINTER_CONSTRAINTS_V1_LIFETIME_PERSISTENT);
    wl_region_destroy(region);
    static const zwp_locked_pointer_v1_listener state={activated,deactivated};
    zwp_locked_pointer_v1_add_listener(lockedPointer,&state,this);
    wl_display_flush(display);
    dispatchTimer.start();
    emit lockedChanged();
    return true;
}
void PointerLock::release() {
    dispatchTimer.stop();
    const bool changed=lockedPointer || active;
    if(lockedPointer) { zwp_locked_pointer_v1_destroy(lockedPointer); lockedPointer=nullptr; }
    if(relativePointer) { zwp_relative_pointer_v1_destroy(relativePointer); relativePointer=nullptr; }
    // Stop receiving absolute events while unlocked; an undispatched private
    // pointer queue would otherwise grow during ordinary desktop use.
    if(pointer) { wl_pointer_release(pointer); pointer=nullptr; }
    active=false;
    if(display)wl_display_flush(display);
    if(changed)emit lockedChanged();
}
void PointerLock::activated(void *data,zwp_locked_pointer_v1 *) { static_cast<PointerLock *>(data)->setLocked(true); }
void PointerLock::deactivated(void *data,zwp_locked_pointer_v1 *) {
    auto self=static_cast<PointerLock *>(data);
    self->setLocked(false);
    QMetaObject::invokeMethod(self,[self] { self->release(); },Qt::QueuedConnection);
}
void PointerLock::moved(void *data,zwp_relative_pointer_v1 *,uint32_t,uint32_t,wl_fixed_t dx,wl_fixed_t dy,wl_fixed_t,wl_fixed_t) {
    auto self=static_cast<PointerLock *>(data);
    if(self->active)emit self->relativeMotion(wl_fixed_to_double(dx),wl_fixed_to_double(dy));
}
#include "pointerlock.moc"
