package ending

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// MontageCycleAssets contains only the original indexed inputs consumed by
// 0x2c405's 0x2c548 party cycle.  Nothing here is a PNG or a guessed UI
// replacement; missing or malformed source assets reject the cycle.
type MontageCycleAssets struct {
	Backdrop  fdother.Frame
	TAI003    fdother.Frame
	Primary   map[int]*figani.Animation
	Secondary map[int]*figani.Animation
	Grid      []byte
	Portraits map[int][]dato.Frame
	Current   *fdtxt.Strings
	Permanent *fdtxt.Strings
	Font      *fdtxt.Font
}

// MontageArchivePaths identifies the remaining player-provided archives and
// separated asset roots used by the native cycle. Originals remain read-only.
type MontageArchivePaths struct {
	FDOTHER       string
	SurfaceRoot   string
	AnimationRoot string
	PortraitRoot  string
	TextRoot      string
	FontRoot      string
}

// LoadMontageCycleAssets decodes the exact resources named by the native
// call-sites: FDOTHER#56, TAI#3, FDOTHER#5, FIGANI group*3/{+1}, DATO unit
// +7, FDTXT_031/FDTXT_000 and the FDOTHER#4 font.  It deliberately takes the
// raw unit records from the caller so no identity or slot meaning is guessed.
func LoadMontageCycleAssets(montage Montage, paths MontageArchivePaths, units [][]byte) (MontageCycleAssets, error) {
	if paths.FDOTHER == "" || paths.SurfaceRoot == "" || paths.AnimationRoot == "" || paths.PortraitRoot == "" || paths.TextRoot == "" || paths.FontRoot == "" || len(units) < 2 {
		return MontageCycleAssets{}, errors.New("ending: incomplete montage archive paths or party")
	}
	for i, unit := range units {
		if len(unit) < 0x21 {
			return MontageCycleAssets{}, fmt.Errorf("ending: unit %d is shorter than native stride", i)
		}
	}
	backdrop, err := fdother.DecodeArchiveSingleFrame(paths.FDOTHER, 56)
	if err != nil {
		return MontageCycleAssets{}, fmt.Errorf("ending: FDOTHER#56: %w", err)
	}
	tai003, err := fdother.LoadSeparatedSingleFrame(paths.SurfaceRoot, "TAI.DAT", 3)
	if err != nil {
		return MontageCycleAssets{}, fmt.Errorf("ending: TAI#3: %w", err)
	}
	if !isTransparentTAI003(tai003) {
		return MontageCycleAssets{}, errors.New("ending: separated TAI#3 is not the proven transparent resource")
	}
	grid := make([]byte, Bytes)
	if err := RenderDialogueFrameGridResource(montage, paths.FDOTHER, grid); err != nil {
		return MontageCycleAssets{}, fmt.Errorf("ending: FDOTHER#5 dialogue grid: %w", err)
	}
	current, err := fdtxt.LoadSeparatedResource(paths.TextRoot, 31)
	if err != nil {
		return MontageCycleAssets{}, fmt.Errorf("ending: separated FDTXT_031: %w", err)
	}
	permanent, err := fdtxt.LoadSeparatedResource(paths.TextRoot, 0)
	if err != nil {
		return MontageCycleAssets{}, fmt.Errorf("ending: separated FDTXT_000: %w", err)
	}
	font, err := fdtxt.LoadSeparatedFont(paths.FontRoot)
	if err != nil {
		return MontageCycleAssets{}, fmt.Errorf("ending: separated FDOTHER#4 font: %w", err)
	}
	assets := MontageCycleAssets{
		Backdrop: backdrop, TAI003: tai003, Primary: make(map[int]*figani.Animation),
		Secondary: make(map[int]*figani.Animation), Grid: grid, Portraits: make(map[int][]dato.Frame),
		Current: current, Permanent: permanent, Font: font,
	}
	for _, unit := range units {
		group := int(unit[7])
		for _, resource := range []int{group * 3, group*3 + 1} {
			if _, ok := assets.Secondary[resource]; ok {
				continue
			}
			animation, err := figani.LoadSeparatedResource(paths.AnimationRoot, resource)
			if err != nil {
				return MontageCycleAssets{}, fmt.Errorf("ending: FIGANI#%d: %w", resource, err)
			}
			assets.Secondary[resource] = animation
			assets.Primary[resource] = animation
		}
		if _, ok := assets.Portraits[group]; !ok {
			frames, err := dato.LoadSeparatedResource(paths.PortraitRoot, group)
			if err != nil {
				return MontageCycleAssets{}, fmt.Errorf("ending: separated portrait %d: %w", group, err)
			}
			assets.Portraits[group] = frames
		}
	}
	return assets, nil
}

