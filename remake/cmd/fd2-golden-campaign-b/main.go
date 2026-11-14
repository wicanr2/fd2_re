// fd2-golden-campaign-b 輸出 internal/campaign「B 組」（設施畫面與城鎮／商店／教會規則）的黃金差分
// 資料，供 Rust 改寫 remake-rs/crates/fd2-campaign 逐欄比對
// （docs/goal/119-goal-rust-bevy-rewrite-20261113.md、remake-rs/crates/fd2-campaign/PORTING.md）。
//
// Go 版凍結於 v.1.1.11；本工具只呼叫 campaign、battle、fdother、fdtxt、fdicon、dato 的公開 API，
// 不改任何遊戲行為。
//   - 真實輸入：原版 FLAME2（FDOTHER.DAT、FDTXT.DAT、DATO.DAT、FDICON.B24）、分離素材包
//     （shop、ui、surfaces、sprites/fdicon、text、fonts 六個目錄）、remake/assets/data 的表格與
//     docs/data/exe_tables 的 EXE 表。單檔記大小與 SHA-256；素材包目錄記排序後的檔案數與清單雜湊
//     （每列「相對路徑\t大小\tSHA-256」）。
//   - 畫面案例：教會、商店、城鎮、職業清單、確認對話框、戰鬥系統問句、讀檔欄、系統資訊與金額
//     捲動，對固定的選擇、游標與脈衝狀態合成，每格畫面記 SHA-256。
//   - 規則案例：以固定種子的 PCG 產生單位（json.Marshal(battle.Unit)），跑 BuyGood、SellSlot、
//     ReviveUnit、ApplyClassChange（math/rand 種子另記）等，結果的單位同樣以 Go 欄位名輸出。
//
// 錯誤欄位是 Go err.Error() 原文；Rust 端逐字比對（標準函式庫錯誤只比接受或拒收，見 Rust 端
// KNOWN_DIFFERENCES）。
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	mrand "math/rand"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

type obj = map[string]any

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func errText(err error) any {
	if err == nil {
		return nil
	}
	return err.Error()
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func shas(frames ...[]byte) []string {
	out := make([]string, len(frames))
	for i, f := range frames {
		out[i] = digest(f)
	}
	return out
}

// frameResult 是畫面案例的共同結果：錯誤與每格畫面的 SHA-256。
func frameResult(err error, frames ...[]byte) obj {
	if err != nil {
		return obj{"err": err.Error(), "frames": []string{}}
	}
	return obj{"err": nil, "frames": shas(frames...)}
}

func framesResult(frames [][]byte, err error) obj {
	return frameResult(err, frames...)
}

func oneResult(frame []byte, err error) obj {
	if err != nil {
		return frameResult(err)
	}
	return frameResult(nil, frame)
}

type exporter struct {
	rng   *rand.Rand
	cases []obj
	flame string
	pack  string
	repo  string

	fdotherPath, fdtxtPath, datoPath, fdiconPath string

	bg14       []byte
	entries14  []fdother.LMI1Entry
	dialogue   []fdother.RawCell
	digits     []fdother.Frame
	strings    *fdtxt.Strings
	font       *fdtxt.Font
	bank       *fdicon.Bank
	cells2     []fdother.RawCell
	loadBox    fdother.LMI1Entry
	revive     []fdother.Frame
	itemArch   battle.NativeItemPanelDataAssets
	itemPack   battle.NativeItemPanelDataAssets
	effectRows []byte
	sysPanels  [4]fdother.Frame
	shops      map[string]*campaign.NativeShopAssets
	towns      map[string]*campaign.NativeTownAssets
	datoCache  map[int][]dato.Frame
	stats      map[int]campaign.ItemStats
	itemIDs    []int
	classTable campaign.ClassChangeTable
	growth     map[int]campaign.ClassChangeGrowth
}

func (e *exporter) add(op string, input obj, result any) {
	e.cases = append(e.cases, obj{"index": len(e.cases), "op": op, "input": input, "result": result})
}

func (e *exporter) portrait(id int) dato.Frame {
	if frames, ok := e.datoCache[id]; ok {
		return frames[0]
	}
	frames := must(dato.DecodeResource(e.datoPath, id))
	e.datoCache[id] = frames
	return frames[0]
}

func fileRow(path, logical string) obj {
	raw := must(os.ReadFile(path))
	return obj{"path": logical, "size": len(raw), "sha256": digest(raw)}
}

// dirRow 記一個素材包目錄的檔案數與清單雜湊（排序後每列「相對路徑\t大小\tSHA-256\n」）。
func dirRow(root, rel string) obj {
	var lines []string
	base := filepath.Join(root, rel)
	err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		r, _ := filepath.Rel(base, path)
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lines = append(lines, fmt.Sprintf("%s\t%d\t%s\n", filepath.ToSlash(r), len(raw), digest(raw)))
		return nil
	})
	if err != nil {
		panic(err)
	}
	sort.Strings(lines)
	return obj{"dir": rel, "files": len(lines), "manifest_sha256": digest([]byte(strings.Join(lines, "")))}
}

func (e *exporter) loadAssets() obj {
	e.fdotherPath = filepath.Join(e.flame, "FDOTHER.DAT")
	e.fdtxtPath = filepath.Join(e.flame, "FDTXT.DAT")
	e.datoPath = filepath.Join(e.flame, "DATO.DAT")
	e.fdiconPath = filepath.Join(e.flame, "FDICON.B24")
	res14 := must(fdother.ReadResource(e.fdotherPath, 14))
	e.entries14 = make([]fdother.LMI1Entry, 17)
	for _, index := range []int{1, 16} {
		e.entries14[index] = must(fdother.ParseLMI1OpaqueEntry(res14, index))
	}
	for index := 3; index <= 10; index++ {
		cell := must(fdother.ParseLMI1RawEntry(res14, index))
		e.entries14[index] = fdother.LMI1Entry{Width: cell.Width, Height: cell.Height, Pixels: cell.Pixels}
	}
	bgFrame := must(fdother.ParseLMI1FrameEntry(res14, 0))
	e.bg14 = make([]byte, 320*200)
	if err := bgFrame.BlitAt(e.bg14, 320, 0, -1); err != nil {
		panic(err)
	}
	for i := 0; i < 9; i++ {
		e.revive = append(e.revive, must(fdother.ParseLMI1FrameEntry(res14, 23+i)))
	}
	res5 := must(fdother.ReadResource(e.fdotherPath, 5))
	e.dialogue = make([]fdother.RawCell, 18)
	for index := 1; index <= 17; index++ {
		e.dialogue[index] = must(fdother.ParseLMI1RawEntry(res5, index))
	}
	for digit := 0; digit < 10; digit++ {
		e.digits = append(e.digits, must(fdother.ParseLMI1FrameEntry(res5, 31+digit)))
	}
	e.strings = must(fdtxt.Parse(must(fdother.ReadResource(e.fdtxtPath, 0))))
	e.font = must(fdtxt.ParseFont(must(fdother.ReadResource(e.fdotherPath, 4))))
	e.bank = must(fdicon.DecodeFile(e.fdiconPath))
	e.cells2 = must(fdother.DecodeRawCellResource(e.fdotherPath, 2))
	entries13 := must(fdother.ParseLMI1(must(fdother.ReadResource(e.fdotherPath, 13))))
	e.loadBox = entries13[16]
	e.itemArch = must(battle.LoadNativeItemPanelDataAssetsArchive(e.fdotherPath, e.fdtxtPath))
	e.itemPack = must(battle.LoadNativeItemPanelDataAssets(e.pack))
	e.effectRows = must(battle.LoadNativeItemEffectRowPrefix(filepath.Join(e.repo, "remake/assets/data/native_item_effect_rows.json")))
	e.sysPanels = must(fdother.LoadSeparatedSystemInfoPanels(filepath.Join(e.pack, "ui")))
	e.shops = map[string]*campaign.NativeShopAssets{}
	for _, r := range []int{12, 13, 29, 63} {
		e.shops[fmt.Sprintf("pack%d", r)] = must(campaign.LoadSeparatedNativeShopAssets(e.pack, r))
		e.shops[fmt.Sprintf("archive%d", r)] = must(campaign.DecodeNativeShopAssets(e.fdotherPath, r))
	}
	e.towns = map[string]*campaign.NativeTownAssets{
		"pack":    must(campaign.LoadNativeTownAssets(e.pack)),
		"archive": must(campaign.DecodeNativeTownAssetsArchive(e.fdotherPath, e.fdiconPath)),
	}
	e.datoCache = map[int][]dato.Frame{}
	e.stats = must(campaign.LoadItemStats(filepath.Join(e.repo, "remake/assets/data/item.json")))
	for id := range e.stats {
		e.itemIDs = append(e.itemIDs, id)
	}
	sort.Ints(e.itemIDs)
	e.classTable = must(campaign.LoadClassChangeTable(filepath.Join(e.repo, "docs/data/exe_tables/class_change_targets.json")))
	e.growth = must(campaign.LoadClassChangeGrowth(filepath.Join(e.repo, "docs/data/exe_tables/growth.json")))

	var files []obj
	for _, name := range []string{"DATO.DAT", "FDICON.B24", "FDOTHER.DAT", "FDTXT.DAT"} {
		files = append(files, fileRow(filepath.Join(e.flame, name), "flame2/"+name))
	}
	for _, rel := range []string{
		"docs/data/exe_tables/class_change_targets.json",
		"docs/data/exe_tables/growth.json",
		"docs/data/exe_tables/revive_fee_rates.json",
		"remake/assets/data/class_change_growth.json",
		"remake/assets/data/class_change_targets.json",
		"remake/assets/data/class_equip_types.json",
		"remake/assets/data/item.json",
		"remake/assets/data/native_item_effect_rows.json",
		"remake/assets/data/revive_fee_rates.json",
	} {
		files = append(files, fileRow(filepath.Join(e.repo, rel), rel))
	}
	var dirs []obj
	for _, rel := range []string{"fonts", "shop", "sprites/fdicon", "surfaces", "text", "ui"} {
		dirs = append(dirs, dirRow(e.pack, rel))
	}
	return obj{"files": files, "pack_dirs": dirs}
}

