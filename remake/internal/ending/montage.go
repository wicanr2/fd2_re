package ending

import (
	"encoding/json"
	"fmt"
	"os"
)

// Montage is the evidence-only transcription of the first 0x2c548 party
// cycle. It records native resource/slot selection without pretending that
// the later portrait, input, and battle-renderer branches are playable.
type Montage struct {
	SchemaVersion int             `json:"schema_version"`
	NativeHandler string          `json:"native_handler"`
	Status        string          `json:"status"`
	Allocations   []MontageBuffer `json:"allocations"`
	Resources     []MontageAsset  `json:"resources"`
	PartyCycle    PartyCycleSpec  `json:"party_cycle"`
	Gate          MontageGate     `json:"gate"`
}

type MontageBuffer struct {
	Bytes   int    `json:"bytes"`
	Purpose string `json:"purpose"`
	Source  string `json:"source"`
}

type MontageAsset struct {
	Archive string `json:"archive"`
	Index   int    `json:"index"`
	Source  string `json:"source"`
	Role    string `json:"role"`
}

type PartyCycleSpec struct {
	Source            string           `json:"source"`
	CountGlobal       string           `json:"count_global"`
	UnitBaseGlobal    string           `json:"unit_base_global"`
	UnitStride        int              `json:"unit_stride"`
	SlotSelection     string           `json:"slot_selection"`
	VisualGroupOffset int              `json:"visual_group_offset"`
	FigureArchive     string           `json:"figure_archive"`
	FigureIndices     []string         `json:"figure_indices"`
	FigureRenderer    string           `json:"figure_renderer"`
	FrameRenderer     string           `json:"frame_renderer"`
	InitialFrames     int              `json:"initial_frames"`
	FrameDelayMS      int              `json:"frame_delay_ms"`
	SourceRange       string           `json:"source_range"`
	PortraitText      PortraitTextSpec `json:"portrait_text"`
}

type PortraitTextSpec struct {
	Source             string             `json:"source"`
	PortraitArchive    string             `json:"portrait_archive"`
	PortraitIndex      string             `json:"portrait_index"`
	CurrentTextTable   string             `json:"current_text_table"`
	PermanentTextTable string             `json:"permanent_text_table"`
	Fields             []MontageTextField `json:"fields"`
	GlyphStyle         MontageGlyphStyle  `json:"glyph_style"`
	Input              MontageInput       `json:"input"`
}

type MontageTextField struct {
	Table       string `json:"table"`
	Index       string `json:"index"`
	Destination string `json:"destination"`
	Meaning     string `json:"meaning"`
}

type MontageGlyphStyle struct {
	Stride     int `json:"stride"`
	Foreground int `json:"foreground"`
	Shadow     int `json:"shadow"`
	Background int `json:"background"`
}

type MontageInput struct {
	Poll       string `json:"poll"`
	SkipAction string `json:"skip_action"`
	Source     string `json:"source"`
}

type MontageGate struct {
	Source string `json:"source"`
	Reason string `json:"reason"`
}

