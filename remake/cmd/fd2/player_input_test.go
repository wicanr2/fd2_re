package main

import (
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// withActiveInput 讓測試暫時使用一個輸入對應層，結束時還原（其他測試維持 activeInput == nil）。
func withActiveInput(t *testing.T, b controlBindings) *playerInput {
	t.Helper()
	in := newPlayerInput(b, controlsLoadOK)
	previous := activeInput
	activeInput = in
	t.Cleanup(func() { activeInput = previous })
	return in
}

func focusedSnapshot() inputDeviceSnapshot { return inputDeviceSnapshot{focused: true} }

func padSnapshot(buttons ...controlPadButton) inputDeviceSnapshot {
	snap := focusedSnapshot()
	pad := gamepadSnapshot{name: "test pad"}
	for _, b := range buttons {
		pad.buttons[b] = true
	}
	snap.pads = []gamepadSnapshot{pad}
	return snap
}

func acceptBelowToolbar(x, y int) (int, int, bool) {
	if y < 24 || x < 0 || x >= logicalW || y >= logicalH {
		return 0, 0, false
	}
	return x, y, true
}

func TestControlKeyTableDefaultIsIdentity(t *testing.T) {
	table := newControlKeyTable(defaultControlBindings())
	for k := ebiten.Key(0); k <= ebiten.KeyMax; k++ {
		if table.produce[k] != k || !reflect.DeepEqual(table.physical[k], []ebiten.Key{k}) {
			t.Fatalf("default table is not identity at %v: produce=%v physical=%v", k, table.produce[k], table.physical[k])
		}
	}
	// 每個動作預設綁自己的邏輯鍵，鍵與手把各不重複。
	b := defaultControlBindings()
	for i, action := range controlActions {
		if b.Keyboard[i] != action.Logical || b.keyboardOwner(action.Logical) != i {
			t.Fatalf("%s default keyboard", action.ID)
		}
		if action.DefaultPad != padNone && b.gamepadOwner(action.DefaultPad) != i {
			t.Fatalf("%s default gamepad", action.ID)
		}
	}
	want := map[string]controlPadButton{"up": padUp, "down": padDown, "left": padLeft, "right": padRight,
		"confirm": padA, "cancel": padB, "space": padX, "controls": padStart}
	for i, action := range controlActions {
		expected, ok := want[action.ID]
		if !ok {
			expected = padNone
		}
		if b.Gamepad[i] != expected {
			t.Fatalf("%s default gamepad %v", action.ID, b.Gamepad[i])
		}
	}
	for _, unused := range []controlPadButton{padY, padLB, padRB, padLT, padRT, padBack} {
		if b.gamepadOwner(unused) >= 0 {
			t.Fatalf("shoulder/select button %v bound by default", unused)
		}
	}
}

func TestControlKeyTableRebindMovesLogicalKey(t *testing.T) {
	b := defaultControlBindings()
	confirm, next := controlActionIndex("confirm"), controlActionIndex("next_unit")
	if owner := b.keyboardOwner(ebiten.KeyZ); owner != next {
		t.Fatal("Z owner", owner)
	}
	b.setKeyboard(confirm, ebiten.KeyZ)
	if b.Keyboard[confirm] != ebiten.KeyZ || b.Keyboard[next] != keyUnbound {
		t.Fatal("conflict replacement did not unbind the previous owner", b.Keyboard[confirm], b.Keyboard[next])
	}
	table := newControlKeyTable(b)
	if table.produce[ebiten.KeyZ] != ebiten.KeyEnter || table.produce[ebiten.KeyEnter] != keyUnbound {
		t.Fatal("Z should produce Enter and physical Enter nothing", table.produce[ebiten.KeyZ], table.produce[ebiten.KeyEnter])
	}
	if len(table.physical[ebiten.KeyZ]) != 0 || !reflect.DeepEqual(table.physical[ebiten.KeyEnter], []ebiten.Key{ebiten.KeyZ}) {
		t.Fatal("reverse table", table.physical[ebiten.KeyZ], table.physical[ebiten.KeyEnter])
	}
	// 沒有指定給任何動作的鍵（WASD、Numpad5）維持原版操作。
	for _, k := range []ebiten.Key{ebiten.KeyW, ebiten.KeyA, ebiten.KeyNumpad5, ebiten.KeyShift} {
		if table.produce[k] != k {
			t.Fatal("unassigned key changed", k)
		}
	}
	b.setKeyboard(controlActionIndex("up"), ebiten.KeyW)
	if newControlKeyTable(b).produce[ebiten.KeyW] != ebiten.KeyArrowUp {
		t.Fatal("W assigned to up")
	}
}

func TestControlBindingsClearResetAndReservedKeys(t *testing.T) {
	b := defaultControlBindings()
	space, confirm := controlActionIndex("space"), controlActionIndex("confirm")
	b.clearAction(space)
	if b.Keyboard[space] != keyUnbound || b.Gamepad[space] != padNone {
		t.Fatal("clear")
	}
	if b.resetConflict(space) >= 0 {
		t.Fatal("reset of a cleared row has no conflict")
	}
	b.resetAction(space)
	if !reflect.DeepEqual(b, defaultControlBindings()) {
		t.Fatal("reset row did not restore defaults")
	}
	b.setGamepad(space, padA)
	if b.Gamepad[confirm] != padNone || b.Gamepad[space] != padA {
		t.Fatal("gamepad conflict replacement")
	}
	if b.resetConflict(confirm) != space {
		t.Fatal("reset conflict should name the row holding the default button")
	}
	b.resetAction(confirm)
	if b.Gamepad[confirm] != padA || b.Gamepad[space] != padNone {
		t.Fatal("reset takes the default back")
	}
	for _, k := range []ebiten.Key{ebiten.KeyShift, ebiten.KeyControlLeft, ebiten.KeyAltRight, ebiten.KeyMeta, ebiten.KeyEscape, keyUnbound, ebiten.KeyMax + 1} {
		if bindableKey(k) {
			t.Fatal("reserved key bindable", k)
		}
	}
	for _, k := range []ebiten.Key{ebiten.KeyEnter, ebiten.KeyZ, ebiten.KeyF12, ebiten.KeyW, ebiten.KeyDigit1} {
		if !bindableKey(k) {
			t.Fatal("ordinary key not bindable", k)
		}
	}
}

func TestControlsSettingsRoundTripCorruptAndPartial(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "controls-settings.json")
	if b, status := loadControlsSettingsFrom(path); status != controlsLoadDefault || !reflect.DeepEqual(b, defaultControlBindings()) {
		t.Fatal("missing file should give defaults")
	}
	b := defaultControlBindings()
	b.setKeyboard(controlActionIndex("confirm"), ebiten.KeyZ)
	b.setGamepad(controlActionIndex("space"), padY)
	b.Gamepad[controlActionIndex("controls")] = padNone
	b.Deadzone = 35
	if err := saveControlsSettingsTo(path, b); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var file controlsSettingsFile
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	wantKeyboard := map[string]string{"confirm": "Z", "next_unit": ""}
	wantPad := map[string]string{"space": "y", "controls": ""}
	if !reflect.DeepEqual(file.Keyboard, wantKeyboard) || !reflect.DeepEqual(file.Gamepad, wantPad) || file.StickDeadzone != 35 || file.SchemaVersion != 1 {
		t.Fatalf("file stores only overrides: %s", raw)
	}
	loaded, status := loadControlsSettingsFrom(path)
	if status != controlsLoadOK || !reflect.DeepEqual(loaded, b) {
		t.Fatalf("round trip status=%v\n got %+v\nwant %+v", status, loaded, b)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatal("atomic save left temporary files", entries)
	}
	for _, corrupt := range []string{"{", `{"schema_version":2}`, `[]`, ""} {
		os.WriteFile(path, []byte(corrupt), 0o644)
		if loaded, status := loadControlsSettingsFrom(path); status != controlsLoadCorrupt || !reflect.DeepEqual(loaded, defaultControlBindings()) {
			t.Fatalf("corrupt %q: status=%v", corrupt, status)
		}
	}
	os.WriteFile(path, []byte(`{"schema_version":1,"keyboard":{"confirm":"NoSuchKey","cancel":"Shift","ghost":"Z","space":"Q"},"gamepad":{"confirm":"turbo"},"stick_deadzone_percent":99}`), 0o644)
	loaded, status = loadControlsSettingsFrom(path)
	want := defaultControlBindings()
	want.setKeyboard(controlActionIndex("space"), ebiten.KeyQ)
	if status != controlsLoadPartial || !reflect.DeepEqual(loaded, want) {
		t.Fatalf("partial file: status=%v got %+v", status, loaded)
	}
}

