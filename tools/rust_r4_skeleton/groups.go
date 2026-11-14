package main

// 階段 2 分組建議：先依檔名把專用檔歸到 120 預估的組別（敵方回合併入戰場操作，共六組），main.go、HD 檔與其他未對上的檔
// 再依呼叫關係（選定函式之間的呼叫邊，雙向計數）逐函式歸到連線最多的組。

import (
	"sort"
	"strings"
)

type groupDef struct {
	ID       string
	Title    string
	prefixes []string
}

var groupDefs = []groupDef{
	{"A", "主迴圈、啟動、標題與開場", []string{"title", "packaged_startup", "assets", "font", "locale", "audio", "run_log",
		"input_audit_trace", "separated_asset_path", "separated_ui_assets", "native_bios_clock", "player_"}},
	{"B", "劇情對話與事件演出", []string{"native_story_", "native_event_dialogue", "native_confirmation_mouth",
		"native_spawn_intro", "native_staging_present", "native_battle_staging_present", "native_field_event",
		"native_turn_event", "native_ch2", "native_palette_", "native_indexed_transition"}},
	// 120 預估的「敵方回合銜接」只有 13 個函式（native_ai_*、native_end_turn_recovery），併入 C
	{"C", "戰場操作、指令環與敵方回合", []string{"action_overlay", "native_player_", "native_map_", "native_battle_name",
		"native_battle_panel_locale", "native_item_panel_ui", "native_turn_staging", "native_system_info_",
		"battle_protection", "native_command_locale", "native_ai_", "native_end_turn_recovery"}},
	{"D", "攻擊演出與動畫指令播放", []string{"native_command", "native_physical_", "native_attack_assets",
		"native_action_death_present", "native_death_", "native_transient_", "native_defeat", "native_level_up_dialogue"}},
	{"E", "城鎮、商店、教會與整備", []string{"native_town_", "native_shop_", "native_church_ui", "native_class_ui",
		"native_hotel_ui", "native_preparation_"}},
	{"F", "存讀檔與設定", []string{"save", "settings", "balance_save", "native_current_save", "native_chapter_slot_save",
		"native_continue_handoff", "native_load_slots_ui"}},
}

// pinnedGroups：Ebiten 的主迴圈入口呼叫每一組的 step／draw，依呼叫邊會被拉到呼叫最多的組；固定歸 A。
var pinnedGroups = map[string]string{
	"main.go:Game.Update": "A",
	"main.go:Game.Draw":   "A",
	"main.go:Game.Layout": "A",
}

// player_ 開頭但屬於 HD／工具列的檔不走檔名規則。
func fileGroup(name string) string {
	if strings.HasPrefix(name, "main.go") || isHDFile(name) {
		return ""
	}
	stem := strings.TrimSuffix(name, ".go")
	best, bestLen := "", 0
	for _, gd := range groupDefs {
		for _, p := range gd.prefixes {
			if (stem == p || strings.HasPrefix(stem, p)) && len(p) > bestLen {
				best, bestLen = gd.ID, len(p)
			}
		}
	}
	return best
}

type groupStat struct {
	ID         string         `json:"id"`
	Title      string         `json:"title"`
	Files      []string       `json:"files"`
	Functions  int            `json:"functions"`
	MainGo     int            `json:"main_go_functions"`
	HD         int            `json:"hd_functions"`
	CallsInto  map[string]int `json:"calls_into"`
	CalledFrom map[string]int `json:"called_from"`
}

func assignGroups(s *scope) (map[*fnInfo]string, []*groupStat) {
	grp := map[*fnInfo]string{}
	for _, fi := range s.selected {
		if gID := fileGroup(fi.file.Name); gID != "" {
			grp[fi] = gID
		}
		if gID, ok := pinnedGroups[fi.file.Name+":"+fi.key]; ok {
			grp[fi] = gID
		}
	}
	// 無向鄰接（只算選定函式）
	adj := map[*fnInfo]map[*fnInfo]int{}
	add := func(a, b *fnInfo) {
		if adj[a] == nil {
			adj[a] = map[*fnInfo]int{}
		}
		adj[a][b]++
	}
	for _, fi := range s.selected {
		for c := range fi.calls {
			if c.sel && c != fi {
				add(fi, c)
				add(c, fi)
			}
		}
	}
	order := map[string]int{}
	for i, gd := range groupDefs {
		order[gd.ID] = i
	}
	for {
		// 找出「已分組鄰居最多」的未分組函式，逐一分配（決定性：權重、檔名、行號）
		var best *fnInfo
		bestG, bestW := "", 0
		for _, fi := range s.selected {
			if grp[fi] != "" {
				continue
			}
			w := map[string]int{}
			for n, c := range adj[fi] {
				if gID := grp[n]; gID != "" {
					w[gID] += c
				}
			}
			g, gw := "", 0
			for _, gd := range groupDefs {
				if w[gd.ID] > gw {
					g, gw = gd.ID, w[gd.ID]
				}
			}
			if gw > bestW || (gw == bestW && gw > 0 && best != nil && less(fi, best)) {
				best, bestG, bestW = fi, g, gw
			}
		}
		if best == nil {
			break
		}
		grp[best] = bestG
	}
	// 沒有任何呼叫邊連到已分組函式的（例如 Update／Draw 的入口）歸 A
	for _, fi := range s.selected {
		if grp[fi] == "" {
			grp[fi] = "A"
		}
	}
	stats := map[string]*groupStat{}
	for _, gd := range groupDefs {
		stats[gd.ID] = &groupStat{ID: gd.ID, Title: gd.Title, CallsInto: map[string]int{}, CalledFrom: map[string]int{}}
	}
	files := map[string]map[string]bool{}
	for _, fi := range s.selected {
		st := stats[grp[fi]]
		st.Functions++
		if fi.file.Name == "main.go" {
			st.MainGo++
		}
		if isHDFile(fi.file.Name) {
			st.HD++
		}
		if files[st.ID] == nil {
			files[st.ID] = map[string]bool{}
		}
		files[st.ID][fi.file.Name] = true
		for c := range fi.calls {
			if c.sel && grp[c] != grp[fi] {
				st.CallsInto[grp[c]]++
				stats[grp[c]].CalledFrom[grp[fi]]++
			}
		}
	}
	var out []*groupStat
	for _, gd := range groupDefs {
		st := stats[gd.ID]
		for f := range files[gd.ID] {
			st.Files = append(st.Files, f)
		}
		sort.Strings(st.Files)
		out = append(out, st)
	}
	_ = order
	return grp, out
}

func less(a, b *fnInfo) bool {
	if a.file.Name != b.file.Name {
		return a.file.Name < b.file.Name
	}
	return a.decl.Pos() < b.decl.Pos()
}
