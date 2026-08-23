package battle

// NativeCompoundStep is an evidence-only step in the 0x27fc9 wrapper.
// Addresses and raw offsets are retained deliberately; this plan does not
// execute a callee or assign a gameplay/effect name.
type NativeCompoundStep struct {
	Callee       uint32
	CommandID    int
	MarkerOffset int
	Amount       int
}

// NativeCompoundCommandPlan returns the verified helper order for IDs 32..35.
// It remains a data-only evidence bridge: bounded player transactions for
// IDs33..35 consume it or the same order elsewhere, while target selection,
// presentation and mutation stay outside this plan itself.
func NativeCompoundCommandPlan(commandID int) ([]NativeCompoundStep, bool) {
	var steps []NativeCompoundStep
	switch commandID {
	case 32:
		steps = []NativeCompoundStep{{Callee: 0x2111a, CommandID: 32}}
	case 33:
		steps = []NativeCompoundStep{
			// Callee==0 denotes the wrapper's direct byte clear, not a call.
			{CommandID: 33, MarkerOffset: 0x25},
			{CommandID: 33, MarkerOffset: 0x26},
			{CommandID: 33, MarkerOffset: 0x27},
			{Callee: 0x211a4, CommandID: 33, Amount: 0x320},
		}
	case 34:
		steps = []NativeCompoundStep{
			{Callee: 0x22721, CommandID: 17, MarkerOffset: 0x22},
			{Callee: 0x22866, CommandID: 18, MarkerOffset: 0x23},
			{Callee: 0x22997, CommandID: 19, MarkerOffset: 0x24},
		}
	case 35:
		steps = []NativeCompoundStep{
			{Callee: 0x22d1b, CommandID: 26, MarkerOffset: 0x25},
			{Callee: 0x22d1b, CommandID: 22, MarkerOffset: 0x27},
			{Callee: 0x22d1b, CommandID: 27, MarkerOffset: 0x26},
		}
	default:
		return nil, false
	}
	return steps, true
}
