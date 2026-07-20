// Package figani decodes FD2's indexed FIGANI battle-animation resources.
// It preserves raw palette indices and transparent RLE spans so callers can
// reproduce 0x2935b on an indexed native surface instead of reusing exported
// RGBA PNGs.
package figani

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type Animation struct{ Frames []Frame }

// Frame holds the 13-byte FIGANI header fields consumed by 0x2935b. X/Y are
// signed native 320x200 coordinates; Pixels is a decoded W×H indexed image
// where Mask distinguishes transparent codec output from palette index zero.
type Frame struct {
	X, Y          int
	Width, Height int
	Pixels, Mask  []byte
	Delay         int
}

func DecodeResource(path string, resource int) (*Animation, error) {
	raw, err := fdother.ReadResource(path, resource)
	if err != nil {
		return nil, err
	}
	return Parse(raw)
}

func Parse(raw []byte) (*Animation, error) {
	if len(raw) < 12 {
		return nil, errors.New("figani: animation is too short")
	}
	n := int(binary.LittleEndian.Uint16(raw))
	if n == 0 || 8+4*n > len(raw) {
		return nil, errors.New("figani: invalid frame table")
	}
	frames := make([]Frame, n)
	previous := 8 + 4*n
	for i := range frames {
		off := int(binary.LittleEndian.Uint32(raw[8+4*i:]))
		end := len(raw)
		if i+1 < n {
			end = int(binary.LittleEndian.Uint32(raw[8+4*(i+1):]))
		}
		if off < previous || off+13 > end || end > len(raw) {
			return nil, fmt.Errorf("figani: invalid frame %d offset", i)
		}
		w, h := int(binary.LittleEndian.Uint16(raw[off+9:])), int(binary.LittleEndian.Uint16(raw[off+11:]))
		if w <= 0 || h <= 0 || w > 1024 || h > 1024 {
			return nil, fmt.Errorf("figani: invalid frame %d geometry", i)
		}
		pixels, mask, err := decodeRLE(raw[off+13:end], w, h)
		if err != nil {
			return nil, fmt.Errorf("figani: frame %d: %w", i, err)
		}
		frames[i] = Frame{X: int(int16(binary.LittleEndian.Uint16(raw[off:]))), Y: int(int16(binary.LittleEndian.Uint16(raw[off+2:]))), Width: w, Height: h, Pixels: pixels, Mask: mask, Delay: int(binary.LittleEndian.Uint16(raw[off+6:]))}
		previous = off
	}
	return &Animation{Frames: frames}, nil
}

func decodeRLE(src []byte, width, height int) ([]byte, []byte, error) {
	pixels, mask := make([]byte, width*height), make([]byte, width*height)
	pos := 0
	for y := 0; y < height; y++ {
		x := 0
		for x < width {
			if pos >= len(src) {
				return nil, nil, fmt.Errorf("RLE ends in row %d", y)
			}
			ctrl := src[pos]
			pos++
			count, mode := int(ctrl&0x3f)+1, ctrl>>6
			span := count
			if mode == 1 {
				span *= 2
			}
			if x+span > width {
				return nil, nil, fmt.Errorf("RLE overruns row %d", y)
			}
			write := func(at int, value byte) { pixels[y*width+at], mask[y*width+at] = value, 1 }
			switch mode {
			case 0:
				if pos >= len(src) {
					return nil, nil, errors.New("colour run lacks value")
				}
				v := src[pos]
				pos++
				for i := 0; i < count; i++ {
					write(x+i, v)
				}
			case 1:
				if pos >= len(src) {
					return nil, nil, errors.New("dither run lacks value")
				}
				v := src[pos]
				pos++
				for i := 0; i < count; i++ {
					write(x+2*i+1, v)
				}
			case 2:
				if pos+count > len(src) {
					return nil, nil, errors.New("literal run exceeds data")
				}
				for i, v := range src[pos : pos+count] {
					write(x+i, v)
				}
				pos += count
			case 3:
				// Native transparent spans preserve their destination.
			}
			x += span
		}
	}
	return pixels, mask, nil
}

// BlitAt reproduces the transparent branch used by the 0x2c548 FIGANI calls.
func (f Frame) BlitAt(dst []byte, stride int) error {
	if f.Width <= 0 || f.Height <= 0 || len(f.Pixels) != f.Width*f.Height || len(f.Mask) != len(f.Pixels) || stride <= 0 || f.X < 0 || f.Y < 0 || f.X+f.Width > stride || (f.Y+f.Height)*stride > len(dst) {
		return errors.New("figani: frame cannot be blitted to destination")
	}
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			i := y*f.Width + x
			if f.Mask[i] != 0 {
				dst[(f.Y+y)*stride+f.X+x] = f.Pixels[i]
			}
		}
	}
	return nil
}
