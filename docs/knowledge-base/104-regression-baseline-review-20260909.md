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

劇情 pan 的規則見下一節；走行捲動 `0x13185` 整段結束時的發布仍是反推，
是目前這個家族唯一還沒有寫入端證據的一處。

## 字串盤點的 review 怎麼跟上行號漂移

`docs/data/fd2-string-inventory.json` 是產生物（不進版控），`string_id` 內含
檔名與行列號，任何非測試 Go 檔的編輯都會讓它漂移。
`docs/data/fd2-string-review.json` 綁定該盤點的 SHA-256，所以流程固定是：

1. 所有會動到 Go 的修改先做完；
2. `go run ./cmd/fd2-string-inventory -repo /src -output docs/data/fd2-string-inventory.json`
   （另跑一次 `-summary` 更新摘要）；
3. `tools/migrate_string_review.py --old-inventory <上一份> --new-inventory <新的>
   --review <舊 review> --output <新 review>`，它以
   `(role, text, file, function)` 簽章對應新舊 ID，字串已刪除的用 `--drop-id` 指名。
   同一個函式裡出現兩次的同一句話（例如 `FD2_SHOT_ATTACK` fixture 的
   「亞雷斯」「盜賊」各兩次）簽章相同，工具會回報「遷移候選數 2」而停下；
   這種要用 `--drop-id` 拿掉，再從新盤點按行列號補回去；
4. 盤點新增的候選逐項分類。本輪 16 項：8 項失敗即關閉診斷歸
   `internal_diagnostic`、5 項（封裝自我檢查與 `FD2_SHOT_ATTACK` 截圖 fixture）歸
   `development`、3 項（視窗標題與兩個主題名）歸 `player_visible`。

上一份盤點必須留著才有辦法做簽章對應；覆蓋掉它就只能從更早的產生物重建。

## 劇情 pan：游標跟著鏡頭走，不是由可見游標反推

`0x135DD` 的迴圈本體只有兩種寫法，X 與 Y 各一對：

```text
0x135F2  mov  [0x51A83], 0            ; overlay selector 關掉
0x135FC  cmp  esi, [0x53AA9]          ; 目標鏡頭 X 到了沒
0x13606  dec  [0x53AB1] / 0x1360C dec [0x53AA9]
0x13614  inc  [0x53AB1] / 0x1361A inc [0x53AA9]
0x13620  push 0 ; call 0x11CAC        ; 每格重繪一次
0x1362A  call 0x4E031
0x13631  cmp  edi, [0x53AAD]          ; 換 Y 軸，同一對寫法
0x13181  pop  edi ; pop esi ; pop ebx ; ret
```

**游標的位移量就是鏡頭的位移量**，整段沒有一條寫可見游標。
dosgolem 收據 [fd2-story-pan-cursor-20260909.json](../data/ui-traces/fd2-story-pan-cursor-20260909.json)
從 START 走完序章，把抓幀邊界設在 `0x11CAC`，三次 pan 共 126 格逐格對上：

| pan | 格數 | 之前 | 之後 |
|---|---|---|---|
| #1 | 37 | cam (0,0)、cur (0,0)、vis (0,0) | cam (3,34)、cur (3,34)、vis (0,0) |
| #2 | 42 | cam (3,4)、cur (8,8)、vis (5,4) | cam (0,43)、cur (5,47)、vis (5,4) |
| #3 | 47 | cam (0,0)、cur (0,0)、vis (0,0) | cam (5,42)、cur (5,42)、vis (0,0) |

每格恰好一張抓幀，也就是**每格一次呈現**，而且 X 先走完才走 Y。
`syncStoryNativeMapPanView` 因此改成把游標平移鏡頭的差量；
`cursor = camera + visible` 只有在恆等式成立時才碰巧一致，而
`0x149F8`（確認移動時只寫游標）會合法地打破它。差別由
`TestStoryPanMovesAbsoluteCursorByCameraDelta` 釘住：起點刻意讓
`visible ≠ cursor − camera`，兩條規則的結果不同。

## 可見游標的界線只在消費端成立

同一份收據另外量到一件事：`0x13185` 走行捲動 15 格之後
`camera_y` 34→20、`cursor_y` 34→19、`visible_y` 0→**−1**——原版容許可見游標
暫時離開 13×8 視窗。pan 與走行期間 overlay selector `[0x51A83]` 都是 0，
`0x1741C` 不會消費可見游標，所以那段期間沒有消費端會讀到出界值。

界線因此分成兩層：

| 層 | 由誰擋 | 擋什麼 |
|---|---|---|
| 狀態 | `validateNativeMapView` | 只擋結構性壞值：偏離超過場地本身。每個寫入端一次只動一格，且伴隨一次留在場內的絕對游標位移，所以偏離量不會超過場地。這是重製端的防溢位界線，不是原版契約 |
| 消費端 | `NativeMapViewState.VisibleCursorInViewport()` | 13×8。任何把可見游標當畫面格座標的路徑都先問它 |

目前的消費端有四處：`fdother.ActionOverlayOrigin`／`ActionOverlaySnapshotOrigin`
（`0x1741C`／`0x179D5`／`0x175A9` 的位址式）、`native_unit_present` 的 LUT 幾何、
`native_command_heal_presentation` 的 transition 幾何，以及節點常數入口
`materializeNativeMapRuntime`——節點常數是「進場當下就要畫出來」的靜止視圖，
游標框與指令環會立刻讀它，所以那條入口仍要求視窗內。

`AdvanceNativeMapWalkStepView` 因此不再把 `visible_y = −1` 判成錯誤；
`TestAdvanceNativeMapWalkStepViewLeavesViewport` 用收據裡的那一步釘住它。

## pan 是通用指令，只有一個實作

動畫引擎會發動鏡頭位移的地方有四處，全部走同一個 `camPanJob`，因此共用
同一條規則：

| 發動點 | 來源 |
|---|---|
| beat `pan` | handler 編譯出來的拍（`0x135DD`）|
| battle event `pan` | 戰鬥事件動作（同一個 `0x135DD`）|
| 回合登場演出 | `native_turn_staging` 的每一次 call |
| 截圖用的快轉 | `fastForwardShotBeats` 直接跳到終點 |

`tile_step` 的拍逐格發布（每格一次呈現，與原版每格一次 `0x11CAC(0)` 對齊），
`frames` 的拍是重製端的插值近似，只在終點發布一次；兩者的終點相同。
第五處是 `native_ch20_sky_key` 的專用 pan，它自己逐格走（`advancePan`），
規則相同，整段開始前的終點預檢也改用鏡頭差量，兩邊算出來的終點才不會分岔。

兩項已知限制：`frames` 模式的節奏是重製端自訂的（原版是每格一格一幀）；
`loadch` 綁定的 `cam_x`／`cam_y` 是重製端的自由捲動鏡頭，可以不對齊格線，
而 pan 的發布要求對齊——目前沒有任何綁定同時具備「非對齊 `cam_x`」與
`tile_step` 的 pan，所以這個組合不會發生，但它不是被擋下來的，只是沒出現。
