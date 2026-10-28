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
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
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
		labels := playerThemeLabels(locale)
		for i, label := range []string{labels[0], labels[1], playerLocaleLabel(locale)} {
			rect := playerLocaleButton
			if i < 2 {
				rect = playerThemeButtons[i]
			}
			face, caption, at := playerToolbarLabelLayout(local, label, rect)
			if face == nil {
				t.Fatal("missing runtime face", locale)
			}
			inkRect := text.BoundString(face, caption).Add(at)
			safe := image.Rect(rect.Min.X*3+12, rect.Min.Y*3+6, rect.Max.X*3-12, rect.Max.Y*3-6)
			if !inkRect.In(safe) {
				t.Fatalf("%s %q ink %v outside %v", locale, caption, inkRect, safe)
			}
		}
	}
}

type floatingToolbarStep struct {
	input, locale     string
	hd, visible, help bool
	ui                string
	speed             int
	fontScale         float64
	fullscreen        string
	poster            string
	posterMenu        string
	posterEdge        *int
	posterFade        *int
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
	inputAcks, pressObserved  bool
	inputTrace                []map[string]any
}

func (q *floatingToolbarProbe) observeInput() (bool, bool) {
	spec := strings.Split(q.steps[q.index].input, ":")[0]
	if strings.HasPrefix(spec, "Click=") {
		return inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft), !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	}
	if strings.HasPrefix(spec, "Move=") {
		var x, y, w, h int
		_, err := fmt.Sscanf(spec, "Move=%d,%d@%dx%d", &x, &y, &w, &h)
		cx, cy := ebiten.CursorPosition()
		return err == nil && cx >= x-1 && cx <= x+1 && cy >= y-1 && cy <= y+1, true
	}
	if i := strings.LastIndex(spec, "+"); i >= 0 {
		spec = spec[i+1:]
	}
	key, ok := map[string]ebiten.Key{"F1": ebiten.KeyF1, "F2": ebiten.KeyF2, "F4": ebiten.KeyF4, "F7": ebiten.KeyF7, "F8": ebiten.KeyF8, "F10": ebiten.KeyF10, "F11": ebiten.KeyF11, "Escape": ebiten.KeyEscape, "Enter": ebiten.KeyEnter}[spec]
	return ok && inpututil.IsKeyJustPressed(key), ok && !ebiten.IsKeyPressed(key)
}

