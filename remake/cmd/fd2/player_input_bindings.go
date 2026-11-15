package main

// player_input_bindings.go — 按鍵設定面板的綁定資料與 controls-settings.json。
//
// 遊戲各處讀的是「邏輯按鍵」：原版操作本來就用的 Enter、Esc、方向鍵與 F 鍵，值沿用
// ebiten.Key。每個可設定的動作（controlAction）對應一個邏輯按鍵，再各有一個鍵盤鍵與一個
// 手把按鈕。玩家沒改過的動作，鍵盤鍵就是邏輯按鍵本身，所以預設對應是恆等，鍵盤行為與
// 加入這一層之前相同。對應表的規則見 newControlKeyTable。
//
// 這是重製端的播放器設定，不是原版行為；不寫戰役存檔，也不改遊戲規則。

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
)

// controlActionGroup 是面板的分組。
type controlActionGroup int

const (
	controlGroupMovement controlActionGroup = iota // 移動與游標
	controlGroupConfirm                            // 確認與取消
	controlGroupSystem                             // 系統功能鍵
	controlGroupCount
)

// controlPadButton 是 Xbox 配置（標準手把配置）的實體按鈕；padNone 表示未設定。
type controlPadButton int

const (
	padNone controlPadButton = iota - 1
	padA
	padB
	padX
	padY
	padLB
	padRB
	padLT
	padRT
	padBack
	padStart
	padL3
	padR3
	padUp
	padDown
	padLeft
	padRight
	padButtonCount
)

// padStandardButtons 把面板按鈕對到 Ebiten 的標準手把配置；中央的 Guide 鍵常被系統保留，不開放。
var padStandardButtons = [padButtonCount]ebiten.StandardGamepadButton{
	padA:     ebiten.StandardGamepadButtonRightBottom,
	padB:     ebiten.StandardGamepadButtonRightRight,
	padX:     ebiten.StandardGamepadButtonRightLeft,
	padY:     ebiten.StandardGamepadButtonRightTop,
	padLB:    ebiten.StandardGamepadButtonFrontTopLeft,
	padRB:    ebiten.StandardGamepadButtonFrontTopRight,
	padLT:    ebiten.StandardGamepadButtonFrontBottomLeft,
	padRT:    ebiten.StandardGamepadButtonFrontBottomRight,
	padBack:  ebiten.StandardGamepadButtonCenterLeft,
	padStart: ebiten.StandardGamepadButtonCenterRight,
	padL3:    ebiten.StandardGamepadButtonLeftStick,
	padR3:    ebiten.StandardGamepadButtonRightStick,
	padUp:    ebiten.StandardGamepadButtonLeftTop,
	padDown:  ebiten.StandardGamepadButtonLeftBottom,
	padLeft:  ebiten.StandardGamepadButtonLeftLeft,
	padRight: ebiten.StandardGamepadButtonLeftRight,
}

// padButtonIDs 是 controls-settings.json 與面板使用的按鈕名稱。
var padButtonIDs = [padButtonCount]string{
	padA: "a", padB: "b", padX: "x", padY: "y", padLB: "lb", padRB: "rb", padLT: "lt", padRT: "rt",
	padBack: "back", padStart: "start", padL3: "l3", padR3: "r3",
	padUp: "dpad_up", padDown: "dpad_down", padLeft: "dpad_left", padRight: "dpad_right",
}

func parsePadButton(id string) (controlPadButton, bool) {
	for button, name := range padButtonIDs {
		if name == id {
			return controlPadButton(button), true
		}
	}
	return padNone, false
}

// controlAction 是面板的一列。Logical 是遊戲程式讀的鍵；DefaultPad 是手把預設。
type controlAction struct {
	ID         string
	Group      controlActionGroup
	Logical    ebiten.Key
	DefaultPad controlPadButton
}

// controlActions 的順序就是面板的順序。手把預設依使用者 2026-11-16 定案：十字鍵與左搖桿
// 是方向鍵、A＝Enter、B＝Esc、X＝Space、Start＝開設定面板，肩鍵與 Select 不綁。
var controlActions = []controlAction{
	{"up", controlGroupMovement, ebiten.KeyArrowUp, padUp},
	{"down", controlGroupMovement, ebiten.KeyArrowDown, padDown},
	{"left", controlGroupMovement, ebiten.KeyArrowLeft, padLeft},
	{"right", controlGroupMovement, ebiten.KeyArrowRight, padRight},
	{"confirm", controlGroupConfirm, ebiten.KeyEnter, padA},
	{"cancel", controlGroupConfirm, ebiten.KeyEscape, padB},
	{"space", controlGroupConfirm, ebiten.KeySpace, padX},
	{"back", controlGroupConfirm, ebiten.KeyBackspace, padNone},
	{"next_unit", controlGroupConfirm, ebiten.KeyZ, padNone},
	{"inspect", controlGroupConfirm, ebiten.KeyHome, padNone},
	{"end_turn", controlGroupConfirm, ebiten.KeyTab, padNone},
	{"help", controlGroupSystem, ebiten.KeyF1, padNone},
	{"theme", controlGroupSystem, ebiten.KeyF2, padNone},
	{"audio", controlGroupSystem, ebiten.KeyF3, padNone},
	{"locale", controlGroupSystem, ebiten.KeyF4, padNone},
	{"quick_save", controlGroupSystem, ebiten.KeyF5, padNone},
	{"music", controlGroupSystem, ebiten.KeyF6, padNone},
	{"font", controlGroupSystem, ebiten.KeyF7, padNone},
	{"speed", controlGroupSystem, ebiten.KeyF8, padNone},
	{"quick_load", controlGroupSystem, ebiten.KeyF9, padNone},
	{"poster", controlGroupSystem, ebiten.KeyF10, padNone},
	{"interface", controlGroupSystem, ebiten.KeyF11, padNone},
	{"controls", controlGroupSystem, ebiten.KeyF12, padStart},
}