// ---- 素材載入本身 ----

func shopDigest(a *campaign.NativeShopAssets) obj {
	var service []string
	for _, pair := range a.ServiceCells {
		for _, c := range pair {
			service = append(service, fmt.Sprintf("%dx%d:%s", c.Width, c.Height, digest(c.Pixels)))
		}
	}
	var compare []string
	for _, c := range a.CompareCells {
		compare = append(compare, fmt.Sprintf("%dx%d:%s", c.Width, c.Height, digest(c.Pixels)))
	}
	var success []string
	for _, f := range a.SuccessFrames {
		indexed, mask, err := f.IndexedLayers()
		if err != nil {
			panic(err)
		}
		success = append(success, fmt.Sprintf("%dx%d:%s:%s", f.Width, f.Height, digest(indexed), digest(mask)))
	}
	var raw []string
	for _, r := range a.RawEntries {
		raw = append(raw, digest(r))
	}
	return obj{
		"resource":   a.ResourceID,
		"background": digest(a.Background),
		"decoration": fmt.Sprintf("%dx%d:%s", a.Decoration.Width, a.Decoration.Height, digest(a.Decoration.Pixels)),
		"gold":       fmt.Sprintf("%dx%d:%s", a.GoldRollStrip.Width, a.GoldRollStrip.Height, digest(a.GoldRollStrip.Pixels)),
		"service":    service,
		"price":      fmt.Sprintf("%dx%d:%s", a.PriceCell.Width, a.PriceCell.Height, digest(a.PriceCell.Pixels)),
		"panel":      fmt.Sprintf("%dx%d:%s", a.Panel.Width, a.Panel.Height, digest(a.Panel.Pixels)),
		"compare":    compare,
		"success":    success,
		"raw":        raw,
	}
}

func townDigest(a *campaign.NativeTownAssets) obj {
	var bgs, pulse []string
	for _, b := range a.Backgrounds {
		bgs = append(bgs, digest(b))
	}
	for _, s := range a.Pulse {
		pulse = append(pulse, digest(s.Pixels)+":"+digest(s.Mask)+":"+digest(s.RemapMask))
	}
	return obj{
		"backgrounds": bgs,
		"label":       fmt.Sprintf("%dx%d:%s", a.Label.Width, a.Label.Height, digest(a.Label.Pixels)),
		"pulse":       pulse,
	}
}

func (e *exporter) genAssets() {
	keys := make([]string, 0, len(e.shops))
	for k := range e.shops {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		e.add("shop_assets", obj{"key": k}, shopDigest(e.shops[k]))
	}
	for _, k := range []string{"archive", "pack"} {
		e.add("town_assets", obj{"key": k}, townDigest(e.towns[k]))
	}
	for _, bad := range []int{-1, 0, 11, 14, 64} {
		_, err := campaign.LoadSeparatedNativeShopAssets(e.pack, bad)
		e.add("shop_assets_reject", obj{"resource": bad}, obj{"err": errText(err)})
		_, err = campaign.DecodeNativeShopAssets(e.fdotherPath, bad)
		e.add("shop_archive_reject", obj{"resource": bad}, obj{"err": errText(err)})
	}
}

// ---- 教會 ----

func (e *exporter) genChurch() {
	p131 := e.portrait(131)
	for _, gold := range []int{0, 7, 12345678, 99999999} {
		for _, text := range []int{585, 586} {
			frame, err := campaign.ComposeNativeChurchScene(e.bg14, e.entries14[1], e.dialogue, e.digits, p131, e.strings, e.font, gold, text)
			e.add("church_scene", obj{"gold": gold, "text": text}, oneResult(frame, err))
		}
		frame, err := campaign.ComposeNativeChurchSceneBase(e.bg14, e.entries14[1], e.dialogue, e.digits, p131, gold)
		e.add("church_scene_base", obj{"gold": gold}, oneResult(frame, err))
	}
	for _, gold := range []int{-1, 100_000_000} {
		frame, err := campaign.ComposeNativeChurchSceneBase(e.bg14, e.entries14[1], e.dialogue, e.digits, p131, gold)
		e.add("church_scene_base", obj{"gold": gold}, oneResult(frame, err))
	}
	for _, opening := range []bool{true, false} {
		frames, err := campaign.NativeChurchMenuTransitionFrames(e.bg14, e.entries14, opening)
		e.add("church_menu_transition", obj{"opening": opening}, framesResult(frames, err))
	}
	for selected := -1; selected <= 4; selected++ {
		for pulse := -1; pulse <= 2; pulse++ {
			frame, err := campaign.ComposeNativeChurchMenuFrame(e.bg14, e.entries14, selected, pulse)
			e.add("church_menu_frame", obj{"selected": selected, "pulse": pulse}, oneResult(frame, err))
		}
	}
	animation, final, err := campaign.ComposeNativeChurchReviveSuccessFrames(e.bg14, e.revive, p131)
	if err == nil {
		animation = append(animation, final)
	}
	e.add("church_revive_success", obj{}, framesResult(animation, err))
	for _, tc := range [][3]int{{506, 10, campaign.NativeShopTextOffset}, {585, 3, 119*320 + 12}, {511, 20, campaign.NativeShopTextOffset}, {9999, 1, 0}} {
		frame := append([]byte(nil), e.bg14...)
		out, err := campaign.ComposeNativeChurchTextWithNameAt(frame, e.strings, e.font, tc[0], tc[1], tc[2])
		e.add("church_text_name", obj{"text": tc[0], "name": tc[1], "offset": tc[2]}, obj{"result": oneResult(out, err), "frame": digest(frame)})
	}
	for _, tc := range []struct{ name, fee int }{{1, 0}, {10, 280}, {23, 99999}, {5, -1}} {
		frame, err := campaign.ComposeNativeReviveConfirmationQuestion(e.bg14, e.strings, e.font, tc.name, tc.fee)
		e.add("revive_confirm", obj{"name": tc.name, "fee": tc.fee}, oneResult(frame, err))
	}
	for counter := -1; counter <= 6; counter++ {
		frame, err := campaign.ComposeNativeTransientExpiryFrame(e.bg14, e.dialogue, p131, e.strings, e.font, counter)
		e.add("transient_expiry", obj{"counter": counter}, oneResult(frame, err))
	}
	dialogue, err := campaign.ComposeNativeTransientExpiryDialogue(e.bg14, e.dialogue, p131)
	if err != nil {
		panic(err)
	}
	for _, tc := range [][2]int{{0, 0}, {0, 0x5a}, {0, 0xff}, {0, 0x100}, {1, 0}, {1, 150}, {1, 65535}, {2, 1}, {1, -1}} {
		frame, err := campaign.ComposeNativeDeathRewardMessageFrame(dialogue, e.strings, e.font, tc[0], tc[1])
		e.add("death_reward_msg", obj{"kind": tc[0], "value": tc[1]}, oneResult(frame, err))
	}
}

