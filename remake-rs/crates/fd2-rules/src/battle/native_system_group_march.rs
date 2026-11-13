//! Go 來源：`remake/internal/battle/native_system_group_march.go`（214 行）。

use super::*;

/// NativeSystemGroupMarchStep 保存 sub_16F55 selector1 對一筆合格 runtime
/// record 的可重播結果。Path 含起點與終點；本切片只接受不觸發全域事件的路徑。
///
/// Go：`native_system_group_march.go:7`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeSystemGroupMarchStep {
    pub unit_index: i64,
    pub path: Vec<Cell>,
    pub events: Vec<NativeSystemGroupMarchEvent>,
}

/// NativeSystemGroupMarchEvent 保存一個已在私有投影驗證的 selector1 中途事件。
/// PathIndex 指向 Path 內事件格；TextIndex／Presentation 只供正式 UI owner 在
/// 關閉確認框前驗證既有可編輯素材，不替 handler 新增語意。
///
/// Go：`native_system_group_march.go:16`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeSystemGroupMarchEvent {
    pub path_index: i64,
    pub event_id: u8,
    pub text_index: i64,
    pub presentation: bool,
}

/// NativeSystemGroupMarchPlan 是確認視窗關閉前完成的私有預演結果。
///
/// Go：`native_system_group_march.go:24`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeSystemGroupMarchPlan {
    pub destination: Cell,
    pub steps: Vec<NativeSystemGroupMarchStep>,
}

impl State {
    /// PlanNativeSystemGroupMarch 重現 0x16FF4..0x1716A 的無事件產品子集：依
    /// runtime record 順序選 (raw+5)&0x85==0、raw+6==2 的單位，逐筆以 selector1
    /// 執行已閉合的 0x14B78 尋路。占位會在私有投影逐筆更新，任一未知事件或來源
    /// 缺失都讓整批失敗，不發布部分移動。
    ///
    /// Go：`native_system_group_march.go:33`，`func (s *State) PlanNativeSystemGroupMarch(destination Cell) (NativeSystemGroupMarchPlan, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn plan_native_system_group_march(&mut self, destination: Cell) -> Result<NativeSystemGroupMarchPlan> {
        // Go 失敗時另回傳只帶 Destination 的空計畫；Rust 的 `Result` 只回錯誤。
        let mut plan = NativeSystemGroupMarchPlan { destination, steps: Vec::new() };
        let cells = self.w.wrapping_mul(self.h);
        if self.w <= 0
            || self.h <= 0
            || destination.x < 0
            || destination.x >= self.w
            || destination.y < 0
            || destination.y >= self.h
            || self.units.is_empty()
            || self.native_movement_cost_rows.len() as i64 != NATIVE_MOVEMENT_COST_ROW_COUNT
            || self.native_terrain_move_codes.len() as i64 != cells
            || self.native_composition_event_bytes.len() as i64 != cells
        {
            return Err(battle_err!("battle: native system group-march provenance is unavailable"));
        }
        let mut working = clone_native_system_group_march_state(self)?;
        // dword_53BEB 是每輪重讀的動態上限；event61 追加的 group1 record
        // 因此也會在同一批後續接受 raw gate，而不是被確認時的 len 快照截斷。
        let rs = NATIVE_RECORD_SIZE as usize;
        let mut index = 0;
        while index < working.units.len() {
            let unit = &working.units[index];
            if !unit.has_native_record_byte5 || !unit.has_native_record_byte6 {
                return Err(battle_err!("battle: native system group-march unit {index} lacks raw gate provenance"));
            }
            if unit.native_record_byte5 & 0x85 != 0 || unit.native_record_byte6 != 2 {
                index += 1;
                continue;
            }
            let records = native_ai_scoring_records(&working.units)?;
            let actor_record = records[index * rs..(index + 1) * rs].to_vec();
            let cost_row = native_ai_cost_row_for_record(&actor_record, &working.native_movement_cost_rows)?;
            let base_flags =
                native_composition_base_flags(working.w, working.h, &working.native_composition_event_bytes)?;
            let movement = working.native_ai_plan_toward_raw_destination(
                Some(index),
                index as i64,
                1,
                &records,
                &actor_record,
                &base_flags,
                &cost_row,
                destination,
            )?;
            if movement.path.is_empty() {
                return Err(battle_err!("battle: native system group-march unit {index} produced no path"));
            }
            let mut step = NativeSystemGroupMarchStep {
                unit_index: index as i64,
                path: movement.path.clone(),
                events: Vec::new(),
            };
            for segment in 0..movement.path.len() - 1 {
                let (a, b) = (movement.path[segment], movement.path[segment + 1]);
                let unit = &mut working.units[index];
                (unit.x, unit.y) = (b.x, b.y);
                unit.native_map_presentation.x = b.x as u8;
                unit.native_map_presentation.y = b.y as u8;
                if b.x != a.x - 1 || b.y != a.y {
                    continue;
                }
                if let (event_id, true) = native_field_event_id_at(Some(&working), b.x, b.y, 1) {
                    let event = plan_native_system_group_march_event(
                        Some(&mut working),
                        Some(index),
                        segment as i64 + 1,
                        event_id,
                    )?;
                    step.events.push(event);
                }
            }
            plan.steps.push(step);
            let unit = &mut working.units[index];
            unit.native_record_byte5 |= 0x80;
            unit.acted = true;
            index += 1;
        }
        if plan.steps.is_empty() {
            return Err(battle_err!("battle: native system group-march has no eligible unit"));
        }
        Ok(plan)
    }
}

