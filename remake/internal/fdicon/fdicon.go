// Package fdicon decodes FD2's 24×24 tactical-map unit sprites. It keeps
// indexed pixels and transparent spans separate so the native raw and LUT
// blitters (0x4deda / 0x4de56) can be reproduced without PNG conversion.
package fdicon

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

const (
	NativeSize            = 24
	NativeMapStride       = 0x1c8
	NativeUnitOriginBytes = 0x75d8
)

type Bank struct{ Sprites []Sprite }

// NativeSelectorCache preserves 0x11019's process-global key-to-slot allocation.
// The native routine compares only its raw-key table; its archive pointer is
// consumed only while materializing a previously unseen twelve-pointer block.
// A caller supplies the raw byte key (the FDFIELD spawn constructor passes b0)
// and receives the slot later written to runtime unit+2. The returned slot is
// not a character, portrait, or direct archive index.
type NativeSelectorCache struct {
	slots map[byte]int
	keys  []byte
	next  int
}

// Clone returns an independent copy of the native key-to-slot allocation.
// Transactional constructors use it to preflight a future batch without
// changing the battle-session cache observed by the renderer.
func (c *NativeSelectorCache) Clone() *NativeSelectorCache {
	if c == nil {
		return nil
	}
	clone := &NativeSelectorCache{
		keys: append([]byte(nil), c.keys...),
		next: c.next,
	}
	if c.slots != nil {
		clone.slots = make(map[byte]int, len(c.slots))
		for key, slot := range c.slots {
			clone.slots[key] = slot
		}
	}
	return clone
}

// SlotFor returns the stable first-seen slot for a raw FDICON key.
func (c *NativeSelectorCache) SlotFor(key int) (int, error) {
	if key < 0 || key > 0xff {
		return 0, errors.New("fdicon: invalid native selector key")
	}
	if c.slots == nil {
		c.slots = make(map[byte]int)
	}
	b := byte(key)
	if slot, ok := c.slots[b]; ok {
		return slot, nil
	}
	slot := c.next
	c.slots[b] = slot
	c.keys = append(c.keys, b)
	c.next++
	return slot, nil
}

// KeyForSlot resolves a runtime unit+2 cache slot back to the raw B24 key
// whose twelve pointers 0x11019 copied into that slot.
func (c *NativeSelectorCache) KeyForSlot(slot int) (int, error) {
	if slot < 0 || slot >= len(c.keys) {
		return 0, errors.New("fdicon: unknown native selector slot")
	}
	return int(c.keys[slot]), nil
}

// SpriteForNativeSlot mirrors the two native stages: unit+2 chooses a cached
// twelve-pointer block, then pose/cycle chooses one pointer within it.
func (b *Bank) SpriteForNativeSlot(cache *NativeSelectorCache, slot, pose, cycle int) (Sprite, error) {
	if cache == nil {
		return Sprite{}, errors.New("fdicon: nil native selector cache")
	}
	key, err := cache.KeyForSlot(slot)
	if err != nil {
		return Sprite{}, err
	}
	return b.SpriteFor(key, pose, cycle)
}

// Sprite preserves native four-mode RLE effects after decoding. Mask marks
// source writes; RemapMask marks mode-3 spans, which 0x4dcc6 remaps from the
// destination rather than treating as ordinary transparency.
type Sprite struct{ Pixels, Mask, RemapMask []byte }

// BlitConstantMaskAt reproduces 0x4ddd7 after the four-mode source stream has
// been decoded. Modes 0/1/2 are exactly Sprite.Mask writes, including source
// value zero; mode 3 remains untouched. The color is a raw palette index.
func (s Sprite) BlitConstantMaskAt(dst []byte, stride, x, y int, index byte) error {
	if len(s.Mask) != NativeSize*NativeSize || stride <= 0 || x < 0 || y < 0 ||
		x+NativeSize > stride || (y+NativeSize)*stride > len(dst) {
		return errors.New("fdicon: constant-mask destination is incomplete")
	}
	next := append([]byte(nil), dst...)
	for row := 0; row < NativeSize; row++ {
		for col := 0; col < NativeSize; col++ {
			if s.Mask[row*NativeSize+col] != 0 {
				next[(y+row)*stride+x+col] = index
			}
		}
	}
	copy(dst, next)
	return nil
}

