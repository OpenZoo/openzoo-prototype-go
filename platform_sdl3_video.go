//go:build sdl3

package main

import (
	"C"
	_ "embed"
)
import (
	"unsafe"

	"github.com/jupiterrider/purego-sdl3/sdl"
)

//go:embed ascii.chr
var charsetData []byte
var textBuffer [25][160]byte
var textColumns int = 80
var blinkState bool = false

var palette = []uint32{
	0x000000FF,
	0x0000AAFF,
	0x00AA00FF,
	0x00AAAAFF,
	0xAA0000FF,
	0xAA00AAFF,
	0xAA5500FF,
	0xAAAAAAFF,
	0x555555FF,
	0x5555FFFF,
	0x55FF55FF,
	0x55FFFFFF,
	0xFF5555FF,
	0xFF55FFFF,
	0xFFFF55FF,
	0xFFFFFFFF,
}

func redrawChar(ix, iy int) {
	// TODO: This is *super* slow.

	px := ix * 8
	py := iy * 14
	var pixels unsafe.Pointer
	var pitch int32
	if !sdl.LockTexture(VideoZTexture, &sdl.Rect{X: int32(px), Y: int32(py), W: 8, H: 14}, &pixels, &pitch) {
		return
	}

	ch := int(textBuffer[iy][ix*2]) * 14
	co := textBuffer[iy][ix*2+1]
	if co >= 0x80 {
		co &= 0x7F
		if blinkState {
			co = (co >> 4) * 0x11
		}
	}
	coBg := palette[co>>4]
	coFg := palette[co&0xF]

	for ly := 0; ly < 14; ly++ {
		c := charsetData[ch+ly]
		for lx := 0; lx < 8; lx++ {
			if (c & 0x80) != 0 {
				*(*uint32)(unsafe.Add(pixels, ly*int(pitch)+lx*4)) = coFg
			} else {
				*(*uint32)(unsafe.Add(pixels, ly*int(pitch)+lx*4)) = coBg
			}
			c <<= 1
		}
	}

	sdl.UnlockTexture(VideoZTexture)
}

func ZooSdlToggleBlinkChars() {
	blinkState = !blinkState
	for iy := 0; iy < 25; iy++ {
		for ix := 0; ix < textColumns; ix++ {
			if textBuffer[iy][ix*2+1] >= 0x81 {
				redrawChar(ix, iy)
			}
		}
	}
}

func IVideoSetMode(columns int) {
	MainThreadSync(func() {
		textColumns = columns
	})
}

func IVideoClrScr(backgroundColor uint8) {
	MainThreadAsync(func() {
		for iy := 0; iy < 25; iy++ {
			for ix := 0; ix < textColumns; ix++ {
				textBuffer[iy][ix*2] = ' '
				textBuffer[iy][ix*2+1] = backgroundColor << 4
				redrawChar(ix, iy)
			}
		}
		VideoUpdateRequested.Store(true)
	})
}

func IVideoWriteText(x, y int16, color byte, text string) {
	if y < 0 || y >= 25 {
		return
	}
	MainThreadAsync(func() {
		for i := 0; i < len(text); i++ {
			textBuffer[y][x*2] = text[i]
			textBuffer[y][x*2+1] = color
			redrawChar(int(x), int(y))
			x++
			if x >= int16(textColumns) {
				x = 0
				y++
				if y >= 25 {
					break
				}
			}
		}
		VideoUpdateRequested.Store(true)
	})
}

func IVideoSetCursorVisible(v bool) {
	MainThreadSync(func() {
		// stub
	})
}

func VideoMove(x, y, width int16, buffer *[]byte, toVideo bool) {
	if toVideo {
		MainThreadAsync(func() {
			if buffer != nil {
				if width > int16(len(*buffer)>>1) {
					width = int16(len(*buffer) >> 1)
				}
				for i := 0; i < int(width)*2; i++ {
					textBuffer[y][int(x)*2+i] = (*buffer)[i]
				}
				for i := 0; i < int(width); i++ {
					redrawChar(int(x)+i, int(y))
				}
			}

			VideoUpdateRequested.Store(true)
		})
	} else {
		// wait for all async video writes to end
		MainThreadSync(func() {

		})
		*buffer = make([]byte, width*2)
		copy(*buffer, textBuffer[y][x*2:(x+width)*2])
	}
}
