package main

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/afm"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/ending"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// nativeEndingPreview is an explicit developer oracle for the recovered
// beginning of 0x2bce5.  It is intentionally separate from campaign endings:
// the timeline currently blocks at the first unrecovered native operation.
// FD2_ENDING_PREFIX=1 activates it with player-provided FDOTHER.DAT/ANI.DAT.
type nativeEndingPreview struct {
	player                *ending.Player
	view                  *ebiten.Image
	last                  time.Time
	remainder             time.Duration
	chapter               int
	queued                bool
	campaignApproximate   bool
	audioCueConsumed      bool
	fdotherPath           string
	fdtxtPath             string
	montage               *ending.MontageCycle
	montageWait           time.Duration
	montageInputPending   bool
	montageStartAttempted bool
	montageStartError     string
	tail                  *ending.MontageTailAssets
	tailStartAttempted    bool
	tailStartError        string
	reviewPartyOutcomes   bool
	reviewCycles          int
}

const nativeEndingTimelinePath = "assets/endings/native_2bce5.json"

func newNativeEndingPreview() (*nativeEndingPreview, error) {
	chapter := 29 // 0x2bce5 branches only on exact native chapter 26.
	if raw := os.Getenv("FD2_ENDING_CHAPTER"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || (value != 26 && value != 29) {
			return nil, fmt.Errorf("ending: FD2_ENDING_CHAPTER must be 26 or 29")
		}
		chapter = value
	}
	return newNativeEndingPreviewForTimeline(nativeEndingTimelinePath, chapter)
}

func newNativeEndingPreviewForCampaign(prefix *campaign.NativeEndingPrefixConfig) (*nativeEndingPreview, error) {
	if !prefix.IsRecoveredPrefixContract() {
		return nil, fmt.Errorf("ending: campaign native prefix is not a recovered 0x2bce5 contract")
	}
	return newNativeEndingPreviewForTimeline(prefix.Timeline, prefix.Chapter)
}

func newNativeEndingPreviewForTimeline(timelinePath string, chapter int) (*nativeEndingPreview, error) {
	if chapter != 26 && chapter != 29 {
		return nil, fmt.Errorf("ending: native chapter must be 26 or 29")
	}
	timeline, err := ending.LoadTimeline(assetPath(timelinePath))
	if err != nil {
		return nil, err
	}
	fdotherPath := playerAssetPath("FD2_FDOTHER", []string{
		"assets/FDOTHER.DAT",
		"../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT",
		"org_game/炎龍騎士團/FLAME2/FDOTHER.DAT",
	})
	if fdotherPath == "" {
		return nil, fmt.Errorf("ending: player-provided FDOTHER.DAT is unavailable")
	}
	aniPath := playerAssetPath("FD2_ANI", aniCandidates)
	if aniPath == "" {
		return nil, fmt.Errorf("ending: player-provided ANI.DAT is unavailable")
	}
	frames, err := fdother.DecodeResource(fdotherPath, timeline.Resource.Index)
	if err != nil {
		return nil, err
	}
	clip, err := afm.DecodeResource(aniPath, 2)
	if err != nil {
		return nil, err
	}
	player, err := ending.NewPlayer(*timeline, frames, clip, ending.NewIndexedCompositor())
	if err != nil {
		return nil, err
	}
	phase0, err := ending.LoadFinalePhase(assetPath("assets/endings/native_2c405.json"))
	if err != nil {
		return nil, err
	}
	fdtxtPath := playerAssetPath("FD2_FDTXT", []string{
		filepath.Join(filepath.Dir(fdotherPath), "FDTXT.DAT"),
		"assets/FDTXT.DAT",
		"../org_game/炎龍騎士團/FLAME2/FDTXT.DAT",
		"org_game/炎龍騎士團/FLAME2/FDTXT.DAT",
	})
	if fdtxtPath == "" {
		return nil, fmt.Errorf("ending: player-provided FDTXT.DAT is unavailable")
	}
	textResource, err := fdother.ReadResource(fdtxtPath, 31)
	if err != nil {
		return nil, err
	}
	fontResource, err := fdother.ReadResource(fdotherPath, 4)
	if err != nil {
		return nil, err
	}
	if err := player.EnableRecoveredPhase0(*phase0, ending.Phase0Assets{
		TextResource: textResource,
		FontResource: fontResource,
	}); err != nil {
		return nil, err
	}
	return &nativeEndingPreview{
		player: player, view: ebiten.NewImage(ending.Width, ending.Height), chapter: chapter,
		fdotherPath: fdotherPath, fdtxtPath: fdtxtPath,
	}, nil
}

