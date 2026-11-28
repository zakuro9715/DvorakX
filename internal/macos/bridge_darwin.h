#include <ApplicationServices/ApplicationServices.h>
#include <stdint.h>

typedef struct {
    CFMachPortRef tap;
    CFRunLoopSourceRef source;
    CFRunLoopRef loop;
    uintptr_t handle;
} DvorakTap;

int dvorak_open(DvorakTap *tap);
void dvorak_run(DvorakTap *tap);
void dvorak_stop(CFRunLoopRef loop);
void dvorak_close(DvorakTap *tap);
int dvorak_emit(CGEventTapProxy proxy, int code, int up, int repeat, uint64_t flags);
