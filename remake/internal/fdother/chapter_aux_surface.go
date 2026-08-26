package fdother

import (
	"encoding/binary"
	"errors"
)

const NativeChapterAuxSurfaceResource = 55

var nativeChapterAuxRowOffsets = [16]byte{2, 3, 3, 4, 4, 4, 3, 3, 2, 1, 1, 0, 0, 0, 1, 1}

// NativeChapterAuxSurface is FDOTHER #55's raw 320x200 indexed payload.
// The name is intentionally structural: the executable does not name the
// player-visible artwork or assign it a terrain meaning.
type NativeChapterAuxSurface struct {
	Pixels []byte
}

// DecodeNativeChapterAuxSurface accepts only sub_10652's exact raw resource.
func DecodeNativeChapterAuxSurface(datPath string) (*NativeChapterAuxSurface, error) {
	raw, err := ReadResource(datPath, NativeChapterAuxSurfaceResource)
	if err != nil {
		return nil, err
	}
	if len(raw) != 4+320*200 || binary.LittleEndian.Uint16(raw) != 320 || binary.LittleEndian.Uint16(raw[2:]) != 200 {
		return nil, errors.New("fdother: native chapter auxiliary surface is not raw 320x200")
	}
	return &NativeChapterAuxSurface{Pixels: append([]byte(nil), raw[4:]...)}, nil
}

// BlitNativeChapterAuxViewport reproduces sub_4EB90's 192 row copies into a
// 320-stride staging surface. Each row advances the 16-entry raw phase table.
func BlitNativeChapterAuxViewport(dst []byte, stride int, surface *NativeChapterAuxSurface, phase int) error {
	if surface == nil || len(surface.Pixels) != 320*200 || phase < 0 || phase >= len(nativeChapterAuxRowOffsets) ||
		stride < 312 || len(dst) < stride*192 {
		return errors.New("fdother: incomplete native chapter auxiliary viewport")
	}
	for row := 0; row < 192; row++ {
		x := int(nativeChapterAuxRowOffsets[(phase+row)%len(nativeChapterAuxRowOffsets)])
		src := row*320 + x
		copy(dst[row*stride:row*stride+312], surface.Pixels[src:src+312])
	}
	return nil
}
