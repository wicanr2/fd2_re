//! Go 來源：`remake/internal/campaign/handler_binding.go`（426 行）。

use super::*;

/// HandlerBinding is campaign-authored evidence that connects one immutable
/// EXE-handler export to remake coordinates, text lines, and acting frames.
/// Overrides are keyed by HandlerBeat.Source.Addr, never by a reused resource
/// id, so a later loadch segment cannot accidentally inherit an earlier scene.
///
/// Go：`handler_binding.go:16`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct HandlerBinding {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"handler_script"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub handler_script: String,
    /// EvidenceOnly keeps generated/raw research bindings out of runnable
    /// campaign playback even when every nested raw operation is typed. A
    /// candidate authored for compiler regression may omit this flag; campaign
    /// nodes must never point at evidence-only bindings.
    /// Go 標籤：`json:"evidence_only,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub evidence_only: bool,
    /// Go 標籤：`json:"acting_resources,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub acting_resources: String,
    /// Go 標籤：`json:"story_index_map,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub story_index_map: String,
    /// Go 標籤：`json:"dialogue_contexts,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "BTreeMap::is_empty")]
    pub dialogue_contexts: BTreeMap<String, HandlerDialogueContext>,
    /// DialogueOverrides are keyed by "source_addr#text_index".  Native
    /// handlers sometimes reuse one call site for mutually exclusive FDTXT
    /// indices; an address-only context cannot represent that branch safely.
    /// Go 標籤：`json:"dialogue_overrides,omitempty"`
    /// Rust：Go 程式以 `== nil` 區分這個欄位的 nil 與空值，所以包成 `Option`。
    #[serde(skip_serializing_if = "Option::is_none")]
    pub dialogue_overrides: Option<BTreeMap<String, HandlerDialog>>,
    /// Go 標籤：`json:"runtime_context,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub runtime_context: Option<HandlerRuntimeContext>,
    /// Go 標籤：`json:"overrides"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub overrides: BTreeMap<String, HandlerBindingOverride>,
    #[serde(skip)]
    pub(crate) story_index: Option<StoryIndexMap>,
    /// Rust：Go 程式以 `== nil` 區分這個欄位的 nil 與空值，所以包成 `Option`。
    #[serde(skip)]
    pub(crate) acting: Option<BTreeMap<i64, Vec<ActingFrame>>>,
}

/// HandlerDialogueContext selects one reusable FDTXT-to-story mapping for a
/// dialog call site.  Script is the manifest's story-relative path; pairing it
/// with SourceDAT is essential because one FDTXT resource can be reused by
/// more than one campaign context.
///
/// Go：`handler_binding.go:42`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct HandlerDialogueContext {
    /// Go 標籤：`json:"source_dat"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub source_dat: String,
    /// Go 標籤：`json:"script"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub script: String,
}

/// HandlerBindingOverride has at most one field for the matching source op.
/// Omitted operations stay unresolved and are reported by CompileHandlerScript.
///
/// Go：`handler_binding.go:49`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct HandlerBindingOverride {
    /// Go 標籤：`json:"loadch,omitempty"`
    #[serde(rename = "loadch", skip_serializing_if = "Option::is_none")]
    pub load_ch: Option<LoadCHState>,
    /// Go 標籤：`json:"pan,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub pan: Option<HandlerPoint>,
    /// Go 標籤：`json:"dialog,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub dialog: Option<HandlerDialog>,
    /// Go 標籤：`json:"act,omitempty"`
    #[serde(rename = "act", skip_serializing_if = "Option::is_none")]
    pub acting: Option<HandlerActing>,
    /// Go 標籤：`json:"layout,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub layout: Option<HandlerLayout>,
    /// Go 標籤：`json:"indexed_transition,omitempty"`
    #[serde(rename = "indexed_transition", skip_serializing_if = "Option::is_none")]
    pub transition: Option<HandlerIndexedTransition>,
    /// Go 標籤：`json:"resource,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub resource: Option<HandlerResource>,
}

