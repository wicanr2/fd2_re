package battle

import "fmt"

// NativePhysicalAttackScoreInput preserves the word/byte inputs consumed by
// FD2.EXE 0x14237 after caller-owned terrain adjustment. RawTargetByte8 and
// RawHelper1DEBEResult deliberately retain address-level names until their
// writer/consumer contracts are independently closed.
type NativePhysicalAttackScoreInput struct {
	ActorWord48          int
	ActorWord4A          int
	TargetWord48         int
	TargetWord4A         int
	TargetWord40         int
	RawTargetByte8       byte
	RawHelper1DEBEResult int
}

// NativePhysicalAttackScore is the priority/score pair compared by 0x14237.
type NativePhysicalAttackScore struct {
	Priority int
	Score    int
}

// NativePhysicalAttackCandidate is one destination/target pair in 0x14237's
// nested enumeration order.
type NativePhysicalAttackCandidate struct {
	DestinationX int
	DestinationY int
	TargetIndex  int
	Inputs       NativePhysicalAttackScoreInput
}

// NativePhysicalAttackSelection retains the winning raw pair and its ranking.
type NativePhysicalAttackSelection struct {
	Candidate NativePhysicalAttackCandidate
	Ranking   NativePhysicalAttackScore
}

// ScoreNativePhysicalAttackCandidate preserves 0x14586..0x144D5:
// candidates with actor word48-target word4A <= 2 start at priority zero; a strict
// score>target word40 raises priority from 8 to 18 and doubles the score;
// helper result 1 adds actor word4A-target word48; raw target byte8 zero
// applies the native signed truncation-toward-zero 3/2 multiplier.
func ScoreNativePhysicalAttackCandidate(in NativePhysicalAttackScoreInput) (NativePhysicalAttackScore, bool, error) {
	for name, value := range map[string]int{
		"actor_word48":  in.ActorWord48,
		"actor_word4a":  in.ActorWord4A,
		"target_word48": in.TargetWord48,
		"target_word4a": in.TargetWord4A,
		"target_word40": in.TargetWord40,
	} {
		if value < 0 || value > 0xffff {
			return NativePhysicalAttackScore{}, false, fmt.Errorf("%s outside native u16 range: %d", name, value)
		}
	}

	score := in.ActorWord48 - in.TargetWord4A
	priority := 0
	if score > 2 {
		priority = 8
	}
	if score > in.TargetWord40 {
		score *= 2
		priority = 18
	}
	if in.RawHelper1DEBEResult == 1 {
		score += in.ActorWord4A - in.TargetWord48
	}
	if in.RawTargetByte8 == 0 {
		score = nativeMulThreeHalvesTowardZero(score)
	}
	return NativePhysicalAttackScore{Priority: priority, Score: score}, true, nil
}

// SelectNativePhysicalAttackCandidate preserves 0x144C5..0x144F4: priority is
// compared before score, and an exact tie keeps the earlier enumeration.
func SelectNativePhysicalAttackCandidate(candidates []NativePhysicalAttackCandidate) (NativePhysicalAttackSelection, bool, error) {
	var best NativePhysicalAttackSelection
	found := false
	for i, candidate := range candidates {
		for name, value := range map[string]int{
			"destination_x": candidate.DestinationX,
			"destination_y": candidate.DestinationY,
			"target_index":  candidate.TargetIndex,
		} {
			if value < 0 || value > 0xff {
				return NativePhysicalAttackSelection{}, false, fmt.Errorf("candidate %d %s outside native byte range: %d", i, name, value)
			}
		}
		ranking, accepted, err := ScoreNativePhysicalAttackCandidate(candidate.Inputs)
		if err != nil {
			return NativePhysicalAttackSelection{}, false, fmt.Errorf("candidate %d: %w", i, err)
		}
		if !accepted {
			continue
		}
		// 0x14248／0x1427E 的初值為 score/priority=0/0；priority0
		// 的非正分不能靠「尚未找到候選」繞過原版比較。
		if ranking.Priority > best.Ranking.Priority ||
			(ranking.Priority == best.Ranking.Priority && ranking.Score > best.Ranking.Score) {
			best = NativePhysicalAttackSelection{Candidate: candidate, Ranking: ranking}
			found = true
		}
	}
	return best, found, nil
}

func nativeMulThreeHalvesTowardZero(value int) int {
	if value < 0 {
		return -((-value * 3) / 2)
	}
	return value * 3 / 2
}
