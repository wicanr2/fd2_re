package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

type nativeChurchUIJob struct {
	frames  [][]byte
	restore []byte
	frame   int
	drawn   bool
	after   func()
}

func parseNativeChurchShotState(spec string) (selection, pulse, gold int, ok bool) {
	parts := strings.Split(spec, ",")
	if len(parts) != 3 {
		return 0, 0, 0, false
	}
	selection, err := strconv.Atoi(parts[0])
	if err != nil || selection < 0 || selection > 3 {
		return 0, 0, 0, false
	}
	pulse, err = strconv.Atoi(parts[1])
	if err != nil || pulse < 0 || pulse > 3 {
		return 0, 0, 0, false
	}
	gold, err = strconv.Atoi(parts[2])
	if err != nil || gold < 0 || gold > 99999999 {
		return 0, 0, 0, false
	}
	return selection, pulse, gold, true
}

// setNativeChurchShotState 只固定正式教會主選單的可見原始狀態。它要求
// campaign、節點、資產與 production owner 都已存在，不建立 church、party
// 或像素；預檢失敗時完整回復，避免截到現代後備畫面。
func (g *Game) setNativeChurchShotState(selection, pulse, gold int) bool {
	if selection < 0 || selection > 3 || pulse < 0 || pulse > 3 ||
		gold < 0 || gold > 99999999 || g.camp == nil ||
		g.nativeClassUI == nil || g.churchMode != "menu" {
		return false
	}
	n := g.camp.Node()
	if n == nil || n.Type != "church" {
		return false
	}
	oldJob, oldSelection, oldPulse, oldGold :=
		g.nativeChurchUIJob, g.churchSel, g.nativeChurchUIPulse, g.gold
	oldClock, oldLastTick, oldHasTick :=
		g.nativeChurchUIClock, g.nativeChurchUILastTick, g.nativeChurchUIHasTick
	oldShotHold := g.nativeChurchUIShotHold
	rollback := func() {
		g.nativeChurchUIJob = oldJob
		g.churchSel = oldSelection
		g.nativeChurchUIPulse = oldPulse
		g.gold = oldGold
		g.nativeChurchUIClock = oldClock
		g.nativeChurchUILastTick = oldLastTick
		g.nativeChurchUIHasTick = oldHasTick
		g.nativeChurchUIShotHold = oldShotHold
	}
	g.nativeChurchUIJob = nil
	g.churchSel = selection
	g.resetNativeChurchUIPulse()
	g.nativeChurchUIPulse = pulse
	g.gold = gold
	g.nativeChurchUIShotHold = true
	if _, ok := g.composeNativeChurchMenuFrame(); !ok {
		rollback()
		return false
	}
	return true
}

func (g *Game) composeNativeChurchScene() ([]byte, bool) {
	a := g.nativeClassUI
	if a == nil || len(a.entries) <= 16 || g.nativeChurchTextIndex < 0 {
		return nil, false
	}
	frame, err := campaign.ComposeNativeChurchScene(
		a.background, a.entries[1], a.dialogue, a.digits, a.portrait,
		a.strings, a.font, g.gold, g.nativeChurchTextIndex,
	)
	return frame, err == nil
}

func (g *Game) composeNativeChurchSceneBase() ([]byte, bool) {
	a := g.nativeClassUI
	if a == nil || len(a.entries) <= 16 {
		return nil, false
	}
	frame, err := campaign.ComposeNativeChurchSceneBase(
		a.background, a.entries[1], a.dialogue, a.digits, a.portrait, g.gold,
	)
	return frame, err == nil
}

func (g *Game) composeNativeChurchSourceFrame() ([]byte, bool) {
	scene, ok := g.composeNativeChurchScene()
	if !ok {
		return nil, false
	}
	frame, err := campaign.NativeChurchMenuBase(scene)
	return frame, err == nil
}

func (g *Game) composeNativeChurchDialogueBase() ([]byte, bool) {
	a := g.nativeClassUI
	source, ok := g.composeNativeChurchSourceFrame()
	if !ok {
		return nil, false
	}
	frame, err := campaign.ComposeNativeChurchDialogueOverlay(source, a.dialogue, a.portrait)
	return frame, err == nil
}

