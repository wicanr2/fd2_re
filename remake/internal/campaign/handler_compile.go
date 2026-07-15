package campaign

import "fmt"

// HandlerPoint is a remake camera coordinate supplied by a campaign-specific
// mapping.  Handler scripts deliberately retain the original grid coordinate;
// there is no assumed global grid-to-pixel formula.
type HandlerPoint struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Frames int `json:"frames,omitempty"`
}

// HandlerDialog identifies the authored remake line(s) corresponding to one
// original FDTXT table/index lookup.  The mapping is explicit because a single
// FDTXT string may be split into several remake lines.
type HandlerDialog struct {
	Line  int   `json:"line"`
	Count int   `json:"count,omitempty"`
	Upper *bool `json:"upper,omitempty"`
	// Script/Scene record the authored text context selected by the preceding
	// loadch or camera transition.  They are metadata until a handler is run
	// through the scene-loading adapter; preserving them prevents line index 0
	// from being ambiguous across different FDTXT resources.
	Script string `json:"script,omitempty"`
	Scene  string `json:"scene,omitempty"`
	// Lines expands one original FDTXT call into individually authored remake
	// lines.  This is required when one original string contains alternating
	// speakers (and therefore different dialogue-box positions).
	Lines []HandlerDialogLine `json:"lines,omitempty"`
}

// HandlerDialogLine is one runtime dialog beat within a HandlerDialog group.
type HandlerDialogLine struct {
	Line  int   `json:"line"`
	Count int   `json:"count,omitempty"`
	Upper *bool `json:"upper,omitempty"`
}

// HandlerBindings holds only evidence-backed, campaign-specific bridges from
// EXE-level identifiers to runtime data.  Nil or failed lookups are reported
// as issues rather than guessed at.
type HandlerBindings struct {
	// Every resolver receives the full input beat, including source.addr.  This
	// permits explicit per-call-site bindings when an index is reused after a
	// later loadch segment.
	Pan    func(HandlerBeat) (HandlerPoint, bool)
	Dialog func(HandlerBeat) (HandlerDialog, bool)
	Acting func(HandlerBeat) ([]ActingFrame, bool)
}

// HandlerCompileIssue identifies a source operation that was intentionally
// not lowered to a runtime Beat.  The caller can surface these in an editor or
// block playback, but no original operation is silently ignored.
type HandlerCompileIssue struct {
	Beat   int
	Op     string
	Source HandlerSource
	Reason string
}

// CompileHandlerScript lowers the subset of a HandlerScript whose remake
// semantics are proven.  It is reusable for a future campaign: campaign data
// supplies mappings for map geometry, text layout, and acting resources while
// this compiler owns no FD2 chapter-specific constants.
func CompileHandlerScript(script *HandlerScript, bindings HandlerBindings) ([]Beat, []HandlerCompileIssue) {
	if script == nil {
		return nil, []HandlerCompileIssue{{Reason: "nil handler script"}}
	}
	beats := make([]Beat, 0, len(script.Beats))
	issues := make([]HandlerCompileIssue, 0)
	issue := func(i int, input HandlerBeat, reason string) {
		issues = append(issues, HandlerCompileIssue{Beat: i, Op: input.Op, Source: input.Source, Reason: reason})
	}
	for i, input := range script.Beats {
		switch input.Op {
		case "delay":
			if input.Ms == nil {
				issue(i, input, "delay lacks an immediate millisecond value")
				continue
			}
			beats = append(beats, Beat{Op: "delay", Ms: *input.Ms})
		case "bgm":
			if input.Track == nil {
				issue(i, input, "bgm lacks immediate track")
				continue
			}
			if *input.Track == -1 {
				beats = append(beats, Beat{Op: "bgm_stop"})
			} else if *input.Track >= 0 {
				beats = append(beats, Beat{Op: "bgm", Track: fmt.Sprintf("FDMUS_%03d", *input.Track)})
			} else {
				issue(i, input, fmt.Sprintf("unsupported negative BGM track %d", *input.Track))
			}
		case "pan":
			if input.GridX == nil || input.GridY == nil || bindings.Pan == nil {
				issue(i, input, "pan requires an explicit grid-to-camera mapping")
				continue
			}
			p, ok := bindings.Pan(input)
			if !ok {
				issue(i, input, "no camera mapping for original grid coordinate")
				continue
			}
			beats = append(beats, Beat{Op: "pan", X: p.X, Y: p.Y, Frames: p.Frames})
		case "dialog":
			if bindings.Dialog == nil {
				issue(i, input, "dialog requires an explicit FDTXT-to-remake-line mapping")
				continue
			}
			d, ok := bindings.Dialog(input)
			if !ok {
				issue(i, input, "no remake line mapping for original FDTXT lookup")
				continue
			}
			if len(d.Lines) == 0 {
				beats = append(beats, Beat{Op: "dialog", Line: d.Line, Count: d.Count, Upper: d.Upper})
				continue
			}
			for _, line := range d.Lines {
				beats = append(beats, Beat{Op: "dialog", Line: line.Line, Count: line.Count, Upper: line.Upper})
			}
		case "act":
			if input.ActingID == nil || bindings.Acting == nil {
				issue(i, input, "act requires an explicit acting-resource mapping")
				continue
			}
			frames, ok := bindings.Acting(input)
			if !ok {
				issue(i, input, "acting resource has not been decoded/mapped")
				continue
			}
			beats = append(beats, Beat{Op: "act", Acting: frames})
		default:
			issue(i, input, "operation has no proven runtime lowering")
		}
	}
	return beats, issues
}
