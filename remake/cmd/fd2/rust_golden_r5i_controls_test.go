package main

// rust_golden_r5i_controls_test.go — R5 階段 4 r5i（Rust 滑鼠、手把與按鍵設定面板）的局部黃金。只輸出資料，
// 未設 FD2_GOLDEN_R5I_OUT 時略過；由 tools/rust_r5i_controls.sh 執行，Rust 端
// remake-rs/crates/fd2-game/src/game/player_controls_panel/rust_golden_r5i_test.rs 讀同一份輸出比對。
//
// 輸出（<out>/）：
//   keys.json      ebiten.Key 的數值與 String()，以及一串名稱經 UnmarshalText 的結果
//   settings.json  隨機綁定寫出的 controls-settings.json 位元組、讀回結果，與手寫檔案的讀檔結果
//   input.json     隨機的滑鼠／手把快照序列經 playerInput.step 後，每一步的指向與 109 個鍵的 duration
//   pointer.json   指向換算（原生／舊版地圖格、指令環方向、標題選單、讀檔四格、城鎮、選項框）
//   panel.json     面板操作序列（直接餵 stepControlsPanel）每一步之後的狀態；擷取點的矩形層雜湊、text.Draw 清單
//                  與整張疊層雜湊，整張疊層另存 panel-<n>.rgba（1920×1200 預乘 RGBA）
//
// 面板的疊層：正式的 drawControlsPanel 畫進 p.controlsSurface 後讀回（整張）；r5iDrawControlsPanelSurface 是同一函式
// 本體的複本，只把 label 閉包的 text.Draw 改成「記錄後視 drawText 決定畫不畫」。每個擷取點都核對「複本畫字」與正式
// 函式的疊層逐位元組相同，所以複本的矩形層與文字清單就是正式函式的。字型是 loadFont() 的 Noto Sans CJK，
// 拿掉 hdFonts（Rust R5 沒有區域襯線字型，見 PORTING.md「r5i」KNOWN_DIFFERENCES）。

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func r5iWriteJSON(t *testing.T, path string, v any) {
	t.Helper()
	raw, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func r5iSHA(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

type r5iBindings struct {
	Keyboard []int `json:"keyboard"`
	Gamepad  []int `json:"gamepad"`
	Deadzone int   `json:"deadzone"`
}

func r5iFromBindings(b controlBindings) r5iBindings {
	out := r5iBindings{Deadzone: b.Deadzone}
	for _, k := range b.Keyboard {
		out.Keyboard = append(out.Keyboard, int(k))
	}
	for _, p := range b.Gamepad {
		out.Gamepad = append(out.Gamepad, int(p))
	}
	return out
}

func r5iToBindings(b r5iBindings) controlBindings {
	out := controlBindings{Deadzone: b.Deadzone}
	for _, k := range b.Keyboard {
		out.Keyboard = append(out.Keyboard, ebiten.Key(k))
	}
	for _, p := range b.Gamepad {
		out.Gamepad = append(out.Gamepad, controlPadButton(p))
	}
	return out
}

// r5iRandomBindings 從預設綁定出發做幾次隨機操作（指定、清除、恢復、防飄移）。
func r5iRandomBindings(rng *rand.Rand) controlBindings {
	b := defaultControlBindings()
	for n := rng.Intn(6); n > 0; n-- {
		action := rng.Intn(len(controlActions))
		switch rng.Intn(5) {
		case 0, 1:
			if k := ebiten.Key(rng.Intn(int(ebiten.KeyMax) + 1)); bindableKey(k) {
				b.setKeyboard(action, k)
			}
		case 2:
			b.setGamepad(action, controlPadButton(rng.Intn(int(padButtonCount))))
		case 3:
			b.clearAction(action)
		default:
			b.resetAction(action)
		}
	}
	if rng.Intn(3) == 0 {
		b.Deadzone = minStickDeadzone + stickDeadzoneStep*rng.Intn(12)
	}
	return b
}

// ---- keys ----

func r5iGoldenKeys(t *testing.T, out string) {
	type keyName struct {
		Code int    `json:"code"`
		Name string `json:"name"`
	}
	type parsed struct {
		Name string `json:"name"`
		Code int    `json:"code"` // -2：UnmarshalText 失敗
	}
	var keys []keyName
	names := []string{"return", "NoSuchKey", "", "Ｚ", "KP5", "Kp5", "graveAccent", "LEFTBRACKET", "Menu", "f13", "key0"}
	for k := ebiten.Key(0); k <= ebiten.KeyMax; k++ {
		keys = append(keys, keyName{Code: int(k), Name: k.String()})
		names = append(names, k.String(), strings.ToLower(k.String()), strings.ToUpper(k.String()))
	}
	for _, alias := range []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "apostrophe", "down", "graveaccent",
		"kp0", "kp1", "kp2", "kp3", "kp4", "kp5", "kp6", "kp7", "kp8", "kp9", "kpadd", "kpdecimal", "kpdivide", "kpenter",
		"kpequal", "kpmultiply", "kpsubtract", "left", "leftbracket", "menu", "right", "rightbracket", "up"} {
		names = append(names, alias)
	}
	var results []parsed
	for _, name := range names {
		var k ebiten.Key
		code := -2
		if k.UnmarshalText([]byte(name)) == nil {
			code = int(k)
		}
		results = append(results, parsed{Name: name, Code: code})
	}
	labels := map[string]string{}
	for k := ebiten.Key(-1); k <= ebiten.KeyMax; k++ {
		labels[fmt.Sprint(int(k))] = controlKeyLabel(k)
	}
	r5iWriteJSON(t, filepath.Join(out, "keys.json"), map[string]any{"keys": keys, "parsed": results, "labels": labels})
}

// ---- settings ----

func r5iGoldenSettings(t *testing.T, out string) {
	type savedCase struct {
		Bindings r5iBindings `json:"bindings"`
		File     string      `json:"file"`
		Loaded   r5iBindings `json:"loaded"`
		Status   int         `json:"status"`
	}
	type loadCase struct {
		File   string      `json:"file"`
		Loaded r5iBindings `json:"loaded"`
		Status int         `json:"status"`
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "controls-settings.json")
	rng := rand.New(rand.NewSource(11))
	var saved []savedCase
	for i := 0; i < 300; i++ {
		b := r5iRandomBindings(rng)
		if err := saveControlsSettingsTo(path, b); err != nil {
			t.Fatal(err)
		}
		raw, _ := os.ReadFile(path)
		loaded, status := loadControlsSettingsFrom(path)
		saved = append(saved, savedCase{Bindings: r5iFromBindings(b), File: string(raw), Loaded: r5iFromBindings(loaded), Status: int(status)})
	}
	files := []string{"{", "", "[]", "null", `{"schema_version":2}`, `{"schema_version":1}`, `{"schema_version":1,"stick_deadzone_percent":0}`,
		`{"schema_version":1,"stick_deadzone_percent":37}`, `{"schema_version":1,"stick_deadzone_percent":3}`,
		`{"schema_version":1,"keyboard":{"confirm":"enter","cancel":"Escape","space":"kp5","back":"Digit1","next_unit":"1"}}`,
		`{"schema_version":1,"keyboard":{"confirm":"Escape"}}`, `{"schema_version":1,"keyboard":{"cancel":"ShiftLeft"}}`,
		`{"schema_version":1,"keyboard":{"up":"W","down":"W"}}`, `{"schema_version":1,"keyboard":{"confirm":"Z"},"gamepad":{"confirm":"","space":"a","x":"b"}}`,
		`{"schema_version":1,"gamepad":{"up":"dpad_down","down":"dpad_up","help":"lt"},"stick_deadzone_percent":60}`,
		`{"schema_version":1,"keyboard":null,"gamepad":null}`, `{"schema_version":1,"keyboard":{"confirm":"NoSuchKey","cancel":"Shift","ghost":"Z","space":"Q"},"gamepad":{"confirm":"turbo"},"stick_deadzone_percent":99}`,
		`{"schema_version":1,"keyboard":{"confirm":""},"gamepad":{"cancel":""},"extra":[1,2]}`}
	var loads []loadCase
	for _, file := range files {
		if err := os.WriteFile(path, []byte(file), 0o644); err != nil {
			t.Fatal(err)
		}
		loaded, status := loadControlsSettingsFrom(path)
		loads = append(loads, loadCase{File: file, Loaded: r5iFromBindings(loaded), Status: int(status)})
	}
	r5iWriteJSON(t, filepath.Join(out, "settings.json"), map[string]any{"saved": saved, "loads": loads})
}

// ---- input ----

type r5iPad struct {
	Name    string               `json:"name"`
	Buttons [padButtonCount]bool `json:"buttons"`
	StickX  float64              `json:"stick_x"`
	StickY  float64              `json:"stick_y"`
}

type r5iSnapshot struct {
	Focused   bool     `json:"focused"`
	CursorX   int      `json:"cursor_x"`
	CursorY   int      `json:"cursor_y"`
	Left      bool     `json:"left"`
	Right     bool     `json:"right"`
	LeftJust  bool     `json:"left_just"`
	RightJust bool     `json:"right_just"`
	WheelY    float64  `json:"wheel_y"`
	Pads      []r5iPad `json:"pads"`
}

func (s r5iSnapshot) snapshot() inputDeviceSnapshot {
	snap := inputDeviceSnapshot{focused: s.Focused, cursorX: s.CursorX, cursorY: s.CursorY, left: s.Left, right: s.Right,
		leftJust: s.LeftJust, rightJust: s.RightJust, wheelY: s.WheelY}
	for _, p := range s.Pads {
		snap.pads = append(snap.pads, gamepadSnapshot{name: p.Name, buttons: p.Buttons, stickX: p.StickX, stickY: p.StickY})
	}
	return snap
}

func r5iGoldenInput(t *testing.T, out string) {
	type step struct {
		Snapshot    r5iSnapshot  `json:"snapshot"`
		GameInput   bool         `json:"game_input"`
		SetBindings *r5iBindings `json:"set_bindings,omitempty"`
		Suppress    []int        `json:"suppress,omitempty"`
		Pointer     [4]int       `json:"pointer"` // active, clicked, x, y
		Durations   map[int]int  `json:"durations"`
		JustPressed []int        `json:"just_pressed"`
		PadCount    int          `json:"pad_count"`
		PadLatched  bool         `json:"pad_latched"`
	}
	type sequence struct {
		Seed     int         `json:"seed"`
		Bindings r5iBindings `json:"bindings"`
		Steps    []step      `json:"steps"`
	}
	var sequences []sequence
	for seed := 1; seed <= 6; seed++ {
		rng := rand.New(rand.NewSource(int64(seed)))
		b := defaultControlBindings()
		if seed > 2 {
			b = r5iRandomBindings(rng)
		}
		in := newPlayerInput(b, controlsLoadOK)
		seq := sequence{Seed: seed, Bindings: r5iFromBindings(in.bindings)}
		cur := r5iSnapshot{Focused: true, CursorX: 300, CursorY: 200}
		padCount := 0
		for i := 0; i < 600; i++ {
			s := cur
			s.Pads = nil
			s.Focused = rng.Intn(20) != 0
			if rng.Intn(3) == 0 {
				s.CursorX, s.CursorY = rng.Intn(680)-20, rng.Intn(440)-20
			}
			left, right := rng.Intn(3) == 0, rng.Intn(6) == 0
			s.LeftJust, s.RightJust = left && !cur.Left, right && !cur.Right
			s.Left, s.Right = left, right
			s.WheelY = []float64{0, 0, 0, 0, 1, -1, 0.5, -0.5, 2, -0.25}[rng.Intn(10)]
			if rng.Intn(40) == 0 {
				padCount = rng.Intn(3)
			}
			for p := 0; p < padCount; p++ {
				pad := r5iPad{Name: fmt.Sprintf("pad%d", p)}
				for button := range pad.Buttons {
					pad.Buttons[button] = rng.Intn(8) == 0
				}
				if rng.Intn(2) == 0 {
					pad.StickX, pad.StickY = rng.Float64()*2-1, rng.Float64()*2-1
				}
				s.Pads = append(s.Pads, pad)
			}
			st := step{Snapshot: s, GameInput: rng.Intn(7) != 0}
			if rng.Intn(150) == 0 {
				nb := r5iFromBindings(r5iRandomBindings(rng))
				st.SetBindings = &nb
				in.setBindings(r5iToBindings(nb))
			}
			ptr := in.step(s.snapshot(), st.GameInput, acceptBelowToolbar)
			if rng.Intn(25) == 0 {
				for _, k := range []ebiten.Key{ebiten.KeyEnter, ebiten.KeyArrowUp, ebiten.KeySpace} {
					if rng.Intn(2) == 0 {
						in.suppressVirtual(k)
						st.Suppress = append(st.Suppress, int(k))
					}
				}
			}
			st.Pointer = [4]int{r5iBool(ptr.active), r5iBool(ptr.clicked), ptr.x, ptr.y}
			st.Durations = map[int]int{}
			for k := ebiten.Key(0); k <= ebiten.KeyMax; k++ {
				if d := in.duration(k); d != 0 {
					st.Durations[int(k)] = d
				}
			}
			st.JustPressed = []int{}
			for _, k := range in.appendJustPressed(nil) {
				st.JustPressed = append(st.JustPressed, int(k))
			}
			st.PadCount, st.PadLatched = in.padCount, in.padLatched
			seq.Steps = append(seq.Steps, st)
			cur = s
		}
		sequences = append(sequences, seq)
	}
	r5iWriteJSON(t, filepath.Join(out, "input.json"), map[string]any{"sequences": sequences})
}

func r5iBool(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ---- pointer ----

func r5iGoldenPointer(t *testing.T, out string) {
	type cell struct {
		Case string `json:"case"`
		X    int    `json:"x"`
		Y    int    `json:"y"`
		Out  [3]int `json:"out"`
	}
	var cells []cell
	for _, camera := range [][2]int{{0, 0}, {3, 5}, {27, 22}} {
		for _, visible := range [][2]int{{6, 4}, {0, 0}, {12, 7}} {
			g := nativePointerGame(camera[0], camera[1])
			g.st.NativeMapViewState.VisibleCursorX, g.st.NativeMapViewState.VisibleCursorY = visible[0], visible[1]
			name := fmt.Sprintf("native cam=%v vis=%v", camera, visible)
			for x := -20; x < 660; x += 7 {
				for y := -20; y < 420; y += 7 {
					g.pointer = pointerTick{active: true, x: x, y: y}
					cx, cy, ok := g.pointerMapCell()
					dir, dok := g.pointerOverlayDirection()
					cells = append(cells, cell{Case: name, X: x, Y: y, Out: [3]int{cx*1000 + cy, r5iBool(ok), dir*10 + r5iBool(dok)}})
				}
			}
		}
	}
	for _, legacy := range []struct {
		camX, camY   float64
		tileW, tileH int
	}{{0, 0, 48, 48}, {96.7, 48.2, 48, 48}, {-30.5, 10, 24, 32}} {
		g := &Game{m: &MapData{W: 20, H: 20, TileW: legacy.tileW, TileH: legacy.tileH}, camX: legacy.camX, camY: legacy.camY}
		name := fmt.Sprintf("legacy %v", legacy)
		for x := -20; x < 660; x += 11 {
			for y := -20; y < 420; y += 11 {
				g.pointer = pointerTick{active: true, x: x, y: y}
				cx, cy, ok := g.pointerMapCell()
				g.curX, g.curY = 3, 4
				g.followPointerOnMap()
				cells = append(cells, cell{Case: name, X: x, Y: y, Out: [3]int{cx*1000 + cy, r5iBool(ok), g.curX*1000 + g.curY}})
			}
		}
	}
	type rects struct {
		Case  string   `json:"case"`
		Rects [][4]int `json:"rects"`
	}
	toRects := func(name string, rs []image.Rectangle) rects {
		out := rects{Case: name, Rects: [][4]int{}}
		for _, r := range rs {
			out.Rects = append(out.Rects, [4]int{r.Min.X, r.Min.Y, r.Max.X, r.Max.Y})
		}
		return out
	}
	var allRects []rects
	type menu struct {
		Case   string `json:"case"`
		Start  int    `json:"start"`
		X      int    `json:"x"`
		Y      int    `json:"y"`
		Result int    `json:"result"`
	}
	var menus []menu
	for _, w := range []int{40, 57, 64, 101} {
		g := &Game{titlePhase: "menu", titleAssets: &titleAssets{}}
		for i := range g.titleAssets.items {
			g.titleAssets.items[i][0] = ebiten.NewImage(w+i*3, 8)
		}
		allRects = append(allRects, toRects(fmt.Sprintf("title w=%d", w), g.titleMenuRects()))
		for start := 0; start < 3; start++ {
			for x := 150; x < 500; x += 13 {
				for y := 320; y < 400; y += 3 {
					g.titleSel, g.titleFlash = start, 0
					g.pointer = pointerTick{active: true, x: x, y: y}
					g.applyPointerToTitleMenu()
					menus = append(menus, menu{Case: fmt.Sprintf("title w=%d", w), Start: start, X: x, Y: y, Result: g.titleSel})
				}
			}
		}
	}
	allRects = append(allRects, toRects("slots", titleSlotRects()))
	for variant := 0; variant < 3; variant++ {
		rs := nativeTownSelectionRects(variant)
		allRects = append(allRects, toRects(fmt.Sprintf("town %d", variant), rs))
		for start := 0; start < 5; start++ {
			for x := 0; x < 640; x += 9 {
				for y := 0; y < 400; y += 9 {
					g := &Game{campSel: start}
					g.pointer = pointerTick{active: true, x: x, y: y}
					g.applyPointerToNativeTown(variant)
					menus = append(menus, menu{Case: fmt.Sprintf("town %d", variant), Start: start, X: x, Y: y, Result: g.campSel})
				}
			}
		}
	}
	for count := 0; count <= 6; count++ {
		allRects = append(allRects, toRects(fmt.Sprintf("choice %d", count), choiceMenuRects(count)))
	}
	r5iWriteJSON(t, filepath.Join(out, "pointer.json"), map[string]any{"cells": cells, "rects": allRects, "menus": menus})
}

// ---- panel ----

type r5iPanelInput struct {
	Up, Down, Left, Right, PageUp, PageDown bool
	Confirm, Back, Clear, Reset, Close      bool
	CaptureKey                              int
	CaptureButton                           int
	CaptureCancel                           bool
	Click, Drag                             bool
	Point                                   [2]int
	Wheel                                   int
}

func (in r5iPanelInput) input() controlsPanelInput {
	return controlsPanelInput{up: in.Up, down: in.Down, left: in.Left, right: in.Right, pageUp: in.PageUp, pageDown: in.PageDown,
		confirm: in.Confirm, back: in.Back, clear: in.Clear, reset: in.Reset, close: in.Close,
		captureKey: ebiten.Key(in.CaptureKey), captureButton: controlPadButton(in.CaptureButton), captureCancel: in.CaptureCancel,
		click: in.Click, drag: in.Drag, point: image.Pt(in.Point[0], in.Point[1]), wheel: in.Wheel}
}

type r5iTextDraw struct {
	Text  string `json:"text"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Size  int    `json:"size"`
	Color [4]int `json:"color"`
	Bound [4]int `json:"bound"`
}

// r5iDrawControlsPanelSurface 是 drawControlsPanel（player_controls_panel.go）的本體複本：只畫到 s（不疊到畫面），
// label 閉包的 text.Draw 改成記錄＋視 drawText 決定。其餘逐行相同；正確性由呼叫端核對「drawText=true 的結果與正式
// 函式的 controlsSurface 逐位元組相同」。
func r5iDrawControlsPanelSurface(p *modernHDPresentation, s *ebiten.Image, rec *[]r5iTextDraw, drawText bool) {
	g := p.Game
	c := &g.controls
	if !c.open {
		return
	}
	s.Fill(color.RGBA{0, 0, 0, 170})
	rect := func(r image.Rectangle, fill color.RGBA) {
		vector.DrawFilledRect(s, float32(r.Min.X*3), float32(r.Min.Y*3), float32(r.Dx()*3), float32(r.Dy()*3), fill, false)
	}
	frame := func(r image.Rectangle, fill, line color.RGBA) {
		rect(r, line)
		vector.DrawFilledRect(s, float32(r.Min.X*3+2), float32(r.Min.Y*3+2), float32(r.Dx()*3-4), float32(r.Dy()*3-4), fill, false)
	}
	f := g.font
	if f != nil {
		if regional := f.hdFonts[g.localeID]; regional != nil {
			f = regional
		}
	}
	label := func(caption string, x, baseline, size int, ink color.RGBA) int {
		if f == nil || caption == "" {
			return 0
		}
		face, _ := f.faceFor(size)
		if face == nil {
			return 0
		}
		b := text.BoundString(face, caption)
		*rec = append(*rec, r5iTextDraw{Text: caption, X: x * 3, Y: baseline * 3, Size: size,
			Color: [4]int{int(ink.R), int(ink.G), int(ink.B), int(ink.A)}, Bound: [4]int{b.Min.X, b.Min.Y, b.Max.X, b.Max.Y}})
		if drawText {
			text.Draw(s, caption, face, x*3, baseline*3, ink)
		}
		return text.BoundString(face, caption).Dx() / 3
	}
	measure := func(caption string, size int) int {
		if f == nil || caption == "" {
			return 0
		}
		face, _ := f.faceFor(size)
		if face == nil {
			return 0
		}
		return (text.BoundString(face, caption).Dx() + 2) / 3
	}
	chip := func(caption string, x, top int, ink color.RGBA) int {
		w := measure(caption, 33) + 12
		frame(image.Rect(x, top, x+w, top+16), controlsColorChip, controlsColorChipLine)
		label(caption, x+6, top+12, 33, ink)
		return w
	}

	frame(controlsPanelRect, controlsColorPanel, controlsColorBorder)
	label(c.text.text("controls.title"), 30, 36, 54, controlsColorGold)
	label(c.text.text("controls.subtitle"), 30, 54, 33, controlsColorMuted)
	padCount := 0
	if g.input != nil {
		padCount = g.input.padCount
	}
	status := c.text.text("controls.pad.disconnected")
	if padCount > 0 {
		status = c.text.text("controls.pad.connected", padCount)
	}
	frame(controlsPadChip, controlsColorChip, controlsColorChipLine)
	label(status, controlsPadChip.Min.X+8, controlsPadChip.Min.Y+16, 33, controlsColorText)

	rect(controlsHeaderBand, controlsColorBand)
	label(c.text.text("controls.column.action"), 36, 77, 36, controlsColorMuted)
	label(c.text.text("controls.column.keyboard"), controlsColumnKeys, 77, 36, controlsColorMuted)
	label(c.text.text("controls.column.gamepad"), controlsColumnPad, 77, 36, controlsColorMuted)

	bindings := defaultControlBindings()
	if g.input != nil {
		bindings = g.input.bindings
	}
	lines := controlsLines()
	for row := 0; row < controlsVisibleLines && c.scroll+row < len(lines); row++ {
		line := lines[c.scroll+row]
		top := controlsListRect.Min.Y + row*controlsLineHeight
		band := image.Rect(controlsListRect.Min.X, top, controlsListRect.Max.X-8, top+controlsLineHeight)
		if line.action < 0 {
			rect(band, controlsColorGroup)
			label(c.text.text(controlsGroupKeys[line.group]), 36, top+15, 33, controlsColorGold)
			continue
		}
		if line.action == c.selected {
			rect(band, controlsColorSelected)
			rect(image.Rect(band.Min.X, band.Min.Y, band.Min.X+2, band.Max.Y), controlsColorGold)
		}
		rect(image.Rect(band.Min.X, band.Max.Y-1, band.Max.X, band.Max.Y), controlsColorGroup)
		label(c.actionName(line.action), 40, top+15, 39, controlsColorText)
		if k := bindings.Keyboard[line.action]; k != keyUnbound {
			chip(controlKeyLabel(k), controlsColumnKeys, top+2, controlsColorText)
		} else {
			label(c.text.text("controls.unbound"), controlsColumnKeys, top+15, 33, controlsColorMuted)
		}
		x := controlsColumnPad
		if button := bindings.Gamepad[line.action]; button != padNone {
			x += chip(c.padLabel(button), x, top+2, controlsColorText) + 4
		}
		if controlActions[line.action].Group == controlGroupMovement {
			arrow := controlKeyLabel(controlActions[line.action].Logical)
			chip(c.text.text("controls.stick")+arrow, x, top+2, controlsColorMuted)
		} else if bindings.Gamepad[line.action] == padNone {
			label(c.text.text("controls.unbound"), x, top+15, 33, controlsColorMuted)
		}
	}
	// 捲軸
	total := len(lines)
	track := image.Rect(controlsListRect.Max.X-6, controlsListRect.Min.Y+2, controlsListRect.Max.X-3, controlsListRect.Max.Y-2)
	rect(track, controlsColorGroup)
	thumbH := track.Dy() * controlsVisibleLines / total
	thumbY := track.Min.Y + (track.Dy()-thumbH)*c.scroll/max(1, total-controlsVisibleLines)
	rect(image.Rect(track.Min.X, thumbY, track.Max.X, thumbY+thumbH), controlsColorChipLine)

	if c.selected == controlsSliderSelects {
		rect(controlsSliderRow, controlsColorSelected)
		rect(image.Rect(controlsSliderRow.Min.X, controlsSliderRow.Min.Y, controlsSliderRow.Min.X+2, controlsSliderRow.Max.Y), controlsColorGold)
	} else {
		rect(controlsSliderRow, controlsColorBand)
	}
	label(c.text.text("controls.deadzone"), 36, 304, 39, controlsColorText)
	rect(controlsSliderTrack, controlsColorChipLine)
	ratio := float64(bindings.Deadzone-minStickDeadzone) / float64(maxStickDeadzone-minStickDeadzone)
	fill := controlsSliderTrack
	fill.Max.X = fill.Min.X + int(ratio*float64(controlsSliderTrack.Dx()))
	rect(fill, controlsColorGold)
	knobX := fill.Max.X
	rect(image.Rect(knobX-4, controlsSliderTrack.Min.Y-5, knobX+4, controlsSliderTrack.Max.Y+5), controlsColorGold)
	label(fmt.Sprintf("%d%%", bindings.Deadzone), controlsSliderTrack.Max.X+14, 304, 39, controlsColorGold)

	label(c.message, 30, 330, 33, controlsColorGold)
	label(c.text.text("controls.help.keyboard"), 30, 352, 30, controlsColorMuted)
	label(c.text.text("controls.help.gamepad"), 30, 370, 30, controlsColorMuted)
	frame(controlsResetAllRect, controlsColorChip, controlsColorChipLine)
	resetLabel := c.text.text("controls.reset_all")
	label(resetLabel, controlsResetAllRect.Min.X+(controlsResetAllRect.Dx()-measure(resetLabel, 36))/2, controlsResetAllRect.Min.Y+18, 36, controlsColorText)
}

// r5iPanelScript 是面板的操作序列：手寫的情境（指定、衝突、清除、恢復、滑桿、點選、捲動、關閉再開）加上隨機操作。
func r5iPanelScript(rng *rand.Rand) []r5iPanelInput {
	none := func() r5iPanelInput { return r5iPanelInput{CaptureKey: int(keyUnbound), CaptureButton: int(padNone)} }
	var steps []r5iPanelInput
	add := func(edit func(*r5iPanelInput)) {
		in := none()
		edit(&in)
		steps = append(steps, in)
	}
	for i := 0; i < 4; i++ {
		add(func(in *r5iPanelInput) { in.Down = true })
	}
	add(func(in *r5iPanelInput) { in.Confirm = true })
	add(func(in *r5iPanelInput) { in.CaptureKey = int(ebiten.KeyShift) })
	add(func(in *r5iPanelInput) { in.CaptureKey = int(ebiten.KeyZ) })
	add(func(in *r5iPanelInput) { in.Confirm = true })
	add(func(in *r5iPanelInput) { in.Down = true })
	add(func(in *r5iPanelInput) { in.Down = true })
	add(func(in *r5iPanelInput) { in.Confirm = true })
	add(func(in *r5iPanelInput) { in.CaptureButton = int(padA) })
	add(func(in *r5iPanelInput) { in.Back = true })
	add(func(in *r5iPanelInput) { in.Clear = true })
	add(func(in *r5iPanelInput) { in.PageDown = true })
	add(func(in *r5iPanelInput) { in.PageDown = true })
	add(func(in *r5iPanelInput) { in.PageDown = true })
	add(func(in *r5iPanelInput) { in.Right = true })
	add(func(in *r5iPanelInput) { in.Right = true })
	add(func(in *r5iPanelInput) { in.Click = true; in.Point = [2]int{300, 299} })
	add(func(in *r5iPanelInput) { in.Drag = true; in.Point = [2]int{400, 299} })
	add(func(in *r5iPanelInput) { in.Wheel = 1 })
	add(func(in *r5iPanelInput) { in.Wheel = 1 })
	add(func(in *r5iPanelInput) { in.Click = true; in.Point = [2]int{100, 82 + 20*3 + 5} })
	add(func(in *r5iPanelInput) { in.Click = true; in.Point = [2]int{100, 82 + 20*3 + 5} })
	add(func(in *r5iPanelInput) { in.CaptureKey = int(ebiten.KeyQ) })
	add(func(in *r5iPanelInput) { in.Reset = true })
	add(func(in *r5iPanelInput) { in.PageUp = true })
	add(func(in *r5iPanelInput) { in.Reset = true })
	add(func(in *r5iPanelInput) { in.Confirm = true })
	add(func(in *r5iPanelInput) { in.Click = true; in.Point = [2]int{500, 350} })
	add(func(in *r5iPanelInput) { in.Back = true })
	for i := 0; i < 400; i++ {
		add(func(in *r5iPanelInput) {
			switch rng.Intn(16) {
			case 0, 1:
				in.Up = true
			case 2, 3:
				in.Down = true
			case 4:
				in.PageUp, in.PageDown = rng.Intn(2) == 0, rng.Intn(2) == 0
			case 5:
				in.Left, in.Right = rng.Intn(2) == 0, rng.Intn(2) == 0
			case 6:
				in.Confirm = true
			case 7:
				in.CaptureKey = rng.Intn(int(ebiten.KeyMax)+2) - 1
			case 8:
				in.CaptureButton = rng.Intn(int(padButtonCount)+1) - 1
			case 9:
				in.Clear = rng.Intn(2) == 0
				in.Reset = !in.Clear
			case 10:
				in.Click, in.Point = true, [2]int{rng.Intn(660) - 10, rng.Intn(420) - 10}
			case 11:
				in.Drag, in.Point = true, [2]int{rng.Intn(660) - 10, rng.Intn(420) - 10}
			case 12:
				in.Wheel = rng.Intn(3) - 1
			case 13:
				in.CaptureCancel = true
			case 14:
				in.Back = rng.Intn(4) == 0
			default:
				in.Close = rng.Intn(10) == 0
			}
		})
	}
	return steps
}

type r5iPanelState struct {
	Open      bool        `json:"open"`
	Selected  int         `json:"selected"`
	Scroll    int         `json:"scroll"`
	Capture   bool        `json:"capture"`
	Conflict  []int       `json:"conflict"`
	Message   string      `json:"message"`
	Dragging  bool        `json:"dragging"`
	Bindings  r5iBindings `json:"bindings"`
	Status    int         `json:"status"`
	Saved     string      `json:"saved"`
	ShotIndex int         `json:"shot"` // -1：這一步之後沒有擷取
}

type r5iPanelShot struct {
	Locale    string        `json:"locale"`
	PadCount  int           `json:"pad_count"`
	RectSHA   string        `json:"rect_sha256"`
	FullSHA   string        `json:"full_sha256"`
	TextDraws []r5iTextDraw `json:"text_draws"`
	File      string        `json:"file"`
}

func r5iGoldenPanel(t *testing.T, out string, probe *goldenFrameProbe) {
	base := loadFont()
	if base == nil {
		t.Fatal("loadFont 失敗（需要 Noto Sans CJK）")
	}
	base.hdFonts = nil
	type run struct {
		Locale string          `json:"locale"`
		Steps  []r5iPanelInput `json:"steps"`
		States []r5iPanelState `json:"states"`
	}
	var runs []run
	var shots []r5iPanelShot
	storedPad := false
	previous := userDataDirCached
	defer func() { userDataDirCached = previous }()
	for li, locale := range localeIDs {
		userDataDirCached = t.TempDir()
		text, err := loadControlsLocaleFrom(filepath.Join("..", "..", "assets", "locales", locale, "controls.json"), locale)
		if err != nil {
			t.Fatal(err)
		}
		g := &Game{input: newPlayerInput(defaultControlBindings(), controlsLoadDefault), localeID: locale, font: base}
		g.openControlsPanel()
		g.controls.text = text
		p := &modernHDPresentation{Game: g}
		rng := rand.New(rand.NewSource(int64(21 + li)))
		steps := r5iPanelScript(rng)
		if locale != "zh-Hant" {
			steps = steps[:40]
		}
		r := run{Locale: locale, Steps: steps}
		storedLocale := false
		shoot := func() int {
			var rects, full []r5iTextDraw
			s := ebiten.NewImage(1920, 1200)
			pix := make([]byte, 1920*1200*4)
			var rectSHA, copySHA, fullSHA string
			var fullPix []byte
			probe.run(func() {
				r5iDrawControlsPanelSurface(p, s, &rects, false)
				s.ReadPixels(pix)
				rectSHA = r5iSHA(pix)
				r5iDrawControlsPanelSurface(p, s, &full, true)
				s.ReadPixels(pix)
				copySHA = r5iSHA(pix)
				target := ebiten.NewImage(logicalW, logicalH)
				p.drawControlsPanel(target, ebiten.GeoM{})
				p.controlsSurface.ReadPixels(pix)
				fullSHA = r5iSHA(pix)
				fullPix = append([]byte(nil), pix...)
			})
			if copySHA != fullSHA {
				t.Fatalf("r5iDrawControlsPanelSurface 與 drawControlsPanel 的疊層不同（複本已過期）：%s %s", copySHA, fullSHA)
			}
			if len(rects) != len(full) {
				t.Fatal("矩形層與整張的 text.Draw 次數不同")
			}
			// 整張疊層（9 MB）只存幾張：前三張、各語系第一張、第一張有手把連線的；其餘只比雜湊與文字清單。
			n := len(shots)
			file := ""
			if n < 3 || !storedLocale || g.input.padCount > 0 && !storedPad {
				file = fmt.Sprintf("panel-%03d.rgba", n)
				if err := os.WriteFile(filepath.Join(out, file), fullPix, 0o644); err != nil {
					t.Fatal(err)
				}
				storedLocale = true
				storedPad = storedPad || g.input.padCount > 0
			}
			shots = append(shots, r5iPanelShot{Locale: locale, PadCount: g.input.padCount, RectSHA: rectSHA, FullSHA: fullSHA, TextDraws: full, File: file})
			return n
		}
		state := func(shot int) r5iPanelState {
			c := g.controls
			st := r5iPanelState{Open: c.open, Selected: c.selected, Scroll: c.scroll, Capture: c.capture, Message: c.message,
				Dragging: c.dragging, Bindings: r5iFromBindings(g.input.bindings), Status: int(g.input.status), ShotIndex: shot}
			if c.conflict != nil {
				st.Conflict = []int{c.conflict.action, c.conflict.other, r5iBool(c.conflict.reset), r5iBool(c.conflict.gamepad),
					int(c.conflict.key), int(c.conflict.button)}
			}
			if raw, err := os.ReadFile(controlsSettingsPath()); err == nil {
				st.Saved = string(raw)
			}
			return st
		}
		for i, in := range steps {
			if !g.controls.open {
				// 關掉之後再開（F12），與正式路徑相同：openControlsPanel 重讀語系，測試再換成同一份檔案。
				g.openControlsPanel()
				g.controls.text = text
			}
			g.stepControlsPanel(in.input())
			shot := -1
			if g.controls.open && (locale == "zh-Hant" && (i < 33 || i%29 == 0) || locale != "zh-Hant" && i%8 == 0) {
				if i%37 == 5 {
					g.input.padCount = 1 + i%3
				}
				shot = shoot()
				g.input.padCount = 0
			}
			r.States = append(r.States, state(shot))
		}
		runs = append(runs, r)
	}
	r5iWriteJSON(t, filepath.Join(out, "panel.json"), map[string]any{"runs": runs, "shots": shots})
}

// TestRustGoldenR5iControls 輸出 r5i 的局部黃金（見檔頭）。只由 tools/rust_r5i_controls.sh 執行。
func TestRustGoldenR5iControls(t *testing.T) {
	out := os.Getenv("FD2_GOLDEN_R5I_OUT")
	if out == "" {
		t.Skip("需要 FD2_GOLDEN_R5I_OUT（R5 r5i 局部黃金，只由 tools/rust_r5i_controls.sh 執行）")
	}
	if activeInput != nil {
		t.Fatal("activeInput 必須是 nil")
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	r5iGoldenKeys(t, out)
	r5iGoldenSettings(t, out)
	r5iGoldenInput(t, out)
	r5iGoldenPointer(t, out)
	probe := &goldenFrameProbe{started: make(chan struct{}), stop: make(chan struct{}),
		work: make(chan func()), parked: make(chan struct{})}
	ebiten.SetVsyncEnabled(false)
	runErr := make(chan error, 1)
	go func() { runErr <- ebiten.RunGame(probe) }()
	select {
	case <-probe.started:
	case err := <-runErr:
		t.Fatalf("Ebiten 未啟動：%v", err)
	case <-time.After(60 * time.Second):
		t.Fatal("Ebiten 60 秒內未進入 Update")
	}
	defer func() {
		close(probe.stop)
		select {
		case <-runErr:
		case <-time.After(30 * time.Second):
			t.Error("Ebiten 未在 30 秒內結束")
		}
	}()
	r5iGoldenPanel(t, out, probe)
	// 檔案清單與雜湊（排序後），方便比對兩次執行。
	entries, _ := os.ReadDir(out)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	var manifest bytes.Buffer
	for _, name := range names {
		if name == "manifest.txt" {
			continue
		}
		raw, _ := os.ReadFile(filepath.Join(out, name))
		fmt.Fprintf(&manifest, "%s  %s\n", r5iSHA(raw), name)
	}
	if err := os.WriteFile(filepath.Join(out, "manifest.txt"), manifest.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}
