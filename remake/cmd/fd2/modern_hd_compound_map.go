package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// A scene belongs to one frozen pre/post-transaction map, not the live state.
type modernHDCompoundMapScene struct {
	input     indexedmap.NativeFrameInput
	work, vga []byte
	palette   color.Palette
	err       error
}

type modernHDCompoundMapSource struct {
	scene       *modernHDCompoundMapScene
	pixels      []byte
	layers      []indexedmap.NativeCommandTailLayer
	maskTargets []indexedmap.NativeCommandHealTailTarget
	maskIndex   byte
	event       bool
	mix         float32
	target      [3]float32
	prepared    *modernHDMapFrame
	theme       *modernStoryPortraitSet
}

func newModernHDCompoundMapScene(in indexedmap.NativeFrameInput, work, vga []byte, palette color.Palette, inputErr error) *modernHDCompoundMapScene {
	s := &modernHDCompoundMapScene{work: append([]byte(nil), work...), vga: append([]byte(nil), vga...), palette: append(color.Palette(nil), palette...), err: inputErr}
	if s.err == nil {
		s.input, s.err = freezeModernHDCommandGeometry(in)
	}
	if s.err == nil {
		w, v := append([]byte(nil), work...), append([]byte(nil), vga...)
		s.err = indexedmap.ComposeNativeFrame(w, v, s.input)
		if s.err == nil && !bytes.Equal(v, vga) {
			s.err = fmt.Errorf("modern HD compound map: frozen input differs from native VGA")
		}
	}
	return s
}

func bindModernHDCompoundMap(frames []nativeCompoundPresentedFrame, scene *modernHDCompoundMapScene, pixels [][]byte,
	layers [][]indexedmap.NativeCommandTailLayer, targets []indexedmap.NativeCommandHealTailTarget, maskIndex byte, event bool, phase string) error {
	if scene == nil || len(frames) == 0 || len(frames) != len(pixels) || layers != nil && len(layers) != len(frames) {
		return fmt.Errorf("modern HD compound map: frame/source count differs")
	}
	for i := range frames {
		if len(pixels[i]) != 64000 || frames[i].hdSources != nil || frames[i].hdMap != nil {
			return fmt.Errorf("modern HD compound map: native frame unavailable or already bound")
		}
	}
	for i := range frames {
		s := &modernHDCompoundMapSource{scene: scene, pixels: append([]byte(nil), pixels[i]...), event: event,
			maskTargets: append([]indexedmap.NativeCommandHealTailTarget(nil), targets...), maskIndex: maskIndex}
		if layers != nil {
			s.layers = append([]indexedmap.NativeCommandTailLayer(nil), layers[i]...)
		}
		frames[i].hdMap, frames[i].hdPhase = s, phase
	}
	return nil
}

func bindModernHDCompoundMapRamp(frames []nativeCompoundPresentedFrame, scene *modernHDCompoundMapScene, pixels [][]byte, rgb [3]byte) error {
	if len(frames) != 41 {
		return fmt.Errorf("modern HD compound map: ramp length differs")
	}
	if err := bindModernHDCompoundMap(frames, scene, pixels, nil, nil, 0, false, "map-ramp"); err != nil {
		return err
	}
	for i := range frames {
		frames[i].hdMap.mix = 1 - float32(i)/40
		for c, v := range rgb {
			frames[i].hdMap.target[c] = float32((v<<2)|(v>>4)) / 255
		}
	}
	return nil
}

func bindModernHDCompoundMapMask(frames []nativeCompoundPresentedFrame, scene *modernHDCompoundMapScene, baseline, masked []byte, targets []indexedmap.NativeCommandHealTailTarget, index byte, phase string) error {
	return bindModernHDCompoundMapMaskStage(frames, scene, scene, [][]byte{baseline, masked}, targets, index, phase)
}

func bindModernHDCompoundMapMaskStage(frames []nativeCompoundPresentedFrame, initial, stage *modernHDCompoundMapScene, pixels [][]byte, targets []indexedmap.NativeCommandHealTailTarget, index byte, phase string) error {
	if len(pixels) != 2 || len(frames) != 11 {
		return fmt.Errorf("modern HD compound map: mask sequence differs")
	}
	for i := range frames {
		scene, mask, event := initial, []indexedmap.NativeCommandHealTailTarget(nil), false
		if i%2 == 1 {
			scene, mask, event = stage, targets, true
		}
		if err := bindModernHDCompoundMap(frames[i:i+1], scene, pixels[i%2:i%2+1], nil, mask, index, event, phase); err != nil {
			return err
		}
	}
	return nil
}

