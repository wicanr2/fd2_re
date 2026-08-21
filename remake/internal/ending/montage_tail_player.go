package ending

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// MontageTailPhase describes the source-bound visual bridge for the proven
// 20-entry 0x2c194 schedule. It implements the reachable header-byte1-zero
// 0x2939d pair loop but does not claim exact call-time record continuity,
// sound ownership, DOS timing or player-path E2.
type MontageTailPhase string

const (
	MontageTailPhaseIntro           MontageTailPhase = "intro"
	MontageTailPhaseRecord0Aux      MontageTailPhase = "record0_aux"
	MontageTailPhaseRecord1Aux      MontageTailPhase = "record1_aux"
	MontageTailPhaseBaseHold        MontageTailPhase = "base_hold"
	MontageTailPhaseOverlay         MontageTailPhase = "fdother_58_overlay"
	MontageTailPhaseFinal           MontageTailPhase = "final"
	MontageTailPhaseCompleted       MontageTailPhase = "completed"
	nativeMontageTailIntroWaitTicks                  = 0x50
)

var (
	nativeMontageTailHorizontalOffsets = [...]int{0, 4, 9, 14, 18, 14}
	nativeMontageTailVerticalOffsets   = [...]int{0, 2, 4, 6, 8, 10}
)

// MontageTailPlayer turns the source-proven resource schedule into a bounded,
// deterministic indexed presentation. Campaign admission is owned by the
// source-bound E1 caller; this player owns no campaign, party, battle or save
// state and does not claim exact native 0x28a6c rendering.
type MontageTailPlayer struct {
	Tail       MontageTail
	Assets     MontageTailAssets
	VisualSets []MontageTailVisualSet
	Entries    []MontageTailEntry
	Compositor *IndexedCompositor
	Phase      MontageTailPhase
	Segment    int
	Frame      int
	Inner      int
	BaseFrame  int
	BaseInner  int
	EffectStep int
	// SoundMarker preserves raw frame +5 for tracing. The player deliberately
	// does not emit audio while the two native sound-owner globals are unknown.
	SoundMarker byte
	DelayTicks  int
}

// NewMontageTailPlayer validates every source asset before exposing any
// playable state. This all-or-nothing admission prevents a missing late entry
// from producing a partially plausible ending.
func NewMontageTailPlayer(tail MontageTail, assets MontageTailAssets, sets []MontageTailVisualSet, compositor *IndexedCompositor) (*MontageTailPlayer, error) {
	entries, err := tail.Plan()
	if err != nil {
		return nil, err
	}
	if compositor == nil || len(assets.LoopPalette) != len(compositor.Palette) || len(assets.LoopFrames) != len(entries) || len(sets) != len(entries) {
		return nil, errors.New("ending: montage tail player source provenance is incomplete")
	}
	if assets.Intro.Width != Width || assets.Intro.Height != Height || assets.Final.Width != Width || assets.Final.Height != Height {
		return nil, errors.New("ending: montage tail player full-screen assets are incomplete")
	}
	for index, set := range sets {
		if set.Plan.Index != index || set.Record0FIGANIBase == nil || set.Record0FIGANIAux == nil ||
			set.Record1FIGANIBase == nil || set.Record1FIGANIAux == nil ||
			len(set.Record0FIGANIBase.Frames) == 0 || len(set.Record0FIGANIAux.Frames) == 0 ||
			len(set.Record1FIGANIBase.Frames) == 0 || len(set.Record1FIGANIAux.Frames) == 0 {
			return nil, fmt.Errorf("ending: montage tail visual set %d is incomplete", index)
		}
		if err := validateMontageTailPlayerAnimation(set.Record0FIGANIBase); err != nil {
			return nil, fmt.Errorf("ending: montage tail entry %d record0 base: %w", index, err)
		}
		if err := validateMontageTailPlayerAnimation(set.Record0FIGANIAux); err != nil {
			return nil, fmt.Errorf("ending: montage tail entry %d record0 aux: %w", index, err)
		}
		if err := validateMontageTailPlayerAnimation(set.Record1FIGANIBase); err != nil {
			return nil, fmt.Errorf("ending: montage tail entry %d record1 base: %w", index, err)
		}
		if err := validateMontageTailPlayerAnimation(set.Record1FIGANIAux); err != nil {
			return nil, fmt.Errorf("ending: montage tail entry %d record1 aux: %w", index, err)
		}
		for name, animation := range map[string]*figani.Animation{
			"record0 base": set.Record0FIGANIBase, "record0 aux": set.Record0FIGANIAux,
			"record1 base": set.Record1FIGANIBase, "record1 aux": set.Record1FIGANIAux,
		} {
			if animation.HeaderByte1 != 0 || int(animation.HeaderByte2) != len(animation.Frames) {
				return nil, fmt.Errorf("ending: montage tail entry %d %s has unsupported native prelude header %d/%d", index, name, animation.HeaderByte1, animation.HeaderByte2)
			}
		}
		for name, animation := range map[string]*figani.Animation{
			"record0 base": set.Record0FIGANIBase, "record1 base": set.Record1FIGANIBase,
		} {
			for frame, descriptor := range animation.Frames {
				if descriptor.Delay <= 0 {
					return nil, fmt.Errorf("ending: montage tail entry %d %s frame %d has invalid base delay %d", index, name, frame, descriptor.Delay)
				}
			}
		}
	}
	return &MontageTailPlayer{
		Tail: tail, Assets: assets, VisualSets: sets, Entries: entries, Compositor: compositor,
		Phase: MontageTailPhaseIntro,
	}, nil
}

