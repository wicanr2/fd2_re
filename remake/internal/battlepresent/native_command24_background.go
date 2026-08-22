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

// BuildNativeCommand24BackgroundFrames reproduces the two sub_29C90 viewport
// loops. It returns no partial frames when an input or transparent blit is
// invalid, so a presentation owner can remain fail-closed before mutation.
func BuildNativeCommand24BackgroundFrames(in NativeCommand24BackgroundInputs) ([][]byte, error) {
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
	frames := make([][]byte, 0, 20)
	for i := 9; i >= 0; i-- {
		if err := in.Layers[i%3].BlitAt(work, nativeCommand24WorkStride, 0, -1); err != nil {
			return nil, err
		}
		frames = append(frames, nativeCommand24Viewport(work, 32*i))
	}

	clear(work)
	for y := 0; y < NativeCommand24BackgroundHeight; y++ {
		copy(work[y*nativeCommand24WorkStride:], in.Target[y*320:(y+1)*320])
	}
	if err := in.TargetIdle.BlitAt(work, nativeCommand24WorkStride); err != nil {
		return nil, err
	}
	for j := 9; j >= 0; j-- {
		if err := in.Layers[(j+2)%3].BlitAt(work, nativeCommand24WorkStride, 320, -1); err != nil {
			return nil, err
		}
		frames = append(frames, nativeCommand24Viewport(work, 32*j))
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
