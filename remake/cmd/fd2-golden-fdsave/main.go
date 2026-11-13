// fd2-golden-fdsave 輸出 internal/fdsave 的黃金差分資料，供 Rust 改寫逐欄比對
// （docs/goal/119-goal-rust-bevy-rewrite-20261113.md）。
//
// Go 版凍結於 v.1.1.11；本工具只讀 fdsave 的公開 API，不改任何遊戲行為。
// 輸入以固定種子的 PCG 產生，連同結果寫進同一份 JSON，Rust 端不需重現亂數產生器。
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"

	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

type contextJSON struct {
	Chapter            int  `json:"chapter"`
	FieldWidth         int  `json:"field_width"`
	FieldHeight        int  `json:"field_height"`
	SelectorGroupCount int  `json:"selector_group_count"`
	TitleTimerTick     int  `json:"title_timer_tick"`
	HasTitleTimerTick  bool `json:"has_title_timer_tick"`
}

type caseJSON struct {
	Index     int            `json:"index"`
	Kind      string         `json:"kind"`
	StoredHex string         `json:"stored_hex"`
	Context   contextJSON    `json:"context"`
	Result    map[string]any `json:"result"`
}

func ints(b []byte) []int {
	out := make([]int, len(b))
	for i, v := range b {
		out[i] = int(v)
	}
	return out
}

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func errString(err error) any {
	if err == nil {
		return nil
	}
	return err.Error()
}

func metaJSON(m fdsave.VerifiedMetadata) map[string]any {
	return map[string]any{
		"chapter": m.Chapter, "roster_count": m.RosterCount, "currency": m.Currency,
		"hud_gate_a": m.HUDGateA, "raw_53af9": m.Raw53AF9, "raw_51e61": m.Raw51E61, "raw_51e62": m.Raw51E62,
	}
}

func viewJSON(r fdsave.PersistentRecord) map[string]any {
	v := r.View()
	inventory := make([][]int, len(v.Inventory))
	for i, cell := range v.Inventory {
		inventory[i] = []int{int(cell.Flags), int(cell.ItemID)}
	}
	return map[string]any{
		"raw_byte5": v.RawByte5, "raw_camp": v.RawCamp, "raw_presentation_key": v.RawPresentationKey,
		"raw_identity": v.RawIdentity, "inventory": inventory, "command_mask": ints(v.CommandMask[:]),
		"race": v.Race, "class": v.Class, "level": v.Level, "transient": ints(v.Transient[:]),
		"raw_byte34": v.RawByte34, "raw_byte35": v.RawByte35, "raw_byte36": v.RawByte36,
		"base_ap": v.BaseAP, "base_dp": v.BaseDP, "movement": v.Movement, "experience": v.Experience,
		"dx": v.DX, "hp": v.HP, "max_hp": v.MaxHP, "mp": v.MP, "max_mp": v.MaxMP,
		"ap": v.AP, "dp": v.DP, "hit": v.HIT, "ev": v.EV,
	}
}

func recordsDigest(records []fdsave.PersistentRecord) string {
	var all []byte
	for _, r := range records {
		all = append(all, r.Raw[:]...)
	}
	return digest(all)
}

func viewsJSON(records []fdsave.PersistentRecord) []map[string]any {
	out := make([]map[string]any, len(records))
	for i, r := range records {
		out[i] = viewJSON(r)
	}
	return out
}

func headerJSON(h fdsave.CurrentRuntimeHeader) map[string]any {
	return map[string]any{
		"turn_counter": h.TurnCounter, "runtime_count": h.RuntimeCount, "chapter": h.Chapter,
		"camera_x": h.CameraX, "camera_y": h.CameraY, "cursor_x": h.CursorX, "cursor_y": h.CursorY,
		"visible_cursor_x": h.VisibleCursorX, "visible_cursor_y": h.VisibleCursorY,
		"persistent_count": h.PersistentCount, "currency": h.Currency, "raw_53af9": h.Raw53AF9,
		"hud_gate_a": h.HUDGateA, "raw_51e61": h.Raw51E61, "raw_51e62": h.Raw51E62,
	}
}