func validateMontageTailPlayerAnimation(animation *figani.Animation) error {
	if animation == nil || len(animation.Frames) == 0 {
		return errors.New("FIGANI animation is unavailable")
	}
	for index, frame := range animation.Frames {
		if frame.X < 0 || frame.Y < 0 || frame.X+frame.Width > Width || frame.Y+frame.Height > Height {
			return fmt.Errorf("FIGANI frame %d is outside the indexed viewport", index)
		}
	}
	return nil
}

// Ready reports that all 20 source transactions completed and the stable
// FDOTHER#59 frame was presented. Further Step calls are rejected so the
// caller can hold that image indefinitely.
func (p *MontageTailPlayer) Ready() bool {
	return p != nil && p.Phase == MontageTailPhaseCompleted
}

// Step presents one source-bound native inner iteration and exposes the next
// raw-tick delay. Each auxiliary descriptor +6 owns that many presentations;
// the paired base scheduler advances after every one. Zero-delay descriptors
// publish no frame. The 20/78 waits around FDOTHER#58 remain caller-owned.
func (p *MontageTailPlayer) Step() error {
	if p == nil || p.Compositor == nil || p.Ready() {
		return errors.New("ending: montage tail player is not running")
	}
	p.DelayTicks = 0
	switch p.Phase {
	case MontageTailPhaseIntro:
		copy(p.Compositor.Palette[:], p.Assets.LoopPalette)
		copy(p.Compositor.Baseline[:], p.Assets.LoopPalette)
		p.Compositor.baselineKnown = true
		clear(p.Compositor.VGA)
		if err := p.Assets.Intro.Blit(p.Compositor.VGA, Width, -1); err != nil {
			return fmt.Errorf("ending: montage tail intro: %w", err)
		}
		// 0x2c1fd calls 0x17aa9(0x50) after FDOTHER#60 is decoded and
		// track 18 is selected, before the following presentation helper.
		p.DelayTicks = nativeMontageTailIntroWaitTicks
		p.Phase = MontageTailPhaseRecord0Aux
		p.resetPairState()
		return nil
	case MontageTailPhaseRecord0Aux:
		set, err := p.currentSet()
		if err != nil {
			return err
		}
		_, done, err := p.stepNativePair(*set, set.Record0FIGANIAux, set.Record1FIGANIBase, true, false)
		if err != nil {
			return err
		}
		if done {
			p.Phase = MontageTailPhaseRecord1Aux
			p.resetPairState()
		}
		return nil
	case MontageTailPhaseRecord1Aux:
		set, err := p.currentSet()
		if err != nil {
			return err
		}
		_, done, err := p.stepNativePair(*set, set.Record1FIGANIAux, set.Record0FIGANIBase, false, true)
		if err != nil {
			return err
		}
		if done {
			p.Phase = MontageTailPhaseBaseHold
			p.resetPairState()
		}
		return nil
	case MontageTailPhaseBaseHold:
		// 0x2c2a6 waits on the last frame already published by the second
		// 0x2939d call; it does not synthesize an extra two-base pose.
		p.DelayTicks = p.Tail.Loop.WaitBeforeFrameTicks
		p.Phase = MontageTailPhaseOverlay
		return nil
	case MontageTailPhaseOverlay:
		if p.Segment < 0 || p.Segment >= len(p.Assets.LoopFrames) {
			return errors.New("ending: montage tail overlay index is out of range")
		}
		if err := p.Assets.LoopFrames[p.Segment].Blit(p.Compositor.VGA, Width, -1); err != nil {
			return fmt.Errorf("ending: montage tail FDOTHER#58 frame %d: %w", p.Segment, err)
		}
		p.DelayTicks = p.Tail.Loop.WaitAfterFrameTicks
		p.Segment++
		p.resetPairState()
		if p.Segment >= len(p.VisualSets) {
			p.Phase = MontageTailPhaseFinal
		} else {
			p.Phase = MontageTailPhaseRecord0Aux
		}
		return nil
	case MontageTailPhaseFinal:
		if err := p.Assets.PresentFinal(p.Compositor); err != nil {
			return err
		}
		p.Phase = MontageTailPhaseCompleted
		return nil
	default:
		return fmt.Errorf("ending: unknown montage tail phase %q", p.Phase)
	}
}

