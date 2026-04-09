//go:build !windows

package apps

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

func GetIconBase64(path string) string {
	if data := iconFromImagePath(path); data != "" {
		return data
	}

	if strings.HasSuffix(strings.ToLower(path), ".desktop") {
		if iconPath := resolveDesktopIcon(path); iconPath != "" {
			if data := iconFromImagePath(iconPath); data != "" {
				return data
			}
		}
	}

	return fallbackIcon(path)
}

func iconFromImagePath(path string) string {
	lower := strings.ToLower(path)
	if !(strings.HasSuffix(lower, ".png") || strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg")) {
		return ""
	}

	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	var img image.Image
	if strings.HasSuffix(lower, ".png") {
		img, err = png.Decode(f)
	} else {
		img, err = jpeg.Decode(f)
	}
	if err != nil {
		return ""
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return ""
	}
	return fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(buf.Bytes()))
}

func resolveDesktopIcon(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	var icon string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Icon=") {
			icon = strings.TrimSpace(strings.TrimPrefix(line, "Icon="))
			break
		}
	}
	if icon == "" {
		return ""
	}

	if filepath.IsAbs(icon) {
		if _, err := os.Stat(icon); err == nil {
			return icon
		}
		for _, ext := range []string{".png", ".jpg", ".jpeg"} {
			if _, err := os.Stat(icon + ext); err == nil {
				return icon + ext
			}
		}
		return ""
	}

	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".local", "share", "icons", "hicolor", "256x256", "apps", icon+".png"),
		filepath.Join(home, ".local", "share", "icons", "hicolor", "128x128", "apps", icon+".png"),
		filepath.Join("/usr/share/icons/hicolor/256x256/apps", icon+".png"),
		filepath.Join("/usr/share/icons/hicolor/128x128/apps", icon+".png"),
		filepath.Join("/usr/share/pixmaps", icon+".png"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

func fallbackIcon(path string) string {
	label := strings.TrimSpace(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	if label == "" {
		label = "A"
	}
	r := []rune(strings.ToUpper(label))
	char := r[0]

	size := 96
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	bg := color.RGBA{R: hashByte(path, 0), G: hashByte(path, 1), B: hashByte(path, 2), A: 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)

	fg := color.RGBA{R: 250, G: 250, B: 250, A: 255}
	drawSimpleGlyph(img, char, fg)

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(buf.Bytes()))
}

func drawSimpleGlyph(img *image.RGBA, char rune, c color.RGBA) {
	// Tiny block glyph fallback: draw a centered bar + dot pattern derived from char code.
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	cx, cy := w/2, h/2

	for y := cy - 24; y <= cy+24; y++ {
		for x := cx - 6; x <= cx+6; x++ {
			img.SetRGBA(x, y, c)
		}
	}

	seed := int(char)
	for i := 0; i < 6; i++ {
		x := cx - 26 + (seed+i*11)%52
		y := cy - 26 + (seed+i*17)%52
		for yy := y; yy < y+6; yy++ {
			for xx := x; xx < x+6; xx++ {
				img.SetRGBA(xx, yy, c)
			}
		}
	}
}

func hashByte(s string, offset int) uint8 {
	h := 2166136261
	for i, b := range []byte(s) {
		h ^= int(b) + offset + i
		h *= 16777619
	}
	v := 40 + (h % 170)
	return uint8(v)
}
