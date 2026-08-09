package fdicon

import "errors"

// NativeTerrainCell is one decoded FDFIELD composition cell. Tile is the raw
// terrain word; BlitMode is composition entry byte+3 (event word high byte).
type NativeTerrainCell struct {
	Tile     uint16
	BlitMode byte
}

// NativeTerrainCursorInfo is the eight-byte result produced by 0x12e38 for
// one FDFIELD composition cell. Tile is the composition word masked to ten
// bits, EventLow is the next byte masked to five bits, and Control is the raw
// four-byte FDSHAP terrain record selected by that tile. The native map HUD
// consumes these values before it optionally looks up a unit at the cursor.
type NativeTerrainCursorInfo struct {
	Tile     uint16
	EventLow byte
	Control  [4]byte
}

// NativeMapHUDLayout is the byte-offset contract of 0x1acf3 inside its
// 456-byte-stride work buffer.  It describes only its proven compositor
// destinations; resource selection and decimal glyph rendering remain the
// responsibility of their separate native primitives.
type NativeMapHUDLayout struct {
	Frame, Terrain int
	AP, DP         int
	Unit, HP       int
}

// NativeCommandBackgroundTarget is the raw per-final-target input consumed
// by 0x2b5e1. Gate is the unlabelled 0x1f183 result; Control is the selected
// four-byte FDSHAP record produced by 0x12e38.
type NativeCommandBackgroundTarget struct {
	Gate    bool
	Control [4]byte
}

// NativeCommandBackgroundSelector reproduces 0x2b5e1's reverse final-target
// scan. It retains the initial selector while a target passes the raw gate
// and it is nonzero; otherwise it replaces it with decoded control byte+2.
// The byte's scene/terrain meaning is deliberately not inferred here.
func NativeCommandBackgroundSelector(initial byte, targets []NativeCommandBackgroundTarget) byte {
	selector := initial
	for i := len(targets) - 1; i >= 0; i-- {
		target := targets[i]
		if !target.Gate || selector == 0 {
			selector = target.Control[2]
		}
	}
	return selector
}

// NativeMapHUDLayoutFor reproduces the fixed offsets used by 0x1acf3 after
// its raw horizontal anchor has been selected.  The original caller always
// supplies the 456-byte map work-buffer stride; accepting another stride
// would hide an adapter mismatch, so it fails closed.
func NativeMapHUDLayoutFor(anchorX, stride int) (NativeMapHUDLayout, error) {
	if stride != NativeMapStride || anchorX < 0 || anchorX+69 > 320 {
		return NativeMapHUDLayout{}, errors.New("fdicon: invalid native map HUD layout")
	}
	base := stride*157 + anchorX
	return NativeMapHUDLayout{
		Frame:   base,
		// 0x1adbf computes ebp + 5*stride + 6 for the terrain icon;
		// it shares the same row-5 slot as the optional unit icon.
		Terrain: base + NativeMapStride*5 + 6,
		AP:      base + stride*8 + 0x2b,
		DP:      base + stride*19 + 0x2b,
		Unit:    base + stride*5 + 6,
		HP:      base + stride*21 + 9,
	}, nil
}

// NativeTerrainCursorInfoForCell reproduces 0x12e38's decode of one raw
// FDFIELD tile/event pair. controls must be the selected FDSHAP control table
// in four-byte records; malformed input is rejected rather than guessed.
func NativeTerrainCursorInfoForCell(tileWord, eventWord uint16, controls []byte) (NativeTerrainCursorInfo, error) {
	tile := tileWord & 0x3ff
	off := int(tile) * 4
	if off+4 > len(controls) {
		return NativeTerrainCursorInfo{}, errors.New("fdicon: terrain cursor control table is too short")
	}
	info := NativeTerrainCursorInfo{Tile: tile, EventLow: byte(eventWord) & 0x1f}
	copy(info.Control[:], controls[off:off+4])
	return info, nil
}

// NativeMapHUDUnitFrameIndex reproduces 0x1ae4d's optional cursor-unit icon
// selector. It indexes the runtime FDICON pointer bank as unit+2 * 12 plus
// raw animation state, except state 3 aliases state 1. The raw state is kept
// unnamed because this HUD-specific use does not establish a gameplay label.
func NativeMapHUDUnitFrameIndex(spriteGroup, rawState int) (int, error) {
	if spriteGroup < 0 || rawState < 0 || rawState > 3 {
		return 0, errors.New("fdicon: invalid native map HUD unit selector")
	}
	if rawState == 3 {
		rawState = 1
	}
	return spriteGroup*12 + rawState, nil
}

// NativeCellCoordinate is a world-cell coordinate in the native field map.
// It deliberately permits coordinates outside the visible/map range: 0x129ec
// delegates that check to 0x12ac6 rather than clipping its call schedule.
type NativeCellCoordinate struct {
	X int
	Y int
}

// NativeForegroundRedrawEligible reproduces 0x129ec's two roster gates. A
// slot must first be active (the 0x3453e unit+5 bit0 query), then pass the
// raw 0x1f183 predicate. That predicate suppresses the foreground pass for
// unit+7!=0x1c with class==0x13 or race in {4,5}. These are raw field values;
// this function deliberately assigns them no gameplay/visual label.
func NativeForegroundRedrawEligible(inactive bool, unit7, race, class byte) bool {
	if inactive || unit7 == 0x1c {
		return !inactive
	}
	return class != 0x13 && race != 4 && race != 5
}

