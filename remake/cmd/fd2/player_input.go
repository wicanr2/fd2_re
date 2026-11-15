package main

// player_input.go — 輸入對應層（Rust 版 fd2-app/src/input.rs 照這一層移植）。
//
// 介面：
//
//	實體輸入 ──► playerInput ──► 邏輯按鍵（ebiten.Key）──► 遊戲各處
//	  鍵盤：controlKeyTable 把實體鍵換成邏輯鍵，每次查詢都即時讀 inpututil。
//	  滑鼠、手把：每次 Update 前 step() 取樣一次，換成「虛擬按住」的邏輯鍵。
//	         滑鼠另外產生 pointerTick（指向哪裡、這一刻有沒有點），由各畫面自己換算成格子或列。
//
// 遊戲程式只呼叫 keyJustPressed／keyPressDuration／appendJustPressedKeys。邏輯鍵的按住時間取
// 鍵盤與虛擬來源的最大值，keyJustPressed 與 inpututil 相同：按住時間等於 1。
// activeInput 只在正式執行（main）設定；測試與工具沒有設定時，三個函式直接回 inpututil 的值。
// 修飾鍵組合（Alt+Enter 全螢幕、修飾鍵秘密商店、Ctrl+F12 除錯 HUD）一律讀實體鍵，不經這一層。

