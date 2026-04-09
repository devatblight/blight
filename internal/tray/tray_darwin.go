//go:build darwin

package tray

// TrayIcon is a no-op on macOS. getlantern/systray defines its own
// AppDelegate Objective-C class, which conflicts with Wails' AppDelegate
// at link time. A native macOS status-bar icon would require a CGO
// implementation that hooks into the existing Wails/Cocoa run loop instead
// of creating a second one.
type TrayIcon struct {
	onShow     func()
	onSettings func()
	onQuit     func()
}

func New(onShow, onSettings, onQuit func()) *TrayIcon {
	return &TrayIcon{
		onShow:     onShow,
		onSettings: onSettings,
		onQuit:     onQuit,
	}
}

func (t *TrayIcon) Start() {}
func (t *TrayIcon) Stop()  {}
