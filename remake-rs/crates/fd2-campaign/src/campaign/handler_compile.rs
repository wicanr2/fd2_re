//! Go 來源：`remake/internal/campaign/handler_compile.go`（1676 行）。

use super::*;

/// HandlerPoint is a remake camera coordinate supplied by a campaign-specific
/// mapping.  Handler scripts deliberately retain the original grid coordinate;
/// there is no assumed global grid-to-pixel formula.
///
/// Go：`handler_compile.go:11`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct HandlerPoint {
    /// Go 標籤：`json:"x"`
    pub x: i64,
    /// Go 標籤：`json:"y"`
    pub y: i64,
    /// Go 標籤：`json:"frames,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub frames: i64,
    /// Go 標籤：`json:"tile_step,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub tile_step: bool,
}

/// HandlerDialog identifies the authored remake line(s) corresponding to one
/// original FDTXT table/index lookup.  The mapping is explicit because a single
/// FDTXT string may be split into several remake lines.
///
/// Go：`handler_compile.go:21`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct HandlerDialog {
    /// Go 標籤：`json:"line"`
    pub line: i64,
    /// Go 標籤：`json:"count,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub count: i64,
    /// Go 標籤：`json:"upper,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub upper: Option<bool>,
    /// Script/Scene record the authored text context selected by the preceding
    /// loadch or camera transition.  They are metadata until a handler is run
    /// through the scene-loading adapter; preserving them prevents line index 0
    /// from being ambiguous across different FDTXT resources.
    /// Go 標籤：`json:"script,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub script: String,
    /// Go 標籤：`json:"scene,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub scene: String,
    /// Go 標籤：`json:"scene_index,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub scene_index: Option<i64>,
    /// Lines expands one original FDTXT call into individually authored remake
    /// lines.  This is required when one original string contains alternating
    /// speakers (and therefore different dialogue-box positions).
    /// Go 標籤：`json:"lines,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub lines: Vec<HandlerDialogLine>,
    /// Segments preserves one native FDTXT lookup whose authored lines cross
    /// scene boundaries. The compiler lowers these in order to ordinary dialog
    /// beats; no text or scene boundary is inferred at runtime.
    /// Go 標籤：`json:"segments,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub segments: Vec<HandlerDialogSegment>,
}

/// HandlerDialogLine is one runtime dialog beat within a HandlerDialog group.
///
/// Go：`handler_compile.go:43`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct HandlerDialogLine {
    /// Go 標籤：`json:"line"`
    pub line: i64,
    /// Go 標籤：`json:"count,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub count: i64,
    /// Go 標籤：`json:"upper,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub upper: Option<bool>,
    /// Go 標籤：`json:"native_dialogue,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_dialogue: Option<NativeDialogueLayout>,
}

/// NativeDialogueLayout 是已證實 FDTXT 呼叫中單句對白的可編輯投影。
/// Pages 保留 FFFD，頁內列保留 FFFE；執行期不得從現代字型的 Text 欄位反推。
///
/// Go：`handler_compile.go:52`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeDialogueLayout {
    /// Go 標籤：`json:"source_dat"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub source_dat: String,
    /// Go 標籤：`json:"string_index"`
    pub string_index: i64,
    /// Go 標籤：`json:"utterance"`
    pub utterance: i64,
    /// Go 標籤：`json:"control"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub control: String,
    /// Go 標籤：`json:"operand"`
    pub operand: i64,
    /// Go 標籤：`json:"pages"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub pages: Vec<Vec<String>>,
    /// GlyphPages 與Pages同形；每個token對應一個原始FDTXT ordinary word。
    /// 單一原版glyph可能映成多個Unicode rune，故不可由Pages反推token數。
    /// Go 標籤：`json:"glyph_pages,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub glyph_pages: Vec<Vec<Vec<String>>>,
    /// GlyphIDs 與token同形，保存原始FDTXT ordinary word（字模索引）。glyph_map 有一字
    /// 多模（584 與 347 都解成「．」，前者是兩點、後者是一點），token 查不回原字模；
    /// 只有這種句子才輸出，其餘由token查正規字模。
    /// Go 標籤：`json:"glyph_ids,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub glyph_ids: Vec<Vec<Vec<i64>>>,
}

// nativeDialogueLineGlyphLimit 保存固定原版FDTXT已觀察到的每種speaker control
// 列寬。它是原始資料的接受上限，不是任意新文字的自動排版規則。

/// nativeGridStepFrames 是原版走行步進家族（`0x12EAA`／`0x1300D`／`0x13185`／
/// `0x13315`）一格的呈現幀數。計數器由 1 起、迴圈條件 `cmp [esp], 7 ; jge`、
/// 遞增在迴圈尾端，所以主體跑六次，`unit+4` 也寫 1..6。
pub(crate) const NATIVE_GRID_STEP_FRAMES: i64 = 6;

/// Go：`handler_compile.go:76`，`func nativeDialogueLineGlyphLimit(control string) (int, bool)`
pub(crate) fn native_dialogue_line_glyph_limit_inner(control: &str) -> (i64, bool) {
    match control {
        "FFEC" => (13, true),
        "FFED" => (15, true),
        "FFEE" | "FFEF" => (14, true),
        _ => (0, false),
    }
}

/// NativeDialogueLineGlyphLimit 提供已證實的 control 列寬，避免多語索引 renderer
/// 複製另一份 FDTXT 常數。
///
/// Go：`handler_compile.go:91`，`func NativeDialogueLineGlyphLimit(control string) (int, bool)`
pub fn native_dialogue_line_glyph_limit(control: &str) -> (i64, bool) {
    native_dialogue_line_glyph_limit_inner(control)
}

impl NativeDialogueLayout {
    /// Go：`handler_compile.go:95`，`func (layout *NativeDialogueLayout) Validate() error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn validate(&self) -> Result<()> {
        let layout = self;
        if layout.source_dat.is_empty()
            || layout.string_index < 0
            || layout.utterance < 0
            || layout.operand < 0
            || layout.operand > 0xffff
        {
            return Err(campaign_err!("native dialogue layout lacks source provenance"));
        }
        let (line_glyph_limit, ok) = native_dialogue_line_glyph_limit_inner(&layout.control);
        if !ok {
            return Err(campaign_err!(
                "native dialogue layout has unsupported control {}",
                battle::go_quote(&layout.control)
            ));
        }
        if layout.pages.is_empty() {
            return Err(campaign_err!("native dialogue layout has no pages"));
        }
        for (page, rows) in layout.pages.iter().enumerate() {
            // 0x16319..0x16365 的 line==3 分支會呼叫 sub_16E24 並把
            // 行號減一。三列是原版可視窗口，不是 raw page 的邏輯列上限；
            // 保留後續 FFFE 列，renderer 再以同一三列窗口消費。
            if rows.is_empty() || rows.len() > 64 {
                return Err(campaign_err!("native dialogue layout page {page} has {} rows", rows.len()));
            }
            if !layout.glyph_pages.is_empty()
                && (page >= layout.glyph_pages.len() || layout.glyph_pages[page].len() != rows.len())
            {
                return Err(campaign_err!("native dialogue layout page {page} glyph rows do not match text rows"));
            }
            for (row, text) in rows.iter().enumerate() {
                let tokens = layout.glyph_tokens(page as i64, row as i64, text)?;
                let count = tokens.len() as i64;
                if count == 0 || count > line_glyph_limit {
                    return Err(campaign_err!(
                        "native dialogue layout page {page} row {row} has {count} glyphs, limit {line_glyph_limit} for {}",
                        layout.control
                    ));
                }
            }
        }
        if !layout.glyph_ids.is_empty() {
            if layout.glyph_ids.len() != layout.pages.len() {
                return Err(campaign_err!(
                    "native dialogue layout has {} glyph id pages for {} text pages",
                    layout.glyph_ids.len(),
                    layout.pages.len()
                ));
            }
            for (page, rows) in layout.pages.iter().enumerate() {
                if layout.glyph_ids[page].len() != rows.len() {
                    return Err(campaign_err!(
                        "native dialogue layout page {page} glyph id rows do not match text rows"
                    ));
                }
                for (row, text) in rows.iter().enumerate() {
                    let tokens = layout.glyph_tokens(page as i64, row as i64, text)?;
                    if layout.glyph_ids[page][row].len() != tokens.len() {
                        return Err(campaign_err!(
                            "native dialogue layout page {page} row {row} has {} glyph ids for {} tokens",
                            layout.glyph_ids[page][row].len(),
                            tokens.len()
                        ));
                    }
                    for &id in &layout.glyph_ids[page][row] {
                        if !(0..0xff00).contains(&id) {
                            return Err(campaign_err!(
                                "native dialogue layout page {page} row {row} glyph id {id} is invalid"
                            ));
                        }
                    }
                }
            }
        }
        if !layout.glyph_pages.is_empty() && layout.glyph_pages.len() != layout.pages.len() {
            return Err(campaign_err!(
                "native dialogue layout has {} glyph pages for {} text pages",
                layout.glyph_pages.len(),
                layout.pages.len()
            ));
        }
        Ok(())
    }

    /// Go：`handler_compile.go:157`，`func (layout *NativeDialogueLayout) glyphTokens(page, row int, text string) ([]string, error)`
    pub(crate) fn glyph_tokens(&self, page: i64, row: i64, text: &str) -> Result<Vec<String>> {
        if self.glyph_pages.is_empty() {
            return Ok(text.chars().map(String::from).collect());
        }
        if page < 0
            || page as u64 >= self.glyph_pages.len() as u64
            || row < 0
            || row as u64 >= self.glyph_pages[page as usize].len() as u64
        {
            return Err(campaign_err!("native dialogue layout page {page} row {row} lacks raw glyph tokens"));
        }
        let tokens = &self.glyph_pages[page as usize][row as usize];
        if tokens.concat() != text {
            return Err(campaign_err!("native dialogue layout page {page} row {row} glyph tokens disagree with text"));
        }
        for (token, value) in tokens.iter().enumerate() {
            if value.is_empty() {
                return Err(campaign_err!("native dialogue layout page {page} row {row} token {token} is empty"));
            }
        }
        Ok(tokens.clone())
    }
}

