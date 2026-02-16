package hotkey

import (
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	procRegisterHotKey   = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey = user32.NewProc("UnregisterHotKey")
	procGetMessage       = user32.NewProc("GetMessageW")
	procPeekMessage      = user32.NewProc("PeekMessageW")
)

const (
	MOD_ALT     = 0x0001
	MOD_CONTROL = 0x0002
	MOD_SHIFT   = 0x0004
	MOD_WIN     = 0x0008

	WM_HOTKEY = 0x0312
	VK_SPACE  = 0x20

	PM_REMOVE = 0x0001
)

type MSG struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

type HotkeyManager struct {
	callback func()
	quit     chan struct{}
}

func New(callback func()) *HotkeyManager {
	return &HotkeyManager{
		callback: callback,
		quit:     make(chan struct{}),
	}
}

func (h *HotkeyManager) Start() error {
	go h.listen()
	return nil
}

func (h *HotkeyManager) Stop() {
	close(h.quit)
}

func (h *HotkeyManager) listen() {
	// CRITICAL: RegisterHotKey and GetMessage must run on the same OS thread.
	// Go goroutines migrate between OS threads unless locked.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	ret, _, _ := procRegisterHotKey.Call(0, 1, MOD_ALT, VK_SPACE)
	if ret == 0 {
		return
	}
	defer procUnregisterHotKey.Call(0, 1)

	var msg MSG
	for {
		select {
		case <-h.quit:
			return
		default:
		}

		// Use PeekMessage with PM_REMOVE so we don't block forever
		// and can check the quit channel periodically
		ret, _, _ := procPeekMessage.Call(
			uintptr(unsafe.Pointer(&msg)),
			0, 0, 0,
			PM_REMOVE,
		)

		if ret == 0 {
			// No message available, sleep briefly to avoid busy-loop
			time.Sleep(50 * time.Millisecond)
			continue
		}

		if msg.Message == WM_HOTKEY {
			h.callback()
		}
	}
}
