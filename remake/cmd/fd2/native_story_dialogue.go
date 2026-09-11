package main

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
)

// startNativeDialogueFrames 接回 0x165D6 的開框前聚焦；背景須在聚焦後保存。
func (g *Game) startNativeDialogueFrames() error {
	if len(g.dialog) != 1 || g.dialog[0].NativeDialogue == nil {
		return g.prepareNativeDialogueFrames()
	}
	n := g.dialog[0].NativeDialogue
	if !n.HasMotionTargetY {
		return errors.New("native story dialogue: focus provenance is unavailable")
	}
	if n.MotionTargetY == 0 {
		return g.prepareNativeDialogueFrames()
	}
	if g.st == nil && g.storyNativeMapState == nil && g.storyNativeMapSource != nil {
		if err := g.materializeNativeStoryMapState(g.storyNativeMapSource); err != nil {
			return err
		}
	}
	units := g.st
	var candidates []*battle.Unit
	if units != nil {
		candidates = units.Units
	} else {
		for i := range g.storyActors {
			candidates = append(candidates, &g.storyActors[i])
		}
	}
	var speaker *battle.Unit
	switch n.Control {
	case "FFED", "FFEC":
		if n.Operand >= 0 && n.Operand < len(candidates) {
			speaker = candidates[n.Operand]
		}
	case "FFEF", "FFEE":
		for _, u := range candidates {
			if u == nil {
				return errors.New("native story dialogue: nil speaker record")
			}
			identity, known := u.NativeIdentity, u.HasNativeIdentity
			if u.HasNativeRecordByte8 {
				identity, known = int(u.NativeRecordByte8), true
			}
			// 正在分派死亡效果的單位在原版還沒被標死（0x1B6B7 只收 +5 bit0 未設的
			// 記錄），重製端則在扣到 0 時就設了 bit0，所以這一筆照樣算在場。
			alive := u.HasNativeRecordByte5 && (u.NativeRecordByte5&1 == 0 || u == g.deathProgramDead)
			if known && identity == n.Operand && alive {
				speaker = u
				break
			}
		}
	}
	if speaker == nil || !speaker.HasNativeMapPresentation || g.focusJob != nil {
		return errors.New("native story dialogue: focus speaker or owner is unavailable")
	}
	if g.hasStoryNativeMapView {
		g.curX, g.curY = g.storyNativeMapView.CursorX, g.storyNativeMapView.CursorY
	} else if g.st != nil && g.st.HasNativeMapViewState {
		g.curX, g.curY = g.st.NativeMapViewState.CursorX, g.st.NativeMapViewState.CursorY
	} else {
		return errors.New("native story dialogue: focus view is unavailable")
	}
	targetX, targetY := int(speaker.NativeMapPresentation.X), int(speaker.NativeMapPresentation.Y)
	if _, _, _, err := g.nativeFocusEndpoint(targetX, targetY); err != nil {
		return err
	}
	pending := g.dialog
	g.dialog = nil
	g.focusJob = &focusUnitJob{targetX: targetX, targetY: targetY, nativeView: true, then: func() {
		g.dialog = pending
		g.nativeMapVGA = nil
		g.dlgShown, g.dlgPhase, g.dlgT = dlgNone, 0, 0
		if err := g.prepareNativeDialogueFrames(); err != nil {
			g.dialog = nil
			g.loadErr = "native story dialogue: " + err.Error()
		}
	}}
	return nil
}