// startCampaignNativeEnding 只啟動已經過明確驗證的戰役節點所選前綴。呼叫端
// 必須以 FD2_APPROXIMATE=1 保護，正常忠實模式不會悄悄跨過未還原的終局蒙太奇邊界。
func (g *Game) startCampaignNativeEnding(prefix *campaign.NativeEndingPrefixConfig) error {
	if g == nil || !g.approximateMode {
		return fmt.Errorf("ending: campaign prefix requires explicit approximate mode")
	}
	preview, err := newNativeEndingPreviewForCampaign(prefix)
	if err != nil {
		return err
	}
	preview.campaignApproximate = true
	g.nativeEnding = preview
	return nil
}

func (p *nativeEndingPreview) atNativeMontageGate() bool {
	return p != nil && p.campaignApproximate && !p.queued && p.player != nil &&
		p.player.State == ending.PlaybackBlocked && p.player.Blocked != nil &&
		p.player.Blocked.Op == "native_finale_montage_opaque" &&
		p.player.Blocked.Source == "0x2c548"
}

func (p *nativeEndingPreview) runningCampaignMontage() bool {
	return p != nil && p.atNativeMontageGate() && !p.reviewPartyOutcomes && p.montage != nil && !p.montage.Ready()
}

// presentingCampaignTerminal marks the stable terminal image recovered from
// the final FDOTHER#59 presentation.  It remains an explicit approximate
// campaign feature until the 20-entry 0x28a6c renderer has its own proven
// adapter and the raw terminal owner reaches a normal player path.
func (p *nativeEndingPreview) presentingCampaignTerminal() bool {
	return p != nil && p.atNativeMontageGate() && p.tail != nil && !p.reviewPartyOutcomes
}

// reviewingCampaignPartyOutcomes is a remake-only, opt-in revisit loop.  It
// reuses the already admitted 0x2c548 party outcome renderer, but does not
// claim that the original's terminal self-loop accepted a modern review
// command.  Returning from it always restores the recovered terminal frame.
func (p *nativeEndingPreview) reviewingCampaignPartyOutcomes() bool {
	return p != nil && p.atNativeMontageGate() && p.tail != nil && p.reviewPartyOutcomes
}

// awaitingCampaignFallback only admits the editable epilogue after the
// recovered montage has completed, or after its source-provenance admission
// has failed.  The game starts the montage before it polls the fallback key.
func (p *nativeEndingPreview) awaitingCampaignFallback() bool {
	return p != nil && p.atNativeMontageGate() && p.montageStartAttempted &&
		(p.montage == nil || (p.montage.Ready() && p.tailStartAttempted && p.tail == nil))
}

// consumeNativeEndingAudioAtGate 只消費 after_gate 與目前已還原 0x2c548
// 邊界精確相符的唯一音訊 cue。後續 track18 只在 #59 定格取得後由另一個明確的
// 近似 adapter 消費，不能把它誤當作此 gate 的精確時序。
func (g *Game) consumeNativeEndingAudioAtGate() {
	if g == nil || g.nativeEnding == nil || !g.nativeEnding.atNativeMontageGate() ||
		g.nativeEnding.audioCueConsumed {
		return
	}
	cue, ok := g.nativeEnding.player.AudioCueAtBlockedBoundary()
	if !ok {
		return
	}
	g.nativeEnding.audioCueConsumed = true
	if cue.Track < 0 {
		g.stopBGM()
		return
	}
	g.playBGMCount(fmt.Sprintf("FDMUS_%03d", cue.Track), cue.DriverArg)
}

