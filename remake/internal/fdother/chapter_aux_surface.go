package fdother

import (
	"encoding/binary"
	"errors"
)

const NativeChapterAuxSurfaceResource = 55

// NativeChapterAuxSurfaceContract 是 sub_10652 對單一 raw chapter 載入的 FDOTHER
// 輔助底面與其固定原始雜湊。
type NativeChapterAuxSurfaceContract struct {
	Resource  int
	RawMD5    string
	RawSHA256 string
}

var nativeChapterAuxSurfaceContracts = [...]NativeChapterAuxSurfaceContract{
	{Resource: 15, RawMD5: "977a1ae51b3f529043e30bd16f03728e", RawSHA256: "49facaa1526718153ebf3163d8f3215e9c9a7f010f1e36a536769a3a7ce3e3ee"},
	{Resource: 55, RawMD5: "710ce98d109298ff0110b1a4fb8fec53", RawSHA256: "a1999b7547bc4eabfb79049ae7cd7d08b12fd4402132e9e6e67b1fb56c981e65"},
}

// NativeChapterAuxSurfaceContracts 回傳所有已證實的輔助底面資源（匯出工具用）。
func NativeChapterAuxSurfaceContracts() []NativeChapterAuxSurfaceContract {
	return append([]NativeChapterAuxSurfaceContract(nil), nativeChapterAuxSurfaceContracts[:]...)
}

// NativeChapterAuxSurfaceFor 是 sub_10652 的章節分支：raw chapter 9／24／25 載入
// FDOTHER #15，28／29 載入 #55；sub_11EEE 對同一組章節在地形圖塊前鋪這張底面
// （docs/data/fd2_chapter_aux_graphics_10652_ida.txt、
// docs/data/ida/fd2_ch29_aux_terrain_surface_ida.txt）。其他章節沒有底面。
func NativeChapterAuxSurfaceFor(chapter int) (NativeChapterAuxSurfaceContract, bool) {
	switch chapter {
	case 9, 24, 25:
		return nativeChapterAuxSurfaceContracts[0], true
	case 28, 29:
		return nativeChapterAuxSurfaceContracts[1], true
	}
	return NativeChapterAuxSurfaceContract{}, false
}

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
