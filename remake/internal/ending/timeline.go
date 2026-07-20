// Package ending loads recovered ending timelines.  The timelines are kept
// separate from campaign handlers because their indexed compositors and native
// text helpers need their own evidence-backed renderer.  Loading never grants
// playback permission: callers must reject Status values other than "ready".
package ending

import (
	"encoding/json"
	"fmt"
	"os"
)

type Timeline struct {
	SchemaVersion int       `json:"schema_version"`
	NativeHandler string    `json:"native_handler"`
	Resource      Resource  `json:"resource"`
	Status        string    `json:"status"`
	Segments      []Segment `json:"segments"`
}

type Resource struct {
	Archive string `json:"archive"`
	Index   int    `json:"index"`
}

// Segment deliberately retains only evidence names and native call arguments.
// It is an editable transcription, not a generic ending scripting language.
type Segment struct {
	Op           string          `json:"op"`
	Source       string          `json:"source"`
	ElseDialogue []DialogueBlock `json:"else_dialogue,omitempty"`
}

// DialogueBlock is a count-aligned FDTXT block recovered from an ending text
// helper. VisualResourceIndex is intentionally not called a portrait until
// archive 0x51a70's resource type is directly established.
type DialogueBlock struct {
	VisualResourceIndex int    `json:"visual_resource_index"`
	SourceDAT           string `json:"source_dat"`
	Script              string `json:"script"`
	StringIndex         int    `json:"string_index"`
	SceneIndex          int    `json:"scene_index"`
	Line                int    `json:"line"`
	Count               int    `json:"count"`
}

func LoadTimeline(path string) (*Timeline, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var timeline Timeline
	if err := json.Unmarshal(raw, &timeline); err != nil {
		return nil, err
	}
	if timeline.SchemaVersion != 1 || timeline.NativeHandler == "" || timeline.Resource.Archive == "" || timeline.Resource.Index < 0 || timeline.Status == "" {
		return nil, fmt.Errorf("ending timeline %q has invalid header", path)
	}
	if len(timeline.Segments) == 0 {
		return nil, fmt.Errorf("ending timeline %q has no recovered segments", path)
	}
	for i, segment := range timeline.Segments {
		if segment.Op == "" || segment.Source == "" {
			return nil, fmt.Errorf("ending timeline %q segment %d is incomplete", path, i)
		}
		for j, block := range segment.ElseDialogue {
			if block.SourceDAT == "" || block.Script == "" || block.StringIndex < 0 || block.SceneIndex < 0 || block.Line < 0 || block.Count <= 0 {
				return nil, fmt.Errorf("ending timeline %q segment %d dialogue %d is incomplete", path, i, j)
			}
		}
	}
	return &timeline, nil
}

// Ready is deliberately false for a recovered-only timeline. It makes a
// future campaign bridge fail closed until every opaque segment has a verified
// renderer/text adapter.
func (t Timeline) Ready() bool { return t.Status == "ready" }