/// HandlerResource binds a native resource-table handle to an editable asset.
/// SFXIndex is optional because LOAD_RES/RELEASE_RES only identify the handle.
///
/// Go：`handler_binding.go:61`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct HandlerResource {
    /// Go 標籤：`json:"resource_id"`
    pub resource_id: i64,
    /// Go 標籤：`json:"archive,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub archive: String,
    /// Go 標籤：`json:"owner,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub owner: String,
    /// Go 標籤：`json:"sfx_index,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sfx_index: Option<i64>,
}

/// HandlerActing is a decoded, editable behavioural transcription.  It never
/// stores the original acting-resource bytes.
///
/// Go：`handler_binding.go:70`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct HandlerActing {
    /// Resource is an explicit decoded resource ID from ActingResources.  It
    /// avoids duplicating the same editable behavioural transcript at every
    /// handler call-site.  Frames remains for a one-off authored transcription.
    /// Go 標籤：`json:"resource,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub resource: Option<i64>,
    /// Go 標籤：`json:"frames,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub frames: Vec<ActingFrame>,
    /// TimingOnly records an original call whose decoded targets are all beyond
    /// the materialized unit_count. The original still runs the frame scheduler
    /// and redraws, but those units are not rendered. It is deliberately
    /// call-site-local evidence, never a property of the shared resource.
    /// Go 標籤：`json:"timing_only,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub timing_only: bool,
}

/// ActingResourceSet is a behavioural transcription of one runtime acting
/// table.  It intentionally contains decoded frame semantics, never original
/// bytes or pointers.  Keys are decimal resource IDs so JSON object ordering
/// cannot alter the data.
///
/// Go：`handler_binding.go:87`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct ActingResourceSet {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"resources"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub resources: BTreeMap<String, Vec<ActingFrame>>,
}

