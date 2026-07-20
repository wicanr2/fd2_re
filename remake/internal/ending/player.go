package ending

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/afm"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// Player advances the evidence-backed prefix of a native ending against a
// millisecond presentation clock.  It deliberately blocks at the first
// unsupported native operation, retaining the last verified VGA frame for a
// UI adapter to display.  It is not a permissive generic scripting runtime.
type Player struct {
	Timeline   Timeline
	Frames     []fdother.Frame
	ANI        *afm.Clip
	Compositor *IndexedCompositor

	Segment int
	WaitMS  int
	ani     *ANIPlayer
	ramp    *paletteRamp
	Blocked *Segment
	State   PlaybackState
}

type paletteRamp struct {
	value, end, step, delay int
	next                    bool
}

func NewPlayer(t Timeline, frames []fdother.Frame, ani *afm.Clip, compositor *IndexedCompositor) (*Player, error) {
	if compositor == nil {
		return nil, errors.New("ending: nil compositor")
	}
	if len(t.Segments) == 0 {
		return nil, errors.New("ending: timeline has no segments")
	}
	return &Player{
		Timeline:   t,
		Frames:     frames,
		ANI:        ani,
		Compositor: compositor,
		State:      PlaybackRunning,
	}, nil
}

// Advance consumes up to elapsedMS of wall-clock time.  The caller should use
// a monotonic presentation clock; zero is useful to present the first ANI
// frame immediately, exactly as 0x20421 does.
func (p *Player) Advance(elapsedMS int) (PlaybackState, error) {
	if elapsedMS < 0 {
		return p.State, errors.New("ending: negative elapsed time")
	}
	if p.State != PlaybackRunning {
		return p.State, nil
	}
	for {
		if p.WaitMS > 0 {
			if elapsedMS < p.WaitMS {
				p.WaitMS -= elapsedMS
				return p.State, nil
			}
			elapsedMS -= p.WaitMS
			p.WaitMS = 0
			if p.ramp != nil {
				p.ramp.next = true
			} else {
				p.Segment++
			}
		}
		if p.ramp != nil {
			if p.ramp.next {
				p.ramp.value += p.ramp.step
				p.ramp.next = false
				if (p.ramp.step < 0 && p.ramp.value < p.ramp.end) || (p.ramp.step > 0 && p.ramp.value > p.ramp.end) {
					p.ramp = nil
					p.Segment++
					continue
				}
			}
			if err := p.Compositor.PaletteDelta(0, 255, p.ramp.value); err != nil {
				return p.State, err
			}
			p.WaitMS = p.ramp.delay
			p.ramp.next = true
			if elapsedMS == 0 {
				return p.State, nil
			}
			continue
		}
		if p.ani != nil {
			done, err := p.ani.Advance(p.Compositor, elapsedMS)
			if err != nil {
				return p.State, err
			}
			if !done {
				return p.State, nil
			}
			p.ani = nil
			p.Segment++
			// ANIPlayer owns its exact cadence.  On its completion tick any
			// remaining scheduler work starts on the following presentation tick.
			return p.State, nil
		}
		if p.Segment >= len(p.Timeline.Segments) {
			p.State = PlaybackCompleted
			return p.State, nil
		}

		s := &p.Timeline.Segments[p.Segment]
		switch s.Op {
		case "blit_frame":
			if err := p.blit(*s); err != nil {
				return p.State, err
			}
			p.Segment++
		case "copy_buffer":
			if s.From != "offscreen" || s.To != "vga" || s.Bytes != Bytes {
				return p.State, fmt.Errorf("ending: unsupported copy at %s", s.Source)
			}
			if err := p.Compositor.CopyToVGA(p.Compositor.Offscreen); err != nil {
				return p.State, err
			}
			p.Segment++
		case "delay_ms":
			if s.Ms < 0 {
				return p.State, fmt.Errorf("ending: invalid delay at %s", s.Source)
			}
			p.WaitMS = s.Ms
			if p.WaitMS == 0 {
				p.Segment++
			}
		case "ani_play":
			if s.ANIResource == nil || *s.ANIResource != 2 || s.FrameDelayMs != 100 || s.Skippable == nil || *s.Skippable || p.ANI == nil {
				return p.State, fmt.Errorf("ending: unavailable ANI at %s", s.Source)
			}
			p.ani = &ANIPlayer{Clip: p.ANI, DelayMs: s.FrameDelayMs}
		case "palette_update":
			if s.PaletteStart == nil || s.PaletteEnd == nil || s.PaletteValue == nil {
				return p.State, fmt.Errorf("ending: incomplete palette update at %s", s.Source)
			}
			if err := p.Compositor.PaletteDelta(*s.PaletteStart, *s.PaletteEnd, *s.PaletteValue); err != nil {
				return p.State, err
			}
			p.Segment++
		case "palette_ramp":
			if s.PaletteStart == nil || s.PaletteEnd == nil || s.PaletteStep == 0 || s.PaletteDelay <= 0 {
				return p.State, fmt.Errorf("ending: incomplete palette ramp at %s", s.Source)
			}
			p.ramp = &paletteRamp{value: *s.PaletteStart, end: *s.PaletteEnd, step: s.PaletteStep, delay: s.PaletteDelay}
		default:
			p.Blocked = s
			p.State = PlaybackBlocked
			return p.State, nil
		}
	}
}

func (p *Player) blit(s Segment) error {
	if s.Frame == nil || *s.Frame < 0 || *s.Frame >= len(p.Frames) || s.Transparent == nil || *s.Transparent != -1 {
		return fmt.Errorf("ending: invalid recovered blit at %s", s.Source)
	}
	var dst []byte
	switch s.Target {
	case "offscreen":
		dst = p.Compositor.Offscreen
	case "vga":
		dst = p.Compositor.VGA
	default:
		return fmt.Errorf("ending: unsupported blit target %q", s.Target)
	}
	return p.Compositor.Blit(p.Frames[*s.Frame], dst, s.Stride)
}

// BlockedDialogue returns the exact branch selected by the native chapter
// comparison at an opaque text call.  The player remains blocked: presenting
// these lines never grants permission to execute following opaque segments.
func (p *Player) BlockedDialogue(chapter int) ([]DialogueBlock, bool) {
	if p.State != PlaybackBlocked || p.Blocked == nil || p.Blocked.Op != "native_text_branch_opaque" {
		return nil, false
	}
	blocks := p.Blocked.ElseDialogue
	if chapter == 26 {
		blocks = p.Blocked.ThenDialogue
	}
	if len(blocks) == 0 {
		return nil, false
	}
	return append([]DialogueBlock(nil), blocks...), true
}
