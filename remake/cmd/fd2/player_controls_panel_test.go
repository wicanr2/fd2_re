package main

import (
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
)

// controlsPanelFixture 開一個使用暫存使用者目錄的面板；不設定 activeInput。
func controlsPanelFixture(t *testing.T) *Game {
	t.Helper()
	previous := userDataDirCached
	userDataDirCached = t.TempDir()
	t.Cleanup(func() { userDataDirCached = previous })
	text, err := loadControlsLocaleFrom(filepath.Join("..", "..", "assets", "locales", "zh-Hant", "controls.json"), "zh-Hant")
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{input: newPlayerInput(defaultControlBindings(), controlsLoadDefault), localeID: "zh-Hant"}
	g.openControlsPanel()
	g.controls.text = text
	return g
}

func panelStep(g *Game, edit func(*controlsPanelInput)) {
	in := noControlsPanelInput()
	edit(&in)
	g.stepControlsPanel(in)
}

func savedControls(t *testing.T) controlsSettingsFile {
	t.Helper()
	raw, err := os.ReadFile(controlsSettingsPath())
	if err != nil {
		t.Fatal(err)
	}
	var file controlsSettingsFile
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	return file
}

func TestControlsLocaleFilesMatchContract(t *testing.T) {
	contract := controlsLocaleContract()
	for _, locale := range localeIDs {
		path := filepath.Join("..", "..", "assets", "locales", locale, "controls.json")
		c, err := loadControlsLocaleFrom(path, locale)
		if err != nil {
			t.Fatal(locale, err)
		}
		if len(c.entries) != len(contract) {
			t.Fatal(locale, "key count")
		}
		if got := c.text("controls.msg.conflict", "Z", "確認"); got == "controls.msg.conflict" || got == "" {
			t.Fatal(locale, "format", got)
		}
		if _, err := loadControlsLocaleFrom(path, "xx"); err == nil {
			t.Fatal("locale header mismatch accepted")
		}
		raw, _ := os.ReadFile(path)
		var file controlsLocaleFile
		if err := json.Unmarshal(raw, &file); err != nil {
			t.Fatal(err)
		}
		for name, mutate := range map[string]func(*controlsLocaleFile){
			"missing": func(f *controlsLocaleFile) { delete(f.Entries, "controls.title") },
			"extra": func(f *controlsLocaleFile) {
				f.Entries["controls.extra"] = controlsLocaleEntry{Text: "x", Variables: []string{}}
			},
			"variables": func(f *controlsLocaleFile) {
				f.Entries["controls.pad.connected"] = controlsLocaleEntry{Text: "%s", Variables: []string{"%s"}}
			},
			"empty": func(f *controlsLocaleFile) { f.Entries["controls.title"] = controlsLocaleEntry{Variables: []string{}} },
		} {
			copied := file
			copied.Entries = map[string]controlsLocaleEntry{}
			for k, v := range file.Entries {
				copied.Entries[k] = v
			}
			mutate(&copied)
			out, _ := json.Marshal(copied)
			bad := filepath.Join(t.TempDir(), "controls.json")
			os.WriteFile(bad, out, 0o644)
			if _, err := loadControlsLocaleFrom(bad, locale); err == nil {
				t.Fatal(locale, name, "accepted")
			}
		}
	}
	var missing *controlsLocale
	if missing.text("controls.title") != "controls.title" {
		t.Fatal("missing locale shows the key")
	}
}