/// LoadHandlerBinding reads an address-keyed binding file.  It validates the
/// structure only; whether it covers a complete handler is reported by the
/// compiler, allowing deliberately incremental RE work.
///
/// Go：`handler_binding.go:95`，`func LoadHandlerBinding(path string) (*HandlerBinding, error)`
pub fn load_handler_binding(path: &str) -> Result<HandlerBinding> {
    let raw = battle::read_file(path)?;
    let mut binding: HandlerBinding = battle::decode_json(&raw)?;
    let q = battle::go_quote(path);
    if binding.schema_version != 1 {
        return Err(campaign_err!("handler binding {q} schema_version={}, want 1", binding.schema_version));
    }
    if binding.handler_script.is_empty() {
        return Err(campaign_err!("handler binding {q} has no handler_script"));
    }
    if !binding.acting_resources.is_empty() {
        let set_path = go_join_dir(path, &binding.acting_resources);
        let resources = load_acting_resource_set(&set_path)
            .map_err(|err| err.wrap(format!("handler binding {q} acting_resources: ")))?;
        binding.acting = Some(resources);
    }
    if !binding.story_index_map.is_empty() {
        let index_path = go_join_dir(path, &binding.story_index_map);
        let index = load_story_index_map(&index_path)
            .map_err(|err| err.wrap(format!("handler binding {q} story_index_map: ")))?;
        binding.story_index = Some(index);
    }
    // Go 以 map 迭代順序檢查（`handler_binding.go:126/134/139`）；只影響多項不合法時先報哪一項。
    for (addr, context) in &binding.dialogue_contexts {
        if addr.is_empty() || context.source_dat.is_empty() || context.script.is_empty() {
            return Err(campaign_err!(
                "handler binding {q} has invalid dialogue context at {}",
                battle::go_quote(addr)
            ));
        }
        if binding.story_index.is_none() {
            return Err(campaign_err!(
                "handler binding {q} dialogue context {} lacks story_index_map",
                battle::go_quote(addr)
            ));
        }
    }
    for (key, dialog) in binding.dialogue_overrides.iter().flatten() {
        if key.is_empty()
            || (dialog.segments.is_empty()
                && dialog.lines.is_empty()
                && dialog.script.is_empty()
                && dialog.scene_index.is_none())
        {
            return Err(campaign_err!(
                "handler binding {q} has invalid dialogue override at {}",
                battle::go_quote(key)
            ));
        }
    }
    for (addr, o) in &binding.overrides {
        let qa = battle::go_quote(addr);
        if addr.is_empty()
            || (o.load_ch.is_none()
                && o.pan.is_none()
                && o.dialog.is_none()
                && o.acting.is_none()
                && o.layout.is_none()
                && o.transition.is_none()
                && o.resource.is_none())
        {
            return Err(campaign_err!("handler binding {q} has empty override at {qa}"));
        }
        if let Some(state) = &o.load_ch
            && (state.chapter < 0
                || state.map.is_empty()
                || state.roster.is_empty()
                || state.slot_count <= 0
                || state.script.is_empty())
        {
            return Err(campaign_err!("handler binding {q} has incomplete loadch override at {qa}"));
        }
        if let Some(acting) = &o.acting {
            if acting.resource.is_some() && !acting.frames.is_empty() {
                return Err(campaign_err!("handler binding {q} act override {qa} mixes resource and inline frames"));
            }
            if acting.resource.is_none() && acting.frames.is_empty() {
                return Err(campaign_err!("handler binding {q} has empty act override at {qa}"));
            }
            if acting.timing_only && acting.resource.is_none() {
                return Err(campaign_err!(
                    "handler binding {q} act override {qa} timing_only requires a decoded resource"
                ));
            }
            if let Some(resource) = acting.resource {
                let Some(acting_set) = &binding.acting else {
                    return Err(campaign_err!(
                        "handler binding {q} act override {qa} has resource without acting_resources"
                    ));
                };
                if !acting_set.contains_key(&resource) {
                    return Err(campaign_err!(
                        "handler binding {q} act override {qa} references missing resource {resource}"
                    ));
                }
            }
        }
    }
    if let Some(context) = &binding.runtime_context {
        if context.slot_count > 0 && !context.slot_counts.is_empty() {
            return Err(campaign_err!("handler binding {q} runtime_context cannot mix slot_count and slot_counts"));
        }
        if context.minimum_slot_count() <= 0 {
            return Err(campaign_err!("handler binding {q} runtime_context needs positive slot_count or slot_counts"));
        }
        let mut previous = 0;
        for &count in &context.slot_counts {
            if count <= previous {
                return Err(campaign_err!(
                    "handler binding {q} runtime_context slot_counts must be unique ascending positive integers"
                ));
            }
            previous = count;
        }
        for (&group, &count) in &context.spawn_groups {
            if group < 0 || count <= 0 {
                return Err(campaign_err!(
                    "handler binding {q} runtime_context has invalid spawn group {group} count {count}"
                ));
            }
        }
    }
    Ok(binding)
}

/// CompileHandlerBinding is the fail-closed runtime/editor entry point for one
/// editable handler binding.  It resolves HandlerScript relative to the
/// binding file, then returns every unresolved source operation as an issue;
/// callers must not start playback while issues remain.
///
/// Go：`handler_binding.go:193`，`func CompileHandlerBinding(path string) ([]Beat, []HandlerCompileIssue, error)`
pub fn compile_handler_binding(path: &str) -> Result<(Vec<Beat>, Vec<HandlerCompileIssue>)> {
    let binding = load_handler_binding(path)?;
    let script = load_handler_script(&go_join_dir(path, &binding.handler_script))
        .map_err(|err| err.wrap(format!("handler binding {} handler_script: ", battle::go_quote(path))))?;
    let (mut beats, mut issues) = compile_handler_script(Some(&script), &binding.compiler_bindings());
    if binding.evidence_only && !script.beats.is_empty() {
        issues.push(HandlerCompileIssue {
            beat: -1,
            op: "evidence_only".into(),
            source: HandlerSource { addr: "binding".into(), ..Default::default() },
            reason: "binding is marked evidence_only and cannot enter runnable playback".into(),
        });
    }
    if issues.is_empty()
        && let Some(context) = &binding.runtime_context
    {
        let mut context = context.clone();
        context.spawn_groups = clone_int_map(Some(&context.spawn_groups));
        beats.insert(0, Beat { op: "runtime_context".into(), runtime_context: Some(context), ..Default::default() });
    }
    Ok((beats, issues))
}

