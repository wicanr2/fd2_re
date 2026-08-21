// Package dato decodes the opaque 4-frame portrait resources used by FD2's
// DATO.DAT loader. It intentionally keeps the indexed pixels and four-frame
// mouth sequence separate from the ending text/layout scheduler.
package dato

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type Frame struct {
	Width, Height int
	Pixels        []byte
}

func DecodeResource(datPath string, resource int) ([]Frame, error) {
	entry, err := fdother.ReadResource(datPath, resource)
	if err != nil {
		return nil, err
	}
	return ParseResource(entry)
}

// ParseResource mirrors the DATO LLLLLL payload: four u32 offsets followed by
// width/height and the 0x4e916 high-run codec. Unlike transparent sprite
// codecs, zero is a literal portrait pixel and is written by BlitAt.
func ParseResource(data []byte) ([]Frame, error) {
	if len(data) < 16 {
		return nil, errors.New("dato: resource is shorter than four offsets")
	}
	offsets := make([]int, 4)
	for i := range offsets {
		offsets[i] = int(binary.LittleEndian.Uint32(data[i*4:]))
		if offsets[i] < 16 || offsets[i] >= len(data) || (i > 0 && offsets[i] < offsets[i-1]) {
			return nil, fmt.Errorf("dato: frame %d offset is invalid", i)
		}
	}
	frames := make([]Frame, 4)
	for i, start := range offsets {
		end := len(data)
		if i+1 < len(offsets) {
			end = offsets[i+1]
		}
		if start+4 > end {
			return nil, fmt.Errorf("dato: frame %d header exceeds bounds", i)
		}
		w := int(binary.LittleEndian.Uint16(data[start:]))
		h := int(binary.LittleEndian.Uint16(data[start+2:]))
		if w <= 0 || h <= 0 || w > 256 || h > 256 {
			return nil, fmt.Errorf("dato: frame %d geometry is invalid", i)
		}
		pixels, err := decodePixels(data[start+4:end], w*h)
		if err != nil {
			return nil, fmt.Errorf("dato: frame %d: %w", i, err)
		}
		frames[i] = Frame{Width: w, Height: h, Pixels: pixels}
	}
	return frames, nil
}

func decodePixels(stream []byte, want int) ([]byte, error) {
	pixels := make([]byte, 0, want)
	for len(pixels) < want {
		if len(stream) == 0 {
			return nil, errors.New("RLE stream ends before frame is filled")
		}
		code := stream[0]
		stream = stream[1:]
		if code <= 0xc0 {
			pixels = append(pixels, code)
			continue
		}
		if len(stream) == 0 {
			return nil, errors.New("RLE repeat lacks a pixel")
		}
		value := stream[0]
		stream = stream[1:]
		run := int(code) - 0xc0
		if run > want-len(pixels) {
			run = want - len(pixels)
		}
		for i := 0; i < run; i++ {
			pixels = append(pixels, value)
		}
	}
	return pixels, nil
}

func (f Frame) BlitAt(dst []byte, stride, x, y int) error {
	if f.Width <= 0 || f.Height <= 0 || len(f.Pixels) != f.Width*f.Height || x < 0 || y < 0 || stride < x+f.Width || y > len(dst)/stride || f.Height > (len(dst)-y*stride)/stride {
		return errors.New("dato: frame destination is too small")
	}
	for row := 0; row < f.Height; row++ {
		copy(dst[(y+row)*stride+x:], f.Pixels[row*f.Width:(row+1)*f.Width])
	}
	return nil
}

// BlitAtOffset is the offset form used by native 0x4e8af callers. The
// caller owns the proven staging anchor; this helper does not invent one.
func (f Frame) BlitAtOffset(dst []byte, stride, offset int) error {
	if stride <= 0 || offset < 0 || offset%stride+f.Width > stride {
		return errors.New("dato: frame offset is invalid")
	}
	return f.BlitAt(dst, stride, offset%stride, offset/stride)
}

// BlitRightToLeftAtOffset 重現0x4E8E1：offset是每列最右端，來源像素依序
// 寫入後遞減目的位址；下一列再從原始offset加stride開始。
func (f Frame) BlitRightToLeftAtOffset(dst []byte, stride, offset int) error {
	if f.Width <= 0 || f.Height <= 0 || len(f.Pixels) != f.Width*f.Height ||
		stride <= 0 || offset < 0 || offset%stride < f.Width-1 ||
		offset/stride+f.Height > len(dst)/stride {
		return errors.New("dato: right-to-left frame destination is too small")
	}
	for row := 0; row < f.Height; row++ {
		right := offset + row*stride
		for column := 0; column < f.Width; column++ {
			dst[right-column] = f.Pixels[row*f.Width+column]
		}
	}
	return nil
}
