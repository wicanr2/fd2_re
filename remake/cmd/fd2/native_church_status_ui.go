package main

import (
	"errors"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func (g *Game) prepareNativeChurchStatus(id int) ([]byte, []byte, bool) {
	unit, ok := g.partyRoster[id]
	if !ok {
		return nil, nil, false
	}
	fdotherPath := nativeFDOTHERPath()
	assetPackRoot := separatedAssetPath("")
	portraitRoot := separatedAssetPath("portraits")
	if fdotherPath == "" || assetPackRoot == "" {
		return nil, nil, false
	}
	record, err := battle.NativeItemPanelRecordForUnit(&unit)
	if err != nil {
		return nil, nil, false
	}
	base := make([]byte, 320*200)
	if err := battle.RenderNativeItemPanelResources(
		fdotherPath, assetPackRoot, portraitRoot, record, base,
	); err != nil {
		return nil, nil, false
	}
	assets, err := battle.LoadNativeItemPanelDataAssets(assetPackRoot)
	if err != nil {
		return nil, nil, false
	}
	itemRows, err := battle.LoadNativeItemEffectRowPrefix(
		assetPath("assets/data/native_item_effect_rows.json"),
	)
	if err != nil {
		return nil, nil, false
	}
	status := append([]byte(nil), base...)
	if err := battle.RenderNativeItemPanelRows(assets, record, -1, itemRows, status); err != nil {
		return nil, nil, false
	}
	ids := unit.NativeCommandIDs()
	if len(ids) == 0 {
		return status, nil, true
	}
	commands := append([]byte(nil), base...)
	if err := battle.RenderNativeCommandOverlay(
		assets, ids, g.nativeCommandBook, -1, commands,
	); err != nil {
		return nil, nil, false
	}
	return status, commands, true
}

func (g *Game) beginNativeChurchStatus(id int) bool {
	status, commands, ok := g.prepareNativeChurchStatus(id)
	if !ok {
		return false
	}
	source, ok := g.composeNativeChurchSourceFrame()
	if !ok {
		return false
	}
	frames, err := nativeChurchPanelFrames(source, status, true)
	if err != nil {
		return false
	}
	g.churchMode = "status_view"
	g.churchStatusID = id
	g.churchStatusPanel = status
	g.churchCommandPanel = commands
	g.nativeClassUIJob = &nativeClassUIJob{frames: frames}
	return true
}

func (g *Game) beginNativeChurchStatusCommandTransition() bool {
	if len(g.churchStatusPanel) != 320*200 || len(g.churchCommandPanel) != 320*200 {
		return false
	}
	source, ok := g.composeNativeChurchSourceFrame()
	if !ok {
		return false
	}
	frames := make([][]byte, 0, 14)
	for i := 0; i <= 6; i++ {
		frame, err := nativeChurchBottomPanelFrame(source, g.churchStatusPanel, i)
		if err != nil {
			return false
		}
		frames = append(frames, frame)
	}
	for i := 6; i >= 0; i-- {
		frame, err := nativeChurchBottomPanelFrame(source, g.churchCommandPanel, i)
		if err != nil {
			return false
		}
		frames = append(frames, frame)
	}
	g.nativeClassUIJob = &nativeClassUIJob{
		frames: frames,
		after:  func() { g.churchMode = "status_commands" },
	}
	return true
}

func (g *Game) closeNativeChurchStatus(panel []byte) {
	source, ok := g.composeNativeChurchSourceFrame()
	if !ok {
		g.returnToNativeStatusRoster()
		return
	}
	frames, err := nativeChurchPanelFrames(source, panel, false)
	if err != nil {
		g.returnToNativeStatusRoster()
		return
	}
	g.nativeClassUIJob = &nativeClassUIJob{
		frames: frames, restore: source, after: g.returnToNativeStatusRoster,
	}
}

func (g *Game) returnToNativeStatusRoster() {
	g.churchMode = "status_roster"
	g.churchStatusID = -1
	g.churchStatusPanel = nil
	g.churchCommandPanel = nil
	g.beginNativeChurchRosterOpening()
}

func (g *Game) drawNativeChurchStatus(screen *ebiten.Image) bool {
	var panel []byte
	switch g.churchMode {
	case "status_view":
		panel = g.churchStatusPanel
	case "status_commands":
		panel = g.churchCommandPanel
	default:
		return false
	}
	source, ok := g.composeNativeChurchSourceFrame()
	if !ok {
		return false
	}
	frame, err := nativeChurchPanelFrame(source, panel, 0)
	if err != nil {
		return false
	}
	g.presentNativeClassFrame(screen, frame)
	return true
}

func nativeChurchPanelFrames(source, panel []byte, opening bool) ([][]byte, error) {
	if len(source) != 320*200 || len(panel) != 320*200 {
		return nil, errors.New("native church status: panel frames require two 320x200 buffers")
	}
	frames := make([][]byte, 0, 12)
	if opening {
		for i := 11; i >= 0; i-- {
			frame, err := nativeChurchPanelFrame(source, panel, i)
			if err != nil {
				return nil, err
			}
			frames = append(frames, frame)
		}
		return frames, nil
	}
	for i := 0; i <= 11; i++ {
		frame, err := nativeChurchPanelFrame(source, panel, i)
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

func nativeChurchPanelFrame(source, panel []byte, index int) ([]byte, error) {
	pass, err := battle.NativeItemPanelFrameFor(index)
	if err != nil || len(source) != 320*200 || len(panel) != 320*200 {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("native church status: invalid panel buffers")
	}
	frame := append([]byte(nil), source...)
	for _, region := range []battle.NativeItemPanelRegion{pass.Left, pass.Upper, pass.Bottom} {
		copyNativeChurchPanelRegion(frame, panel, region)
	}
	return frame, nil
}

func nativeChurchBottomPanelFrame(source, panel []byte, step int) ([]byte, error) {
	if len(source) != 320*200 || len(panel) != 320*200 || step < 0 || step > 6 {
		return nil, errors.New("native church status: invalid bottom transition")
	}
	frame := append([]byte(nil), source...)
	copyNativeChurchPanelRegion(frame, panel, battle.NativeItemPanelRegion{
		Enabled: true, SourceX: 5, SourceY: 7, DestX: 5, DestY: 7, Width: 86, Height: 86,
	})
	copyNativeChurchPanelRegion(frame, panel, battle.NativeItemPanelRegion{
		Enabled: true, SourceX: 92, SourceY: 7, DestX: 92, DestY: 7, Width: 223, Height: 86,
	})
	height := 102
	destinationY := 94 + 16*step
	if destinationY+height > 200 {
		height = 200 - destinationY
	}
	copyNativeChurchPanelRegion(frame, panel, battle.NativeItemPanelRegion{
		Enabled: height > 0, SourceX: 5, SourceY: 94,
		DestX: 5, DestY: destinationY, Width: 310, Height: height,
	})
	return frame, nil
}

func copyNativeChurchPanelRegion(dst, src []byte, region battle.NativeItemPanelRegion) {
	if !region.Enabled || region.Width <= 0 || region.Height <= 0 {
		return
	}
	for y := 0; y < region.Height; y++ {
		source := (region.SourceY+y)*320 + region.SourceX
		destination := (region.DestY+y)*320 + region.DestX
		copy(dst[destination:destination+region.Width], src[source:source+region.Width])
	}
}