// ---- 職業清單、名冊、確認框 ----

func (e *exporter) sprite(group, pose, cycle int) fdicon.Sprite {
	return must(e.bank.SpriteFor(group, pose, cycle))
}

func (e *exporter) genClassAndRoster() {
	for k := 0; k < 12; k++ {
		n := 1 + e.rng.IntN(3)
		var rows [][]int
		var native []campaign.NativeClassListRow
		for i := 0; i < n; i++ {
			row := []int{e.rng.IntN(30), e.rng.IntN(4), e.rng.IntN(3), 1 + e.rng.IntN(30), e.rng.IntN(20), 18 + e.rng.IntN(10)}
			rows = append(rows, row)
			native = append(native, campaign.NativeClassListRow{Sprite: e.sprite(row[0], row[1], row[2]), NameTextIndex: row[3], CurrentClassTextID: row[4], TargetClassTextID: row[5]})
		}
		selected := e.rng.IntN(n + 1)
		frame, err := campaign.ComposeNativeClassListFrame(e.bg14, e.entries14[16], native, selected, e.strings, e.font)
		res := obj{"frame": oneResult(frame, err)}
		if err == nil {
			res["opening"] = framesResult(campaign.NativeClassListOpeningFrames(e.bg14, frame))
			res["closing"] = framesResult(campaign.NativeClassListClosingFrames(e.bg14, frame))
		}
		e.add("class_list", obj{"rows": rows, "selected": selected}, res)
	}
	for k := 0; k < 12; k++ {
		n := 1 + e.rng.IntN(6)
		var rows [][]int
		var native []campaign.NativeRosterRow
		for i := 0; i < n; i++ {
			row := []int{e.rng.IntN(30), e.rng.IntN(4), e.rng.IntN(3), 1 + e.rng.IntN(30)}
			rows = append(rows, row)
			native = append(native, campaign.NativeRosterRow{Sprite: e.sprite(row[0], row[1], row[2]), NameTextIndex: row[3]})
		}
		selected := e.rng.IntN(n + 1)
		for _, names := range []bool{true, false} {
			var frame []byte
			var err error
			if names {
				frame, err = campaign.ComposeNativeRosterFrame(e.bg14, e.entries14[16], native, selected, e.strings, e.font)
			} else {
				frame, err = campaign.ComposeNativeRosterFrameWithoutNames(e.bg14, e.entries14[16], native, selected)
			}
			e.add("roster", obj{"rows": rows, "selected": selected, "names": names}, oneResult(frame, err))
		}
	}
	for _, name := range []int{1, 10, 25} {
		for selected := 0; selected <= 2; selected++ {
			for pulse := 0; pulse <= 1; pulse++ {
				frame, err := campaign.ComposeNativeClassConfirmationFrame(e.bg14, e.cells2, e.strings, e.font, name, selected, pulse)
				e.add("class_confirm", obj{"name": name, "selected": selected, "pulse": pulse}, oneResult(frame, err))
			}
		}
		question, err := campaign.ComposeNativeClassConfirmationQuestion(e.bg14, e.strings, e.font, name)
		res := obj{"question": oneResult(question, err)}
		if err == nil {
			res["opening"] = framesResult(campaign.NativeClassConfirmationOpeningFrames(question, e.cells2))
			res["closing"] = framesResult(campaign.NativeClassConfirmationClosingFrames(question, e.cells2))
		}
		e.add("class_confirm_question", obj{"name": name}, res)
	}
}

// ---- 整備與戰鬥系統問句 ----

func (e *exporter) genPreparation() {
	p := e.portrait(0x4b)
	e.add("preparation", obj{"kind": "question"}, oneResult(campaign.ComposeNativePreparationConfirmationQuestion(e.bg14, e.dialogue, p, e.strings, e.font)))
	e.add("preparation", obj{"kind": "departure"}, oneResult(campaign.ComposeNativePreparationDepartureQuestion(e.bg14, e.dialogue, p, e.strings, e.font)))
	e.add("preparation", obj{"kind": "record"}, oneResult(campaign.ComposeNativePreparationRecordQuestion(e.bg14, e.dialogue, p, e.strings, e.font)))
	for _, identity := range []int{0, 1, 5, 31, 32} {
		e.add("preparation", obj{"kind": "required", "identity": identity}, oneResult(campaign.ComposeNativePreparationRequiredMessage(e.bg14, e.dialogue, p, e.strings, e.font, identity)))
	}
	for selected := 0; selected <= 1; selected++ {
		for pulse := 0; pulse <= 1; pulse++ {
			e.add("preparation", obj{"kind": "frame", "selected": selected, "pulse": pulse}, oneResult(campaign.ComposeNativePreparationConfirmationFrame(e.bg14, e.cells2, e.dialogue, p, e.strings, e.font, selected, pulse)))
		}
	}
	dialogue := must(campaign.ComposeNativePreparationConfirmationDialogue(e.bg14, e.dialogue, p))
	question := must(campaign.ComposeNativePreparationConfirmationQuestion(e.bg14, e.dialogue, p, e.strings, e.font))
	e.add("preparation", obj{"kind": "opening"}, framesResult(campaign.NativePreparationConfirmationOpeningFrames(e.bg14, dialogue, question, e.cells2)))
	e.add("preparation", obj{"kind": "closing"}, framesResult(campaign.NativePreparationConfirmationClosingFrames(e.bg14, dialogue, question, e.cells2)))
	e.add("preparation", obj{"kind": "portrait"}, oneResult(campaign.ComposeNativeConfirmationPortrait(question, p)))
	rect, mirror, err := campaign.NativePreparationPortraitLayout(p)
	e.add("preparation_layout", obj{}, obj{"err": errText(err), "rect": []int{rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y}, "mirror": mirror})

	base := must(campaign.ComposeNativeChurchDialogueOverlayAt(e.bg14, e.dialogue, p, campaign.NativeFacilityPortraitOffset(0x4b)))
	questions := map[string]func() ([]byte, error){
		"end":  func() ([]byte, error) { return campaign.ComposeNativeBattleEndTurnQuestion(base, p, e.strings, e.font) },
		"exit": func() ([]byte, error) { return campaign.ComposeNativeBattleExitQuestion(base, p, e.strings, e.font) },
		"save": func() ([]byte, error) {
			return campaign.ComposeNativeBattleCurrentSaveQuestion(base, p, e.strings, e.font)
		},
		"load": func() ([]byte, error) {
			return campaign.ComposeNativeBattleCurrentLoadQuestion(base, p, e.strings, e.font)
		},
		"march": func() ([]byte, error) {
			return campaign.ComposeNativeBattleGroupMarchQuestion(base, p, e.strings, e.font)
		},
		"discard": func() ([]byte, error) {
			return campaign.ComposeNativeDeathRewardDiscardQuestion(base, p, e.strings, e.font)
		},
		"treasure": func() ([]byte, error) {
			return campaign.ComposeNativeTreasureItemQuestion(base, p, e.strings, e.font, false)
		},
		"treasure_hidden": func() ([]byte, error) {
			return campaign.ComposeNativeTreasureItemQuestion(base, p, e.strings, e.font, true)
		},
	}
	kinds := []string{"discard", "end", "exit", "load", "march", "save", "treasure", "treasure_hidden"}
	for _, kind := range kinds {
		e.add("battle_question", obj{"kind": kind}, oneResult(questions[kind]()))
	}
	question = must(questions["end"]())
	responses := map[string]func(bool) ([][]byte, error){
		"end": func(a bool) ([][]byte, error) {
			return campaign.NativeBattleEndTurnResponseFrames(question, e.strings, e.font, a)
		},
		"exit": func(a bool) ([][]byte, error) {
			return campaign.NativeBattleExitResponseFrames(question, e.strings, e.font, a)
		},
		"save": func(a bool) ([][]byte, error) {
			return campaign.NativeBattleCurrentSaveResponseFrames(question, e.strings, e.font, a)
		},
		"load": func(a bool) ([][]byte, error) {
			return campaign.NativeBattleCurrentLoadResponseFrames(question, e.strings, e.font, a)
		},
		"march": func(a bool) ([][]byte, error) {
			return campaign.NativeBattleGroupMarchResponseFrames(question, e.strings, e.font, a)
		},
	}
	for _, kind := range []string{"end", "exit", "load", "march", "save"} {
		for _, accepted := range []bool{true, false} {
			e.add("battle_response", obj{"kind": kind, "accepted": accepted}, framesResult(responses[kind](accepted)))
		}
	}
	for _, accepted := range []bool{true, false} {
		e.add("battle_end_response", obj{"accepted": accepted}, oneResult(campaign.ComposeNativeBattleEndTurnResponse(question, e.strings, e.font, accepted)))
	}
	for _, item := range []int{-1, 0, 0x5a, 0xc0, 0xff, 0x100} {
		for _, hidden := range []bool{false, true} {
			e.add("treasure_item", obj{"item": item, "hidden": hidden}, framesResult(campaign.NativeTreasureItemResponseFrames(question, e.strings, e.font, item, hidden)))
		}
		e.add("death_abandon", obj{"item": item}, framesResult(campaign.NativeDeathRewardAbandonResponseFrames(question, e.strings, e.font, item)))
		e.add("death_cancel", obj{"item": item}, oneResult(campaign.ComposeNativeDeathRewardSelectorCancel(base, p, e.strings, e.font, item)))
	}
	for _, value := range []int{-1, 0, 7, 150, 65535, 65536} {
		for _, hidden := range []bool{false, true} {
			e.add("treasure_gold", obj{"value": value, "hidden": hidden}, framesResult(campaign.NativeTreasureGoldResponseFrames(question, e.strings, e.font, value, hidden)))
		}
	}
}

