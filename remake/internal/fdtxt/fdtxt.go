// Package fdtxt decodes the original FD2 FDTXT word table and its companion
// 16x16 bitmap font.  It deliberately stops before interpreting scene-only
// control codes: callers receive those words unchanged rather than silently
// turning an original page break or portrait switch into plain text.
package fdtxt

import (
	"encoding/binary"
	"fmt"
)

const (
	GlyphWidth   = 16
	GlyphHeight  = 16
	GlyphBytes   = 32
	StringEnd    = 0xffff
	ControlMin   = 0xff00
	OpeningGlyph = 557
)

// Strings is an offset-table FDTXT resource. Every word slice excludes its
// terminal 0xffff but retains all FFxx control codes.
type Strings struct{ words [][]uint16 }

func Parse(data []byte) (*Strings, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("fdtxt: resource shorter than first offset")
	}
	first := int(binary.LittleEndian.Uint16(data[:2]))
	if first == 0 || first%2 != 0 || first > len(data) {
		return nil, fmt.Errorf("fdtxt: invalid first offset %#x", first)
	}
	count := first / 2
	offsets := make([]int, count)
	for i := range offsets {
		offsets[i] = int(binary.LittleEndian.Uint16(data[i*2:]))
		if offsets[i] < first || offsets[i] > len(data) || (offsets[i]%2) != 0 || (i > 0 && offsets[i] < offsets[i-1]) {
			return nil, fmt.Errorf("fdtxt: invalid offset at string %d", i)
		}
	}
	out := make([][]uint16, count)
	for i, start := range offsets {
		end := len(data)
		if i+1 < len(offsets) {
			end = offsets[i+1]
		}
		words := make([]uint16, 0, (end-start)/2)
		for p := start; p < end; p += 2 {
			word := binary.LittleEndian.Uint16(data[p:])
			if word == StringEnd {
				break
			}
			words = append(words, word)
		}
		out[i] = words
	}
	return &Strings{words: out}, nil
}

func (s *Strings) Count() int { return len(s.words) }

// Words returns an independent copy, making edits to an editable remake
// script unable to mutate its original-resource evidence.
func (s *Strings) Words(index int) ([]uint16, error) {
	if s == nil || index < 0 || index >= len(s.words) {
		return nil, fmt.Errorf("fdtxt: string index %d out of range", index)
	}
	return append([]uint16(nil), s.words[index]...), nil
}

// LogicalLocator identifies one visible utterance inside the physical FDTXT
// offset-table strings. A native selector can refer to this logical sequence;
// it must not be assumed to be a physical offset-table index.
type LogicalLocator struct {
	RawStringIndex    int
	RawUtteranceIndex int
}

// LocateLogicalUtterance follows the same structural rule used by the
// count-aligned story exporter: an utterance begins when a non-control chunk
// starts with a speaker operand followed by the opening-quote glyph. FFxx
// words delimit chunks and are deliberately not reinterpreted here.
func (s *Strings) LocateLogicalUtterance(index int) (LogicalLocator, error) {
	if s == nil || index < 0 {
		return LogicalLocator{}, fmt.Errorf("fdtxt: logical utterance index %d out of range", index)
	}
	seen := 0
	for rawIndex, words := range s.words {
		chunk := make([]uint16, 0)
		local := 0
		for _, word := range append(append([]uint16(nil), words...), StringEnd) {
			if word < ControlMin {
				chunk = append(chunk, word)
				continue
			}
			if len(chunk) >= 2 && chunk[1] == OpeningGlyph {
				if seen == index {
					return LogicalLocator{RawStringIndex: rawIndex, RawUtteranceIndex: local}, nil
				}
				seen++
				local++
			}
			chunk = chunk[:0]
		}
	}
	return LogicalLocator{}, fmt.Errorf("fdtxt: logical utterance index %d out of range", index)
}

