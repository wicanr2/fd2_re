// Package campaign — 劇本節點圖系統(doc 19):把「固定線性流程」變成可分支的有向圖。
// 節點 = 一個遊戲段落(story/battle/town/preparation/choice/event/ending),轉場依結果(win/lose/next/optN)
// 與旗標決定下一節點;敗北可走敗北路線而非 game over。
package campaign

import (
	"encoding/json"
	"fmt"
	"os"
)

// Line 一句對話(story 節點內嵌)。Speaker 是靜態或稽核用 DATO 頭像 id；
// SpeakerSlot 保存原版 FFED/FFEC 的 runtime unit direct index，執行時必須
// 從該 unit 的 Portrait 解析，不能把 slot 數字誤當全域角色 id。
type Line struct {
	Speaker     int    `json:"speaker"`
	SpeakerSlot *int   `json:"speaker_slot,omitempty"`
	Upper       *bool  `json:"upper,omitempty"`
	Text        string `json:"text"`
}

// Option choice 節點的選項;If 非空時需旗標為真才顯示。
type Option struct {
	Label string `json:"label"`
	To    string `json:"to"`
	If    string `json:"if,omitempty"`
}

// NativeTownSecretGate is the editable form of the 0x2cde0..0x2cef7
// three-byte town-record gate. Hidden selection 5 is entered only when the
// current visible selection and BIOS function-key scan both match.
type NativeTownSecretGate struct {
	Selection int    `json:"selection"`
	ScanCode  int    `json:"scan_code"`
	To        string `json:"to"`
}

// Good 商店商品(名稱/價格;id 對映 EXE item.json)。
type Good struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

// Actor story+Map 場景背景上的靜態角色(cutscene NPC/主角,無 AI/戰鬥邏輯,純擺位;
// Fig=地圖 sprite 組 id(同 battle.Unit.Fig,恆等於角色 id,doc31);Dir:0下1左2上3右。
// 座標/肖像多可直接取自 FDFIELD 該地圖出場位置段(見 tools/parse_field.py positions),
// 見 doc23 §4 補述(map32 王座廳:國王portrait48@(7,5)/王后portrait66@(10,5))。
//
// FromX/FromY/WalkFrames(doc46 §5.3,走位動畫):若 WalkFrames>0 且 From 與 X/Y 不同,
// 進節點時該角色從 (FromX,FromY) 走位到 (X,Y),耗時 WalkFrames 幀(60fps);
// 例:map31 密林「發現悠妮與蓋亞」一幕,索爾/亞雷斯從比劍點走到發現點(FDFIELD 出場位置證實
// 兩組座標相距 14 格,非同格瞬移,見 doc46 §2)。
type Actor struct {
	Fig        int `json:"fig"`
	X          int `json:"x"`
	Y          int `json:"y"`
	Dir        int `json:"dir,omitempty"`
	FromX      int `json:"from_x,omitempty"`
	FromY      int `json:"from_y,omitempty"`
	WalkFrames int `json:"walk_frames,omitempty"`
}

// ActorWalk 節點「退場」走位動畫(doc46 §5.3):對白播完、換場前,已在場的 actor 先走一段路
// 再淡出(例:王座廳索爾對白說完沿紅毯走下場,~1.5s)。Fig 指定 Node.Actors 裡哪一個角色。
type ActorWalk struct {
	Fig    int  `json:"fig"`
	ToX    int  `json:"to_x"`
	ToY    int  `json:"to_y"`
	Frames int  `json:"frames"`
	Dir    *int `json:"dir,omitempty"` // 走完後面向(指標,nil=保留走位末向;指定則覆蓋,如索爾走到亞雷斯旁定住面右)
}

// ActingUnit is one original acting frame target.  Slot is the original
// FDFIELD/unit-array index and is the authoritative identity for decoded
// 0x1366a data: a roster can contain many guards with the same Fig.  Fig is a
// legacy fallback for hand-authored scenes that have no original roster; an
// editable transcription of a decoded handler must use Slot instead.  Pose
// follows the original direction encoding: 0 down, 1 left, 2 up, 3 right.
type ActingUnit struct {
	Slot *int `json:"slot,omitempty"`
	Fig  int  `json:"fig,omitempty"`
	Pose int  `json:"pose"`
}

// ActingFrame 是原版 0x1366a 資源的一幀行為轉錄，不包含原始 bytes。
// Special=false = bit7=0 正常模式：每個 Beat 沿 Pose 移動一格；Special=true = bit7=1：
// 原地顯示／姿態，Beat 只表示停留節奏。規則見 doc50 §1.2。
type ActingFrame struct {
	Beats   int          `json:"beats"`
	Special bool         `json:"special,omitempty"`
	Units   []ActingUnit `json:"units"`
}

// LoadCHState is the editable remake state selected by an original LOADCH
// call.  The original routine loads all three together: FDFIELD map, FDFIELD
// roster, and FDTXT (doc23 §4).  Keeping those paths in one value makes an
// incomplete reconstruction impossible to mistake for a harmless map change.
// Paths are asset-root relative (for example assets/maps/map5/map5_units.json)
// rather than relative to the handler binding file.
type LoadCHState struct {
	Chapter       int    `json:"chapter"`
	Map           string `json:"map"`
	Roster        string `json:"roster"`
	SlotCount     int    `json:"slot_count"`
	Script        string `json:"script"`
	PartyScenario string `json:"party_scenario,omitempty"` // persistent party constructed before FDFIELD groups
	PartyOrder    []int  `json:"party_order,omitempty"`    // original JOIN chronology; direct-replay fallback and runtime assertion
	CamX          int    `json:"cam_x,omitempty"`
	CamY          int    `json:"cam_y,omitempty"`
	CamMaxY       int    `json:"cam_max_y,omitempty"`
}

