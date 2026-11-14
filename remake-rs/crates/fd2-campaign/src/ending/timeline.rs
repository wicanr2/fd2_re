//! Go 來源：`remake/internal/ending/timeline.go`（210 行）。
//!
//! Package ending loads recovered ending timelines.  The timelines are kept
//! separate from campaign handlers because their indexed compositors and native
//! text helpers need their own evidence-backed renderer.  Loading never grants
//! playback permission: callers must reject Status values other than "ready".

use super::*;

/// Go：`timeline.go:13`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct Timeline {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"native_handler"`
    pub native_handler: String,
    /// Go 標籤：`json:"resource"`
    pub resource: Resource,
    /// Go 標籤：`json:"status"`
    pub status: String,
    /// Go 標籤：`json:"segments"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub segments: Vec<Segment>,
    /// Go 標籤：`json:"audio_cues,omitempty"`
    #[serde(deserialize_with = "battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub audio_cues: Vec<AudioCue>,
}

/// Go：`timeline.go:22`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct Resource {
    /// Go 標籤：`json:"archive"`
    pub archive: String,
    /// Go 標籤：`json:"index"`
    pub index: i64,
}

/// AudioCue is an address-preserving transcription of one native 0x25977
/// call. Track=-1 is the native stop request; all other values are the raw
/// FDMUS.DAT resource index. Trigger is deliberately a source-stage label,
/// not a claim that the current fail-closed Player can execute that stage.
///
/// Go：`timeline.go:31`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct AudioCue {
    /// Go 標籤：`json:"source"`
    pub source: String,
    /// Go 標籤：`json:"track"`
    pub track: i64,
    /// Go 標籤：`json:"driver_arg"`
    pub driver_arg: i64,
    /// Go 標籤：`json:"runtime_stage"`
    pub runtime_stage: String,
    /// AfterGate 是允許消費此音訊的精確 raw 停止邊界。空字串代表該 cue
    /// 仍只有觀測用途，因為其原版畫面／流程 owner 尚未還原。
    /// Go 標籤：`json:"after_gate,omitempty"`
    #[serde(skip_serializing_if = "String::is_empty")]
    pub after_gate: String,
    /// Go 標籤：`json:"trigger"`
    pub trigger: String,
}

