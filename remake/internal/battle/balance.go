package battle

import "fmt"

// BalanceOptions is an optional, progress-local remake extension. Nil preserves
// the original rules. Keys are persistent record identities, never selectors.
type BalanceOptions struct {
	Growth       GrowthRates         `json:"growth"`
	Characters   map[int]GrowthRates `json:"characters,omitempty"`
	SkillHit     map[int]map[int]int `json:"skill_hit,omitempty"`
	EnemyPercent int                 `json:"enemy_percent"`
}

type GrowthRates struct{ HP, MP, AP, DP, DX int }

func OriginalGrowthRates() GrowthRates { return GrowthRates{100, 100, 100, 100, 100} }
func NewBalanceOptions() *BalanceOptions {
	return &BalanceOptions{Growth: OriginalGrowthRates(), EnemyPercent: 100}
}

func (o *BalanceOptions) Validate() error {
	if o == nil {
		return nil
	}
	check := func(r GrowthRates) bool {
		for _, n := range []int{r.HP, r.MP, r.AP, r.DP, r.DX} {
			if n < 0 || n > 500 {
				return false
			}
		}
		return true
	}
	if !check(o.Growth) || o.EnemyPercent < 10 || o.EnemyPercent > 500 {
		return fmt.Errorf("能力設定倍率超出範圍")
	}
	for id, r := range o.Characters {
		if id < 0 || id > 31 || !check(r) {
			return fmt.Errorf("成長設定角色或倍率無效")
		}
	}
	for id, rows := range o.SkillHit {
		if id < 0 || id > 31 {
			return fmt.Errorf("命中設定角色無效")
		}
		for cmd, n := range rows {
			if !NativeCommandHasHitRoll(cmd) || n < 0 || n > 100 {
				return fmt.Errorf("招式沒有可調命中骰或命中率無效")
			}
		}
	}
	return nil
}

func (o *BalanceOptions) IsOriginal() bool {
	return o == nil || o.Growth == OriginalGrowthRates() && len(o.Characters) == 0 && len(o.SkillHit) == 0 && o.EnemyPercent == 100
}

func BalanceIdentity(u *Unit) (int, bool) {
	if u == nil || u.Camp != Own || !u.HasNativeIdentity || u.NativeIdentity < 0 || u.NativeIdentity > 31 {
		return 0, false
	}
	return u.NativeIdentity, true
}

func NativeCommandHasHitRoll(id int) bool {
	return id >= 0 && id <= 12 || id == 22 || id == 26 || id == 27 || id == 32
}

func (s *State) CommandHit(actor *Unit, commandID, original int) int {
	id, ok := BalanceIdentity(actor)
	if !ok || s == nil || s.Balance == nil || !NativeCommandHasHitRoll(commandID) {
		return original
	}
	if n, exists := s.Balance.SkillHit[id][commandID]; exists {
		return n
	}
	return original
}

func (s *State) balanceGrowthRow(u *Unit, row GrowthRow) GrowthRow {
	if s == nil || s.Balance == nil {
		return row
	}
	id, ok := BalanceIdentity(u)
	if !ok {
		return row
	}
	r := s.Balance.Growth
	if own, exists := s.Balance.Characters[id]; exists {
		r = own
	}
	row.balanceRates = &r
	return row
}

// BalanceStats stores the unscaled enemy baseline for idempotent application
// and native current-battle sidecar persistence.
type BalanceStats struct{ HP, MP, AP, DP, DX int }

func (u *Unit) BalanceStats() BalanceStats {
	ap, dp := u.AP, u.DP
	if u.EquipmentBaseSet {
		ap, dp = u.BaseAP, u.BaseDP
	}
	return BalanceStats{u.MaxHP, u.MaxMP, ap, dp, u.DX}
}

// SetBalanceStats preserves equipment contributions and never heals a unit.
func (u *Unit) SetBalanceStats(v BalanceStats) error {
	if u == nil || v.HP < 1 || v.HP > 32767 || v.MP < 0 || v.MP > 32767 || v.AP < 0 || v.AP > 32767 || v.DP < 0 || v.DP > 32767 || v.DX < 0 || v.DX > 32767 {
		return fmt.Errorf("能力值超出0至32767範圍，生命上限須至少1")
	}
	old := u.BalanceStats()
	if u.AP+v.AP-old.AP > 32767 || u.DP+v.DP-old.DP > 32767 || u.HIT+v.DX-old.DX > 32767 || u.EV+v.DX-old.DX > 32767 {
		return fmt.Errorf("能力值與裝備加成合計超出32767")
	}
	u.MaxHP, u.MaxMP = v.HP, v.MP
	u.AP += v.AP - old.AP
	u.DP += v.DP - old.DP
	if u.EquipmentBaseSet {
		u.BaseAP = v.AP
		u.BaseDP = v.DP
		u.BaseHIT += v.DX - old.DX
		u.BaseEV += v.DX - old.DX
	}
	u.HIT += v.DX - old.DX
	u.EV += v.DX - old.DX
	u.DX = v.DX
	if u.HP > u.MaxHP {
		u.HP = u.MaxHP
	}
	if u.MP > u.MaxMP {
		u.MP = u.MaxMP
	}
	if u.HasNativeRecordWord42 {
		u.NativeRecordWord42 = uint16(u.MaxHP)
	}
	if u.HasNativeRecordWord46 {
		u.NativeRecordWord46 = uint16(u.MaxMP)
	}
	return nil
}

func scaleBalance(n, p int) int {
	v := n * p / 100
	if v > 32767 {
		return 32767
	}
	if v < 0 {
		return 0
	}
	return v
}

func (s *State) ApplyEnemyStrength() {
	if s == nil {
		return
	}
	pct := 100
	if s.Balance != nil {
		pct = s.Balance.EnemyPercent
	}
	for _, u := range s.Units {
		if u == nil || u.Camp != Enemy || u.nativeConstructorPending {
			continue
		}
		if pct == 100 && u.EnemyBalanceBase == nil {
			continue
		}
		if u.EnemyBalanceBase == nil {
			base := u.BalanceStats()
			u.EnemyBalanceBase = &base
			u.EnemyBalancePercent = 100
		}
		if u.EnemyBalancePercent == pct {
			continue
		}
		base := *u.EnemyBalanceBase
		v := BalanceStats{scaleBalance(base.HP, pct), scaleBalance(base.MP, pct), scaleBalance(base.AP, pct), scaleBalance(base.DP, pct), scaleBalance(base.DX, pct)}
		if v.HP < 1 {
			v.HP = 1
		}
		hp, mp, oldHP, oldMP := u.HP, u.MP, u.MaxHP, u.MaxMP
		if u.SetBalanceStats(v) != nil {
			continue
		}
		if oldHP > 0 {
			u.HP = hp * v.HP / oldHP
			if hp > 0 && u.HP == 0 {
				u.HP = 1
			}
		}
		if oldMP > 0 {
			u.MP = mp * v.MP / oldMP
		}
		u.EnemyBalancePercent = pct
	}
}
