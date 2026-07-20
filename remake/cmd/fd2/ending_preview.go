package main

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/afm"
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
	return &nativeEndingPreview{player: player, view: ebiten.NewImage(ending.Width, ending.Height)}, nil
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
	if g.shotPath != "" && g.frame == g.shotFrame {
		saveShot(screen, g.shotPath)
	}
}
