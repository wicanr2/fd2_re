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
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestModernHDTransientExpiryNativeCoverage(t *testing.T) {
	final := bytes.Repeat([]byte{77}, 64000)
	var all [2][][]byte
	for i, fill := range []byte{0, 255} {
		source := bytes.Repeat([]byte{fill}, 64000)
		opening, err := campaign.NativeClassListOpeningFrames(source, final)
		if err != nil {
			t.Fatal(err)
		}
		closing, err := campaign.NativeClassListClosingFrames(source, final)
		if err != nil {
			t.Fatal(err)
		}
		all[i] = append(append(opening, closing...), source)
	}
	job := &nativeClassUIJob{frames: all[0][:11], restore: all[0][11], expiryText: &nativeTransientExpiryPresentation{}}
	for i := range all[0] {
		job.frame = i
		_, region, ok := modernHDTransientExpiryRegion(job)
		if !ok {
			t.Fatal("known expiry phase rejected", i)
		}
		for p := range all[0][i] {
			written := all[0][i][p] == all[1][i][p]
			if written != image.Pt(p%320, p/320).In(region) {
				t.Fatalf("native coverage differs at phase%d pixel%d", i, p)
			}
		}
	}
	job.frame = 12
	if _, _, ok := modernHDTransientExpiryRegion(job); ok {
		t.Fatal("unknown phase accepted")
	}
	job.frame, job.expiryText = 0, nil
	if _, _, ok := modernHDTransientExpiryRegion(job); ok {
		t.Fatal("unrelated class UI accepted")
	}
	g := &Game{transientUI: true, nativeClassUIJob: job, modernStoryPortraits: &modernStoryPortraitSet{hasHD: true}}
	if g.modernHDTransientExpiryPortraitAllowed() {
		t.Fatal("unrelated transient portrait guard removed")
	}
}

func expiryHDTestGame(t *testing.T, set *modernStoryPortraitSet, locale string) (*Game, *battle.Unit) {
	t.Helper()
	g, unit := rewardHDTestGame(t)
	// Explicit typed constructor, not a normally acquired status or save.
	g.st.HasNativeRuntimeUnitProjection = false
	unit.HasNativeRecordByte6, unit.NativeRecordByte6 = true, 2
	unit.NativeTransient = [6]byte{1, 1, 1, 1, 1, 1}
	unit.HP, unit.MaxHP = 100, 100
	unit.EquipmentBaseSet, unit.BaseAP, unit.BaseDP, unit.DX = true, 100, 60, 20
	g.localeID = locale
	var err error
	g.localeCatalog, err = loadOfficialLocale(locale)
	if err != nil {
		t.Fatal(err)
	}
	g.localeContent, err = loadOfficialLocaleContent(locale)
	if err != nil {
		t.Fatal(err)
	}
	g.localeEntities, err = loadOfficialLocaleEntities(locale)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.applyTheme(set); err != nil {
		t.Fatal(err)
	}
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	return g, unit
}

func TestModernHDTransientExpiryRejectsBeforeCountdown(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	g, unit := expiryHDTestGame(t, nil, "zh-Hant")
	// An incomplete nonempty HD world must fail before the HP/status transaction.
	g.modernStoryPortraits = &modernStoryPortraitSet{hasHD: true, mapSpritesHD: map[int][]image.Image{0: nil}}
	before, state, rng := *unit, g.st, g.nativeRNGState
	if err := g.beginNativeTransientPhases([]byte{2}, nil); err == nil || !strings.Contains(err.Error(), "modern HD expiry") {
		t.Fatal("incomplete HD source did not reach its source guard", err)
	}
	if g.st != state || g.st.Units[0] != unit || !reflect.DeepEqual(before, *unit) ||
		g.nativeRNGState != rng || g.nativeClassUIJob != nil || g.nativeLevelUpDialogue != nil || g.transientUI {
		t.Fatal("failed HD preflight published countdown, HP, RNG or owner")
	}
	g.modernStoryPortraits = nil
	g.localeID, g.localeCatalog = "en", nil
	if err := g.beginNativeTransientPhases([]byte{2}, nil); err == nil || g.st != state || !reflect.DeepEqual(before, *unit) {
		t.Fatal("missing translation published a status transaction")
	}
}

