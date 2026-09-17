// fd2-chapter-slot 由一份真實 FD2.SAV 的章節槽出發，套用後續各章戰後 handler 裡
// 已證實的持續隊伍寫入（sub_112A5 JOIN、0x1c220 grant_item、set_chapter），
// 加上明示的升級／金幣政策，建構「第 N 章已通關」的存檔槽。
//
// 它只寫已證實語意的欄位：JOIN 紀錄走 campaign.MaterializePersistentRecord，
// 升級走 0x1E292 的成長列（記錄 +7 選列）與 0x1B750 的裝備重算
// （battle.ApplyNativeEquipmentRecalc），其餘 bytes 全部保留基底存檔原值。
// 政策性欄位（每章升幾級、金幣）不是原版證據，manifest 逐項標成 assumption。
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

const (
	recordCamp        = 6
	recordKey         = 7
	recordIdentity    = 8
	recordInventory   = 0x0a
	recordCommandMask = 0x1a
	recordLevel       = 0x21
	recordBaseAP      = 0x37
	recordBaseDP      = 0x39
	recordDX          = 0x3e
	recordHP          = 0x40
	recordMaxHP       = 0x42
	recordMP          = 0x44
	recordMaxMP       = 0x46
	ownCamp           = 2
)

type handlerFile struct {
	Chapter int              `json:"chapter"`
	Phase   string           `json:"phase"`
	Handler string           `json:"handler"`
	Beats   []map[string]any `json:"beats"`
	Diag    map[string]any   `json:"diagnostics"`
}

type appliedOp struct {
	Chapter    int    `json:"chapter"`
	Op         string `json:"op"`
	CharID     *int   `json:"char_id,omitempty"`
	ItemID     *int   `json:"item_id,omitempty"`
	Source     any    `json:"source,omitempty"`
	Result     string `json:"result"`
	RosterSlot *int   `json:"roster_slot,omitempty"`
}

type assumption struct {
	Chapter int    `json:"chapter"`
	Kind    string `json:"kind"`
	Detail  string `json:"detail"`
}

type levelStep struct {
	Chapter   int    `json:"chapter"`
	Slot      int    `json:"roster_slot"`
	Key       int    `json:"record_key7"`
	FromLevel int    `json:"from_level"`
	ToLevel   int    `json:"to_level"`
	Gains     [5]int `json:"gains_ap_dp_dx_hp_mp"`
	Learned   []int  `json:"learned_command_ids"`
}

type recordSummary struct {
	Slot     int    `json:"roster_slot"`
	Key7     int    `json:"key7"`
	Identity int    `json:"identity8"`
	Race     int    `json:"race"`
	Class    int    `json:"class"`
	Level    int    `json:"level"`
	Exp      int    `json:"exp"`
	HP       int    `json:"hp"`
	MaxHP    int    `json:"max_hp"`
	MP       int    `json:"mp"`
	MaxMP    int    `json:"max_mp"`
	AP       int    `json:"ap"`
	DP       int    `json:"dp"`
	HIT      int    `json:"hit"`
	EV       int    `json:"ev"`
	Items    string `json:"inventory_cells_hex"`
}

type manifest struct {
	SchemaVersion   int               `json:"schema_version"`
	Kind            string            `json:"kind"`
	Tool            string            `json:"tool"`
	BaseSave        string            `json:"base_save"`
	BaseSHA256      string            `json:"base_sha256"`
	BaseSlot        int               `json:"base_slot"`
	BaseChapter     int               `json:"base_chapter"`
	TargetChapter   int               `json:"target_chapter"`
	OutSlot         int               `json:"out_slot"`
	OutputSHA256    string            `json:"output_sha256"`
	EvidenceSources map[string]string `json:"evidence_sources"`
	Applied         []appliedOp       `json:"applied"`
	LevelPolicy     string            `json:"level_policy"`
	Seed            int64             `json:"seed"`
	LevelSteps      []levelStep       `json:"level_steps"`
	Gold            *uint32           `json:"gold,omitempty"`
	Assumptions     []assumption      `json:"assumptions"`
	Roster          []recordSummary   `json:"roster"`
	RosterCount     int               `json:"roster_count"`
	RecalcCheck     string            `json:"recalc_check"`
	KnownDeviations []string          `json:"known_deviations"`
}