// BeatCondition is the runtime form of a proven handler predicate.
type BeatCondition struct {
	Op                       string          `json:"op"`
	UnitSlots                []int           `json:"unit_slots,omitempty"`
	NativeInventoryItemID    *int            `json:"native_inventory_item_id,omitempty"`
	NativePersistentIdentity *int            `json:"native_persistent_identity,omitempty"`
	Threshold                *int            `json:"threshold,omitempty"`
	NativeRound              *int            `json:"native_round,omitempty"`
	NativeRecordWordOffset   *int            `json:"native_record_word_offset,omitempty"`
	NativeRecordWordValue    *int            `json:"native_record_word_value,omitempty"`
	UnitSlot                 *int            `json:"unit_slot,omitempty"`
	Any                      []BeatCondition `json:"any,omitempty"`
	CharID                   *int            `json:"char_id,omitempty"`
	EventStateIndex          *int            `json:"event_state_index,omitempty"`
	EventStateValue          *int            `json:"event_state_value,omitempty"`
	RequiredSlotCount        *int            `json:"required_slot_count,omitempty"`
}

// HandlerUnitLayout is one absolute runtime-slot placement recovered from a
// native post-battle layout routine. Coordinates remain original map tiles;
// CamX/CamY below are the verified remake pixel origin.
type HandlerUnitLayout struct {
	Slot int `json:"slot"`
	X    int `json:"x"`
	Y    int `json:"y"`
	Pose int `json:"pose"`
}

type HandlerLayout struct {
	Units []HandlerUnitLayout `json:"units"`
	CamX  int                 `json:"cam_x"`
	CamY  int                 `json:"cam_y"`
}

// NativeMapViewConfig is an explicitly sourced original 13x8 tactical view.
// Values are map tiles, not remake pixels.
type NativeMapViewConfig struct {
	CameraX        int  `json:"camera_x"`
	CameraY        int  `json:"camera_y"`
	CursorX        int  `json:"cursor_x"`
	CursorY        int  `json:"cursor_y"`
	VisibleCursorX int  `json:"visible_cursor_x"`
	VisibleCursorY int  `json:"visible_cursor_y"`
	RangeMode      *int `json:"range_mode"`
}

// NativeMapHUDConfig carries only the persistent 0x1acf3 raw globals. Gate
// bytes remain byte-valued because native code tests nonzero, not boolean one.
type NativeMapHUDConfig struct {
	DisplayGateA int `json:"display_gate_a"`
	DisplayGateB int `json:"display_gate_b"`
	AnchorX      int `json:"anchor_x"`
}

// NativeMapHUDInheritedConfig declares a battle-controller gate B while gate A
// and the anchor are inherited from their separately proven persistent owner.
// The original currently proves only controller entry value 1.
type NativeMapHUDInheritedConfig struct {
	DisplayGateB int `json:"display_gate_b"`
}

// HandlerIndexedTransition records the recovered 0x24618 double-buffer
// choreography. Tile geometry, radial radius progression, and row bounds are
// kept explicit because this is not a generic fade; the PNG renderer may only
// execute it once an indexed descriptor adapter is available.
type HandlerIndexedTransition struct {
	TileX int `json:"tile_x"`
	TileY int `json:"tile_y"`
	// CursorSource optionally resolves the first two native 0x24618 arguments
	// from the battle's raw relative cursor globals.  The only accepted source
	// is native_relative_cursor, which preserves [0x53ab9]/[0x53abd] rather than
	// replacing them with a guessed absolute map coordinate.
	CursorSource  string `json:"cursor_source,omitempty"`
	CursorXOffset int    `json:"cursor_x_offset,omitempty"`
	CursorYOffset int    `json:"cursor_y_offset,omitempty"`
	// RadialRadius and RadialRadiusStep are the third/fourth native arguments.
	// 0x24618 advances the radius on every pass; 0x22046 passes it to the two
	// 0x219ad radial LUT-remap calls and derives its final rect radius from it.
	// They are not source-buffer coordinates or the radial scale (which is
	// fixed to 16 by 0x22046).
	RadialRadius     int `json:"radial_radius"`
	RadialRadiusStep int `json:"radial_radius_step"`
	// StartY/EndY are the row range passed to 0x22046; this routine reads its
	// indexed buffers from globals, so these are not source-image coordinates
	// or a blit width. EndY is exclusive.
	StartY            int `json:"start_y"`
	EndY              int `json:"end_y"`
	ClipWidth         int `json:"clip_width"`
	ClipHeight        int `json:"clip_height"`
	Frames            int `json:"frames"`
	FrameDelayMs      int `json:"frame_delay_ms"`
	TailDelayMs       int `json:"tail_delay_ms"`
	PaletteRangeStart int `json:"palette_range_start"`
	PaletteRangeEnd   int `json:"palette_range_end"`
	PaletteDeltaStart int `json:"palette_delta_start"`
	PaletteDeltaEnd   int `json:"palette_delta_end"`
	PaletteDeltaStep  int `json:"palette_delta_step"`
	PaletteDelayMs    int `json:"palette_delay_ms"`
}

// NativePaletteFadeOut is the exact 0x1f882 DAC-darkening schedule.  It is
// separate from palette_update (0x11df2) and generic story fades: an indexed
// DAC adapter must execute every inclusive step before a handler can play it.
type NativePaletteFadeOut struct {
	Start   int `json:"start"`
	End     int `json:"end"`
	DelayMs int `json:"delay_ms"`
}

// NativePaletteFadeIn is the exact 0x1f525 whole-DAC schedule. The helper
// writes baseline-minus-delta for inclusive deltas 64..0, waiting 2ms after
// every write. It is not the remake's generic RGBA story fade.
type NativePaletteFadeIn struct {
	Start   int `json:"start"`
	End     int `json:"end"`
	DelayMs int `json:"delay_ms"`
}

// HandlerRawUnitBytePatch keeps an original record offset visible instead of
// assigning a gameplay name from one handler write. Runtime currently accepts
// only offsets with an independently preserved typed storage location.
type HandlerRawUnitBytePatch struct {
	Offset int `json:"offset"`
	Value  int `json:"value"`
}