func (p *MontageTailPlayer) resetPairState() {
	p.Frame = 0
	p.Inner = 0
	p.BaseFrame = 0
	p.BaseInner = 0
	p.EffectStep = 0
	p.SoundMarker = 0
}

func (p *MontageTailPlayer) currentSet() (*MontageTailVisualSet, error) {
	if p.Segment < 0 || p.Segment >= len(p.VisualSets) {
		return nil, errors.New("ending: montage tail visual index is out of range")
	}
	return &p.VisualSets[p.Segment], nil
}

func (p *MontageTailPlayer) stepNativePair(set MontageTailVisualSet, auxiliary, base *figani.Animation, record0, terminal bool) (presented, done bool, err error) {
	lastEffect := -1
	if terminal {
		for index, frame := range auxiliary.Frames {
			if frame.RawByte4 != 0 {
				lastEffect = index
			}
		}
	}
	for skipped := 0; skipped <= len(auxiliary.Frames); skipped++ {
		if p.Frame >= len(auxiliary.Frames) {
			return false, true, nil
		}
		frameIndex := p.Frame
		frame := auxiliary.Frames[frameIndex]
		if p.Inner == 0 && frame.RawByte4 != 0 {
			p.EffectStep = len(nativeMontageTailHorizontalOffsets) - 1
		}
		if frame.Delay <= 0 {
			p.Frame++
			p.Inner = 0
			if terminal && lastEffect >= 0 && frameIndex == lastEffect {
				return false, true, nil
			}
			continue
		}
		baseFrame := base.Frames[p.BaseFrame]
		var opaqueFill *byte
		fill := byte(33)
		if frame.RawByte4 != 0 && p.Inner == 0 {
			opaqueFill = &fill
		}
		if err := p.renderSourceBoundPair(set, frame, baseFrame, record0, p.EffectStep, opaqueFill); err != nil {
			return false, false, err
		}
		p.SoundMarker = frame.RawByte5
		p.DelayTicks = 1
		p.advanceBaseScheduler(base)
		p.Inner++
		if p.EffectStep > 0 {
			p.EffectStep--
		}
		if p.Inner >= frame.Delay {
			p.Frame++
			p.Inner = 0
			if terminal && lastEffect >= 0 && frameIndex == lastEffect {
				return true, true, nil
			}
		}
		return true, false, nil
	}
	return false, false, errors.New("ending: montage tail zero-delay frame scan exceeded animation")
}

