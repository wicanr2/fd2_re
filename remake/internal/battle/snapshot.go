// snapshot.go — 戰場 JSON 快照（重製端自有存檔擴充，56「2026-11-13 #303 戰場JSON快照」）。
//
// 快照不是原版 FD2.SAV ABI。State 每個欄位都必須歸入 stateSnapshotFieldClass 的
// 三類之一，snapshot_test 以反射列舉欄位；新增欄位沒分類就失敗，避免新狀態漏存後
// 讀檔靜默偏差。
package battle

import (
	"errors"
	"fmt"
	"sort"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// StateSnapshotVersion 只在快照欄位語意改變時遞增；讀到不同版本一律拒絕。
const StateSnapshotVersion = 1

const (
	snapshotSaved     = "saved"     // 存入快照
	snapshotBound     = "bound"     // 讀檔時由應用層從資產與劇本重新綁定
	snapshotTransient = "transient" // 受理存檔時必須是零值
)

var stateSnapshotFieldClass = map[string]string{
	"Balance":                        snapshotBound,
	"W":                              snapshotSaved,
	"H":                              snapshotSaved,
	"Units":                          snapshotSaved,
	"NativeMapSelectorCache":         snapshotSaved,
	"NativeMapSelectorError":         snapshotSaved,
	"NativeMapCycleState":            snapshotSaved,
	"HasNativeMapCycleState":         snapshotSaved,
	"NativeTerrainPhaseState":        snapshotSaved,
	"HasNativeTerrainPhaseState":     snapshotSaved,
	"NativeTerrainFlipState":         snapshotSaved,
	"NativeUnitPixelShiftState":      snapshotSaved,
	"HasNativeMapBinaryTimingState":  snapshotSaved,
	"NativeMapHUDState":              snapshotSaved,
	"HasNativeMapHUDState":           snapshotSaved,
	"NativeMapViewState":             snapshotSaved,
	"HasNativeMapViewState":          snapshotSaved,
	"NativeMapRangeMode":             snapshotSaved,
	"HasNativeMapRangeModeState":     snapshotSaved,
	"Roster":                         snapshotSaved,
	"PendingGroups":                  snapshotSaved,
	"nativeFutureItemRows":           snapshotBound,
	"nativeAIScan":                   snapshotTransient,
	"OwnDeploy":                      snapshotSaved,
	"Turn":                           snapshotSaved,
	"NativeRoundCounter":             snapshotSaved,
	"Flags":                          snapshotSaved,
	"NativeEventState":               snapshotSaved,
	"Cost":                           snapshotSaved,
	"NativeCompositionEventBytes":    snapshotSaved,
	"NativeMapEventGrid":             snapshotSaved,
	"HasNativeMapEventGrid":          snapshotSaved,
	"NativeFieldControlRaw":          snapshotSaved,
	"NativeTurnEventControls":        snapshotSaved,
	"HasNativeTurnEventControlState": snapshotSaved,
	"NativeChestControls":            snapshotSaved,
	"HasNativeChestControlState":     snapshotSaved,
	"NativeFieldUnitControls":        snapshotSaved,
	"HasNativeFieldControlState":     snapshotSaved,
	"NativeRuntimeRecords":           snapshotSaved,
	"HasNativeRuntimeUnitProjection": snapshotSaved,
	"HasNativePendingGroupBinding":   snapshotSaved,
	"NativeFieldEventSlots":          snapshotSaved,
	"NativeFieldEvents":              snapshotSaved,
	"NativeFieldEventRules":          snapshotBound,
	"NativeTileBlitModes":            snapshotSaved,
	"NativeTerrainControl":           snapshotSaved,
	"NativeTerrainMoveCodes":         snapshotSaved,
	"nativeMovementCostRows":         snapshotBound,
	"SpellBook":                      snapshotSaved,
	"NativeCommandBook":              snapshotBound,
	"NativeCommandResistances":       snapshotBound,
	"CommandLearn":                   snapshotBound,
	"CommandLearnSelectors":          snapshotBound,
	"NativeGrowthRows":               snapshotBound,
	"NativeGrowthRollObserver":       snapshotBound,
	"NativeDeathExpCancel":           snapshotBound,
	"AICommandSpell":                 snapshotSaved,
	"Treasures":                      snapshotSaved,
	"NativeTreasureEventRules":       snapshotBound,
	"NativeExperienceAccumulator":    snapshotTransient,
	"OpenedTreasure":                 snapshotSaved,
}

// SnapshotTreasure 是 Treasures 的 JSON 形式；map 的鍵 Cell 不是字串。
type SnapshotTreasure struct {
	Cell     Cell     `json:"cell"`
	Treasure Treasure `json:"treasure"`
}

// StateSnapshot 是 State 的 saved 類欄位。指標陣列的 aliasing 不保存：Units 與
// Roster 是兩個獨立陣列，快照前先驗證沒有共用同一個 *Unit。
type StateSnapshot struct {
	Version int `json:"version"`

	W     int     `json:"w"`
	H     int     `json:"h"`
	Units []*Unit `json:"units"`
	// UnitsConstructorPending／RosterConstructorPending 保存未匯出的
	// nativeConstructorPending；Roster 的待命單位登場時仍須跑原生 constructor。
	UnitsConstructorPending  []bool `json:"units_constructor_pending"`
	RosterConstructorPending []bool `json:"roster_constructor_pending"`

	HasNativeMapSelectorCache  bool   `json:"has_native_map_selector_cache"`
	NativeMapSelectorKeys      []byte `json:"native_map_selector_keys"`
	NativeMapSelectorErrorText string `json:"native_map_selector_error"`

	NativeMapCycleState           fdicon.NativeMapSpriteCycleState `json:"native_map_cycle_state"`
	HasNativeMapCycleState        bool                             `json:"has_native_map_cycle_state"`
	NativeTerrainPhaseState       fdother.NativeTerrainPhaseState  `json:"native_terrain_phase_state"`
	HasNativeTerrainPhaseState    bool                             `json:"has_native_terrain_phase_state"`
	NativeTerrainFlipState        fdicon.NativeBinaryTickState     `json:"native_terrain_flip_state"`
	NativeUnitPixelShiftState     fdicon.NativeBinaryTickState     `json:"native_unit_pixel_shift_state"`
	HasNativeMapBinaryTimingState bool                             `json:"has_native_map_binary_timing_state"`
	NativeMapHUDState             NativeMapHUDRuntimeState         `json:"native_map_hud_state"`
	HasNativeMapHUDState          bool                             `json:"has_native_map_hud_state"`
	NativeMapViewState            NativeMapViewState               `json:"native_map_view_state"`
	HasNativeMapViewState         bool                             `json:"has_native_map_view_state"`
	NativeMapRangeMode            int                              `json:"native_map_range_mode"`
	HasNativeMapRangeModeState    bool                             `json:"has_native_map_range_mode_state"`

	Roster        []*Unit      `json:"roster"`
	PendingGroups map[int]bool `json:"pending_groups"`
	OwnDeploy     []Cell       `json:"own_deploy"`

	Turn                        int             `json:"turn"`
	NativeRoundCounter          int             `json:"native_round_counter"`
	Flags                       map[string]bool `json:"flags"`
	NativeEventState            [0x20]byte      `json:"native_event_state"`
	Cost                        []int           `json:"cost"`
	NativeCompositionEventBytes []byte          `json:"native_composition_event_bytes"`
	NativeMapEventGrid          []byte          `json:"native_map_event_grid"`
	HasNativeMapEventGrid       bool            `json:"has_native_map_event_grid"`
	NativeFieldControlRaw       []byte          `json:"native_field_control_raw"`

	NativeTurnEventControls        [16]NativeTurnEventControl `json:"native_turn_event_controls"`
	HasNativeTurnEventControlState bool                       `json:"has_native_turn_event_control_state"`
	NativeChestControls            [16]NativeChestControl     `json:"native_chest_controls"`
	HasNativeChestControlState     bool                       `json:"has_native_chest_control_state"`
	NativeFieldUnitControls        []NativeFieldUnitControl   `json:"native_field_unit_controls"`
	HasNativeFieldControlState     bool                       `json:"has_native_field_control_state"`
	NativeRuntimeRecords           []NativeRuntimeRecordState `json:"native_runtime_records"`
	HasNativeRuntimeUnitProjection bool                       `json:"has_native_runtime_unit_projection"`
	HasNativePendingGroupBinding   bool                       `json:"has_native_pending_group_binding"`

	NativeFieldEventSlots  []int              `json:"native_field_event_slots"`
	NativeFieldEvents      []NativeFieldEvent `json:"native_field_events"`
	NativeTileBlitModes    []byte             `json:"native_tile_blit_modes"`
	NativeTerrainControl   []byte             `json:"native_terrain_control"`
	NativeTerrainMoveCodes []byte             `json:"native_terrain_move_codes"`

	SpellBook      []Spell            `json:"spell_book"`
	AICommandSpell map[int]int        `json:"ai_command_spell"`
	Treasures      []SnapshotTreasure `json:"treasures"`
	OpenedTreasure map[int]bool       `json:"opened_treasure"`
}

// SnapshotReady 回報 transient 類欄位是否都是零值；不是就不能存快照。
func (s *State) SnapshotReady() error {
	if s == nil {
		return errors.New("battle snapshot: nil state")
	}
	if s.nativeAIScan != (nativeAIScanState{}) {
		return errors.New("battle snapshot: enemy AI scan is active")
	}
	if s.NativeExperienceAccumulator != 0 {
		return errors.New("battle snapshot: action experience is pending")
	}
	return nil
}

// Snapshot 深複製 saved 類欄位。輸出與 State 不共用任何可變記憶體。
func (s *State) Snapshot() (StateSnapshot, error) {
	if err := s.SnapshotReady(); err != nil {
		return StateSnapshot{}, err
	}
	seen := make(map[*Unit]bool, len(s.Units)+len(s.Roster))
	for _, list := range [][]*Unit{s.Units, s.Roster} {
		for _, u := range list {
			if u == nil {
				continue
			}
			if seen[u] {
				return StateSnapshot{}, errors.New("battle snapshot: unit pointer shared between arrays")
			}
			seen[u] = true
		}
	}
	out := StateSnapshot{
		Version: StateSnapshotVersion,
		W:       s.W, H: s.H,
		NativeMapCycleState: s.NativeMapCycleState, HasNativeMapCycleState: s.HasNativeMapCycleState,
		NativeTerrainPhaseState: s.NativeTerrainPhaseState, HasNativeTerrainPhaseState: s.HasNativeTerrainPhaseState,
		NativeTerrainFlipState: s.NativeTerrainFlipState, NativeUnitPixelShiftState: s.NativeUnitPixelShiftState,
		HasNativeMapBinaryTimingState: s.HasNativeMapBinaryTimingState,
		NativeMapHUDState:             s.NativeMapHUDState, HasNativeMapHUDState: s.HasNativeMapHUDState,
		NativeMapViewState: s.NativeMapViewState, HasNativeMapViewState: s.HasNativeMapViewState,
		NativeMapRangeMode: s.NativeMapRangeMode, HasNativeMapRangeModeState: s.HasNativeMapRangeModeState,
		Turn: s.Turn, NativeRoundCounter: s.NativeRoundCounter,
		NativeEventState:      s.NativeEventState,
		HasNativeMapEventGrid: s.HasNativeMapEventGrid,
		NativeTurnEventControls: s.NativeTurnEventControls, HasNativeTurnEventControlState: s.HasNativeTurnEventControlState,
		NativeChestControls: s.NativeChestControls, HasNativeChestControlState: s.HasNativeChestControlState,
		HasNativeFieldControlState:     s.HasNativeFieldControlState,
		HasNativeRuntimeUnitProjection: s.HasNativeRuntimeUnitProjection,
		HasNativePendingGroupBinding:   s.HasNativePendingGroupBinding,
	}
	out.Units, out.UnitsConstructorPending = snapshotUnits(s.Units)
	out.Roster, out.RosterConstructorPending = snapshotUnits(s.Roster)
	if s.NativeMapSelectorCache != nil {
		out.HasNativeMapSelectorCache = true
		out.NativeMapSelectorKeys = s.NativeMapSelectorCache.Keys()
	}
	if s.NativeMapSelectorError != nil {
		out.NativeMapSelectorErrorText = s.NativeMapSelectorError.Error()
		if out.NativeMapSelectorErrorText == "" {
			out.NativeMapSelectorErrorText = "native map selector rejected"
		}
	}
	out.PendingGroups = cloneMap(s.PendingGroups)
	out.OwnDeploy = cloneSlice(s.OwnDeploy)
	out.Flags = cloneMap(s.Flags)
	out.Cost = cloneSlice(s.Cost)
	out.NativeCompositionEventBytes = cloneSlice(s.NativeCompositionEventBytes)
	out.NativeMapEventGrid = cloneSlice(s.NativeMapEventGrid)
	out.NativeFieldControlRaw = cloneSlice(s.NativeFieldControlRaw)
	out.NativeFieldUnitControls = cloneSlice(s.NativeFieldUnitControls)
	out.NativeRuntimeRecords = cloneSlice(s.NativeRuntimeRecords)
	out.NativeFieldEventSlots = cloneSlice(s.NativeFieldEventSlots)
	out.NativeFieldEvents = cloneSlice(s.NativeFieldEvents)
	out.NativeTileBlitModes = cloneSlice(s.NativeTileBlitModes)
	out.NativeTerrainControl = cloneSlice(s.NativeTerrainControl)
	out.NativeTerrainMoveCodes = cloneSlice(s.NativeTerrainMoveCodes)
	out.SpellBook = cloneSlice(s.SpellBook)
	out.AICommandSpell = cloneMap(s.AICommandSpell)
	out.OpenedTreasure = cloneMap(s.OpenedTreasure)
	if s.Treasures != nil {
		out.Treasures = make([]SnapshotTreasure, 0, len(s.Treasures))
	}
	for cell, treasure := range s.Treasures {
		out.Treasures = append(out.Treasures, SnapshotTreasure{Cell: cell, Treasure: treasure})
	}
	sort.Slice(out.Treasures, func(i, j int) bool {
		a, b := out.Treasures[i].Cell, out.Treasures[j].Cell
		return a.Y < b.Y || (a.Y == b.Y && a.X < b.X)
	})
	return out, nil
}

// RestoreSnapshot 把 saved 類欄位寫到一份由同一章資產載入、且已完成應用層綁定的
// State。驗證全部通過才寫入；失敗時 s 不變。
func (s *State) RestoreSnapshot(snap StateSnapshot) error {
	if s == nil {
		return errors.New("battle snapshot: nil state")
	}
	if snap.Version != StateSnapshotVersion {
		return fmt.Errorf("battle snapshot: version %d, want %d", snap.Version, StateSnapshotVersion)
	}
	if snap.W <= 0 || snap.H <= 0 {
		return errors.New("battle snapshot: invalid field size")
	}
	if len(snap.Cost) != 0 && len(snap.Cost) != snap.W*snap.H {
		return errors.New("battle snapshot: cost size mismatch")
	}
	if len(snap.UnitsConstructorPending) != 0 && len(snap.UnitsConstructorPending) != len(snap.Units) ||
		len(snap.RosterConstructorPending) != 0 && len(snap.RosterConstructorPending) != len(snap.Roster) {
		return errors.New("battle snapshot: constructor flag count mismatch")
	}
	var cache *fdicon.NativeSelectorCache
	if snap.HasNativeMapSelectorCache {
		var err error
		if cache, err = fdicon.NewNativeSelectorCacheFromKeys(snap.NativeMapSelectorKeys); err != nil {
			return fmt.Errorf("battle snapshot: %w", err)
		}
	}
	treasures := make(map[Cell]Treasure, len(snap.Treasures))
	for _, t := range snap.Treasures {
		if _, dup := treasures[t.Cell]; dup {
			return errors.New("battle snapshot: duplicate treasure cell")
		}
		treasures[t.Cell] = t.Treasure
	}
	if snap.Treasures == nil {
		treasures = nil
	}

	s.W, s.H = snap.W, snap.H
	s.Units = restoreUnits(snap.Units, snap.UnitsConstructorPending)
	s.Roster = restoreUnits(snap.Roster, snap.RosterConstructorPending)
	s.NativeMapSelectorCache = cache
	s.NativeMapSelectorError = nil
	if snap.NativeMapSelectorErrorText != "" {
		s.NativeMapSelectorError = errors.New(snap.NativeMapSelectorErrorText)
	}
	s.NativeMapCycleState, s.HasNativeMapCycleState = snap.NativeMapCycleState, snap.HasNativeMapCycleState
	s.NativeTerrainPhaseState, s.HasNativeTerrainPhaseState = snap.NativeTerrainPhaseState, snap.HasNativeTerrainPhaseState
	s.NativeTerrainFlipState, s.NativeUnitPixelShiftState = snap.NativeTerrainFlipState, snap.NativeUnitPixelShiftState
	s.HasNativeMapBinaryTimingState = snap.HasNativeMapBinaryTimingState
	s.NativeMapHUDState, s.HasNativeMapHUDState = snap.NativeMapHUDState, snap.HasNativeMapHUDState
	s.NativeMapViewState, s.HasNativeMapViewState = snap.NativeMapViewState, snap.HasNativeMapViewState
	s.NativeMapRangeMode, s.HasNativeMapRangeModeState = snap.NativeMapRangeMode, snap.HasNativeMapRangeModeState
	s.PendingGroups = cloneMap(snap.PendingGroups)
	s.OwnDeploy = cloneSlice(snap.OwnDeploy)
	s.Turn, s.NativeRoundCounter = snap.Turn, snap.NativeRoundCounter
	s.Flags = cloneMap(snap.Flags)
	s.NativeEventState = snap.NativeEventState
	s.Cost = cloneSlice(snap.Cost)
	s.NativeCompositionEventBytes = cloneSlice(snap.NativeCompositionEventBytes)
	s.NativeMapEventGrid = cloneSlice(snap.NativeMapEventGrid)
	s.HasNativeMapEventGrid = snap.HasNativeMapEventGrid
	s.NativeFieldControlRaw = cloneSlice(snap.NativeFieldControlRaw)
	s.NativeTurnEventControls, s.HasNativeTurnEventControlState = snap.NativeTurnEventControls, snap.HasNativeTurnEventControlState
	s.NativeChestControls, s.HasNativeChestControlState = snap.NativeChestControls, snap.HasNativeChestControlState
	s.NativeFieldUnitControls = cloneSlice(snap.NativeFieldUnitControls)
	s.HasNativeFieldControlState = snap.HasNativeFieldControlState
	s.NativeRuntimeRecords = cloneSlice(snap.NativeRuntimeRecords)
	s.HasNativeRuntimeUnitProjection = snap.HasNativeRuntimeUnitProjection
	s.HasNativePendingGroupBinding = snap.HasNativePendingGroupBinding
	s.NativeFieldEventSlots = cloneSlice(snap.NativeFieldEventSlots)
	s.NativeFieldEvents = cloneSlice(snap.NativeFieldEvents)
	s.NativeTileBlitModes = cloneSlice(snap.NativeTileBlitModes)
	s.NativeTerrainControl = cloneSlice(snap.NativeTerrainControl)
	s.NativeTerrainMoveCodes = cloneSlice(snap.NativeTerrainMoveCodes)
	s.SpellBook = cloneSlice(snap.SpellBook)
	s.AICommandSpell = cloneMap(snap.AICommandSpell)
	s.Treasures = treasures
	s.OpenedTreasure = cloneMap(snap.OpenedTreasure)
	s.nativeAIScan = nativeAIScanState{}
	s.NativeExperienceAccumulator = 0
	return nil
}

func snapshotUnits(list []*Unit) ([]*Unit, []bool) {
	if list == nil {
		return nil, nil
	}
	out := make([]*Unit, len(list))
	var pending []bool
	for i, u := range list {
		if u == nil {
			continue
		}
		c := *u
		out[i] = &c
		if u.nativeConstructorPending {
			if pending == nil {
				pending = make([]bool, len(list))
			}
			pending[i] = true
		}
	}
	return out, pending
}

func restoreUnits(list []*Unit, pending []bool) []*Unit {
	if list == nil {
		return nil
	}
	out := make([]*Unit, len(list))
	for i, u := range list {
		if u == nil {
			continue
		}
		c := *u
		c.nativeConstructorPending = len(pending) > i && pending[i]
		out[i] = &c
	}
	return out
}

func cloneMap[K comparable, V any](m map[K]V) map[K]V {
	if m == nil {
		return nil
	}
	out := make(map[K]V, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// cloneSlice 深複製並保留 nil 與空切片的差別；往返後要與原值逐欄相等。
func cloneSlice[T any](s []T) []T {
	if s == nil {
		return nil
	}
	return append(make([]T, 0, len(s)), s...)
}

// FiredEventRef 記錄一個已觸發事件的劇本索引與 ID；兩者都要相符才還原。
type FiredEventRef struct {
	Index int    `json:"index"`
	ID    string `json:"id"`
}

// FiredEvents 回傳已觸發的事件（依劇本順序）。未匯出的 fired 旗標是劇本的
// 執行期狀態，快照需要它才不會在讀檔後重播事件。
func (sc *Scenario) FiredEvents() []FiredEventRef {
	if sc == nil {
		return nil
	}
	var refs []FiredEventRef
	for i := range sc.Events {
		if sc.Events[i].fired {
			refs = append(refs, FiredEventRef{Index: i, ID: sc.Events[i].ID})
		}
	}
	return refs
}

// PendingJoinCount 回報尚未發布的 JOIN；快照受理時必須為零。
func (sc *Scenario) PendingJoinCount() int {
	if sc == nil {
		return 0
	}
	return len(sc.pendingJoins)
}

// RestoreFiredEvents 還原 fired；索引越界、ID 不符或重複時拒絕且不改變劇本。
func (sc *Scenario) RestoreFiredEvents(refs []FiredEventRef) error {
	if sc == nil {
		if len(refs) == 0 {
			return nil
		}
		return errors.New("battle snapshot: fired events without scenario")
	}
	want := make([]bool, len(sc.Events))
	for _, ref := range refs {
		if ref.Index < 0 || ref.Index >= len(sc.Events) || sc.Events[ref.Index].ID != ref.ID {
			return fmt.Errorf("battle snapshot: scenario event %d %q does not match", ref.Index, ref.ID)
		}
		if want[ref.Index] {
			return fmt.Errorf("battle snapshot: repeated scenario event %d", ref.Index)
		}
		want[ref.Index] = true
	}
	for i := range sc.Events {
		sc.Events[i].fired = want[i]
	}
	sc.pendingJoins = nil
	return nil
}
