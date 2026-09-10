# 對拍控制序列

這裡的 `.jsonl` 是原版側（dosgolem `apps/fd2/cmd/oracle`）的控制序列，由
[`tools/dosgolem_oracle.sh`](../../../tools/dosgolem_oracle.sh) 執行。收進版控是
因為收據要能重生：序列變了，收據就不是同一件事。

每行一個 JSON 物件，欄位見
[`96-parity-toolchain-20260909.md`](../../knowledge-base/96-parity-toolchain-20260909.md)。

| 檔案 | 走到哪 | 重製端對應 |
|---|---|---|
| `ch01-move-attack.jsonl` | 第一關第 1 回合：選 (8,16) 的亞雷斯、移動到 (6,19)、攻擊 (5,19) 的盜賊 | `TestDumpChapterOneMoveAttackFrames`（`FD2_ATTACK_DUMP`） |
| `ch01-phase-banner.jsonl` | 第一關第 1 回合：開系統面板、選 END、確認 YES，進敵方回合橫幅 | `phase_banner_glyph_test.go`（比的是字樣落點，不是整幀） |
| `ch01-clear.jsonl` | 第一關打到敵方全滅（閉環，見下） | 待補 |

## 開環與閉環

前兩份是**開環**：每一格寫死送哪一個鍵，座標藏在方向鍵次數裡。它只在「起點固定」
時成立，所以只能寫單一回合的短路徑。

`ch01-clear.jsonl` 是**閉環**：`sweep_battle` 每一步回讀 `current.json`，用
`units` 挑還沒行動的我方單位（record `+5` bit7）、算落腳格、送鍵之後再確認單位
座標真的變了。第二回合以後每個單位都在上一回合走到的位置，開環寫不出來——不是
麻煩而是做不到。移動成本受地形影響、可達範圍在狀態層看不到，所以可達與否一律
由「送 enter 之後座標變了沒」裁決，猜不到就試，試不成就換下一格。

閉環的決策邏輯在 [`tools/dosgolem_oracle_drive.py`](../../../tools/dosgolem_oracle_drive.py)，
純函式部分由 [`tools/test_dosgolem_oracle_drive.py`](../../../tools/test_dosgolem_oracle_drive.py)
逐條驗兩個方向：那些函式錯了不會噴錯，只會安靜地把按鍵送到錯的地方。

兩側的輸入層不同——原版送 BIOS 按鍵，重製端直接驅動同一條狀態機——但走過的節點
與座標相同，逐幀畫面因此可以對照。座標寫死在兩邊：原版在序列的方向鍵次數裡，
重製端在測試檔頂端的常數裡，改一邊就要改另一邊。

`ch01-phase-banner.jsonl` 這一組比的不是整幀：兩側的地圖底圖本來就不同，能對照
的是原版畫面上的字樣像素與落點。原版側用
[`tools/match_lmi1_glyph.py`](../../../tools/match_lmi1_glyph.py) 做模板匹配，
重製端側由 `remake/cmd/fd2/phase_banner_glyph_test.go` 綁住同一份字模與同一組
座標。
