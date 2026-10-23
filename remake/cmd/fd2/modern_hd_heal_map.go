package main

import (
	"errors"
	"image"
	"image/color"
	"math"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type modernHDHealFrames struct {
	Front, Effects, Digits []*modernHDMapFrame
	Steady, Event, Mask    *modernHDMapFrame
}

func healHDCounts(pass fdother.IndexedTransitionPass) ([]byte, []byte, error) {
	terrain, objects := make([]byte, indexedmap.NativeUnitPresentWorkSize), make([]byte, indexedmap.NativeUnitPresentWorkSize)
	add := make([]byte, 256)
	for i := range add {
		add[i] = byte(i + 1)
	}
	if err := fdother.ApplyRadialLUTRemap(terrain, 456, add, pass.FirstRadial); err != nil {
		return nil, nil, err
	}
	for _, dst := range [][]byte{terrain, objects} {
		if err := fdother.ApplyRadialLUTRemap(dst, 456, add, pass.SecondRadial); err != nil {
			return nil, nil, err
		}
		if err := fdother.ApplyCenteredRectLUTRemap(dst, 456, add, pass.FinalRect); err != nil {
			return nil, nil, err
		}
	}
	return terrain, objects, nil
}

func healHDCountAt(counts []byte, x, y int) byte {
	if x < 0 || x >= 312 || y < 0 || y >= 192 {
		return 0
	}
	return counts[0x8088+y*456+x]
}

func hasHealHDBank(bank *modernMapOverlayHDResource, first, last int) bool {
	if bank == nil {
		return false
	}
	for i := first; i <= last; i++ {
		if bank.frames[i] != nil {
			return true
		}
		if _, ok := bank.native[i]; ok {
			return true
		}
	}
	return false
}

func validateHealHDBanks(g *Game, set *modernStoryPortraitSet) (bool, error) {
	effects, digits := hasHealHDBank(set.mapEffects, 57, 63), hasHealHDBank(set.mapOverlays, 105, 114)
	if effects != digits {
		return false, errors.New("modern HD heal: partial declared group")
	}
	if !effects {
		return false, nil
	}
	for _, spec := range [][3]int{{6, 57, 63}, {5, 105, 114}} {
		bank, entries := set.mapEffects, g.nativeMapAssets.FDOTHER6
		if spec[0] == 5 {
			bank, entries = set.mapOverlays, g.nativeMapAssets.CommandHealDigits
		}
		for index := spec[1]; index <= spec[2]; index++ {
			if index >= len(entries) || bank.native[index] != modernLMIHDDescription(index, entries[index]) || bank.frames[index] == nil || bank.frames[index].Bounds() != image.Rect(0, 0, entries[index].Width*6, entries[index].Height*6) {
				return false, errors.New("modern HD heal: declared LMI source unavailable or changed")
			}
		}
	}
	return true, nil
}

// Native masks operate in the actual work-buffer address space. Counts are
// sampled only after the same 0x8088 viewport offset as the native producer.
func (g *Game) healHDFront(j *nativeCommandHealPresentationJob, set *modernStoryPortraitSet, normal *modernHDMapFrame, index int) (*modernHDMapFrame, error) {
	pass, lut := j.hdPasses[index], j.hdLUTs[index]
	tc, oc, err := healHDCounts(pass)
	if err != nil {
		return nil, err
	}
	in := nativeTransitionInput(j.hdInput)
	in.Units, in.ForegroundUnits = nil, nil
	work, vga := make([]byte, indexedmap.NativeUnitPresentWorkSize), make([]byte, 64000)
	if err := indexedmap.ComposeNativeTransitionFrame(work, vga, in, pass, lut); err != nil {
		return nil, err
	}
	f := commandHDObjectFrame(normal, vga, j.palette)
	var luts [4][]byte
	for count := 0; count < 4; count++ {
		luts[count] = make([]byte, 256)
		for n := 0; n < 256; n++ {
			v := byte(n)
			for c := 0; c < count; c++ {
				v = lut[v]
			}
			luts[count][n] = v
		}
	}
	assets := set.terrainHD[g.nativeMapAssets.MapIndex]
	for _, tile := range normal.Terrain {
		rect := tile.Rect.Add(image.Pt(-24, -24))
		col, row := tile.Rect.Min.X/144, tile.Rect.Min.Y/144
		// Normal rectangles include the native four-pixel inset.
		col, row = (tile.Rect.Min.X-24)/144, (tile.Rect.Min.Y-24)/144
		at := (j.hdInput.Frame.CameraY+row)*j.hdInput.Frame.MapWidth + j.hdInput.Frame.CameraX + col
		if at < 0 || at >= len(j.hdInput.Frame.Cells) {
			return nil, errors.New("modern HD heal: terrain cell absent")
		}
		cell := j.hdInput.Frame.Cells[at]
		var shades [4]*image.RGBA
		for count := 0; count < 4; count++ {
			mapped := append([]byte(nil), luts[count]...)
			if cell.BlitMode != 0xff {
				for n := range mapped {
					mapped[n] = luts[count][j.hdInput.Frame.LUT[n]]
				}
			}
			ctx, e := terrainColourContext(g.nativeMapAssets.MapIndex, g.nativeMapAssets.Palette, j.palette, mapped)
			if e != nil {
				return nil, e
			}
			shades[count] = assets[tile.Frame].shade(ctx, false)
		}
		picture := image.NewRGBA(image.Rect(0, 0, 144, 144))
		for y := 0; y < 144; y++ {
			for x := 0; x < 144; x++ {
				c := healHDCountAt(tc, (rect.Min.X+x)/6, (rect.Min.Y+y)/6)
				picture.SetRGBA(x, y, shades[c].RGBAAt(x, y))
			}
		}
		f.Terrain = append(f.Terrain, modernHDTerrainDraw{Image: picture, Rect: rect})
	}
	// Use existing HD foreground images for recoloured units, preserving the
	// original order: all units, then the actual foreground. No GPU API changes.
	f.Units = nil
	f.Foreground = nil
	ctx, e := terrainColourContext(0, j.palette, j.palette, lut)
	if e != nil {
		return nil, e
	}
	colours := make([]byte, 256)
	for i := range colours {
		colours[i] = byte(i)
	}
	weights := make(map[uint32]modernTerrainColourWeights)
	shade := func(src image.Image, rect image.Rectangle, band bool) *image.RGBA {
		out := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
		for y := 0; y < rect.Dy(); y++ {
			for x := 0; x < rect.Dx(); x++ {
				v := color.NRGBAModel.Convert(src.At(src.Bounds().Min.X+x*src.Bounds().Dx()/rect.Dx(), src.Bounds().Min.Y+y*src.Bounds().Dy()/rect.Dy())).(color.NRGBA)
				if v.A == 0 {
					continue
				}
				if band {
					v.R = byte(math.Round(float64(v.R) * .55))
					v.G = byte(math.Round(float64(v.G) * .55))
					v.B = byte(math.Round(float64(v.B) * .6))
				}
				count := healHDCountAt(oc, (rect.Min.X+x)/6, (rect.Min.Y+y)/6)
				if count != 0 {
					key := uint32(v.R)<<16 | uint32(v.G)<<8 | uint32(v.B)
					w, ok := weights[key]
					if !ok {
						w = terrainColourWeights(color.RGBA{v.R, v.G, v.B, 255}, colours, ctx.Base)
						weights[key] = w
					}
					values := [3]float64{float64(v.R), float64(v.G), float64(v.B)}
					for k, weight := range w.Weights {
						n := w.Indices[k]
						r, gg, b, _ := j.palette[luts[count][n]].RGBA()
						to := [3]float64{float64(r >> 8), float64(gg >> 8), float64(b >> 8)}
						for c := range values {
							values[c] += weight * (to[c] - ctx.Base[n][c])
						}
					}
					clamp := func(v float64) byte { return byte(math.Round(math.Max(0, math.Min(255, v)))) }
					v.R, v.G, v.B = clamp(values[0]), clamp(values[1]), clamp(values[2])
				}
				out.Set(x, y, v)
			}
		}
		return out
	}
	for _, u := range normal.Units {
		rect := u.Rect.Add(image.Pt(-24, -24))
		f.Foreground = append(f.Foreground, modernHDMapForegroundDraw{HDImage: shade(set.mapSpritesHD[u.Group][u.Frame], rect, u.PaletteBand), Rect: rect})
	}
	for _, fg := range normal.Foreground {
		rect := fg.Rect.Add(image.Pt(-24, -24))
		picture := fg.HDImage
		if picture != nil {
			picture = shade(picture, rect, false)
		} else {
			picture = image.NewRGBA(image.Rect(0, 0, 144, 144))
			sprite := j.hdInput.Frame.ForegroundBank.Sprites[fg.SourceFrame]
			for y := 0; y < 144; y++ {
				for x := 0; x < 144; x++ {
					at := y/6*24 + x/6
					if sprite.Mask[at] == 0 {
						continue
					}
					v := sprite.Pixels[at]
					if fg.BlitMode != 0xff {
						v = j.hdInput.Frame.LUT[v]
					}
					v = luts[healHDCountAt(oc, (rect.Min.X+x)/6, (rect.Min.Y+y)/6)][v]
					picture.Set(x, y, j.palette[v])
				}
			}
		}
		f.Foreground = append(f.Foreground, modernHDMapForegroundDraw{HDImage: picture, Rect: rect})
	}
	return f, nil
}

func (g *Game) prepareModernHDHeal(j *nativeCommandHealPresentationJob, set *modernStoryPortraitSet) (*modernHDHealFrames, error) {
	if j == nil || set == nil {
		return nil, errors.New("modern HD heal: owner/theme absent")
	}
	has, err := validateHealHDBanks(g, set)
	if err != nil {
		return nil, err
	}
	if len(set.mapSpritesHD) != 125 {
		if has {
			return nil, errors.New("modern HD heal: incomplete HD world")
		}
		return nil, nil
	}
	if j.hdInputErr != nil {
		return nil, j.hdInputErr
	}
	if len(j.hdPasses) != 16 || len(j.hdLUTs) != 16 || len(j.effectSources) != 7 {
		return nil, errors.New("modern HD heal: incomplete producer")
	}
	if j.hdCachedTheme == set && j.hdCached != nil && len(j.hdCached.Digits) == len(j.digitFrames) {
		return j.hdCached, nil
	}
	steady, err := g.prepareModernHDCommandSteady(j.hdBaseline, j.baselineWork, j.baselineVGA, j.palette, set)
	if err != nil {
		return nil, err
	}
	objects, err := g.prepareModernHDCommandSteady(j.hdInput, j.baselineWork, j.baselineVGA, j.palette, set)
	if err != nil {
		return nil, err
	}
	frames := &modernHDHealFrames{Steady: steady, Event: modernPlayer9EventFrame(steady, j.baselineVGA, j.palette)}
	for index := range j.hdPasses {
		f, e := g.healHDFront(j, set, objects, index)
		if e != nil {
			return nil, e
		}
		frames.Front = append(frames.Front, f)
	}
	layers := func(sources []indexedmap.NativeCommandTailLayer) (*modernHDMapFrame, error) {
		f := *frames.Event
		for _, source := range sources {
			bank, entries := set.mapEffects, g.nativeMapAssets.FDOTHER6
			if source.Resource == 5 {
				bank, entries = set.mapOverlays, g.nativeMapAssets.CommandHealDigits
			} else if source.Resource != 6 {
				return nil, errors.New("modern HD heal: unknown LMI source")
			}
			if source.Entry < 0 || source.Entry >= len(entries) {
				return nil, errors.New("modern HD heal: source absent")
			}
			entry := entries[source.Entry]
			var pic *image.RGBA
			if has {
				pic = bank.frames[source.Entry]
				if pic == nil || bank.native[source.Entry] != modernLMIHDDescription(source.Entry, entry) {
					return nil, errors.New("modern HD heal: bound source changed")
				}
			} else {
				pic = image.NewRGBA(image.Rect(0, 0, entry.Width*6, entry.Height*6))
				for y := 0; y < entry.Height*6; y++ {
					for x := 0; x < entry.Width*6; x++ {
						v := entry.Pixels[y/6*entry.Width+x/6]
						if v != 0 {
							pic.Set(x, y, j.palette[v])
						}
					}
				}
			}
			f.Overlays = append(f.Overlays, modernHDMapOverlayDraw{Entry: source.Entry, Image: pic, Rect: pic.Bounds().Add(image.Pt(source.X*6, source.Y*6))})
		}
		return &f, nil
	}
	for _, s := range j.effectSources {
		f, e := layers(s)
		if e != nil {
			return nil, e
		}
		frames.Effects = append(frames.Effects, f)
	}
	for _, s := range j.digitSources {
		f, e := layers(s)
		if e != nil {
			return nil, e
		}
		frames.Digits = append(frames.Digits, f)
	}
	mask := *frames.Event
	cycle := j.hdInput.Frame.IdleCycle
	if cycle == 3 {
		cycle = 2
	}
	for _, target := range j.effectTargets {
		group, e := j.hdInput.Frame.SelectorCache.KeyForSlot(target.SelectorSlot)
		if e != nil {
			return nil, e
		}
		images := set.mapSpritesHD[group]
		if cycle < 0 || cycle >= len(images) {
			return nil, errors.New("modern HD heal: mask selector absent")
		}
		im := images[cycle]
		pic := image.NewRGBA(image.Rect(0, 0, 144, 144))
		r, gg, b, _ := j.palette[j.tailSchedule.MaskIndex].RGBA()
		for y := 0; y < 144; y++ {
			for x := 0; x < 144; x++ {
				_, _, _, alpha := im.At(x, y).RGBA()
				pic.Set(x, y, color.NRGBA{byte(r >> 8), byte(gg >> 8), byte(b >> 8), byte(alpha >> 8)})
			}
		}
		x := (target.X - j.hdInput.Frame.CameraX) * 24
		y := (target.Y-j.hdInput.Frame.CameraY)*24 - 6
		mask.Overlays = append(mask.Overlays, modernHDMapOverlayDraw{Entry: -1, Image: pic, Rect: pic.Bounds().Add(image.Pt(x*6, y*6))})
	}
	frames.Mask = &mask
	j.hdCached, j.hdCachedTheme = frames, set
	return frames, nil
}

// Keep the only native geometry copy identical to the producer, including its
// distinct camera sources. The steady snapshot has its own frozen provenance.
func (g *Game) freezeHealHDInput(in indexedmap.NativeTransitionFrameInput) (indexedmap.NativeFrameInput, indexedmap.NativeFrameInput, error) {
	baseline, err := g.freezeNativeCommand9MapInput()
	if err != nil {
		return baseline, baseline, err
	}
	geometry := baseline
	f := &geometry.Frame
	f.TerrainBank, f.UnitBank, f.ForegroundBank, f.SelectorCache = in.TerrainBank, in.UnitBank, in.ForegroundBank, in.SelectorCache
	f.Cells, f.Controls, f.LUT = in.Cells, in.Controls, in.TerrainLUT
	f.MapWidth, f.CameraX, f.CameraY = in.MapWidth, in.CameraX, in.CameraY
	f.Flip, f.TerrainCycle, f.IdleCycle, f.MovingCycle, f.PixelShift = in.Flip, in.TerrainCycle, in.IdleCycle, in.MovingCycle, in.PixelShift
	f.Units, f.ForegroundUnits = in.Units, in.ForegroundUnits
	f.RangeMode = 0
	f.SelectionOverlay = nil
	f.ChapterAux = nil
	f.Parallax, f.ParallaxScrollX, f.ParallaxScrollY = in.Parallax, in.ParallaxScrollX, in.ParallaxScrollY
	geometry, err = freezeModernHDCommandGeometry(geometry)
	return geometry, baseline, err
}
