# 104 — 回歸基線的 39 項失敗重新盤點（2026-09-09）

`regression-baseline-20260909` 的 39 項失敗大多不是功能缺陷，而是**手工組出來的
測試 Game 缺語系資料**。連同語言包字串、指令環幀數與第一關視圖一起修掉 30 項，
剩下的 9 項各有成因，也已逐項閉合：以
`tools/remake_go_test.sh <素材根>` 對 `./...` 重跑，**39 項全部修復、沒有新增失敗**。

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

以 `./cmd/fd2` 單獨量：只改這一項，該套件的失敗由 34 降到 8，也就是**修掉 26 個**。

## 次因：存檔訊息的期待值沒跟著語言包走

語言包的 `save.saved` 是 `已存檔（槽位 %d：%s）`（全形括號、槽位後有空格），
**12 處**測試還寫著舊的半形形式。語言包是來源，測試跟著改；這 12 處對應 3 個
先前失敗的測試（其餘 9 處在更早就被別的原因擋住，改完才會走到）。

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

## 各自成因的 9 項

| 測試 | 成因 | 修法 |
|---|---|---|
| `TestChapter25PostMaterializesSlot70JoinsPartyAndReachesTown26SaveBoundary` | 原生對話會把游標移到說話者所在格，HUD 因此要求那一格單位的完整 raw 出處；主角隊由可編輯腳本的 `party` 物化時沒有記錄 +0x42／+0x46 | `Scenario.PartyUnits` 沿用持久名冊、JOIN 與待登場三個建構子的同一條關係，把已授權的最大 HP／MP 標記為 +0x42／+0x46 |
| `TestChapterTwentyOneSkyKeyBattleResultReachesTownAndSaveBoundary` | `0x4DFCC` 是 process-global 相位，取樣點之後還會被推進 | 期待值寫成 68 次演出 FirstFrames ＋ 1 次 pan 重繪，兩個常數各自具名 |
| `TestNativeEvent61AttackWaitsForPresentationCompletion` | 分離素材包沒有 `locales/`，戰鬥保護讀 entities 失敗即關閉 | fixture 改用疊上儲存庫語言包的符號連結根，與 `tools/remake_go_test.sh` 同一種作法 |
| `TestReviewedChapterTwoCampaignTranslationsAndItemEntities` | 英文審核稿 5 句與語言包不一致 | 語言包是來源，審核稿跟著改 |
| `TestReviewedGoCandidatesMatchCurrentInventory` | 字串盤點的 review 綁定過期（行號漂移＋新增候選）| 重生盤點後以 `tools/migrate_string_review.py` 依簽章遷移，再分類新候選（見下）|
| `TestChapter1Turn3JoinsHanoBeforeSpawningHisGroup` | 舊斷言期待哈瓦特是 allied NPC | map0 建構資料的 group7 raw `+6 = 2`、ch01 的 spawn_group 也寫 `own`，測試改成 `Own` |
| `TestCompileChapter27PostMapsFDTXT028StringSeven` | 舊斷言期待一個 `count=5` 的群組拍 | binding 為每一行各自帶 `native_dialogue`（utterance 0..4），編譯結果是五個 dialog 拍；折成群組會丟掉逐行版面 |
| `TestChapterThreePostBattleSpeakerControlCodes` | 說話者 77 的舊期待值「約」是把 speaker id 當字模索引讀出來的 | 同場景另外四句都寫刺客隊長，期待值改成「刺客隊長」|
| `TestSeparatedNativeMapHUDFramesMatchFixedArchive` | 比的是內部表示：封存經 `ParseSingleFrame` 帶 0x4E63D 的 RLE 位元組，分離包帶已展開的 Indexed+Mask | 兩邊各自 `Blit` 到同一張底圖再逐位元組比 |

## `battle_ch01` 的視圖：兩條入口，兩組值

`battle_ch01` 的六個視圖全域有兩組值，兩組都有出處，但**指的不是同一條入口**：

