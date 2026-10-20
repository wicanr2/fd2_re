package main

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

type modernHDBattleSource struct {
	Archive                       string
	Frame                         figani.Frame
	Resource, Index, DX, DY, Fill int
	Clip                          *image.Rectangle
}

type modernHDBattleSources struct {
	Base   []byte
	Layers []modernHDBattleSource
}

func modernHDBattleBodySources(base []byte, attack, idle figani.Frame, side byte,
	present battlepresent.NativePhysicalBodyPresent, attackResource, attackFrame, idleResource, idleFrame int) *modernHDBattleSources {
	a := modernHDBattleSource{Frame: attack, Resource: attackResource, Index: attackFrame, Fill: -1}
	b := modernHDBattleSource{Frame: idle, Resource: idleResource, Index: idleFrame, DX: present.DX, DY: present.DY, Fill: present.OpaqueFill}
	layers := []modernHDBattleSource{b, a}
	if (side == 0 && attack.RawByte7&1 == 0) || (side != 0 && attack.RawByte7&1 != 0) {
		layers = []modernHDBattleSource{a, b}
	}
	return &modernHDBattleSources{Base: base, Layers: layers}
}

func (scene *nativePhysicalScene) modernResources() []int {
	if scene == nil {
		return nil
	}
	resources := []int{scene.actorBattleFig * 3, scene.actorBattleFig*3 + 1, scene.targetBattleFig * 3}
	if scene.bodyResources != nil && scene.bodyResources.counter != nil {
		resources = append(resources, scene.targetBattleFig*3+1)
	}
	return resources
}

func (g *Game) modernHDBattleWanted(scene *nativePhysicalScene) bool {
	set := g.modernStoryPortraits
	if set == nil || !set.hasHD || scene == nil || scene.bodyResources == nil || scene.bodyResources.attack == nil {
		return false
	}
	for _, resource := range scene.modernResources() {
		if set.battleAnimations[resource] != nil {
			return true
		}
	}
	return false
}

func (g *Game) syncModernHDBattleOwner() {
	if g.modernHDBattleGPU == nil {
		return
	}
	var owner any
	if g.atk != nil {
		owner = g.atk.nativeScene
	}
	switch {
	case g.nativeCmd0Presentation != nil:
		owner = g.nativeCmd0Presentation
	case g.nativeCmd32Presentation != nil:
		owner = g.nativeCmd32Presentation
	case g.nativeCmd33Presentation != nil:
		owner = g.nativeCmd33Presentation
	case g.nativeCmd34Presentation != nil:
		owner = g.nativeCmd34Presentation
	case g.nativeCmd35Presentation != nil:
		owner = g.nativeCmd35Presentation
	}
	r := g.modernHDBattleGPU
	if r.owner != owner || r.theme != g.modernStoryPortraits {
		r.resetSources()
		r.owner, r.theme = owner, g.modernStoryPortraits
	}
}

func (g *Game) modernHDBattlePlan(sources *modernHDBattleSources, palette color.Palette, subtract float32) (modernHDBattlePlan, error) {
	if sources == nil {
		return modernHDBattlePlan{}, fmt.Errorf("modern HD battle: sources unavailable")
	}
	plan := modernHDBattlePlan{Base: sources.Base, Palette: palette, HDSubtract: subtract}
	for _, source := range sources.Layers {
		var picture image.Image
		var key string
		if source.Resource >= 0 {
			var err error
			var entries map[int]*modernBattleHDResource
			if g.modernStoryPortraits != nil {
				entries = g.modernStoryPortraits.battleAnimations
			}
			switch source.Archive {
			case "", "FIGANI.DAT":
			case "FDOTHER.DAT":
				if g.modernStoryPortraits != nil {
					entries = g.modernStoryPortraits.battleEffects
				}
			default:
				return modernHDBattlePlan{}, fmt.Errorf("modern HD battle: unsupported source archive")
			}
			picture, err = modernBattleHDFrame(entries, source.Resource, source.Index)
			if err != nil {
				return modernHDBattlePlan{}, err
			}
			if picture != nil {
				key = entries[source.Resource].hashes[source.Index]
			}
		}
		f := source.Frame
		rect := image.Rect((f.X+source.DX)*6, (f.Y+source.DY)*6, (f.X+source.DX+f.Width)*6, (f.Y+source.DY+f.Height)*6)
		var clip *image.Rectangle
		if source.Clip != nil {
			c := source.Clip
			if c.Min.X > c.Max.X || c.Min.Y > c.Max.Y || c.Min.X < -(1<<20)/6 || c.Min.Y < -(1<<20)/6 || c.Max.X > (1<<20)/6 || c.Max.Y > (1<<20)/6 {
				return modernHDBattlePlan{}, fmt.Errorf("modern HD battle: invalid source clip")
			}
			value := image.Rectangle{Min: c.Min.Mul(6), Max: c.Max.Mul(6)}
			clip = &value
		}
		plan.Layers = append(plan.Layers, modernHDBattleLayer{Native: f, HD: picture, HDKey: key, Rect: rect, Fill: source.Fill, Clip: clip})
	}
	return plan, nil
}

