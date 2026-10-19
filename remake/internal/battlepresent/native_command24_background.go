package battlepresent

import (
	"errors"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

const (
	NativeCommand24BackgroundWidth  = 320
	NativeCommand24BackgroundHeight = 200
	nativeCommand24WorkStride       = 640
)

// NativeCommand24BackgroundInputs is the caller-owned input to sub_29C90's
// two ten-present viewport slides. Source and Target are already composed
// 320x200 indexed bases; keeping that boundary explicit avoids guessing the
// separate sub_2A289 status-panel compositor.
type NativeCommand24BackgroundInputs struct {
	Layers     [3]fdother.Frame
	Source     []byte
	Target     []byte
	TargetIdle figani.Frame
}

// NativeCommand24BackgroundPresent retains the work-buffer composition at a
// viewport boundary. BGLayers is the ordered write history in the other half;
// transparent BG pixels can retain an earlier layer in that history.
type NativeCommand24BackgroundPresent struct {
	Pixels                   []byte
	Incoming                 bool
	ContentX, BGX, ViewportX int
	BGLayers                 []int
}

func backgroundPixels(presents []NativeCommand24BackgroundPresent) [][]byte {
	frames := make([][]byte, len(presents))
	for i, present := range presents {
		frames[i] = present.Pixels
	}
	return frames
}

// BuildNativeCommand24BackgroundFrames reproduces the two sub_29C90 viewport
// loops. It returns no partial frames when an input or transparent blit is
// invalid, so a presentation owner can remain fail-closed before mutation.
func BuildNativeCommand24BackgroundFrames(in NativeCommand24BackgroundInputs) ([][]byte, error) {
	presents, err := BuildNativeCommand24BackgroundPresents(in)
	if err != nil {
		return nil, err
	}
	return backgroundPixels(presents), nil
}

func BuildNativeCommand24BackgroundPresents(in NativeCommand24BackgroundInputs) ([]NativeCommand24BackgroundPresent, error) {
	const frameBytes = NativeCommand24BackgroundWidth * NativeCommand24BackgroundHeight
	if len(in.Source) != frameBytes || len(in.Target) != frameBytes {
		return nil, errors.New("battlepresent: command24 background bases must be 320x200")
	}
	for i, layer := range in.Layers {
		if layer.Width != NativeCommand24BackgroundWidth || layer.Height != 100 {
			return nil, errors.New("battlepresent: command24 BG layer geometry mismatch")
		}
		layer.X, layer.Y = 0, 50
		in.Layers[i] = layer
	}
	work := make([]byte, nativeCommand24WorkStride*NativeCommand24BackgroundHeight)
	for y := 0; y < NativeCommand24BackgroundHeight; y++ {
		copy(work[y*nativeCommand24WorkStride+320:], in.Source[y*320:(y+1)*320])
	}
	frames := make([]NativeCommand24BackgroundPresent, 0, 20)
	var history []int
	for i := 9; i >= 0; i-- {
		if err := in.Layers[i%3].BlitAt(work, nativeCommand24WorkStride, 0, -1); err != nil {
			return nil, err
		}
		history = append(history, i%3)
		frames = append(frames, NativeCommand24BackgroundPresent{Pixels: nativeCommand24Viewport(work, 32*i), ContentX: 320, BGX: 0, ViewportX: 32 * i, BGLayers: append([]int(nil), history...)})
	}

	clear(work)
	for y := 0; y < NativeCommand24BackgroundHeight; y++ {
		copy(work[y*nativeCommand24WorkStride:], in.Target[y*320:(y+1)*320])
	}
	if err := in.TargetIdle.BlitAt(work, nativeCommand24WorkStride); err != nil {
		return nil, err
	}
	history = nil
	for j := 9; j >= 0; j-- {
		if err := in.Layers[(j+2)%3].BlitAt(work, nativeCommand24WorkStride, 320, -1); err != nil {
			return nil, err
		}
		history = append(history, (j+2)%3)
		frames = append(frames, NativeCommand24BackgroundPresent{Pixels: nativeCommand24Viewport(work, 32*j), Incoming: true, ContentX: 0, BGX: 320, ViewportX: 32 * j, BGLayers: append([]int(nil), history...)})
	}
	return frames, nil
}

// BuildNativePhysicalRightBackgroundFrames 實作 0x29DED 的右向捲動。
// Source 是 caller 留在 work640 左半的完整畫面。Target 已按 BG→TAI→panel
// 建立；0x29DED 的明確 memset 後才複製到右半。兩段共 9+10 presents。
// 證據與 READY 契約：docs/data/ida/fd2_physical_background_selection_20261004.json。
func BuildNativePhysicalRightBackgroundFrames(in NativeCommand24BackgroundInputs) ([][]byte, error) {
	presents, err := BuildNativePhysicalRightBackgroundPresents(in)
	if err != nil {
		return nil, err
	}
	return backgroundPixels(presents), nil
}

func BuildNativePhysicalRightBackgroundPresents(in NativeCommand24BackgroundInputs) ([]NativeCommand24BackgroundPresent, error) {
	const frameBytes = NativeCommand24BackgroundWidth * NativeCommand24BackgroundHeight
	if len(in.Source) != frameBytes || len(in.Target) != frameBytes {
		return nil, errors.New("battlepresent: physical background bases must be 320x200")
	}
	for i, layer := range in.Layers {
		if layer.Width != NativeCommand24BackgroundWidth || layer.Height != 100 {
			return nil, errors.New("battlepresent: physical BG layer geometry mismatch")
		}
		layer.X, layer.Y = 0, 50
		in.Layers[i] = layer
	}
	work := make([]byte, nativeCommand24WorkStride*NativeCommand24BackgroundHeight)
	for y := 0; y < NativeCommand24BackgroundHeight; y++ {
		copy(work[y*nativeCommand24WorkStride:], in.Source[y*320:(y+1)*320])
	}
	frames := make([]NativeCommand24BackgroundPresent, 0, 19)
	var history []int
	for i := 1; i < 10; i++ {
		if err := in.Layers[i%3].BlitAt(work, nativeCommand24WorkStride, 320, -1); err != nil {
			return nil, err
		}
		history = append(history, i%3)
		frames = append(frames, NativeCommand24BackgroundPresent{Pixels: nativeCommand24Viewport(work, 32*i), ContentX: 0, BGX: 320, ViewportX: 32 * i, BGLayers: append([]int(nil), history...)})
	}
	clear(work)
	for y := 0; y < NativeCommand24BackgroundHeight; y++ {
		copy(work[y*nativeCommand24WorkStride+320:], in.Target[y*320:(y+1)*320])
	}
	if err := in.TargetIdle.BlitAtBase(work, nativeCommand24WorkStride, 320); err != nil {
		return nil, err
	}
	history = nil
	for i := 1; i <= 10; i++ {
		if err := in.Layers[(i+1)%3].BlitAt(work, nativeCommand24WorkStride, 0, -1); err != nil {
			return nil, err
		}
		history = append(history, (i+1)%3)
		frames = append(frames, NativeCommand24BackgroundPresent{Pixels: nativeCommand24Viewport(work, 32*i), Incoming: true, ContentX: 320, BGX: 0, ViewportX: 32 * i, BGLayers: append([]int(nil), history...)})
	}
	return frames, nil
}

func nativeCommand24Viewport(work []byte, base int) []byte {
	out := make([]byte, NativeCommand24BackgroundWidth*NativeCommand24BackgroundHeight)
	for y := 0; y < NativeCommand24BackgroundHeight; y++ {
		start := y*nativeCommand24WorkStride + base
		copy(out[y*NativeCommand24BackgroundWidth:(y+1)*NativeCommand24BackgroundWidth], work[start:start+NativeCommand24BackgroundWidth])
	}
	return out
}
