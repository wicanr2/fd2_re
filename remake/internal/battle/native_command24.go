package battle

import (
	"fmt"
	"math/rand"
)

// NativeCommand24Damage is the state result of player-dispatched native
// command 24.  It is deliberately separate from NativeCommandDamage: this
// route does not read the command's damage/hit bytes or class resistance.
type NativeCommand24Damage struct {
	Target                  *Unit
	Amount                  int
	Damage                  int
	HPBefore                int
	HPAfter                 int
	NativeRecordByte5Before byte
	NativeRecordByte5After  byte
}

// NativeCommandDerivedStrikePlan is a mutation-free result of the recovered
// 0x276EC selector, damage RNG and HP clamp. Presentation owners may publish
// MP and each target HP at their raw FIGANI markers without rerolling damage.
type NativeCommandDerivedStrikePlan struct {
	Actor     *Unit
	CommandID int
	MPBefore  int
	MPAfter   int
	Results   []NativeCommand24Damage
}

func nativeCommandDerivedStrikeMultiplier(commandID int) (int, bool) {
	switch commandID {
	case 24:
		return 15, true
	case 28:
		return 20, true
	case 29:
		return 12, true
	case 31:
		return 18, true
	default:
		return 0, false
	}
}

// ResolveNativeCommandDerivedStrikeDamage mirrors the signed amount path in
// 0x276EC -> 0x1C81F: trunc(actor derived AP * multiplier / 10) minus target
// derived DP, then trunc(amount * 9 / 10) plus trunc(rand()%100 * amount / 1000).
// Go integer division truncates toward zero, matching x86 IDIV here.
func ResolveNativeCommandDerivedStrikeDamage(actorAP, targetDP, multiplier int, rng *rand.Rand) (amount, damage int, err error) {
	if rng == nil {
		return 0, 0, fmt.Errorf("nil rng")
	}
	if multiplier <= 0 {
		return 0, 0, fmt.Errorf("invalid native derived-strike multiplier=%d", multiplier)
	}
	amount = actorAP*multiplier/10 - targetDP
	damage = amount*9/10 + rng.Intn(100)*amount/1000
	return amount, damage, nil
}

// ResolveNativeCommand24Damage preserves the ID24-specific public helper.
func ResolveNativeCommand24Damage(actorAP, targetDP int, rng *rand.Rand) (amount, damage int, err error) {
	return ResolveNativeCommandDerivedStrikeDamage(actorAP, targetDP, 15, rng)
}

// ExecuteNativeCommand24 is the recovered non-UI player state slice for
// 0x1CFF0 command 0x18 -> 0x2A6BD -> 0x276EC.  The generic two-stage target
// list and record24 MP debit are followed by the derived-AP/DP damage route.
// The original resource has separate actor and target markers, but command24
// uses denominator 1 and publishes the complete HP delta at its target marker.
// This state-only helper applies that final delta without claiming presentation.
func (s *State) ExecuteNativeCommand24(actor, confirmed *Unit, rng *rand.Rand) ([]NativeCommand24Damage, error) {
	return s.ExecuteNativeCommandDerivedStrike(actor, confirmed, 24, rng)
}

// ExecuteNativeCommandDerivedStrike is the state-only 0x276EC family with
// proven player dispatches 24, 28, 29 and 31. ID30 has its own special
// cursor-selector entry point below; IDs32..35 use 0x27FC9 and stay closed.
func (s *State) ExecuteNativeCommandDerivedStrike(actor, confirmed *Unit, commandID int, rng *rand.Rand) ([]NativeCommand24Damage, error) {
	plan, err := s.PlanNativeCommandDerivedStrike(actor, confirmed, commandID, rng)
	if err != nil {
		return nil, err
	}
	if err := ApplyNativeCommandDerivedStrikeMP(plan); err != nil {
		return nil, err
	}
	for i := range plan.Results {
		if err := ApplyNativeCommandDerivedStrikeTarget(plan, i); err != nil {
			return nil, err
		}
	}
	if err := CompleteNativeCommandDerivedStrike(plan); err != nil {
		return nil, err
	}
	return append([]NativeCommand24Damage(nil), plan.Results...), nil
}

// PlanNativeCommandDerivedStrike validates every target and consumes the
// caller-owned RNG, but leaves MP, HP and completion state unchanged.
func (s *State) PlanNativeCommandDerivedStrike(actor, confirmed *Unit, commandID int, rng *rand.Rand) (*NativeCommandDerivedStrikePlan, error) {
	if s == nil || rng == nil {
		return nil, fmt.Errorf("missing native derived-strike state/rng")
	}
	multiplier, ok := nativeCommandDerivedStrikeMultiplier(commandID)
	if !ok {
		return nil, fmt.Errorf("native derived-strike command unavailable id=%d", commandID)
	}
	if len(s.NativeCommandBook) != 36 || s.NativeCommandBook[commandID].ID != commandID {
		return nil, fmt.Errorf("native derived-strike record unavailable id=%d", commandID)
	}
	record := s.NativeCommandBook[commandID]
	flags, err := s.NativeCommandBaseFlags()
	if err != nil {
		return nil, err
	}
	targets, err := NativeCommandEffectTargets(s.W, s.H, actor, confirmed, record.SelectionMode, record.EffectMode, record.TargetCode, flags, s.Units)
	if err != nil {
		return nil, err
	}
	if actor == nil || record.MPCost < 0 || record.MPCost > 0xff || actor.MP < record.MPCost {
		return nil, fmt.Errorf("native command %d insufficient MP", commandID)
	}
	plan := &NativeCommandDerivedStrikePlan{
		Actor: actor, CommandID: commandID, MPBefore: actor.MP, MPAfter: actor.MP - record.MPCost,
		Results: make([]NativeCommand24Damage, 0, len(targets)),
	}
	for _, target := range targets {
		if target == nil {
			return nil, fmt.Errorf("native command %d has nil target", commandID)
		}
		amount, damage, err := ResolveNativeCommandDerivedStrikeDamage(actor.AP, target.DP, multiplier, rng)
		if err != nil {
			return nil, err
		}
		appliedDamage := damage
		if appliedDamage < 0 {
			appliedDamage = 0
		}
		if appliedDamage > target.HP {
			appliedDamage = target.HP
		}
		hpAfter := target.HP - appliedDamage
		rawByte5After := target.NativeRecordByte5
		if hpAfter == 0 && target.HasNativeRecordByte5 {
			rawByte5After = 1
		}
		plan.Results = append(plan.Results, NativeCommand24Damage{
			Target: target, Amount: amount, Damage: appliedDamage,
			HPBefore: target.HP, HPAfter: hpAfter,
			NativeRecordByte5Before: target.NativeRecordByte5,
			NativeRecordByte5After:  rawByte5After,
		})
	}
	return plan, nil
}

