package ending

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
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
	Source              string                  `json:"source"`
	CountGlobal         string                  `json:"count_global"`
	UnitBaseGlobal      string                  `json:"unit_base_global"`
	UnitStride          int                     `json:"unit_stride"`
	SlotSelection       string                  `json:"slot_selection"`
	VisualGroupOffset   int                     `json:"visual_group_offset"`
	FigureArchive       string                  `json:"figure_archive"`
	FigureIndices       []string                `json:"figure_indices"`
	FigureRenderer      string                  `json:"figure_renderer"`
	FrameRenderer       string                  `json:"frame_renderer"`
	InitialFrames       int                     `json:"initial_frames"`
	FrameDelayTicks     int                     `json:"frame_delay_ticks"`
	SourceRange         string                  `json:"source_range"`
	PortraitText        PortraitTextSpec        `json:"portrait_text"`
	DialogueFrameLayout DialogueFrameLayoutSpec `json:"dialogue_frame_layout"`
	FigureFade          FigureFadeSpec          `json:"figure_fade"`
	MirrorBranch        MirrorBranchSpec        `json:"mirror_branch"`
}

type FigureFadeSpec struct {
	Source             string          `json:"source"`
	UnitSideOffset     int             `json:"unit_side_offset"`
	RequiredUnitSide   int             `json:"required_unit_side"`
	WorkStride         int             `json:"work_stride"`
	ViewportWidth      int             `json:"viewport_width"`
	ViewportHeight     int             `json:"viewport_height"`
	Platform           FigureFadeAsset `json:"platform"`
	Figure             FigureFadeAsset `json:"figure"`
	StageStart         int             `json:"stage_start"`
	StageEnd           int             `json:"stage_end"`
	StageShiftBytes    int             `json:"stage_shift_bytes"`
	PaletteFormula     string          `json:"palette_delta_formula"`
	RestoreBytes       int             `json:"restore_buffer_bytes"`
	RestoreCopy        string          `json:"restore_copy"`
	PostFigureSnapshot string          `json:"post_figure_snapshot"`
}

