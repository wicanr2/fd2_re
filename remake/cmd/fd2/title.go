// title.go — 開頭動畫與主選單。目前接入已驗證的發行商畫面、AFM／FDOTHER
// 資源、個別選單輸入與交錯排程；兩段 wipe、logo 揭示與精確 DAC 時序仍未達 E2。
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
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/wicanr2/fd2_re/remake/internal/afm"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type titleAssets struct {
	scroll         *ebiten.Image       // 320×735 立繪
	title          *ebiten.Image       // 320×200 標題畫面
	publisher      *ebiten.Image       // FDOTHER #74 + palette #76 漢堂發行商畫面
	redFadeFrames  [20]*ebiten.Image   // sub_286BD phase 40→0，FDOTHER #69..73/#101
	titleFadeFrame [20]*ebiten.Image   // sub_286BD phase 0→40，FDOTHER #7/#8
	items          [3][2]*ebiten.Image // START/LOAD/CONTINUE ×(未選/選中)
	cutStatic      [2]*ebiten.Image    // 靜態幕:0=守護者(FDOTHER#100)、1=浮空城(#75)
	aniPath        string              // 玩家自備 ANI.DAT 路徑(""=無,退回捲動 fallback)
}

const (
	titleMenuTopY = 164
	titleMenuGapY = 9
)

func titleMenuY(item int) int {
	return titleMenuTopY + item*titleMenuGapY
}

// applyTitleMenuEvent 是標題主選單輸入的單一具型別擁有者；Ebiten 鍵盤與
// 決定性玩家路徑測試都必須經過此接縫。
func (g *Game) applyTitleMenuEvent(event TitleMenuEvent) bool {
	if g.titlePhase != "menu" {
		return false
	}
	menu := TitleMenuState{Selection: g.titleSel, FlashTicks: g.titleFlash}
	if event == TitleMenuUp || event == TitleMenuDown {
		g.playSFX(sfxCursor)
	} else if event == TitleMenuConfirm {
		g.playSFX(sfxConfirm)
	}
	action := menu.Step(event)
	g.titleSel, g.titleFlash = menu.Selection, menu.FlashTicks
	switch action {
	case TitleMenuLoadSlots:
		g.titleSlotSel = 0
		g.titlePhase = "loadslots"
	case TitleMenuContinue:
		if err := g.loadNativeContinueFromCurrentSnapshot(
			os.Getenv("FD2_NATIVE_SAVE"),
		); err != nil {
			g.msg = err.Error()
		}
	case TitleMenuStart:
		g.titlePhase = ""
	}
	return true
}

// applyTitleSlotEvent 擁有四槽有界 selector 與原子 LOAD 確認；原版資料驗證失敗時
// 保持 selector 啟用，不發布部分狀態。
func (g *Game) applyTitleSlotEvent(event TitleSlotEvent) bool {
	if g.titlePhase != "loadslots" {
		return false
	}
	slots := TitleSlotState{Selection: g.titleSlotSel}
	if event == TitleSlotUp || event == TitleSlotDown {
		g.playSFX(sfxCursor)
	}
	selected, confirm, cancel := slots.Step(event)
	g.titleSlotSel = slots.Selection
	if cancel {
		g.titlePhase = "menu"
		return true
	}
	if confirm {
		g.playSFX(sfxConfirm)
		return g.confirmTitleLoadSlot(selected)
	}
	return true
}

// 開場過場腳本把 sub_1F894 已證實的 esi=535→0 捲動與 AFM／FDOTHER 插播
// 投影成60Hz typed steps。原始位址、caller 與直接指令見
// docs/data/ida/fd2_title_scroll_schedule_ida.txt；這裡只擁有 runtime 排程。
type cutStep struct {
	kind       string // "publisher"／"afm"／"static"／"scroll"／"hold"／palette steps
	res        int    // afm:ANI.DAT index；static:cutStatic index
	tick       int    // afm:每幀；其他:整個 step 的60Hz幀數
	skip       bool   // afm 是否可由任意鍵中斷
	scrollFrom int    // scroll 的原版 esi 起點
	scrollTo   int    // scroll 的原版 esi 終點
}