func ApplyNativeCommandDerivedStrikeMP(plan *NativeCommandDerivedStrikePlan) error {
	if plan == nil || plan.Actor == nil || plan.Actor.MP != plan.MPBefore || plan.Actor.Acted {
		return fmt.Errorf("native derived-strike MP publish state changed")
	}
	plan.Actor.MP = plan.MPAfter
	return nil
}

func ApplyNativeCommandDerivedStrikeTarget(plan *NativeCommandDerivedStrikePlan, index int) error {
	if plan == nil || index < 0 || index >= len(plan.Results) {
		return fmt.Errorf("native derived-strike target publish index unavailable")
	}
	result := plan.Results[index]
	if result.Target == nil || result.Target.HP != result.HPBefore ||
		(result.Target.HasNativeRecordByte5 && result.Target.NativeRecordByte5 != result.NativeRecordByte5Before) {
		return fmt.Errorf("native derived-strike target publish state changed")
	}
	result.Target.HP = result.HPAfter
	if result.Target.HasNativeRecordByte5 {
		result.Target.NativeRecordByte5 = result.NativeRecordByte5After
	}
	return nil
}

func CompleteNativeCommandDerivedStrike(plan *NativeCommandDerivedStrikePlan) error {
	if plan == nil || plan.Actor == nil || plan.Actor.MP != plan.MPAfter || plan.Actor.Acted {
		return fmt.Errorf("native derived-strike completion state changed")
	}
	for _, result := range plan.Results {
		if result.Target == nil || result.Target.HP != result.HPAfter ||
			(result.Target.HasNativeRecordByte5 && result.Target.NativeRecordByte5 != result.NativeRecordByte5After) {
			return fmt.Errorf("native derived-strike target is not published")
		}
	}
	// 0x18D8C applies the invoking actor's completion bit only after 0x1CFF0
	// returns success; presentation owners call this after their final frame.
	plan.Actor.Acted = true
	return nil
}

// ExecuteNativeCommand30 is the bounded player state slice for command 30:
// 0x1CFF0 first validates a normal 0x14818 candidate, then 0x149F8 traces
// from the saved pre-confirm cursor toward the confirmed cursor for
// record+3-0x10 steps, before 0x2A6BD -> 0x276EC applies multiplier 18.
// Its indexed multi-hit presentation and UI cursor lifecycle remain outside
// this method; callers must supply both recovered cursor positions.
func (s *State) ExecuteNativeCommand30(actor *Unit, savedCursor, confirmedCursor Cell, rng *rand.Rand) ([]NativeCommand24Damage, error) {
	if s == nil || actor == nil || rng == nil || len(s.NativeCommandBook) != 36 || s.NativeCommandBook[30].ID != 30 {
		return nil, fmt.Errorf("native command 30 state unavailable")
	}
	record := s.NativeCommandBook[30]
	if record.SelectionMode < 0x10 {
		return nil, fmt.Errorf("native command 30 invalid selector mode=%d", record.SelectionMode)
	}
	// The preceding 0x14818 -> 0x115B6 confirmation is still required. It
	// proves the provided confirmed cursor is a valid target candidate before
	// the special line selector is allowed to mutate MP or HP.
	flags, err := s.NativeCommandBaseFlags()
	if err != nil {
		return nil, err
	}
	selection, err := NativeCommandTargets(s.W, s.H, Cell{X: actor.X, Y: actor.Y}, record.SelectionMode, record.TargetCode, flags, s.Units)
	if err != nil {
		return nil, err
	}
	confirmed := false
	for _, candidate := range selection {
		if candidate.X == confirmedCursor.X && candidate.Y == confirmedCursor.Y {
			confirmed = true
			break
		}
	}
	if !confirmed {
		return nil, fmt.Errorf("confirmed cursor is not a native command 30 candidate")
	}
	targets, err := NativeCommand30Targets(s.W, s.H, savedCursor, confirmedCursor, record.SelectionMode-0x10, s.Units)
	if err != nil {
		return nil, err
	}
	if !SpendNativeCommandMP(actor, record.MPCost) {
		return nil, fmt.Errorf("native command 30 insufficient MP")
	}
	results := make([]NativeCommand24Damage, 0, len(targets))
	for _, target := range targets {
		amount, damage, err := ResolveNativeCommandDerivedStrikeDamage(actor.AP, target.DP, 18, rng)
		if err != nil {
			return nil, err
		}
		target.ApplyHPDamage(damage)
		results = append(results, NativeCommand24Damage{Target: target, Amount: amount, Damage: damage})
	}
	actor.Acted = true
	return results, nil
}
