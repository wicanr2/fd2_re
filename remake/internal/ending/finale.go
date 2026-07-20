package ending

import (
	"encoding/json"
	"fmt"
	"os"
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
	NativeSelector              int    `json:"native_selector"`
	RawStringIndex              int    `json:"raw_string_index"`
	RawUtteranceIndex           int    `json:"raw_utterance_index"`
	Script                      string `json:"script"`
	SceneIndex                  int    `json:"scene_index"`
	Line                        int    `json:"line"`
	Count                       int    `json:"count"`
	StagingBytes                int    `json:"staging_bytes"`
	TextOffset                  int    `json:"text_offset"`
	Stride                      int    `json:"stride"`
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
		p.Source != "0x2c469" || p.SourceDAT != "FDTXT_030" || p.NativeSelector != 44 || p.RawStringIndex != 10 || p.RawUtteranceIndex != 6 || p.Script != "ch30.json" || p.SceneIndex != 3 || p.Line != 6 || p.Count != 1 ||
		p.StagingBytes != 0x36b00 || p.TextOffset != 0x12c30 || p.Stride != Width || p.ViewportRows != Height || p.Iterations != 500 || p.DelayMS != 1 ||
		p.BaselinePaletteInitialDelta != 40 || p.FadeOutThroughIteration != 300 || p.PaletteStepCadence != 5 ||
		phase.Gate.Source != "0x2c548" || phase.Gate.Reason == "" {
		return nil, fmt.Errorf("ending finale phase %q is incomplete or unsupported", path)
	}
	return &phase, nil
}

// Ready always remains false until the exact native text compositor and its
// staging source have an evidence-backed remake adapter.
func (p FinalePhase) Ready() bool { return false }