import (
	"image"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// activeInput 是正式執行的輸入對應；nil 時鍵盤恆等、沒有滑鼠與手把轉換。
var activeInput *playerInput

func keyPressDuration(k ebiten.Key) int {
	if activeInput == nil {
		return inpututil.KeyPressDuration(k)
	}
	return activeInput.duration(k)
}

func keyJustPressed(k ebiten.Key) bool {
	if activeInput == nil {
		return inpututil.IsKeyJustPressed(k)
	}
	return activeInput.duration(k) == 1
}

// appendJustPressedKeys 回傳這一刻剛按下的邏輯鍵；「任一鍵」的判斷用它。
func appendJustPressedKeys(keys []ebiten.Key) []ebiten.Key {
	if activeInput == nil {
		return inpututil.AppendJustPressedKeys(keys)
	}
	return activeInput.appendJustPressed(keys)
}

// controlKeyTable：produce[實體鍵] = 邏輯鍵（-1 表示這個實體鍵不產生任何邏輯鍵）。
// 規則：所有動作的邏輯鍵先設為不產生，再把每個動作目前的鍵盤鍵指向該動作的邏輯鍵；其餘鍵恆等。
// 預設綁定下每個動作的鍵盤鍵就是自己的邏輯鍵，所以整張表是恆等。
type controlKeyTable struct {
	produce  [ebiten.KeyMax + 1]ebiten.Key
	physical [ebiten.KeyMax + 1][]ebiten.Key // 邏輯鍵 → 會產生它的實體鍵
}

func newControlKeyTable(b controlBindings) *controlKeyTable {
	t := &controlKeyTable{}
	for k := range t.produce {
		t.produce[k] = ebiten.Key(k)
	}
	for _, action := range controlActions {
		t.produce[action.Logical] = keyUnbound
	}
	for i, action := range controlActions {
		if k := b.Keyboard[i]; k >= 0 && k <= ebiten.KeyMax {
			t.produce[k] = action.Logical
		}
	}
	for k, logical := range t.produce {
		if logical >= 0 {
			t.physical[logical] = append(t.physical[logical], ebiten.Key(k))
		}
	}
	return t
}

// pointerTick 是這一次 Update 的滑鼠指向，座標是 640×400 遊戲畫布。只有滑鼠真的移動或按下
// 左鍵的那一刻 active 才為 true；靜止的滑鼠不影響游標與選單。
type pointerTick struct {
	active, clicked bool
	x, y            int
}

// inputDeviceSnapshot 是一次取樣的滑鼠與手把狀態；測試用假的快照驅動 step()。
type inputDeviceSnapshot struct {
	focused             bool
	cursorX, cursorY    int
	left, right         bool
	leftJust, rightJust bool
	wheelY              float64
	pads                []gamepadSnapshot
}

type gamepadSnapshot struct {
	name           string
	buttons        [padButtonCount]bool
	stickX, stickY float64
}

type playerInput struct {
	bindings controlBindings
	status   controlsLoadStatus
	table    *controlKeyTable

	held       map[ebiten.Key]int  // 虛擬按住時間（滑鼠鍵、手把）
	pulse      map[ebiten.Key]bool // 只維持這一刻的按下（滾輪）
	suppressed map[ebiten.Key]bool // 這一刻作廢的虛擬按下（點到不能選的項目）

	cursorX, cursorY      int
	cursorKnown           bool
	mouseLeft, mouseRight bool // 從遊戲區開始按下的滑鼠鍵
	wheel                 float64
	padLatched            bool // 失焦、斷線或關面板後，要等手把全部放開、搖桿回中央才再讀
	padCount              int
	padNames              []string
}

func newPlayerInput(b controlBindings, status controlsLoadStatus) *playerInput {
	in := &playerInput{status: status, held: map[ebiten.Key]int{}, pulse: map[ebiten.Key]bool{}, suppressed: map[ebiten.Key]bool{}}
	in.setBindings(b)
	return in
}

func (in *playerInput) setBindings(b controlBindings) {
	in.bindings = b.clone()
	in.bindings.Deadzone = normalizeStickDeadzone(in.bindings.Deadzone)
	in.table = newControlKeyTable(in.bindings)
}

func (in *playerInput) keyboardDuration(k ebiten.Key) int {
	if k < 0 || k > ebiten.KeyMax {
		return inpututil.KeyPressDuration(k)
	}
	d := 0
	for _, physical := range in.table.physical[k] {
		if v := inpututil.KeyPressDuration(physical); v > d {
			d = v
		}
	}
	return d
}

func (in *playerInput) virtualDuration(k ebiten.Key) int {
	if in.suppressed[k] {
		return 0
	}
	if in.pulse[k] {
		return 1
	}
	return in.held[k]
}

func (in *playerInput) duration(k ebiten.Key) int {
	d := in.keyboardDuration(k)
	if v := in.virtualDuration(k); v > d {
		d = v
	}
	return d
}

func (in *playerInput) appendJustPressed(keys []ebiten.Key) []ebiten.Key {
	seen := map[ebiten.Key]bool{}
	for _, physical := range inpututil.AppendJustPressedKeys(nil) {
		logical := physical
		if physical >= 0 && physical <= ebiten.KeyMax {
			logical = in.table.produce[physical]
		}
		if logical >= 0 && !seen[logical] && in.duration(logical) == 1 {
			seen[logical] = true
			keys = append(keys, logical)
		}
	}
	var virtual []ebiten.Key
	for k := range in.held {
		virtual = append(virtual, k)
	}
	for k := range in.pulse {
		virtual = append(virtual, k)
	}
	sort.Slice(virtual, func(i, j int) bool { return virtual[i] < virtual[j] })
	for _, k := range virtual {
		if !seen[k] && in.virtualDuration(k) == 1 && in.duration(k) == 1 {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	return keys
}

// suppressVirtual 作廢這一刻由滑鼠或手把產生的按下；鍵盤照常。
func (in *playerInput) suppressVirtual(k ebiten.Key) {
	in.suppressed[k] = true
}

// step 每次 Update 取樣一次。gameInput 為 false（設定面板、能力設定或工具列吃掉輸入）時，
// 手把照樣計時（關掉面板的那一下不會又傳進遊戲），滑鼠鍵不算進遊戲。accept 把視窗座標換成
// 遊戲畫布座標，不在遊戲區（工具列、海報側欄、下拉選單）回 false。
func (in *playerInput) step(snap inputDeviceSnapshot, gameInput bool, accept func(x, y int) (int, int, bool)) pointerTick {
	for k := range in.pulse {
		delete(in.pulse, k)
	}
	for k := range in.suppressed {
		delete(in.suppressed, k)
	}
	held := map[ebiten.Key]bool{}
	var ptr pointerTick
	if !snap.focused {
		in.mouseLeft, in.mouseRight, in.cursorKnown, in.wheel = false, false, false, 0
		in.padLatched = true
	} else {
		ptr = in.stepMouse(snap, gameInput, accept, held)
		in.stepGamepads(snap, held)
	}
	for k := range in.held {
		if !held[k] {
			delete(in.held, k)
		}
	}
	for k := range held {
		in.held[k]++
	}
	return ptr
}

func (in *playerInput) stepMouse(snap inputDeviceSnapshot, gameInput bool, accept func(x, y int) (int, int, bool), held map[ebiten.Key]bool) pointerTick {
	var ptr pointerTick
	x, y, ok := 0, 0, false
	if gameInput && accept != nil {
		x, y, ok = accept(snap.cursorX, snap.cursorY)
	}
	moved := in.cursorKnown && (snap.cursorX != in.cursorX || snap.cursorY != in.cursorY)
	in.cursorX, in.cursorY, in.cursorKnown = snap.cursorX, snap.cursorY, true
	if !snap.left {
		in.mouseLeft = false
	} else if snap.leftJust && ok {
		in.mouseLeft = true
		ptr.clicked = true
	}
	if !snap.right {
		in.mouseRight = false
	} else if snap.rightJust && ok {
		in.mouseRight = true
	}
	if in.mouseLeft {
		held[ebiten.KeyEnter] = true
	}
	if in.mouseRight {
		held[ebiten.KeyEscape] = true
	}
	// 滾輪一格是一次上／下；觸控板的小數位移累積到一格才算。
	if ok {
		in.wheel = math.Max(-1.5, math.Min(1.5, in.wheel+snap.wheelY))
		if in.wheel >= 1 {
			in.pulse[ebiten.KeyArrowUp] = true
			in.wheel--
		} else if in.wheel <= -1 {
			in.pulse[ebiten.KeyArrowDown] = true
			in.wheel++
		}
	} else {
		in.wheel = 0
	}
	if ok && (moved || ptr.clicked) {
		ptr.active, ptr.x, ptr.y = true, x, y
	}
	return ptr
}

func (in *playerInput) stepGamepads(snap inputDeviceSnapshot, held map[ebiten.Key]bool) {
	if len(snap.pads) != in.padCount {
		in.padLatched = true // 接上或拔掉手把：所有按鈕先放開、搖桿回中央才開始讀
	}
	in.padCount = len(snap.pads)
	in.padNames = in.padNames[:0]
	var buttons [padButtonCount]bool
	stickX, stickY := 0.0, 0.0
	for _, pad := range snap.pads {
		in.padNames = append(in.padNames, pad.name)
		for button, pressed := range pad.buttons {
			buttons[button] = buttons[button] || pressed
		}
		if math.Abs(pad.stickX) > math.Abs(stickX) {
			stickX = pad.stickX
		}
		if math.Abs(pad.stickY) > math.Abs(stickY) {
			stickY = pad.stickY
		}
	}
	direction := stickDirection(stickX, stickY, in.bindings.Deadzone)
	if in.padLatched {
		for _, pressed := range buttons {
			if pressed {
				return
			}
		}
		if direction != keyUnbound {
			return
		}
		in.padLatched = false
	}
	for i, action := range controlActions {
		if button := in.bindings.Gamepad[i]; button != padNone && buttons[button] {
			held[action.Logical] = true
		}
	}
	if direction != keyUnbound {
		held[direction] = true
	}
}

// stickDirection 只取主軸，避免斜推同時送出兩個方向；死區以內不算。
func stickDirection(x, y float64, deadzone int) ebiten.Key {
	limit := float64(normalizeStickDeadzone(deadzone)) / 100
	if math.Abs(x) <= limit && math.Abs(y) <= limit {
		return keyUnbound
	}
	if math.Abs(x) >= math.Abs(y) {
		if x < 0 {
			return ebiten.KeyArrowLeft
		}
		return ebiten.KeyArrowRight
	}
	if y < 0 {
		return ebiten.KeyArrowUp
	}
	return ebiten.KeyArrowDown
}

// readInputDevices 讀 Ebiten 的滑鼠與手把。只認標準手把配置（Windows XInput 手把屬於這類）。
func readInputDevices() inputDeviceSnapshot {
	x, y := ebiten.CursorPosition()
	_, wheelY := ebiten.Wheel()
	snap := inputDeviceSnapshot{
		focused: ebiten.IsFocused(), cursorX: x, cursorY: y,
		left: ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft), right: ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight),
		leftJust:  inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft),
		rightJust: inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight),
		wheelY:    wheelY,
	}
	for _, id := range ebiten.AppendGamepadIDs(nil) {
		if !ebiten.IsStandardGamepadLayoutAvailable(id) {
			continue
		}
		pad := gamepadSnapshot{name: ebiten.GamepadName(id),
			stickX: ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal),
			stickY: ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickVertical)}
		for button, standard := range padStandardButtons {
			pad.buttons[button] = ebiten.IsStandardGamepadButtonPressed(id, standard)
		}
		snap.pads = append(snap.pads, pad)
	}
	return snap
}

// sampleInput 在每次 Update 取樣一次（包括設定面板與能力設定開著、遊戲 Update 暫停時）。
func (p *modernHDPresentation) sampleInput(gameInput bool) {
	g := p.Game
	g.pointer = pointerTick{}
	if g.input == nil || g.input != activeInput {
		return
	}
	g.pointer = g.input.step(readInputDevices(), gameInput, func(x, y int) (int, int, bool) {
		return g.playerTouchCoordinates(x, y)
	})
}

// pointerIn 回傳指向點落在哪一列（沒有則 -1）。
func pointerIn(rects []image.Rectangle, x, y int) int {
	for i, rect := range rects {
		if image.Pt(x, y).In(rect) {
			return i
		}
	}
	return -1
}
