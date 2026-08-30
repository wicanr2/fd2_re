package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type nativeFieldEvent61Job struct {
	plan     battle.NativeFieldEvent61Plan
	frames   [][]byte
	frame    int
	drawn    bool
	lastTick int
	hasTick  bool
	clock    nativeBIOSClock
	after    func()
}

func event61DialogueActions(g *Game, sceneIndex, line, count int) ([]battle.Action, bool) {
	lines := loadStoryScriptAt(
		"assets/story/ch26.json", "", &sceneIndex,
	)
	if line < 0 || count <= 0 || line+count > len(lines) {
		return nil, false
	}
	actions := make([]battle.Action, 0, count)
	for _, source := range lines[line : line+count] {
		speaker := source.Speaker
		if source.SpeakerSlot != nil {
			return nil, false
		}
		text, err := g.localizedStoryText(source)
		if err != nil {
			g.loadErr = "event61 locale: " + err.Error()
			return nil, false
		}
		actions = append(actions, battle.Action{Type: "dialogue", Speaker: speaker, Text: text})
	}
	return actions, true
}

// beginNativeFieldEvent61 owns the selector1 event only after a successful
// unit action. A coordinate without event61 returns false; an event61 whose
// editable/original provenance is malformed blocks its mutation and reports
// the error rather than silently finishing a guessed handler.
func (g *Game) beginNativeFieldEvent61(actor *battle.Unit, after func()) bool {
	if g == nil || g.st == nil || actor == nil || g.nativeFieldEvent61 != nil {
		return false
	}
	eventID, bound := battle.NativeFieldEventIDAt(g.st, actor.X, actor.Y, 1)
	if !bound || eventID != 61 {
		return false
	}
	plan, err := battle.PlanNativeFieldEvent61(g.st, actor, actor.X, actor.Y)
	if err != nil {
		g.loadErr = err.Error()
		g.msg = "事件 61 的原版證據不完整，已停止事件寫入"
		if after != nil {
			after()
		}
		return true
	}
	if plan.MissingItem {
		actions, ok := event61DialogueActions(g, 0, 10, 1)
		if !ok {
			g.loadErr = "event61: FDTXT2 editable dialogue unavailable"
			if after != nil {
				after()
			}
			return true
		}
		g.startBattleEvent(actions, after)
		return true
	}
	actions, ok := event61DialogueActions(g, 0, 11, 1)
	if !ok {
		g.loadErr = "event61: FDTXT3 editable dialogue unavailable"
		if after != nil {
			after()
		}
		return true
	}
	g.startBattleEvent(actions, func() {
		if err := g.beginNativeFieldEvent61Presentation(plan, after); err != nil {
			g.loadErr = err.Error()
			if after != nil {
				after()
			}
		}
	})
	return true
}

func (g *Game) beginNativeFieldEvent61Presentation(
	plan battle.NativeFieldEvent61Plan,
	after func(),
) error {
	if g.nativeFieldEvent61 != nil {
		return fmt.Errorf("event61: presentation is already active")
	}
	// 0x356b7 closes FDTXT #3 through 0x16c57/0x196cb before loading
	// FDOTHER#45. The resulting source is the neutral tactical redraw, not the
	// action/range overlay which led into selector1.
	if !g.resetNativeTargetField() {
		return fmt.Errorf(
			"event61: neutral target field is unavailable (%dx%d/%d)",
			g.st.W, g.st.H, len(g.st.NativeTileBlitModes),
		)
	}
	if !g.st.MaterializeNativeMapRangeMode(1) {
		return fmt.Errorf("event61: neutral range mode is unavailable")
	}
	if err := g.composeNativeMapFrame(); err != nil {
		return fmt.Errorf("event61: native source frame: %w", err)
	}
	if plan.Presentation.Resource != 45 {
		return fmt.Errorf("event61: unsupported presentation resource %d", plan.Presentation.Resource)
	}
	decoded, err := fdother.LoadSeparatedEvent61Frames(
		separatedAssetPath("animations/fdother_045_event61"),
	)
	if err != nil {
		return fmt.Errorf("event61: separated FDOTHER resource: %w", err)
	}
	if len(decoded) != plan.Presentation.Frames {
		return fmt.Errorf(
			"event61: FDOTHER frame count=%d want=%d",
			len(decoded), plan.Presentation.Frames,
		)
	}
	current := append([]byte(nil), g.nativeMapVGA...)
	frames := make([][]byte, len(decoded))
	for index, frame := range decoded {
		if err := frame.BlitAt(
			current, plan.Presentation.Stride,
			plan.Presentation.DestinationOffset,
			plan.Presentation.Transparent,
		); err != nil {
			return fmt.Errorf("event61: frame %d: %w", index, err)
		}
		frames[index] = append([]byte(nil), current...)
	}
	g.nativeFieldEvent61 = &nativeFieldEvent61Job{
		plan: plan, frames: frames, after: after,
	}
	return nil
}