func TestPlayerInputGamepadButtonsRepeatAndDeadzone(t *testing.T) {
	in := withActiveInput(t, defaultControlBindings())
	in.step(padSnapshot(padA), true, nil)
	if keyPressDuration(ebiten.KeyEnter) != 0 {
		t.Fatal("a pad connected with a button held is latched until released")
	}
	in.step(padSnapshot(), true, nil)
	in.step(padSnapshot(padA), true, nil)
	if !keyJustPressed(ebiten.KeyEnter) || keyPressDuration(ebiten.KeyEnter) != 1 {
		t.Fatal("A should press Enter")
	}
	keys := appendJustPressedKeys(nil)
	if len(keys) != 1 || keys[0] != ebiten.KeyEnter {
		t.Fatal("any-key list", keys)
	}
	in.step(padSnapshot(padA), true, nil)
	if keyJustPressed(ebiten.KeyEnter) || keyPressDuration(ebiten.KeyEnter) != 2 {
		t.Fatal("held A is one press")
	}
	in.step(padSnapshot(), true, nil)
	if keyPressDuration(ebiten.KeyEnter) != 0 {
		t.Fatal("release")
	}
	for button, key := range map[controlPadButton]ebiten.Key{padB: ebiten.KeyEscape, padX: ebiten.KeySpace, padStart: ebiten.KeyF12, padUp: ebiten.KeyArrowUp} {
		in.step(padSnapshot(button), true, nil)
		if !keyJustPressed(key) {
			t.Fatal("default gamepad mapping", button, key)
		}
		in.step(padSnapshot(), true, nil)
	}
	for _, unused := range []controlPadButton{padY, padLB, padRB, padBack} {
		in.step(padSnapshot(unused), true, nil)
		if len(appendJustPressedKeys(nil)) != 0 {
			t.Fatal("unbound button produced a key", unused)
		}
		in.step(padSnapshot(), true, nil)
	}
	// 十字鍵按住與鍵盤方向鍵同節奏：第 1 格、第 12 格、之後每 5 格。
	var repeats []int
	for tick := 1; tick <= 22; tick++ {
		in.step(padSnapshot(padDown), true, nil)
		if keyRepeat(ebiten.KeyArrowDown) {
			repeats = append(repeats, tick)
		}
	}
	if !reflect.DeepEqual(repeats, []int{1, 12, 17, 22}) {
		t.Fatal("repeat cadence", repeats)
	}
	in.step(padSnapshot(), true, nil)
	stick := func(x, y float64) inputDeviceSnapshot {
		snap := focusedSnapshot()
		snap.pads = []gamepadSnapshot{{stickX: x, stickY: y}}
		return snap
	}
	in.step(stick(0.15, -0.19), true, nil)
	if len(appendJustPressedKeys(nil)) != 0 {
		t.Fatal("inside 20% dead zone")
	}
	in.step(stick(0.6, 0.3), true, nil)
	if !keyJustPressed(ebiten.KeyArrowRight) || keyPressDuration(ebiten.KeyArrowDown) != 0 {
		t.Fatal("dominant axis right only")
	}
	in.step(stick(0, 0), true, nil)
	in.step(stick(-0.2, -0.7), true, nil)
	if !keyJustPressed(ebiten.KeyArrowUp) {
		t.Fatal("stick up")
	}
	b := defaultControlBindings()
	b.Deadzone = 50
	in.setBindings(b)
	in.step(stick(0, 0), true, nil)
	in.step(stick(0.45, 0), true, nil)
	if len(appendJustPressedKeys(nil)) != 0 {
		t.Fatal("dead zone setting ignored")
	}
}

