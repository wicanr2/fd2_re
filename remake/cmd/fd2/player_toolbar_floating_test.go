package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"golang.org/x/image/font"
)

func TestFloatingToolbarGeometryAndOwnership(t *testing.T) {
	p := &modernHDPresentation{Game: &Game{}, toolbar: true, floatingToolbar: true}
	if w, h := p.Layout(1, 1); w != 640 || h != 400 || p.toolbarOffset() != 0 {
		t.Fatalf("floating layout %d,%d offset %d", w, h, p.toolbarOffset())
	}
	p.stepToolbarVisibility(image.Pt(320, 4), nil)
	if p.toolbarVisible() {
		t.Fatal("hidden bar expanded its hot edge")
	}
	p.stepToolbarVisibility(image.Pt(320, 3), nil)
	if !p.toolbarVisible() {
		t.Fatal("edge did not reveal")
	}
	p.stepToolbarVisibility(image.Pt(320, 23), nil)
	if p.toolbarTicks != 90 {
		t.Fatal("hover did not retain")
	}
	for i := 0; i < 89; i++ {
		p.stepToolbarVisibility(image.Pt(-1, -1), nil)
	}
	if !p.toolbarVisible() {
		t.Fatal("bar hid before grace period")
	}
	p.stepToolbarVisibility(image.Pt(320, 24), nil)
	if p.toolbarVisible() {
		t.Fatal("bar did not hide")
	}
	p.stepToolbarVisibility(image.Pt(-1, -1), []image.Point{image.Pt(320, 1)})
	if !p.toolbarVisible() {
		t.Fatal("touch did not reveal")
	}
	for _, row := range []struct {
		top     int
		block   bool
		y, want int
		ok      bool
	}{
		{0, false, 12, 12, true}, {0, true, 12, 0, false},
		{0, true, 24, 24, true}, {24, false, 12, 0, false}, {24, false, 24, 0, true},
	} {
		g := &Game{playerViewportTop: row.top, playerToolbarTouchBlocked: row.block}
		x, y, ok := g.playerTouchCoordinates(300, row.y)
		if ok != row.ok || ok && (x != 300 || y != row.want) {
			t.Fatalf("touch %+v => %d,%d,%v", row, x, y, ok)
		}
	}
	p.floatingToolbar = false
	if w, h := p.Layout(1, 1); w != 640 || h != 424 || !p.toolbarVisible() {
		t.Fatal("fixed compatibility changed")
	}
	p.toolbar = false
	if _, h := p.Layout(1, 1); h != 400 || p.toolbarVisible() {
		t.Fatal("no-toolbar compatibility changed")
	}
}

func TestFloatingToolbarLocaleTextBounds(t *testing.T) {
	f := loadFont()
	for _, locale := range []string{"zh-Hant", "zh-Hans", "en", "ja"} {
		local := f
		if regional := f.hdFonts[locale]; regional != nil {
			local = regional
		}
		face, _ := local.faceFor(36)
		if face == nil {
			t.Fatal("missing runtime face", locale)
		}
		labels := playerThemeLabels(locale)
		for i, label := range []string{labels[0], labels[1], playerLocaleLabel(locale)} {
			rect := playerLocaleButton
			if i < 2 {
				rect = playerThemeButtons[i]
			}
			ink, _ := font.BoundString(face, label)
			inkRect := image.Rect(ink.Min.X.Floor(), ink.Min.Y.Floor(), ink.Max.X.Ceil(), ink.Max.Y.Ceil()).Add(image.Pt((rect.Min.X+8)*3, playerToolbarTextBaseline(face, label)))
			safe := image.Rect((rect.Min.X+2)*3, (rect.Min.Y+1)*3, (rect.Max.X-2)*3, (rect.Max.Y-1)*3)
			if !inkRect.In(safe) {
				t.Fatalf("%s %q ink %v outside %v", locale, label, inkRect, safe)
			}
		}
	}
}

type floatingToolbarStep struct {
	input, locale     string
	hd, visible, help bool
}
type floatingToolbarProbe struct {
	p                         *modernHDPresentation
	driver, out               string
	steps                     []floatingToolbarStep
	index, updates            int
	started, ready, completed bool
	inputDone                 bool
	hideWait                  int
	done                      chan error
	err                       error
	checks                    []map[string]any
	changes                   []bool
}

