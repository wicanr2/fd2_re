# 100 — 對白出現時的重繪路徑與畫面幾何（2026-09-09）

使用者回報第 7 項：對話框出現時整個畫面晃一下，上下左右出現黑邊。用逐幀
擷取加 `-eip-watch` 量原版，兩個問題一次答完。

## 收據來源

原版執行器為 dosgolem `apps/fd2/cmd/oracle`（commit `f627cd1`），由
[`tools/dosgolem_oracle.sh`](../../tools/dosgolem_oracle.sh) 驅動。固定
`FD2.EXE`：357074 位元組，MD5 `b97caf2239a27a896069d03549d96e1e`，
SHA-256 `222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f`。
收據見
[fd2-dialogue-viewport-20260909.json](../data/ui-traces/fd2-dialogue-viewport-20260909.json)。

入口是標題選單的 START，以最快速率送 Enter 走完序幕，取第一關的戰場對白。

## 幾何：地圖沒有位移，黑邊也不是新出現的

每張 320×200 索引畫面量四邊各有幾列／幾行完全同色（索引 0 即黑）：

| 畫面 | 上 | 下 | 左 | 右 |
|---|---|---|---|---|
| 對白尚未出現 | 4 | 4 | 4 | 4 |
| 對白框在畫面下半 | 4 | **2** | 4 | 4 |
| 對白結束 | 4 | 4 | 4 | 4 |
| 玩家取得操作權 | 4 | 4 | 4 | 4 |

**原版戰場畫面本來就有 4 px 黑邊**（312×192 的地圖窗格置於 (4,4)）。對白出現
時上／左／右完全不變，只有對白框那一側縮成 2 px——那是對白框畫到邊界上，
不是地圖被縮小。同一張畫面上半部的地形位置與對白前逐格相同。

## 路徑：對白不換世界層

`-eip-watch` 以兩次對白等待（`0x16CF3`）之間為一個循環加總：

| 位址 | 角色 | 每個對白循環 |
|---|---|---|
| `0x11CAC` | 整幀排程 | 1 |
| `0x122DC` | 範圍／游標圖示 | 1 |
| `0x11EEE` | 地形 | 1 |
| `0x127A9` | 前景 | 1 |
| `0x127E0` | 單位繪製公式 | 11 |
| `0x24618` | 轉場整幀 | 0 |

十四個循環都一樣。**對白期間地圖仍走 `0x11CAC` 這條重繪路徑，層集合與平時
相同**，對白框是疊在上面的另一層。原版沒有為了對白換一條世界層管線。

## 重製端的差異與修正

`legacyViewport` 為 `g.storyBG || g.battleEvent != nil`。戰場對白讓
`battleEvent` 非空，於是 `nativeMapFrameAdmission` 回 false，世界層從原生整幀
切到 320×200 離屏再放大的正規化管線。兩條管線的原點與邊界處理不同，切換的
那一幀整個畫面就會跳一下，黑邊幾何也跟著換。

現在只有 `storyBG` 場景背景才交給正規化管線；戰場對白保留原生整幀，對白框
照舊畫在上面。這與原版的層次相同：地圖層不動，對白是上面的一層。

## 2026-09-14 storyBG 同狀態補證

王座廳第一句已由目前 dosgolem `apps/fd2/cmd/oracle`（commit
`5c607a70bbd8843a5b63d9a92ebef63aeea6512e`）從未修改的 normal START 路徑
重生。原版 control boundary 12（165,473,882 steps）與重製決定性 frame 362
同為鏡頭格 `(3,20)`、焦點 `(8,21)`；21 筆角色的 slot／FIG／格座標全部一致，
重製端另固定為 `story_ch00_handler` beat 4、來源 `0x32382`、
`FDTXT_033#0` utterance 0。輸入腳本、每個 control boundary 的 PNG／狀態雜湊、
runner 與比較結果見
[`storybg-dialogue-original-vs-remake-e1.json`](../data/ui-traces/storybg-dialogue-original-vs-remake-e1.json)。

dosgolem `parity` 將重製 `640×400` 以明示的 `nearest_2x` 正規化至
`320×200`；同相位全畫面為 63,518／64,000 像素相同（99.246875%，RGB 平均
絕對誤差 0.63049）。`y=112..199` 的對白 overlay 28,160 個像素完全一致，
上、左、右邊界也各自完全一致；482 個差異像素全落在 `y=21..111` 的人物
sprite。原版兩個穩定 control boundary 與重製三個 `frame/8 % 3` sprite 相位
另做六組交叉比較，六張差異遮罩的 SHA-256 完全相同，排除把不同動畫時點硬湊
成 `same-state`。這關閉第一句 lower／left storyBG 對白框、文字與視窗邊界的
同狀態 `RUNTIME-E1`，但不把上半部人物 sprite 差異說成逐像素一致。

同一原版正常讀鍵鏈在第二次 Enter 後由一筆 BIOS read 推進，焦點由 `(8,21)`
移至國王 `(7,5)`、鏡頭格移至 `(3,4)`；重製端回歸
`TestStoryBGFirstDialogueEnterAdvancesToKing` 從完整戰役 runtime 起點重播，並由
production `handleNativeStoryInput` 到達相同的下一位說話者。原版雖是未修改
normal START 玩家路徑，重製端仍使用決定性截圖鉤子，故本切片不冒稱完整
`PLAYER-E2`。

## 尚未涵蓋

- 對白框本身的繪製位址與版面。本輪只量地圖層與幾何。
- 其他 `storyBG` 說話者、upper／right 版面、控制碼與完整戰役逐場對拍。第一句
  lower／left 已由上述 2026-09-14 收據關閉為同狀態 `RUNTIME-E1`；不能外推成
  所有故事對白已完成。
- `0x1AD72`（HUD）在這一段為 0，是因為該時點 HUD 尚未啟用，**不代表對白會
  關掉 HUD**。
