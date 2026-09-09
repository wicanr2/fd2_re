package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// 依 dosgolem 原版收據（docs/data/ui-traces/fd2-map-walk-pose-20260909.json）：
// 走行途中 raw +3 姿態是移動方向、+4 動作每格 1..6；整段抵達時兩者都回到 0，
// 並保持到玩家下一個指令。重製端先前停在最後一格的行走方向。
func TestBattleWalkArrivalRestoresNativeRestPose(t *testing.T) {
	cases := []struct {
		name     string
		path     []battle.Cell
		walkPose int
	}{
		{"往上兩格", []battle.Cell{{X: 8, Y: 16}, {X: 8, Y: 15}, {X: 8, Y: 14}}, 2},
		{"往左兩格", []battle.Cell{{X: 8, Y: 16}, {X: 7, Y: 16}, {X: 6, Y: 16}}, 1},
		{"往右一格", []battle.Cell{{X: 8, Y: 16}, {X: 9, Y: 16}}, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			unit := &battle.Unit{HasNativeMapPresentation: true}
			start := c.path[0]
			if !unit.SetMapPlacement(start.X, start.Y, nativeMapRestPose) {
				t.Fatal("無法建立起始位置")
			}
			g := &Game{
				m:  &MapData{W: 32, H: 32, TileW: 24, TileH: 24},
				st: &battle.State{W: 32, H: 32, Units: []*battle.Unit{unit}},
			}
			g.walk = &walkAnim{u: unit, path: c.path, then: func() {}}

			sawWalkPose := false
			for steps := 0; g.walk != nil && steps < 64; steps++ {
				g.stepBattleWalk()
				if g.walk != nil && unit.NativeMapPresentation.Pose == byte(c.walkPose) {
					sawWalkPose = true
					if m := unit.NativeMapPresentation.Motion; m > 6 {
						t.Fatalf("行走中 +4 動作 %d 超出原版的 1..6", m)
					}
				}
			}
			if g.walk != nil {
				t.Fatal("移動沒有在有界步數內完成")
			}
			if !sawWalkPose {
				t.Fatalf("行走途中沒有出現移動方向姿態 %d", c.walkPose)
			}
			last := c.path[len(c.path)-1]
			got := unit.NativeMapPresentation
			if int(got.X) != last.X || int(got.Y) != last.Y {
				t.Fatalf("終點座標 (%d,%d)，預期 (%d,%d)", got.X, got.Y, last.X, last.Y)
			}
			if got.Pose != nativeMapRestPose || got.Motion != 0 {
				t.Fatalf("抵達後 raw +3/+4 = %d/%d，原版為 %d/0",
					got.Pose, got.Motion, nativeMapRestPose)
			}
			if unit.Dir != nativeMapRestPose {
				t.Fatalf("抵達後導覽用 Dir = %d，預期 %d", unit.Dir, nativeMapRestPose)
			}
		})
	}
}
