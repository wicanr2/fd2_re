package battle

import (
	"math/rand"
	"testing"
)

// 原生戰場的單位不帶名字（身分走 NativeIdentity），升級時成長列由記錄 +7 選
// （0x1E2F8 → 0x4E4D1 = 0x620A1 + 11 × selector）。用名字查會一律查不到，等級
// 照加、數值不長——第一關跑完身分 4 升到 lv2 而 MaxHP 仍是 48 就是這個樣子。
func TestNativeGrowthRowFollowsRecordPlus7(t *testing.T) {
	rows, err := LoadNativeGrowthRows("../../../docs/data/exe_tables/growth.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 68 {
		t.Fatalf("成長表 %d 列，應為 68", len(rows))
	}
	// idx 4：ap [6,9) dp [4,5) dx [2,2] hp [8,11) mp [0,0]，即手寫表的亞雷斯 騎士。
	want := GrowthRow{AP: StatRange{6, 8}, DP: StatRange{4, 4}, DX: StatRange{2, 2},
		HP: StatRange{8, 10}, MP: StatRange{0, 0}}
	if rows[4] != want {
		t.Fatalf("row4=%+v，應為 %+v", rows[4], want)
	}
	if rows[4] != growthTable["亞雷斯"]["騎士"] {
		t.Fatalf("EXE row4 與手寫表的亞雷斯 騎士不一致：%+v vs %+v", rows[4], growthTable["亞雷斯"]["騎士"])
	}

	st := &State{NativeGrowthRows: rows}
	u := &Unit{Camp: Own, Lv: 1, MaxHP: 48, HP: 48, BattleFig: 4, HasBattleFig: true}
	events := st.GainExp(u, 100, rand.New(rand.NewSource(1)))
	if len(events) != 1 || u.Lv != 2 {
		t.Fatalf("升級 events=%d lv=%d", len(events), u.Lv)
	}
	if gain := u.MaxHP - 48; gain < 8 || gain > 10 {
		t.Fatalf("MaxHP 成長 %d，應在 row4 的 8..10 之內", gain)
	}

	// 反對照：沒有原版 +7、名字也查不到，升級只加等級。
	legacy := &Unit{Camp: Own, Lv: 1, MaxHP: 48, HP: 48}
	if events := st.GainExp(legacy, 100, rand.New(rand.NewSource(1))); len(events) != 0 ||
		legacy.Lv != 2 || legacy.MaxHP != 48 {
		t.Fatalf("無 +7 的單位：events=%d lv=%d maxhp=%d", len(events), legacy.Lv, legacy.MaxHP)
	}
}
