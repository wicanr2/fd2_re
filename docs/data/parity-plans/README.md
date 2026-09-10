# 對拍控制序列

這裡的 `.jsonl` 是原版側（dosgolem `apps/fd2/cmd/oracle`）的控制序列，由
[`tools/dosgolem_oracle.sh`](../../../tools/dosgolem_oracle.sh) 執行。收進版控是
因為收據要能重生：序列變了，收據就不是同一件事。

每行一個 JSON 物件，欄位見
[`96-parity-toolchain-20260909.md`](../../knowledge-base/96-parity-toolchain-20260909.md)。

| 檔案 | 走到哪 | 重製端對應 |
|---|---|---|
| `ch01-move-attack.jsonl` | 第一關第 1 回合：選 (8,16) 的亞雷斯、移動到 (6,19)、攻擊 (5,19) 的盜賊 | `TestDumpChapterOneMoveAttackFrames`（`FD2_ATTACK_DUMP`） |

兩側的輸入層不同——原版送 BIOS 按鍵，重製端直接驅動同一條狀態機——但走過的節點
與座標相同，逐幀畫面因此可以對照。座標寫死在兩邊：原版在序列的方向鍵次數裡，
重製端在測試檔頂端的常數裡，改一邊就要改另一邊。
