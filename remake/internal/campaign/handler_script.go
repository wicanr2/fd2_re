package campaign

import (
	"encoding/json"
	"fmt"
	"os"
)

// HandlerSource preserves the original EXE call site for every editable beat.
// It is audit metadata, not a runtime address used by the remake.
type HandlerSource struct {
	Addr   string `json:"addr"`
	Target string `json:"target,omitempty"`
}

// HandlerBeat is the lossless editable IR exported from one hard-coded EXE
// handler.  Fields are intentionally sparse: each Op uses only its matching
// fields, and RawArgs keeps unclassified native calls visible to editors.
type HandlerBeat struct {
	Op           string        `json:"op"`
	Source       HandlerSource `json:"source,omitempty"`
	Chapter      *int          `json:"chapter,omitempty"`
	ChapterExpr  any           `json:"chapter_expr,omitempty"`
	GridX        *int          `json:"grid_x,omitempty"`
	GridY        *int          `json:"grid_y,omitempty"`
	TextIndex    any           `json:"text_index,omitempty"`
	TextTable    string        `json:"text_table,omitempty"`
	ActingID     *int          `json:"acting_id,omitempty"`
	UnitSlot     *int          `json:"unit_slot,omitempty"`
	Group        *int          `json:"group,omitempty"`
	CharID       *int          `json:"char_id,omitempty"`
	Track        *int          `json:"track,omitempty"`
	Loop         *int          `json:"loop,omitempty"`
	Direction    *int          `json:"direction,omitempty"`
	Repeat       *int          `json:"repeat,omitempty"`
	Ms           *int          `json:"ms,omitempty"`
	Variant      string        `json:"variant,omitempty"`
	Value        any           `json:"value,omitempty"`
	NativeTarget string        `json:"native_target,omitempty"`
	RawArgs      []any         `json:"raw_args,omitempty"`
	Args         []any         `json:"args,omitempty"`
}

// HandlerScript is a chapter pre/post handler in editable JSON form.  It is
// deliberately distinct from Beat: it preserves original grid/text/resource
// identifiers so a campaign author can edit or audit source choreography
// before compiling it to a map-specific runtime Beat sequence.
type HandlerScript struct {
	SchemaVersion int            `json:"schema_version"`
	Chapter       int            `json:"chapter"`
	Phase         string         `json:"phase"`
	Handler       string         `json:"handler"`
	Beats         []HandlerBeat  `json:"beats"`
	Diagnostics   map[string]int `json:"diagnostics,omitempty"`
}

// LoadHandlerScript reads an editable EXE-handler export and rejects malformed
// scripts early.  Unknown operations are valid data: they are explicitly
// preserved until their native semantics have been RE'd.
func LoadHandlerScript(path string) (*HandlerScript, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var script HandlerScript
	if err := json.Unmarshal(raw, &script); err != nil {
		return nil, err
	}
	if script.SchemaVersion != 1 {
		return nil, fmt.Errorf("handler script %q schema_version=%d, want 1", path, script.SchemaVersion)
	}
	if script.Phase != "pre" && script.Phase != "post" {
		return nil, fmt.Errorf("handler script %q has invalid phase %q", path, script.Phase)
	}
	if script.Handler == "" {
		return nil, fmt.Errorf("handler script %q has no handler", path)
	}
	for i, beat := range script.Beats {
		if beat.Op == "" {
			return nil, fmt.Errorf("handler script %q beat %d has no op", path, i)
		}
	}
	return &script, nil
}
