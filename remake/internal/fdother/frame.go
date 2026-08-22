// Package fdother decodes the frame-table resources used by FD2's indexed
// compositor.  It deliberately exposes indexed pixels only: palette and UI
// scheduling belong to their respective callers.
//
// The format and RLE below are direct translations of FD2.EXE 0x2935b and
// 0x4e63d.  In particular, transparent commands preserve the destination;
// they must not be converted to palette index zero as a PNG exporter would.
package fdother

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

// Frame is one FDOTHER frame-table descriptor. X/Y are its destination offset
// embedded in the resource; Pixels starts with little-endian Width/Height.
type Frame struct {
	X, Y          int
	Width, Height int
	Pixels        []byte
}

// DecodeResource opens a player-provided LLLLLL .DAT archive and parses one
// raw FDOTHER resource. It mirrors 0x111ba's archive-only loading boundary:
// no conversion or decompression is performed here.
func DecodeResource(datPath string, resource int) ([]Frame, error) {
	entry, err := ReadResource(datPath, resource)
	if err != nil {
		return nil, err
	}
	return ParseFrames(entry)
}

// DecodeArchiveSingleFrame reads one LLLLLL archive entry whose payload is a
// width/height header followed directly by 0x4e63d four-mode RLE.  BG.DAT
// uses this form; keeping it separate from ParseFrames avoids inventing an
// offset table where the native loader has none.
func DecodeArchiveSingleFrame(datPath string, resource int) (Frame, error) {
	entry, err := ReadResource(datPath, resource)
	if err != nil {
		return Frame{}, err
	}
	return ParseSingleFrame(entry)
}

// ReadResource returns one raw LLLLLL archive entry without assuming a frame
// table. FDOTHER #4 (the native FDTXT bitmap font) is one such non-frame
// payload; its decoding belongs to internal/fdtxt.
func ReadResource(datPath string, resource int) ([]byte, error) {
	data, err := os.ReadFile(datPath)
	if err != nil {
		return nil, err
	}
	entry, err := ArchiveEntry(data, resource)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), entry...), nil
}

// ReadNestedResource preserves the two 0x111BA boundaries used by battle
// sound banks such as command0's FDOTHER #82/sub1. It returns raw bytes only;
// sample rate, channel layout and playback remain caller-owned evidence.
func ReadNestedResource(datPath string, resource, nested int) ([]byte, error) {
	outer, err := ReadResource(datPath, resource)
	if err != nil {
		return nil, err
	}
	entry, err := ArchiveEntry(outer, nested)
	if err != nil {
		return nil, fmt.Errorf("fdother: nested resource %d/%d: %w", resource, nested, err)
	}
	return append([]byte(nil), entry...), nil
}

// ArchiveEntry extracts one raw entry from an LLLLLL archive byte slice.
// FDOTHER #81 is itself a nested LLLLLL archive; exposing this boundary lets
// callers validate nested provenance without pretending its payload is a
// normal FDOTHER frame table.
func ArchiveEntry(data []byte, resource int) ([]byte, error) {
	if len(data) < 10 || string(data[:6]) != "LLLLLL" {
		return nil, errors.New("fdother: missing LLLLLL archive magic")
	}
	first := int(binary.LittleEndian.Uint32(data[6:]))
	if first < 10 || first > len(data) || (first-6)%4 != 0 {
		return nil, errors.New("fdother: invalid archive directory")
	}
	count := (first - 6) / 4
	if resource < 0 || resource >= count {
		return nil, errors.New("fdother: archive resource is out of range")
	}
	start := int(binary.LittleEndian.Uint32(data[6+4*resource:]))
	end := len(data)
	if resource+1 < count {
		end = int(binary.LittleEndian.Uint32(data[6+4*(resource+1):]))
	}
	if start < first || start > end || end > len(data) {
		return nil, errors.New("fdother: invalid archive resource bounds")
	}
	return data[start:end], nil
}

func archiveEntry(data []byte, resource int) ([]byte, error) {
	return ArchiveEntry(data, resource)
}

