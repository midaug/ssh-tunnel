//go:build ignore

package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

// 生成托盘图标：深色圆角方块背景 + 白色隧道/通道图形
// macOS 状态栏推荐尺寸 22x22（含 padding），模板图标（单色透明）

func main() {
	// 托盘图标：彩色（活跃）+ 灰色（空闲）两套
	colorImg := genTrayIconImage(44, false)
	grayImg := genTrayIconImage(44, true)
	writePNG("internal/tray/icon.png", colorImg)
	writePNG("internal/tray/icon_gray.png", grayImg)
	// Windows 托盘需要 ICO 格式
	writeICO("internal/tray/icon.ico", colorImg, []int{32, 16})
	writeICO("internal/tray/icon_gray.ico", grayImg, []int{32, 16})
	// 应用图标：彩色 1024x1024
	appImg := genAppIconImage(1024)
	writePNG("build/appicon.png", appImg)
	// Windows 应用 ICO（多分辨率）
	writeICO("build/windows/icon.ico", appImg, []int{256, 128, 64, 48, 32, 16})
}

func genTrayIconImage(size int, gray bool) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	cx, cy := float64(size)/2, float64(size)/2
	r := float64(size) * 0.46

	// 圆形背景：活跃时蓝色，空闲时灰色
	bg := color.NRGBA{37, 99, 235, 255}
	if gray {
		bg = color.NRGBA{156, 163, 175, 255} // 柔和灰
	}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			dx, dy := px-cx, py-cy
			d := math.Sqrt(dx*dx + dy*dy)
			if d <= r {
				img.Set(x, y, bg)
			}
		}
	}

	// 白色隧道图形
	white := color.NRGBA{255, 255, 255, 255}
	lineW := size / 9
	if lineW < 2 {
		lineW = 2
	}
	yTop := size * 3 / 8
	yBot := size * 5 / 8
	xLeft := size * 5 / 18
	xRight := size * 13 / 18
	for x := xLeft; x <= xRight; x++ {
		for dy := -lineW / 2; dy <= lineW/2; dy++ {
			img.Set(x, yTop+dy, white)
			img.Set(x, yBot+dy, white)
		}
	}
	// 中间箭头
	yMid := size / 2
	arrowLen := (xRight - xLeft) * 2 / 3
	axStart := xLeft + (xRight-xLeft)/6
	for i := 0; i < arrowLen; i++ {
		ax := axStart + i
		half := i * lineW / arrowLen
		for dy := -half; dy <= half; dy++ {
			img.Set(ax, yMid+dy, white)
		}
	}
	for i := 0; i < lineW*2; i++ {
		ax := axStart + arrowLen + i
		if ax >= size {
			break
		}
		half := (arrowLen - i) * lineW / arrowLen
		if half < 0 {
			half = 0
		}
		for dy := -half; dy <= half; dy++ {
			img.Set(ax, yMid+dy, white)
		}
	}
	return img
}

func genAppIconImage(size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))

	// 背景渐变圆角方块
	bg1 := color.NRGBA{30, 64, 175, 255} // 深蓝
	bg2 := color.NRGBA{59, 130, 246, 255} // 亮蓝
	cx, cy := float64(size)/2, float64(size)/2
	radius := float64(size) * 0.46
	corner := float64(size) * 0.22

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			// 圆角判断
			if !inRoundedRect(px, py, cx-radius, cy-radius, cx+radius, cy+radius, corner) {
				continue
			}
			// 垂直渐变
			t := (py - (cy - radius)) / (2 * radius)
			c := lerp(bg1, bg2, t)
			img.Set(x, y, c)
		}
	}

	// 绘制白色隧道图形：两个同心圆弧 + 中间通道
	white := color.NRGBA{255, 255, 255, 255}
	lineW := size / 16
	if lineW < 4 {
		lineW = 4
	}

	// 两条水平通道线
	yTop := int(float64(size) * 0.38)
	yBot := int(float64(size) * 0.62)
	xLeft := int(float64(size) * 0.28)
	xRight := int(float64(size) * 0.72)
	for x := xLeft; x <= xRight; x++ {
		for dy := -lineW / 2; dy <= lineW/2; dy++ {
			if inRoundedRect(float64(x)+0.5, float64(yTop+dy)+0.5, cx-radius, cy-radius, cx+radius, cy+radius, corner) {
				img.Set(x, yTop+dy, white)
			}
			if inRoundedRect(float64(x)+0.5, float64(yBot+dy)+0.5, cx-radius, cy-radius, cx+radius, cy+radius, corner) {
				img.Set(x, yBot+dy, white)
			}
		}
	}

	// 中间箭头
	yMid := size / 2
	arrowLen := (xRight - xLeft) * 3 / 4
	axStart := xLeft + (xRight-xLeft)/8
	for i := 0; i < arrowLen; i++ {
		ax := axStart + i
		half := i * lineW * 3 / 2 / arrowLen
		for dy := -half; dy <= half; dy++ {
			if inRoundedRect(float64(ax)+0.5, float64(yMid+dy)+0.5, cx-radius, cy-radius, cx+radius, cy+radius, corner) {
				img.Set(ax, yMid+dy, white)
			}
		}
	}
	// 箭头尖端
	for i := 0; i < lineW*3; i++ {
		ax := axStart + arrowLen + i
		if ax >= size {
			break
		}
		half := (arrowLen - i) * lineW * 3 / 2 / arrowLen
		if half < 0 {
			half = 0
		}
		for dy := -half; dy <= half; dy++ {
			if inRoundedRect(float64(ax)+0.5, float64(yMid+dy)+0.5, cx-radius, cy-radius, cx+radius, cy+radius, corner) {
				img.Set(ax, yMid+dy, white)
			}
		}
	}
	return img
}