var cutScript = []cutStep{
	{kind: "publisher", tick: 119}, // 8幀淡入＋103幀31 BIOS tick近似＋8幀淡出
	{kind: "afm", res: 3, tick: 5, skip: true},
	{kind: "scroll", tick: 153, scrollFrom: 535, scrollTo: 450},
	{kind: "static", res: 0, tick: 45}, // FDOTHER #100，原版 esi=450 插播
	{kind: "scroll", tick: 216, scrollFrom: 450, scrollTo: 330},
	{kind: "afm", res: 4, tick: 5},
	{kind: "afm", res: 5, tick: 3},
	{kind: "scroll", tick: 216, scrollFrom: 330, scrollTo: 210},
	{kind: "afm", res: 6, tick: 5},
	{kind: "afm", res: 7, tick: 3},
	{kind: "scroll", tick: 180, scrollFrom: 210, scrollTo: 110},
	{kind: "afm", res: 8, tick: 5},
	{kind: "scroll", tick: 153, scrollFrom: 110, scrollTo: 25},
	{kind: "afm", res: 0, tick: 1},
	{kind: "scroll", tick: 27, scrollFrom: 25, scrollTo: 10},
	{kind: "static", res: 1, tick: 60}, // FDOTHER #75，原版 esi=10 插播
	{kind: "scroll", tick: 18, scrollFrom: 10, scrollTo: 0},
	{kind: "hold", tick: 60}, // 原版 esi=0 額外1000ms
	{kind: "redfade", tick: 20},
	{kind: "redhold", tick: 6},
	{kind: "afm", res: 1, tick: 1, skip: true},
	{kind: "titlefade", tick: 20},
}

func titleLogoTransitionIndex() int {
	for i, step := range cutScript {
		if step.kind == "redfade" {
			return i
		}
	}
	return len(cutScript)
}

// aniCandidates 找玩家自備 ANI.DAT(未夾帶版權素材,執行期解碼)。
var aniCandidates = []string{
	"assets/ANI.DAT",
	"../org_game/炎龍騎士團/FLAME2/ANI.DAT",
	"org_game/炎龍騎士團/FLAME2/ANI.DAT",
}