// NativeForegroundRedrawCells reproduces the exact 0x129ec call order after
// one unit sprite has been drawn. It always redraws the unit cell then the
// cell above it. A nonzero unit+4 movement offset adds one pose-dependent
// neighbour: down, left, up (two cells), or right for every other pose value.
//
// The native callee owns map/camera bounds checks, so this pure schedule must
// not discard negative or off-screen coordinates. The returned count is two
// when stationary and three while moving.
func NativeForegroundRedrawCells(x, y int, pose byte, movementOffset int) ([3]NativeCellCoordinate, int) {
	cells := [3]NativeCellCoordinate{{X: x, Y: y}, {X: x, Y: y - 1}}
	if movementOffset == 0 {
		return cells, 2
	}
	switch pose {
	case 0:
		cells[2] = NativeCellCoordinate{X: x, Y: y + 1}
	case 1:
		cells[2] = NativeCellCoordinate{X: x - 1, Y: y}
	case 2:
		cells[2] = NativeCellCoordinate{X: x, Y: y - 2}
	default:
		cells[2] = NativeCellCoordinate{X: x + 1, Y: y}
	}
	return cells, 3
}

// NativeTerrainFrameIndex reproduces 0x11eee's FDSHAP descriptor selector.
// tile is the composition word's low 10 bits; flags is the selected terrain
// control byte. flip is native 0x53a40 (0 or 1), and cycle is 0x53c0b.
// Flag priority is 0x08, then 0x10, then 0x04, then the base tile.
func NativeTerrainFrameIndex(tile int, flags byte, flip, cycle int) (int, error) {
	if tile < 0 || tile > 0x3ff || (flip != 0 && flip != 1) {
		return 0, errors.New("fdicon: invalid native terrain frame selector")
	}
	if flags&0x08 != 0 {
		return tile + 2*flip, nil
	}
	if flags&0x10 != 0 {
		return tile + truncDiv2(cycle), nil
	}
	if flags&0x04 != 0 {
		return tile + flip, nil
	}
	return tile, nil
}

func truncDiv2(v int) int {
	if v < 0 {
		return -((-v) / 2)
	}
	return v / 2
}

// NativeForegroundFrameIndex reproduces 0x12ac6's foreground selector. Only
// terrain-control bit 0x80 draws a foreground cell. Bit 0x08 adds two times
// the native flip, and the FDSHAP offset lookup is one entry past that index
// (base+0x0a rather than the terrain pass's base+0x06).
func NativeForegroundFrameIndex(tile int, flags byte, flip int) (int, bool, error) {
	if tile < 0 || tile > 0x3ff || (flip != 0 && flip != 1) {
		return 0, false, errors.New("fdicon: invalid native foreground selector")
	}
	if flags&0x80 == 0 {
		return 0, false, nil
	}
	if flags&0x08 != 0 {
		tile += 2 * flip
	}
	return tile + 1, true, nil
}

// BlitNativeTerrainCell composes one already-selected FDFIELD terrain cell as
// 0x11eee does: select the FDSHAP frame, then use raw 0x4deda only for entry
// byte+3 == 0xff, otherwise use LUT-aware 0x4dcc6. Camera iteration, LUT
// phase selection and foreground redraw remain responsibilities of its caller.
func (b *Bank) BlitNativeTerrainCell(dst []byte, stride, x, y, tile int, flags, blitMode byte, flip, cycle int, lut []byte) error {
	index, err := NativeTerrainFrameIndex(tile, flags, flip, cycle)
	if err != nil {
		return err
	}
	sprite, err := b.SpriteFor(index/12, (index%12)/3, index%3)
	if err != nil {
		return err
	}
	if blitMode == 0xff {
		return sprite.BlitAt(dst, stride, x, y)
	}
	return sprite.BlitLUT(dst, stride, x, y, lut)
}

// BlitNativeForegroundCell reproduces 0x12ac6 after unit drawing. A missing
// foreground flag is a no-op; the alternate branch uses 0x4dd52 semantics,
// whose mode-3 spans preserve destination pixels.
func (b *Bank) BlitNativeForegroundCell(dst []byte, stride, x, y, tile int, flags, blitMode byte, flip int, lut []byte) error {
	index, present, err := NativeForegroundFrameIndex(tile, flags, flip)
	if err != nil || !present {
		return err
	}
	sprite, err := b.SpriteFor(index/12, (index%12)/3, index%3)
	if err != nil {
		return err
	}
	if blitMode == 0xff {
		return sprite.BlitAt(dst, stride, x, y)
	}
	return sprite.BlitLUTTransparent(dst, stride, x, y, lut)
}

// BlitNativeTerrainRegion reproduces 0x11eee's visible-cell loop. controls is
// the raw selected FDSHAP terrain-control table (four bytes per base tile);
// only byte 0 is consumed here, exactly as the native code does. The caller
// owns timing and chooses the explicit LUT; this routine does not advance it.
func (b *Bank) BlitNativeTerrainRegion(dst []byte, stride, dstX, dstY, mapWidth int, cells []NativeTerrainCell, controls []byte, mapX, mapY, width, height, flip, cycle int, lut []byte) error {
	if mapWidth <= 0 || width < 0 || height < 0 || mapX < 0 || mapY < 0 || len(cells)%mapWidth != 0 || mapX+width > mapWidth || mapY+height > len(cells)/mapWidth {
		return errors.New("fdicon: invalid native terrain region")
	}
	for row := 0; row < height; row++ {
		for col := 0; col < width; col++ {
			cell := cells[(mapY+row)*mapWidth+mapX+col]
			tile := int(cell.Tile & 0x3ff)
			if tile*4 >= len(controls) {
				return errors.New("fdicon: terrain control table is too short")
			}
			if err := b.BlitNativeTerrainCell(dst, stride, dstX+col*NativeSize, dstY+row*NativeSize, tile, controls[tile*4], cell.BlitMode, flip, cycle, lut); err != nil {
				return err
			}
		}
	}
	return nil
}
