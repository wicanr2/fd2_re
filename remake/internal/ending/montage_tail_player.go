package ending

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// MontageTailPhase describes the deliberately approximate visual bridge for
// the proven 20-entry 0x2c194 schedule. It never claims to be the native
// 0x28a6c renderer: the exact call-time records and status-panel inputs remain
// outside this player, while the original archive selectors, frame order and
// caller-owned waits remain explicit.
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

// MontageTailPlayer turns the source-proven resource schedule into a bounded,
// deterministic indexed presentation for explicit approximate mode. It owns
// no campaign, party, battle or save state.
type MontageTailPlayer struct {
	Tail       MontageTail
	Assets     MontageTailAssets
	VisualSets []MontageTailVisualSet
	Compositor *IndexedCompositor
	Phase      MontageTailPhase
	Segment    int
	Frame      int
	DelayTicks int
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
	}
	return &MontageTailPlayer{
		Tail: tail, Assets: assets, VisualSets: sets, Compositor: compositor,
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

// Step presents one approximate visual iteration and exposes the following
// native-tick delay to the caller. FIGANI descriptor +6 is retained verbatim
// as the animation delay, including the source-valid zero-delay terminal frame
// in FIGANI#337; the 20/78 waits around FDOTHER#58 come from the recovered
// 0x2c194 caller contract.
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
		p.Frame = 0
		return nil
	case MontageTailPhaseRecord0Aux:
		set, err := p.currentSet()
		if err != nil {
			return err
		}
		animation := set.Record0FIGANIAux
		if p.Frame >= len(animation.Frames) {
			p.Phase = MontageTailPhaseRecord1Aux
			p.Frame = 0
			return nil
		}
		if err := p.renderApproximatePair(*set, animation.Frames[p.Frame], true); err != nil {
			return err
		}
		p.DelayTicks = animation.Frames[p.Frame].Delay
		p.Frame++
		return nil
	case MontageTailPhaseRecord1Aux:
		set, err := p.currentSet()
		if err != nil {
			return err
		}
		animation := set.Record1FIGANIAux
		if p.Frame >= len(animation.Frames) {
			p.Phase = MontageTailPhaseBaseHold
			p.Frame = 0
			return nil
		}
		if err := p.renderApproximatePair(*set, animation.Frames[p.Frame], false); err != nil {
			return err
		}
		p.DelayTicks = animation.Frames[p.Frame].Delay
		p.Frame++
		return nil
	case MontageTailPhaseBaseHold:
		set, err := p.currentSet()
		if err != nil {
			return err
		}
		if err := p.renderApproximatePair(*set, figani.Frame{}, false); err != nil {
			return err
		}
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
		p.Frame = 0
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

func (p *MontageTailPlayer) currentSet() (*MontageTailVisualSet, error) {
	if p.Segment < 0 || p.Segment >= len(p.VisualSets) {
		return nil, errors.New("ending: montage tail visual index is out of range")
	}
	return &p.VisualSets[p.Segment], nil
}

// renderApproximatePair keeps only the source-proven geometry needed for a
// useful remake presentation: BG at y=50, TAI at (164,157), both base poses,
// and the current auxiliary FIGANI frame. Native status panels, slide helpers,
// effects and sounds remain unclaimed until their full inputs are available.
func (p *MontageTailPlayer) renderApproximatePair(set MontageTailVisualSet, auxiliary figani.Frame, record0 bool) error {
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
	if auxiliary.Width == 0 && auxiliary.Height == 0 {
		if err := set.Record1FIGANIBase.Frames[0].BlitAt(p.Compositor.VGA, Width); err != nil {
			return fmt.Errorf("ending: montage tail record1 base FIGANI#%d: %w", set.Plan.Record1FIGANIBase, err)
		}
		if err := set.Record0FIGANIBase.Frames[0].BlitAt(p.Compositor.VGA, Width); err != nil {
			return fmt.Errorf("ending: montage tail record0 base FIGANI#%d: %w", set.Plan.Record0FIGANIBase, err)
		}
		return nil
	}
	// The two proven 0x2939d calls pair record0's auxiliary animation with
	// record1's base, then record1's auxiliary animation with record0's base.
	// Do not draw the animated record's own base underneath it: that creates a
	// remake-only double figure which the call shape does not support.
	base := set.Record0FIGANIBase
	baseResource := set.Plan.Record0FIGANIBase
	if record0 {
		base = set.Record1FIGANIBase
		baseResource = set.Plan.Record1FIGANIBase
	}
	if err := base.Frames[0].BlitAt(p.Compositor.VGA, Width); err != nil {
		return fmt.Errorf("ending: montage tail paired base FIGANI#%d: %w", baseResource, err)
	}
	if err := auxiliary.BlitAt(p.Compositor.VGA, Width); err != nil {
		owner := "record1"
		if record0 {
			owner = "record0"
		}
		return fmt.Errorf("ending: montage tail %s auxiliary FIGANI: %w", owner, err)
	}
	return nil
}