type builder struct {
	handlers    map[int]handlerFile // key: set_chapter 的目標章
	constructor campaign.NativeJoinConstructorTable
	itemRows    []byte
	growth      map[int]battle.GrowthRow
	learn       map[int][]battle.CommandLearnEntry
	learnSel    map[int]int
	rng         *rand.Rand
	roster      []byte
	meta        []byte
	count       int
	applied     []appliedOp
	assumptions []assumption
	steps       []levelStep
	levelsPer   int
	overrides   map[int]int
	eventStates map[[2]int]int // key: {章, 狀態表索引}；只收 -event-states 明示的值
	target      int
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fd2-chapter-slot:", err)
		os.Exit(1)
	}
}

func run() error {
	basePath := flag.String("base", "", "基底 FD2.SAV（真實通關存檔或先前建構的槽）")
	baseSlot := flag.Int("base-slot", 0, "基底槽 0..3")
	target := flag.Int("target", 0, "目標「已通關」章節（槽 chapter byte）")
	outDir := flag.String("out-dir", "", "輸出目錄（FD2.SAV 與 manifest.json）")
	outSlot := flag.Int("out-slot", -1, "輸出槽，預設同 base-slot")
	gold := flag.Int("gold", -1, "金幣覆寫；負值表示保留基底")
	levelsPer := flag.Int("levels-per-chapter", 0, "每通關一章，既有隊員各升幾級（政策，非證據；正對照顯示實際升級只發生在有擊殺的人，預設 0）")
	overrideText := flag.String("level-overrides", "", "以 key7=level 指定最終等級，逗號分隔（攻略校準用）")
	boostAP := flag.Int("boost-base-ap", 0, "每筆名冊記錄的基底 AP（+0x37）加值；政策值，114 定案，套完再跑 0x1145A 重算")
	boostDP := flag.Int("boost-base-dp", 0, "每筆名冊記錄的基底 DP（+0x39）加值；政策值，拉太高敵人會不攻擊（0x14237 差值 <= 2 略過）")
	boostDX := flag.Int("boost-base-dx", 0, "每筆名冊記錄的 DX（+0x3E，HIT 與 EV 共用基底）加值；EV 不可大於等於敵人 HIT")
	eventStateText := flag.String("event-states", "", "以 章:索引=值 指定戰後 handler 讀到的戰場狀態表值，逗號分隔（政策，非證據；例如 7:17=1 讓 ch06_post 走 JOIN12）")
	seed := flag.Int64("seed", 0, "成長擲骰種子；0 表示用 target")
	assetsDir := flag.String("assets", "assets", "remake 資產根（含 data/、cutscenes/handlers/）")
	checkRecalc := flag.Bool("check-recalc", true, "先對基底每筆隊員套 0x1B750 重算，確認是恆等（工具自檢）")
	flag.Parse()
	if *basePath == "" || *outDir == "" || *target <= 0 || *target > 30 {
		return errors.New("需要 -base、-out-dir 與 1..30 的 -target")
	}
	if *outSlot < 0 {
		*outSlot = *baseSlot
	}
	if *seed == 0 {
		*seed = int64(*target)
	}

	stored, err := os.ReadFile(*basePath)
	if err != nil {
		return err
	}
	plain, err := fdsave.Decode(stored)
	if err != nil {
		return fmt.Errorf("基底解碼：%w", err)
	}
	slot, err := fdsave.ReadSlot(plain, *baseSlot)
	if err != nil {
		return err
	}
	verified, err := fdsave.ReadVerifiedMetadata(plain, *baseSlot)
	if err != nil {
		return err
	}
	if verified.Chapter == 0xff {
		return errors.New("基底槽是空槽")
	}
	if int(verified.Chapter) >= *target {
		return fmt.Errorf("基底已是第 %d 章，目標 %d 不在其後", verified.Chapter, *target)
	}
	if int(verified.RosterCount) > fdsave.RosterUnits {
		return fmt.Errorf("基底 roster_count=%d 超過 %d", verified.RosterCount, fdsave.RosterUnits)
	}

	b := &builder{
		roster:      slot.Roster,
		meta:        slot.Metadata,
		count:       int(verified.RosterCount),
		rng:         rand.New(rand.NewSource(*seed)),
		levelsPer:   *levelsPer,
		overrides:   map[int]int{},
		eventStates: map[[2]int]int{},
		target:      *target,
	}
	if err := b.parseOverrides(*overrideText); err != nil {
		return err
	}
	if err := b.parseEventStates(*eventStateText); err != nil {
		return err
	}
	sources, err := b.loadTables(*assetsDir)
	if err != nil {
		return err
	}
	recalcNote := "skipped"
	if *checkRecalc {
		recalcNote, err = b.recalcIdentityCheck()
		if err != nil {
			return err
		}
	}

	for chapter := int(verified.Chapter) + 1; chapter <= *target; chapter++ {
		if err := b.applyChapter(chapter); err != nil {
			return fmt.Errorf("第 %d 章：%w", chapter, err)
		}
	}
	if err := b.applyBoost(*target, *boostAP, *boostDP, *boostDX); err != nil {
		return err
	}
	var goldOut *uint32
	if *gold >= 0 {
		value := uint32(*gold)
		binary.LittleEndian.PutUint32(b.meta[2:6], value)
		goldOut = &value
		b.assumptions = append(b.assumptions, assumption{Chapter: *target, Kind: "gold", Detail: fmt.Sprintf("金幣覆寫為 %d（攻略估值，非原版證據）", value)})
	}
	b.meta[0] = byte(*target)
	b.meta[1] = byte(b.count)

	out, err := fdsave.WriteSlot(plain, *outSlot, fdsave.Slot{Roster: b.roster, Metadata: b.meta})
	if err != nil {
		return err
	}
	encoded, err := fdsave.Encode(out)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		return err
	}
	savePath := filepath.Join(*outDir, "FD2.SAV")
	if err := os.WriteFile(savePath, encoded, 0o644); err != nil {
		return err
	}
	m := manifest{
		SchemaVersion:   1,
		Kind:            "fd2_chapter_slot_manifest",
		Tool:            "remake/cmd/fd2-chapter-slot",
		BaseSave:        *basePath,
		BaseSHA256:      sha(stored),
		BaseSlot:        *baseSlot,
		BaseChapter:     int(verified.Chapter),
		TargetChapter:   *target,
		OutSlot:         *outSlot,
		OutputSHA256:    sha(encoded),
		EvidenceSources: sources,
		Applied:         b.applied,
		LevelPolicy:     fmt.Sprintf("levels-per-chapter=%d overrides=%s（政策，非原版證據；升級寫入依 0x1E292／0x1E529／0x1B750）", *levelsPer, *overrideText),
		Seed:            *seed,
		LevelSteps:      b.steps,
		Gold:            goldOut,
		Assumptions:     b.assumptions,
		Roster:          b.summaries(),
		RosterCount:     b.count,
		RecalcCheck:     recalcNote,
		KnownDeviations: knownDeviations,
	}
	blob, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(*outDir, "manifest.json"), append(blob, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s sha256=%s chapter=%d roster=%d applied=%d level_steps=%d\n",
		savePath, m.OutputSHA256, *target, b.count, len(b.applied), len(b.steps))
	return nil
}

