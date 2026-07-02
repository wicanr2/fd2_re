// title.go — 開頭動畫 + 主選單(忠實 doc23 反組譯):
// ① 魔王立繪 320×735(FDOTHER #0x45-0x49 五幀直疊)由下往上垂直捲動(視窗 200 高,
//    src y=535→0,原版 0x1fa85;任意鍵跳過)+ 淡入(0x1f525 palette fade 對映 ColorScale)。
// ② 抹除轉場 → 標題畫面(FDOTHER #7 sub0,FLAME DRAGON logo,palette=FDOTHER #8)
//    + 三選單項 START/LOAD/CONTINUE(#7 sub1-6 未選/選中素材)。
// ③ 選單:↑↓ wrap+游標音、Enter 確認(選中項閃 4 次,0x1fe2c);無 ESC 分支(doc23 §3)。
package main

import (
	"bytes"
	"fmt"
	"image"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type titleAssets struct {
	scroll *ebiten.Image    // 320×735 立繪
	title  *ebiten.Image    // 320×200 標題畫面
	items  [3][2]*ebiten.Image // START/LOAD/CONTINUE ×(未選/選中)
}

func loadTitleAssets() *titleAssets {
	ld := func(p string) *ebiten.Image {
		raw, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		im, _, err := image.Decode(bytes.NewReader(raw))
		if err != nil {
			return nil
		}
		return ebiten.NewImageFromImage(im)
	}
	t := &titleAssets{scroll: ld("assets/title/scroll_big.png"), title: ld("assets/title/title.png")}
	for i := 0; i < 3; i++ {
		t.items[i][0] = ld(fmt.Sprintf("assets/title/menu_%d.png", i*2+1))
		t.items[i][1] = ld(fmt.Sprintf("assets/title/menu_%d.png", i*2+2))
	}
	if t.scroll == nil || t.title == nil {
		return nil // 素材缺(玩家未自備)→ 跳過開頭直接進遊戲
	}
	return t
}

// titleUpdate 處理開頭動畫/主選單輸入。回傳 true = 仍在 title 流程。
func (g *Game) titleUpdate() bool {
	switch g.titlePhase {
	case "scroll":
		if g.scrollY >= 534 { // 開場即配樂(使用者記憶:登登登登磅礡進場;曲號待 dosbox 對照)
			g.playBGM("FDMUS_004")
		}
		g.scrollY -= 1.5 // 捲動速度(原版逐列複製;待 dosbox 錄影校)
		anyKey := len(inpututil.AppendJustPressedKeys(nil)) > 0
		if g.scrollY <= 0 || anyKey {
			g.titlePhase = "menu"
			g.titleSel = 0
		}
		return true
	case "menu":
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
			g.titleSel = (g.titleSel + 2) % 3 // wrap(doc23)
			g.playSFX(sfxCursor)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			g.titleSel = (g.titleSel + 1) % 3
			g.playSFX(sfxCursor)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.playSFX(sfxConfirm)
			g.titleFlash = 24 // 選中項閃 4 次(6 tick/次,原版 0x1fe2c 閃爍後回傳)
		}
		if g.titleFlash > 0 {
			g.titleFlash--
			if g.titleFlash == 0 {
				switch g.titleSel {
				case 1, 2: // LOAD / CONTINUE:讀檔(CONTINUE 原版=接續戰役,存檔語意,先同 LOAD)
					g.loadGame()
				}
				g.titlePhase = "" // START 或讀檔後 → 進遊戲
			}
		}
		return true
	}
	return false
}

// drawTitle 畫開頭動畫/主選單。
func (g *Game) drawTitle(screen *ebiten.Image) {
	ta := g.titleAssets
	switch g.titlePhase {
	case "scroll":
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(0, -g.scrollY*2) // 視窗=大圖的 y=scrollY 起 200 列
		// 淡入:捲動前 60 tick 從黑亮起(對映原版 palette fade-in 0x1f525)
		if fade := (535 - g.scrollY) / 60; fade < 1 {
			op.ColorScale.Scale(float32(fade), float32(fade), float32(fade), 1)
		}
		screen.DrawImage(ta.scroll, op)
	case "menu":
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		screen.DrawImage(ta.title, op)
		// 選單項(置中;y 位置對照原版標題畫面下半)
		for i := 0; i < 3; i++ {
			st := 0
			if i == g.titleSel && (g.titleFlash == 0 || (g.titleFlash/3)%2 == 0) {
				st = 1 // 選中(閃爍時交替)
			}
			it := ta.items[i][st]
			if it == nil {
				continue
			}
			b := it.Bounds()
			iop := &ebiten.DrawImageOptions{}
			iop.GeoM.Scale(2, 2)
			iop.GeoM.Translate(float64((320-b.Dx())/2*2), float64((166+i*12)*2)) // 置中,y=166/178/190@320(避開卷軸副標;待 dosbox 校)
			screen.DrawImage(it, iop)
		}
	}
}
