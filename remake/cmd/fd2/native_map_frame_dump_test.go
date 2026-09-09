package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// TestDumpChapterOneMoveFrames 是重製端的逐幀擷取，對應 dosgolem oracle 的
// `-frame-dir`（見 docs/knowledge-base/96-parity-toolchain-20260909.md）。
//
// 它用生產端的 composeNativeMapFrame 落地 320×200 索引畫面，格式與原版收據
// 相同，供逐張比對。狀態層看不出「多畫了什麼」——走行期間兩側的 overlay
// selector 都是 1，畫面卻一邊有游標白框一邊沒有。
//
// 需要 FD2_FRAME_DUMP 指向可寫目錄才會跑，預設略過。
func TestDumpChapterOneMoveFrames(t *testing.T) {
	out := os.Getenv("FD2_FRAME_DUMP")
	if out == "" {
		t.Skip("需要 FD2_FRAME_DUMP 指向輸出目錄")
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
		pal := g.nativeMapAssets.Palette
		if len(g.nativeMapDAC) == 256*3 {
			if p, e := fdother.VGAPaletteFromDAC(g.nativeMapDAC); e == nil {
				pal = p
			}
		}
		pic := image.NewPaletted(image.Rect(0, 0, 320, 200), pal)
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
		seg, tick := -1, -1
		if g.walk != nil {
			seg, tick = g.walk.seg, g.walk.tick
		}
		rec := map[string]any{
			"index": index, "file": name, "tag": tag,
			"cursor":         []int{g.curX, g.curY},
			"walk_seg":       seg,
			"walk_tick":      tick,
			"indexed_sha256": fmt.Sprintf("%x", sha256.Sum256(g.nativeMapVGA)),
		}
		if g.sel != nil {
			rec["sel"] = []int{g.sel.X, g.sel.Y}
			rec["raw"] = g.sel.NativeMapPresentation
		}
		d, _ := json.Marshal(rec)
		fmt.Fprintf(log, "%s\n", d)
		index++
	}

	if !g.positionScreenshotCursor(8, 16) {
		t.Fatal("游標無法就位")
	}
	g.confirm()
	if g.sel == nil {
		t.Fatalf("沒有選到單位 err=%q", g.loadErr)
	}
	// g.reach 是 map，直接取第一個會讓每次跑的終點不同，收據就無法重現。
	// 先收集距離 2 的可達空格再排序，固定取同一個。
	candidates := []battle.Cell{}
	for c := range g.reach {
		dx, dy := c.X-g.sel.X, c.Y-g.sel.Y
		if dx < 0 {
			dx = -dx
		}
		if dy < 0 {
			dy = -dy
		}
		if dx+dy == 2 && g.st.UnitAt(c.X, c.Y) == nil {
			candidates = append(candidates, c)
		}
	}
	if len(candidates) == 0 {
		t.Fatal("找不到距離 2 的可達空格")
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Y != candidates[j].Y {
			return candidates[i].Y < candidates[j].Y
		}
		return candidates[i].X < candidates[j].X
	})
	dest := candidates[0]
	if !g.positionScreenshotCursor(dest.X, dest.Y) {
		t.Fatal("游標無法移到終點")
	}
	dump("confirm-1")
	g.confirm()
	if g.walk == nil {
		t.Fatalf("沒有開始移動 err=%q", g.loadErr)
	}
	dump("confirm+0")
	for frame := 0; frame < 40; frame++ {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
		walking := g.walk != nil
		dump(fmt.Sprintf("f%d", frame))
		if !walking {
			break
		}
	}
	t.Logf("dumped %d frames to %s (dest=%+v)", index, out, dest)
}
