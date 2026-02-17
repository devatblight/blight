package apps

import (
	"blight/internal/debug"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var (
	ole32Lnk              = syscall.NewLazyDLL("ole32.dll")
	procCoCreateInstance  = ole32Lnk.NewProc("CoCreateInstance")
	procCoInitializeExLnk = ole32Lnk.NewProc("CoInitializeEx")
)

// COM GUIDs for IShellLink
var (
	CLSID_ShellLink = syscall.GUID{
		Data1: 0x00021401,
		Data2: 0x0000,
		Data3: 0x0000,
		Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
	}
	IID_IShellLinkW = syscall.GUID{
		Data1: 0x000214F9,
		Data2: 0x0000,
		Data3: 0x0000,
		Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
	}
	IID_IPersistFile = syscall.GUID{
		Data1: 0x0000010B,
		Data2: 0x0000,
		Data3: 0x0000,
		Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
	}
)

const (
	CLSCTX_INPROC_SERVER = 0x1
	STGM_READ            = 0x00000000
)

// ResolveLnkTarget resolves a .lnk shortcut to its target executable path.
// Uses COM IShellLink interface — no shell windows opened.
func ResolveLnkTarget(lnkPath string) string {
	log := debug.Get()
	log.Debug("lnk: resolving shortcut", map[string]interface{}{"path": lnkPath})

	procCoInitializeExLnk.Call(0, 0)

	var shellLink uintptr
	hr, _, lastErr := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&CLSID_ShellLink)),
		0,
		CLSCTX_INPROC_SERVER,
		uintptr(unsafe.Pointer(&IID_IShellLinkW)),
		uintptr(unsafe.Pointer(&shellLink)),
	)
	if hr != 0 || shellLink == 0 {
		log.Error("lnk: CoCreateInstance failed", map[string]interface{}{
			"hr":    fmt.Sprintf("0x%08X", hr),
			"error": fmt.Sprintf("%v", lastErr),
		})
		return ""
	}
	defer comRelease(shellLink)

	// QueryInterface for IPersistFile (vtable[0] = QueryInterface)
	var persistFile uintptr
	shellLinkVtable := getVtable(shellLink)
	hr, _, lastErr = syscall.SyscallN(shellLinkVtable[0], shellLink,
		uintptr(unsafe.Pointer(&IID_IPersistFile)),
		uintptr(unsafe.Pointer(&persistFile)))
	if hr != 0 || persistFile == 0 {
		log.Error("lnk: QueryInterface(IPersistFile) failed", map[string]interface{}{
			"hr": fmt.Sprintf("0x%08X", hr),
		})
		return ""
	}
	defer comRelease(persistFile)

	// IPersistFile::Load — vtable index 5 (IUnknown[3] + IPersist::GetClassID[1] + Load[1])
	lnkPathWide, _ := syscall.UTF16PtrFromString(lnkPath)
	persistFileVtable := getVtable(persistFile)
	hr, _, lastErr = syscall.SyscallN(persistFileVtable[5], persistFile,
		uintptr(unsafe.Pointer(lnkPathWide)), STGM_READ)
	if hr != 0 {
		log.Error("lnk: IPersistFile::Load failed", map[string]interface{}{
			"hr":   fmt.Sprintf("0x%08X", hr),
			"path": lnkPath,
		})
		return ""
	}

	// IShellLinkW::GetPath — vtable index 3 (IUnknown[3] + GetPath[0])
	pathBuffer := make([]uint16, 260)
	hr, _, _ = syscall.SyscallN(shellLinkVtable[3], shellLink,
		uintptr(unsafe.Pointer(&pathBuffer[0])), 260, 0, 0)
	if hr != 0 {
		log.Error("lnk: IShellLinkW::GetPath failed", map[string]interface{}{
			"hr": fmt.Sprintf("0x%08X", hr),
		})
		return ""
	}

	targetPath := syscall.UTF16ToString(pathBuffer)
	if targetPath == "" {
		log.Debug("lnk: resolved target is empty", map[string]interface{}{"lnk": lnkPath})
		return ""
	}

	log.Debug("lnk: resolved target", map[string]interface{}{
		"lnk":    lnkPath,
		"target": targetPath,
	})
	return targetPath
}

// FindAppIcon looks for a .ico file in the target app's directory.
// Many apps (like Discord) ship an app.ico alongside their executable.
// Avoids false positives from system directories like system32.
func FindAppIcon(targetPath string) string {
	log := debug.Get()

	if targetPath == "" {
		return ""
	}

	targetDir := filepath.Dir(targetPath)
	targetName := strings.TrimSuffix(filepath.Base(targetPath), filepath.Ext(filepath.Base(targetPath)))
	targetNameLower := strings.ToLower(targetName)

	// Skip system directories that contain random .ico files (e.g., OneDrive.ico)
	skipDirs := []string{"system32", "syswow64", "sysnative", "windows"}

	// Check the target directory and one level up
	dirsToCheck := []string{targetDir, filepath.Dir(targetDir)}

	// Preferred .ico names (in priority order)
	preferredNames := []string{
		"app.ico", "icon.ico",
		targetNameLower + ".ico",
	}

	for _, dir := range dirsToCheck {
		dirLower := strings.ToLower(dir)

		// Skip system directories
		isSystemDir := false
		for _, skip := range skipDirs {
			if strings.Contains(dirLower, `\`+skip+`\`) || strings.HasSuffix(dirLower, `\`+skip) {
				isSystemDir = true
				break
			}
		}
		if isSystemDir {
			log.Debug("lnk: skipping system dir for .ico search", map[string]interface{}{
				"dir": dir,
			})
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		// First pass: look for preferred .ico names
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			nameLower := strings.ToLower(entry.Name())
			for _, preferred := range preferredNames {
				if nameLower == preferred {
					icoPath := filepath.Join(dir, entry.Name())
					log.Debug("lnk: found preferred .ico file", map[string]interface{}{
						"icoPath": icoPath,
						"target":  targetPath,
					})
					return icoPath
				}
			}
		}

		// Second pass: if directory is small (not a system dump), take any .ico
		if len(entries) < 50 {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				if strings.EqualFold(filepath.Ext(entry.Name()), ".ico") {
					icoPath := filepath.Join(dir, entry.Name())
					log.Debug("lnk: found .ico file (fallback)", map[string]interface{}{
						"icoPath": icoPath,
						"target":  targetPath,
					})
					return icoPath
				}
			}
		}
	}

	log.Debug("lnk: no suitable .ico file found", map[string]interface{}{
		"target": targetPath,
	})
	return ""
}

// getVtable reads the COM vtable from an interface pointer.
func getVtable(comObject uintptr) []uintptr {
	vtablePtr := *(*uintptr)(unsafe.Pointer(comObject))
	return (*[20]uintptr)(unsafe.Pointer(vtablePtr))[:]
}

// comRelease calls IUnknown::Release (vtable[2]) on a COM object.
func comRelease(comObject uintptr) {
	vtable := getVtable(comObject)
	syscall.SyscallN(vtable[2], comObject)
}
