package ending

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

// FinalePhase is an editable, evidence-only transcription of a native finale
// stage.  It intentionally has no Player adapter until every renderer input
// for that stage is recovered.
type FinalePhase struct {
	SchemaVersion int             `json:"schema_version"`
	NativeHandler string          `json:"native_handler"`
	Status        string          `json:"status"`
	Phase         FinalePhaseSpec `json:"phase"`
	Gate          FinalePhaseGate `json:"gate"`
}

type FinalePhaseSpec struct {
	Source                      string `json:"source"`
	SourceDAT                   string `json:"source_dat"`
	StringIndex                 int    `json:"string_index"`
	Script                      string `json:"script"`
	SceneIndex                  int    `json:"scene_index"`
	Line                        int    `json:"line"`
	Count                       int    `json:"count"`
	StagingBytes                int    `json:"staging_bytes"`
	TextOffset                  int    `json:"text_offset"`
	Stride                      int    `json:"stride"`
	LineAdvanceRows             int    `json:"line_advance_rows"`
	ViewportRows                int    `json:"viewport_rows"`
	Iterations                  int    `json:"iterations"`
	DelayMS                     int    `json:"delay_ms"`
	BaselinePaletteInitialDelta int    `json:"baseline_palette_initial_delta"`
	FadeOutThroughIteration     int    `json:"fade_out_through_iteration"`
	PaletteStepCadence          int    `json:"palette_step_cadence"`
}

type FinalePhaseGate struct {
	Source string `json:"source"`
	Reason string `json:"reason"`
}

func LoadFinalePhase(path string) (*FinalePhase, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var phase FinalePhase
	if err := json.Unmarshal(raw, &phase); err != nil {
		return nil, err
	}
	p := phase.Phase
	if phase.SchemaVersion != 1 || phase.NativeHandler != "0x2c405" || phase.Status != "recovered_phase0_only_fail_closed" ||
		p.Source != "0x2c469" || p.SourceDAT != "FDTXT_031" || p.StringIndex != 44 || p.Script != "ch32.json" || p.SceneIndex != 0 || p.Line != 0 || p.Count != 1 ||
		p.StagingBytes != 0x36b00 || p.TextOffset != 0x12c30 || p.Stride != Width || p.LineAdvanceRows != 25 || p.ViewportRows != Height || p.Iterations != 500 || p.DelayMS != 1 ||
		p.BaselinePaletteInitialDelta != 40 || p.FadeOutThroughIteration != 300 || p.PaletteStepCadence != 5 ||
		phase.Gate.Source != "0x2c548" || phase.Gate.Reason == "" {
		return nil, fmt.Errorf("ending finale phase %q is incomplete or unsupported", path)
	}
	return &phase, nil
}

// Ready always remains false until the exact native text compositor and its
// staging source have an evidence-backed remake adapter.
func (p FinalePhase) Ready() bool { return false }

// ComposePhase0Text reproduces the fully evidenced glyph-only part of
// 0x2c469: physical FDTXT_031 string #44 is written continuously from
// staging+0x12c30 with 16-byte glyph advance and the caller's native colour
// arguments (stride=320, foreground=CD, shadow=4C, transparent background).
// FFFE is the exactly recovered line advance to destination + line*25*stride;
// every other FFxx word remains rejected so this does not claim to be the
// unrecovered general 0x15f84 control-code renderer.
func (p FinalePhase) ComposePhase0Text(staging, textResource, fontResource []byte) (int, error) {
	if p.Ready() || len(staging) != p.Phase.StagingBytes {
		return 0, fmt.Errorf("ending: invalid phase-0 staging surface")
	}
	strings, err := fdtxt.Parse(textResource)
	if err != nil {
		return 0, err
	}
	words, err := strings.Words(p.Phase.StringIndex)
	if err != nil {
		return 0, err
	}
	font, err := fdtxt.ParseFont(fontResource)
	if err != nil {
		return 0, err
	}
	base := p.Phase.TextOffset
	line := 0
	for _, word := range words {
		if word == 0xfffe {
			line++
			base = p.Phase.TextOffset + line*p.Phase.LineAdvanceRows*p.Phase.Stride
			continue
		}
		if word >= fdtxt.ControlMin {
			return 0, fmt.Errorf("ending: phase-0 string has unsupported control %#x", word)
		}
		if err := font.BlitNativeGlyph(staging, p.Phase.Stride, base, int(word), fdtxt.NativeGlyphStyle{Foreground: 0xcd, Shadow: 0x4c}); err != nil {
			return 0, err
		}
		base += fdtxt.GlyphWidth
	}
	return base, nil
}