func (q *floatingToolbarProbe) Update() error {
	if q.err != nil || q.index >= len(q.steps) {
		return ebiten.Termination
	}
	if q.completed {
		return nil
	}
	q.updates++
	if q.updates > 2400 {
		q.err = fmt.Errorf("floating toolbar exceeded update bound")
		return ebiten.Termination
	}
	if q.ready && !q.started {
		q.started = true
		input := q.steps[q.index].input
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			raw, err := exec.CommandContext(ctx, q.driver, input).CombinedOutput()
			if writeErr := os.WriteFile(filepath.Join(q.out, fmt.Sprintf("input-%02d.log", q.index)), raw, 0644); err == nil {
				err = writeErr
			}
			q.done <- err
		}()
	}
	g := q.p.Game
	before := g.modernStoryPortraits
	gold, rng, node, curX, curY := g.gold, g.nativeRNGState, g.camp.Node(), g.curX, g.curY
	if err := q.p.Update(); err != nil {
		q.err = err
		return ebiten.Termination
	}
	if before != g.modernStoryPortraits {
		q.changes = append(q.changes, g.modernStoryPortraits != nil)
	}
	if g.gold != gold || g.nativeRNGState != rng || g.camp.Node() != node || g.curX != curX || g.curY != curY {
		q.err = fmt.Errorf("toolbar changed game state")
		return ebiten.Termination
	}
	select {
	case err := <-q.done:
		if err != nil {
			q.err = err
			return ebiten.Termination
		}
		q.inputDone = true
	default:
	}
	if !q.inputDone {
		return nil
	}
	s := q.steps[q.index]
	if !s.visible && q.p.toolbarVisible() && q.hideWait < playerToolbarHideTicks {
		// The contract counts Update calls, not wall time spent on GPU readback.
		q.hideWait++
		return nil
	}
	if (g.modernStoryPortraits != nil) != s.hd || g.localeID != s.locale || q.p.toolbarVisible() != s.visible || g.helpVisible != s.help || g.playerThemeRequested || g.playerLocaleRequested {
		x, y := ebiten.CursorPosition()
		q.err = fmt.Errorf("step %d: hd=%v locale=%s visible=%v ticks=%d pointer=%d,%d help=%v", q.index, g.modernStoryPortraits != nil, g.localeID, q.p.toolbarVisible(), q.p.toolbarTicks, x, y, g.helpVisible)
		return ebiten.Termination
	}
	q.completed = true // DrawFinalScreen checks and records before advancing.
	return nil
}
func (q *floatingToolbarProbe) Layout(w, h int) (int, int) { return q.p.Layout(w, h) }
func (q *floatingToolbarProbe) Draw(screen *ebiten.Image)  { q.p.Draw(screen) }
func (q *floatingToolbarProbe) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geometry ebiten.GeoM) {
	q.p.DrawFinalScreen(screen, offscreen, geometry)
	q.ready = true
	if !q.completed || q.err != nil {
		return
	}
	offset := q.p.toolbarOffset() * 3
	final, control := ebiten.NewImage(1920, 1200+offset), ebiten.NewImage(1920, 1200)
	defer final.Dispose()
	defer control.Dispose()
	var m ebiten.GeoM
	m.Scale(3, 3)
	q.p.DrawFinalScreen(final, offscreen, m)
	base := &modernHDPresentation{Game: q.p.Game}
	controlSource := offscreen
	if offset > 0 {
		controlSource = q.p.gameSurface
	}
	base.DrawFinalScreen(control, controlSource, m)
	a, b := make([]byte, 1920*(1200+offset)*4), make([]byte, 1920*1200*4)
	final.ReadPixels(a)
	control.ReadPixels(b)
	cut := offset * 1920 * 4
	if offset == 0 && q.p.toolbarVisible() {
		cut = 1920 * 72 * 4
		if bytes.Equal(a[:cut], b[:cut]) {
			q.err = fmt.Errorf("visible toolbar absent")
			return
		}
	}
	controlCut := 0
	if offset == 0 {
		controlCut = cut
	}
	if !bytes.Equal(a[cut:], b[controlCut:]) {
		q.err = fmt.Errorf("toolbar displaced or changed game outside its row")
		return
	}
	file, err := os.Create(filepath.Join(q.out, fmt.Sprintf("floating-%02d.png", q.index)))
	if err == nil {
		err = png.Encode(file, &image.RGBA{Pix: a, Stride: 1920 * 4, Rect: image.Rect(0, 0, 1920, 1200+offset)})
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
	}
	if err != nil {
		q.err = err
		return
	}
	q.checks = append(q.checks, map[string]any{"step": q.index, "input": q.steps[q.index].input, "locale": q.p.localeID, "hd": q.p.modernStoryPortraits != nil, "visible": q.p.toolbarVisible(), "help": q.p.helpVisible, "unchanged_game_bytes": len(a) - cut})
	q.index++
	if q.index < len(q.steps) {
		q.completed = false
		q.started = false
		q.inputDone = false
		q.hideWait = 0
	}
}

