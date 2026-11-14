//! Go 來源：`remake/internal/campaign/handler_script.go`（228 行）。

use super::*;

/// HandlerSource preserves the original EXE call site for every editable beat.
/// It is audit metadata, not a runtime address used by the remake.
///
/// Go：`handler_script.go:11`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct HandlerSource {
    /// Go 標籤：`json:"addr"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub addr: String,
    /// Go 標籤：`json:"target,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub target: String,
}

/// HandlerNativeCall preserves one nested raw call site inside a composite
/// handler primitive.  It is evidence metadata, not a generic runtime call.
///
/// Go：`handler_script.go:18`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct HandlerNativeCall {
    /// Go 標籤：`json:"source"`
    pub source: HandlerSource,
    /// Go 標籤：`json:"raw_args,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub raw_args: Vec<serde_json::Value>,
}

/// NativeCh23Loop is the two-loop presentation schedule recovered from raw
/// ch23 post.  The renderer deliberately consumes this as fail-closed evidence:
/// all native call sites, register-shaped arguments, stage values, and external
/// raw sources remain visible until an indexed adapter is proven.
///
/// Go：`handler_script.go:27`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeCh23Loop {
    /// Go 標籤：`json:"phase"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub phase: String,
    /// Go 標籤：`json:"repeat"`
    pub repeat: i64,
    /// Go 標籤：`json:"draw"`
    pub draw: HandlerNativeCall,
    /// Go 標籤：`json:"tick"`
    pub tick: HandlerNativeCall,
    /// Go 標籤：`json:"palette,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub palette: Option<HandlerNativeCall>,
    /// Go 標籤：`json:"stage"`
    pub stage: HandlerNativeCall,
    /// Go 標籤：`json:"stage_values"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub stage_values: Vec<i64>,
    /// Go 標籤：`json:"stage_latch_source"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub stage_latch_source: String,
    /// Go 標籤：`json:"tick_counter_source"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub tick_counter_source: String,
    /// Go 標籤：`json:"palette_table_source,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub palette_table_source: String,
}

/// Native2189ALoop preserves the shared ten-pass indexed presentation helper
/// called by the ch22 post handler.  The inner call-sites and push-shaped raw
/// arguments remain evidence metadata; no portrait, effect, or gameplay name
/// is inferred from the buffer copies. Runtime accepts only the two recovered
/// outer call shapes and requires complete indexed provenance.
///
/// Go：`handler_script.go:45`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct Native2189ALoop {
    /// Slot/InitialRadius/RadiusStep are filled by the handler compiler from the
    /// three outer raw immediates. They are not authored separately in JSON, so
    /// evidence metadata and runtime ABI cannot drift apart.
    /// Go 標籤：`json:"slot,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub slot: i64,
    /// Go 標籤：`json:"initial_radius,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub initial_radius: i64,
    /// Go 標籤：`json:"radius_step,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub radius_step: i64,
    /// Go 標籤：`json:"repeat"`
    pub repeat: i64,
    /// Go 標籤：`json:"step_source"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub step_source: String,
    /// Go 標籤：`json:"work_offset"`
    pub work_offset: i64,
    /// Go 標籤：`json:"work_stride"`
    pub work_stride: i64,
    /// Go 標籤：`json:"map_rows"`
    pub map_rows: i64,
    /// Go 標籤：`json:"map_columns"`
    pub map_columns: i64,
    /// Go 標籤：`json:"clip_width"`
    pub clip_width: i64,
    /// Go 標籤：`json:"clip_height"`
    pub clip_height: i64,
    /// Go 標籤：`json:"present_stride"`
    pub present_stride: i64,
    /// Go 標籤：`json:"map_draw"`
    pub map_draw: HandlerNativeCall,
    /// Go 標籤：`json:"composite"`
    pub composite: HandlerNativeCall,
    /// Go 標籤：`json:"stage"`
    pub stage: HandlerNativeCall,
    /// Go 標籤：`json:"present"`
    pub present: HandlerNativeCall,
    /// Go 標籤：`json:"tail"`
    pub tail: HandlerNativeCall,
}