func (g *Game) publishModernHDBattle(screen *ebiten.Image, sources *modernHDBattleSources, palette color.Palette, subtract float32) error {
	plan, err := g.modernHDBattlePlan(sources, palette, subtract)
	if err != nil {
		return err
	}
	return g.publishModernHDBattlePlan(screen, plan)
}

func (g *Game) publishModernHDBattlePlan(screen *ebiten.Image, plan modernHDBattlePlan) error {
	if g.modernHDBattleGPU == nil {
		g.modernHDBattleGPU = &modernHDBattleGPU{}
	}
	g.syncModernHDBattleOwner()
	if err := g.modernHDBattleGPU.draw(plan); err != nil {
		return err
	}
	screen.Clear()
	g.modernHDBattleVisible = true
	return nil
}

func (g *Game) modernHDBattlePreludeSources(scene *nativePhysicalScene, stage int) (*modernHDBattleSources, error) {
	if scene == nil || scene.bodyResources == nil || stage < 0 || stage > 8 || len(scene.preludeBase) != 64000 {
		return nil, fmt.Errorf("modern HD battle: prelude sources unavailable")
	}
	r := scene.bodyResources
	base := append([]byte(nil), scene.preludeBase...)
	a := modernHDBattleSource{Frame: r.actorIdle.Frames[0], Resource: scene.actorBattleFig * 3, Fill: -1}
	b := modernHDBattleSource{Frame: r.targetIdle.Frames[0], Resource: scene.targetBattleFig * 3, Fill: -1}
	if r.actorSide == 0 {
		if !scene.selection.HasSeparateBackgrounds {
			platform := r.platform
			platform.X, platform.Y = 164, 157
			if err := platform.Blit(base, 320, -1); err != nil {
				return nil, err
			}
		}
		a.DX = -stage * 10
		if scene.selection.HasSeparateBackgrounds {
			return &modernHDBattleSources{Base: base, Layers: []modernHDBattleSource{a}}, nil
		}
		return &modernHDBattleSources{Base: base, Layers: []modernHDBattleSource{a, b}}, nil
	}
	platform := r.platform
	p := modernHDBattleSource{Frame: figani.Frame{X: 164, Y: 157, Width: platform.Width, Height: platform.Height,
		Pixels: platform.Indexed, Mask: platform.Mask}, Resource: -1, DX: stage * 10, Fill: -1}
	a.DX = stage * 10
	if scene.selection.HasSeparateBackgrounds {
		return &modernHDBattleSources{Base: base, Layers: []modernHDBattleSource{p, a}}, nil
	}
	return &modernHDBattleSources{Base: base, Layers: []modernHDBattleSource{b, p, a}}, nil
}

func (g *Game) drawModernHDBattlePrelude(screen *ebiten.Image, scene *nativePhysicalScene) (bool, error) {
	if !g.modernHDBattleWanted(scene) {
		return false, nil
	}
	frame := scene.prelude[scene.preludeFrame]
	sources, err := g.modernHDBattlePreludeSources(scene, frame.Stage)
	if err != nil {
		return false, err
	}
	palette, err := fdother.ParseVGAPalette(frame.DAC)
	if err != nil {
		return false, err
	}
	if err := g.publishModernHDBattle(screen, sources, palette, float32(frame.Stage*6)/63); err != nil {
		return false, err
	}
	return true, nil
}

func (g *Game) drawModernHDBattleBody(screen *ebiten.Image, scene *nativePhysicalScene, job *nativePhysicalBodyJob) (bool, error) {
	if !g.modernHDBattleWanted(scene) || job.hdSources == nil {
		return false, nil
	}
	palette := append(color.Palette(nil), g.nativeUIPalette...)
	if job.palette0 != nil {
		palette[0] = *job.palette0
	}
	if err := g.publishModernHDBattle(screen, job.hdSources, palette, 0); err != nil {
		return false, err
	}
	return true, nil
}
