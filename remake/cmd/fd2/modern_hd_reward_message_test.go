package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func rewardHDTestGame(t *testing.T) (*Game, *battle.Unit) {
	t.Helper()
	g, actor, _ := ordinaryCommand0HDTestGame(t)
	var err error
	g.nativePreparationUI, err = loadNativePreparationUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.nativeClassUI, err = loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.font = loadFont()
	if g.font == nil {
		t.Fatal("Noto font required")
	}
	_, g.nativeBattleGlyphs, err = loadNativeBattleNameAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.sel, g.ring, g.nativeCommand0Targeting = nil, false, false
	// resetBattle's opening dialogue is unrelated to an action-completion
	// reward. This controlled fixture starts after that intro has finished.
	g.dialog, g.dlgShown = nil, dlgNone
	g.bannerT, g.banner = 0, ""
	g.gold = 123
	return g, actor
}

func TestModernHDRewardPanelCoverage(t *testing.T) {
	source0, source255 := make([]byte, 64000), bytes.Repeat([]byte{255}, 64000)
	final := bytes.Repeat([]byte{77}, 64000)
	for _, closing := range []bool{false, true} {
		var a, b [][]byte
		var err error
		if closing {
			a, err = campaign.NativeClassListClosingFrames(source0, final)
			b, _ = campaign.NativeClassListClosingFrames(source255, final)
		} else {
			a, err = campaign.NativeClassListOpeningFrames(source0, final)
			b, _ = campaign.NativeClassListOpeningFrames(source255, final)
		}
		if err != nil {
			t.Fatal(err)
		}
		job := &nativeClassUIJob{frames: a}
		message := &nativeDeathRewardMessageState{opening: job}
		if closing {
			message.opening, message.closing = nil, job
		}
		for i := range a {
			job.frame = i
			shift, r, ok := modernHDRewardRegion(message, job)
			if !ok {
				t.Fatal("native panel rejected")
			}
			wantShift := (5 - i) * 13
			if closing {
				wantShift = (i + 1) * 13
			}
			if shift != wantShift {
				t.Fatal("panel displacement changed")
			}
			for p := range a[i] {
				if written := a[i][p] == b[i][p]; written != image.Pt(p%320, p/320).In(r) {
					t.Fatalf("coverage closing%t frame%d pixel%d", closing, i, p)
				}
			}
		}
	}
	m := &nativeDeathRewardMessageState{}
	if _, _, ok := modernHDRewardRegion(m, nil); ok {
		t.Fatal("unacknowledged empty owner admitted")
	}
	if _, _, ok := modernHDRewardRegion(m, &nativeClassUIJob{frames: make([][]byte, 6)}); ok {
		t.Fatal("unrelated job admitted")
	}
	job := &nativeClassUIJob{frame: 5, frames: make([][]byte, 5), restore: source0}
	m.closing = job
	if _, r, ok := modernHDRewardRegion(m, job); !ok || !r.Empty() {
		t.Fatal("restore retained panel")
	}
}

func TestModernHDRewardTextAndNativeAssets(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	g, actor := rewardHDTestGame(t)
	screen := ebiten.NewImage(640, 400)
	g.modernStoryPortraits = &modernStoryPortraitSet{hasHD: true}
	for _, spec := range [][2]int{{0, 0}, {0, 192}, {1, 0}, {1, 123456789}} {
		calls := 0
		if err := g.beginNativeDeathRewardMessage(pendingNativeDeathRewardMessage{killer: actor, kind: spec[0], value: spec[1]}, actor, func() { calls++ }); err != nil {
			t.Fatal(err)
		}
		m := g.nativeSystemEndTurnUI.rewardMessage
		if m.hdInputErr != nil {
			t.Fatal(m.hdInputErr)
		}
		before := append([]byte(nil), m.final...)
		state := *actor
		assets, font := g.nativePreparationUI, g.nativePreparationUI.status.Font
		originalRuns := 0
		for frame := 0; frame < 13; frame++ {
			done := g.beginModernHDText(screen)
			if !g.drawNativeSystemEndTurn(screen) {
				t.Fatal("message not drawn", frame)
			}
			if frame == 6 {
				originalRuns = len(g.modernHDText.runs)
				if originalRuns < 2 {
					t.Fatal("no high resolution text", spec)
				}
				if spec[0] == 1 {
					for _, digit := range strconv.Itoa(spec[1]) {
						found := false
						for _, run := range g.modernHDText.runs {
							found = found || run.text == string(digit)
						}
						if !found {
							t.Fatalf("dynamic gold digit %c remained native", digit)
						}
					}
				}
			}
			for _, r := range g.modernHDText.runs {
				if r.clip.Empty() || !r.clip.In(image.Rect(10, 224, 630, 400)) {
					t.Fatal("text outside moving panel", r.clip)
				}
			}
			done()
			if g.nativePreparationUI != assets || assets.status.Font != font || !bytes.Equal(before, m.final) || !reflect.DeepEqual(state, *actor) {
				t.Fatal("Draw mutated frozen state or native assets")
			}
			if frame == 6 {
				g.acknowledgeNativeDeathRewardMessage()
			} else {
				g.stepNativeClassUILifecycle(time.Time{})
			}
		}
		if g.nativeSystemEndTurnUI != nil || calls != 1 {
			t.Fatal("message did not continue exactly once")
		}
	}
	broken := &nativeDeathRewardMessageState{hdInputErr: fmt.Errorf("fixed source mismatch"), hdWork: make([]byte, 1), hdPalette: g.nativeClassUI.palette}
	old := g.modernStoryPortraits
	g.nativeSystemEndTurnUI = &nativeSystemEndTurnUIState{source: make([]byte, 64000), rewardMessage: broken}
	bad := &modernStoryPortraitSet{mapSpritesHD: make(map[int][]image.Image)}
	for i := 0; i < 125; i++ {
		bad.mapSpritesHD[i] = nil
	}
	if err := g.applyTheme(bad); err == nil || g.modernStoryPortraits != old || broken.hdWorld != nil {
		t.Fatal("bad frozen source partially published")
	}
}

