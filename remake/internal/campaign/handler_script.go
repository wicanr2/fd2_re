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

// HandlerNativeCall preserves one nested raw call site inside a composite
// handler primitive.  It is evidence metadata, not a generic runtime call.
type HandlerNativeCall struct {
	Source  HandlerSource `json:"source"`
	RawArgs []any         `json:"raw_args,omitempty"`
}

// NativeCh23Loop is the two-loop presentation schedule recovered from raw
// ch23 post.  The renderer deliberately consumes this as fail-closed evidence:
// all native call sites, register-shaped arguments, stage values, and external
// raw sources remain visible until an indexed adapter is proven.
type NativeCh23Loop struct {
	Phase              string             `json:"phase"`
	Repeat             int                `json:"repeat"`
	Draw               HandlerNativeCall  `json:"draw"`
	Tick               HandlerNativeCall  `json:"tick"`
	Palette            *HandlerNativeCall `json:"palette,omitempty"`
	Stage              HandlerNativeCall  `json:"stage"`
	StageValues        []int              `json:"stage_values"`
	StageLatchSource   string             `json:"stage_latch_source"`
	TickCounterSource  string             `json:"tick_counter_source"`
	PaletteTableSource string             `json:"palette_table_source,omitempty"`
}

// Native2189ALoop preserves the shared ten-pass indexed presentation helper
// called by the ch22 post handler.  The inner call-sites and push-shaped raw
// arguments remain evidence metadata; no portrait, effect, or gameplay name
// is inferred from the buffer copies.  The runtime deliberately rejects this
// payload until an indexed 0x2189a adapter is proven.
type Native2189ALoop struct {
	Repeat        int               `json:"repeat"`
	StepSource    string            `json:"step_source"`
	WorkOffset    int               `json:"work_offset"`
	WorkStride    int               `json:"work_stride"`
	MapRows       int               `json:"map_rows"`
	MapColumns    int               `json:"map_columns"`
	ClipWidth     int               `json:"clip_width"`
	ClipHeight    int               `json:"clip_height"`
	PresentStride int               `json:"present_stride"`
	MapDraw       HandlerNativeCall `json:"map_draw"`
	Composite     HandlerNativeCall `json:"composite"`
	Stage         HandlerNativeCall `json:"stage"`
	Present       HandlerNativeCall `json:"present"`
	Tail          HandlerNativeCall `json:"tail"`
}

// HandlerCondition is the editable, evidence-level predicate attached to a
// structured handler branch.  Each predicate is a direct transcription of a
// native helper; this is deliberately not a generic expression language.
type HandlerCondition struct {
	Op        string `json:"op"`
	UnitSlots []int  `json:"unit_slots,omitempty"`
	// NativeInventoryItemID is the unsigned byte passed to 0x24b14. The
	// compiler accepts it only for the exact raw inventory-prefix predicate;
	// it is not a normalized inventory search.
	NativeInventoryItemID *int `json:"native_inventory_item_id,omitempty"`
	// NativePersistentIdentity is the unsigned byte compared at persistent
	// record+0x08 by 0x24bde/0x33499. The record field remains raw evidence,
	// not a character-name assertion.
	NativePersistentIdentity *int `json:"native_persistent_identity,omitempty"`
	// Threshold is used only by raw count predicates; it is never inferred
	// from a roster size or normalized alive count.
	Threshold *int `json:"threshold,omitempty"`
	// NativeRound is the raw [0x53bef] comparison used by handlers such as
	// ch15_post. It is intentionally separate from the normalized battle Turn.
	NativeRound *int `json:"native_round,omitempty"`
	// NativeRecordWordOffset/Value identify a direct raw record-word compare;
	// only the recovered +0x42 u16 contract is currently accepted.
	NativeRecordWordOffset *int `json:"native_record_word_offset,omitempty"`
	NativeRecordWordValue  *int `json:"native_record_word_value,omitempty"`
	// UnitSlot selects the raw runtime record for native_record_word_gte.
	UnitSlot *int `json:"unit_slot,omitempty"`
	// Any contains a bounded OR of already-proven native predicates. It is
	// intentionally not a general expression language; the compiler accepts
	// only the raw predicate ops it knows how to validate.
	Any []HandlerCondition `json:"any,omitempty"`
	// CharID is the one-byte permanent-player ID accepted by native 0x33499.
	// It is meaningful only for roster_has, never a portrait/NPC identifier.
	CharID          *int `json:"char_id,omitempty"`
	EventStateIndex *int `json:"event_state_index,omitempty"`
	EventStateValue *int `json:"event_state_value,omitempty"`
	// RequiredSlotCount 是此條件成立時由同一原生 producer 證實的精確
	// runtime frontier。它讓分支內 slot 操作可驗證，但不會擴張外層 frontier。
	RequiredSlotCount *int `json:"required_slot_count,omitempty"`
}

