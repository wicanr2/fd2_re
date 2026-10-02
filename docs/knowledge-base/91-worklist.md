# 91 — FD2 remake 目前未完成項

未完成項的唯一權威是 GitHub Issues（[`wicanr2/fd2_re`](https://github.com/wicanr2/fd2_re/issues?q=is%3Aissue+label%3Aworklist)，
標籤 `worklist`）。[`docs/data/fd2-worklist.json`](../data/fd2-worklist.json) 是
`tools/fd2_worklist_issues.py pull` 拉下來的快照；下面那一節由
[`tools/fd2_worklist.py`](../../tools/fd2_worklist.py) 從快照產生，**不要手改**，
改了下一次 pull／render 就會被蓋掉。

每一條掛一個 `verify`：跑起來為真代表這一條仍然未完成，為假就是東西做好了而條目
沒改。這樣「動手前先確認那條還成不成立」不必靠人記得。

```sh
tools/fd2_worklist_issues.py pull   # 從 GitHub 拉下開著的條目，重寫快照（主機，要 gh 登入）
tools/fd2_worklist.py verify        # 逐條檢查，發現訊號消失的條目就以 exit 1 收場
tools/fd2_worklist.py render        # 重寫下面的產生區塊
python3 -m unittest discover -s tools -p 'test_fd2_worklist.py'
```

條目做完就在 GitHub 關掉那個 issue（`tools/fd2_worklist_issues.py close <id> <留言>`），不是在這裡打勾，也不是改快照。

其他入口：

- 每一題的 RE／資料／正式執行期／E2 分層現況：[`58-fd2-exe-re-coverage.md`](58-fd2-exe-re-coverage.md)
- 介面覆蓋率與尚未關閉的關卡：[`57-ui-evidence-matrix.md`](57-ui-evidence-matrix.md)
- 各輪的工作記錄、勘誤與位址出處：[`91-worklist-history.md`](91-worklist-history.md)
- 已閉合位址的「不要重做」索引在 `58`；重開條件是輸入雜湊不同、原始指令或跳表
  直接反證、同狀態執行結果矛盾，或主證據缺少它聲稱具備的 writer／consumer。

<!-- BEGIN fd2_worklist.py render；不要手改這一段 -->

共 24 條未完成項。權威是 GitHub Issues（標籤 `worklist`），[`docs/data/fd2-worklist.json`](../data/fd2-worklist.json) 是拉下來的快照，本節由 [`tools/fd2_worklist.py`](../../tools/fd2_worklist.py) 產生。

`要人判` 的條目沒有機器訊號，每一輪都會被列出來——沉默不等於通過。
新增、修改、關閉條目都在 GitHub 上做（[`tools/fd2_worklist_issues.py`](../../tools/fd2_worklist_issues.py) 的 `new`／`close`），之後 `pull` 更新快照。

## re — 原版證據還沒閉合

### 第十五章回合事件13／18／38尚缺完整動作與正式消費端

`ch15-turn-events-13-18-38` · RE待解 · [#78](https://github.com/wicanr2/fd2_re/issues/78) · 仍未完成 · 要人判

#76 第十五章抽樣前核對原版 FDFIELD_043 控制列：第4回合event13 raw camp1、第7回合event38 raw camp0、第9回合event18 raw camp0。現有 ch15 劇本只將event38降為spawn group1；native_death_events.json尚無三者的完整轉寫，因此既有同步工具無法還原鏡頭、模式與對白等其餘動作。先查58及現有reinforcement_eax_sources證據，僅匯出缺少的有限handler及writer／consumer，形成READY規格後同步legacy／canonical並以本章原版抽樣驗收。

怎樣算做完：保留固定EXE雜湊、IDA9.4原始名稱／位址／bytes、完整直接控制流及caller／consumer；三個事件有分級canonical證據、READY typed規格及正式消費端，按原始camp與回合時機執行；固定建構槽第4／7／9回合抽樣通過行為與完整RGB≤640，不以spawn-only或重製單元測試當作原版一致。

證據：`['docs/data/turn_events.json', 'docs/data/event_id_groups.json', 'docs/data/ida/fd2_reinforcement_eax_sources.json', 'remake/assets/scenarios/ch15.json', 'remake/assets/maps/native_turn_event_controls.json']`

## data — 可編輯資料還沒就緒

### 現代美術主題仍是原型狀態

`modern-theme-prototype-status` · 工作 · [#4](https://github.com/wicanr2/fd2_re/issues/4) · 仍未完成 · 自承還在 remake/assets/themes/modern/catalog.json

現代主題 catalog 目前登錄 `"status": "prototype"`。逐組審查頭身比例、原版配色、輪廓與戰場尺寸可讀性尚未走完，早／中／晚期地圖的接縫、前景遮擋、人物、游標與 HUD 也還沒抽測。

怎樣算做完：catalog 轉為正式狀態，且抽測涵蓋早／中／晚期地圖與戰場尺寸可讀性。

### 四筆角色身份的多名稱已分類，剩人複核

`editor-identity-ambiguity` · 工作 · [#6](https://github.com/wicanr2/fd2_re/issues/6) · 仍未完成 · 自承還在 remake/assets/editor-canonical/character-identity.json

native-0／native-1／native-7／native-96 的多個候選名稱已由資料本身分成兩類：章節互不重疊的是同一身份在不同段落的稱呼（索爾／索爾(少年)、刺客／蘭斯洛特），同章且共同前綴加單一編號的是多個雜兵共用一個 sprite（強盜 B/C/L/M/N）。診斷都帶著章節依據，severity 從 error 降為 note，canonical 已無未分類衝突。剩下的是人複核那個分類對不對——判準是從資料算的，不是從劇情知識來的。

怎樣算做完：人複核四筆的分類；若有誤判就調整判準並重生 bundle。

### 回合事件處理器只轉寫了 spawn：其餘章的 AI 模式改寫、鏡頭、演出與對白待逐章轉寫

`turn-event-handlers-partial-transcription` · 缺陷 · [#33](https://github.com/wicanr2/fd2_re/issues/33) · 仍未完成 · 要人判

FDFIELD 回合事件（docs/data/turn_events.json）在 gen_campaign.py 只降成「第 N 回合登場 group」的 spawn_group；處理器（全域事件表 0x51B91）裡的 0x3419C AI 模式範圍、0x135DD 鏡頭、0x1366A 演出與 0x15F84 對白都沒有進劇本，所以原版在敵方回合前會播的對白重製端不播、敵方 AI 模式也不會照劇本改。第四章 event 11 與第五章 event 14–17 已用 tools/extract_native_death_events.py 轉寫（指令逐條核對）並由新的 tools/sync_native_turn_events.py --chapters 4,5 降成劇本動作；其餘 57 筆回合事件仍是 spawn 版本，依 111 逐章推進時逐章轉寫、跑該章對拍後再擴 --chapters。ch13 的 event 5 已轉寫但那一章還沒對拍，不先換。

怎樣算做完：tools/sync_native_turn_events.py --check（不帶 --chapters）通過，報告裡「尚未轉寫的回合事件」為 0，且每一章的擴充都有該章 111 收據。

## runtime — 還沒接進正式執行期

### 升級五行訊息與 END 回復圖示／音效只有數值沒有演出

`levelup-and-end-recovery-presentation` · 缺陷 · [#29](https://github.com/wicanr2/fd2_re/issues/29) · 仍未完成 · 自承還在 remake/cmd/fd2/main.go

原版 `0x1E292` 升級時以 `0x15F84` 逐行顯示「升級！」與 AP／DP／DX／HP／MP 增量（FDTXT #0x1E8..#0x1EE），每行之間 `0x16559`／`0x16E24` 等待按鍵；`0x1A30B` 開頭的我方回復在每個回復的單位上畫 `0x1DA16` 圖示並播音效 4。重製端 `AwardExpNative` 與 `ApplyNativeEndTurnRecovery` 只改數值（第四章對拍已靠這兩條把回合 4 的行為 gate 推到只剩 RNG 時序差），畫面與等待節奏沒有接，所以原版側這幾格的幀在重製側對不到同狀態。要做：升級訊息用 indexed 資源逐行顯示與等待；END 回復加圖示與音效；重播測試的 `ackPresents` 收進這兩種工作。

怎樣算做完：第四章收據裡升級（seq 644..649）與 END 回復（seq 665..671）的原版幀在重製側有同狀態幀且落在像素預算內。

### 酒店服務 2 讀檔（0x301F4）與傳聞後回酒店選單尚未接

`hotel-load-service-0x301f4` · 缺陷 · [#32](https://github.com/wicanr2/fd2_re/issues/32) · 仍未完成 · 自承還在 remake/cmd/fd2/main.go

酒店 `0x2FC85` 的原生介面（資源 13 框與四圖示、DATO 0x81 店主、存檔槽列表 `0x30550`、「記錄儲存完畢」）已在 `remake/cmd/fd2/native_hotel_ui.go` 接上（#27），第四章收據 seq 1198 逐像素相同。還缺兩項：服務 2 讀檔（`0x301F4`，四槽列表載入後直接進該存檔的城鎮）目前仍只把 raw selector 對成 `fdother.ResolveNativeHotelServiceRoute` 的路由並回一句「原生介面尚未接這一項」；服務 0 打聽消息的 story 播完，原版回到酒店選單，重製端經 `hotel_chNN.rumor` 播完回城鎮。要做：讀檔走標題 LOAD 同一條載入路徑但回城鎮節點；傳聞 story 結束後回酒店選單（戰役資料的 story `next` 指回 `hotel_chNN`，或酒店節點自己接 story）。

怎樣算做完：酒店選服務 2 能列四槽並載入存檔進該章城鎮；傳聞播完回酒店選單；`applyHotelServiceSelection` 的自承訊息拿掉。

### 出口／整備確認提示：原版停在 YES 上畫的是 action cell 49，重製端畫 cell 48（每章 departure_prompt／town_enter 固定差 60 像素）

`parity-departure-prompt-yes-pulse-phase` · 缺陷 · [#35](https://github.com/wicanr2/fd2_re/issues/35) · 仍未完成 · 要人判

ch04 與 ch05 收據的 departure_prompt 與 town_enter 四個點都差 60 像素、同一個框 [235,173,252,179]。把原版 checkpoint-0038 的 YES 區域（24×16，座標 232,168）逐格對 ui/action_cells 的 78 個 cell：cell_049 差 0、cell_048 差 60。重製端 ComposeNativeConfirmationChoices 對選中項畫 base+pulse（YES 48／49），checkpoint 當下 nativeClassUIPulse/2 是 0，原版是 1。要查的是 0x19953 選中閃爍的起始相位（提示一開就是 cell 49？還是相位由 BIOS tick 決定而 checkpoint 剛好落在 1），確定後改重製端的起始相位或讓重播對這個點出 pulse 0／1 兩個變體。NO（cell 51／52）在這四個點沒差。收據：docs/data/ui-traces/parity-ch04.json、parity-ch05.json frames.points kind=departure_prompt／town_enter。

怎樣算做完：parity-ch04.json 與 parity-ch05.json 的 departure_prompt／town_enter 點 diff_pixels 為 0，且 56 記下 0x19953 起始相位的證據（哪一條指令、哪個全域）。

### 重播端在 town_enter／attack_armed 沒出 DATO 嘴型相位與 DAC 循環色相位的變體（ch06 seq 1603 335 px、seq 1053 8 px）

`parity-replay-dato-mouth-and-dac-cycle-variants` · 缺陷 · [#38](https://github.com/wicanr2/fd2_re/issues/38) · 仍未完成 · 要人判

第六章收據 parity-ch06.json 的 18 個 diff_pixels>0 點裡有兩個新形狀，不屬於 #34（指令環開啟步）與 #35（YES pulse）：(1) seq 1603 town_enter（酒店入口對白）335 px，框 [25,146,252,179]：左側 x 24–47、y 146–158 是店主 DATO 頭像的嘴型／眼部幀不同（原版 sub_16C57 的嘴型倒數吃 rand()%30，r4 eip-trace 在對白期間 0x16C9E 呼叫 0x4E893 21 次），右側 x 240–252 是 YES 的 pulse（#35 同一形狀）；重播端 town_enter 只出一張（phases 1）。(2) seq 1053 attack_armed 8 px，框 [106,110,115,115]：8 個像素都是調色盤 index 225，原版 DAC (44,73,142)、重製 (48,77,146)，是循環色差一步（0x11d40 DAC 寫入的相位），44 個 idle 變體都取不到；ch04／ch05 沒出現這個形狀。兩個都是「原版沒記錄的時間相位，重播端沒出對應變體」，和 #34 同一類處置：重播端在這兩種點多出變體（嘴型倒數 0..N／閉合、DAC 循環相位），verifier 取最小；或證明原版在 checkpoint 當下的相位由什麼決定（rand%30 的序列、BIOS tick）直接算出來。

怎樣算做完：重跑 ch06 remake 側後 parity-ch06.json 的 seq 1603 只剩 #35 的 60 px、seq 1053 為 0；56 記下嘴型倒數與 index 225 循環色在 checkpoint 當下的相位規則。

### 重製端沒接城鎮出發的十步縮放暗化（0x2D190..0x2D275）與進戰場的 64 步淡入（0x1F544）

`town-departure-zoom-and-battle-fade-in-not-in-remake` · 缺陷 · [#39](https://github.com/wicanr2/fd2_re/issues/39) · 仍未完成 · 要人判

#37 用原版側探針（work/parity-slot-ch07/probe-fade，FD2_ORACLE_FRAME_EIP=0x11D40 抽幀、FD2_ORACLE_EIP_TRACE=0x1F882,0x11D40）把出口 YES 之後到 battle_start 之間的兩段過場閉合了（56 §城鎮進戰場的過場、輔助基準 docs/figures/parity-ch07-town-fade-probe.png）：(1) 0x2D190..0x2D275 城鎮出發：ebx=1..10 十步，每步依 [0x5412B] 城鎮變體的 0x52635／0x52647 表內插座標呼叫 0x2FB9F 重畫（建築往中心放大），0x373C4 搬到 VGA，0x11D40(0,0xFF,4k) 把 DAC 壓暗，迴圈後 0x11D40(0,0xFF,0x40) 全黑、0x375C0 清 VGA；(2) 0x1F42D 進戰場：LOADCH 之後 0x1F544 呼叫 0x11D40(0,0xFF,level) 64 次、level 0x40→0 從黑淡入地圖，之後才是戰前 handler 的對白與 battle_start 游標。正式重製端從城鎮出發直接進戰場，兩段都沒接；章收據的 departure_prompt／battle_start 都在這段之外，gate 看不到。做法：把 0x2FB9F 的縮放重畫與 0x11D40 的 DAC 等級寫成 indexed composer 的兩個過場工作（城鎮側十步、戰場側 64 步），用探針幀（每步一幀，index parity-ch07-town-fade-probe.json）做同狀態對照；0x2FB9F 的內插表 0x52635／0x52647 與「建築往中心放大」目前是強推論，接之前先把它反組譯成規則。

怎樣算做完：重製端從城鎮出口 YES 到戰場第一幀之間播出十步縮放暗化與 64 步淡入；用 FD2_ORACLE_FRAME_EIP=0x11D40 的探針幀逐步對照（每步 diff 在 640 px 預算內）並記進 57；56 的 0x2FB9F 內插表升為已證實。

### 攻擊演出收尾、0x1DEAE 與訊息關框三類重繪還沒經過 HUD anchor 入口

`hud-anchor-attack-and-message-redraws` · 缺陷 · [#41](https://github.com/wicanr2/fd2_re/issues/41) · 仍未完成 · 自承還在 docs/knowledge-base/56-fd2-remake-sdd.md

#40 把 anchor 評估收成單一入口 `redrawNativeMapHUD`（游標鍵步、`0x12CEA` 開頭與逐步、`0x1A30B` 返回後、戰場進場），第十章章收據 268 個畫面點全部一致。還沒接的是三類原版 `0x11CAC` 呼叫點：攻擊演出收尾（`0x1CFF0` 的 `0x1D3FF`、`0x1548E` 的 `0x15510`／`0x1563B`／`0x1565C`）、`0x1DB65` 的 `0x1DEAE`、訊息關框 `0x19742`。呼叫點清單在 `docs/data/ida/fd2_redraw_callers_hud_gates_ida.txt`，對照表在 `docs/knowledge-base/56` §#40（那三列標「未接」）。它們對應到重製端哪一段整幀重組、當時閘 B 與可見游標是什麼，都還沒逐點證明；在拿到收據之前不要為了收斂差異自行加評估點。

2026-09-18 定案：不為了這條重跑已 passed 的章；等之後排程到的新章收據順帶驗收。

怎樣算做完：以原版 eip-trace（或同狀態擷取）證明這三類重繪發生的時機與當時的閘門／可見游標，接到同一個入口。不為了驗收重跑已 passed 的章（使用者 2026-09-18 定案）：以之後正常排程的新章收據順帶確認即可，沒有出現對應時點就在 56 §#40 記錄仍未被覆蓋。

### 第十五章缺record64倒下的原版敗北判定

`ch15-ally64-defeat-rule` · 缺陷 · [#79](https://github.com/wicanr2/fd2_re/issues/79) · 仍未完成 · 要人判

#76 完整原版sample-original-r1於第6回合退出，未涵蓋第7／9回合。最後有效checkpoint1143我方16人仍活著，友軍record64 HP0／raw+5 bit0=1。既有battle_events.json記raw章14 handler0x20822查record64，但ch15劇本尚無原生結果規則。先用IDA核對直接writer與結果consumer，保留失敗樣本及未知限制，READY後接既有型別規則。原版退出原因在直接核對前只列強推論，不把它當執行器缺口或全隊陣亡。

怎樣算做完：固定EXE identity、IDA9.4原始20822完整bytes與結果writer／consumer保存canonical並登記58；READY後接入ch15及canonical原生敗北規則，普通玩家操作觸發友軍64倒下後走既有敗北owner；保持其他友軍倒下與我方尚存不誤判、raw旗標缺失失敗即關閉。完整#76另採正常護援保留第4／7／9回合，不提前清敵、不鎖HP。

證據：`['docs/data/battle_events.json', 'work/parity-slot-ch15/sample-original-r1/checkpoint-1143.json', 'docs/data/parity-plans/ch15-sample.jsonl', 'remake/assets/scenarios/ch15.json']`

### 第十五章原版與重製有18筆AI入口順序分岔

`ch15-ai-entry-divergence` · 缺陷 · [#81](https://github.com/wicanr2/fd2_re/issues/81) · 仍未完成 · 要人判

怎樣算做完：保留第一個原版／重製分岔的seq、raw單位與模式、執行前RNG條件及原始證據；按RE→READY修正真正缺口；同槽AI入口零順序分岔及嚴格畫面／行為比較，完整Go回歸通過。#76整章第7／9回合與戰後仍獨立驗收。

證據：`['work/parity-slot-ch15/sample-remake-r2.log', 'work/parity-slot-ch15/sample-original-r1', 'work/parity-slot-ch15/sample-remake-r2/checkpoints.jsonl', 'remake/cmd/fd2/chapter_parity_replay_test.go']`

### 第十五章攻擊結算 seq849 畫面仍差16098像素

`ch15-attack-result-0849-frame` · 缺陷 · [#82](https://github.com/wicanr2/fd2_re/issues/82) · 仍未完成 · 要人判

第十五章同槽重播r3消除了AI分岔，66行為點均通過；61整幀比較只有seq849 attack_result超出640px門檻，差16098px。須核對原版同狀態畫面與正式介面owner；不得遮蔽像素或放寬門檻。

怎樣算做完：seq849原版／重製同狀態畫面差異不超過既有640px，具有原始證據、READY規格、正式owner修正與相關回歸；完整章仍須另驗戰後與存檔。

證據：`work/parity-slot-ch15/sample-verify-r3.json`

## player — 缺未修改一般玩家路徑的驗收（PLAYER-E2）

### 戰鬥交易缺代表性玩家路徑驗收

`battle-transaction-e2` · 工作 · [#12](https://github.com/wicanr2/fd2_re/issues/12) · 仍未完成 · 要人判

正常 producer 的物品、command 與敵方 AI 都已達 `RUNTIME-E1`。缺的是代表性玩家與敵方回合的精確演出、音訊與未修改一般玩家路徑的 `PLAYER-E2`。

怎樣算做完：挑代表性的玩家與敵方回合，以未修改路徑取得同狀態逐幀與音訊對照。

### 戰後節點缺完整玩家路徑驗收

`campaign-postbattle-e2-full` · 工作 · [#14](https://github.com/wicanr2/fd2_re/issues/14) · 仍未完成 · 還沒出現

已綁定的章節各自有窄 `RUNTIME-E1`，但沒有每一章都以同狀態走過戰前對白、戰鬥節拍、戰後節點、城鎮與存檔邊界並與原版比較。

2026-09-15 決議（取代 2026-08-23 的「交人工遊玩回報」）：由代理程式依 [`111` 目標提示詞](https://github.com/wicanr2/fd2_re/blob/main/docs/goal/111-goal-original-parity-campaign-20260915.md) 用 dosgolem 逐章推進 30 章。起點用受版控工具依攻略校準的建構槽跳關，章內全程正常鍵盤輸入，抽樣完戰鬥節拍後以 `force-enemy-clear` 進戰後節點；這條路徑視同該章 `PLAYER-E2`。四個 gate（行為、節點、交易零容忍；代表畫面 ≤1% 差異像素）都過才算一章完成，進度台帳是 `docs/data/parity-campaign-progress.json`。

怎樣算做完：30 章各自依 111 的章工作單元取得原版／重製收據，四個 gate 全過；parity-campaign-progress.json 的 all_chapters_passed 為 true。

### 曲號與音效還需要人耳確認

`bgm-sfx-listening` · 工作 · [#15](https://github.com/wicanr2/fd2_re/issues/15) · 仍未完成 · 要人判

開場配樂曲號、戰鬥曲與勝利曲的對應、以及 UI 音效 index 2..0xb 的語意畫面，都要實際聽辨才能定案。容器內無音訊裝置，驗不了。

怎樣算做完：逐項聽辨後修正曲號對映與音效語意記錄。

### 第十二章地圖 tile27 的 mode3 單像素殘差

`ch12-hud-terrain-single-pixel` · 缺陷 · [#52](https://github.com/wicanr2/fd2_re/issues/52) · 仍未完成 · 要人判

2026-10-01 定位勘誤：目前正式 treasure-original-r2／treasure-formal-r2 收據的17個差異點，均位於地圖 tile27 的局部像素(7,9)，不是HUD地形縮圖。13張完整畫面只差1px，另4張與指令環差異共存；完整四gate仍通過，本項維持可選像素修飾、不阻塞111。

原版IDA 9.4既有解碼契約核對：0x11EEE的0x1220C／0x12220分別呼叫raw 0x4DEDA及LUT 0x4DCC6。mode3在raw路徑保留目的底色，在LUT路徑讀目的像素並映射。每張殘差的實際分支及底色寫入來源尚未知，不能用archive byte+3推論runtime分支，也不能改成固定色。

主證據：docs/data/ida/fd2_terrain_mode3_review_20261001.json（輸入雜湊、工具／位址空間、原始bytes、分級與17點影像雜湊）。舊HUD解釋保留於Issue歷史與專案交接，新增證據以未遮罩座標及camera→world tile對照否定該定位；不宣稱根因已閉合。

怎樣算做完：定位單像素來源的原生writer／繪圖契約，依READY規格修正並以相同狀態點驗證，不遮蔽或調預算。

### 第十五章缺完整原版與重製四項對拍收據

`ch15-full-chapter-parity` · 工作 · [#76](https://github.com/wicanr2/fd2_re/issues/76) · 仍未完成 · 要人判

第十四章已於36678491完成完整驗收。接續111／114逐章工作單元，先核對第十五章正常LOAD起手、實際名冊前沿及可編輯事件，再跑有界原版與完整同槽重製。未知行為先登記RE或缺陷工單，不猜補正式路徑。

怎樣算做完：以已核對ch02-cleared基底建構相同第十五章槽，levels6、seed4、state7:17=1與AP+200／DP+0／DX+60政策不變；章內正常鍵盤抽樣主要事件及敵方回合，再依111清敵驗戰後、城鎮、交易與酒店存檔。原版與完整Go串行，四項全過、至少12張320×200全RGB≤640px，保存manifest／plan／正式收據並同步56／57／58、台帳與首頁。

證據：`['docs/data/parity-campaign-progress.json', 'docs/goal/111-goal-original-parity-campaign-20260915.md', 'docs/goal/114-goal-boosted-slot-and-ch09-parity-20260917.md']`

## release — 發行、平台與封包

### 網頁版前景玩家路徑待最終自動驗收

`wasm-web-release` · 工作 · [#7](https://github.com/wicanr2/fd2_re/issues/7) · 仍未完成 · 還沒出現

建置、資產與存檔都已通過；剩餘缺口是前景瀏覽器中的開場至第一關操作，以及重新載入後的存檔讀回。

2026-09-14 決議：本項延到其餘非「最後處理」worklist 完成後才做。屆時由代理程式在 Docker／Xvfb 啟動非 headless 瀏覽器，以 CDP 將頁面帶到前景並自動送正常輸入、截圖與檢查 localStorage；不再要求使用者人工確認。若 Chromium 仍把頁面標成 hidden，先修正可重現的前景瀏覽器工具鏈，不得把 hidden 分頁結果冒充通過。

卡在：依使用者 2026-09-14 裁定，排在所有非「優先級:最後」worklist 完成之後。

怎樣算做完：其餘非最後處理 worklist 完成後，在 Docker／Xvfb 的非 headless Chromium 中由 CDP `Page.bringToFront` 確認 `visibilityState=visible`，從目前 WASM 產物正常走到第一關可操作，完成一次存檔、重新載入與讀回；保存輸入、畫格、console、localStorage 前後值與截圖到 docs/data/ui-traces/wasm-browser-player-path-e1.json，並由自動測試驗證。

### Android 封包沒有建置

`android-package` · 工作 · [#8](https://github.com/wicanr2/fd2_re/issues/8) · 仍未完成 · 還沒出現

觸控輸入已支援，但沒有 `ebitenmobile bind` → `.aar` → APK 的建置流程。

怎樣算做完：產出可安裝的 APK，並在實機確認啟動、存檔與音訊。

### Windows 與 macOS 還沒有實機抽測

`release-platform-acceptance` · 工作 · [#9](https://github.com/wicanr2/fd2_re/issues/9) · 仍未完成 · 要人判

三平台公開封包已由 CI 產出，Linux 經啟動與解包驗證。Windows 與 macOS 只做過 ZIP 內容驗證，沒有實機啟動、存檔與音訊抽測。

怎樣算做完：兩個平台各自在實機完成啟動、存檔／讀檔與音訊抽測並記錄結果。

## tooling — 工具、編輯器與工作流程

### 戰場編輯器待最終自動端到端驗收

`battlefield-editor-mvp` · 工作 · [#10](https://github.com/wicanr2/fd2_re/issues/10) · 仍未完成 · 還沒出現

戰場編輯器已具備地圖與單位編輯、部署格、波次總覽、復原及格式保真測試；缺的是由瀏覽器編輯正式複本、存回、重生 canonical，再由正式 runtime 載入的端到端收據。

2026-09-14 決議：本項延到其餘非「最後處理」worklist 完成後才做，屆時由代理程式以瀏覽器自動化操作可丟棄複本並驗證，不再要求使用者人工畫圖或授權目錄。

卡在：依使用者 2026-09-14 裁定，排在所有非「優先級:最後」worklist 完成之後。

怎樣算做完：其餘非最後處理 worklist 完成後，在 Docker 中啟動 editor server 與非 headless 瀏覽器，對一張原版地圖的可丟棄複本執行圖塊、單位與部署格各一項編輯，經正式 save API 存回並重生 canonical；正式 runtime 載入後核對地形成本、單位、部署格與畫面，最後還原來源零差異。保存 docs/data/ui-traces/battlefield-editor-roundtrip-e1.json 並由自動測試驗證。

### 劇情編輯器待最終自動端到端驗收

`campaign-editor-ui` · 工作 · [#11](https://github.com/wicanr2/fd2_re/issues/11) · 仍未完成 · 還沒出現

劇情編輯器五個分頁與節點圖均已可用，現有原版路線 round-trip 已通過；缺的是建立一條原版沒有的敗北／choice 路線，再由正式引擎走通首尾的端到端收據。

2026-09-14 決議：本項延到其餘非「最後處理」worklist 完成後才做。屆時由代理程式以瀏覽器自動化建立可丟棄測試路線並驅動引擎驗收，不再要求使用者人工設計或試玩。

卡在：依使用者 2026-09-14 裁定，排在所有非「優先級:最後」worklist 完成之後。

怎樣算做完：其餘非最後處理 worklist 完成後，在 Docker／非 headless 瀏覽器中用編輯器對可丟棄複本新增一條 battle.on_lose 敗北路線與一個依旗標過濾的 choice 分支，透過正式 save API 存回、重生 canonical，並由正式引擎以決定性輸入分別走通兩個分支首尾；還原來源零差異。保存 docs/data/ui-traces/campaign-editor-custom-route-e1.json 並由自動測試驗證。

### 章對拍：指令環四步開啟中途的原版 checkpoint 沒有對應的重播變體（每個 move／stay 點殘留 66–628 像素）

`parity-replay-ring-open-step-variants` · 缺陷 · [#34](https://github.com/wicanr2/fd2_re/issues/34) · 仍未完成 · 要人判

ch04（14 點）與 ch05（16 點）收據裡所有 move／stay 畫面點的殘差都是同一類：原版 checkpoint 拍在 0x1741c 的四步指令環開啟動畫中途（圖示逐列揭示到一半），重製端 beginActionOverlayOpen 也有這四步，但 chapter_parity_replay_test.go 只對 idle 相位出變體、指令環一律是開完的那一幀，所以 verifier 比到的最小差就是揭示中途與開完之間的差。ch05 seq 859 已到 628 像素，預算 640，第六章敵人更多、圖示更大時很可能直接爆預算變成假失敗。要做的是重播端在 ring 狀態的點多出開啟步 0–3 的變體（乘上 idle 相位），verifier 照現有規則取最小；不改引擎的開啟動畫本身。收據：docs/data/ui-traces/parity-ch04.json、parity-ch05.json 的 frames.points（kind=move／stay）；56 §第六章的五章回顧表。

怎樣算做完：重跑 ch04 r13／ch05 r5 的重製側，parity-ch04.json 與 parity-ch05.json 裡 kind=move／stay 的 diff_pixels 全部為 0（或明寫剩餘的點是哪一個開啟步都對不上、為什麼）。

<!-- END fd2_worklist.py render -->
