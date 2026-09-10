package main

import (
	"math"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// TestPhaseBannerScheduleFollowsTheBiosTickContract 釘住回合橫幅的原版節奏：
// 進場 16 步、中段 delay()、退場 17 步，每一步一個 BIOS tick。證據見
// docs/knowledge-base/105-phase-banner-timing-20260910.md。
func TestPhaseBannerScheduleFollowsTheBiosTickContract(t *testing.T) {
	for _, tc := range []struct {
		banner string
		hold   float64
	}{
		{phaseBannerEnemyText, phaseBannerEnemyHoldMillis},
		{phaseBannerPlayerText, phaseBannerPlayerHoldMillis},
	} {
		if got := phaseBannerHoldMillis(tc.banner); got != tc.hold {
			t.Fatalf("%s 中段停留 %.0f 毫秒，收據是 %.0f", tc.banner, got, tc.hold)
		}
		want := indexedmap.PhaseBannerSteps*indexedmap.PhaseBannerStepMillis + tc.hold
		if got := phaseBannerTotalMillis(tc.banner); math.Abs(got-want) > 1e-9 {
			t.Fatalf("%s 全長 %.3f 毫秒，應為 %.3f", tc.banner, got, want)
		}

		g := &Game{banner: tc.banner, bannerT: phaseBannerFrames(tc.banner)}
		seen := make([]int, 0, indexedmap.PhaseBannerSteps)
		last := -1
		for g.bannerT > 0 {
			step := g.phaseBannerStep()
			if step < 0 {
				t.Fatalf("%s 橫幅未結束卻回 -1", tc.banner)
			}
			if step < last {
				t.Fatalf("%s 步驟由 %d 倒退到 %d", tc.banner, last, step)
			}
			if step != last {
				seen = append(seen, step)
				last = step
			}
			g.bannerT--
		}
		if len(seen) != indexedmap.PhaseBannerSteps {
			t.Fatalf("%s 走過 %d 步，收據是 %d 步",
				tc.banner, len(seen), indexedmap.PhaseBannerSteps)
		}
		for i, step := range seen {
			if step != i {
				t.Fatalf("%s 第 %d 個出現的步驟是 %d", tc.banner, i, step)
			}
		}
		if g.phaseBannerStep() != -1 {
			t.Fatalf("%s 橫幅結束後仍回報步驟", tc.banner)
		}
	}
}

// TestPhaseBannerHoldKeepsTheLastEnterStep 釘住中段停留期間畫面停在進場最後
// 一步（邊長 16），而不是先跳到退場的邊長 17。
func TestPhaseBannerHoldKeepsTheLastEnterStep(t *testing.T) {
	g := &Game{banner: phaseBannerPlayerText}
	total := phaseBannerFrames(phaseBannerPlayerText)
	enterMillis := indexedmap.PhaseBannerEnterSteps * indexedmap.PhaseBannerStepMillis
	hold := phaseBannerHoldMillis(phaseBannerPlayerText)
	for _, at := range []float64{enterMillis, enterMillis + hold/2} {
		g.bannerT = total - int(at*60/1000)
		if got, want := g.phaseBannerStep(), indexedmap.PhaseBannerEnterSteps-1; got != want {
			t.Fatalf("停留 %.1f 毫秒時是第 %d 步，應停在第 %d 步", at, got, want)
		}
	}
	// 幀是離散的（60 Hz ＝ 16.7 毫秒），所以要往後推一整幀才算離開停留段。
	g.bannerT = total - (int(math.Ceil((enterMillis+hold)*60/1000)) + 1)
	if got := g.phaseBannerStep(); got < indexedmap.PhaseBannerEnterSteps {
		t.Fatalf("停留結束後仍是第 %d 步，應進退場段", got)
	}
}