// finishCampaignNativeEndingFallback 只在明確近似模式且原始資產 admission
// 失敗時返回可編輯結語。成功呈現 terminal frame 的路徑會永久停留在原版
// 對應的終局畫面，不會回退成 generic ending。
func (g *Game) finishCampaignNativeEndingFallback() bool {
	if g == nil || g.nativeEnding == nil || !g.nativeEnding.awaitingCampaignFallback() {
		return false
	}
	notice := "原始結局素材不足，以下顯示可編輯結語。"
	if g.nativeEnding.montage != nil && g.nativeEnding.montage.Ready() {
		notice = "已播放可驗證的結局前段與角色蒙太奇，但終局靜態畫面素材不足；以下顯示可編輯結語。"
	}
	g.stopBGM()
	g.nativeEnding = nil
	g.endingNotice = notice
	return true
}

// nativeEndingMontageRecords materializes only the raw byte fields consumed
// by 0x2c548.  It intentionally reads the persistent JOIN roster in the same
// deployed order used by LOADCH, rather than projecting the current battle
// map, a normalized figure id, or a static chapter roster.
func nativeEndingMontageRecords(order []int, roster map[int]battle.Unit) ([][]byte, []byte, error) {
	if len(order) < 2 || len(roster) == 0 {
		return nil, nil, fmt.Errorf("ending: native montage needs at least two persistent party records")
	}
	units := make([][]byte, 0, len(order))
	groups := make([]byte, 0, len(order))
	for _, id := range order {
		unit, ok := roster[id]
		if !ok || !unit.HasNativeRecordByte6 || !unit.HasBattleFig || !unit.HasNativeRecordClass || unit.BattleFig < 0 || unit.BattleFig > 0xff {
			return nil, nil, fmt.Errorf("ending: persistent party record %d lacks raw montage provenance", id)
		}
		identity := unit.NativeRecordByte8
		if !unit.HasNativeRecordByte8 {
			// Persistent JOIN records use +8 as NativeIdentity.  This is a
			// source-carried fallback, not an inference from Fig or BattleFig.
			if !unit.HasNativeIdentity || unit.NativeIdentity < 0 || unit.NativeIdentity > 0xff {
				return nil, nil, fmt.Errorf("ending: persistent party record %d lacks raw +8 provenance", id)
			}
			identity = byte(unit.NativeIdentity)
		}
		record := make([]byte, 0x21)
		record[6] = unit.NativeRecordByte6
		record[7] = byte(unit.BattleFig)
		record[8] = identity
		record[0x20] = unit.NativeRecordClass
		units = append(units, record)
		groups = append(groups, record[7])
	}
	return units, groups, nil
}

func (p *nativeEndingPreview) montageArchivePaths() (ending.MontageArchivePaths, error) {
	if p == nil || p.fdotherPath == "" || p.fdtxtPath == "" {
		return ending.MontageArchivePaths{}, fmt.Errorf("ending: prefix archive provenance is unavailable")
	}
	base := filepath.Dir(p.fdotherPath)
	resolve := func(environment, name string) string {
		return playerAssetPath(environment, []string{
			filepath.Join(base, name),
			"assets/" + name,
			"../org_game/炎龍騎士團/FLAME2/" + name,
			"org_game/炎龍騎士團/FLAME2/" + name,
		})
	}
	paths := ending.MontageArchivePaths{
		FDOTHER: p.fdotherPath,
		FDTXT:   p.fdtxtPath,
		TAI:     resolve("FD2_TAI", "TAI.DAT"),
		FIGANI:  resolve("FD2_FIGANI", "FIGANI.DAT"),
		DATO:    resolve("FD2_DATO", "DATO.DAT"),
	}
	if paths.TAI == "" || paths.FIGANI == "" || paths.DATO == "" {
		return ending.MontageArchivePaths{}, fmt.Errorf("ending: player-provided montage archives are unavailable")
	}
	return paths, nil
}

