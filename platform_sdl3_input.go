//go:build sdl3

package main

import (
	"sync"

	"github.com/jupiterrider/purego-sdl3/sdl"
)

var pcScancodeMap = []byte{
	0,
	0, 0, 0,
	0x1E, 0x30, 0x2E, 0x20, 0x12, 0x21, 0x22, 0x23, 0x17,
	0x24, 0x25, 0x26, 0x32, 0x31, 0x18, 0x19, 0x10, 0x13,
	0x1F, 0x14, 0x16, 0x2F, 0x11, 0x2D, 0x15, 0x2C,
	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	0x1C, 0x01, 0x0E, 0x0F, 0x39,
	0x0C, 0x0D, 0x1A, 0x1B, 0x2B,
	0x2B, 0x27, 0x28, 0x29,
	0x33, 0x34, 0x35, 0x3A,
	0x3B, 0x3C, 0x3D, 0x3E, 0x3F, 0x40, 0x41, 0x42, 0x43, 0x44, 0x57, 0x58,
	0x37, 0x46, 0, 0x52, 0x47, 0x49, 0x53, 0x4F, 0x51,
	0x4D, 0x4B, 0x50, 0x48, 0x45,
}

var KeyQueueLock = sync.Mutex{}
var KeyQueue = make([]byte, 0)

func KeysUpdateModifiers() {
	// stub
}

func ParseSDLKeyboardEvent(e sdl.KeyboardEvent) {
	KeysLeftShiftHeld = (e.Mod & sdl.KeymodLShift) != 0
	KeysRightShiftHeld = (e.Mod & sdl.KeymodRShift) != 0
	KeysShiftHeld = (e.Mod & sdl.KeymodShift) != 0
	KeysCtrlHeld = (e.Mod & sdl.KeymodCtrl) != 0
	KeysAltHeld = (e.Mod & sdl.KeymodAlt) != 0

	if e.Type == sdl.EventKeyDown {
		k := byte(0)
		if KeysAltHeld && e.Key == 'p' {
			k = KEY_ALT_P
		} else if e.Key > 0 && e.Key < 127 {
			k = byte(e.Key)
		} else if e.Scancode <= 83 {
			k = byte(pcScancodeMap[e.Scancode] + 128)
		}

		if ((k & 0x7F) != 0) && !(k >= 32 && k < 127) {
			KeyQueueLock.Lock()
			defer KeyQueueLock.Unlock()

			// KeyQueue = append(KeyQueue, k)
			KeyQueue = []byte{k}
		}
	}
}

func ParseSDLTextInputEvent(e sdl.TextInputEvent) {
	if e.Type == sdl.EventTextInput {
		if e.Text()[0] >= 32 && e.Text()[0] < 127 {
			KeyQueue = []byte{e.Text()[0]}
		}
	}
}

func KeyPressed() bool {
	KeyQueueLock.Lock()
	defer KeyQueueLock.Unlock()

	return len(KeyQueue) > 0
}

func ReadKey() byte {
	KeyQueueLock.Lock()
	defer KeyQueueLock.Unlock()

	if len(KeyQueue) <= 0 {
		return 0
	} else {
		v := KeyQueue[0]
		KeyQueue = KeyQueue[1:]
		return v
	}
}
