//! Go 來源：`remake/internal/campaign/handler_binding.go`（426 行）。

use super::*;

/// HandlerBinding is campaign-authored evidence that connects one immutable
/// EXE-handler export to remake coordinates, text lines, and acting frames.
/// Overrides are keyed by HandlerBeat.Source.Addr, never by a reused resource
/// id, so a later loadch segment cannot accidentally inherit an earlier scene.
///
/// Go：`handler_binding.go:16`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct HandlerBinding {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"handler_script"`
    pub handler_script: String,
    /// EvidenceOnly keeps generated/raw research bindings out of runnable
    /// campaign playback even when every nested raw operation is typed. A
    /// candidate authored for compiler regression may omit this flag; campaign
    /// nodes must never point at evidence-only bindings.
    /// Go 標籤：`json:"evidence_only,omitempty"`
    pub evidence_only: bool,
    /// Go 標籤：`json:"acting_resources,omitempty"`
    pub acting_resources: String,
    /// Go 標籤：`json:"story_index_map,omitempty"`
    pub story_index_map: String,
    /// Go 標籤：`json:"dialogue_contexts,omitempty"`
    pub dialogue_contexts: BTreeMap<String, HandlerDialogueContext>,
    /// DialogueOverrides are keyed by "source_addr#text_index".  Native
    /// handlers sometimes reuse one call site for mutually exclusive FDTXT
    /// indices; an address-only context cannot represent that branch safely.
    /// Go 標籤：`json:"dialogue_overrides,omitempty"`
    /// Rust：Go 程式以 `== nil` 區分這個欄位的 nil 與空值，所以包成 `Option`。
    pub dialogue_overrides: Option<BTreeMap<String, HandlerDialog>>,
    /// Go 標籤：`json:"runtime_context,omitempty"`
    pub runtime_context: Option<HandlerRuntimeContext>,
    /// Go 標籤：`json:"overrides"`
    pub overrides: BTreeMap<String, HandlerBindingOverride>,
    pub(crate) story_index: Option<StoryIndexMap>,
    /// Rust：Go 程式以 `== nil` 區分這個欄位的 nil 與空值，所以包成 `Option`。
    pub(crate) acting: Option<BTreeMap<i64, Vec<ActingFrame>>>,
}

/// HandlerDialogueContext selects one reusable FDTXT-to-story mapping for a
/// dialog call site.  Script is the manifest's story-relative path; pairing it
/// with SourceDAT is essential because one FDTXT resource can be reused by
/// more than one campaign context.
///
/// Go：`handler_binding.go:42`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct HandlerDialogueContext {
    /// Go 標籤：`json:"source_dat"`
    pub source_dat: String,
    /// Go 標籤：`json:"script"`
    pub script: String,
}

/// HandlerBindingOverride has at most one field for the matching source op.
/// Omitted operations stay unresolved and are reported by CompileHandlerScript.
///
/// Go：`handler_binding.go:49`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct HandlerBindingOverride {
    /// Go 標籤：`json:"loadch,omitempty"`
    pub load_ch: Option<LoadCHState>,
    /// Go 標籤：`json:"pan,omitempty"`
    pub pan: Option<HandlerPoint>,
    /// Go 標籤：`json:"dialog,omitempty"`
    pub dialog: Option<HandlerDialog>,
    /// Go 標籤：`json:"act,omitempty"`
    pub acting: Option<HandlerActing>,
    /// Go 標籤：`json:"layout,omitempty"`
    pub layout: Option<HandlerLayout>,
    /// Go 標籤：`json:"indexed_transition,omitempty"`
    pub transition: Option<HandlerIndexedTransition>,
    /// Go 標籤：`json:"resource,omitempty"`
    pub resource: Option<HandlerResource>,
}

/// HandlerResource binds a native resource-table handle to an editable asset.
/// SFXIndex is optional because LOAD_RES/RELEASE_RES only identify the handle.
///
/// Go：`handler_binding.go:61`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct HandlerResource {
    /// Go 標籤：`json:"resource_id"`
    pub resource_id: i64,
    /// Go 標籤：`json:"archive,omitempty"`
    pub archive: String,
    /// Go 標籤：`json:"owner,omitempty"`
    pub owner: String,
    /// Go 標籤：`json:"sfx_index,omitempty"`
    pub sfx_index: Option<i64>,
}

