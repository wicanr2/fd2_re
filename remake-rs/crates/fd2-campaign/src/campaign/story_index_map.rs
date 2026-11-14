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
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct StoryIndexMap {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"mapping_kind"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub mapping_kind: String,
    /// Go 標籤：`json:"resources"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub resources: Vec<StoryIndexResource>,
    /// Go 標籤：`json:"diagnostics,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub diagnostics: Vec<StoryIndexIssue>,
    #[serde(skip)]
    pub(crate) by_source: BTreeMap<String, BTreeMap<String, StoryIndexScriptMapping>>,
}

/// StoryIndexResource corresponds to one FDTXT_NNN resource.  Multiple story
/// scripts may validly reuse it (for example the prologue and later chapter
/// branches), so callers must also supply Script to Lookup.
///
/// Go：`story_index_map.go:29`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct StoryIndexResource {
    /// Go 標籤：`json:"source_dat"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub source_dat: String,
    /// Go 標籤：`json:"raw_string_count"`
    pub raw_string_count: i64,
    /// Go 標籤：`json:"raw_utterance_count"`
    pub raw_utterance_count: i64,
    /// Go 標籤：`json:"script_mappings"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub script_mappings: Vec<StoryIndexScriptMapping>,
}

/// StoryIndexScriptMapping maps every original offset-table string in one
/// count-aligned script.  It never represents a best-effort text match.
///
/// Go：`story_index_map.go:38`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct StoryIndexScriptMapping {
    /// Go 標籤：`json:"script"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub script: String,
    /// Go 標籤：`json:"source_dat"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub source_dat: String,
    /// Go 標籤：`json:"status"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub status: String,
    /// Go 標籤：`json:"raw_utterance_count"`
    pub raw_utterance_count: i64,
    /// Go 標籤：`json:"story_line_count"`
    pub story_line_count: i64,
    /// Go 標籤：`json:"mappings"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub mappings: Vec<StoryIndexStringMap>,
}

/// StoryIndexStringMap expands one FDTXT string.  Targets stays a slice since
/// a real original string can cross authored scene boundaries.
///
/// Go：`story_index_map.go:49`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct StoryIndexStringMap {
    /// Go 標籤：`json:"string_index"`
    pub string_index: i64,
    /// Go 標籤：`json:"utterance_count"`
    pub utterance_count: i64,
    /// Go 標籤：`json:"targets"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub targets: Vec<StoryIndexTarget>,
}

/// StoryIndexTarget identifies a contiguous range in one editable scene.
/// SceneIndex is authoritative when Scene is empty because the source story
/// format permits an intentionally unlabeled scene.
///
/// Go：`story_index_map.go:58`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct StoryIndexTarget {
    /// Go 標籤：`json:"scene"`
    pub scene: Option<String>,
    /// Go 標籤：`json:"scene_index"`
    pub scene_index: i64,
    /// Go 標籤：`json:"lines"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub lines: Vec<i64>,
}

/// StoryIndexIssue is preserved exporter evidence for resources that cannot be
/// mechanically mapped yet.  Its optional fields intentionally reflect the
/// heterogeneous diagnostics produced by the extraction tool.
///
/// Go：`story_index_map.go:67`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct StoryIndexIssue {
    /// Go 標籤：`json:"kind"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub kind: String,
    /// Go 標籤：`json:"source_dat,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub source_dat: String,
    /// Go 標籤：`json:"script,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub script: String,
    /// Go 標籤：`json:"raw_utterance_count,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub raw_utterance_count: Option<i64>,
    /// Go 標籤：`json:"story_line_count,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub story_line_count: Option<i64>,
    /// Go 標籤：`json:"message,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub message: String,
}