// HandlerRepeatHint records a directly recovered fixed-count loop around one
// native operation. It is evidence metadata, not a generic scripting loop.
type HandlerRepeatHint struct {
	LoopBackTo string `json:"loop_back_to"`
	Limit      int    `json:"limit"`
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
	UnitSlotExpr any           `json:"unit_slot_expr,omitempty"`
	Group        *int          `json:"group,omitempty"`
	// RawPlacementGate is the exact byte observed by 0x10C50 at [0x53AFA].
	// Pointer form preserves an explicit zero and lets the compiler reject a
	// lossy original-handler export instead of guessing from the group number.
	RawPlacementGate  *int                      `json:"raw_placement_gate,omitempty"`
	CharID            *int                      `json:"char_id,omitempty"`
	ItemID            *int                      `json:"item_id,omitempty"`
	ResourceID        *int                      `json:"resource_id,omitempty"`
	SFXIndex          *int                      `json:"sfx_index,omitempty"`
	Track             *int                      `json:"track,omitempty"`
	Loop              *int                      `json:"loop,omitempty"`
	Direction         *int                      `json:"direction,omitempty"`
	Repeat            *int                      `json:"repeat,omitempty"`
	RepeatHint        *HandlerRepeatHint        `json:"repeat_hint,omitempty"`
	UnitPresent       *HandlerUnitPresent       `json:"unit_present,omitempty"`
	DirectRecordPatch *HandlerDirectRecordPatch `json:"direct_record_patch,omitempty"`
	IndexedTransition *HandlerIndexedTransition `json:"indexed_transition,omitempty"`
	NativeCh23Loop    *NativeCh23Loop           `json:"native_ch23_loop,omitempty"`
	Native2189ALoop   *Native2189ALoop          `json:"native_2189a_loop,omitempty"`
	Ms                *int                      `json:"ms,omitempty"`
	Variant           string                    `json:"variant,omitempty"`
	Value             any                       `json:"value,omitempty"`
	NativeTarget      string                    `json:"native_target,omitempty"`
	// NativeSemantic is the evidence-index label emitted for a classified
	// native call. It is audit metadata; lowering still keys on NativeTarget,
	// source address, raw arguments and explicit bindings.
	NativeSemantic   string            `json:"native_semantic,omitempty"`
	NativeConfidence string            `json:"native_confidence,omitempty"`
	NativeEvidence   []string          `json:"native_evidence,omitempty"`
	RawArgs          []any             `json:"raw_args,omitempty"`
	Args             []any             `json:"args,omitempty"`
	Condition        *HandlerCondition `json:"condition,omitempty"`
	Then             []HandlerBeat     `json:"then,omitempty"`
	Else             []HandlerBeat     `json:"else,omitempty"`
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
	if err := validateHandlerBeats(path, "beats", script.Beats); err != nil {
		return nil, err
	}
	return &script, nil
}

func validateHandlerBeats(path, location string, beats []HandlerBeat) error {
	for i, beat := range beats {
		at := fmt.Sprintf("%s[%d]", location, i)
		if beat.Op == "" {
			return fmt.Errorf("handler script %q %s has no op", path, at)
		}
		if beat.Op == "native_call" || beat.Op == "unresolved_native_call" {
			if beat.NativeTarget == "" || beat.NativeSemantic == "" ||
				beat.NativeConfidence == "" || len(beat.NativeEvidence) == 0 {
				return fmt.Errorf("handler script %q %s native call lacks target/semantic/confidence/evidence", path, at)
			}
		}
		if beat.Op == "if" {
			if err := validateHandlerBeats(path, at+".then", beat.Then); err != nil {
				return err
			}
			if err := validateHandlerBeats(path, at+".else", beat.Else); err != nil {
				return err
			}
		}
	}
	return nil
}
