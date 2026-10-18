package campaign

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

const NativeGoldRollDelayMilliseconds = 10

// NativeGoldRollWindow records the literal strip row selected for each digit.
// Active limits indexed writes to the digits changed by this pass; Rows also
// records stationary digits so another renderer never infers them from pixels.
type NativeGoldRollWindow struct {
	Rows   [8]int
	Active [8]bool
}

func PlanNativeGoldRoll(oldGold, newGold int, credit bool) ([]NativeGoldRollWindow, error) {
	if oldGold < 0 || oldGold > 99_999_999 || newGold < 0 || newGold > 99_999_999 ||
		(credit && newGold < oldGold) || (!credit && newGold > oldGold) {
		return nil, errors.New("campaign: invalid native gold roll plan")
	}
	oldText, newText := fmt.Sprintf("%08d", oldGold), fmt.Sprintf("%08d", newGold)
	var current, target [8]int
	for i := range current {
		current[i], target[i] = int(oldText[i]-'0'), int(newText[i]-'0')
	}
	var windows []NativeGoldRollWindow
	for {
		var active [8]bool
		changed := false
		for digit := range current {
			active[digit] = current[digit] != target[digit]
			if active[digit] {
				changed = true
				if !credit {
					current[digit] = (current[digit] + 9) % 10
				}
			}
		}
		if !changed {
			return windows, nil
		}
		for phase := 0; phase < 9; phase++ {
			window := NativeGoldRollWindow{Active: active}
			for digit := range current {
				window.Rows[digit] = current[digit] * 9
				if active[digit] {
					if credit {
						window.Rows[digit] += phase + 1
					} else {
						window.Rows[digit] += 8 - phase
					}
				}
			}
			windows = append(windows, window)
		}
		if credit {
			for digit := range current {
				if active[digit] {
					current[digit] = (current[digit] + 1) % 10
				}
			}
		}
	}
}

func composeNativeGoldRoll(background []byte, strip fdother.RawCell, windows []NativeGoldRollWindow) [][]byte {
	staged := background
	var frames [][]byte
	for _, window := range windows {
		frame := append([]byte(nil), staged...)
		for digit, active := range window.Active {
			if !active {
				continue
			}
			for row := 0; row < 9; row++ {
				src := (window.Rows[digit] + row) * strip.Width
				dst := NativeShopGoldRollOffset + digit*6 + row*NativeShopWidth
				copy(frame[dst:dst+6], strip.Pixels[src:src+6])
			}
		}
		frames = append(frames, frame)
		staged = frame
	}
	return frames
}

// ComposeNativeGoldCreditFrames reproduces 0x2d3ff's eight-digit upward
// odometer. The native routine commits the balance before drawing these
// frames. Each differing digit starts at phase one, advances through nine
// overlapping 6x9 windows, then increments with decimal wrap.
func ComposeNativeGoldCreditFrames(
	background []byte,
	strip fdother.RawCell,
	oldGold, amount int,
) (frames [][]byte, newGold int, err error) {
	if len(background) != NativeShopWidth*NativeShopHeight ||
		strip.Width != 6 || strip.Height != 99 ||
		len(strip.Pixels) != strip.Width*strip.Height ||
		oldGold < 0 || oldGold > 99_999_999 ||
		amount < 0 || amount > 99_999_999-oldGold {
		return nil, oldGold, errors.New(
			"campaign: invalid native gold credit state",
		)
	}
	newGold = oldGold + amount
	windows, _ := PlanNativeGoldRoll(oldGold, newGold, true)
	frames = composeNativeGoldRoll(background, strip, windows)
	return frames, newGold, nil
}

// ComposeNativeGoldDebitFrames reproduces 0x2d516's eight-digit downward
// odometer. The native routine commits the balance before drawing these
// frames, then advances every differing digit independently through nine
// overlapping 6x9 windows of FDOTHER's 6x99 roll strip.
func ComposeNativeGoldDebitFrames(
	background []byte,
	strip fdother.RawCell,
	oldGold, amount int,
) (frames [][]byte, newGold int, err error) {
	if len(background) != NativeShopWidth*NativeShopHeight ||
		strip.Width != 6 || strip.Height != 99 ||
		len(strip.Pixels) != strip.Width*strip.Height ||
		oldGold < 0 || oldGold > 99_999_999 ||
		amount < 0 || amount > oldGold {
		return nil, oldGold, errors.New(
			"campaign: invalid native gold debit state",
		)
	}
	newGold = oldGold - amount
	windows, _ := PlanNativeGoldRoll(oldGold, newGold, false)
	frames = composeNativeGoldRoll(background, strip, windows)
	return frames, newGold, nil
}
