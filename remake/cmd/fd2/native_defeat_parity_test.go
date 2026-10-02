package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// #79：只承接同一原版 run 真正抵達的標題選單，不能補造語意動作。
func observedDefeatTitleSequence(raw []byte) (int, error) {
	var state struct {
		Runner     string            `json:"runner"`
		EXESHA     string            `json:"exe_sha256"`
		EIP        string            `json:"eip"`
		Seq        int               `json:"control_seq"`
		Chain      []string          `json:"input_chain"`
		Input      string            `json:"input_kind"`
		Injections []json.RawMessage `json:"state_injections"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		return 0, err
	}
	chain := []string{"0x1FE60", "0x25ECD", "0x25DC2", "0x45D91", "0x3CB91"}
	if state.Runner != "dosgolem" || state.EXESHA != "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f" ||
		state.EIP != "0x36D98" || state.Seq <= 0 || len(state.Chain) != len(chain) ||
		state.Input != "normal BIOS keys" || state.Injections == nil || len(state.Injections) != 0 {
		return 0, fmt.Errorf("原版終態不是已證實的標題選單")
	}
	for i := range chain {
		if state.Chain[i] != chain[i] {
			return 0, fmt.Errorf("原版標題輸入鏈不符：%v", state.Chain)
		}
	}
	return state.Seq, nil
}

func (r *parityReplay) verifyObservedDefeatTail() {
	if r.battle != "battle_ch15" {
		r.t.Fatal("此終態尾端規格只驗第十五章；其他章須使用既有正式 mark")
	}
	raw, err := os.ReadFile(filepath.Join(r.run, "current.json"))
	if err != nil {
		r.t.Fatal(err)
	}
	seq, err := observedDefeatTitleSequence(raw)
	if err != nil || seq <= r.prevSeq {
		r.t.Fatalf("原版敗北尾端缺少可比來源：seq=%d last=%d err=%v", seq, r.prevSeq, err)
	}
	r.verifyNativeDefeatReturnTitle(parityAction{Seq: seq})
}

func TestObservedDefeatTitleRejectsUnknownTerminal(t *testing.T) {
	valid := map[string]any{"runner": "dosgolem", "exe_sha256": "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f",
		"eip": "0x36D98", "control_seq": 1431, "input_chain": []string{"0x1FE60", "0x25ECD", "0x25DC2", "0x45D91", "0x3CB91"},
		"input_kind": "normal BIOS keys", "state_injections": []string{}}
	for _, key := range []string{"", "runner", "exe_sha256", "eip", "control_seq", "input_chain", "input_kind", "state_injections"} {
		t.Run(key, func(t *testing.T) {
			candidate := map[string]any{}
			for k, v := range valid {
				candidate[k] = v
			}
			if key != "" {
				delete(candidate, key)
			}
			raw, err := json.Marshal(candidate)
			if err != nil {
				t.Fatal(err)
			}
			seq, err := observedDefeatTitleSequence(raw)
			if key == "" && (err != nil || seq != 1431) {
				t.Fatal(seq, err)
			}
			if key != "" && err == nil {
				t.Fatal("未知原版終態不得承接敗北尾端", key)
			}
		})
	}
	for _, injections := range []any{nil, []string{"force-enemy-clear"}} {
		valid["state_injections"] = injections
		raw, _ := json.Marshal(valid)
		if _, err := observedDefeatTitleSequence(raw); err == nil {
			t.Fatal("未知或注入終態不得作為敗北證據")
		}
	}
	valid["state_injections"] = []string{}
	valid["input_chain"] = []string{"0x1FE60", "0x25ECD", "0x25DC2", "0x45D91", "0xDEADBEEF"}
	raw, _ := json.Marshal(valid)
	if _, err := observedDefeatTitleSequence(raw); err == nil {
		t.Fatal("部分輸入鏈相同不足以證明返回標題")
	}
}

// 正式章重播的敗北尾端。初始槽、玩家動作及每次 AI／攻擊的受控亂數
// 皆由既有 parityReplay 決定；此處只替換宿主時鐘，不改單位或結果。
func (r *parityReplay) verifyNativeDefeatReturnTitle(action parityAction) {
	t, g := r.t, r.g
	if !pump(t, g, ch01FrameBudget*6, func() bool {
		if len(g.dialog) > 0 && g.battleEvent != nil && storyEnterReady(g) {
			g.handleBattleEventDialogueInput(true)
		}
		return g.result != ""
	}) || g.result != "lose" || g.nativeDefeat == nil {
		t.Fatalf("正常章重播未抵達原生敗北：result=%s err=%s", g.result, g.loadErr)
	}
	if g.confirmBattleResult() {
		t.Fatal("確認鍵跳過敗北提示")
	}
	now := time.Unix(1, 0)
	screen := ebiten.NewImage(640, 400)
	seen := map[int]bool{}
	frames := []map[string]any{}
	protectedIndex := 14
	if r.battle == "battle_ch13" {
		protectedIndex = 59
	}
	if r.battle == "battle_ch15" {
		protectedIndex = 64
	}
	protected := g.st.Units[protectedIndex]
	resultRules := append([]string(nil), g.nativeResultMatchedRules...)
	resultRound := g.st.NativeRoundCounter
	initialAlliesActive := []int{}
	if r.battle == "battle_ch13" {
		for i := 15; i <= 26; i++ {
			if g.st.Units[i].NativeRecordByte5&1 == 0 {
				initialAlliesActive = append(initialAlliesActive, i)
			}
		}
	}
	for k := 0; g.nativeDefeat != nil && k < 500; k++ {
		j := g.nativeDefeat
		g.Draw(screen)
		step := j.plan[j.pos]
		if j.ramp == nil && step.Kind == fdother.PendingCode1WaitTick && (step.Count == 9 || step.Count == 36) && !seen[step.Count] {
			seen[step.Count] = true
			name := fmt.Sprintf("defeat-%d-ticks.png", step.Count)
			img := image.NewPaletted(image.Rect(0, 0, 320, 200), j.palette)
			copy(img.Pix, j.vga)
			f, err := os.Create(filepath.Join(r.out, name))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(f, img); err != nil {
				f.Close()
				t.Fatal(err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
			frames = append(frames, map[string]any{"file": name, "hold_bios_ticks": step.Count, "indexed_sha256": fmt.Sprintf("%x", sha256.Sum256(j.vga)), "dac_sha256": fmt.Sprintf("%x", sha256.Sum256(j.dac))})
		}
		g.stepNativeDefeat(now)
		if g.loadErr != "" {
			t.Fatal(g.loadErr)
		}
		now = now.Add(nativeBIOSTickPeriod)
	}
	if len(seen) != 2 || g.nativeDefeat != nil || g.titlePhase != "cutscene" || g.cutIdx != 0 || !g.titleNeedsNewCampaign || g.st != nil || g.aiBusy {
		t.Fatal("敗北兩幀／完整標題返回／戰鬥清理不完整")
	}
	// 只讓既有標題播放器自行走完；不注入 menu、不略過不可中斷片段。
	for k := 0; g.titlePhase != "menu" && k < journeyTitleFrames; k++ {
		g.titleUpdate()
	}
	if g.titlePhase != "menu" {
		t.Fatal("返回的完整標題播放器未交出選單")
	}
	r.checkpoint("defeat_return_title", action.Seq, "title", false)
	data := map[string]any{
		"schema_version": 1, "status": "passed", "method": "正常 LOAD／章內玩家動作與決策點受控 RNG；只以決定性宿主時鐘推進呈現",
		"result": "lose", "protected_record": protectedIndex, "protected_hp": protected.HP, "protected_raw_byte5": protected.NativeRecordByte5,
		"native_result_rules": resultRules, "native_round": resultRound, "initial_allies_active_records": initialAlliesActive,
		"frames": frames, "returned_title_sequence": "publisher→既有完整開場→menu", "title_selection": g.titleSel,
		"input_can_skip_defeat": false, "clock_limit": "BIOS tick 依既有規格近似；2ms DAC 每步呈現，60Hz延長亞幀等待",
		"save_unchanged": true,
	}
	slot, err := os.ReadFile(os.Getenv("FD2_PARITY_SLOT"))
	if err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(os.Getenv("FD2_NATIVE_SAVE"))
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(slot) != sha256.Sum256(saved) {
		t.Fatal("敗北改寫原生存檔")
	}
	data["slot_sha256"] = fmt.Sprintf("%x", sha256.Sum256(slot))
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.out, "defeat-presentation.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
