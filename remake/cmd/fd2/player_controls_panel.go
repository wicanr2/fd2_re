package main

// player_controls_panel.go — F12「手把與自訂按鍵」設定面板（版面依 mm3 使用者核可的分組表格）。
//
// 開啟：F12（邏輯鍵，可改綁）、手把 Start（預設綁 F12）、工具列「按鍵」。開著時遊戲 Update
// 暫停、遊戲時鐘凍結，與能力設定相同。面板自己的操作讀實體鍵與手把固定按鈕，不經玩家綁定，
// 改壞綁定也不會被鎖在面板外：
//
//	鍵盤：↑↓ 選列、PageUp／PageDown、Enter 指定、Delete 清除此列、R 恢復此列預設、
//	      Esc 取消或返回、F12 關閉、←→ 調整防飄移（選到滑桿列時）。
//	手把：十字鍵或左搖桿選列、A 指定、B 取消或返回、X 清除、Y 預設、Start 關閉。
//	滑鼠：點選列（再點一次＝指定）、滾輪捲動、拖曳滑桿、右鍵＝返回、「全部恢復預設」按鈕。
//
// 指定時再按要用的鍵盤鍵或手把按鈕；已被別列使用就先提示，確認後才取代，別列改成未設定。
// 每次變更立即寫 controls-settings.json。滑鼠游標速度不提供：FD2 沒有遊戲內的模擬游標，
// 滑鼠直接用系統指標，指向格子與選單。

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type controlsConflict struct {
	action, other  int
	reset, gamepad bool
	key            ebiten.Key
	button         controlPadButton
}

type controlsPanel struct {
	open       bool
	selected   int // 0..len(controlActions)-1 是動作；len(controlActions) 是防飄移滑桿
	scroll     int // 第一條可見的行（行含分組標題）
	capture    bool
	conflict   *controlsConflict
	message    string
	dragging   bool
	stickHeld  [4]int // 面板內的搖桿連發計數（上、下、左、右）
	text       *controlsLocale
	textLocale string
}

// controlsPanelInput 是一次 Update 收到的面板操作；實體鍵盤、滑鼠與手把都先換成這個，
// 測試直接餵這個結構。
type controlsPanelInput struct {
	up, down, left, right, pageUp, pageDown bool
	confirm, back, clear, reset, close      bool
	captureKey                              ebiten.Key
	captureButton                           controlPadButton
	captureCancel                           bool
	click, drag                             bool
	point                                   image.Point
	wheel                                   int
}

func noControlsPanelInput() controlsPanelInput {
	return controlsPanelInput{captureKey: keyUnbound, captureButton: padNone}
}

// 面板幾何（640×400 畫布座標）。
var (
	controlsPanelRect     = image.Rect(16, 10, 624, 390)
	controlsPadChip       = image.Rect(436, 20, 610, 44)
	controlsHeaderBand    = image.Rect(24, 62, 616, 82)
	controlsListRect      = image.Rect(24, 82, 616, 282)
	controlsSliderRow     = image.Rect(24, 286, 616, 312)
	controlsSliderTrack   = image.Rect(236, 296, 456, 302)
	controlsResetAllRect  = image.Rect(478, 338, 610, 364)
	controlsColumnKeys    = 296
	controlsColumnPad     = 412
	controlsLineHeight    = 20
	controlsVisibleLines  = 10
	controlsSliderSelects = len(controlActions)
)

// controlsLine 是清單的一行：分組標題（action = -1）或動作列。
type controlsLine struct {
	group  controlActionGroup
	action int
}

func controlsLines() []controlsLine {
	var lines []controlsLine
	for group := controlActionGroup(0); group < controlGroupCount; group++ {
		lines = append(lines, controlsLine{group: group, action: -1})
		for i, action := range controlActions {
			if action.Group == group {
				lines = append(lines, controlsLine{group: group, action: i})
			}
		}
	}
	return lines
}