func TestPlayerInputRebindingAppliesToGamepad(t *testing.T) {
	b := defaultControlBindings()
	b.setGamepad(controlActionIndex("space"), padA) // A 改成空白鍵，確認改成未設定
	in := withActiveInput(t, b)
	in.step(padSnapshot(), true, nil)
	in.step(padSnapshot(padA), true, nil)
	if !keyJustPressed(ebiten.KeySpace) || keyJustPressed(ebiten.KeyEnter) {
		t.Fatal("rebound A")
	}
}

func TestPlayerInputFocusLossAndDisconnectReleaseInput(t *testing.T) {
	in := withActiveInput(t, defaultControlBindings())
	in.step(padSnapshot(), true, nil)
	in.step(padSnapshot(padA), true, nil)
	if !keyJustPressed(ebiten.KeyEnter) {
		t.Fatal("press")
	}
	lost := padSnapshot(padA)
	lost.focused = false
	in.step(lost, true, nil)
	if keyPressDuration(ebiten.KeyEnter) != 0 {
		t.Fatal("focus loss should release")
	}
	in.step(padSnapshot(padA), true, nil)
	if keyPressDuration(ebiten.KeyEnter) != 0 {
		t.Fatal("button held across focus loss must be released first")
	}
	held := focusedSnapshot()
	held.pads = []gamepadSnapshot{{stickX: 0.9}}
	in.step(held, true, nil)
	if keyPressDuration(ebiten.KeyArrowRight) != 0 {
		t.Fatal("stick must return to centre before reading")
	}
	in.step(padSnapshot(), true, nil)
	in.step(padSnapshot(padA), true, nil)
	if !keyJustPressed(ebiten.KeyEnter) {
		t.Fatal("after release the pad works again")
	}
	in.step(focusedSnapshot(), true, nil) // 拔掉手把
	if keyPressDuration(ebiten.KeyEnter) != 0 {
		t.Fatal("disconnect releases")
	}
	in.step(padSnapshot(padA), true, nil)
	if keyPressDuration(ebiten.KeyEnter) != 0 {
		t.Fatal("reconnected pad is latched until released")
	}
}