/// Go：`handler_compile.go:181`，`func cloneNativeDialogueLayout(layout *NativeDialogueLayout) *NativeDialogueLayout`
pub(crate) fn clone_native_dialogue_layout(layout: Option<&NativeDialogueLayout>) -> Option<NativeDialogueLayout> {
    // Go 逐層複製切片；Rust 的 clone 本來就是深複製，結果相同。
    layout.cloned()
}

/// HandlerDialogSegment is one contiguous scene range within a native lookup.
///
/// Go：`handler_compile.go:212`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct HandlerDialogSegment {
    /// Go 標籤：`json:"script,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub script: String,
    /// Go 標籤：`json:"scene,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub scene: String,
    /// Go 標籤：`json:"scene_index,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub scene_index: Option<i64>,
    /// Go 標籤：`json:"lines"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub lines: Vec<HandlerDialogLine>,
    /// Go 標籤：`json:"upper,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub upper: Option<bool>,
}

/// HandlerBindings holds only evidence-backed, campaign-specific bridges from
/// EXE-level identifiers to runtime data.  Nil or failed lookups are reported
/// as issues rather than guessed at.
///
/// Go：`handler_compile.go:223`
#[derive(Debug, Clone, Default)]
pub struct HandlerBindings {
    /// Every resolver receives the full input beat, including source.addr.  This
    /// permits explicit per-call-site bindings when an index is reused after a
    /// later loadch segment.
    pub pan: Option<Callback<dyn Fn(HandlerBeat) -> (HandlerPoint, bool) + Send + Sync>>,
    pub dialog: Option<Callback<dyn Fn(HandlerBeat) -> (HandlerDialog, bool) + Send + Sync>>,
    pub acting: Option<Callback<dyn Fn(HandlerBeat) -> (Vec<ActingFrame>, bool) + Send + Sync>>,
    pub load_ch: Option<Callback<dyn Fn(HandlerBeat) -> (LoadCHState, bool) + Send + Sync>>,
    pub layout: Option<Callback<dyn Fn(HandlerBeat) -> (HandlerLayout, bool) + Send + Sync>>,
    pub transition: Option<Callback<dyn Fn(HandlerBeat) -> (HandlerIndexedTransition, bool) + Send + Sync>>,
    pub resource: Option<Callback<dyn Fn(HandlerBeat) -> (HandlerResource, bool) + Send + Sync>>,
    /// RuntimeContext is present for a handler entered with an existing canonical
    /// unit array (not through LOADCH), such as a post-battle handler. It makes
    /// slot validation and SPAWN cardinality explicit instead of guessing from a
    /// chapter number.
    pub runtime_context: Option<HandlerRuntimeContext>,
}

/// Go：`handler_compile.go:241`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct HandlerRuntimeContext {
    /// Go 標籤：`json:"slot_count"`
    pub slot_count: i64,
    /// Go 標籤：`json:"slot_counts,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub slot_counts: Vec<i64>,
    /// Go 標籤：`json:"spawn_groups,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "BTreeMap::is_empty")]
    pub spawn_groups: BTreeMap<i64, i64>,
    /// Go 標籤：`json:"story_viewport,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub story_viewport: bool,
}

impl HandlerRuntimeContext {
    /// MinimumSlotCount is the smallest allowed *materialized* runtime shape in the
    /// authored binding contract. It is not the native allocation capacity: the
    /// original battle buffer is larger and [0x53beb] is the append count. SlotCount
    /// preserves the original exact-context form; SlotCounts models optional native
    /// reinforcement groups (for example 15 or 27 slots at a post-battle entry).
    ///
    /// Go：`handler_compile.go:253`，`func (context *HandlerRuntimeContext) MinimumSlotCount() int`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn minimum_slot_count(&self) -> i64 {
        if self.slot_count > 0 {
            return self.slot_count;
        }
        let mut minimum = 0;
        for &count in &self.slot_counts {
            if count > 0 && (minimum == 0 || count < minimum) {
                minimum = count;
            }
        }
        minimum
    }

    /// MaximumSlotCount is an upper bound for static compilation of branch-local
    /// operations. Runtime playback still requires AcceptsSlotCount to match one
    /// exact frontier; using the maximum here only lets a proven spawn in one arm
    /// validate the optional slot that arm creates.
    ///
    /// Go：`handler_compile.go:273`，`func (context *HandlerRuntimeContext) MaximumSlotCount() int`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn maximum_slot_count(&self) -> i64 {
        if self.slot_count > 0 {
            return self.slot_count;
        }
        let mut maximum = 0;
        for &count in &self.slot_counts {
            if count > maximum {
                maximum = count;
            }
        }
        maximum
    }

    /// Go：`handler_compile.go:289`，`func (context *HandlerRuntimeContext) AcceptsSlotCount(count int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn accepts_slot_count(&self, count: i64) -> bool {
        if self.slot_count > 0 {
            return count == self.slot_count;
        }
        self.slot_counts.contains(&count)
    }
}

/// HandlerCompileIssue identifies a source operation that was intentionally
/// not lowered to a runtime Beat.  The caller can surface these in an editor or
/// block playback, but no original operation is silently ignored.
///
/// Go：`handler_compile.go:307`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct HandlerCompileIssue {
    pub beat: i64,
    pub op: String,
    pub source: HandlerSource,
    pub reason: String,
}

/// CompileHandlerScript lowers the subset of a HandlerScript whose remake
/// semantics are proven.  It is reusable for a future campaign: campaign data
/// supplies mappings for map geometry, text layout, and acting resources while
/// this compiler owns no FD2 chapter-specific constants.
///
/// Go：`handler_compile.go:318`，`func CompileHandlerScript(script *HandlerScript, bindings HandlerBindings) ([]Beat, []HandlerCompileIssue)`
pub fn compile_handler_script(
    script: Option<&HandlerScript>,
    bindings: &HandlerBindings,
) -> (Vec<Beat>, Vec<HandlerCompileIssue>) {
    let mut active_slot_count = 0;
    let mut minimum_slot_count = 0;
    if let Some(context) = &bindings.runtime_context {
        active_slot_count = context.maximum_slot_count();
        minimum_slot_count = context.minimum_slot_count();
    }
    compile_handler_script_inner(script, bindings, active_slot_count, minimum_slot_count)
}

