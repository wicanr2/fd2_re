//! Go 來源：`remake/internal/ending/montage.go`（349 行）。

use super::*;

/// Montage is the evidence-only transcription of the first 0x2c548 party
/// cycle. It records native resource/slot selection without pretending that
/// the later portrait, input, and battle-renderer branches are playable.
///
/// Go：`montage.go:14`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct Montage {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"native_handler"`
    pub native_handler: String,
    /// Go 標籤：`json:"status"`
    pub status: String,
    /// Go 標籤：`json:"allocations"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub allocations: Vec<MontageBuffer>,
    /// Go 標籤：`json:"resources"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub resources: Vec<MontageAsset>,
    /// Go 標籤：`json:"party_cycle"`
    pub party_cycle: PartyCycleSpec,
    /// Go 標籤：`json:"gate"`
    pub gate: MontageGate,
}

/// Go：`montage.go:24`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct MontageBuffer {
    /// Go 標籤：`json:"bytes"`
    pub bytes: i64,
    /// Go 標籤：`json:"purpose"`
    pub purpose: String,
    /// Go 標籤：`json:"source"`
    pub source: String,
}

/// Go：`montage.go:30`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct MontageAsset {
    /// Go 標籤：`json:"archive"`
    pub archive: String,
    /// Go 標籤：`json:"index"`
    pub index: i64,
    /// Go 標籤：`json:"source"`
    pub source: String,
    /// Go 標籤：`json:"role"`
    pub role: String,
}

/// Go：`montage.go:37`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct PartyCycleSpec {
    /// Go 標籤：`json:"source"`
    pub source: String,
    /// Go 標籤：`json:"count_global"`
    pub count_global: String,
    /// Go 標籤：`json:"unit_base_global"`
    pub unit_base_global: String,
    /// Go 標籤：`json:"unit_stride"`
    pub unit_stride: i64,
    /// Go 標籤：`json:"slot_selection"`
    pub slot_selection: String,
    /// Go 標籤：`json:"visual_group_offset"`
    pub visual_group_offset: i64,
    /// Go 標籤：`json:"figure_archive"`
    pub figure_archive: String,
    /// Go 標籤：`json:"figure_indices"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub figure_indices: Vec<String>,
    /// Go 標籤：`json:"figure_renderer"`
    pub figure_renderer: String,
    /// Go 標籤：`json:"frame_renderer"`
    pub frame_renderer: String,
    /// Go 標籤：`json:"initial_frames"`
    pub initial_frames: i64,
    /// Go 標籤：`json:"frame_delay_ticks"`
    pub frame_delay_ticks: i64,
    /// Go 標籤：`json:"source_range"`
    pub source_range: String,
    /// Go 標籤：`json:"portrait_text"`
    pub portrait_text: PortraitTextSpec,
    /// Go 標籤：`json:"dialogue_frame_layout"`
    pub dialogue_frame_layout: DialogueFrameLayoutSpec,
    /// Go 標籤：`json:"figure_fade"`
    pub figure_fade: FigureFadeSpec,
    /// Go 標籤：`json:"mirror_branch"`
    pub mirror_branch: MirrorBranchSpec,
}

/// Go：`montage.go:57`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct FigureFadeSpec {
    /// Go 標籤：`json:"source"`
    pub source: String,
    /// Go 標籤：`json:"unit_side_offset"`
    pub unit_side_offset: i64,
    /// Go 標籤：`json:"required_unit_side"`
    pub required_unit_side: i64,
    /// Go 標籤：`json:"work_stride"`
    pub work_stride: i64,
    /// Go 標籤：`json:"viewport_width"`
    pub viewport_width: i64,
    /// Go 標籤：`json:"viewport_height"`
    pub viewport_height: i64,
    /// Go 標籤：`json:"platform"`
    pub platform: FigureFadeAsset,
    /// Go 標籤：`json:"figure"`
    pub figure: FigureFadeAsset,
    /// Go 標籤：`json:"stage_start"`
    pub stage_start: i64,
    /// Go 標籤：`json:"stage_end"`
    pub stage_end: i64,
    /// Go 標籤：`json:"stage_shift_bytes"`
    pub stage_shift_bytes: i64,
    /// Go 標籤：`json:"palette_delta_formula"`
    #[serde(rename = "palette_delta_formula")]
    pub palette_formula: String,
    /// Go 標籤：`json:"restore_buffer_bytes"`
    #[serde(rename = "restore_buffer_bytes")]
    pub restore_bytes: i64,
    /// Go 標籤：`json:"restore_copy"`
    pub restore_copy: String,
    /// Go 標籤：`json:"post_figure_snapshot"`
    pub post_figure_snapshot: String,
}