// knownDeviations 是正對照（ch01→ch02、ch02→ch03 對真實通關槽）查到、本工具刻意不模擬的差異。
var knownDeviations = []string{
	"經驗值、擊殺帶來的升級、金幣與掉落物品（死亡獎勵、寶箱）全部依實際遊玩而定，本工具不模擬；等級與金幣只能靠 -levels-per-chapter／-level-overrides／-gold 明示。",
	"紀錄 +0x00..+0x04（戰場座標等）與 +0x28..+0x36 隨遊玩漂移，保留基底值。",
	"JOIN 建構器輸出的物品格 6／7 是 80 00，真實原版存檔是 80 ff（旗標 bit7 置位時 item byte 不被消費）；差異已登記待解。",
}

func sha(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (b *builder) parseOverrides(text string) error {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	for _, pair := range strings.Split(text, ",") {
		kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(kv) != 2 {
			return fmt.Errorf("level-overrides 格式錯誤：%q", pair)
		}
		key, err1 := strconv.Atoi(kv[0])
		level, err2 := strconv.Atoi(kv[1])
		if err1 != nil || err2 != nil || level <= 0 {
			return fmt.Errorf("level-overrides 格式錯誤：%q", pair)
		}
		b.overrides[key] = level
	}
	return nil
}

func (b *builder) parseEventStates(text string) error {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	for _, pair := range strings.Split(text, ",") {
		kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		ci := []string{}
		if len(kv) == 2 {
			ci = strings.SplitN(kv[0], ":", 2)
		}
		if len(ci) != 2 {
			return fmt.Errorf("event-states 格式錯誤：%q", pair)
		}
		chapter, err1 := strconv.Atoi(ci[0])
		index, err2 := strconv.Atoi(ci[1])
		value, err3 := strconv.Atoi(kv[1])
		if err1 != nil || err2 != nil || err3 != nil || index < 0 || value < 0 || value > 0xFF {
			return fmt.Errorf("event-states 格式錯誤：%q", pair)
		}
		b.eventStates[[2]int{chapter, index}] = value
	}
	return nil
}

func (b *builder) loadTables(assetsDir string) (map[string]string, error) {
	sources := map[string]string{}
	handlerDir := filepath.Join(assetsDir, "cutscenes", "handlers")
	entries, err := os.ReadDir(handlerDir)
	if err != nil {
		return nil, err
	}
	b.handlers = map[int]handlerFile{}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, "_post.json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(handlerDir, name))
		if err != nil {
			return nil, err
		}
		var h handlerFile
		if err := json.Unmarshal(raw, &h); err != nil {
			return nil, fmt.Errorf("%s：%w", name, err)
		}
		targets := collectSetChapter(h.Beats)
		if len(targets) != 1 {
			return nil, fmt.Errorf("%s 的 set_chapter 不唯一：%v", name, targets)
		}
		if prev, dup := b.handlers[targets[0]]; dup {
			return nil, fmt.Errorf("第 %d 章有兩個戰後 handler：%s 與 %s", targets[0], prev.Handler, h.Handler)
		}
		b.handlers[targets[0]] = h
		sources["handler_ch"+fmt.Sprintf("%02d", targets[0])] = filepath.Join("assets/cutscenes/handlers", name) + " @" + h.Handler
	}

	constructorPath := filepath.Join(assetsDir, "data", "native_join_constructor.json")
	b.constructor, err = campaign.LoadNativeJoinConstructorTable(constructorPath)
	if err != nil {
		return nil, err
	}
	sources["join_constructor"] = "assets/data/native_join_constructor.json（sub_112A5，0x55ba1／0x55ea1）"
	b.itemRows, err = battle.LoadNativeItemEffectRowPrefix(filepath.Join(assetsDir, "data", "native_item_effect_rows.json"))
	if err != nil {
		return nil, err
	}
	sources["item_rows"] = "assets/data/native_item_effect_rows.json（0x602ad 列表，0x1145a／0x1B750 重算）"
	b.growth, err = battle.LoadNativeGrowthRows(filepath.Join(assetsDir, "data", "class_change_growth.json"))
	if err != nil {
		return nil, err
	}
	sources["growth"] = "assets/data/class_change_growth.json（0x55EA1，68 列，+7 選列）"
	learnPath := filepath.Join(assetsDir, "data", "command_learn.json")
	b.learn, err = battle.LoadCommandLearn(learnPath)
	if err != nil {
		return nil, err
	}
	b.learnSel, err = battle.LoadCommandLearnSelectors(filepath.Join(assetsDir, "data", "class_change_growth.json"))
	if err != nil {
		return nil, err
	}
	sources["command_learn"] = "assets/data/command_learn.json（升級學指令，learn_idx 來自成長列）"
	return sources, nil
}

