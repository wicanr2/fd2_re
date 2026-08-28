package main

import (
	"fmt"
	"image/color"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/afm"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/ending"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// nativeEndingPreview preserves the recovered 0x2bce5 prefix and also owns
// the source-bound campaign ending path. It still grades call-time record and
// global continuity, sound/input ownership and player-path E2 separately;
// FD2_ENDING_PREFIX=1 remains a developer-only direct preview entry.
type nativeEndingPreview struct {
	player                *ending.Player
	view                  *ebiten.Image
	last                  time.Time
	remainder             time.Duration
	chapter               int
	queued                bool
	campaignSourceBound   bool
	audioCueConsumed      bool
	fdotherPath           string
	fdtxtPath             string
	montage               *ending.MontageCycle
	montageWait           time.Duration
	montageInputPending   bool
	montageStartAttempted bool
	montageStartError     string
	tail                  *ending.MontageTailAssets
	tailPlayer            *ending.MontageTailPlayer
	tailWait              time.Duration
	tailStartAttempted    bool
	tailStartError        string
	reviewPartyOutcomes   bool
	reviewCycles          int
	dialogue              *ending.NativeDialoguePlayback
	dialogueView          *ebiten.Image
	dialogueMouth         dato.MouthState
	dialogueMouthReady    bool
}

const nativeEndingTimelinePath = "assets/endings/native_2bce5.json"

func newNativeEndingPreview() (*nativeEndingPreview, error) {
	// 預設使用內部第29章預覽；保留已還原的章節分支，且只接受明確驗證過的
	// native chapter 26或29。
	chapter := 29
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

// startCampaignNativeEnding 只啟動已經過明確驗證的戰役節點所選前綴。後續
// renderer仍以來源約束 E1 標示，不把近似時鐘或未證實音效冒稱為原版 E2。
func (g *Game) startCampaignNativeEnding(prefix *campaign.NativeEndingPrefixConfig) error {
	if g == nil {
		return fmt.Errorf("ending: campaign prefix requires a game owner")
	}
	preview, err := newNativeEndingPreviewForCampaign(prefix)
	if err != nil {
		return err
	}
	preview.campaignSourceBound = true
	g.nativeEnding = preview
	return nil
}

func (p *nativeEndingPreview) atNativeMontageGate() bool {
	return p != nil && p.campaignSourceBound && !p.queued && p.player != nil &&
		p.player.State == ending.PlaybackBlocked && p.player.Blocked != nil &&
		p.player.Blocked.Op == "native_finale_montage_opaque" &&
		p.player.Blocked.Source == "0x2c548"
}

func (p *nativeEndingPreview) runningCampaignMontage() bool {
	return p != nil && p.atNativeMontageGate() && !p.reviewPartyOutcomes && p.montage != nil && !p.montage.Ready()
}

func (p *nativeEndingPreview) runningCampaignTail() bool {
	return p != nil && p.atNativeMontageGate() && !p.reviewPartyOutcomes && p.tail != nil &&
		p.tailPlayer != nil && !p.tailPlayer.Ready()
}

// presentingCampaignTerminal marks the stable terminal image recovered from
// the final FDOTHER#59 presentation.  It is a source-bound E1 campaign feature;
// call-time record continuity and DOS visual/audio E2 remain separately graded.
func (p *nativeEndingPreview) presentingCampaignTerminal() bool {
	return p != nil && p.atNativeMontageGate() && p.tail != nil && p.tailPlayer != nil &&
		p.tailPlayer.Ready() && !p.reviewPartyOutcomes
}

// reviewingCampaignPartyOutcomes is a remake-only, opt-in revisit loop.  It
// reuses the already admitted 0x2c548 party outcome renderer, but does not
// claim that the original's terminal self-loop accepted a modern review
// command.  Returning from it always restores the recovered terminal frame.
func (p *nativeEndingPreview) reviewingCampaignPartyOutcomes() bool {
	return p != nil && p.atNativeMontageGate() && p.tail != nil && p.tailPlayer != nil &&
		p.tailPlayer.Ready() && p.reviewPartyOutcomes
}

// applyNativeEndingInput 是正式 Game.Update 與回歸共用的終局輸入 owner。
// anyKey 只保存0x2c950已證實的raw-change條件；它不猜特定DOS scan code。
func (g *Game) applyNativeEndingInput(confirm, escape, anyKey bool) error {
	if g == nil || g.nativeEnding == nil {
		return nil
	}
	if g.nativeEnding.reviewingCampaignPartyOutcomes() && (confirm || escape) {
		return g.returnCampaignTerminalFromReview()
	}
	if g.nativeEnding.presentingCampaignTerminal() && confirm {
		return g.startCampaignPartyOutcomeReview()
	}
	if !g.nativeEnding.reviewingCampaignPartyOutcomes() && g.nativeEnding.montage != nil &&
		!g.nativeEnding.montage.Ready() && anyKey {
		g.nativeEnding.montageInputPending = true
	}
	return nil
}

// awaitingCampaignFallback only admits the editable epilogue after the
// recovered montage has completed, or after its source-provenance admission
// has failed.  The game starts the montage before it polls the fallback key.
func (p *nativeEndingPreview) awaitingCampaignFallback() bool {
	return p != nil && p.atNativeMontageGate() && p.montageStartAttempted &&
		(p.montage == nil || (p.montage.Ready() && p.tailStartAttempted && (p.tail == nil || p.tailPlayer == nil)))
}

// consumeNativeEndingAudioAtGate 只消費 after_gate 與目前已還原 0x2c548
// 邊界精確相符的唯一音訊 cue。後續 tail_stop／track18 由來源約束 E1 尾段
// adapter 在20組尾段前依已證實順序消費；只保留 DOS wall-clock 為未知。
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
	g.emitNativeEndingAudioCue(cue)
}

func (g *Game) emitNativeEndingAudioCue(cue ending.AudioCue) {
	if cue.Track < 0 {
		g.stopBGM()
		return
	}
	g.playBGMCount(fmt.Sprintf("FDMUS_%03d", cue.Track), cue.DriverArg)
}

// finishCampaignNativeEndingFallback 只在來源約束的原始資產預檢失敗時
// 返回可編輯結語。成功呈現 terminal frame 的路徑會永久停留在原版
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
		FDOTHER:      p.fdotherPath,
		FDTXT:        p.fdtxtPath,
		TAI:          resolve("FD2_TAI", "TAI.DAT"),
		FIGANI:       resolve("FD2_FIGANI", "FIGANI.DAT"),
		PortraitRoot: separatedAssetPath("portraits"),
	}
	if paths.TAI == "" || paths.FIGANI == "" {
		return ending.MontageArchivePaths{}, fmt.Errorf("ending: player-provided montage archives are unavailable")
	}
	return paths, nil
}

