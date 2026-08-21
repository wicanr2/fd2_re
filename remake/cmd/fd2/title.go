// title.go — 開頭動畫與主選單。目前接入已驗證的 AFM／FDOTHER 資源、
// 個別選單輸入與局部排程；完整原版幕序、交錯捲動與 DAC 時序仍未達 E2。
// ① 魔王立繪 320×735(FDOTHER #0x45-0x49 五幀直疊)由下往上垂直捲動(視窗 200 高,
//
//	src y=535→0,原版 0x1fa85;任意鍵跳過)。淡入目前仍用 ColorScale 作 E1
//	可玩近似；原版 0x1f525 是65次六位元 DAC 寫入，尚未接到 title indexed path。
//
// ② 抹除轉場 → 標題畫面(FDOTHER #7 sub0,FLAME DRAGON logo,palette=FDOTHER #8)
//   - 三選單項 START/LOAD/CONTINUE(#7 sub1-6 未選/選中素材)。
//
// ③ 選單:↑↓ wrap+游標音、Enter 確認(選中項閃 4 次,0x1fe2c);無 ESC 分支(doc23 §3)。
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/wicanr2/fd2_re/remake/internal/afm"
)

type titleAssets struct {
	scroll    *ebiten.Image       // 320×735 立繪
	title     *ebiten.Image       // 320×200 標題畫面
	items     [3][2]*ebiten.Image // START/LOAD/CONTINUE ×(未選/選中)
	cutStatic [2]*ebiten.Image    // 靜態幕:0=守護者(FDOTHER#100)、1=浮空城(#75)
	aniPath   string              // 玩家自備 ANI.DAT 路徑(""=無,退回捲動 fallback)
}

// 開場過場腳本是重製端 E1 排程：AFM 動畫幕由 ANI.DAT 解碼，FDOTHER 靜態幕
// 使用已辨識資源；doc39 §10 已推翻舊 cutSeq，指出原版捲動會穿插在各幕之間，
// 目前 cutScript 尚未重現該完整交錯順序。delay 與 skippable 僅保留已證實的
// 局部輸入，不得把這份表當成原版完整排程真值。
type cutStep struct {
	kind string // "afm"=ANI.DAT 動畫幕 / "static"=FDOTHER 靜態幕
	res  int    // afm:ANI.DAT 資源號 / static:cutStatic 索引(0 守護者/1 浮空城)
	tick int    // afm:每幀停留 tick / static:整幕 hold tick
	skip bool   // 是否可按鍵跳過
}

var cutScript = []cutStep{
	{"afm", 3, 5, true},       // 守護者(動畫)
	{"static", 0, 45, false},  // ①守護者靜態收尾(FDOTHER#100,esi=0x1c2)
	{"afm", 4, 5, false},      // 索爾
	{"afm", 5, 3, false},      // 屠龍
	{"afm", 6, 5, false},      // 二角
	{"afm", 7, 3, false},      // 騎馬夜行
	{"afm", 8, 5, false},      // 群像
	{"afm", 0, 1, false},      // 金鎖(96 幀)
	{"static", 1, 60, false},  // ⑥滿月浮空城(FDOTHER#75,esi=0x0a,含 +1000ms 停留)
	{"scroll", 0, 220, false}, // 魔王立繪垂直捲動(FDOTHER#0x45-49),捲到頂露出⑨惡魔臉特寫(doc23 §2.4⑦)
	{"afm", 1, 1, true},       // 標題「2」logo(硬切紅閃光後)
}

// aniCandidates 找玩家自備 ANI.DAT(未夾帶版權素材,執行期解碼)。
var aniCandidates = []string{
	"assets/ANI.DAT",
	"../org_game/炎龍騎士團/FLAME2/ANI.DAT",
	"org_game/炎龍騎士團/FLAME2/ANI.DAT",
}