func TestControlsPanelAssignConflictConfirmAndCancel(t *testing.T) {
	g := controlsPanelFixture(t)
	confirm, next := controlActionIndex("confirm"), controlActionIndex("next_unit")
	for g.controls.selected != confirm {
		panelStep(g, func(in *controlsPanelInput) { in.down = true })
	}
	panelStep(g, func(in *controlsPanelInput) { in.confirm = true })
	if !g.controls.capture || g.controls.message != g.controls.text.text("controls.msg.capture") {
		t.Fatal("Enter starts capture")
	}
	panelStep(g, func(in *controlsPanelInput) { in.captureKey = ebiten.KeyShift })
	if !g.controls.capture || g.controls.message != g.controls.text.text("controls.msg.reserved") {
		t.Fatal("modifier is reserved and capture continues")
	}
	panelStep(g, func(in *controlsPanelInput) { in.captureKey = ebiten.KeyZ })
	if g.controls.conflict == nil || g.input.bindings.Keyboard[confirm] != ebiten.KeyEnter {
		t.Fatal("duplicate binding must prompt before replacing")
	}
	if want := g.controls.text.text("controls.msg.conflict", "Z", "換下一位角色"); g.controls.message != want {
		t.Fatal("conflict message", g.controls.message)
	}
	panelStep(g, func(in *controlsPanelInput) { in.back = true })
	if g.controls.conflict != nil || g.input.bindings.Keyboard[confirm] != ebiten.KeyEnter || !g.controls.open {
		t.Fatal("cancel keeps the old binding")
	}
	if _, err := os.Stat(controlsSettingsPath()); err == nil {
		t.Fatal("cancel must not save")
	}
	panelStep(g, func(in *controlsPanelInput) { in.confirm = true })
	panelStep(g, func(in *controlsPanelInput) { in.captureKey = ebiten.KeyZ })
	panelStep(g, func(in *controlsPanelInput) { in.confirm = true })
	if g.input.bindings.Keyboard[confirm] != ebiten.KeyZ || g.input.bindings.Keyboard[next] != keyUnbound {
		t.Fatal("confirmed replacement")
	}
	if g.input.table.produce[ebiten.KeyZ] != ebiten.KeyEnter {
		t.Fatal("mapping table not rebuilt")
	}
	file := savedControls(t)
	if !reflect.DeepEqual(file.Keyboard, map[string]string{"confirm": "Z", "next_unit": ""}) {
		t.Fatal("saved immediately", file.Keyboard)
	}
	// 手把：X 改給確認，A 原本就是確認的 → 不衝突；A 指定給空白鍵 → 衝突、確認取代。
	space := controlActionIndex("space")
	g.controls.selected = space
	panelStep(g, func(in *controlsPanelInput) { in.confirm = true })
	panelStep(g, func(in *controlsPanelInput) { in.captureButton = padA })
	if g.controls.conflict == nil || !g.controls.conflict.gamepad {
		t.Fatal("gamepad conflict")
	}
	panelStep(g, func(in *controlsPanelInput) { in.click = true; in.point = image.Pt(5, 5) })
	if g.input.bindings.Gamepad[space] != padA || g.input.bindings.Gamepad[confirm] != padNone {
		t.Fatal("click confirms the replacement")
	}
	panelStep(g, func(in *controlsPanelInput) { in.confirm = true })
	panelStep(g, func(in *controlsPanelInput) { in.captureCancel = true })
	if g.controls.capture || g.input.bindings.Gamepad[space] != padA {
		t.Fatal("B/Esc cancels capture")
	}
	loaded, status := loadControlsSettings()
	if status != controlsLoadOK || !reflect.DeepEqual(loaded, g.input.bindings) {
		t.Fatal("reopen keeps the settings")
	}
}

func TestControlsPanelClearResetSliderAndClose(t *testing.T) {
	g := controlsPanelFixture(t)
	confirm, space := controlActionIndex("confirm"), controlActionIndex("space")
	g.controls.selected = space
	panelStep(g, func(in *controlsPanelInput) { in.clear = true })
	if g.input.bindings.Keyboard[space] != keyUnbound || g.input.bindings.Gamepad[space] != padNone {
		t.Fatal("Delete clears keyboard and gamepad")
	}
	panelStep(g, func(in *controlsPanelInput) { in.reset = true })
	if !reflect.DeepEqual(g.input.bindings, defaultControlBindings()) {
		t.Fatal("R restores the row")
	}
	// 空白鍵這列改成 Enter，確認那列恢復預設時要先提示。
	b := g.input.bindings.clone()
	b.setKeyboard(space, ebiten.KeyEnter)
	g.input.setBindings(b)
	g.controls.selected = confirm
	panelStep(g, func(in *controlsPanelInput) { in.reset = true })
	if g.controls.conflict == nil || !g.controls.conflict.reset || g.input.bindings.Keyboard[confirm] != keyUnbound {
		t.Fatal("reset conflict prompts first")
	}
	if want := g.controls.text.text("controls.msg.conflict", "Enter", g.controls.actionName(space)); g.controls.message != want {
		t.Fatal("reset conflict message", g.controls.message)
	}
	panelStep(g, func(in *controlsPanelInput) { in.confirm = true })
	if g.input.bindings.Keyboard[confirm] != ebiten.KeyEnter || g.input.bindings.Keyboard[space] != keyUnbound {
		t.Fatal("confirmed reset")
	}
	g.controls.selected = controlsSliderSelects - 1
	panelStep(g, func(in *controlsPanelInput) { in.down = true })
	if g.controls.selected != controlsSliderSelects {
		t.Fatal("slider row is after the last action")
	}
	panelStep(g, func(in *controlsPanelInput) { in.right = true })
	if g.input.bindings.Deadzone != 25 || savedControls(t).StickDeadzone != 25 {
		t.Fatal("slider right")
	}
	for i := 0; i < 20; i++ {
		panelStep(g, func(in *controlsPanelInput) { in.left = true })
	}
	if g.input.bindings.Deadzone != minStickDeadzone {
		t.Fatal("slider clamps", g.input.bindings.Deadzone)
	}
	panelStep(g, func(in *controlsPanelInput) {
		in.click = true
		in.point = image.Pt(controlsSliderTrack.Max.X, controlsSliderRow.Min.Y+3)
	})
	if g.input.bindings.Deadzone != maxStickDeadzone {
		t.Fatal("slider click", g.input.bindings.Deadzone)
	}
	panelStep(g, func(in *controlsPanelInput) { in.click = true; in.point = controlsResetAllRect.Min.Add(image.Pt(2, 2)) })
	if !reflect.DeepEqual(g.input.bindings, defaultControlBindings()) || g.controls.message != g.controls.text.text("controls.msg.reset_all") {
		t.Fatal("reset all")
	}
	// 滑鼠：點一列選取，再點同一列開始指定；滾輪捲動。
	g.controls.selected = 0
	g.controls.scroll = 0
	lineY := func(action int) int {
		return controlsListRect.Min.Y + (controlsLineOf(action)-g.controls.scroll)*controlsLineHeight + 5
	}
	panelStep(g, func(in *controlsPanelInput) { in.click = true; in.point = image.Pt(100, lineY(confirm)) })
	if g.controls.selected != confirm || g.controls.capture {
		t.Fatal("click selects")
	}
	panelStep(g, func(in *controlsPanelInput) { in.click = true; in.point = image.Pt(100, lineY(confirm)) })
	if !g.controls.capture {
		t.Fatal("second click starts capture")
	}
	panelStep(g, func(in *controlsPanelInput) { in.back = true })
	panelStep(g, func(in *controlsPanelInput) { in.wheel = -1 })
	panelStep(g, func(in *controlsPanelInput) { in.wheel = -1 })
	if g.controls.scroll != 2 {
		t.Fatal("wheel scroll", g.controls.scroll)
	}
	for i := 0; i < 40; i++ {
		panelStep(g, func(in *controlsPanelInput) { in.wheel = -1 })
	}
	if g.controls.scroll != len(controlsLines())-controlsVisibleLines {
		t.Fatal("scroll clamp", g.controls.scroll)
	}
	g.controls.selected = controlActionIndex("controls")
	g.controls.ensureVisible()
	line := controlsLineOf(g.controls.selected)
	if line < g.controls.scroll || line >= g.controls.scroll+controlsVisibleLines {
		t.Fatal("selected row visible")
	}
	panelStep(g, func(in *controlsPanelInput) { in.back = true })
	if g.controls.open {
		t.Fatal("Esc returns to the game")
	}
	g.openControlsPanel()
	g.controls.capture = true
	panelStep(g, func(in *controlsPanelInput) { in.close = true })
	if g.controls.open {
		t.Fatal("F12/Start closes from any state")
	}
}