/// Segment deliberately retains only evidence names and native call arguments.
/// It is an editable transcription, not a generic ending scripting language.
///
/// Go：`timeline.go:44`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct Segment {
    /// Go 標籤：`json:"op"`
    pub op: String,
    /// Go 標籤：`json:"source"`
    pub source: String,
    /// Go 標籤：`json:"frame,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub frame: Option<i64>,
    /// Go 標籤：`json:"first_frame,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub first_frame: Option<i64>,
    /// Go 標籤：`json:"last_frame,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_frame: Option<i64>,
    /// Go 標籤：`json:"target,omitempty"`
    #[serde(skip_serializing_if = "String::is_empty")]
    pub target: String,
    /// Go 標籤：`json:"stride,omitempty"`
    #[serde(skip_serializing_if = "battle::go_is_zero")]
    pub stride: i64,
    /// Go 標籤：`json:"transparent,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub transparent: Option<i64>,
    /// Go 標籤：`json:"bytes,omitempty"`
    #[serde(skip_serializing_if = "battle::go_is_zero")]
    pub bytes: i64,
    /// Go 標籤：`json:"from,omitempty"`
    #[serde(skip_serializing_if = "String::is_empty")]
    pub from: String,
    /// Go 標籤：`json:"to,omitempty"`
    #[serde(skip_serializing_if = "String::is_empty")]
    pub to: String,
    /// Go 標籤：`json:"ms,omitempty"`
    #[serde(skip_serializing_if = "battle::go_is_zero")]
    pub ms: i64,
    /// Go 標籤：`json:"resource,omitempty"`
    #[serde(rename = "resource", skip_serializing_if = "Option::is_none")]
    pub ani_resource: Option<i64>,
    /// Go 標籤：`json:"frame_delay_ms,omitempty"`
    #[serde(skip_serializing_if = "battle::go_is_zero")]
    pub frame_delay_ms: i64,
    /// Go 標籤：`json:"skippable,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub skippable: Option<bool>,
    /// Go 標籤：`json:"start,omitempty"`
    #[serde(rename = "start", skip_serializing_if = "Option::is_none")]
    pub palette_start: Option<i64>,
    /// Go 標籤：`json:"end,omitempty"`
    #[serde(rename = "end", skip_serializing_if = "Option::is_none")]
    pub palette_end: Option<i64>,
    /// Go 標籤：`json:"value,omitempty"`
    #[serde(rename = "value", skip_serializing_if = "Option::is_none")]
    pub palette_value: Option<i64>,
    /// Go 標籤：`json:"step,omitempty"`
    #[serde(rename = "step", skip_serializing_if = "battle::go_is_zero")]
    pub palette_step: i64,
    /// Go 標籤：`json:"delay_ms,omitempty"`
    #[serde(rename = "delay_ms", skip_serializing_if = "battle::go_is_zero")]
    pub palette_delay: i64,
    /// Go 標籤：`json:"repeat,omitempty"`
    #[serde(skip_serializing_if = "battle::go_is_zero")]
    pub repeat: i64,
    /// Go 標籤：`json:"tail_delay_ms,omitempty"`
    #[serde(rename = "tail_delay_ms", skip_serializing_if = "battle::go_is_zero")]
    pub tail_delay: i64,
    /// Go 標籤：`json:"first_frame_formula,omitempty"`
    #[serde(skip_serializing_if = "String::is_empty")]
    pub first_frame_formula: String,
    /// Go 標籤：`json:"second_frame_formula,omitempty"`
    #[serde(skip_serializing_if = "String::is_empty")]
    pub second_frame_formula: String,
    /// Go 標籤：`json:"then_dialogue,omitempty"`
    #[serde(deserialize_with = "battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub then_dialogue: Vec<DialogueBlock>,
    /// Go 標籤：`json:"else_dialogue,omitempty"`
    #[serde(deserialize_with = "battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub else_dialogue: Vec<DialogueBlock>,
}

/// DialogueBlock is a count-aligned FDTXT block recovered from an ending text
/// helper. PortraitID is a DATO.DAT resource index: 0x2c39b forwards its first
/// caller argument to 0x1956b, which loads archive 0x51a70 (DATO.DAT).
///
/// Go：`timeline.go:76`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct DialogueBlock {
    /// PortraitID 是 sub_2C39B arg0 的 opening initial portrait，不代表
    /// FDTXT block 內每個 utterance 的 speaker。
    /// Go 標籤：`json:"portrait_id"`
    pub portrait_id: i64,
    /// Go 標籤：`json:"source_dat"`
    pub source_dat: String,
    /// Go 標籤：`json:"script"`
    pub script: String,
    /// Go 標籤：`json:"string_index"`
    pub string_index: i64,
    /// Go 標籤：`json:"scene_index"`
    pub scene_index: i64,
    /// Go 標籤：`json:"line"`
    pub line: i64,
    /// Go 標籤：`json:"count"`
    pub count: i64,
    /// Go 標籤：`json:"native_utterances"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub native_utterances: Vec<NativeDialogueUtterance>,
}

/// NativeDialogueUtterance 保存 sub_2C39B caller portrait 之後，真正由
/// FDTXT speaker control 或 caller fallback 選出的逐句 speaker 與 raw 分頁。
///
/// Go：`timeline.go:91`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeDialogueUtterance {
    /// Go 標籤：`json:"control_source"`
    pub control_source: String,
    /// Go 標籤：`json:"control"`
    pub control: String,
    /// Go 標籤：`json:"operand"`
    pub operand: i64,
    /// Go 標籤：`json:"pages"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub pages: Vec<Vec<String>>,
}

