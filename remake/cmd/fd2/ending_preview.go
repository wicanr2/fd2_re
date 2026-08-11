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
	player              *ending.Player
	view                *ebiten.Image
	last                time.Time
	remainder           time.Duration
	chapter             int
	queued              bool
	campaignApproximate bool
	audioCueConsumed    bool
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
	return &nativeEndingPreview{player: player, view: ebiten.NewImage(ending.Width, ending.Height), chapter: chapter}, nil
}

// startCampaignNativeEnding 只啟動已經過明確驗證的戰役節點所選前綴。呼叫端
// 必須以 FD2_APPROXIMATE=1 保護，正常忠實模式不會悄悄跨過未還原的終局蒙太奇邊界。
func (g *Game) startCampaignNativeEnding(prefix *campaign.NativeEndingPrefixConfig) error {
	if g == nil {
		return fmt.Errorf("ending: nil game")
	}
	preview, err := newNativeEndingPreviewForCampaign(prefix)
	if err != nil {
		return err
	}
	preview.campaignApproximate = true
	g.nativeEnding = preview
	return nil
}

func (p *nativeEndingPreview) awaitingCampaignFallback() bool {
	return p != nil && p.campaignApproximate && !p.queued && p.player != nil &&
		p.player.State == ending.PlaybackBlocked && p.player.Blocked != nil &&
		p.player.Blocked.Op == "native_finale_montage_opaque" &&
		p.player.Blocked.Source == "0x2c548"
}

// consumeNativeEndingAudioAtGate 只消費 after_gate 與目前已還原 0x2c548
// 邊界精確相符的唯一音訊 cue。後續停曲／track18 沒有已還原的 owner，刻意不碰。
func (g *Game) consumeNativeEndingAudioAtGate() {
	if g == nil || g.nativeEnding == nil || !g.nativeEnding.awaitingCampaignFallback() ||
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

// finishCampaignNativeEndingFallback 只在明確近似模式的未還原蒙太奇閘門返回。
// 它絕不恢復 native Player，也不宣稱已執行原版終局 renderer。
func (g *Game) finishCampaignNativeEndingFallback() bool {
	if g == nil || g.nativeEnding == nil || !g.nativeEnding.awaitingCampaignFallback() {
		return false
	}
	g.stopBGM()
	g.nativeEnding = nil
	g.endingNotice = "原版結局尾段尚未還原；以下顯示可編輯結語。"
	return true
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

func (p *nativeEndingPreview) advance(now time.Time) error {
	elapsed := 0
	if !p.last.IsZero() {
		p.remainder += now.Sub(p.last)
		elapsed = int(p.remainder / time.Millisecond)
		p.remainder -= time.Duration(elapsed) * time.Millisecond
	}
	p.last = now
	if _, err := p.player.Advance(elapsed); err != nil {
		return err
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
	if g.nativeEnding.awaitingCampaignFallback() && g.font != nil {
		panel := ebiten.NewImage(logicalW-32, 42)
		panel.Fill(color.RGBA{0x10, 0x1c, 0x40, 0xe8})
		pop := &ebiten.DrawImageOptions{}
		pop.GeoM.Translate(16, logicalH-56)
		screen.DrawImage(panel, pop)
		g.font.Draw(screen, "已播放可驗證的結局前段；按 Enter 顯示可編輯結語。", 30, logicalH-44, 0.9,
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