/// Go：`montage.go:75`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct FigureFadeAsset {
    /// Go 標籤：`json:"resource"`
    pub resource: String,
    /// Go 標籤：`json:"x"`
    pub x: i64,
    /// Go 標籤：`json:"y"`
    pub y: i64,
    /// Go 標籤：`json:"effect"`
    pub effect: String,
    /// Go 標籤：`json:"frame"`
    pub frame: i64,
}

/// MirrorBranchSpec is the separately recovered unit[+6]==0 path in
/// 0x29164. It is metadata only: loading it does not enable a renderer.
///
/// Go：`montage.go:85`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct MirrorBranchSpec {
    /// Go 標籤：`json:"source"`
    pub source: String,
    /// Go 標籤：`json:"required_unit_side"`
    pub required_unit_side: i64,
    /// Go 標籤：`json:"side_flag_argument"`
    pub side_flag_argument: String,
    /// Go 標籤：`json:"work_stride"`
    pub work_stride: i64,
    /// Go 標籤：`json:"stage_start"`
    pub stage_start: i64,
    /// Go 標籤：`json:"stage_end"`
    pub stage_end: i64,
    /// Go 標籤：`json:"stage_shift_bytes"`
    pub stage_shift_bytes: i64,
    /// Go 標籤：`json:"palette_delta_formula"`
    pub palette_delta_formula: String,
    /// Go 標籤：`json:"primary_source"`
    pub primary_source: String,
    /// Go 標籤：`json:"secondary_source"`
    pub secondary_source: String,
    /// Go 標籤：`json:"secondary_when"`
    pub secondary_when: String,
    /// Go 標籤：`json:"platform_when"`
    pub platform_when: String,
    /// Go 標籤：`json:"platform"`
    pub platform: FigureFadeAsset,
    /// Go 標籤：`json:"renderer"`
    pub renderer: String,
}

/// Go：`montage.go:102`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct PortraitTextSpec {
    /// Go 標籤：`json:"source"`
    pub source: String,
    /// Go 標籤：`json:"portrait_archive"`
    pub portrait_archive: String,
    /// Go 標籤：`json:"portrait_index"`
    pub portrait_index: String,
    /// Go 標籤：`json:"current_text_table"`
    pub current_text_table: String,
    /// Go 標籤：`json:"permanent_text_table"`
    pub permanent_text_table: String,
    /// Go 標籤：`json:"fields"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub fields: Vec<MontageTextField>,
    /// Go 標籤：`json:"glyph_style"`
    pub glyph_style: MontageGlyphStyle,
    /// Go 標籤：`json:"cycle"`
    pub cycle: PortraitCycleSpec,
    /// Go 標籤：`json:"input"`
    pub input: MontageInput,
}

/// Go：`montage.go:114`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct PortraitCycleSpec {
    /// Go 標籤：`json:"source"`
    pub source: String,
    /// Go 標籤：`json:"destination"`
    pub destination: i64,
    /// Go 標籤：`json:"normal_frame"`
    pub normal_frame: i64,
    /// Go 標籤：`json:"mouth_frame"`
    pub mouth_frame: i64,
    /// Go 標籤：`json:"initial_countdown"`
    pub initial_countdown: i64,
    /// Go 標籤：`json:"reset_formula"`
    pub reset_formula: String,
    /// Go 標籤：`json:"decrement"`
    pub decrement: String,
    /// Go 標籤：`json:"mouth_condition"`
    pub mouth_condition: String,
    /// Go 標籤：`json:"regular_iterations"`
    pub regular_iterations: i64,
    /// Go 標籤：`json:"final_loop_index"`
    pub final_loop_index: i64,
    /// Go 標籤：`json:"final_iterations"`
    pub final_iterations: i64,
    /// Go 標籤：`json:"special_epilogue_from_tick"`
    pub special_epilogue_from_tick: i64,
    /// Go 標籤：`json:"wait_ticks"`
    pub wait_ticks: i64,
}

