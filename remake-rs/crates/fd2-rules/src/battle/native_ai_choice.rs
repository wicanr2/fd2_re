//! Go 來源：`remake/internal/battle/native_ai_choice.go`（34 行）。

use super::*;

/// NativeAISpellCandidate is the already-resolved raw candidate tuple consumed
/// by 0x1598a after command availability and target resolution. CommandValue
/// is the command record word at +0; X/Y are caller-provided raw coordinates.
///
/// Go：`native_ai_choice.go:6`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeAISpellCandidate {
    pub command_id: i64,
    pub command_value: i64,
    pub x: i64,
    pub y: i64,
    pub score: i64,
    /// TargetIndices preserves the raw runtime record order emitted by the
    /// candidate's 0x14818 target pass.  The command consumer may use the first
    /// index as the confirmed cursor while re-running its own effect geometry;
    /// it must never infer a target from normalized Camp or distance alone.
    pub target_indices: Vec<u8>,
}

/// SelectNativeAISpellCandidate preserves 0x1598a's strict score comparison:
/// greater score wins; equal score compares raw command +0; an exact tie keeps
/// the first candidate. It does not perform MP, target, status, or UI work.
///
/// Go：`native_ai_choice.go:22`，`func SelectNativeAISpellCandidate(candidates []NativeAISpellCandidate) (NativeAISpellCandidate, bool)`
pub fn select_native_ai_spell_candidate(candidates: &[NativeAISpellCandidate]) -> (NativeAISpellCandidate, bool) {
    todo!("battle/native_ai_choice.go:SelectNativeAISpellCandidate")
}