// ParseFrames parses the raw archive entry returned by FD2's 0x111ba loader.
// Only fields consumed by 0x2935b are accepted; malformed offsets and frame
// bounds are rejected rather than guessed.
func ParseFrames(data []byte) ([]Frame, error) {
	if len(data) < 8 {
		return nil, errors.New("fdother: frame table is too short")
	}
	n := int(binary.LittleEndian.Uint16(data))
	if n == 0 || 8+4*n > len(data) {
		return nil, errors.New("fdother: invalid frame count/table")
	}
	frames := make([]Frame, n)
	previous := 8 + 4*n
	for i := 0; i < n; i++ {
		off := int(binary.LittleEndian.Uint32(data[8+4*i:]))
		end := len(data)
		if i+1 < n {
			end = int(binary.LittleEndian.Uint32(data[8+4*(i+1):]))
		}
		if off < previous || off+13 > end || end > len(data) {
			return nil, fmt.Errorf("fdother: frame %d offset is invalid", i)
		}
		width := int(binary.LittleEndian.Uint16(data[off+9:]))
		height := int(binary.LittleEndian.Uint16(data[off+11:]))
		if width == 0 || height == 0 {
			return nil, fmt.Errorf("fdother: frame %d has empty dimensions", i)
		}
		frames[i] = Frame{
			X:      int(binary.LittleEndian.Uint16(data[off:])),
			Y:      int(binary.LittleEndian.Uint16(data[off+2:])),
			Width:  width,
			Height: height,
			Pixels: data[off+9 : end],
		}
		previous = off
	}
	return frames, nil
}

// ParseSingleFrame parses a raw 0x4e63d payload with Width/Height at offset
// zero and no offset table (the finale's FDOTHER #56 form).
func ParseSingleFrame(data []byte) (Frame, error) {
	if len(data) < 4 {
		return Frame{}, errors.New("fdother: single frame is too short")
	}
	w, h := int(binary.LittleEndian.Uint16(data)), int(binary.LittleEndian.Uint16(data[2:]))
	if w == 0 || h == 0 {
		return Frame{}, errors.New("fdother: single frame has empty dimensions")
	}
	return Frame{Width: w, Height: h, Pixels: data}, nil
}

// Blit applies the exact transparent (-1) branch of FD2.EXE 0x4e63d to dst.
// dst is an indexed framebuffer with the given row stride.  The original
// supports two non-transparent palette remap modes too; no verified ending
// caller uses them, so this adapter rejects them rather than inventing rules.
func (f Frame) Blit(dst []byte, stride, transparent int) error {
	return f.BlitAt(dst, stride, 0, transparent)
}

// BlitAt is the same verified transparent decoder with an explicit byte
// origin inside a larger surface (used by 640-wide native ending work buffers).
func (f Frame) BlitAt(dst []byte, stride, base, transparent int) error {
	if transparent != -1 {
		return errors.New("fdother: only verified transparent blit is supported")
	}
	if f.X < 0 || f.Y < 0 || f.Width <= 0 || f.Height <= 0 || stride < f.X+f.Width {
		return errors.New("fdother: invalid destination geometry")
	}
	if base < 0 || base > len(dst) || f.Y > (len(dst)-base-f.X)/stride || f.Height > (len(dst)-base)/stride-f.Y {
		return errors.New("fdother: destination is too small")
	}
	pos := 4 // 0x4e63d consumes Width/Height from the payload before RLE.
	for y := 0; y < f.Height; y++ {
		row, written := base+(f.Y+y)*stride+f.X, 0
		for written < f.Width {
			if pos >= len(f.Pixels) {
				return fmt.Errorf("fdother: frame RLE ends in row %d", y)
			}
			ctrl := f.Pixels[pos]
			pos++
			count := int(ctrl&0x3f) + 1
			mode := ctrl >> 6
			advance := count
			if mode == 1 {
				advance *= 2
			}
			if written+advance > f.Width {
				return fmt.Errorf("fdother: frame RLE overruns row %d", y)
			}
			switch mode {
			case 0: // colour run
				if pos >= len(f.Pixels) {
					return errors.New("fdother: colour run lacks a value")
				}
				v := f.Pixels[pos]
				pos++
				for i := 0; i < count; i++ {
					dst[row+written+i] = v
				}
			case 1: // dither: preserve, colour, preserve, colour ...
				if pos >= len(f.Pixels) {
					return errors.New("fdother: dither run lacks a value")
				}
				v := f.Pixels[pos]
				pos++
				for i := 0; i < count; i++ {
					dst[row+written+2*i+1] = v
				}
			case 2: // literal
				if pos+count > len(f.Pixels) {
					return errors.New("fdother: literal run exceeds frame data")
				}
				copy(dst[row+written:row+written+count], f.Pixels[pos:pos+count])
				pos += count
			case 3: // transparent skip: leave the destination unchanged
			}
			written += advance
		}
	}
	return nil
}
