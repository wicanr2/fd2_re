package main

import (
	"errors"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
)

func (g *Game) prepareNativeDialogueFrames() error {
	g.nativeDialogueFrames = nil
	g.nativeDialogueProgressive = nil
	g.nativeDialogueProgress = -1
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
	if err != nil || len(portraits) == 0 {
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
	frames := make([][]byte, len(layout.Pages))
	progressive := make([][][]byte, len(layout.Pages))
	for page := range layout.Pages {
		progressive[page], err = campaign.ComposeNativeStoryDialogueProgressiveFrames(
			g.nativeMapVGA, g.nativeClassUI.dialogue, portraits[0],
			g.nativeClassUI.font, g.nativeBattleGlyphs, layout, page,
		)
		if err != nil {
			return err
		}
		frames[page] = progressive[page][len(progressive[page])-1]
	}
	g.nativeDialogueFrames = frames
	g.nativeDialogueProgressive = progressive
	return nil
}

func (g *Game) stepNativeStoryDialogueProgress() {
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

func (g *Game) drawNativeStoryDialogue(screen *ebiten.Image) bool {
	if g == nil || len(g.dialog) == 0 || g.dialog[len(g.dialog)-1].NativeDialogue == nil {
		return false
	}
	// 本切片只呈現已證實的穩定頁。原生開／關框插值尚未實作時，保持 indexed
	// 地圖可見，避免短暫閃出正規化對話介面。
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
	g.presentNativeClassFrame(screen, frames[progress])
	return true
}
