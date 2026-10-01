package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// nativeDefeatJob 擁有 READY §5.2.1 的有限呈現，不讀鍵盤、不寫存檔。
// 原始定位與分級：fd2_pending_code1_return_title_20261001.json。
type nativeDefeatJob struct {
	plan               []fdother.PendingCode1PresentationStep
	pos                int
	frames             []fdother.Frame
	vga, dac, baseline []byte
	palette            color.Palette
	ramp               *nativePaletteRampJob
	drawn              bool
	waitStart          time.Time
	title              *titleAssets
}

func (g *Game) beginNativeDefeatIfConfigured() {
	if g.result != "lose" || g.camp == nil || g.camp.Node() == nil || g.camp.Node().NativeDefeatReturnTitle == nil {
		return
	}
	if err := g.startNativeDefeat(); err != nil {
		g.loadErr = "原生敗北呈現：" + err.Error()
		g.startupBlocked = true
	}
}

func (g *Game) startNativeDefeat() error {
	n := g.camp.Node()
	if n == nil || n.Type != "battle" || n.OnLose != "" || !n.NativeDefeatReturnTitle.IsRecoveredContract() || g.nativeDefeat != nil {
		return errors.New("敗北返回標題契約無效")
	}
	if !nativeMapAssetsAvailable(g.nativeMapAssets) || g.st == nil || g.m == nil {
		return errors.New("缺少原生戰場與色盤")
	}
	frames, err := fdother.LoadSeparatedPendingCode1Frames(separatedAssetPath("animations/fdother_079_pending_code1"))
	if err != nil {
		return err
	}
	for i, frame := range frames {
		if err := frame.Blit(make([]byte, indexedmap.NativeMapVGASize), 320, -1); err != nil {
			return fmt.Errorf("第%d幀：%w", i, err)
		}
	}
	// enterTitleMenu 會釋放 ANI，返回完整開場前須重新原子預檢。
	title, err := loadTitleAssets()
	if err != nil {
		return err
	}
	rollback := snapshotNativeCh20SkyKeyState(g)
	if err := g.composeNativeMapFrame(); err != nil {
		rollback()
		return err
	}
	baseline := append([]byte(nil), g.nativeMapAssets.PaletteDAC...)
	dac := append([]byte(nil), g.nativeMapDAC...)
	palette, err := fdother.VGAPaletteFromDAC(dac)
	if err != nil {
		rollback()
		return err
	}
	j := &nativeDefeatJob{
		plan: fdother.NativePendingCode1PresentationPlan(), frames: frames,
		vga: append([]byte(nil), g.nativeMapVGA...), baseline: baseline,
		dac: dac, palette: palette, title: title,
	}
	// 全部可失敗的素材解碼完成，才發布 owner 並停止 BGM。
	g.nativeDefeat = j
	g.stopBGM()
	j.pos = 1 // stop_bgm 已執行；下一拍是已證實的 WaitTick(1)。
	g.helpVisible = false
	return nil
}

func (j *nativeDefeatJob) beginRamp(start, end int) error {
	deltas, err := buildInclusivePaletteDeltas(start, end)
	if err != nil {
		return err
	}
	j.ramp = &nativePaletteRampJob{deltas: deltas, delayMs: 2,
		vga: j.vga, dac: j.dac, baseline: j.baseline}
	return j.ramp.applyCurrent()
}

func (g *Game) stepNativeDefeat(now time.Time) {
	j := g.nativeDefeat
	if j == nil || g.loadErr != "" {
		return
	}
	if j.ramp != nil {
		if !j.ramp.drawn {
			return
		}
		j.ramp.step++
		if j.ramp.step < len(j.ramp.deltas) {
			if err := j.ramp.applyCurrent(); err != nil {
				g.loadErr = err.Error()
			}
			return
		}
		j.dac, j.palette = j.ramp.dac, j.ramp.palette
		j.ramp = nil
		j.pos++
	}
	for j.pos < len(j.plan) {
		step := j.plan[j.pos]
		switch step.Kind {
		case fdother.PendingCode1WaitTick:
			if !j.drawn {
				return
			}
			if j.waitStart.IsZero() {
				j.waitStart = now
				return
			}
			if now.Sub(j.waitStart) < time.Duration(step.Count)*nativeBIOSTickPeriod {
				return
			}
		case fdother.PendingCode1PreparePalette:
			if err := j.beginRamp(0, 63); err != nil {
				g.loadErr = err.Error()
			}
			return
		case fdother.PendingCode1ClearScreen:
			clear(j.vga)
		case fdother.PendingCode1DrawFrame:
			if err := j.frames[step.Frame].Blit(j.vga, 320, -1); err != nil {
				g.loadErr = err.Error()
				return
			}
		case fdother.PendingCode1FadeIn:
			if err := j.beginRamp(64, 0); err != nil {
				g.loadErr = err.Error()
			}
			return
		case fdother.PendingCode1Release:
			g.finishNativeDefeat(j.title)
			return
		default:
			g.loadErr = "未知原生敗北呈現步驟"
			return
		}
		j.pos++
		j.drawn, j.waitStart = false, time.Time{}
	}
}