// ---- 讀檔欄與系統資訊 ----

func (e *exporter) genLoadAndSystem() {
	for k := 0; k < 10; k++ {
		var slots [4]campaign.NativeLoadSlot
		var in [][2]int
		for i := range slots {
			empty := e.rng.IntN(3) == 0
			chapter := e.rng.IntN(30)
			slots[i] = campaign.NativeLoadSlot{Empty: empty, Chapter: byte(chapter)}
			flag := 0
			if empty {
				flag = 1
			}
			in = append(in, [2]int{flag, chapter})
		}
		selection := e.rng.IntN(5)
		frame, err := campaign.ComposeNativeLoadSlotsFrame(e.bg14, e.loadBox, e.strings, e.font, slots, selection)
		e.add("load_slots", obj{"slots": in, "selection": selection}, oneResult(frame, err))
	}
	assets := campaign.NativeSystemInfoAssets{Panels: e.sysPanels, Numbers: e.itemPack, Font: e.itemPack.Font}
	for k := 0; k < 10; k++ {
		in := campaign.NativeSystemInfoInput{
			RawChapter:    byte(e.rng.IntN(31)),
			RawRound:      byte(e.rng.IntN(256)),
			Currency:      uint32(e.rng.IntN(100_000_001)),
			RawCampCounts: [3]int{e.rng.IntN(101) - 1, e.rng.IntN(100), e.rng.IntN(100)},
		}
		lines := [2]int{0x250 + e.rng.IntN(32), 0x270 + e.rng.IntN(32)}
		in.Lines[0] = must(e.strings.Words(lines[0]))
		in.Lines[1] = must(e.strings.Words(lines[1]))
		frame, err := campaign.ComposeNativeSystemInfoSurface(assets, in)
		e.add("system_info", obj{
			"chapter": int(in.RawChapter), "round": int(in.RawRound), "currency": int(in.Currency),
			"camps": in.RawCampCounts, "lines": lines,
		}, oneResult(frame, err))
	}
}

// ---- 商店 ----

