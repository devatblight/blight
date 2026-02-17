package hotkey

import (
	"blight/internal/debug"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procSetWindowsHookEx    = user32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	procCallNextHookEx      = user32.NewProc("CallNextHookEx")
	procPeekMessage         = user32.NewProc("PeekMessageW")
	procGetAsyncKeyState    = user32.NewProc("GetAsyncKeyState")
)

const (
	WH_KEYBOARD_LL = 13
	WM_KEYDOWN     = 0x0100
	WM_KEYUP       = 0x0101
	WM_SYSKEYDOWN  = 0x0104
	WM_SYSKEYUP    = 0x0105

	VK_SPACE = 0x20
	VK_MENU  = 0x12 // Alt key
)

// KBDLLHOOKSTRUCT contains info about a low-level keyboard event.
type KBDLLHOOKSTRUCT struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type MSG struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

type HotkeyManager struct {
	callback   func()
	quit       chan struct{}
	hookHandle uintptr
	altPressed atomic.Bool
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
	log := debug.Get()

	// Low-level hooks MUST stay on one OS thread with an active message pump.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	log.Info("hotkey: installing low-level keyboard hook for Alt+Space")

	// Create the hook callback
	hookCallback := func(nCode int, wParam uintptr, lParam uintptr) uintptr {
		if nCode >= 0 {
			kbData := (*KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))

			switch wParam {
			case WM_SYSKEYDOWN:
				if kbData.VkCode == VK_SPACE {
					// Check if Alt is held down
					altState, _, _ := procGetAsyncKeyState.Call(VK_MENU)
					if altState&0x8000 != 0 {
						log.Info("hotkey: Alt+Space pressed")
						// Run callback in a separate goroutine to avoid blocking the hook
						go h.callback()
						// Return 1 to consume this keypress — prevents Windows from
						// showing the system menu
						return 1
					}
				}
			}
		}

		// Pass to next hook in the chain
		ret, _, _ := procCallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
		return ret
	}

	hookProc := syscall.NewCallback(hookCallback)

	hookHandle, _, hookErr := procSetWindowsHookEx.Call(
		WH_KEYBOARD_LL,
		hookProc,
		0, // hMod — 0 for global hooks
		0, // dwThreadId — 0 for all threads
	)

	if hookHandle == 0 {
		log.Error("hotkey: SetWindowsHookEx FAILED", map[string]interface{}{
			"error": hookErr.Error(),
		})
		return
	}

	h.hookHandle = hookHandle
	log.Info("hotkey: keyboard hook installed successfully")

	// Run a message pump — required for low-level hooks to receive events.
	// Without this, Windows won't deliver WH_KEYBOARD_LL callbacks.
	var msg MSG
	for {
		select {
		case <-h.quit:
			procUnhookWindowsHookEx.Call(h.hookHandle)
			log.Info("hotkey: hook removed, shutting down")
			return
		default:
		}

		ret, _, _ := procPeekMessage.Call(
			uintptr(unsafe.Pointer(&msg)),
			0, 0, 0,
			1, // PM_REMOVE
		)
		if ret != 0 {
			// Dispatch any messages to keep the pump alive
			continue
		}
		time.Sleep(10 * time.Millisecond)
	}
}