/// Go：`montage.go:130`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct MontageTextField {
    /// Go 標籤：`json:"table"`
    pub table: String,
    /// Go 標籤：`json:"index"`
    pub index: String,
    /// Go 標籤：`json:"destination"`
    pub destination: String,
    /// Go 標籤：`json:"meaning"`
    pub meaning: String,
}

/// PortraitTextPlan is the verified index/destination dataflow emitted by
/// 0x2c7a4..0x2c967 for one party unit. It intentionally stops before glyph
/// rendering; EDI is the native outer portrait-loop counter, not a unit byte.
///
/// Go：`montage.go:140`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct PortraitTextPlan {
    pub portrait_id: i64,
    pub name_label: TextPlacement,
    pub character_name: TextPlacement,
    pub class_label: TextPlacement,
    pub class_name: TextPlacement,
    pub epilogue: TextPlacement,
}

/// Go：`montage.go:149`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct TextPlacement {
    pub table: String,
    pub index: i64,
    pub destination: i64,
}

/// Go：`montage.go:155`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct MontageGlyphStyle {
    /// Go 標籤：`json:"stride"`
    pub stride: i64,
    /// Go 標籤：`json:"foreground"`
    pub foreground: i64,
    /// Go 標籤：`json:"shadow"`
    pub shadow: i64,
    /// Go 標籤：`json:"background"`
    pub background: i64,
}

/// Go：`montage.go:162`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct MontageInput {
    /// Go 標籤：`json:"poll"`
    pub poll: String,
    /// Go 標籤：`json:"skip_action"`
    pub skip_action: String,
    /// Go 標籤：`json:"source"`
    pub source: String,
}

/// Go：`montage.go:168`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct DialogueFrameLayoutSpec {
    /// Go 標籤：`json:"source"`
    pub source: String,
    /// Go 標籤：`json:"archive"`
    pub archive: String,
    /// Go 標籤：`json:"resource"`
    pub resource: i64,
    /// Go 標籤：`json:"destination"`
    pub destination: String,
    /// Go 標籤：`json:"stride"`
    pub stride: i64,
    /// Go 標籤：`json:"arg8"`
    pub arg8: i64,
    /// Go 標籤：`json:"argC"`
    #[serde(rename = "argC")]
    pub arg_c: i64,
    /// Go 標籤：`json:"arg10"`
    pub arg10: i64,
    /// Go 標籤：`json:"arg14"`
    pub arg14: i64,
    /// Go 標籤：`json:"role"`
    pub role: String,
}

/// Go：`montage.go:181`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct DialogueFramePlacement {
    pub resource_index: i64,
    pub destination_byte: i64,
}

/// Go：`montage.go:186`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct MontageGate {
    /// Go 標籤：`json:"source"`
    pub source: String,
    /// Go 標籤：`json:"reason"`
    pub reason: String,
}