// HandlerUnitRecordPatch is a sparse direct-write transcription. Pointer
// fields distinguish an explicit zero from an omitted write; untouched record
// bytes remain untouched.
type HandlerUnitRecordPatch struct {
	Slot     int                       `json:"slot"`
	X        *int                      `json:"x,omitempty"`
	Y        *int                      `json:"y,omitempty"`
	Pose     *int                      `json:"pose,omitempty"`
	RawBytes []HandlerRawUnitBytePatch `json:"raw_bytes,omitempty"`
}

// HandlerDirectRecordPatch represents hard-coded writes that occur between
// calls and were therefore invisible to the call-only handler exporter. View
// uses the already recovered raw camera/cursor globals; values stay additive
// evidence tied to the source address in Beat.Source.
type HandlerDirectRecordPatch struct {
	Units []HandlerUnitRecordPatch `json:"units"`
	View  *NativeMapViewConfig     `json:"view,omitempty"`
}

// NativePalettePulse is the exact whole-DAC pulse performed by 0x35e5a.
// It is deliberately not represented as a generic fade: the first and
// second ramps have different inclusive bounds and are separated by a hold.
// An indexed DAC adapter must execute this schedule before a handler can play
// it.
type NativePalettePulse struct {
	RiseStart   int `json:"rise_start"`
	RiseEnd     int `json:"rise_end"`
	RiseDelayMs int `json:"rise_delay_ms"`
	HoldMs      int `json:"hold_ms"`
	FallStart   int `json:"fall_start"`
	FallEnd     int `json:"fall_end"`
	FallDelayMs int `json:"fall_delay_ms"`
}

// NativePaletteBlackout is the exact ch07 post-battle terminal operation:
// 0x11d40 subtracts 64 from every six-bit DAC component in entries 0..255,
// which clamps the complete palette to black, and the caller immediately
// clears all 0xFA00 mode-13h framebuffer bytes.  Keep the framebuffer clear
// extent explicit so this cannot become a generic or guessed fade.
type NativePaletteBlackout struct {
	Start      int `json:"start"`
	End        int `json:"end"`
	Delta      int `json:"delta"`
	ClearBytes int `json:"clear_bytes"`
}

// NativeCh20SkyKeySequence 是 raw ch20 post 的固定合成演出契約。它保留
// 0x24336 內部的鏡頭格、FDOTHER 幀界線、AFM 資源與兩段色盤停留，不把
// 這段 caller-specific choreography 泛化成任意動畫播放器。
type NativeCh20SkyKeySequence struct {
	PanGridX           int  `json:"pan_grid_x"`
	PanGridY           int  `json:"pan_grid_y"`
	FDOTHERResource    int  `json:"fdother_resource"`
	FDOTHERFrameCount  int  `json:"fdother_frame_count"`
	BaseFrame          int  `json:"base_frame"`
	FirstFrameStart    int  `json:"first_frame_start"`
	FirstFrameEnd      int  `json:"first_frame_end"`
	FrameWaitBIOSTicks int  `json:"frame_wait_bios_ticks"`
	PaletteCycleFirst  bool `json:"palette_cycle_first"`
	ANIResource        int  `json:"ani_resource"`
	ANIFrameCount      int  `json:"ani_frame_count"`
	ANIFrameDelayMs    int  `json:"ani_frame_delay_ms"`
	ANISkippable       bool `json:"ani_skippable"`
	PaletteStart       int  `json:"palette_start"`
	PaletteEnd         int  `json:"palette_end"`
	FlashDelta         int  `json:"flash_delta"`
	FlashHoldMs        int  `json:"flash_hold_ms"`
	RestoreDelta       int  `json:"restore_delta"`
	RestoreHoldMs      int  `json:"restore_hold_ms"`
	TailFrameStart     int  `json:"tail_frame_start"`
	TailFrameEnd       int  `json:"tail_frame_end"`
}

// IsRecoveredContract 限制正式執行器只接受 IDA／Capstone 已共同固定的
// 0x24336 payload。可編輯 JSON 仍完整顯示每個值，但改動後必須先補新證據，
// 不會悄悄取得原版語意。
func (s *NativeCh20SkyKeySequence) IsRecoveredContract() bool {
	return s != nil &&
		s.PanGridX == 14 && s.PanGridY == 8 &&
		s.FDOTHERResource == 34 && s.FDOTHERFrameCount == 101 &&
		s.BaseFrame == 0 && s.FirstFrameStart == 1 && s.FirstFrameEnd == 68 &&
		s.FrameWaitBIOSTicks == 3 && s.PaletteCycleFirst &&
		s.ANIResource == 0 && s.ANIFrameCount == 96 &&
		s.ANIFrameDelayMs == 15 && !s.ANISkippable &&
		s.PaletteStart == 0 && s.PaletteEnd == 255 &&
		s.FlashDelta == 63 && s.FlashHoldMs == 100 &&
		s.RestoreDelta == 0 && s.RestoreHoldMs == 500 &&
		s.TailFrameStart == 69 && s.TailFrameEnd == 100
}

// NativeStagingPresent preserves the exact 0x33f78 wrapper ABI. It focuses
// (FocusX, FocusY), then invokes 0x22253 with (Slot, X, Y, X, Y). The callee
// has a recovered 11+6+10 indexed choreography. The shared battle-state
// presenter exists, but this wrapper remains data-only until its preceding
// focus and story-runtime-array owner are wired without guessing.
type NativeStagingPresent struct {
	Slot   int `json:"slot"`
	X      int `json:"x"`
	Y      int `json:"y"`
	FocusX int `json:"focus_x"`
	FocusY int `json:"focus_y"`
}

// NativeUnitPresent preserves 0x22253's five-argument ABI. LastRuntimeSlot is
// source-specific evidence for ch28 post's [0x53BEB]-1 caller; it is not a
// general script selector and is accepted only for call site 0x25535.
type NativeUnitPresent struct {
	Slot            int  `json:"slot,omitempty"`
	LastRuntimeSlot bool `json:"last_runtime_slot,omitempty"`
	NewX            int  `json:"new_x"`
	NewY            int  `json:"new_y"`
	VisualX         int  `json:"visual_x"`
	VisualY         int  `json:"visual_y"`
}