// startCampaignNativeMontage 只在來源約束 E1 的終局邊界啟動。它以當前
// persistent roster 作為具型別、可回查原始記錄的載體；這不證明未修改
// 原版的一般玩家路徑已到達 0x2c548。
func (g *Game) startCampaignNativeMontage() error {
	if g == nil || g.nativeEnding == nil || !g.nativeEnding.atNativeMontageGate() {
		return fmt.Errorf("ending: native montage requires the source-bound campaign gate")
	}
	p := g.nativeEnding
	if p.montageStartAttempted {
		return nil
	}
	p.montageStartAttempted = true
	// 最終戰部署只是暫時選擇；終局回顧依永久 JOIN 時序涵蓋所有入隊角色。
	// 部署成員已由 EndingPartySnapshotOnWin 帶入戰果，後備成員則保留冷讀快照。
	order := append([]int(nil), g.partyJoinOrder...)
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

// startCampaignNativeTail admits a source-bound E1 visual bridge for
// the proven 20-entry 0x2c194 resource schedule. It preserves original archive
// selectors and delays, but does not claim unresolved sound ownership or
// call-time record bit equality as DOS E2.
func (g *Game) startCampaignNativeTail() error {
	if g == nil || g.nativeEnding == nil || !g.nativeEnding.atNativeMontageGate() ||
		g.nativeEnding.montage == nil || !g.nativeEnding.montage.Ready() {
		return fmt.Errorf("ending: terminal tail requires a completed source-bound montage")
	}
	p := g.nativeEnding
	if p.tailStartAttempted {
		if p.tail != nil && p.tailPlayer != nil {
			return nil
		}
		return fmt.Errorf("ending: terminal tail admission already failed")
	}
	p.tailStartAttempted = true
	stopCue, ok := p.player.AudioCueForRuntimeStage("tail_stop")
	if !ok || stopCue.Source != "0x2c1ac" || stopCue.Track != -1 || stopCue.DriverArg != 1 {
		p.tailStartError = "ending: verified tail stop cue is unavailable"
		return fmt.Errorf("%s", p.tailStartError)
	}
	startCue, ok := p.player.AudioCueForRuntimeStage("tail_start")
	if !ok || startCue.Source != "0x2c1f5" || startCue.Track != 18 || startCue.DriverArg != 0 {
		p.tailStartError = "ending: verified tail start cue is unavailable"
		return fmt.Errorf("%s", p.tailStartError)
	}
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
	paths, err := p.montageArchivePaths()
	if err != nil {
		p.tailStartError = err.Error()
		return err
	}
	bgPath := playerAssetPath("FD2_BG", []string{
		filepath.Join(filepath.Dir(p.fdotherPath), "BG.DAT"),
		"assets/BG.DAT",
		"../org_game/炎龍騎士團/FLAME2/BG.DAT",
		"org_game/炎龍騎士團/FLAME2/BG.DAT",
	})
	sets, err := ending.LoadMontageTailVisualSets(*tail, ending.MontageTailVisualPaths{
		TAI: paths.TAI, BG: bgPath, FIGANI: paths.FIGANI,
	})
	if err != nil {
		p.tailStartError = err.Error()
		return err
	}
	player, err := ending.NewMontageTailPlayer(*tail, assets, sets, p.player.Compositor)
	if err != nil {
		p.tailStartError = err.Error()
		return err
	}
	p.tail = &assets
	p.tailPlayer = player
	p.tailWait = 0
	p.last = time.Time{}
	// 資產完整 admission 後才依資料發布兩筆 cue，避免深層載入失敗留下半套音訊
	// 狀態；順序與 raw 參數已證實，兩筆之間的 DOS wall-clock 仍不宣稱精確。
	g.emitNativeEndingAudioCue(stopCue)
	g.emitNativeEndingAudioCue(startCue)
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
	p.tailWait = 0
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
	p.tailWait = 0
	p.last = time.Time{}
	if p.view != nil {
		p.view.WritePixels(p.player.Compositor.RGBA().Pix)
	}
	return nil
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
	playback, err := g.prepareNativeEndingDialogue(blocks)
	if err != nil {
		return err
	}
	p.dialogue = playback
	p.dialogueView = ebiten.NewImage(ending.Width, ending.Height)
	p.dialogueMouth = dato.MouthState{}
	p.dialogueMouthReady = false
	p.queued = true
	return g.syncNativeEndingDialogueView()
}

func (g *Game) prepareNativeEndingDialogue(blocks []ending.DialogueBlock) (*ending.NativeDialoguePlayback, error) {
	if g == nil || g.nativeEnding == nil || g.nativeEnding.player == nil || g.nativeEnding.player.Compositor == nil || len(g.nativeEnding.player.Compositor.VGA) != ending.Bytes {
		return nil, fmt.Errorf("ending: indexed dialogue source is unavailable")
	}
	font, glyphs, err := loadNativeBattleNameAssets()
	if err != nil {
		return nil, err
	}
	resource5, err := fdother.ReadResource(g.nativeEnding.fdotherPath, 5)
	if err != nil {
		return nil, err
	}
	dialogueCells := make([]fdother.RawCell, 20)
	for index := range dialogueCells {
		dialogueCells[index], err = fdother.ParseLMI1RawEntry(resource5, index)
		if err != nil {
			return nil, err
		}
	}
	background := append([]byte(nil), g.nativeEnding.player.Compositor.VGA...)
	prepared := make([]ending.NativeDialogueBlockFrames, 0, len(blocks))
	for blockIndex, block := range blocks {
		if block.PortraitID >= 0x80 && block.PortraitID <= 0x84 {
			return nil, fmt.Errorf("ending: block %d portrait %d requires an unimplemented special anchor", blockIndex, block.PortraitID)
		}
		initial, err := loadNativeSeparatedPortrait(block.PortraitID)
		if err != nil || len(initial) < 4 {
			return nil, fmt.Errorf("ending: block %d initial portrait %d is unavailable", blockIndex, block.PortraitID)
		}
		initialBase, err := ending.ComposeNativeDialogueBase(background, dialogueCells, initial[0])
		if err != nil {
			return nil, err
		}
		opening, err := ending.ComposeNativeDialogueOpeningFrames(background, initialBase)
		if err != nil {
			return nil, err
		}
		closing, err := ending.ComposeNativeDialogueClosingFrames(background, initialBase)
		if err != nil {
			return nil, err
		}
		utterances := make([]ending.NativeDialogueUtteranceFrames, len(block.NativeUtterances))
		for utteranceIndex, native := range block.NativeUtterances {
			if native.Operand >= 0x80 && native.Operand <= 0x84 {
				return nil, fmt.Errorf("ending: block %d utterance %d portrait %d requires an unimplemented special anchor", blockIndex, utteranceIndex, native.Operand)
			}
			portraits, err := loadNativeSeparatedPortrait(native.Operand)
			if err != nil || len(portraits) < 4 {
				return nil, fmt.Errorf("ending: block %d utterance %d portrait %d is unavailable", blockIndex, utteranceIndex, native.Operand)
			}
			base, err := ending.ComposeNativeDialogueBase(background, dialogueCells, portraits[0])
			if err != nil {
				return nil, err
			}
			utterances[utteranceIndex].Pages = make([][][]byte, len(native.Pages))
			utterances[utteranceIndex].MouthOpen = make([][]byte, len(native.Pages))
			for page := range native.Pages {
				frames, err := ending.ComposeNativeDialogueProgressiveFrames(base, font, glyphs, native.Pages, page)
				if err != nil {
					return nil, err
				}
				utterances[utteranceIndex].Pages[page] = frames
				mouth, err := ending.ComposeNativeDialogueMouthFrame(frames[len(frames)-1], portraits[3])
				if err != nil {
					return nil, err
				}
				utterances[utteranceIndex].MouthOpen[page] = mouth
			}
		}
		prepared = append(prepared, ending.NativeDialogueBlockFrames{Opening: opening, Utterances: utterances, Closing: closing})
	}
	return ending.NewNativeDialoguePlayback(prepared)
}

func (g *Game) syncNativeEndingDialogueView() error {
	p := g.nativeEnding
	if p == nil || p.dialogue == nil || p.dialogueView == nil {
		return nil
	}
	frame := p.dialogue.CurrentFrame(p.dialogueMouth.Open)
	rgba, err := p.player.Compositor.RGBAFrame(frame)
	if err != nil {
		return err
	}
	p.dialogueView.WritePixels(rgba.Pix)
	return nil
}

func (g *Game) stepNativeEndingDialogue(confirm bool) error {
	p := g.nativeEnding
	if p == nil || p.dialogue == nil {
		return nil
	}
	wasWaiting := p.dialogue.Waiting()
	if confirm {
		p.dialogue.Confirm()
	} else {
		p.dialogue.Step()
	}
	if p.dialogue.Done() {
		p.dialogue, p.dialogueView = nil, nil
		p.dialogueMouth, p.dialogueMouthReady = dato.MouthState{}, false
		g.resumeNativeEndingDialogue()
		return nil
	}
	if !p.dialogue.Waiting() {
		p.dialogueMouth, p.dialogueMouthReady = dato.MouthState{}, false
	} else if !wasWaiting || !p.dialogueMouthReady {
		p.dialogueMouth = dato.MouthState{Countdown: rand.Intn(30) + 2}
		p.dialogueMouthReady = true
	} else if g.frame%2 == 0 {
		randomMod30 := 0
		if p.dialogueMouth.Open {
			randomMod30 = rand.Intn(30)
		}
		next, err := p.dialogueMouth.Tick(randomMod30)
		if err != nil {
			return err
		}
		p.dialogueMouth = next
	}
	return g.syncNativeEndingDialogueView()
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
	p.dialogue, p.dialogueView = nil, nil
	p.dialogueMouth, p.dialogueMouthReady = dato.MouthState{}, false
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

func (p *nativeEndingPreview) advanceTail(elapsed time.Duration) error {
	if p == nil || p.tailPlayer == nil {
		return fmt.Errorf("ending: montage tail runtime is incomplete")
	}
	remaining := elapsed
	for steps := 0; steps < 4096; steps++ {
		if p.tailPlayer.Ready() {
			return nil
		}
		if p.tailWait > 0 {
			if remaining < p.tailWait {
				p.tailWait -= remaining
				return nil
			}
			remaining -= p.tailWait
			p.tailWait = 0
		}
		if err := p.tailPlayer.Step(); err != nil {
			return err
		}
		p.tailWait = time.Duration(p.tailPlayer.DelayTicks) * approximateNativeMontageTick
		if remaining == 0 && p.tailWait > 0 {
			return nil
		}
	}
	return fmt.Errorf("ending: montage tail advance exceeded bounded step budget")
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
	} else if p.runningCampaignTail() {
		if err := p.advanceTail(time.Duration(elapsed) * time.Millisecond); err != nil {
			return err
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

// nativeEndingEvidenceOverlay 只提供玩家主動開啟的除錯 HUD。來源等級與
// 播放說明不是原版終局的一部分，正式成功路徑不得污染320×200畫布。
func (g *Game) nativeEndingEvidenceOverlay() (string, float64, bool) {
	if g == nil || !g.debug || g.nativeEnding == nil {
		return "", 0, false
	}
	if g.nativeEnding.runningCampaignMontage() {
		return "角色蒙太奇（來源約束 E1）：任意按鍵會在本輪結束後略過中間角色。", 23, true
	}
	if g.nativeEnding.runningCampaignTail() {
		return "終局蒙太奇（原版資源、來源約束 E1）：完成後將停留在 THE END。", 25, true
	}
	return "", 0, false
}

func (g *Game) drawNativeEndingPreview(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 0xff})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	view := g.nativeEnding.view
	if g.nativeEnding.dialogueView != nil {
		view = g.nativeEnding.dialogueView
	}
	screen.DrawImage(view, op)
	if message, x, ok := g.nativeEndingEvidenceOverlay(); ok && g.font != nil {
		panel := ebiten.NewImage(logicalW-32, 42)
		panel.Fill(color.RGBA{0x10, 0x1c, 0x40, 0xe8})
		pop := &ebiten.DrawImageOptions{}
		pop.GeoM.Translate(16, logicalH-56)
		screen.DrawImage(panel, pop)
		g.font.Draw(screen, message, x, logicalH-44, 0.78,
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