func controlActionIndex(id string) int {
	for i, action := range controlActions {
		if action.ID == id {
			return i
		}
	}
	return -1
}

const (
	defaultStickDeadzone = 20 // 百分比
	minStickDeadzone     = 5
	maxStickDeadzone     = 60
	stickDeadzoneStep    = 5
)

const keyUnbound = ebiten.Key(-1)

// controlBindings 是每個動作目前的鍵盤鍵與手把按鈕（索引同 controlActions）。
type controlBindings struct {
	Keyboard []ebiten.Key
	Gamepad  []controlPadButton
	Deadzone int
}

func defaultControlBindings() controlBindings {
	b := controlBindings{
		Keyboard: make([]ebiten.Key, len(controlActions)),
		Gamepad:  make([]controlPadButton, len(controlActions)),
		Deadzone: defaultStickDeadzone,
	}
	for i, action := range controlActions {
		b.Keyboard[i], b.Gamepad[i] = action.Logical, action.DefaultPad
	}
	return b
}

func (b controlBindings) clone() controlBindings {
	return controlBindings{
		Keyboard: append([]ebiten.Key(nil), b.Keyboard...),
		Gamepad:  append([]controlPadButton(nil), b.Gamepad...),
		Deadzone: b.Deadzone,
	}
}

// bindableKey 擋掉修飾鍵（Shift／Ctrl／Alt／Meta 留給秘密商店、Alt+Enter 與 Ctrl+F12）與 Esc
// （設定面板的取消鍵）。Esc 仍是「取消」的預設鍵，只能用恢復預設取回。
func bindableKey(k ebiten.Key) bool {
	if k < 0 || k > ebiten.KeyMax {
		return false
	}
	switch k {
	case ebiten.KeyShift, ebiten.KeyShiftLeft, ebiten.KeyShiftRight,
		ebiten.KeyControl, ebiten.KeyControlLeft, ebiten.KeyControlRight,
		ebiten.KeyAlt, ebiten.KeyAltLeft, ebiten.KeyAltRight,
		ebiten.KeyMeta, ebiten.KeyMetaLeft, ebiten.KeyMetaRight, ebiten.KeyEscape:
		return false
	}
	return true
}

func (b controlBindings) keyboardOwner(k ebiten.Key) int {
	for i, bound := range b.Keyboard {
		if bound == k && k != keyUnbound {
			return i
		}
	}
	return -1
}

func (b controlBindings) gamepadOwner(button controlPadButton) int {
	for i, bound := range b.Gamepad {
		if bound == button && button != padNone {
			return i
		}
	}
	return -1
}

// setKeyboard 指定一個動作的鍵盤鍵；同一鍵原本屬於別的動作時，那個動作改成未設定。
// 是否先提示由呼叫端決定（面板先問，讀檔直接套用）。
func (b controlBindings) setKeyboard(action int, k ebiten.Key) {
	if owner := b.keyboardOwner(k); owner >= 0 && owner != action {
		b.Keyboard[owner] = keyUnbound
	}
	b.Keyboard[action] = k
}

func (b controlBindings) setGamepad(action int, button controlPadButton) {
	if owner := b.gamepadOwner(button); owner >= 0 && owner != action {
		b.Gamepad[owner] = padNone
	}
	b.Gamepad[action] = button
}

// clearAction 清除該列的鍵盤與手把。
func (b controlBindings) clearAction(action int) {
	b.Keyboard[action], b.Gamepad[action] = keyUnbound, padNone
}

// resetAction 讓該列回到預設；預設鍵若被別列占用，別列改成未設定。
func (b controlBindings) resetAction(action int) {
	b.setKeyboard(action, controlActions[action].Logical)
	if pad := controlActions[action].DefaultPad; pad != padNone {
		b.setGamepad(action, pad)
	} else {
		b.Gamepad[action] = padNone
	}
}

