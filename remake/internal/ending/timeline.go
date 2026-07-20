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
	Op     string `json:"op"`
	Source string `json:"source"`
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
	}
	return &timeline, nil
}

// Ready is deliberately false for a recovered-only timeline. It makes a
// future campaign bridge fail closed until every opaque segment has a verified
// renderer/text adapter.
func (t Timeline) Ready() bool { return t.Status == "ready" }
