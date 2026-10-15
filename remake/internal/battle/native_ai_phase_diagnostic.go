package battle

import (
	"fmt"
	"math"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// NativeAIPhaseZeroInput supplies the caller-owned movement row and optional
// 0x1f183 gate for one eligible selector-zero record. UnitIndex is a raw
// runtime roster index, not a persistent-party or character identity.
type NativeAIPhaseZeroInput struct {
	UnitIndex int
	CostRow   []byte
	Skip      func(unitIndex int) bool
}

// NativeAIPhaseZeroDiagnostic preserves both score-producer results computed
// in the original 0x1d8ba order. It is evidence/diagnostic state only and does
// not authorize execution of either winner.
type NativeAIPhaseZeroDiagnostic struct {
	UnitIndex          int
	CommandMaskResult  NativeAI1598AScoreResult
	ItemCommandResult  NativeAI1567EScoreResult
	PhaseGateScorePair fdother.NativePhasePreselectionScores
}

// BuildNativeAIPhaseDiagnosticPlan connects the proven, side-effect-free part
// of 0x1d8ba:
//
//	eligible raw +6==0 record
//	-> 0x1598a(unit,0)
//	-> 0x1567e(unit,0)
//	-> signed [0x53c23] >= 6 || [0x53c33] >= 6
//
// It also returns the three-pass scan plan, but never invokes 0x13a9f,
// [0x51a8f]/[0x51b91], the chapter handler table, or [0x53ecc] early exit.
// Every eligible selector-zero record must have exactly one explicit input;
// missing or extra provenance fails closed.
func BuildNativeAIPhaseDiagnosticPlan(
	st *State,
	book []NativeCommandRecord,
	itemRows []byte,
	inputs []NativeAIPhaseZeroInput,
) (fdother.NativePhaseUnitScanPlan, []NativeAIPhaseZeroDiagnostic, error) {
	var empty fdother.NativePhaseUnitScanPlan
	if st == nil || len(st.Units) == 0 || len(st.Units) > 0x100 {
		return empty, nil, fmt.Errorf("native AI phase state is malformed")
	}
	records, err := NativeAIScoringRecords(st.Units)
	if err != nil {
		return empty, nil, err
	}
	baseFlags, err := NativeCompositionBaseFlags(
		st.W, st.H, st.NativeCompositionEventBytes,
	)
	if err != nil {
		return empty, nil, err
	}
	count := len(st.Units)
	zeroScores := make([]fdother.NativePhasePreselectionScores, count)
	admission, err := fdother.PlanNativePhaseUnitScans(records, count, zeroScores)
	if err != nil {
		return empty, nil, err
	}

	inputByUnit := make(map[int]NativeAIPhaseZeroInput, len(inputs))
	for _, input := range inputs {
		if input.UnitIndex < 0 || input.UnitIndex >= count {
			return empty, nil, fmt.Errorf(
				"native AI phase input unit %d is out of bounds", input.UnitIndex,
			)
		}
		if _, exists := inputByUnit[input.UnitIndex]; exists {
			return empty, nil, fmt.Errorf(
				"native AI phase input unit %d is duplicated", input.UnitIndex,
			)
		}
		inputByUnit[input.UnitIndex] = input
	}

	eligible := make(map[int]bool, len(admission.ZeroPreselection))
	diagnostics := make(
		[]NativeAIPhaseZeroDiagnostic, 0, len(admission.ZeroPreselection),
	)
	for _, candidate := range admission.ZeroPreselection {
		unitIndex := candidate.UnitIndex
		eligible[unitIndex] = true
		input, ok := inputByUnit[unitIndex]
		if !ok {
			return empty, nil, fmt.Errorf(
				"native AI phase input for eligible unit %d is unavailable", unitIndex,
			)
		}
		commandMask, err := ScoreNativeAI1598A(
			st.W, st.H, records, count, unitIndex, 0, st.Units[unitIndex],
			book, baseFlags, st.NativeTerrainMoveCodes,
			input.CostRow, input.Skip,
		)
		if err != nil {
			return empty, nil, err
		}
		itemCommand, err := ScoreNativeAI1567E(
			st.W, st.H, records, count, unitIndex, 0,
			itemRows, book, baseFlags,
		)
		if err != nil {
			return empty, nil, err
		}
		if commandMask.MaxScore > math.MaxInt32 ||
			itemCommand.MaxScore > math.MaxInt32 {
			return empty, nil, fmt.Errorf(
				"native AI phase score for unit %d exceeds signed dword", unitIndex,
			)
		}
		pair := fdother.NativePhasePreselectionScores{
			SpellCandidateScore53C23: int32(commandMask.MaxScore),
			ItemCommandScore53C33:    int32(itemCommand.MaxScore),
		}
		zeroScores[unitIndex] = pair
		diagnostics = append(diagnostics, NativeAIPhaseZeroDiagnostic{
			UnitIndex: unitIndex, CommandMaskResult: commandMask,
			ItemCommandResult: itemCommand, PhaseGateScorePair: pair,
		})
	}
	for unitIndex := range inputByUnit {
		if !eligible[unitIndex] {
			return empty, nil, fmt.Errorf(
				"native AI phase input unit %d is not selector-zero eligible", unitIndex,
			)
		}
	}
	plan, err := fdother.PlanNativePhaseUnitScans(records, count, zeroScores)
	if err != nil {
		return empty, nil, err
	}
	return plan, diagnostics, nil
}