// Prepare every candidate privately. A failed F2 cannot replace the active map.
func (g *Game) stageModernHDCompoundMap(frames []nativeCompoundPresentedFrame, set *modernStoryPortraitSet) (func(), error) {
	noop := func() {}
	if len(frames) == 0 || set == nil || len(set.mapSpritesHD) == 0 {
		return noop, nil
	}
	if len(set.mapSpritesHD) != 125 {
		return nil, fmt.Errorf("modern HD compound map: incomplete HD world")
	}
	type preparedScene struct{ steady, event *modernHDMapFrame }
	scenes := make(map[*modernHDCompoundMapScene]preparedScene)
	type imageKey struct {
		scene           *modernHDCompoundMapScene
		resource, index int
	}
	images := make(map[imageKey]*image.RGBA)
	prepared := make(map[*modernHDCompoundMapSource]*modernHDMapFrame)
	for _, frame := range frames {
		s := frame.hdMap
		if s == nil {
			continue
		}
		if s.theme == set && s.prepared != nil {
			prepared[s] = s.prepared
			continue
		}
		scene := s.scene
		if scene == nil {
			return nil, fmt.Errorf("modern HD compound map: frozen source absent")
		}
		if scene.err != nil {
			return nil, fmt.Errorf("modern HD compound map: frozen source unavailable: %w", scene.err)
		}
		if len(scene.palette) != 256 || len(scene.work) != indexedmap.NativeUnitPresentWorkSize || len(scene.vga) != 64000 {
			return nil, fmt.Errorf("modern HD compound map: incomplete snapshot")
		}
		base, ok := scenes[scene]
		if !ok {
			steady, err := g.prepareModernHDCommandSteady(scene.input, scene.work, scene.vga, scene.palette, set)
			if err != nil {
				return nil, err
			}
			base = preparedScene{steady: steady, event: modernPlayer9EventFrame(steady, scene.vga, scene.palette)}
			scenes[scene] = base
		}
		picture := *base.steady
		if s.event {
			picture = *base.event
		}
		picture.CompoundOwner = scene
		picture.Overlays = append([]modernHDMapOverlayDraw(nil), picture.Overlays...)
		work, vga := append([]byte(nil), scene.work...), append([]byte(nil), scene.vga...)
		for _, source := range s.layers {
			entries, bank := g.nativeMapAssets.FDOTHER6, set.mapEffects
			if source.Resource == 5 {
				entries, bank = g.nativeMapAssets.CommandHealDigits, set.mapOverlays
			} else if source.Resource != 6 {
				return nil, fmt.Errorf("modern HD compound map: invalid LMI resource")
			}
			if source.Entry < 0 || source.Entry >= len(entries) {
				return nil, fmt.Errorf("modern HD compound map: missing LMI entry")
			}
			entry := entries[source.Entry]
			if err := entry.BlitAt(work, 456, source.X+0x8088%456, source.Y+0x8088/456, false); err != nil {
				return nil, err
			}
			key := imageKey{scene, source.Resource, source.Entry}
			im := images[key]
			if im == nil {
				if bank != nil {
					im = bank.frames[source.Entry]
					_, declared := bank.native[source.Entry]
					if declared || im != nil {
						if im == nil || bank.native[source.Entry] != modernLMIHDDescription(source.Entry, entry) || im.Bounds() != image.Rect(0, 0, entry.Width*6, entry.Height*6) {
							return nil, fmt.Errorf("modern HD compound map: changed declared LMI source %d:%d", source.Resource, source.Entry)
						}
					}
				}
				if im == nil {
					im = image.NewRGBA(image.Rect(0, 0, entry.Width*6, entry.Height*6))
					for y := 0; y < entry.Height*6; y++ {
						for x := 0; x < entry.Width*6; x++ {
							v := entry.Pixels[y/6*entry.Width+x/6]
							if v != 0 {
								im.Set(x, y, scene.palette[v])
							}
						}
					}
				}
				images[key] = im
			}
			picture.Overlays = append(picture.Overlays, modernHDMapOverlayDraw{Entry: source.Entry, Image: im, Rect: im.Bounds().Add(image.Pt(source.X*6, source.Y*6))})
		}
		if len(s.maskTargets) > 0 {
			in := scene.input.Frame
			if err := indexedmap.ComposeNativeCommandHealMaskFrame(work, vga, scene.work, g.nativeMapAssets.Units, in.SelectorCache, s.maskTargets, in.CameraX, in.CameraY, in.IdleCycle, s.maskIndex); err != nil {
				return nil, err
			}
			cycle := in.IdleCycle
			if cycle == 3 {
				cycle = 2
			}
			for _, target := range s.maskTargets {
				group, err := in.SelectorCache.KeyForSlot(target.SelectorSlot)
				if err != nil {
					return nil, err
				}
				list := set.mapSpritesHD[group]
				if cycle < 0 || cycle >= len(list) || list[cycle] == nil {
					return nil, fmt.Errorf("modern HD compound map: missing mask selector")
				}
				im := list[cycle]
				mask := image.NewRGBA(image.Rect(0, 0, 144, 144))
				red, green, blue, _ := scene.palette[s.maskIndex].RGBA()
				for y := 0; y < 144; y++ {
					for x := 0; x < 144; x++ {
						_, _, _, alpha := im.At(im.Bounds().Min.X+x, im.Bounds().Min.Y+y).RGBA()
						mask.Set(x, y, color.NRGBA{byte(red >> 8), byte(green >> 8), byte(blue >> 8), byte(alpha >> 8)})
					}
				}
				x := (target.X - in.CameraX) * 24
				y := (target.Y-in.CameraY)*24 - 6
				picture.Overlays = append(picture.Overlays, modernHDMapOverlayDraw{Entry: -1, Image: mask, Rect: mask.Bounds().Add(image.Pt(x*6, y*6))})
			}
		} else if s.event {
			if err := indexedmap.CopyNativeUnitPresentViewport(vga, work); err != nil {
				return nil, err
			}
		}
		if !bytes.Equal(vga, s.pixels) {
			return nil, fmt.Errorf("modern HD compound map: native %s reconstruction differs", frame.hdPhase)
		}
		prepared[s] = &picture
	}
	return func() {
		for source, picture := range prepared {
			source.prepared, source.theme = picture, set
		}
	}, nil
}