func (e *exporter) genShop() {
	resourceOf := map[int]int{1: 12, 3: 29, 5: 63}
	portraitOf := map[int]int{1: 0x80, 3: 0x82, 5: 0x84}
	for _, src := range []string{"pack", "archive"} {
		for _, r := range []int{12, 13, 29, 63} {
			shop := e.shops[fmt.Sprintf("%s%d", src, r)]
			for _, pid := range []int{0x80, 0x81, 0x84, 0x4b} {
				p := e.portrait(pid)
				frame, err := campaign.ComposeNativeShopScene(shop, e.dialogue, e.digits, p, pid, e.strings, e.font, 1000*r, 0x1b8)
				e.add("shop_scene", obj{"shop": src, "resource": r, "portrait": pid, "gold": 1000 * r, "text": 0x1b8, "text_layer": true}, oneResult(frame, err))
				frame, err = campaign.ComposeNativeShopSceneWithoutText(shop, e.dialogue, e.digits, p, pid, 99999999)
				e.add("shop_scene", obj{"shop": src, "resource": r, "portrait": pid, "gold": 99999999, "text": 0, "text_layer": false}, oneResult(frame, err))
			}
			for _, gold := range []int{0, 1000, 99999999, -1} {
				frame, err := campaign.ComposeNativeShopBareScene(shop, e.digits, gold)
				e.add("shop_bare", obj{"shop": src, "resource": r, "gold": gold}, oneResult(frame, err))
			}
		}
	}
	for _, r := range []int{12, 29, 63} {
		shop := e.shops[fmt.Sprintf("pack%d", r)]
		stable := must(campaign.ComposeNativeShopBareScene(shop, e.digits, 4321))
		for step := -1; step <= 4; step++ {
			e.add("shop_service", obj{"resource": r, "kind": "opening", "step": step}, oneResult(campaign.ComposeNativeShopServiceOpeningFrame(stable, shop, step)))
			e.add("shop_service", obj{"resource": r, "kind": "closing", "step": step}, oneResult(campaign.ComposeNativeShopServiceClosingFrame(stable, shop, step)))
		}
		for selected := -1; selected <= 4; selected++ {
			for phase := 0; phase <= 4; phase++ {
				e.add("shop_service", obj{"resource": r, "kind": "steady", "selected": selected, "phase": phase}, oneResult(campaign.ComposeNativeShopServiceSteadyFrame(stable, shop, selected, phase)))
			}
		}
		for k := 0; k < 6; k++ {
			n := 1 + e.rng.IntN(10)
			items := make([]int, n)
			for i := range items {
				items[i] = e.itemIDs[e.rng.IntN(len(e.itemIDs))]
			}
			start := 2 * e.rng.IntN(3)
			selected := e.rng.IntN(n + 1)
			mode := e.rng.IntN(2)
			for _, names := range []bool{true, false} {
				var frame []byte
				var err error
				if names {
					frame, err = campaign.ComposeNativeShopItemListFrame(stable, shop, e.itemArch, items, start, selected, e.effectRows, battle.NativeFacilityPriceMode(mode))
				} else {
					frame, err = campaign.ComposeNativeShopItemListFrameWithoutNames(stable, shop, e.itemArch, items, start, selected, e.effectRows, battle.NativeFacilityPriceMode(mode))
				}
				e.add("shop_item_list", obj{"resource": r, "items": items, "start": start, "selected": selected, "mode": mode, "names": names}, oneResult(frame, err))
			}
		}
	}
	for _, variant := range []int{1, 3, 5} {
		r := resourceOf[variant]
		shop := e.shops[fmt.Sprintf("pack%d", r)]
		pid := portraitOf[variant]
		p := e.portrait(pid)
		source := must(campaign.ComposeNativeShopScene(shop, e.dialogue, e.digits, p, pid, e.strings, e.font, 12345678, 0x1f5))
		for message := 0; message <= 4; message++ {
			for _, hub := range []int{variant, variant - 1, 6} {
				for _, item := range []int{0, 0x80, 0xc0} {
					price := 50 + item
					frame, err := campaign.ComposeNativeShopPurchaseMessage(source, e.dialogue, p, pid, e.strings, e.font, campaign.NativeShopPurchaseMessage(message), hub, item, price)
					e.add("shop_purchase_msg", obj{"variant": variant, "message": message, "hub": hub, "item": item, "price": price}, oneResult(frame, err))
				}
			}
			for selected := 0; selected <= 1; selected++ {
				for pulse := 0; pulse <= 1; pulse++ {
					frame, err := campaign.ComposeNativeShopPurchaseConfirmation(source, e.dialogue, p, pid, e.strings, e.font, e.cells2, campaign.NativeShopPurchaseMessage(message), variant, 3, 120, selected, pulse)
					e.add("shop_purchase_confirm", obj{"variant": variant, "message": message, "selected": selected, "pulse": pulse}, oneResult(frame, err))
				}
			}
		}
		question := must(campaign.ComposeNativeShopPurchaseMessage(source, e.dialogue, p, pid, e.strings, e.font, campaign.NativeShopPurchaseQuestion, variant, 0, 50))
		closing := must(campaign.NativeClassConfirmationClosingFrames(question, e.cells2))
		for _, hub := range []int{variant, 0, 6} {
			post := append([]byte(nil), closing[len(closing)-1]...)
			frame, err := campaign.ComposeNativeShopPurchaseInsufficientGold(post, e.strings, e.font, hub)
			e.add("shop_insufficient", obj{"variant": variant, "hub": hub}, obj{"result": oneResult(frame, err), "post": digest(post)})
		}
		for _, hub := range []int{0, 1, 4, 6} {
			e.add("shop_recipient_full", obj{"variant": variant, "hub": hub, "name": 3}, oneResult(campaign.ComposeNativeShopPurchaseRecipientFull(source, e.dialogue, p, pid, e.strings, e.font, hub, 3)))
		}
		for _, hub := range []int{0, 3, 6} {
			e.add("shop_sell_question", obj{"variant": variant, "hub": hub}, oneResult(campaign.ComposeNativeShopSellQuestion(source, e.dialogue, p, pid, e.strings, e.font, e.cells2, hub, 0x81, 75, 1, 0)))
			e.add("shop_sell_base", obj{"variant": variant, "hub": hub}, oneResult(campaign.ComposeNativeShopSellQuestionBase(source, e.dialogue, p, pid, e.strings, e.font, hub, 0x81, 75)))
			e.add("shop_sell_empty", obj{"variant": variant, "hub": hub}, oneResult(campaign.ComposeNativeShopSellEmpty(source, e.dialogue, p, pid, e.strings, e.font, hub, 4)))
		}
		for message := 0; message <= 3; message++ {
			for _, name := range []int{-1, 6} {
				frame, err := campaign.ComposeNativeShopTransferMessage(source, e.dialogue, p, pid, e.strings, e.font, campaign.NativeShopTransferMessage(message), name)
				e.add("shop_transfer", obj{"variant": variant, "message": message, "name": name}, oneResult(frame, err))
			}
		}
		bare := must(campaign.ComposeNativeShopBareScene(shop, e.digits, 1000))
		for _, hub := range []int{variant, 4} {
			animation, final, err := campaign.ComposeNativeShopPurchaseSuccessFrames(bare, shop, p, pid, hub)
			if err == nil {
				animation = append(animation, final)
			}
			e.add("shop_success", obj{"variant": variant, "hub": hub}, framesResult(animation, err))
		}
		var rows []campaign.NativeRosterRow
		var rowIn [][]int
		for i := 0; i < 1+e.rng.IntN(6); i++ {
			row := []int{e.rng.IntN(30), 0, 0, 1 + e.rng.IntN(30)}
			rowIn = append(rowIn, row)
			rows = append(rows, campaign.NativeRosterRow{Sprite: e.sprite(row[0], row[1], row[2]), NameTextIndex: row[3]})
		}
		for _, itemType := range []int{0x1f, 0x20} {
			frame, err := campaign.ComposeNativeShopConsumableRecipientFrame(source, shop, rows, 0, itemType, e.strings, e.font)
			e.add("shop_consumable_recipient", obj{"variant": variant, "rows": rowIn, "item_type": itemType}, oneResult(frame, err))
		}
		var equipRows []campaign.NativeShopEquipmentRecipientRow
		var equipIn []obj
		for i := 0; i < 1+e.rng.IntN(3); i++ {
			group, name := e.rng.IntN(30), 1+e.rng.IntN(30)
			var cur, cand [4]int
			for s := range cur {
				cur[s], cand[s] = e.rng.IntN(300), e.rng.IntN(300)
			}
			equipIn = append(equipIn, obj{"group": group, "name": name, "current": cur, "candidate": cand})
			equipRows = append(equipRows, campaign.NativeShopEquipmentRecipientRow{Sprite: e.sprite(group, 0, 0), NameTextIndex: name, Current: cur, Candidate: cand})
		}
		for selected := 0; selected <= len(equipRows); selected++ {
			for _, names := range []bool{true, false} {
				var frame []byte
				var err error
				if names {
					frame, err = campaign.ComposeNativeShopEquipmentRecipientFrame(source, shop, e.itemArch, equipRows, selected)
				} else {
					frame, err = campaign.ComposeNativeShopEquipmentRecipientFrameWithoutNames(source, shop, e.itemArch, equipRows, selected)
				}
				e.add("shop_equipment_recipient", obj{"variant": variant, "rows": equipIn, "selected": selected, "names": names}, oneResult(frame, err))
			}
		}
	}
	for hub := 0; hub <= 6; hub++ {
		plan, err := campaign.PlanNativeShopPurchaseSuccess(hub)
		e.add("shop_success_plan", obj{"hub": hub}, obj{"err": errText(err), "plan": plan})
	}
	for message := 0; message <= 5; message++ {
		for hub := -1; hub <= 6; hub++ {
			index, ok := campaign.NativeShopPurchaseTextIndex(campaign.NativeShopPurchaseMessage(message), hub)
			e.add("shop_text_index", obj{"message": message, "hub": hub}, obj{"index": index, "ok": ok})
		}
	}
	for id := 0x7e; id <= 0x86; id++ {
		e.add("facility_portrait_offset", obj{"id": id}, campaign.NativeFacilityPortraitOffset(id))
	}
}

// ---- 金額捲動 ----

func (e *exporter) genGold() {
	shop := e.shops["pack12"]
	base := must(campaign.ComposeNativeShopBareScene(shop, e.digits, 0))
	pairs := [][3]int{{1000, 50, 0}, {0, 37, 1}, {10_000_000, 1, 0}, {9_999_999, 1, 1}, {100, 0, 0}, {100, 0, 1}, {100, 101, 0}, {99_999_999, 1, 1}, {-1, 1, 1}}
	for k := 0; k < 10; k++ {
		old := e.rng.IntN(100_000_000)
		credit := e.rng.IntN(2)
		limit := old
		if credit == 1 {
			limit = 99_999_999 - old
		}
		pairs = append(pairs, [3]int{old, e.rng.IntN(min(limit, 5000) + 1), credit})
	}
	for _, p := range pairs {
		var frames [][]byte
		var next int
		var err error
		if p[2] == 1 {
			frames, next, err = campaign.ComposeNativeGoldCreditFrames(base, shop.GoldRollStrip, p[0], p[1])
		} else {
			frames, next, err = campaign.ComposeNativeGoldDebitFrames(base, shop.GoldRollStrip, p[0], p[1])
		}
		res := framesResult(frames, err)
		res["next"] = next
		e.add("gold_roll", obj{"old": p[0], "amount": p[1], "credit": p[2] == 1}, res)
	}
	for k := 0; k < 40; k++ {
		old, next := e.rng.IntN(100_000_000), e.rng.IntN(100_000_000)
		if k%5 == 0 {
			next = old + e.rng.IntN(200) - 100
		}
		credit := e.rng.IntN(2) == 1
		windows, err := campaign.PlanNativeGoldRoll(old, next, credit)
		rows := make([][]any, len(windows))
		for i, w := range windows {
			rows[i] = []any{w.Rows, w.Active}
		}
		e.add("gold_plan", obj{"old": old, "next": next, "credit": credit}, obj{"err": errText(err), "windows": rows})
	}
}

