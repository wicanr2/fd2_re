package battle

import "testing"

// 0x18f6a：mode=row+0xc 當 flood 預算、radius=row+0xb 排除內圈，落在
// 0x4dbfc 基底旗標上。射程 1..2 的武器：距離 0 的自己格 0xff，1..2 有預算值。
func TestNativePlayerAttackTargetFieldMarksWeaponDiamond(t *testing.T) {
	const w, h = 7, 7
	s := &State{W: w, H: h, NativeCompositionEventBytes: make([]byte, w*h)}
	rows := make([]byte, 0x40*NativeItemEffectRowSize)
	rows[0x34*NativeItemEffectRowSize+0x0b] = 1
	rows[0x34*NativeItemEffectRowSize+0x0c] = 2
	if err := s.BindNativeFutureItemRows(rows); err != nil {
		t.Fatal(err)
	}
	u := &Unit{X: 3, Y: 3, Inventory: []int{0x34, 0xa4}, NativeInventoryFlags: []int{0x40, 0x40}}
	field, err := s.NativePlayerAttackTargetField(u)
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			d := absInt(x-3) + absInt(y-3)
			got := field[y*w+x]
			switch {
			case d == 0 || d > 2:
				if got != 0xff {
					t.Fatalf("(%d,%d) d=%d got %#x want 0xff", x, y, d, got)
				}
			default:
				if got != byte(2-d) {
					t.Fatalf("(%d,%d) d=%d got %#x want %#x", x, y, d, got, 2-d)
				}
			}
		}
	}
	if _, err := s.NativePlayerAttackTargetField(&Unit{X: 3, Y: 3}); err == nil {
		t.Fatal("沒有已裝備武器仍回傳射程")
	}
}
