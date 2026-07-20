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
	"github.com/wicanr2/fd2_re/remake/internal/ending"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// nativeEndingPreview is an explicit developer oracle for the recovered
// beginning of 0x2bce5.  It is intentionally separate from campaign endings:
// the timeline currently blocks at the first unrecovered native operation.
// FD2_ENDING_PREFIX=1 activates it with player-provided FDOTHER.DAT/ANI.DAT.
type nativeEndingPreview struct {
	player    *ending.Player
	view      *ebiten.Image
	last      time.Time
	remainder time.Duration
	chapter   int
	queued    bool
}

func newNativeEndingPreview() (*nativeEndingPreview, error) {
	timeline, err := ending.LoadTimeline(assetPath("assets/endings/native_2bce5.json"))
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
	chapter := 29 // 0x2bce5 branches only on exact native chapter 26.
	if raw := os.Getenv("FD2_ENDING_CHAPTER"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || (value != 26 && value != 29) {
			return nil, fmt.Errorf("ending: FD2_ENDING_CHAPTER must be 26 or 29")
		}
		chapter = value
	}
	return &nativeEndingPreview{player: player, view: ebiten.NewImage(ending.Width, ending.Height), chapter: chapter}, nil
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
	if g.shotPath != "" && g.frame == g.shotFrame {
		saveShot(screen, g.shotPath)
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
