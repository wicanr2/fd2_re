package main

import (
	"bytes"
	"fmt"
	"image/color"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

type modernOrdinarySceneryMatcher struct {
	source modernHDBattleSource
	basis  *modernBattleSurfaceLUTBasis
	luts   map[int][]byte
}

func (m *modernOrdinarySceneryMatcher) match(base []byte) (modernHDBattleSource, error) {
	if len(base) != 64000 {
		return modernHDBattleSource{}, fmt.Errorf("modern HD ordinary scenery: incomplete base")
	}
	f := m.source.Frame
	for index := 10; index <= 15; index++ {
		lookup := m.luts[index]
		if index != 10 && len(lookup) != 256 {
			continue
		}
		matches := true
		for i, mask := range f.Mask {
			if mask == 0 {
				continue
			}
			value := f.Pixels[i]
			if lookup != nil {
				value = lookup[value]
			}
			if base[(f.Y+i/f.Width)*320+f.X+i%f.Width] != value {
				matches = false
				break
			}
		}
		if matches {
			source := m.source
			if lookup != nil {
				var err error
				source.SurfaceLUT, err = m.basis.profile(index, lookup)
				if err != nil {
					return source, err
				}
			}
			return source, nil
		}
	}
	return modernHDBattleSource{}, fmt.Errorf("modern HD ordinary scenery: %s:%d does not match a native LUT variant", m.source.Archive, m.source.Resource)
}

// Match only exact, proven native variants. No phase names or approximate
// colours are used to infer which background was present in a frame.
func bindModernHDOrdinaryScenery(bank modernHDCommandBankSource, background fdother.Frame, bgIndex int,
	platform fdother.Frame, taiIndex int, palette color.Palette, luts [][]byte) error {
	if !modernBattleBackdropGeometry(background) {
		return nil
	}
	bg, err := modernHDBattleSurfaceFrame(background)
	if err != nil {
		return err
	}
	bg.X, bg.Y = 0, 50
	tai, err := modernBattlePlatformSource(platform, taiIndex)
	if err != nil {
		return err
	}
	originals := []modernHDBattleSource{{Archive: "BG.DAT", Resource: bgIndex, Frame: bg, Fill: -1}, tai}
	matchers := make([]modernOrdinarySceneryMatcher, 2)
	for i, source := range originals {
		basis, err := newModernBattleSurfaceLUTBasis(source.Frame, palette)
		if err != nil {
			return err
		}
		matchers[i] = modernOrdinarySceneryMatcher{source: source, basis: basis, luts: make(map[int][]byte)}
		for index := 11; index <= 15 && index < len(luts); index++ {
			if len(luts[index]) == 256 {
				matchers[i].luts[index] = append([]byte(nil), luts[index]...)
			}
		}
	}
	frames := bank.sourceFrames()
	staged := make([]*modernHDBattleSources, len(frames))
	for index, frame := range frames {
		if frame.hdSources == nil {
			continue
		}
		sources := *frame.hdSources
		sources.Layers = append([]modernHDBattleSource(nil), sources.Layers...)
		moving := false
		for i, layer := range sources.Layers {
			if layer.Resource >= 0 || layer.PanelText != nil {
				continue
			}
			f, expected := layer.Frame, tai.Frame
			if frame.hdPhase != "prelude" || moving || layer.Archive != "" || layer.Fill != -1 ||
				f.X != expected.X || f.Y != expected.Y || f.Width != expected.Width || f.Height != expected.Height ||
				!bytes.Equal(f.Pixels, expected.Pixels) || !bytes.Equal(f.Mask, expected.Mask) {
				return fmt.Errorf("modern HD ordinary scenery: moving platform differs")
			}
			layer.Archive, layer.Resource = "TAI.DAT", taiIndex
			sources.Layers[i], moving = layer, true
		}
		selected, err := matchers[0].match(sources.Base)
		if err != nil {
			return fmt.Errorf("%s: %w", frame.hdPhase, err)
		}
		scenery := []modernHDBattleSource{selected}
		if !moving {
			selected, err = matchers[1].match(sources.Base)
			if err != nil {
				return fmt.Errorf("%s: %w", frame.hdPhase, err)
			}
			scenery = append(scenery, selected)
		}
		if err := bindModernPhysicalBaseScenery(&sources, scenery); err != nil {
			return err
		}
		rebuilt := append([]byte(nil), sources.Base...)
		for _, source := range sources.BaseScenery {
			if source.PanelText != nil {
				continue
			}
			f := source.Frame
			if source.SurfaceLUT != nil {
				f, err = source.SurfaceLUT.nativeFrame(f)
				if err != nil {
					return err
				}
			}
			if err := f.BlitAt(rebuilt, 320); err != nil {
				return err
			}
		}
		if !bytes.Equal(rebuilt, frame.hdSources.Base) {
			return fmt.Errorf("modern HD ordinary scenery: complete native reconstruction differs")
		}
		staged[index] = &sources
	}
	// sourceFrames returns value copies carrying pointers to the actual banks.
	// Publish through those pointers only after every candidate was validated.
	for index, source := range staged {
		if source != nil {
			*frames[index].hdSources = *source
		}
	}
	return nil
}

func modernBattleSurfaceNativeFrame(source modernHDBattleSource) (figani.Frame, error) {
	if source.SurfaceLUT == nil {
		return source.Frame, nil
	}
	return source.SurfaceLUT.nativeFrame(source.Frame)
}
