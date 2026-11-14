//! Go 來源：`remake/internal/campaign/story_index_map.go`（171 行）。

use super::*;

/// StoryIndexMap is a conservative bridge from an original FDTXT string index
/// to the editable story file and line range that preserves its dialogue.  It
/// contains only mappings whose raw logical-utterance total exactly matches the
/// authored story line total; diagnostics deliberately remain outside Lookup.
///
/// Script paths are relative to the story directory supplied to
/// export_story_index_map.py.  They are not EXE paths and may be freely edited
/// by a remake campaign author.
///
/// Go：`story_index_map.go:17`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct StoryIndexMap {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"mapping_kind"`
    pub mapping_kind: String,
    /// Go 標籤：`json:"resources"`
    pub resources: Vec<StoryIndexResource>,
    /// Go 標籤：`json:"diagnostics,omitempty"`
    pub diagnostics: Vec<StoryIndexIssue>,
    pub(crate) by_source: BTreeMap<String, BTreeMap<String, StoryIndexScriptMapping>>,
}

/// StoryIndexResource corresponds to one FDTXT_NNN resource.  Multiple story
/// scripts may validly reuse it (for example the prologue and later chapter
/// branches), so callers must also supply Script to Lookup.
///
/// Go：`story_index_map.go:29`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct StoryIndexResource {
    /// Go 標籤：`json:"source_dat"`
    pub source_dat: String,
    /// Go 標籤：`json:"raw_string_count"`
    pub raw_string_count: i64,
    /// Go 標籤：`json:"raw_utterance_count"`
    pub raw_utterance_count: i64,
    /// Go 標籤：`json:"script_mappings"`
    pub script_mappings: Vec<StoryIndexScriptMapping>,
}

/// StoryIndexScriptMapping maps every original offset-table string in one
/// count-aligned script.  It never represents a best-effort text match.
///
/// Go：`story_index_map.go:38`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct StoryIndexScriptMapping {
    /// Go 標籤：`json:"script"`
    pub script: String,
    /// Go 標籤：`json:"source_dat"`
    pub source_dat: String,
    /// Go 標籤：`json:"status"`
    pub status: String,
    /// Go 標籤：`json:"raw_utterance_count"`
    pub raw_utterance_count: i64,
    /// Go 標籤：`json:"story_line_count"`
    pub story_line_count: i64,
    /// Go 標籤：`json:"mappings"`
    pub mappings: Vec<StoryIndexStringMap>,
}

/// StoryIndexStringMap expands one FDTXT string.  Targets stays a slice since
/// a real original string can cross authored scene boundaries.
///
/// Go：`story_index_map.go:49`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct StoryIndexStringMap {
    /// Go 標籤：`json:"string_index"`
    pub string_index: i64,
    /// Go 標籤：`json:"utterance_count"`
    pub utterance_count: i64,
    /// Go 標籤：`json:"targets"`
    pub targets: Vec<StoryIndexTarget>,
}

/// StoryIndexTarget identifies a contiguous range in one editable scene.
/// SceneIndex is authoritative when Scene is empty because the source story
/// format permits an intentionally unlabeled scene.
///
/// Go：`story_index_map.go:58`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct StoryIndexTarget {
    /// Go 標籤：`json:"scene"`
    pub scene: Option<String>,
    /// Go 標籤：`json:"scene_index"`
    pub scene_index: i64,
    /// Go 標籤：`json:"lines"`
    pub lines: Vec<i64>,
}

/// StoryIndexIssue is preserved exporter evidence for resources that cannot be
/// mechanically mapped yet.  Its optional fields intentionally reflect the
/// heterogeneous diagnostics produced by the extraction tool.
///
/// Go：`story_index_map.go:67`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct StoryIndexIssue {
    /// Go 標籤：`json:"kind"`
    pub kind: String,
    /// Go 標籤：`json:"source_dat,omitempty"`
    pub source_dat: String,
    /// Go 標籤：`json:"script,omitempty"`
    pub script: String,
    /// Go 標籤：`json:"raw_utterance_count,omitempty"`
    pub raw_utterance_count: Option<i64>,
    /// Go 標籤：`json:"story_line_count,omitempty"`
    pub story_line_count: Option<i64>,
    /// Go 標籤：`json:"message,omitempty"`
    pub message: String,
}

/// LoadStoryIndexMap reads and validates a count-aligned-only manifest emitted
/// by tools/export_story_index_map.py.  Invalid data is rejected rather than
/// silently becoming an incorrect dialogue binding.
///
/// Go：`story_index_map.go:79`，`func LoadStoryIndexMap(path string) (*StoryIndexMap, error)`
pub fn load_story_index_map(path: &str) -> Result<StoryIndexMap> {
    todo!("campaign/story_index_map.go:LoadStoryIndexMap")
}

/// Go：`story_index_map.go:117`，`func validateStoryIndexMapping(resource StoryIndexResource, mapping StoryIndexScriptMapping) error`
pub(crate) fn validate_story_index_mapping(
    resource: &StoryIndexResource,
    mapping: &StoryIndexScriptMapping,
) -> Result<()> {
    todo!("campaign/story_index_map.go:validateStoryIndexMapping")
}

impl StoryIndexMap {
    /// Lookup returns the exact editable targets for sourceDAT/stringIndex in one
    /// script.  The bool is false for an exporter diagnostic, missing resource,
    /// wrong script context, or absent raw index; callers must treat all four as
    /// unresolved rather than infer a line number.
    ///
    /// Go：`story_index_map.go:159`，`func (index *StoryIndexMap) Lookup(sourceDAT, script string, stringIndex int) ([]StoryIndexTarget, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn lookup(&self, source_dat: &str, script: &str, string_index: i64) -> (Vec<StoryIndexTarget>, bool) {
        todo!("campaign/story_index_map.go:StoryIndexMap.Lookup")
    }
}