func collectSetChapter(beats []map[string]any) []int {
	var out []int
	for _, beat := range beats {
		if beat["op"] == "set_chapter" {
			if v, ok := beat["chapter"].(float64); ok {
				out = append(out, int(v))
			}
		}
		for _, branch := range []string{"then", "else"} {
			if nested, ok := beat[branch].([]any); ok {
				out = append(out, collectSetChapter(toBeatList(nested))...)
			}
		}
	}
	return out
}

func toBeatList(items []any) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

// recalcIdentityCheck 對基底每筆隊員套 0x1B750 重算，結果必須與存檔一致；
// 不一致代表物品表或重算轉寫與這份存檔不合，工具不該繼續。
func (b *builder) recalcIdentityCheck() (string, error) {
	for i := 0; i < b.count; i++ {
		record := b.record(i)
		if record[recordCamp] != ownCamp {
			continue
		}
		copyRecord := append([]byte(nil), record...)
		if err := battle.ApplyNativeEquipmentRecalc(copyRecord, b.itemRows); err != nil {
			return "", fmt.Errorf("基底第 %d 筆重算：%w", i, err)
		}
		if !equalBytes(copyRecord[0x48:0x50], record[0x48:0x50]) {
			return "", fmt.Errorf("基底第 %d 筆（key7=%d）重算後 +0x48..+0x4f 不同：%x vs %x",
				i, record[recordKey], copyRecord[0x48:0x50], record[0x48:0x50])
		}
	}
	return fmt.Sprintf("基底 %d 筆隊員套 0x1B750 重算皆恆等", b.count), nil
}

