package main

import (
	"errors"
	"math/rand"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
)

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
	if g.nativeBattleFont == nil || g.nativeBattleGlyphs == nil {
		font, glyphs, err := loadNativeBattleNameAssets()
		if err != nil {
			return err
		}
		g.nativeBattleFont, g.nativeBattleGlyphs = font, glyphs
	}
	if len(g.nativeMapVGA) != 320*200 {
		if err := g.composeNativeMapFrame(); err != nil {
			return err
		}
	}
	if len(g.nativeMapVGA) != 320*200 {
		return errors.New("native story dialogue: indexed map/font/frame assets are unavailable")
	}
	datoPath := nativeDATOPath()
	if datoPath == "" {
		return errors.New("native story dialogue: DATO.DAT is unavailable")
	}
	dl := g.dialog[0]
	portraits, err := dato.DecodeResource(filepath.Clean(datoPath), dl.Speaker)
	if err != nil || len(portraits) < 4 {
		return errors.New("native story dialogue: speaker portrait is unavailable")
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
	wantUpper := layout.Control == "FFEF" || layout.Control == "FFED"
	if dl.Upper == nil || *dl.Upper != wantUpper {
		return errors.New("native story dialogue: control and upper/lower binding disagree")
	}
	opening, err := campaign.ComposeNativeStoryDialogueOpeningFrames(
		g.nativeMapVGA, g.nativeClassUI.dialogue, layout,
	)
	if err != nil {
		return err
	}
	if !native.HasMotionTargetY {
		return errors.New("native story dialogue: closing motion target provenance is unavailable")
	}
	if native.MotionTargetY != 0 && !g.hasStoryNativeMapView {
		return errors.New("native story dialogue: closing cursor/view provenance is unavailable")
	}
	closing, err := campaign.ComposeNativeStoryDialogueClosingFrames(
		g.nativeMapVGA, g.nativeClassUI.dialogue, layout, native.MotionTargetY,
		g.storyNativeMapView.VisibleCursorX, g.storyNativeMapView.VisibleCursorY,
	)
	if err != nil {
		return err
	}
	frames := make([][]byte, len(layout.Pages))
	progressive := make([][][]byte, len(layout.Pages))
	mouthOpen := make([][]byte, len(layout.Pages))
	for page := range layout.Pages {
		progressive[page], err = campaign.ComposeNativeStoryDialogueProgressiveFrames(
			g.nativeMapVGA, g.nativeClassUI.dialogue, portraits[0],
			g.nativeClassUI.font, g.nativeBattleGlyphs, layout, page,
		)
		if err != nil {
			return err
		}
		frames[page] = progressive[page][len(progressive[page])-1]
		mouthOpen[page], err = campaign.ComposeNativeStoryDialogueMouthFrame(
			frames[page], portraits[3], layout,
		)
		if err != nil {
			return err
		}
	}
	g.nativeDialogueFrames = frames
	g.nativeDialogueProgressive = progressive
	g.nativeDialogueMouthOpen = mouthOpen
	g.nativeDialogueOpening = opening
	g.nativeDialogueClosing = closing
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
	if g.mouthOpen && progress == len(frames)-1 && g.dlgPage < len(g.nativeDialogueMouthOpen) {
		frame = g.nativeDialogueMouthOpen[g.dlgPage]
	}
	g.presentNativeClassFrame(screen, frame)
	return true
}
