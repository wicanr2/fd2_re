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

// Animation is a FIGANI resource. The native container uses byte 0 as the
// total frame count, byte 1 as a prelude flag and byte 2 as the prelude frame
// count. HeaderByte4 is consumed by 0x2b659 but its gameplay meaning is not
// yet named. Keeping the raw header bytes prevents presentation-side tracing
// from silently discarding original inputs.
type Animation struct {
	Frames      []Frame
	HeaderByte1 byte
	HeaderByte2 byte
	HeaderByte4 byte
}

// NativeScheduler mirrors 0x2b9a1's two-byte global state (0x540fd frame
// index, 0x540fc subframe counter). It selects a frame before advancing; the
// caller remains responsible for the 0x2935b indexed blit.
type NativeScheduler struct {
	FrameIndex int
	Subframe   int
}

// Step returns the frame to render for this call. advance=false is the native
// arg4==0 initialization call: it clears subframe and performs no rendering.
func (s *NativeScheduler) Step(animation *Animation, advance bool) (int, bool, error) {
	if s == nil || animation == nil || len(animation.Frames) == 0 || s.FrameIndex < 0 || s.FrameIndex >= len(animation.Frames) || s.Subframe < 0 {
		return 0, false, errors.New("figani: invalid native scheduler state")
	}
	if !advance {
		s.Subframe = 0
		return s.FrameIndex, false, nil
	}
	selected := s.FrameIndex
	delay := animation.Frames[selected].Delay
	s.Subframe++
	if s.Subframe >= delay {
		s.Subframe = 0
		s.FrameIndex++
		if s.FrameIndex >= len(animation.Frames) {
			s.FrameIndex = 0
		}
	}
	return selected, true, nil
}

// DisplayScheduler adapts the native descriptor delay to a caller's display
// ticks.  The native routine advances one subframe only after the current
// frame has been presented; ticksPerNative lets a renderer retain an explicit
// speed control without discarding the raw +6 delay.  It contains no pixel or
// hit-effect semantics, so it is safe to use for a presentation-only bridge.
type DisplayScheduler struct {
	native         NativeScheduler
	animation      Animation
	ticksPerNative int
	frameStarts    []int
	bodyTicks      int
	displayTick    int
	currentFrame   int
	done           bool
}

// NewDisplayScheduler constructs a strict delay timeline from raw descriptor
// delays.  Zero or negative delays are rejected instead of being normalised.
func NewDisplayScheduler(delays []int, ticksPerNative int) (*DisplayScheduler, error) {
	if len(delays) == 0 {
		return nil, errors.New("figani: display scheduler has no frames")
	}
	if ticksPerNative <= 0 {
		return nil, errors.New("figani: display scheduler has invalid tick scale")
	}
	starts := make([]int, len(delays)+1)
	frames := make([]Frame, len(delays))
	total := 0
	for i, delay := range delays {
		if delay <= 0 {
			return nil, fmt.Errorf("figani: frame %d has invalid delay %d", i, delay)
		}
		starts[i] = total
		frames[i] = Frame{Delay: delay}
		total += delay * ticksPerNative
	}
	starts[len(delays)] = total
	return &DisplayScheduler{
		animation:      Animation{Frames: frames},
		ticksPerNative: ticksPerNative,
		frameStarts:    starts,
		bodyTicks:      total,
	}, nil
}

// Step presents one caller display tick and returns the selected native frame.
// The final tick returns done=true; a caller may keep drawing CurrentFrame
// during a separately owned tail hold without advancing this state machine.
func (s *DisplayScheduler) Step() (frame int, presented, done bool, err error) {
	if s == nil || len(s.animation.Frames) == 0 || s.ticksPerNative <= 0 || s.bodyTicks <= 0 {
		return 0, false, false, errors.New("figani: invalid display scheduler state")
	}
	if s.done {
		return s.currentFrame, false, true, nil
	}
	if s.displayTick%s.ticksPerNative == 0 {
		selected, rendered, stepErr := s.native.Step(&s.animation, true)
		if stepErr != nil {
			return 0, false, false, stepErr
		}
		if !rendered {
			return 0, false, false, errors.New("figani: native display step did not present")
		}
		s.currentFrame = selected
	}
	s.displayTick++
	if s.displayTick >= s.bodyTicks {
		s.done = true
	}
	return s.currentFrame, true, s.done, nil
}

// BodyTicks returns the exact scaled duration before a caller-owned tail hold.
func (s *DisplayScheduler) BodyTicks() int {
	if s == nil {
		return 0
	}
	return s.bodyTicks
}