// startCampaignNativeMontage is limited to FD2_APPROXIMATE=1.  It uses the
// current persistent roster only as a typed, source-provenance carrier; it
// does not claim that the unmodified general-player path reached 0x2c548.
func (g *Game) startCampaignNativeMontage() error {
	if g == nil || !g.approximateMode || g.nativeEnding == nil || !g.nativeEnding.atNativeMontageGate() {
		return fmt.Errorf("ending: native montage requires explicit approximate mode at its recovered gate")
	}
	p := g.nativeEnding
	if p.montageStartAttempted {
		return nil
	}
	p.montageStartAttempted = true
	order := g.loadCHPartyOrder(nil)
	units, groups, err := nativeEndingMontageRecords(order, g.partyRoster)
	if err != nil {
		p.montageStartError = err.Error()
		return err
	}
	montage, err := ending.LoadMontage(assetPath("assets/endings/native_2c548.json"))
	if err != nil {
		p.montageStartError = err.Error()
		return err
	}
	paths, err := p.montageArchivePaths()
	if err != nil {
		p.montageStartError = err.Error()
		return err
	}
	assets, err := ending.LoadMontageCycleAssets(*montage, paths, units)
	if err != nil {
		p.montageStartError = err.Error()
		return err
	}
	cycle, err := ending.NewMontageCycle(*montage, assets, units, groups, p.player.Compositor)
	if err != nil {
		p.montageStartError = err.Error()
		return err
	}
	p.montage = cycle
	return nil
}

// startCampaignNativeTail admits only the stable frame at the end of the
// recovered 0x2c194 resource sequence. The original 0x28a6c visual chain is
// proven, but its post-0x2c548 call-time input and remake adapter remain
// unresolved, so this does not label the whole tail as reproduced.
// It replaces the old generic epilogue only in explicit approximate mode and
// leaves the terminal image on screen for the player to revisit.
func (g *Game) startCampaignNativeTail() error {
	if g == nil || !g.approximateMode || g.nativeEnding == nil || !g.nativeEnding.atNativeMontageGate() ||
		g.nativeEnding.montage == nil || !g.nativeEnding.montage.Ready() {
		return fmt.Errorf("ending: terminal tail requires a completed approximate montage")
	}
	p := g.nativeEnding
	if p.tailStartAttempted {
		if p.tail != nil {
			return nil
		}
		return fmt.Errorf("ending: terminal tail admission already failed")
	}
	p.tailStartAttempted = true
	tail, err := ending.LoadMontageTail(assetPath("assets/endings/native_2c194_tail.json"))
	if err != nil {
		p.tailStartError = err.Error()
		return err
	}
	assets, err := ending.LoadMontageTailAssets(*tail, p.fdotherPath)
	if err != nil {
		p.tailStartError = err.Error()
		return err
	}
	if err := assets.PresentFinal(p.player.Compositor); err != nil {
		p.tailStartError = err.Error()
		return err
	}
	p.tail = &assets
	if p.view != nil {
		p.view.WritePixels(p.player.Compositor.RGBA().Pix)
	}
	// 原版在尾段先停止前曲，並在 20-entry loop 前啟動 FDMUS_018。
	// 這個近似端只在穩定的終局畫面取得後消費同一資源，不宣稱其間隔
	// 與尚未接入的 0x28a6c loop 完全相同。
	g.stopBGM()
	g.playBGMCount("FDMUS_018", 0)
	return nil
}

