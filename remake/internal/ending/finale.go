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