func (g *Game) stepNativeFieldEvent61Tick(rawTick int) {
	job := g.nativeFieldEvent61
	if job == nil || !job.drawn {
		return
	}
	if !job.hasTick {
		job.lastTick, job.hasTick = rawTick, true
		return
	}
	delta := int16(uint16(rawTick) - uint16(job.lastTick))
	if int(delta) < job.plan.Presentation.DelayTicks {
		return
	}
	job.lastTick = rawTick
	job.drawn = false
	job.frame++
	if job.frame < len(job.frames) {
		return
	}
	after := job.after
	joined, err := battle.CommitNativeFieldEvent61(
		g.st, job.plan, len(job.frames),
	)
	g.nativeFieldEvent61 = nil
	if err != nil {
		g.loadErr = err.Error()
		if after != nil {
			after()
		}
		return
	}
	if err := g.persistNativeFieldEvent61Join(joined); err != nil {
		g.loadErr = err.Error()
		if after != nil {
			after()
		}
		return
	}
	actions, ok := event61DialogueActions(g, 1, 0, 10)
	if !ok {
		g.loadErr = "event61: FDTXT4 editable dialogue unavailable"
		if after != nil {
			after()
		}
		return
	}
	g.startBattleEvent(actions, after)
}

func (g *Game) persistNativeFieldEvent61Join(id int) error {
	if !campaign.JoinableCharacterID(id) || g.st == nil {
		return fmt.Errorf("event61: invalid JOIN character %d", id)
	}
	var joined *battle.Unit
	for _, unit := range g.st.Units {
		if unit != nil && unit.Fig == id && unit.Camp == battle.Own {
			if joined != nil {
				return fmt.Errorf("event61: character %d has multiple own records", id)
			}
			joined = unit
		}
	}
	if joined == nil {
		return fmt.Errorf("event61: character %d own record is absent", id)
	}
	g.initializeEquipmentBases(&battle.State{Units: []*battle.Unit{joined}})
	if g.partyRoster == nil {
		g.partyRoster = make(map[int]battle.Unit)
	}
	if g.partyMembers == nil {
		g.partyMembers = make(map[int]bool)
	}
	if !g.partyMembers[id] {
		g.partyMembers[id] = true
		g.partyJoinOrder = append(g.partyJoinOrder, id)
	}
	g.partyRoster[id] = cloneNativeShopUnit(*joined)
	return nil
}

func (g *Game) drawNativeFieldEvent61(screen *ebiten.Image) bool {
	job := g.nativeFieldEvent61
	if job == nil || job.frame < 0 || job.frame >= len(job.frames) ||
		g.nativeMapAssets == nil || len(g.nativeMapAssets.Palette) < 256 {
		return false
	}
	g.presentNativeClassFrameWithPalette(
		screen, job.frames[job.frame], g.nativeMapAssets.Palette,
	)
	job.drawn = true
	return true
}
