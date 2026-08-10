package battle

// NativeAISpellCandidate is the already-resolved raw candidate tuple consumed
// by 0x1598a after command availability and target resolution. CommandValue
// is the command record word at +0; X/Y are caller-provided raw coordinates.
type NativeAISpellCandidate struct {
	CommandID    int
	CommandValue int
	X            int
	Y            int
	Score        int
	// TargetIndices preserves the raw runtime record order emitted by the
	// candidate's 0x14818 target pass.  The command consumer may use the first
	// index as the confirmed cursor while re-running its own effect geometry;
	// it must never infer a target from normalized Camp or distance alone.
	TargetIndices []byte
}

// SelectNativeAISpellCandidate preserves 0x1598a's strict score comparison:
// greater score wins; equal score compares raw command +0; an exact tie keeps
// the first candidate. It does not perform MP, target, status, or UI work.
func SelectNativeAISpellCandidate(candidates []NativeAISpellCandidate) (NativeAISpellCandidate, bool) {
	if len(candidates) == 0 {
		return NativeAISpellCandidate{}, false
	}
	best := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate.Score > best.Score || (candidate.Score == best.Score && candidate.CommandValue > best.CommandValue) {
			best = candidate
		}
	}
	best.TargetIndices = append([]byte(nil), best.TargetIndices...)
	return best, true
}
