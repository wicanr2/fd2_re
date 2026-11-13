//! Go 來源：`remake/internal/battle/native_ai_idle_recovery.go`（185 行）。

use super::*;

/// NativeAIIdleRecoveryDecision preserves the raw decision boundary of
/// 0x13fd4.  The fields are offsets and values, not normalized status names.
/// Accepted is false for an equal-HP record or either non-zero raw gate.
///
/// Go：`native_ai_idle_recovery.go:11`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeAIIdleRecoveryDecision {
    pub unit: i64,
    pub accepted: bool,
    pub current_hp: u16,
    pub maximum_hp: u16,
    pub next_hp: u16,
    pub raw_gate25: u8,
    pub raw_gate26: u8,
}

/// NativeAIIdleRecoveryCopy preserves the raw 0x11EB0 copy arguments used by
/// the recovery presentation. SourceOffset/DestinationOffset retain the first
/// two raw pointer/offset arguments; SourceStride, DestinationStride, Rows and
/// Length expose the directly decoded source stride, destination stride, row
/// count and copy width. These are ABI fields, not renderer or gameplay names.
///
/// Go：`native_ai_idle_recovery.go:26`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeAIIdleRecoveryCopy {
    pub source_offset: u32,
    pub destination_offset: u32,
    pub source_stride: u32,
    pub destination_stride: u32,
    pub rows: u32,
    pub length: u32,
}

/// NativeAIIdleRecoveryDecode preserves the two 0x1DA16 calls surrounding the
/// indexed-buffer copy. ResourceBase remains an expression because IDA proves
/// the caller-owned pointer expression, not a high-level asset name.
///
/// Go：`native_ai_idle_recovery.go:38`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeAIIdleRecoveryDecode {
    pub resource_base: String,
    pub mode: u32,
    pub tail: u32,
}

/// NativeAIIdleRecoveryCall preserves the raw unit argument of 0x12D7B.
/// Address is an original executable address, not a gameplay action name.
///
/// Go：`native_ai_idle_recovery.go:46`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeAIIdleRecoveryCall {
    pub address: u32,
    pub unit: i64,
}

/// NativeAIIdleRecoveryGlobalWrite preserves the two raw writes to 0x51A83.
/// The address/value pair is intentionally not renamed as a status flag.
///
/// Go：`native_ai_idle_recovery.go:53`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeAIIdleRecoveryGlobalWrite {
    pub address: u32,
    pub value: u32,
}

/// NativeAIIdleRecoveryPresentation is the caller-owned presentation contract
/// for 0x13FD4. It contains only fixed raw arguments proved by IDA; a renderer
/// or audio owner must explicitly consume it before HP is committed.
///
/// Go：`native_ai_idle_recovery.go:61`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeAIIdleRecoveryPresentation {
    pub unit: i64,
    pub before_hp: u16,
    pub after_hp: u16,
    pub before_global_write: NativeAIIdleRecoveryGlobalWrite,
    pub coordinate_call: NativeAIIdleRecoveryCall,
    pub sample_handle_expr: String,
    pub sample_index: u32,
    pub sample_loop_count: u32,
    pub first_decode: NativeAIIdleRecoveryDecode,
    pub second_decode: NativeAIIdleRecoveryDecode,
    pub first_copy: NativeAIIdleRecoveryCopy,
    pub second_copy: NativeAIIdleRecoveryCopy,
    pub wait_ticks: [u32; 3],
    pub after_global_write: NativeAIIdleRecoveryGlobalWrite,
}

pub(crate) const NATIVE_AI_IDLE_RECOVERY_SAMPLE_HANDLE: &str = "[0x53EEC]";

pub(crate) const NATIVE_AI_IDLE_RECOVERY_RESOURCE_BASE: &str = "[0x53A49]+0x8088";

/// BuildNativeAIIdleRecoveryPresentation materializes the exact raw sequence
/// after the HP/gate preflight. The returned value is detached from records so
/// the callback cannot mutate battle state through a borrowed slice.
///
/// Go：`native_ai_idle_recovery.go:86`，`func BuildNativeAIIdleRecoveryPresentation(decision NativeAIIdleRecoveryDecision) (NativeAIIdleRecoveryPresentation, error)`
pub fn build_native_ai_idle_recovery_presentation(
    decision: NativeAIIdleRecoveryDecision,
) -> Result<NativeAIIdleRecoveryPresentation> {
    todo!("battle/native_ai_idle_recovery.go:BuildNativeAIIdleRecoveryPresentation")
}

/// ApplyNativeAIIdleRecoveryWithPresentation performs the 0x13FD4 transaction
/// only after the caller has acknowledged the complete raw presentation. A
/// changed record or a callback error leaves the HP bytes untouched.
///
/// Go：`native_ai_idle_recovery.go:112`，`func ApplyNativeAIIdleRecoveryWithPresentation( records []byte, count, unit int, present func(NativeAIIdleRecoveryPresentation) error, ) (bool, error)`
pub fn apply_native_ai_idle_recovery_with_presentation(
    records: &mut [u8],
    count: i64,
    unit: i64,
    present: Option<&mut dyn FnMut(NativeAIIdleRecoveryPresentation) -> Result<()>>,
) -> Result<bool> {
    todo!("battle/native_ai_idle_recovery.go:ApplyNativeAIIdleRecoveryWithPresentation")
}

/// PlanNativeAIIdleRecovery is the side-effect-free E0 adapter for 0x13fd4.
/// It reads only the raw record fields used by the native function and never
/// writes the detached snapshot.  Presentation/wait calls remain outside this
/// contract.
///
/// Go：`native_ai_idle_recovery.go:150`，`func PlanNativeAIIdleRecovery(records []byte, count, unit int) (NativeAIIdleRecoveryDecision, error)`
pub fn plan_native_ai_idle_recovery(records: &[u8], count: i64, unit: i64) -> Result<NativeAIIdleRecoveryDecision> {
    todo!("battle/native_ai_idle_recovery.go:PlanNativeAIIdleRecovery")
}

/// ApplyNativeAIIdleRecovery commits a previously verified 0x13fd4 decision.
/// The decision is recomputed from the same raw snapshot immediately before
/// writing, so a caller cannot accidentally apply a stale normalized HP value.
///
/// Go：`native_ai_idle_recovery.go:177`，`func ApplyNativeAIIdleRecovery(records []byte, count, unit int) (bool, error)`
pub fn apply_native_ai_idle_recovery(records: &mut [u8], count: i64, unit: i64) -> Result<bool> {
    todo!("battle/native_ai_idle_recovery.go:ApplyNativeAIIdleRecovery")
}
