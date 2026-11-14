package main

// R4 F 組（存讀檔與設定）的 Go→Rust 黃金資料輸出（docs/goal/120）。只在設定
// FD2_RUST_GOLDEN_SAVE_OUT 時執行，輸出 Go 寫出的存檔／設定／能力設定附檔位元組與讀回
// 結果，不改任何遊戲行為；由 tools/rust_golden_save.sh 驅動，Rust 端在
// remake-rs/crates/fd2-game/src/game/save/golden_save_test.rs 比對。
//
// 真實輸入：FD2_RUST_GOLDEN_SAVE_REAL 指向的目錄底下、FD2_RUST_GOLDEN_SAVE_REAL_FILES
// 列出（以冒號分隔、已排序）的 Go 版存檔與設定檔；位元組、大小與 SHA-256 寫進黃金檔。

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

const rustGoldenSaveSeed = 20261114

type rustGoldenSaveFile struct {
	Schema     string               `json:"schema"`
	GoVersion  string               `json:"go_version"`
	Seed       int64                `json:"seed"`
	RealInputs []rustGoldenRealFile `json:"real_inputs"`
	Plains     []string             `json:"plains"`
	Cases      []map[string]any     `json:"cases"`
}

type rustGoldenRealFile struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Size   int    `json:"size"`
	SHA256 string `json:"sha256"`
}

func rustGoldenB64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