/// Go：`handler_binding.go:219`，`func cloneIntMap(input map[int]int) map[int]int`
pub(crate) fn clone_int_map(input: Option<&BTreeMap<i64, i64>>) -> BTreeMap<i64, i64> {
    // Go 對 nil map 回傳 nil；Rust 的 BTreeMap 不分 nil 與空，兩者都回傳空表。
    input.cloned().unwrap_or_default()
}

impl HandlerBinding {
    /// CompilerBindings exposes this binding file to the generic compiler.  A
    /// missing override returns false, which deliberately becomes a compile issue.
    ///
    /// Go：`handler_binding.go:232`，`func (binding *HandlerBinding) CompilerBindings() HandlerBindings`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn compiler_bindings(&self) -> HandlerBindings {
        // Go 的閉包捕捉 `*HandlerBinding`；Rust 以 `Arc` 共用同一份（綁定載入後不再改寫）。
        let binding = std::sync::Arc::new(self.clone());
        fn lookup(binding: &HandlerBinding, input: &HandlerBeat) -> Option<HandlerBindingOverride> {
            // A native acting call can share one call instruction while its
            // branch-local immediate is pushed at different source addresses.
            // Prefer an address-plus-resource override before the ordinary address
            // key so the original call site remains the stable identity.
            if input.op == "act"
                && let Some(acting_id) = input.acting_id
                && let Some(scoped) = binding.overrides.get(&format!("{}#{acting_id}", input.source.addr))
            {
                return Some(scoped.clone());
            }
            binding.overrides.get(&input.source.addr).cloned()
        }
        let b = std::sync::Arc::clone(&binding);
        let load_ch = move |input: HandlerBeat| match lookup(&b, &input).and_then(|o| o.load_ch) {
            Some(state) => (state, true),
            None => (LoadCHState::default(), false),
        };
        let b = std::sync::Arc::clone(&binding);
        let pan = move |input: HandlerBeat| match lookup(&b, &input).and_then(|o| o.pan) {
            Some(point) => (point, true),
            None => (HandlerPoint::default(), false),
        };
        let b = std::sync::Arc::clone(&binding);
        let dialog = move |input: HandlerBeat| {
            if let Some(dialog) = lookup(&b, &input).and_then(|o| o.dialog) {
                // Hand-authored overrides retain priority for observations such as
                // the prologue's non-default upper dialogue-box placement.
                return (dialog, true);
            }
            let (dialog, ok) = b.dialogue_override(&input);
            if ok {
                return (dialog, true);
            }
            b.indexed_dialog(&input)
        };
        let b = std::sync::Arc::clone(&binding);
        let acting = move |input: HandlerBeat| {
            let Some(acting) = lookup(&b, &input).and_then(|o| o.acting) else {
                return (Vec::new(), false);
            };
            let Some(resource) = acting.resource else {
                return (acting.frames, true);
            };
            if input.acting_id != Some(resource) {
                return (Vec::new(), false);
            }
            let frames = b.acting.as_ref().and_then(|set| set.get(&resource));
            let Some(frames) = frames else {
                return (Vec::new(), false);
            };
            if !acting.timing_only {
                return (frames.clone(), true);
            }
            // Preserve the original frame/mode scheduler but erase writes to
            // inactive unit-array memory. This is the visible/runtime projection
            // proven for map31, not a generic bounds-error escape hatch.
            let timing = frames
                .iter()
                .map(|frame| ActingFrame { beats: frame.beats, special: frame.special, units: Vec::new() })
                .collect();
            (timing, true)
        };
        let b = std::sync::Arc::clone(&binding);
        let layout = move |input: HandlerBeat| match lookup(&b, &input).and_then(|o| o.layout) {
            Some(layout) => (layout, true),
            None => (HandlerLayout::default(), false),
        };
        let b = std::sync::Arc::clone(&binding);
        let transition = move |input: HandlerBeat| match lookup(&b, &input).and_then(|o| o.transition) {
            Some(transition) => (transition, true),
            None => (HandlerIndexedTransition::default(), false),
        };
        let b = std::sync::Arc::clone(&binding);
        let resource = move |input: HandlerBeat| match lookup(&b, &input).and_then(|o| o.resource) {
            Some(resource) if resource.resource_id >= 0 => (resource, true),
            _ => (HandlerResource::default(), false),
        };
        HandlerBindings {
            runtime_context: binding.runtime_context.clone(),
            load_ch: Some(Callback::new(std::sync::Arc::new(load_ch))),
            pan: Some(Callback::new(std::sync::Arc::new(pan))),
            dialog: Some(Callback::new(std::sync::Arc::new(dialog))),
            acting: Some(Callback::new(std::sync::Arc::new(acting))),
            layout: Some(Callback::new(std::sync::Arc::new(layout))),
            transition: Some(Callback::new(std::sync::Arc::new(transition))),
            resource: Some(Callback::new(std::sync::Arc::new(resource))),
        }
    }
}

