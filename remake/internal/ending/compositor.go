package ending

import (
	"errors"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

const (
	Width  = 320
	Height = 200
	Bytes  = Width * Height
)

// IndexedCompositor is the verified memory model used by 0x2bce5: VGA plus
// two 320x200 indexed buffers and a 6-bit VGA DAC palette. It contains no UI
// adapter and cannot by itself make a recovered ending playable.
type IndexedCompositor struct {
	VGA, Offscreen, Work []byte
	Palette              [768]byte
}

func NewIndexedCompositor() *IndexedCompositor {
	return &IndexedCompositor{VGA: make([]byte, Bytes), Offscreen: make([]byte, Bytes), Work: make([]byte, Bytes)}
}

func (c *IndexedCompositor) CopyToVGA(source []byte) error {
	if len(source) != Bytes || len(c.VGA) != Bytes {
		return errors.New("ending: invalid 320x200 copy buffer")
	}
	copy(c.VGA, source)
	return nil
}

func (c *IndexedCompositor) Blit(frame fdother.Frame, destination []byte, stride int) error {
	return frame.Blit(destination, stride, -1)
}

// PaletteDelta reproduces 0x11df2's clamp-to-0..63 RGB addition for every
// palette entry in the inclusive index range.
func (c *IndexedCompositor) PaletteDelta(start, end, delta int) error {
	if start < 0 || end < start || end > 255 {
		return errors.New("ending: invalid palette range")
	}
	for i := start * 3; i <= end*3+2; i++ {
		v := int(c.Palette[i]) + delta
		if v < 0 {
			v = 0
		}
		if v > 63 {
			v = 63
		}
		c.Palette[i] = byte(v)
	}
	return nil
}