func (g *Game) prepareNativeDialogueFrames() error {
	g.nativeDialogueFrames = nil
	g.nativeDialogueProgressive = nil
	g.nativeDialogueMouthOpen = nil
	g.nativeDialogueProgress = -1
	g.nativeDialogueOpening = nil
	g.nativeDialogueClosing = nil
	g.nativeDialogueClosingT = 0
	g.nativeDialogueClosingLive = false
	g.resetNativeStoryDialogueMouth()
	g.nativeDialogueModernPortrait = nil
	g.nativeDialogueSpeakingFrame = 0
	g.nativeDialoguePortraits = nil
	g.nativeDialogueLayout = nil
	g.nativeDialogueGlyphSteps = nil
	if len(g.dialog) == 0 || g.dialog[len(g.dialog)-1].NativeDialogue == nil {
		return nil
	}
	if len(g.dialog) != 1 {
		return errors.New("native story dialogue: one typed utterance per beat is required")
	}
	if g.nativeClassUI == nil {
		assets, err := loadNativeClassUIAssets()
		if err != nil {
			return err
		}
		g.nativeClassUI = assets
	}
	if g.localeID == "zh-Hant" && (g.nativeBattleFont == nil || g.nativeBattleGlyphs == nil) {
		font, glyphs, err := loadNativeBattleNameAssets()
		if err != nil {
			return err
		}
		g.nativeBattleFont, g.nativeBattleGlyphs = font, glyphs
	}
	if g.st != nil && !g.st.HasNativeMapViewState &&
		g.camp != nil && g.camp.NodeID() == "postbattle_ch28_persist" &&
		g.dialog[0].NativeDialogue.SourceDAT == "FDTXT_028" &&
		g.dialog[0].NativeDialogue.StringIndex == 7 {
		if err := g.materializeInheritedNativeDialogueView(); err != nil {
			return err
		}
	}
	if g.st == nil && g.storyNativeMapState == nil && g.storyNativeMapSource != nil {
		if err := g.materializeNativeStoryMapState(g.storyNativeMapSource); err != nil {
			return err
		}
	}
	if g.st == nil && g.storyNativeMapState != nil {
		if err := g.composeNativeStoryMapFrame(); err != nil {
			return err
		}
	} else if len(g.nativeMapVGA) != 320*200 {
		if err := g.composeNativeMapFrame(); err != nil {
			return err
		}
	}
	if len(g.nativeMapVGA) != 320*200 {
		return errors.New("native story dialogue: indexed map/font/frame assets are unavailable")
	}
	dl := g.dialog[0]
	if g.modernStoryPortraits != nil {
		portrait, ok := g.modernStoryPortraits.portraits[dl.Speaker]
		if !ok {
			return fmt.Errorf("native story dialogue: modern portrait for speaker %d is unavailable", dl.Speaker)
		}
		g.nativeDialogueModernPortrait = portrait
	}
	portraits, err := loadNativeSeparatedPortrait(dl.Speaker)
	if err != nil || len(portraits) < 4 {
		return fmt.Errorf("native story dialogue: speaker portrait %d is unavailable (frames=%d): %v", dl.Speaker, len(portraits), err)
	}
	native := dl.NativeDialogue
	layout := &campaign.NativeDialogueLayout{
		SourceDAT: native.SourceDAT, StringIndex: native.StringIndex,
		Utterance: native.Utterance, Control: native.Control, Operand: native.Operand,
		Pages: make([][]string, len(native.Pages)),
	}
	for i := range native.Pages {
		layout.Pages[i] = append([]string(nil), native.Pages[i]...)
	}
	if len(native.GlyphPages) != 0 {
		layout.GlyphPages = make([][][]string, len(native.GlyphPages))
		for page := range native.GlyphPages {
			layout.GlyphPages[page] = make([][]string, len(native.GlyphPages[page]))
			for row := range native.GlyphPages[page] {
				layout.GlyphPages[page][row] = append([]string(nil), native.GlyphPages[page][row]...)
			}
		}
	}
	wantUpper := layout.Control == "FFEF" || layout.Control == "FFED"
	if dl.Upper == nil || *dl.Upper != wantUpper {
		return errors.New("native story dialogue: control and upper/lower binding disagree")
	}
	opening, err := campaign.ComposeNativeStoryDialogueOpeningFrames(
		g.nativeMapVGA, g.nativeClassUI.dialogue, layout,
	)
	if err != nil {
		return fmt.Errorf("native story dialogue: opening frames: %w", err)
	}
	if !native.HasMotionTargetY {
		return errors.New("native story dialogue: closing motion target provenance is unavailable")
	}
	motionTargetY := native.MotionTargetY
	// raw ch27_post 沒有留下五句各自的說話者螢幕座標；沿用玩家游標會把
	// 下框滑到無關甚至越界的位置。99% 玩家可見模式保留五階段收框與背景
	// 還原，只省略這段無來源的額外滑動。
	if g.camp != nil && g.camp.NodeID() == "postbattle_ch28_persist" &&
		native.SourceDAT == "FDTXT_028" && native.StringIndex == 7 {
		motionTargetY = 0
	}
	visibleCursorX, visibleCursorY := 0, 0
	if motionTargetY != 0 {
		switch {
		case g.st != nil && g.st.HasNativeMapViewState:
			visibleCursorX = g.st.NativeMapViewState.VisibleCursorX
			visibleCursorY = g.st.NativeMapViewState.VisibleCursorY
		case g.hasStoryNativeMapView:
			visibleCursorX = g.storyNativeMapView.VisibleCursorX
			visibleCursorY = g.storyNativeMapView.VisibleCursorY
		default:
			return errors.New("native story dialogue: closing cursor/view provenance is unavailable")
		}
	}
	closing, err := campaign.ComposeNativeStoryDialogueClosingFrames(
		g.nativeMapVGA, g.nativeClassUI.dialogue, layout, motionTargetY,
		visibleCursorX, visibleCursorY,
	)
	if err != nil {
		return fmt.Errorf("native story dialogue: closing frames: %w", err)
	}
	frames := make([][]byte, len(layout.Pages))
	progressive := make([][][]byte, len(layout.Pages))
	mouthOpen := make([][]byte, len(layout.Pages))
	glyphSteps := make([][]bool, len(layout.Pages))
	for page := range layout.Pages {
		if g.localeID == "zh-Hant" {
			progressive[page], err = campaign.ComposeNativeStoryDialogueProgressiveFrames(
				g.nativeMapVGA, g.nativeClassUI.dialogue, portraits[0],
				g.nativeClassUI.font, g.nativeBattleGlyphs, layout, page,
			)
		} else {
			progressive[page], err = composeLocalizedNativeDialogueProgressiveFrames(
				g.nativeMapVGA, g.nativeClassUI.dialogue, portraits[0], g.font, layout, page,
			)
		}
		if err != nil {
			return fmt.Errorf("native story dialogue: page %d progressive frames: %w", page, err)
		}
		if g.localeID == "zh-Hant" {
			glyphSteps[page], err = campaign.NativeStoryDialogueGlyphSteps(layout, page)
			if err != nil || len(glyphSteps[page]) != len(progressive[page]) {
				return fmt.Errorf("native story dialogue: glyph timeline mismatch: %v", err)
			}
		}
		frames[page] = progressive[page][len(progressive[page])-1]
		if g.localeID == "zh-Hant" {
			mouthOpen[page], err = campaign.ComposeNativeStoryDialogueMouthFrame(
				frames[page], portraits[3], layout,
			)
			if err != nil {
				return fmt.Errorf("native story dialogue: page %d mouth frame: %w", page, err)
			}
		} else {
			// 使用者裁決：非繁中不要求嘴型；固定閉嘴 stable page。
			mouthOpen[page] = append([]byte(nil), frames[page]...)
		}
	}
	g.nativeDialogueFrames = frames
	g.nativeDialogueProgressive = progressive
	g.nativeDialogueMouthOpen = mouthOpen
	g.nativeDialogueOpening = opening
	g.nativeDialogueClosing = closing
	g.nativeDialoguePortraits = portraits
	g.nativeDialogueLayout = layout
	g.nativeDialogueGlyphSteps = glyphSteps
	return nil
}

