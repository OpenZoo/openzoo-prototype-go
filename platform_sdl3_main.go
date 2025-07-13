//go:build sdl3

package main

// typedef unsigned char Uint8;
// void ZooSdlAudioCallback(void *userdata, Uint8 *stream, int len);
import "C"
import (
	_ "embed"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/jupiterrider/purego-sdl3/sdl"
)

var mainTaskQueue = make(chan func())
var FrameTickCond = sync.NewCond(&sync.Mutex{})
var PitTickCond = sync.NewCond(&sync.Mutex{})
var VideoWindow *sdl.Window
var VideoRenderer *sdl.Renderer
var VideoZTexture *sdl.Texture
var VideoUpdateRequested atomic.Bool
var timerTicks int

func MainThreadAsync(f func()) {
	mainTaskQueue <- f
}

func MainThreadSync(f func()) {
	done := make(chan bool, 1)
	mainTaskQueue <- func() {
		f()
		done <- true
	}
	<-done
}

func TimerTicks() int {
	return timerTicks
}

func MemAvail() int32 {
	// stub
	return 655360
}

func SetCBreak(v bool) {
	// stub
}

func Idle(mode IdleMode) {
	switch mode {
	case IdleUntilFrame:
		FrameTickCond.L.Lock()
		FrameTickCond.Wait()
		FrameTickCond.L.Unlock()
	case IdleUntilPit:
		PitTickCond.L.Lock()
		PitTickCond.Wait()
		PitTickCond.L.Unlock()
	}
}

func Delay(ms uint32) {
	// sdl.Delay(ms)
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func updateSdlEvents() {
	var event sdl.Event
	for sdl.PollEvent(&event) {
		switch event.Type() {
		case sdl.EventKeyDown:
			ParseSDLKeyboardEvent(event.Key())
		case sdl.EventTextInput:
			ParseSDLTextInputEvent(event.Text())
		}
	}
}

func ZooSdlAudioCallback(userdata unsafe.Pointer, stream *sdl.AudioStream, additionalAmount int32, totalAmount int32) {
	buf := make([]byte, additionalAmount)
	bufPtr := (*uint8)(unsafe.Pointer(&buf[0]))

	CurrentAudioSimulator.Simulate(buf)
	sdl.PutAudioStreamData(stream, bufPtr, additionalAmount)
}

func main() {
	runtime.LockOSThread()
	if runtime.NumCPU() > 2 {
		runtime.GOMAXPROCS(2)
	}

	if !sdl.Init(sdl.InitVideo | sdl.InitAudio | sdl.InitGamepad) {
		panic(sdl.GetError())
	}
	defer sdl.Quit()

	if !sdl.CreateWindowAndRenderer("OpenZoo/Go", 640, 350, 0, &VideoWindow, &VideoRenderer) {
		panic(sdl.GetError())
	}
	defer sdl.DestroyRenderer(VideoRenderer)
	defer sdl.DestroyWindow(VideoWindow)

	if VideoZTexture = sdl.CreateTexture(VideoRenderer, sdl.PixelFormatABGR32, sdl.TextureAccessStreaming, 640, 350); VideoZTexture == nil {
		panic(sdl.GetError())
	}
	defer sdl.DestroyTexture(VideoZTexture)

	/* file, _ := os.Create("./cpu.pprof")
	pprof.StartCPUProfile(file)
	defer pprof.StopCPUProfile() */

	frameTicker := time.NewTicker(16666667 * time.Nanosecond)
	pitTicker := time.NewTicker(55 * time.Millisecond)
	blinkTicker := time.NewTicker(266666667 * time.Nanosecond)
	tickerDone := make(chan bool)

	CurrentAudioSimulator = NewAudioSimulatorNearest(48000, byte(32))
	audioSpec := sdl.AudioSpec{
		Freq:     48000,
		Format:   sdl.AudioU8,
		Channels: 1,
	}
	var audioStream *sdl.AudioStream
	if audioStream = sdl.OpenAudioDeviceStream(sdl.AudioDeviceDefaultPlayback, &audioSpec, sdl.NewAudioStreamCallback(ZooSdlAudioCallback), nil); audioStream == nil {
		panic(sdl.GetError())
	}
	defer sdl.DestroyAudioStream(audioStream)
	sdl.ResumeAudioStreamDevice(audioStream)

	sdl.StartTextInput(VideoWindow)
	defer sdl.StopTextInput(VideoWindow)

	go func() {
		for {
			select {
			case <-blinkTicker.C:
				MainThreadAsync(ZooSdlToggleBlinkChars)
			case <-frameTicker.C:
				MainThreadAsync(updateSdlEvents)
				MainThreadAsync(func() {
					sdl.RenderTexture(VideoRenderer, VideoZTexture, nil, nil)
					sdl.RenderPresent(VideoRenderer)
				})
				FrameTickCond.Broadcast()
			case <-pitTicker.C:
				SoundTimerHandler()
				timerTicks++
				PitTickCond.Broadcast()
			case <-tickerDone:
				return
			}
		}
	}()

	go func() {
		ZZTMain()

		frameTicker.Stop()
		pitTicker.Stop()
		blinkTicker.Stop()
		tickerDone <- true

		close(mainTaskQueue)
	}()

	for f := range mainTaskQueue {
		f()
	}
}