func loadTitleAssets() *titleAssets {
	ld := func(p string) *ebiten.Image {
		raw, err := os.ReadFile(assetPath(p))
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
	t.cutStatic[0] = ld("assets/title/cut_guardian.png") // 缺→該靜態幕自動跳過
	t.cutStatic[1] = ld("assets/title/cut_castle.png")
	if t.scroll == nil || t.title == nil {
		// 素材缺(玩家未自備,或 cwd 不是 remake/ 導致相對路徑解不到)→ 跳過開頭直接進遊戲。
		// playfix #2:silent 失敗很難查,留診斷訊息(不影響行為,只在 stderr 印一行)。
		cwd, _ := os.Getwd()
		fmt.Fprintf(os.Stderr, "fd2: 開場素材缺失(scroll=%v title=%v,cwd=%s,exeDir=%s)→ 跳過開頭動畫\n",
			t.scroll != nil, t.title != nil, cwd, exeDir())
		return nil
	}
	if p := os.Getenv("FD2_ANI"); p != "" {
		aniCandidates = append([]string{p}, aniCandidates...)
	}
	for _, p := range aniCandidates {
		rp := p
		if strings.HasPrefix(p, "assets/") { // ANI.DAT 是玩家自備版權檔,走四層查找(含 exeDir)
			rp = assetPath(p)
		} else if _, err := os.Stat(rp); err != nil && exeDir() != "" {
			// 開發時的 "../org_game/..." 是 cwd 相對路徑;cwd 不是 remake/ 時額外試執行檔目錄。
			if _, err2 := os.Stat(filepath.Join(exeDir(), p)); err2 == nil {
				rp = filepath.Join(exeDir(), p)
			}
		}
		if _, err := os.Stat(rp); err == nil {
			t.aniPath = rp
			break
		}
	}
	return t
}

// loadCutClip 執行期解碼指定 ANI.DAT 資源號為 ebiten 影格。失敗回 nil。
func (g *Game) loadCutClip(res int) []*ebiten.Image {
	if g.titleAssets.aniPath == "" {
		return nil
	}
	clip, err := afm.DecodeResource(g.titleAssets.aniPath, res)
	if err != nil {
		return nil
	}
	out := make([]*ebiten.Image, len(clip.Frames))
	for i, f := range clip.Frames {
		out[i] = ebiten.NewImageFromImage(f)
	}
	return out
}

// cutAdvance 前進到下一步驟(重置該步狀態);全部播完 → 進選單。
func (g *Game) cutAdvance() {
	g.cutIdx++
	g.cutCur = nil
	g.cutFrame, g.cutTick = 0, 0
	if g.cutIdx >= len(cutScript) {
		g.titlePhase = "menu"
		g.titleSel = 0
	}
}

// titleUpdate 處理開頭動畫/主選單輸入。回傳 true = 仍在 title 流程。
func (g *Game) titleUpdate() bool {
	switch g.titlePhase {
	case "cutscene":
		if g.cutIdx >= len(cutScript) {
			g.titlePhase = "menu"
			g.titleSel = 0
			return true
		}
		step := cutScript[g.cutIdx]
		if g.cutIdx == 0 && g.cutFrame == 0 && g.cutTick == 0 {
			g.playBGM("FDMUS_018") // 開場/標題曲(RE 確認:boot 0x025db5 play_bgm(18,0),doc12 §15)
		}
		// 按鍵跳過:該步 skippable 才可按任意鍵跳(原版旗標);ESC 一律跳整段(remake 便利)。
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) ||
			(step.skip && len(inpututil.AppendJustPressedKeys(nil)) > 0) {
			g.titlePhase = "menu"
			g.titleSel = 0
			g.cutCur = nil
			return true
		}
		if step.kind == "static" { // FDOTHER 靜態幕:hold step.tick 個 tick
			if g.titleAssets.cutStatic[step.res] == nil { // 素材缺 → 跳過此幕
				g.cutAdvance()
				return true
			}
			g.cutTick++
			if g.cutTick >= step.tick {
				g.cutAdvance()
			}
			return true
		}
		if step.kind == "scroll" { // 魔王立繪垂直捲動,捲到頂(惡魔臉)→ 下一幕
			if g.titleAssets.scroll == nil {
				g.cutAdvance()
				return true
			}
			g.cutTick++
			g.scrollY = 535 * (1 - float64(g.cutTick)/float64(step.tick)) // 535→0(底→頂)
			if g.cutTick >= step.tick {
				g.scrollY = 0
				g.cutAdvance()
			}
			return true
		}
		// AFM 動畫幕
		if g.cutCur == nil { // 進入此幕:執行期解碼
			g.cutCur = g.loadCutClip(step.res)
			g.cutFrame, g.cutTick = 0, 0
			if g.cutCur == nil { // 解碼失敗 → 跳過此幕(不整段中止)
				g.cutAdvance()
				return true
			}
		}
		g.cutTick++
		if g.cutTick >= step.tick {
			g.cutTick = 0
			g.cutFrame++
			if g.cutFrame >= len(g.cutCur) { // 此幕播完 → 下一步
				g.cutAdvance()
			}
		}
		return true
	case "scroll":
		if g.scrollY >= 534 { // 開場即配樂(使用者記憶:登登登登磅礡進場;曲號待 dosbox 對照)
			g.playBGM("FDMUS_018") // 同開場曲(RE 確認,取代舊猜測 FDMUS_004)
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
		menu := TitleMenuState{Selection: g.titleSel, FlashTicks: g.titleFlash}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
			menu.Step(TitleMenuUp)
			g.playSFX(sfxCursor)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			menu.Step(TitleMenuDown)
			g.playSFX(sfxCursor)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			menu.Step(TitleMenuConfirm)
			g.playSFX(sfxConfirm)
		}
		if action := menu.Step(TitleMenuTick); action != TitleMenuNoAction {
			g.titleSel, g.titleFlash = menu.Selection, menu.FlashTicks
			switch action {
			case TitleMenuLoadSlots:
				g.titleSlotSel = 0
				g.titlePhase = "loadslots"
				return true
			case TitleMenuContinue:
				// Original selection 2 restores FD2.SAV's current-runtime
				// snapshot through 0x10010; it is not remake JSON slot 0.  The
				// caller supplies the original save and signed BIOS seed; every
				// missing/ambiguous owner remains fail-closed in the title.
				if err := g.loadNativeContinueFromCurrentSnapshot(
					os.Getenv("FD2_NATIVE_SAVE"),
				); err != nil {
					g.msg = err.Error()
				}
				return true
			}
			g.titlePhase = "" // START 或讀檔後 → 進遊戲
		}
		g.titleSel, g.titleFlash = menu.Selection, menu.FlashTicks
		return true
	case "loadslots":
		slots := TitleSlotState{Selection: g.titleSlotSel}
		// Native 0x30550 selector: four bounded slots, no wrap, Enter/Space
		// confirms and Esc cancels back to the title menu.
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) && g.titleSlotSel > 0 {
			slots.Step(TitleSlotUp)
			g.playSFX(sfxCursor)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) && g.titleSlotSel < 3 {
			slots.Step(TitleSlotDown)
			g.playSFX(sfxCursor)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			slots.Step(TitleSlotCancel)
			g.titlePhase = "menu"
			return true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			selected, confirm, _ := slots.Step(TitleSlotConfirm)
			if confirm {
				confirmable, native := nativeLoadSlotConfirmable(selected)
				if !confirmable {
					g.msg = "空的存檔槽"
					return true
				}
				if native {
					if err := g.loadNativeGameFromSlot(
						os.Getenv("FD2_NATIVE_SAVE"), selected,
					); err != nil {
						g.msg = err.Error()
						return true
					}
				} else {
					g.loadGameFromSlot(selected)
				}
			}
			g.titlePhase = ""
		}
		g.titleSlotSel = slots.Selection
		return true
	}
	return false
}

