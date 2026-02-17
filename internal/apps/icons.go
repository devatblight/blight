package apps

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

var (
	shell32            = syscall.NewLazyDLL("shell32.dll")
	user32             = syscall.NewLazyDLL("user32.dll")
	gdi32              = syscall.NewLazyDLL("gdi32.dll")
	ole32              = syscall.NewLazyDLL("ole32.dll")
	procSHGetFileInfo  = shell32.NewProc("SHGetFileInfoW")
	procSHGetImageList = shell32.NewProc("SHGetImageList")
	procDestroyIcon    = user32.NewProc("DestroyIcon")
	procGetIconInfo    = user32.NewProc("GetIconInfo")
	procGetDIBits      = gdi32.NewProc("GetDIBits")
	procCreateCompatDC = gdi32.NewProc("CreateCompatibleDC")
	procDeleteDC       = gdi32.NewProc("DeleteDC")
	procDeleteObject   = gdi32.NewProc("DeleteObject")
	procGetObject      = gdi32.NewProc("GetObjectW")
	procCoInitializeEx = ole32.NewProc("CoInitializeEx")
)

const (
	shgfiIcon         = 0x000000100
	shgfiSmallIcon    = 0x000000001
	shgfiLargeIcon    = 0x000000000
	shgfiSYSICONINDEX = 0x000004000
	biRGB             = 0

	// Image list sizes
	SHIL_LARGE      = 0 // 32x32
	SHIL_SMALL      = 1 // 16x16
	SHIL_EXTRALARGE = 2 // 48x48
	SHIL_JUMBO      = 4 // 256x256 (Vista+)
)

// IID_IImageList GUID
var IID_IImageList = syscall.GUID{
	Data1: 0x46EB5926,
	Data2: 0x582E,
	Data3: 0x4017,
	Data4: [8]byte{0x9F, 0xDF, 0xE8, 0x99, 0x8D, 0xAA, 0x09, 0x50},
}

type shFileInfo struct {
	HIcon         syscall.Handle
	IIcon         int32
	DwAttributes  uint32
	SzDisplayName [260]uint16
	SzTypeName    [80]uint16
}

type iconInfo struct {
	FIcon    int32
	XHotspot int32
	YHotspot int32
	HbmMask  syscall.Handle
	HbmColor syscall.Handle
}

type bitmap struct {
	Type       int32
	Width      int32
	Height     int32
	WidthBytes int32
	Planes     uint16
	BitsPixel  uint16
	Bits       uintptr
}

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

var (
	iconCache   sync.Map
	comInitOnce sync.Once
)

func GetIconBase64(path string) string {
	if cached, ok := iconCache.Load(path); ok {
		return cached.(string)
	}

	iconPath := path

	// For .lnk files, resolve the target to avoid the shortcut arrow overlay.
	// 1. Try to find a .ico file in the target app's directory (highest quality)
	// 2. Otherwise extract from the target exe (not the .lnk)
	if strings.HasSuffix(strings.ToLower(path), ".lnk") {
		targetPath := ResolveLnkTarget(path)
		if targetPath != "" {
			// Check for a .ico file in the app directory
			icoPath := FindAppIcon(targetPath)
			if icoPath != "" {
				iconPath = icoPath
			} else {
				iconPath = targetPath
			}
		}
	}

	data := extractIconHQ(iconPath)
	if data == "" {
		data = extractIcon(iconPath)
	}
	iconCache.Store(path, data)
	return data
}

// extractIconHQ extracts a 48x48 (extra-large) icon via SHGetImageList.
// This produces crisp high-quality icons like Flow Launcher / Raycast.
func extractIconHQ(path string) string {
	comInitOnce.Do(func() {
		procCoInitializeEx.Call(0, 0)
	})

	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return ""
	}

	// Get the system icon index for this file
	var sfi shFileInfo
	ret, _, _ := procSHGetFileInfo.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0,
		uintptr(unsafe.Pointer(&sfi)),
		unsafe.Sizeof(sfi),
		shgfiSYSICONINDEX,
	)
	if ret == 0 {
		return ""
	}
	iconIndex := sfi.IIcon

	// Get the extra-large (48x48) system image list
	var imageList uintptr
	hr, _, _ := procSHGetImageList.Call(
		SHIL_EXTRALARGE,
		uintptr(unsafe.Pointer(&IID_IImageList)),
		uintptr(unsafe.Pointer(&imageList)),
	)
	if hr != 0 || imageList == 0 {
		return ""
	}

	// IImageList::GetIcon is the 9th method in the vtable (index 8, 0-based)
	// https://learn.microsoft.com/en-us/windows/win32/api/commoncontrols/nf-commoncontrols-iimagelist-geticon
	vtable := *(*[20]uintptr)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(imageList))))
	getIconFn := vtable[9] // GetIcon is at index 9

	var hIcon uintptr
	// ILD_TRANSPARENT = 1
	syscall.SyscallN(getIconFn, imageList, uintptr(iconIndex), 1, uintptr(unsafe.Pointer(&hIcon)))
	if hIcon == 0 {
		return ""
	}
	defer procDestroyIcon.Call(hIcon)

	return hIconToPngBase64(syscall.Handle(hIcon))
}

// extractIcon is the fallback — uses SHGetFileInfo with SHGFI_LARGEICON (32x32).
func extractIcon(path string) string {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return ""
	}

	var sfi shFileInfo
	ret, _, _ := procSHGetFileInfo.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0,
		uintptr(unsafe.Pointer(&sfi)),
		unsafe.Sizeof(sfi),
		shgfiIcon|shgfiLargeIcon,
	)

	if ret == 0 || sfi.HIcon == 0 {
		return ""
	}
	defer procDestroyIcon.Call(uintptr(sfi.HIcon))

	return hIconToPngBase64(sfi.HIcon)
}

func hIconToPngBase64(hIcon syscall.Handle) string {
	var ii iconInfo
	ret, _, _ := procGetIconInfo.Call(uintptr(hIcon), uintptr(unsafe.Pointer(&ii)))
	if ret == 0 {
		return ""
	}

	if ii.HbmMask != 0 {
		defer procDeleteObject.Call(uintptr(ii.HbmMask))
	}
	if ii.HbmColor == 0 {
		return ""
	}
	defer procDeleteObject.Call(uintptr(ii.HbmColor))

	var bm bitmap
	procGetObject.Call(uintptr(ii.HbmColor), unsafe.Sizeof(bm), uintptr(unsafe.Pointer(&bm)))

	width := int(bm.Width)
	height := int(bm.Height)
	if width == 0 || height == 0 {
		return ""
	}

	hdc, _, _ := procCreateCompatDC.Call(0)
	if hdc == 0 {
		return ""
	}
	defer procDeleteDC.Call(hdc)

	bih := bitmapInfoHeader{
		Size:     uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		Width:    int32(width),
		Height:   -int32(height), // top-down
		Planes:   1,
		BitCount: 32,
	}

	pixels := make([]byte, width*height*4)

	procGetDIBits.Call(
		hdc,
		uintptr(ii.HbmColor),
		0,
		uintptr(height),
		uintptr(unsafe.Pointer(&pixels[0])),
		uintptr(unsafe.Pointer(&bih)),
		0, // DIB_RGB_COLORS
	)

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := (y*width + x) * 4
			// BGRA → RGBA
			img.Set(x, y, color.RGBA{
				R: pixels[i+2],
				G: pixels[i+1],
				B: pixels[i],
				A: pixels[i+3],
			})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return ""
	}

	return fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(buf.Bytes()))
}
