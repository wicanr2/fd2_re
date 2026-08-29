package main

import (
	"errors"
	"image/color"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// nativeMapAssets is an all-or-nothing original-resource bundle. It remains
// separate from the PNG renderer until indexed presentation is ready.
type nativeMapAssets struct {
	MapIndex int
	Frames   indexedmap.NativeMapHUDFrames
	Terrain  *fdicon.Bank
	Range    *fdicon.Bank
	Units    *fdicon.Bank
	Controls []byte
	// LUTs is FDOTHER#3's raw 256-byte remap bank. Entries 1..9 are the
	// verified 0x24618 transition selectors; loading them here does not by
	// itself authorize scene presentation.
	LUTs    [][]byte
	Palette color.Palette
	// PaletteDAC retains FDOTHER#0's exact 768-byte, six-bit DAC baseline.
	// Native 0x11df2 effects require mutable raw components; color.Palette is
	// presentation-only and must not be used as the effect state.
	PaletteDAC []byte
	// SpawnIntro is FDOTHER #9's exact twelve-entry LMI1 bank consumed by
	// sub_32999. Map initialization preflights it with the shared battle bundle
	// so an otherwise valid map cannot fail halfway through a roster event.
	SpawnIntro []fdother.LMI1Entry
	// CommandHealDigits is the complete FDOTHER #5 LMI1 bank. The historical
	// field name is retained for existing command-tail callers; ch28 post also
	// consumes exact entries 0x44..0x4f without assigning them semantic names.
	// FDOTHER6 is the complete shared #6 bank consumed by command 13..16 and
	// 0x22253; neither path may rename the archive itself after one effect.
	CommandHealDigits []fdother.LMI1Entry
	FDOTHER6          []fdother.LMI1Entry
	ChapterAux        *fdother.NativeChapterAuxSurface
}

func loadNativeMapAssets(mapDir string) (*nativeMapAssets, error) {
	mapIndex, err := fdother.MapIndexFromAssetPath(mapDir)
	if err != nil {
		return nil, err
	}
	frames, err := indexedmap.LoadSeparatedNativeMapHUDFrames(separatedAssetPath("ui"))
	if err != nil {
		return nil, err
	}
	terrain, controls, err := fdicon.LoadSeparatedFDSHAPBank(separatedAssetPath("tilesets/fdshap"), mapIndex)
	if err != nil {
		return nil, err
	}
	units, err := fdicon.LoadSeparatedBank(separatedAssetPath("sprites/fdicon"))
	if err != nil {
		return nil, err
	}
	rangeBank, err := fdother.LoadSeparatedRangeOverlayBank(separatedAssetPath("sprites/fdother_001_range_overlay"))
	if err != nil {
		return nil, err
	}
	luts, err := fdother.LoadSeparatedLUTBank(separatedAssetPath("palette"))
	if err != nil || len(luts) <= 9 {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("native map assets: FDOTHER#3 LUT bank lacks transition entries 1..9")
	}
	paletteRaw, palette, err := loadNativeBattlePalette()
	if err != nil {
		return nil, err
	}
	spawnIntro, err := fdother.LoadSeparatedLMI1Bank(
		separatedAssetPath("animations/fdother_009_spawn_intro"),
		fdother.NativeSpawnIntroFrameResource,
	)
	if err != nil {
		return nil, err
	}
	commandHealDigits, err := fdother.LoadSeparatedLMI1Bank(
		separatedAssetPath("ui/fdother_005_lmi1_opaque"),
		fdother.NativeCommandHealTailDigitResource,
	)
	if err != nil {
		return nil, err
	}
	fdother6, err := fdother.LoadSeparatedLMI1Bank(
		separatedAssetPath("effects/fdother_006_lmi1_opaque"),
		fdother.NativeCommandHealTailEffectResource,
	)
	if err != nil {
		return nil, err
	}
	var chapterAux *fdother.NativeChapterAuxSurface
	if mapIndex == 28 || mapIndex == 29 {
		chapterAux, err = fdother.LoadSeparatedChapterAuxSurface(separatedAssetPath("surfaces"))
		if err != nil {
			return nil, err
		}
	}
	return &nativeMapAssets{
		MapIndex: mapIndex, Frames: frames,
		Terrain: terrain, Range: rangeBank, Units: units,
		Controls: controls, LUTs: luts, Palette: palette,
		PaletteDAC:        append([]byte(nil), paletteRaw...),
		SpawnIntro:        spawnIntro,
		CommandHealDigits: commandHealDigits,
		FDOTHER6:          fdother6,
		ChapterAux:        chapterAux,
	}, nil
}

func nativeMapAssetsAvailable(a *nativeMapAssets) bool {
	if a == nil || a.Terrain == nil || a.Range == nil || a.Units == nil ||
		len(a.Controls) == 0 || len(a.LUTs) <= 9 || len(a.Palette) != 256 ||
		len(a.PaletteDAC) != 256*3 {
		return false
	}
	for i := 1; i <= 9; i++ {
		if len(a.LUTs[i]) != 256 {
			return false
		}
	}
	if (a.MapIndex == 28 || a.MapIndex == 29) && (a.ChapterAux == nil || len(a.ChapterAux.Pixels) != 320*200) {
		return false
	}
	return true
}

func nativeSpawnIntroAssetsAvailable(a *nativeMapAssets) bool {
	if !nativeMapAssetsAvailable(a) || len(a.SpawnIntro) != fdother.NativeSpawnIntroPassCount {
		return false
	}
	for _, entry := range a.SpawnIntro {
		if entry.Width <= 0 || entry.Height <= 0 || len(entry.Pixels) != entry.Width*entry.Height {
			return false
		}
	}
	return true
}