// NativeCh28PostPresent preserves sub_35BBA(20) plus its mandatory sub_1DB65
// consumer. Fields are raw archive/index/buffer contracts, not inferred visual
// or audio names.
type NativeCh28PostPresent struct {
	StartSlot       int `json:"start_slot"`
	PoseFrames      int `json:"pose_frames"`
	FDOTHERResource int `json:"fdother_resource"`
	EntryFirst      int `json:"entry_first"`
	EntryLast       int `json:"entry_last"`
	SFXResource     int `json:"sfx_resource"`
	SFXIndex        int `json:"sfx_index"`
	SFXArg          int `json:"sfx_arg"`
	WorkSize        int `json:"work_size"`
	WorkStride      int `json:"work_stride"`
	ViewBase        int `json:"view_base"`
	VisibleColumns  int `json:"visible_columns"`
	VisibleRows     int `json:"visible_rows"`
}

func (p NativeCh28PostPresent) IsRecoveredContract() bool {
	return p.StartSlot == 20 && p.PoseFrames == 13 && p.FDOTHERResource == 5 &&
		p.EntryFirst == 0x44 && p.EntryLast == 0x4f && p.SFXResource == 31 &&
		p.SFXIndex == 3 && p.SFXArg == 1 && p.WorkSize == 0x25680 &&
		p.WorkStride == 0x1c8 && p.ViewBase == 0x8088 &&
		p.VisibleColumns == 13 && p.VisibleRows == 8
}

// HandlerUnitPresent retains the formerly recovered metadata shape for native
// 0x22253. It is not currently compilable: later direct trace found 11+6+10
// presentation phases, so this six-frame schema is deliberately rejected
// until a full indexed choreography representation exists.
type HandlerUnitPresent struct {
	Slot         int `json:"slot"`
	X            int `json:"x"`
	Y            int `json:"y"`
	Frames       int `json:"frames"`
	FrameDelayMs int `json:"frame_delay_ms"`
	TailTicks    int `json:"tail_ticks"`
}