// drawTitle 畫開頭動畫/主選單。
func (g *Game) drawTitle(screen *ebiten.Image) {
	ta := g.titleAssets
	switch g.titlePhase {
	case "cutscene":
		if g.cutIdx >= len(cutScript) {
			return
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		switch step := cutScript[g.cutIdx]; {
		case step.kind == "static":
			if img := ta.cutStatic[step.res]; img != nil {
				screen.DrawImage(img, op)
			}
		case step.kind == "scroll":
			if ta.scroll != nil {
				op.GeoM.Translate(0, -g.scrollY*2) // 視窗=大圖 y=scrollY 起 200 列
				screen.DrawImage(ta.scroll, op)
			}
		default:
			if g.cutCur != nil && g.cutFrame < len(g.cutCur) {
				screen.DrawImage(g.cutCur[g.cutFrame], op)
			}
		}
	case "scroll":
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(0, -g.scrollY*2) // 視窗=大圖的 y=scrollY 起 200 列
		// 淡入：捲動前60 tick從黑亮起。這是 E1 RGBA 可玩近似，不等同
		// 原版 0x1f525 的 delta 64→0 共65次六位元 DAC 寫入。
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
		// 音源設定提示（F2 切換預錄 OGG；只近似原版 SETSOUND 的選擇意圖）
		if g.font != nil {
			g.font.Draw(screen, "♪ F2  "+bgmSourceName[g.bgmSource], 8, float64(logicalH)-24, 0.9,
				color.RGBA{0xa0, 0xc0, 0xff, 0xff})
		}
	case "loadslots":
		if g.drawNativeLoadSlots(screen) {
			return
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		screen.DrawImage(ta.title, op)
		if g.font == nil {
			return
		}
		box := ebiten.NewImage(360, 250)
		box.Fill(color.RGBA{0x08, 0x14, 0x30, 0xf0})
		bop := &ebiten.DrawImageOptions{}
		bop.GeoM.Translate(140, 70)
		screen.DrawImage(box, bop)
		g.font.Draw(screen, "讀取存檔（↑↓選擇，Enter 確認，Esc 返回）", 158, 86, 0.9, color.RGBA{0xff, 0xe0, 0x90, 0xff})
		for slot := 0; slot < 4; slot++ {
			c := color.RGBA{0xd0, 0xd8, 0xe8, 0xff}
			prefix := "　"
			if slot == g.titleSlotSel {
				prefix = "▶"
				c = color.RGBA{0xff, 0xff, 0xff, 0xff}
			}
			label := fmt.Sprintf("%s槽位 %d", prefix, slot+1)
			if _, err := os.Stat(saveSlotPath(slot)); err != nil {
				label += "　（空）"
			} else {
				label += "　（已有存檔）"
			}
			g.font.Draw(screen, label, 176, 124+float64(slot)*34, 1.0, c)
		}
	}
}
