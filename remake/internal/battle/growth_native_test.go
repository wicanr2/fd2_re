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
	// 0x1E529 只寫 +0x42，目前 HP +0x40 不變。
	if u.HP != 48 {
		t.Fatalf("升級後目前 HP=%d，原版不補，應維持 48", u.HP)
	}

	// 反對照：沒有原版 +7、名字也查不到，升級只加等級。
	legacy := &Unit{Camp: Own, Lv: 1, MaxHP: 48, HP: 48}
	if events := st.GainExp(legacy, 100, rand.New(rand.NewSource(1))); len(events) != 0 ||
		legacy.Lv != 2 || legacy.MaxHP != 48 {
		t.Fatalf("無 +7 的單位：events=%d lv=%d maxhp=%d", len(events), legacy.Lv, legacy.MaxHP)
	}
}

// TestNativeLevelCapFollows1E292 釘住 0x1E292 的上限與經驗歸零：
//
//	0x1E2E0 cmp edx,0x1E / 0x1E2E5 cmp edx,0x1F → 機兵比 0x63（99），其餘比 0x28（40）
//	0x1E2F2 je 0x1317D                          → 相等就整段返回，連經驗都不加
//	0x1E3EE sub [esp+4],0x64                     → 每升一級扣 100
//	0x1E403 cmp eax,0x63 / 0x1E4F6 cmp eax,0x1E  → 機兵到 99、任何人到 30，剩下的經驗歸零
//
// 修改指南 references/text/modify1.md 第 11～13 條改的正是這幾個常數，其中 30 級那條
// 被它稱為原版錯誤；忠實模式照原版。
func TestNativeLevelCapFollows1E292(t *testing.T) {
	rows, err := LoadNativeGrowthRows("../../../docs/data/exe_tables/growth.json")
	if err != nil {
		t.Fatal(err)
	}
	st := &State{NativeGrowthRows: rows}
	rng := rand.New(rand.NewSource(1))
	unit := func(fig, lv int, exp float64) *Unit {
		return &Unit{Camp: Own, Lv: lv, Exp: exp, HP: 50, MaxHP: 50,
			BattleFig: fig, HasBattleFig: true}
	}

	cases := []struct {
		name         string
		fig, lv      int
		exp, award   float64
		wantAccepted float64
		wantLv       int
		wantExp      float64
	}{
		{"一般單位在 40 級整段返回", 4, 40, 10, 50, 0, 40, 10},
		{"一般單位升到 40 級不歸零", 4, 39, 90, 50, 50, 40, 40},
		{"升到 30 級剩下的經驗歸零（原版錯誤）", 4, 29, 90, 50, 50, 30, 0},
		{"機兵過了 40 級照常收經驗", 0x1e, 40, 10, 50, 50, 40, 60},
		{"機兵升到 30 級同樣歸零", 0x1f, 29, 90, 50, 50, 30, 0},
		{"機兵升到 99 級歸零", 0x1e, 98, 90, 50, 50, 99, 0},
		{"機兵在 99 級整段返回", 0x1f, 99, 0, 50, 0, 99, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			u := unit(c.fig, c.lv, c.exp)
			accepted, _ := st.AwardExp(u, c.award, rng)
			if accepted != c.wantAccepted || u.Lv != c.wantLv || u.Exp != c.wantExp {
				t.Fatalf("收下 %v lv %d exp %v，應為 %v／%d／%v",
					accepted, u.Lv, u.Exp, c.wantAccepted, c.wantLv, c.wantExp)
			}
		})
	}

	// 0x1E2D6 test [esi+5],1：死亡旗標在的單位不收經驗。
	dead := unit(4, 5, 0)
	dead.HP = 0
	dead.NativeRecordByte5, dead.HasNativeRecordByte5 = 1, true
	if accepted, events := st.AwardExp(dead, 50, rng); accepted != 0 || events != nil || dead.Exp != 0 {
		t.Fatalf("陣亡單位收下 %v、exp %v", accepted, dead.Exp)
	}

	// 沒有原版 +7 的單位（重製端自訂資料）不套上限。
	legacy := &Unit{Camp: Own, Lv: 40, Exp: 10, HP: 50, MaxHP: 50}
	if accepted, _ := st.AwardExp(legacy, 50, rng); accepted != 50 || legacy.Exp != 60 {
		t.Fatalf("無 +7 的單位收下 %v、exp %v", accepted, legacy.Exp)
	}
}

// TestCappedAttackReportsNoExperience：到上限時攻擊結果的 ExpGained 必須是 0，
// 否則畫面會照樣附上「得到經驗值」，而原版在上限時整段不顯示。
func TestCappedAttackReportsNoExperience(t *testing.T) {
	rows, err := LoadNativeGrowthRows("../../../docs/data/exe_tables/growth.json")
	if err != nil {
		t.Fatal(err)
	}
	st := &State{NativeGrowthRows: rows}
	capped := &Unit{Camp: Own, Lv: 40, HP: 50, MaxHP: 50, BattleFig: 4, HasBattleFig: true}
	accepted, events := st.AwardExp(capped, 30, rand.New(rand.NewSource(1)))
	if accepted != 0 || events != nil {
		t.Fatalf("上限單位收下 %v、升級 %d 次", accepted, len(events))
	}
}