func continueJSON(snapshot fdsave.CurrentSnapshot, c contextJSON) map[string]any {
	input, err := fdsave.BuildContinueRuntimeInput(snapshot, fdsave.ContinueRuntimeContext{
		Chapter: c.Chapter, FieldWidth: c.FieldWidth, FieldHeight: c.FieldHeight,
		SelectorGroupCount: c.SelectorGroupCount, TitleTimerTick: c.TitleTimerTick,
		HasTitleTimerTick: c.HasTitleTimerTick,
	})
	if err != nil {
		return map[string]any{"err": err.Error()}
	}
	selector := make([][]int, len(input.RuntimeRecords))
	for i, r := range input.RuntimeRecords {
		selector[i] = []int{int(r.SelectorKey), int(r.SelectorSlot)}
	}
	fc := input.FieldControl
	turn := make([][]int, len(fc.TurnEvents))
	for i, e := range fc.TurnEvents {
		turn[i] = []int{int(e.Turn), int(e.EventID), int(e.RawCamp)}
	}
	field := make([][]int, len(fc.FieldEvents))
	for i, e := range fc.FieldEvents {
		field[i] = []int{int(e.EventID), int(e.Selector)}
	}
	chests := make([][]int, len(fc.Chests))
	for i, e := range fc.Chests {
		chests[i] = []int{int(e.RawType), int(e.Value)}
	}
	var units []byte
	for _, u := range fc.Units {
		units = append(units, u.Raw[:]...)
	}
	owners := make([]string, len(input.UnresolvedOwners))
	for i, o := range input.UnresolvedOwners {
		owners[i] = string(o)
	}
	mp, ts := input.MapPresentation, input.MapTimingSeed
	return map[string]any{
		"err":      nil,
		"selector": selector,
		"map_presentation": map[string]any{
			"opening_range_mode": mp.OpeningRangeMode, "interactive_range_mode": mp.InteractiveRangeMode,
			"hud_gate_b": mp.HUDGateB, "hud_anchor_x": mp.HUDAnchorX,
		},
		"timing": []int{ts.SpriteIdleCycle, ts.SpriteMovingCycle, ts.SpriteLastTimerTick, ts.TerrainPhase,
			ts.TerrainPhaseLastTimerTick, ts.TerrainPhaseOverride, ts.TerrainFlip, ts.TerrainFlipLastTimerTick,
			ts.UnitPixelShift, ts.UnitPixelShiftLastTimerTick},
		"field_control": map[string]any{
			"map_selector": fc.RawMapSelector, "own_deploy": fc.RawOwnDeployCount, "unit_count": fc.RawUnitCount,
			"turn_events": turn, "field_events": field, "chests": chests,
			"units_len": len(fc.Units), "units_sha256": digest(units),
		},
		"ready":     input.ReadyForContinue(),
		"owners":    owners,
		"validated": input.ValidatedForRuntimeBridge(),
	}
}

func summarize(index int, stored []byte, c contextJSON) map[string]any {
	out := map[string]any{}
	plain, err := fdsave.Decode(stored)
	out["decode_err"] = errString(err)
	if len(stored) != fdsave.FileSize {
		return out
	}
	if err != nil {
		plain = fdsave.XOREnvelope(stored)
	}
	out["plain_sha256"] = digest(plain)
	sum, _ := fdsave.Checksum(plain)
	out["checksum"] = sum
	encoded, _ := fdsave.Encode(plain)
	out["encode_sha256"] = digest(encoded)

	slots := []map[string]any{}
	for slot := 0; slot <= fdsave.SlotCount; slot++ {
		row := map[string]any{}
		start, end, err := fdsave.SlotBounds(slot)
		row["bounds_err"] = errString(err)
		row["bounds"] = []int{start, end}
		if meta, err := fdsave.ReadVerifiedMetadata(plain, slot); err != nil {
			row["meta_err"] = err.Error()
		} else {
			row["meta"] = metaJSON(meta)
		}
		if snap, err := fdsave.InspectChapterSlot(plain, slot); err != nil {
			row["inspect_err"] = err.Error()
		} else {
			row["inspect"] = map[string]any{
				"verified": metaJSON(snap.Verified), "metadata_hex": hex.EncodeToString(snap.Metadata[:]),
				"records_sha256": recordsDigest(snap.Records[:]), "source_sha256": snap.SourceSaveSHA256,
				"active_views": viewsJSON(snap.ActiveRecords()),
			}
		}
		slots = append(slots, row)
	}
	out["slots"] = slots

	snapshot, err := fdsave.InspectCurrentSnapshot(plain)
	if err != nil {
		out["current"] = map[string]any{"err": err.Error()}
	} else {
		out["current"] = map[string]any{
			"err":                         nil,
			"header":                      headerJSON(snapshot.Header),
			"native_field_control_sha256": digest(snapshot.NativeFieldControl[:]),
			"persistent_sha256":           recordsDigest(snapshot.PersistentRecords[:]),
			"runtime_sha256":              recordsDigest(snapshot.RuntimeRecords),
			"event_state_hex":             hex.EncodeToString(snapshot.NativeEventState[:]),
			"active_persistent_views":     viewsJSON(snapshot.ActivePersistentRecords()),
			"runtime_views":               viewsJSON(snapshot.RuntimeRecords),
		}
		out["continue"] = continueJSON(snapshot, c)
		written, err := fdsave.WriteCurrentSnapshot(plain, snapshot)
		out["write_current_err"] = errString(err)
		if err == nil {
			out["write_current_sha256"] = digest(written)
		}
		snapshot.Header.TurnCounter++
		snapshot.Header.Currency ^= 0x5a5a
		snapshot.NativeFieldControl[2] = byte(index % 40)
		written, err = fdsave.WriteCurrentSnapshot(plain, snapshot)
		out["write_current_mutated_err"] = errString(err)
		if err == nil {
			out["write_current_mutated_sha256"] = digest(written)
		}
	}

	source, _ := fdsave.ReadSlot(plain, (index+1)%fdsave.SlotCount)
	written, err := fdsave.WriteSlot(plain, index%fdsave.SlotCount, source)
	out["write_slot_err"] = errString(err)
	if err == nil {
		out["write_slot_sha256"] = digest(written)
	}
	return out
}

