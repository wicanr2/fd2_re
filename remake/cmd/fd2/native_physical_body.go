package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// nativePhysicalBodyResources 在方向、HP與RNG修改前建立。
// 正式入口與證據：docs/data/ida/fd2_physical_background_selection_20261004.json。
type nativePhysicalBodyResources struct {
	attack, counter, actorIdle, targetIdle *figani.Animation
	actorBG, targetBG                      fdother.Frame // 原版 54103／54107，反擊不交換。
	layers                                 [3]fdother.Frame
	platform                               fdother.Frame
	base                                   []byte // 29164 返回後的共享 base。
	actorSide, targetSide                  byte
	mainSounds, counterSounds              map[int][]byte
}

// 一個工作項目可只有cue；只有present項目才等待正式Draw。
// body項目共用不可變的panel base，不為每個idle tick建立GPU貼圖。
type nativePhysicalBodyJob struct {
	hdSources    *modernHDBattleSources
	hdPhase      string
	pixels       []byte
	base         []byte
	attack, idle figani.Frame
	present      *battlepresent.NativePhysicalBodyPresent
	rawSide      byte
	palette0     *color.NRGBA
	waitMillis   float64
	cue          int
	sounds       map[int][]byte
}

type nativePhysicalBodyPlayback struct {
	jobs              []nativePhysicalBodyJob
	index             int
	drawn, cued       bool
	elapsedMillis     float64
	image             *ebiten.Image
	lastPresent       int
	returnWaitMillis  float64 // 正常2909D／2909F的17AA9(6)，與descriptor等待分開。
	returnRampStarted bool
}

func (g *Game) prepareNativePhysicalBodyResources(scene *nativePhysicalScene, actor, target *battle.Unit, attack *figani.Animation) error {
	r := &nativePhysicalBodyResources{attack: attack, actorSide: actor.NativeRecordByte6, targetSide: target.NativeRecordByte6, actorBG: scene.backgroundFrame}
	var err error
	r.actorIdle, err = figani.LoadSeparatedResource(separatedAssetPath("animations"), actor.BattleFig*3)
	if err != nil {
		return err
	}
	r.targetIdle, err = figani.LoadSeparatedResource(separatedAssetPath("animations"), target.BattleFig*3)
	if err != nil {
		return err
	}
	if g.st.NativePhysicalCounterattackEligible(actor, target) {
		r.counter, err = figani.LoadSeparatedResource(separatedAssetPath("animations"), target.BattleFig*3+1)
		if err != nil {
			return err
		}
		if attack.HeaderByte1 == 0 && r.counter.HeaderByte1 != 0 {
			return errors.New("native physical counter requires unavailable caller BG layers")
		}
	}
	r.platform, err = fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "TAI.DAT", int(scene.selection.TAI))
	if err != nil {
		return err
	}
	r.targetBG = scene.backgroundFrame
	if scene.selection.HasSeparateBackgrounds {
		r.targetBG = scene.targetBackground
		for i := range r.layers {
			r.layers[i], err = fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "BG.DAT", i)
			if err != nil {
				return err
			}
		}
	}
	r.base, err = g.nativePhysicalBase(scene, scene.actorRecord, scene.targetRecord)
	if err != nil {
		return err
	}
	// 29164: 29258..29271在raw side非零時無條件保存TAI；
	// 2927E..2929E在raw side零且mode零時保存TAI。
	if r.actorSide != 0 || attack.HeaderByte1 == 0 {
		tai := r.platform
		tai.X, tai.Y = 164, 157
		if err := tai.Blit(r.base, 320, -1); err != nil {
			return err
		}
	}
	for _, pair := range [][2]*figani.Animation{{r.attack, r.targetIdle}, {r.counter, r.actorIdle}} {
		if pair[0] == nil {
			continue
		}
		if _, err := battlepresent.BuildNativePhysicalBodyPlan(pair[0], pair[1], 0, 1, []battle.NativePhysicalStrike{{}}); err != nil {
			return err
		}
		for _, a := range pair {
			for _, f := range a.Frames {
				if f.Width <= 0 || f.Height <= 0 || f.Width > 1024 || f.Height > 1024 || len(f.Pixels) != f.Width*f.Height || len(f.Mask) != len(f.Pixels) {
					return errors.New("native physical body FIGANI shape unavailable")
				}
			}
		}
		if pair[0].HeaderByte1 != 0 {
			for _, f := range pair[0].Frames[:int(pair[0].HeaderByte2)] {
				if err := f.BlitAt(make([]byte, 64000), 320); err != nil {
					return err
				}
			}
		}
	}
	r.mainSounds, err = g.nativePhysicalSoundBank(attack)
	if err != nil {
		return err
	}
	if r.counter != nil {
		r.counterSounds, err = g.nativePhysicalSoundBank(r.counter)
		if err != nil {
			return err
		}
	}
	if g.modernStoryPortraits != nil {
		if err := g.modernStoryPortraits.validatePhysicalScenery(scene); err != nil {
			return err
		}
		resources := []int{actor.BattleFig * 3, actor.BattleFig*3 + 1, target.BattleFig * 3}
		if r.counter != nil {
			resources = append(resources, target.BattleFig*3+1)
		}
		if err := g.modernStoryPortraits.prepareBattleHDResources(resources...); err != nil {
			return fmt.Errorf("native physical HD source preflight: %w", err)
		}
	}
	scene.bodyResources = r
	return nil
}