/// compileHandlerScript carries the proven pre-branch slot frontier into each
/// arm.  Branches still may not change the outer compiler context: after a
/// merge, callers may rely only on slots known before the branch.  The minimum
/// frontier is tracked separately from the maximum so a static operation cannot
/// accidentally require a slot that the binding has not proven materialized in
/// every optional runtime shape. This is a candidate fail-closed contract, not
/// a claim that the native 96-record allocation is physically absent.
///
/// Go：`handler_compile.go:335`，`func compileHandlerScript(script *HandlerScript, bindings HandlerBindings, activeSlotCount, minimumSlotCount int) ([]Beat, []HandlerCompileIssue)`
pub(crate) fn compile_handler_script_inner(
    script: Option<&HandlerScript>,
    bindings: &HandlerBindings,
    active_slot_count: i64,
    minimum_slot_count: i64,
) -> (Vec<Beat>, Vec<HandlerCompileIssue>) {
    let Some(script) = script else {
        return (Vec::new(), vec![HandlerCompileIssue { reason: "nil handler script".into(), ..Default::default() }]);
    };
    let mut active_slot_count = active_slot_count;
    let mut minimum_slot_count = minimum_slot_count;
    let mut beats: Vec<Beat> = Vec::with_capacity(script.beats.len());
    let mut issues: Vec<HandlerCompileIssue> = Vec::new();
    let mut consumed_loop_delay_beat: i64 = -1;
    let runtime =
        |input: &HandlerBeat, op: &str| Beat { op: op.into(), source: input.source.addr.clone(), ..Default::default() };
    for (i, input) in script.beats.iter().enumerate() {
        macro_rules! issue {
            ($reason:expr) => {
                issues.push(HandlerCompileIssue {
                    beat: i as i64,
                    op: input.op.clone(),
                    source: input.source.clone(),
                    reason: ($reason).into(),
                })
            };
        }
        if i as i64 == consumed_loop_delay_beat {
            consumed_loop_delay_beat = -1;
            continue;
        }
        match input.op.as_str() {
            "if" => {
                let Some(input_condition) = &input.condition else {
                    issue!("if requires a proven condition");
                    continue;
                };
                let mut condition = BeatCondition { op: input_condition.op.clone(), ..Default::default() };
                let mut then_slot_count = active_slot_count;
                let mut then_minimum_slot_count = minimum_slot_count;
                match input_condition.op.as_str() {
                    "any_unit_inactive" => {
                        if input_condition.unit_slots.is_empty() {
                            issue!("any_unit_inactive requires at least one runtime unit slot");
                            continue;
                        }
                        let mut seen = BTreeSet::new();
                        let mut valid_slots = true;
                        for &slot in &input_condition.unit_slots {
                            if slot < 0 || seen.contains(&slot) || (active_slot_count > 0 && slot >= active_slot_count)
                            {
                                valid_slots = false;
                                break;
                            }
                            seen.insert(slot);
                        }
                        if !valid_slots {
                            issue!(
                                "any_unit_inactive slots must be unique non-negative integers within the active runtime context"
                            );
                            continue;
                        }
                        condition.unit_slots = input_condition.unit_slots.clone();
                    }
                    "native_inventory_item_present" => {
                        let Some(item_id) = input_condition.native_inventory_item_id.filter(|v| (0..=0xff).contains(v))
                        else {
                            issue!("native_inventory_item_present requires an unsigned raw item byte");
                            continue;
                        };
                        condition.native_inventory_item_id = Some(item_id);
                    }
                    "native_persistent_identity_present" => {
                        let Some(identity) =
                            input_condition.native_persistent_identity.filter(|v| (0..=0xff).contains(v))
                        else {
                            issue!("native_persistent_identity_present requires an unsigned raw record+0x08 byte");
                            continue;
                        };
                        condition.native_persistent_identity = Some(identity);
                    }
                    "native_inactive_count_gt" => {
                        if input_condition.unit_slots.is_empty() || !input_condition.threshold.is_some_and(|t| t >= 0) {
                            issue!("native_inactive_count_gt requires unit_slots and a non-negative threshold");
                            continue;
                        }
                        let mut seen = BTreeSet::new();
                        let mut valid_slots = true;
                        for &slot in &input_condition.unit_slots {
                            if slot < 0 || seen.contains(&slot) || (active_slot_count > 0 && slot >= active_slot_count)
                            {
                                valid_slots = false;
                                break;
                            }
                            seen.insert(slot);
                        }
                        if !valid_slots {
                            issue!(
                                "native_inactive_count_gt slots must be unique non-negative integers within the active runtime context"
                            );
                            continue;
                        }
                        condition.unit_slots = input_condition.unit_slots.clone();
                        condition.threshold = input_condition.threshold;
                    }
                    "native_round_gt" => {
                        let Some(threshold) = input_condition.native_round.filter(|v| *v >= 0) else {
                            issue!("native_round_gt requires a non-negative raw round threshold");
                            continue;
                        };
                        condition.native_round = Some(threshold);
                    }
                    "native_round_lt" => {
                        let Some(threshold) = input_condition.native_round.filter(|v| *v >= 0) else {
                            issue!("native_round_lt requires a non-negative raw round threshold");
                            continue;
                        };
                        condition.native_round = Some(threshold);
                    }
                    "native_record_word_gte" => {
                        let (Some(slot), Some(0x42), Some(value)) = (
                            input_condition.unit_slot.filter(|v| *v >= 0),
                            input_condition.native_record_word_offset,
                            input_condition.native_record_word_value.filter(|v| (0..=0xffff).contains(v)),
                        ) else {
                            issue!("native_record_word_gte requires unit_slot, raw offset 0x42, and u16 value");
                            continue;
                        };
                        if active_slot_count > 0 && slot >= active_slot_count {
                            issue!("native_record_word_gte unit_slot exceeds active runtime context");
                            continue;
                        }
                        condition.unit_slot = Some(slot);
                        condition.native_record_word_offset = Some(0x42);
                        condition.native_record_word_value = Some(value);
                    }
                    "native_any_of" => {
                        if input_condition.any.is_empty() {
                            issue!("native_any_of requires at least one proven raw predicate");
                            continue;
                        }
                        for child in &input_condition.any {
                            let mut compiled = BeatCondition { op: child.op.clone(), ..Default::default() };
                            match child.op.as_str() {
                                "native_round_gt" => {
                                    let Some(v) = child.native_round.filter(|v| *v >= 0) else {
                                        issue!("native_any_of contains invalid native_round_gt");
                                        continue;
                                    };
                                    compiled.native_round = Some(v);
                                }
                                "native_inactive_count_gt" => {
                                    if child.unit_slots.is_empty() || !child.threshold.is_some_and(|t| t >= 0) {
                                        issue!("native_any_of contains invalid native_inactive_count_gt");
                                        continue;
                                    }
                                    // Go `handler_compile.go:467` 這個迴圈不 break，整列看完才判定。
                                    let mut seen = BTreeSet::new();
                                    let mut valid = true;
                                    for &slot in &child.unit_slots {
                                        if slot < 0
                                            || seen.contains(&slot)
                                            || (active_slot_count > 0 && slot >= active_slot_count)
                                        {
                                            valid = false;
                                        }
                                        seen.insert(slot);
                                    }
                                    if !valid {
                                        issue!("native_any_of contains invalid inactive-count slots");
                                        continue;
                                    }
                                    compiled.unit_slots = child.unit_slots.clone();
                                    compiled.threshold = child.threshold;
                                }
                                _ => {
                                    issue!(format!(
                                        "native_any_of contains unsupported predicate {}",
                                        battle::go_quote(&child.op)
                                    ));
                                    continue;
                                }
                            }
                            condition.any.push(compiled);
                        }
                        if condition.any.len() != input_condition.any.len() {
                            continue;
                        }
                    }
                    "roster_has" => {
                        let Some(char_id) = input_condition.char_id.filter(|v| joinable_character_id(*v)) else {
                            issue!("roster_has requires an original 0..31 permanent-player char_id");
                            continue;
                        };
                        condition.char_id = Some(char_id);
                    }
                    "native_event_state_nonzero" => {
                        let Some(index) = input_condition.event_state_index.filter(|v| (0..0x20).contains(v)) else {
                            issue!("native_event_state_nonzero requires a 0..31 event_state_index");
                            continue;
                        };
                        condition.event_state_index = Some(index);
                    }
                    "native_event_state_eq" => {
                        let index = input_condition.event_state_index.filter(|v| (0..0x20).contains(v));
                        let value = input_condition.event_state_value.filter(|v| (0..=0xff).contains(v));
                        let required = input_condition.required_slot_count.filter(|v| *v > 0);
                        let (Some(index), Some(value), Some(required)) = (index, value, required) else {
                            issue!(
                                "native_event_state_eq requires a raw index/value and a runtime frontier listed by the binding"
                            );
                            continue;
                        };
                        if (active_slot_count > 0 && required < active_slot_count)
                            || !bindings.runtime_context.as_ref().is_some_and(|c| c.accepts_slot_count(required))
                        {
                            issue!(
                                "native_event_state_eq requires a raw index/value and a runtime frontier listed by the binding"
                            );
                            continue;
                        }
                        condition.event_state_index = Some(index);
                        condition.event_state_value = Some(value);
                        condition.required_slot_count = Some(required);
                        then_slot_count = required;
                        then_minimum_slot_count = required;
                    }
                    _ => {
                        issue!("if requires a proven condition");
                        continue;
                    }
                }
                if handler_branch_changes_compile_context(&input.then)
                    || handler_branch_changes_compile_context(&input.r#else)
                {
                    issue!("if arms cannot change loadch or chapter context before a proven merge model exists");
                    continue;
                }
                // A fixed runtime_context (or a proven LOADCH frontier before this
                // branch) is already the active-slot model.  Without it, retain the
                // conservative rejection: acting/slot operations in an arm could
                // otherwise observe a branch-local roster shape that is not merged.
                if active_slot_count == 0
                    && (handler_branch_needs_active_load_ch(&input.then)
                        || handler_branch_needs_active_load_ch(&input.r#else))
                {
                    issue!("if arms cannot use active-slot operations before branch compiler context is modeled");
                    continue;
                }
                let then_script = HandlerScript { beats: input.then.clone(), ..Default::default() };
                let else_script = HandlerScript { beats: input.r#else.clone(), ..Default::default() };
                let (then_beats, then_issues) = compile_handler_script_inner(
                    Some(&then_script),
                    bindings,
                    then_slot_count,
                    then_minimum_slot_count,
                );
                let (else_beats, else_issues) =
                    compile_handler_script_inner(Some(&else_script), bindings, active_slot_count, minimum_slot_count);
                if !then_issues.is_empty() || !else_issues.is_empty() {
                    for mut branch_issue in then_issues {
                        branch_issue.reason = format!("if then: {}", branch_issue.reason);
                        issues.push(branch_issue);
                    }
                    for mut branch_issue in else_issues {
                        branch_issue.reason = format!("if else: {}", branch_issue.reason);
                        issues.push(branch_issue);
                    }
                    continue;
                }
                let mut beat = runtime(input, "if");
                beat.condition = Some(condition);
                beat.then = then_beats;
                beat.r#else = else_beats;
                beats.push(beat);
            }
            "loadch" => {
                let Some(load_ch) = &bindings.load_ch else {
                    issue!("loadch requires an explicit map, roster, and story-context mapping");
                    continue;
                };
                let (state, ok) = (load_ch.0)(input.clone());
                if !ok {
                    issue!("no complete remake state mapping for original loadch");
                    continue;
                }
                if state.chapter < 0
                    || state.map.is_empty()
                    || state.roster.is_empty()
                    || state.slot_count <= 0
                    || state.script.is_empty()
                {
                    issue!("loadch mapping must declare non-negative chapter plus map, roster, slot_count, and script");
                    continue;
                }
                if let Some(chapter) = input.chapter
                    && chapter != state.chapter
                {
                    issue!(format!("loadch chapter {chapter} disagrees with binding chapter {}", state.chapter));
                    continue;
                }
                let mut beat = runtime(input, "loadch");
                let slot_count = state.slot_count;
                beat.load_ch = Some(state);
                beats.push(beat);
                active_slot_count = slot_count;
                minimum_slot_count = slot_count;
            }
            "delay" => {
                let Some(ms) = input.ms else {
                    issue!("delay lacks an immediate millisecond value");
                    continue;
                };
                let mut beat = runtime(input, "delay");
                beat.ms = ms;
                beats.push(beat);
            }
            "bgm" => {
                let Some(track) = input.track else {
                    issue!("bgm lacks immediate track");
                    continue;
                };
                if track == -1 {
                    beats.push(runtime(input, "bgm_stop"));
                } else if track >= 0 {
                    let mut beat = runtime(input, "bgm");
                    beat.track = format!("FDMUS_{track:03}");
                    beats.push(beat);
                } else {
                    issue!(format!("unsupported negative BGM track {track}"));
                }
            }
            "pan" => {
                let (Some(_), Some(_), Some(pan)) = (input.grid_x, input.grid_y, &bindings.pan) else {
                    issue!("pan requires an explicit grid-to-camera mapping");
                    continue;
                };
                let (p, ok) = (pan.0)(input.clone());
                if !ok {
                    issue!("no camera mapping for original grid coordinate");
                    continue;
                }
                let mut beat = runtime(input, "pan");
                beat.x = p.x;
                beat.y = p.y;
                beat.frames = p.frames;
                beat.tile_step = p.tile_step;
                beats.push(beat);
            }
            "dialog" => {
                let Some(dialog) = &bindings.dialog else {
                    issue!("dialog requires an explicit FDTXT-to-remake-line mapping");
                    continue;
                };
                let (d, ok) = (dialog.0)(input.clone());
                if !ok {
                    issue!("no remake line mapping for original FDTXT lookup");
                    continue;
                }
                if d.segments.is_empty() && d.lines.is_empty() {
                    let mut beat = runtime(input, "dialog");
                    beat.line = d.line;
                    beat.count = d.count;
                    beat.upper = d.upper;
                    beat.script = d.script.clone();
                    beat.scene = d.scene.clone();
                    beat.scene_index = d.scene_index;
                    beats.push(beat);
                    continue;
                }
                if d.segments.is_empty() {
                    for line in &d.lines {
                        let mut beat = runtime(input, "dialog");
                        beat.line = line.line;
                        beat.count = line.count;
                        beat.upper = line.upper;
                        if let Err(err) = attach_native_dialogue_layout(&mut beat, line.native_dialogue.as_ref()) {
                            issue!(err.message().to_string());
                            continue;
                        }
                        beat.script = d.script.clone();
                        beat.scene = d.scene.clone();
                        beat.scene_index = d.scene_index;
                        beats.push(beat);
                    }
                    continue;
                }
                for segment in &d.segments {
                    if segment.lines.is_empty() {
                        let mut beat = runtime(input, "dialog");
                        beat.upper = segment.upper;
                        beat.script = segment.script.clone();
                        beat.scene = segment.scene.clone();
                        beat.scene_index = segment.scene_index;
                        beats.push(beat);
                        continue;
                    }
                    for line in &segment.lines {
                        let mut beat = runtime(input, "dialog");
                        beat.line = line.line;
                        beat.count = line.count;
                        beat.upper = line.upper;
                        if line.upper.is_none() {
                            beat.upper = segment.upper;
                        }
                        if let Err(err) = attach_native_dialogue_layout(&mut beat, line.native_dialogue.as_ref()) {
                            issue!(err.message().to_string());
                            continue;
                        }
                        beat.script = segment.script.clone();
                        beat.scene = segment.scene.clone();
                        beat.scene_index = segment.scene_index;
                        beats.push(beat);
                    }
                }
            }
            "act" => {
                let (Some(_), Some(acting)) = (input.acting_id, &bindings.acting) else {
                    issue!("act requires an explicit acting-resource mapping");
                    continue;
                };
                let (frames, ok) = (acting.0)(input.clone());
                if !ok {
                    issue!("acting resource has not been decoded/mapped");
                    continue;
                }
                if active_slot_count > 0 && acting_uses_unavailable_slot(&frames, active_slot_count) {
                    issue!(format!(
                        "acting references roster slot outside active loadch slot_count={active_slot_count}"
                    ));
                    continue;
                }
                let mut beat = runtime(input, "act");
                beat.acting = frames;
                beats.push(beat);
            }
            "scroll_step" => {
                // 0x13185(slot) is one complete grid step upward, including its
                // sub-tile drawing ticks and camera follow. HandlerScript folds the
                // counted loop into Repeat, so one runtime beat retains both the
                // original slot identity and exact number of grid steps.
                //
                // 一格是六張呈現幀，不是七張：`0x1320B` 把計數器設成 1，迴圈條件
                // 在 `0x13274` 是 `cmp [esp], 7 ; jge 離開`，計數器在迴圈尾端
                // `0x13271` 才遞增，所以主體跑 1..6，`unit+4` 也寫 1..6。單位座標、
                // 絕對游標與鏡頭在 `0x132F5` 之後才提交。同一結論另有 dosgolem
                // 收據（每格 `+4` 走 1..6，跨格直接重設為 1），見
                // docs/knowledge-base/99-move-confirm-cursor-20260909.md。
                let (Some(slot), Some(repeat)) = (input.unit_slot.filter(|v| *v >= 0), input.repeat.filter(|v| *v > 0))
                else {
                    issue!("scroll_step requires a non-negative runtime slot and positive repeat count");
                    continue;
                };
                if active_slot_count <= slot {
                    issue!(format!("scroll_step slot {slot} is outside active loadch slot_count={active_slot_count}"));
                    continue;
                }
                let mut beat = runtime(input, "scroll_step");
                beat.slot = Some(slot);
                beat.steps = repeat;
                beat.frames = repeat.wrapping_mul(NATIVE_GRID_STEP_FRAMES);
                beat.follow = true;
                beats.push(beat);
            }
            "spawn" => {
                // The group immediate selects FDFIELD rows, but placement additionally
                // depends on the exact call-site byte [0x53AFA]. Both are required.
                let Some(group) = input.group else {
                    issue!("spawn lacks an original FDFIELD group");
                    continue;
                };
                let Some(gate) = input.raw_placement_gate.filter(|v| (0..=0xff).contains(v)) else {
                    issue!("spawn requires an explicit raw_placement_gate byte");
                    continue;
                };
                if active_slot_count <= 0 {
                    issue!("spawn requires a preceding complete loadch roster");
                    continue;
                }
                let mut expected_count = 0;
                if let Some(context) = &bindings.runtime_context {
                    let size = context.spawn_groups.get(&group).copied();
                    let Some(size) = size.filter(|v| *v > 0) else {
                        issue!("spawn requires an explicit positive runtime-context group cardinality");
                        continue;
                    };
                    active_slot_count = active_slot_count.wrapping_add(size);
                    minimum_slot_count = minimum_slot_count.wrapping_add(size);
                    expected_count = size;
                }
                let mut beat = runtime(input, "spawn");
                beat.group = group;
                beat.count = expected_count;
                beat.raw_placement_gate = Some(gate);
                beats.push(beat);
            }
            "spawn_intro" => {
                // 0x32999(group) calls the same 0x10b4e constructor as SPAWN, then
                // performs 12 indexed presentation passes from FDOTHER #9. The caller's
                // following 0x1366a ACTING is a separate operation. Frames deliberately
                // remains zero: 12 renderer ticks are not an equivalent implementation.
                let Some(group) = input.group else {
                    issue!("spawn_intro lacks an original FDFIELD group");
                    continue;
                };
                let Some(gate) = input.raw_placement_gate.filter(|v| (0..=0xff).contains(v)) else {
                    issue!("spawn_intro requires an explicit raw_placement_gate byte");
                    continue;
                };
                if active_slot_count <= 0 {
                    issue!("spawn_intro requires a preceding complete loadch roster");
                    continue;
                }
                let mut beat = runtime(input, "spawn_intro");
                beat.group = group;
                beat.raw_placement_gate = Some(gate);
                beats.push(beat);
            }
            "deactivate_unit" => {
                // 0x32975(unit_idx) writes unit[idx].flags=1. Constructor and
                // The authored op is a caller-specific raw byte+5 writer; OnField is
                // retained only as the engine projection for old bindings.
                if input.unit_slot.is_none()
                    && input.unit_slot_expr == "ebx"
                    && let Some(hint) = &input.repeat_hint
                {
                    if hint.limit <= 0 {
                        issue!("deactivate_unit repeat_hint limit must be positive");
                        continue;
                    }
                    if active_slot_count > 0 && hint.limit > active_slot_count {
                        issue!(format!(
                            "deactivate_unit repeat limit {} exceeds active loadch slot_count={active_slot_count}",
                            hint.limit
                        ));
                        continue;
                    }
                    for slot in 0..hint.limit {
                        let mut beat = runtime(input, "deactivate_unit");
                        beat.slot = Some(slot);
                        beats.push(beat);
                    }
                    continue;
                }
                let Some(slot) = input.unit_slot.filter(|v| *v >= 0) else {
                    issue!("deactivate_unit lacks a non-negative runtime slot");
                    continue;
                };
                if active_slot_count <= slot {
                    issue!(format!(
                        "deactivate_unit slot {slot} is outside active loadch slot_count={active_slot_count}"
                    ));
                    continue;
                }
                let mut beat = runtime(input, "deactivate_unit");
                beat.slot = Some(slot);
                beats.push(beat);
            }
            "reactivate_nonzero_hp" => {
                // 兩個原生counted loop只復位非零HP。336ED另寫pose2。
                let mut known_loop = input.source.addr == "0x33cea" && input.direction.is_none();
                if input.source.addr == "0x336ed" {
                    known_loop =
                        input.repeat_hint.as_ref().is_some_and(|h| h.limit == 16) && input.direction == Some(2);
                }
                let hint_limit = input.repeat_hint.as_ref().map(|h| h.limit);
                if !known_loop
                    || !input.source.target.is_empty()
                    || input.unit_slot_expr != "ebx"
                    || !hint_limit.is_some_and(|l| l > 0)
                {
                    issue!("reactivate_nonzero_hp requires a proven counted record loop");
                    continue;
                }
                let limit = hint_limit.unwrap_or_default();
                if active_slot_count <= 0 || limit > active_slot_count {
                    issue!(format!(
                        "reactivate_nonzero_hp limit {limit} exceeds active loadch slot_count={active_slot_count}"
                    ));
                    continue;
                }
                let mut beat = runtime(input, "reactivate_nonzero_hp");
                beat.count = limit;
                beat.dir = input.direction;
                beats.push(beat);
            }
            "reset_pose" => {
                // 0x134e4 writes pose=0 to every materialized unit and waits 20ms.
                let mut beat = runtime(input, "reset_pose");
                beat.ms = 20;
                beats.push(beat);
            }
            "redraw" => {
                // Standalone 0x11cac(0) presents the already-materialized scene.
                let mut beat = runtime(input, "redraw");
                beat.frames = 1;
                beats.push(beat);
            }
            "native_ch23_loop" => {
                let Some(r#loop) = &input.native_ch23_loop else {
                    issue!("native_ch23_loop requires an explicit raw loop payload");
                    continue;
                };
                if let Err(err) = validate_native_ch23_loop(input, r#loop) {
                    issue!(err.message().to_string());
                    continue;
                }
                let mut beat = runtime(input, "native_ch23_loop");
                beat.native_ch23_loop = Some(r#loop.clone());
                beats.push(beat);
            }
            "native_2189a_loop" => {
                let Some(r#loop) = &input.native2189_a_loop else {
                    issue!("native_2189a_loop requires an explicit raw loop payload");
                    continue;
                };
                if let Err(err) = validate_native2189_a_loop(input, r#loop) {
                    issue!(err.message().to_string());
                    continue;
                }
                let mut copy_loop = r#loop.clone();
                copy_loop.slot = immediate_handler_int(&input.raw_args, 0).0;
                copy_loop.initial_radius = immediate_handler_int(&input.raw_args, 1).0;
                copy_loop.radius_step = immediate_handler_int(&input.raw_args, 2).0;
                let mut beat = runtime(input, "native_2189a_loop");
                beat.native2189_a_loop = Some(copy_loop);
                beats.push(beat);
            }
            "focus_unit" => {
                // 0x12d7b reads the selected unit X/Y and delegates to 0x12cea,
                // which walks the cursor there X-first/Y-second and scrolls only at
                // the original 13x8 viewport safe bands. Runtime owns that stateful path.
                let Some(slot) = input.unit_slot.filter(|v| *v >= 0) else {
                    issue!("focus_unit lacks a non-negative runtime slot");
                    continue;
                };
                if active_slot_count <= slot {
                    issue!(format!("focus_unit slot {slot} is outside active loadch slot_count={active_slot_count}"));
                    continue;
                }
                let mut beat = runtime(input, "focus_unit");
                beat.slot = Some(slot);
                beats.push(beat);
            }
            "join" => {
                let Some(char_id) = input.char_id else {
                    issue!("join lacks an original player char_id");
                    continue;
                };
                if !joinable_character_id(char_id) {
                    issue!(format!("join char_id {char_id} is outside the original 0..31 player roster"));
                    continue;
                }
                let mut beat = runtime(input, "join");
                beat.char_id = char_id;
                beats.push(beat);
            }
            // 0x11506 is the parameterless post-battle projection from the
            // current runtime unit array back to the persistent player roster.
            "sync_party" => beats.push(runtime(input, "sync_party")),
            // 0x25089 is the post-handler persistent player-roster cleanup.
            // Direct LE disassembly shows byte +5 cleared and current HP/MP
            // reloaded from +0x42/+0x46 for every roster entry; keep it as an
            // explicit editable primitive rather than folding it into sync_party.
            "reset_persistent_roster_state" => beats.push(runtime(input, "reset_persistent_roster_state")),
            "set_chapter" => {
                let Some(chapter) = input.chapter.filter(|v| *v >= 0) else {
                    issue!("set_chapter requires a non-negative immediate chapter");
                    continue;
                };
                let mut beat = runtime(input, "set_chapter");
                beat.chapter = Some(chapter);
                beats.push(beat);
            }
            "grant_item" => {
                let Some(item_id) = input.item_id.filter(|v| (0..=0xff).contains(v)) else {
                    issue!("grant_item requires an unsigned byte item_id");
                    continue;
                };
                let mut beat = runtime(input, "grant_item");
                beat.item_id = Some(item_id);
                beats.push(beat);
            }
            "load_res" | "play_sfx" | "release_res" => {
                let Some(resource_fn) = &bindings.resource else {
                    issue!(format!("{} requires an explicit resource binding", input.op));
                    continue;
                };
                let (resource, ok) = (resource_fn.0)(input.clone());
                if !ok || resource.resource_id < 0 {
                    issue!(format!("{} has no valid resource binding", input.op));
                    continue;
                }
                if input.op == "play_sfx" && !resource.sfx_index.is_some_and(|v| v >= -1) {
                    issue!("play_sfx requires an explicit sfx_index >= -1");
                    continue;
                }
                let mut beat = runtime(input, &input.op);
                beat.resource_id = Some(resource.resource_id);
                beat.resource_archive = resource.archive.clone();
                beat.resource_owner = resource.owner.clone();
                beat.sfx_index = resource.sfx_index;
                beats.push(beat);
            }
            "prepare_chapter_aux_graphics" => {
                if input.source.addr != "0x24a9a" || input.source.target != "0x10652" {
                    issue!("prepare_chapter_aux_graphics requires recovered ch22 call-site 0x24a9a");
                    continue;
                }
                beats.push(runtime(input, "native_ch22_prepare_aux"));
            }
            "palette_fade" => {
                // 0x1f525 performs baseline-minus-delta writes for inclusive deltas
                // 64..0 and waits 2ms after every write. Keep it separate from the
                // remake's generic RGBA story fade and reject exporter drift.
                if input.source.target != "0x1f525" || !input.raw_args.is_empty() || !input.args.is_empty() {
                    issue!("palette_fade requires exact no-argument 0x1f525 source");
                    continue;
                }
                if input.source.addr == "0x3241f" {
                    // ch00 map32_runtime predates raw FDICON-key provenance, so its
                    // indexed scene cannot yet be composed without guessing Fig==key.
                    // Retain the established playable RGBA approximation only at this
                    // exact call site; it is not evidence for 0x1f525 parity and must
                    // not become a generic fallback for ch09/ch19 post-battle handlers.
                    let mut beat = runtime(input, "fade");
                    beat.out = false;
                    beats.push(beat);
                    continue;
                }
                let mut beat = runtime(input, "native_palette_fade_in");
                beat.native_palette_fade_in = Some(NativePaletteFadeIn { start: 64, end: 0, delay_ms: 2 });
                beats.push(beat);
            }
            "unknown" | "native_call" | "unresolved_native_call" => {
                if input.op != "unknown"
                    && (input.native_semantic.is_empty()
                        || input.native_confidence.is_empty()
                        || input.native_evidence.is_empty())
                {
                    issue!("native call requires semantic/confidence/evidence metadata");
                    continue;
                }
                let target = input.native_target.as_str();
                if target == "0x4dbfc" {
                    if input.source.addr != "0x24a92" || input.source.target != "0x4dbfc" || input.raw_args.len() != 1 {
                        issue!("0x4DBFC ch22 reset requires exact call-site 0x24a92 and one raw owner");
                        continue;
                    }
                    beats.push(runtime(input, "native_ch22_reset_grid"));
                    continue;
                }
                if target == "0x35bba" {
                    let (value, ok) = immediate_handler_int(&input.raw_args, 0);
                    if input.source.addr != "0x254c0"
                        || input.source.target != "0x35bba"
                        || input.raw_args.len() != 1
                        || !ok
                        || value != 20
                    {
                        issue!("0x35BBA ch28 post lowering requires exact 0x254c0 argument 20");
                        continue;
                    }
                    let mut beat = runtime(input, "native_ch28_post_present");
                    beat.native_ch28_post_present = Some(NativeCh28PostPresent {
                        start_slot: 20,
                        pose_frames: 13,
                        fdother_resource: 5,
                        entry_first: 0x44,
                        entry_last: 0x4f,
                        sfx_resource: 31,
                        sfx_index: 3,
                        sfx_arg: 1,
                        work_size: 0x25680,
                        work_stride: 0x1c8,
                        view_base: 0x8088,
                        visible_columns: 13,
                        visible_rows: 8,
                    });
                    beats.push(beat);
                    continue;
                }
                if target == "0x22253" {
                    // ch28 post is the only dynamic-last-slot direct caller currently
                    // admitted. The exporter retains the first four immediate PUSHes and
                    // EBX for [0x53BEB]-1; cdecl reverses them into
                    // (lastSlot,15,10,15,10).
                    let mut valid =
                        input.source.addr == "0x25535" && input.source.target == "0x22253" && input.raw_args.len() == 5;
                    let want = [10, 15, 10, 15];
                    for (arg, expected) in want.iter().enumerate() {
                        let (value, ok) = immediate_handler_int(&input.raw_args, arg as i64);
                        valid = valid && ok && value == *expected;
                    }
                    // Go `handler_compile.go:1010` 直接取 `input.RawArgs[4]`，少於五個引數時 panic；
                    // Rust 照做（索引越界同樣 panic）。
                    let last = input.raw_args[4].as_str();
                    valid = valid && last == Some("ebx");
                    if !valid {
                        issue!("0x22253 direct lowering requires exact ch28 post 0x25535 PUSH payload");
                        continue;
                    }
                    let mut beat = runtime(input, "native_unit_present");
                    beat.native_unit_present = Some(NativeUnitPresent {
                        last_runtime_slot: true,
                        new_x: 15,
                        new_y: 10,
                        visual_x: 15,
                        visual_y: 10,
                        ..Default::default()
                    });
                    beats.push(beat);
                    continue;
                }
                // raw ch20 post calls the no-argument 0x24336 sequence exactly once.
                // The callee owns its (14,8) pan, FDOTHER #34 frame split, ANI #0
                // playback and palette flash. Keep the complete fixed payload in one
                // editable beat and key lowering to the proven caller, not merely the
                // shared-looking asset numbers.
                if target == "0x24336" {
                    if input.source.addr != "0x242c9" || input.source.target != "0x24336" || !input.raw_args.is_empty()
                    {
                        issue!("0x24336 sky-key sequence requires exact no-argument callsite 0x242c9");
                        continue;
                    }
                    let mut beat = runtime(input, "native_ch20_sky_key_sequence");
                    beat.native_ch20_sky_key = Some(NativeCh20SkyKeySequence {
                        pan_grid_x: 14,
                        pan_grid_y: 8,
                        fdother_resource: 34,
                        fdother_frame_count: 101,
                        base_frame: 0,
                        first_frame_start: 1,
                        first_frame_end: 68,
                        frame_wait_bios_ticks: 3,
                        palette_cycle_first: true,
                        ani_resource: 0,
                        ani_frame_count: 96,
                        ani_frame_delay_ms: 15,
                        ani_skippable: false,
                        palette_start: 0,
                        palette_end: 255,
                        flash_delta: 63,
                        flash_hold_ms: 100,
                        restore_delta: 0,
                        restore_hold_ms: 500,
                        tail_frame_start: 69,
                        tail_frame_end: 100,
                    });
                    beats.push(beat);
                    continue;
                }
                // ch07 post-battle has one fully immediate use of 0x11d40 followed
                // directly by memset(0xA0000,0,0xFA00).  VGA DAC components are
                // six-bit values (0..63), so subtracting 64 over entries 0..255
                // deterministically clamps the complete display to black.  Key the
                // lowering to the original call site and exact source PUSH order;
                // other 0x11d40 callers remain closed because many use registers or
                // only partial palette ranges.
                if target == "0x11d40" && input.source.addr == "0x23599" && input.source.target == "0x11d40" {
                    let (delta, ok_delta) = immediate_handler_int(&input.raw_args, 0);
                    let (end, ok_end) = immediate_handler_int(&input.raw_args, 1);
                    let (start, ok_start) = immediate_handler_int(&input.raw_args, 2);
                    if input.raw_args.len() != 3
                        || !ok_start
                        || !ok_end
                        || !ok_delta
                        || start != 0
                        || end != 255
                        || delta != 64
                    {
                        issue!("0x23599 blackout requires exact source PUSH order 64/255/0");
                        continue;
                    }
                    let mut beat = runtime(input, "native_palette_blackout");
                    beat.native_palette_blackout =
                        Some(NativePaletteBlackout { start, end, delta, clear_bytes: 0xFA00 });
                    beats.push(beat);
                    continue;
                }
                // 0x12cea(x,y) is the native X-first/Y-second camera focus loop.
                // Handler exports preserve the source PUSH order (y,x), so reverse the
                // two immediate arguments before lowering to the tile-step camera pan.
                if target == "0x12cea" {
                    if input.raw_args.len() < 2 {
                        issue!("0x12cea focus requires immediate x/y arguments");
                        continue;
                    }
                    let (y, ok_y) = immediate_handler_int(&input.raw_args, 0);
                    let (x, ok_x) = immediate_handler_int(&input.raw_args, 1);
                    if !ok_x || !ok_y || x < 0 || y < 0 {
                        issue!("0x12cea focus requires immediate non-negative x/y");
                        continue;
                    }
                    let mut pan = runtime(input, "pan");
                    pan.x = x.wrapping_mul(24);
                    pan.y = y.wrapping_mul(24);
                    pan.frames = 30;
                    pan.tile_step = true;
                    beats.push(pan);
                    continue;
                }
                // 0x35822 is the late-game staging helper used by ch27/ch28 handlers.
                // Handler exports preserve the source PUSH order (group,y,x), while
                // the callee passes (x,y) to 0x135dd and group to 0x10b4e.  Capstone
                // disassembly shows a camera pan to (x,y), direct spawn of group,
                // 300ms wait, a full-DAC delta=255 saturation to white, 200ms hold,
                // baseline restore with delta=0, then a redraw. Preserve the raw 255
                // argument and choreography as
                // ordinary editable beats instead of leaving the whole handler opaque.
                if target == "0x35822" {
                    let (group, ok_group) = immediate_handler_int(&input.raw_args, 0);
                    let (y, ok_y) = immediate_handler_int(&input.raw_args, 1);
                    let (x, ok_x) = immediate_handler_int(&input.raw_args, 2);
                    if !ok_x || !ok_y || !ok_group || x < 0 || y < 0 || !(0..=255).contains(&group) {
                        issue!("0x35822 staging helper requires immediate source PUSH order group/y/x");
                        continue;
                    }
                    let mut pan = runtime(input, "pan");
                    pan.x = x.wrapping_mul(24);
                    pan.y = y.wrapping_mul(24);
                    pan.frames = 30;
                    pan.tile_step = true;
                    beats.push(pan);
                    let mut spawn = runtime(input, "spawn");
                    spawn.group = group;
                    if let Some(context) = &bindings.runtime_context {
                        spawn.count = context.spawn_groups.get(&group).copied().unwrap_or(0);
                    }
                    beats.push(spawn);
                    let mut wait = runtime(input, "delay");
                    wait.ms = 300;
                    beats.push(wait);
                    let mut palette = runtime(input, "palette_update");
                    (palette.palette_start, palette.palette_end, palette.palette_delta) = (0, 255, 255);
                    beats.push(palette);
                    let mut delay = runtime(input, "delay");
                    delay.ms = 200;
                    beats.push(delay);
                    let mut palette = runtime(input, "palette_update");
                    (palette.palette_start, palette.palette_end, palette.palette_delta) = (0, 255, 0);
                    beats.push(palette);
                    let mut redraw = runtime(input, "redraw");
                    redraw.frames = 1;
                    beats.push(redraw);
                    continue;
                }
                if target == "0x37416"
                    && let Some(resource_fn) = &bindings.resource
                {
                    let (resource, ok) = (resource_fn.0)(input.clone());
                    if ok && resource.resource_id >= 0 {
                        let mut beat = runtime(input, "release_res");
                        beat.resource_id = Some(resource.resource_id);
                        beats.push(beat);
                        continue;
                    }
                }
                // 0x11df2 is a proven one-shot VGA DAC range update. Handler
                // exports keep it as unknown until this exact native signature is
                // recognized; it must not be confused with the black-overlay fade.
                if target == "0x11df2" {
                    // These handlers preserve a register-shaped loop body as one palette
                    // beat followed by one delay beat.  Direction and step are caller-local:
                    // ch22/ch28 include ascending forms, while later calls descend.  Expand
                    // the complete body and consume the adjacent exported delay exactly once.
                    if input.raw_args.len() >= 3 && input.raw_args[0].as_str() == Some("ebx") {
                        let (end, ok_end) = immediate_handler_int(&input.raw_args, 1);
                        let (delta, ok_delta) = immediate_handler_int(&input.raw_args, 2);
                        if ok_end && ok_delta && end == 255 && delta == 0 {
                            let (start, stop, step, known): (i64, i64, i64, bool) = match input.source.addr.as_str() {
                                "0x24a24" => (0, 0x3e, 2, true),
                                "0x256eb" => (0, 0x3f, 1, true),
                                "0x25733" | "0x258cd" => (0x3e, 0, -1, true),
                                _ => (0, 0, 0, false),
                            };
                            if !known || i + 1 >= script.beats.len() {
                                issue!(
                                    "register-bound 0x11df2 loop requires a recovered caller and adjacent 4ms delay"
                                );
                                continue;
                            }
                            let delay_input = &script.beats[i + 1];
                            if delay_input.op != "delay"
                                || delay_input.ms != Some(4)
                                || delay_input.source.target != "0x375b2"
                            {
                                issue!("register-bound 0x11df2 loop has no exact adjacent 4ms delay body");
                                continue;
                            }
                            let mut value = start;
                            loop {
                                let mut palette = runtime(input, "palette_update");
                                (palette.palette_start, palette.palette_end, palette.palette_delta) =
                                    (value, end, delta);
                                beats.push(palette);
                                let mut wait = runtime(delay_input, "delay");
                                wait.ms = 4;
                                beats.push(wait);
                                if value == stop {
                                    break;
                                }
                                value = value.wrapping_add(step);
                            }
                            consumed_loop_delay_beat = i as i64 + 1;
                            continue;
                        }
                    }
                    let (start, ok_start) = immediate_handler_int(&input.raw_args, 0);
                    let (end, ok_end) = immediate_handler_int(&input.raw_args, 1);
                    let (delta, ok_delta) = immediate_handler_int(&input.raw_args, 2);
                    if !ok_start
                        || !ok_end
                        || !ok_delta
                        || start < 0
                        || end < start
                        || end > 255
                        || !(-63..=255).contains(&delta)
                    {
                        issue!("0x11df2 palette_update requires immediate start/end/delta within VGA range");
                        continue;
                    }
                    let mut beat = runtime(input, "palette_update");
                    (beat.palette_start, beat.palette_end, beat.palette_delta) = (start, end, delta);
                    beats.push(beat);
                    continue;
                }
                if target == "0x24b4d" {
                    let (frames, ok) = immediate_handler_int(&input.raw_args, 0);
                    if !ok || frames <= 0 || frames > 255 {
                        issue!("0x24b4d transition_reveal requires a positive immediate frame count");
                        continue;
                    }
                    let mut beat = runtime(input, "transition_reveal");
                    beat.reveal_frames = frames;
                    beat.reveal_delay_ms = 20;
                    beats.push(beat);
                    continue;
                }
                if target == "0x25052" {
                    // 0x25052(start, delay_ms) is a palette-brightness ramp, not
                    // a generic fade: it calls 0x11df2(0, 255, start..0), waiting
                    // after every inclusive step.
                    let (start, ok_start) = immediate_handler_int(&input.raw_args, 0);
                    let (delay, ok_delay) = immediate_handler_int(&input.raw_args, 1);
                    if !ok_start || !ok_delay || !(0..=63).contains(&start) || delay < 0 {
                        issue!("0x25052 palette ramp requires immediate start 0..63 and non-negative delay");
                        continue;
                    }
                    for delta in (0..=start).rev() {
                        let mut palette = runtime(input, "palette_update");
                        (palette.palette_start, palette.palette_end, palette.palette_delta) = (0, 255, delta);
                        beats.push(palette);
                        let mut wait = runtime(input, "delay");
                        wait.ms = delay;
                        beats.push(wait);
                    }
                    continue;
                }
                if target == "0x1f882" {
                    // The handler exporter records the caller's register snapshot for
                    // this no-argument helper as [ebx,esi,edi]. Those are provenance,
                    // not call arguments; reject all other payloads fail-closed.
                    if !input.raw_args.is_empty() && !native_palette_fade_register_snapshot(&input.raw_args) {
                        issue!("0x1f882 palette fade takes no arguments");
                        continue;
                    }
                    let mut beat = runtime(input, "native_palette_fade_out");
                    beat.native_palette_fade = Some(NativePaletteFadeOut { start: 0, end: 63, delay_ms: 2 });
                    beats.push(beat);
                    continue;
                }
                if target == "0x13536" {
                    if !input.raw_args.is_empty() {
                        issue!("0x13536 record-bit7 clear takes no arguments");
                        continue;
                    }
                    // 0x13536 scans every materialized native record and clears only
                    // byte +5 bit7. Keep this raw operation distinct from resetting
                    // the remake's Acted projection.
                    beats.push(runtime(input, "clear_native_record_bit7"));
                    continue;
                }
                if target == "0x35e5a" {
                    if !input.raw_args.is_empty() {
                        issue!("0x35e5a palette pulse takes no arguments");
                        continue;
                    }
                    // 0x35e5a applies 0x11df2(0,255,delta) for inclusive
                    // 0..63, holds 400ms, then applies inclusive 62..0. Keep
                    // the asymmetric endpoints as data instead of approximating it
                    // as a generic fade or silently losing the final DAC value.
                    let mut beat = runtime(input, "native_palette_pulse");
                    beat.native_palette_pulse = Some(NativePalettePulse {
                        rise_start: 0,
                        rise_end: 63,
                        rise_delay_ms: 8,
                        hold_ms: 400,
                        fall_start: 62,
                        fall_end: 0,
                        fall_delay_ms: 8,
                    });
                    beats.push(beat);
                    continue;
                }
                if target == "0x33f78" {
                    if input.raw_args.len() != 3 {
                        issue!("0x33f78 staging wrapper requires three immediate arguments");
                        continue;
                    }
                    let (y, ok_y) = immediate_handler_int(&input.raw_args, 0);
                    let (x, ok_x) = immediate_handler_int(&input.raw_args, 1);
                    let (slot, ok_slot) = immediate_handler_int(&input.raw_args, 2);
                    if !ok_y || !ok_x || !ok_slot || x < 0 || y < 0 || slot < 0 {
                        issue!("0x33f78 staging wrapper requires non-negative immediate y/x/slot arguments");
                        continue;
                    }
                    // Extracted arguments retain native push order [y,x,slot]. The
                    // wrapper calls 0x12cea(x,y), then 0x22253(slot,x,y,x,y).
                    let mut beat = runtime(input, "native_staging_present");
                    beat.native_staging_present = Some(NativeStagingPresent { slot, x, y });
                    beats.push(beat);
                    continue;
                }
                if target == "0x24618" {
                    let Some(transition_fn) = &bindings.transition else {
                        issue!("0x24618 indexed transition requires an explicit editable binding");
                        continue;
                    };
                    let (transition, ok) = (transition_fn.0)(input.clone());
                    if !ok
                        || transition.frames != 9
                        || transition.frame_delay_ms != 5
                        || transition.tail_delay_ms != 500
                        || transition.start_y != 0
                        || transition.end_y != 0xc0
                        || transition.clip_width != 0x138
                        || transition.clip_height != 0xc0
                        || transition.palette_range_start != 0
                        || transition.palette_range_end != 255
                        || transition.palette_delta_start != 0
                        || transition.palette_delta_end != 62
                        || transition.palette_delta_step != 2
                        || transition.palette_delay_ms != 4
                    {
                        issue!("0x24618 binding lacks recovered 9-frame/500ms/palette timing");
                        continue;
                    }
                    let mut beat = runtime(input, "indexed_transition");
                    beat.indexed_transition = Some(transition);
                    beats.push(beat);
                    continue;
                }
                if target == "0x25089" {
                    // Persistent player-roster cleanup used by ch26/ch29 normal and
                    // ending paths; semantics are kept separate from sync_party.
                    beats.push(runtime(input, "reset_persistent_roster_state"));
                    continue;
                }
                if target == "0x2bce5"
                    && input.source.addr == "0x25970"
                    && script.chapter == 29
                    && script.phase == "post"
                {
                    // #236 READY：最後原生call之後只有25975自迴圈，轉給既有ending owner。
                    beats.push(runtime(input, "ending_handoff"));
                    continue;
                }
                if target == "0x17aa9" {
                    // Native global wait samples the DOS BIOS tick counter (about
                    // 54.9ms/tick), not the remake's 60Hz presentation frame. Three
                    // display frames (~50ms) is the closest deterministic runtime
                    // boundary; retain the native count rather than treating it as
                    // one 16.7ms frame.
                    let (ticks, ok) = immediate_handler_int(&input.raw_args, 0);
                    if !ok || ticks <= 0 || ticks > 120 {
                        issue!("0x17aa9 tick wait requires a positive bounded immediate count");
                        continue;
                    }
                    let mut beat = runtime(input, "delay");
                    beat.frames = ticks * 3;
                    beats.push(beat);
                    continue;
                }
                issue!("operation has no proven runtime lowering");
            }
            "unit_present" => {
                // This legacy shape carries only the six-frame 0x22547 tail and no
                // proven caller. The source-specific native_unit_present lowering
                // owns 0x25535's complete battle-state choreography; reject every
                // remaining legacy payload rather than borrowing that caller ABI.
                if input.unit_present.is_none() {
                    issue!("unit_present requires an explicit placement payload");
                    continue;
                }
                issue!("unit_present is blocked: legacy payload lacks a proven native 0x22253 caller ABI");
            }
            "direct_record_patch" => {
                let proven_source = input.source.addr == "0x2362d"
                    || input.source.addr == "0x23ec4"
                    || input.source.addr == "0x33346" // 0x3332B 第十章開場：record 50／51 的 +0x26 寫 100
                    || input.source.addr == "0x25116"; // 第27章前16筆raw+5歸零；fd2_ch27_flow_20261016.json。
                let patch = match &input.direct_record_patch {
                    Some(patch)
                        if proven_source
                            && input.source.target.is_empty()
                            && !patch.units.is_empty()
                            && active_slot_count > 0 =>
                    {
                        patch
                    }
                    _ => {
                        issue!("direct_record_patch requires an active runtime frontier and unit writes");
                        continue;
                    }
                };
                let mut seen_slots = BTreeSet::new();
                let mut valid = true;
                for unit in &patch.units {
                    if unit.slot < 0
                        || unit.slot >= active_slot_count
                        || seen_slots.contains(&unit.slot)
                        || unit.x.is_none() != unit.y.is_none()
                        || (unit.x.is_none() && unit.pose.is_none() && unit.raw_bytes.is_empty())
                    {
                        valid = false;
                        break;
                    }
                    seen_slots.insert(unit.slot);
                    if let (Some(x), Some(y)) = (unit.x, unit.y)
                        && (!(0..=0xff).contains(&x) || !(0..=0xff).contains(&y))
                    {
                        valid = false;
                        break;
                    }
                    if unit.pose.is_some_and(|p| !(0..=3).contains(&p)) {
                        valid = false;
                        break;
                    }
                    let mut seen_offsets = BTreeSet::new();
                    for raw in &unit.raw_bytes {
                        if (raw.offset != 5 && !(0x22..=0x27).contains(&raw.offset))
                            || !(0..=0xff).contains(&raw.value)
                            || seen_offsets.contains(&raw.offset)
                        {
                            valid = false;
                            break;
                        }
                        seen_offsets.insert(raw.offset);
                    }
                    if !valid {
                        break;
                    }
                }
                if let Some(view) = &patch.view
                    && (view.camera_x < 0
                        || view.camera_y < 0
                        || view.cursor_x < 0
                        || view.cursor_y < 0
                        || view.visible_cursor_x != view.cursor_x.wrapping_sub(view.camera_x)
                        || view.visible_cursor_y != view.cursor_y.wrapping_sub(view.camera_y))
                {
                    valid = false;
                }
                if !valid {
                    issue!("direct_record_patch contains an invalid slot, sparse field, raw offset, or view");
                    continue;
                }
                let mut beat = runtime(input, "direct_record_patch");
                beat.direct_record_patch = Some(patch.clone());
                beats.push(beat);
            }
            "layout_units" => {
                let Some(layout_fn) = &bindings.layout else {
                    issue!("layout_units requires an explicit runtime-slot layout mapping");
                    continue;
                };
                let (layout, ok) = (layout_fn.0)(input.clone());
                if !ok || layout.units.is_empty() {
                    issue!("no complete unit layout mapping for native layout call");
                    continue;
                }
                let mut seen = BTreeSet::new();
                let mut valid = active_slot_count > 0 && minimum_slot_count > 0;
                for unit in &layout.units {
                    if unit.slot < 0
                        || unit.slot >= active_slot_count
                        || unit.slot >= minimum_slot_count
                        || unit.pose < 0
                        || unit.pose > 3
                        || seen.contains(&unit.slot)
                    {
                        valid = false;
                        break;
                    }
                    seen.insert(unit.slot);
                }
                if !valid {
                    issue!(format!(
                        "layout_units needs unique slots and poses 0..3 within every allowed materialized runtime frontier (minimum slot_count={minimum_slot_count}, maximum slot_count={active_slot_count}; native buffer capacity is not inferred)"
                    ));
                    continue;
                }
                let mut layout_beat = runtime(input, "layout_units");
                layout_beat.layout = Some(layout);
                beats.push(layout_beat);
                let mut redraw = runtime(input, "redraw");
                redraw.frames = 1;
                beats.push(redraw);
                let mut fade = runtime(input, "fade");
                fade.out = false;
                beats.push(fade);
                let mut delay = runtime(input, "delay");
                delay.ms = 200;
                beats.push(delay);
            }
            _ => issue!("operation has no proven runtime lowering"),
        }
    }
    (beats, issues)
}

/// Go：`handler_compile.go:1462`，`func attachNativeDialogueLayout(beat *Beat, layout *NativeDialogueLayout) error`
pub(crate) fn attach_native_dialogue_layout(beat: &mut Beat, layout: Option<&NativeDialogueLayout>) -> Result<()> {
    let Some(layout) = layout else {
        return Ok(());
    };
    layout.validate()?;
    let want_upper = layout.control == "FFEF" || layout.control == "FFED";
    if beat.upper != Some(want_upper) {
        // Go `handler_compile.go:1484` 以 `%v` 印 `*bool`：nil 是 `<nil>`，非 nil 是指標位址（每次執行
        // 不同）。Rust 非 nil 時改印布林值，列入黃金差分的 KNOWN_DIFFERENCES。
        let shown = match beat.upper {
            None => "<nil>".to_string(),
            Some(v) => v.to_string(),
        };
        return Err(campaign_err!("native dialogue control {} disagrees with typed upper={shown}", layout.control));
    }
    beat.native_dialogue = clone_native_dialogue_layout(Some(layout));
    Ok(())
}

/// Go：`handler_compile.go:1477`，`func immediateHandlerInt(args []any, index int) (int, bool)`
pub(crate) fn immediate_handler_int(args: &[serde_json::Value], index: i64) -> (i64, bool) {
    if index < 0 || index as u64 >= args.len() as u64 {
        return (0, false);
    }
    go_any_int(&args[index as usize])
}

/// Go：`handler_compile.go:1493`，`func validateNativeCh23Loop(input HandlerBeat, loop NativeCh23Loop) error`
pub(crate) fn validate_native_ch23_loop(input: &HandlerBeat, r#loop: &NativeCh23Loop) -> Result<()> {
    use serde_json::json;
    if r#loop.stage_latch_source != "byte_0x51a10" || r#loop.tick_counter_source != "[0x46c]" {
        return Err(campaign_err!("native_ch23_loop requires raw latch byte_0x51a10 and tick source [0x46c]"));
    }
    match r#loop.phase.as_str() {
        "initial" => {
            if input.source.addr != "0x24c61"
                || input.source.target != "0x11cac"
                || r#loop.repeat != 30
                || !native_call_matches(Some(&r#loop.draw), "0x24c61", "0x11cac", &[json!(1)])
                || !native_call_matches(Some(&r#loop.tick), "0x24c6b", "0x17aa9", &[json!(1)])
                || r#loop.palette.is_some()
                || !native_call_matches(Some(&r#loop.stage), "0x24c81", "0x24d22", &[json!("edi")])
                || !native_stage_values(&r#loop.stage_values, 2, 9)
            {
                return Err(campaign_err!(
                    "native_ch23_loop initial phase does not match 0x24c61..0x24c81 raw schedule"
                ));
            }
        }
        "palette" => {
            if input.source.addr != "0x24cc1"
                || input.source.target != "0x11d40"
                || r#loop.repeat != 12
                || !native_call_matches(
                    r#loop.palette.as_ref(),
                    "0x24cc1",
                    "0x11d40",
                    &[json!("esi"), json!(255), json!(0)],
                )
                || !native_call_matches(Some(&r#loop.draw), "0x24cd3", "0x11cac", &[json!(0)])
                || !native_call_matches(Some(&r#loop.tick), "0x24cdd", "0x17aa9", &[json!(1)])
                || !native_call_matches(Some(&r#loop.stage), "0x24cf2", "0x24d22", &[json!("edi")])
                || !native_stage_values(&r#loop.stage_values, 10, 14)
                || r#loop.palette_table_source != "0x60003"
            {
                return Err(campaign_err!(
                    "native_ch23_loop palette phase does not match 0x24cc1..0x24cf2 raw schedule"
                ));
            }
        }
        _ => {
            return Err(campaign_err!("native_ch23_loop has unknown phase {}", battle::go_quote(&r#loop.phase)));
        }
    }
    Ok(())
}

/// Go：`handler_compile.go:1521`，`func validateNative2189ALoop(input HandlerBeat, loop Native2189ALoop) error`
pub(crate) fn validate_native2189_a_loop(input: &HandlerBeat, r#loop: &Native2189ALoop) -> Result<()> {
    if input.source.target != "0x2189a" || input.raw_args.len() != 3 {
        return Err(campaign_err!("native_2189a_loop requires the original 0x2189a call and three raw immediates"));
    }
    let mut valid_outer = false;
    for want in [[10, 15, 1], [16, 30, 1]] {
        let mut valid = true;
        for (i, expected) in want.iter().enumerate() {
            let (got, ok) = immediate_handler_int(&input.raw_args, i as i64);
            if !ok || got != *expected {
                valid = false;
                break;
            }
        }
        if valid {
            valid_outer = true;
            break;
        }
    }
    if !valid_outer {
        return Err(campaign_err!(
            "native_2189a_loop outer raw immediates are not one of the recovered ch22 call-sites"
        ));
    }
    let l = r#loop;
    if l.repeat != 10
        || l.step_source != "caller_arg9"
        || l.work_offset != 0x8088
        || l.work_stride != 456
        || l.map_rows != 13
        || l.map_columns != 8
        || l.clip_width != 312
        || l.clip_height != 192
        || l.present_stride != 320
    {
        return Err(campaign_err!("native_2189a_loop geometry/timing differs from 0x2189a"));
    }
    if !native2189a_call_shape(&l.map_draw, "0x21914", "0x11eee", 6)
        || immediate_or_invalid(&l.map_draw.raw_args, 2) != 8
        || immediate_or_invalid(&l.map_draw.raw_args, 3) != 13
        || immediate_or_invalid(&l.map_draw.raw_args, 4) != 456
    {
        return Err(campaign_err!("native_2189a_loop map-draw call-site is not the recovered 13x8 setup"));
    }
    if !native2189a_call_shape(&l.composite, "0x21955", "0x219ad", 7)
        || immediate_or_invalid(&l.composite.raw_args, 1) != 192
        || immediate_or_invalid(&l.composite.raw_args, 2) != 0
        || immediate_or_invalid(&l.composite.raw_args, 3) != 12
    {
        return Err(campaign_err!("native_2189a_loop composite call-site is not the recovered 312x192 pass"));
    }
    if !native2189a_call_shape(&l.stage, "0x2195d", "0x127a9", 0) {
        return Err(campaign_err!("native_2189a_loop stage call-site is not proven"));
    }
    if !native2189a_call_shape(&l.present, "0x21986", "0x11eb0", 6)
        || immediate_or_invalid(&l.present.raw_args, 0) != 192
        || immediate_or_invalid(&l.present.raw_args, 1) != 456
        || immediate_or_invalid(&l.present.raw_args, 2) != 312
        || immediate_or_invalid(&l.present.raw_args, 4) != 320
    {
        return Err(campaign_err!("native_2189a_loop present call-site is not the recovered viewport"));
    }
    if !native2189a_call_shape(&l.tail, "0x219a3", "0x11cac", 1) || immediate_or_invalid(&l.tail.raw_args, 0) != 0 {
        return Err(campaign_err!("native_2189a_loop tail redraw call-site is not proven"));
    }
    Ok(())
}

/// Go：`handler_compile.go:1569`，`func native2189aCallShape(call HandlerNativeCall, addr, target string, rawCount int) bool`
pub(crate) fn native2189a_call_shape(call: &HandlerNativeCall, addr: &str, target: &str, raw_count: i64) -> bool {
    call.source.addr == addr && call.source.target == target && call.raw_args.len() as i64 == raw_count
}

/// Go：`handler_compile.go:1573`，`func immediateOrInvalid(args []any, index int) int`
pub(crate) fn immediate_or_invalid(args: &[serde_json::Value], index: i64) -> i64 {
    let (value, ok) = immediate_handler_int(args, index);
    if !ok {
        return -1;
    }
    value
}

/// Go：`handler_compile.go:1581`，`func nativeCallMatches(call *HandlerNativeCall, addr, target string, want []any) bool`
pub(crate) fn native_call_matches(
    call: Option<&HandlerNativeCall>,
    addr: &str,
    target: &str,
    want: &[serde_json::Value],
) -> bool {
    let Some(call) = call else {
        return false;
    };
    if call.source.addr != addr || call.source.target != target || call.raw_args.len() != want.len() {
        return false;
    }
    for (index, expected) in want.iter().enumerate() {
        match expected {
            serde_json::Value::String(value) => {
                if call.raw_args[index].as_str() != Some(value.as_str()) {
                    return false;
                }
            }
            // Go 的期望值是 `int`；`serde_json::Value` 以整數 Number 表達。
            serde_json::Value::Number(n) if n.as_i64().is_some() => {
                let (got, ok) = immediate_handler_int(&call.raw_args, index as i64);
                if !ok || Some(got) != n.as_i64() {
                    return false;
                }
            }
            _ => return false,
        }
    }
    true
}

/// Go：`handler_compile.go:1607`，`func nativeStageValues(values []int, start, end int) bool`
pub(crate) fn native_stage_values(values: &[i64], start: i64, end: i64) -> bool {
    if values.len() as i64 != end.wrapping_sub(start).wrapping_add(1) {
        return false;
    }
    for (index, &value) in values.iter().enumerate() {
        if value != start.wrapping_add(index as i64) {
            return false;
        }
    }
    true
}

/// Go：`handler_compile.go:1619`，`func nativePaletteFadeRegisterSnapshot(args []any) bool`
pub(crate) fn native_palette_fade_register_snapshot(args: &[serde_json::Value]) -> bool {
    if args.len() != 3 {
        return false;
    }
    let want = ["ebx", "esi", "edi"];
    args.iter().zip(want).all(|(value, register)| value.as_str() == Some(register))
}

/// JoinableCharacterID identifies the original permanent-player roster.  This
/// is not a portrait range: NPC and scene-only portraits share the wider ID
/// space and must never acquire party membership through JOIN.
///
/// Go：`handler_compile.go:1635`，`func JoinableCharacterID(id int) bool`
pub fn joinable_character_id(id: i64) -> bool {
    (0..32).contains(&id)
}

/// Go：`handler_compile.go:1637`，`func handlerBranchChangesCompileContext(beats []HandlerBeat) bool`
pub(crate) fn handler_branch_changes_compile_context(beats: &[HandlerBeat]) -> bool {
    for beat in beats {
        if beat.op == "loadch" || beat.op == "set_chapter" {
            return true;
        }
        if beat.op == "if"
            && (handler_branch_changes_compile_context(&beat.then)
                || handler_branch_changes_compile_context(&beat.r#else))
        {
            return true;
        }
    }
    false
}

/// Go：`handler_compile.go:1649`，`func handlerBranchNeedsActiveLoadCH(beats []HandlerBeat) bool`
pub(crate) fn handler_branch_needs_active_load_ch(beats: &[HandlerBeat]) -> bool {
    for beat in beats {
        // A proven LOADCH already establishes the roster frontier for a
        // conditional spawn.  SPAWN's local cardinality update is deliberately
        // discarded at the branch merge, so later beats cannot assume the new
        // slots exist on both arms.  Other slot-addressed operations still need
        // an explicit branch-context model.
        if matches!(
            beat.op.as_str(),
            "act" | "scroll_step" | "spawn_intro" | "deactivate_unit" | "reactivate_nonzero_hp" | "focus_unit"
        ) {
            return true;
        }
        if beat.op == "if"
            && (handler_branch_needs_active_load_ch(&beat.then) || handler_branch_needs_active_load_ch(&beat.r#else))
        {
            return true;
        }
    }
    false
}

/// Go：`handler_compile.go:1667`，`func actingUsesUnavailableSlot(frames []ActingFrame, slotCount int) bool`
pub(crate) fn acting_uses_unavailable_slot(frames: &[ActingFrame], slot_count: i64) -> bool {
    frames
        .iter()
        .flat_map(|frame| &frame.units)
        .any(|unit| unit.slot.is_some_and(|slot| slot < 0 || slot >= slot_count))
}

/// Go `any` 裡的整數判定：`immediateHandlerInt`（`handler_compile.go:1490`）的型別分支。
///
/// Go 由 JSON 解碼的數字一律是 `float64`：`int(value)` 截斷，只有截斷前後相等才算成功。Go 程式
/// 自建的 `int`／`int64` 一律成功；在 `serde_json::Value` 裡兩者都是整數 Number。超出 `i64` 的
/// 正整數在 Go 是 `float64`，`int()` 轉換超出範圍（amd64 得到 `math.MinInt64`），判定失敗。
pub(crate) fn go_any_int(value: &serde_json::Value) -> (i64, bool) {
    let serde_json::Value::Number(n) = value else {
        return (0, false);
    };
    if let Some(v) = n.as_i64() {
        return (v, true);
    }
    let f = n.as_f64().unwrap_or(f64::NAN);
    // Go 的 `int(f)` 對範圍外的值在 amd64 得到 `math.MinInt64`；範圍內照常截斷。
    let truncated = if f.is_nan() || !(-(i64::MAX as f64)..i64::MAX as f64).contains(&f) { i64::MIN } else { f as i64 };
    (truncated, f == truncated as f64)
}

// 測試：Go `remake/internal/campaign/handler_compile_test.go` 逐條移植。
#[cfg(test)]
mod handler_compile_test;
