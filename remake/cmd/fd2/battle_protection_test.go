package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestBattleProtectionUsesSourceIdentityAcrossDisplayLocales(t *testing.T) {
	for _, locale := range []string{"zh-Hant", "en", "ja", "zh-Hans"} {
		t.Run(locale, func(t *testing.T) {
			display, err := loadOfficialLocaleEntities(locale)
			if err != nil {
				t.Fatal(err)
			}
			source, err := loadOfficialLocaleEntities("zh-Hant")
			if err != nil {
				t.Fatal(err)
			}
			for _, rawByte := range []bool{false, true} {
				u := &battle.Unit{HP: 42, OnField: true, NativeIdentity: 0, HasNativeIdentity: true,
					NativeRecordByte8: 0, HasNativeRecordByte8: rawByte}
				g := &Game{st: &battle.State{Units: []*battle.Unit{u}}, localeEntities: display, sourceBattleEntities: source}
				if alive, err := g.protectedBattleUnitAlive("索爾"); err != nil || !alive {
					t.Fatalf("living source identity: %v %v", alive, err)
				}
				u.HP = 0
				if alive, err := g.protectedBattleUnitAlive("索爾"); err != nil || alive {
					t.Fatalf("dead source identity: %v %v", alive, err)
				}
			}
		})
	}
}