/// HandlerActing is a decoded, editable behavioural transcription.  It never
/// stores the original acting-resource bytes.
///
/// Go：`handler_binding.go:70`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct HandlerActing {
    /// Resource is an explicit decoded resource ID from ActingResources.  It
    /// avoids duplicating the same editable behavioural transcript at every
    /// handler call-site.  Frames remains for a one-off authored transcription.
    /// Go 標籤：`json:"resource,omitempty"`
    pub resource: Option<i64>,
    /// Go 標籤：`json:"frames,omitempty"`
    pub frames: Vec<ActingFrame>,
    /// TimingOnly records an original call whose decoded targets are all beyond
    /// the materialized unit_count. The original still runs the frame scheduler
    /// and redraws, but those units are not rendered. It is deliberately
    /// call-site-local evidence, never a property of the shared resource.
    /// Go 標籤：`json:"timing_only,omitempty"`
    pub timing_only: bool,
}

/// ActingResourceSet is a behavioural transcription of one runtime acting
/// table.  It intentionally contains decoded frame semantics, never original
/// bytes or pointers.  Keys are decimal resource IDs so JSON object ordering
/// cannot alter the data.
///
/// Go：`handler_binding.go:87`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ActingResourceSet {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"resources"`
    pub resources: BTreeMap<String, Vec<ActingFrame>>,
}

/// LoadHandlerBinding reads an address-keyed binding file.  It validates the
/// structure only; whether it covers a complete handler is reported by the
/// compiler, allowing deliberately incremental RE work.
///
/// Go：`handler_binding.go:95`，`func LoadHandlerBinding(path string) (*HandlerBinding, error)`
pub fn load_handler_binding(path: &str) -> Result<HandlerBinding> {
    todo!("campaign/handler_binding.go:LoadHandlerBinding")
}

/// CompileHandlerBinding is the fail-closed runtime/editor entry point for one
/// editable handler binding.  It resolves HandlerScript relative to the
/// binding file, then returns every unresolved source operation as an issue;
/// callers must not start playback while issues remain.
///
/// Go：`handler_binding.go:193`，`func CompileHandlerBinding(path string) ([]Beat, []HandlerCompileIssue, error)`
pub fn compile_handler_binding(path: &str) -> Result<(Vec<Beat>, Vec<HandlerCompileIssue>)> {
    todo!("campaign/handler_binding.go:CompileHandlerBinding")
}

/// Go：`handler_binding.go:219`，`func cloneIntMap(input map[int]int) map[int]int`
pub(crate) fn clone_int_map(input: Option<&BTreeMap<i64, i64>>) -> BTreeMap<i64, i64> {
    todo!("campaign/handler_binding.go:cloneIntMap")
}

impl HandlerBinding {
    /// CompilerBindings exposes this binding file to the generic compiler.  A
    /// missing override returns false, which deliberately becomes a compile issue.
    ///
    /// Go：`handler_binding.go:232`，`func (binding *HandlerBinding) CompilerBindings() HandlerBindings`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn compiler_bindings(&self) -> HandlerBindings {
        todo!("campaign/handler_binding.go:HandlerBinding.CompilerBindings")
    }
}

/// LoadActingResourceSet reads a generated or hand-authored behavioural
/// transcript.  It verifies IDs and frame arrays before a handler binding can
/// claim to execute them.
///
/// Go：`handler_binding.go:330`，`func LoadActingResourceSet(path string) (map[int][]ActingFrame, error)`
pub fn load_acting_resource_set(path: &str) -> Result<BTreeMap<i64, Vec<ActingFrame>>> {
    todo!("campaign/handler_binding.go:LoadActingResourceSet")
}

impl HandlerBinding {
    /// Go：`handler_binding.go:356`，`func (binding *HandlerBinding) indexedDialog(input HandlerBeat) (HandlerDialog, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn indexed_dialog(&self, input: &HandlerBeat) -> (HandlerDialog, bool) {
        todo!("campaign/handler_binding.go:HandlerBinding.indexedDialog")
    }

    /// Go：`handler_binding.go:402`，`func (binding *HandlerBinding) dialogueOverride(input HandlerBeat) (HandlerDialog, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn dialogue_override(&self, input: &HandlerBeat) -> (HandlerDialog, bool) {
        todo!("campaign/handler_binding.go:HandlerBinding.dialogueOverride")
    }
}

/// Go：`handler_binding.go:414`，`func handlerTextIndex(value any) (int, bool)`
pub(crate) fn handler_text_index(value: &serde_json::Value) -> (i64, bool) {
    todo!("campaign/handler_binding.go:handlerTextIndex")
}
