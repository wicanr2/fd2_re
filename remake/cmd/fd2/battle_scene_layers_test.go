package main

import "testing"

// TestBattleSceneDrawsPedestalRightBeforeOwnFigure 釘住「台座跟我方走」。
//
// 先前順序固定是守方 → 台座 → 攻方，於是敵方攻擊我方時我方成了守方、被畫在
// 台座之前，腳步被台座蓋住。原版兩種攻守方向下我方都踩在台座上。
func TestBattleSceneDrawsPedestalRightBeforeOwnFigure(t *testing.T) {
	cases := []struct {
		name   string
		atkOwn bool
		own    battleSceneLayer
	}{
		{"我方攻擊", true, battleLayerAttackerFigure},
		{"我方被攻擊", false, battleLayerDefenderFigure},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			order := battleSceneLayerOrder(c.atkOwn)
			pedestal, own := -1, -1
			seen := map[battleSceneLayer]bool{}
			for i, layer := range order {
				if seen[layer] {
					t.Fatalf("圖層 %d 重複出現：%v", layer, order)
				}
				seen[layer] = true
				switch layer {
				case battleLayerOwnPedestal:
					pedestal = i
				case c.own:
					own = i
				}
			}
			if len(seen) != 3 {
				t.Fatalf("三個圖層沒有各出現一次：%v", order)
			}
			if pedestal < 0 || own < 0 {
				t.Fatalf("找不到台座或我方 figure：%v", order)
			}
			if own != pedestal+1 {
				t.Fatalf("我方 figure 在第 %d 層、台座在第 %d 層；台座必須緊接在我方之前", own, pedestal)
			}
		})
	}
}
