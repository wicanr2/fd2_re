//! Go 來源：`remake/internal/battle/native_map_view.go`（318 行）。

use super::*;

pub(crate) const NATIVE_MAP_VIEW_WIDTH: i64 = 13;

pub(crate) const NATIVE_MAP_VIEW_HEIGHT: i64 = 8;

/// Non-constant writers zero-extend a record byte and add two.  Together
/// with literal 0/1/6 writers this bounds the observed dword selector.
pub(crate) const NATIVE_MAP_OVERLAY_SELECTOR_MAX: i64 = 0x101;

/// NativeMapViewState owns camera [0x53aa9/0x53aad], absolute cursor
/// [0x53ab1/0x53ab5], and camera-relative visible cursor
/// [0x53ab9/0x53abd].
///
/// Go：`native_map_view.go:16`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeMapViewState {
    pub camera_x: i64,
    pub camera_y: i64,
    pub cursor_x: i64,
    pub cursor_y: i64,
    pub visible_cursor_x: i64,
    pub visible_cursor_y: i64,
}

/// validateNativeMapView 只檢查三組全域各自的界線，**不要求**
/// `visible == cursor - camera`。原版沒有那條恆等式：`[0x53AB9]`／`[0x53ABD]`
/// 的寫入端只有八處（IDA 9.4 data xref，見
/// docs/data/ida/fd2_visible_cursor_writers_ida.txt）——四個鍵盤游標處理器
/// `0x11B48`／`0x11B9B`／`0x11BFA`／`0x11C59`、四個走行步進
/// `0x12EAA`／`0x1300D`／`0x13185`／`0x13315`，加上開機初始化與章節重設的歸零。
/// 劇情捲動 `0x135DD` 與直接設定游標的 `0x149F8` 都只動 cursor／camera，
/// 不碰 visible，所以 visible 會合法地停在舊值。強制恆等式會讓那個狀態
/// 表達不出來，也會把原版真實走過的畫面判成錯誤。
///
/// Go：`native_map_view.go:31`，`func validateNativeMapView(view NativeMapViewState, width, height int) error`
pub(crate) fn validate_native_map_view(view: NativeMapViewState, width: i64, height: i64) -> Result<()> {
    todo!("battle/native_map_view.go:validateNativeMapView")
}

impl NativeMapViewState {
    /// VisibleCursorInViewport 回報可見游標有沒有落在 13×8 視窗內。
    ///
    /// 這是**消費端**的判準，不是狀態的合法性判準：0x1741C（指令環展開）在
    /// 0x1743F..0x1746A 以 `visible_x * 24 + visible_y * 24 * 0x1C8` 算 framebuffer
    /// 位址，出界就會畫到視窗外。任何要把可見游標當畫面格座標用的路徑都要先問過
    /// 這個函式再動手；狀態本身可以合法地出界（原版走行捲動就會）。
    /// CursorInField 回報絕對游標有沒有落在地圖內。同 VisibleCursorInViewport，這是
    /// 消費端的判準而不是狀態的合法性判準——劇情 pan 會合法地把它推出地圖。
    ///
    /// Go：`native_map_view.go:70`，`func (v NativeMapViewState) CursorInField(width, height int) bool`
    pub fn cursor_in_field(self, width: i64, height: i64) -> bool {
        todo!("battle/native_map_view.go:NativeMapViewState.CursorInField")
    }

    /// Go：`native_map_view.go:74`，`func (v NativeMapViewState) VisibleCursorInViewport() bool`
    pub fn visible_cursor_in_viewport(self) -> bool {
        todo!("battle/native_map_view.go:NativeMapViewState.VisibleCursorInViewport")
    }
}
impl State {
    /// Go：`native_map_view.go:79`，`func (s *State) MaterializeNativeMapViewState(view NativeMapViewState) error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn materialize_native_map_view_state(&mut self, view: NativeMapViewState) -> Result<()> {
        todo!("battle/native_map_view.go:State.MaterializeNativeMapViewState")
    }

    /// Go：`native_map_view.go:91`，`func (s *State) MaterializeNativeMapRangeMode(mode int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn materialize_native_map_range_mode(&mut self, mode: i64) -> bool {
        todo!("battle/native_map_view.go:State.MaterializeNativeMapRangeMode")
    }
}

