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
	"image/color"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/wicanr2/fd2_re/remake/internal/afm"
)

type titleAssets struct {
	scroll *ebiten.Image    // 320×735 立繪
	title  *ebiten.Image    // 320×200 標題畫面
	items  [3][2]*ebiten.Image // START/LOAD/CONTINUE ×(未選/選中)
	aniPath string          // 玩家自備 ANI.DAT 路徑(""=無,退回捲動 fallback)
}

// cutSeq — 開場過場的 AFM 資源播放順序(dosbox 實拍分鏡,doc23 §2.4 / doc39):
// 守護者→索爾→屠龍→二角→滿月→騎馬夜行→隊伍群像→金鎖→標題「2」logo。
// 精確排程(播放器 5 呼叫點對應哪段)仍是 RE 缺口,inter-scene 節奏用近似值。
var cutSeq = []int{3, 4, 5, 6, 2, 7, 8, 0, 1}

// cutTicksPerFrame — 過場每幀停留 tick(60fps;289 幀/~32s≈6)。近似,待播放器排程 RE 校正。
const cutTicksPerFrame = 6

// aniCandidates 找玩家自備 ANI.DAT(未夾帶版權素材,執行期解碼)。
var aniCandidates = []string{
	"assets/ANI.DAT",
	"../org_game/炎龍騎士團/FLAME2/ANI.DAT",
	"org_game/炎龍騎士團/FLAME2/ANI.DAT",
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
	if p := os.Getenv("FD2_ANI"); p != "" {
		aniCandidates = append([]string{p}, aniCandidates...)
	}
	for _, p := range aniCandidates {
		if _, err := os.Stat(p); err == nil {
			t.aniPath = p
			break
		}
	}
	return t
}

// loadCutClip 執行期解碼 cutSeq 的第 idx 個 AFM 資源為 ebiten 影格。失敗回 nil。
func (g *Game) loadCutClip(idx int) []*ebiten.Image {
	if idx < 0 || idx >= len(cutSeq) || g.titleAssets.aniPath == "" {
		return nil
	}
	clip, err := afm.DecodeResource(g.titleAssets.aniPath, cutSeq[idx])
	if err != nil {
		return nil
	}
	out := make([]*ebiten.Image, len(clip.Frames))
	for i, f := range clip.Frames {
		out[i] = ebiten.NewImageFromImage(f)
	}
	return out
}

// titleUpdate 處理開頭動畫/主選單輸入。回傳 true = 仍在 title 流程。
func (g *Game) titleUpdate() bool {
	switch g.titlePhase {
	case "cutscene":
		if g.cutCur == nil { // 進入或切到下一幕:載入該資源
			g.cutCur = g.loadCutClip(g.cutIdx)
			g.cutFrame, g.cutTick = 0, 0
			if g.cutCur == nil { // 該幕解碼失敗 → 跳過整段過場
				g.titlePhase = "menu"
				g.titleSel = 0
				return true
			}
		}
		if g.cutIdx == 0 && g.cutFrame == 0 && g.cutTick == 0 {
			g.playBGM("FDMUS_004") // 開場配樂(曲號待實聽驗證)
		}
		// 任意鍵跳過整段過場 → 直接進選單
		if len(inpututil.AppendJustPressedKeys(nil)) > 0 {
			g.titlePhase = "menu"
			g.titleSel = 0
			g.cutCur = nil
			return true
		}
		g.cutTick++
		if g.cutTick >= cutTicksPerFrame {
			g.cutTick = 0
			g.cutFrame++
			if g.cutFrame >= len(g.cutCur) { // 該幕播完 → 下一幕
				g.cutIdx++
				g.cutCur = nil
				if g.cutIdx >= len(cutSeq) { // 全部播完 → 選單(logo「2」即最後一幕)
					g.titlePhase = "menu"
					g.titleSel = 0
				}
			}
		}
		return true
	case "scroll":
		if g.scrollY >= 534 { // 開場即配樂(使用者記憶:登登登登磅礡進場;曲號待 dosbox 對照)
			g.playBGM("FDMUS_004")
		}
		g.scrollY -= 1.5 // 捲動速度(原版逐列複製;待 dosbox 錄影校)
		anyKey := len(inpututil.AppendJustPressedKeys(nil)) > 0
		if g.scrollY <= 0 || anyKey {
			g.titlePhase = "logozoom" // dosbox 實拍(doc23 §2.4):紅閃→「2」縮入→白閃→選單
			g.titleTick = 0
		}
		return true
	case "logozoom":
		g.titleTick++
		if g.titleTick > 50 || len(inpututil.AppendJustPressedKeys(nil)) > 0 { // 紅閃12+縮放30+白閃8
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
	case "cutscene":
		if g.cutCur != nil && g.cutFrame < len(g.cutCur) {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(2, 2)
			screen.DrawImage(g.cutCur[g.cutFrame], op)
		}
	case "scroll":
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(0, -g.scrollY*2) // 視窗=大圖的 y=scrollY 起 200 列
		// 淡入:捲動前 60 tick 從黑亮起(對映原版 palette fade-in 0x1f525)
		if fade := (535 - g.scrollY) / 60; fade < 1 {
			op.ColorScale.Scale(float32(fade), float32(fade), float32(fade), 1)
		}
		screen.DrawImage(ta.scroll, op)
	case "logozoom":
		t := g.titleTick
		switch {
		case t <= 12: // 全螢幕紅閃(實拍:硬切純紅)
			screen.Fill(color.RGBA{0xc8, 0x10, 0x10, 0xff})
		case t <= 42: // 標題縮放進場(3.0→1.0,~0.5s;實拍「2」縮入之近似——單獨「2」圖層待 ANI RE)
			p := float64(t-12) / 30
			sc := 2 * (3 - 2*p) // 6→2
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(-160, -100) // 以畫面中心縮放
			op.GeoM.Scale(sc, sc)
			op.GeoM.Translate(320, 200)
			screen.DrawImage(ta.title, op)
		default: // 全螢幕白閃(bloom)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(2, 2)
			screen.DrawImage(ta.title, op)
			w := ebiten.NewImage(logicalW, logicalH)
			a := uint8(255 - (t-42)*30)
			w.Fill(color.RGBA{a, a, a, a})
			screen.DrawImage(w, nil)
		}
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
			iop.GeoM.Translate(float64((320-b.Dx())/2*2), float64((162+i*9)*2)) // dosbox 實拍座標:y=162/171/180、間距9@320
			screen.DrawImage(it, iop)
		}
	}
}
