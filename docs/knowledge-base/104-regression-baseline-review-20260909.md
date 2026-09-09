# 104 — 回歸基線的 39 項失敗重新盤點（2026-09-09）

`regression-baseline-20260909` 的 39 項失敗大多不是功能缺陷，而是**手工組出來的
測試 Game 缺語系資料**。連同語言包字串、指令環幀數與第一關視圖一起修完之後
剩 9 項，每一項都有各自的成因。

## 主因：`&Game{}` 沒有語系

`loadGame()` 一定會設 `localeID` 並載入 catalog／content／entities；手工
`&Game{…}` 不會，於是 `localeID` 是空字串。空字串不等於繁中：

| 生產端檢查 | 空 `localeID` 的後果 |
|---|---|
| `renderLocalizedNativeBattlePanel`：`localeID == "zh-Hant"` 才走原版面板 | 走本地化分支，缺 `localeEntities`／字型就失敗即關閉 |
| `localizedStoryText`：`localeID == "zh-Hant"` 且無 `line_id` 才回退 `line.Text` | 回報「story line lacks canonical line_id」 |
| `localeMessage`：需要 `localeCatalog` | 回報「official catalog is unavailable」，`g.msg` 留空 |

測試因此停在「語系缺件」，而不是它要驗的那條功能路徑。修法是共用的
`attachOfficialLocale(t, g)`（`remake/cmd/fd2/locale_fixture_test.go`）：設
`zh-Hant` 並載入三份官方資料，跟正式路徑一致。

這一項就修掉 24 個。

## 次因：存檔訊息的期待值沒跟著語言包走

語言包的 `save.saved` 是 `已存檔（槽位 %d：%s）`（全形括號、槽位後有空格），
12 處測試還寫著舊的半形形式。語言包是來源，測試跟著改。

## 第三項：指令環展開的最後一幀

`TestNativeActionOffsetXYMatchesFinalOpenFrame` 期待三倍差值，實作給四倍。
以 Capstone 讀 `0x1741C` 判定**實作是對的**：

```text
0x1746C..0x17483  四個位移初始化為 0x390
0x1757C           cmp [esp+0x14], 4 ; jge 離開
0x17587..0x17598  sub 0x8E8 / sub 6 / add 6 / add 0x8E8
0x175A0           call 0x17643
0x174F9..0x17537  貼四格（esi 0..3）
0x17578           inc [esp+0x14]
```

計數器 0..3 共四張呈現幀，而**每一張都先加過差值**，所以是一到四倍；最後
一張是四倍 `(0,-18) (-24,2) (24,2) (0,22)`。期待值改成四倍並註明出處。

## 剩下的 9 項

| 測試 | 成因 |
|---|---|
| `TestChapter25PostMaterializesSlot70JoinsPartyAndReachesTown26SaveBoundary` | 手工 fixture 缺 HUD／drawable selector 狀態，原生對話組幀失敗即關閉 |
| `TestChapterTwentyOneSkyKeyBattleResultReachesTownAndSaveBoundary` | 先卡在 `story_ch21_post_sky_key_intro` 的視圖界線（見下），繞過之後是 `0x4DFCC` 相對循環 start=7 得 12、期待 11 |
| `TestNativeEvent61AttackWaitsForPresentationCompletion` | 演出結束後 selector1 事件沒有被交還 |
| `TestReviewedChapterTwoCampaignTranslationsAndItemEntities` | 英文審核稿與語言包內容不一致 |
| `TestReviewedGoCandidatesMatchCurrentInventory` | 字串盤點的 review binding sha 過期 |
| `TestChapter1Turn3JoinsHanoBeforeSpawningHisGroup` | 哈瓦特陣營改成 `own` 之後，測試仍期待 allied NPC |
| `TestCompileChapter27PostMapsFDTXT028StringSeven` | ch27_post 對白編譯出 ch28.json 的第 11–15 句 |
| `TestChapterThreePostBattleSpeakerControlCodes` | ch03 scene1 說話者 77 解成「刺客隊長」，期待「約」 |
| `TestSeparatedNativeMapHUDFramesMatchFixedArchive` | 分離包 HUD 面板與固定封存不同 |

前兩項是 fixture 與流程缺口，後面幾項各自需要原版證據才能判誰對誰錯；
本輪不憑測試或資料任一側單方面改結論。

## `battle_ch01` 的視圖：兩條入口，兩組值

`campaign_full.json` 的節點常數是 `(0,13)`／`(7,14)`／`(7,1)`，
`TestFullCampaignCarriesVerifiedChapterOneNativeMapRuntime` 原本釘
`(1,13)`／`(8,17)`／`(7,4)`。兩組都有出處，但**指的不是同一條入口**：

| 入口 | 值 | 出處 |
|---|---|---|
| START（走完 ch00 handler，聚焦 slot 0）| `(0,13)`／`(7,14)`／`(7,1)` | `TestCh00CompiledHandlerCarriesItsExactRuntimeRosterIntoChapterOne` 以完整 ch00→ch01 交接實跑 |
| CONTINUE（隨遊戲附帶的 `FD2.SAV`）| `(1,13)`／`(8,17)`／`(7,4)` | dosgolem 收據 [fd2-move-confirm-cursor-20260909.json](../data/ui-traces/fd2-move-confirm-cursor-20260909.json) |

存檔會帶自己的視圖，所以兩者不同是合理的。節點常數跟著 START——那一條有
完整路徑的實跑在背書——靜態測試改成同一組並在原處註明 CONTINUE 的觀測。
要判定節點常數是否也該覆蓋 CONTINUE，得再取一份「同一節點、兩條入口」的
原版收據；本輪沒有。

## 工具鏈

`fd2-go-test-local` 的 `xvfb-run` 會偶發卡死（SIGUSR1 交握落在 `wait` 之前），
映像改內建 [`with-xvfb`](../../tools/docker/with-xvfb.sh)，回歸改由受版控的
[`tools/remake_go_test.sh`](../../tools/remake_go_test.sh) 驅動，它會直接跟
基線做差異比對。細節見 [96](96-parity-toolchain-20260909.md)。

## 拿掉恆等式之後新浮出來的一項

`validateNativeMapView` 改成檢查 13×8 視窗界線之後，`story_ch21_post_sky_key_intro`
的視圖被擋下來：它的可見游標落在視窗外。舊的恆等式檢查抓不到這種狀態——
只要 `cursor = camera + visible` 成立，visible 再大都會通過，而
`syncStoryNativeMapPanView` 正是用那條式子反推 cursor，所以恆等式永遠成立。

視窗界線是有出處的：消費端 `0x1741C` 以 `visible_x * 24 + visible_y * 24 * 0x1C8`
把可見游標當成 13×8 視窗內的格座標，超出就畫到視窗外。這一項在該測試修好
之前不會單獨浮出來，暫記於此。