/// HandlerCondition is the editable, evidence-level predicate attached to a
/// structured handler branch.  Each predicate is a direct transcription of a
/// native helper; this is deliberately not a generic expression language.
///
/// Go：`handler_script.go:71`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct HandlerCondition {
    /// Go 標籤：`json:"op"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub op: String,
    /// Go 標籤：`json:"unit_slots,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub unit_slots: Vec<i64>,
    /// NativeInventoryItemID is the unsigned byte passed to 0x24b14. The
    /// compiler accepts it only for the exact raw inventory-prefix predicate;
    /// it is not a normalized inventory search.
    /// Go 標籤：`json:"native_inventory_item_id,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_inventory_item_id: Option<i64>,
    /// NativePersistentIdentity is the unsigned byte compared at persistent
    /// record+0x08 by 0x24bde/0x33499. The record field remains raw evidence,
    /// not a character-name assertion.
    /// Go 標籤：`json:"native_persistent_identity,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_persistent_identity: Option<i64>,
    /// Threshold is used only by raw count predicates; it is never inferred
    /// from a roster size or normalized alive count.
    /// Go 標籤：`json:"threshold,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub threshold: Option<i64>,
    /// NativeRound is the raw [0x53bef] comparison used by handlers such as
    /// ch15_post. It is intentionally separate from the normalized battle Turn.
    /// Go 標籤：`json:"native_round,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_round: Option<i64>,
    /// NativeRecordWordOffset/Value identify a direct raw record-word compare;
    /// only the recovered +0x42 u16 contract is currently accepted.
    /// Go 標籤：`json:"native_record_word_offset,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_record_word_offset: Option<i64>,
    /// Go 標籤：`json:"native_record_word_value,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_record_word_value: Option<i64>,
    /// UnitSlot selects the raw runtime record for native_record_word_gte.
    /// Go 標籤：`json:"unit_slot,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub unit_slot: Option<i64>,
    /// Any contains a bounded OR of already-proven native predicates. It is
    /// intentionally not a general expression language; the compiler accepts
    /// only the raw predicate ops it knows how to validate.
    /// Go 標籤：`json:"any,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub any: Vec<HandlerCondition>,
    /// CharID is the one-byte permanent-player ID accepted by native 0x33499.
    /// It is meaningful only for roster_has, never a portrait/NPC identifier.
    /// Go 標籤：`json:"char_id,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub char_id: Option<i64>,
    /// Go 標籤：`json:"event_state_index,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub event_state_index: Option<i64>,
    /// Go 標籤：`json:"event_state_value,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub event_state_value: Option<i64>,
    /// RequiredSlotCount 是此條件成立時由同一原生 producer 證實的精確
    /// runtime frontier。它讓分支內 slot 操作可驗證，但不會擴張外層 frontier。
    /// Go 標籤：`json:"required_slot_count,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub required_slot_count: Option<i64>,
}

/// HandlerRepeatHint records a directly recovered fixed-count loop around one
/// native operation. It is evidence metadata, not a generic scripting loop.
///
/// Go：`handler_script.go:110`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct HandlerRepeatHint {
    /// Go 標籤：`json:"loop_back_to"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub loop_back_to: String,
    /// Go 標籤：`json:"limit"`
    pub limit: i64,
}

/// HandlerBeat is the lossless editable IR exported from one hard-coded EXE
/// handler.  Fields are intentionally sparse: each Op uses only its matching
/// fields, and RawArgs keeps unclassified native calls visible to editors.
///
/// Go：`handler_script.go:118`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct HandlerBeat {
    /// Go 標籤：`json:"op"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub op: String,
    /// Go 標籤：`json:"source,omitempty"`
    pub source: HandlerSource,
    /// Go 標籤：`json:"chapter,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub chapter: Option<i64>,
    /// Go 標籤：`json:"chapter_expr,omitempty"`
    #[serde(skip_serializing_if = "serde_json::Value::is_null")]
    pub chapter_expr: serde_json::Value,
    /// Go 標籤：`json:"grid_x,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub grid_x: Option<i64>,
    /// Go 標籤：`json:"grid_y,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub grid_y: Option<i64>,
    /// Go 標籤：`json:"text_index,omitempty"`
    #[serde(skip_serializing_if = "serde_json::Value::is_null")]
    pub text_index: serde_json::Value,
    /// Go 標籤：`json:"text_table,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub text_table: String,
    /// Go 標籤：`json:"acting_id,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub acting_id: Option<i64>,
    /// Go 標籤：`json:"unit_slot,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub unit_slot: Option<i64>,
    /// Go 標籤：`json:"unit_slot_expr,omitempty"`
    #[serde(skip_serializing_if = "serde_json::Value::is_null")]
    pub unit_slot_expr: serde_json::Value,
    /// Go 標籤：`json:"group,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub group: Option<i64>,
    /// RawPlacementGate is the exact byte observed by 0x10C50 at [0x53AFA].
    /// Pointer form preserves an explicit zero and lets the compiler reject a
    /// lossy original-handler export instead of guessing from the group number.
    /// Go 標籤：`json:"raw_placement_gate,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub raw_placement_gate: Option<i64>,
    /// Go 標籤：`json:"char_id,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub char_id: Option<i64>,
    /// Go 標籤：`json:"item_id,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub item_id: Option<i64>,
    /// Go 標籤：`json:"resource_id,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub resource_id: Option<i64>,
    /// Go 標籤：`json:"sfx_index,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sfx_index: Option<i64>,
    /// Go 標籤：`json:"track,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub track: Option<i64>,
    /// Go 標籤：`json:"loop,omitempty"`
    #[serde(rename = "loop", skip_serializing_if = "Option::is_none")]
    pub r#loop: Option<i64>,
    /// Go 標籤：`json:"direction,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub direction: Option<i64>,
    /// Go 標籤：`json:"repeat,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub repeat: Option<i64>,
    /// Go 標籤：`json:"repeat_hint,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub repeat_hint: Option<HandlerRepeatHint>,
    /// Go 標籤：`json:"unit_present,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub unit_present: Option<HandlerUnitPresent>,
    /// Go 標籤：`json:"direct_record_patch,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub direct_record_patch: Option<HandlerDirectRecordPatch>,
    /// Go 標籤：`json:"indexed_transition,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub indexed_transition: Option<HandlerIndexedTransition>,
    /// Go 標籤：`json:"native_ch23_loop,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_ch23_loop: Option<NativeCh23Loop>,
    /// Go 標籤：`json:"native_2189a_loop,omitempty"`
    #[serde(rename = "native_2189a_loop", skip_serializing_if = "Option::is_none")]
    pub native2189_a_loop: Option<Native2189ALoop>,
    /// Go 標籤：`json:"ms,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub ms: Option<i64>,
    /// Go 標籤：`json:"variant,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub variant: String,
    /// Go 標籤：`json:"value,omitempty"`
    #[serde(skip_serializing_if = "serde_json::Value::is_null")]
    pub value: serde_json::Value,
    /// Go 標籤：`json:"native_target,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub native_target: String,
    /// NativeSemantic is the evidence-index label emitted for a classified
    /// native call. It is audit metadata; lowering still keys on NativeTarget,
    /// source address, raw arguments and explicit bindings.
    /// Go 標籤：`json:"native_semantic,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub native_semantic: String,
    /// Go 標籤：`json:"native_confidence,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub native_confidence: String,
    /// Go 標籤：`json:"native_evidence,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub native_evidence: Vec<String>,
    /// Go 標籤：`json:"raw_args,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub raw_args: Vec<serde_json::Value>,
    /// Go 標籤：`json:"args,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub args: Vec<serde_json::Value>,
    /// Go 標籤：`json:"condition,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub condition: Option<HandlerCondition>,
    /// Go 標籤：`json:"then,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub then: Vec<HandlerBeat>,
    /// Go 標籤：`json:"else,omitempty"`
    #[serde(
        rename = "else",
        deserialize_with = "fd2_rules::battle::null_as_default",
        skip_serializing_if = "Vec::is_empty"
    )]
    pub r#else: Vec<HandlerBeat>,
}