func rustGoldenRaw(t *testing.T, v any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// rustGoldenIntPairs 把整數鍵 map 寫成依鍵升冪排序的 [鍵, 值] 陣列，nil 與空都是 []。
func rustGoldenIntPairs[V any](t *testing.T, m map[int]V) []any {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	out := []any{}
	for _, k := range keys {
		out = append(out, []any{k, rustGoldenRaw(t, m[k])})
	}
	return out
}

func rustGoldenIntPtr(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

// rustGoldenSaveView 是 saveData 讀回後逐欄的中立表示：nil 與空分開記，整數鍵 map 寫成
// 排序後的配對；battle 套件的型別（Unit、BalanceOptions、battleSnapshot）以 Go 自己的
// json.Marshal 表示。
func rustGoldenSaveView(t *testing.T, d saveData) map[string]any {
	return map[string]any{
		"balance":           rustGoldenRaw(t, d.Balance),
		"node":              d.Node,
		"flags_nil":         d.Flags == nil,
		"flags":             rustGoldenRaw(t, d.Flags),
		"gold":              d.Gold,
		"items_nil":         d.Items == nil,
		"items":             rustGoldenRaw(t, d.Items),
		"party_members":     rustGoldenIntPairs(t, d.PartyMembers),
		"party_join_order":  rustGoldenRaw(t, d.PartyJoinOrder),
		"party_deploy":      rustGoldenIntPairs(t, d.PartyDeploy),
		"party_roster":      rustGoldenIntPairs(t, d.PartyRoster),
		"chapter":           d.Chapter,
		"native_hud_gate_a": rustGoldenIntPtr(d.NativeHUDGateA),
		"native_raw_53af9":  rustGoldenIntPtr(d.NativeRaw53AF9),
		"native_raw_51e61":  rustGoldenIntPtr(d.NativeRaw51E61),
		"native_raw_51e62":  rustGoldenIntPtr(d.NativeRaw51E62),
		"battle_snapshot":   rustGoldenRaw(t, d.BattleSnapshot),
	}
}

func rustGoldenMarshalIndent(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// rustGoldenSaveCase 以 loadJSONGameFromSlot／nativeLoadSlotMetadata 同樣的方式讀入 raw：
// 先 json.Unmarshal 到零值 saveData，記下是否接受、讀回欄位、重新寫出的位元組，以及
// 拿掉 battle 套件子樹（balance 以外）後的外層位元組。
func rustGoldenSaveCase(t *testing.T, id, origin string, raw []byte) map[string]any {
	c := map[string]any{"kind": "save", "id": id, "origin": origin, "input": rustGoldenB64(raw)}
	var d saveData
	if err := json.Unmarshal(raw, &d); err != nil {
		c["accepted"] = false
		return c
	}
	c["accepted"] = true
	c["chapter_in_range"] = d.Chapter >= 0 && d.Chapter <= 0xff
	c["topology_errors"] = rustGoldenErrorSet(func() error { return validateSavePartyTopology(d) })
	c["balance_error"] = ""
	if err := d.Balance.Validate(); err != nil {
		c["balance_error"] = err.Error()
	}
	c["view"] = rustGoldenSaveView(t, d)
	c["rewritten"] = rustGoldenB64(rustGoldenMarshalIndent(t, d))
	outer := d
	outer.PartyRoster = nil
	outer.BattleSnapshot = nil
	c["outer"] = rustGoldenB64(rustGoldenMarshalIndent(t, outer))
	return c
}

// rustGoldenErrorSet 重跑 300 次收集不同的錯誤文字（"" 是成功）。Go 以 map 迭代順序檢查的
// 函式，在多筆違規時每次可能報不同的一筆；黃金記下全部可能，Rust 的結果必須是其中之一。
func rustGoldenErrorSet(run func() error) []string {
	seen := map[string]bool{}
	for i := 0; i < 300; i++ {
		msg := ""
		if err := run(); err != nil {
			msg = err.Error()
		}
		seen[msg] = true
	}
	out := make([]string, 0, len(seen))
	for msg := range seen {
		out = append(out, msg)
	}
	sort.Strings(out)
	return out
}

func rustGoldenSettingsView(s settings) map[string]any {
	var poster any
	if s.Poster != nil {
		poster = map[string]any{"set": s.Poster.Set, "edge": s.Poster.Edge, "fade_ms": s.Poster.FadeMS}
	}
	return map[string]any{
		"bgm_source": s.BGMSource, "locale_id": s.LocaleID, "font_scale": s.FontScale,
		"interface_mode": s.InterfaceMode, "theme_id": s.ThemeID, "game_speed": s.GameSpeed,
		"battle_animation_speed": s.BattleAnimationSpeed, "hd_texture_quality": s.HDTextureQuality,
		"poster": poster,
	}
}

// rustGoldenSettingsCase 記錄三件事：零值起點的 json.Unmarshal（設定檔的格式本身）、
// loadSettings 起點上的 json.Unmarshal（正規化前），以及 loadSettings 的最終結果。
func rustGoldenSettingsCase(t *testing.T, dir, id, origin string, raw []byte) map[string]any {
	c := map[string]any{"kind": "settings", "id": id, "origin": origin, "input": rustGoldenB64(raw)}
	var zero settings
	if err := json.Unmarshal(raw, &zero); err != nil {
		c["accepted"] = false
		_, isSyntax := err.(*json.SyntaxError)
		c["syntax_error"] = isSyntax
	} else {
		c["accepted"] = true
		c["view"] = rustGoldenSettingsView(zero)
		c["rewritten"] = rustGoldenB64(rustGoldenMarshalIndent(t, zero))
	}
	base := settings{BGMSource: "fm", LocaleID: "zh-Hant", FontScale: defaultFontScale}
	_ = json.Unmarshal(raw, &base)
	c["base_view"] = rustGoldenSettingsView(base)
	userDataDirCached = dir
	if err := os.WriteFile(settingsPath(), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	c["loaded_view"] = rustGoldenSettingsView(loadSettings())
	return c
}

// rustGoldenSidecarErrorKind：本套件自己的錯誤要逐字比，JSON 解碼與 os 的錯誤只比失敗。
func rustGoldenSidecarErrorKind(msg string) string {
	switch msg {
	case "能力設定附檔過大", "能力設定附檔版本或筆數無效", "敵軍基準缺少倍率", "敵軍基準無效",
		"能力設定倍率超出範圍", "成長設定角色或倍率無效", "命中設定角色無效", "招式沒有可調命中骰或命中率無效":
		return "contract"
	}
	return "decode"
}

func rustGoldenSidecarView(t *testing.T, d balanceSidecar) map[string]any {
	keys := make([]string, 0, len(d.Entries))
	for k := range d.Entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	entries := []any{}
	for _, k := range keys {
		v := d.Entries[k]
		entries = append(entries, []any{k, map[string]any{
			"options": rustGoldenRaw(t, v.Options), "enemy_bases": rustGoldenIntPairs(t, v.EnemyBases),
		}})
	}
	return map[string]any{"version": d.Version, "entries_nil": d.Entries == nil, "entries": entries}
}

func rustGoldenSidecarCase(t *testing.T, dir, id string, raw []byte, absent bool, pad int) map[string]any {
	c := map[string]any{"kind": "sidecar", "id": id, "absent": absent, "pad_spaces": pad}
	path := filepath.Join(dir, id+"-FD2.SAV")
	if !absent {
		c["input"] = rustGoldenB64(raw)
		full := append(append([]byte(nil), raw...), bytes.Repeat([]byte(" "), pad)...)
		if err := os.WriteFile(path+".fd2-options.json", full, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	errs := rustGoldenErrorSet(func() error { _, err := readBalanceSidecar(path); return err })
	kinds := make([]string, len(errs))
	for i, msg := range errs {
		kinds[i] = rustGoldenSidecarErrorKind(msg)
	}
	c["errors"], c["error_kinds"] = errs, kinds
	d, err := readBalanceSidecar(path)
	if err != nil {
		return c
	}
	c["view"] = rustGoldenSidecarView(t, d)
	c["rewritten"] = rustGoldenB64(rustGoldenMarshalIndent(t, d))
	return c
}

func rustGoldenSyntheticUnit(r *rand.Rand, fig int) battle.Unit {
	names := []string{"索爾", "悠妮", "哈瓦特", "Sol \"the\" \\ knight", ""}
	u := battle.Unit{Fig: fig, Name: names[r.Intn(len(names))], Lv: 1 + r.Intn(40), HP: r.Intn(200), MaxHP: 1 + r.Intn(300),
		MP: r.Intn(80), MaxMP: r.Intn(100), MV: r.Intn(8), ClassID: r.Intn(40), Portrait: r.Intn(0x60)}
	switch r.Intn(3) {
	case 0:
		u.Exp = float64(r.Intn(100))
	case 1:
		u.Exp = float64(r.Intn(1000)) / 8
	}
	if r.Intn(2) == 0 {
		u.Spells = []int{r.Intn(40), r.Intn(40)}
	}
	if r.Intn(2) == 0 {
		u.Inventory = []int{0xc6, r.Intn(0x100)}
		u.Equipped = []bool{true, false}
	}
	return u
}

func rustGoldenBalance(r *rand.Rand) *battle.BalanceOptions {
	switch r.Intn(4) {
	case 0:
		return nil
	case 1:
		return battle.NewBalanceOptions()
	}
	b := battle.NewBalanceOptions()
	b.EnemyPercent = []int{50, 100, 200, 500, 5}[r.Intn(5)]
	b.Growth.HP = r.Intn(501)
	if r.Intn(2) == 0 {
		b.Characters = map[int]battle.GrowthRates{}
		for _, id := range []int{0, 2, 9, 10, 12, 31} {
			if r.Intn(2) == 0 {
				b.Characters[id] = battle.GrowthRates{HP: r.Intn(501), MP: 100, AP: r.Intn(501), DP: 100, DX: 0}
			}
		}
	}
	if r.Intn(2) == 0 {
		b.SkillHit = map[int]map[int]int{}
		for _, id := range []int{1, 11, 3} {
			b.SkillHit[id] = map[int]int{0: r.Intn(101), 10: r.Intn(101), 2: r.Intn(101), 32: 100}
		}
	}
	return b
}

func rustGoldenIntPtrChoice(r *rand.Rand) *int {
	switch r.Intn(5) {
	case 0:
		return nil
	case 4:
		v := []int{2, -1, 256}[r.Intn(3)]
		return &v
	}
	v := r.Intn(2)
	return &v
}

func rustGoldenRandomSave(r *rand.Rand, units []battle.Unit) saveData {
	nodes := []string{"town_ch02", "hotel_ch02", "preparation_ch03", "story_ch01", "battle_ch01", "", "節點"}
	d := saveData{Node: nodes[r.Intn(len(nodes))], Gold: []int{0, 1, 279, 1000, -5, 1 << 40}[r.Intn(6)]}
	switch r.Intn(3) {
	case 1:
		d.Flags = map[string]bool{}
	case 2:
		d.Flags = map[string]bool{}
		for i := 0; i < 1+r.Intn(12); i++ {
			d.Flags[fmt.Sprintf("flag_%d", r.Intn(40))] = r.Intn(2) == 0
		}
	}
	switch r.Intn(3) {
	case 1:
		d.Items = []string{}
	case 2:
		d.Items = []string{"sky-key", "天空之鑰"}[:1+r.Intn(2)]
	}
	if r.Intn(4) != 0 {
		d.PartyMembers, d.PartyDeploy, d.PartyRoster = map[int]bool{}, map[int]bool{}, map[int]battle.Unit{}
		for _, id := range r.Perm(32)[:1+r.Intn(14)] {
			d.PartyMembers[id] = r.Intn(6) != 0
			d.PartyJoinOrder = append(d.PartyJoinOrder, id)
			if r.Intn(2) == 0 {
				d.PartyDeploy[id] = r.Intn(3) != 0
			}
			if r.Intn(3) != 0 {
				if len(units) > 0 && r.Intn(2) == 0 {
					u := units[r.Intn(len(units))]
					d.PartyRoster[id] = u
				} else {
					d.PartyRoster[id] = rustGoldenSyntheticUnit(r, id)
				}
			}
		}
	}
	d.Chapter = []int{0, 0, 1, 2, 7, 30, 255, -1, 256}[r.Intn(9)]
	d.NativeHUDGateA = rustGoldenIntPtrChoice(r)
	d.NativeRaw53AF9 = rustGoldenIntPtrChoice(r)
	d.NativeRaw51E61 = rustGoldenIntPtrChoice(r)
	d.NativeRaw51E62 = rustGoldenIntPtrChoice(r)
	d.Balance = rustGoldenBalance(r)
	return d
}

func rustGoldenRandomSettings(r *rand.Rand) settings {
	s := settings{
		BGMSource: []string{"", "fm", "mt32", "opl"}[r.Intn(4)],
		LocaleID:  []string{"", "zh-Hant", "zh-Hans", "ja", "en", "xx"}[r.Intn(6)],
		FontScale: []float64{0, 0.8, 0.9, 1, 1.1, 1.25, 0.85, 3, -1, 1.0 / 3}[r.Intn(10)],
	}
	if r.Intn(2) == 0 {
		s.InterfaceMode = []string{"original", "hd", "bad"}[r.Intn(3)]
	}
	if r.Intn(2) == 0 {
		s.ThemeID = []string{"faithful", modernHandpaintedThemeID, "other"}[r.Intn(3)]
	}
	s.GameSpeed = []int{0, 0, 1, 2, 3, 4, 7, -1}[r.Intn(8)]
	s.BattleAnimationSpeed = []int{0, 0, 1, 2, 3, 5, -2}[r.Intn(7)]
	s.HDTextureQuality = []string{"", "full", "balanced", "light", "ultra"}[r.Intn(5)]
	if r.Intn(2) == 0 {
		p := playerPosterSettings{Set: []string{"off", "hd", "classic", "x", ""}[r.Intn(5)],
			Edge: []int{0, 48, 96, 97, -1}[r.Intn(5)], FadeMS: []int{0, 300, 600, 1200, 5}[r.Intn(5)]}
		s.Poster = &p
	}
	return s
}

func TestRustGoldenSave(t *testing.T) {
	outPath := os.Getenv("FD2_RUST_GOLDEN_SAVE_OUT")
	if outPath == "" {
		t.Skip("FD2_RUST_GOLDEN_SAVE_OUT 未設定")
	}
	oldCache := userDataDirCached
	t.Cleanup(func() { userDataDirCached = oldCache })
	r := rand.New(rand.NewSource(rustGoldenSaveSeed))
	out := rustGoldenSaveFile{Schema: "fd2-rust-golden-save-v1", GoVersion: runtime.Version(), Seed: rustGoldenSaveSeed}
	add := func(c map[string]any) { out.Cases = append(out.Cases, c) }

	// 真實持續隊伍單位：JOIN 建構器實際建立的紀錄（同 TestStartNativeSlot…）。
	join, err := campaign.LoadNativeJoinConstructorTable(assetPath("assets/data/native_join_constructor.json"))
	if err != nil {
		t.Fatal(err)
	}
	itemRows, err := battle.LoadNativeItemEffectRowPrefix(assetPath("assets/data/native_item_effect_rows.json"))
	if err != nil {
		t.Fatal(err)
	}
	var units []battle.Unit
	for id := 0; id < 40; id++ {
		if u, err := join.MaterializePersistentUnit(id, battle.Unit{Fig: id}, itemRows); err == nil {
			units = append(units, u)
		}
	}
	if len(units) == 0 {
		t.Fatal("沒有可用的 JOIN 建構列")
	}

	// 1. Go 寫出的 saveData：固定邊界值與固定種子的隨機組合。
	gateZero, gateOne := 0, 1
	fixed := []saveData{
		{},
		{Node: "town_ch02", Flags: map[string]bool{}, Items: []string{}},
		{Node: "story_ch02", Flags: map[string]bool{"won_ch01": true}, Gold: 321,
			PartyMembers: map[int]bool{0: true, 9: true}, PartyJoinOrder: []int{0, 9},
			PartyDeploy: map[int]bool{0: true, 9: false},
			PartyRoster: map[int]battle.Unit{9: {Fig: 9, Name: "悠妮", Lv: 4, HP: 23, MaxHP: 37, MP: 18, MaxMP: 24, Exp: 67.5,
				Spells: []int{0, 4, 13}, Inventory: []int{0xc6, 0x64}}},
			Chapter: 1, NativeHUDGateA: &gateZero},
		{Node: "town", Gold: 99, NativeHUDGateA: &gateOne, NativeRaw53AF9: &gateZero, NativeRaw51E61: &gateOne, NativeRaw51E62: &gateOne,
			Chapter: 255},
	}
	for i, d := range fixed {
		add(rustGoldenSaveCase(t, fmt.Sprintf("save-fixed-%02d", i), "go-marshal", rustGoldenMarshalIndent(t, d)))
	}
	for i := 0; i < 160; i++ {
		d := rustGoldenRandomSave(r, units)
		add(rustGoldenSaveCase(t, fmt.Sprintf("save-random-%03d", i), "go-marshal", rustGoldenMarshalIndent(t, d)))
	}

	// 2. 手寫輸入：讀取端的 nil／空、未知欄位、整數鍵順序、損壞 JSON。
	handwritten := []string{
		`{}`, `null`, `[]`, `{"chapter":0}`, `{"chapter":7,"node":"town_ch08"}`,
		`{"node":"x","flags":null,"items":[]}`, `{"node":"x","flags":{},"items":null}`,
		`{"party_members":{"10":true,"2":false,"0":true},"party_join_order":[0,10],"party_deploy":{"10":true}}`,
		`{"party_members":null,"party_deploy":{},"party_roster":null}`,
		`{"node":"town","unknown_field":[1,2,3],"gold":12}`, `{"node":"town","gold":12}   `,
		`{"node":"town"} trailing`, `{"node":`, `{"gold":"12"}`, `{"gold":1.5}`,
		`{"native_hud_gate_a":0,"native_raw_53af9":1,"native_raw_51e61":0,"native_raw_51e62":1}`,
		`{"native_hud_gate_a":null}`, `{"balance":null,"battle_snapshot":null}`,
		`{"balance":{"growth":{"HP":100,"MP":100,"AP":100,"DP":100,"DX":100},"enemy_percent":200}}`,
		`{"balance":{"growth":{"HP":100,"MP":100,"AP":100,"DP":100,"DX":100},"enemy_percent":5}}`,
		`{"balance":{"growth":{"HP":100,"MP":100,"AP":100,"DP":100,"DX":100},"characters":{"12":{"HP":1,"MP":2,"AP":3,"DP":4,"DX":5},"3":{"HP":0,"MP":0,"AP":0,"DP":0,"DX":0}},"skill_hit":{"4":{"11":50,"2":7}},"enemy_percent":100}}`,
		`{"party_roster":{"3":{"Name":"索爾","Fig":3,"Exp":12.25,"Spells":null,"Inventory":[]}}}`,
		`{"node":"漢字\n\"q\""}`,
	}
	for i, raw := range handwritten {
		add(rustGoldenSaveCase(t, fmt.Sprintf("save-hand-%02d", i), "handwritten", []byte(raw)))
	}

	// 3. 正式流程：第一章戰鬥中 F5 存檔（附戰場快照）與酒店存檔前的節點邊界。
	battleDir := t.TempDir()
	userDataDirCached = battleDir
	g := loadChapterOneStoryBattle(t)
	userDataDirCached = battleDir
	if err := g.saveGameToSlot(1); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(saveSlotPath(1))
	if err != nil {
		t.Fatal(err)
	}
	add(rustGoldenSaveCase(t, "save-flow-ch01-battle-turn1", "go-flow", raw))
	playOneOwnUnit(t, g)
	userDataDirCached = battleDir
	if err := g.saveGameToSlot(2); err != nil {
		t.Fatal(err)
	}
	if raw, err = os.ReadFile(saveSlotPath(2)); err != nil {
		t.Fatal(err)
	}
	add(rustGoldenSaveCase(t, "save-flow-ch01-battle-after-move", "go-flow", raw))

	// 4. 設定檔。
	settingsDir := t.TempDir()
	settingsFixed := []settings{
		{}, {BGMSource: "fm", LocaleID: "zh-Hant", FontScale: 1},
		{BGMSource: "mt32", LocaleID: "ja", FontScale: 0.8, InterfaceMode: "hd", ThemeID: modernHandpaintedThemeID, GameSpeed: 1,
			BattleAnimationSpeed: 2, HDTextureQuality: "light", Poster: &playerPosterSettings{Set: "hd", Edge: 48, FadeMS: 600}},
		{FontScale: 1.25, Poster: &playerPosterSettings{}},
	}
	for i, s := range settingsFixed {
		add(rustGoldenSettingsCase(t, settingsDir, fmt.Sprintf("settings-fixed-%02d", i), "go-marshal", rustGoldenMarshalIndent(t, s)))
	}
	for i := 0; i < 60; i++ {
		s := rustGoldenRandomSettings(r)
		add(rustGoldenSettingsCase(t, settingsDir, fmt.Sprintf("settings-random-%02d", i), "go-marshal", rustGoldenMarshalIndent(t, s)))
	}
	settingsHand := []string{
		`{}`, `null`, `{"locale_id":"xx"}`, `{"bgm_source":"mt32"}`, `{"bgm_source":null,"font_scale":null,"poster":null}`,
		`{"font_scale":0.95}`, `{"font_scale":1e-300}`, `{"locale_id":"ja","locale_id":"en"}`,
		`{"poster":{"set":"hd","edge":100,"fade_ms":5}}`, `{"poster":{}}`, `{"bgm_source":"mt32"} x`, `[]`, `{"interface_mode":"original"`,
		`{"game_speed":3,"battle_animation_speed":4,"hd_texture_quality":"balanced"}`,
	}
	for i, raw := range settingsHand {
		add(rustGoldenSettingsCase(t, settingsDir, fmt.Sprintf("settings-hand-%02d", i), "handwritten", []byte(raw)))
	}

	// 5. 能力設定附檔：balanceSaveKey 與 readBalanceSidecar。
	plainRandom := make([]byte, fdsave.FileSize)
	r.Read(plainRandom)
	plainZero := make([]byte, fdsave.FileSize)
	plainChanged := append([]byte(nil), plainRandom...)
	a, _, _ := fdsave.SlotBounds(1)
	plainChanged[a]++
	plains := [][]byte{plainRandom, plainZero, plainChanged, plainRandom[:5]}
	for _, p := range plains {
		out.Plains = append(out.Plains, rustGoldenB64(p))
	}
	for pi, p := range plains {
		for slot := -2; slot <= 4; slot++ {
			c := map[string]any{"kind": "balance_key", "id": fmt.Sprintf("key-%d-%d", pi, slot), "plain": pi, "slot": slot}
			key, err := balanceSaveKey(p, slot)
			c["key"], c["error"] = key, ""
			if err != nil {
				c["error"] = err.Error()
			}
			add(c)
		}
	}
	sidecarDir := t.TempDir()
	keyOf := func(p []byte, slot int) string {
		k, err := balanceSaveKey(p, slot)
		if err != nil {
			t.Fatal(err)
		}
		return k
	}
	sidecars := []balanceSidecar{
		{Version: 1, Entries: map[string]balanceSavedState{}},
		{Version: 1, Entries: map[string]balanceSavedState{keyOf(plainRandom, 0): {Options: battle.NewBalanceOptions()}}},
	}
	for i := 0; i < 30; i++ {
		d := balanceSidecar{Version: 1, Entries: map[string]balanceSavedState{}}
		for j := 0; j < r.Intn(4); j++ {
			state := balanceSavedState{Options: rustGoldenBalance(r)}
			if r.Intn(2) == 0 {
				state.EnemyBases = map[int]battle.BalanceStats{}
				for _, idx := range r.Perm(96)[:1+r.Intn(14)] {
					state.EnemyBases[idx] = battle.BalanceStats{HP: 1 + r.Intn(32767), MP: r.Intn(100), AP: r.Intn(500), DP: r.Intn(500), DX: r.Intn(100)}
				}
			}
			d.Entries[keyOf(plains[r.Intn(3)], r.Intn(5)-1)] = state
		}
		sidecars = append(sidecars, d)
	}
	for i, d := range sidecars {
		add(rustGoldenSidecarCase(t, sidecarDir, fmt.Sprintf("sidecar-go-%02d", i), rustGoldenMarshalIndent(t, d), false, 0))
	}
	tooMany := balanceSidecar{Version: 1, Entries: map[string]balanceSavedState{}}
	for i := 0; i < 11; i++ {
		tooMany.Entries[fmt.Sprintf("slot%d:%02d", i%4, i)] = balanceSavedState{}
	}
	sidecarHand := []struct {
		raw    string
		absent bool
		pad    int
	}{
		{absent: true}, {raw: `{}`}, {raw: `{"version":2,"entries":{}}`}, {raw: `{"entries":null}`},
		{raw: `{"version":1,"entries":{"k":{"enemy_bases":{"3":{"HP":5,"MP":0,"AP":0,"DP":0,"DX":0}}}}}`},
		{raw: `{"version":1,"entries":{"k":{"options":{"growth":{"HP":100,"MP":100,"AP":100,"DP":100,"DX":100},"enemy_percent":200},"enemy_bases":{"96":{"HP":5}}}}}`},
		{raw: `{"version":1,"entries":{"k":{"options":{"growth":{"HP":100,"MP":100,"AP":100,"DP":100,"DX":100},"enemy_percent":200},"enemy_bases":{"0":{"HP":0}}}}}`},
		{raw: `{"version":1,"entries":{"k":{"options":{"growth":{"HP":900,"MP":100,"AP":100,"DP":100,"DX":100},"enemy_percent":200}}}}`},
		{raw: `{"version":1,"entries":{}} trailing garbage`}, {raw: ``}, {raw: `{"version":`}, {raw: `{"version":1,"entries":{}}`, pad: 1024 * 1024},
		{raw: string(rustGoldenMarshalIndent(t, tooMany))},
	}
	for i, h := range sidecarHand {
		add(rustGoldenSidecarCase(t, sidecarDir, fmt.Sprintf("sidecar-hand-%02d", i), []byte(h.raw), h.absent, h.pad))
	}

	// 6. 真實 Go 版檔案（work/ 底下的對拍輸出與玩家資料夾複本）。
	if dir := os.Getenv("FD2_RUST_GOLDEN_SAVE_REAL"); dir != "" {
		names := strings.Split(os.Getenv("FD2_RUST_GOLDEN_SAVE_REAL_FILES"), ":")
		if !sort.StringsAreSorted(names) {
			t.Fatal("FD2_RUST_GOLDEN_SAVE_REAL_FILES 必須已排序")
		}
		for _, name := range names {
			raw, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(raw)
			kind := "save"
			if strings.HasSuffix(name, "fd2_settings.json") {
				kind = "settings"
			}
			out.RealInputs = append(out.RealInputs, rustGoldenRealFile{Name: name, Kind: kind, Size: len(raw), SHA256: hex.EncodeToString(sum[:])})
			if kind == "settings" {
				add(rustGoldenSettingsCase(t, settingsDir, "real-"+name, "real", raw))
			} else {
				add(rustGoldenSaveCase(t, "real-"+name, "real", raw))
			}
		}
	}

	raw, err = json.MarshalIndent(out, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outPath, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("寫出 %d 個案例、%d 個真實檔到 %s", len(out.Cases), len(out.RealInputs), outPath)
}