func equalBytes(a, c []byte) bool {
	if len(a) != len(c) {
		return false
	}
	for i := range a {
		if a[i] != c[i] {
			return false
		}
	}
	return true
}

func (b *builder) record(i int) []byte {
	return b.roster[i*fdsave.UnitSize : (i+1)*fdsave.UnitSize]
}

func (b *builder) applyChapter(chapter int) error {
	// 先升級：這一章打完才有 JOIN，新成員不吃這一章的經驗。
	for i := 0; i < b.count; i++ {
		record := b.record(i)
		if record[recordCamp] != ownCamp {
			continue
		}
		key := int(record[recordKey])
		targetLevel := int(record[recordLevel]) + b.levelsPer
		if override, ok := b.overrides[key]; ok && chapter == b.finalChapterFor() {
			targetLevel = override
		}
		for int(record[recordLevel]) < targetLevel {
			if !b.levelUp(chapter, i) {
				break
			}
		}
		// 通關存檔一律 HP＝MaxHP、MP＝MaxMP（ch01–03 真實存檔皆如此）；升級本身不碰目前值。
		copy(record[recordHP:recordHP+2], record[recordMaxHP:recordMaxHP+2])
		copy(record[recordMP:recordMP+2], record[recordMaxMP:recordMaxMP+2])
	}
	h, ok := b.handlers[chapter]
	if !ok {
		return fmt.Errorf("找不到 set_chapter=%d 的戰後 handler", chapter)
	}
	return b.applyBeats(chapter, h.Beats)
}

// finalChapterFor 回傳 overrides 該在哪一章套用：只在最後一章，避免中途就被鎖死。
func (b *builder) finalChapterFor() int { return b.target }