/// Go：`montage.go:191`，`func LoadMontage(path string) (*Montage, error)`
pub fn load_montage(path: &str) -> Result<Montage> {
    let raw = battle::read_file(path)?;
    let m: Montage = battle::decode_json(&raw)?;
    let s = |v: &str| v.to_string();
    let pc = &m.party_cycle;
    let pt = &pc.portrait_text;
    let ff = &pc.figure_fade;
    let field = |table: &str, index: &str, destination: &str, meaning: &str| MontageTextField {
        table: s(table),
        index: s(index),
        destination: s(destination),
        meaning: s(meaning),
    };
    let tai_platform =
        FigureFadeAsset { resource: s("TAI.DAT#3"), x: 164, y: 157, effect: s("transparent_noop"), frame: 0 };
    let ok = m.schema_version == 1
        && m.native_handler == "0x2c548"
        && m.status == "mapped_first_party_cycle_fail_closed"
        && m.allocations.len() == 3
        && m.allocations[0].bytes == 0x1f400
        && m.allocations[1].bytes == 0xfa00
        && m.allocations[2].bytes == 0xfa00
        && m.resources.len() == 2
        && m.resources[0].archive == "TAI.DAT"
        && m.resources[0].index == 3
        && m.resources[0].role == "raw 10x3 fully-transparent sprite; renderer role unrecovered"
        && m.resources[1].archive == "FDOTHER.DAT"
        && m.resources[1].index == 56
        && pc.source == "0x2c5d7"
        && pc.count_global == "0x53bfb"
        && pc.unit_base_global == "0x53a45"
        && pc.unit_stride == 0x50
        && pc.visual_group_offset == 7
        && pc.slot_selection == "i==0?1:i==1?0:i"
        && pc.figure_archive == "FIGANI.DAT"
        && pc.figure_indices.len() == 2
        && pc.figure_indices[0] == "group*3+1"
        && pc.figure_indices[1] == "group*3"
        && pc.figure_renderer == "0x29164"
        && pc.frame_renderer == "0x2b9a1"
        && pc.initial_frames == 20
        && pc.frame_delay_ticks == 1
        && pc.source_range == "0x2c5e3..0x2c9a9"
        && pt.source == "0x2c7a4..0x2c967"
        && pt.portrait_archive == "DATO.DAT"
        && pt.portrait_index == "unit[+7]"
        && pt.current_text_table == "FDTXT_031"
        && pt.permanent_text_table == "FDTXT_000"
        && pt.fields.len() == 5
        && pt.fields[0] == field("current", "10", "staging+0x16e9", "name_label")
        && pt.fields[1] == field("permanent", "unit[+8]+1", "staging+0x171b", "character_name")
        && pt.fields[2] == field("current", "11", "staging+0x2fe9", "class_label")
        && pt.fields[3] == field("permanent", "unit[+0x20]+0x96", "staging+0x301b", "class_name")
        && pt.fields[4] == field("current", "edi<0xdc ? unit[+8]+0x0c : 0x2d", "staging+0x7d08", "epilogue")
        && pc.dialogue_frame_layout
            == (DialogueFrameLayoutSpec {
                source: s("0x2c773->0x168b6"),
                archive: s("FDOTHER.DAT"),
                resource: 5,
                destination: s("portrait_restore_buffer_C"),
                stride: 320,
                arg8: 5,
                arg_c: 7,
                arg10: 5,
                arg14: 5,
                role: s("native dialogue-frame/grid staging; later DATO portrait paste is 0x4e8af"),
            })
        && pt.glyph_style == (MontageGlyphStyle { stride: 320, foreground: 0xcd, shadow: 0x4c, background: 0 })
        && pt.cycle
            == (PortraitCycleSpec {
                source: s("0x2c430,0x2c43f,0x2c7cf..0x2c9a9"),
                destination: 0x0c88,
                normal_frame: 0,
                mouth_frame: 3,
                initial_countdown: 0,
                reset_formula: s("(random_byte&0x1f)+0x28"),
                decrement: s("only_when_nonzero_before_frame_selection"),
                mouth_condition: s("countdown<2"),
                regular_iterations: 0xdc,
                final_loop_index: 0,
                final_iterations: 0x1b8,
                special_epilogue_from_tick: 0xdc,
                wait_ticks: 1,
            })
        && pt.input
            == (MontageInput {
                poll: s("0x10620"),
                skip_action: s("outer_counter=1;0x4e031"),
                source: s("0x2c950..0x2c961"),
            })
        && ff.source == "0x291197..0x29258"
        && ff.unit_side_offset == 6
        && ff.required_unit_side == 1
        && ff.work_stride == 640
        && ff.viewport_width == 320
        && ff.viewport_height == 200
        && ff.platform == tai_platform
        && ff.figure == (FigureFadeAsset { resource: s("secondary_figani"), frame: 0, ..Default::default() })
        && ff.stage_start == 8
        && ff.stage_end == 0
        && ff.stage_shift_bytes == 10
        && ff.palette_formula == "stage*6"
        && ff.restore_bytes == 0xfa00
        && ff.restore_copy == "B->A(dstStride640,srcStride320,width320,rows200)"
        && ff.post_figure_snapshot == "memmove(C,B,64000)"
        && pc.mirror_branch
            == (MirrorBranchSpec {
                source: s("0x2927e..0x29357"),
                required_unit_side: 0,
                side_flag_argument: s("arg4"),
                work_stride: 640,
                stage_start: 8,
                stage_end: 0,
                stage_shift_bytes: 10,
                palette_delta_formula: s("stage*6"),
                primary_source: s("staging+0x140-stage*10"),
                secondary_source: s("staging+0x140"),
                secondary_when: s("arg4==0"),
                platform_when: s("arg4==0"),
                platform: tai_platform.clone(),
                renderer: s("0x29164"),
            })
        && m.gate.source == "0x2c5e3"
        && !m.gate.reason.is_empty();
    if !ok {
        return Err(campaign_err!("ending montage {} is incomplete or unsupported", battle::go_quote(path)));
    }
    Ok(m)
}

