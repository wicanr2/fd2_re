package indexedmap

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
)

const nativeCommandSamplingUnit = 0xc00

func floorDivMod(value, divisor int) (int, int) {
	q, r := value/divisor, value%divisor
	if r < 0 {
		r += divisor
		q--
	}
	return q, r
}

// ComposeNativeCommand10To12Surface reproduces one 0x1F558 sampling pass,
// the following 0x127E0 object redraw, and the final 312x192 viewport copy.
// xOrigin/yOrigin/increment are the untouched signed table values from
// 0x52096/0x520A2/0x520AE. Input rejection leaves work and VGA unchanged.
func ComposeNativeCommand10To12Surface(work, vga []byte, in NativeTransitionFrameInput, xOrigin, yOrigin, increment int) error {
	if len(work) != NativeUnitPresentWorkSize || len(vga) != NativeMapVGASize || in.TerrainBank == nil ||
		in.MapWidth <= 0 || len(in.Cells)%in.MapWidth != 0 || len(in.Controls)%4 != 0 || increment <= 0 {
		return errors.New("indexedmap: incomplete native command10-12 sampling input")
	}
	mapHeight := len(in.Cells) / in.MapWidth
	terrain := make([]byte, NativeMapVGASize)
	startX := in.CameraX*nativeCommandSamplingUnit + 13*(nativeCommandSamplingUnit/2) + xOrigin - 156*increment
	startY := in.CameraY*nativeCommandSamplingUnit + 8*(nativeCommandSamplingUnit/2) + yOrigin - 96*increment
	for y := 0; y < viewHeight; y++ {
		tileY, remY := floorDivMod(startY+y*increment, nativeCommandSamplingUnit)
		pixelY := remY / 128
		for x := 0; x < steadyViewportWidth; x++ {
			tileX, remX := floorDivMod(startX+x*increment, nativeCommandSamplingUnit)
			pixelX := remX / 128
			if tileX < 0 || tileX >= in.MapWidth || tileY < 0 || tileY >= mapHeight {
				continue
			}
			cell := in.Cells[tileY*in.MapWidth+tileX]
			tile := int(cell.Tile & 0x3ff)
			if tile*4 >= len(in.Controls) {
				return fmt.Errorf("indexedmap: native command10-12 terrain control %d unavailable", tile)
			}
			frame, err := fdicon.NativeTerrainFrameIndex(tile, in.Controls[tile*4], in.Flip, in.TerrainCycle)
			if err != nil {
				return err
			}
			sprite, err := in.TerrainBank.SpriteFor(frame/12, (frame%12)/3, frame%3)
			if err != nil || len(sprite.Pixels) != fdicon.NativeSize*fdicon.NativeSize {
				return fmt.Errorf("indexedmap: native command10-12 terrain frame %d unavailable", frame)
			}
			terrain[steadyViewportOffset+y*viewWidth+x] = sprite.Pixels[pixelY*fdicon.NativeSize+pixelX]
		}
	}
	nextWork, nextVGA := make([]byte, len(work)), append([]byte(nil), vga...)
	for row := 0; row < viewHeight; row++ {
		copy(nextWork[workBase+row*workStride:workBase+row*workStride+steadyViewportWidth],
			terrain[steadyViewportOffset+row*viewWidth:steadyViewportOffset+row*viewWidth+steadyViewportWidth])
	}
	if err := RedrawNativeUnitPresentObjects(nextWork, in); err != nil {
		return err
	}
	if err := CopyNativeUnitPresentViewport(nextVGA, nextWork); err != nil {
		return err
	}
	copy(work, nextWork)
	copy(vga, nextVGA)
	return nil
}
