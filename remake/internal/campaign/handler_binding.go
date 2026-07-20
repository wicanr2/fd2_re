package campaign

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
)

// HandlerBinding is campaign-authored evidence that connects one immutable
// EXE-handler export to remake coordinates, text lines, and acting frames.
// Overrides are keyed by HandlerBeat.Source.Addr, never by a reused resource
// id, so a later loadch segment cannot accidentally inherit an earlier scene.
type HandlerBinding struct {
	SchemaVersion    int                               `json:"schema_version"`
	HandlerScript    string                            `json:"handler_script"`
	ActingResources  string                            `json:"acting_resources,omitempty"`
	StoryIndexMap    string                            `json:"story_index_map,omitempty"`
	DialogueContexts map[string]HandlerDialogueContext `json:"dialogue_contexts,omitempty"`
	RuntimeContext   *HandlerRuntimeContext            `json:"runtime_context,omitempty"`
	Overrides        map[string]HandlerBindingOverride `json:"overrides"`

	storyIndex *StoryIndexMap
	acting     map[int][]ActingFrame
}

// HandlerDialogueContext selects one reusable FDTXT-to-story mapping for a
// dialog call site.  Script is the manifest's story-relative path; pairing it
// with SourceDAT is essential because one FDTXT resource can be reused by
// more than one campaign context.
type HandlerDialogueContext struct {
	SourceDAT string `json:"source_dat"`
	Script    string `json:"script"`
}

// HandlerBindingOverride has at most one field for the matching source op.
// Omitted operations stay unresolved and are reported by CompileHandlerScript.
type HandlerBindingOverride struct {
	LoadCH   *LoadCHState     `json:"loadch,omitempty"`
	Pan      *HandlerPoint    `json:"pan,omitempty"`
	Dialog   *HandlerDialog   `json:"dialog,omitempty"`
	Acting   *HandlerActing   `json:"act,omitempty"`
	Layout   *HandlerLayout   `json:"layout,omitempty"`
	Resource *HandlerResource `json:"resource,omitempty"`
}

// HandlerResource binds a native resource-table handle to an editable asset.
// SFXIndex is optional because LOAD_RES/RELEASE_RES only identify the handle.
type HandlerResource struct {
	ResourceID int  `json:"resource_id"`
	SFXIndex   *int `json:"sfx_index,omitempty"`
}

// HandlerActing is a decoded, editable behavioural transcription.  It never
// stores the original acting-resource bytes.
type HandlerActing struct {
	// Resource is an explicit decoded resource ID from ActingResources.  It
	// avoids duplicating the same editable behavioural transcript at every
	// handler call-site.  Frames remains for a one-off authored transcription.
	Resource *int          `json:"resource,omitempty"`
	Frames   []ActingFrame `json:"frames,omitempty"`
	// TimingOnly records an original call whose decoded targets are all beyond
	// the materialized unit_count. The original still runs the frame scheduler
	// and redraws, but those units are not rendered. It is deliberately
	// call-site-local evidence, never a property of the shared resource.
	TimingOnly bool `json:"timing_only,omitempty"`
}

// ActingResourceSet is a behavioural transcription of one runtime acting
// table.  It intentionally contains decoded frame semantics, never original
// bytes or pointers.  Keys are decimal resource IDs so JSON object ordering
// cannot alter the data.
type ActingResourceSet struct {
	SchemaVersion int                      `json:"schema_version"`
	Resources     map[string][]ActingFrame `json:"resources"`
}