// 2BC9A索引525D6的bytes30..35；header4零表示無bank。
// 既有證據：docs/data/ida/fd2_command24_presentation_ida.txt。
func (g *Game) nativePhysicalSoundBank(animation *figani.Animation) (map[int][]byte, error) {
	if animation.HeaderByte4 == 0 {
		return nil, nil
	}
	if animation.HeaderByte4 > 6 {
		return nil, errors.New("native physical FIGANI sound selector unavailable")
	}
	resource := 48 + int(animation.HeaderByte4) - 1
	if g.nativePhysicalSoundBanks == nil {
		g.nativePhysicalSoundBanks = make(map[int]map[int][]byte)
	}
	bank := g.nativePhysicalSoundBanks[resource]
	if bank == nil {
		var err error
		bank, err = decodeSeparatedSoundBank(resource)
		if err != nil {
			return nil, err
		}
		g.nativePhysicalSoundBanks[resource] = bank
	}
	for _, f := range animation.Frames {
		if f.RawByte5 != 0 && (len(bank[int(f.RawByte5)]) == 0 || (f.RawByte4 != 0 && len(bank[0]) == 0)) {
			return nil, fmt.Errorf("native physical FDOTHER%d cue%d or miss cue0 unavailable", resource, f.RawByte5)
		}
	}
	return bank, nil
}

func (g *Game) nativePhysicalScroll(scene *nativePhysicalScene, source []byte, background fdother.Frame, record []byte, index int, idle figani.Frame, right bool) ([]byte, []battlepresent.NativeCommand24BackgroundPresent, error) {
	base := make([]byte, 64000) // 29C90／29DED 的明確 base memset。
	background.X, background.Y = 0, 50
	if err := background.Blit(base, 320, -1); err != nil {
		return nil, nil, err
	}
	if right {
		platform := scene.bodyResources.platform
		platform.X, platform.Y = 164, 157
		if err := platform.Blit(base, 320, -1); err != nil {
			return nil, nil, err
		}
	}
	if err := g.renderPhysicalBattlePanel(scene, record, base, index); err != nil {
		return nil, nil, err
	}
	in := battlepresent.NativeCommand24BackgroundInputs{Layers: scene.bodyResources.layers, Source: source, Target: base, TargetIdle: idle}
	frames, err := nativePhysicalBackgroundPresents(in, right)
	return base, frames, err
}