func controlsLineOf(action int) int {
	for i, line := range controlsLines() {
		if line.action == action {
			return i
		}
	}
	return 0
}

func (g *Game) openControlsPanel() {
	if g.input == nil {
		g.input = newPlayerInput(loadControlsSettings())
	}
	text, err := loadControlsLocale(g.localeID)
	if err != nil {
		g.loadErr = "controls locale: " + err.Error()
	}
	g.controls = controlsPanel{open: true, text: text, textLocale: g.localeID}
	switch g.input.status {
	case controlsLoadCorrupt:
		g.controls.message = text.text("controls.msg.corrupt")
	case controlsLoadPartial:
		g.controls.message = text.text("controls.msg.partial")
	default:
		g.controls.message = text.text("controls.msg.ready")
	}
	// 工具列在一般 Update 之前開面板；比照能力設定在消費端記錄稽核。
	g.recordInputAuditState()
}

func (g *Game) closeControlsPanel() {
	g.controls = controlsPanel{}
}

func (c *controlsPanel) actionName(action int) string {
	return c.text.text("controls.action." + controlActions[action].ID)
}

func (c *controlsPanel) ensureVisible() {
	if c.selected >= controlsSliderSelects {
		return
	}
	line := controlsLineOf(c.selected)
	if line-1 >= 0 && controlsLines()[line-1].action < 0 {
		line-- // 分組第一列連同標題一起露出
	}
	if line < c.scroll {
		c.scroll = line
	}
	last := controlsLineOf(c.selected)
	if last >= c.scroll+controlsVisibleLines {
		c.scroll = last - controlsVisibleLines + 1
	}
	c.clampScroll()
}

func (c *controlsPanel) clampScroll() {
	maxScroll := len(controlsLines()) - controlsVisibleLines
	if c.scroll > maxScroll {
		c.scroll = maxScroll
	}
	if c.scroll < 0 {
		c.scroll = 0
	}
}

// commitControls 套用並立即保存；保存失敗時綁定仍生效，訊息告訴玩家。
func (g *Game) commitControls(b controlBindings, messageKey string) {
	c := &g.controls
	g.input.setBindings(b)
	g.input.status = controlsLoadOK
	if err := saveControlsSettings(g.input.bindings); err != nil {
		c.message = c.text.text("controls.msg.save_failed", err.Error())
		return
	}
	c.message = c.text.text(messageKey)
}

func (g *Game) stepControlsPanel(in controlsPanelInput) {
	c := &g.controls
	if !c.open || g.input == nil {
		return
	}
	if in.close {
		g.closeControlsPanel()
		return
	}
	if c.conflict != nil {
		switch {
		case in.back:
			c.conflict = nil
			c.message = c.text.text("controls.msg.ready")
		case in.confirm || in.click:
			g.applyControlsConflict()
		}
		return
	}
	if c.capture {
		switch {
		case in.captureCancel || in.back || in.click:
			c.capture = false
			c.message = c.text.text("controls.msg.ready")
		case in.captureKey != keyUnbound:
			g.captureControlsKey(in.captureKey)
		case in.captureButton != padNone:
			g.captureControlsButton(in.captureButton)
		}
		return
	}
	if in.back {
		g.closeControlsPanel()
		return
	}
	last := controlsSliderSelects
	switch {
	case in.up && c.selected > 0:
		c.selected--
		c.ensureVisible()
	case in.down && c.selected < last:
		c.selected++
		c.ensureVisible()
	case in.pageUp:
		c.selected = max(0, c.selected-controlsVisibleLines+1)
		c.ensureVisible()
	case in.pageDown:
		c.selected = min(last, c.selected+controlsVisibleLines-1)
		c.ensureVisible()
	}
	if in.wheel != 0 {
		c.scroll -= in.wheel
		c.clampScroll()
	}
	if c.selected == last && (in.left || in.right) {
		b := g.input.bindings.clone()
		if in.left {
			b.Deadzone = max(minStickDeadzone, b.Deadzone-stickDeadzoneStep)
		} else {
			b.Deadzone = min(maxStickDeadzone, b.Deadzone+stickDeadzoneStep)
		}
		g.commitControls(b, "controls.msg.assigned")
	}
	if in.click || in.drag {
		g.controlsPanelPoint(in.point, in.click)
		return
	}
	if c.selected >= last {
		return
	}
	switch {
	case in.confirm:
		c.capture = true
		c.message = c.text.text("controls.msg.capture")
	case in.clear:
		b := g.input.bindings.clone()
		b.clearAction(c.selected)
		g.commitControls(b, "controls.msg.cleared")
	case in.reset:
		if other := g.input.bindings.resetConflict(c.selected); other >= 0 {
			// resetConflict 先看鍵盤再看手把；提示顯示被拿走的那一個。
			label := c.padLabel(controlActions[c.selected].DefaultPad)
			if key := controlActions[c.selected].Logical; g.input.bindings.keyboardOwner(key) == other {
				label = controlKeyLabel(key)
			}
			c.conflict = &controlsConflict{action: c.selected, other: other, reset: true}
			c.message = c.text.text("controls.msg.conflict", label, c.actionName(other))
			return
		}
		b := g.input.bindings.clone()
		b.resetAction(c.selected)
		g.commitControls(b, "controls.msg.reset_row")
	}
}