// Beat 過場原語(doc 50 §1/§2):cutscene 節點的 beats 通常依序執行；if 會在 runtime
// 選一條 structured arm 插入目前拍之後，再回到共同 continuation。
// 一比一對映原版 EXE handler 的呼叫序列(LOADCH/PAN/TXT/ACT/SPAWN/JOIN/BGM/FADE/DELAY)。
// 每個 op 只用到自己相關的欄位,其餘留零值即可(同 Node 的稀疏欄位風格)。
type Beat struct {
	Op                    string                    `json:"op"`               // loadch/pan/walk/dialog/act/spawn/spawn_intro/deactivate_unit/reactivate_nonzero_hp/reset_pose/redraw/...
	Source                string                    `json:"source,omitempty"` // original handler call-site; empty for authored-only beats
	Condition             *BeatCondition            `json:"condition,omitempty"`
	Then                  []Beat                    `json:"then,omitempty"`
	Else                  []Beat                    `json:"else,omitempty"`
	RuntimeContext        *HandlerRuntimeContext    `json:"runtime_context,omitempty"`
	Layout                *HandlerLayout            `json:"layout,omitempty"`
	IndexedTransition     *HandlerIndexedTransition `json:"indexed_transition,omitempty"`
	NativePaletteFade     *NativePaletteFadeOut     `json:"native_palette_fade_out,omitempty"`
	NativePaletteFadeIn   *NativePaletteFadeIn      `json:"native_palette_fade_in,omitempty"`
	NativePalettePulse    *NativePalettePulse       `json:"native_palette_pulse,omitempty"`
	NativePaletteBlackout *NativePaletteBlackout    `json:"native_palette_blackout,omitempty"`
	NativeCh20SkyKey      *NativeCh20SkyKeySequence `json:"native_ch20_sky_key_sequence,omitempty"`
	NativeStagingPresent  *NativeStagingPresent     `json:"native_staging_present,omitempty"`
	NativeUnitPresent     *NativeUnitPresent        `json:"native_unit_present,omitempty"`
	NativeCh28PostPresent *NativeCh28PostPresent    `json:"native_ch28_post_present,omitempty"`
	NativeCh23Loop        *NativeCh23Loop           `json:"native_ch23_loop,omitempty"`
	Native2189ALoop       *Native2189ALoop          `json:"native_2189a_loop,omitempty"`
	UnitPresent           *HandlerUnitPresent       `json:"unit_present,omitempty"`
	DirectRecordPatch     *HandlerDirectRecordPatch `json:"direct_record_patch,omitempty"`

	// loadch: atomically replace the active map, FDFIELD roster and FDTXT
	// story context.  It is deliberately a nested required state object so a
	// handler cannot compile a map-only imitation of original 0x205da.
	LoadCH *LoadCHState `json:"loadch,omitempty"`

	// pan/walk 共用:目標格(walk 用 X/Y 當終點);pan 的 X/Y 沿用 Node.CamX/CamY 語意——
	// 已由畫面回饋校準的「像素座標」,不是 doc47 §3 原始 grid(col,row)值(grid→px 未逐點驗證,
	// 不自行換算,見 rulebook 62)。
	X        int  `json:"x,omitempty"`
	Y        int  `json:"y,omitempty"`
	FromX    int  `json:"from_x,omitempty"` // walk 起點;省略=沿用該角色目前座標(接續上一拍)
	FromY    int  `json:"from_y,omitempty"`
	Fig      int  `json:"fig,omitempty"`       // walk/act:對應 Node.Actors 裡的角色(依 Fig 尋找,同 ActorWalk)
	Slot     *int `json:"slot,omitempty"`      // original runtime unit-array slot; identity-critical handler primitives only
	Frames   int  `json:"frames,omitempty"`    // pan/walk/fade 位移或漸變幀數;delay 用幀數(見 Ms)
	TileStep bool `json:"tile_step,omitempty"` // pan:0x135dd 每 tick 先 X 後 Y 移一個 tile
	Follow   bool `json:"follow,omitempty"`    // walk:走位期間鏡頭鎖定跟隨(doc47 §9,同 Node.FollowWalk 機制)
	Dir      *int `json:"dir,omitempty"`       // walk:走完後面向(指標,nil=保留走位末向;指定則面向它,如索爾走前面轉身面向亞雷斯)
	Steps    int  `json:"steps,omitempty"`     // scroll_step:原版 0x13185 的重複上移格數
	// palette_update: one-shot VGA DAC range update recovered from 0x11df2.
	PaletteStart int `json:"palette_start,omitempty"`
	PaletteEnd   int `json:"palette_end,omitempty"`
	PaletteDelta int `json:"palette_delta,omitempty"`
	// transition_reveal: native 0x24b4d alternating-buffer present loop.
	RevealFrames  int `json:"reveal_frames,omitempty"`
	RevealDelayMs int `json:"reveal_delay_ms,omitempty"`

	// act:Acting 非空時播放原版 acting frame 的行為轉錄：正常 frame 每 Beat 依 Pose 搬一格，
	// special frame 只原地換姿態(doc50 §1.2)。Poses/PoseFrames 是舊的原地姿態近似欄位，
	// 為舊場景相容而保留；不可與 Acting 混用。
	Acting     []ActingFrame `json:"acting_frames,omitempty"`
	Poses      []int         `json:"poses,omitempty"`
	PoseFrames int           `json:"pose_frames,omitempty"` // 每個 pose 停留幀數(預設見 main.go)

	// dialog:章文本第 Line 條起連續 Count 句(Count 省略=1)。Line 對應目前節點 Script+Scene
	// 載入的那份 lines(同 Node.Scene 語意),不是 FDTXT 原始 idx(譯文精校版常把一條原文拆成
	// 多句對白,見 doc47 §7 教訓:機制懂了但內容沒逐句對齊前不假裝一一對應)。
	Line       int    `json:"line,omitempty"`
	Count      int    `json:"count,omitempty"`
	Script     string `json:"script,omitempty"`      // handler compiler context; empty=Node.Script
	Scene      string `json:"scene,omitempty"`       // handler compiler context; empty can mean unlabeled scene
	SceneIndex *int   `json:"scene_index,omitempty"` // authoritative for unlabeled/reused scene labels

	// Upper:dialog 對話框上下位置覆蓋(指標,nil=沿用預設規則「說話者 id>=32 走上框」)。
	// 草地撞見幕實測(doc55 截圖 18-03-10):亞雷斯(id4,<32)進場那句仍走上框——原版並非單純按
	// id 分上下框,推測與進場/位置有關,尚未逆得通則;先開這個 per-beat 覆蓋做最小修正,別動全域規則。
	Upper *bool `json:"upper,omitempty"`

	Group int `json:"group,omitempty"` // spawn:原版 FDFIELD 群組編號
	// RawPlacementGate retains the exact per-call [0x53AFA] byte. Handler
	// compiler output always supplies it; nil is reserved for authored legacy
	// campaigns which intentionally use the normalized compatibility append.
	RawPlacementGate *int `json:"raw_placement_gate,omitempty"`
	// CharID is JOIN's permanent-player identity.  It intentionally remains
	// separate from a scene actor's Fig/portrait: JOIN accepts only the
	// original 0..31 player roster, while a cutscene may contain arbitrary
	// NPC portraits (for example map31's shop-clerk portrait 75).
	CharID int    `json:"char_id,omitempty"`
	Track  string `json:"track,omitempty"` // bgm:曲目 id(對映 assets/bgm)

	Out bool `json:"out,omitempty"` // fade:true=淡出 false=淡入(重用 storyFade,doc46 §5.2)

	Ms int `json:"ms,omitempty"` // delay:毫秒(原版 0x375b2 語意);換算成 60fps 幀數,Frames 優先

	// set_chapter: original [0x53c03] campaign/resource chapter assignment.
	// Pointer form preserves chapter zero as an explicit editable value.
	Chapter *int `json:"chapter,omitempty"`

	// grant_item: original unsigned-byte item identity.
	ItemID     *int `json:"item_id,omitempty"`
	ResourceID *int `json:"resource_id,omitempty"`
	// ResourceArchive/Owner preserve 0x111BA's filename and old/new handle
	// owner when a bare numeric archive index would be ambiguous.
	ResourceArchive string `json:"resource_archive,omitempty"`
	ResourceOwner   string `json:"resource_owner,omitempty"`
	SFXIndex        *int   `json:"sfx_index,omitempty"`
}

// NativeEndingPrefixConfig 是已證實原版結局前綴的狹義戰役入口。它保留精確
// timeline／handler，不會把不完整的 ending decoder 變成泛用終局場景。runtime
// 只會在 FD2_APPROXIMATE=1 接受它；預設忠實模式仍在未還原尾段失敗即關閉。
type NativeEndingPrefixConfig struct {
	Timeline string `json:"timeline"`
	Handler  string `json:"handler"`
	Chapter  int    `json:"chapter"`
	Mode     string `json:"mode"`
}

const NativeEndingPrefixRecoveredOnly = "recovered_prefix_only_fail_closed"

// IsRecoveredPrefixContract 即使呼叫端在記憶體內建構 Campaign、略過 JSON
// 載入驗證，仍將唯一支援的結局前綴限制在已證實合約內。
func (c *NativeEndingPrefixConfig) IsRecoveredPrefixContract() bool {
	return c != nil && c.Timeline == "assets/endings/native_2bce5.json" &&
		c.Handler == "0x2bce5" && (c.Chapter == 26 || c.Chapter == 29) &&
		c.Mode == NativeEndingPrefixRecoveredOnly
}

