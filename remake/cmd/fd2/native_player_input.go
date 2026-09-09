package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// 中立輸入依 docs/knowledge-base/92 的 0x117E7／0x1A781 契約。
// raw 篩選與索引獨立保存，不以顯示名稱、Camp 或 Acted 猜造原始旗標。
func (g *Game) cycleNativePlayerUnit() {
	if g.st == nil || !g.st.HasNativeMapViewState || g.sel != nil || len(g.st.Units) == 0 {
		return
	}
	records, err := battle.NativeAIScoringRecords(g.st.Units)
	if err != nil {
		g.loadErr = "native player cycle: " + err.Error()
		return
	}
	count := len(g.st.Units)
	if g.nativeNextPlayerIndex < 0 || g.nativeNextPlayerIndex >= count {
		g.loadErr = "native player cycle: index outside runtime records"
		return
	}
	for offset := 0; offset < count; offset++ {
		i := (g.nativeNextPlayerIndex + offset) % count
		r := records[i*0x50 : (i+1)*0x50]
		if r[5]&0x85 == 0 && r[6] == 2 {
			if g.beginNativePlayerFocus(g.st.Units[i]) {
				g.nativeNextPlayerIndex = (i + 1) % count
			}
			return
		}
	}
}

func (g *Game) beginNativePlayerFocus(u *battle.Unit) bool {
	if g.st == nil || !g.st.HasNativeMapViewState || u == nil || !u.HasNativeMapPresentation {
		g.loadErr = "native player focus: raw position unavailable"
		return false
	}
	p := u.NativeMapPresentation
	if int(p.X) >= g.st.W || int(p.Y) >= g.st.H {
		g.loadErr = "native player focus: position outside field"
		return false
	}
	g.nativePlayerFocus = &battle.Cell{X: int(p.X), Y: int(p.Y)}
	return true
}

func (g *Game) stepNativePlayerFocus() {
	p := g.nativePlayerFocus
	if p == nil || g.st == nil {
		return
	}
	v := g.st.NativeMapViewState
	dx, dy := 0, 0
	switch {
	case v.CursorX > p.X:
		dx = -1
	case v.CursorX < p.X:
		dx = 1
	case v.CursorY > p.Y:
		dy = -1
	case v.CursorY < p.Y:
		dy = 1
	default:
		g.nativePlayerFocus = nil
		return
	}
	if moved, ok := g.st.MoveNativeMapCursor(dx, dy); !ok || !moved {
		g.loadErr = "native player focus: cursor step rejected"
		g.nativePlayerFocus = nil
		return
	}
	g.syncNativeMapView()
	if g.st.HasNativeMapHUDState {
		v = g.st.NativeMapViewState
		g.st.AdvanceNativeMapHUDAnchor(v.VisibleCursorX, v.VisibleCursorY)
	}
	if g.curX == p.X && g.curY == p.Y {
		g.nativePlayerFocus = nil
	}
}

// inspectNativePlayerUnit 的回傳值表示已消費確認鍵，包括原版拒絕的特殊角色。
func (g *Game) inspectNativePlayerUnit(u *battle.Unit, force bool) (bool, error) {
	records, err := battle.NativeAIScoringRecords([]*battle.Unit{u})
	if err != nil {
		return true, fmt.Errorf("native player status: %w", err)
	}
	r := records[:0x50]
	if r[7] == 0x79 || r[0x1f] == 10 {
		return true, nil
	}
	if !force && r[6] == 2 && r[5]&0x80 == 0 && r[0x26] == 0 {
		return false, nil
	}
	return true, g.beginNativePlayerStatus(u)
}

type nativePlayerStatusState struct {
	source, status, commands, steady []byte
	frames                           [][]byte
	frame                            int
	phase                            string
	drawn                            bool
}

func (g *Game) beginNativePlayerStatus(u *battle.Unit) error {
	status, commands, ok := g.prepareNativeUnitStatus(u)
	if !ok || len(g.nativeUIPalette) != 256 {
		return fmt.Errorf("native player status: panel resources unavailable")
	}
	if err := g.composeNativeMapFrame(); err != nil {
		return fmt.Errorf("native player status: %w", err)
	}
	source := append([]byte(nil), g.nativeMapVGA...)
	frames, err := nativeChurchPanelFrames(source, status, true)
	if err != nil {
		return err
	}
	steady, err := nativeChurchPanelFrame(source, status, 0)
	if err != nil {
		return err
	}
	g.nativePlayerStatus = &nativePlayerStatusState{
		source: source, status: status, commands: commands, steady: steady,
		frames: frames, phase: "opening",
	}
	return nil
}

func (g *Game) stepNativePlayerStatus() {
	s := g.nativePlayerStatus
	if s == nil || !s.drawn || len(s.frames) == 0 {
		return
	}
	s.drawn = false
	s.frame++
	if s.frame < len(s.frames) {
		return
	}
	if s.phase == "closing" {
		g.nativePlayerStatus = nil
		return
	}
	s.frames, s.frame = nil, 0
	if s.phase == "opening" {
		s.phase = "status"
	} else {
		s.phase = "commands"
	}
}

func (g *Game) advanceNativePlayerStatus() {
	s := g.nativePlayerStatus
	if s == nil || len(s.frames) != 0 {
		return
	}
	if s.phase == "status" && len(s.commands) != 0 {
		for i := 0; i <= 6; i++ {
			frame, err := nativeChurchBottomPanelFrame(s.source, s.status, i)
			if err != nil {
				g.loadErr = err.Error()
				return
			}
			s.frames = append(s.frames, frame)
		}
		for i := 6; i >= 0; i-- {
			frame, err := nativeChurchBottomPanelFrame(s.source, s.commands, i)
			if err != nil {
				g.loadErr = err.Error()
				return
			}
			s.frames = append(s.frames, frame)
		}
		s.steady, _ = nativeChurchPanelFrame(s.source, s.commands, 0)
		s.phase, s.frame, s.drawn = "transition", 0, false
		return
	}
	panel := s.status
	if s.phase == "commands" {
		panel = s.commands
	}
	frames, err := nativeChurchPanelFrames(s.source, panel, false)
	if err != nil {
		g.loadErr = err.Error()
		return
	}
	s.frames, s.phase, s.frame, s.drawn = frames, "closing", 0, false
}

func (g *Game) drawNativePlayerStatus(screen *ebiten.Image) {
	s := g.nativePlayerStatus
	if s == nil {
		return
	}
	frame := s.steady
	if len(s.frames) > 0 {
		frame = s.frames[s.frame]
	}
	g.presentNativeClassFrameWithPalette(screen, frame, g.nativeUIPalette)
	s.drawn = true
}