// LoadHandlerBinding reads an address-keyed binding file.  It validates the
// structure only; whether it covers a complete handler is reported by the
// compiler, allowing deliberately incremental RE work.
func LoadHandlerBinding(path string) (*HandlerBinding, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var binding HandlerBinding
	if err := json.Unmarshal(raw, &binding); err != nil {
		return nil, err
	}
	if binding.SchemaVersion != 1 {
		return nil, fmt.Errorf("handler binding %q schema_version=%d, want 1", path, binding.SchemaVersion)
	}
	if binding.HandlerScript == "" {
		return nil, fmt.Errorf("handler binding %q has no handler_script", path)
	}
	if binding.ActingResources != "" {
		setPath := filepath.Join(filepath.Dir(path), binding.ActingResources)
		resources, err := LoadActingResourceSet(setPath)
		if err != nil {
			return nil, fmt.Errorf("handler binding %q acting_resources: %w", path, err)
		}
		binding.acting = resources
	}
	if binding.StoryIndexMap != "" {
		indexPath := filepath.Join(filepath.Dir(path), binding.StoryIndexMap)
		index, err := LoadStoryIndexMap(indexPath)
		if err != nil {
			return nil, fmt.Errorf("handler binding %q story_index_map: %w", path, err)
		}
		binding.storyIndex = index
	}
	for addr, context := range binding.DialogueContexts {
		if addr == "" || context.SourceDAT == "" || context.Script == "" {
			return nil, fmt.Errorf("handler binding %q has invalid dialogue context at %q", path, addr)
		}
		if binding.storyIndex == nil {
			return nil, fmt.Errorf("handler binding %q dialogue context %q lacks story_index_map", path, addr)
		}
	}
	for addr, override := range binding.Overrides {
		if addr == "" || (override.LoadCH == nil && override.Pan == nil && override.Dialog == nil && override.Acting == nil && override.Layout == nil && override.Resource == nil) {
			return nil, fmt.Errorf("handler binding %q has empty override at %q", path, addr)
		}
		if state := override.LoadCH; state != nil && (state.Chapter < 0 || state.Map == "" || state.Roster == "" || state.SlotCount <= 0 || state.Script == "") {
			return nil, fmt.Errorf("handler binding %q has incomplete loadch override at %q", path, addr)
		}
		if acting := override.Acting; acting != nil {
			if acting.Resource != nil && len(acting.Frames) != 0 {
				return nil, fmt.Errorf("handler binding %q act override %q mixes resource and inline frames", path, addr)
			}
			if acting.Resource == nil && len(acting.Frames) == 0 {
				return nil, fmt.Errorf("handler binding %q has empty act override at %q", path, addr)
			}
			if acting.TimingOnly && acting.Resource == nil {
				return nil, fmt.Errorf("handler binding %q act override %q timing_only requires a decoded resource", path, addr)
			}
			if acting.Resource != nil {
				if binding.acting == nil {
					return nil, fmt.Errorf("handler binding %q act override %q has resource without acting_resources", path, addr)
				}
				if _, ok := binding.acting[*acting.Resource]; !ok {
					return nil, fmt.Errorf("handler binding %q act override %q references missing resource %d", path, addr, *acting.Resource)
				}
			}
		}
	}
	if context := binding.RuntimeContext; context != nil {
		if context.SlotCount > 0 && len(context.SlotCounts) > 0 {
			return nil, fmt.Errorf("handler binding %q runtime_context cannot mix slot_count and slot_counts", path)
		}
		if context.MinimumSlotCount() <= 0 {
			return nil, fmt.Errorf("handler binding %q runtime_context needs positive slot_count or slot_counts", path)
		}
		previous := 0
		for _, count := range context.SlotCounts {
			if count <= previous {
				return nil, fmt.Errorf("handler binding %q runtime_context slot_counts must be unique ascending positive integers", path)
			}
			previous = count
		}
		for group, count := range context.SpawnGroups {
			if group < 0 || count <= 0 {
				return nil, fmt.Errorf("handler binding %q runtime_context has invalid spawn group %d count %d", path, group, count)
			}
		}
	}
	return &binding, nil
}

// CompileHandlerBinding is the fail-closed runtime/editor entry point for one
// editable handler binding.  It resolves HandlerScript relative to the
// binding file, then returns every unresolved source operation as an issue;
// callers must not start playback while issues remain.
func CompileHandlerBinding(path string) ([]Beat, []HandlerCompileIssue, error) {
	binding, err := LoadHandlerBinding(path)
	if err != nil {
		return nil, nil, err
	}
	script, err := LoadHandlerScript(filepath.Join(filepath.Dir(path), binding.HandlerScript))
	if err != nil {
		return nil, nil, fmt.Errorf("handler binding %q handler_script: %w", path, err)
	}
	beats, issues := CompileHandlerScript(script, binding.CompilerBindings())
	if len(issues) == 0 && binding.RuntimeContext != nil {
		context := *binding.RuntimeContext
		context.SpawnGroups = cloneIntMap(context.SpawnGroups)
		context.SlotCounts = append([]int(nil), context.SlotCounts...)
		beats = append([]Beat{{Op: "runtime_context", RuntimeContext: &context}}, beats...)
	}
	return beats, issues, nil
}