// Node 節點。Type: story / cutscene / battle / town / preparation / church / choice /
// inventory_gate / inventory_recipe / event / shop / ending。
// cutscene(doc 50):story 的 beats 驅動版——用 Beats 一比一承接原版章 handler 的原語序列,
// 對白與走位/演出天然交錯(平面序列,非「一幕一段」)。Map/Actors/BGM/ExitWalk(s) 等欄位與
// story 共用同一套場景設置(進節點時的初始擺位、退場走位、淡出轉場),Beats 只負責節點「進行中」
// 的编排;story 節點型別保留相容,兩者可並存於同一 campaign(逐步遷移,doc50 §2)。
type Node struct {
	Type     string `json:"type"`
	Lines    []Line `json:"lines,omitempty"`    // story:對白(內嵌;Script 有檔時被覆蓋)
	Script   string `json:"script,omitempty"`   // story:本機劇情文本檔(assets/story/chNN.json,不入庫)
	Scenario string `json:"scenario,omitempty"` // battle:戰場事件劇本檔
	Map      string `json:"map,omitempty"`      // battle:戰場資產目錄;story:場景背景圖(doc23 §4:
	// 原版序幕王城/草地背景是 FDFIELD map32 複合場景,與戰場同一渲染器非另開圖片系統;
	// story 填同一 assets/maps/mapN 目錄即可換場景背景;battle 空=沿用當前)
	Units                 string                       `json:"units,omitempty"` // battle:單位配置檔
	CamX                  int                          `json:"cam_x,omitempty"` // story+Map:固定鏡頭像素座標(場景不跟游標走,取代預設 focusOnParty)
	CamY                  int                          `json:"cam_y,omitempty"`
	NativeMapView         *NativeMapViewConfig         `json:"native_map_view,omitempty"`
	NativeMapHUD          *NativeMapHUDConfig          `json:"native_map_hud,omitempty"`
	NativeMapHUDInherited *NativeMapHUDInheritedConfig `json:"native_map_hud_inherited,omitempty"`
	Actors                []Actor                      `json:"actors,omitempty"` // story+Map:場景背景上的靜態角色擺位
	Scene                 string                       `json:"scene,omitempty"`  // story+Script:只取 Script 檔裡 label 對映的那個 scene(doc46 §5.2;
	// 空=舊行為,整份 Script 攤平全部 scenes 成一條對白隊列——別讓一個節點播完整份劇本)
	ExitWalk  *ActorWalk  `json:"exit_walk,omitempty"`  // story:對白播完、換場前先走一段路再淡出(doc46 §5.3;單一角色)
	ExitWalks []ActorWalk `json:"exit_walks,omitempty"` // 多角色同時退場(全部走完才轉場)。
	// ⚠ 更正(doc55 逐幀量測 2026-07-05):早前「草地幕兩人一起走離」是錯的——實測=亞雷斯對話中先走近、
	// 索爾對話後才單獨走到亞雷斯旁、隨即淡出(無一起走離畫面)。草地幕已改「亞雷斯進場走位+索爾單人 ExitWalk」;
	// 本欄保留供其他真有「多人同時退場」的幕使用;並行多角色不在 Beat.walk 單角色設計內,收尾沿用本欄不重造輪子)
	Beats          []Beat `json:"beats,omitempty"`           // cutscene:過場原語序列(doc 50);Beats 跑完後走 ExitWalk(s)+淡出+Advance,同 story 節點收尾
	HandlerBinding string `json:"handler_binding,omitempty"` // cutscene:editable handler binding; runtime must reject unresolved compile issues
	AutoAdvance    int    `json:"auto_advance,omitempty"`    // story:無對白/Script 時,進節點後幾幀自動轉場(doc46 行軍蒙太奇)
	WalkFirst      bool   `json:"walk_first,omitempty"`      // story:進場走位全走完才顯示對白(2-1:王座廳索爾沿紅毯走到王座前對話框才出現)
	FollowWalk     bool   `json:"follow_walk,omitempty"`     // story:走位期間鏡頭鎖定跟隨走位者(原版 13×8 格視野長廊運鏡,doc25 0x11eee)
	CamMaxY        int    `json:"cam_max_y,omitempty"`       // story:鏡頭 Y 上限(px;0=不限)。王座廳=808 擋住 map32 底部草地段
	// (原版第一幕畫面無草地,索爾從畫面外沿紅毯走入,使用者回饋 2026-07-04 #1)
	BGM    string `json:"bgm,omitempty"`
	Next   string `json:"next,omitempty"`    // story/event
	OnWin  string `json:"on_win,omitempty"`  // battle
	OnLose string `json:"on_lose,omitempty"` // battle(敗北路線;空=game over)
	// ApproximateWinSync 是重製近似模式的終局資料邊界；只允許勝利直接進
	// ending 的 battle 使用，且不可當作原版 handler 證據。
	ApproximateWinSync bool                      `json:"approximate_sync_party_on_win,omitempty"`
	Protect            string                    `json:"protect,omitempty"`             // battle:保護目標；空值沿用主角索爾
	ItemID             *int                      `json:"item_id,omitempty"`             // inventory_gate:原版 unsigned-byte item identity
	IfPresent          string                    `json:"if_present,omitempty"`          // inventory_gate:全隊任一角色持有 ItemID
	IfMissing          string                    `json:"if_missing,omitempty"`          // inventory_gate:全隊皆未持有 ItemID
	ItemIDs            []int                     `json:"item_ids,omitempty"`            // inventory_recipe:逐 item×runtime slot 計數／移除
	SlotCount          int                       `json:"slot_count,omitempty"`          // inventory_recipe:只掃前 N 個 runtime records
	RequiredMatches    int                       `json:"required_matches,omitempty"`    // inventory_recipe:原版要求的精確命中組合數
	RewardItemID       *int                      `json:"reward_item_id,omitempty"`      // inventory_recipe:成功後 grant 的 item
	IfCrafted          string                    `json:"if_crafted,omitempty"`          // inventory_recipe:成功 arm
	IfInsufficient     string                    `json:"if_insufficient,omitempty"`     // inventory_recipe:命中數不符 arm
	Prompt             string                    `json:"prompt,omitempty"`              // choice/preparation
	PartyLimit         int                       `json:"party_limit,omitempty"`         // preparation: original 0x318ad selection cap (15, late route 19)
	Cancel             string                    `json:"cancel,omitempty"`              // preparation: native confirmation/selection cancellation target
	Town               string                    `json:"town,omitempty"`                // town:原版戰後城鎮/營地名稱(可編輯、可存檔的整備 hub)
	NativeTownVariant  *int                      `json:"native_town_variant,omitempty"` // town:0/1/2→FDOTHER#11/#61/#62
	NativeSecretGate   *NativeTownSecretGate     `json:"native_secret_gate,omitempty"`  // town:selection+BIOS scan→reveal selection5
	Options            []Option                  `json:"options,omitempty"`             // choice
	SetFlags           map[string]bool           `json:"set_flags,omitempty"`
	Text               string                    `json:"text,omitempty"` // ending:結語
	NativeEndingPrefix *NativeEndingPrefixConfig `json:"native_ending_prefix,omitempty"`
	Goods              []Good                    `json:"goods,omitempty"`              // shop:商品
	NativeHubVariant   int                       `json:"native_hub_variant,omitempty"` // shop:0=custom/generic; original owner uses 1 weapon, 3 item, 5 secret
	Secret             []Good                    `json:"secret,omitempty"`             // shop:legacy authored conditional goods
	SecretIf           string                    `json:"secret_if,omitempty"`          // shop:legacy authored flag; not the proven native hidden-entry gate
}