// ---- 城鎮 ----

func (e *exporter) genTown() {
	for _, src := range []string{"pack", "archive"} {
		assets := e.towns[src]
		for variant := 0; variant < 3; variant++ {
			for selection := 0; selection < 6; selection++ {
				for pulse := 0; pulse < 4; pulse++ {
					frame, err := campaign.ComposeNativeTownFrame(assets, e.strings, e.font, variant, selection, pulse)
					e.add("town_frame", obj{"town": src, "variant": variant, "selection": selection, "pulse": pulse, "cursor": true}, oneResult(frame, err))
					if src == "pack" {
						frame, err = campaign.ComposeNativeTownFrameWithoutCursor(assets, e.strings, e.font, variant, selection, pulse)
						e.add("town_frame", obj{"town": src, "variant": variant, "selection": selection, "pulse": pulse, "cursor": false}, oneResult(frame, err))
					}
				}
			}
		}
	}
	for _, s := range [][3]int{{-1, 0, 0}, {3, 0, 0}, {0, 6, 0}, {0, 0, 4}, {0, -1, 0}} {
		frame, err := campaign.ComposeNativeTownFrame(e.towns["pack"], e.strings, e.font, s[0], s[1], s[2])
		e.add("town_frame", obj{"town": "pack", "variant": s[0], "selection": s[1], "pulse": s[2], "cursor": true}, oneResult(frame, err))
	}
	for variant := -1; variant <= 3; variant++ {
		for selection := -1; selection <= 6; selection++ {
			for pulse := -1; pulse <= 4; pulse++ {
				g, err := campaign.NativeTownCursorGeometryFor(variant, selection, pulse)
				e.add("town_cursor", obj{"variant": variant, "selection": selection, "pulse": pulse}, obj{
					"err": errText(err), "position": []int{g.Position.X, g.Position.Y}, "frame": g.Frame,
					"viewport": []int{g.Viewport.Min.X, g.Viewport.Min.Y, g.Viewport.Max.X, g.Viewport.Max.Y},
				})
			}
		}
	}
	town := must(campaign.ComposeNativeTownFrame(e.towns["pack"], e.strings, e.font, 1, 2, 0))
	for variant := -1; variant <= 3; variant++ {
		for selection := -1; selection <= 6; selection++ {
			for step := 0; step <= 11; step++ {
				g, err := campaign.NativeTownDepartureGeometryFor(variant, selection, step)
				e.add("town_departure_geometry", obj{"variant": variant, "selection": selection, "step": step}, obj{
					"err": errText(err), "center": []int{g.Center.X, g.Center.Y}, "spacing": g.Spacing,
				})
				if err == nil && (step == 1 || step == 5 || step == 10) {
					frame, err := campaign.ComposeNativeTownDepartureFrame(town, variant, selection, step)
					e.add("town_departure", obj{"variant": variant, "selection": selection, "step": step}, oneResult(frame, err))
				}
			}
		}
	}
}

// ---- 規則：游標、視窗與表格 ----

func (e *exporter) genSelectors() {
	var church, confirm, candidate, three, twoSel, twoWin, names [][]int
	for current := -2; current <= 5; current++ {
		for delta := -2; delta <= 2; delta++ {
			church = append(church, []int{current, delta, campaign.AdvanceNativeChurchServiceSelection(current, delta)})
			confirm = append(confirm, []int{current, delta, campaign.AdvanceNativeClassConfirmation(current, delta)})
		}
	}
	for count := -1; count <= 8; count++ {
		for selected := -1; selected <= 9; selected++ {
			s, v := campaign.NativeClassCandidateWindow(count, selected)
			candidate = append(candidate, []int{count, selected, s, v})
			for start := -1; start <= 9; start++ {
				s, v := campaign.NativeThreeRowWindow(count, selected, start)
				three = append(three, []int{count, selected, start, s, v})
			}
			for delta := -3; delta <= 3; delta++ {
				twoSel = append(twoSel, []int{selected, count, delta, campaign.AdvanceNativeTwoColumnSelection(selected, count, delta)})
			}
		}
	}
	for count := -1; count <= 13; count++ {
		for selected := -1; selected <= 13; selected++ {
			for start := -1; start <= 10; start++ {
				s, v := campaign.NativeTwoColumnWindow(count, selected, start)
				twoWin = append(twoWin, []int{count, selected, start, s, v})
			}
		}
	}
	var nameOut []string
	for id := -2; id <= 31; id++ {
		names = append(names, []int{id})
		nameOut = append(nameOut, campaign.ClassName(id))
	}
	e.add("selectors", obj{}, obj{
		"church": church, "confirm": confirm, "candidate": candidate, "three": three,
		"two_selection": twoSel, "two_window": twoWin, "class_names": nameOut,
	})
}

func intMap[V any](m map[int]V) [][]any {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	out := make([][]any, len(keys))
	for i, k := range keys {
		out[i] = []any{k, m[k]}
	}
	return out
}

func (e *exporter) genTables() {
	rel := func(p string) string { return filepath.Join(e.repo, p) }
	rates, err := campaign.LoadReviveFeeRates(rel("docs/data/exe_tables/revive_fee_rates.json"))
	e.add("table_revive_fee", obj{"path": "docs/data/exe_tables/revive_fee_rates.json"}, obj{"err": errText(err), "rates": rates})
	rates, err = campaign.LoadReviveFeeRates(rel("remake/assets/data/revive_fee_rates.json"))
	e.add("table_revive_fee", obj{"path": "remake/assets/data/revive_fee_rates.json"}, obj{"err": errText(err), "rates": rates})
	for _, p := range []string{"docs/data/exe_tables/class_change_targets.json", "remake/assets/data/class_change_targets.json"} {
		t, err := campaign.LoadClassChangeTable(rel(p))
		e.add("table_class_change", obj{"path": p}, obj{"err": errText(err), "current": intMap(t.Current), "targets": intMap(t.Targets)})
	}
	for _, p := range []string{"docs/data/exe_tables/growth.json", "remake/assets/data/class_change_growth.json"} {
		g, err := campaign.LoadClassChangeGrowth(rel(p))
		e.add("table_growth", obj{"path": p}, obj{"err": errText(err), "rows": intMap(g)})
	}
	stats, err := campaign.LoadItemStats(rel("remake/assets/data/item.json"))
	e.add("table_item_stats", obj{"path": "remake/assets/data/item.json"}, obj{"err": errText(err), "rows": intMap(stats)})
	prices, err := campaign.LoadItemPrices(rel("remake/assets/data/item.json"))
	e.add("table_item_prices", obj{"path": "remake/assets/data/item.json"}, obj{"err": errText(err), "rows": intMap(prices)})
	types, equip, err := campaign.LoadShopEligibility(rel("remake/assets/data/item.json"), rel("remake/assets/data/class_equip_types.json"))
	e.add("table_shop_eligibility", obj{}, obj{"err": errText(err), "types": intMap(types), "equip": intMap(equip)})
	var canEquip [][]any
	for class := -1; class <= 30; class++ {
		for itemType := 0; itemType <= 33; itemType++ {
			canEquip = append(canEquip, []any{class, itemType, campaign.CanEquip(class, itemType, equip)})
		}
	}
	e.add("can_equip", obj{}, canEquip)
}

// ---- 規則：單位 ----

func (e *exporter) randItem() int {
	switch e.rng.IntN(10) {
	case 0:
		return 0xfe // 不在 item.json
	default:
		return e.itemIDs[e.rng.IntN(len(e.itemIDs))]
	}
}