func (g *Game) controlsPanelPoint(point image.Point, click bool) {
	c := &g.controls
	if click {
		c.dragging = false // 新的一次按下重新判斷位置
	}
	if c.dragging || click && point.In(controlsSliderRow) {
		c.selected, c.dragging = controlsSliderSelects, true
		span := controlsSliderTrack.Dx()
		ratio := float64(point.X-controlsSliderTrack.Min.X) / float64(span)
		value := minStickDeadzone + int(ratio*float64(maxStickDeadzone-minStickDeadzone)+0.5)
		value = max(minStickDeadzone, min(maxStickDeadzone, value))
		value -= value % stickDeadzoneStep
		if value != g.input.bindings.Deadzone {
			b := g.input.bindings.clone()
			b.Deadzone = value
			g.commitControls(b, "controls.msg.assigned")
		}
		return
	}
	if !click {
		return
	}
	if point.In(controlsResetAllRect) {
		g.commitControls(defaultControlBindings(), "controls.msg.reset_all")
		return
	}
	if !point.In(controlsListRect) {
		return
	}
	line := c.scroll + (point.Y-controlsListRect.Min.Y)/controlsLineHeight
	lines := controlsLines()
	if line < 0 || line >= len(lines) || lines[line].action < 0 {
		return
	}
	if lines[line].action == c.selected {
		c.capture = true
		c.message = c.text.text("controls.msg.capture")
		return
	}
	c.selected = lines[line].action
}

func (g *Game) captureControlsKey(k ebiten.Key) {
	c := &g.controls
	if !bindableKey(k) {
		c.message = c.text.text("controls.msg.reserved")
		return
	}
	c.capture = false
	if owner := g.input.bindings.keyboardOwner(k); owner >= 0 && owner != c.selected {
		c.conflict = &controlsConflict{action: c.selected, other: owner, key: k}
		c.message = c.text.text("controls.msg.conflict", controlKeyLabel(k), c.actionName(owner))
		return
	}
	b := g.input.bindings.clone()
	b.setKeyboard(c.selected, k)
	g.commitControls(b, "controls.msg.assigned")
}

func (g *Game) captureControlsButton(button controlPadButton) {
	c := &g.controls
	c.capture = false
	if owner := g.input.bindings.gamepadOwner(button); owner >= 0 && owner != c.selected {
		c.conflict = &controlsConflict{action: c.selected, other: owner, gamepad: true, button: button}
		c.message = c.text.text("controls.msg.conflict", c.padLabel(button), c.actionName(owner))
		return
	}
	b := g.input.bindings.clone()
	b.setGamepad(c.selected, button)
	g.commitControls(b, "controls.msg.assigned")
}