func (p *MontageTailPlayer) advanceBaseScheduler(base *figani.Animation) {
	p.BaseInner++
	if p.BaseInner >= base.Frames[p.BaseFrame].Delay {
		p.BaseInner = 0
		p.BaseFrame++
		if p.BaseFrame >= len(base.Frames) {
			p.BaseFrame = 0
		}
	}
}

// renderSourceBoundPair reproduces the reachable header-byte1-zero branch of
// the two 0x2939d calls. The native 400-stride staging margin is represented
// by clipped translated blits on the final 320-wide indexed surface.
func (p *MontageTailPlayer) renderSourceBoundPair(set MontageTailVisualSet, auxiliary, base figani.Frame, record0 bool, effectStep int, opaqueFill *byte) error {
	clear(p.Compositor.VGA)
	bg := set.BG
	bg.Y = 50
	if err := bg.Blit(p.Compositor.VGA, Width, -1); err != nil {
		return fmt.Errorf("ending: montage tail BG#%d: %w", set.Plan.BG, err)
	}
	tai := set.TAI
	tai.X, tai.Y = 164, 157
	if err := tai.Blit(p.Compositor.VGA, Width, -1); err != nil {
		return fmt.Errorf("ending: montage tail TAI#%d: %w", set.Plan.TAI, err)
	}
	// The two proven 0x2939d calls pair record0's auxiliary animation with
	// record1's base, then record1's auxiliary animation with record0's base.
	// Do not draw the animated record's own base underneath it: that creates a
	// remake-only double figure which the call shape does not support.
	baseResource := set.Plan.Record0FIGANIBase
	side := p.Entries[p.Segment].Record1Byte6
	if record0 {
		baseResource = set.Plan.Record1FIGANIBase
		side = p.Entries[p.Segment].Record0Byte6
	}
	if effectStep < 0 || effectStep >= len(nativeMontageTailHorizontalOffsets) {
		return errors.New("ending: montage tail effect displacement is out of range")
	}
	dx, dy := nativeMontageTailHorizontalOffsets[effectStep], nativeMontageTailVerticalOffsets[effectStep]
	if side != 0 {
		dx, dy = -dx, -dy
	}
	drawAuxiliary := func() error {
		return auxiliary.BlitTranslated(p.Compositor.VGA, Width, 0, 0, nil)
	}
	drawBase := func() error {
		return base.BlitTranslated(p.Compositor.VGA, Width, dx, dy, opaqueFill)
	}
	auxFirst := (side != 0 && auxiliary.RawByte7&1 != 0) || (side == 0 && auxiliary.RawByte7&1 == 0)
	if auxFirst {
		if err := drawAuxiliary(); err != nil {
			return fmt.Errorf("ending: montage tail auxiliary FIGANI: %w", err)
		}
	}
	if err := drawBase(); err != nil {
		return fmt.Errorf("ending: montage tail paired base FIGANI#%d: %w", baseResource, err)
	}
	if !auxFirst {
		if err := drawAuxiliary(); err != nil {
			return fmt.Errorf("ending: montage tail auxiliary FIGANI: %w", err)
		}
	}
	if auxiliary.Width <= 0 || auxiliary.Height <= 0 {
		owner := "record1"
		if record0 {
			owner = "record0"
		}
		return fmt.Errorf("ending: montage tail %s auxiliary FIGANI is empty", owner)
	}
	return nil
}
