package apps

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"sync"
	"syscall"
	"unsafe"
)

var (
	shell32            = syscall.NewLazyDLL("shell32.dll")
	user32             = syscall.NewLazyDLL("user32.dll")
	gdi32              = syscall.NewLazyDLL("gdi32.dll")
	procSHGetFileInfo  = shell32.NewProc("SHGetFileInfoW")
	procDestroyIcon    = user32.NewProc("DestroyIcon")
	procGetIconInfo    = user32.NewProc("GetIconInfo")
	procGetDIBits      = gdi32.NewProc("GetDIBits")
	procCreateCompatDC = gdi32.NewProc("CreateCompatibleDC")
	procDeleteDC       = gdi32.NewProc("DeleteDC")
	procDeleteObject   = gdi32.NewProc("DeleteObject")
	procGetObject      = gdi32.NewProc("GetObjectW")
)

const (
	shgfiIcon      = 0x000000100
	shgfiSmallIcon = 0x000000001
	shgfiLargeIcon = 0x000000000
	biRGB          = 0
)

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

var iconCache sync.Map

func GetIconBase64(path string) string {
	if cached, ok := iconCache.Load(path); ok {
		return cached.(string)
	}

	data := extractIcon(path)
	iconCache.Store(path, data)
	return data
}

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