// MontageCyclePhase is the directly recovered first-party schedule.  The
// final palette fade is kept as a distinct phase because its native helper
// has a different delay unit from the BIOS-tick waits in the party loops.
type MontageCyclePhase string

const (
	MontagePhaseFigureFade MontageCyclePhase = "figure_fade"
	MontagePhaseSecondary  MontageCyclePhase = "secondary_intro"
	MontagePhasePrimary    MontageCyclePhase = "primary_animation"
	MontagePhasePortrait   MontageCyclePhase = "portrait_text"
	MontagePhaseFadeOut    MontageCyclePhase = "palette_fade_out"
	MontagePhaseCompleted  MontageCyclePhase = "completed"
)

// MontageCycle is a deterministic indexed executor for the verified
// 0x2c548 schedule.  Step performs one native render iteration; the caller
// owns the returned delay units and the raw random byte used by 0x4e893.
// ObserveInputChange is deliberately a separate raw event hook: 0x2c950 only
// observes a changed input-buffer condition during portrait presentation; it
// does not decode a particular key in this branch.
type MontageCycle struct {
	Montage                 Montage
	Assets                  MontageCycleAssets
	Units                   [][]byte
	Plans                   []PartyCyclePlan
	Compositor              *IndexedCompositor
	PlanIndex               int
	Phase                   MontageCyclePhase
	FadeIndex               int
	Secondary               *figani.Animation
	Primary                 *figani.Animation
	SecondarySM             figani.NativeScheduler
	PrimaryIdx              int
	Portrait                int
	PortraitMax             int
	PortraitSM              MontagePortraitState
	FadeOut                 int
	DelayTicks              int
	DelayMS                 int
	skipToFinal             bool
	portraitBoundaryPending bool
}

