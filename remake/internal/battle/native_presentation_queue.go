package battle

import "fmt"

// NativePresentationDigit is one entry emitted by FD2's 0x1e0db numeric
// presentation queue. PositionCode, Target and Digit are intentionally raw;
// the renderer owns their eventual screen meaning.
type NativePresentationDigit struct {
	PositionCode int
	Target       int
	Digit        int
}

// AppendNativePresentationMiss preserves sub_1E1DC's raw descriptor table at
// LE linear 0x5204A. The bytes 74 75 76 76 are hexadecimal, not decimal.
// See docs/data/ida/fd2_command9_player_ai_presentation_ida.txt.
func AppendNativePresentationMiss(queue []NativePresentationDigit, target int) []NativePresentationDigit {
	for slot, descriptor := range [...]int{0x74, 0x75, 0x76, 0x76} {
		queue = append(queue, NativePresentationDigit{
			PositionCode: [...]int{2, 8, 12, 17}[slot], Target: target, Digit: descriptor,
		})
	}
	return queue
}

// AppendNativePresentationDigits reproduces the data-only part of 0x1e0db.
// The native caller performs the camera test before entering the formatter;
// inCamera therefore controls the native no-op path. Digits are right
// aligned in four slots and each non-leading digit receives digitBias-48.
// It does not assign HP/MP/damage/heal semantics or render anything.
func AppendNativePresentationDigits(queue []NativePresentationDigit, value, digitBias, target int, inCamera bool) ([]NativePresentationDigit, error) {
	if digitBias < 0 || digitBias > 0xff {
		return nil, fmt.Errorf("native presentation: malformed digit bias %d", digitBias)
	}
	if target < 0 || target > 0xff {
		return nil, fmt.Errorf("native presentation: malformed target %d", target)
	}
	if !inCamera {
		return append([]NativePresentationDigit(nil), queue...), nil
	}
	digits := []byte(fmt.Sprintf("%d", value))
	if len(digits) > 4 {
		// The native loop still reads v12[0..3] when strlen(value)>3; preserve
		// that fixed-buffer behavior rather than normalizing to a number.
		digits = digits[:4]
	}
	out := append([]NativePresentationDigit(nil), queue...)
	for i := 0; i < 4; i++ {
		digit := 0
		if len(digits) > 3-i {
			digit = int(digits[i-(4-len(digits))]) + digitBias - '0'
		}
		out = append(out, NativePresentationDigit{PositionCode: 5*i + 2, Target: target, Digit: digit})
	}
	return out, nil
}