/// HandlerScript is a chapter pre/post handler in editable JSON form.  It is
/// deliberately distinct from Beat: it preserves original grid/text/resource
/// identifiers so a campaign author can edit or audit source choreography
/// before compiling it to a map-specific runtime Beat sequence.
///
/// Go：`handler_script.go:170`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct HandlerScript {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"chapter"`
    pub chapter: i64,
    /// Go 標籤：`json:"phase"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub phase: String,
    /// Go 標籤：`json:"handler"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub handler: String,
    /// Go 標籤：`json:"beats"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub beats: Vec<HandlerBeat>,
    /// Go 標籤：`json:"diagnostics,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "BTreeMap::is_empty")]
    pub diagnostics: BTreeMap<String, i64>,
}

/// LoadHandlerScript reads an editable EXE-handler export and rejects malformed
/// scripts early.  Unknown operations are valid data: they are explicitly
/// preserved until their native semantics have been RE'd.
///
/// Go：`handler_script.go:182`，`func LoadHandlerScript(path string) (*HandlerScript, error)`
pub fn load_handler_script(path: &str) -> Result<HandlerScript> {
    let raw = battle::read_file(path)?;
    let script: HandlerScript = battle::decode_json(&raw)?;
    let q = battle::go_quote(path);
    if script.schema_version != 1 {
        return Err(campaign_err!("handler script {q} schema_version={}, want 1", script.schema_version));
    }
    if script.phase != "pre" && script.phase != "post" {
        return Err(campaign_err!("handler script {q} has invalid phase {}", battle::go_quote(&script.phase)));
    }
    if script.handler.is_empty() {
        return Err(campaign_err!("handler script {q} has no handler"));
    }
    validate_handler_beats(path, "beats", &script.beats)?;
    Ok(script)
}

/// Go：`handler_script.go:206`，`func validateHandlerBeats(path, location string, beats []HandlerBeat) error`
pub(crate) fn validate_handler_beats(path: &str, location: &str, beats: &[HandlerBeat]) -> Result<()> {
    let q = battle::go_quote(path);
    for (i, beat) in beats.iter().enumerate() {
        let at = format!("{location}[{i}]");
        if beat.op.is_empty() {
            return Err(campaign_err!("handler script {q} {at} has no op"));
        }
        if (beat.op == "native_call" || beat.op == "unresolved_native_call")
            && (beat.native_target.is_empty()
                || beat.native_semantic.is_empty()
                || beat.native_confidence.is_empty()
                || beat.native_evidence.is_empty())
        {
            return Err(campaign_err!("handler script {q} {at} native call lacks target/semantic/confidence/evidence"));
        }
        if beat.op == "if" {
            validate_handler_beats(path, &format!("{at}.then"), &beat.then)?;
            validate_handler_beats(path, &format!("{at}.else"), &beat.r#else)?;
        }
    }
    Ok(())
}

// 測試：Go `remake/internal/campaign/handler_script_test.go` 逐條移植。
#[cfg(test)]
mod handler_script_test;