/// LoadActingResourceSet reads a generated or hand-authored behavioural
/// transcript.  It verifies IDs and frame arrays before a handler binding can
/// claim to execute them.
///
/// Go：`handler_binding.go:330`，`func LoadActingResourceSet(path string) (map[int][]ActingFrame, error)`
pub fn load_acting_resource_set(path: &str) -> Result<BTreeMap<i64, Vec<ActingFrame>>> {
    let raw = battle::read_file(path)?;
    let set: ActingResourceSet = battle::decode_json(&raw)?;
    let q = battle::go_quote(path);
    if set.schema_version != 1 || set.resources.is_empty() {
        return Err(campaign_err!("acting resource set {q} needs schema_version=1 and resources"));
    }
    let mut resources = BTreeMap::new();
    // Go 以 map 迭代順序處理（`handler_binding.go:343`）；只影響多個鍵不合法或重複時先報哪一個。
    for (key, frames) in &set.resources {
        // Go `strconv.Atoi`：可帶正負號的十進位，與 Rust `i64` 的 `parse` 接受範圍相同。
        let id = match key.parse::<i64>() {
            Ok(id) if id >= 0 && !frames.is_empty() => id,
            _ => return Err(campaign_err!("acting resource set {q} has invalid resource {}", battle::go_quote(key))),
        };
        if resources.contains_key(&id) {
            return Err(campaign_err!("acting resource set {q} repeats resource {id}"));
        }
        resources.insert(id, frames.clone());
    }
    Ok(resources)
}

