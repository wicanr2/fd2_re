package main

import (
	"fmt"
	"image/color"

	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// Selection closures are frozen by the same checked writer used by player9.
// A failed capture remains attached to the candidate until HD admission.
func (g *Game) freezeModernHDModifierScene(palette color.Palette) *modernHDCompoundMapScene {
	in, err := g.freezeNativeCommand9MapInput()
	return &modernHDCompoundMapScene{input: in, err: err,
		work: append([]byte(nil), g.nativeMapWork...), vga: append([]byte(nil), g.nativeMapVGA...),
		palette: append(color.Palette(nil), palette...)}
}

func bindModernHDModifierPalette(frames []nativeCompoundPresentedFrame, scene *modernHDCompoundMapScene, palettes []color.Palette) error {
	if scene == nil || len(scene.palette) != 256 || len(frames) != 8 || len(palettes) != 8 {
		return fmt.Errorf("modern HD modifier: palette source count differs")
	}
	scenes := make(map[[4]uint32]*modernHDCompoundMapScene)
	for i, palette := range palettes {
		if len(palette) != 256 || palette[0] == nil {
			return fmt.Errorf("modern HD modifier: palette source absent")
		}
		for index := 1; index < 256; index++ {
			if palette[index] == nil || scene.palette[index] == nil {
				return fmt.Errorf("modern HD modifier: incomplete palette")
			}
			r, g, b, a := palette[index].RGBA()
			r0, g0, b0, a0 := scene.palette[index].RGBA()
			if r != r0 || g != g0 || b != b0 || a != a0 {
				return fmt.Errorf("modern HD modifier: palette changed outside entry zero")
			}
		}
		r, g, b, a := palette[0].RGBA()
		key := [4]uint32{r, g, b, a}
		coloured := scenes[key]
		if coloured == nil {
			copy := *scene
			copy.palette = append(color.Palette(nil), palette...)
			coloured = &copy
			scenes[key] = coloured
		}
		if err := bindModernHDCompoundMap(frames[i:i+1], coloured, [][]byte{scene.vga}, nil, nil, 0, false, "map-palette"); err != nil {
			return err
		}
	}
	return nil
}

type modernHDModifierSources struct {
	initial, post               *modernHDCompoundMapScene
	palettes                    []color.Palette
	effects, digits             [][]byte
	effectSources, digitSources [][]indexedmap.NativeCommandTailLayer
	targets                     []indexedmap.NativeCommandHealTailTarget
	mask                        []byte
	maskIndex                   byte
}

// Only provenance is attached here. The native producer owns all image bytes,
// stage counts, samples and the separate transaction publication boundary.
func bindModernHDModifierFrames(frames []nativeCompoundPresentedFrame, source modernHDModifierSources) error {
	prefix, effects, digits := len(source.palettes), len(source.effects), len(source.digits)
	if source.initial == nil || source.post == nil || (prefix != 0 && prefix != 8) ||
		effects == 0 || len(source.effectSources) != effects || len(source.digitSources) != digits ||
		len(frames) != prefix+effects+11+digits {
		return fmt.Errorf("modern HD modifier: source schedule differs")
	}
	if prefix != 0 {
		if err := bindModernHDModifierPalette(frames[:prefix], source.initial, source.palettes); err != nil {
			return err
		}
	}
	if err := bindModernHDCompoundMap(frames[prefix:prefix+effects], source.initial, source.effects, source.effectSources, nil, 0, true, "map-effect"); err != nil {
		return err
	}
	if err := bindModernHDCompoundMapMask(frames[prefix+effects:prefix+effects+11], source.initial,
		source.initial.vga, source.mask, source.targets, source.maskIndex, "map-mask"); err != nil {
		return err
	}
	if digits != 0 {
		return bindModernHDCompoundMap(frames[prefix+effects+11:], source.post, source.digits, source.digitSources, nil, 0, true, "map-digits")
	}
	return nil
}
