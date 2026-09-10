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

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// 對拍座標。這三格是 dosgolem 那一側實際走過的同一組操作：游標移到 (8,16) 的
// 亞雷斯、移動到 (6,19)、攻擊 (5,19) 的盜賊。收據見
// docs/data/ui-traces/fd2-move-attack-original-20260910.json。兩邊用同一組座標，
// 逐幀畫面才有得比。
const (
	attackDumpActorX, attackDumpActorY   = 8, 16
	attackDumpDestX, attackDumpDestY     = 6, 19
	attackDumpTargetX, attackDumpTargetY = 5, 19
)

// TestDumpChapterOneMoveAttackFrames 是重製端的「移動＋攻擊」逐幀擷取，對應
// dosgolem oracle 的同一組操作。原版那側是送 BIOS 按鍵（up→enter→左左下下下→
// enter→enter→enter），重製端這側直接驅動 Game 的同一條狀態機——兩邊的輸入層
// 不同，但走過的節點與座標相同，逐幀畫面因此可以對照。
//
// 需要 FD2_ATTACK_DUMP 指向輸出目錄，跟 TestDumpChapterOneMoveFrames 一樣預設
// 跳過。
func TestDumpChapterOneMoveAttackFrames(t *testing.T) {
	out := os.Getenv("FD2_ATTACK_DUMP")
	if out == "" {
		t.Skip("需要 FD2_ATTACK_DUMP 指向輸出目錄")
	}
	t.Setenv("FD2_TITLE", "0")
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_CAMPAIGN", defaultPlayerCampaign)
	g := loadGame()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	if err := g.fastForwardShotCampaign(); err != nil {
		t.Fatal(err)
	}

	index := 0
	log, err := os.Create(filepath.Join(out, "frames.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	dump := func(tag string) {
		if err := g.composeNativeMapFrame(); err != nil {
			t.Fatalf("%s 組不出原生整幀：%v", tag, err)
		}
		palette := g.nativeMapAssets.Palette
		if len(g.nativeMapDAC) == 256*3 {
			if p, e := fdother.VGAPaletteFromDAC(g.nativeMapDAC); e == nil {
				palette = p
			}
		}
		pic := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
		copy(pic.Pix, g.nativeMapVGA)
		name := fmt.Sprintf("frame-%06d.png", index)
		f, e := os.Create(filepath.Join(out, name))
		if e != nil {
			t.Fatal(e)
		}
		if e = png.Encode(f, pic); e != nil {
			t.Fatal(e)
		}
		f.Close()
		record := map[string]any{
			"index": index, "file": name, "tag": tag,
			"cursor":         []int{g.curX, g.curY},
			"ring":           g.ring,
			"ring_sel":       g.ringSel,
			"moved":          g.moved,
			"walking":        g.walk != nil,
			"attack_anim":    g.atk != nil,
			"msg":            g.msg,
			"indexed_sha256": fmt.Sprintf("%x", sha256.Sum256(g.nativeMapVGA)),
		}
		if g.sel != nil {
			record["sel"] = []int{g.sel.X, g.sel.Y}
		}
		units := []map[string]any{}
		for _, u := range g.st.Units {
			if u.HP <= 0 {
				continue
			}
			units = append(units, map[string]any{
				"camp": int(u.Camp), "x": u.X, "y": u.Y, "hp": u.HP,
			})
		}
		record["units"] = units
		encoded, _ := json.Marshal(record)
		fmt.Fprintf(log, "%s\n", encoded)
		index++
	}

	// 1. 游標到我方單位並選中——原版是 up 之後一次 enter。
	if !g.positionScreenshotCursor(attackDumpActorX, attackDumpActorY) {
		t.Fatal("游標移不到出發格")
	}
	dump("cursor-on-actor")
	g.confirm()
	if g.sel == nil {
		t.Fatalf("沒有選到單位 err=%q", g.loadErr)
	}
	if g.sel.X != attackDumpActorX || g.sel.Y != attackDumpActorY {
		t.Fatalf("選到的是 (%d,%d)，不是出發格", g.sel.X, g.sel.Y)
	}
	dump("selected")

	// 2. 游標到終點並確認移動——原版是左左下下下之後一次 enter。
	destination := battle.Cell{X: attackDumpDestX, Y: attackDumpDestY}
	if _, ok := g.reach[destination]; !ok {
		t.Fatalf("(%d,%d) 不在可達範圍內，對拍座標要重挑",
			attackDumpDestX, attackDumpDestY)
	}
	if !g.positionScreenshotCursor(attackDumpDestX, attackDumpDestY) {
		t.Fatal("游標移不到終點")
	}
	dump("cursor-on-dest")
	g.confirm()
	if g.walk == nil {
		t.Fatalf("沒有開始移動 err=%q", g.loadErr)
	}
	for frame := 0; frame < 80 && g.walk != nil; frame++ {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
		dump(fmt.Sprintf("walk%d", frame))
	}
	if g.sel.X != attackDumpDestX || g.sel.Y != attackDumpDestY {
		t.Fatalf("走完停在 (%d,%d)，不是終點", g.sel.X, g.sel.Y)
	}

	// 3. 走完會自動開指令環（beginBattleActionOverlay）。原版四向是
	//    ↑0 攻擊／←1 法術／→2 物品／↓3 待機，由 0x18D8C 的 switch 釘死。
	for frame := 0; frame < 60 && !g.ring; frame++ {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if !g.ring {
		t.Fatalf("移動之後沒有開指令環 err=%q", g.loadErr)
	}
	dump("ring-open")
	if !nativeActionSelectable(g.actionOverlayAvailability(), 0) {
		t.Fatal("攻擊方向不可用，這一組對拍座標打不到人")
	}
	g.ringSel = 0

	// 4. 確認攻擊＝關環之後進選目標，與 ringInput 的 case 0 同一條路。
	//
	// 開合動畫的推進掛在「這一幀被畫過」（stepActionOverlayLifecycle 的
	// actionOverlayDrawn 閘門）——正式路徑由繪製那側標記，離屏 dump 不走 Draw，
	// 所以這裡在組完幀之後自己標，等同於「這一幀畫出去了」。
	g.beginActionOverlayClose(func() {})
	for frame := 0; frame < 60 && g.ring; frame++ {
		if err := g.composeNativeMapFrame(); err != nil {
			t.Fatal(err)
		}
		g.markActionOverlayDrawn()
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if g.ring {
		t.Fatal("指令環沒有關閉")
	}
	dump("target-mode")

	// 5. 游標到目標並確認——原版是游標自動跳到可攻擊的敵人之後一次 enter。
	target := g.st.UnitAt(attackDumpTargetX, attackDumpTargetY)
	if target == nil {
		t.Fatalf("(%d,%d) 沒有單位，對拍座標要重挑",
			attackDumpTargetX, attackDumpTargetY)
	}
	// 攻擊完成之後 g.sel 會被清掉（單位行動完畢），所以先留住指標。
	actor := g.sel
	beforeActor, beforeTarget := actor.HP, target.HP
	if !g.positionScreenshotCursor(attackDumpTargetX, attackDumpTargetY) {
		t.Fatal("游標移不到目標")
	}
	dump("cursor-on-target")
	g.confirm()
	for frame := 0; frame < 200; frame++ {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
		dump(fmt.Sprintf("attack%d", frame))
		if target.HP != beforeTarget || actor.HP != beforeActor {
			break
		}
	}
	// 打空（miss）時兩邊 HP 都不變，那仍然是一次完整的攻擊——判定看行動有沒有
	// 結算，不看 HP。
	if !actor.Acted {
		t.Fatalf("攻擊沒有結算：actor.Acted=false，msg=%q", g.msg)
	}
	t.Logf("dumped %d frames to %s；攻方 HP %d→%d，守方 HP %d→%d，msg=%q",
		index, out, beforeActor, actor.HP, beforeTarget, target.HP, g.msg)
}