func (b *builder) applyBeats(chapter int, beats []map[string]any) error {
	for _, beat := range beats {
		switch beat["op"] {
		case "join":
			id := int(beat["char_id"].(float64))
			if err := b.join(chapter, id, beat["source"]); err != nil {
				return err
			}
		case "grant_item":
			item := int(beat["item_id"].(float64))
			b.grantItem(chapter, item, beat["source"])
		case "if":
			branch, why := b.decideBranch(chapter, beat)
			b.assumptions = append(b.assumptions, assumption{Chapter: chapter, Kind: "if", Detail: why})
			if nested, ok := beat[branch].([]any); ok {
				if err := b.applyBeats(chapter, toBeatList(nested)); err != nil {
					return err
				}
			}
		case "deactivate_unit":
			b.assumptions = append(b.assumptions, assumption{Chapter: chapter, Kind: "deactivate_unit",
				Detail: "handler 有 deactivate_unit；本工具不改持續隊伍（只影響戰場 runtime），若原版此處會移除隊員需另證"})
		}
	}
	return nil
}

// decideBranch 用「一般玩家最佳情況」決定 if 分支：無人陣亡、回合數不超限；
// 能從已建構的隊伍判定的（roster_has、物品在不在）就照實判定。
// 戰場狀態表的值來自該章戰鬥裡的事件（例如格子事件、回合事件寫的 state），存檔裡沒有；
// 沒以 -event-states 明示就當 0。它可能決定 JOIN（ch06_post 的 state17），不只影響對白。
func (b *builder) decideBranch(chapter int, beat map[string]any) (string, string) {
	cond, _ := beat["condition"].(map[string]any)
	op, _ := cond["op"].(string)
	switch op {
	case "any_unit_inactive":
		return "else", "any_unit_inactive→false（假設無人陣亡）"
	case "roster_has", "native_persistent_identity_present":
		id := intField(cond, "char_id", "native_persistent_identity")
		if b.hasIdentity(id) {
			return "then", fmt.Sprintf("%s(%d)→true（已在隊伍）", op, id)
		}
		return "else", fmt.Sprintf("%s(%d)→false（不在隊伍）", op, id)
	case "native_inventory_item_present":
		item := intField(cond, "native_inventory_item_id")
		if b.hasItem(item) {
			return "then", fmt.Sprintf("物品 %d 在隊伍→true", item)
		}
		return "else", fmt.Sprintf("物品 %d 不在隊伍→false", item)
	case "native_round_gt", "native_inactive_count_gt", "native_any_of":
		return "else", op + "→false（假設回合數未超限、陣亡數未超限）"
	case "native_event_state_eq", "native_event_state_nonzero":
		index := intField(cond, "event_state_index")
		value, given := b.eventStates[[2]int{chapter, index}]
		source := "-event-states 明示"
		if !given {
			value, source = 0, "未明示，當 0"
		}
		taken := value != 0
		if op == "native_event_state_eq" {
			taken = value == intField(cond, "event_state_value")
		}
		branch := "else"
		if taken {
			branch = "then"
		}
		return branch, fmt.Sprintf("%s(state%d)→%t（第 %d 章戰場狀態 %d＝%d，%s）", op, index, taken, chapter, index, value, source)
	}
	return "else", fmt.Sprintf("未知條件 %q→else", op)
}

func intField(m map[string]any, keys ...string) int {
	for _, key := range keys {
		if v, ok := m[key].(float64); ok {
			return int(v)
		}
	}
	return -1
}

func (b *builder) hasIdentity(id int) bool {
	for i := 0; i < b.count; i++ {
		if int(b.record(i)[recordIdentity]) == id {
			return true
		}
	}
	return false
}

func (b *builder) hasItem(item int) bool {
	for i := 0; i < b.count; i++ {
		record := b.record(i)
		for slot := 0; slot < 8; slot++ {
			cell := recordInventory + slot*2
			if record[cell]&0x80 == 0 && int(record[cell+1]) == item {
				return true
			}
		}
	}
	return false
}