func loadNativeSeparatedPortrait(speaker int) ([]dato.Frame, error) {
	return dato.LoadSeparatedResource(separatedAssetPath("portraits"), speaker)
}

// materializeInheritedNativeDialogueView 保留戰鬥控制器的現行視圖。
// raw ch27_post 在呼叫對話前沒有另寫鏡頭／游標，因此只有完整圖塊對齊的
// controller 狀態可以成為 indexed source；這不是章節專用的座標後備值。
func (g *Game) materializeInheritedNativeDialogueView() error {
	if g == nil || g.st == nil || g.m == nil || g.m.TileW <= 0 || g.m.TileH <= 0 {
		return errors.New("native story dialogue: inherited battle view is unavailable")
	}
	if g.camX < 0 || g.camY < 0 {
		return errors.New("native story dialogue: inherited battle camera is negative")
	}
	cameraX, cameraY := int(g.camX)/g.m.TileW, int(g.camY)/g.m.TileH
	if float64(cameraX*g.m.TileW) != g.camX || float64(cameraY*g.m.TileH) != g.camY {
		return errors.New("native story dialogue: inherited battle camera is not tile-aligned")
	}
	view := battle.NativeMapViewState{
		CameraX: cameraX, CameraY: cameraY,
		CursorX: g.curX, CursorY: g.curY,
		VisibleCursorX: g.curX - cameraX,
		VisibleCursorY: g.curY - cameraY,
	}
	if err := g.st.MaterializeNativeMapViewState(view); err != nil {
		return fmt.Errorf("native story dialogue: inherited battle view: %w", err)
	}
	return nil
}

