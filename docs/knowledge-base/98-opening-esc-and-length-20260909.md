# 98 — 開場的 ESC 與長度（2026-09-09）

使用者回報第 1 項是「開場過長，沒有辦法按下 ESC 即時跳到下一個」。這是兩個
問題：原版的 ESC 到底做什麼，以及原版的開場有多長。兩題各取了一份收據，
答案不一樣——ESC 是真缺陷，長度不是。

## 收據來源

原版執行器為 dosgolem `apps/fd2/cmd/oracle`，由
[`tools/dosgolem_oracle.sh`](../../tools/dosgolem_oracle.sh) 驅動。固定
`FD2.EXE`：357074 位元組，MD5 `b97caf2239a27a896069d03549d96e1e`，
SHA-256 `222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f`。

- ESC 三臂對照：[fd2-opening-esc-20260909.json](../data/ui-traces/fd2-opening-esc-20260909.json)
- 開場長度：[fd2-opening-length-20260909.json](../data/ui-traces/fd2-opening-length-20260909.json)

入口是標題選單的初始選項 START，直接送 Enter 進入序幕，不使用隨遊戲附帶的
存檔，也沒有狀態注入。

## ESC 在原版做什麼

三臂只差在送的鍵，步數、順序、起點完全相同；每格兩千萬指令（約 20 虛擬秒），
遠長於逐字寫入，所以不會把「還沒寫完」誤讀成「不接受按鍵」。

| 臂 | 送的鍵 | 結果 |
|---|---|---|
| esc | 每格一次 ESC | 19 個控制邊界的畫面 PNG |
| enter | 每格一次 Enter | 與 esc 臂**逐格 sha256 相同** |
| none | 不送鍵 | 第 4 個邊界起與前兩臂分岔，第 8 個邊界起連續 11 格完全相同（約 220 虛擬秒畫面不動）|

兩個結論：**原版的故事等待接受 ESC，且與 Enter 完全同義**；而且 **ESC 不是
跳過整段的捷徑**——若會跳段，esc 臂應該領先 enter 臂，實際上兩臂逐格相同。

## 開場有多長

長度那一臂改用「上一下被吃掉就立刻再按」的最快速率：每格 30 萬指令送一次
Enter，但只在 BIOS 環形緩衝已被遊戲取空時才送。這個閘門是必要的——固定速率
送鍵會在遊戲消化不及時撐爆 15 格緩衝，`Enqueue` 直接回「緩衝已滿」。

oracle 的時間模型是每條指令 1 微秒，虛擬秒即指令數除以一百萬。

| 里程碑 | 虛擬秒 | 已被取走的鍵數 |
|---|---|---|
| 送出 START 的 Enter | 0 | 0 |
| 戰場對白出現在畫面上 | 179.7 | 108 |
| 戰場 HUD 出現，玩家取得操作權 | 187.2 | 122 |

自 cp630 起每一格送出的鍵都立刻被取走，代表已離開故事等待、進入選單輪詢；
那之後的 Enter 是在翻選單，不屬於開場。

重製端量同一段：以每幀都嘗試推進（玩家能達到的最快速率）跑第 0 章，到 ch01
的 `spawn_intro` 拍為止是 **7298 幀（121.6 秒）、101 次有效確認**。

所以**開場長度本身沒有退化**：原版也要按約 120 次、走約 187 虛擬秒，重製端
是 101 次、122 秒。使用者感覺過長，是因為 ESC 按下去沒有反應，只剩 Enter
一條路——而原版兩條路都通。

> 虛擬秒不等於實機牆鐘。每指令 1 微秒約當 1 MIPS，遊戲中不由計時器節流的
> 段落在實機上會更快。跨側比較以**按鍵次數**為準，時間只作量級參考。

## 重製端的修正

`campInput` 的 story 與 cutscene 分支只把 Enter／Space 交給
`handleNativeStoryInput`，Escape 完全沒有進入這條路徑。現在
`nativeStoryInput` 多一個 `escape` 欄位，由 `advance()` 與 `enter` 合流；
兩個分支都填上 `inpututil.IsKeyJustPressed(ebiten.KeyEscape)`。

回歸 `TestNativeStoryAdvanceAcceptsEscapeLikeEnter` 對三種輸入各驗一次：
Enter 與 ESC 的狀態轉移完全相同，不送鍵則停在原句——對應收據的三臂。

## 對拍工具的兩項擴充

- oracle 的 checkpoint 與 current 加上 `kbd_pending`（BIOS 環形緩衝
  0x41a／0x41c 的頭尾差）與 `kbd_reads`（遊戲實際取走的鍵數）。有了前者才
  能問「原版走完這段要按幾次」而不撐爆緩衝；後者是有效推進次數，和送出的
  鍵數不是同一件事。
- 驅動的控制序列多三個選用欄位：`repeat` 重複本行、`gate: kbd_empty` 只在
  緩衝取空時送鍵、`until: units_present` 提前結束。1400 格的計畫因此是兩行。

`until: units_present` 這一輪沒派上用場：單位陣列在開場就已經有內容，不能
用來判斷「已經進戰場」。判斷戰場開始目前只能看畫面。

## 尚未涵蓋

- 戰鬥中的對白（battle event、回合起手）是否同樣接受 ESC。本輪只取了故事
  對白的樣本，`handleBattleEventDialogueInput` 與戰鬥起手對白仍只收
  Enter／Space。
- 標題選單與各式選單對 ESC 的反應。
- ESC 之外的鍵是否也推進。本輪只比對 ESC 與 Enter 兩個。