func (g *Game) applyControlsConflict() {
	c := &g.controls
	conflict := c.conflict
	c.conflict = nil
	b := g.input.bindings.clone()
	switch {
	case conflict.reset:
		b.resetAction(conflict.action)
		g.commitControls(b, "controls.msg.reset_row")
	case conflict.gamepad:
		b.setGamepad(conflict.action, conflict.button)
		g.commitControls(b, "controls.msg.assigned")
	default:
		b.setKeyboard(conflict.action, conflict.key)
		g.commitControls(b, "controls.msg.assigned")
	}
}

// controlKeyLabel 是鍵盤格顯示的鍵名。
func controlKeyLabel(k ebiten.Key) string {
	switch k {
	case keyUnbound:
		return ""
	case ebiten.KeyArrowUp:
		return "↑"
	case ebiten.KeyArrowDown:
		return "↓"
	case ebiten.KeyArrowLeft:
		return "←"
	case ebiten.KeyArrowRight:
		return "→"
	case ebiten.KeyEscape:
		return "Esc"
	}
	return strings.TrimPrefix(k.String(), "Digit")
}

var padFaceLabels = [padButtonCount]string{
	padA: "A", padB: "B", padX: "X", padY: "Y", padLB: "LB", padRB: "RB", padLT: "LT", padRT: "RT",
	padBack: "Back", padStart: "Start", padL3: "L3", padR3: "R3",
}

func (c *controlsPanel) padLabel(button controlPadButton) string {
	switch button {
	case padNone:
		return ""
	case padUp:
		return c.text.text("controls.dpad") + "↑"
	case padDown:
		return c.text.text("controls.dpad") + "↓"
	case padLeft:
		return c.text.text("controls.dpad") + "←"
	case padRight:
		return c.text.text("controls.dpad") + "→"
	}
	return padFaceLabels[button]
}

// ---- 實體輸入 → controlsPanelInput ----

func repeatTick(d int) bool {
	return d == 1 || d >= 12 && (d-12)%5 == 0
}

