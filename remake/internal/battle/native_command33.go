package battle

import (
	"encoding/binary"
	"fmt"
	"math"
)

// NativeCompoundCommand33Result exposes the proven raw HP batch, target
// snapshots and RNG boundary. The indexed owner consumes these snapshots;
// the score/EXP consumer remains unresolved.
type NativeCompoundCommand33Result struct {
	Restore NativeRawRestoreBatch
	Targets []NativeCompoundCommand33Target
}

type NativeCompoundCommand33Target struct {
	Target                          *Unit
	HPBefore, HPAfter               int
	TransientBefore, TransientAfter [6]byte
}

type nativeCompoundCommand33Target struct {
	target                 *Unit
	hpBefore, hpAfter      int
	transientBefore, after [6]byte
}

// NativeCompoundCommand33Plan keeps the 0x211A4 clear/restore batch private
// until the indexed owner reaches the post-mask publication boundary.
type NativeCompoundCommand33Plan struct {
	Actor            *Unit
	Result           NativeCompoundCommand33Result
	ActorActedBefore bool
	RNGBefore        uint16
	targets          []nativeCompoundCommand33Target
	published        bool
	completed        bool
}

// ExecuteNativeCompoundCommand33 implements the bounded player class-19 state
// transaction at 0x285A1..0x285ED. Proven player FIGANI resources bypass the
// known MP debit sink, so record33 remains an availability gate only.
func (s *State) ExecuteNativeCompoundCommand33(actor, confirmed *Unit, rngState uint16) (NativeCompoundCommand33Result, error) {
	plan, err := s.PlanNativeCompoundCommand33(actor, confirmed, rngState)
	if err != nil {
		return NativeCompoundCommand33Result{}, err
	}
	if err := PublishNativeCompoundCommand33(plan); err != nil {
		return NativeCompoundCommand33Result{}, err
	}
	if err := CompleteNativeCompoundCommand33(plan); err != nil {
		_ = AbortNativeCompoundCommand33(plan)
		return NativeCompoundCommand33Result{}, err
	}
	return plan.Result, nil
}

// PlanNativeCompoundCommand33 computes the complete raw clear/restore result
// without mutating the actor, targets or caller-owned RNG state.
func (s *State) PlanNativeCompoundCommand33(actor, confirmed *Unit, rngState uint16) (*NativeCompoundCommand33Plan, error) {
	const commandID = 33
	if s == nil || actor == nil || actor.Acted || !actor.HasNativeRecordClass ||
		actor.NativeRecordClass != 19 || !actor.HasBattleFig || !nativeCompoundPlayerSelector(actor.BattleFig) ||
		len(s.NativeCommandBook) != NativeCommandRecordCount || s.NativeCommandBook[commandID].ID != commandID {
		return nil, fmt.Errorf("native compound command 33 player provenance unavailable")
	}
	record := s.NativeCommandBook[commandID]
	if record.MPCost < 0 || actor.MP < record.MPCost {
		return nil, fmt.Errorf("native compound command 33 MP gate failed")
	}
	flags, err := s.NativeCommandBaseFlags()
	if err != nil {
		return nil, err
	}
	targets, err := NativeCommandEffectTargets(
		s.W, s.H, actor, confirmed, record.SelectionMode, record.EffectMode,
		record.TargetCode, flags, s.Units,
	)
	if err != nil {
		return nil, err
	}
	records, indices, err := nativeCompoundCommandTargetRecords(targets)
	if err != nil {
		return nil, err
	}
	seen := make(map[*Unit]struct{}, len(targets))
	for index, target := range targets {
		if _, duplicate := seen[target]; duplicate {
			return nil, fmt.Errorf("native compound command 33 target %d duplicated", index)
		}
		seen[target] = struct{}{}
	}
	for index := range targets {
		base := index * nativeRecordSize
		clear(records[base+0x25 : base+0x28])
	}
	restore, err := ApplyNativeRawHPRestoreList(records, indices, 0x320, rngState)
	if err != nil {
		return nil, err
	}
	planned := make([]nativeCompoundCommand33Target, len(targets))
	for index, target := range targets {
		base := index * nativeRecordSize
		planned[index] = nativeCompoundCommand33Target{
			target: target, hpBefore: target.HP,
			hpAfter:         int(binary.LittleEndian.Uint16(records[base+0x40 : base+0x42])),
			transientBefore: target.NativeTransient,
		}
		copy(planned[index].after[:], records[base+0x22:base+0x28])
	}
	resultTargets := make([]NativeCompoundCommand33Target, len(planned))
	for index, target := range planned {
		resultTargets[index] = NativeCompoundCommand33Target{
			Target: target.target, HPBefore: target.hpBefore, HPAfter: target.hpAfter,
			TransientBefore: target.transientBefore, TransientAfter: target.after,
		}
	}
	return &NativeCompoundCommand33Plan{
		Actor: actor, Result: NativeCompoundCommand33Result{Restore: restore, Targets: resultTargets},
		ActorActedBefore: actor.Acted, RNGBefore: rngState, targets: planned,
	}, nil
}