func nativeStoryOpeningFrameIndex(tick, count int) int {
	if count <= 0 {
		return -1
	}
	index := tick - 1
	if index < 0 {
		index = 0
	}
	if index >= count {
		index = count - 1
	}
	return index
}

func (g *Game) stepNativeStoryDialogueProgress() {
	if g != nil && g.nativeDialogueClosingLive {
		if g.nativeDialogueClosingT < len(g.nativeDialogueClosing)-1 {
			g.nativeDialogueClosingT++
			return
		}
		g.finishNativeStoryDialogueClosing()
		return
	}
	if g == nil || g.dlgPhase != 0 || len(g.dialog) == 0 ||
		g.dialog[len(g.dialog)-1].NativeDialogue == nil || g.dlgScrollT > 0 ||
		g.dlgPage < 0 || g.dlgPage >= len(g.nativeDialogueProgressive) {
		return
	}
	frames := g.nativeDialogueProgressive[g.dlgPage]
	if len(frames) > 0 && g.nativeDialogueProgress < len(frames)-1 {
		g.nativeDialogueProgress++
		if g.dlgPage < len(g.nativeDialogueGlyphSteps) &&
			g.nativeDialogueProgress < len(g.nativeDialogueGlyphSteps[g.dlgPage]) &&
			g.nativeDialogueGlyphSteps[g.dlgPage][g.nativeDialogueProgress] {
			g.nativeDialogueSpeakingHalf++
			if g.nativeDialogueSpeakingHalf == 2 {
				g.nativeDialogueSpeakingHalf = 0
				g.nativeDialogueSpeakingCycle = (g.nativeDialogueSpeakingCycle + 1) % 4
				g.nativeDialogueSpeakingFrame = g.nativeDialogueSpeakingCycle
				if g.nativeDialogueSpeakingFrame == 3 {
					g.nativeDialogueSpeakingFrame = 1
				}
			}
		}
	}
}

func (g *Game) beginNativeStoryDialogueClosing() bool {
	if g == nil || g.nativeDialogueClosingLive || len(g.dialog) == 0 ||
		g.dialog[len(g.dialog)-1].NativeDialogue == nil || len(g.nativeDialogueClosing) == 0 {
		return false
	}
	g.nativeDialogueClosingLive = true
	g.resetNativeStoryDialogueMouth()
	// campInput 位於本幀一般 step 之後；立即發布 frame0，避免先出現空白幀。
	g.nativeDialogueClosingT = 0
	return true
}

func (g *Game) resetNativeStoryDialogueMouth() {
	if g == nil {
		return
	}
	g.nativeDialogueMouthReady = false
	g.nativeDialogueArrowTicks = 0
	g.mouthState = dato.MouthState{}
	g.mouthOpen, g.mouthTimer = false, 0
}

func (g *Game) nativeStoryDialogueAtInputWait() bool {
	if g == nil || g.nativeDialogueClosingLive || g.dlgPhase != 0 || g.dlgScrollT != 0 ||
		len(g.dialog) == 0 || g.dialog[len(g.dialog)-1].NativeDialogue == nil ||
		g.dlgPage < 0 || g.dlgPage >= len(g.nativeDialogueProgressive) ||
		g.dlgPage >= len(g.nativeDialogueMouthOpen) || len(g.nativeDialogueMouthOpen[g.dlgPage]) != 320*200 {
		return false
	}
	frames := g.nativeDialogueProgressive[g.dlgPage]
	return len(frames) > 0 && g.nativeDialogueProgress >= len(frames)-1
}

func (g *Game) stepDialogueMouth() {
	if g == nil || len(g.dialog) == 0 {
		g.resetNativeStoryDialogueMouth()
		return
	}
	if g.dialog[len(g.dialog)-1].NativeDialogue == nil {
		if g.frame%2 != 0 {
			return
		}
		randomMod30 := 0
		if g.mouthState.Open {
			randomMod30 = rand.Intn(30)
		}
		if next, err := g.mouthState.Tick(randomMod30); err == nil {
			g.mouthState, g.mouthOpen, g.mouthTimer = next, next.FrameIndex() == 3, next.Countdown
		}
		return
	}
	if !g.nativeStoryDialogueAtInputWait() {
		g.resetNativeStoryDialogueMouth()
		return
	}
	if !g.nativeDialogueMouthReady {
		g.mouthState = dato.MouthState{Countdown: rand.Intn(30) + 2}
		g.nativeDialogueMouthReady = true
		g.mouthOpen, g.mouthTimer = false, g.mouthState.Countdown
		return
	}
	g.nativeDialogueArrowTicks++
	if g.frame%2 != 0 {
		return
	}
	randomMod30 := 0
	if g.mouthState.Open {
		randomMod30 = rand.Intn(30)
	}
	if next, err := g.mouthState.Tick(randomMod30); err == nil {
		g.mouthState, g.mouthOpen, g.mouthTimer = next, next.FrameIndex() == 3, next.Countdown
	}
}

