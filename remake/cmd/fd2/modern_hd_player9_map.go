package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type modernHDPlayer9Frames struct {
	Palette, Effects, Results []*modernHDMapFrame
	Hold                      *modernHDMapFrame
	UsesHDEffects             bool
}

// Capture the input published with the actual steady snapshot. Legacy direct
// fixtures may supply a reconstructed input only if its complete VGA matches.
func (g *Game) freezeNativeCommand9MapInput() (indexedmap.NativeFrameInput, error) {
	var in indexedmap.NativeFrameInput
	if g.nativeMapFrameSource != nil {
		in = *g.nativeMapFrameSource
	} else {
		var err error
		in, err = g.nativeCh28PostFrameInput(g.st)
		if err != nil {
			return in, err
		}
	}
	var err error
	in, err = freezeCompoundMapInput(in)
	if err != nil {
		return in, err
	}
	work, vga := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	if g.nativeMapAssets != nil && g.nativeMapAssets.MapIndex == 23 {
		if g.nativeCh23State == nil {
			return in, errors.New("modern HD player9: same-compose stage absent")
		}
		if err := indexedmap.SeedNativeCh23Staging(work, g.nativeCh23State.staging); err != nil {
			return in, err
		}
	}
	if err := indexedmap.ComposeNativeFrame(work, vga, in); err != nil {
		return in, err
	}
	if !bytes.Equal(vga, g.nativeMapVGA) {
		return in, errors.New("modern HD player9: typed input differs from frozen VGA snapshot")
	}
	return in, nil
}

func modernPlayer9EventFrame(normal *modernHDMapFrame, pixels []byte, palette color.Palette) *modernHDMapFrame {
	frame := *normal
	frame.Background = image.NewRGBA(image.Rect(0, 0, 320, 200))
	frame.UI = image.NewRGBA(frame.Background.Bounds())
	if normal.Range != nil {
		frame.Range = image.NewRGBA(frame.Background.Bounds())
	}
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			if x < 312 && y < 192 {
				frame.Background.Set(x, y, normal.Background.At(x+4, y+4))
				frame.UI.Set(x, y, normal.UI.At(x+4, y+4))
				if frame.Range != nil {
					frame.Range.Set(x, y, normal.Range.At(x+4, y+4))
				}
			} else {
				frame.Background.Set(x, y, palette[pixels[y*320+x]])
			}
		}
	}
	shift := image.Pt(-24, -24)
	if normal.HUDPanel != nil {
		panel := *normal.HUDPanel
		panel.Rect = panel.Rect.Add(shift)
		frame.HUDPanel = &panel
	}
	if normal.HUDUnit != nil {
		patch := *normal.HUDUnit
		patch.Unit.Rect = patch.Unit.Rect.Add(shift)
		frame.HUDUnit = &patch
	}
	frame.Units = append([]modernHDMapUnitDraw(nil), normal.Units...)
	frame.Terrain = append([]modernHDTerrainDraw(nil), normal.Terrain...)
	frame.Foreground = append([]modernHDMapForegroundDraw(nil), normal.Foreground...)
	for i := range frame.Units {
		frame.Units[i].Rect = frame.Units[i].Rect.Add(shift)
	}
	for i := range frame.Terrain {
		frame.Terrain[i].Rect = frame.Terrain[i].Rect.Add(shift)
	}
	for i := range frame.Foreground {
		frame.Foreground[i].Rect = frame.Foreground[i].Rect.Add(shift)
	}
	frame.Viewport = image.Rect(0, 0, 312*6, 192*6)
	frame.OverlaysAboveUI = true
	return &frame
}