// resetCampaignPartyReviewCycle creates another pass through the already
// source-admitted party outcome cycle.  It is intentionally available only
// after the terminal image is on screen: the terminal remains the default
// ending, while review is a modern optional extension rather than a guessed
// original input path.
func (p *nativeEndingPreview) resetCampaignPartyReviewCycle() error {
	if p == nil || p.player == nil || p.player.Compositor == nil || p.tail == nil || p.montage == nil || !p.montage.Ready() {
		return fmt.Errorf("ending: party outcome review requires a completed terminal montage")
	}
	template := p.montage
	groups := make([]byte, len(template.Units))
	for i, unit := range template.Units {
		if len(unit) <= 7 {
			return fmt.Errorf("ending: party outcome review unit %d lacks raw +7 provenance", i)
		}
		groups[i] = unit[7]
	}
	// A completed cycle has faded its palette.  The native tail restores a
	// saved palette before the final image; use the same recovered baseline
	// before beginning this explicitly non-native repeat.
	if err := p.tail.PresentFinal(p.player.Compositor); err != nil {
		return err
	}
	cycle, err := ending.NewMontageCycle(template.Montage, template.Assets, template.Units, groups, p.player.Compositor)
	if err != nil {
		return fmt.Errorf("ending: party outcome review: %w", err)
	}
	p.montage = cycle
	p.montageWait = 0
	p.montageInputPending = false
	p.last = time.Time{}
	return nil
}

// startCampaignPartyOutcomeReview starts a continuous replay of every party
// member's recovered ending outcome.  Enter/Space opt in from the terminal;
// Enter/Space/Escape returns to the terminal.  Those controls are deliberately
// documented as remake behavior, not as a DOS input reconstruction.
func (g *Game) startCampaignPartyOutcomeReview() error {
	if g == nil || g.nativeEnding == nil || !g.nativeEnding.presentingCampaignTerminal() {
		return fmt.Errorf("ending: party outcome review requires the terminal image")
	}
	p := g.nativeEnding
	if err := p.resetCampaignPartyReviewCycle(); err != nil {
		return err
	}
	p.reviewPartyOutcomes = true
	p.reviewCycles++
	return nil
}

// returnCampaignTerminalFromReview leaves the optional remake review loop and
// restores the held source-derived terminal image.  It never advances the
// campaign or emulates the original DOS keyboard self-loop.
func (g *Game) returnCampaignTerminalFromReview() error {
	if g == nil || g.nativeEnding == nil || !g.nativeEnding.reviewingCampaignPartyOutcomes() {
		return fmt.Errorf("ending: party outcome review is not active")
	}
	p := g.nativeEnding
	if err := p.tail.PresentFinal(p.player.Compositor); err != nil {
		return err
	}
	p.reviewPartyOutcomes = false
	p.montageWait = 0
	p.montageInputPending = false
	p.last = time.Time{}
	if p.view != nil {
		p.view.WritePixels(p.player.Compositor.RGBA().Pix)
	}
	return nil
}

func nativeEndingDialogLines(blocks []ending.DialogueBlock) ([]battle.DialogLine, error) {
	var out []battle.DialogLine
	for _, block := range blocks {
		lines := loadStoryScriptAt(handlerStoryPath(block.Script), "", &block.SceneIndex)
		if block.Line < 0 || block.Count <= 0 || block.Line+block.Count > len(lines) {
			return nil, fmt.Errorf("ending: dialogue %s scene=%d line=%d count=%d is unavailable", block.Script, block.SceneIndex, block.Line, block.Count)
		}
		for _, line := range lines[block.Line : block.Line+block.Count] {
			out = append(out, battle.DialogLine{Speaker: block.PortraitID, Text: line.Text})
		}
	}
	return out, nil
}

func (g *Game) queueNativeEndingDialogue() error {
	p := g.nativeEnding
	if p == nil || p.queued {
		return nil
	}
	blocks, ok := p.player.BlockedDialogue(p.chapter)
	if !ok {
		return nil
	}
	lines, err := nativeEndingDialogLines(blocks)
	if err != nil {
		return err
	}
	for i := len(lines) - 1; i >= 0; i-- {
		g.dialog = append(g.dialog, lines[i])
	}
	g.dlgPage, g.dlgScrollT, g.dlgScrollFrom = 0, 0, 0
	p.queued = true
	return nil
}