func TestPlayerInputMouseButtonsWheelAndPointer(t *testing.T) {
	in := withActiveInput(t, defaultControlBindings())
	mouse := func(x, y int, left, leftJust, right, rightJust bool, wheel float64) inputDeviceSnapshot {
		return inputDeviceSnapshot{focused: true, cursorX: x, cursorY: y, left: left, leftJust: leftJust, right: right, rightJust: rightJust, wheelY: wheel}
	}
	if p := in.step(mouse(100, 100, false, false, false, false, 0), true, acceptBelowToolbar); p.active {
		t.Fatal("first sample has no movement")
	}
	if p := in.step(mouse(100, 100, false, false, false, false, 0), true, acceptBelowToolbar); p.active {
		t.Fatal("static mouse must not point")
	}
	if p := in.step(mouse(130, 90, false, false, false, false, 0), true, acceptBelowToolbar); !p.active || p.clicked || p.x != 130 || p.y != 90 {
		t.Fatal("moved mouse points", p)
	}
	p := in.step(mouse(130, 90, true, true, false, false, 0), true, acceptBelowToolbar)
	if !p.active || !p.clicked || !keyJustPressed(ebiten.KeyEnter) {
		t.Fatal("left click = Enter", p)
	}
	in.step(mouse(130, 90, true, false, false, false, 0), true, acceptBelowToolbar)
	if keyJustPressed(ebiten.KeyEnter) || keyPressDuration(ebiten.KeyEnter) != 2 {
		t.Fatal("held left is one press")
	}
	in.step(mouse(130, 90, false, false, false, false, 0), true, acceptBelowToolbar)
	in.step(mouse(130, 10, true, true, false, false, 0), true, acceptBelowToolbar)
	if keyPressDuration(ebiten.KeyEnter) != 0 {
		t.Fatal("click on the toolbar must not reach the game")
	}
	in.step(mouse(130, 90, true, false, false, false, 0), true, acceptBelowToolbar)
	if keyPressDuration(ebiten.KeyEnter) != 0 {
		t.Fatal("a press that started outside stays outside")
	}
	in.step(mouse(130, 90, false, false, true, true, 0), true, acceptBelowToolbar)
	if !keyJustPressed(ebiten.KeyEscape) {
		t.Fatal("right click = Esc")
	}
	in.step(mouse(130, 90, false, false, false, false, 1), true, acceptBelowToolbar)
	if !keyJustPressed(ebiten.KeyArrowUp) || !keyRepeat(ebiten.KeyArrowUp) {
		t.Fatal("wheel up = one Up")
	}
	in.step(mouse(130, 90, false, false, false, false, 0), true, acceptBelowToolbar)
	if keyPressDuration(ebiten.KeyArrowUp) != 0 {
		t.Fatal("wheel pulse lasts one tick")
	}
	in.step(mouse(130, 90, false, false, false, false, -0.5), true, acceptBelowToolbar)
	if keyPressDuration(ebiten.KeyArrowDown) != 0 {
		t.Fatal("half notch is not a step")
	}
	in.step(mouse(130, 90, false, false, false, false, -0.5), true, acceptBelowToolbar)
	if !keyJustPressed(ebiten.KeyArrowDown) {
		t.Fatal("two half notches make one Down")
	}
	in.step(mouse(130, 90, false, false, false, false, -1), true, acceptBelowToolbar)
	in.step(mouse(130, 90, false, false, false, false, -1), true, acceptBelowToolbar)
	if !keyJustPressed(ebiten.KeyArrowDown) {
		t.Fatal("consecutive wheel notches are separate presses")
	}
	// 設定面板開著（gameInput=false）：點擊不進遊戲。
	if p := in.step(mouse(200, 200, true, true, false, false, 0), false, acceptBelowToolbar); p.active || keyPressDuration(ebiten.KeyEnter) != 0 {
		t.Fatal("panel owns the mouse")
	}
}

