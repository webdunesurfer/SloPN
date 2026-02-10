//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	shell32  = windows.NewLazySystemDLL("shell32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procDestroyMenu         = user32.NewProc("DestroyMenu")
	procLoadImageW          = user32.NewProc("LoadImageW")
	procCreatePopupMenu     = user32.NewProc("CreatePopupMenu")
	procAppendMenuW         = user32.NewProc("AppendMenuW")
	procTrackPopupMenu      = user32.NewProc("TrackPopupMenu")
	procGetCursorPos        = user32.NewProc("GetCursorPos")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procShellNotifyIconW    = shell32.NewProc("Shell_NotifyIconW")
)

const (
	WM_DESTROY       = 0x0002
	WM_COMMAND       = 0x0111
	WM_USER          = 0x0400
	WM_TRAYICON      = WM_USER + 1
	NIM_ADD          = 0x0000
	NIM_MODIFY       = 0x0001
	NIM_DELETE       = 0x0002
	NIF_MESSAGE      = 0x0001
	NIF_ICON         = 0x0002
	NIF_TIP          = 0x0004
	IMAGE_ICON       = 1
	LR_LOADFROMFILE  = 0x0010
	LR_DEFAULTSIZE   = 0x0040
	MF_STRING        = 0x0000
	TPM_RETURNCMD    = 0x0100
	TPM_NONOTIFY     = 0x0080
	WM_RBUTTONUP     = 0x0205
	WM_LBUTTONUP     = 0x0202
)

type WNDCLASSEX struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   windows.Handle
	Icon       windows.Handle
	Cursor     windows.Handle
	Background windows.Handle
	MenuName   *uint16
	ClassName  *uint16
	IconSm     windows.Handle
}

type NOTIFYICONDATA struct {
	Size             uint32
	Wnd              windows.Handle
	ID               uint32
	Flags            uint32
	CallbackMessage  uint32
	Icon             windows.Handle
	Tip              [128]uint16
	State            uint32
	StateMask        uint32
	Info             [256]uint16
	Timeout          uint32
	InfoTitle        [64]uint16
	InfoFlags        uint32
	GuidItem         windows.GUID
	BalloonIcon      windows.Handle
}

type POINT struct {
	X int32
	Y int32
}

var (
	hTrayWnd windows.Handle
	hIcon    windows.Handle
	nid      NOTIFYICONDATA
	menuCmds = make(map[uintptr]func())
)

func initTray(ctx context.Context) {
	go runTrayLoop(ctx)
}

func updateTrayStatus(connected bool) {
	// TODO: Change icon here if needed
}

func runTrayLoop(ctx context.Context) {
	// 1. Prepare Icon File
	// Win32 LoadImage needs a file path. We dump the embedded icon to temp.
	tmpIcon := filepath.Join(os.TempDir(), "slopn_tray.ico")
	if err := os.WriteFile(tmpIcon, icon, 0644); err != nil {
		fmt.Println("Failed to save icon:", err)
		return
	}
	defer os.Remove(tmpIcon)

	// 2. Register Window Class
	className, _ := windows.UTF16PtrFromString("SloPNTrayClass")
	hInstance := windows.Handle(0) // GetModuleHandle(NULL)

	wndProc := syscall.NewCallback(wndProcCallback)

	wc := WNDCLASSEX{
		Size:      uint32(unsafe.Sizeof(WNDCLASSEX{})),
		Instance:  hInstance,
		ClassName: className,
		WndProc:   wndProc,
	}

	if ret, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); ret == 0 {
		fmt.Println("RegisterClassEx failed")
		return
	}

	// 3. Create Hidden Window
	hTrayWndVal, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(className)),
		0, 0, 0, 0, 0,
		0, 0, uintptr(hInstance), 0,
	)
	hTrayWnd = windows.Handle(hTrayWndVal)

	if hTrayWnd == 0 {
		fmt.Println("CreateWindowEx failed")
		return
	}

	// 4. Load Icon
	iconPath, _ := windows.UTF16PtrFromString(tmpIcon)
	hIconVal, _, _ := procLoadImageW.Call(
		0,
		uintptr(unsafe.Pointer(iconPath)),
		IMAGE_ICON,
		0, 0,
		LR_LOADFROMFILE|LR_DEFAULTSIZE,
	)
	hIcon = windows.Handle(hIconVal)

	// 5. Add Tray Icon
	nid.Size = uint32(unsafe.Sizeof(nid))
	nid.Wnd = hTrayWnd
	nid.ID = 1
	nid.Flags = NIF_ICON | NIF_MESSAGE | NIF_TIP
	nid.CallbackMessage = WM_TRAYICON
	nid.Icon = hIcon
	copy(nid.Tip[:], windows.StringToUTF16("SloPN VPN"))

	procShellNotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(&nid)))

	// 6. Define Menu Actions
	menuCmds[1001] = func() {
		runtime.WindowShow(ctx)
		runtime.WindowUnminimise(ctx)
	}
	menuCmds[1002] = func() {
		wailsApp.ShowAbout()
	}
	menuCmds[1003] = func() {
		wailsApp.Disconnect()
		runtime.Quit(ctx)
	}

	// 7. Message Loop
	var msg struct {
		Hwnd    windows.Handle
		Message uint32
		WParam  uintptr
		LParam  uintptr
		Time    uint32
		Pt      POINT
	}

	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if ret == 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}

	// Cleanup
	procShellNotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&nid)))
}

func wndProcCallback(hwnd windows.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_TRAYICON:
		if lParam == WM_LBUTTONUP {
			// Left click: Show app
			if action, ok := menuCmds[1001]; ok {
				action()
			}
		} else if lParam == WM_RBUTTONUP {
			// Right click: Show menu
			showContextMenu()
		}
		return 0
	case WM_COMMAND:
		id := wParam & 0xffff
		if action, ok := menuCmds[id]; ok {
			action()
		}
		return 0
	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	}

	ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

func showContextMenu() {
	hMenu, _, _ := procCreatePopupMenu.Call()
	
	insertMenu(hMenu, "Show SloPN", 1001)
	insertMenu(hMenu, "About", 1002)
	// Separator? Windows AppendMenu with MF_SEPARATOR (0x800)
	procAppendMenuW.Call(hMenu, 0x800, 0, 0)
	insertMenu(hMenu, "Quit", 1003)

	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	
	// Required for menu to disappear when clicking outside
	procSetForegroundWindow.Call(uintptr(hTrayWnd))
	
	procTrackPopupMenu.Call(hMenu, TPM_RETURNCMD|TPM_NONOTIFY, uintptr(pt.X), uintptr(pt.Y), 0, uintptr(hTrayWnd), 0)
	procDestroyMenu.Call(hMenu)
}

func insertMenu(hMenu uintptr, text string, id uintptr) {
	utf16, _ := windows.UTF16PtrFromString(text)
	procAppendMenuW.Call(hMenu, MF_STRING, id, uintptr(unsafe.Pointer(utf16)))
}
