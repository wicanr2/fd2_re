package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// 正式章重播的敗北尾端。初始槽、玩家動作及每次 AI／攻擊的受控亂數
// 皆由既有 parityReplay 決定；此處只替換宿主時鐘，不改單位或結果。
func (r *parityReplay) verifyNativeDefeatReturnTitle(action parityAction) {
	t, g := r.t, r.g
	if !pump(t, g, ch01FrameBudget*6, func() bool { return g.result != "" }) || g.result != "lose" || g.nativeDefeat == nil {
		t.Fatalf("正常章重播未抵達原生敗北：result=%s err=%s", g.result, g.loadErr)
	}
	if g.confirmBattleResult() {
		t.Fatal("確認鍵跳過敗北提示")
	}
	now := time.Unix(1, 0)
	screen := ebiten.NewImage(640, 400)
	seen := map[int]bool{}
	frames := []map[string]any{}
	protected := g.st.Units[14]
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
		"result": "lose", "protected_record": 14, "protected_hp": protected.HP, "protected_raw_byte5": protected.NativeRecordByte5,
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