type rewardHDGPUCase struct {
	g           *Game
	actor       *battle.Unit
	kind, value int
	calls       *int
	snapshot    battle.Unit
	rng         uint16
	final       []byte
}
type rewardHDGPUProbe struct {
	set          *modernStoryPortraitSet
	cases        []rewardHDGPUCase
	index, frame int
	err          error
	out          string
	rows         []map[string]any
	final        *ebiten.Image
	pixels       []byte
	world        []byte
}

func (p *rewardHDGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *rewardHDGPUProbe) Update() error {
	if p.err != nil || p.index == len(p.cases) {
		return ebiten.Termination
	}
	return nil
}
func (p *rewardHDGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index >= len(p.cases) {
		return
	}
	c := &p.cases[p.index]
	g := c.g
	m := g.nativeSystemEndTurnUI.rewardMessage
	job := g.nativeClassUIJob
	g.frame++
	g.Draw(screen)
	if g.loadErr != "" || !g.modernHDWorldVisible || g.modernHDMapPresented != m.hdWorld || (job != nil && !job.drawn) {
		p.err = fmt.Errorf("complete Game.Draw missing: %s", g.loadErr)
		return
	}
	if !reflect.DeepEqual(c.snapshot, *c.actor) || g.nativeRNGState != c.rng || !bytes.Equal(c.final, m.final) || g.gold != 123 || *c.calls != 0 {
		p.err = fmt.Errorf("reward changed state before close")
		return
	}
	var geom ebiten.GeoM
	geom.Scale(3, 3)
	presentation := &modernHDPresentation{Game: g}
	p.final.Clear()
	presentation.DrawFinalScreen(p.final, screen, geom)
	p.final.ReadPixels(p.pixels)
	digest := sha256.Sum256(p.pixels)
	_, region, ok := modernHDRewardRegion(m, job)
	if !ok {
		p.err = fmt.Errorf("unknown panel phase")
		return
	}
	g.modernHDMapGPU.World.ReadPixels(p.world)
	// The complete final screen outside the actual moving panel must remain
	// the frozen HD world, including places where a native snapshot differs.
	outside, portraitPoints := 0, 0
	for y := 0; y < 1200; y++ {
		for x := 0; x < 1920; x++ {
			if image.Pt(x/6, y/6).In(region) {
				continue
			}
			at := (y*1920 + x) * 4
			for ch := 0; ch < 4; ch++ {
				d := int(p.pixels[at+ch]) - int(p.world[at+ch])
				if d < -2 || d > 2 {
					pic := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
					copy(pic.Pix, p.pixels)
					if f, e := os.Create(filepath.Join(p.out, "rejected-final.png")); e == nil {
						_ = png.Encode(f, pic)
						_ = f.Close()
					}
					if f, e := os.Create(filepath.Join(p.out, "rejected-logical.png")); e == nil {
						_ = png.Encode(f, screen)
						_ = f.Close()
					}
					p.err = fmt.Errorf("HD world covered outside panel at%d,%d frame%d channel%d got%d want%d text%d overlay%v region%v", x, y, p.frame, ch, p.pixels[at+ch], p.world[at+ch], len(g.modernHDText.runs), g.modernHDOverlay != nil, region)
					return
				}
			}
			outside++
		}
	}
	if overlay := g.modernHDOverlay; overlay != nil {
		src := overlay.portrait.hdImage
		at := overlay.rect.Intersect(overlay.clip)
		for y := at.Min.Y*3 + 1; y < at.Max.Y*3; y += 3 {
			for x := at.Min.X*3 + 1; x < at.Max.X*3; x += 3 {
				sx := src.Bounds().Dx() - 1 - (x - overlay.rect.Min.X*3)
				sy := y - overlay.rect.Min.Y*3
				r, gg, b, a := src.At(sx, sy).RGBA()
				if a != 65535 {
					continue
				}
				i := (y*1920 + x) * 4
				for ch, want := range []uint32{r >> 8, gg >> 8, b >> 8} {
					d := int(p.pixels[i+ch]) - int(want)
					if d < -2 || d > 2 {
						p.err = fmt.Errorf("HD portrait source/mirror mismatch at%d,%d", x, y)
						return
					}
				}
				portraitPoints++
			}
		}
		if portraitPoints < 100 {
			p.err = fmt.Errorf("empty portrait source check")
			return
		}
	}
	if err := g.applyTheme(nil); err != nil {
		p.err = err
		return
	}
	g.Draw(screen)
	if g.modernHDWorldVisible || g.modernHDOverlay != nil {
		p.err = fmt.Errorf("faithful retained HD")
		return
	}
	if err := g.applyTheme(p.set); err != nil {
		p.err = err
		return
	}
	g.Draw(screen)
	p.final.Clear()
	presentation.DrawFinalScreen(p.final, screen, geom)
	p.final.ReadPixels(p.pixels)
	if sha256.Sum256(p.pixels) != digest || g.nativeSystemEndTurnUI.rewardMessage != m || g.nativeClassUIJob != job || !reflect.DeepEqual(c.snapshot, *c.actor) {
		p.err = fmt.Errorf("F2 changed owner/state/frame")
		return
	}
	row := map[string]any{"kind": c.kind, "value": c.value, "phase_index": p.frame, "outside_hd_pixels": outside, "portrait_source_points": portraitPoints, "text_runs": len(g.modernHDText.runs), "final_sha256": fmt.Sprintf("%x", digest), "f2_roundtrip": true}
	if p.frame == 0 || p.frame == 5 || p.frame == 6 || p.frame == 9 || p.frame == 12 {
		name := fmt.Sprintf("reward-kind%d-phase%02d.png", c.kind, p.frame)
		f, e := os.Create(filepath.Join(p.out, name))
		if e != nil {
			p.err = e
			return
		}
		pic := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
		copy(pic.Pix, p.pixels)
		e = png.Encode(f, pic)
		ce := f.Close()
		if e != nil {
			p.err = e
			return
		}
		if ce != nil {
			p.err = ce
			return
		}
		row["capture"] = name
	}
	p.rows = append(p.rows, row)
	if p.frame == 6 {
		g.acknowledgeNativeDeathRewardMessage()
	} else {
		job.drawn = false
		old := job.frame
		g.stepNativeClassUILifecycle(time.Time{})
		if job.frame != old {
			p.err = fmt.Errorf("undrawn frame advanced")
			return
		}
		job.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	p.frame++
	if p.frame == 13 {
		want := 123
		if c.kind == 1 {
			want += c.value
		}
		if g.nativeSystemEndTurnUI != nil || g.gold != want || *c.calls != 1 || !reflect.DeepEqual(c.snapshot, *c.actor) || g.nativeRNGState != c.rng {
			p.err = fmt.Errorf("reward final transaction/continuation changed")
			return
		}
		p.index++
		p.frame = 0
	}
}

func TestModernHDRewardCompleteGameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_REWARD_GPU") != "1" {
		t.Skip("dedicated GPU process required")
	}
	t.Setenv("FD2_MUTE", "1")
	set, err := loadModernStoryPortraitSet(os.Getenv("FD2_TEST_REWARD_CATALOG"), os.Getenv("FD2_TEST_REWARD_PACK"))
	if err != nil {
		t.Fatal(err)
	}
	out := os.Getenv("FD2_TEST_REWARD_OUT")
	if out == "" {
		t.Fatal("output required")
	}
	if err = os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &rewardHDGPUProbe{set: set, out: out, final: ebiten.NewImage(1920, 1200), pixels: make([]byte, 1920*1200*4), world: make([]byte, 1920*1200*4)}
	for _, spec := range [][2]int{{0, 192}, {1, 123456}} {
		g, actor := rewardHDTestGame(t)
		g.modernStoryPortraits = set
		calls := new(int)
		if err = g.beginNativeDeathRewardMessage(pendingNativeDeathRewardMessage{killer: actor, kind: spec[0], value: spec[1]}, actor, func() { *calls++ }); err != nil {
			t.Fatal(err)
		}
		if set.portraits[actor.BattleFig] == nil || set.portraits[actor.BattleFig].hdImage == nil {
			t.Fatal("fixture lacks HD portrait")
		}
		m := g.nativeSystemEndTurnUI.rewardMessage
		p.cases = append(p.cases, rewardHDGPUCase{g: g, actor: actor, kind: spec[0], value: spec[1], calls: calls, snapshot: *actor, rng: g.nativeRNGState, final: append([]byte(nil), m.final...)})
	}
	ebiten.SetVsyncEnabled(false)
	if err = ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || p.index != 2 || len(p.rows) != 26 {
		t.Fatalf("reward GPU cases%d rows%d: %v", p.index, len(p.rows), p.err)
	}
	b, _ := json.MarshalIndent(map[string]any{"status": "PASS-REWARD-HD-RUNTIME-E1", "cases": 2, "game_frames": 26, "f2_round_trips": 26, "rows": p.rows, "normal_player_verified": false}, "", "  ")
	if err = os.WriteFile(filepath.Join(out, "receipt.json"), append(b, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