func (b *builder) join(chapter, id int, source any) error {
	entry := appliedOp{Chapter: chapter, Op: "join", CharID: &id, Source: source}
	if !campaign.JoinableCharacterID(id) {
		return fmt.Errorf("join char_id=%d 不合法", id)
	}
	if b.hasIdentity(id) {
		entry.Result = "already_present"
		b.applied = append(b.applied, entry)
		return nil
	}
	if b.count >= fdsave.RosterUnits {
		return fmt.Errorf("join char_id=%d：隊伍已滿 %d", id, fdsave.RosterUnits)
	}
	record, err := b.constructor.MaterializePersistentRecord(id, b.itemRows)
	if err != nil {
		return err
	}
	// sub_112A5 寫進 persistent_count 那一格。ch01→ch02 真實存檔的新成員（id 8）
	// 在建構器沒寫到的 +0x28..+0x30 全是 0，只有 +0x31 是 0xff，與建構器的零初始
	// 紀錄一致；所以整筆覆寫，不保留槽區那一格的舊 bytes（那是存檔緩衝殘值）。
	copy(b.record(b.count), record.Raw[:])
	slot := b.count
	b.count++
	entry.Result = "appended"
	entry.RosterSlot = &slot
	b.applied = append(b.applied, entry)
	return nil
}

func (b *builder) grantItem(chapter, item int, source any) {
	entry := appliedOp{Chapter: chapter, Op: "grant_item", ItemID: &item, Source: source}
	for i := 0; i < b.count; i++ {
		record := b.record(i)
		if record[recordCamp] != ownCamp {
			continue
		}
		for slot := 0; slot < 8; slot++ {
			cell := recordInventory + slot*2
			if record[cell]&0x80 != 0 {
				record[cell], record[cell+1] = 0, byte(item)
				entry.Result = "granted"
				entry.RosterSlot = &i
				b.applied = append(b.applied, entry)
				return
			}
		}
	}
	entry.Result = "dropped_all_full"
	b.applied = append(b.applied, entry)
}

// levelUp 轉寫 0x1E292 的升一級：+7 選成長列、擲骰加到 +0x37／+0x39／+0x3e／+0x42／+0x46、
// +0x21 加一、學該級指令、最後 0x1B750 重算 +0x48..+0x4e。目前 HP／MP 不動。
func (b *builder) levelUp(chapter, slot int) bool {
	record := b.record(slot)
	key := int(record[recordKey])
	level := int(record[recordLevel])
	capLevel := 40
	if key == 0x1e || key == 0x1f {
		capLevel = 99
	}
	if level >= capLevel {
		return false
	}
	row, ok := b.growth[key]
	if !ok {
		b.assumptions = append(b.assumptions, assumption{Chapter: chapter, Kind: "growth_missing",
			Detail: fmt.Sprintf("roster %d key7=%d 沒有成長列，跳過升級", slot, key)})
		return false
	}
	gains := [5]int{
		rollRange(b.rng, row.AP), rollRange(b.rng, row.DP), rollRange(b.rng, row.DX),
		rollRange(b.rng, row.HP), rollRange(b.rng, row.MP),
	}
	record[recordLevel] = byte(level + 1)
	addWord(record, recordBaseAP, gains[0])
	addWord(record, recordBaseDP, gains[1])
	addWord(record, recordDX, gains[2])
	addWord(record, recordMaxHP, gains[3])
	addWord(record, recordMaxMP, gains[4])
	var learned []int
	if learnIdx, ok := b.learnSel[key]; ok {
		for _, entry := range b.learn[learnIdx] {
			if entry.RequiredLevel == level+1 {
				record[recordCommandMask+entry.CommandID/8] |= 1 << (entry.CommandID % 8)
				learned = append(learned, entry.CommandID)
			}
		}
	}
	if err := battle.ApplyNativeEquipmentRecalc(record, b.itemRows); err != nil {
		b.assumptions = append(b.assumptions, assumption{Chapter: chapter, Kind: "recalc_failed", Detail: err.Error()})
	}
	b.steps = append(b.steps, levelStep{Chapter: chapter, Slot: slot, Key: key, FromLevel: level, ToLevel: level + 1, Gains: gains, Learned: learned})
	return true
}