func TestPlayerInputSuppressedClick(t *testing.T) {
	in := withActiveInput(t, defaultControlBindings())
	in.step(inputDeviceSnapshot{focused: true, cursorX: 50, cursorY: 50}, true, acceptBelowToolbar)
	in.step(inputDeviceSnapshot{focused: true, cursorX: 50, cursorY: 50, left: true, leftJust: true}, true, acceptBelowToolbar)
	in.suppressVirtual(ebiten.KeyEnter)
	if keyJustPressed(ebiten.KeyEnter) || len(appendJustPressedKeys(nil)) != 0 {
		t.Fatal("suppressed click still confirms")
	}
}

func TestNilActiveInputReadsKeyboardDirectly(t *testing.T) {
	if activeInput != nil {
		t.Fatal("tests must not leave activeInput set")
	}
	if keyJustPressed(ebiten.KeyEnter) || keyPressDuration(ebiten.KeyEnter) != 0 || len(appendJustPressedKeys(nil)) != 0 {
		t.Fatal("no keyboard in tests")
	}
}

func nativePointerGame(cameraX, cameraY int) *Game {
	st := &battle.State{W: 40, H: 30, HasNativeMapViewState: true}
	st.NativeMapViewState = battle.NativeMapViewState{CameraX: cameraX, CameraY: cameraY, CursorX: cameraX + 6, CursorY: cameraY + 4, VisibleCursorX: 6, VisibleCursorY: 4}
	return &Game{st: st, m: &MapData{W: 40, H: 30, TileW: 48, TileH: 48}}
}

func TestPointerMapCellUsesNativeViewportGeometry(t *testing.T) {
	g := nativePointerGame(3, 5)
	g.pointer = pointerTick{active: true, x: 8 + 48*2 + 5, y: 8 + 48*3 + 47}
	if x, y, ok := g.pointerMapCell(); !ok || x != 5 || y != 8 {
		t.Fatal("cell", x, y, ok)
	}
	for _, outside := range []image.Point{{7, 100}, {8 + 48*13, 100}, {100, 8 + 48*8}} {
		g.pointer = pointerTick{active: true, x: outside.X, y: outside.Y}
		if _, _, ok := g.pointerMapCell(); ok {
			t.Fatal("outside the native viewport", outside)
		}
	}
	g.pointer = pointerTick{x: 100, y: 100}
	if _, _, ok := g.pointerMapCell(); ok {
		t.Fatal("inactive pointer")
	}
	legacy := &Game{m: &MapData{W: 20, H: 20, TileW: 48, TileH: 48}, camX: 96, camY: 48, curX: 2, curY: 1,
		pointer: pointerTick{active: true, x: 150, y: 60}}
	if x, y, ok := legacy.pointerMapCell(); !ok || x != 5 || y != 2 {
		t.Fatal("legacy touch conversion", x, y, ok)
	}
	legacy.followPointerOnMap()
	if legacy.curX != 5 || legacy.curY != 2 {
		t.Fatal("cursor follows pointer", legacy.curX, legacy.curY)
	}
}

