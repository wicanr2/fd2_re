package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func (g *Game) composeNativeChurchTransferItemFrame() ([]byte, bool) {
	a := g.nativeClassUI
	if a == nil || g.churchMode != "transfer_item" ||
		g.churchTransferSource < 0 || len(g.churchTransferItems) == 0 {
		return nil, false
	}
	sourceUnit, ok := g.partyRoster[g.churchTransferSource]
	if !ok {
		return nil, false
	}
	start, visible := campaign.NativeTwoColumnWindow(
		len(g.churchTransferItems), g.churchSel, g.churchItemStart,
	)
	if visible == 0 {
		return nil, false
	}
	g.churchItemStart = start
	itemIDs := make([]int, len(g.churchTransferItems))
	for i, slot := range g.churchTransferItems {
		if slot < 0 || slot >= len(sourceUnit.Inventory) {
			return nil, false
		}
		itemIDs[i] = sourceUnit.Inventory[slot]
	}
	background, ok := g.composeNativeChurchSourceFrame()
	if !ok {
		return nil, false
	}
	frame := append([]byte(nil), background...)
	if err := a.panel.BlitOpaqueAt(frame, 320, 5, 112, false); err != nil {
		return nil, false
	}
	assets, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		return nil, false
	}
	rows, err := battle.LoadNativeItemEffectRowPrefix(
		assetPath("assets/data/native_item_effect_rows.json"),
	)
	if err != nil {
		return nil, false
	}
	if err := battle.RenderNativeTransferItemRows(
		assets, a.priceCell, itemIDs, start, g.churchSel, rows, frame,
	); err != nil {
		return nil, false
	}
	return frame, true
}

func (g *Game) drawNativeChurchTransferItem(screen *ebiten.Image) bool {
	frame, ok := g.composeNativeChurchTransferItemFrame()
	if !ok {
		return false
	}
	g.presentNativeClassFrame(screen, frame)
	return true
}

func (g *Game) beginNativeChurchTransferItemOpening() bool {
	final, ok := g.composeNativeChurchTransferItemFrame()
	if !ok {
		return false
	}
	source, ok := g.composeNativeChurchSourceFrame()
	if !ok {
		return false
	}
	frames, err := campaign.NativeClassListOpeningFrames(source, final)
	if err != nil || len(frames) != 6 {
		return false
	}
	g.nativeClassUIJob = &nativeClassUIJob{frames: frames}
	return true
}

func (g *Game) beginNativeChurchTransferItemClosing(after func()) bool {
	final, ok := g.composeNativeChurchTransferItemFrame()
	if !ok {
		return false
	}
	source, ok := g.composeNativeChurchSourceFrame()
	if !ok {
		return false
	}
	frames, err := campaign.NativeClassListClosingFrames(source, final)
	if err != nil || len(frames) != 5 {
		return false
	}
	g.nativeClassUIJob = &nativeClassUIJob{
		frames: frames, restore: source, after: after,
	}
	return true
}

func (g *Game) returnToNativeTransferSource() {
	g.churchMode = "transfer_source"
	g.churchTransferDest = -1
	g.churchIDs = g.churchTransferSourceIDs()
	g.churchSel = 0
	g.churchRosterStart = 0
	g.churchItemStart = 0
	g.nativeChurchTextIndex = 512
	g.beginNativeChurchRosterOpening()
}

func (g *Game) composeNativeChurchTransferFullMessage() ([]byte, bool) {
	a := g.nativeClassUI
	if a == nil || g.churchMode != "transfer_full" ||
		g.churchTransferDest < 0 {
		return nil, false
	}
	destination, ok := g.partyRoster[g.churchTransferDest]
	if !ok || !destination.HasNativeIdentity {
		return nil, false
	}
	source, ok := g.composeNativeChurchSourceFrame()
	if !ok {
		return nil, false
	}
	frame, err := campaign.ComposeNativeChurchDialogueOverlay(
		source, a.dialogue, a.portrait,
	)
	if err != nil {
		return nil, false
	}
	frame, err = campaign.ComposeNativeChurchTextWithNameAt(
		frame, a.strings, a.font, 506, destination.NativeIdentity+1, 119*320+12,
	)
	return frame, err == nil
}

func (g *Game) drawNativeChurchTransferFull(screen *ebiten.Image) bool {
	frame, ok := g.composeNativeChurchTransferFullMessage()
	if !ok {
		return false
	}
	g.presentNativeClassFrame(screen, frame)
	return true
}

func (g *Game) beginNativeChurchTransferFullOpening() bool {
	final, ok := g.composeNativeChurchTransferFullMessage()
	if !ok {
		return false
	}
	source, ok := g.composeNativeChurchSourceFrame()
	if !ok {
		return false
	}
	frames, err := campaign.NativeClassListOpeningFrames(source, final)
	if err != nil || len(frames) != 6 {
		return false
	}
	g.nativeClassUIJob = &nativeClassUIJob{frames: frames}
	return true
}

func (g *Game) beginNativeChurchTransferFullClosing(after func()) bool {
	final, ok := g.composeNativeChurchTransferFullMessage()
	if !ok {
		return false
	}
	source, ok := g.composeNativeChurchSourceFrame()
	if !ok {
		return false
	}
	frames, err := campaign.NativeClassListClosingFrames(source, final)
	if err != nil || len(frames) != 5 {
		return false
	}
	g.nativeClassUIJob = &nativeClassUIJob{
		frames: frames, restore: source, after: after,
	}
	return true
}