/// LoadStoryIndexMap reads and validates a count-aligned-only manifest emitted
/// by tools/export_story_index_map.py.  Invalid data is rejected rather than
/// silently becoming an incorrect dialogue binding.
///
/// Go：`story_index_map.go:79`，`func LoadStoryIndexMap(path string) (*StoryIndexMap, error)`
pub fn load_story_index_map(path: &str) -> Result<StoryIndexMap> {
    let raw = battle::read_file(path)?;
    let mut result: StoryIndexMap = battle::decode_json(&raw)?;
    let q = battle::go_quote(path);
    if result.schema_version != 1 {
        return Err(campaign_err!("story index map {q} schema_version={}, want 1", result.schema_version));
    }
    if result.mapping_kind != "count_aligned_only" {
        return Err(campaign_err!(
            "story index map {q} mapping_kind={}, want count_aligned_only",
            battle::go_quote(&result.mapping_kind)
        ));
    }
    let mut by_source: BTreeMap<String, BTreeMap<String, StoryIndexScriptMapping>> = BTreeMap::new();
    for resource in &result.resources {
        if resource.source_dat.is_empty() || resource.raw_string_count < 0 || resource.raw_utterance_count < 0 {
            // Go 以 `%#v` 印整個結構（Go 語法傾印）；Rust 改印 Debug，列入 KNOWN_DIFFERENCES。
            return Err(campaign_err!("story index map {q} has invalid resource {resource:?}"));
        }
        let by_script = by_source.entry(resource.source_dat.clone()).or_default();
        for mapping in &resource.script_mappings {
            validate_story_index_mapping(resource, mapping)
                .map_err(|err| err.wrap(format!("story index map {q}: ")))?;
            if by_script.contains_key(&mapping.script) {
                return Err(campaign_err!(
                    "story index map {q} has duplicate mapping for {}/{}",
                    resource.source_dat,
                    mapping.script
                ));
            }
            by_script.insert(mapping.script.clone(), mapping.clone());
        }
    }
    result.by_source = by_source;
    Ok(result)
}

/// Go：`story_index_map.go:117`，`func validateStoryIndexMapping(resource StoryIndexResource, mapping StoryIndexScriptMapping) error`
pub(crate) fn validate_story_index_mapping(
    resource: &StoryIndexResource,
    mapping: &StoryIndexScriptMapping,
) -> Result<()> {
    if mapping.script.is_empty() || mapping.source_dat != resource.source_dat || mapping.status != "count_aligned" {
        // Go 以 `%#v` 印整個結構；Rust 改印 Debug，列入 KNOWN_DIFFERENCES。
        return Err(campaign_err!("invalid script mapping {mapping:?} for {}", resource.source_dat));
    }
    if mapping.raw_utterance_count != resource.raw_utterance_count
        || mapping.story_line_count != resource.raw_utterance_count
    {
        return Err(campaign_err!(
            "mapping {} has unaligned counts raw={} story={} resource={}",
            mapping.script,
            mapping.raw_utterance_count,
            mapping.story_line_count,
            resource.raw_utterance_count
        ));
    }
    if mapping.mappings.len() as i64 != resource.raw_string_count {
        return Err(campaign_err!(
            "mapping {} has {} strings, want {}",
            mapping.script,
            mapping.mappings.len(),
            resource.raw_string_count
        ));
    }
    let mut utterances: i64 = 0;
    for (index, item) in mapping.mappings.iter().enumerate() {
        if item.string_index != index as i64 || item.utterance_count < 0 || item.targets.is_empty() {
            return Err(campaign_err!("mapping {} has invalid string entry {index}", mapping.script));
        }
        let mut lines: i64 = 0;
        for target in &item.targets {
            if target.scene_index < 0 || target.lines.is_empty() {
                return Err(campaign_err!("mapping {} string {index} has invalid target", mapping.script));
            }
            for (line_index, &line) in target.lines.iter().enumerate() {
                if line < 0 || (line_index > 0 && line != target.lines[line_index - 1].wrapping_add(1)) {
                    return Err(campaign_err!(
                        "mapping {} string {index} has non-contiguous target lines",
                        mapping.script
                    ));
                }
            }
            lines = lines.wrapping_add(target.lines.len() as i64);
        }
        if lines != item.utterance_count {
            return Err(campaign_err!(
                "mapping {} string {index} maps {lines} lines, want {}",
                mapping.script,
                item.utterance_count
            ));
        }
        utterances = utterances.wrapping_add(item.utterance_count);
    }
    if utterances != resource.raw_utterance_count {
        return Err(campaign_err!(
            "mapping {} maps {utterances} utterances, want {}",
            mapping.script,
            resource.raw_utterance_count
        ));
    }
    Ok(())
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
        if string_index < 0 {
            return (Vec::new(), false);
        }
        let Some(mapping) = self.by_source.get(source_dat).and_then(|m| m.get(script)) else {
            return (Vec::new(), false);
        };
        if string_index as u64 >= mapping.mappings.len() as u64 {
            return (Vec::new(), false);
        }
        (mapping.mappings[string_index as usize].targets.clone(), true)
    }
}

// 測試：Go `remake/internal/campaign/story_index_map_test.go` 逐條移植。
#[cfg(test)]
mod story_index_map_test;