func TestModernHDTransientExpiryNativeChain(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	g, unit := expiryHDTestGame(t, nil, "zh-Hant")
	calls := 0
	if err := g.beginNativeTransientPhases([]byte{2}, func() { calls++ }); err != nil {
		t.Fatal(err)
	}
	before := *unit
	var rng uint16
	seen := map[int]bool{}
	screen := ebiten.NewImage(640, 400)
	for i := 0; calls == 0 && i < 1500; i++ {
		g.frame++
		g.Draw(screen)
		if job := g.nativeClassUIJob; job != nil && job.expiryText != nil {
			if len(seen) == 0 {
				rng = g.nativeRNGState
			}
			seen[job.expiryText.counter] = true
		}
		g.stepNativeStoryDialogueProgress()
		g.stepNativeLevelUpDialogue(false)
		g.stepNativeClassUILifecycle(time.Time{})
		if g.loadErr != "" {
			t.Fatal(g.loadErr)
		}
	}
	if calls != 1 || len(seen) != 6 || g.transientUI || g.nativeClassUIJob != nil ||
		!reflect.DeepEqual(before, *unit) || g.nativeRNGState != rng {
		t.Fatalf("native damage/expiry chain calls%d counters%d", calls, len(seen))
	}
}

type expiryHDGPUProbe struct {
	t                    *testing.T
	set                  *modernStoryPortraitSet
	g                    *Game
	unit                 *battle.Unit
	before               battle.Unit
	rng                  uint16
	calls                int
	localeIndex          int
	expiryFrames         int
	prefixFrames         int
	err                  error
	out                  string
	final                *ebiten.Image
	pixels, world        []byte
	withoutText          []byte
	rows                 []map[string]any
	rejectedSources      int
	f2RoundTrips         int
	chromeSource         dialogueChromeSource
	chromePoints         int
	legacyChromeRejected bool
}

var expiryHDLocales = []string{"zh-Hant", "zh-Hans", "en", "ja"}

func (p *expiryHDGPUProbe) startCase() {
	p.g, p.unit = expiryHDTestGame(p.t, p.set, expiryHDLocales[p.localeIndex])
	p.chromeSource = nativeDialogueChromeSource(p.g.nativeClassUI.dialogue, p.g.nativeClassUI.palette)
	p.calls, p.expiryFrames, p.prefixFrames = 0, 0, 0
	if err := p.g.beginNativeTransientPhases([]byte{2}, func() { p.calls++ }); err != nil {
		p.err = err
		return
	}
	if p.unit.HP != 90 || p.unit.NativeTransient != [6]byte{} || p.g.nativeLevelUpDialogue == nil {
		p.err = fmt.Errorf("typed status phase did not produce damage then expiry")
		return
	}
	p.before = *p.unit
}

