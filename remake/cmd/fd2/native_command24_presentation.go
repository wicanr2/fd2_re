package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

type nativeCommand24PresentationJob struct {
	actor, target        *battle.Unit
	plan                 *battle.NativeCommandDerivedStrikePlan
	schedule             figani.NativeCommand24Schedule
	effectFrames         []*ebiten.Image
	effectPositions      [][2]int
	targetFrames         []*ebiten.Image
	targetPositions      [][2]int
	targetDelays         []int
	frame, repeat        int
	targetFrame          int
	targetRepeat         int
	shakeCounter         int
	drawn                bool
	mpPublished          bool
	damagePublished      bool
	actorMPBefore        int
	actorActedBefore     bool
	targetHPBefore       int
	targetRawByte5Before byte
	then                 func([]battle.NativeCommand24Damage)
}

func nativeFIGANIPath() string {
	for _, key := range []string{"FD2_ORIGINAL_FIGANI", "FD2_FIGANI"} {
		if path := os.Getenv(key); path != "" {
			return path
		}
	}
	path := assetPath("assets/original/FIGANI.DAT")
	if fileExists(path) {
		return path
	}
	return ""
}

func nativeFIGANIImages(animation *figani.Animation, palette color.Palette) ([]*ebiten.Image, [][2]int, error) {
	if animation == nil || len(animation.Frames) == 0 || len(palette) < 256 {
		return nil, nil, errors.New("native FIGANI image inputs unavailable")
	}
	images := make([]*ebiten.Image, len(animation.Frames))
	positions := make([][2]int, len(animation.Frames))
	for i, frame := range animation.Frames {
		if frame.Width <= 0 || frame.Height <= 0 || len(frame.Pixels) != frame.Width*frame.Height || len(frame.Mask) != len(frame.Pixels) {
			return nil, nil, fmt.Errorf("native FIGANI frame %d is malformed", i)
		}
		img := image.NewRGBA(image.Rect(0, 0, frame.Width, frame.Height))
		for at, index := range frame.Pixels {
			if frame.Mask[at] == 0 {
				continue
			}
			r, g, b, _ := palette[index].RGBA()
			img.SetRGBA(at%frame.Width, at/frame.Width, color.RGBA{R: byte(r >> 8), G: byte(g >> 8), B: byte(b >> 8), A: 0xff})
		}
		images[i] = ebiten.NewImageFromImage(img)
		positions[i] = [2]int{frame.X, frame.Y}
	}
	return images, positions, nil
}

func (g *Game) startNativeCommand24Presentation(actor, target *battle.Unit, then func([]battle.NativeCommand24Damage)) error {
	if g == nil || g.st == nil || g.rng == nil || actor == nil || target == nil ||
		g.nativeCmd24Presentation != nil || g.nativeHealPresentation != nil ||
		g.nativeModifierPresentation != nil || g.atk != nil {
		return errors.New("native command24 presentation context unavailable")
	}
	if !actor.HasBattleFig || actor.BattleFig != 32 || !target.HasBattleFig {
		return errors.New("native command24 requires proven selector32 actor and target BattleFig")
	}
	if g.bg == nil || g.tai == nil || g.panel == nil || len(g.nativeUIPalette) < 256 {
		return errors.New("native command24 battle background/panel/palette unavailable")
	}
	archive := nativeFIGANIPath()
	if archive == "" {
		return errors.New("native command24 player-provided FIGANI.DAT unavailable")
	}
	effect, err := figani.DecodeResource(archive, actor.BattleFig*3+2)
	if err != nil {
		return err
	}
	schedule, err := figani.BuildNativeCommand24Schedule(effect)
	if err != nil {
		return err
	}
	targetIdle, err := figani.DecodeResource(archive, target.BattleFig*3)
	if err != nil {
		return err
	}
	if len(targetIdle.Frames) == 0 {
		return errors.New("native command24 target idle FIGANI is empty")
	}
	effectImages, effectPositions, err := nativeFIGANIImages(effect, g.nativeUIPalette)
	if err != nil {
		return err
	}
	targetImages, targetPositions, err := nativeFIGANIImages(targetIdle, g.nativeUIPalette)
	if err != nil {
		return err
	}
	targetDelays := make([]int, len(targetIdle.Frames))
	for i, frame := range targetIdle.Frames {
		if frame.Delay <= 0 {
			return fmt.Errorf("native command24 target idle frame %d has invalid delay", i)
		}
		targetDelays[i] = frame.Delay
	}
	if !osMuteOrShot(g) && (len(g.sfxCommand24Actor) == 0 || len(g.sfxCommand24Target) == 0) {
		return errors.New("native command24 FDOTHER #53 samples3/2 unavailable")
	}
	plan, err := g.st.PlanNativeCommandDerivedStrike(actor, target, 24, g.rng)
	if err != nil {
		return err
	}
	if len(plan.Results) != 1 || plan.Results[0].Target != target {
		return errors.New("native command24 normal player path is not a single target")
	}
	g.nativeCmd24Presentation = &nativeCommand24PresentationJob{
		actor: actor, target: target, plan: plan, schedule: schedule,
		effectFrames: effectImages, effectPositions: effectPositions,
		targetFrames: targetImages, targetPositions: targetPositions, targetDelays: targetDelays,
		actorMPBefore: actor.MP, actorActedBefore: actor.Acted, targetHPBefore: target.HP,
		targetRawByte5Before: target.NativeRecordByte5,
		then:                 then,
	}
	return nil
}