func (g *Game) composeNativeChurchMenuFrame() ([]byte, bool) {
	a := g.nativeClassUI
	scene, ok := g.composeNativeChurchScene()
	if !ok || g.churchSel < 0 || g.churchSel > 3 {
		return nil, false
	}
	frame, err := campaign.ComposeNativeChurchMenuFrame(
		scene, a.entries, g.churchSel, g.nativeChurchUIPulse/2,
	)
	return frame, err == nil
}

func (g *Game) beginNativeChurchMenuOpening() bool {
	a := g.nativeClassUI
	scene, ok := g.composeNativeChurchScene()
	if !ok {
		return false
	}
	frames, err := campaign.NativeChurchMenuTransitionFrames(scene, a.entries, true)
	if err != nil || len(frames) != 4 {
		return false
	}
	g.resetNativeChurchUIPulse()
	g.nativeChurchUIJob = &nativeChurchUIJob{frames: frames}
	return true
}

func (g *Game) beginNativeChurchMenuClosing(after func()) bool {
	a := g.nativeClassUI
	scene, ok := g.composeNativeChurchScene()
	if !ok {
		return false
	}
	frames, err := campaign.NativeChurchMenuTransitionFrames(scene, a.entries, false)
	if err != nil || len(frames) != 4 {
		return false
	}
	restore, err := campaign.NativeChurchMenuBase(scene)
	if err != nil {
		return false
	}
	g.nativeChurchUIJob = &nativeChurchUIJob{
		frames: frames, restore: restore, after: after,
	}
	return true
}

func (g *Game) stepNativeChurchUILifecycle(now time.Time) {
	job := g.nativeChurchUIJob
	if job != nil && job.drawn {
		job.drawn = false
		if job.frame < len(job.frames) {
			job.frame++
			if job.frame < len(job.frames) || len(job.restore) != 0 {
				return
			}
		}
		if job.frame >= len(job.frames) {
			after := job.after
			g.nativeChurchUIJob = nil
			if after != nil {
				after()
			}
		}
	}
	if g.nativeChurchUIJob == nil && g.churchMode == "menu" && !g.nativeChurchUIShotHold {
		g.stepNativeChurchUIPulseTick(g.nativeChurchUIClock.Sample(now))
	}
}

func (g *Game) drawNativeChurchUIJob(screen *ebiten.Image) bool {
	job := g.nativeChurchUIJob
	if job == nil {
		return false
	}
	frame := job.frame
	if frame < len(job.frames) {
		g.presentNativeClassFrame(screen, job.frames[frame])
		job.drawn = true
		return true
	}
	if len(job.restore) == 320*200 {
		g.presentNativeClassFrame(screen, job.restore)
		job.drawn = true
		return true
	}
	return false
}

func (g *Game) drawNativeChurchMenu(screen *ebiten.Image) bool {
	if g.churchMode != "menu" {
		return false
	}
	frame, ok := g.composeNativeChurchMenuFrame()
	if !ok {
		return false
	}
	g.presentNativeClassFrame(screen, frame)
	return true
}

func (g *Game) nativeChurchUIBlocksInput() bool {
	return g.nativeChurchUIJob != nil
}

func (g *Game) resetNativeChurchUIPulse() {
	g.nativeChurchUIClock.Reset()
	g.nativeChurchUIPulse = 2
	g.nativeChurchUILastTick = 0
	g.nativeChurchUIHasTick = false
}

func (g *Game) stepNativeChurchUIPulseTick(rawTick int) {
	if !g.nativeChurchUIHasTick {
		g.nativeChurchUILastTick = rawTick
		g.nativeChurchUIHasTick = true
		return
	}
	delta := int16(uint16(rawTick) - uint16(g.nativeChurchUILastTick))
	if delta < 2 {
		return
	}
	g.nativeChurchUILastTick = rawTick
	g.nativeChurchUIPulse = (g.nativeChurchUIPulse + 1) & 3
}

func (g *Game) returnToNativeChurchMenu() {
	g.churchMode = "menu"
	g.churchIDs = nil
	g.churchRosterStart = 0
	g.churchStatusID = -1
	g.churchStatusPanel = nil
	g.churchCommandPanel = nil
	g.churchReviveID = -1
	g.churchReviveFee = 0
	g.churchBranches = nil
	g.churchClassID = -1
	g.churchSel = 0
	g.nativeChurchTextIndex = 586
	g.beginNativeChurchMenuOpening()
}