type FigureFadeAsset struct {
	Resource string `json:"resource"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Effect   string `json:"effect"`
	Frame    int    `json:"frame"`
}

// MirrorBranchSpec is the separately recovered unit[+6]==0 path in
// 0x29164. It is metadata only: loading it does not enable a renderer.
type MirrorBranchSpec struct {
	Source              string          `json:"source"`
	RequiredUnitSide    int             `json:"required_unit_side"`
	SideFlagArgument    string          `json:"side_flag_argument"`
	WorkStride          int             `json:"work_stride"`
	StageStart          int             `json:"stage_start"`
	StageEnd            int             `json:"stage_end"`
	StageShiftBytes     int             `json:"stage_shift_bytes"`
	PaletteDeltaFormula string          `json:"palette_delta_formula"`
	PrimarySource       string          `json:"primary_source"`
	SecondarySource     string          `json:"secondary_source"`
	SecondaryWhen       string          `json:"secondary_when"`
	PlatformWhen        string          `json:"platform_when"`
	Platform            FigureFadeAsset `json:"platform"`
	Renderer            string          `json:"renderer"`
}

type PortraitTextSpec struct {
	Source             string             `json:"source"`
	PortraitArchive    string             `json:"portrait_archive"`
	PortraitIndex      string             `json:"portrait_index"`
	CurrentTextTable   string             `json:"current_text_table"`
	PermanentTextTable string             `json:"permanent_text_table"`
	Fields             []MontageTextField `json:"fields"`
	GlyphStyle         MontageGlyphStyle  `json:"glyph_style"`
	Cycle              PortraitCycleSpec  `json:"cycle"`
	Input              MontageInput       `json:"input"`
}

type PortraitCycleSpec struct {
	Source                  string `json:"source"`
	Destination             int    `json:"destination"`
	NormalFrame             int    `json:"normal_frame"`
	MouthFrame              int    `json:"mouth_frame"`
	InitialCountdown        int    `json:"initial_countdown"`
	ResetFormula            string `json:"reset_formula"`
	Decrement               string `json:"decrement"`
	MouthCondition          string `json:"mouth_condition"`
	RegularIterations       int    `json:"regular_iterations"`
	FinalLoopIndex          int    `json:"final_loop_index"`
	FinalIterations         int    `json:"final_iterations"`
	SpecialEpilogueFromTick int    `json:"special_epilogue_from_tick"`
	WaitTicks               int    `json:"wait_ticks"`
}

type MontageTextField struct {
	Table       string `json:"table"`
	Index       string `json:"index"`
	Destination string `json:"destination"`
	Meaning     string `json:"meaning"`
}

// PortraitTextPlan is the verified index/destination dataflow emitted by
// 0x2c7a4..0x2c967 for one party unit. It intentionally stops before glyph
// rendering; EDI is the native outer portrait-loop counter, not a unit byte.
type PortraitTextPlan struct {
	PortraitID    int
	NameLabel     TextPlacement
	CharacterName TextPlacement
	ClassLabel    TextPlacement
	ClassName     TextPlacement
	Epilogue      TextPlacement
}

type TextPlacement struct {
	Table       string
	Index       int
	Destination int
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

type DialogueFrameLayoutSpec struct {
	Source      string `json:"source"`
	Archive     string `json:"archive"`
	Resource    int    `json:"resource"`
	Destination string `json:"destination"`
	Stride      int    `json:"stride"`
	Arg8        int    `json:"arg8"`
	ArgC        int    `json:"argC"`
	Arg10       int    `json:"arg10"`
	Arg14       int    `json:"arg14"`
	Role        string `json:"role"`
}

type DialogueFramePlacement struct {
	ResourceIndex   int
	DestinationByte int
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
		m.PartyCycle.FigureRenderer != "0x29164" || m.PartyCycle.FrameRenderer != "0x2b9a1" || m.PartyCycle.InitialFrames != 20 || m.PartyCycle.FrameDelayTicks != 1 || m.PartyCycle.SourceRange != "0x2c5e3..0x2c9a9" ||
		m.PartyCycle.PortraitText.Source != "0x2c7a4..0x2c967" || m.PartyCycle.PortraitText.PortraitArchive != "DATO.DAT" || m.PartyCycle.PortraitText.PortraitIndex != "unit[+7]" || m.PartyCycle.PortraitText.CurrentTextTable != "FDTXT_031" || m.PartyCycle.PortraitText.PermanentTextTable != "FDTXT_000" || len(m.PartyCycle.PortraitText.Fields) != 5 ||
		m.PartyCycle.PortraitText.Fields[0] != (MontageTextField{Table: "current", Index: "10", Destination: "staging+0x16e9", Meaning: "name_label"}) || m.PartyCycle.PortraitText.Fields[1] != (MontageTextField{Table: "permanent", Index: "unit[+8]+1", Destination: "staging+0x171b", Meaning: "character_name"}) || m.PartyCycle.PortraitText.Fields[2] != (MontageTextField{Table: "current", Index: "11", Destination: "staging+0x2fe9", Meaning: "class_label"}) || m.PartyCycle.PortraitText.Fields[3] != (MontageTextField{Table: "permanent", Index: "unit[+0x20]+0x96", Destination: "staging+0x301b", Meaning: "class_name"}) || m.PartyCycle.PortraitText.Fields[4] != (MontageTextField{Table: "current", Index: "edi<0xdc ? unit[+8]+0x0c : 0x2d", Destination: "staging+0x7d08", Meaning: "epilogue"}) ||
		m.PartyCycle.DialogueFrameLayout != (DialogueFrameLayoutSpec{Source: "0x2c773->0x168b6", Archive: "FDOTHER.DAT", Resource: 5, Destination: "portrait_restore_buffer_C", Stride: 320, Arg8: 5, ArgC: 7, Arg10: 5, Arg14: 5, Role: "native dialogue-frame/grid staging; later DATO portrait paste is 0x4e8af"}) ||
		m.PartyCycle.PortraitText.GlyphStyle != (MontageGlyphStyle{Stride: 320, Foreground: 0xcd, Shadow: 0x4c, Background: 0}) ||
		m.PartyCycle.PortraitText.Cycle != (PortraitCycleSpec{Source: "0x2c430,0x2c43f,0x2c7cf..0x2c9a9", Destination: 0x0c88, NormalFrame: 0, MouthFrame: 3, InitialCountdown: 0, ResetFormula: "(random_byte&0x1f)+0x28", Decrement: "only_when_nonzero_before_frame_selection", MouthCondition: "countdown<2", RegularIterations: 0xdc, FinalLoopIndex: 0, FinalIterations: 0x1b8, SpecialEpilogueFromTick: 0xdc, WaitTicks: 1}) ||
		m.PartyCycle.PortraitText.Input != (MontageInput{Poll: "0x10620", SkipAction: "outer_counter=1;0x4e031", Source: "0x2c950..0x2c961"}) ||
		m.PartyCycle.FigureFade.Source != "0x291197..0x29258" || m.PartyCycle.FigureFade.UnitSideOffset != 6 || m.PartyCycle.FigureFade.RequiredUnitSide != 1 || m.PartyCycle.FigureFade.WorkStride != 640 || m.PartyCycle.FigureFade.ViewportWidth != 320 || m.PartyCycle.FigureFade.ViewportHeight != 200 || m.PartyCycle.FigureFade.Platform != (FigureFadeAsset{Resource: "TAI.DAT#3", X: 164, Y: 157, Effect: "transparent_noop"}) || m.PartyCycle.FigureFade.Figure != (FigureFadeAsset{Resource: "secondary_figani", Frame: 0}) || m.PartyCycle.FigureFade.StageStart != 8 || m.PartyCycle.FigureFade.StageEnd != 0 || m.PartyCycle.FigureFade.StageShiftBytes != 10 || m.PartyCycle.FigureFade.PaletteFormula != "stage*6" || m.PartyCycle.FigureFade.RestoreBytes != 0xfa00 || m.PartyCycle.FigureFade.RestoreCopy != "B->A(dstStride640,srcStride320,width320,rows200)" || m.PartyCycle.FigureFade.PostFigureSnapshot != "memmove(C,B,64000)" ||
		m.PartyCycle.MirrorBranch != (MirrorBranchSpec{Source: "0x2927e..0x29357", RequiredUnitSide: 0, SideFlagArgument: "arg4", WorkStride: 640, StageStart: 8, StageEnd: 0, StageShiftBytes: 10, PaletteDeltaFormula: "stage*6", PrimarySource: "staging+0x140-stage*10", SecondarySource: "staging+0x140", SecondaryWhen: "arg4==0", PlatformWhen: "arg4==0", Platform: FigureFadeAsset{Resource: "TAI.DAT#3", X: 164, Y: 157, Effect: "transparent_noop"}, Renderer: "0x29164"}) ||
		m.Gate.Source != "0x2c5e3" || m.Gate.Reason == "" {
		return nil, fmt.Errorf("ending montage %q is incomplete or unsupported", path)
	}
	return &m, nil
}

// PlanPortraitText materializes the five native 0x2c548 text calls without
// interpreting their glyph/control-code renderer. edi is the outer party
// portrait-loop counter observed at 0x2c8f7; the special 0x2d index is chosen
// only once edi reaches 0xdc.
func (m Montage) PlanPortraitText(unit []byte, edi int) (PortraitTextPlan, error) {
	if m.Status != "mapped_first_party_cycle_fail_closed" || len(unit) < 0x21 || edi < 0 {
		return PortraitTextPlan{}, fmt.Errorf("ending: invalid portrait text inputs")
	}
	epilogue := int(unit[8]) + 0x0c
	if edi >= 0xdc {
		epilogue = 0x2d
	}
	return PortraitTextPlan{
		PortraitID:    int(unit[7]),
		NameLabel:     TextPlacement{Table: "current", Index: 10, Destination: 0x16e9},
		CharacterName: TextPlacement{Table: "permanent", Index: int(unit[8]) + 1, Destination: 0x171b},
		ClassLabel:    TextPlacement{Table: "current", Index: 11, Destination: 0x2fe9},
		ClassName:     TextPlacement{Table: "permanent", Index: int(unit[0x20]) + 0x96, Destination: 0x301b},
		Epilogue:      TextPlacement{Table: "current", Index: epilogue, Destination: 0x7d08},
	}, nil
}

// PlanDialogueFrameGrid transcribes every sub_1685c call made by 0x168b6 for the
// recovered (stride=320, arg8=5, argC=7, arg10=5, arg14=5) invocation. The
// result is raw resource-index/destination data; it intentionally does not
// assign names such as frame, border, or portrait to any cell.
func (m Montage) PlanDialogueFrameGrid() ([]DialogueFramePlacement, error) {
	d := m.PartyCycle.DialogueFrameLayout
	if m.Status != "mapped_first_party_cycle_fail_closed" || d.Stride != 320 || d.Arg8 != 5 || d.ArgC != 7 || d.Arg10 != 5 || d.Arg14 != 5 {
		return nil, fmt.Errorf("ending: unavailable dialogue frame grid")
	}
	raw, err := fdother.PlanNativeDialogueFrameGrid(d.Stride, d.Arg8, d.ArgC, d.Arg10, d.Arg14)
	if err != nil {
		return nil, err
	}
	placements := make([]DialogueFramePlacement, len(raw))
	for i, placement := range raw {
		placements[i] = DialogueFramePlacement{
			ResourceIndex:   placement.ResourceIndex,
			DestinationByte: placement.DestinationByte,
		}
	}
	return placements, nil
}

// FigureFadePass is one fully evidenced 0x29164 non-mirrored presentation.
// SourceOffset is a byte offset into the 640-stride work surface. The caller
// owns the un-recovered restore/copy sequencing; this plan never substitutes
// an RGBA or generic battle fade.
type FigureFadePass struct {
	Stage, SourceOffset, PaletteDelta int
}

func (m Montage) PlanFigureFade(unitSide int) ([]FigureFadePass, error) {
	s := m.PartyCycle.FigureFade
	// 0x2918f tests unit[+6] for zero.  The JSON contract retains
	// required_unit_side=1 as its first observed representative, but it is not
	// an equality predicate: every nonzero raw value takes this branch.
	if m.Status != "mapped_first_party_cycle_fail_closed" || s.RequiredUnitSide != 1 || unitSide == 0 || s.StageStart != 8 || s.StageEnd != 0 || s.StageShiftBytes != 10 || s.PaletteFormula != "stage*6" {
		return nil, fmt.Errorf("ending: unavailable or mirrored figure fade")
	}
	passes := make([]FigureFadePass, 0, s.StageStart-s.StageEnd+1)
	for stage := s.StageStart; stage >= s.StageEnd; stage-- {
		passes = append(passes, FigureFadePass{Stage: stage, SourceOffset: stage * s.StageShiftBytes, PaletteDelta: stage * 6})
	}
	return passes, nil
}

// MirrorFigureFadePass is the exact scheduling plan for 0x29164's
// unit[+6]==0 branch. It carries no pixels and therefore cannot be used as a
// renderer substitute; it only makes the recovered source offsets and the
// arg4-gated draws executable by a future indexed adapter.
type MirrorFigureFadePass struct {
	Stage               int
	PrimarySourceOffset int
	PaletteDelta        int
	DrawSecondary       bool
	DrawPlatform        bool
}

func (m Montage) PlanMirrorFigureFade(unitSide, sideFlag int) ([]MirrorFigureFadePass, error) {
	s := m.PartyCycle.MirrorBranch
	if m.Status != "mapped_first_party_cycle_fail_closed" || unitSide != s.RequiredUnitSide ||
		s.StageStart != 8 || s.StageEnd != 0 || s.StageShiftBytes != 10 || s.PaletteDeltaFormula != "stage*6" ||
		s.SideFlagArgument != "arg4" || sideFlag < 0 {
		return nil, fmt.Errorf("ending: unavailable mirror figure fade")
	}
	passes := make([]MirrorFigureFadePass, 0, s.StageStart-s.StageEnd+1)
	for stage := s.StageStart; stage >= s.StageEnd; stage-- {
		passes = append(passes, MirrorFigureFadePass{
			Stage: stage, PrimarySourceOffset: 0x140 - stage*s.StageShiftBytes,
			PaletteDelta: stage * 6, DrawSecondary: sideFlag == 0, DrawPlatform: sideFlag == 0,
		})
	}
	return passes, nil
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
	FrameDelayTicks int
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
		plans = append(plans, PartyCyclePlan{LoopIndex: i, UnitSlot: slot, VisualGroup: group, PrimaryFIGANI: group*3 + 1, SecondaryFIGANI: group * 3, Frames: m.PartyCycle.InitialFrames, FrameDelayTicks: m.PartyCycle.FrameDelayTicks})
	}
	return plans, nil
}