// applyBoost 是 114 的建構槽強化政策：對名冊每筆記錄把基底 AP／DP 與 DX 加上政策值，再跑
// 0x1145A 重算讓 +0x48..+0x4E 與裝備一致。不直接改 +0x48..+0x4E：之後的換裝或升級重算會蓋回去，
// 原版與重製端就會在不同時點分岔。加完超出有號 word 範圍就失敗，不截斷。
func (b *builder) applyBoost(chapter, ap, dp, dx int) error {
	if ap == 0 && dp == 0 && dx == 0 {
		return nil
	}
	if ap < 0 || dp < 0 || dx < 0 {
		return fmt.Errorf("boost 政策值不可為負：ap=%d dp=%d dx=%d", ap, dp, dx)
	}
	var detail []string
	for slot := 0; slot < b.count; slot++ {
		record := b.record(slot)
		before := [4]int16{}
		for i, off := range [...]int{0x48, 0x4a, 0x4c, 0x4e} {
			before[i] = int16(binary.LittleEndian.Uint16(record[off:]))
		}
		for _, field := range []struct{ off, delta int }{{recordBaseAP, ap}, {recordBaseDP, dp}, {recordDX, dx}} {
			value := int(int16(binary.LittleEndian.Uint16(record[field.off:]))) + field.delta
			if value > 0x7fff {
				return fmt.Errorf("boost：名冊 %d +0x%x 加 %d 後是 %d，超出有號 word", slot, field.off, field.delta, value)
			}
			addWord(record, field.off, field.delta)
		}
		if err := battle.ApplyNativeEquipmentRecalc(record, b.itemRows); err != nil {
			return fmt.Errorf("boost：名冊 %d 重算：%w", slot, err)
		}
		after := [4]int16{}
		for i, off := range [...]int{0x48, 0x4a, 0x4c, 0x4e} {
			after[i] = int16(binary.LittleEndian.Uint16(record[off:]))
		}
		detail = append(detail, fmt.Sprintf("roster %d id8=%d AP/DP/HIT/EV %v→%v", slot, record[recordIdentity], before, after))
	}
	b.assumptions = append(b.assumptions, assumption{Chapter: chapter, Kind: "boost", Detail: fmt.Sprintf(
		"基底 AP +%d、DP +%d、DX +%d（114 政策值，非原版證據；收據不得用來談傷害、存活或敵方選目標），套用 %d 筆：%s",
		ap, dp, dx, b.count, strings.Join(detail, "；"))})
	return nil
}

func rollRange(rng *rand.Rand, r battle.StatRange) int {
	if r.Max <= r.Min {
		return r.Min
	}
	return r.Min + rng.Intn(r.Max-r.Min+1)
}

func addWord(record []byte, off, delta int) {
	value := int(int16(binary.LittleEndian.Uint16(record[off:])))
	binary.LittleEndian.PutUint16(record[off:], uint16(value+delta))
}

func (b *builder) summaries() []recordSummary {
	out := make([]recordSummary, 0, b.count)
	for i := 0; i < b.count; i++ {
		r := b.record(i)
		w := func(off int) int { return int(int16(binary.LittleEndian.Uint16(r[off:]))) }
		out = append(out, recordSummary{
			Slot: i, Key7: int(r[recordKey]), Identity: int(r[recordIdentity]),
			Race: int(r[0x1f]), Class: int(r[0x20]), Level: int(r[recordLevel]), Exp: int(r[0x3c]),
			HP: w(recordHP), MaxHP: w(recordMaxHP), MP: w(recordMP), MaxMP: w(recordMaxMP),
			AP: w(0x48), DP: w(0x4a), HIT: w(0x4c), EV: w(0x4e),
			Items: hex.EncodeToString(r[recordInventory : recordInventory+16]),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Slot < out[j].Slot })
	return out
}