func cloneIntMap(input map[int]int) map[int]int {
	if input == nil {
		return nil
	}
	output := make(map[int]int, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

// CompilerBindings exposes this binding file to the generic compiler.  A
// missing override returns false, which deliberately becomes a compile issue.
func (binding *HandlerBinding) CompilerBindings() HandlerBindings {
	lookup := func(input HandlerBeat) (HandlerBindingOverride, bool) {
		if binding == nil {
			return HandlerBindingOverride{}, false
		}
		override, ok := binding.Overrides[input.Source.Addr]
		return override, ok
	}
	return HandlerBindings{
		RuntimeContext: binding.RuntimeContext,
		LoadCH: func(input HandlerBeat) (LoadCHState, bool) {
			override, ok := lookup(input)
			if !ok || override.LoadCH == nil {
				return LoadCHState{}, false
			}
			return *override.LoadCH, true
		},
		Pan: func(input HandlerBeat) (HandlerPoint, bool) {
			override, ok := lookup(input)
			if !ok || override.Pan == nil {
				return HandlerPoint{}, false
			}
			return *override.Pan, true
		},
		Dialog: func(input HandlerBeat) (HandlerDialog, bool) {
			override, ok := lookup(input)
			if ok && override.Dialog != nil {
				// Hand-authored overrides retain priority for observations such as
				// the prologue's non-default upper dialogue-box placement.
				return *override.Dialog, true
			}
			return binding.indexedDialog(input)
		},
		Acting: func(input HandlerBeat) ([]ActingFrame, bool) {
			override, ok := lookup(input)
			if !ok || override.Acting == nil {
				return nil, false
			}
			if override.Acting.Resource != nil {
				if input.ActingID == nil || *input.ActingID != *override.Acting.Resource {
					return nil, false
				}
				frames, ok := binding.acting[*override.Acting.Resource]
				if !ok || !override.Acting.TimingOnly {
					return frames, ok
				}
				// Preserve the original frame/mode scheduler but erase writes to
				// inactive unit-array memory. This is the visible/runtime projection
				// proven for map31, not a generic bounds-error escape hatch.
				timing := make([]ActingFrame, len(frames))
				for i, frame := range frames {
					timing[i] = ActingFrame{Beats: frame.Beats, Special: frame.Special}
				}
				return timing, true
			}
			return override.Acting.Frames, true
		},
		Layout: func(input HandlerBeat) (HandlerLayout, bool) {
			override, ok := lookup(input)
			if !ok || override.Layout == nil {
				return HandlerLayout{}, false
			}
			layout := *override.Layout
			layout.Units = append([]HandlerUnitLayout(nil), override.Layout.Units...)
			return layout, true
		},
		Resource: func(input HandlerBeat) (HandlerResource, bool) {
			override, ok := lookup(input)
			if !ok || override.Resource == nil || override.Resource.ResourceID < 0 {
				return HandlerResource{}, false
			}
			return *override.Resource, true
		},
	}
}

// LoadActingResourceSet reads a generated or hand-authored behavioural
// transcript.  It verifies IDs and frame arrays before a handler binding can
// claim to execute them.
func LoadActingResourceSet(path string) (map[int][]ActingFrame, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var set ActingResourceSet
	if err := json.Unmarshal(raw, &set); err != nil {
		return nil, err
	}
	if set.SchemaVersion != 1 || len(set.Resources) == 0 {
		return nil, fmt.Errorf("acting resource set %q needs schema_version=1 and resources", path)
	}
	resources := make(map[int][]ActingFrame, len(set.Resources))
	for key, frames := range set.Resources {
		id, err := strconv.Atoi(key)
		if err != nil || id < 0 || len(frames) == 0 {
			return nil, fmt.Errorf("acting resource set %q has invalid resource %q", path, key)
		}
		if _, exists := resources[id]; exists {
			return nil, fmt.Errorf("acting resource set %q repeats resource %d", path, id)
		}
		resources[id] = frames
	}
	return resources, nil
}

func (binding *HandlerBinding) indexedDialog(input HandlerBeat) (HandlerDialog, bool) {
	if binding == nil || binding.storyIndex == nil {
		return HandlerDialog{}, false
	}
	context, ok := binding.DialogueContexts[input.Source.Addr]
	if !ok {
		return HandlerDialog{}, false
	}
	stringIndex, ok := handlerTextIndex(input.TextIndex)
	if !ok {
		return HandlerDialog{}, false
	}
	targets, ok := binding.storyIndex.Lookup(context.SourceDAT, context.Script, stringIndex)
	if !ok || len(targets) == 0 {
		return HandlerDialog{}, false
	}
	if len(targets) > 1 {
		dialog := HandlerDialog{Segments: make([]HandlerDialogSegment, 0, len(targets))}
		for _, target := range targets {
			sceneIndex := target.SceneIndex
			segment := HandlerDialogSegment{Script: context.Script, SceneIndex: &sceneIndex}
			if target.Scene != nil {
				segment.Scene = *target.Scene
			}
			for _, line := range target.Lines {
				segment.Lines = append(segment.Lines, HandlerDialogLine{Line: line})
			}
			if len(segment.Lines) == 0 {
				return HandlerDialog{}, false
			}
			dialog.Segments = append(dialog.Segments, segment)
		}
		return dialog, true
	}
	target := targets[0]
	sceneIndex := target.SceneIndex
	dialog := HandlerDialog{Script: context.Script, SceneIndex: &sceneIndex}
	if target.Scene != nil {
		dialog.Scene = *target.Scene
	}
	for _, line := range target.Lines {
		dialog.Lines = append(dialog.Lines, HandlerDialogLine{Line: line})
	}
	return dialog, true
}

func handlerTextIndex(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, v >= 0
	case float64:
		if v < 0 || math.Trunc(v) != v || v > math.MaxInt {
			return 0, false
		}
		return int(v), true
	default:
		return 0, false
	}
}