func TestPointerOverlayDirectionAndGate(t *testing.T) {
	g := nativePointerGame(0, 0)
	cx, cy := 8+6*48+24, 8+4*48+24
	for _, item := range []struct {
		dx, dy, want int
		ok           bool
	}{{0, -48, 0, true}, {-48, 5, 1, true}, {48, -5, 2, true}, {3, 48, 3, true}, {0, 0, 0, false}, {10, 10, 0, false}, {200, 0, 0, false}} {
		g.pointer = pointerTick{active: true, x: cx + item.dx, y: cy + item.dy}
		if direction, ok := g.pointerOverlayDirection(); ok != item.ok || ok && direction != item.want {
			t.Fatal("overlay direction", item, direction, ok)
		}
	}
	g.ringSel = 0
	g.pointer = pointerTick{active: true, x: cx + 48, y: cy}
	g.applyPointerToOverlay([4]int{0, 0, 1, 0}) // 右邊不可用：原版閘門不改選擇
	if g.ringSel != 0 {
		t.Fatal("unavailable direction selected")
	}
	g.applyPointerToOverlay([4]int{0, 0, 0, 0})
	if g.ringSel != 2 {
		t.Fatal("available direction")
	}
}

func TestPointerSelectsTitleMenuSlotsAndTown(t *testing.T) {
	g := &Game{titlePhase: "menu", titleAssets: &titleAssets{}}
	for i := range g.titleAssets.items {
		g.titleAssets.items[i][0] = ebiten.NewImage(40, 8)
	}
	rects := g.titleMenuRects()
	at := rects[2].Min.Add(image.Pt(3, 3))
	g.pointer = pointerTick{active: true, x: at.X, y: at.Y}
	g.applyPointerToTitleMenu()
	if g.titleSel != 2 {
		t.Fatal("title menu", g.titleSel)
	}
	g.titleFlash = 5
	at = rects[0].Min.Add(image.Pt(3, 3))
	g.pointer = pointerTick{active: true, x: at.X, y: at.Y}
	g.applyPointerToTitleMenu()
	if g.titleSel != 2 {
		t.Fatal("confirmation flash must not move the selection")
	}
	g = &Game{titlePhase: "loadslots"}
	at = titleSlotRects()[3].Min.Add(image.Pt(5, 5))
	g.pointer = pointerTick{active: true, x: at.X, y: at.Y}
	g.applyPointerToTitleSlots()
	if g.titleSlotSel != 3 {
		t.Fatal("load slot", g.titleSlotSel)
	}
	for variant := 0; variant < 3; variant++ {
		rects := nativeTownSelectionRects(variant)
		if len(rects) != 5 {
			t.Fatal("town rects")
		}
		for target := 0; target < 5; target++ {
			g = &Game{campSel: 0}
			at = rects[target].Min.Add(image.Pt(rects[target].Dx()/2, rects[target].Dy()/2))
			g.pointer = pointerTick{active: true, x: at.X, y: at.Y}
			g.applyPointerToNativeTown(variant)
			if g.campSel != target {
				t.Fatal("town", variant, target, g.campSel)
			}
		}
	}
	if pointerIn(choiceMenuRects(3), 300, 162+2*28+5) != 2 || pointerIn(choiceMenuRects(3), 100, 170) != -1 {
		t.Fatal("choice rows")
	}
}

func TestF12OpensControlsPanelAndKeepsDebugHUD(t *testing.T) {
	in := withActiveInput(t, defaultControlBindings())
	g := &Game{input: in}
	in.step(padSnapshot(), true, nil)
	in.step(padSnapshot(padStart), true, nil)
	if err := g.Update(); err != nil {
		t.Fatal(err)
	}
	if !g.controls.open || g.debug {
		t.Fatal("Start → F12 opens the panel, not the debug HUD", g.controls.open, g.debug)
	}
}
