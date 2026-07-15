package campaign

import (
	"encoding/json"
	"fmt"
	"os"
)

// StoryIndexMap is a conservative bridge from an original FDTXT string index
// to the editable story file and line range that preserves its dialogue.  It
// contains only mappings whose raw logical-utterance total exactly matches the
// authored story line total; diagnostics deliberately remain outside Lookup.
//
// Script paths are relative to the story directory supplied to
// export_story_index_map.py.  They are not EXE paths and may be freely edited
// by a remake campaign author.
type StoryIndexMap struct {
	SchemaVersion int                  `json:"schema_version"`
	MappingKind   string               `json:"mapping_kind"`
	Resources     []StoryIndexResource `json:"resources"`
	Diagnostics   []StoryIndexIssue    `json:"diagnostics,omitempty"`

	bySource map[string]map[string]StoryIndexScriptMapping
}

// StoryIndexResource corresponds to one FDTXT_NNN resource.  Multiple story
// scripts may validly reuse it (for example the prologue and later chapter
// branches), so callers must also supply Script to Lookup.
type StoryIndexResource struct {
	SourceDAT         string                    `json:"source_dat"`
	RawStringCount    int                       `json:"raw_string_count"`
	RawUtteranceCount int                       `json:"raw_utterance_count"`
	ScriptMappings    []StoryIndexScriptMapping `json:"script_mappings"`
}

// StoryIndexScriptMapping maps every original offset-table string in one
// count-aligned script.  It never represents a best-effort text match.
type StoryIndexScriptMapping struct {
	Script            string                `json:"script"`
	SourceDAT         string                `json:"source_dat"`
	Status            string                `json:"status"`
	RawUtteranceCount int                   `json:"raw_utterance_count"`
	StoryLineCount    int                   `json:"story_line_count"`
	Mappings          []StoryIndexStringMap `json:"mappings"`
}

// StoryIndexStringMap expands one FDTXT string.  Targets stays a slice since
// a real original string can cross authored scene boundaries.
type StoryIndexStringMap struct {
	StringIndex    int                `json:"string_index"`
	UtteranceCount int                `json:"utterance_count"`
	Targets        []StoryIndexTarget `json:"targets"`
}

// StoryIndexTarget identifies a contiguous range in one editable scene.
// SceneIndex is authoritative when Scene is empty because the source story
// format permits an intentionally unlabeled scene.
type StoryIndexTarget struct {
	Scene      *string `json:"scene"`
	SceneIndex int     `json:"scene_index"`
	Lines      []int   `json:"lines"`
}

// StoryIndexIssue is preserved exporter evidence for resources that cannot be
// mechanically mapped yet.  Its optional fields intentionally reflect the
// heterogeneous diagnostics produced by the extraction tool.
type StoryIndexIssue struct {
	Kind              string `json:"kind"`
	SourceDAT         string `json:"source_dat,omitempty"`
	Script            string `json:"script,omitempty"`
	RawUtteranceCount *int   `json:"raw_utterance_count,omitempty"`
	StoryLineCount    *int   `json:"story_line_count,omitempty"`
	Message           string `json:"message,omitempty"`
}

// LoadStoryIndexMap reads and validates a count-aligned-only manifest emitted
// by tools/export_story_index_map.py.  Invalid data is rejected rather than
// silently becoming an incorrect dialogue binding.
func LoadStoryIndexMap(path string) (*StoryIndexMap, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var result StoryIndexMap
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	if result.SchemaVersion != 1 {
		return nil, fmt.Errorf("story index map %q schema_version=%d, want 1", path, result.SchemaVersion)
	}
	if result.MappingKind != "count_aligned_only" {
		return nil, fmt.Errorf("story index map %q mapping_kind=%q, want count_aligned_only", path, result.MappingKind)
	}
	result.bySource = make(map[string]map[string]StoryIndexScriptMapping)
	for _, resource := range result.Resources {
		if resource.SourceDAT == "" || resource.RawStringCount < 0 || resource.RawUtteranceCount < 0 {
			return nil, fmt.Errorf("story index map %q has invalid resource %#v", path, resource)
		}
		byScript := result.bySource[resource.SourceDAT]
		if byScript == nil {
			byScript = make(map[string]StoryIndexScriptMapping)
			result.bySource[resource.SourceDAT] = byScript
		}
		for _, mapping := range resource.ScriptMappings {
			if err := validateStoryIndexMapping(resource, mapping); err != nil {
				return nil, fmt.Errorf("story index map %q: %w", path, err)
			}
			if _, duplicate := byScript[mapping.Script]; duplicate {
				return nil, fmt.Errorf("story index map %q has duplicate mapping for %s/%s", path, resource.SourceDAT, mapping.Script)
			}
			byScript[mapping.Script] = mapping
		}
	}
	return &result, nil
}

func validateStoryIndexMapping(resource StoryIndexResource, mapping StoryIndexScriptMapping) error {
	if mapping.Script == "" || mapping.SourceDAT != resource.SourceDAT || mapping.Status != "count_aligned" {
		return fmt.Errorf("invalid script mapping %#v for %s", mapping, resource.SourceDAT)
	}
	if mapping.RawUtteranceCount != resource.RawUtteranceCount || mapping.StoryLineCount != resource.RawUtteranceCount {
		return fmt.Errorf("mapping %s has unaligned counts raw=%d story=%d resource=%d", mapping.Script, mapping.RawUtteranceCount, mapping.StoryLineCount, resource.RawUtteranceCount)
	}
	if len(mapping.Mappings) != resource.RawStringCount {
		return fmt.Errorf("mapping %s has %d strings, want %d", mapping.Script, len(mapping.Mappings), resource.RawStringCount)
	}
	utterances := 0
	for index, item := range mapping.Mappings {
		if item.StringIndex != index || item.UtteranceCount < 0 || len(item.Targets) == 0 {
			return fmt.Errorf("mapping %s has invalid string entry %d", mapping.Script, index)
		}
		lines := 0
		for _, target := range item.Targets {
			if target.SceneIndex < 0 || len(target.Lines) == 0 {
				return fmt.Errorf("mapping %s string %d has invalid target", mapping.Script, index)
			}
			for lineIndex, line := range target.Lines {
				if line < 0 || (lineIndex > 0 && line != target.Lines[lineIndex-1]+1) {
					return fmt.Errorf("mapping %s string %d has non-contiguous target lines", mapping.Script, index)
				}
			}
			lines += len(target.Lines)
		}
		if lines != item.UtteranceCount {
			return fmt.Errorf("mapping %s string %d maps %d lines, want %d", mapping.Script, index, lines, item.UtteranceCount)
		}
		utterances += item.UtteranceCount
	}
	if utterances != resource.RawUtteranceCount {
		return fmt.Errorf("mapping %s maps %d utterances, want %d", mapping.Script, utterances, resource.RawUtteranceCount)
	}
	return nil
}

// Lookup returns the exact editable targets for sourceDAT/stringIndex in one
// script.  The bool is false for an exporter diagnostic, missing resource,
// wrong script context, or absent raw index; callers must treat all four as
// unresolved rather than infer a line number.
func (index *StoryIndexMap) Lookup(sourceDAT, script string, stringIndex int) ([]StoryIndexTarget, bool) {
	if index == nil || stringIndex < 0 {
		return nil, false
	}
	mapping, ok := index.bySource[sourceDAT][script]
	if !ok || stringIndex >= len(mapping.Mappings) {
		return nil, false
	}
	item := mapping.Mappings[stringIndex]
	targets := make([]StoryIndexTarget, len(item.Targets))
	copy(targets, item.Targets)
	return targets, true
}