// resumeNativeEndingDialogue releases exactly the text gate whose queued
// lines have all been acknowledged. Resetting queued is required because
// 0x2bce5 reaches a second, independently recovered 0x2c39b call later in the
// same timeline.
func (g *Game) resumeNativeEndingDialogue() bool {
	p := g.nativeEnding
	if p == nil || !p.queued || !p.player.ResumeBlockedDialogue() {
		return false
	}
	p.queued = false
	return true
}

func playerAssetPath(environment string, candidates []string) string {
	if p := os.Getenv(environment); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
		return ""
	}
	for _, candidate := range candidates {
		p := candidate
		if filepath.IsAbs(candidate) || !strings.HasPrefix(candidate, "assets/") {
			if _, err := os.Stat(p); err != nil && exeDir() != "" {
				p = filepath.Join(exeDir(), candidate)
			}
		} else {
			p = assetPath(candidate)
		}
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

const approximateNativeMontageTick = 55 * time.Millisecond

func (p *nativeEndingPreview) advanceMontage(elapsed time.Duration, nativeRNG *uint16) error {
	if p == nil || p.montage == nil || nativeRNG == nil {
		return fmt.Errorf("ending: montage runtime is incomplete")
	}
	remaining := elapsed
	for steps := 0; steps < 1024; steps++ {
		if p.montage.Ready() {
			return nil
		}
		if p.montageWait > 0 {
			if remaining < p.montageWait {
				p.montageWait -= remaining
				return nil
			}
			remaining -= p.montageWait
			p.montageWait = 0
		}
		if p.montageInputPending && p.montage.ObserveInputChange() {
			p.montageInputPending = false
		}
		randomByte := byte(0)
		if p.montage.NeedsRandomByte() {
			*nativeRNG = fdother.NativeRNGStep(*nativeRNG)
			randomByte = byte(*nativeRNG)
		}
		if err := p.montage.Step(randomByte); err != nil {
			return err
		}
		if p.montage.Ready() {
			return nil
		}
		p.montageWait = time.Duration(p.montage.DelayTicks)*approximateNativeMontageTick + time.Duration(p.montage.DelayMS)*time.Millisecond
		if remaining == 0 && p.montageWait > 0 {
			return nil
		}
	}
	return fmt.Errorf("ending: montage advance exceeded bounded step budget")
}

func (p *nativeEndingPreview) advance(now time.Time, nativeRNG *uint16) error {
	elapsed := 0
	if !p.last.IsZero() {
		p.remainder += now.Sub(p.last)
		elapsed = int(p.remainder / time.Millisecond)
		p.remainder -= time.Duration(elapsed) * time.Millisecond
	}
	p.last = now
	if p.reviewingCampaignPartyOutcomes() {
		if p.montage == nil {
			return fmt.Errorf("ending: party outcome review has no cycle")
		}
		if !p.montage.Ready() {
			if err := p.advanceMontage(time.Duration(elapsed)*time.Millisecond, nativeRNG); err != nil {
				return err
			}
		}
		if p.montage.Ready() {
			if err := p.resetCampaignPartyReviewCycle(); err != nil {
				return err
			}
			p.reviewCycles++
		}
	} else if p.tail != nil {
		// The native caller self-loops after 0x2bce5 returns.  Keep the
		// recovered terminal frame stable instead of advancing a generic
		// campaign node or requiring a DOS keyboard emulation layer.
	} else if p.montage != nil && !p.montage.Ready() {
		if err := p.advanceMontage(time.Duration(elapsed)*time.Millisecond, nativeRNG); err != nil {
			return err
		}
	} else {
		if _, err := p.player.Advance(elapsed); err != nil {
			return err
		}
	}
	p.view.WritePixels(p.player.Compositor.RGBA().Pix)
	return nil
}

func (g *Game) drawNativeEndingPreview(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 0xff})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(g.nativeEnding.view, op)
	g.drawNativeEndingDialogue(screen)
	if g.nativeEnding.runningCampaignMontage() && g.font != nil {
		panel := ebiten.NewImage(logicalW-32, 42)
		panel.Fill(color.RGBA{0x10, 0x1c, 0x40, 0xe8})
		pop := &ebiten.DrawImageOptions{}
		pop.GeoM.Translate(16, logicalH-56)
		screen.DrawImage(panel, pop)
		g.font.Draw(screen, "角色蒙太奇（近似戰役資料）：任意按鍵會在本輪結束後略過中間角色。", 23, logicalH-44, 0.78,
			color.RGBA{0xff, 0xe0, 0x90, 0xff})
	} else if g.nativeEnding.awaitingCampaignFallback() && g.font != nil {
		panel := ebiten.NewImage(logicalW-32, 42)
		panel.Fill(color.RGBA{0x10, 0x1c, 0x40, 0xe8})
		pop := &ebiten.DrawImageOptions{}
		pop.GeoM.Translate(16, logicalH-56)
		screen.DrawImage(panel, pop)
		message := "已播放可驗證的結局前段；按 Enter 顯示可編輯結語。"
		if g.nativeEnding.montage != nil && g.nativeEnding.montage.Ready() {
			message = "角色蒙太奇後無法載入終局畫面；按 Enter 顯示可編輯結語。"
		} else if g.nativeEnding.montageStartError != "" {
			message = "角色蒙太奇需要完整的原始隊伍記錄與素材；按 Enter 顯示可編輯結語。"
		}
		g.font.Draw(screen, message, 30, logicalH-44, 0.82,
			color.RGBA{0xff, 0xe0, 0x90, 0xff})
	}
	if g.shotPath != "" && !g.shotTaken && g.frame >= g.shotFrame {
		// 原生結局前綴與其他場景共用同一證據掛鉤：輸出狀態旁車、回報載入
		// 錯誤，並讓 Update 結束有界的截圖流程。
		g.captureShot(screen)
	}
}