func (g *Game) modernCompoundMapFrames() []nativeCompoundPresentedFrame {
	switch {
	case g.nativeCmd32Presentation != nil:
		return g.nativeCmd32Presentation.frames
	case g.nativeCmd33Presentation != nil:
		return g.nativeCmd33Presentation.frames
	case g.nativeCmd34Presentation != nil:
		return g.nativeCmd34Presentation.frames
	case g.nativeCmd35Presentation != nil:
		return g.nativeCmd35Presentation.frames
	}
	return nil
}

func (g *Game) drawModernHDCompoundMapFrame(screen *ebiten.Image, frame *nativeCompoundPresentedFrame) (bool, error) {
	s := frame.hdMap
	if s == nil || s.prepared == nil || s.theme != g.modernStoryPortraits || g.modernStoryPortraits == nil {
		return false, nil
	}
	if !g.drawPreparedModernHDMapFrame(screen, s.prepared, false) {
		return false, fmt.Errorf("modern HD compound map: prepared world not drawn")
	}
	if s.mix != 0 {
		r := g.modernHDMapGPU
		if r.CompoundMix == nil {
			r.CompoundMix = ebiten.NewImage(modernHDWidth, modernHDHeight)
		}
		if r.CompoundShader == nil {
			var err error
			r.CompoundShader, err = ebiten.NewShader([]byte(modernHDBattleShader))
			if err != nil {
				return false, err
			}
		}
		r.CompoundMix.Clear()
		op := &ebiten.DrawRectShaderOptions{}
		op.Images[0] = r.World
		op.Uniforms = map[string]interface{}{"FillColor": []float32{0, 0, 0, 0}, "Subtract": float32(0), "Tint": s.target[:], "Mix": s.mix}
		r.CompoundMix.DrawRectShader(modernHDWidth, modernHDHeight, r.CompoundShader, op)
		r.World.Clear()
		r.World.DrawImage(r.CompoundMix, nil)
	}
	return true, nil
}