// NewMontageCycle validates all source provenance before exposing an
// executable renderer.  In particular, the caller must provide the palette
// baseline captured from a native indexed presentation; an all-zero default
// is not accepted as a substitute.
func NewMontageCycle(m Montage, assets MontageCycleAssets, units [][]byte, groups []byte, compositor *IndexedCompositor) (*MontageCycle, error) {
	if m.Status != "mapped_first_party_cycle_fail_closed" || compositor == nil || !compositor.BaselineKnown() || len(units) < 2 || len(units) != len(groups) || len(assets.Grid) != Bytes || !isTransparentTAI003(assets.TAI003) {
		return nil, errors.New("ending: montage source provenance is incomplete")
	}
	if assets.Backdrop.Width != Width || assets.Backdrop.Height != Height || assets.Current == nil || assets.Permanent == nil || assets.Font == nil {
		return nil, errors.New("ending: montage backdrop or text assets are incomplete")
	}
	plans, err := m.PlanPartyCycle(groups)
	if err != nil {
		return nil, err
	}
	for i, unit := range units {
		if len(unit) < 0x21 {
			return nil, fmt.Errorf("ending: unit %d is shorter than native stride", i)
		}
		group := int(unit[7])
		if _, ok := assets.Portraits[group]; !ok || len(assets.Portraits[group]) < 4 {
			return nil, fmt.Errorf("ending: DATO portrait %d is unavailable", group)
		}
	}
	for _, plan := range plans {
		primary := assets.Primary[plan.PrimaryFIGANI]
		secondary := assets.Secondary[plan.SecondaryFIGANI]
		if primary == nil || secondary == nil || len(primary.Frames) == 0 || len(secondary.Frames) == 0 {
			return nil, fmt.Errorf("ending: FIGANI resources %d/%d are unavailable", plan.PrimaryFIGANI, plan.SecondaryFIGANI)
		}
		for _, frame := range primary.Frames {
			if frame.Delay <= 0 {
				return nil, fmt.Errorf("ending: FIGANI#%d contains zero frame delay", plan.PrimaryFIGANI)
			}
		}
	}
	p := &MontageCycle{
		Montage: m, Assets: assets, Units: units, Plans: plans, Compositor: compositor,
		Phase: MontagePhaseFigureFade,
	}
	if err := p.preparePlan(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *MontageCycle) preparePlan() error {
	plan := p.Plans[p.PlanIndex]
	p.Secondary = p.Assets.Secondary[plan.SecondaryFIGANI]
	p.Primary = p.Assets.Primary[plan.PrimaryFIGANI]
	p.FadeIndex = 0
	p.Portrait = 0
	p.PortraitSM = MontagePortraitState{}
	p.PrimaryIdx = 0
	p.DelayTicks = 0
	p.DelayMS = 0
	p.portraitBoundaryPending = false
	if err := p.renderBackdrop(); err != nil {
		return err
	}
	return nil
}

func (p *MontageCycle) renderBackdrop() error {
	clear(p.Compositor.Offscreen)
	if err := p.Assets.Backdrop.Blit(p.Compositor.Offscreen, Width, -1); err != nil {
		return fmt.Errorf("ending: FDOTHER#56 backdrop: %w", err)
	}
	return nil
}

// Step advances exactly one recovered rendering iteration.  randomByte is
// supplied by the caller because the original obtains it from 0x4e893; the
// executor never invents a replacement RNG.  The delay fields describe the
// native wait following the produced frame.
func (p *MontageCycle) Step(randomByte byte) error {
	if p == nil || p.Compositor == nil || p.Phase == MontagePhaseCompleted {
		return errors.New("ending: montage cycle is not running")
	}
	p.DelayTicks, p.DelayMS = 0, 0
	var plan PartyCyclePlan
	var unit []byte
	if p.Phase != MontagePhaseFadeOut {
		if p.PlanIndex < 0 || p.PlanIndex >= len(p.Plans) {
			return errors.New("ending: montage plan index is out of range")
		}
		plan = p.Plans[p.PlanIndex]
		unit = p.Units[plan.UnitSlot]
	}
	switch p.Phase {
	case MontagePhaseFigureFade:
		if p.FadeIndex >= 9 {
			p.Phase = MontagePhaseSecondary
			p.SecondarySM = figani.NativeScheduler{}
			if _, _, err := p.SecondarySM.Step(p.Secondary, false); err != nil {
				return err
			}
			return nil
		}
		frame := p.Secondary.Frames[0]
		if unit[6] != 0 {
			passes, err := p.Montage.PlanFigureFade(int(unit[6]))
			if err != nil {
				return err
			}
			if err := RenderFigureFadePass(p.Compositor, p.Compositor.Work, p.Compositor.Offscreen, p.Assets.TAI003, frame, passes[p.FadeIndex]); err != nil {
				return err
			}
		} else {
			passes, err := p.Montage.PlanMirrorFigureFade(0, 1)
			if err != nil {
				return err
			}
			if err := CopyRect(p.Compositor.Work, Width*2, p.Compositor.Offscreen, Width, Width, Height, Width); err != nil {
				return err
			}
			if err := RenderMirrorFigureFadePass(p.Compositor, p.Compositor.Work, p.Assets.TAI003, frame, frame, passes[p.FadeIndex]); err != nil {
				return err
			}
		}
		p.FadeIndex++
		return nil
	case MontagePhaseSecondary:
		index, _, err := p.SecondarySM.Step(p.Secondary, true)
		if err != nil {
			return err
		}
		if err := p.renderFIGANI(p.Secondary.Frames[index]); err != nil {
			return err
		}
		p.DelayTicks = 1
		p.Portrait++
		if p.Portrait >= 20 {
			p.Phase = MontagePhasePrimary
			p.Portrait = 0
		}
		return nil
	case MontagePhasePrimary:
		if p.PrimaryIdx >= len(p.Primary.Frames) {
			p.Phase = MontagePhasePortrait
			p.Portrait = 0
			p.PortraitMax, _ = NativeMontagePortraitIterations(plan.LoopIndex)
			p.PortraitSM = MontagePortraitState{}
			p.DelayTicks = 0
			return nil
		}
		if err := p.renderFIGANI(p.Primary.Frames[p.PrimaryIdx]); err != nil {
			return err
		}
		p.DelayTicks = p.Primary.Frames[p.PrimaryIdx].Delay
		p.PrimaryIdx++
		return nil
	case MontagePhasePortrait:
		// The native loop polls 0x10620 only after present+0x17aa9(1), then
		// increments EDI and decides whether to leave this portrait.  Keep that
		// boundary separate from rendering so an input arriving during the wait
		// can still affect the final current portrait, rather than one later.
		if p.portraitBoundaryPending {
			p.portraitBoundaryPending = false
			p.Portrait++
			if p.Portrait >= p.PortraitMax {
				nextPlan := p.PlanIndex + 1
				// At 0x2c950 a pending raw input change sets the native outer
				// counter to one.  The current portrait still completes; the next
				// outer-loop iteration is therefore i=0, rather than an immediate
				// abort.  The exact key code remains intentionally outside this API.
				if p.skipToFinal {
					nextPlan = len(p.Plans) - 1
					p.skipToFinal = false
				}
				p.PlanIndex = nextPlan
				if p.PlanIndex >= len(p.Plans) {
					p.Phase = MontagePhaseFadeOut
					p.FadeOut = 0
				} else {
					p.Phase = MontagePhaseFigureFade
					if err := p.preparePlan(); err != nil {
						return err
					}
				}
			}
			return nil
		}
		portraitFrames := p.Assets.Portraits[int(unit[7])]
		frameIndex, err := p.PortraitSM.Step(randomByte)
		if err != nil {
			return err
		}
		frame, err := ComposeMontagePortraitFrame(p.Montage, p.Assets.Grid, unit, p.Portrait, frameIndex, portraitFrames, p.Assets.Current, p.Assets.Permanent, p.Assets.Font)
		if err != nil {
			return err
		}
		secondaryIndex, _, err := p.SecondarySM.Step(p.Secondary, true)
		if err != nil {
			return err
		}
		if err := p.Secondary.Frames[secondaryIndex].BlitAt(frame, Width); err != nil {
			return err
		}
		if err := p.Compositor.CopyToVGA(frame); err != nil {
			return err
		}
		p.DelayTicks = 1
		p.portraitBoundaryPending = true
		return nil
	case MontagePhaseFadeOut:
		if p.FadeOut >= 64 {
			p.Phase = MontagePhaseCompleted
			return nil
		}
		if err := p.Compositor.PaletteDelta(0, 255, p.FadeOut); err != nil {
			return err
		}
		p.FadeOut++
		p.DelayMS = 2
		return nil
	default:
		return errors.New("ending: unknown montage phase")
	}
}

// NeedsRandomByte reports the sole recovered call site where 0x2c99c uses
// 0x4e893: portrait countdown reset.  Callers must not advance the native RNG
// on every rendered frame.
func (p *MontageCycle) NeedsRandomByte() bool {
	return p != nil && p.Phase == MontagePhasePortrait && !p.portraitBoundaryPending && p.PortraitSM.Countdown == 0
}

// ObserveInputChange receives the raw condition observed at 0x2c950, not a
// named keyboard mapping.  The native branch is only polled while a portrait
// is being presented and, except for the final loop, causes middle portraits
// to be skipped after the current one finishes.
func (p *MontageCycle) ObserveInputChange() bool {
	if p == nil || p.Phase != MontagePhasePortrait || !p.portraitBoundaryPending {
		return false
	}
	if p.PlanIndex+1 < len(p.Plans) {
		p.skipToFinal = true
	}
	return true
}

func (p *MontageCycle) renderFIGANI(frame figani.Frame) error {
	clear(p.Compositor.Offscreen)
	if err := frame.BlitAt(p.Compositor.Offscreen, Width); err != nil {
		return err
	}
	return p.Compositor.CopyToVGA(p.Compositor.Offscreen)
}

// Ready reports only completion of the standalone indexed cycle.  It does
// not grant campaign transition or generic input semantics.
func (p MontageCycle) Ready() bool { return p.Phase == MontagePhaseCompleted }
