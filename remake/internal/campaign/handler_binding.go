package campaign

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

// HandlerBinding is campaign-authored evidence that connects one immutable
// EXE-handler export to remake coordinates, text lines, and acting frames.
// Overrides are keyed by HandlerBeat.Source.Addr, never by a reused resource
// id, so a later loadch segment cannot accidentally inherit an earlier scene.
type HandlerBinding struct {
	SchemaVersion    int                               `json:"schema_version"`
	HandlerScript    string                            `json:"handler_script"`
	StoryIndexMap    string                            `json:"story_index_map,omitempty"`
	DialogueContexts map[string]HandlerDialogueContext `json:"dialogue_contexts,omitempty"`
	Overrides        map[string]HandlerBindingOverride `json:"overrides"`

	storyIndex *StoryIndexMap
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
	LoadCH *LoadCHState   `json:"loadch,omitempty"`
	Pan    *HandlerPoint  `json:"pan,omitempty"`
	Dialog *HandlerDialog `json:"dialog,omitempty"`
	Acting *HandlerActing `json:"act,omitempty"`
}

// HandlerActing is a decoded, editable behavioural transcription.  It never
// stores the original acting-resource bytes.
type HandlerActing struct {
	Frames []ActingFrame `json:"frames"`
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
		if addr == "" || (override.LoadCH == nil && override.Pan == nil && override.Dialog == nil && override.Acting == nil) {
			return nil, fmt.Errorf("handler binding %q has empty override at %q", path, addr)
		}
		if state := override.LoadCH; state != nil && (state.Chapter < 0 || state.Map == "" || state.Roster == "" || state.Script == "") {
			return nil, fmt.Errorf("handler binding %q has incomplete loadch override at %q", path, addr)
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
	return beats, issues, nil
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
			return override.Acting.Frames, true
		},
	}
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
	if !ok || len(targets) != 1 {
		// A string crossing scenes needs a scene-transition adapter; lowering it
		// as one runtime dialog would silently use the wrong current scene.
		return HandlerDialog{}, false
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