// randUnit 產生一個單位：數值、背包與（多數時候一致的）八格原生投影。
func (e *exporter) randUnit() battle.Unit {
	r := e.rng
	u := battle.Unit{
		Fig: r.IntN(32), Lv: r.IntN(42) - 1, Portrait: r.IntN(0x24), ClassID: r.IntN(29),
		HP: r.IntN(200), MaxHP: 1 + r.IntN(300), MP: r.IntN(80), MaxMP: r.IntN(120),
		AP: r.IntN(300), DP: r.IntN(300), DX: r.IntN(200), HIT: r.IntN(200), EV: r.IntN(200), MV: 3 + r.IntN(5),
		AtkMin: r.IntN(2), AtkMax: 1 + r.IntN(2), Exp: float64(r.IntN(100)),
		BattleFig: r.IntN(0x80), HasBattleFig: r.IntN(4) != 0,
		NativeRecordByte5: byte(r.IntN(4)), HasNativeRecordByte5: r.IntN(5) != 0,
		NativeRecordByte6: byte(r.IntN(4)), HasNativeRecordByte6: r.IntN(5) != 0,
		NativeRecordRace: byte(r.IntN(12)), HasNativeRecordRace: r.IntN(5) != 0,
		NativeRecordClass: byte(r.IntN(29)), HasNativeRecordClass: r.IntN(2) == 0,
		MapSelectorKey: r.IntN(0x40), HasMapSelectorKey: r.IntN(2) == 0,
		MapSelectorSlot: r.IntN(8), HasMapSelectorSlot: r.IntN(2) == 0,
		OnField:        r.IntN(2) == 0,
		NativeIdentity: r.IntN(32), HasNativeIdentity: r.IntN(2) == 0,
	}
	if r.IntN(3) == 0 {
		u.BattleFig = 0x79
	}
	n := r.IntN(9)
	for i := 0; i < n; i++ {
		u.Inventory = append(u.Inventory, e.randItem())
		u.Equipped = append(u.Equipped, r.IntN(3) == 0)
	}
	if r.IntN(6) == 0 && len(u.Equipped) > 0 {
		u.Equipped = u.Equipped[:r.IntN(len(u.Equipped))]
	}
	switch r.IntN(4) {
	case 0: // 沒有原生投影
	case 1, 2: // 一致的八格投影（洞的位置隨機）
		u.InventorySlots = make([]int, 8)
		u.NativeInventoryFlags = make([]int, 8)
		holes := 8 - len(u.Inventory)
		compact := 0
		for raw := 0; raw < 8; raw++ {
			if compact < len(u.Inventory) && (holes == 0 || r.IntN(3) != 0) {
				u.InventorySlots[raw] = u.Inventory[compact]
				if compact < len(u.Equipped) && u.Equipped[compact] {
					u.NativeInventoryFlags[raw] = 0x40
				}
				compact++
				continue
			}
			if holes > 0 && compact < len(u.Inventory) {
				holes--
			}
			u.InventorySlots[raw] = 0xff
			if r.IntN(3) == 0 {
				u.InventorySlots[raw] = r.IntN(0x100)
			}
			u.NativeInventoryFlags[raw] = 0x80
		}
		for ; compact < len(u.Inventory); compact++ {
			// 洞數用完仍有剩：最後幾格改回實際物品（投影可能失效，正好測拒收路徑）。
			u.InventorySlots[8-len(u.Inventory)+compact] = u.Inventory[compact]
			u.NativeInventoryFlags[8-len(u.Inventory)+compact] = 0
		}
	case 3: // 隨機八格（多半不一致）
		u.InventorySlots = make([]int, 8)
		u.NativeInventoryFlags = make([]int, 8)
		for raw := 0; raw < 8; raw++ {
			u.InventorySlots[raw] = r.IntN(0x100)
			u.NativeInventoryFlags[raw] = []int{0, 0x40, 0x80}[r.IntN(3)]
		}
	}
	if r.IntN(2) == 0 {
		u.EquipmentBaseSet = true
		u.BaseAP, u.BaseDP, u.BaseHIT, u.BaseEV, u.BaseMV = r.IntN(200), r.IntN(200), r.IntN(200), r.IntN(200), 3+r.IntN(5)
		u.BaseAtkMin, u.BaseAtkMax = r.IntN(2), 1+r.IntN(2)
		if r.IntN(8) == 0 {
			u.BaseAP = 0x8000 + r.IntN(10)
		}
	}
	return u
}

func unitJSON(u *battle.Unit) any {
	if u == nil {
		return nil
	}
	raw := must(json.Marshal(u))
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		panic(err)
	}
	return v
}

func cloneUnit(u battle.Unit) battle.Unit {
	raw := must(json.Marshal(u))
	var out battle.Unit
	if err := json.Unmarshal(raw, &out); err != nil {
		panic(err)
	}
	return out
}