// FrameStart returns the scaled display tick at which a frame is first
// presented.  It is useful for a separately evidenced visual marker, but does
// not itself assign attack, damage or sound semantics.
func (s *DisplayScheduler) FrameStart(frame int) (int, bool) {
	if s == nil || frame < 0 || frame >= len(s.animation.Frames) {
		return 0, false
	}
	return s.frameStarts[frame], true
}

// CurrentFrame returns the last selected frame without advancing the schedule.
func (s *DisplayScheduler) CurrentFrame() int {
	if s == nil {
		return 0
	}
	return s.currentFrame
}

// Frame holds the 13-byte FIGANI header fields consumed by 0x2935b. X/Y are
// signed native 320x200 coordinates; Pixels is a decoded W×H indexed image
// where Mask distinguishes transparent codec output from palette index zero.
type Frame struct {
	X, Y          int
	Width, Height int
	Pixels, Mask  []byte
	Delay         int
	RawByte4      byte
	RawByte5      byte
	RawByte7      byte
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
	// 0x29409 reads byte 0, while 0x29510/0x295c3 independently read
	// bytes 1 and 2. Treating bytes 0..1 as a u16 rejects every original
	// animation whose native prelude flag is set.
	n := int(raw[0])
	if n == 0 || 8+4*n > len(raw) {
		return nil, errors.New("figani: invalid frame table")
	}
	if raw[1] != 0 && int(raw[2]) > n {
		return nil, errors.New("figani: prelude frame count exceeds total frames")
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
		// 0x2939d consumes +4, +5, +6 and +7 as four independent bytes.
		// In particular, +6 is the frame delay; retaining the other bytes by
		// raw offset avoids assigning an unclosed effect/sound/z-order meaning.
		// Reading a u16 at +6 turns byte7 value 6 plus delay 38 into a bogus 1574.
		frames[i] = Frame{
			X: int(int16(binary.LittleEndian.Uint16(raw[off:]))), Y: int(int16(binary.LittleEndian.Uint16(raw[off+2:]))),
			Width: w, Height: h, Pixels: pixels, Mask: mask,
			RawByte4: raw[off+4], RawByte5: raw[off+5], Delay: int(raw[off+6]), RawByte7: raw[off+7],
		}
		previous = off
	}
	return &Animation{Frames: frames, HeaderByte1: raw[1], HeaderByte2: raw[2], HeaderByte4: raw[4]}, nil
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
	return f.BlitAtBase(dst, stride, 0)
}

// BlitAtBase is BlitAt with an explicit byte origin within a larger native
// work surface. 0x29164 uses it for its stage*10 shifts into a 640-stride
// buffer; the frame's signed coordinates remain untouched.
func (f Frame) BlitAtBase(dst []byte, stride, base int) error {
	if f.Width <= 0 || f.Height <= 0 || len(f.Pixels) != f.Width*f.Height || len(f.Mask) != len(f.Pixels) || stride <= 0 || base < 0 || base > len(dst) || f.X < 0 || f.Y < 0 || f.X+f.Width > stride || base+(f.Y+f.Height)*stride > len(dst) {
		return errors.New("figani: frame cannot be blitted to destination")
	}
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			i := y*f.Width + x
			if f.Mask[i] != 0 {
				dst[base+(f.Y+y)*stride+f.X+x] = f.Pixels[i]
			}
		}
	}
	return nil
}

// BlitTranslated reproduces the 0x2939d staging-buffer displacement while
// presenting on a bounded indexed surface. Pixels outside that surface are
// clipped, matching the native 400-stride staging margin followed by a
// 320-wide viewport copy. When opaqueFill is non-nil, every decoded opaque
// pixel receives that palette index while transparent spans still preserve
// the destination, matching 0x4e63d's 0..255 branch. Nil retains the source
// palette indices as the native -1 branch does.
func (f Frame) BlitTranslated(dst []byte, stride, dx, dy int, opaqueFill *byte) error {
	if f.Width <= 0 || f.Height <= 0 || len(f.Pixels) != f.Width*f.Height || len(f.Mask) != len(f.Pixels) || stride <= 0 || len(dst)%stride != 0 {
		return errors.New("figani: translated frame cannot be blitted to destination")
	}
	height := len(dst) / stride
	for y := 0; y < f.Height; y++ {
		dstY := f.Y + dy + y
		if dstY < 0 || dstY >= height {
			continue
		}
		for x := 0; x < f.Width; x++ {
			dstX := f.X + dx + x
			if dstX < 0 || dstX >= stride {
				continue
			}
			i := y*f.Width + x
			if f.Mask[i] != 0 {
				value := f.Pixels[i]
				if opaqueFill != nil {
					value = *opaqueFill
				}
				dst[dstY*stride+dstX] = value
			}
		}
	}
	return nil
}