func loadTitleAssets() (*titleAssets, error) {
	t := &titleAssets{}
	packRoot := separatedAssetPath("")
	publisher, err := loadSeparatedTitlePublisher(packRoot)
	if err != nil {
		return nil, fmt.Errorf("標題發行商分離素材：%w", err)
	}
	t.publisher = ebiten.NewImageFromImage(publisher)
	if err := loadSeparatedTitleFDOTHER(packRoot, t); err != nil {
		return nil, err
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
	return t, nil
}

func loadSeparatedTitleFDOTHER(packRoot string, target *titleAssets) error {
	if target == nil {
		return fmt.Errorf("標題素材 target 不可為 nil")
	}
	surfaceRoot, paletteRoot := filepath.Join(packRoot, "surfaces"), filepath.Join(packRoot, "palette")
	scrollDAC, scrollPalette, err := fdother.LoadSeparatedFDOTHERPalette(paletteRoot, 101)
	if err != nil {
		return fmt.Errorf("標題捲動調色盤：%w", err)
	}
	scrollPixels := make([]byte, 320*735)
	for index, resource := range []int{69, 70, 71, 72, 73} {
		frame, err := fdother.LoadSeparatedSingleFrame(surfaceRoot, "FDOTHER.DAT", resource)
		if err != nil {
			return fmt.Errorf("標題捲動 FDOTHER #%d：%w", resource, err)
		}
		if frame.Width != 320 || frame.Height != 147 {
			return fmt.Errorf("標題捲動 FDOTHER #%d 幾何=%dx%d", resource, frame.Width, frame.Height)
		}
		frame.Y = index * 147
		if err := frame.Blit(scrollPixels, 320, -1); err != nil {
			return err
		}
	}
	scrollImage := image.NewPaletted(image.Rect(0, 0, 320, 735), scrollPalette)
	copy(scrollImage.Pix, scrollPixels)
	target.scroll = ebiten.NewImageFromImage(scrollImage)

	titleDAC, titlePalette, err := fdother.LoadSeparatedFDOTHERPalette(paletteRoot, 8)
	if err != nil {
		return fmt.Errorf("標題主畫面調色盤：%w", err)
	}
	titleFrame, err := fdother.LoadSeparatedNestedSingleFrame(surfaceRoot, 7, 0)
	if err != nil {
		return fmt.Errorf("標題主畫面：%w", err)
	}
	if titleFrame.Width != 320 || titleFrame.Height != 200 {
		return fmt.Errorf("標題主畫面幾何=%dx%d", titleFrame.Width, titleFrame.Height)
	}
	titlePixels := make([]byte, 320*200)
	if err := titleFrame.Blit(titlePixels, 320, -1); err != nil {
		return err
	}
	titleImage := image.NewPaletted(image.Rect(0, 0, 320, 200), titlePalette)
	copy(titleImage.Pix, titlePixels)
	target.title = ebiten.NewImageFromImage(titleImage)

	wantMenuGeometry := [6]image.Point{{61, 7}, {61, 7}, {62, 7}, {62, 7}, {62, 8}, {62, 8}}
	for nested := 1; nested <= 6; nested++ {
		frame, err := fdother.LoadSeparatedNestedSingleFrame(surfaceRoot, 7, nested)
		if err != nil {
			return fmt.Errorf("標題選單 FDOTHER #7/%d：%w", nested, err)
		}
		want := wantMenuGeometry[nested-1]
		if frame.Width != want.X || frame.Height != want.Y {
			return fmt.Errorf("標題選單 FDOTHER #7/%d 幾何=%dx%d", nested, frame.Width, frame.Height)
		}
		target.items[(nested-1)/2][(nested-1)%2] = ebiten.NewImageFromImage(maskedRGBA(frame, titlePalette))
	}

	for slot, pair := range []struct{ resource, palette int }{{100, 99}, {75, 76}} {
		frame, err := fdother.LoadSeparatedSingleFrame(surfaceRoot, "FDOTHER.DAT", pair.resource)
		if err != nil {
			return fmt.Errorf("標題靜態幕 FDOTHER #%d：%w", pair.resource, err)
		}
		if frame.Width != 320 || frame.Height != 200 {
			return fmt.Errorf("標題靜態幕 FDOTHER #%d 幾何=%dx%d", pair.resource, frame.Width, frame.Height)
		}
		_, palette, err := fdother.LoadSeparatedFDOTHERPalette(paletteRoot, pair.palette)
		if err != nil {
			return err
		}
		pixels := make([]byte, 320*200)
		if err := frame.Blit(pixels, 320, -1); err != nil {
			return err
		}
		indexed := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
		copy(indexed.Pix, pixels)
		target.cutStatic[slot] = ebiten.NewImageFromImage(indexed)
	}

	red, title, err := buildTitlePaletteTransitions(scrollPixels[:320*200], scrollDAC, titlePixels, titleDAC)
	if err != nil {
		return err
	}
	for index := range red {
		target.redFadeFrames[index] = ebiten.NewImageFromImage(red[index])
		target.titleFadeFrame[index] = ebiten.NewImageFromImage(title[index])
	}
	return nil
}

func maskedRGBA(frame fdother.Frame, palette color.Palette) *image.NRGBA {
	result := image.NewNRGBA(image.Rect(0, 0, frame.Width, frame.Height))
	for index, mask := range frame.Mask {
		if mask == 0 {
			continue
		}
		r, g, b, a := palette[frame.Indexed[index]].RGBA()
		result.Pix[index*4+0] = byte(r >> 8)
		result.Pix[index*4+1] = byte(g >> 8)
		result.Pix[index*4+2] = byte(b >> 8)
		result.Pix[index*4+3] = byte(a >> 8)
	}
	return result
}

// decodeNativeTitlePublisher 僅消費玩家自備 FDOTHER.DAT。#74 的四模式 RLE、
// #76 的六位元 DAC palette 與320×200幾何均不符合時即拒絕，不使用猜測 fallback。
func decodeNativeTitlePublisher(path string) (*image.Paletted, error) {
	if path == "" {
		return nil, fmt.Errorf("FDOTHER.DAT unavailable")
	}
	frameRaw, err := fdother.ReadResource(path, 74)
	if err != nil {
		return nil, err
	}
	paletteRaw, err := fdother.ReadResource(path, 76)
	if err != nil {
		return nil, err
	}
	frame, err := fdother.ParseSingleFrame(frameRaw)
	if err != nil {
		return nil, err
	}
	if frame.Width != 320 || frame.Height != 200 {
		return nil, fmt.Errorf("title publisher geometry=%dx%d, want 320x200", frame.Width, frame.Height)
	}
	pixels := make([]byte, 320*200)
	if err := frame.Blit(pixels, 320, -1); err != nil {
		return nil, err
	}
	palette, err := fdother.ParseVGAPalette(paletteRaw)
	if err != nil {
		return nil, err
	}
	indexed := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
	copy(indexed.Pix, pixels)
	return indexed, nil
}

// loadSeparatedTitlePublisher 是正式 runtime owner；只消費分離 surface與DAC，
// 不在失敗後回退 FDOTHER.DAT。
func loadSeparatedTitlePublisher(packRoot string) (*image.Paletted, error) {
	frame, err := fdother.LoadSeparatedSingleFrame(filepath.Join(packRoot, "surfaces"), "FDOTHER.DAT", 74)
	if err != nil {
		return nil, err
	}
	if frame.Width != 320 || frame.Height != 200 {
		return nil, fmt.Errorf("title publisher geometry=%dx%d, want 320x200", frame.Width, frame.Height)
	}
	pixels := make([]byte, 320*200)
	if err := frame.Blit(pixels, 320, -1); err != nil {
		return nil, err
	}
	_, palette, err := fdother.LoadSeparatedFDOTHERPalette(filepath.Join(packRoot, "palette"), 76)
	if err != nil {
		return nil, err
	}
	indexed := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
	copy(indexed.Pix, pixels)
	return indexed, nil
}

func nativeTitlePaletted(pixels, dac []byte) (*image.Paletted, error) {
	if len(pixels) != 320*200 {
		return nil, fmt.Errorf("title indexed framebuffer=%d, want 64000", len(pixels))
	}
	palette, err := fdother.ParseVGAPalette(dac)
	if err != nil {
		return nil, err
	}
	indexed := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
	copy(indexed.Pix, pixels)
	return indexed, nil
}

// decodeNativeTitlePaletteTransitions 以玩家自備 FDOTHER 重建 sub_286BD 的兩個
// caller-specific indexed source。20個60Hz幀保留原版0／40端點與signed /40公式。
func decodeNativeTitlePaletteTransitions(path string) ([20]*image.Paletted, [20]*image.Paletted, error) {
	var redFrames, titleFrames [20]*image.Paletted
	if path == "" {
		return redFrames, titleFrames, fmt.Errorf("FDOTHER.DAT unavailable")
	}
	scrollDAC, err := fdother.ReadResource(path, 101)
	if err != nil {
		return redFrames, titleFrames, err
	}
	scroll := make([]byte, 320*735)
	for i := 0; i < 5; i++ {
		raw, readErr := fdother.ReadResource(path, 69+i)
		if readErr != nil {
			return redFrames, titleFrames, readErr
		}
		frame, parseErr := fdother.ParseSingleFrame(raw)
		if parseErr != nil {
			return redFrames, titleFrames, parseErr
		}
		if frame.Width != 320 || frame.Height != 147 {
			return redFrames, titleFrames, fmt.Errorf("title scroll resource %d geometry=%dx%d", 69+i, frame.Width, frame.Height)
		}
		frame.Y = 147 * i
		if err := frame.Blit(scroll, 320, -1); err != nil {
			return redFrames, titleFrames, err
		}
	}

	titleArchive, err := fdother.ReadResource(path, 7)
	if err != nil {
		return redFrames, titleFrames, err
	}
	titleRaw, err := fdother.ArchiveEntry(titleArchive, 0)
	if err != nil {
		return redFrames, titleFrames, err
	}
	titleFrame, err := fdother.ParseSingleFrame(titleRaw)
	if err != nil {
		return redFrames, titleFrames, err
	}
	if titleFrame.Width != 320 || titleFrame.Height != 200 {
		return redFrames, titleFrames, fmt.Errorf("title base geometry=%dx%d", titleFrame.Width, titleFrame.Height)
	}
	titlePixels := make([]byte, 320*200)
	if err := titleFrame.Blit(titlePixels, 320, -1); err != nil {
		return redFrames, titleFrames, err
	}
	titleDAC, err := fdother.ReadResource(path, 8)
	if err != nil {
		return redFrames, titleFrames, err
	}

	return buildTitlePaletteTransitions(scroll[:320*200], scrollDAC, titlePixels, titleDAC)
}

func buildTitlePaletteTransitions(scrollPixels, scrollDAC, titlePixels, titleDAC []byte) ([20]*image.Paletted, [20]*image.Paletted, error) {
	var redFrames, titleFrames [20]*image.Paletted
	if len(scrollPixels) != 320*200 || len(titlePixels) != 320*200 {
		return redFrames, titleFrames, fmt.Errorf("title transition source geometry mismatch")
	}
	for tick := 0; tick < 20; tick++ {
		phase := titlePalettePhase(tick)
		redDAC, interpolateErr := fdother.InterpolateNativeCompoundDAC(
			scrollDAC, 0, 255, 40-phase, [3]byte{63, 0, 0},
		)
		if interpolateErr != nil {
			return redFrames, titleFrames, interpolateErr
		}
		var err error
		redFrames[tick], err = nativeTitlePaletted(scrollPixels, redDAC)
		if err != nil {
			return redFrames, titleFrames, err
		}
		fadeDAC, interpolateErr := fdother.InterpolateNativeCompoundDAC(
			titleDAC, 0, 255, phase, [3]byte{56, 60, 63},
		)
		if interpolateErr != nil {
			return redFrames, titleFrames, interpolateErr
		}
		titleFrames[tick], err = nativeTitlePaletted(titlePixels, fadeDAC)
		if err != nil {
			return redFrames, titleFrames, err
		}
	}
	return redFrames, titleFrames, nil
}

func titlePublisherBrightness(tick int) float32 {
	switch {
	case tick < 0 || tick >= 119:
		return 0
	case tick < 8:
		return float32(tick+1) / 8
	case tick < 111:
		return 1
	default:
		return float32(119-tick) / 8
	}
}

// titlePalettePhase 把原版41個8ms DAC相位投影到20個60Hz畫面，保留0與40端點。
func titlePalettePhase(tick int) int {
	if tick <= 0 {
		return 0
	}
	if tick >= 19 {
		return 40
	}
	return (tick*40 + 9) / 19
}

func titlePaletteBlend(op *ebiten.DrawImageOptions, factor float64, base color.RGBA) {
	op.ColorM.Scale(factor, factor, factor, 1)
	op.ColorM.Translate(
		(1-factor)*float64(base.R)/255,
		(1-factor)*float64(base.G)/255,
		(1-factor)*float64(base.B)/255,
		0,
	)
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
		return
	}
	if next := cutScript[g.cutIdx]; next.kind == "scroll" {
		g.scrollY = float64(next.scrollFrom)
	}
}

