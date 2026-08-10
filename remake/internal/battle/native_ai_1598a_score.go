package battle

import (
	"encoding/binary"
	"fmt"
)

// NativeAI1598AScoreResult preserves the numeric maximum written to
// [0x53c23]. PositiveWinner is present only when a candidate scored above
// zero. The all-zero command-word local remains unproven and is not invented.
type NativeAI1598AScoreResult struct {
	MaxScore          int
	PositiveWinner    NativeAISpellCandidate
	HasPositiveWinner bool
}

// ScoreNativeAI1598A connects the proven 0x1598a slices without executing a
// command: availability, destination/target groups, 0x15b77 family scoring,
// and the strict positive-score winner comparison.
func ScoreNativeAI1598A(
	w, h int,
	records []byte,
	count, actor, selector int,
	unit *Unit,
	book []NativeCommandRecord,
	baseFlags, terrainMoveCodes, costRow []byte,
	skip func(int) bool,
) (NativeAI1598AScoreResult, error) {
	if unit == nil || count <= 0 || count > 0x100 || actor < 0 || actor >= count ||
		len(records) != count*nativeRecordSize || w <= 0 || h <= 0 ||
		len(baseFlags) != w*h || len(terrainMoveCodes) != w*h ||
		len(costRow) != NativeMovementCostRowSize || (selector != 0 && selector != 1) ||
		len(book) != NativeCommandRecordCount {
		return NativeAI1598AScoreResult{}, fmt.Errorf("native AI 0x1598a inputs are malformed")
	}
	actorRecord := records[actor*nativeRecordSize:]
	for index, value := range unit.NativeCommandMask {
		if actorRecord[0x1a+index] != value {
			return NativeAI1598AScoreResult{}, fmt.Errorf("native AI 0x1598a actor mask disagrees with record")
		}
	}
	if unit.MP < 0 || unit.MP > 0xffff ||
		binary.LittleEndian.Uint16(actorRecord[0x44:0x46]) != uint16(unit.MP) {
		return NativeAI1598AScoreResult{}, fmt.Errorf("native AI 0x1598a actor MP disagrees with record")
	}
	for id, command := range book {
		if command.ID != id {
			return NativeAI1598AScoreResult{}, fmt.Errorf("native AI 0x1598a command book is malformed")
		}
	}
	candidates := make([]NativeAISpellCandidate, 0)
	for _, commandID := range NativeAvailableAIScoredCommandIDs(unit, book) {
		command := book[commandID]
		groups, err := NativeAIScoredCommandCandidateGroups(
			w, h, records, count, actor, selector, command,
			baseFlags, terrainMoveCodes, costRow,
		)
		if err != nil {
			return NativeAI1598AScoreResult{}, err
		}
		scores, err := ScoreNativeAIScoredCommandGroups(
			records, commandID, command.Damage, groups, skip,
		)
		if err != nil {
			return NativeAI1598AScoreResult{}, err
		}
		for _, score := range scores {
			if score.Score < 0 {
				return NativeAI1598AScoreResult{}, fmt.Errorf("native AI 0x1598a score is negative")
			}
			if score.Score == 0 {
				continue
			}
			candidates = append(candidates, NativeAISpellCandidate{
				CommandID:     commandID,
				CommandValue:  command.Damage,
				X:             score.Destination.X,
				Y:             score.Destination.Y,
				Score:         score.Score,
				TargetIndices: append([]byte(nil), score.TargetIndices...),
			})
		}
	}
	winner, ok := SelectNativeAISpellCandidate(candidates)
	if !ok {
		return NativeAI1598AScoreResult{}, nil
	}
	return NativeAI1598AScoreResult{
		MaxScore: winner.Score, PositiveWinner: winner, HasPositiveWinner: true,
	}, nil
}