func (e *exporter) genUnitRules() {
	r := e.rng
	rates := must(campaign.LoadReviveFeeRates(filepath.Join(e.repo, "docs/data/exe_tables/revive_fee_rates.json")))
	for k := 0; k < 60; k++ {
		u := e.randUnit()
		if r.IntN(2) == 0 {
			u.NativeRecordByte5, u.HasNativeRecordByte5 = u.NativeRecordByte5|1, true
		}
		in := unitJSON(&u)
		gold := r.IntN(60000)
		fee := rates[r.IntN(len(rates))]
		if k%10 == 0 {
			fee = -1
		}
		w := cloneUnit(u)
		g, cost, err := campaign.ReviveUnit(gold, &w, fee)
		e.add("revive", obj{"unit": in, "gold": gold, "fee": fee}, obj{"gold": g, "cost": cost, "err": errText(err), "unit": unitJSON(&w), "can_revive": campaign.CanRevive(&u), "can_change_class": campaign.CanChangeClass(&u)})
	}
	g, cost, err := campaign.ReviveUnit(100, nil, 7)
	e.add("revive", obj{"unit": nil, "gold": 100, "fee": 7}, obj{"gold": g, "cost": cost, "err": errText(err), "unit": nil, "can_revive": false, "can_change_class": false})

	for k := 0; k < 60; k++ {
		u := e.randUnit()
		in := unitJSON(&u)
		branch, ok := campaign.NativeClassChangeTarget(&u, e.classTable)
		target, class, group := branch.Portrait, branch.ClassID, branch.MobilityIncrement
		remove := branch.InventoryIndex
		if !ok || k%7 == 0 {
			target, class, group, remove = r.IntN(0x110)-8, r.IntN(0x110)-8, r.IntN(5), r.IntN(10)-2
		}
		row, hasRow := e.growth[target]
		if !hasRow || k%9 == 0 {
			row = campaign.ClassChangeGrowth{AP: [2]int{r.IntN(10), r.IntN(10)}, DP: [2]int{3, 3}, DX: [2]int{1, 4}, HP: [2]int{5, 9}, MP: [2]int{0, 0}}
		}
		var seed any
		var rng *mrand.Rand
		if k%11 != 5 {
			s := int64(r.IntN(1 << 30))
			seed, rng = s, mrand.New(mrand.NewSource(s))
		}
		w := cloneUnit(u)
		err := campaign.ApplyClassChange(&w, target, class, group, row, rng, remove)
		after := unitJSON(&w)
		w2 := cloneUnit(w)
		campaign.RecomputeAfterClassChange(&w2, e.stats)
		e.add("class_change", obj{"unit": in, "target": target, "class": class, "group": group, "row": row, "seed": seed, "remove": remove},
			obj{"branch": branch, "ok": ok, "err": errText(err), "unit": after, "recomputed": unitJSON(&w2)})
	}

	for k := 0; k < 15; k++ {
		roster := map[int]battle.Unit{}
		var rosterIn []obj
		var ids []int
		for i := 0; i < 1+r.IntN(12); i++ {
			id := r.IntN(40)
			if _, dup := roster[id]; dup {
				continue
			}
			u := battle.Unit{Fig: id, Lv: 15 + r.IntN(10), Portrait: r.IntN(0x16)}
			roster[id] = u
			rosterIn = append(rosterIn, obj{"id": id, "unit": unitJSON(&u)})
			ids = append(ids, id)
		}
		sort.Slice(rosterIn, func(a, b int) bool { return rosterIn[a]["id"].(int) < rosterIn[b]["id"].(int) })
		order := append([]int(nil), ids...)
		r.Shuffle(len(order), func(a, b int) { order[a], order[b] = order[b], order[a] })
		full := k%3 != 0
		if !full {
			order = order[:len(order)/2]
			order = append(append(order, 99), order[0:min(1, len(order))]...)
		}
		e.add("candidates", obj{"roster": rosterIn, "order": order, "full": full}, campaign.ClassChangeCandidates(roster, order))
	}

	type unitOp struct {
		name string
		run  func(u *battle.Unit, in obj) obj
	}
	ops := []unitOp{
		{"buy", func(u *battle.Unit, in obj) obj {
			good := campaign.Good{ID: e.randItem(), Name: "x", Price: r.IntN(500) - 20}
			gold := r.IntN(600)
			in["good"], in["gold"] = obj{"id": good.ID, "price": good.Price}, gold
			g, err := campaign.BuyGood(gold, u, good)
			return obj{"gold": g, "err": errText(err)}
		}},
		{"reserve", func(u *battle.Unit, in obj) obj {
			good := campaign.Good{ID: r.IntN(0x110) - 8, Name: "x", Price: r.IntN(500)}
			gold := r.IntN(600)
			in["good"], in["gold"] = obj{"id": good.ID, "price": good.Price}, gold
			slot, err := campaign.ReserveGood(gold, u, good)
			return obj{"slot": slot, "err": errText(err), "finalized": campaign.FinalizeGood(gold, good)}
		}},
		{"sell_good", func(u *battle.Unit, in obj) obj {
			item := e.randItem()
			if len(u.Inventory) > 0 && r.IntN(2) == 0 {
				item = u.Inventory[r.IntN(len(u.Inventory))]
			}
			gold, price := r.IntN(600), r.IntN(400)-10
			in["item"], in["gold"], in["price"] = item, gold, price
			g, err := campaign.SellGood(gold, u, item, price)
			return obj{"gold": g, "err": errText(err)}
		}},
		{"sell_slot", func(u *battle.Unit, in obj) obj {
			slot, gold, price := r.IntN(10)-1, r.IntN(600), r.IntN(400)-10
			in["slot"], in["gold"], in["price"] = slot, gold, price
			g, err := campaign.SellSlot(gold, u, slot, price)
			return obj{"gold": g, "err": errText(err)}
		}},
		{"sell_native", func(u *battle.Unit, in obj) obj {
			slot, gold, price := r.IntN(10)-1, r.IntN(600), r.IntN(400)-10
			in["slot"], in["gold"], in["price"] = slot, gold, price
			g, err := campaign.SellNativeSlot(gold, u, slot, price, e.stats)
			return obj{"gold": g, "err": errText(err)}
		}},
		{"equip", func(u *battle.Unit, in obj) obj {
			slot := r.IntN(10) - 1
			in["slot"] = slot
			return obj{"err": errText(campaign.EquipItem(u, slot, e.stats))}
		}},
		{"equip_native", func(u *battle.Unit, in obj) obj {
			slot := r.IntN(10) - 1
			in["slot"] = slot
			return obj{"err": errText(campaign.EquipNativeCompactSlot(u, slot, e.stats))}
		}},
		{"recompute", func(u *battle.Unit, in obj) obj { campaign.RecomputeEquipment(u, e.stats); return obj{} }},
		{"attack_range", func(u *battle.Unit, in obj) obj { campaign.ApplyEquippedAttackRange(u, e.stats); return obj{} }},
		{"after_class_change", func(u *battle.Unit, in obj) obj { campaign.RecomputeAfterClassChange(u, e.stats); return obj{} }},
		{"init_base", func(u *battle.Unit, in obj) obj { campaign.InitializeEquipmentBase(u, e.stats); return obj{} }},
		{"equipment_record", func(u *battle.Unit, in obj) obj {
			if r.IntN(5) != 0 {
				u.HasNativeRecordByte6, u.HasNativeRecordRace, u.HasNativeRecordClass = true, true, true
				u.HasNativeIdentity, u.EquipmentBaseSet = r.IntN(6) != 0, r.IntN(6) != 0
				u.Exp = float64(int(u.Exp))
				if u.Lv < 0 {
					u.Lv = 0
				}
				if len(u.InventorySlots) != 8 {
					u.InventorySlots = []int{0, 0x80, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
					u.NativeInventoryFlags = []int{0x40, 0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
				}
				in["unit"] = unitJSON(u)
			}
			record, err := campaign.NativeShopEquipmentRecordForUnit(u)
			out := obj{"err": errText(err), "record": hex.EncodeToString(record)}
			if err == nil {
				item := r.IntN(0x110) - 4
				in["item"] = item
				cand, err := campaign.NativeShopEquipmentCandidateStats(record, item, e.effectRows)
				cur, cerr := campaign.NativeShopEquipmentCurrentStats(record)
				out["candidate"], out["candidate_err"], out["current"], out["current_err"] = cand, errText(err), cur, errText(cerr)
			}
			return out
		}},
	}
	for _, op := range ops {
		for k := 0; k < 40; k++ {
			u := e.randUnit()
			in := obj{"unit": unitJSON(&u)}
			res := op.run(&u, in)
			res["unit"] = unitJSON(&u)
			e.add("unit_"+op.name, in, res)
		}
	}
	for k := 0; k < 20; k++ {
		var units []*battle.Unit
		var in []any
		for i := 0; i < r.IntN(8); i++ {
			u := e.randUnit()
			if k%4 != 0 {
				u.HasNativeRecordByte5, u.HasNativeRecordByte6, u.HasBattleFig, u.HasNativeRecordRace = true, true, true, true
			}
			units = append(units, &u)
			in = append(in, unitJSON(&u))
		}
		counts, err := campaign.NativeSystemInfoCampCounts(units)
		e.add("camp_counts", obj{"units": in}, obj{"counts": counts, "err": errText(err)})
	}
	for k := 0; k < 20; k++ {
		record := make([]byte, 0x50)
		for i := range record {
			record[i] = byte(r.IntN(256))
		}
		if k%4 == 0 {
			record = record[:r.IntN(0x50)]
		}
		item := r.IntN(0x110) - 4
		cand, err := campaign.NativeShopEquipmentCandidateStats(record, item, e.effectRows)
		cur, cerr := campaign.NativeShopEquipmentCurrentStats(record)
		e.add("candidate_stats", obj{"record": hex.EncodeToString(record), "item": item}, obj{"candidate": cand, "err": errText(err), "current": cur, "current_err": errText(cerr)})
	}
}

func main() {
	out := flag.String("out", "", "輸出 JSON 路徑")
	flame := flag.String("flame2", "", "原版 FLAME2 目錄")
	pack := flag.String("pack", "", "分離素材包根目錄")
	repo := flag.String("repo", "..", "儲存庫根目錄（讀 remake/assets 與 docs）")
	seed := flag.Uint64("seed", 20261114, "PCG 種子")
	flag.Parse()
	if *out == "" || *flame == "" || *pack == "" {
		fmt.Fprintln(os.Stderr, "需要 -out、-flame2 與 -pack")
		os.Exit(2)
	}
	e := &exporter{rng: rand.New(rand.NewPCG(*seed, 0x43414d42)), flame: *flame, pack: *pack, repo: *repo}
	real := e.loadAssets()
	e.genAssets()
	e.genChurch()
	e.genClassAndRoster()
	e.genPreparation()
	e.genLoadAndSystem()
	e.genShop()
	e.genGold()
	e.genTown()
	e.genSelectors()
	e.genTables()
	e.genUnitRules()
	doc := obj{"schema": "fd2-golden-campaign-b/1", "seed": *seed, "real": real, "cases": e.cases}
	data := must(json.Marshal(doc))
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("cases=%d bytes=%d\n", len(e.cases), len(data))
}
