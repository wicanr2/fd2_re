package main

import (
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

type nativeClassUIJob struct {
	frames   [][]byte
	restore  []byte
	frame    int
	drawn    bool
	after    func()
	timeline []nativeClassUITimelineStep
	started  time.Time
	elapsed  time.Duration
	// 0x196CB 的戰場關框 owner 才宣告 0x19742；城鎮共用 job 不推定重繪。
	redrawNativeMapAfterRestore bool
}

type nativeClassUITimelineStep struct {
	frame    []byte
	palette  color.Palette
	duration time.Duration
}

func (g *Game) beginNativeClassListOpening() bool {
	final, ok := g.composeNativeClassListFrame()
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

func (g *Game) beginNativeClassListClosing(after func()) bool {
	final, ok := g.composeNativeClassListFrame()
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
	g.nativeClassUIJob = &nativeClassUIJob{frames: frames, restore: source, after: after}
	return true
}

func (g *Game) beginNativeClassConfirmationOpening() bool {
	question, ok := g.composeNativeClassConfirmationQuestion()
	if !ok {
		return false
	}
	frames, err := campaign.NativeClassConfirmationOpeningFrames(question, g.nativeClassUI.choices)
	if err != nil || len(frames) != 4 {
		return false
	}
	g.resetNativeClassUIPulse()
	g.nativeClassUIJob = &nativeClassUIJob{frames: frames}
	return true
}

func (g *Game) beginNativeClassConfirmationClosing(after func()) bool {
	question, ok := g.composeNativeClassConfirmationQuestion()
	if !ok {
		return false
	}
	choiceFrames, err := campaign.NativeClassConfirmationClosingFrames(question, g.nativeClassUI.choices)
	if err != nil || len(choiceFrames) != 4 {
		return false
	}
	source, ok := g.composeNativeChurchSourceFrame()
	if !ok {
		return false
	}
	dialogue, ok := g.composeNativeChurchDialogueBase()
	if !ok {
		return false
	}
	dialogueFrames, err := campaign.NativeClassListClosingFrames(source, dialogue)
	if err != nil || len(dialogueFrames) != 5 {
		return false
	}
	frames := append(choiceFrames, dialogueFrames...)
	g.nativeClassUIJob = &nativeClassUIJob{frames: frames, restore: source, after: after}
	return true
}

// stepNativeClassUILifecycle advances only a frame that Draw acknowledged.
// The continuation runs after the final closing frame has been presented.
func (g *Game) stepNativeClassUILifecycle(now time.Time) {
	job := g.nativeClassUIJob
	if job != nil {
		g.nativeConfirmationMouthOwner = nil
	}
	if job != nil && len(job.timeline) != 0 {
		if job.started.IsZero() {
			job.started = now
		}
		job.elapsed = now.Sub(job.started)
		total := time.Duration(0)
		for _, step := range job.timeline {
			total += step.duration
		}
		if job.frame == 1 && job.drawn {
			after := job.after
			g.nativeClassUIJob = nil
			if after != nil {
				after()
			}
		}
		return
	}
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
			g.nativeClassUIJob = nil
			if job.redrawNativeMapAfterRestore && len(job.restore) != 0 {
				g.redrawNativeMapHUD()
			}
			if after != nil {
				after()
			}
		}
	}
	preparationConfirm := false
	preparationPrompt := false
	if g.camp != nil {
		if node := g.camp.Node(); node != nil && node.Type == "preparation" {
			preparationConfirm = g.prepConfirm
			preparationPrompt = !g.prepSelecting && !g.prepConfirm && !g.prepRecordSlots
		}
	}
	if g.nativeClassUIJob == nil &&
		(g.churchMode == "class_confirm" || g.churchMode == "revive_confirm" ||
			preparationConfirm || preparationPrompt || g.nativeSystemEndTurnConfirm) {
		owner := g.nativeConfirmationMouthWaitingOwner()
		g.stepNativeConfirmationMouth(owner, false)
		advanced := g.stepNativeClassUIPulseTick(g.nativeClassUIClock.Sample(now))
		g.stepNativeConfirmationMouth(owner, advanced)
	} else {
		g.stepNativeConfirmationMouth(nil, false)
	}
}

func (g *Game) drawNativeClassUIJob(screen *ebiten.Image) bool {
	job := g.nativeClassUIJob
	if job == nil || job.frame < 0 {
		return false
	}
	if len(job.timeline) != 0 {
		elapsed := job.elapsed
		total := time.Duration(0)
		for _, candidate := range job.timeline {
			total += candidate.duration
		}
		step := job.timeline[len(job.timeline)-1]
		for _, candidate := range job.timeline {
			if elapsed < candidate.duration {
				step = candidate
				break
			}
			elapsed -= candidate.duration
		}
		if job.elapsed >= total {
			job.frame = 1
		}
		g.presentNativeClassFrameWithPalette(screen, step.frame, step.palette)
		job.drawn = true
		return true
	}
	if job.frame < len(job.frames) {
		g.presentNativeClassFrame(screen, job.frames[job.frame])
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

func (g *Game) nativeClassUIBlocksInput() bool {
	return g.nativeClassUIJob != nil
}

func (g *Game) resetNativeClassUIPulse() {
	g.nativeClassUIClock.Reset()
	g.nativeClassUIPulse = 0
	g.nativeClassUILastTick = 0
	g.nativeClassUIHasTick = false
}

// 0x19B1E sign-extends the BIOS word before a 32-bit subtraction. A negative
// delta also advances once; only deltas zero and one wait. See #35 evidence
// docs/data/ida/fd2_confirmation_pulse_20261005.json.
func (g *Game) stepNativeClassUIPulseTick(rawTick int) bool {
	return g.stepNativeClassUIPulseTicks(rawTick, rawTick)
}

// The DOS loop reads again at 0x19B4E before storing the latch at 0x19B51.
// The host-time adapter approximates that short gap with the same sample.
func (g *Game) stepNativeClassUIPulseTicks(rawTick, latchTick int) bool {
	rawTick = int(int16(uint16(rawTick)))
	latchTick = int(int16(uint16(latchTick)))
	if !g.nativeClassUIHasTick {
		g.nativeClassUILastTick = rawTick
		g.nativeClassUIHasTick = true
		return false
	}
	delta := rawTick - g.nativeClassUILastTick
	if delta >= 0 && delta < 2 {
		return false
	}
	g.nativeClassUILastTick = latchTick
	g.nativeClassUIPulse = (g.nativeClassUIPulse + 1) & 3
	return true
}

func (g *Game) returnToNativeClassList() {
	g.churchMode = "class"
	g.churchBranches = nil
	g.churchClassID = -1
	g.churchSel = 0
	g.churchVerticalStart = 0
	g.beginNativeClassListOpening()
}