func (g *Game) prepareModernHDPlayer9Frames(job *nativeCommand9PlayerJob, set *modernStoryPortraitSet) (*modernHDPlayer9Frames, error) {
	if job == nil || set == nil {
		return nil, errors.New("modern HD player9: owner/theme absent")
	}
	if len(set.mapSpritesHD) != 125 {
		if set.mapEffects != nil || (set.mapOverlays != nil && set.mapOverlays.frames[94] != nil) {
			return nil, errors.New("modern HD player9: registered effects require the complete HD world")
		}
		return nil, nil
	} // Legacy portrait-only themes.
	if job.hdInputErr != nil {
		return nil, job.hdInputErr
	}
	if len(job.palettes) != 8 || len(job.effectFrames) != 27 || len(job.resultFrames) != 22 || len(job.effectSources) != 27 || len(job.resultSources) != 22 {
		return nil, errors.New("modern HD player9: incomplete native producer")
	}
	scene := job.hdScene
	if err := scene.validate(g); err != nil {
		return nil, err
	}
	// Check the owners original producer as well as the independent snapshot.
	probe := *scene
	probe.input, probe.work, probe.vga = job.hdInput, job.baselineWork, job.baselineVGA
	digest, err := probe.digest()
	if err != nil || digest != scene.signature {
		return nil, errors.New("modern HD player9: owner producer changed")
	}
	palette := scene.palette
	hasEffects := set.mapEffects != nil
	hasResults := set.mapOverlays != nil && set.mapOverlays.frames[94] != nil
	if hasEffects != hasResults {
		return nil, errors.New("modern HD player9: partial registered effect/result group")
	}
	if hasEffects {
		for _, resource := range []int{5, 6} {
			bank, entries, count := set.mapOverlays, g.nativeMapAssets.CommandHealDigits, 13
			if resource == 6 {
				bank, entries, count = set.mapEffects, g.nativeMapAssets.FDOTHER6, 27
			}
			for _, index := range modernMapOverlayHDIndices(resource, count) {
				if index >= len(entries) || bank.native[index] != modernLMIHDDescription(index, entries[index]) {
					return nil, errors.New("modern HD player9: incomplete native bank")
				}
				picture := bank.frames[index]
				if picture == nil || picture.Bounds() != image.Rect(0, 0, entries[index].Width*6, entries[index].Height*6) {
					return nil, errors.New("modern HD player9: incomplete registered bank")
				}
			}
		}
	}
	view := *g
	view.modernStoryPortraits = set
	prepare := func(p color.Palette) (*modernHDMapFrame, error) {
		frame, e := view.prepareModernHDMapFrameStageLayout(scene.input, scene.work, p, true, -1, scene.stage)
		if e != nil {
			return nil, e
		}
		for y := 0; y < 200; y++ {
			for x := 0; x < 320; x++ {
				if x < 4 || x >= 316 || y < 4 || y >= 196 {
					frame.Background.Set(x, y, p[job.baselineVGA[y*320+x]])
				}
			}
		}
		frame.OverlaysAboveUI = true
		return frame, nil
	}
	frames := &modernHDPlayer9Frames{UsesHDEffects: hasEffects}
	for _, p := range job.palettes {
		frame, e := prepare(p)
		if e != nil {
			return nil, e
		}
		frames.Palette = append(frames.Palette, frame)
	}
	frames.Hold, err = prepare(palette)
	if err != nil {
		return nil, err
	}
	base := modernPlayer9EventFrame(frames.Hold, job.baselineVGA, palette)
	nativeFallback := map[[2]int]*image.RGBA{}
	layers := func(sources []indexedmap.NativeCommandTailLayer) (*modernHDMapFrame, error) {
		frame := *base
		for _, source := range sources {
			var bank *modernMapOverlayHDResource
			var entries []fdother.LMI1Entry
			switch source.Resource {
			case 6:
				bank, entries = set.mapEffects, g.nativeMapAssets.FDOTHER6
			case 5:
				bank, entries = set.mapOverlays, g.nativeMapAssets.CommandHealDigits
			default:
				return nil, errors.New("modern HD player9: unknown LMI source")
			}
			if source.Entry < 0 || source.Entry >= len(entries) {
				return nil, errors.New("modern HD player9: source entry absent")
			}
			entry := entries[source.Entry]
			var picture *image.RGBA
			if hasEffects {
				if bank == nil || bank.native[source.Entry] != modernLMIHDDescription(source.Entry, entry) {
					return nil, fmt.Errorf("modern HD player9: native source %d/%d changed", source.Resource, source.Entry)
				}
				picture = bank.frames[source.Entry]
				if picture == nil || picture.Bounds() != image.Rect(0, 0, entry.Width*6, entry.Height*6) {
					return nil, errors.New("modern HD player9: registered source absent")
				}
			} else {
				key := [2]int{source.Resource, source.Entry}
				picture = nativeFallback[key]
				if picture == nil {
					picture = image.NewRGBA(image.Rect(0, 0, entry.Width*6, entry.Height*6))
					for y := 0; y < entry.Height; y++ {
						for x := 0; x < entry.Width; x++ {
							value := entry.Pixels[y*entry.Width+x]
							if value == 0 {
								continue
							}
							for dy := 0; dy < 6; dy++ {
								for dx := 0; dx < 6; dx++ {
									picture.Set(x*6+dx, y*6+dy, palette[value])
								}
							}
						}
					}
					nativeFallback[key] = picture
				}
			}
			frame.Overlays = append(frame.Overlays, modernHDMapOverlayDraw{Entry: source.Entry, Image: picture, Rect: picture.Bounds().Add(image.Pt(source.X*6, source.Y*6))})
		}
		return &frame, nil
	}
	for _, sources := range job.effectSources {
		frame, e := layers(sources)
		if e != nil {
			return nil, e
		}
		frames.Effects = append(frames.Effects, frame)
	}
	for _, sources := range job.resultSources {
		frame, e := layers(sources)
		if e != nil {
			return nil, e
		}
		frames.Results = append(frames.Results, frame)
	}
	return frames, nil
}
