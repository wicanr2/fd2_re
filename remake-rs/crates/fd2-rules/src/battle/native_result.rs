//! Go 來源：`remake/internal/battle/native_result.go`（125 行）。

use super::*;

/// NativeResultRule 保存已閉合的章節結果覆寫；Source 只是原始 writer 的
/// 出處，正式執行依具型別條件，不依 FD2.EXE 位址分派。
/// READY 契約與主證據見56及fd2_ch13_result_conditions_20261001.json、
/// fd2_ch15_result_conditions_20261002.json；原版無對白列的Actions為空。
///
/// Go：`native_result.go:9`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeResultRule {
    /// Go 標籤：`json:"id"`
    #[serde(deserialize_with = "null_as_default")]
    pub id: String,
    /// Go 標籤：`json:"source"`
    #[serde(deserialize_with = "null_as_default")]
    pub source: String,
    /// Go 標籤：`json:"records_all_inactive"`
    #[serde(deserialize_with = "null_as_default")]
    pub records_all_inactive: Vec<i64>,
    /// Go 標籤：`json:"event_states_all_nonzero,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub event_states_all_nonzero: Vec<i64>,
    /// Go 標籤：`json:"round_greater_than,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub round_greater_than: Option<i64>,
    /// Go 標籤：`json:"code"`
    pub code: i64,
    /// Go 標籤：`json:"actions"`
    #[serde(deserialize_with = "null_as_default")]
    pub actions: Vec<Action>,
}

impl NativeResultRule {
    /// Go：`native_result.go:19`，`func (r NativeResultRule) Validate() error`
    pub fn validate(&self) -> Result<()> {
        let state_rule = self.code == 2
            && !self.event_states_all_nonzero.is_empty()
            && self.records_all_inactive.is_empty()
            && self.round_greater_than.is_none()
            && self.actions.is_empty();
        let record_rule = (self.code == 1
            || (self.code == 2 && self.round_greater_than.is_none() && self.actions.is_empty()))
            && !self.records_all_inactive.is_empty()
            && self.event_states_all_nonzero.is_empty();
        if self.id.is_empty() || self.source.is_empty() || (!state_rule && !record_rule) {
            return Err(battle_err!("原生結果列缺少來源或已閉合的記錄／事件狀態條件"));
        }
        if self.round_greater_than.is_some_and(|round| round < 0) {
            return Err(battle_err!("原生結果列回合門檻為負值"));
        }
        let mut seen = BTreeSet::new();
        let state_len = State::default().native_event_state.len() as i64;
        for &index in &self.event_states_all_nonzero {
            if index < 0 || index >= state_len || seen.contains(&index) {
                return Err(battle_err!("原生結果列事件狀態索引{index}無效或重複"));
            }
            seen.insert(index);
        }
        for &index in &self.records_all_inactive {
            if index < 0 || seen.contains(&index) {
                return Err(battle_err!("原生結果列記錄索引{index}無效或重複"));
            }
            seen.insert(index);
        }
        for (i, action) in self.actions.iter().enumerate() {
            let bad = match (&action.native_dialogue_ref, action.native_text_index) {
                (Some(r), Some(text_index)) => {
                    action.r#type != "dialogue"
                        || action.native_source.is_empty()
                        || text_index != r.string_index
                        || r.script.is_empty()
                        || r.source_dat.is_empty()
                        || r.scene_index < 0
                        || r.line < 0
                        || r.pages.is_empty()
                        || action.native_when.is_some()
                        || action.native_event_id.is_some()
                }
                _ => true,
            };
            if bad {
                return Err(battle_err!("原生結果列動作{i}缺少一致的獨立對白來源"));
            }
        }
        Ok(())
    }

    /// Match 在該列真正執行時讀 raw 狀態；回合閘先於記錄查詢，不能因
    /// 尚未生成的59而提前拒絕，也不能用HP代替sub_3453E的+5 bit0。
    ///
    /// Go：`native_result.go:55`，`func (r NativeResultRule) Match(st *State) (bool, error)`
    pub fn r#match(&self, st: Option<&State>) -> Result<bool> {
        self.validate()?;
        let Some(st) = st else {
            return Err(battle_err!("原生結果列缺少戰場"));
        };
        if !self.event_states_all_nonzero.is_empty() && !st.has_native_turn_event_control_state {
            return Err(battle_err!("原生結果列缺少事件狀態來源"));
        }
        if let Some(round) = self.round_greater_than {
            if st.native_round_counter <= 0 {
                return Err(battle_err!("原生結果列缺少原始回合計數"));
            }
            if st.native_round_counter <= round {
                return Ok(false);
            }
        }
        let mut all = true;
        for &index in &self.event_states_all_nonzero {
            if st.native_event_state[index as usize] == 0 {
                all = false;
            }
        }
        for &index in &self.records_all_inactive {
            if index >= st.units.len() as i64 || !st.units[index as usize].has_native_record_byte5 {
                return Err(battle_err!("原生結果列缺少記錄{index}的raw +5"));
            }
            if st.units[index as usize].native_record_byte5 & 1 == 0 {
                all = false;
            }
        }
        Ok(all)
    }
}

/// NativeInitialResultCode preserves 20B72's event-state-only result owner.
/// READY: fd2_ch29_flow_20261017.json chapter_result_contract.
///
/// Go：`native_result.go:92`，`func NativeInitialResultCode(st *State, sc *Scenario) (int, error)`
pub fn native_initial_result_code(st: Option<&State>, sc: Option<&Scenario>) -> Result<i64> {
    let Some((sc, initial)) = sc.and_then(|sc| sc.native_result_initial_code.map(|code| (sc, code))) else {
        return native_default_result_code(st);
    };
    let known_handler = sc.native_result_handler == "0x20b72" || sc.native_result_handler == "0x20bf5";
    let Some(st) =
        st.filter(|st| initial == 0 && known_handler && st.units.len() >= 2 && st.has_native_turn_event_control_state)
    else {
        return Err(battle_err!("原生章節結果初始值缺少已證實來源"));
    };
    if sc.native_result_handler == "0x20bf5" && st.units.len() <= 20 {
        return Err(battle_err!("第30章結果缺少原始record20"));
    }
    Ok(0)
}

/// NativeDefaultResultCode 是sub_205BE的已閉合三值基礎規則：僅掃已生成的
/// runtime陣列；raw+6=0且+5 bit0=0使結果2改0，最後記錄0的bit0使其改1。
///
/// Go：`native_result.go:108`，`func NativeDefaultResultCode(st *State) (int, error)`
pub fn native_default_result_code(st: Option<&State>) -> Result<i64> {
    let Some(st) = st.filter(|st| !st.units.is_empty()) else {
        return Err(battle_err!("原生基礎結果缺少runtime記錄"));
    };
    let mut code = 2;
    for (index, u) in st.units.iter().enumerate() {
        if !u.has_native_record_byte5 || !u.has_native_record_byte6 {
            return Err(battle_err!("原生基礎結果缺少記錄{index}的raw +5／+6"));
        }
        if u.native_record_byte6 == 0 && u.native_record_byte5 & 1 == 0 {
            code = 0;
        }
    }
    if st.units[0].native_record_byte5 & 1 != 0 {
        code = 1;
    }
    Ok(code)
}

#[cfg(test)]
mod native_result_test;