// Campaign 整張節點圖。
type Campaign struct {
	Title string           `json:"title"`
	Start string           `json:"start"`
	Flags map[string]bool  `json:"flags"`
	Nodes map[string]*Node `json:"nodes"`
}

// Load 讀 campaign.json 並驗證轉場目標都存在。
func Load(path string) (*Campaign, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Campaign
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	if _, ok := c.Nodes[c.Start]; !ok {
		return nil, fmt.Errorf("start 節點 %q 不存在", c.Start)
	}
	check := func(from, to string) error {
		if to == "" {
			return nil
		}
		if _, ok := c.Nodes[to]; !ok {
			return fmt.Errorf("節點 %q 的轉場目標 %q 不存在", from, to)
		}
		return nil
	}
	for id, n := range c.Nodes {
		for _, to := range []string{n.Next, n.Cancel, n.OnWin, n.OnLose, n.IfPresent, n.IfMissing, n.IfCrafted, n.IfInsufficient} {
			if err := check(id, to); err != nil {
				return nil, err
			}
		}
		if n.Type == "inventory_gate" {
			if n.ItemID == nil || *n.ItemID < 0 || *n.ItemID > 255 {
				return nil, fmt.Errorf("inventory_gate 節點 %q 的 item_id 必須是 0..255", id)
			}
			if n.IfPresent == "" || n.IfMissing == "" {
				return nil, fmt.Errorf("inventory_gate 節點 %q 必須同時定義 if_present / if_missing", id)
			}
		}
		if n.Type == "inventory_recipe" {
			if len(n.ItemIDs) == 0 || n.SlotCount <= 0 || n.RequiredMatches <= 0 {
				return nil, fmt.Errorf("inventory_recipe 節點 %q 必須定義 item_ids / slot_count / required_matches", id)
			}
			for _, itemID := range n.ItemIDs {
				if itemID < 0 || itemID > 255 {
					return nil, fmt.Errorf("inventory_recipe 節點 %q 的 item_ids 必須是 0..255", id)
				}
			}
			if n.RewardItemID == nil || *n.RewardItemID < 0 || *n.RewardItemID > 255 {
				return nil, fmt.Errorf("inventory_recipe 節點 %q 的 reward_item_id 必須是 0..255", id)
			}
			if n.IfCrafted == "" || n.IfInsufficient == "" {
				return nil, fmt.Errorf("inventory_recipe 節點 %q 必須同時定義 if_crafted / if_insufficient", id)
			}
		}
		if n.Type == "shop" && n.NativeHubVariant != 0 &&
			n.NativeHubVariant != 1 && n.NativeHubVariant != 3 &&
			n.NativeHubVariant != 5 {
			return nil, fmt.Errorf(
				"shop 節點 %q 的 native_hub_variant 必須是 1、3 或 5",
				id,
			)
		}
		if n.NativeSecretGate != nil {
			gate := n.NativeSecretGate
			if n.Type != "town" || gate.Selection < 0 || gate.Selection > 4 ||
				gate.ScanCode < 0x54 || gate.ScanCode > 0x71 ||
				gate.To == "" || n.NativeTownVariant == nil ||
				check(id, gate.To) != nil {
				return nil, fmt.Errorf(
					"town 節點 %q 的 native_secret_gate 無效",
					id,
				)
			}
		}
		if n.NativeTownVariant != nil &&
			(n.Type != "town" || *n.NativeTownVariant < 0 ||
				*n.NativeTownVariant > 2) {
			return nil, fmt.Errorf(
				"town 節點 %q 的 native_town_variant 無效",
				id,
			)
		}
		if n.NativeEndingPrefix != nil {
			prefix := n.NativeEndingPrefix
			if n.Type != "ending" || !prefix.IsRecoveredPrefixContract() {
				return nil, fmt.Errorf(
					"ending 節點 %q 的 native_ending_prefix 不是已證實的 0x2bce5 recovered-only 合約",
					id,
				)
			}
		}
		if n.ApproximateWinSync {
			ending := c.Nodes[n.OnWin]
			if n.Type != "battle" || n.OnWin == "" || ending == nil || ending.Type != "ending" {
				return nil, fmt.Errorf(
					"battle 節點 %q 的 approximate_sync_party_on_win 只能用於直接進入 ending 的勝利邊",
					id,
				)
			}
		}
		if n.NativeMapHUD != nil && n.NativeMapView == nil {
			return nil, fmt.Errorf("battle 節點 %q 不可在缺少 native_map_view 時單獨定義 native_map_hud", id)
		}
		if n.NativeMapHUDInherited != nil {
			if n.NativeMapView == nil {
				return nil, fmt.Errorf("battle 節點 %q 不可在缺少 native_map_view 時繼承 native HUD", id)
			}
			if n.NativeMapHUD != nil {
				return nil, fmt.Errorf("battle 節點 %q 不可同時定義固定與繼承 native HUD", id)
			}
			if n.NativeMapHUDInherited.DisplayGateB != 1 {
				return nil, fmt.Errorf("battle 節點 %q 的 native_map_hud_inherited.display_gate_b 目前只允許已證實的 controller entry 1", id)
			}
		}
		if n.NativeMapView != nil {
			mode := n.NativeMapView.RangeMode
			// Campaign JSON currently owns two directly proven entry states:
			// 0 is the no-overlay state retained by ch26_pre on return, while 1
			// is the CONTINUE/post-bootstrap state at 0x1060c. Runtime command
			// and item writers own every other selector transition.
			if mode == nil || (*mode != 0 && *mode != 1) {
				return nil, fmt.Errorf("battle 節點 %q 的 native_map_view.range_mode 目前只允許已證實的 entry selector 0/1", id)
			}
		}
		if n.NativeMapHUD != nil {
			hud := n.NativeMapHUD
			if hud.DisplayGateA < 0 || hud.DisplayGateA > 0xff ||
				hud.DisplayGateB < 0 || hud.DisplayGateB > 0xff ||
				hud.AnchorX < 0 || hud.AnchorX > 0xfb {
				return nil, fmt.Errorf("battle 節點 %q 的 native_map_hud 超出原版 raw 範圍", id)
			}
		}
		for _, o := range n.Options {
			if err := check(id, o.To); err != nil {
				return nil, err
			}
		}
	}
	return &c, nil
}