func LoadMontage(path string) (*Montage, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Montage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if m.SchemaVersion != 1 || m.NativeHandler != "0x2c548" || m.Status != "mapped_first_party_cycle_fail_closed" ||
		len(m.Allocations) != 3 || m.Allocations[0].Bytes != 0x1f400 || m.Allocations[1].Bytes != 0xfa00 || m.Allocations[2].Bytes != 0xfa00 ||
		len(m.Resources) != 2 || m.Resources[0].Archive != "TAI.DAT" || m.Resources[0].Index != 3 || m.Resources[0].Role != "raw 10x3 fully-transparent sprite; renderer role unrecovered" || m.Resources[1].Archive != "FDOTHER.DAT" || m.Resources[1].Index != 56 ||
		m.PartyCycle.Source != "0x2c5d7" || m.PartyCycle.CountGlobal != "0x53bfb" || m.PartyCycle.UnitBaseGlobal != "0x53a45" || m.PartyCycle.UnitStride != 0x50 || m.PartyCycle.VisualGroupOffset != 7 ||
		m.PartyCycle.SlotSelection != "i==0?1:i==1?0:i" ||
		m.PartyCycle.FigureArchive != "FIGANI.DAT" || len(m.PartyCycle.FigureIndices) != 2 || m.PartyCycle.FigureIndices[0] != "group*3+1" || m.PartyCycle.FigureIndices[1] != "group*3" ||
		m.PartyCycle.FigureRenderer != "0x29164" || m.PartyCycle.FrameRenderer != "0x2b9a1" || m.PartyCycle.InitialFrames != 20 || m.PartyCycle.FrameDelayMS != 1 || m.PartyCycle.SourceRange != "0x2c5e3..0x2c9a9" ||
		m.PartyCycle.PortraitText.Source != "0x2c7a4..0x2c967" || m.PartyCycle.PortraitText.PortraitArchive != "DATO.DAT" || m.PartyCycle.PortraitText.PortraitIndex != "unit[+7]" || m.PartyCycle.PortraitText.CurrentTextTable != "FDTXT_031" || m.PartyCycle.PortraitText.PermanentTextTable != "FDTXT_000" || len(m.PartyCycle.PortraitText.Fields) != 5 ||
		m.PartyCycle.PortraitText.Fields[0] != (MontageTextField{Table: "current", Index: "10", Destination: "staging+0x16e9", Meaning: "name_label"}) || m.PartyCycle.PortraitText.Fields[1] != (MontageTextField{Table: "permanent", Index: "unit[+8]+1", Destination: "staging+0x171b", Meaning: "character_name"}) || m.PartyCycle.PortraitText.Fields[2] != (MontageTextField{Table: "current", Index: "11", Destination: "staging+0x2fe9", Meaning: "class_label"}) || m.PartyCycle.PortraitText.Fields[3] != (MontageTextField{Table: "permanent", Index: "unit[+0x20]+0x96", Destination: "staging+0x301b", Meaning: "class_name"}) || m.PartyCycle.PortraitText.Fields[4] != (MontageTextField{Table: "current", Index: "unit[+8]+0x0c|45", Destination: "staging+0x7d08", Meaning: "epilogue"}) ||
		m.PartyCycle.PortraitText.GlyphStyle != (MontageGlyphStyle{Stride: 320, Foreground: 0xcd, Shadow: 0x4c, Background: 0}) || m.PartyCycle.PortraitText.Input != (MontageInput{Poll: "0x10620", SkipAction: "outer_counter=1;0x4e031", Source: "0x2c950..0x2c961"}) ||
		m.Gate.Source != "0x2c5e3" || m.Gate.Reason == "" {
		return nil, fmt.Errorf("ending montage %q is incomplete or unsupported", path)
	}
	return &m, nil
}

// PartyCyclePlan is the exact recoverable selection portion of the native
// loop. It iterates loop indexes descending, but the native first two unit
// slots are deliberately swapped (i=0→slot1, i=1→slot0). It uses the visual
// group at unit+7 to derive these two FIGANI resource ids.
// It intentionally contains no UI or renderer approximation.
type PartyCyclePlan struct {
	LoopIndex       int
	UnitSlot        int
	VisualGroup     int
	PrimaryFIGANI   int
	SecondaryFIGANI int
	Frames          int
	FrameDelayMS    int
}

func (m Montage) PlanPartyCycle(groups []byte) ([]PartyCyclePlan, error) {
	if m.Status != "mapped_first_party_cycle_fail_closed" || len(groups) < 2 {
		return nil, fmt.Errorf("ending: unavailable montage party cycle")
	}
	plans := make([]PartyCyclePlan, 0, len(groups))
	for i := len(groups) - 1; i >= 0; i-- {
		slot := i
		if i == 0 {
			slot = 1
		} else if i == 1 {
			slot = 0
		}
		group := int(groups[slot])
		plans = append(plans, PartyCyclePlan{LoopIndex: i, UnitSlot: slot, VisualGroup: group, PrimaryFIGANI: group*3 + 1, SecondaryFIGANI: group * 3, Frames: m.PartyCycle.InitialFrames, FrameDelayMS: m.PartyCycle.FrameDelayMS})
	}
	return plans, nil
}