/// Go：`native_system_group_march.go:112`，`func cloneNativeSystemGroupMarchUnit(unit *Unit) *Unit`
pub(crate) fn clone_native_system_group_march_unit(unit: Option<&Unit>) -> Option<Unit> {
    // Go 先淺複製再逐一複製五個切片；Rust 的 `clone()` 本來就是深複製。
    unit.cloned()
}

/// Go：`native_system_group_march.go:125`，`func cloneNativeSystemGroupMarchState(source *State) (State, error)`
pub(crate) fn clone_native_system_group_march_state(source: &mut State) -> Result<State> {
    // Go `working := *source` 是淺複製：只換掉選擇器快取、Units、Roster 與 NativeFieldControlRaw，
    // 其餘 map／切片與來源共用。預演路徑（0x14B78 尋路、event61／75 的投影提交）只寫這幾個欄位、
    // 事件狀態表與控制列（兩者是 Go 陣列，複製時就分開），所以 Rust 的整份深複製結果相同。
    // Go 對 nil 單位回錯誤的分支在 Rust 不存在（`Vec<Unit>` 沒有 nil）。
    let mut working = source.clone();
    working.units =
        source.units.iter().map(|u| clone_native_system_group_march_unit(Some(u)).unwrap_or_default()).collect();
    working.roster =
        source.roster.iter().map(|u| clone_native_system_group_march_unit(Some(u)).unwrap_or_default()).collect();
    Ok(working)
}

/// Go：`native_system_group_march.go:148`，`func planNativeSystemGroupMarchEvent( working *State, trigger *Unit, pathIndex int, eventID byte, ) (NativeSystemGroupMarchEvent, error)`
/// 所有權：`trigger` 是 `working.units` 的索引（Go 傳 `*Unit`）。
pub(crate) fn plan_native_system_group_march_event(
    working: Option<&mut State>,
    trigger: Option<UnitIdx>,
    path_index: i64,
    event_id: u8,
) -> Result<NativeSystemGroupMarchEvent> {
    let mut event = NativeSystemGroupMarchEvent { path_index, event_id, ..Default::default() };
    // Go 直接讀 trigger.X／Y（nil 會 panic）；呼叫端一律傳 working.units 內的索引。
    let (tx, ty) = {
        let st = working.as_deref().expect("planNativeSystemGroupMarchEvent: Go 對 nil State 解參照");
        let unit = &st.units[trigger.expect("planNativeSystemGroupMarchEvent: Go 對 nil trigger 解參照")];
        (unit.x, unit.y)
    };
    match event_id {
        61 => {
            let plan = match plan_native_field_event61(working.as_deref(), trigger, tx, ty) {
                Ok(plan) => plan,
                Err(err) => return Err(battle_err!("battle: native system group-march event61: {err}")),
            };
            event.text_index = plan.text_index;
            event.presentation = !plan.missing_item;
            if !plan.missing_item
                && let Err(err) = commit_native_field_event61(working, &plan, plan.required_frames)
            {
                return Err(battle_err!("battle: native system group-march event61 projection: {err}"));
            }
        }
        75 => {
            let plan = match plan_native_field_event75(working.as_deref(), trigger, tx, ty) {
                Ok(plan) => plan,
                Err(err) => return Err(battle_err!("battle: native system group-march event75: {err}")),
            };
            event.text_index = plan.text_index;
            if plan.activate
                && let Err(err) = commit_native_field_event75(working, &plan)
            {
                return Err(battle_err!("battle: native system group-march event75 projection: {err}"));
            }
        }
        _ => {
            return Err(battle_err!("battle: native system group-march event {event_id} has no atomic runtime owner"));
        }
    }
    Ok(event)
}

impl State {
    /// CommitNativeSystemGroupMarchStep 原子發布一筆已預演移動；動畫 owner 必須在
    /// path 完整呈現後呼叫。它對應 selector1 每筆的姿態清零與 0x13512 bit7 writer。
    ///
    /// Go：`native_system_group_march.go:189`，`func (s *State) CommitNativeSystemGroupMarchStep(step NativeSystemGroupMarchStep) error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn commit_native_system_group_march_step(&mut self, step: &NativeSystemGroupMarchStep) -> Result<()> {
        if step.unit_index < 0 || step.unit_index >= self.units.len() as i64 || step.path.is_empty() {
            return Err(battle_err!("battle: native system group-march step is malformed"));
        }
        let unit = &mut self.units[step.unit_index as usize];
        let (start, end) = (step.path[0], step.path[step.path.len() - 1]);
        if !unit.has_native_map_presentation
            || !unit.has_native_record_byte5
            || !unit.has_native_record_byte6
            || unit.native_record_byte5 & 0x85 != 0
            || unit.native_record_byte6 != 2
        {
            return Err(battle_err!("battle: native system group-march step no longer matches runtime state"));
        }
        let at_start = unit.x == start.x
            && unit.y == start.y
            && unit.native_map_presentation.x as i64 == start.x
            && unit.native_map_presentation.y as i64 == start.y;
        let at_end = unit.x == end.x
            && unit.y == end.y
            && unit.native_map_presentation.x as i64 == end.x
            && unit.native_map_presentation.y as i64 == end.y;
        if !at_start && !at_end {
            return Err(battle_err!("battle: native system group-march step no longer matches runtime position"));
        }
        (unit.x, unit.y) = (end.x, end.y);
        unit.native_map_presentation.x = end.x as u8;
        unit.native_map_presentation.y = end.y as u8;
        unit.native_map_presentation.pose = 0;
        unit.native_record_byte5 |= 0x80;
        unit.acted = true;
        Ok(())
    }
}

#[cfg(test)]
mod native_system_group_march_test;