impl Montage {
    /// PlanPortraitText materializes the five native 0x2c548 text calls without
    /// interpreting their glyph/control-code renderer. edi is the outer party
    /// portrait-loop counter observed at 0x2c8f7; the special 0x2d index is chosen
    /// only once edi reaches 0xdc.
    ///
    /// Go：`montage.go:225`，`func (m Montage) PlanPortraitText(unit []byte, edi int) (PortraitTextPlan, error)`
    pub fn plan_portrait_text(&self, unit: &[u8], edi: i64) -> Result<PortraitTextPlan> {
        if self.status != "mapped_first_party_cycle_fail_closed" || unit.len() < 0x21 || edi < 0 {
            return Err(campaign_err!("ending: invalid portrait text inputs"));
        }
        let mut epilogue = i64::from(unit[8]) + 0x0c;
        if edi >= 0xdc {
            epilogue = 0x2d;
        }
        let place =
            |table: &str, index: i64, destination: i64| TextPlacement { table: table.into(), index, destination };
        Ok(PortraitTextPlan {
            portrait_id: i64::from(unit[7]),
            name_label: place("current", 10, 0x16e9),
            character_name: place("permanent", i64::from(unit[8]) + 1, 0x171b),
            class_label: place("current", 11, 0x2fe9),
            class_name: place("permanent", i64::from(unit[0x20]) + 0x96, 0x301b),
            epilogue: place("current", epilogue, 0x7d08),
        })
    }

    /// PlanDialogueFrameGrid transcribes every sub_1685c call made by 0x168b6 for the
    /// recovered (stride=320, arg8=5, argC=7, arg10=5, arg14=5) invocation. The
    /// result is raw resource-index/destination data; it intentionally does not
    /// assign names such as frame, border, or portrait to any cell.
    ///
    /// Go：`montage.go:247`，`func (m Montage) PlanDialogueFrameGrid() ([]DialogueFramePlacement, error)`
    pub fn plan_dialogue_frame_grid(&self) -> Result<Vec<DialogueFramePlacement>> {
        let d = &self.party_cycle.dialogue_frame_layout;
        if self.status != "mapped_first_party_cycle_fail_closed"
            || d.stride != 320
            || d.arg8 != 5
            || d.arg_c != 7
            || d.arg10 != 5
            || d.arg14 != 5
        {
            return Err(campaign_err!("ending: unavailable dialogue frame grid"));
        }
        let raw = fdother::plan_native_dialogue_frame_grid(d.stride, d.arg8, d.arg_c, d.arg10, d.arg14)?;
        Ok(raw
            .iter()
            .map(|placement| DialogueFramePlacement {
                resource_index: placement.resource_index,
                destination_byte: placement.destination_byte,
            })
            .collect())
    }
}

/// FigureFadePass is one fully evidenced 0x29164 non-mirrored presentation.
/// SourceOffset is a byte offset into the 640-stride work surface. The caller
/// owns the un-recovered restore/copy sequencing; this plan never substitutes
/// an RGBA or generic battle fade.
///
/// Go：`montage.go:270`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct FigureFadePass {
    pub stage: i64,
    pub source_offset: i64,
    pub palette_delta: i64,
}