func (p *expiryHDGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *expiryHDGPUProbe) Update() error {
	if p.err != nil || p.localeIndex == len(expiryHDLocales) {
		return ebiten.Termination
	}
	return nil
}
func (p *expiryHDGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.localeIndex == len(expiryHDLocales) {
		return
	}
	g := p.g
	g.frame++
	g.Draw(screen)
	if g.loadErr != "" {
		p.err = fmt.Errorf("Game.Draw: %s", g.loadErr)
		return
	}
	if g.nativeLevelUpDialogue != nil {
		p.prefixFrames++
		g.stepNativeStoryDialogueProgress()
		g.stepNativeLevelUpDialogue(false)
		g.stepNativeClassUILifecycle(time.Time{})
		if p.prefixFrames > 1200 || p.calls != 0 {
			p.err = fmt.Errorf("damage prefix did not naturally hand off")
		}
		return
	}
	job := g.nativeClassUIJob
	if job == nil || job.expiryText == nil || !job.drawn || !g.transientUI || !g.modernHDWorldVisible ||
		g.modernHDMapPresented != job.expiryText.world {
		p.err = fmt.Errorf("expiry owner or frozen HD world missing at phase%d: %s", p.expiryFrames, g.loadErr)
		return
	}
	if p.expiryFrames == 0 {
		p.rng = g.nativeRNGState
		if p.localeIndex == 0 {
			message := job.expiryText
			world := message.world
			for _, source := range [][]byte{message.source, message.scene.vga, message.scene.work} {
				source[0] ^= 1
				err := g.applyTheme(p.set)
				source[0] ^= 1
				if err == nil || g.modernStoryPortraits != p.set || message.world != world || g.nativeClassUIJob != job ||
					!reflect.DeepEqual(p.before, *p.unit) || g.nativeRNGState != p.rng {
					p.err = fmt.Errorf("corrupt frozen expiry source partially published")
					return
				}
				p.rejectedSources++
			}
		}
	}
	if job.expiryText.counter != p.expiryFrames/12 || job.frame != p.expiryFrames%12 ||
		!reflect.DeepEqual(p.before, *p.unit) || p.calls != 0 || g.nativeRNGState != p.rng {
		p.err = fmt.Errorf("expiry order/state/RNG changed")
		return
	}
	var geometry ebiten.GeoM
	geometry.Scale(3, 3)
	presentation := &modernHDPresentation{Game: g}
	p.final.Clear()
	presentation.DrawFinalScreen(p.final, screen, geometry)
	p.final.ReadPixels(p.pixels)
	digest := sha256.Sum256(p.pixels)
	shift, region, ok := modernHDTransientExpiryRegion(job)
	if !ok {
		p.err = fmt.Errorf("unknown expiry phase")
		return
	}
	chromePoints := 0
	if !region.Empty() {
		if len(g.modernHDText.panels) != 1 {
			p.err = fmt.Errorf("HD expiry chrome not staged")
			return
		}
		full := image.Rect(5, 112+shift, 315, 198+shift)
		var err error
		chromePoints, err = checkDialogueChromePixels(p.pixels, p.chromeSource, full)
		if err != nil {
			p.err = err
			return
		}
		p.chromePoints += chromePoints
		if p.localeIndex == 0 && p.expiryFrames == 5 {
			panels := g.modernHDText.panels
			g.modernHDText.panels = nil
			p.final.Clear()
			presentation.DrawFinalScreen(p.final, screen, geometry)
			p.final.ReadPixels(p.withoutText)
			g.modernHDText.panels = panels
			if _, err := checkDialogueChromePixels(p.withoutText, p.chromeSource, full); err == nil {
				p.err = fmt.Errorf("legacy enlarged dialogue border accepted")
				return
			}
			p.legacyChromeRejected = true
		}
	} else if len(g.modernHDText.panels) != 0 {
		p.err = fmt.Errorf("HD expiry chrome survived restore")
		return
	}
	g.modernHDMapGPU.World.ReadPixels(p.world)
	outside, portraitPoints := 0, 0
	for y := 0; y < 1200; y++ {
		for x := 0; x < 1920; x++ {
			if image.Pt(x/6, y/6).In(region) {
				continue
			}
			i := (y*1920 + x) * 4
			for c := 0; c < 4; c++ {
				d := int(p.pixels[i+c]) - int(p.world[i+c])
				if d < -2 || d > 2 {
					p.err = fmt.Errorf("frozen world covered outside panel at%d,%d phase%d", x, y, p.expiryFrames)
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
				sx, sy := src.Bounds().Dx()-1-(x-overlay.rect.Min.X*3), y-overlay.rect.Min.Y*3
				r, green, b, a := src.At(sx, sy).RGBA()
				if a != 65535 {
					continue
				}
				i := (y*1920 + x) * 4
				for c, want := range []uint32{r >> 8, green >> 8, b >> 8} {
					d := int(p.pixels[i+c]) - int(want)
					if d < -2 || d > 2 {
						p.err = fmt.Errorf("HD portrait source or mirror differs at%d,%d", x, y)
						return
					}
				}
				portraitPoints++
			}
		}
		if portraitPoints < 100 {
			p.err = fmt.Errorf("insufficient HD portrait samples")
			return
		}
	} else if job.frame < 11 {
		p.err = fmt.Errorf("expiry transient guard hid HD portrait")
		return
	}
	for _, run := range g.modernHDText.runs {
		if run.clip.Empty() || !run.clip.In(image.Rect(region.Min.X*2, region.Min.Y*2, region.Max.X*2, region.Max.Y*2)) {
			p.err = fmt.Errorf("HD text escaped moving panel")
			return
		}
	}
	if job.frame == 5 && len(g.modernHDText.runs) == 0 {
		p.err = fmt.Errorf("expiry retained native text")
		return
	}
	var visibleText strings.Builder
	for i := 1; i < len(g.modernHDText.runs); i += 2 {
		visibleText.WriteString(g.modernHDText.runs[i].text)
	}
	if job.frame == 5 && g.localeID != "zh-Hant" {
		want, err := g.localeCatalog.Format(nativeTransientExpiryLocaleKeys[job.expiryText.counter])
		if err != nil || strings.Join(strings.Fields(visibleText.String()), "") != strings.Join(strings.Fields(want), "") {
			p.err = fmt.Errorf("displayed expiry text did not use the selected locale: %q", visibleText.String())
			return
		}
	}
	textPixels := 0
	if job.frame == 5 {
		runs := g.modernHDText.runs
		g.modernHDText.runs = nil
		p.final.Clear()
		presentation.DrawFinalScreen(p.final, screen, geometry)
		p.final.ReadPixels(p.withoutText)
		g.modernHDText.runs = runs
		for i := 0; i < len(p.pixels); i += 4 {
			if bytes.Equal(p.pixels[i:i+4], p.withoutText[i:i+4]) {
				continue
			}
			point := image.Pt((i/4)%1920, (i/4)/1920)
			covered := false
			for _, run := range runs {
				clip := image.Rect(run.clip.Min.X*3, run.clip.Min.Y*3, run.clip.Max.X*3, run.clip.Max.Y*3)
				covered = covered || point.In(clip)
			}
			if !covered {
				p.err = fmt.Errorf("actual HD glyph ink escaped text field at%v", point)
				return
			}
			textPixels++
		}
		if textPixels < 100 {
			p.err = fmt.Errorf("declared HD text did not reach final pixels")
			return
		}
	}
	f2 := job.frame == 5 || job.expiryText.counter == 0 && (job.frame == 0 || job.frame == 9 || job.frame == 11)
	if f2 {
		if err := g.applyTheme(nil); err != nil {
			p.err = err
			return
		}
		g.Draw(screen)
		if g.modernHDWorldVisible || g.modernHDOverlay != nil || g.nativeClassUIJob != job {
			p.err = fmt.Errorf("F2 original retained HD or changed owner")
			return
		}
		if err := g.applyTheme(p.set); err != nil {
			p.err = err
			return
		}
		g.Draw(screen)
		p.final.Clear()
		presentation.DrawFinalScreen(p.final, screen, geometry)
		p.final.ReadPixels(p.pixels)
		if sha256.Sum256(p.pixels) != digest || g.nativeClassUIJob != job || !reflect.DeepEqual(p.before, *p.unit) || g.nativeRNGState != p.rng {
			p.err = fmt.Errorf("F2 changed frame, state or final pixels")
			return
		}
		p.f2RoundTrips++
	}
	row := map[string]any{"locale": expiryHDLocales[p.localeIndex], "counter": job.expiryText.counter, "frame": job.frame, "outside_hd_pixels": outside, "portrait_source_points": portraitPoints, "dialogue_chrome_pixels": chromePoints, "text_runs": len(g.modernHDText.runs), "actual_text_pixels": textPixels, "visible_text": visibleText.String(), "f2_roundtrip": f2, "final_rgba_sha256": fmt.Sprintf("%x", digest)}
	if job.frame == 5 || job.expiryText.counter == 0 && (job.frame == 0 || job.frame == 9 || job.frame == 11) {
		name := fmt.Sprintf("expiry-%s-counter%d-frame%02d.png", expiryHDLocales[p.localeIndex], job.expiryText.counter, job.frame)
		f, err := os.Create(filepath.Join(p.out, name))
		if err != nil {
			p.err = err
			return
		}
		pic := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
		copy(pic.Pix, p.pixels)
		err = png.Encode(f, pic)
		closeErr := f.Close()
		if err != nil {
			p.err = err
			return
		}
		if closeErr != nil {
			p.err = closeErr
			return
		}
		row["capture"] = name
	}
	p.rows = append(p.rows, row)
	job.drawn = false
	g.stepNativeClassUILifecycle(time.Time{})
	if g.nativeClassUIJob != job || job.frame != p.expiryFrames%12 {
		p.err = fmt.Errorf("undrawn expiry advanced")
		return
	}
	job.drawn = true
	g.stepNativeClassUILifecycle(time.Time{})
	p.expiryFrames++
	if p.expiryFrames == 72 {
		if p.calls != 1 || g.nativeClassUIJob != nil || g.transientUI || !reflect.DeepEqual(p.before, *p.unit) || g.nativeRNGState != p.rng {
			p.err = fmt.Errorf("expiry failed final restore/one continuation")
			return
		}
		p.localeIndex++
		if p.localeIndex < len(expiryHDLocales) {
			p.startCase()
		}
	}
}

func TestModernHDTransientExpiryCompleteGameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_EXPIRY_GPU") != "1" {
		t.Skip("dedicated GPU process required")
	}
	t.Setenv("FD2_MUTE", "1")
	set, err := loadModernStoryPortraitSet(os.Getenv("FD2_TEST_EXPIRY_CATALOG"), os.Getenv("FD2_TEST_EXPIRY_PACK"))
	if err != nil {
		t.Fatal(err)
	}
	out := os.Getenv("FD2_TEST_EXPIRY_OUT")
	if out == "" {
		t.Fatal("output required")
	}
	if err = os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &expiryHDGPUProbe{t: t, set: set, out: out, final: ebiten.NewImage(1920, 1200), pixels: make([]byte, 1920*1200*4), world: make([]byte, 1920*1200*4), withoutText: make([]byte, 1920*1200*4)}
	p.startCase()
	ebiten.SetVsyncEnabled(false)
	if err = ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || p.localeIndex != 4 || len(p.rows) != 288 || p.rejectedSources != 3 || p.f2RoundTrips != 36 || !p.legacyChromeRejected || p.chromePoints == 0 {
		t.Fatalf("locales%d frames%d: %v", p.localeIndex, len(p.rows), p.err)
	}
	data, _ := json.MarshalIndent(map[string]any{"status": "PASS-EXPIRY-HD-RUNTIME-E1", "locales": expiryHDLocales, "counters": 6, "game_frames": 288, "f2_roundtrips": p.f2RoundTrips, "rejected_sources": p.rejectedSources, "dialogue_chrome": true, "chrome_source": p.chromeSource, "chrome_pixels": p.chromePoints, "legacy_chrome_rejected": p.legacyChromeRejected, "normal_player_verified": false, "constructor": "six typed counters=1, selector2, HP100/100, base AP100/DP60/DX20; actual damage message then expiry chain", "rows": p.rows}, "", "  ")
	if err = os.WriteFile(filepath.Join(out, "receipt.json"), append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