func TestFloatingToolbarRealInputGPU(t *testing.T) {
	if os.Getenv("FD2_FLOATING_TOOLBAR_GPU") != "1" {
		t.Skip("dedicated real input process")
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	set := facilityHDTestSet(t)
	g := facilityHDTestGame(t, set)
	variant := 0
	g.camp = campaign.NewRunner(&campaign.Campaign{Start: "town", Nodes: map[string]*campaign.Node{"town": {Type: "town", NativeTownVariant: &variant}}})
	g.nativeShopMode = ""
	g.nativeShopUIJob = nil
	if err := g.saveGameToSlot(0); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(saveSlotPath(0))
	if err != nil {
		t.Fatal(err)
	}
	out, driver := os.Getenv("FD2_TEST_TOOLBAR_OUT"), os.Getenv("FD2_TEST_XTEST_INPUT")
	if out == "" || driver == "" {
		t.Fatal("explicit output and tracked input binary required")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	q := &floatingToolbarProbe{p: &modernHDPresentation{Game: g, toolbar: true, floatingToolbar: true, faithfulWidth: 1280, faithfulHeight: 800}, driver: driver, out: out, done: make(chan error, 1), steps: []floatingToolbarStep{
		{"Move=320,300@640x400:2000", "zh-Hant", true, false, false},
		{"Move=320,1@640x400:400", "zh-Hant", true, true, false},
		{"Click=220,12@640x400:400", "zh-Hant", true, true, false},
		{"Click=80,12@640x400:400", "zh-Hant", false, true, false},
		{"F2:400", "zh-Hant", true, true, false},
		{"Click=380,12@640x400:400", "zh-Hans", true, true, false},
		{"F4:400", "ja", true, true, false},
		{"Click=380,12@640x400:400", "en", true, true, false},
		{"Click=380,12@640x400:400", "zh-Hant", true, true, false},
		{"F1:300", "zh-Hant", true, true, true},
		{"Click=80,12@640x400:300", "zh-Hant", true, true, true},
		{"Click=380,12@640x400:300", "zh-Hant", true, true, true},
		{"F1:300", "zh-Hant", true, true, false},
		{"Move=320,300@640x400:2000", "zh-Hant", true, false, false},
		{"F2:400", "zh-Hant", false, false, false},
		{"F2:400", "zh-Hant", true, false, false},
	}}
	ebiten.SetWindowSize(1280, 800)
	ebiten.SetWindowTitle("FD2 Floating Toolbar Input")
	if err := ebiten.RunGame(q); err != nil {
		t.Fatal(err)
	}
	if q.err != nil {
		t.Fatal(q.err)
	}
	if len(q.checks) != len(q.steps) {
		t.Fatalf("incomplete checks %d", len(q.checks))
	}
	if err := g.saveGameToSlot(0); err != nil {
		t.Fatal(err)
	}
	resaved, err := os.ReadFile(saveSlotPath(0))
	if err != nil || !bytes.Equal(saved, resaved) {
		t.Fatalf("toolbar changed save bytes: %v", err)
	}
	raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-FLOATING-TOOLBAR-REAL-INPUT", "checks": q.checks, "updates": q.updates, "save_bytes_unchanged": len(saved), "scope": "controlled town GUI; real pointer, click, F2/F4/F1; not full normal campaign or platform evidence"}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
