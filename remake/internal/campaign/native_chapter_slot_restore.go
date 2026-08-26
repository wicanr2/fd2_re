package campaign

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

const (
	nativeIntermissionGateCount   = 30
	nativeIntermissionGateAddress = "0x526b9"
)

// NativeIntermissionGateSource binds the editable table to the exact original
// executable and byte address from which it was transcribed.
type NativeIntermissionGateSource struct {
	ReferenceFile string `json:"reference_file"`
	EXESHA256     string `json:"exe_sha256"`
	Address       string `json:"address"`
}

// NativeIntermissionGateTable is the original 30-byte table consumed by
// 0x2cad7. Zero enters the selectable town hub; nonzero enters preparation
// directly. Entries use the raw [0x53c03] chapter index, not display chapter.
type NativeIntermissionGateTable struct {
	SchemaVersion           int                               `json:"schema_version"`
	Source                  NativeIntermissionGateSource      `json:"source"`
	Entries                 []byte                            `json:"entries"`
	EffectiveRouteOverrides []NativeIntermissionRouteOverride `json:"effective_route_overrides"`
}

// NativeIntermissionRouteOverride preserves a player-visible effective route
// that cannot be derived from the 0x526b9 binary gate alone. It supplements,
// rather than rewrites, the original table and must carry reviewable evidence.
type NativeIntermissionRouteOverride struct {
	RawChapter    int    `json:"raw_chapter"`
	EntryNode     string `json:"entry_node"`
	Evidence      string `json:"evidence"`
	EvidenceLevel string `json:"evidence_level"`
}

// LoadNativeIntermissionGateTable loads and validates a complete version-bound
// transcription. Missing, reordered, non-binary, or wrong-version data fails
// closed before campaign state can be changed.
func LoadNativeIntermissionGateTable(
	path string,
) (*NativeIntermissionGateTable, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("native intermission gate: %w", err)
	}
	var table NativeIntermissionGateTable
	if err := json.Unmarshal(raw, &table); err != nil {
		return nil, fmt.Errorf("native intermission gate: %w", err)
	}
	if err := table.validate(); err != nil {
		return nil, err
	}
	return &table, nil
}

func (table *NativeIntermissionGateTable) validate() error {
	if table == nil {
		return fmt.Errorf("native intermission gate: missing table")
	}
	if table.SchemaVersion != 2 {
		return fmt.Errorf(
			"native intermission gate: schema version=%d, want 2",
			table.SchemaVersion,
		)
	}
	if table.Source.ReferenceFile != "docs/data/fd2-reference-files.json" ||
		table.Source.EXESHA256 != nativeCharacterEXESHA256 ||
		table.Source.Address != nativeIntermissionGateAddress {
		return fmt.Errorf(
			"native intermission gate: source does not match reference FD2.EXE",
		)
	}
	if len(table.Entries) != nativeIntermissionGateCount {
		return fmt.Errorf(
			"native intermission gate: entries=%d, want %d",
			len(table.Entries), nativeIntermissionGateCount,
		)
	}
	for index, value := range table.Entries {
		if value > 1 {
			return fmt.Errorf(
				"native intermission gate: entry %d=%d, want 0 or 1",
				index, value,
			)
		}
	}
	seen := make(map[int]bool)
	for _, override := range table.EffectiveRouteOverrides {
		if override.RawChapter < 0 || override.RawChapter >= len(table.Entries) ||
			override.EntryNode == "" || override.Evidence == "" ||
			override.EvidenceLevel != "confirmed-fixed-input" || seen[override.RawChapter] {
			return fmt.Errorf(
				"native intermission gate: invalid effective route override for raw chapter %d",
				override.RawChapter,
			)
		}
		seen[override.RawChapter] = true
	}
	return nil
}

// NativeChapterSlotRestorePlan is a fully validated, side-effect-free restore
// transaction. Raw option bytes are preserved but not connected to gameplay
// consumers whose semantics remain open.
type NativeChapterSlotRestorePlan struct {
	Slot               int
	NativeChapterIndex int
	DisplayChapter     int
	EntryNode          string
	Currency           uint32
	HUDGateA           byte
	Raw53AF9           byte
	Raw51E61           byte
	Raw51E62           byte
	PartyMembers       map[int]bool
	PartyJoinOrder     []int
	PartyRoster        map[int]battle.Unit
}