// Phase0Player is the exact 500-pass row-scroll portion after 0x2c469. It is
// intentionally separate from the ending Player: completing this phase does
// not authorise execution of the unrecovered 0x2c548 montage.
type Phase0Player struct {
	phase        FinalePhase
	staging      []byte
	compositor   *IndexedCompositor
	iteration    int
	paletteDelta int
	waitMS       int
	done         bool
}

// Phase0Assets keeps every non-global input needed by the recovered phase.
// Baseline must come from the preceding native presentation state; callers
// cannot silently substitute a convenience palette.
type Phase0Assets struct {
	TextResource []byte
	FontResource []byte
	Baseline     [768]byte
}

func NewPhase0PlayerFromAssets(phase FinalePhase, assets Phase0Assets, compositor *IndexedCompositor) (*Phase0Player, error) {
	if compositor == nil {
		return nil, fmt.Errorf("ending: nil phase-0 compositor")
	}
	staging := make([]byte, phase.Phase.StagingBytes)
	if _, err := phase.ComposePhase0Text(staging, assets.TextResource, assets.FontResource); err != nil {
		return nil, err
	}
	copy(compositor.Baseline[:], assets.Baseline[:])
	return NewPhase0Player(phase, staging, compositor)
}

func NewPhase0Player(phase FinalePhase, staging []byte, compositor *IndexedCompositor) (*Phase0Player, error) {
	if phase.Ready() || len(staging) != phase.Phase.StagingBytes || compositor == nil {
		return nil, fmt.Errorf("ending: invalid phase-0 player")
	}
	return &Phase0Player{phase: phase, staging: staging, compositor: compositor, paletteDelta: phase.Phase.BaselinePaletteInitialDelta}, nil
}

// Advance consumes the native 1ms waits. The first iteration presents
// immediately, as the original falls through to 0x2c4b4 before its first
// 0x17aa9(1) call.
func (p *Phase0Player) Advance(elapsedMS int) (done bool, err error) {
	if elapsedMS < 0 {
		return p.done, fmt.Errorf("ending: negative phase-0 elapsed time")
	}
	if p.done {
		return true, nil
	}
	for {
		if p.waitMS > 0 {
			if elapsedMS < p.waitMS {
				p.waitMS -= elapsedMS
				return false, nil
			}
			elapsedMS -= p.waitMS
			p.waitMS = 0
			p.iteration++
		}
		if p.iteration >= p.phase.Phase.Iterations {
			p.done = true
			return true, nil
		}
		if err := p.compositor.SetBaselineDelta(0, 255, p.paletteDelta); err != nil {
			return false, err
		}
		offset := p.iteration * p.phase.Phase.Stride
		if err := CopyRect(p.compositor.VGA, Width, p.staging[offset:], p.phase.Phase.Stride, Width, p.phase.Phase.ViewportRows, 0); err != nil {
			return false, err
		}
		// 0x2c4f9: the fade-out branch is entered only for i<200; its
		// decrement follows the current frame's palette/copy work.
		if p.iteration < 200 && p.paletteDelta > 0 && p.iteration%p.phase.Phase.PaletteStepCadence == 0 {
			p.paletteDelta--
		}
		// 0x2c51b: fade-in starts strictly after i=300, also after the
		// current frame has been presented.
		if p.iteration > p.phase.Phase.FadeOutThroughIteration && p.iteration%p.phase.Phase.PaletteStepCadence == 0 {
			p.paletteDelta++
		}
		p.waitMS = p.phase.Phase.DelayMS
		if elapsedMS == 0 {
			return false, nil
		}
	}
}
