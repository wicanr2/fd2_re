package main

import (
	"errors"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type nativePreparationUIAssets struct {
	roster   *fdother.NativePreparationAssets
	status   battle.NativeItemPanelDataAssets
	choices  []fdother.RawCell
	dialogue []fdother.RawCell
	portrait dato.Frame
}

func (g *Game) stepNativePreparationCycleTick(rawTick int) {
	state := fdicon.AdvanceNativeMapSpriteCycles(
		fdicon.NativeMapSpriteCycleState{
			Idle: g.prepIdleCycle, LastTimerTick: g.prepLastTick,
		},
		rawTick,
	)
	g.prepIdleCycle = state.Idle
	g.prepLastTick = state.LastTimerTick
}

func (g *Game) stepNativePreparationUILifecycle(now time.Time) {
	if g.camp == nil || g.nativePreparationUI == nil || !g.prepSelecting {
		return
	}
	n := g.camp.Node()
	if n == nil || n.Type != "preparation" {
		return
	}
	if g.prepShotCycleFrozen {
		return
	}
	g.stepNativePreparationCycleTick(g.prepClock.Sample(now))
}

func loadNativePreparationUIAssets() (*nativePreparationUIAssets, error) {
	fdotherPath := nativeFDOTHERPath()
	if fdotherPath == "" {
		return nil, errors.New("native preparation UI: FDOTHER.DAT unavailable")
	}
	roster, err := fdother.DecodeNativePreparationAssets(
		fdotherPath,
		filepath.Join(filepath.Dir(fdotherPath), "FDICON.B24"),
	)
	if err != nil {
		return nil, err
	}
	status, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		return nil, err
	}
	choices, err := fdother.DecodeRawCellResource(fdotherPath, 2)
	if err != nil {
		return nil, err
	}
	resource5, err := fdother.ReadResource(fdotherPath, 5)
	if err != nil {
		return nil, err
	}
	dialogue := make([]fdother.RawCell, 20)
	for index := 1; index <= 19; index++ {
		dialogue[index], err = fdother.ParseLMI1RawEntry(resource5, index)
		if err != nil {
			return nil, err
		}
	}
	portraits, err := loadNativeSeparatedPortrait(0x4b)
	if err != nil || len(portraits) == 0 {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("native preparation UI: DATO#75 has no frames")
	}
	return &nativePreparationUIAssets{
		roster: roster, status: status, choices: choices,
		dialogue: dialogue, portrait: portraits[0],
	}, nil
}

func (g *Game) composeNativePreparationFrame() ([]byte, bool) {
	if g.camp == nil || g.nativePreparationUI == nil || g.nativeClassUI == nil ||
		(!g.prepSelecting && !g.prepConfirm) || len(g.prepIDs) == 0 {
		return nil, false
	}
	n := g.camp.Node()
	if n == nil || n.Type != "preparation" {
		return nil, false
	}
	keys := make([]int, len(g.prepIDs))
	selected := make([]bool, len(g.prepIDs))
	for i, id := range g.prepIDs {
		unit, ok := g.partyRoster[id]
		if !ok || !unit.HasMapSelectorKey {
			return nil, false
		}
		keys[i] = unit.MapSelectorKey
		selected[i] = g.partyDeploy[id]
	}
	cycle, err := fdicon.NativeFrameIndex(0, false, g.prepIdleCycle, 0)
	if err != nil {
		return nil, false
	}
	frame, err := fdother.ComposeNativePreparationFrame(
		g.nativePreparationUI.roster,
		keys, selected, g.prepSel, cycle, g.prepLimit,
	)
	if err != nil {
		return nil, false
	}
	unit := g.partyRoster[g.prepIDs[g.prepSel]]
	record, err := battle.NativeItemPanelRecordForUnit(&unit)
	if err != nil {
		return nil, false
	}
	if err := battle.RenderNativeItemPanelData(
		g.nativePreparationUI.status, record, frame,
	); err != nil {
		return nil, false
	}
	return frame, true
}

func (g *Game) composeNativePreparationConfirmationFrame() ([]byte, bool) {
	if !g.prepConfirm || g.nativeClassUI == nil {
		return nil, false
	}
	background, ok := g.composeNativePreparationFrame()
	if !ok {
		return nil, false
	}
	frame, err := campaign.ComposeNativePreparationConfirmationFrame(
		background,
		g.nativePreparationUI.choices,
		g.nativePreparationUI.dialogue,
		g.nativePreparationUI.portrait,
		g.nativePreparationUI.status.Strings,
		g.nativePreparationUI.status.Font,
		g.prepConfirmSel,
		g.nativeClassUIPulse/2,
	)
	if err != nil {
		return nil, false
	}
	return frame, true
}

func (g *Game) nativePreparationPromptActive() bool {
	if g.camp == nil || g.prepSelecting || g.prepConfirm {
		return false
	}
	node := g.camp.Node()
	return node != nil && node.Type == "preparation"
}

func (g *Game) composeNativePreparationPromptDialogue() ([]byte, bool) {
	if !g.nativePreparationPromptActive() ||
		g.nativePreparationUI == nil || len(g.prepPromptSource) != 320*200 {
		return nil, false
	}
	frame, err := campaign.ComposeNativePreparationConfirmationDialogue(
		g.prepPromptSource,
		g.nativePreparationUI.dialogue,
		g.nativePreparationUI.portrait,
	)
	return frame, err == nil
}