func generate(index int, rng *rand.Rand) (string, []byte, contextJSON) {
	kinds := []string{"random", "shaped", "shaped-header-flip", "shaped-slots"}
	kind := kinds[index%len(kinds)]
	plain := make([]byte, fdsave.FileSize)
	for i := range plain {
		plain[i] = byte(rng.Uint32())
	}
	c := contextJSON{
		Chapter: rng.IntN(32), FieldWidth: rng.IntN(300), FieldHeight: rng.IntN(300),
		SelectorGroupCount: rng.IntN(300) - 10, TitleTimerTick: rng.IntN(0x12000) - 0x9000,
		HasTitleTimerTick: rng.IntN(10) != 0,
	}
	if kind != "random" {
		width, height := 13+rng.IntN(30), 8+rng.IntN(20)
		camX, camY := rng.IntN(width-12), rng.IntN(height-7)
		curX := camX + rng.IntN(min(13, width-camX))
		curY := camY + rng.IntN(min(8, height-camY))
		rc := rng.IntN(9)
		if index%16 == 5 {
			rc = fdsave.RosterUnits*3 + 1
		}
		groups := 1 + rng.IntN(20)
		h := plain[fdsave.CurrentRuntimeHeaderOffset:]
		h[1], h[2] = byte(rc), byte(rng.IntN(32))
		h[3], h[4], h[5], h[6] = byte(camX), byte(camY), byte(curX), byte(curY)
		h[7], h[8], h[9] = byte(curX-camX), byte(curY-camY), byte(rng.IntN(34))
		plain[fdsave.CurrentFieldControlOffset+2] = byte(rng.IntN(fdsave.CurrentFieldControlUnitCap + 3))
		for r := 0; r < rc && r < fdsave.RosterUnits*3; r++ {
			raw := plain[fdsave.CurrentRuntimeRosterOffset+r*fdsave.UnitSize:]
			raw[7] = byte(rng.IntN(groups + 1))
			if raw[5]&1 == 0 && rng.IntN(4) != 0 {
				raw[0], raw[1] = byte(rng.IntN(width)), byte(rng.IntN(height))
				raw[3], raw[4] = byte(rng.IntN(4)), byte(rng.IntN(7))
			}
		}
		for slot := 0; slot < fdsave.SlotCount; slot++ {
			start, _, _ := fdsave.SlotBounds(slot)
			meta := plain[start+fdsave.RosterSize:]
			if rng.IntN(4) == 0 {
				meta[0] = 0xff
			} else {
				meta[0] = byte(rng.IntN(30))
			}
			meta[1] = byte(rng.IntN(35))
		}
		c = contextJSON{
			Chapter: int(h[2]), FieldWidth: width, FieldHeight: height, SelectorGroupCount: groups,
			TitleTimerTick: rng.IntN(0x12000) - 0x9000, HasTitleTimerTick: rng.IntN(10) != 0,
		}
		if rng.IntN(8) == 0 {
			c.Chapter++
		}
		if kind == "shaped-header-flip" {
			h[rng.IntN(fdsave.CurrentRuntimeHeaderSize)] ^= byte(1 << rng.IntN(8))
		}
	}
	stored, _ := fdsave.Encode(plain)
	if index%5 == 0 {
		stored[rng.IntN(len(stored))] ^= byte(1 << rng.IntN(8))
	}
	if index%37 == 0 {
		stored = stored[:len(stored)-1]
	}
	return kind, stored, c
}

func main() {
	out := flag.String("out", "", "輸出 JSON 路徑")
	cases := flag.Int("cases", 192, "隨機案例數")
	seed := flag.Uint64("seed", 20261113, "PCG 種子")
	var saves multiFlag
	flag.Var(&saves, "save", "額外的真實 FD2.SAV（可重複）")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "需要 -out")
		os.Exit(2)
	}
	rng := rand.New(rand.NewPCG(*seed, 0x46443253))
	var rows []caseJSON
	for i := 0; i < *cases; i++ {
		kind, stored, c := generate(i, rng)
		rows = append(rows, caseJSON{Index: i, Kind: kind, StoredHex: hex.EncodeToString(stored), Context: c,
			Result: summarize(i, stored, c)})
	}
	for _, path := range saves {
		stored, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		i := len(rows)
		c := contextJSON{Chapter: 0, FieldWidth: 30, FieldHeight: 20, SelectorGroupCount: 16, HasTitleTimerTick: true}
		rows = append(rows, caseJSON{Index: i, Kind: "real-save", StoredHex: hex.EncodeToString(stored), Context: c,
			Result: summarize(i, stored, c)})
	}
	doc := map[string]any{
		"schema": "fd2-golden-fdsave/1", "seed": *seed, "cases": rows,
	}
	data, err := json.Marshal(doc)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("fd2-golden-fdsave: %d cases sha256=%s\n", len(rows), digest(data))
}

type multiFlag []string

func (m *multiFlag) String() string     { return fmt.Sprint(*m) }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }
