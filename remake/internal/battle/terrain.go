// terrain.go — 地形攻防修正(doc02 §3.2、doc01 §5、doc11 AI 評分公式;worklist gap-audit
// doc42 第 5 項)。青衫攻略 notes.md 百分比表(三類地形):
//
//	一般(草地/平地)  AP +5%  DP -5%
//	森林             AP -5%  DP +10%
//	沼澤             AP -5%  DP -5%
//
// doc01 §5 用「移動代碼(byte1)」反組譯出同一組數字(modify2.md 靜態定義),兩份獨立
// 來源在 森林/沼澤 上完全吻合;唯獨「一般」代碼 0 的 DP 值,doc01 反組譯結果是 DP+0%,
// 與 notes.md 的 DP-5% 不一致(doc01 line ~119-122 已記錄此矛盾)。本檔採 notes.md/doc02
// 數字為準(依交付規格「公式照 doc02 青衫」),此為已知、誠實記錄的資料落差,非本檔疏漏。
package battle

// TerrainAPDPPct 回傳 (x,y) 這格的地形攻防修正百分比(doc02 §3.2)。
//
// 資料來源限制(誠實記錄,非臆測):目前唯一可用的 per-tile 地形資料是 s.Cost(MoveCost
// 用的步行成本,由 tools/export_engine_assets.py 從地形控制表 move_code 換算)。該換算
// 刻意把「森林」(move_code 2/3)與「正常」(move_code 0)都存成 cost=1(見
// docs/knowledge-base/01-container-and-asset-formats.md §5「remake 尚無騎兵/步行/飛行
// 兵種分類,先不罰步行」),所以本函式**無法從 cost 值分辨森林**,只能分辨
// 正常/沼澤/不可通行 三類。森林修正(AP-5%,DP+10%)需等 map.json 匯出管線補上獨立的
// 地形代碼欄位才能精確接上——那超出本輪授權範圍(只能動 internal/battle/*.go +
// tools/export_units.py,不碰 export_engine_assets.py/main.go),故 cost==1 一律先套
// 「正常地形」百分比,森林修正待後續資料管線補完後再收斂。
//
// s.Cost 為 nil(如舊測試直接手寫 State{})或座標越界:MoveCost 依其自身慣例回 1(平地),
// 本函式沿用同一慣例,一併套「正常地形」百分比(與 MoveCost 對 nil/越界的預設一致)。
func (s *State) TerrainAPDPPct(x, y int) (apPct, dpPct int) {
	switch s.MoveCost(x, y) {
	case 2: // 沼澤(export_engine_assets.py:move_code 4 -> cost 2)
		return -5, -5
	case 1: // 正常(含目前無法分辨的森林,見檔頭與本函式註解;s.Cost==nil/越界也落此分支)
		return 5, -5
	default: // 不可通行(cost>=BLOCKED_COST)等其他值:不套地形修正
		return 0, 0
	}
}