// BuildNativeChapterSlotRestorePlan converts one native four-slot snapshot
// into an existing editable intermission node. The original save callers sit
// inside 0x2cad7/inn flows, after post-battle inventory gates; therefore this
// must not replay inventory_recipe_ch21 or inventory_gate_ch27.
func BuildNativeChapterSlotRestorePlan(
	snapshot fdsave.ChapterSlotSnapshot,
	catalog *NativeCharacterCatalog,
	table *NativeIntermissionGateTable,
	graph *Campaign,
) (NativeChapterSlotRestorePlan, error) {
	if err := catalog.validate(); err != nil {
		return NativeChapterSlotRestorePlan{}, err
	}
	if err := table.validate(); err != nil {
		return NativeChapterSlotRestorePlan{}, err
	}
	if graph == nil {
		return NativeChapterSlotRestorePlan{}, fmt.Errorf(
			"native chapter restore: missing campaign graph",
		)
	}
	if snapshot.Slot < 0 || snapshot.Slot >= fdsave.SlotCount {
		return NativeChapterSlotRestorePlan{}, fmt.Errorf(
			"native chapter restore: slot %d outside 0..%d",
			snapshot.Slot, fdsave.SlotCount-1,
		)
	}
	if int(snapshot.Verified.RosterCount) > fdsave.RosterUnits {
		return NativeChapterSlotRestorePlan{}, fmt.Errorf(
			"native chapter restore: roster count %d exceeds native capacity %d",
			snapshot.Verified.RosterCount, fdsave.RosterUnits,
		)
	}
	rawChapter := int(snapshot.Verified.Chapter)
	if rawChapter < 0 || rawChapter >= len(table.Entries) {
		return NativeChapterSlotRestorePlan{}, fmt.Errorf(
			"native chapter restore: raw chapter %d outside gate table",
			rawChapter,
		)
	}
	displayChapter := rawChapter + 1
	nodeType := "town"
	nodeID := fmt.Sprintf("town_ch%02d", displayChapter)
	if table.Entries[rawChapter] != 0 {
		nodeType = "preparation"
		nodeID = fmt.Sprintf("preparation_ch%02d", displayChapter)
	}
	for _, override := range table.EffectiveRouteOverrides {
		if override.RawChapter == rawChapter {
			nodeID = override.EntryNode
			if strings.HasPrefix(nodeID, "preparation_") {
				nodeType = "preparation"
			} else if strings.HasPrefix(nodeID, "town_") {
				nodeType = "town"
			} else {
				return NativeChapterSlotRestorePlan{}, fmt.Errorf(
					"native chapter restore: unsupported effective route %q", nodeID,
				)
			}
			break
		}
	}
	node, ok := graph.Nodes[nodeID]
	if !ok || node == nil || node.Type != nodeType {
		return NativeChapterSlotRestorePlan{}, fmt.Errorf(
			"native chapter restore: %s node %q is unavailable",
			nodeType, nodeID,
		)
	}

	plan := NativeChapterSlotRestorePlan{
		Slot:               snapshot.Slot,
		NativeChapterIndex: rawChapter,
		DisplayChapter:     displayChapter,
		EntryNode:          nodeID,
		Currency:           snapshot.Verified.Currency,
		HUDGateA:           snapshot.Verified.HUDGateA,
		Raw53AF9:           snapshot.Verified.Raw53AF9,
		Raw51E61:           snapshot.Verified.Raw51E61,
		Raw51E62:           snapshot.Verified.Raw51E62,
		PartyMembers:       make(map[int]bool),
		PartyRoster:        make(map[int]battle.Unit),
	}
	for index, record := range snapshot.ActiveRecords() {
		unit, err := MaterializeNativePersistentPartyRecord(record, catalog)
		if err != nil {
			return NativeChapterSlotRestorePlan{}, fmt.Errorf(
				"native chapter restore: roster record %d: %w", index, err,
			)
		}
		identity := unit.NativeIdentity
		if plan.PartyMembers[identity] {
			return NativeChapterSlotRestorePlan{}, fmt.Errorf(
				"native chapter restore: duplicate identity %d", identity,
			)
		}
		plan.PartyMembers[identity] = true
		plan.PartyJoinOrder = append(plan.PartyJoinOrder, identity)
		plan.PartyRoster[identity] = unit
	}
	return plan, nil
}