// Runner 執行狀態:目前節點 + 旗標。
type Runner struct {
	C     *Campaign
	Cur   string
	Flags map[string]bool
}

// NewRunner 從起點開跑(複製初始旗標)。
func NewRunner(c *Campaign) *Runner {
	f := map[string]bool{}
	for k, v := range c.Flags {
		f[k] = v
	}
	return &Runner{C: c, Cur: c.Start, Flags: f}
}

// Node 目前節點。
func (r *Runner) Node() *Node { return r.C.Nodes[r.Cur] }

// NodeID exposes the editable node key for runtime data fallback decisions.
// Keeping this separate from Node avoids making callers infer identity from
// the node payload (which is intentionally allowed to be identical across
// multiple story segments).
func (r *Runner) NodeID() string { return r.Cur }

// Visible 回傳 choice 節點依旗標過濾後的選項。
func (r *Runner) Visible() []Option {
	n := r.Node()
	var out []Option
	for _, o := range n.Options {
		if o.If == "" || r.Flags[o.If] {
			out = append(out, o)
		}
	}
	return out
}

// ShopGoods returns editable goods and the legacy authored conditional list.
// SecretIf is a remake schema feature; it does not prove the native hidden
// selection-5 keyboard/modifier gate.
func (r *Runner) ShopGoods() []Good {
	n := r.Node()
	out := append([]Good{}, n.Goods...)
	if n.SecretIf != "" && r.Flags[n.SecretIf] {
		out = append(out, n.Secret...)
	}
	return out
}

// MatchNativeTownSecret reports whether the current raw selection and BIOS
// scan reveal native selection 5. 0x2cde0..0x2cef7 does not dispatch here:
// the town owner redraws selection 5 and waits for a later confirmation.
func (r *Runner) MatchNativeTownSecret(selection, scanCode int) bool {
	n := r.Node()
	if n == nil || n.Type != "town" || n.NativeSecretGate == nil {
		return false
	}
	gate := n.NativeSecretGate
	if selection != gate.Selection || scanCode != gate.ScanCode ||
		r.C.Nodes[gate.To] == nil {
		return false
	}
	return true
}

// ConfirmNativeTownSecret dispatches the already revealed native selection 5.
// It is deliberately separate from MatchNativeTownSecret so the original
// reveal/redraw/confirm lifecycle cannot collapse into a one-key transition.
func (r *Runner) ConfirmNativeTownSecret(selection int) bool {
	n := r.Node()
	if n == nil || n.Type != "town" || selection != 5 ||
		n.NativeSecretGate == nil ||
		r.C.Nodes[n.NativeSecretGate.To] == nil {
		return false
	}
	r.Cur = n.NativeSecretGate.To
	return true
}

// Advance 依結果離開目前節點:套用 set_flags,回傳下一節點 id(""=流程結束/game over)。
// outcome:story/event→忽略;battle→"win"/"lose";choice→"optN"(過濾後 index)。
func (r *Runner) Advance(outcome string) string {
	n := r.Node()
	if n == nil {
		return ""
	}
	for k, v := range n.SetFlags {
		r.Flags[k] = v
	}
	next := ""
	switch n.Type {
	case "battle":
		if outcome == "win" {
			next = n.OnWin
		} else {
			next = n.OnLose
		}
	case "choice", "town":
		var i int
		if _, err := fmt.Sscanf(outcome, "opt%d", &i); err == nil {
			if vis := r.Visible(); i >= 0 && i < len(vis) {
				next = vis[i].To
			}
		}
	case "inventory_gate":
		if outcome == "present" {
			next = n.IfPresent
		} else if outcome == "missing" {
			next = n.IfMissing
		}
	case "inventory_recipe":
		if outcome == "crafted" {
			next = n.IfCrafted
		} else if outcome == "insufficient" {
			next = n.IfInsufficient
		}
	case "preparation":
		if outcome == "cancel" {
			next = n.Cancel
		} else {
			next = n.Next
		}
	case "ending":
		next = ""
	default: // story / event / shop
		next = n.Next
	}
	r.Cur = next
	return next
}
