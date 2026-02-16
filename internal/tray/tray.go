package tray

import (
	"runtime"
	"syscall"
	"unsafe"
)

var (
	shell32             = syscall.NewLazyDLL("shell32.dll")
	user32              = syscall.NewLazyDLL("user32.dll")
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	procShellNotifyIcon = shell32.NewProc("Shell_NotifyIconW")
	procCreatePopupMenu = user32.NewProc("CreatePopupMenu")
	procAppendMenu      = user32.NewProc("AppendMenuW")
	procTrackPopupMenu  = user32.NewProc("TrackPopupMenuEx")
	procCreateWindowEx  = user32.NewProc("CreateWindowExW")
	procDefWindowProc   = user32.NewProc("DefWindowProcW")
	procRegisterClassEx = user32.NewProc("RegisterClassExW")
	procGetMessage      = user32.NewProc("GetMessageW")
	procTranslateMsg    = user32.NewProc("TranslateMessage")
	procDispatchMsg     = user32.NewProc("DispatchMessageW")
	procDestroyMenu     = user32.NewProc("DestroyMenu")
	procGetCursorPos    = user32.NewProc("GetCursorPos")
	procSetForeground   = user32.NewProc("SetForegroundWindow")
	procPostMessage     = user32.NewProc("PostMessageW")
	procGetModuleHandle = kernel32.NewProc("GetModuleHandleW")
	procLoadIcon        = user32.NewProc("LoadIconW")
)

const (
	NIM_ADD    = 0x00000000
	NIM_DELETE = 0x00000002
	NIM_MODIFY = 0x00000001

	NIF_MESSAGE = 0x00000001
	NIF_ICON    = 0x00000002
	NIF_TIP     = 0x00000004

	WM_USER      = 0x0400
	WM_TRAYICON  = WM_USER + 1
	WM_COMMAND   = 0x0111
	WM_CLOSE     = 0x0010
	WM_LBUTTONUP = 0x0202
	WM_RBUTTONUP = 0x0205

	MF_STRING    = 0x0000
	MF_SEPARATOR = 0x0800

	TPM_BOTTOMALIGN = 0x0020
	TPM_LEFTALIGN   = 0x0000
	TPM_RETURNCMD   = 0x0100

	IDI_APPLICATION = 32512

	IDM_SHOW     = 1000
	IDM_SETTINGS = 1001
	IDM_QUIT     = 1002
)

type NOTIFYICONDATAW struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
}

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type POINT struct {
	X, Y int32
}

type MSG struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

type TrayIcon struct {
	onShow     func()
	onSettings func()
	onQuit     func()
	hwnd       uintptr
	nid        NOTIFYICONDATAW
	quit       chan struct{}
}

func New(onShow, onSettings, onQuit func()) *TrayIcon {
	return &TrayIcon{
		onShow:     onShow,
		onSettings: onSettings,
		onQuit:     onQuit,
		quit:       make(chan struct{}),
	}
}

func (t *TrayIcon) Start() {
	go t.run()
}

func (t *TrayIcon) Stop() {
	procShellNotifyIcon.Call(NIM_DELETE, uintptr(unsafe.Pointer(&t.nid)))
	if t.hwnd != 0 {
		procPostMessage.Call(t.hwnd, WM_CLOSE, 0, 0)
	}
}

func (t *TrayIcon) run() {
	// Window message processing MUST stay on one OS thread
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hInstance, _, _ := procGetModuleHandle.Call(0)
	className, _ := syscall.UTF16PtrFromString("BlightTray")

	wc := WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		LpfnWndProc:   syscall.NewCallback(t.wndProc),
		HInstance:     hInstance,
		LpszClassName: className,
	}

	procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))

	t.hwnd, _, _ = procCreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		0, 0,
		0, 0, 0, 0, 0, 0,
		hInstance, 0,
	)

	// Use system default icon
	hIcon, _, _ := procLoadIcon.Call(0, IDI_APPLICATION)

	t.nid = NOTIFYICONDATAW{
		CbSize:           uint32(unsafe.Sizeof(NOTIFYICONDATAW{})),
		HWnd:             t.hwnd,
		UID:              1,
		UFlags:           NIF_MESSAGE | NIF_ICON | NIF_TIP,
		UCallbackMessage: WM_TRAYICON,
		HIcon:            hIcon,
	}

	tip, _ := syscall.UTF16FromString("blight")
	copy(t.nid.SzTip[:], tip)

	procShellNotifyIcon.Call(NIM_ADD, uintptr(unsafe.Pointer(&t.nid)))

	// Standard message loop
	var msg MSG
	for {
		ret, _, _ := procGetMessage.Call(
			uintptr(unsafe.Pointer(&msg)),
			0, 0, 0,
		)
		if ret == 0 || ret == uintptr(^uintptr(0)) {
			break
		}
		procTranslateMsg.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMsg.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func (t *TrayIcon) wndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_TRAYICON:
		switch lParam {
		case WM_LBUTTONUP:
			t.onShow()
		case WM_RBUTTONUP:
			t.showContextMenu()
		}
		return 0
	case WM_COMMAND:
		menuID := wParam & 0xFFFF
		switch menuID {
		case IDM_SHOW:
			t.onShow()
		case IDM_SETTINGS:
			if t.onSettings != nil {
				t.onSettings()
			}
		case IDM_QUIT:
			t.onQuit()
		}
		return 0
	}
	ret, _, _ := procDefWindowProc.Call(hwnd, msg, wParam, lParam)
	return ret
}

func (t *TrayIcon) showContextMenu() {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return
	}

	showStr, _ := syscall.UTF16PtrFromString("Show blight")
	settingsStr, _ := syscall.UTF16PtrFromString("Settings")
	quitStr, _ := syscall.UTF16PtrFromString("Quit")

	procAppendMenu.Call(menu, MF_STRING, IDM_SHOW, uintptr(unsafe.Pointer(showStr)))
	procAppendMenu.Call(menu, MF_STRING, IDM_SETTINGS, uintptr(unsafe.Pointer(settingsStr)))
	procAppendMenu.Call(menu, MF_SEPARATOR, 0, 0)
	procAppendMenu.Call(menu, MF_STRING, IDM_QUIT, uintptr(unsafe.Pointer(quitStr)))

	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	// SetForegroundWindow is required before TrackPopupMenu or the menu won't close properly
	procSetForeground.Call(t.hwnd)

	// Use TPM_RETURNCMD so the menu item selection is returned directly.
	// This avoids issues with WM_COMMAND not being delivered.
	ret, _, _ := procTrackPopupMenu.Call(
		menu,
		TPM_LEFTALIGN|TPM_BOTTOMALIGN|TPM_RETURNCMD,
		uintptr(pt.X), uintptr(pt.Y),
		0, t.hwnd, 0,
	)

	procDestroyMenu.Call(menu)

	// Process the returned command directly
	switch ret {
	case IDM_SHOW:
		t.onShow()
	case IDM_SETTINGS:
		if t.onSettings != nil {
			t.onSettings()
		}
	case IDM_QUIT:
		t.onQuit()
	}

	// Post null message to force menu dismiss (Windows quirk)
	procPostMessage.Call(t.hwnd, 0, 0, 0)
}