func (g *Game) composeNativePreparationPromptQuestion() ([]byte, bool) {
	if !g.nativePreparationPromptActive() ||
		g.nativePreparationUI == nil || len(g.prepPromptSource) != 320*200 {
		return nil, false
	}
	node := g.camp.Node()
	var (
		frame []byte
		err   error
	)
	if node.Cancel != "" {
		frame, err = campaign.ComposeNativePreparationDepartureQuestion(
			g.prepPromptSource,
			g.nativePreparationUI.dialogue,
			g.nativePreparationUI.portrait,
			g.nativePreparationUI.status.Strings,
			g.nativePreparationUI.status.Font,
		)
	} else {
		frame, err = campaign.ComposeNativePreparationRecordQuestion(
			g.prepPromptSource,
			g.nativePreparationUI.dialogue,
			g.nativePreparationUI.portrait,
			g.nativePreparationUI.status.Strings,
			g.nativePreparationUI.status.Font,
		)
	}
	return frame, err == nil
}

func (g *Game) composeNativePreparationPromptFrame() ([]byte, bool) {
	question, ok := g.composeNativePreparationPromptQuestion()
	if !ok {
		return nil, false
	}
	frame, err := campaign.ComposeNativeConfirmationChoices(
		question,
		g.nativePreparationUI.choices,
		g.prepConfirmSel,
		g.nativeClassUIPulse/2,
	)
	return frame, err == nil
}

func (g *Game) beginNativePreparationPromptOpening() bool {
	if len(g.prepPromptSource) != 320*200 {
		return false
	}
	dialogue, ok := g.composeNativePreparationPromptDialogue()
	if !ok {
		return false
	}
	question, ok := g.composeNativePreparationPromptQuestion()
	if !ok {
		return false
	}
	frames, err := campaign.NativePreparationConfirmationOpeningFrames(
		g.prepPromptSource, dialogue, question, g.nativePreparationUI.choices,
	)
	if err != nil || len(frames) != 10 {
		return false
	}
	g.resetNativeClassUIPulse()
	g.nativeClassUIJob = &nativeClassUIJob{frames: frames}
	return true
}

func (g *Game) beginNativePreparationPromptClosing(after func()) bool {
	if len(g.prepPromptSource) != 320*200 {
		return false
	}
	dialogue, ok := g.composeNativePreparationPromptDialogue()
	if !ok {
		return false
	}
	question, ok := g.composeNativePreparationPromptQuestion()
	if !ok {
		return false
	}
	frames, err := campaign.NativePreparationConfirmationClosingFrames(
		g.prepPromptSource, dialogue, question, g.nativePreparationUI.choices,
	)
	if err != nil || len(frames) != 9 {
		return false
	}
	g.nativeClassUIJob = &nativeClassUIJob{
		frames:  frames,
		restore: append([]byte(nil), g.prepPromptSource...),
		after:   after,
	}
	return true
}

func (g *Game) composeNativePreparationConfirmationDialogue() ([]byte, bool) {
	background, ok := g.composeNativePreparationFrame()
	if !ok {
		return nil, false
	}
	frame, err := campaign.ComposeNativePreparationConfirmationDialogue(
		background,
		g.nativePreparationUI.dialogue,
		g.nativePreparationUI.portrait,
	)
	return frame, err == nil
}

func (g *Game) composeNativePreparationConfirmationQuestion() ([]byte, bool) {
	background, ok := g.composeNativePreparationFrame()
	if !ok {
		return nil, false
	}
	frame, err := campaign.ComposeNativePreparationConfirmationQuestion(
		background,
		g.nativePreparationUI.dialogue,
		g.nativePreparationUI.portrait,
		g.nativePreparationUI.status.Strings,
		g.nativePreparationUI.status.Font,
	)
	return frame, err == nil
}

func (g *Game) beginNativePreparationConfirmationOpening() bool {
	source, ok := g.composeNativePreparationFrame()
	if !ok {
		return false
	}
	dialogue, ok := g.composeNativePreparationConfirmationDialogue()
	if !ok {
		return false
	}
	question, ok := g.composeNativePreparationConfirmationQuestion()
	if !ok {
		return false
	}
	frames, err := campaign.NativePreparationConfirmationOpeningFrames(
		source, dialogue, question, g.nativePreparationUI.choices,
	)
	if err != nil || len(frames) != 10 {
		return false
	}
	g.resetNativeClassUIPulse()
	g.nativeClassUIJob = &nativeClassUIJob{frames: frames}
	return true
}

func (g *Game) beginNativePreparationConfirmationClosing(after func()) bool {
	source, ok := g.composeNativePreparationFrame()
	if !ok {
		return false
	}
	dialogue, ok := g.composeNativePreparationConfirmationDialogue()
	if !ok {
		return false
	}
	question, ok := g.composeNativePreparationConfirmationQuestion()
	if !ok {
		return false
	}
	frames, err := campaign.NativePreparationConfirmationClosingFrames(
		source, dialogue, question, g.nativePreparationUI.choices,
	)
	if err != nil || len(frames) != 9 {
		return false
	}
	g.nativeClassUIJob = &nativeClassUIJob{
		frames: frames, restore: source, after: after,
	}
	return true
}

func (g *Game) drawNativePreparation(screen *ebiten.Image) bool {
	if g.drawNativeClassUIJob(screen) {
		return true
	}
	if g.nativePreparationPromptActive() {
		frame, ok := g.composeNativePreparationPromptFrame()
		if !ok {
			return false
		}
		g.presentNativeClassFrame(screen, frame)
		return true
	}
	var (
		frame []byte
		ok    bool
	)
	if g.prepConfirm {
		frame, ok = g.composeNativePreparationConfirmationFrame()
	} else {
		frame, ok = g.composeNativePreparationFrame()
	}
	if !ok {
		return false
	}
	g.presentNativeClassFrame(screen, frame)
	return true
}
