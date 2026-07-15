package campaign

import (
	"encoding/json"
	"fmt"
	"os"
)

// HandlerBinding is campaign-authored evidence that connects one immutable
// EXE-handler export to remake coordinates, text lines, and acting frames.
// Overrides are keyed by HandlerBeat.Source.Addr, never by a reused resource
// id, so a later loadch segment cannot accidentally inherit an earlier scene.
type HandlerBinding struct {
	SchemaVersion int                               `json:"schema_version"`
	HandlerScript string                            `json:"handler_script"`
	Overrides     map[string]HandlerBindingOverride `json:"overrides"`
}

// HandlerBindingOverride has at most one field for the matching source op.
// Omitted operations stay unresolved and are reported by CompileHandlerScript.
type HandlerBindingOverride struct {
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
	for addr, override := range binding.Overrides {
		if addr == "" || (override.Pan == nil && override.Dialog == nil && override.Acting == nil) {
			return nil, fmt.Errorf("handler binding %q has empty override at %q", path, addr)
		}
	}
	return &binding, nil
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
		Pan: func(input HandlerBeat) (HandlerPoint, bool) {
			override, ok := lookup(input)
			if !ok || override.Pan == nil {
				return HandlerPoint{}, false
			}
			return *override.Pan, true
		},
		Dialog: func(input HandlerBeat) (HandlerDialog, bool) {
			override, ok := lookup(input)
			if !ok || override.Dialog == nil {
				return HandlerDialog{}, false
			}
			return *override.Dialog, true
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