func (q *floatingToolbarProbe) Update() error {
	if q.err != nil || q.index >= len(q.steps) {
		return ebiten.Termination
	}
	if q.completed {
		return nil
	}
	q.updates++
	if q.updates > 21600 {
		q.err = fmt.Errorf("floating toolbar exceeded update bound")
		return ebiten.Termination
	}
	if q.ready && !q.started {
		q.started = true
		input := q.steps[q.index].input
		index := q.index
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, q.driver, input)
			if q.inputAcks {
				command.Env = append(os.Environ(), "FD2_XTEST_PRESS_ACK="+filepath.Join(q.out, fmt.Sprintf("press-%02d.ack", index)), "FD2_XTEST_RELEASE_ACK="+filepath.Join(q.out, fmt.Sprintf("release-%02d.ack", index)))
			}
			raw, err := command.CombinedOutput()
			if writeErr := os.WriteFile(filepath.Join(q.out, fmt.Sprintf("input-%02d.log", index)), raw, 0644); err == nil {
				err = writeErr
			}
			q.done <- err
		}()
	}
	g := q.p.Game
	pressed, released := q.observeInput()
	before := g.modernStoryPortraits
	gold, rng, node, curX, curY := g.gold, g.nativeRNGState, g.camp.Node(), g.curX, g.curY
	if err := q.p.Update(); err != nil {
		q.err = err
		return ebiten.Termination
	}
	if q.inputAcks && q.started {
		phase := ""
		if !q.pressObserved && pressed {
			q.pressObserved, phase = true, "press"
		} else if q.pressObserved && released {
			phase = "release"
		}
		if phase != "" {
			path := filepath.Join(q.out, fmt.Sprintf("%s-%02d.ack", phase, q.index))
			if _, err := os.Stat(path); os.IsNotExist(err) {
				q.inputTrace = append(q.inputTrace, map[string]any{"step": q.index, "phase": phase, "controls_allowed": g.playerPosterControlsAllowed(), "poster": q.p.poster.settings.Set, "message": g.msg})
				if err := os.WriteFile(path, []byte("observed\n"), 0644); err != nil {
					q.err = err
					return ebiten.Termination
				}
			}
		}
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
	if (g.modernStoryPortraits != nil) != s.hd || g.localeID != s.locale || q.p.toolbarVisible() != s.visible || g.helpVisible != s.help || g.playerThemeRequested || g.playerLocaleRequested || g.playerInterfaceRequested || g.playerSpeedRequested || g.playerFontRequested || g.playerFullscreenRequested {
		x, y := ebiten.CursorPosition()
		q.err = fmt.Errorf("step %d: hd=%v locale=%s visible=%v ticks=%d pointer=%d,%d help=%v msg=%q loadErr=%q", q.index, g.modernStoryPortraits != nil, g.localeID, q.p.toolbarVisible(), q.p.toolbarTicks, x, y, g.helpVisible, g.msg, g.loadErr)
		return ebiten.Termination
	}
	if s.ui != "" && g.interfaceMode != s.ui || s.speed != 0 && g.playerSpeed != s.speed || s.fontScale != 0 && g.fontScale != s.fontScale || s.fullscreen != "" && ebiten.IsFullscreen() != (s.fullscreen == "on") {
		q.err = fmt.Errorf("step %d: UI=%s speed=%d font=%g fullscreen=%v; expected %+v", q.index, g.interfaceMode, g.playerSpeed, g.fontScale, ebiten.IsFullscreen(), s)
		return ebiten.Termination
	}
	if s.poster != "" && q.p.poster.settings.Set != s.poster ||
		s.posterMenu != "" && q.p.poster.menuOpen != (s.posterMenu == "on") ||
		s.posterEdge != nil && q.p.poster.settings.Edge != *s.posterEdge ||
		s.posterFade != nil && q.p.poster.settings.FadeMS != *s.posterFade {
		q.err = fmt.Errorf("step %d: poster=%+v menu=%v allowed=%v physical=%v; expected %+v", q.index, q.p.poster.settings, q.p.poster.menuOpen, g.playerPosterControlsAllowed(), q.inputTrace, s)
		return ebiten.Termination
	}
	q.completed = true // DrawFinalScreen checks and records before advancing.
	return nil
}
func (q *floatingToolbarProbe) Layout(w, h int) (int, int) { return q.p.Layout(w, h) }
func (q *floatingToolbarProbe) Draw(screen *ebiten.Image)  { q.p.Draw(screen) }
func (q *floatingToolbarProbe) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geometry ebiten.GeoM) {
	q.p.DrawFinalScreen(screen, offscreen, geometry)
	if offscreen.Bounds().Dx() != q.p.layoutWidth() || offscreen.Bounds().Dy() != logicalH+q.p.toolbarOffset() {
		return // Let Ebiten publish the new logical size before comparing it.
	}
	q.ready = true
	if !q.completed || q.err != nil {
		return
	}
	offset := q.p.toolbarOffset() * 3
	width, left := q.p.layoutWidth()*3, q.p.posterOffset()*3
	final, control := ebiten.NewImage(width, 1200+offset), ebiten.NewImage(1920, 1200)
	defer final.Dispose()
	defer control.Dispose()
	var m ebiten.GeoM
	m.Scale(3, 3)
	q.p.DrawFinalScreen(final, offscreen, m)
	base := &modernHDPresentation{Game: q.p.Game}
	controlSource := offscreen
	if offset > 0 || left > 0 {
		controlSource = q.p.gameSurface
	}
	base.DrawFinalScreen(control, controlSource, m)
	a, b := make([]byte, width*(1200+offset)*4), make([]byte, 1920*1200*4)
	final.ReadPixels(a)
	control.ReadPixels(b)
	unchanged, toolbarDiff, menuDiff := 0, false, false
	menu := image.Rect(playerPosterMenu.Min.X*3, playerPosterMenu.Min.Y*3-offset, playerPosterMenu.Max.X*3, playerPosterMenu.Max.Y*3-offset)
	for y := 0; y < 1200; y++ {
		for x := 0; x < 1920; x++ {
			ai, bi := ((y+offset)*width+left+x)*4, (y*1920+x)*4
			equal := bytes.Equal(a[ai:ai+4], b[bi:bi+4])
			if offset == 0 && q.p.toolbarVisible() && y < 72 {
				toolbarDiff = toolbarDiff || !equal
				continue
			}
			if q.p.poster.menuOpen && image.Pt(x, y).In(menu) {
				menuDiff = menuDiff || !equal
				continue
			}
			if !equal {
				q.err = fmt.Errorf("toolbar or poster changed game outside controls at %d,%d", x, y)
				return
			}
			unchanged += 4
		}
	}
	if offset == 0 && q.p.toolbarVisible() && !toolbarDiff || q.p.poster.menuOpen && !menuDiff {
		q.err = fmt.Errorf("visible toolbar or menu absent")
		return
	}
	file, err := os.Create(filepath.Join(q.out, fmt.Sprintf("floating-%02d.png", q.index)))
	if err == nil {
		err = png.Encode(file, &image.RGBA{Pix: a, Stride: width * 4, Rect: image.Rect(0, 0, width, 1200+offset)})
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
	}
	if err != nil {
		q.err = err
		return
	}
	q.checks = append(q.checks, map[string]any{"step": q.index, "input": q.steps[q.index].input, "locale": q.p.localeID, "hd": q.p.modernStoryPortraits != nil, "visible": q.p.toolbarVisible(), "interface_mode": q.p.interfaceMode, "speed": q.p.playerSpeed, "font_scale": q.p.fontScale, "fullscreen": ebiten.IsFullscreen(), "help": q.p.helpVisible, "unchanged_game_bytes": unchanged, "poster": q.p.poster.settings, "poster_menu": q.p.poster.menuOpen, "width": width, "backdrop": q.p.poster.backdropSource})
	q.index++
	if q.index < len(q.steps) {
		q.completed = false
		q.started = false
		q.inputDone = false
		q.pressObserved = false
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
	g.interfaceMode = playerInterfaceHD
	g.modernThemeCache = set
	g.playerSpeed = 1
	g.applyFontScale(1)
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
		{input: "Move=320,300@640x400:2000", locale: "zh-Hant", hd: true, visible: false, help: false},
		{input: "Move=320,1@640x400:400", locale: "zh-Hant", hd: true, visible: true, help: false},
		{input: "Click=106,12@640x400:400", locale: "zh-Hant", hd: true, visible: true, help: false},
		{input: "Click=43,12@640x400:400", locale: "zh-Hant", hd: false, visible: true, help: false},
		{input: "F2:400", locale: "zh-Hant", hd: true, visible: true, help: false},
		{input: "Click=288,12@640x400:400", locale: "zh-Hans", hd: true, visible: true, help: false},
		{input: "F4:400", locale: "ja", hd: true, visible: true, help: false},
		{input: "Click=288,12@640x400:400", locale: "en", hd: true, visible: true, help: false},
		{input: "Click=288,12@640x400:400", locale: "zh-Hant", hd: true, visible: true, help: false},
		{input: "F1:300", locale: "zh-Hant", hd: true, visible: true, help: true},
		{input: "Click=43,12@640x400:300", locale: "zh-Hant", hd: true, visible: true, help: true},
		{input: "Click=288,12@640x400:300", locale: "zh-Hant", hd: true, visible: true, help: true},
		{input: "F1:300", locale: "zh-Hant", hd: true, visible: true, help: false},
		{input: "Move=320,300@640x400:2000", locale: "zh-Hant", hd: true, visible: false, help: false},
		{input: "F2:400", locale: "zh-Hant", hd: false, visible: false, help: false},
		{input: "F2:400", locale: "zh-Hant", hd: true, visible: false, help: false},
		{input: "Move=320,1@640x400:200", locale: "zh-Hant", hd: true, visible: true, ui: "hd", speed: 1, fontScale: 1},
		{input: "Click=180,12@640x400:300", locale: "zh-Hant", hd: true, visible: true, ui: "original"},
		{input: "F11:300", locale: "zh-Hant", hd: true, visible: true, ui: "hd"},
		{input: "F2:300", locale: "zh-Hant", hd: false, visible: true, ui: "hd"},
		{input: "F11:300", locale: "zh-Hant", hd: false, visible: true, ui: "original"},
		{input: "F2:300", locale: "zh-Hant", hd: true, visible: true, ui: "original"},
		{input: "F11:300", locale: "zh-Hant", hd: true, visible: true, ui: "hd"},
		{input: "Click=375,12@640x400:300", locale: "zh-Hant", hd: true, visible: true, ui: "hd", fontScale: 1.1},
		{input: "F7:300", locale: "zh-Hant", hd: true, visible: true, ui: "hd", fontScale: 1.25},
		{input: "F7:300", locale: "zh-Hant", hd: true, visible: true, ui: "hd", fontScale: 0.8},
		{input: "F8:300", locale: "zh-Hant", hd: true, visible: true, ui: "hd", speed: 2},
		{input: "Click=440,12@640x400:300", locale: "zh-Hant", hd: true, visible: true, ui: "hd", speed: 3},
		{input: "F8:300", locale: "zh-Hant", hd: true, visible: true, ui: "hd", speed: 4},
		{input: "F8:300", locale: "zh-Hant", hd: true, visible: true, ui: "hd", speed: 1},
		{input: "Alt+Enter:300", locale: "zh-Hant", hd: true, visible: true, ui: "hd", fullscreen: "on"},
		{input: "Click=585,12@640x400:300", locale: "zh-Hant", hd: true, visible: true, ui: "hd", fullscreen: "off"},
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