// resetConflict 回報恢復預設會從哪一列拿走鍵（-1 表示不會）。
func (b controlBindings) resetConflict(action int) int {
	if owner := b.keyboardOwner(controlActions[action].Logical); owner >= 0 && owner != action {
		return owner
	}
	if pad := controlActions[action].DefaultPad; pad != padNone {
		if owner := b.gamepadOwner(pad); owner >= 0 && owner != action {
			return owner
		}
	}
	return -1
}

func normalizeStickDeadzone(v int) int {
	if v < minStickDeadzone || v > maxStickDeadzone {
		return defaultStickDeadzone
	}
	return v - v%stickDeadzoneStep
}

// ---- controls-settings.json ----

// controlsSettingsFile 只記與預設不同的列；空字串表示玩家清除了該列。之後新增的動作沿用預設。
type controlsSettingsFile struct {
	SchemaVersion int               `json:"schema_version"`
	Keyboard      map[string]string `json:"keyboard,omitempty"`
	Gamepad       map[string]string `json:"gamepad,omitempty"`
	StickDeadzone int               `json:"stick_deadzone_percent"`
}

const controlsSettingsSchema = 1

// controlsSettingsPath 與遊戲存檔、fd2_settings.json 分開，放在同一個使用者資料目錄。
func controlsSettingsPath() string { return userDataPath("controls-settings.json") }

type controlsLoadStatus int

const (
	controlsLoadDefault controlsLoadStatus = iota // 沒有檔案
	controlsLoadOK
	controlsLoadPartial // 有無法辨識的列，已略過
	controlsLoadCorrupt // 整份無法讀取，改用預設；下次變更時重寫
)

func loadControlsSettings() (controlBindings, controlsLoadStatus) {
	return loadControlsSettingsFrom(controlsSettingsPath())
}

func loadControlsSettingsFrom(path string) (controlBindings, controlsLoadStatus) {
	b := defaultControlBindings()
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return b, controlsLoadDefault
		}
		return b, controlsLoadCorrupt
	}
	var file controlsSettingsFile
	if err := json.Unmarshal(raw, &file); err != nil || file.SchemaVersion != controlsSettingsSchema {
		return b, controlsLoadCorrupt
	}
	status := controlsLoadOK
	// 依面板順序套用，結果與玩家在面板上逐列確認取代相同，不受 JSON 物件順序影響。
	for i, action := range controlActions {
		if name, ok := file.Keyboard[action.ID]; ok {
			if name == "" {
				b.Keyboard[i] = keyUnbound
			} else {
				var k ebiten.Key
				if k.UnmarshalText([]byte(name)) != nil || !bindableKey(k) && k != action.Logical {
					status = controlsLoadPartial
				} else {
					b.setKeyboard(i, k)
				}
			}
		}
		if name, ok := file.Gamepad[action.ID]; ok {
			if name == "" {
				b.Gamepad[i] = padNone
			} else if button, valid := parsePadButton(name); valid {
				b.setGamepad(i, button)
			} else {
				status = controlsLoadPartial
			}
		}
	}
	for id := range file.Keyboard {
		if controlActionIndex(id) < 0 {
			status = controlsLoadPartial
		}
	}
	for id := range file.Gamepad {
		if controlActionIndex(id) < 0 {
			status = controlsLoadPartial
		}
	}
	if file.StickDeadzone != 0 {
		b.Deadzone = normalizeStickDeadzone(file.StickDeadzone)
		if b.Deadzone != file.StickDeadzone {
			status = controlsLoadPartial
		}
	}
	return b, status
}

func (b controlBindings) settingsFile() controlsSettingsFile {
	file := controlsSettingsFile{SchemaVersion: controlsSettingsSchema, StickDeadzone: b.Deadzone}
	for i, action := range controlActions {
		if k := b.Keyboard[i]; k != action.Logical {
			if file.Keyboard == nil {
				file.Keyboard = map[string]string{}
			}
			name := ""
			if k != keyUnbound {
				text, _ := k.MarshalText()
				name = string(text)
			}
			file.Keyboard[action.ID] = name
		}
		if button := b.Gamepad[i]; button != action.DefaultPad {
			if file.Gamepad == nil {
				file.Gamepad = map[string]string{}
			}
			name := ""
			if button != padNone {
				name = padButtonIDs[button]
			}
			file.Gamepad[action.ID] = name
		}
	}
	return file
}

func saveControlsSettings(b controlBindings) error {
	return saveControlsSettingsTo(controlsSettingsPath(), b)
}

// saveControlsSettingsTo 先寫暫存檔再改名，中途失敗不會留下半份設定。
func saveControlsSettingsTo(path string, b controlBindings) error {
	raw, err := json.MarshalIndent(b.settingsFile(), "", " ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".controls-settings-*.json")
	if err != nil {
		return err
	}
	name := temp.Name()
	_, writeErr := temp.Write(append(raw, '\n'))
	closeErr := temp.Close()
	if writeErr != nil || closeErr != nil {
		os.Remove(name)
		return fmt.Errorf("write %s: %v %v", filepath.Base(path), writeErr, closeErr)
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}