| 入口 | 值 | 出處 |
|---|---|---|
| START（走完 ch00 handler，聚焦 slot 0）| `(0,13)`／`(7,14)`／`(7,1)` | `TestCh00CompiledHandlerCarriesItsExactRuntimeRosterIntoChapterOne` 以完整 ch00→ch01 交接實跑 |
| CONTINUE（隨遊戲附帶的 `FD2.SAV`）| `(1,13)`／`(8,17)`／`(7,4)` | dosgolem 收據 [fd2-move-confirm-cursor-20260909.json](../data/ui-traces/fd2-move-confirm-cursor-20260909.json) |

存檔會帶自己的視圖，所以兩者不同是合理的。`campaign_full.json` 的節點常數與
`TestFullCampaignCarriesVerifiedChapterOneNativeMapRuntime` 都跟著 START，因為
那一條有完整路徑的實跑在背書；CONTINUE 的觀測在測試原處註明。要判定節點常數
是否也該涵蓋 CONTINUE，得再取一份「同一節點、兩條入口」的原版收據。

## 工具鏈

`fd2-go-test-local` 的 `xvfb-run` 會偶發卡死（SIGUSR1 交握落在 `wait` 之前），
映像改內建 [`with-xvfb`](../../tools/docker/with-xvfb.sh)，回歸改由受版控的
[`tools/remake_go_test.sh`](../../tools/remake_go_test.sh) 驅動，它會直接跟
基線做差異比對。細節見 [96](96-parity-toolchain-20260909.md)。

## 視窗界線取代恆等式

`validateNativeMapView` 檢查的是 13×8 視窗界線，不是
`visible = cursor − camera` 的恆等式。界線有出處：消費端 `0x1741C` 以
`visible_x * 24 + visible_y * 24 * 0x1C8` 把可見游標當成視窗內的格座標，
超出就畫到視窗外。恆等式則沒有出處——原版沒有任何一處由 `cursor − camera`
重算可見游標（寫入端清單見
[`fd2_visible_cursor_writers_ida.txt`](../data/ida/fd2_visible_cursor_writers_ida.txt)），
鏡頭捲過游標之後那個減法會給出負值。

換成界線之後，`stepFocusUnit` 與 `nativeFocusEndpoint` 都改成沿用已追蹤的
可見游標；只有「完全沒有那份狀態」的直接進場 renderer 才走
`nativeMapFocusVisibleSeed` 把 `cursor − camera` 夾回視窗。四項錯誤訊息也各自
分開（欄位太小／鏡頭出界／游標出界／可見游標出界），失敗時直接指出是哪一項。

**仍未閉合**：劇情 pan 的 `syncStoryNativeMapPanView` 還是用
`cursor = camera + visible` 反推游標。原版的劇情捲動 `0x135DD` 不寫可見游標，
它對絕對游標做什麼尚未從指令解出，所以這條反推目前沒有寫入端證據撐著；界線
檢查抓不到它（反推出來的值一定落在視窗內）。要閉合得回到 `0x135DD` 本體看它
有沒有寫 `[0x53AB1]`／`[0x53AB5]`。

## 字串盤點的 review 怎麼跟上行號漂移

`docs/data/fd2-string-inventory.json` 是產生物（不進版控），`string_id` 內含
檔名與行列號，任何非測試 Go 檔的編輯都會讓它漂移。
`docs/data/fd2-string-review.json` 綁定該盤點的 SHA-256，所以流程固定是：

1. 所有會動到 Go 的修改先做完；
2. `go run ./cmd/fd2-string-inventory -repo /src -output docs/data/fd2-string-inventory.json`
   （另跑一次 `-summary` 更新摘要）；
3. `tools/migrate_string_review.py --old-inventory <上一份> --new-inventory <新的>
   --review <舊 review> --output <新 review>`，它以
   `(role, text, file, function)` 簽章對應新舊 ID，字串已刪除的用 `--drop-id` 指名；
4. 盤點新增的候選逐項分類。本輪 16 項：8 項失敗即關閉診斷歸
   `internal_diagnostic`、5 項（封裝自我檢查與 `FD2_SHOT_ATTACK` 截圖 fixture）歸
   `development`、3 項（視窗標題與兩個主題名）歸 `player_visible`。

上一份盤點必須留著才有辦法做簽章對應；覆蓋掉它就只能從更早的產生物重建。
