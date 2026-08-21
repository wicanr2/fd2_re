package campaign

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

const NativeGoldRollDelayMilliseconds = 10

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
	oldText := fmt.Sprintf("%08d", oldGold)
	newText := fmt.Sprintf("%08d", newGold)
	var current, target [8]int
	for i := range current {
		current[i] = int(oldText[i] - '0')
		target[i] = int(newText[i] - '0')
	}

	staged := append([]byte(nil), background...)
	for {
		var rolling [8]bool
		changed := false
		for digit := range current {
			if current[digit] == target[digit] {
				continue
			}
			rolling[digit] = true
			changed = true
		}
		if !changed {
			break
		}
		for phase := 1; phase <= 9; phase++ {
			frame := append([]byte(nil), staged...)
			for digit := range current {
				if !rolling[digit] {
					continue
				}
				startRow := current[digit]*9 + phase
				for row := 0; row < 9; row++ {
					src := (startRow + row) * strip.Width
					dst := NativeShopGoldRollOffset + digit*6 +
						row*NativeShopWidth
					copy(frame[dst:dst+6], strip.Pixels[src:src+6])
				}
			}
			frames = append(frames, frame)
			staged = frame
		}
		for digit := range current {
			if !rolling[digit] {
				continue
			}
			current[digit]++
			if current[digit] == 10 {
				current[digit] = 0
			}
		}
	}
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
	oldText := fmt.Sprintf("%08d", oldGold)
	newText := fmt.Sprintf("%08d", newGold)
	var current, target [8]int
	for i := range current {
		current[i] = int(oldText[i] - '0')
		target[i] = int(newText[i] - '0')
	}

	staged := append([]byte(nil), background...)
	for {
		var rolling [8]bool
		changed := false
		for digit := range current {
			if current[digit] == target[digit] {
				continue
			}
			rolling[digit] = true
			changed = true
			current[digit]--
			if current[digit] < 0 {
				current[digit] = 9
			}
		}
		if !changed {
			break
		}
		for phase := 0; phase < 9; phase++ {
			frame := append([]byte(nil), staged...)
			for digit := range current {
				if !rolling[digit] {
					continue
				}
				startRow := current[digit]*9 + 8 - phase
				for row := 0; row < 9; row++ {
					src := (startRow + row) * strip.Width
					dst := NativeShopGoldRollOffset + digit*6 +
						row*NativeShopWidth
					copy(frame[dst:dst+6], strip.Pixels[src:src+6])
				}
			}
			frames = append(frames, frame)
			staged = frame
		}
	}
	return frames, newGold, nil
}
