package main

/*
#include <stdlib.h>
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
void init_tray(const char* title);
void update_tray_status(int connected);
*/
import "C"
import (
	"unsafe"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Exported callbacks for Objective-C
//export tray_callback_show
func tray_callback_show() {
	if wailsApp != nil && wailsApp.ctx != nil {
		runtime.WindowShow(wailsApp.ctx)
		runtime.WindowUnminimise(wailsApp.ctx)
	}
}

//export tray_callback_quit
func tray_callback_quit() {
	if wailsApp != nil {
		// Attempt to disconnect if connected
		wailsApp.Disconnect()
		if wailsApp.ctx != nil {
			runtime.Quit(wailsApp.ctx)
		}
	}
}

func initTray(title string) {
	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))
	C.init_tray(cTitle)
}

func updateTrayStatus(connected bool) {
	cConnected := C.int(0)
	if connected {
		cConnected = C.int(1)
	}
	C.update_tray_status(cConnected)
}