func PublishNativeCompoundCommand33(plan *NativeCompoundCommand33Plan) error {
	if plan == nil || plan.Actor == nil || plan.completed || plan.published ||
		plan.Actor.Acted != plan.ActorActedBefore || len(plan.targets) == 0 ||
		len(plan.Result.Restore.Results) != len(plan.targets) {
		return fmt.Errorf("native compound command 33 publication unavailable")
	}
	for index, target := range plan.targets {
		if target.target == nil || target.target.HP != target.hpBefore ||
			target.target.NativeTransient != target.transientBefore {
			return fmt.Errorf("native compound command 33 target %d state changed", index)
		}
	}
	for _, target := range plan.targets {
		target.target.HP = target.hpAfter
		target.target.NativeTransient = target.after
	}
	plan.published = true
	return nil
}

func CompleteNativeCompoundCommand33(plan *NativeCompoundCommand33Plan) error {
	if plan == nil || plan.Actor == nil || plan.completed || !plan.published ||
		plan.Actor.Acted != plan.ActorActedBefore {
		return fmt.Errorf("native compound command 33 completion unavailable")
	}
	for index, target := range plan.targets {
		if target.target == nil || target.target.HP != target.hpAfter || target.target.NativeTransient != target.after {
			return fmt.Errorf("native compound command 33 target %d incomplete", index)
		}
	}
	plan.Actor.Acted = true
	plan.completed = true
	return nil
}

func AbortNativeCompoundCommand33(plan *NativeCompoundCommand33Plan) error {
	if plan == nil || plan.Actor == nil || plan.completed || plan.Actor.Acted != plan.ActorActedBefore {
		return fmt.Errorf("native compound command 33 rollback unavailable")
	}
	if !plan.published {
		return nil
	}
	for index, target := range plan.targets {
		if target.target == nil || target.target.HP != target.hpAfter || target.target.NativeTransient != target.after {
			return fmt.Errorf("native compound command 33 target %d rollback state changed", index)
		}
	}
	for index := len(plan.targets) - 1; index >= 0; index-- {
		target := plan.targets[index]
		target.target.HP = target.hpBefore
		target.target.NativeTransient = target.transientBefore
	}
	plan.published = false
	return nil
}

func nativeCompoundCommandTargetRecords(targets []*Unit) ([]byte, []byte, error) {
	if len(targets) == 0 || len(targets) > 256 {
		return nil, nil, fmt.Errorf("native compound command target count=%d", len(targets))
	}
	records := make([]byte, len(targets)*nativeRecordSize)
	indices := make([]byte, len(targets))
	for index, target := range targets {
		if target == nil || !target.HasNativeRecordClass || !target.HasBattleFig ||
			target.Lv < 0 || target.Lv > math.MaxUint8 || target.BattleFig < 0 || target.BattleFig > math.MaxUint8 ||
			target.HP < 0 || target.HP > math.MaxUint16 || target.MaxHP < 0 || target.MaxHP > math.MaxUint16 {
			return nil, nil, fmt.Errorf("native compound command target %d lacks raw provenance", index)
		}
		base := index * nativeRecordSize
		records[base+5] = target.NativeRecordByte5
		records[base+7] = byte(target.BattleFig)
		records[base+0x20], records[base+0x21] = target.NativeRecordClass, byte(target.Lv)
		copy(records[base+0x22:base+0x28], target.NativeTransient[:])
		binary.LittleEndian.PutUint16(records[base+0x40:base+0x42], uint16(target.HP))
		binary.LittleEndian.PutUint16(records[base+0x42:base+0x44], uint16(target.MaxHP))
		indices[index] = byte(index)
	}
	return records, indices, nil
}
