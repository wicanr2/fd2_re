package fdother

import (
	"encoding/binary"
	"errors"
)

// DecodeLMI1FrameResource reads one LMI1 directory entry whose native caller
// sends it to 0x4e63d.  LMI1 is a directory, not a universal pixel codec:
// this path deliberately remains separate from ParseLMI1's 0x4e916 cells.
func DecodeLMI1FrameResource(datPath string, resource, index int) (Frame, error) {
	data, err := ReadResource(datPath, resource)
	if err != nil {
		return Frame{}, err
	}
	return ParseLMI1FrameEntry(data, index)
}

// ParseLMI1FrameEntry extracts a single LMI1 entry with a width/height header
// followed by the four-mode RLE consumed by 0x4e63d.  It does not infer that
// neighbouring entries use the same codec.
func ParseLMI1FrameEntry(data []byte, index int) (Frame, error) {
	if len(data) < 6 || string(data[:4]) != "LMI1" {
		return Frame{}, errors.New("fdother: missing LMI1 magic")
	}
	count := int(binary.LittleEndian.Uint16(data[4:]))
	tableEnd := 6 + count*4
	if count == 0 || tableEnd > len(data) || index < 0 || index >= count {
		return Frame{}, errors.New("fdother: LMI1 frame index is invalid")
	}
	off := int(binary.LittleEndian.Uint32(data[6+index*4:]))
	if off < tableEnd || off+4 > len(data) {
		return Frame{}, errors.New("fdother: LMI1 frame offset is invalid")
	}
	return ParseSingleFrame(data[off:])
}

// DecodeLMI1RawEntry reads the distinct 0x1685c→0x4e9bb path. The LMI1
// directory is reused, but its selected cell is copied as raw width×height
// bytes; no 0x4e916/0x4e63d run codec is applied.
func DecodeLMI1RawEntry(datPath string, resource, index int) (RawCell, error) {
	data, err := ReadResource(datPath, resource)
	if err != nil {
		return RawCell{}, err
	}
	return ParseLMI1RawEntry(data, index)
}

func ParseLMI1RawEntry(data []byte, index int) (RawCell, error) {
	if len(data) < 6 || string(data[:4]) != "LMI1" {
		return RawCell{}, errors.New("fdother: missing LMI1 magic")
	}
	count := int(binary.LittleEndian.Uint16(data[4:]))
	tableEnd := 6 + count*4
	if count == 0 || tableEnd > len(data) || index < 0 || index >= count {
		return RawCell{}, errors.New("fdother: LMI1 raw entry index is invalid")
	}
	off := int(binary.LittleEndian.Uint32(data[6+index*4:]))
	end := len(data)
	if index+1 < count {
		end = int(binary.LittleEndian.Uint32(data[6+(index+1)*4:]))
	}
	if off < tableEnd || off+4 > end || end > len(data) {
		return RawCell{}, errors.New("fdother: LMI1 raw entry bounds are invalid")
	}
	w := int(binary.LittleEndian.Uint16(data[off:]))
	h := int(binary.LittleEndian.Uint16(data[off+2:]))
	if w <= 0 || h <= 0 || w > (end-off-4)/h {
		return RawCell{}, errors.New("fdother: LMI1 raw entry geometry is invalid")
	}
	pixels := append([]byte(nil), data[off+4:off+4+w*h]...)
	return RawCell{Width: w, Height: h, Pixels: pixels}, nil
}

// ParseLMI1OpaqueEntry extracts one LMI1 directory entry whose caller sends
// its width/height payload to 0x4e8af. That primitive decodes the 0x4e916
// high-run stream and writes every pixel, including palette index zero.
func ParseLMI1OpaqueEntry(data []byte, index int) (LMI1Entry, error) {
	if len(data) < 6 || string(data[:4]) != "LMI1" {
		return LMI1Entry{}, errors.New("fdother: missing LMI1 magic")
	}
	count := int(binary.LittleEndian.Uint16(data[4:]))
	tableEnd := 6 + count*4
	if count == 0 || tableEnd > len(data) || index < 0 || index >= count {
		return LMI1Entry{}, errors.New("fdother: LMI1 opaque entry index is invalid")
	}
	off := int(binary.LittleEndian.Uint32(data[6+index*4:]))
	if off < tableEnd || off+4 > len(data) {
		return LMI1Entry{}, errors.New("fdother: LMI1 opaque entry offset is invalid")
	}
	return ParseOpaqueRunCell(data[off:])
}
