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
	return g.prepareNativeUnitStatus(&unit)
}

// prepareNativeUnitStatus 共用原版 0x17AED 的面板，不改呼叫者的角色或返回路徑。
func (g *Game) prepareNativeUnitStatus(unit *battle.Unit) ([]byte, []byte, bool) {
	_, status, commands, ok := g.prepareNativeUnitStatusData(unit)
	return status, commands, ok
}

func (g *Game) prepareNativeUnitStatusData(unit *battle.Unit) (*nativeUnitStatusData, []byte, []byte, bool) {
	if unit == nil {
		return nil, nil, nil, false
	}
	assetPackRoot := separatedAssetPath("")
	portraitRoot := separatedAssetPath("portraits")
	if assetPackRoot == "" {
		return nil, nil, nil, false
	}
	record, err := battle.NativeItemPanelRecordForUnit(unit)
	if err != nil {
		return nil, nil, nil, false
	}
	base := make([]byte, 320*200)
	if err := battle.RenderNativeItemPanelBaseResources(
		assetPackRoot, portraitRoot, int(record[7]), base,
	); err != nil {
		return nil, nil, nil, false
	}
	assets, err := battle.LoadNativeItemPanelDataAssets(assetPackRoot)
	if err != nil {
		return nil, nil, nil, false
	}
	itemRows, err := battle.LoadNativeItemEffectRowPrefix(
		assetPath("assets/data/native_item_effect_rows.json"),
	)
	if err != nil {
		return nil, nil, nil, false
	}
	data := &nativeUnitStatusData{record: record, base: base, itemRows: itemRows, assets: assets,
		commandIDs: unit.NativeCommandIDs(), commandBook: append([]battle.NativeCommandRecord(nil), g.nativeCommandBook...)}
	status, ok := data.compose(assets, false)
	if !ok {
		return nil, nil, nil, false
	}
	if len(data.commandIDs) == 0 {
		return data, status, nil, true
	}
	commands, ok := data.compose(assets, true)
	if !ok {
		return nil, nil, nil, false
	}
	return data, status, commands, true
}

func (g *Game) beginNativeChurchStatus(id int) bool {
	unit, ok := g.partyRoster[id]
	if !ok {
		return false
	}
	data, status, commands, ok := g.prepareNativeUnitStatusData(&unit)
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
	g.churchStatusData = data
	g.nativeClassUIJob = &nativeClassUIJob{frames: frames,
		statusText: snapshotNativeStatusTransition(source, data, nativeStatusOpening, false)}
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
		frames:     frames,
		after:      func() { g.churchMode = "status_commands" },
		statusText: snapshotNativeStatusTransition(source, g.churchStatusData, nativeStatusToCommands, false),
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
		statusText: snapshotNativeStatusTransition(source, g.churchStatusData, nativeStatusClosing, g.churchMode == nativeChurchModeStatusCommands),
	}
}

func (g *Game) returnToNativeStatusRoster() {
	g.churchMode = "status_roster"
	g.churchStatusID = -1
	g.churchStatusPanel = nil
	g.churchCommandPanel = nil
	g.churchStatusData = nil
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
	if g.drawModernHDUnitStatus(screen, source, g.churchStatusData, g.churchMode == nativeChurchModeStatusCommands, g.nativeClassUI.palette) {
		return true
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
	regions, ok := nativeChurchPanelRegions(index)
	if !ok || len(source) != 320*200 || len(panel) != 320*200 {
		return nil, errors.New("native church status: invalid panel buffers")
	}
	frame := append([]byte(nil), source...)
	for _, region := range regions {
		copyNativeChurchPanelRegion(frame, panel, region)
	}
	return frame, nil
}

func nativeChurchPanelRegions(index int) ([]battle.NativeItemPanelRegion, bool) {
	pass, err := battle.NativeItemPanelFrameFor(index)
	return []battle.NativeItemPanelRegion{pass.Left, pass.Upper, pass.Bottom}, err == nil
}

func nativeChurchBottomPanelFrame(source, panel []byte, step int) ([]byte, error) {
	regions, ok := nativeChurchBottomPanelRegions(step)
	if len(source) != 320*200 || len(panel) != 320*200 || !ok {
		return nil, errors.New("native church status: invalid bottom transition")
	}
	frame := append([]byte(nil), source...)
	for _, region := range regions {
		copyNativeChurchPanelRegion(frame, panel, region)
	}
	return frame, nil
}

func nativeChurchBottomPanelRegions(step int) ([]battle.NativeItemPanelRegion, bool) {
	if step < 0 || step > 6 {
		return nil, false
	}
	regions := []battle.NativeItemPanelRegion{{
		Enabled: true, SourceX: 5, SourceY: 7, DestX: 5, DestY: 7, Width: 86, Height: 86,
	}, {
		Enabled: true, SourceX: 92, SourceY: 7, DestX: 92, DestY: 7, Width: 223, Height: 86,
	}}
	height := 102
	destinationY := 94 + 16*step
	if destinationY+height > 200 {
		height = 200 - destinationY
	}
	regions = append(regions, battle.NativeItemPanelRegion{
		Enabled: height > 0, SourceX: 5, SourceY: 94,
		DestX: 5, DestY: destinationY, Width: 310, Height: height,
	})
	return regions, true
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
