package battle

import (
	"fmt"
	"math/rand"
)

// NativeCommand24Damage is the state result of player-dispatched native
// command 24.  It is deliberately separate from NativeCommandDamage: this
// route does not read the command's damage/hit bytes or class resistance.
type NativeCommand24Damage struct {
	Target *Unit
	Amount int
	// Rolled 是 0x1C81F 內的擲出值（截斷到 0 與剩餘 HP 之前），經驗公式用它。
	Rolled                  int
	Damage                  int
	HPBefore                int
	HPAfter                 int
	NativeRecordByte5Before byte
	NativeRecordByte5After  byte
	deathFlagBefore         byte
	deathFlagValidBefore    bool
}

// NativeCommandDerivedStrikePlan is a mutation-free result of the recovered
// 0x276EC selector, damage RNG and HP clamp. Presentation owners may publish
// MP and each target HP at their raw FIGANI markers without rerolling damage.
type NativeCommandDerivedStrikePlan struct {
	Actor                 *Unit
	CommandID             int
	MPBefore              int
	MPAfter               int
	Results               []NativeCommand24Damage
	TargetImpactCount     int
	publishedTargetStages []int
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
		if commandID == 28 {
			for stage := 1; stage <= plan.TargetImpactCount; stage++ {
				if err := ApplyNativeCommand28TargetStage(plan, i, stage); err != nil {
					return nil, err
				}
			}
		} else {
			if err := ApplyNativeCommandDerivedStrikeTarget(plan, i); err != nil {
				return nil, err
			}
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
	if len(s.NativeCommandBook) != 36 || commandID < 0 || commandID >= len(s.NativeCommandBook) || s.NativeCommandBook[commandID].ID != commandID {
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
	return s.planNativeCommandDerivedStrikeTargets(actor, targets, commandID, rng)
}

// NativeCommand29EffectTargets keeps the original self-centered cursor separate
// from the affected enemy units. The direct-unit entry above remains available.
func (s *State) NativeCommand29EffectTargets(actor *Unit, cursor Cell) ([]*Unit, error) {
	if s == nil || actor == nil || actor.Camp == Enemy || len(s.NativeCommandBook) != NativeCommandRecordCount || s.NativeCommandBook[29].ID != 29 {
		return nil, fmt.Errorf("native player command29 record/actor unavailable")
	}
	record := s.NativeCommandBook[29]
	flags, err := s.NativeCommandBaseFlags()
	if err != nil {
		return nil, err
	}
	if record.SelectionMode == 0 {
		return NativeCommandSelfCenteredTargets(s.W, s.H, actor, cursor, record.EffectMode, flags, s.Units)
	}
	return NativeCommandEffectTargetsAtCursor(s.W, s.H, actor, cursor, record.SelectionMode, record.EffectMode, record.TargetCode, flags, s.Units)
}

func (s *State) PlanNativeCommand29AtCursor(actor *Unit, cursor Cell, rng *rand.Rand) (*NativeCommandDerivedStrikePlan, error) {
	targets, err := s.NativeCommand29EffectTargets(actor, cursor)
	if err != nil {
		return nil, err
	}
	return s.planNativeCommandDerivedStrikeTargets(actor, targets, 29, rng)
}

func (s *State) planNativeCommandDerivedStrikeTargets(actor *Unit, targets []*Unit, commandID int, rng *rand.Rand) (*NativeCommandDerivedStrikePlan, error) {
	if s == nil || rng == nil || commandID < 0 || commandID >= len(s.NativeCommandBook) || len(s.NativeCommandBook) != NativeCommandRecordCount || s.NativeCommandBook[commandID].ID != commandID {
		return nil, fmt.Errorf("native derived-strike record/rng unavailable id=%d", commandID)
	}
	multiplier, ok := nativeCommandDerivedStrikeMultiplier(commandID)
	if !ok {
		return nil, fmt.Errorf("native derived-strike command unavailable id=%d", commandID)
	}
	record := s.NativeCommandBook[commandID]
	if actor == nil || record.MPCost < 0 || record.MPCost > 0xff || actor.MP < record.MPCost {
		return nil, fmt.Errorf("native command %d insufficient MP", commandID)
	}
	if commandID == 28 && (!actor.HasBattleFig || actor.BattleFig != 19) {
		return nil, fmt.Errorf("native command28 requires proven selector19 impact contract")
	}
	plan := &NativeCommandDerivedStrikePlan{
		Actor: actor, CommandID: commandID, MPBefore: actor.MP, MPAfter: actor.MP - record.MPCost,
		Results: make([]NativeCommand24Damage, 0, len(targets)),
	}
	if commandID == 28 {
		// FIGANI59 has eight target raw+4 markers. Other selectors require
		// their own reviewed contract rather than a single-marker guess.
		plan.TargetImpactCount = 8
		plan.publishedTargetStages = make([]int, len(targets))
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
		// 0x27C44 clamps the roll to original HP before command28 divides
		// its cumulative eight impact stages. Keep the final capped damage.
		if appliedDamage > target.HP {
			appliedDamage = target.HP
		}
		hpAfter := target.HP - appliedDamage
		rawByte5After := target.NativeRecordByte5
		if hpAfter == 0 && target.HasNativeRecordByte5 {
			rawByte5After = 1
		}
		plan.Results = append(plan.Results, NativeCommand24Damage{
			Target: target, Amount: amount, Rolled: damage, Damage: appliedDamage,
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
	if plan.CommandID == 28 {
		return fmt.Errorf("native command28 requires ordered impact-stage publication")
	}
	result := plan.Results[index]
	if result.Target == nil || result.Target.HP != result.HPBefore ||
		(result.Target.HasNativeRecordByte5 && result.Target.NativeRecordByte5 != result.NativeRecordByte5Before) {
		return fmt.Errorf("native derived-strike target publish state changed")
	}
	if plan.CommandID == 30 {
		result.Target.ApplyHPDamage(result.HPBefore - result.HPAfter)
		return nil
	}
	result.Target.HP = result.HPAfter
	if result.Target.HasNativeRecordByte5 {
		result.Target.NativeRecordByte5 = result.NativeRecordByte5After
	}
	return nil
}

// NativeCommand28StageResult is a frozen panel/HP projection. Stage0 denotes
// the original target; stages1..8 use the original capped-damage denominator.
func NativeCommand28StageResult(plan *NativeCommandDerivedStrikePlan, index, stage int) (NativeCommand24Damage, error) {
	if plan == nil || plan.CommandID != 28 || plan.TargetImpactCount != 8 || index < 0 || index >= len(plan.Results) || stage < 0 || stage > 8 {
		return NativeCommand24Damage{}, fmt.Errorf("native command28 stage projection unavailable")
	}
	result := plan.Results[index]
	if result.Target == nil {
		return NativeCommand24Damage{}, fmt.Errorf("native command28 target projection unavailable")
	}
	result.Damage = result.Damage * stage / 8
	result.HPAfter = result.HPBefore - result.Damage
	result.NativeRecordByte5After = result.NativeRecordByte5Before
	if result.HPAfter == 0 && result.Target.HasNativeRecordByte5 {
		result.NativeRecordByte5After = 1
	}
	return result, nil
}

func ApplyNativeCommand28TargetStage(plan *NativeCommandDerivedStrikePlan, index, stage int) error {
	if plan == nil || plan.Actor == nil || plan.Actor.MP != plan.MPAfter || plan.Actor.Acted || index < 0 || index >= len(plan.publishedTargetStages) || stage != plan.publishedTargetStages[index]+1 {
		return fmt.Errorf("native command28 stage publication state changed")
	}
	before, err := NativeCommand28StageResult(plan, index, stage-1)
	if err != nil {
		return err
	}
	after, err := NativeCommand28StageResult(plan, index, stage)
	if err != nil {
		return err
	}
	target := after.Target
	if target == nil || target.HP != before.HPAfter || (target.HasNativeRecordByte5 && target.NativeRecordByte5 != before.NativeRecordByte5After) {
		return fmt.Errorf("native command28 target changed before impact")
	}
	target.HP = after.HPAfter
	if target.HasNativeRecordByte5 {
		target.NativeRecordByte5 = after.NativeRecordByte5After
	}
	plan.publishedTargetStages[index] = stage
	return nil
}

// RestoreNativeCommand30Targets restores the death-presentation provenance as
// well as HP/raw +5. It is confined to the ID30 owner transaction.
func RestoreNativeCommand30Targets(plan *NativeCommandDerivedStrikePlan) {
	if plan == nil || plan.CommandID != 30 {
		return
	}
	for _, result := range plan.Results {
		if result.Target == nil {
			continue
		}
		result.Target.HP = result.HPBefore
		result.Target.NativeRecordByte5 = result.NativeRecordByte5Before
		result.Target.nativeDeathFlagBefore = result.deathFlagBefore
		result.Target.nativeDeathFlagValid = result.deathFlagValidBefore
	}
}

func CompleteNativeCommandDerivedStrike(plan *NativeCommandDerivedStrikePlan) error {
	if plan == nil || plan.Actor == nil || plan.Actor.MP != plan.MPAfter || plan.Actor.Acted {
		return fmt.Errorf("native derived-strike completion state changed")
	}
	if plan.CommandID == 28 {
		if len(plan.publishedTargetStages) != len(plan.Results) {
			return fmt.Errorf("native command28 impact publication unavailable")
		}
		for _, stage := range plan.publishedTargetStages {
			if stage != 8 {
				return fmt.Errorf("native command28 impacts are incomplete")
			}
		}
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

// NativeCommand30EffectTargets validates the bounded player selector for command30:
// 0x1CFF0 first validates a normal 0x14818 candidate, then 0x149F8 traces
// from the saved pre-confirm cursor toward the confirmed cursor for
// record+3-0x10 steps, before 0x2A6BD -> 0x276EC applies multiplier 18.
// Its indexed multi-hit presentation and UI cursor lifecycle remain outside
// this method; callers must supply both recovered cursor positions.
func (s *State) NativeCommand30EffectTargets(actor *Unit, savedCursor, confirmedCursor Cell) ([]*Unit, error) {
	if s == nil || actor == nil || len(s.NativeCommandBook) != 36 || s.NativeCommandBook[30].ID != 30 {
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
	return NativeCommand30Targets(s.W, s.H, savedCursor, confirmedCursor, record.SelectionMode-0x10, s.Units)
}

// PlanNativeCommand30 shares the proven selector with the existing state-only
// control, while freezing each HP result for the indexed/HD marker publisher.
func (s *State) PlanNativeCommand30(actor *Unit, savedCursor, confirmedCursor Cell, rng *rand.Rand) (*NativeCommandDerivedStrikePlan, error) {
	if rng == nil {
		return nil, fmt.Errorf("native command 30 rng unavailable")
	}
	targets, err := s.NativeCommand30EffectTargets(actor, savedCursor, confirmedCursor)
	if err != nil {
		return nil, err
	}
	record := s.NativeCommandBook[30]
	if actor.Acted || record.MPCost < 0 || actor.MP < record.MPCost {
		return nil, fmt.Errorf("native command 30 insufficient MP or completed actor")
	}
	plan := &NativeCommandDerivedStrikePlan{Actor: actor, CommandID: 30, MPBefore: actor.MP, MPAfter: actor.MP - record.MPCost}
	for _, target := range targets {
		amount, damage, err := ResolveNativeCommandDerivedStrikeDamage(actor.AP, target.DP, 18, rng)
		if err != nil {
			return nil, err
		}
		applied := damage
		if applied < 0 {
			applied = 0
		}
		if applied > target.HP {
			applied = target.HP
		}
		after := target.NativeRecordByte5
		if applied > 0 && target.HP-applied == 0 && target.HasNativeRecordByte5 {
			after = 1
		}
		// Damage keeps the existing ID30 state-only result's unclamped roll.
		plan.Results = append(plan.Results, NativeCommand24Damage{Target: target, Amount: amount, Rolled: damage, Damage: damage, HPBefore: target.HP, HPAfter: target.HP - applied, NativeRecordByte5Before: target.NativeRecordByte5, NativeRecordByte5After: after, deathFlagBefore: target.nativeDeathFlagBefore, deathFlagValidBefore: target.nativeDeathFlagValid})
	}
	return plan, nil
}

func (s *State) ExecuteNativeCommand30(actor *Unit, savedCursor, confirmedCursor Cell, rng *rand.Rand) ([]NativeCommand24Damage, error) {
	if rng == nil {
		return nil, fmt.Errorf("native command 30 state unavailable")
	}
	targets, err := s.NativeCommand30EffectTargets(actor, savedCursor, confirmedCursor)
	if err != nil {
		return nil, err
	}
	record := s.NativeCommandBook[30]
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
		results = append(results, NativeCommand24Damage{Target: target, Amount: amount, Rolled: damage, Damage: damage})
	}
	actor.Acted = true
	return results, nil
}
