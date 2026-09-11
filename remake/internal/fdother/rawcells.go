package fdother

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// RawCell is an uncompressed indexed cell stored in FDOTHER's untagged
// offset-bank form. FDOTHER #2 uses this form for the native action overlay.
// Palette index zero preserves the destination, matching 0x4e9e4.
type RawCell struct {
	Width, Height int
	Pixels        []byte
}

// ParseRawCell decodes one width/height + width*height literal indexed cell.
// It is the direct resource-entry form consumed by 0x4e9e4; palette index
// zero remains transparent at blit time.
func ParseRawCell(data []byte) (RawCell, error) {
	if len(data) < 4 {
		return RawCell{}, errors.New("fdother: raw cell is too short")
	}
	width := int(binary.LittleEndian.Uint16(data))
	height := int(binary.LittleEndian.Uint16(data[2:]))
	if width <= 0 || height <= 0 || width > (len(data)-4)/height {
		return RawCell{}, errors.New("fdother: raw cell geometry is invalid")
	}
	pixels := append([]byte(nil), data[4:4+width*height]...)
	return RawCell{Width: width, Height: height, Pixels: pixels}, nil
}

// DecodeRawCellResource reads a raw FDOTHER archive entry whose first u32 is
// the byte end of a u32-offset directory. It deliberately does not try this
// parser for LMI1 or frame-table resources.
func DecodeRawCellResource(datPath string, resource int) ([]RawCell, error) {
	data, err := ReadResource(datPath, resource)
	if err != nil {
		return nil, err
	}
	return ParseRawCellBank(data)
}

// ParseRawCellBank parses the untagged directory used by FDOTHER #2:
//
//	+0  u32 directory_end (= first cell offset)
//	+4  u32 next cell offset
//	...
//	cell: u16 width, u16 height, width*height raw indexed pixels
//
// Native 0x4e9e4 consumes exactly width*height bytes, so directory offsets
// define cell starts rather than compressed stream ends.
func ParseRawCellBank(data []byte) ([]RawCell, error) {
	if len(data) < 4 {
		return nil, errors.New("fdother: raw cell bank is too short")
	}
	directoryEnd := int(binary.LittleEndian.Uint32(data[:4]))
	if directoryEnd < 4 || directoryEnd%4 != 0 || directoryEnd > len(data) {
		return nil, errors.New("fdother: invalid raw cell directory")
	}
	count := directoryEnd / 4
	cells := make([]RawCell, count)
	previous := directoryEnd
	for i := 0; i < count; i++ {
		off := int(binary.LittleEndian.Uint32(data[4*i:]))
		if off < previous || off+4 > len(data) {
			return nil, fmt.Errorf("fdother: raw cell %d offset is invalid", i)
		}
		width := int(binary.LittleEndian.Uint16(data[off:]))
		height := int(binary.LittleEndian.Uint16(data[off+2:]))
		if width <= 0 || height <= 0 || width > (len(data)-off-4)/height {
			return nil, fmt.Errorf("fdother: raw cell %d geometry is invalid", i)
		}
		end := off + 4 + width*height
		if end > len(data) {
			return nil, fmt.Errorf("fdother: raw cell %d pixels exceed resource", i)
		}
		cells[i] = RawCell{Width: width, Height: height, Pixels: data[off+4 : end]}
		previous = off
	}
	return cells, nil
}

// BlitAt applies 0x4e9e4's direct indexed copy. Zero source pixels leave the
// destination intact; the routine has no inferred palette conversion.
func (c RawCell) BlitAt(dst []byte, stride, x, y int) error {
	if c.Width <= 0 || c.Height <= 0 || len(c.Pixels) != c.Width*c.Height {
		return errors.New("fdother: invalid raw cell")
	}
	if x < 0 || y < 0 || stride < x+c.Width || y > len(dst)/stride || c.Height > (len(dst)-y*stride)/stride {
		return errors.New("fdother: raw cell destination is too small")
	}
	for row := 0; row < c.Height; row++ {
		for col := 0; col < c.Width; col++ {
			v := c.Pixels[row*c.Width+col]
			if v != 0 {
				dst[(y+row)*stride+x+col] = v
			}
		}
	}
	return nil
}

// BlitAtClipped 是 BlitAt 的裁邊版本，給起點落在視窗外的原生滑動用。
//
// 可見游標（`[0x53AB9]`／`[0x53ABD]`）在原版是自由的 dword，八個寫入端沒有一個檢查
// 範圍——走行步進在鏡頭還能捲時照樣遞減，收據量到過 -1。對白收框的滑動終點就是那個
// 值，所以終點在畫面外是合法狀態，不是錯誤。嚴格的 BlitAt 契約維持不變。
func (c RawCell) BlitAtClipped(dst []byte, stride, x, y int) error {
	if c.Width <= 0 || c.Height <= 0 || len(c.Pixels) != c.Width*c.Height {
		return errors.New("fdother: invalid raw cell")
	}
	if stride <= 0 || len(dst) < stride {
		return errors.New("fdother: clipped raw cell surface is invalid")
	}
	// 整格落在畫面外是**合法**的中間狀態，不是錯誤。滑動終點是可見游標，而可見
	// 游標可以是負的；x = 24×(-3)+4 = -68 時整格在左界外，那一幀原版也只是什麼
	// 都沒畫。回錯誤等於把合法狀態當失敗，整場停在那裡。
	if y >= len(dst)/stride || y+c.Height <= 0 || x >= stride || x+c.Width <= 0 {
		return nil
	}
	for row := 0; row < c.Height; row++ {
		dy := y + row
		if dy < 0 || dy >= len(dst)/stride {
			continue
		}
		for col := 0; col < c.Width; col++ {
			dx := x + col
			if dx < 0 || dx >= stride {
				continue
			}
			if v := c.Pixels[row*c.Width+col]; v != 0 {
				dst[dy*stride+dx] = v
			}
		}
	}
	return nil
}

// BlitOpaqueAtOffset reproduces 0x4e9bb's direct row copy. Unlike BlitAt,
// zero bytes are written as literal indexed pixels because this is the
// FDOTHER#5 dialogue-frame path, not the transparent 0x4e9e4 path.
func (c RawCell) BlitOpaqueAtOffset(dst []byte, stride, offset int) error {
	if c.Width <= 0 || c.Height <= 0 || len(c.Pixels) != c.Width*c.Height || stride <= 0 || offset < 0 || offset%stride+c.Width > stride || offset > len(dst) || c.Height > (len(dst)-offset)/stride {
		return errors.New("fdother: opaque raw cell destination is too small")
	}
	for row := 0; row < c.Height; row++ {
		copy(dst[offset+row*stride:offset+row*stride+c.Width], c.Pixels[row*c.Width:(row+1)*c.Width])
	}
	return nil
}
