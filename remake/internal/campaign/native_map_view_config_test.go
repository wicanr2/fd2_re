package campaign

import "testing"

// TestNativeMapViewConfigEntryContract 釘住「節點常數」與「執行期狀態」是兩套
// 契約。執行期的可見游標不被夾（原版八個寫入端都不檢查範圍，走行捲動會把它
// 寫到視窗外）；節點常數是進場即繪的靜止視圖，游標框與指令環立刻消費它，
// 所以要求視窗內、而且與 cursor-camera 一致。
func TestNativeMapViewConfigEntryContract(t *testing.T) {
	const w, h = 24, 24
	good := NativeMapViewConfig{
		CameraX: 1, CameraY: 13, CursorX: 8, CursorY: 17,
		VisibleCursorX: 7, VisibleCursorY: 4,
	}
	if err := good.Validate(w, h); err != nil {
		t.Fatalf("拒絕了合法的入口視圖：%v", err)
	}
	for _, c := range []struct {
		name string
		view NativeMapViewConfig
	}{
		{"可見游標出界", NativeMapViewConfig{
			CameraX: 1, CameraY: 13, CursorX: 8, CursorY: 17,
			VisibleCursorX: 13, VisibleCursorY: 4}},
		{"可見游標為負", NativeMapViewConfig{
			CameraX: 1, CameraY: 13, CursorX: 8, CursorY: 12,
			VisibleCursorX: 7, VisibleCursorY: -1}},
		{"可見游標與 cursor-camera 不一致", NativeMapViewConfig{
			CameraX: 1, CameraY: 13, CursorX: 8, CursorY: 17,
			VisibleCursorX: 7, VisibleCursorY: 3}},
		{"鏡頭越界", NativeMapViewConfig{
			CameraX: 1, CameraY: h - 7, CursorX: 8, CursorY: 18,
			VisibleCursorX: 7, VisibleCursorY: 1}},
		{"游標出場", NativeMapViewConfig{
			CameraX: 1, CameraY: 13, CursorX: w, CursorY: 17,
			VisibleCursorX: w - 1, VisibleCursorY: 4}},
	} {
		if err := c.view.Validate(w, h); err == nil {
			t.Fatalf("%s：入口視圖 %+v 被接受了", c.name, c.view)
		}
	}
	if err := good.Validate(12, 24); err == nil {
		t.Fatal("接受了小於 13×8 視窗的場地")
	}
}

// TestVersionedNodeViewsSatisfyTheEntryContract 讓受版控的節點常數自己驗一次。
func TestVersionedNodeViewsSatisfyTheEntryContract(t *testing.T) {
	data, err := Load("../../assets/scenarios/campaign_full.json")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for id, node := range data.Nodes {
		if node.NativeMapView == nil {
			continue
		}
		checked++
		view := *node.NativeMapView
		if view.VisibleCursorX != view.CursorX-view.CameraX ||
			view.VisibleCursorY != view.CursorY-view.CameraY ||
			view.VisibleCursorX < 0 || view.VisibleCursorX >= 13 ||
			view.VisibleCursorY < 0 || view.VisibleCursorY >= 8 {
			t.Fatalf("節點 %s 的入口視圖不符契約：%+v", id, view)
		}
	}
	if checked == 0 {
		t.Fatal("沒有節點帶 native_map_view，這個檢查等於沒跑")
	}
}