// drawNativeEndingDialogue retains the ordinary DATO portrait/font/dialogue
// semantics for the recovered 0x2c39b blocks while the indexed ending image
// remains visible underneath.  Later native operations stay blocked.
func (g *Game) drawNativeEndingDialogue(screen *ebiten.Image) {
	if g.font == nil || len(g.dialog) == 0 {
		return
	}
	dl := g.dialog[len(g.dialog)-1]
	upper := dl.Speaker >= 32
	by := 198.0
	if upper {
		by = 4
	}
	box := ebiten.NewImage(620, 198)
	box.Fill(color.RGBA{0x2c, 0x44, 0x84, 0xf2})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(10, by)
	screen.DrawImage(box, op)
	const scale = 2.1
	hx, tx, ty := 16.0, 216.0, by+24
	hy := by + (198-80*scale)/2
	if upper {
		hx, tx, ty = float64(logicalW)-16-80*scale, 32, by+46
	}
	if fr := g.portraits[dl.Speaker]; len(fr) > 0 {
		po := &ebiten.DrawImageOptions{}
		if upper {
			po.GeoM.Scale(scale, scale)
			po.GeoM.Translate(hx, hy)
		} else {
			po.GeoM.Scale(-scale, scale)
			po.GeoM.Translate(hx+80*scale, hy)
		}
		screen.DrawImage(fr[0], po)
	} else {
		tx = 32
	}
	lines := dlgWrap(dl)
	start := g.dlgPage * 3
	for i := 0; i < 3 && start+i < len(lines); i++ {
		g.font.Draw(screen, lines[start+i], tx, ty+float64(i)*38, 1.7, color.RGBA{0xf0, 0xf4, 0xff, 0xff})
	}
}