impl Montage {
    /// Go：`montage.go:274`，`func (m Montage) PlanFigureFade(unitSide int) ([]FigureFadePass, error)`
    pub fn plan_figure_fade(&self, unit_side: i64) -> Result<Vec<FigureFadePass>> {
        let s = &self.party_cycle.figure_fade;
        // 0x2918f tests unit[+6] for zero.  The JSON contract retains
        // required_unit_side=1 as its first observed representative, but it is not
        // an equality predicate: every nonzero raw value takes this branch.
        if self.status != "mapped_first_party_cycle_fail_closed"
            || s.required_unit_side != 1
            || unit_side == 0
            || s.stage_start != 8
            || s.stage_end != 0
            || s.stage_shift_bytes != 10
            || s.palette_formula != "stage*6"
        {
            return Err(campaign_err!("ending: unavailable or mirrored figure fade"));
        }
        Ok((s.stage_end..=s.stage_start)
            .rev()
            .map(|stage| FigureFadePass { stage, source_offset: stage * s.stage_shift_bytes, palette_delta: stage * 6 })
            .collect())
    }
}

/// MirrorFigureFadePass is the exact scheduling plan for 0x29164's
/// unit[+6]==0 branch. It carries no pixels and therefore cannot be used as a
/// renderer substitute; it only makes the recovered source offsets and the
/// arg4-gated draws executable by a future indexed adapter.
///
/// Go：`montage.go:293`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct MirrorFigureFadePass {
    pub stage: i64,
    pub primary_source_offset: i64,
    pub palette_delta: i64,
    pub draw_secondary: bool,
    pub draw_platform: bool,
}

impl Montage {
    /// Go：`montage.go:301`，`func (m Montage) PlanMirrorFigureFade(unitSide, sideFlag int) ([]MirrorFigureFadePass, error)`
    pub fn plan_mirror_figure_fade(&self, unit_side: i64, side_flag: i64) -> Result<Vec<MirrorFigureFadePass>> {
        let s = &self.party_cycle.mirror_branch;
        if self.status != "mapped_first_party_cycle_fail_closed"
            || unit_side != s.required_unit_side
            || s.stage_start != 8
            || s.stage_end != 0
            || s.stage_shift_bytes != 10
            || s.palette_delta_formula != "stage*6"
            || s.side_flag_argument != "arg4"
            || side_flag < 0
        {
            return Err(campaign_err!("ending: unavailable mirror figure fade"));
        }
        Ok((s.stage_end..=s.stage_start)
            .rev()
            .map(|stage| MirrorFigureFadePass {
                stage,
                primary_source_offset: 0x140 - stage * s.stage_shift_bytes,
                palette_delta: stage * 6,
                draw_secondary: side_flag == 0,
                draw_platform: side_flag == 0,
            })
            .collect())
    }
}

/// PartyCyclePlan is the exact recoverable selection portion of the native
/// loop. It iterates loop indexes descending, but the native first two unit
/// slots are deliberately swapped (i=0→slot1, i=1→slot0). It uses the visual
/// group at unit+7 to derive these two FIGANI resource ids.
/// It intentionally contains no UI or renderer approximation.
///
/// Go：`montage.go:323`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct PartyCyclePlan {
    pub loop_index: i64,
    pub unit_slot: i64,
    pub visual_group: i64,
    pub primary_figani: i64,
    pub secondary_figani: i64,
    pub frames: i64,
    pub frame_delay_ticks: i64,
}

impl Montage {
    /// Go：`montage.go:333`，`func (m Montage) PlanPartyCycle(groups []byte) ([]PartyCyclePlan, error)`
    pub fn plan_party_cycle(&self, groups: &[u8]) -> Result<Vec<PartyCyclePlan>> {
        if self.status != "mapped_first_party_cycle_fail_closed" || groups.len() < 2 {
            return Err(campaign_err!("ending: unavailable montage party cycle"));
        }
        let mut plans = Vec::with_capacity(groups.len());
        for i in (0..groups.len()).rev() {
            let slot = match i {
                0 => 1,
                1 => 0,
                _ => i,
            };
            let group = i64::from(groups[slot]);
            plans.push(PartyCyclePlan {
                loop_index: i as i64,
                unit_slot: slot as i64,
                visual_group: group,
                primary_figani: group * 3 + 1,
                secondary_figani: group * 3,
                frames: self.party_cycle.initial_frames,
                frame_delay_ticks: self.party_cycle.frame_delay_ticks,
            });
        }
        Ok(plans)
    }
}

#[cfg(test)]
pub(crate) mod montage_test;