// trySkipTitleCutStep 保存兩個原版 input owner：AFM 第三參數只中斷明確
// skippable 的當前幕；sub_1F894 捲動列後的 pending key 則直接進標題揭示。
func (g *Game) trySkipTitleCutStep(step cutStep, anyKey bool) bool {
	if !anyKey {
		return false
	}
	if step.kind == "scroll" {
		// sub_1F894 0x1FC59..0x1FC66：每列後的 pending key 直接進
		// wipe／ANI #1，不是 AFM 第三參數的「只中斷當前幕」。
		g.cutIdx = titleLogoTransitionIndex()
		g.cutCur = nil
		g.cutFrame, g.cutTick = 0, 0
		return true
	}
	if !step.skip {
		return false
	}
	g.cutAdvance()
	return true
}

// titleUpdate 處理開頭動畫/主選單輸入。回傳 true = 仍在 title 流程。
func (g *Game) titleUpdate() bool {
	// Cross-platform projection of the original title caller's BIOS low word.
	// It is sampled throughout the normal title path, not synthesized from the save.
	g.nativeTitleClock.Sample(time.Now())
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
		// 原版第三參數只允許中斷當前 AFM 幕；不可跳過後續不可略過的幕。
		if g.trySkipTitleCutStep(
			step, len(inpututil.AppendJustPressedKeys(nil)) > 0,
		) {
			return true
		}
		if step.kind == "publisher" {
			if g.titleAssets.publisher == nil {
				g.cutAdvance()
				return true
			}
			g.cutTick++
			if g.cutTick >= step.tick {
				g.cutAdvance()
			}
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
		if step.kind == "hold" {
			g.cutTick++
			if g.cutTick >= step.tick {
				g.cutAdvance()
			}
			return true
		}
		if step.kind == "redfade" || step.kind == "redhold" || step.kind == "titlefade" {
			g.cutTick++
			if g.cutTick >= step.tick {
				g.cutAdvance()
			}
			return true
		}
		if step.kind == "scroll" { // 原版 esi 區間；AFM／靜態幕只暫停，不重設位置
			if g.titleAssets.scroll == nil {
				g.cutAdvance()
				return true
			}
			g.cutTick++
			progress := float64(g.cutTick) / float64(step.tick)
			g.scrollY = float64(step.scrollFrom) +
				float64(step.scrollTo-step.scrollFrom)*progress
			if g.cutTick >= step.tick {
				g.scrollY = float64(step.scrollTo)
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
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
			g.applyTitleMenuEvent(TitleMenuUp)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			g.applyTitleMenuEvent(TitleMenuDown)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.applyTitleMenuEvent(TitleMenuConfirm)
		}
		g.applyTitleMenuEvent(TitleMenuTick)
		return true
	case "loadslots":
		// Native 0x30550 selector: four bounded slots, no wrap, Enter/Space
		// confirms and Esc cancels back to the title menu.
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) && g.titleSlotSel > 0 {
			g.applyTitleSlotEvent(TitleSlotUp)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) && g.titleSlotSel < 3 {
			g.applyTitleSlotEvent(TitleSlotDown)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.applyTitleSlotEvent(TitleSlotCancel)
			return true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.applyTitleSlotEvent(TitleSlotConfirm)
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
		if g.cutIdx >= len(cutScript) {
			return
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		switch step := cutScript[g.cutIdx]; {
		case step.kind == "publisher":
			if ta.publisher != nil {
				s := titlePublisherBrightness(g.cutTick)
				op.ColorScale.Scale(s, s, s, 1)
				screen.DrawImage(ta.publisher, op)
			}
		case step.kind == "static":
			if img := ta.cutStatic[step.res]; img != nil {
				screen.DrawImage(img, op)
			}
		case step.kind == "scroll":
			if ta.scroll != nil {
				op.GeoM.Translate(0, -g.scrollY*2) // 視窗=大圖 y=scrollY 起 200 列
				screen.DrawImage(ta.scroll, op)
			}
		case step.kind == "hold":
			if ta.scroll != nil {
				op.GeoM.Translate(0, -g.scrollY*2)
				screen.DrawImage(ta.scroll, op)
			}
		case step.kind == "redfade":
			if frame := ta.redFadeFrames[min(g.cutTick, len(ta.redFadeFrames)-1)]; frame != nil {
				screen.DrawImage(frame, op)
			} else if ta.scroll != nil {
				factor := float64(40-titlePalettePhase(g.cutTick)) / 40
				titlePaletteBlend(op, factor, color.RGBA{R: 0xff, A: 0xff})
				screen.DrawImage(ta.scroll, op)
			}
		case step.kind == "redhold":
			screen.Fill(color.RGBA{R: 0xff, A: 0xff})
		case step.kind == "titlefade":
			if frame := ta.titleFadeFrame[min(g.cutTick, len(ta.titleFadeFrame)-1)]; frame != nil {
				screen.DrawImage(frame, op)
			} else if ta.title != nil {
				factor := float64(titlePalettePhase(g.cutTick)) / 40
				titlePaletteBlend(op, factor, color.RGBA{R: 0xe3, G: 0xf3, B: 0xff, A: 0xff})
				screen.DrawImage(ta.title, op)
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
			iop.GeoM.Translate(float64((320-b.Dx())/2*2), float64(titleMenuY(i)*2)) // 同狀態 DOSBox 最佳匹配:y=164/173/182、間距9@320
			screen.DrawImage(it, iop)
		}
		// F2 仍可切換重製端音源，但原版標題沒有常駐提示；忠實畫面不疊診斷文字。
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