/// PlaybackState tells a presentation adapter why a native ending timeline
/// stopped.  Blocked is intentional: it marks the first operation whose exact
/// renderer has not been recovered, rather than replacing it with a remake
/// approximation.
///
/// Go：`timeline.go:102`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct PlaybackState(pub &'static str);

pub const PLAYBACK_RUNNING: PlaybackState = PlaybackState("running");

pub const PLAYBACK_BLOCKED: PlaybackState = PlaybackState("blocked");

pub const PLAYBACK_COMPLETED: PlaybackState = PlaybackState("completed");

/// Go：`timeline.go:110`，`func LoadTimeline(path string) (*Timeline, error)`
pub fn load_timeline(path: &str) -> Result<Timeline> {
    let raw = battle::read_file(path)?;
    let timeline: Timeline = battle::decode_json(&raw)?;
    let q = battle::go_quote(path);
    if timeline.schema_version != 1
        || timeline.native_handler.is_empty()
        || timeline.resource.archive.is_empty()
        || timeline.resource.index < 0
        || timeline.status.is_empty()
    {
        return Err(campaign_err!("ending timeline {q} has invalid header"));
    }
    if timeline.segments.is_empty() {
        return Err(campaign_err!("ending timeline {q} has no recovered segments"));
    }
    let mut seen_audio_gates = BTreeSet::new();
    let mut seen_audio_sources = BTreeSet::new();
    let mut seen_audio_stages = BTreeSet::new();
    let valid_audio_stages = ["party_cycle_start", "tail_stop", "tail_start"];
    for (i, cue) in timeline.audio_cues.iter().enumerate() {
        if cue.source.is_empty()
            || cue.track < -1
            || cue.track > 0xff
            || cue.driver_arg < 0
            || cue.driver_arg > 1
            || cue.trigger.is_empty()
            || !valid_audio_stages.contains(&cue.runtime_stage.as_str())
        {
            return Err(campaign_err!("ending timeline {q} audio cue {i} is incomplete"));
        }
        if seen_audio_sources.contains(&cue.source) || seen_audio_stages.contains(&cue.runtime_stage) {
            return Err(campaign_err!("ending timeline {q} audio cue {i} duplicates source or runtime stage"));
        }
        seen_audio_sources.insert(cue.source.clone());
        seen_audio_stages.insert(cue.runtime_stage.clone());
        if !cue.after_gate.is_empty() {
            if cue.after_gate != "0x2c548" || seen_audio_gates.contains(&cue.after_gate) {
                return Err(campaign_err!("ending timeline {q} audio cue {i} has an unverified or duplicate gate"));
            }
            seen_audio_gates.insert(cue.after_gate.clone());
        }
    }
    for (i, segment) in timeline.segments.iter().enumerate() {
        if segment.op.is_empty() || segment.source.is_empty() {
            return Err(campaign_err!("ending timeline {q} segment {i} is incomplete"));
        }
        if segment.op == "blit_frame"
            && (segment.frame.is_none()
                || segment.stride <= 0
                || segment.target.is_empty()
                || segment.transparent.is_none())
        {
            return Err(campaign_err!("ending timeline {q} segment {i} has incomplete blit"));
        }
        if segment.op == "blit_frame_sequence" {
            let sequence_ok = match (segment.first_frame, segment.last_frame) {
                (Some(first), Some(last)) => {
                    first >= 0
                        && last >= first
                        && segment.stride > 0
                        && !segment.target.is_empty()
                        && segment.transparent.is_some()
                        && segment.palette_delay > 0
                }
                _ => false,
            };
            if !sequence_ok {
                return Err(campaign_err!("ending timeline {q} segment {i} has incomplete frame sequence"));
            }
        }
        if segment.op == "copy_buffer" && (segment.bytes != BYTES || segment.from.is_empty() || segment.to.is_empty()) {
            return Err(campaign_err!("ending timeline {q} segment {i} has incomplete copy"));
        }
        if segment.op == "ani_play"
            && (segment.ani_resource != Some(2) || segment.frame_delay_ms != 100 || segment.skippable != Some(false))
        {
            return Err(campaign_err!("ending timeline {q} segment {i} has incomplete ANI playback"));
        }
        let valid_ramp = || match (segment.palette_start, segment.palette_end) {
            (Some(start), Some(end)) => {
                segment.palette_step != 0
                    && segment.palette_delay > 0
                    && (0..=63).contains(&start)
                    && (0..=63).contains(&end)
                    && (end - start).wrapping_mul(segment.palette_step) >= 0
            }
            _ => false,
        };
        if segment.op == "palette_ramp" && !valid_ramp() {
            return Err(campaign_err!("ending timeline {q} segment {i} has invalid palette ramp"));
        }
        if segment.op == "palette_ramp_repeat" && (!valid_ramp() || segment.repeat <= 0 || segment.tail_delay < 0) {
            return Err(campaign_err!("ending timeline {q} segment {i} has invalid repeated palette ramp"));
        }
        if (segment.op == "native_composite_loop_opaque" || segment.op == "native_composite_loop_baseline")
            && (segment.first_frame_formula != "(i%4)+1" || segment.second_frame_formula != "(i%4)+5")
        {
            return Err(campaign_err!("ending timeline {q} segment {i} has unsupported composite frame formulas"));
        }
        for (j, block) in segment.then_dialogue.iter().chain(segment.else_dialogue.iter()).enumerate() {
            if block.portrait_id < 0
                || block.source_dat.is_empty()
                || block.script.is_empty()
                || block.string_index < 0
                || block.scene_index < 0
                || block.line < 0
                || block.count <= 0
            {
                return Err(campaign_err!("ending timeline {q} segment {i} dialogue {j} is incomplete"));
            }
            if block.native_utterances.len() as i64 != block.count {
                return Err(campaign_err!(
                    "ending timeline {q} segment {i} dialogue {j} has {} native utterances for count {}",
                    block.native_utterances.len(),
                    block.count
                ));
            }
            for (k, utterance) in block.native_utterances.iter().enumerate() {
                let valid_fdtxt_control = ["FFEC", "FFED", "FFEE", "FFEF"].contains(&utterance.control.as_str());
                let valid_source = (utterance.control_source == "fdtxt" && valid_fdtxt_control)
                    || (utterance.control_source == "caller_2c39b"
                        && utterance.control.is_empty()
                        && utterance.operand == block.portrait_id);
                if !valid_source || utterance.operand < 0 || utterance.operand > 0xffff || utterance.pages.is_empty() {
                    return Err(campaign_err!(
                        "ending timeline {q} segment {i} dialogue {j} utterance {k} has invalid control provenance"
                    ));
                }
                for (page, rows) in utterance.pages.iter().enumerate() {
                    if rows.is_empty() || rows.len() > 4 {
                        return Err(campaign_err!(
                            "ending timeline {q} segment {i} dialogue {j} utterance {k} page {page} has {} rows",
                            rows.len()
                        ));
                    }
                    for (row, text) in rows.iter().enumerate() {
                        // Go `len([]rune(text))`：無效 UTF-8 在 Go 算成 U+FFFD；Rust 的 String 必定是合法 UTF-8。
                        let count = text.chars().count();
                        if count == 0 || count > 13 {
                            return Err(campaign_err!(
                                "ending timeline {q} segment {i} dialogue {j} utterance {k} page {page} row {row} has {count} glyphs"
                            ));
                        }
                    }
                }
            }
        }
    }
    Ok(timeline)
}

impl Timeline {
    /// Ready is deliberately false for a recovered-only timeline. It makes a
    /// future campaign bridge fail closed until every opaque segment has a verified
    /// renderer/text adapter.
    ///
    /// Go：`timeline.go:210`，`func (t Timeline) Ready() bool`
    pub fn ready(&self) -> bool {
        self.status == "ready"
    }
}

#[cfg(test)]
pub(crate) mod timeline_test;