impl HandlerBinding {
    /// Go：`handler_binding.go:356`，`func (binding *HandlerBinding) indexedDialog(input HandlerBeat) (HandlerDialog, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn indexed_dialog(&self, input: &HandlerBeat) -> (HandlerDialog, bool) {
        let Some(story_index) = &self.story_index else {
            return (HandlerDialog::default(), false);
        };
        let Some(context) = self.dialogue_contexts.get(&input.source.addr) else {
            return (HandlerDialog::default(), false);
        };
        let (string_index, ok) = handler_text_index(&input.text_index);
        if !ok {
            return (HandlerDialog::default(), false);
        }
        let (targets, ok) = story_index.lookup(&context.source_dat, &context.script, string_index);
        if !ok || targets.is_empty() {
            return (HandlerDialog::default(), false);
        }
        if targets.len() > 1 {
            let mut dialog = HandlerDialog { segments: Vec::with_capacity(targets.len()), ..Default::default() };
            for target in &targets {
                let mut segment = HandlerDialogSegment {
                    script: context.script.clone(),
                    scene_index: Some(target.scene_index),
                    ..Default::default()
                };
                if let Some(scene) = &target.scene {
                    segment.scene = scene.clone();
                }
                for &line in &target.lines {
                    segment.lines.push(HandlerDialogLine { line, ..Default::default() });
                }
                if segment.lines.is_empty() {
                    return (HandlerDialog::default(), false);
                }
                dialog.segments.push(segment);
            }
            return (dialog, true);
        }
        let target = &targets[0];
        let mut dialog = HandlerDialog {
            script: context.script.clone(),
            scene_index: Some(target.scene_index),
            ..Default::default()
        };
        if let Some(scene) = &target.scene {
            dialog.scene = scene.clone();
        }
        for &line in &target.lines {
            dialog.lines.push(HandlerDialogLine { line, ..Default::default() });
        }
        (dialog, true)
    }

    /// Go：`handler_binding.go:402`，`func (binding *HandlerBinding) dialogueOverride(input HandlerBeat) (HandlerDialog, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn dialogue_override(&self, input: &HandlerBeat) -> (HandlerDialog, bool) {
        let Some(overrides) = &self.dialogue_overrides else {
            return (HandlerDialog::default(), false);
        };
        if input.source.addr.is_empty() {
            return (HandlerDialog::default(), false);
        }
        let (text_index, ok) = handler_text_index(&input.text_index);
        if !ok {
            return (HandlerDialog::default(), false);
        }
        match overrides.get(&format!("{}#{text_index}", input.source.addr)) {
            Some(dialog) => (dialog.clone(), true),
            None => (HandlerDialog::default(), false),
        }
    }
}

/// Go：`handler_binding.go:414`，`func handlerTextIndex(value any) (int, bool)`
pub(crate) fn handler_text_index(value: &serde_json::Value) -> (i64, bool) {
    // Go 的 `int` 分支（Go 程式自建的值）與 `float64` 分支（JSON 解碼的數字）在 `serde_json::Value`
    // 都是 Number：整數 Number 照 `int` 分支（`v >= 0`），其餘照 `float64` 分支。
    let serde_json::Value::Number(n) = value else {
        return (0, false);
    };
    if let Some(v) = n.as_i64() {
        return (v, v >= 0);
    }
    let v = n.as_f64().unwrap_or(f64::NAN);
    // Go：`v < 0 || math.Trunc(v) != v || v > math.MaxInt`；`math.MaxInt` 轉成 float64 是 2^63。
    if v.is_nan() || v < 0.0 || v.trunc() != v || v > i64::MAX as f64 {
        return (0, false);
    }
    // 恰好 2^63（`i64::MAX as f64`）時 Go `int(v)` 在 amd64 得到 `math.MinInt64`。
    let int = if v >= i64::MAX as f64 { i64::MIN } else { v as i64 };
    (int, true)
}

/// Go `filepath.Join(filepath.Dir(path), rel)`（`handler_binding.go:113/121/165`）。
///
/// 以 `std::path` 組路徑，不移植 `filepath.Clean` 的字面清理（`rust-port-no-go-stdlib`）：含 `..`
/// 的路徑在 Rust 保留原字樣，指向的檔案相同，只有錯誤訊息裡的路徑字樣可能不同（黃金差分的
/// KNOWN_DIFFERENCES）。
pub(crate) fn go_join_dir(path: &str, rel: &str) -> String {
    let dir = std::path::Path::new(path).parent().unwrap_or(std::path::Path::new(""));
    dir.join(rel).to_string_lossy().into_owned()
}