func (p *modernHDPresentation) readControlsPanelInput() controlsPanelInput {
	c := &p.Game.controls
	in := noControlsPanelInput()
	raw := inpututil.KeyPressDuration
	in.up, in.down = repeatTick(raw(ebiten.KeyArrowUp)), repeatTick(raw(ebiten.KeyArrowDown))
	in.left, in.right = repeatTick(raw(ebiten.KeyArrowLeft)), repeatTick(raw(ebiten.KeyArrowRight))
	in.pageUp, in.pageDown = inpututil.IsKeyJustPressed(ebiten.KeyPageUp), inpututil.IsKeyJustPressed(ebiten.KeyPageDown)
	in.confirm = inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter)
	in.back = inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
	in.clear = inpututil.IsKeyJustPressed(ebiten.KeyDelete)
	in.reset = inpututil.IsKeyJustPressed(ebiten.KeyR)
	in.close = inpututil.IsKeyJustPressed(ebiten.KeyF12)
	if c.capture {
		for _, k := range inpututil.AppendJustPressedKeys(nil) {
			if k == ebiten.KeyEscape {
				in.captureCancel = true
				break
			}
			in.captureKey = k
			break
		}
		in.close = false
	}
	var stick [4]bool
	deadzone := defaultStickDeadzone
	if p.Game.input != nil {
		deadzone = p.Game.input.bindings.Deadzone
	}
	for _, id := range ebiten.AppendGamepadIDs(nil) {
		if !ebiten.IsStandardGamepadLayoutAvailable(id) {
			continue
		}
		just := func(b controlPadButton) bool {
			return inpututil.IsStandardGamepadButtonJustPressed(id, padStandardButtons[b])
		}
		hold := func(b controlPadButton) int {
			return inpututil.StandardGamepadButtonPressDuration(id, padStandardButtons[b])
		}
		if c.capture {
			for button := controlPadButton(0); button < padButtonCount; button++ {
				if just(button) && in.captureButton == padNone {
					if button == padB {
						in.captureCancel = true
					} else {
						in.captureButton = button
					}
				}
			}
			continue
		}
		in.up = in.up || repeatTick(hold(padUp))
		in.down = in.down || repeatTick(hold(padDown))
		in.left = in.left || repeatTick(hold(padLeft))
		in.right = in.right || repeatTick(hold(padRight))
		in.confirm = in.confirm || just(padA)
		in.back = in.back || just(padB)
		in.clear = in.clear || just(padX)
		in.reset = in.reset || just(padY)
		in.close = in.close || just(padStart)
		switch stickDirection(ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal),
			ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickVertical), deadzone) {
		case ebiten.KeyArrowUp:
			stick[0] = true
		case ebiten.KeyArrowDown:
			stick[1] = true
		case ebiten.KeyArrowLeft:
			stick[2] = true
		case ebiten.KeyArrowRight:
			stick[3] = true
		}
	}
	for i, held := range stick {
		if held {
			c.stickHeld[i]++
		} else {
			c.stickHeld[i] = 0
		}
	}
	in.up = in.up || repeatTick(c.stickHeld[0])
	in.down = in.down || repeatTick(c.stickHeld[1])
	in.left = in.left || repeatTick(c.stickHeld[2])
	in.right = in.right || repeatTick(c.stickHeld[3])
	x, y := ebiten.CursorPosition()
	in.point = image.Pt(x-p.posterOffset(), y-p.toolbarOffset())
	in.click = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		c.dragging = false
	}
	in.drag = c.dragging && !in.click
	if _, wheel := ebiten.Wheel(); wheel >= 1 {
		in.wheel = 1
	} else if wheel <= -1 {
		in.wheel = -1
	}
	return in
}

// updateControlsPanel 在播放器 Update 最前面；面板開著就吃掉這一次 Update。
func (p *modernHDPresentation) updateControlsPanel() bool {
	g := p.Game
	if !g.controls.open {
		return false
	}
	defer g.recordInputAuditState()
	g.playerClock.wall = time.Now() // 與能力設定相同：面板開著時遊戲時鐘不走
	g.stepControlsPanel(p.readControlsPanelInput())
	return true
}

// ---- 繪製（1920×1200 疊層，同能力設定的三倍畫布）----

var (
	controlsColorPanel    = color.RGBA{24, 30, 43, 255}
	controlsColorBorder   = color.RGBA{70, 82, 104, 255}
	controlsColorBand     = color.RGBA{36, 43, 58, 255}
	controlsColorGroup    = color.RGBA{30, 36, 50, 255}
	controlsColorSelected = color.RGBA{60, 53, 36, 255}
	controlsColorGold     = color.RGBA{226, 194, 122, 255}
	controlsColorText     = color.RGBA{232, 235, 241, 255}
	controlsColorMuted    = color.RGBA{158, 168, 186, 255}
	controlsColorChip     = color.RGBA{30, 40, 56, 255}
	controlsColorChipLine = color.RGBA{112, 128, 154, 255}
)

func (p *modernHDPresentation) drawControlsPanel(screen ebiten.FinalScreen, geometry ebiten.GeoM) {
	g := p.Game
	c := &g.controls
	if !c.open {
		return
	}
	if p.controlsSurface == nil {
		p.controlsSurface = ebiten.NewImage(1920, 1200)
	}
	s := p.controlsSurface
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
		text.Draw(s, caption, face, x*3, baseline*3, ink)
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

	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(1.0/3, 1.0/3)
	op.GeoM.Concat(geometry)
	screen.DrawImage(s, op)
}
