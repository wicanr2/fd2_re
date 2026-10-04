package battlepresent

import (
	"errors"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// NativePhysicalBodyFrame 保留 0x2939D 的 descriptor 邊界，包含零次呈現。
// 證據與 READY 契約：docs/data/ida/fd2_physical_background_selection_20261004.json。
type NativePhysicalBodyFrame struct {
	Strike, Frame, HP int
	UpdatePanel       bool
	Cue               int // -1 表示不呼叫 0x25A96；0 是有效的 miss cue。
	Presents          []NativePhysicalBodyPresent
}

type NativePhysicalBodyPresent struct {
	IdleFrame, DX, DY int
	OpaqueFill        int // -1 保留來源；33 是原版 opaque index。
	StatusPulse       bool
	CritPulse         bool
}

// BuildNativePhysicalBodyPlan 只消費已結算的 Roll，不讀取或推進 RNG。
// idle 與 impact phase 跨同一個 0x2939D 的多揮保留。
func BuildNativePhysicalBodyPlan(attack, idle *figani.Animation, rawSide byte, initialHP int, strikes []battle.NativePhysicalStrike) ([]NativePhysicalBodyFrame, error) {
	if attack == nil || idle == nil || len(attack.Frames) == 0 || len(idle.Frames) == 0 || len(idle.Frames) > 255 || initialHP < 0 || len(strikes) == 0 {
		return nil, errors.New("battlepresent: physical body inputs unavailable")
	}
	start := 0
	if attack.HeaderByte1 != 0 {
		start = int(attack.HeaderByte2)
	}
	if start > len(attack.Frames) {
		return nil, errors.New("battlepresent: physical body header exceeds frames")
	}
	hits := 0
	for _, f := range attack.Frames {
		if f.Delay < 0 || f.Delay > 255 {
			return nil, errors.New("battlepresent: physical body delay is not a byte")
		}
		if f.RawByte4 != 0 {
			hits++
		}
	}
	for _, f := range idle.Frames {
		if f.Delay < 0 || f.Delay > 255 {
			return nil, errors.New("battlepresent: physical idle delay is not a byte")
		}
	}
	if hits == 0 {
		hits = 1
	}
	horizontal := [...]int{0, 4, 9, 14, 18, 14}
	vertical := [...]int{0, 2, 4, 6, 8, 10}
	phase, fill, idleFrame := 0, -1, 0
	var idleCounter byte
	hp := initialHP
	var plan []NativePhysicalBodyFrame
	for si, strike := range strikes {
		if strike.Roll.Damage < 0 || strike.DefenderHP < 0 || hp == 0 {
			return nil, errors.New("battlepresent: physical strike has invalid HP or damage")
		}
		before, hitStep := hp, 0
		for fi := start; fi < len(attack.Frames); fi++ {
			f := attack.Frames[fi]
			item := NativePhysicalBodyFrame{Strike: si, Frame: fi, HP: hp, Cue: -1}
			if f.RawByte4 != 0 {
				hitStep++
				hp = before - strike.Roll.Damage*hitStep/hits
				if hp < 0 {
					hp = 0
				}
				item.HP, item.UpdatePanel = hp, true
				if !strike.Roll.Missed {
					phase, fill = 5, 33
				}
			}
			if f.RawByte5 != 0 {
				item.Cue = int(f.RawByte5)
				if f.RawByte4 != 0 && strike.Roll.Missed {
					item.Cue = 0
				}
			}
			for inner := 0; inner < f.Delay; inner++ {
				dx, dy := horizontal[phase], vertical[phase]
				if attack.HeaderByte1 != 0 {
					dy = 0
				}
				if rawSide != 0 {
					dx, dy = -dx, -dy
				}
				pulse := f.RawByte4 == 1 && hitStep == hits
				item.Presents = append(item.Presents, NativePhysicalBodyPresent{
					IdleFrame: idleFrame, DX: dx, DY: dy, OpaqueFill: fill,
					StatusPulse: pulse && strike.Roll.Status, CritPulse: pulse && strike.Roll.Crit,
				})
				idleCounter++ // 原版 uint8 increment 後才作等號比較。
				if int(idleCounter) == idle.Frames[idleFrame].Delay {
					idleCounter = 0
					idleFrame++
					if idleFrame == len(idle.Frames) {
						idleFrame = 0
					}
				}
				if phase != 0 {
					phase--
				}
				fill = -1
			}
			plan = append(plan, item)
		}
	}
	return plan, nil
}

// ComposeNativePhysicalBodyPresent 以有界 viewport 表達原版 stride400 的
// 40-pixel staging margin；只裁切 viewport 外像素，不遮蔽比較畫布。
func ComposeNativePhysicalBodyPresent(base []byte, attack, idle figani.Frame, rawSide byte, present NativePhysicalBodyPresent) ([]byte, error) {
	if len(base) != 320*200 || (present.OpaqueFill != -1 && present.OpaqueFill != 33) {
		return nil, errors.New("battlepresent: physical body base or fill is invalid")
	}
	out := append([]byte(nil), base...)
	var fill *byte
	value := byte(33)
	if present.OpaqueFill == 33 {
		fill = &value
	}
	attackFirst := (rawSide == 0 && attack.RawByte7&1 == 0) || (rawSide != 0 && attack.RawByte7&1 != 0)
	if attackFirst {
		if err := attack.BlitTranslated(out, 320, 0, 0, nil); err != nil {
			return nil, err
		}
	}
	if err := idle.BlitTranslated(out, 320, present.DX, present.DY, fill); err != nil {
		return nil, err
	}
	if !attackFirst {
		if err := attack.BlitTranslated(out, 320, 0, 0, nil); err != nil {
			return nil, err
		}
	}
	return out, nil
}
