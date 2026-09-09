package main

import "github.com/wicanr2/fd2_re/remake/internal/localization"

// protect 保存作者使用的來源姓名；原生角色不必把姓名複製到執行期 record。
func (g *Game) protectedBattleUnitAlive(protect string) (bool, error) {
	if protect == "" {
		return true, nil
	}
	for _, u := range g.st.Units {
		if u == nil || !u.Alive() {
			continue
		}
		name := u.Name
		if u.HasNativeRecordByte8 || u.HasNativeIdentity {
			if g.sourceBattleEntities == nil {
				catalog, err := localization.LoadOfficialEntities(separatedAssetPath("locales"), "zh-Hant")
				if err != nil {
					return false, err
				}
				g.sourceBattleEntities = catalog
			}
			var err error
			if u.HasNativeRecordByte8 {
				name, err = g.sourceBattleEntities.BattleName(int(u.NativeRecordByte8))
			} else {
				// 舊版來源化角色保存 persistent identity，未另存 runtime +8。
				name, err = g.sourceBattleEntities.CharacterName(u.NativeIdentity)
			}
			if err != nil {
				return false, err
			}
		}
		if name == protect {
			return true, nil
		}
	}
	return false, nil
}