/// NativeMapOverlaySelectorFromRecordByte preserves the recurring writer
/// dword_51A83 = recordByte + 2 at 0x15140, 0x153b1, 0x1bd14 and 0x1d188.
/// Values above six are meaningful to target validation even though 0x122dc
/// has no drawing branch for them.
///
/// Go：`native_map_view.go:104`，`func NativeMapOverlaySelectorFromRecordByte(recordByte byte) int`
pub fn native_map_overlay_selector_from_record_byte(record_byte: u8) -> i64 {
    todo!("battle/native_map_view.go:NativeMapOverlaySelectorFromRecordByte")
}

impl State {
    /// MoveNativeMapCursor reproduces the four helpers at 0x11b48..0x11cac，走行步進
    /// 家族 `0x12EAA`／`0x1300D`／`0x13185`／`0x13315` 在格邊界的效果與它相同：
    /// 絕對游標一定移動一格，另外二選一——可見游標在安全帶內就自己移動，否則改捲
    /// 鏡頭。安全帶是 13×8 視窗裡的 X 2..10、Y 2..5。
    /// moved is false at a field edge; ok is false for malformed state/input.
    ///
    /// Go：`native_map_view.go:113`，`func (s *State) MoveNativeMapCursor(dx, dy int) (moved, ok bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    /// Go 具名回傳值依序為：moved、ok。
    pub fn move_native_map_cursor(&mut self, dx: i64, dy: i64) -> (bool, bool) {
        todo!("battle/native_map_view.go:State.MoveNativeMapCursor")
    }

    /// AdvanceNativeMapWalkStepView 重現走行步進家族
    /// `0x12EAA`（下）／`0x1300D`（左）／`0x13185`（上）／`0x13315`（右）在一格
    /// 內對三組視圖全域的效果：絕對游標一定跟著單位走一格，另外二選一——可見游標
    /// 在安全帶內就自己移動，否則捲鏡頭一格。
    ///
    /// 判準與鍵盤處理器不同：走行步進算的是**單位自己的相對列／行**
    /// （`0x131DE` 的 `unitY - [0x53AAD]`、`0x12EEE` 的同式），不是已存的可見
    /// 游標值。可見游標在確認瞬間的 `0x149F8` 跳格之後可能是舊值，兩者會不一樣。
    ///
    /// unitX／unitY 是這一步**開始前**單位所在的格。
    ///
    /// Go：`native_map_view.go:180`，`func (s *State) AdvanceNativeMapWalkStepView(unitX, unitY, dx, dy int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn advance_native_map_walk_step_view(&mut self, unit_x: i64, unit_y: i64, dx: i64, dy: i64) -> bool {
        todo!("battle/native_map_view.go:State.AdvanceNativeMapWalkStepView")
    }
}

/// AdvanceNativeMapWalkStepViewState 是同一條規則的純函式版本。劇情走位的視圖
/// 不在 battle.State 裡（LOADCH 場景只有 storyNativeMapView），但走的是同一個
/// 0x13185 家族，不能各自再寫一份分支。
///
/// Go：`native_map_view.go:195`，`func AdvanceNativeMapWalkStepViewState( view NativeMapViewState, width, height, unitX, unitY, dx, dy int, ) (NativeMapViewState, bool)`
pub fn advance_native_map_walk_step_view_state(
    view: NativeMapViewState,
    width: i64,
    height: i64,
    unit_x: i64,
    unit_y: i64,
    dx: i64,
    dy: i64,
) -> (NativeMapViewState, bool) {
    todo!("battle/native_map_view.go:AdvanceNativeMapWalkStepViewState")
}