// BlitNativeDamageBlendAt reproduces 0x4dc34's write consumer after the
// four-mode RLE stream has been decoded. rawBase is the caller's untouched
// low-byte argument; no palette-band meaning is inferred here.
func (s Sprite) BlitNativeDamageBlendAt(dst []byte, stride, x, y, blend int, rawBase byte) error {
	if len(s.Pixels) != NativeSize*NativeSize || len(s.Mask) != NativeSize*NativeSize ||
		stride <= 0 || x < 0 || y < 0 || x+NativeSize > stride ||
		(y+NativeSize)*stride > len(dst) || blend < 0 || blend > 7 {
		return errors.New("fdicon: damage-blend destination is incomplete")
	}
	next := append([]byte(nil), dst...)
	for row := 0; row < NativeSize; row++ {
		for col := 0; col < NativeSize; col++ {
			i := row*NativeSize + col
			if s.Mask[i] != 0 {
				next[(y+row)*stride+x+col] = byte((int(s.Pixels[i])+blend)&7) + rawBase
			}
		}
	}
	copy(dst, next)
	return nil
}

// SpriteFor implements the native 0x127e0 selector after 0x11019 has built
// its pointer table: group×12 + pose×3 + cycle. Pose is runtime +3; cycle is
// resolved from the global idle/moving counters, not directly from unit +4.
func (b *Bank) SpriteFor(group, pose, cycle int) (Sprite, error) {
	if b == nil || group < 0 || pose < 0 || pose >= 4 || cycle < 0 || cycle >= 3 {
		return Sprite{}, errors.New("fdicon: invalid native sprite selector")
	}
	i := group*12 + pose*3 + cycle
	if i < 0 || i >= len(b.Sprites) {
		return Sprite{}, errors.New("fdicon: sprite selector is out of bank")
	}
	return b.Sprites[i], nil
}

// NativeFrameIndex reproduces 0x127e0's cycle selector. motionOffset is the
// raw unit+4 movement offset: zero selects the idle counter, nonzero selects
// the moving counter. Native counters run 0..3 but map 3 back to frame 1;
// unit+0x26 forces the base frame regardless of either counter.
func NativeFrameIndex(motionOffset int, forceBase bool, idleCycle, movingCycle int) (int, error) {
	if idleCycle < 0 || idleCycle > 3 || movingCycle < 0 || movingCycle > 3 {
		return 0, errors.New("fdicon: invalid native cycle")
	}
	cycle := idleCycle
	if motionOffset != 0 {
		cycle = movingCycle
	}
	if cycle == 3 {
		cycle = 1
	}
	if forceBase {
		cycle = 0
	}
	return cycle, nil
}

// NativePlacementOffset reproduces 0x127e0's byte destination before it is
// added to the native 0x53a49 framebuffer. Map cells are 24×24 indexed pixels
// in a 456-byte stride. motionOffset is unit+4 and advances in byte space in
// the runtime pose direction: down, left, up, right. forceBase is unit+0x26
// nonzero; the original then adds its toggled global pixelShift (0 or 1).
// This returns a byte offset rather than pretending it is a GUI coordinate.
func NativePlacementOffset(x, y, cameraX, cameraY, pose, motionOffset, pixelShift int, forceBase bool) (int, error) {
	if pose < 0 || pose >= 4 || pixelShift < 0 || pixelShift > 1 {
		return 0, errors.New("fdicon: invalid native placement")
	}
	directionOffset := [4]int{NativeSize * NativeMapStride, -4, -NativeSize * NativeMapStride, 4}
	offset := NativeUnitOriginBytes + (y-cameraY)*NativeSize*NativeMapStride + (x-cameraX)*NativeSize + motionOffset*directionOffset[pose]
	if forceBase {
		offset += pixelShift
	}
	return offset, nil
}

func DecodeFile(path string) (*Bank, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(raw)
}