// Font is FDOTHER resource #4: packed 16x16, 1bpp glyphs with MSB at the
// left edge of each row.
type Font struct{ data []byte }

func ParseFont(data []byte) (*Font, error) {
	if len(data) == 0 || len(data)%GlyphBytes != 0 {
		return nil, fmt.Errorf("fdtxt: invalid 16x16 font length %d", len(data))
	}
	return &Font{data: append([]byte(nil), data...)}, nil
}

func (f *Font) GlyphCount() int {
	if f == nil {
		return 0
	}
	return len(f.data) / GlyphBytes
}

// GlyphBit reports the exact 1bpp source bit; it does not impose a palette
// colour or interpret an FFxx control word.
func (f *Font) GlyphBit(index, x, y int) (bool, error) {
	if f == nil || index < 0 || index >= f.GlyphCount() || x < 0 || x >= GlyphWidth || y < 0 || y >= GlyphHeight {
		return false, fmt.Errorf("fdtxt: glyph coordinate out of range")
	}
	row := binary.BigEndian.Uint16(f.data[index*GlyphBytes+y*2:])
	return row&(uint16(0x8000)>>x) != 0, nil
}

// BlitGlyph writes only set source bits to an indexed destination. Zero bits
// are transparent, exactly matching the source font's 1bpp mask semantics.
// Palette choice belongs to the still-unrecovered 0x4ea2a call ABI, so callers
// must supply it explicitly rather than inheriting a guessed UI colour.
func (f *Font) BlitGlyph(dst []byte, stride, base, index int, color byte) error {
	if f == nil || index < 0 || index >= f.GlyphCount() || stride < GlyphWidth || base < 0 {
		return fmt.Errorf("fdtxt: invalid glyph blit")
	}
	for y := 0; y < GlyphHeight; y++ {
		start := base + y*stride
		if start+GlyphWidth > len(dst) {
			return fmt.Errorf("fdtxt: glyph blit exceeds destination")
		}
		row := binary.BigEndian.Uint16(f.data[index*GlyphBytes+y*2:])
		for x := 0; x < GlyphWidth; x++ {
			if row&(uint16(0x8000)>>x) != 0 {
				dst[start+x] = color
			}
		}
	}
	return nil
}

// NativeGlyphStyle is the exact visible colour portion of 0x4ea2a's ABI.
// Foreground is written for each set 1bpp source bit; Shadow is written one
// pixel to its left and one row below; Background fills the 16x16 cell only
// when it is non-zero.
type NativeGlyphStyle struct {
	Foreground byte
	Shadow     byte
	Background byte
}

// BlitNativeGlyph reproduces 0x4ea2a's 16x16 loop.  Its bounds checks are
// intentionally stricter than the original raw pointer writes: a glyph needs
// one pixel of left margin and one shadow row below the cell.
func (f *Font) BlitNativeGlyph(dst []byte, stride, base, index int, style NativeGlyphStyle) error {
	if f == nil || index < 0 || index >= f.GlyphCount() || stride < GlyphWidth || base < 1 {
		return fmt.Errorf("fdtxt: invalid native glyph blit")
	}
	if base+GlyphHeight*stride+GlyphWidth > len(dst) {
		return fmt.Errorf("fdtxt: native glyph blit exceeds destination")
	}
	if style.Background != 0 {
		for y := 0; y < GlyphHeight; y++ {
			for x := 0; x < GlyphWidth; x++ {
				dst[base+y*stride+x] = style.Background
			}
		}
	}
	for y := 0; y < GlyphHeight; y++ {
		row := binary.BigEndian.Uint16(f.data[index*GlyphBytes+y*2:])
		for x := 0; x < GlyphWidth; x++ {
			if row&(uint16(0x8000)>>x) == 0 {
				continue
			}
			pos := base + y*stride + x
			dst[pos] = style.Foreground
			dst[pos-1] = style.Shadow
			dst[pos+stride] = style.Shadow
		}
	}
	return nil
}