impl State {
    /// FocusNativeMapCursor 重現 `0x12CEA`：把絕對游標移到指定格，**先 X 後 Y**，
    /// 每一格都呼叫對應的鍵盤游標處理器（`0x12D13→0x11C59` 左、`0x12D1A→0x11BFA`
    /// 右、`0x12D4C→0x11B48` 上、`0x12D53→0x11B9B` 下），所以可見游標與鏡頭照同一
    /// 套安全帶規則跟著走，不會脫節。玩家確認移動時走的就是這條路：`0x18A26`（以及
    /// 目標無效時的 `0x189AA`）呼叫 `0x12CEA`，參數是 `0x18960` 存下來的游標值；
    /// `0x12D7B` 是同一條路的包裝，先讀單位記錄的 x/y 再呼叫 `0x12CEA`。
    ///
    /// 原版每移一格重繪一次（`0x12D01`／`0x12D3B` 的 `0x11CAC(0)`），游標是「滑」
    /// 回去的；這裡把整段迴圈在同一幀走完，狀態相同、少了中間那幾張畫面。
    ///
    /// 早期把這條路標成 `0x149F8` 並實作成「只寫絕對游標的瞬跳」是誤讀：
    /// `0x149F8..0x14B16` 是沿直線收集單位的 helper，`0x14AFE` 結尾會把游標復原，
    /// 淨效果是零。那個誤讀會讓可見游標每移動一次就累積一格偏差。
    ///
    /// Go：`native_map_view.go:252`，`func (s *State) FocusNativeMapCursor(x, y int) bool`
    pub fn focus_native_map_cursor(&mut self, x: i64, y: i64) -> bool {
        todo!("battle/native_map_view.go:State.FocusNativeMapCursor")
    }
}

/// NativeFocusTraceHook 是對拍診斷掛勾：非 nil 時每次 0x12CEA 聚焦前回報目標與當下視圖，
/// 讓重播測試（FD2_FOCUS_TRACE_OUT）逐筆對照原版 eip-trace 的 0x12CEA 參數。戰場視圖上的
/// focusUnitJob（0x13FD4 原地回復）與回合開頭聚焦也在各自的起點呼叫它。
///
/// Go：`native_map_view.go:259`
///
/// Rust 骨架：Go 的可改寫全域函式變數改成 `RwLock` 保護的選用 callback；`None` 對應 Go 的 nil。
pub static NATIVE_FOCUS_TRACE_HOOK: RwLock<Option<Callback<dyn Fn(i64, i64, NativeMapViewState) + Send + Sync>>> =
    RwLock::new(None);

impl State {
    /// FocusNativeMapCursorSteps 是 FocusNativeMapCursor 的逐步版本：每走一格呼叫一次
    /// onStep（走之前、走之後的視圖），讓呼叫端照 0x11C59 家族的重繪規則決定要不要
    /// 更新 HUD anchor。
    ///
    /// Go：`native_map_view.go:264`，`func (s *State) FocusNativeMapCursorSteps(x, y int, onStep func(before, after NativeMapViewState)) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn focus_native_map_cursor_steps(
        &mut self,
        x: i64,
        y: i64,
        on_step: Option<&mut dyn FnMut(NativeMapViewState, NativeMapViewState)>,
    ) -> bool {
        todo!("battle/native_map_view.go:State.FocusNativeMapCursorSteps")
    }

    /// NativeMapCursorStepRedraws 是四個游標鍵處理器（0x11B48／0x11B9B／0x11BFA／0x11C59）
    /// 收尾的重繪規則：捲了鏡頭一定重繪；只動可見游標時要 [0x51A83] 非 0 才 `0x11CAC(0)`。
    /// 沒重繪就沒有 0x1ACF3，HUD anchor（0x1AD2A）不會被評估。移動確認後的游標走回單位
    /// （0x18A14 先把 [0x51A83] 寫 0 再 0x18A26→0x12CEA）就是靠這條不翻小窗。
    ///
    /// Go：`native_map_view.go:310`，`func (s *State) NativeMapCursorStepRedraws(before, after NativeMapViewState) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_map_cursor_step_redraws(&self, before: NativeMapViewState, after: NativeMapViewState) -> bool {
        todo!("battle/native_map_view.go:State.NativeMapCursorStepRedraws")
    }
}
