//! Go 來源：`remake/internal/ending/timeline.go`（210 行）。
//!
//! Package ending loads recovered ending timelines.  The timelines are kept
//! separate from campaign handlers because their indexed compositors and native
//! text helpers need their own evidence-backed renderer.  Loading never grants
//! playback permission: callers must reject Status values other than "ready".

use super::*;

/// Go：`timeline.go:13`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
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
    pub segments: Vec<Segment>,
    /// Go 標籤：`json:"audio_cues,omitempty"`
    pub audio_cues: Vec<AudioCue>,
}

/// Go：`timeline.go:22`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
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
#[derive(Debug, Clone, Default, PartialEq, Eq)]
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
    pub after_gate: String,
    /// Go 標籤：`json:"trigger"`
    pub trigger: String,
}

/// Segment deliberately retains only evidence names and native call arguments.
/// It is an editable transcription, not a generic ending scripting language.
///
/// Go：`timeline.go:44`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Segment {
    /// Go 標籤：`json:"op"`
    pub op: String,
    /// Go 標籤：`json:"source"`
    pub source: String,
    /// Go 標籤：`json:"frame,omitempty"`
    pub frame: Option<i64>,
    /// Go 標籤：`json:"first_frame,omitempty"`
    pub first_frame: Option<i64>,
    /// Go 標籤：`json:"last_frame,omitempty"`
    pub last_frame: Option<i64>,
    /// Go 標籤：`json:"target,omitempty"`
    pub target: String,
    /// Go 標籤：`json:"stride,omitempty"`
    pub stride: i64,
    /// Go 標籤：`json:"transparent,omitempty"`
    pub transparent: Option<i64>,
    /// Go 標籤：`json:"bytes,omitempty"`
    pub bytes: i64,
    /// Go 標籤：`json:"from,omitempty"`
    pub from: String,
    /// Go 標籤：`json:"to,omitempty"`
    pub to: String,
    /// Go 標籤：`json:"ms,omitempty"`
    pub ms: i64,
    /// Go 標籤：`json:"resource,omitempty"`
    pub ani_resource: Option<i64>,
    /// Go 標籤：`json:"frame_delay_ms,omitempty"`
    pub frame_delay_ms: i64,
    /// Go 標籤：`json:"skippable,omitempty"`
    pub skippable: Option<bool>,
    /// Go 標籤：`json:"start,omitempty"`
    pub palette_start: Option<i64>,
    /// Go 標籤：`json:"end,omitempty"`
    pub palette_end: Option<i64>,
    /// Go 標籤：`json:"value,omitempty"`
    pub palette_value: Option<i64>,
    /// Go 標籤：`json:"step,omitempty"`
    pub palette_step: i64,
    /// Go 標籤：`json:"delay_ms,omitempty"`
    pub palette_delay: i64,
    /// Go 標籤：`json:"repeat,omitempty"`
    pub repeat: i64,
    /// Go 標籤：`json:"tail_delay_ms,omitempty"`
    pub tail_delay: i64,
    /// Go 標籤：`json:"first_frame_formula,omitempty"`
    pub first_frame_formula: String,
    /// Go 標籤：`json:"second_frame_formula,omitempty"`
    pub second_frame_formula: String,
    /// Go 標籤：`json:"then_dialogue,omitempty"`
    pub then_dialogue: Vec<DialogueBlock>,
    /// Go 標籤：`json:"else_dialogue,omitempty"`
    pub else_dialogue: Vec<DialogueBlock>,
}

/// DialogueBlock is a count-aligned FDTXT block recovered from an ending text
/// helper. PortraitID is a DATO.DAT resource index: 0x2c39b forwards its first
/// caller argument to 0x1956b, which loads archive 0x51a70 (DATO.DAT).
///
/// Go：`timeline.go:76`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
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
    pub native_utterances: Vec<NativeDialogueUtterance>,
}

/// NativeDialogueUtterance 保存 sub_2C39B caller portrait 之後，真正由
/// FDTXT speaker control 或 caller fallback 選出的逐句 speaker 與 raw 分頁。
///
/// Go：`timeline.go:91`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeDialogueUtterance {
    /// Go 標籤：`json:"control_source"`
    pub control_source: String,
    /// Go 標籤：`json:"control"`
    pub control: String,
    /// Go 標籤：`json:"operand"`
    pub operand: i64,
    /// Go 標籤：`json:"pages"`
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
    todo!("ending/timeline.go:LoadTimeline")
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