func (g *Game) finishNativeStoryDialogueClosing() {
	if g == nil || !g.nativeDialogueClosingLive {
		return
	}
	g.nativeDialogueClosingLive = false
	g.resetNativeStoryDialogueMouth()
	g.nativeDialogueClosingT = 0
	if len(g.dialog) > 0 {
		g.dialog = g.dialog[:len(g.dialog)-1]
	}
	g.dlgPage, g.dlgScrollT, g.dlgScrollFrom, g.nativeDialogueProgress = 0, 0, 0, -1
	g.dlgShown, g.dlgPhase, g.dlgT = dlgNone, 0, 0
	if g.battleEvent != nil {
		g.advanceBattleEvent()
		return
	}
	if g.camp != nil && g.camp.Node() != nil && g.camp.Node().Type == "cutscene" {
		g.beatAdvance()
	}
}

func (g *Game) drawNativeStoryDialogue(screen *ebiten.Image) bool {
	if g == nil || len(g.dialog) == 0 || g.dialog[len(g.dialog)-1].NativeDialogue == nil {
		return false
	}
	if g.nativeDialogueClosingLive {
		index := g.nativeDialogueClosingT
		if index >= len(g.nativeDialogueClosing) {
			index = len(g.nativeDialogueClosing) - 1
		}
		if index >= 0 {
			g.presentNativeClassFrame(screen, g.nativeDialogueClosing[index])
		}
		return true
	}
	// caller-specific 開框、穩定頁與收框都只呈現完整預建的 indexed frame；
	// 任一階段缺失時不回退到正規化對話介面。
	if g.dlgPhase == 2 {
		index := nativeStoryOpeningFrameIndex(g.dlgT, len(g.nativeDialogueOpening))
		if index >= 0 {
			g.presentNativeClassFrame(screen, g.nativeDialogueOpening[index])
		}
		return true
	}
	if g.dlgPhase != 0 {
		return true
	}
	if g.dlgPage < 0 || g.dlgPage >= len(g.nativeDialogueProgressive) {
		return true
	}
	frames := g.nativeDialogueProgressive[g.dlgPage]
	if len(frames) == 0 {
		return true
	}
	progress := g.nativeDialogueProgress
	if progress < 0 {
		progress = 0
	}
	if progress >= len(frames) {
		progress = len(frames) - 1
	}
	frame := frames[progress]
	if g.localeID == "zh-Hant" && progress > 0 && progress < len(frames)-1 &&
		g.nativeDialogueSpeakingFrame > 0 && len(g.nativeDialoguePortraits) == 4 {
		var err error
		frame, err = campaign.ComposeNativeStoryDialogueMouthFrame(frame,
			g.nativeDialoguePortraits[g.nativeDialogueSpeakingFrame],
			g.nativeDialogueLayout)
		if err != nil {
			g.loadErr = "native story dialogue: " + err.Error()
			return true
		}
	}
	if g.mouthOpen && progress == len(frames)-1 && g.dlgPage < len(g.nativeDialogueMouthOpen) {
		frame = g.nativeDialogueMouthOpen[g.dlgPage]
	}
	if g.nativeStoryDialogueAtInputWait() && g.dlgPage+1 < len(g.nativeDialogueProgressive) {
		var err error
		frame, err = campaign.ComposeNativeStoryDialogueWaitArrow(frame, g.nativeClassUI.dialogue,
			g.nativeDialogueLayout, (g.nativeDialogueArrowTicks/6)%2)
		if err != nil {
			g.loadErr = "native story dialogue: " + err.Error()
			return true
		}
	}
	if g.nativeDialogueModernPortrait != nil {
		rect, err := campaign.NativeStoryDialoguePortraitRect(
			g.dialog[len(g.dialog)-1].NativeDialogue.Control,
		)
		if err != nil {
			return true
		}
		if err := g.presentNativeClassFrameWithOverlay(
			screen, frame, g.nativeDialogueModernPortrait.image, rect,
		); err != nil {
			return true
		}
	} else {
		g.presentNativeClassFrame(screen, frame)
	}
	return true
}