// Parse reads the FDICON.B24 header {u16 width,height,count; u32 offsets[]}.
func Parse(raw []byte) (*Bank, error) {
	if len(raw) < 6 {
		return nil, errors.New("fdicon: file is too short")
	}
	w, h, n := int(binary.LittleEndian.Uint16(raw)), int(binary.LittleEndian.Uint16(raw[2:])), int(binary.LittleEndian.Uint16(raw[4:]))
	if w != NativeSize || h != NativeSize || n == 0 || 6+n*4 > len(raw) {
		return nil, errors.New("fdicon: invalid header")
	}
	bank := &Bank{Sprites: make([]Sprite, n)}
	previous := 6 + n*4
	for i := range bank.Sprites {
		off := int(binary.LittleEndian.Uint32(raw[6+i*4:]))
		end := len(raw)
		if i+1 < n {
			end = int(binary.LittleEndian.Uint32(raw[6+(i+1)*4:]))
		}
		if off < previous || off >= end || end > len(raw) {
			return nil, fmt.Errorf("fdicon: invalid sprite %d offset", i)
		}
		pixels, mask, remapMask, err := decode(raw[off:end])
		if err != nil {
			return nil, fmt.Errorf("fdicon: sprite %d: %w", i, err)
		}
		bank.Sprites[i] = Sprite{Pixels: pixels, Mask: mask, RemapMask: remapMask}
		previous = off
	}
	return bank, nil
}

func decode(src []byte) ([]byte, []byte, []byte, error) {
	pixels, mask, remapMask := make([]byte, NativeSize*NativeSize), make([]byte, NativeSize*NativeSize), make([]byte, NativeSize*NativeSize)
	p := 0
	for y := 0; y < NativeSize; y++ {
		x := 0
		for x < NativeSize {
			if p >= len(src) {
				return nil, nil, nil, errors.New("RLE ends early")
			}
			c := src[p]
			p++
			count, mode := int(c&0x3f)+1, c>>6
			span := count
			if mode == 1 {
				span *= 2
			}
			if x+span > NativeSize {
				return nil, nil, nil, errors.New("RLE overruns row")
			}
			write := func(at int, v byte) { pixels[y*NativeSize+at], mask[y*NativeSize+at] = v, 1 }
			switch mode {
			case 0:
				if p >= len(src) {
					return nil, nil, nil, errors.New("run lacks value")
				}
				v := src[p]
				p++
				for i := 0; i < count; i++ {
					write(x+i, v)
				}
			case 1:
				if p >= len(src) {
					return nil, nil, nil, errors.New("dither lacks value")
				}
				v := src[p]
				p++
				for i := 0; i < count; i++ {
					write(x+2*i+1, v)
				}
			case 2:
				if p+count > len(src) {
					return nil, nil, nil, errors.New("literal exceeds data")
				}
				for i, v := range src[p : p+count] {
					write(x+i, v)
				}
				p += count
			case 3:
				for i := 0; i < count; i++ {
					remapMask[y*NativeSize+x+i] = 1
				}
			}
			x += span
		}
	}
	return pixels, mask, remapMask, nil
}

// BlitAt is native 0x4deda: raw indexed RLE, preserving transparent spans.
func (s Sprite) BlitAt(dst []byte, stride, x, y int) error {
	return s.blit(dst, stride, x, y, false)
}

// BlitPaletteBand is native 0x4de56. It maps each opaque source index to
// (index & 7) + 0x18; it is not a general 256-byte LUT path.
func (s Sprite) BlitPaletteBand(dst []byte, stride, x, y int) error {
	return s.blit(dst, stride, x, y, true)
}

// BlitForNativeFlags mirrors 0x127e0's test of runtime unit+5 bit7: clear
// selects 0x4deda raw pixels, set selects 0x4de56's palette-band pixels.
func (s Sprite) BlitForNativeFlags(dst []byte, stride, x, y int, flags byte) error {
	return s.blit(dst, stride, x, y, flags&0x80 != 0)
}

// BlitForNativeFlagsAtOffset is the same 0x127e0 raw/palette-band branch as
// BlitForNativeFlags, but accepts the already-computed native work-buffer byte
// offset.  The original routine passes a pointer rather than a clipped (x,y)
// pair, so this form deliberately permits a 24-pixel span to cross a stride
// boundary while still rejecting an out-of-buffer write.
func (s Sprite) BlitForNativeFlagsAtOffset(dst []byte, stride, offset int, flags byte) error {
	return s.blitAtOffset(dst, stride, offset, flags&0x80 != 0)
}

