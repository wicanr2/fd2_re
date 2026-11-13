//! Go 來源：`remake/internal/battle/native_ai_phase_diagnostic.go`（139 行）。

use super::*;

/// NativeAIPhaseZeroInput supplies the caller-owned movement row and optional
/// 0x1f183 gate for one eligible selector-zero record. UnitIndex is a raw
/// runtime roster index, not a persistent-party or character identity.
///
/// Go：`native_ai_phase_diagnostic.go:13`
#[derive(Debug, Clone, Default)]
pub struct NativeAIPhaseZeroInput {
    pub unit_index: i64,
    pub cost_row: Vec<u8>,
    pub skip: Option<Callback<dyn Fn(i64) -> bool + Send + Sync>>,
}

/// NativeAIPhaseZeroDiagnostic preserves both score-producer results computed
/// in the original 0x1d8ba order. It is evidence/diagnostic state only and does
/// not authorize execution of either winner.
///
/// Go：`native_ai_phase_diagnostic.go:22`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeAIPhaseZeroDiagnostic {
    pub unit_index: i64,
    pub command_mask_result: NativeAI1598AScoreResult,
    pub item_command_result: NativeAI1567EScoreResult,
    pub phase_gate_score_pair: fdother::NativePhasePreselectionScores,
}

/// BuildNativeAIPhaseDiagnosticPlan connects the proven, side-effect-free part
/// of 0x1d8ba:
///
///     eligible raw +6==0 record
///     -> 0x1598a(unit,0)
///     -> 0x1567e(unit,0)
///     -> signed [0x53c23] >= 6 || [0x53c33] >= 6
///
/// It also returns the three-pass scan plan, but never invokes 0x13a9f,
/// [0x51a8f]/[0x51b91], the chapter handler table, or [0x53ecc] early exit.
/// Every eligible selector-zero record must have exactly one explicit input;
/// missing or extra provenance fails closed.
///
/// Go：`native_ai_phase_diagnostic.go:41`，`func BuildNativeAIPhaseDiagnosticPlan( st *State, book []NativeCommandRecord, itemRows []byte, inputs []NativeAIPhaseZeroInput, ) (fdother.NativePhaseUnitScanPlan, []NativeAIPhaseZeroDiagnostic, error)`
pub fn build_native_ai_phase_diagnostic_plan(
    st: Option<&State>,
    book: &[NativeCommandRecord],
    item_rows: &[u8],
    inputs: &[NativeAIPhaseZeroInput],
) -> Result<(fdother::NativePhaseUnitScanPlan, Vec<NativeAIPhaseZeroDiagnostic>)> {
    let Some(st) = st.filter(|st| !st.units.is_empty() && st.units.len() <= 0x100) else {
        return Err(battle_err!("native AI phase state is malformed"));
    };
    let records = native_ai_scoring_records(&st.units)?;
    let base_flags = native_composition_base_flags(st.w, st.h, &st.native_composition_event_bytes)?;
    let count = st.units.len() as i64;
    let mut zero_scores = vec![fdother::NativePhasePreselectionScores::default(); count as usize];
    let admission = fdother::plan_native_phase_unit_scans(&records, count, &zero_scores)?;

    let mut input_by_unit: BTreeMap<i64, &NativeAIPhaseZeroInput> = BTreeMap::new();
    for input in inputs {
        if input.unit_index < 0 || input.unit_index >= count {
            return Err(battle_err!("native AI phase input unit {} is out of bounds", input.unit_index));
        }
        if input_by_unit.contains_key(&input.unit_index) {
            return Err(battle_err!("native AI phase input unit {} is duplicated", input.unit_index));
        }
        input_by_unit.insert(input.unit_index, input);
    }

    let mut eligible = BTreeSet::new();
    let mut diagnostics = Vec::with_capacity(admission.zero_preselection.len());
    for candidate in &admission.zero_preselection {
        let unit_index = candidate.unit_index;
        eligible.insert(unit_index);
        let Some(input) = input_by_unit.get(&unit_index) else {
            return Err(battle_err!("native AI phase input for eligible unit {unit_index} is unavailable"));
        };
        let mut skip_fn = input.skip.as_ref().map(|cb| move |index: i64| cb(index));
        let command_mask = score_native_ai1598_a(
            st.w,
            st.h,
            &records,
            count,
            unit_index,
            0,
            Some(&st.units[unit_index as usize]),
            book,
            &base_flags,
            &st.native_terrain_move_codes,
            &input.cost_row,
            skip_fn.as_mut().map(|f| f as &mut dyn FnMut(i64) -> bool),
        )?;
        let item_command =
            score_native_ai1567_e(st.w, st.h, &records, count, unit_index, 0, item_rows, book, &base_flags)?;
        if command_mask.max_score > i32::MAX as i64 || item_command.max_score > i32::MAX as i64 {
            return Err(battle_err!("native AI phase score for unit {unit_index} exceeds signed dword"));
        }
        let pair = fdother::NativePhasePreselectionScores {
            spell_candidate_score_53c23: command_mask.max_score as i32,
            item_command_score_53c33: item_command.max_score as i32,
        };
        zero_scores[unit_index as usize] = pair;
        diagnostics.push(NativeAIPhaseZeroDiagnostic {
            unit_index,
            command_mask_result: command_mask,
            item_command_result: item_command,
            phase_gate_score_pair: pair,
        });
    }
    // Go 迭代 map：多筆不合格輸入時報哪一筆隨 map 順序而定；Rust 依單位索引升冪（battle.rs 模組文件）。
    for &unit_index in input_by_unit.keys() {
        if !eligible.contains(&unit_index) {
            return Err(battle_err!("native AI phase input unit {unit_index} is not selector-zero eligible"));
        }
    }
    let plan = fdother::plan_native_phase_unit_scans(&records, count, &zero_scores)?;
    Ok((plan, diagnostics))
}

#[cfg(test)]
mod native_ai_phase_diagnostic_test;
