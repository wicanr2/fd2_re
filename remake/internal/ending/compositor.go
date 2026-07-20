package ending

import (
	"errors"
	"fmt"
	"image"

	"github.com/wicanr2/fd2_re/remake/internal/afm"
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

func (c *IndexedCompositor) RGBA() *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, Width, Height))
	for i, index := range c.VGA {
		p := int(index) * 3
		out.Pix[i*4] = (c.Palette[p] << 2) | (c.Palette[p] >> 4)
		out.Pix[i*4+1] = (c.Palette[p+1] << 2) | (c.Palette[p+1] >> 4)
		out.Pix[i*4+2] = (c.Palette[p+2] << 2) | (c.Palette[p+2] >> 4)
		out.Pix[i*4+3] = 0xff
	}
	return out
}

type ANIPlayer struct {
	Clip                    *afm.Clip
	DelayMs, Elapsed, Frame int
	Started                 bool
}

func (p *ANIPlayer) Advance(c *IndexedCompositor, elapsedMs int) (bool, error) {
	if p.Clip == nil || p.DelayMs <= 0 || p.Frame >= len(p.Clip.IndexedFrames) {
		return true, errors.New("ending: invalid ANI player")
	}
	if !p.Started {
		if err := c.PresentANIFrame(p.Clip, p.Frame); err != nil {
			return true, err
		}
		p.Started = true
		p.Frame++
	}
	if p.Frame >= len(p.Clip.IndexedFrames) {
		return true, nil
	}
	p.Elapsed += elapsedMs
	for p.Elapsed >= p.DelayMs {
		p.Elapsed -= p.DelayMs
		if err := c.PresentANIFrame(p.Clip, p.Frame); err != nil {
			return true, err
		}
		p.Frame++
		if p.Frame >= len(p.Clip.IndexedFrames) {
			return true, nil
		}
	}
	return false, nil
}

// RunRecoveredPrefix executes only fully recovered indexed operations. It
// stops at the first native-only operation, preserving the fail-closed ending
// contract while making the evidence-backed prefix testable.
func (c *IndexedCompositor) RunRecoveredPrefix(t Timeline, frames []fdother.Frame) (int, error) {
	for i, s := range t.Segments {
		switch s.Op {
		case "blit_frame":
			if s.Frame == nil || *s.Frame < 0 || *s.Frame >= len(frames) || s.Transparent == nil || *s.Transparent != -1 {
				return i, fmt.Errorf("ending: invalid recovered blit at %s", s.Source)
			}
			var dst []byte
			switch s.Target {
			case "offscreen":
				dst = c.Offscreen
			case "vga":
				dst = c.VGA
			default:
				return i, fmt.Errorf("ending: unknown target %q", s.Target)
			}
			if err := c.Blit(frames[*s.Frame], dst, s.Stride); err != nil {
				return i, err
			}
		case "copy_buffer":
			if s.From != "offscreen" || s.To != "vga" {
				return i, fmt.Errorf("ending: unknown copy %s→%s", s.From, s.To)
			}
			if err := c.CopyToVGA(c.Offscreen); err != nil {
				return i, err
			}
		case "palette_update":
			if s.PaletteStart == nil || s.PaletteEnd == nil || s.PaletteValue == nil {
				return i, fmt.Errorf("ending: incomplete palette update")
			}
			if err := c.PaletteDelta(*s.PaletteStart, *s.PaletteEnd, *s.PaletteValue); err != nil {
				return i, err
			}
		case "delay_ms":
			// Timing is owned by a future presentation clock; no state mutation.
		default:
			return i, fmt.Errorf("ending: native-only segment %s at %s", s.Op, s.Source)
		}
	}
	return len(t.Segments), nil
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

// PresentANI applies one AFM frame exactly as its VM leaves it: indexed VGA
// bytes and the accompanying 6-bit DAC palette replace the current state.
func (c *IndexedCompositor) PresentANI(frame, palette []byte) error {
	if len(frame) != Bytes || len(palette) != len(c.Palette) {
		return errors.New("ending: invalid ANI indexed snapshot")
	}
	copy(c.VGA, frame)
	copy(c.Palette[:], palette)
	return nil
}

func (c *IndexedCompositor) PresentANIFrame(clip *afm.Clip, index int) error {
	if clip == nil || index < 0 || index >= len(clip.IndexedFrames) || index >= len(clip.Palettes) {
		return errors.New("ending: ANI frame is unavailable")
	}
	return c.PresentANI(clip.IndexedFrames[index], clip.Palettes[index])
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