// writeICO 将 1024x1024 图像缩放为多分辨率，写入 Windows ICO 文件
func writeICO(path string, src image.Image, sizes []int) {
	type dirEntry struct {
		Width   uint8
		Height  uint8
		Colors  uint8
		_       uint8
		Planes  uint16
		BPP     uint16
		Size    uint32
		Offset  uint32
	}

	var pngs [][]byte
	for _, s := range sizes {
		scaled := scaleImage(src, s)
		var buf bytes.Buffer
		_ = png.Encode(&buf, scaled)
		pngs = append(pngs, buf.Bytes())
	}

	count := uint16(len(sizes))
	header := make([]byte, 6)
	binary.LittleEndian.PutUint16(header[0:], 0)      // reserved
	binary.LittleEndian.PutUint16(header[2:], 1)      // type = ICO
	binary.LittleEndian.PutUint16(header[4:], count)

	dataOffset := uint32(6 + 16*len(sizes))
	var entries []byte
	for i, s := range sizes {
		w := uint8(s)
		if s == 256 {
			w = 0
		}
		e := dirEntry{
			Width:  w,
			Height: w,
			Colors: 0,
			Planes: 1,
			BPP:    32,
			Size:   uint32(len(pngs[i])),
			Offset: dataOffset,
		}
		var eb bytes.Buffer
		_ = binary.Write(&eb, binary.LittleEndian, e)
		entries = append(entries, eb.Bytes()...)
		dataOffset += uint32(len(pngs[i]))
	}

	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.Write(header)
	f.Write(entries)
	for _, p := range pngs {
		f.Write(p)
	}
}

// scaleImage 最近邻缩放
func scaleImage(src image.Image, size int) image.Image {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	xRatio := float64(b.Dx()) / float64(size)
	yRatio := float64(b.Dy()) / float64(size)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			sx := int(float64(x) * xRatio)
			sy := int(float64(y) * yRatio)
			dst.Set(x, y, src.At(b.Min.X+sx, b.Min.Y+sy))
		}
	}
	return dst
}

func inRoundedRect(x, y, x0, y0, x1, y1, r float64) bool {
	if x < x0 || x > x1 || y < y0 || y > y1 {
		return false
	}
	cx, cy := x, y
	// 距离最近的角
	if cx < x0+r {
		cx = x0 + r
	} else if cx > x1-r {
		cx = x1 - r
	}
	if cy < y0+r {
		cy = y0 + r
	} else if cy > y1-r {
		cy = y1 - r
	}
	dx, dy := x-cx, y-cy
	return dx*dx+dy*dy <= r*r
}

func lerp(c1, c2 color.NRGBA, t float64) color.NRGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return color.NRGBA{
		R: uint8(float64(c1.R) + (float64(c2.R)-float64(c1.R))*t),
		G: uint8(float64(c1.G) + (float64(c2.G)-float64(c1.G))*t),
		B: uint8(float64(c1.B) + (float64(c2.B)-float64(c1.B))*t),
		A: 255,
	}
}

func writePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}