func (g *Game) attachNativePhysicalBody(scene *nativePhysicalScene, result battle.NativePhysicalAttackResult) error {
	if scene == nil {
		return nil // 無原生地圖狀態的既有相容呈現。
	}
	r := scene.bodyResources
	if r == nil {
		return errors.New("native physical body resources not preflighted")
	}
	p := &nativePhysicalBodyPlayback{lastPresent: -1, returnWaitMillis: 6 * indexedmap.PhaseBannerStepMillis}
	base := r.base
	var baseScenery []modernHDBattleSource
	if scene.selection.HasSeparateBackgrounds {
		var err error
		baseScenery, err = modernPhysicalScenery(scene, r.actorSide != 0)
		if err != nil {
			return err
		}
	}
	bindBase := func(sources *modernHDBattleSources) error {
		if err := bindModernPhysicalBaseScenery(sources, baseScenery); err != nil {
			return err
		}
		bindModernPhysicalPanelText(scene, sources)
		return nil
	}
	actorRecord, targetRecord := append([]byte(nil), scene.actorRecord...), append([]byte(nil), scene.targetRecord...)
	appendScroll := func(source []byte, sourceLayers *modernHDBattleSources, background fdother.Frame, backgroundIndex byte, record []byte, index int, idle figani.Frame, idleResource int, right bool) error {
		nextBase, presents, err := g.nativePhysicalScroll(scene, source, background, record, index, idle, right)
		if err != nil {
			return err
		}
		nextScenery, err := modernPhysicalBackdropScenery(scene, background, backgroundIndex, right)
		if err != nil {
			return err
		}
		targetSources := &modernHDBattleSources{Base: nextBase}
		if err := bindModernPhysicalBaseScenery(targetSources, nextScenery); err != nil {
			return err
		}
		bindModernPhysicalPanelText(scene, sourceLayers)
		bindModernPhysicalPanelText(scene, targetSources)
		bound, err := modernHDBattleScrollScenerySources(presents, sourceLayers, targetSources, idle, idleResource, r.layers)
		if err != nil {
			return err
		}
		for i, present := range presents {
			phase := "scroll-out"
			if present.Incoming {
				phase = "scroll-in"
			}
			p.jobs = append(p.jobs, nativePhysicalBodyJob{pixels: present.Pixels, hdSources: bound[i], hdPhase: phase, cue: -1})
		}
		base = nextBase
		baseScenery = nextScenery
		return nil
	}
	appendStage := func(attack, idle *figani.Animation, side byte, strikes []battle.NativePhysicalStrike, currentActor, currentTarget []byte, actorIndex, targetIndex int, sounds map[int][]byte, attackResource, idleResource int, bodyPhase string) error {
		if attack == nil {
			return errors.New("native physical counter was not preflighted")
		}
		plan, err := battlepresent.BuildNativePhysicalBodyPlan(attack, idle, side, int(binary.LittleEndian.Uint16(currentTarget[64:])), strikes)
		if err != nil {
			return err
		}
		planCursor := 0
		for si := range strikes {
			if attack.HeaderByte1 != 0 {
				source := make([]byte, 64000) // 2952A明確work清除，header2=0時仍為零。
				sourceLayers := &modernHDBattleSources{Base: source}
				for fi := 0; fi < int(attack.HeaderByte2); fi++ {
					f := attack.Frames[fi]
					source = append([]byte(nil), base...)
					if err := f.BlitAt(source, 320); err != nil {
						return err
					}
					cue := -1
					if f.RawByte5 != 0 {
						cue = int(f.RawByte5)
					}
					sourceLayers = &modernHDBattleSources{Base: base, Layers: []modernHDBattleSource{{Frame: f, Resource: attackResource, Index: fi, Fill: -1}}}
					if err := bindBase(sourceLayers); err != nil {
						return err
					}
					p.jobs = append(p.jobs, nativePhysicalBodyJob{pixels: source, hdSources: sourceLayers, hdPhase: "departure", waitMillis: float64(f.Delay) * indexedmap.PhaseBannerStepMillis, cue: cue, sounds: sounds})
				}
				if err := appendScroll(source, sourceLayers, r.targetBG, modernPhysicalTargetBGIndex(scene), currentTarget, targetIndex, idle.Frames[0], idleResource, side == 0); err != nil {
					return err
				}
			}
			for planCursor < len(plan) && plan[planCursor].Strike == si {
				f := plan[planCursor]
				planCursor++
				if f.UpdatePanel {
					binary.LittleEndian.PutUint16(currentTarget[64:], uint16(f.HP))
					base = append([]byte(nil), base...)
					if err := g.renderPhysicalBattlePanel(scene, currentTarget, base, targetIndex); err != nil {
						return err
					}
				}
				if len(f.Presents) == 0 && f.Cue >= 0 {
					p.jobs = append(p.jobs, nativePhysicalBodyJob{cue: f.Cue, sounds: sounds})
				}
				for inner, present := range f.Presents {
					present := present
					job := nativePhysicalBodyJob{base: base, attack: attack.Frames[f.Frame], idle: idle.Frames[present.IdleFrame], rawSide: side, present: &present, cue: -1, sounds: sounds, hdPhase: bodyPhase}
					job.hdSources = modernHDBattleBodySources(base, job.attack, job.idle, side, present, attackResource, f.Frame, idleResource, present.IdleFrame)
					if err := bindBase(job.hdSources); err != nil {
						return err
					}
					if inner == 0 {
						job.cue = f.Cue
					}
					if !present.StatusPulse && !present.CritPulse {
						job.waitMillis = indexedmap.PhaseBannerStepMillis
						p.jobs = append(p.jobs, job)
						continue
					}
					p.jobs = append(p.jobs, job)
					pulse := job
					pulse.cue = -1
					black := color.NRGBA{A: 255}
					if present.StatusPulse {
						green := color.NRGBA{R: 4, G: 130, A: 255} // DAC(1,32,0)
						pulse.palette0, pulse.waitMillis = &green, 20
						p.jobs = append(p.jobs, pulse)
						pulse.palette0, pulse.waitMillis = &black, 0
						p.jobs = append(p.jobs, pulse)
					}
					if present.CritPulse {
						white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
						pulse.palette0, pulse.waitMillis = &white, 20
						p.jobs = append(p.jobs, pulse)
					}
					pulse.palette0, pulse.waitMillis = &black, indexedmap.PhaseBannerStepMillis
					if present.CritPulse {
						pulse.waitMillis += 40
					}
					p.jobs = append(p.jobs, pulse)
				}
			}
			if si+1 < len(strikes) && attack.HeaderByte1 == 1 {
				source := append([]byte(nil), base...)
				if err := idle.Frames[0].BlitAt(source, 320); err != nil {
					return err
				}
				sourceLayers := &modernHDBattleSources{Base: base, Layers: []modernHDBattleSource{{Frame: idle.Frames[0], Resource: idleResource, Index: 0, Fill: -1}}}
				if err := bindBase(sourceLayers); err != nil {
					return err
				}
				if err := appendScroll(source, sourceLayers, r.actorBG, scene.selection.BaseBG, currentActor, actorIndex, attack.Frames[0], attackResource, side != 0); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := appendStage(r.attack, r.targetIdle, r.actorSide, result.AttackStrikes, actorRecord, targetRecord, scene.actorIndex, scene.targetIndex, r.mainSounds, scene.actorBattleFig*3+1, scene.targetBattleFig*3, "body"); err != nil {
		return err
	}
	if result.Counter != nil {
		if err := appendStage(r.counter, r.actorIdle, r.targetSide, result.CounterStrikes, targetRecord, actorRecord, scene.targetIndex, scene.actorIndex, r.counterSounds, scene.targetBattleFig*3+1, scene.actorBattleFig*3, "counter"); err != nil {
			return err
		}
	}
	if r.attack.HeaderByte1 == 0 {
		pixels := append([]byte(nil), base...)
		platform := r.platform
		platform.X, platform.Y = 164, 157
		if err := platform.Blit(pixels, 320, -1); err != nil {
			return err
		}
		hdRestore := &modernHDBattleSources{Base: append([]byte(nil), pixels...), Layers: []modernHDBattleSource{
			{Frame: r.targetIdle.Frames[0], Resource: scene.targetBattleFig * 3, Fill: -1},
			{Frame: r.actorIdle.Frames[0], Resource: scene.actorBattleFig * 3, Fill: -1},
		}}
		if err := r.targetIdle.Frames[0].BlitAt(pixels, 320); err != nil {
			return err
		}
		if err := r.actorIdle.Frames[0].BlitAt(pixels, 320); err != nil {
			return err
		}
		p.jobs = append(p.jobs, nativePhysicalBodyJob{pixels: pixels, cue: -1, hdSources: hdRestore, hdPhase: "restore"}) // 28F4D..28FD6的final restore。
	}
	bound := make(map[*modernHDBattleSources]bool)
	for i := range p.jobs {
		if sources := p.jobs[i].hdSources; sources != nil && !bound[sources] {
			if err := bindModernPhysicalScenery(scene, sources); err != nil {
				return err
			}
			bound[sources] = true
		}
		if len(p.jobs[i].pixels) != 0 || p.jobs[i].present != nil {
			p.lastPresent = i
		}
	}
	scene.body = p
	return nil
}

func (job *nativePhysicalBodyJob) indexed() ([]byte, error) {
	if job.present == nil {
		return job.pixels, nil
	}
	return battlepresent.ComposeNativePhysicalBodyPresent(job.base, job.attack, job.idle, job.rawSide, *job.present)
}

func (g *Game) drawNativePhysicalBody(screen *ebiten.Image, scene *nativePhysicalScene) error {
	p := scene.body
	if p == nil {
		return errors.New("native physical body has no current present")
	}
	index := p.index
	if index == len(p.jobs) && p.returnWaitMillis > 0 {
		index = p.lastPresent // caller等待保留最後影格，不重播cue或新增present。
	}
	if index < 0 || index >= len(p.jobs) {
		return errors.New("native physical body has no current present")
	}
	job := &p.jobs[index]
	if drawn, err := g.drawModernHDBattleBody(screen, scene, job); err != nil {
		return err
	} else if drawn {
		p.drawn = true
		return nil
	}
	if p.image == nil {
		pixels, err := job.indexed()
		if err != nil {
			return err
		}
		if len(pixels) == 0 {
			return nil // cue-only項目由Update消費。
		}
		palette := append(color.Palette(nil), g.nativeUIPalette...)
		if job.palette0 != nil {
			palette[0] = *job.palette0
		}
		img := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
		copy(img.Pix, pixels)
		p.image = ebiten.NewImageFromImage(img)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(p.image, op)
	p.drawn = true
	return nil
}

// stepNativePhysicalBodyMillis 以60Hz來源播放20/40ms與BIOS等待。
// 每個實際present必須由Draw確認；不累積缺畫面的時間、不跨格追趕。
func (g *Game) stepNativePhysicalBodyMillis(scene *nativePhysicalScene, elapsed float64) error {
	p := scene.body
	if p.returnRampStarted {
		return nil // 既有全DAC owner等待最後Draw後，才交接map與after。
	}
	if p.index == len(p.jobs) {
		p.elapsedMillis += elapsed
	}
	for p.index < len(p.jobs) {
		job := &p.jobs[p.index]
		if !p.cued {
			if job.cue >= 0 {
				g.playNativePhysicalSound(job.sounds[job.cue])
			}
			p.cued = true
		}
		if len(job.pixels) != 0 || job.present != nil {
			if !p.drawn {
				return nil
			}
			p.elapsedMillis += elapsed
			if p.elapsedMillis < job.waitMillis {
				return nil
			}
		}
		if p.image != nil && p.index < p.lastPresent && (len(job.pixels) != 0 || job.present != nil) {
			p.image.Dispose()
			p.image = nil
		}
		p.index++
		remaining := p.elapsedMillis - job.waitMillis
		if remaining < 0 || job.waitMillis == 0 {
			remaining = 0
		}
		p.drawn, p.cued, p.elapsedMillis = false, false, remaining
		if p.index < len(p.jobs) && (len(job.pixels) != 0 || job.present != nil) {
			next := &p.jobs[p.index]
			if len(next.pixels) != 0 || next.present != nil {
				return nil
			}
		}
	}
	if p.elapsedMillis < p.returnWaitMillis {
		return nil
	}
	if g.nativeMapAssets != nil {
		// 290A7→1F882在最後body畫布上依序寫64份DAC；不能用舊map VGA。
		// READY：physical_owner_return_dac_spec。沿既有Draw確認與delay2近似。
		if p.lastPresent < 0 || p.lastPresent >= len(p.jobs) {
			return errors.New("native physical return present unavailable")
		}
		pixels, err := p.jobs[p.lastPresent].indexed()
		if err != nil {
			return err
		}
		if err := g.startNativePaletteRampOnFrame(pixels, 0, 63, 2, g.finishAttackPresentation); err != nil {
			return fmt.Errorf("native physical return DAC: %w", err)
		}
		g.nativePaletteRamp.hdBattleSources = p.jobs[p.lastPresent].hdSources
		p.returnRampStarted = true
		return nil
	}
	g.finishAttackPresentation()
	if g.nativeFieldEvent61 != nil || g.battleEvent != nil {
		return errAttackPresentationYield
	}
	return nil
}