func TestControlsPanelReportsCorruptSettings(t *testing.T) {
	g := controlsPanelFixture(t)
	g.input.status = controlsLoadCorrupt
	text := g.controls.text
	g.openControlsPanel()
	g.controls.text = text
	if g.controls.message != "controls.msg.corrupt" && g.controls.message != text.text("controls.msg.corrupt") {
		t.Fatal("corrupt notice", g.controls.message)
	}
}

func TestToolbarControlsButtonOpensPanelAndFits(t *testing.T) {
	controls := []image.Rectangle{playerThemeButtons[0], playerThemeButtons[1], playerInterfaceButton, playerLocaleButton,
		playerFontButton, playerSpeedButton, playerPosterButton, playerFullscreenButton, playerControlsButton, playerBalanceButton}
	for i, box := range controls {
		if !box.In(image.Rect(0, 0, 640, 24)) {
			t.Fatal("control outside one row", box)
		}
		for _, previous := range controls[:i] {
			if box.Overlaps(previous) {
				t.Fatal("controls overlap", box, previous)
			}
		}
	}
	f := loadFont()
	if f == nil {
		t.Fatal("Noto required")
	}
	for box, labels := range map[image.Rectangle][]string{
		playerControlsButton:   {"按鍵", "按键", "キー", "Keys"},
		playerBalanceButton:    {"能力"},
		playerFullscreenButton: {"全螢幕", "全屏", "全画面", "Full"},
	} {
		for _, label := range labels {
			face, caption, at := playerToolbarLabelLayout(f, label, box)
			if face == nil {
				t.Fatal("missing font")
			}
			if caption != label {
				t.Fatal("toolbar label truncated", label, caption, box)
			}
			if ink := text.BoundString(face, caption).Add(at); !ink.In(image.Rect(box.Min.X*3+12, box.Min.Y*3+6, box.Max.X*3-12, box.Max.Y*3-6)) {
				t.Fatal("toolbar text escaped", label, box)
			}
		}
	}
	previous := userDataDirCached
	userDataDirCached = t.TempDir()
	t.Cleanup(func() { userDataDirCached = previous })
	g := &Game{}
	p := &modernHDPresentation{Game: g}
	p.queueToolbarPoint(image.Pt((playerControlsButton.Min.X+playerControlsButton.Max.X)/2, 12))
	if !g.controls.open || !p.poster.consumeInput || g.balanceUI.open || g.playerFullscreenRequested {
		t.Fatal("controls button")
	}
}