// BlitLUT reproduces 0x4dcc6. Source-written RLE pixels are translated through
// lut; mode-3 spans translate the existing destination index through lut.
// Dither holes remain untouched, as in the native mode-1 loop.
func (s Sprite) BlitLUT(dst []byte, stride, x, y int, lut []byte) error {
	if len(lut) != 256 || len(s.Pixels) != NativeSize*NativeSize || len(s.Mask) != len(s.Pixels) || len(s.RemapMask) != len(s.Pixels) || stride < x+NativeSize || x < 0 || y < 0 || y+NativeSize > len(dst)/stride {
		return errors.New("fdicon: invalid LUT blit")
	}
	for row := 0; row < NativeSize; row++ {
		for col := 0; col < NativeSize; col++ {
			i := row*NativeSize + col
			d := (y+row)*stride + x + col
			if s.Mask[i] != 0 {
				dst[d] = lut[s.Pixels[i]]
			} else if s.RemapMask[i] != 0 {
				dst[d] = lut[dst[d]]
			}
		}
	}
	return nil
}

// BlitLUTTransparent reproduces 0x4dd52: source writes are LUT-mapped while
// mode-3 spans preserve the destination (unlike 0x4dcc6's BlitLUT).
func (s Sprite) BlitLUTTransparent(dst []byte, stride, x, y int, lut []byte) error {
	if len(lut) != 256 || len(s.Pixels) != NativeSize*NativeSize || len(s.Mask) != len(s.Pixels) || stride < x+NativeSize || x < 0 || y < 0 || y+NativeSize > len(dst)/stride {
		return errors.New("fdicon: invalid transparent LUT blit")
	}
	for row := 0; row < NativeSize; row++ {
		for col := 0; col < NativeSize; col++ {
			i := row*NativeSize + col
			if s.Mask[i] != 0 {
				dst[(y+row)*stride+x+col] = lut[s.Pixels[i]]
			}
		}
	}
	return nil
}

// BlitLUTTransparentAtOffset is 0x4dd52 with the native pointer-style
// destination used by 0x12ac6.  Mode-3 spans remain untouched.
func (s Sprite) BlitLUTTransparentAtOffset(dst []byte, stride, offset int, lut []byte) error {
	if len(lut) != 256 || len(s.Pixels) != NativeSize*NativeSize || len(s.Mask) != len(s.Pixels) || stride <= 0 || offset < 0 || offset+(NativeSize-1)*stride+NativeSize > len(dst) {
		return errors.New("fdicon: invalid transparent LUT offset blit")
	}
	for row := 0; row < NativeSize; row++ {
		for col := 0; col < NativeSize; col++ {
			i := row*NativeSize + col
			if s.Mask[i] != 0 {
				dst[offset+row*stride+col] = lut[s.Pixels[i]]
			}
		}
	}
	return nil
}

func (s Sprite) blit(dst []byte, stride, x, y int, paletteBand bool) error {
	if len(s.Pixels) != NativeSize*NativeSize || len(s.Mask) != len(s.Pixels) || stride < x+NativeSize || x < 0 || y < 0 || y+NativeSize > len(dst)/stride {
		return errors.New("fdicon: invalid blit")
	}
	for row := 0; row < NativeSize; row++ {
		for col := 0; col < NativeSize; col++ {
			i := row*NativeSize + col
			if s.Mask[i] != 0 {
				v := s.Pixels[i]
				if paletteBand {
					v = (v & 7) + 0x18
				}
				dst[(y+row)*stride+x+col] = v
			}
		}
	}
	return nil
}

func (s Sprite) blitAtOffset(dst []byte, stride, offset int, paletteBand bool) error {
	if len(s.Pixels) != NativeSize*NativeSize || len(s.Mask) != len(s.Pixels) || stride <= 0 || offset < 0 || offset+(NativeSize-1)*stride+NativeSize > len(dst) {
		return errors.New("fdicon: invalid offset blit")
	}
	for row := 0; row < NativeSize; row++ {
		for col := 0; col < NativeSize; col++ {
			i := row*NativeSize + col
			if s.Mask[i] != 0 {
				v := s.Pixels[i]
				if paletteBand {
					v = (v & 7) + 0x18
				}
				dst[offset+row*stride+col] = v
			}
		}
	}
	return nil
}
