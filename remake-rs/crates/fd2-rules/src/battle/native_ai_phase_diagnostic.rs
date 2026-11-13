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
    todo!("battle/native_ai_phase_diagnostic.go:BuildNativeAIPhaseDiagnosticPlan")
}
