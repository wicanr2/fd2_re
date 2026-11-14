//! Go 來源：`remake/internal/ending/montage.go`（349 行）。

use super::*;

/// Montage is the evidence-only transcription of the first 0x2c548 party
/// cycle. It records native resource/slot selection without pretending that
/// the later portrait, input, and battle-renderer branches are playable.
///
/// Go：`montage.go:14`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Montage {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"native_handler"`
    pub native_handler: String,
    /// Go 標籤：`json:"status"`
    pub status: String,
    /// Go 標籤：`json:"allocations"`
    pub allocations: Vec<MontageBuffer>,
    /// Go 標籤：`json:"resources"`
    pub resources: Vec<MontageAsset>,
    /// Go 標籤：`json:"party_cycle"`
    pub party_cycle: PartyCycleSpec,
    /// Go 標籤：`json:"gate"`
    pub gate: MontageGate,
}

/// Go：`montage.go:24`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MontageBuffer {
    /// Go 標籤：`json:"bytes"`
    pub bytes: i64,
    /// Go 標籤：`json:"purpose"`
    pub purpose: String,
    /// Go 標籤：`json:"source"`
    pub source: String,
}

/// Go：`montage.go:30`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
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
#[derive(Debug, Clone, Default, PartialEq, Eq)]
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
#[derive(Debug, Clone, Default, PartialEq, Eq)]
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
    pub palette_formula: String,
    /// Go 標籤：`json:"restore_buffer_bytes"`
    pub restore_bytes: i64,
    /// Go 標籤：`json:"restore_copy"`
    pub restore_copy: String,
    /// Go 標籤：`json:"post_figure_snapshot"`
    pub post_figure_snapshot: String,
}

/// Go：`montage.go:75`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
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
#[derive(Debug, Clone, Default, PartialEq, Eq)]
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
#[derive(Debug, Clone, Default, PartialEq, Eq)]
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
    pub fields: Vec<MontageTextField>,
    /// Go 標籤：`json:"glyph_style"`
    pub glyph_style: MontageGlyphStyle,
    /// Go 標籤：`json:"cycle"`
    pub cycle: PortraitCycleSpec,
    /// Go 標籤：`json:"input"`
    pub input: MontageInput,
}

/// Go：`montage.go:114`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
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
#[derive(Debug, Clone, Default, PartialEq, Eq)]
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
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
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
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MontageInput {
    /// Go 標籤：`json:"poll"`
    pub poll: String,
    /// Go 標籤：`json:"skip_action"`
    pub skip_action: String,
    /// Go 標籤：`json:"source"`
    pub source: String,
}

/// Go：`montage.go:168`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
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
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MontageGate {
    /// Go 標籤：`json:"source"`
    pub source: String,
    /// Go 標籤：`json:"reason"`
    pub reason: String,
}

/// Go：`montage.go:191`，`func LoadMontage(path string) (*Montage, error)`
pub fn load_montage(path: &str) -> Result<Montage> {
    todo!("ending/montage.go:LoadMontage")
}

impl Montage {
    /// PlanPortraitText materializes the five native 0x2c548 text calls without
    /// interpreting their glyph/control-code renderer. edi is the outer party
    /// portrait-loop counter observed at 0x2c8f7; the special 0x2d index is chosen
    /// only once edi reaches 0xdc.
    ///
    /// Go：`montage.go:225`，`func (m Montage) PlanPortraitText(unit []byte, edi int) (PortraitTextPlan, error)`
    pub fn plan_portrait_text(&self, unit: &[u8], edi: i64) -> Result<PortraitTextPlan> {
        todo!("ending/montage.go:Montage.PlanPortraitText")
    }

    /// PlanDialogueFrameGrid transcribes every sub_1685c call made by 0x168b6 for the
    /// recovered (stride=320, arg8=5, argC=7, arg10=5, arg14=5) invocation. The
    /// result is raw resource-index/destination data; it intentionally does not
    /// assign names such as frame, border, or portrait to any cell.
    ///
    /// Go：`montage.go:247`，`func (m Montage) PlanDialogueFrameGrid() ([]DialogueFramePlacement, error)`
    pub fn plan_dialogue_frame_grid(&self) -> Result<Vec<DialogueFramePlacement>> {
        todo!("ending/montage.go:Montage.PlanDialogueFrameGrid")
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
        todo!("ending/montage.go:Montage.PlanFigureFade")
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
        todo!("ending/montage.go:Montage.PlanMirrorFigureFade")
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
        todo!("ending/montage.go:Montage.PlanPartyCycle")
    }
}
