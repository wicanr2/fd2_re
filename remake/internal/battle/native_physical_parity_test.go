package battle

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// native_physical_parity_test.go — 拿原版收據的數字驗傷害公式。
//
// 輸入全部來自 dosgolem 的 checkpoint（攻擊前那一格，step 368,483,500 的 80 位元組
// record）與地圖資料，不是挑出來湊答案的：
//
//	攻方 idx2 亞雷斯 (6,19)：HP 48、AP 26、DP 6、HIT 92、EV 2、職業 3（騎士）
//	守方 idx11 盜賊  (5,19)：HP 28、AP 24、DP 4、HIT 97、EV 2、職業 7（盜賊）
//	兩格的地形碼都是 0（map0 的 tiles×native_terrain_control 第 2 個位元組）
//	職業暴擊率：騎士 3、盜賊 0（0x5239B，見 native_combat_tables.json）
//
// 原版實測結果（docs/data/ui-traces/fd2-counterattack-20260910.json）：
// 守方 28→8（傷害 20），1,200,000 指令後攻方 48→31（反擊傷害 17）。
const (
	parityActorAP, parityActorDP  = 26, 6
	parityActorHit, parityActorEV = 92, 2
	parityActorCrit               = 3 // 騎士
	parityThiefAP, parityThiefDP  = 24, 4
	parityThiefHit, parityThiefEV = 97, 2
	parityThiefCrit               = 0 // 盜賊
	parityTerrainAPPct            = 5 // 地形碼 0 的 0x51A12[0]
	parityTerrainDPPct            = 0 // 地形碼 0 的 0x51A2A[0]

	parityOriginalMainDamage    = 20
	parityOriginalCounterDamage = 17
)

func parityMainRoll(state uint16) NativePhysicalRoll {
	return NativePhysicalRoll{
		AttackerAP: parityActorAP, DefenderDP: parityThiefDP,
		AttackerHit: parityActorHit, DefenderEV: parityThiefEV,
		AttackerCritPct:      parityActorCrit,
		AttackerTerrainAPPct: parityTerrainAPPct,
		DefenderTerrainDPPct: parityTerrainDPPct,
		RNGState:             state,
	}
}

func parityCounterRoll(state uint16) NativePhysicalRoll {
	// 反擊把攻守對調：盜賊打騎士，兩人站的格子不變，所以地形百分比也不變。
	return NativePhysicalRoll{
		AttackerAP: parityThiefAP, DefenderDP: parityActorDP,
		AttackerHit: parityThiefHit, DefenderEV: parityActorEV,
		AttackerCritPct:      parityThiefCrit,
		AttackerTerrainAPPct: parityTerrainAPPct,
		DefenderTerrainDPPct: parityTerrainDPPct,
		RNGState:             state,
	}
}

// TestNativeCounterDamageIsAlwaysSeventeen 釘住反擊那一擊：盜賊的職業暴擊率是 0，
// 而 9*(25−6)/10 ＝ 17 讓隨機加成的跨距只有 1，所以只要命中，傷害在**任何** RNG
// 狀態下都是 17——與原版實測的 48→31 完全相同。這是不靠挑 seed 的對拍。
func TestNativeCounterDamageIsAlwaysSeventeen(t *testing.T) {
	hits := 0
	for state := 0; state <= 0xFFFF; state++ {
		got, err := RollNativePhysicalDamage(parityCounterRoll(uint16(state)))
		if err != nil {
			t.Fatal(err)
		}
		if got.Missed {
			continue
		}
		hits++
		if got.Crit {
			t.Fatalf("狀態 %#x 暴擊了，但盜賊的職業暴擊率是 0", state)
		}
		if got.Damage != parityOriginalCounterDamage {
			t.Fatalf("狀態 %#x 的反擊傷害是 %d，原版實測是 %d",
				state, got.Damage, parityOriginalCounterDamage)
		}
	}
	if hits == 0 {
		t.Fatal("六萬多個狀態沒有一次命中，這支測試等於沒驗到東西")
	}
	// 命中率 97−2 ＝ 95%，允許 RNG 分布的偏差。
	if ratio := float64(hits) / 65536; ratio < 0.9 || ratio > 1.0 {
		t.Fatalf("命中比例 %.3f 偏離 95%% 太多", ratio)
	}
}

// TestNativeMainDamageMatchesTheOriginalReceipt 釘住主攻那一擊：未暴擊時
// 9*(27−4)/10 ＝ 20，隨機加成跨距 2，所以傷害只可能是 20 或 21；原版實測的
// 28→8 正是 20。暴擊會讓守方防禦減半而得到 22，與實測不符——也就是那一次沒暴擊。
func TestNativeMainDamageMatchesTheOriginalReceipt(t *testing.T) {
	seen := map[int]int{}
	for state := 0; state <= 0xFFFF; state++ {
		got, err := RollNativePhysicalDamage(parityMainRoll(uint16(state)))
		if err != nil {
			t.Fatal(err)
		}
		if got.Missed {
			continue
		}
		seen[got.Damage]++
	}
	for _, damage := range []int{parityOriginalMainDamage, parityOriginalMainDamage + 1} {
		if seen[damage] == 0 {
			t.Fatalf("沒有任何狀態算出傷害 %d", damage)
		}
	}
	for damage := range seen {
		switch damage {
		case parityOriginalMainDamage, parityOriginalMainDamage + 1, 22, 23:
			// 20/21 是未暴擊、22/23 是暴擊（守方防禦 4→2）。
		default:
			t.Fatalf("出現預期外的傷害 %d，公式或輸入有問題", damage)
		}
	}
	if seen[22]+seen[23] == 0 {
		t.Fatal("騎士暴擊率 3%% 卻一次都沒暴擊，暴擊那條沒被驗到")
	}
}

// TestNativeExchangeReproducesTheOriginalHPTimeline 找出能同時重現原版兩次結算的
// RNG 狀態：主攻 20、反擊 17，中間隔一次 sub_2939D 開頭的 3% 判定。找得到就表示
// 公式與 RNG 的消耗順序都與原版一致。
func TestNativeExchangeReproducesTheOriginalHPTimeline(t *testing.T) {
	matches := 0
	for state := 0; state <= 0xFFFF; state++ {
		// sub_2939D 開頭先擲 3% 決定要不要揮兩次；這裡只找單擊的那條。
		rng := fdother.NativeRNGStep(uint16(state))
		if int(rng)%100 < 3 {
			continue
		}
		main, err := RollNativePhysicalDamage(parityMainRoll(rng))
		if err != nil {
			t.Fatal(err)
		}
		if main.Missed || main.Damage != parityOriginalMainDamage {
			continue
		}
		rng = fdother.NativeRNGStep(main.RNGState)
		if int(rng)%100 < 3 {
			continue
		}
		counter, err := RollNativePhysicalDamage(parityCounterRoll(rng))
		if err != nil {
			t.Fatal(err)
		}
		if counter.Missed || counter.Damage != parityOriginalCounterDamage {
			continue
		}
		matches++
	}
	if matches == 0 {
		t.Fatal("找不到任何 RNG 狀態能同時給出原版的 20 與 17；" +
			"公式或 RNG 的消耗順序與原版不一致")
	}
	t.Logf("有 %d 個起始狀態重現原版的兩次結算（守方 28→8、攻方 48→31）", matches)
}