func (g *Game) drawNativeDefeat(screen *ebiten.Image) {
	j := g.nativeDefeat
	if j == nil {
		return
	}
	palette := j.palette
	if j.ramp != nil {
		palette = j.ramp.palette
	}
	img := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
	copy(img.Pix, j.vga)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(ebiten.NewImageFromImage(img), op)
	j.drawn = true
	if j.ramp != nil {
		j.ramp.drawn = true
	}
}

func (g *Game) clearDefeatedBattleTransientState() {
	g.nativeChapterResult, g.nativeResultMatchedRules = nil, nil
	g.cancelNativeCommand32Presentation()
	g.cancelNativeCommand33Presentation()
	g.resetActionOverlayLifecycle()
	g.result, g.msg = "", ""
	g.sel, g.reach, g.dialog, g.atk, g.walk = nil, nil, nil, nil, nil
	g.aiBusy, g.ring, g.moved = false, false, false
	g.aiPhaseClearBit7Pending, g.aiPhaseSelector0Pending, g.aiPhaseSelector0SweepPend, g.aiAllyPhasePending = false, false, false, false
	g.nativeAIActionPlan, g.nativeAICommandModifier, g.nativeAIItemPresentation = nil, nil, nil
	g.pendingDeathPrograms, g.pendingNativeDeathRewards, g.pendingNativeRewardMsgs = nil, nil, nil
	g.deathProgramRunning = false
	g.deathProgramKiller, g.deathProgramDead, g.deathRewarded = nil, nil, nil
	g.nativeDeathRewardUI, g.nativeDeathRewardThen, g.nativeFieldEventPending = nil, nil, nil
	g.nativeCmd0Presentation, g.nativeCmd1Presentation, g.nativeCmd2Presentation = nil, nil, nil
	g.nativeCmd3Presentation, g.nativeCmd5Presentation, g.nativeCmd6Presentation = nil, nil, nil
	g.nativeCmd7Presentation, g.nativeCmd8Presentation, g.nativeCmd9Player = nil, nil, nil
	g.nativeCmd9AIPresentation, g.nativeCmd1012, g.nativeCmd24Presentation = nil, nil, nil
	g.nativeCmd29Presentation, g.nativeCmd34Presentation, g.nativeCmd35Presentation = nil, nil, nil
	g.battleEvent, g.nativeTurnStaging, g.nativeSystemEndTurnUI = nil, nil, nil
	g.nativeClassUIJob, g.nativeFieldEvent61, g.nativeAIIdleRecovery = nil, nil, nil
	g.nativePaletteRamp, g.nativePalettePulse = nil, nil
	g.indexedTransition, g.spawnIntroTransition, g.nativeUnitPresent = nil, nil, nil
	g.actJob, g.camPan, g.focusJob, g.nativePlayerFocus = nil, nil, nil, nil
	g.fade, g.transitionReveal = nil, nil
	g.storyWalks, g.beats, g.campLines = nil, nil, nil
	g.storyAutoAdvance, g.beatIdx, g.beatDelay = 0, -1, 0
	g.nativeCh20SkyKey, g.nativeCh23Loop, g.native2189A, g.nativeCh28PostPresent, g.nativeCh22Reload = nil, nil, nil, nil, nil
	g.bannerT, g.bannerFrame = 0, nil
	g.nativeFullDACWhite, g.nativeFullDACBlack = false, false
	g.nativeMapWork, g.nativeMapVGA, g.nativeMapDAC = nil, nil, nil
	g.nativeMapClock.Reset()
	g.st, g.sc = nil, nil
	g.storyActors, g.storyRoster, g.storySpawned = nil, nil, nil
	g.storyCompositionEventBytes = nil
	g.storyRosterPath, g.storyPartyScenario = "", ""
	// 避免舊戰鬥回呼在 START／LOAD／CONTINUE 後繼續。
	g.clearShopTransientStateForLoad()
	g.clearChurchTransientStateForLoad()
}

func (g *Game) finishNativeDefeat(title *titleAssets) {
	g.captureNativeMapHUDPersistence()
	g.clearDefeatedBattleTransientState()
	g.nativeDefeat = nil
	g.titleNeedsNewCampaign = true
	g.titleAssets = title
	g.titlePhase = "cutscene"
	g.titleSel, g.titleSlotSel, g.titleFlash, g.titleTick = 0, 0, 0, 0
	g.cutIdx, g.cutFrame, g.cutTick, g.cutCur = 0, 0, 0, nil
	g.playBGM("FDMUS_018")
}

func (g *Game) restartCampaignFromTitle() error {
	if g.camp == nil || g.camp.C == nil || g.camp.C.Nodes[g.camp.C.Start] == nil {
		return errors.New("新戰役缺少起點")
	}
	g.clearDefeatedBattleTransientState()
	g.camp = campaign.NewRunner(g.camp.C)
	g.partyMembers, g.partyRoster, g.partyDeploy = nil, nil, nil
	g.partyJoinOrder, g.items = nil, nil
	g.gold, g.handlerChapter = 0, 0
	g.nativeCurrentSavePlain, g.nativeChapterSlotPlain = nil, nil
	g.nativeChapterSlotBaseline, g.nativeChapterRestore = nil, nil
	g.hasHandlerInheritedMapView = false
	g.nativeContinueOpeningConfirm = false
	g.loadErr = ""
	g.enterNode()
	if g.loadErr != "" {
		return errors.New(g.loadErr)
	}
	g.titleNeedsNewCampaign = false
	return nil
}
