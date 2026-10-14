#pragma once
#include <QQuickItem>
#include <QTimer>
#include <wayland-client.h>

struct zwp_pointer_constraints_v1;
struct zwp_relative_pointer_manager_v1;
struct zwp_locked_pointer_v1;
struct zwp_relative_pointer_v1;

// Pointer lock uses the compositor's protocols on Qt's existing connection;
// no global mouse warping or private Qt headers are needed.
class PointerLock : public QQuickItem {
    Q_OBJECT
    Q_PROPERTY(bool locked READ locked NOTIFY lockedChanged)
    Q_PROPERTY(bool requested READ requested NOTIFY lockedChanged)
    Q_PROPERTY(bool supported READ supported NOTIFY supportedChanged)
    wl_display *display=nullptr;
    wl_event_queue *queue=nullptr;
    wl_registry *registry=nullptr;
    wl_seat *seat=nullptr;
    wl_pointer *pointer=nullptr;
    zwp_pointer_constraints_v1 *constraints=nullptr;
    zwp_relative_pointer_manager_v1 *relativeManager=nullptr;
    zwp_locked_pointer_v1 *lockedPointer=nullptr;
    zwp_relative_pointer_v1 *relativePointer=nullptr;
    QTimer dispatchTimer;
    QTimer resizeTimer;
    QMetaObject::Connection activeConnection, visibleConnection, frameConnection;
    QRect regionBounds;
    bool active=false;
    bool initialize();
    void setLocked(bool value);
    void updateRegion();
protected:
    void updatePolish() override;
public:
    PointerLock(QQuickItem *parent=nullptr);
    ~PointerLock() override;
    bool locked() const { return active; }
    bool requested() const { return lockedPointer!=nullptr; }
    bool supported() const { return constraints && relativeManager && seat; }
    Q_INVOKABLE bool lock();
    Q_INVOKABLE void release();
    static void global(void *data,wl_registry *registry,uint32_t name,const char *interface,uint32_t version);
    static void globalRemoved(void *,wl_registry *,uint32_t);
    static void activated(void *data,zwp_locked_pointer_v1 *);
    static void deactivated(void *data,zwp_locked_pointer_v1 *);
    static void moved(void *data,zwp_relative_pointer_v1 *,uint32_t,uint32_t,wl_fixed_t dx,wl_fixed_t dy,wl_fixed_t,wl_fixed_t);
signals:
    void lockedChanged();
    void supportedChanged();
    void relativeMotion(qreal dx,qreal dy);
};
