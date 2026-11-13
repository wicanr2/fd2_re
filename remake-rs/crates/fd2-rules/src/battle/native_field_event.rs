//! Go 來源：`remake/internal/battle/native_field_event.go`（471 行）。

use super::*;

/// Go：`native_field_event.go:5`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeFieldEvent75Plan {
    pub event_id: u8,
    pub text_index: i64,
    pub activate: bool,
    pub noop: bool,
    pub(crate) trigger: Option<UnitIdx>,
    pub(crate) rule: Option<NativeFieldEventRule>,
}

/// PlanNativeFieldEvent75 驗證 map28 selector1 handler，不修改 event-state table
/// 或 live turn row。raw +8 != 9 選 FDTXT_029 index0，執行期另以觸發 record +7
/// 呈現肖像；raw +8 == 9 選 index1，並在對話完成後才可啟動 dormant rows。
///
/// Go：`native_field_event.go:18`，`func PlanNativeFieldEvent75(st *State, trigger *Unit, x, y int) (NativeFieldEvent75Plan, error)`
/// 所有權：`trigger` 是 `st.units` 的索引（Go 傳 `*Unit`）。
pub fn plan_native_field_event75(
    st: Option<&State>,
    trigger: Option<UnitIdx>,
    x: i64,
    y: i64,
) -> Result<NativeFieldEvent75Plan> {
    todo!("battle/native_field_event.go:PlanNativeFieldEvent75")
}

/// CommitNativeFieldEvent75 在 FDTXT_029 index1 完成後重驗 plan，再原子提交
/// event-state 與 typed live rows；只有 state 帶完整 raw field provenance 時才一併
/// 核對並更新該 raw view。
///
/// Go：`native_field_event.go:86`，`func CommitNativeFieldEvent75(st *State, plan NativeFieldEvent75Plan) error`
pub fn commit_native_field_event75(st: Option<&mut State>, plan: &NativeFieldEvent75Plan) -> Result<()> {
    todo!("battle/native_field_event.go:CommitNativeFieldEvent75")
}

/// NativeFieldEventIDAt 保存 0x13a44 的格子事件 selector：
/// 該格必須具有已驗證的 0..15 slot，event_id 不得為 0xff，且列內 selector
/// 必須等於 caller selector。資料缺失時失敗即關閉。
///
/// Go：`native_field_event.go:127`，`func NativeFieldEventIDAt( st *State, x, y int, selector byte, ) (byte, bool)`
pub fn native_field_event_id_at(st: Option<&State>, x: i64, y: i64, selector: u8) -> (u8, bool) {
    todo!("battle/native_field_event.go:NativeFieldEventIDAt")
}

/// ApplyNativeFieldModeEvent 執行目前已閉合的 mode-range 規則。
/// selector 的呼叫時機仍由上層保存；非 mode 規則（包含 event61）不在此猜測執行。
///
/// Go：`native_field_event.go:154`，`func ApplyNativeFieldModeEvent( st *State, trigger *Unit, x, y int, selector byte, ) (byte, bool)`
/// 所有權：`trigger` 是 `st.units` 的索引（Go 傳 `*Unit`）。
pub fn apply_native_field_mode_event(
    st: Option<&mut State>,
    trigger: Option<UnitIdx>,
    x: i64,
    y: i64,
    selector: u8,
) -> (u8, bool) {
    todo!("battle/native_field_event.go:ApplyNativeFieldModeEvent")
}

/// ApplyNativeFieldTurnActivationEvent 執行已閉合的 event62 mutation：selector0
/// 對到 event62 時，把休眠 row0（event63/raw camp0）排到 native round+1，並
/// 設定 battle-local state byte17。editable rule 與完整 raw row 必須先一致，
/// 才能改動任一值。
///
/// Go：`native_field_event.go:223`，`func ApplyNativeFieldTurnActivationEvent( st *State, x, y int, selector byte, ) (byte, error)`
pub fn apply_native_field_turn_activation_event(st: Option<&mut State>, x: i64, y: i64, selector: u8) -> Result<u8> {
    todo!("battle/native_field_event.go:ApplyNativeFieldTurnActivationEvent")
}

/// ApplyNativeChapter28FieldTurnActivation 保存3599B／35AB8的區域啟動。
/// 65不讀觸發者陣營；69先讀觸發者raw +6，再查state17／18。
/// READY與原始定位見fd2_ch28_flow_20261016.json。
///
/// Go：`native_field_event.go:264`，`func ApplyNativeChapter28FieldTurnActivation(st *State, trigger *Unit, x, y int, selector byte) (byte, error)`
/// 所有權：`trigger` 是 `st.units` 的索引（Go 傳 `*Unit`）。
pub fn apply_native_chapter28_field_turn_activation(
    st: Option<&mut State>,
    trigger: Option<UnitIdx>,
    x: i64,
    y: i64,
    selector: u8,
) -> Result<u8> {
    todo!("battle/native_field_event.go:ApplyNativeChapter28FieldTurnActivation")
}

/// Go：`native_field_event.go:310`，`func commitNativeFieldTurnActivation(st *State, eventID byte, rule *NativeFieldEventRule) (byte, error)`
pub(crate) fn commit_native_field_turn_activation(
    st: &mut State,
    event_id: u8,
    rule: &NativeFieldEventRule,
) -> Result<u8> {
    todo!("battle/native_field_event.go:commitNativeFieldTurnActivation")
}

/// Go：`native_field_event.go:342`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeFieldEvent61Plan {
    pub event_id: u8,
    pub missing_item: bool,
    pub text_index: i64,
    pub final_text: i64,
    pub presentation: NativeFieldPresentation,
    pub join_character: i64,
    pub(crate) trigger: Option<UnitIdx>,
    pub(crate) item_index: i64,
    pub(crate) state_index: i64,
    pub(crate) spawn_group: i64,
    pub(crate) required_frames: i64,
}

/// PlanNativeFieldEvent61 validates the complete editable/native boundary
/// before the UI starts text #3 and the 59-frame presentation. Missing item is
/// a valid, non-mutating native outcome; malformed provenance fails closed.
///
/// Go：`native_field_event.go:360`，`func PlanNativeFieldEvent61( st *State, trigger *Unit, x, y int, ) (NativeFieldEvent61Plan, error)`
/// 所有權：`trigger` 是 `st.units` 的索引（Go 傳 `*Unit`）。
pub fn plan_native_field_event61(
    st: Option<&State>,
    trigger: Option<UnitIdx>,
    x: i64,
    y: i64,
) -> Result<NativeFieldEvent61Plan> {
    todo!("battle/native_field_event.go:PlanNativeFieldEvent61")
}

/// CommitNativeFieldEvent61 applies only the mutations after the caller has
/// presented all native frames. It revalidates the plan so cancellation or an
/// intervening inventory change cannot consume an item or partially set state.
/// Persistent JOIN is intentionally returned to the campaign owner.
///
/// Go：`native_field_event.go:436`，`func CommitNativeFieldEvent61( st *State, plan NativeFieldEvent61Plan, presentedFrames int, ) (int, error)`
pub fn commit_native_field_event61(
    st: Option<&mut State>,
    plan: &NativeFieldEvent61Plan,
    presented_frames: i64,
) -> Result<i64> {
    todo!("battle/native_field_event.go:CommitNativeFieldEvent61")
}
