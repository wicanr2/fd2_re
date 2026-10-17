package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

// 正常 START 及明示啟用的章重播，酒店保存須寫出原版格式並經新 Game 正式 LOAD 冷讀。
// 只驗已知玩家狀態；完整原版差異另由有界政策逐筆核對。
func (r *parityReplay) verifyChapterSlotSave(slot int) {
	if os.Getenv("FD2_PARITY_ENTRY") != "start" && os.Getenv("FD2_PARITY_VERIFY_COLD_LOAD") != "1" {
		return
	}
	t, g := r.t, r.g
	if g.nativeChapterSlotBaseline == nil || g.nativeChapterSlotSaveErr != nil {
		t.Fatal("酒店未建立正式保存基底")
	}
	stored, err := os.ReadFile(nativeCurrentSavePath())
	if err != nil {
		t.Fatal(err)
	}
	plain, err := fdsave.Decode(stored)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := fdsave.InspectChapterSlot(plain, slot)
	if err != nil {
		t.Fatal(err)
	}
	if int(snapshot.Verified.Chapter) != g.handlerChapter || int(snapshot.Verified.RosterCount) != len(g.partyRoster) || int(snapshot.Verified.Currency) != g.gold {
		t.Fatal("章正式保存 metadata 不同")
	}
	options := g.currentNativeSystemOptions()
	// 用新 Game 經正式標題 LOAD 冷讀正式保存。只驗目前已知玩家狀態，
	// 不把此結果提升為原版完整 byte 相同或未知欄位語意已閉合。
	cold := loadGame()
	cold.titlePhase = "loadslots"
	loaded := cold.confirmTitleLoadSlot(slot)
	if !loaded || cold.loadErr != "" || cold.camp.NodeID() != r.currentTown() || cold.gold != g.gold || cold.handlerChapter != g.handlerChapter || !reflect.DeepEqual(cold.partyJoinOrder, g.partyJoinOrder) {
		t.Fatalf("章保存正常 LOAD 冷讀失敗：node=%s msg=%s err=%s", cold.camp.NodeID(), cold.msg, cold.loadErr)
	}
	for id, expected := range g.partyRoster {
		actual, exists := cold.partyRoster[id]
		if !exists || actual.HP != expected.HP || actual.MaxHP != expected.MaxHP || actual.MP != expected.MP || actual.MaxMP != expected.MaxMP || actual.Lv != expected.Lv || actual.Exp != expected.Exp || actual.AP != expected.AP || actual.DP != expected.DP || actual.DX != expected.DX || actual.HIT != expected.HIT || actual.EV != expected.EV || actual.MV != expected.MV || !reflect.DeepEqual(actual.InventorySlots, expected.InventorySlots) || !reflect.DeepEqual(actual.NativeInventoryFlags, expected.NativeInventoryFlags) {
			t.Fatalf("章保存正常 LOAD 角色 %d 玩家狀態不同", id)
		}
	}
	if !reflect.DeepEqual(cold.currentNativeSystemOptions(), options) {
		t.Fatal("章保存正常 LOAD 系統選項不同")
	}
	if raw, err := os.ReadFile(saveSlotPath(slot)); err != nil {
		t.Fatal(err)
	} else if err := os.WriteFile(filepath.Join(r.out, "hotel-save.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("正式保存與 LOAD 冷讀通過：chapter=%d roster=%d gold=%d", snapshot.Verified.Chapter, snapshot.Verified.RosterCount, snapshot.Verified.Currency)
}

// 首章READY契約：fd2_ch01_flow_20261017.json entry_contract。
// 此入口只為章重播增加正常START，不改正式標題或戰役規則。
func verifyParityStartSource(run string) error {
	var boot struct {
		SHA        string   `json:"exe_sha256"`
		Chain      []string `json:"input_chain"`
		Reads      int      `json:"kbd_reads"`
		Pending    int      `json:"kbd_pending"`
		Injections []any    `json:"state_injections"`
	}
	raw, err := os.ReadFile(filepath.Join(run, "checkpoint-0000.json"))
	if err != nil || json.Unmarshal(raw, &boot) != nil ||
		boot.SHA != "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f" ||
		boot.Reads != 0 || boot.Pending != 0 || boot.Injections == nil || len(boot.Injections) != 0 {
		return fmt.Errorf("START缺少固定原版未注入的標題初態")
	}
	title := false
	for _, address := range boot.Chain {
		title = title || address == "0x1FE60"
	}
	if !title {
		return fmt.Errorf("START缺少1FE60標題owner")
	}
	control, err := os.Open(filepath.Join(run, "control-history.jsonl"))
	if err != nil {
		return err
	}
	defer control.Close()
	scanner := bufio.NewScanner(control)
	var first struct {
		Seq int    `json:"seq"`
		Key string `json:"key"`
	}
	if !scanner.Scan() || json.Unmarshal(scanner.Bytes(), &first) != nil || first.Seq != 1 || first.Key != "enter" {
		return fmt.Errorf("START首鍵不是正常Enter；LOAD／CONTINUE不得冒稱START")
	}
	trace, err := os.Open(filepath.Join(run, "eip-trace.jsonl"))
	if err != nil {
		return err
	}
	defer trace.Close()
	scanner = bufio.NewScanner(trace)
	scanner.Buffer(make([]byte, 4096), 4<<20)
	for scanner.Scan() {
		var entry struct {
			EIP   string   `json:"eip"`
			Stack []string `json:"stack"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return err
		}
		if entry.EIP == "0x3231B" && len(entry.Stack) > 0 && entry.Stack[0] == "0x25F17" {
			return nil
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return fmt.Errorf("START缺少25F17→3231B正常序章caller")
}

func (r *parityReplay) startFromNormalTitle() {
	t, g := r.t, r.g
	t.Helper()
	if err := verifyParityStartSource(r.run); err != nil {
		t.Fatal(err)
	}
	if g.titlePhase != "cutscene" {
		t.Fatalf("正常START未從開場動畫開始：%q", g.titlePhase)
	}
	r.startEntryTrace = newJourneyTrace()
	r.startEntryTrace.Seed = "4"
	for frame := 0; g.titlePhase != "menu"; frame++ {
		if frame >= journeyTitleFrames || g.titlePhase == "" {
			t.Fatal("正常開場動畫未進標題選單")
		}
		r.startEntryTrace.observe(g)
		ackPresents(g)
		if err := parityUpdateGame(g); err != nil {
			t.Fatal(err)
		}
	}
	if g.titleSel != 0 || !g.applyTitleMenuEvent(TitleMenuConfirm) {
		t.Fatal("正常START未由標題owner消費")
	}
	r.startEntryTrace.press("title")
	if !pump(t, g, 600, func() bool { return g.titlePhase == "" }) ||
		g.camp == nil || g.camp.NodeID() != journeyOpeningNode {
		t.Fatal("START未正常進入序章")
	}
}

func TestParityStartRejectsLoadContinueAndMissingOriginalCaller(t *testing.T) {
	for _, fault := range []string{"", "load_key", "prior_reads", "exe", "title_owner", "opening_caller"} {
		t.Run(fault, func(t *testing.T) {
			dir := t.TempDir()
			boot := map[string]any{"exe_sha256": "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f",
				"input_chain": []string{"0x1FE60"}, "kbd_reads": 0, "kbd_pending": 0, "state_injections": []any{}}
			key, caller := "enter", "0x25F17"
			switch fault {
			case "load_key":
				key = "down"
			case "prior_reads":
				boot["kbd_reads"] = 1
			case "exe":
				boot["exe_sha256"] = "wrong"
			case "title_owner":
				boot["input_chain"] = []string{"0x117F8"}
			case "opening_caller":
				caller = "0x0"
			}
			for name, value := range map[string]any{
				"checkpoint-0000.json":  boot,
				"control-history.jsonl": map[string]any{"seq": 1, "key": key},
				"eip-trace.jsonl":       map[string]any{"eip": "0x3231B", "stack": []string{caller}},
			} {
				raw, err := json.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, name), append(raw, '\n'), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := verifyParityStartSource(dir); (err == nil) != (fault == "") {
				t.Fatalf("fault=%q err=%v", fault, err)
			}
		})
	}
}
