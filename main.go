package main

import (
	"embed"
	"fmt"
	"os"
	"os/exec"

	"blight/internal/debug"
	"blight/internal/installer"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

var Version = "0.0.1"

func main() {
	log := debug.Init()
	defer log.Close()
	defer log.RecoverPanic("main")

	// Settings window mode — launched by the tray or keyboard shortcut.
	// Runs as a proper non-frameless window, no hotkey, no tray icon.
	if len(os.Args) > 1 && os.Args[1] == "--settings" {
		runSettingsWindow(log)
		return
	}

	log.Info("blight starting up", map[string]interface{}{"version": Version})

	if !log.Enabled() {
		installed, err := installer.IsInstalled()
		if err != nil {
			log.Error("failed to check install status", map[string]interface{}{"error": err.Error()})
		} else if !installed {
			log.Info("not installed, attempting to install...")
			newPath, err := installer.Install()
			if err != nil {
				log.Error("installation failed", map[string]interface{}{"error": err.Error()})
			} else {
				log.Info("installed successfully, relaunching", map[string]interface{}{"path": newPath})
				cmd := exec.Command(newPath)
				if err := cmd.Start(); err != nil {
					log.Error("failed to launch installed app", map[string]interface{}{"error": err.Error()})
				} else {
					os.Exit(0)
				}
			}
		}
	}

	if log.Enabled() {
		port, err := debug.StartConsole(log)
		if err != nil {
			log.Error("failed to start debug console", map[string]interface{}{"error": err.Error()})
		} else {
			log.Info("debug console started", map[string]interface{}{"port": port, "url": fmt.Sprintf("http://127.0.0.1:%d", port)})
			debug.OpenInBrowser(port)
		}
	}

	app := NewApp(Version)
	log.Info("app instance created")

	err := wails.Run(&options.App{
		Title:            "blight",
		Width:            750,
		Height:           500,
		DisableResize:    true,
		Frameless:        true,
		AlwaysOnTop:      true,
		StartHidden:      false,
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent:              true,
			WindowIsTranslucent:               true,
			DisableWindowIcon:                 false,
			DisableFramelessWindowDecorations: true,
			Theme:                             windows.SystemDefault,
			BackdropType:                      windows.Mica,
		},
	})

	if err != nil {
		log.Fatal("wails.Run failed", map[string]interface{}{"error": err.Error()})
		println("Error:", err.Error())
	}

	log.Info("blight shutting down gracefully")
}

func runSettingsWindow(log *debug.Logger) {
	log.Info("running in settings-window mode")

	app := NewSettingsApp(Version)

	err := wails.Run(&options.App{
		Title:            "blight Settings",
		Width:            680,
		Height:           560,
		DisableResize:    false,
		MinWidth:         480,
		MinHeight:        400,
		Frameless:        true,
		AlwaysOnTop:      false,
		StartHidden:      false,
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.settingsShutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent:              true,
			WindowIsTranslucent:               true,
			DisableFramelessWindowDecorations: true,
			Theme:                             windows.SystemDefault,
			BackdropType:                      windows.Mica,
		},
	})

	if err != nil {
		log.Fatal("settings wails.Run failed", map[string]interface{}{"error": err.Error()})
	}
}