func (g *Game) failNativeCommand24Presentation(err error) {
	j := g.nativeCmd24Presentation
	if j == nil {
		return
	}
	j.actor.MP, j.actor.Acted, j.target.HP = j.actorMPBefore, j.actorActedBefore, j.targetHPBefore
	if j.target.HasNativeRecordByte5 {
		j.target.NativeRecordByte5 = j.targetRawByte5Before
	}
	g.nativeCmd24Presentation = nil
	g.loadErr = "native command24 presentation: " + err.Error()
}

func (g *Game) stepNativeCommand24Presentation() {
	j := g.nativeCmd24Presentation
	if j == nil || !j.drawn {
		return
	}
	j.drawn = false
	frame := j.frame
	if j.damagePublished && j.shakeCounter >= 0 {
		j.shakeCounter--
	}
	if frame == j.schedule.ActorImpactFrame && !j.mpPublished {
		if err := battle.ApplyNativeCommandDerivedStrikeMP(j.plan); err != nil {
			g.failNativeCommand24Presentation(err)
			return
		}
		j.mpPublished = true
		g.playRaw(g.sfxCommand24Actor)
	}
	if frame == j.schedule.TargetImpactFrame && !j.damagePublished {
		if !j.mpPublished {
			g.failNativeCommand24Presentation(errors.New("target marker preceded actor MP marker"))
			return
		}
		if err := battle.ApplyNativeCommandDerivedStrikeTarget(j.plan, 0); err != nil {
			g.failNativeCommand24Presentation(err)
			return
		}
		j.damagePublished = true
		j.shakeCounter = 5
		g.playRaw(g.sfxCommand24Target)
	}
	j.targetRepeat++
	if j.targetRepeat >= j.targetDelays[j.targetFrame] {
		j.targetRepeat = 0
		j.targetFrame = (j.targetFrame + 1) % len(j.targetFrames)
	}
	j.repeat++
	if j.repeat < nativeCommand24RawFramesDelay(j) {
		return
	}
	j.repeat = 0
	j.frame++
	if j.frame < len(j.effectFrames) {
		return
	}
	if !j.mpPublished || !j.damagePublished {
		g.failNativeCommand24Presentation(errors.New("required raw marker was not presented"))
		return
	}
	if err := battle.CompleteNativeCommandDerivedStrike(j.plan); err != nil {
		g.failNativeCommand24Presentation(err)
		return
	}
	then, results := j.then, append([]battle.NativeCommand24Damage(nil), j.plan.Results...)
	g.nativeCmd24Presentation = nil
	if then != nil {
		then(results)
	}
}

func nativeCommand24RawFramesDelay(j *nativeCommand24PresentationJob) int {
	if j == nil || j.frame < 0 || j.frame >= len(nativeCommand24RawDelays) {
		return 0
	}
	// BuildNativeCommand24Schedule already fixed the complete resource98 raw
	// signature; the explicit table keeps Draw timing independent of PNG count.
	return int(nativeCommand24RawDelays[j.frame])
}

var nativeCommand24RawDelays = [15]byte{2, 2, 2, 8, 2, 2, 2, 2, 8, 2, 2, 6, 2, 2, 4}

func (g *Game) drawNativeCommand24Presentation(screen *ebiten.Image) bool {
	j := g.nativeCmd24Presentation
	if j == nil || j.frame < 0 || j.frame >= len(j.effectFrames) ||
		j.targetFrame < 0 || j.targetFrame >= len(j.targetFrames) {
		return false
	}
	screen.Fill(color.Black)
	if g.bg != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(0, 100)
		screen.DrawImage(g.bg, op)
	}
	if g.font != nil {
		displayMP := j.actor.MP
		if j.frame == j.schedule.ActorImpactFrame && !j.mpPublished {
			displayMP = j.plan.MPAfter
		}
		g.drawBattlePanel(screen, 342, 8, j.actor.Name, j.actor.Lv, j.actor.HP, j.actor.MaxHP, displayMP)
		displayHP := j.target.HP
		if (j.frame == j.schedule.TargetImpactFrame || j.damagePublished) && len(j.plan.Results) == 1 {
			displayHP = j.plan.Results[0].HPAfter
		}
		g.drawBattlePanel(screen, 0, 308, j.target.Name, j.target.Lv, displayHP, j.target.MaxHP, j.target.MP)
	}
	if g.tai != nil {
		tb := g.tai.Bounds()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(482-float64(tb.Dx()), 356-float64(tb.Dy()))
		screen.DrawImage(g.tai, op)
	}
	shake := 0
	if j.frame == j.schedule.TargetImpactFrame && !j.damagePublished {
		shake = j.schedule.ShakeOffsets[5]
	} else if j.damagePublished && j.shakeCounter >= 0 && j.shakeCounter < len(j.schedule.ShakeOffsets) {
		shake = j.schedule.ShakeOffsets[j.shakeCounter]
	}
	targetPos := j.targetPositions[j.targetFrame]
	targetOp := &ebiten.DrawImageOptions{}
	targetOp.GeoM.Scale(2, 2)
	targetOp.GeoM.Translate(float64((targetPos[0]-shake)*2), float64(targetPos[1]*2))
	screen.DrawImage(j.targetFrames[j.targetFrame], targetOp)
	effectPos := j.effectPositions[j.frame]
	effectOp := &ebiten.DrawImageOptions{}
	effectOp.GeoM.Scale(2, 2)
	effectOp.GeoM.Translate(float64(effectPos[0]*2), float64(effectPos[1]*2))
	screen.DrawImage(j.effectFrames[j.frame], effectOp)
	j.drawn = true
	return true
}
