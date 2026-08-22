package campaign

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

func nativeSystemInfoFixture(t *testing.T) NativeSystemInfoAssets {
	t.Helper()
	assets := NativeSystemInfoAssets{
		Numbers: battle.NativeItemPanelDataAssets{Frames: map[int]fdother.Frame{}},
	}
	geometry := [4][2]int{{102, 17}, {170, 117}, {170, 16}, {63, 15}}
	for index, size := range geometry {
		assets.Panels[index] = fdother.LMI1Entry{
			Width: size[0], Height: size[1], Pixels: make([]byte, size[0]*size[1]),
		}
		for pixel := range assets.Panels[index].Pixels {
			assets.Panels[index].Pixels[pixel] = byte(0x80 + index)
		}
	}
	for index := 31; index <= 51; index++ {
		assets.Numbers.Frames[index] = fdother.Frame{
			Width: 1, Height: 1, Pixels: []byte{1, 0, 1, 0, 0, byte(index)},
		}
	}
	fontRaw := make([]byte, fdtxt.GlyphBytes)
	fontRaw[0] = 0x80
	font, err := fdtxt.ParseFont(fontRaw)
	if err != nil {
		t.Fatal(err)
	}
	assets.Font = font
	return assets
}

func TestComposeNativeSystemInfoSurfaceUses1B41DPanelAndNumberAnchors(t *testing.T) {
	frame, err := ComposeNativeSystemInfoSurface(nativeSystemInfoFixture(t), NativeSystemInfoInput{
		RawChapter: 1, RawRound: 23, Currency: 456,
		RawCampCounts: [3]int{7, 8, 9}, Lines: [2][]uint16{{0}, {0}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, point := range []struct {
		x, y int
		want byte
	}{
		{109, 19, 0x80}, {75, 37, 0x81}, {75, 155, 0x82}, {129, 172, 0x83},
		{143, 24, 42}, {188, 24, 42}, {140, 176, 31},
		{120, 159, 42}, {182, 159, 42}, {228, 159, 42},
		{80, 61, 0xcd}, {80, 116, 0xcd},
	} {
		if got := frame[point.y*320+point.x]; got != point.want {
			t.Fatalf("pixel (%d,%d)=%#x, want %#x", point.x, point.y, got, point.want)
		}
	}
}

func TestNativeSystemInfoCampCountsUsesRaw1B41DPredicate(t *testing.T) {
	unit := func(camp, byte5, fig, race byte) *battle.Unit {
		return &battle.Unit{
			NativeRecordByte5: byte5, HasNativeRecordByte5: true,
			NativeRecordByte6: camp, HasNativeRecordByte6: true,
			BattleFig: int(fig), HasBattleFig: true,
			NativeRecordRace: race, HasNativeRecordRace: true,
		}
	}
	counts, err := NativeSystemInfoCampCounts([]*battle.Unit{
		unit(0, 0, 1, 1), unit(1, 0, 2, 2), unit(2, 0, 3, 3),
		unit(0, 1, 4, 4), unit(1, 0, 0x79, 5), unit(2, 0, 6, 10),
		unit(3, 0, 7, 7),
	})
	if err != nil || counts != [3]int{1, 1, 1} {
		t.Fatalf("counts=%v err=%v", counts, err)
	}
	missing := unit(0, 0, 1, 1)
	missing.HasNativeRecordRace = false
	if got, err := NativeSystemInfoCampCounts([]*battle.Unit{missing}); err == nil || got != [3]int{} {
		t.Fatalf("missing provenance counts=%v err=%v", got, err)
	}
}

func TestComposeNativeSystemInfoSurfaceRejectsInvalidPanelBeforePublication(t *testing.T) {
	assets := nativeSystemInfoFixture(t)
	assets.Panels[2].Width--
	frame, err := ComposeNativeSystemInfoSurface(assets, NativeSystemInfoInput{})
	if err == nil || frame != nil {
		t.Fatalf("invalid panel published frame=%v err=%v", frame != nil, err)
	}
}
