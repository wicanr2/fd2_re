# 交接文件 — 給下一個 session(2026-07-06)

> 炎龍騎士團2 RE + Go/Ebiten remake(`/home/anr2/cht/fd2`,repo `wicanr2/fd2_re` main)。
> 記憶檔(`~/.claude/projects/.../memory/`)會自動載入=長期真相;本檔補「這段 session 的當前狀態 + 開放線索」。
> **動手前先讀:記憶索引 MEMORY.md、`docs/knowledge-base/00-index.md`(問題導向路由)、`doc50`(過場機制主檔)。**

> **2026-07-15 Codex 更正**：撤回舊 `0x207718`、id−48、74-resource 與「context 差 48 entries」
> 結論，它們來自錯 context／錯時刻的 acting dump。EXE 靜態 directory 是 106 entries（file+0x565d8，
> data=`file+0x53e00+offset`）；getter 以 source ACT ID 直接索引，沒有 chapter-local window。已驗 ACT99：caller
> `0x32343`、getter immediate=`0x2077d8`、table[99]=`0x208493`、bytes=`01 06 01 02 02`，即 slot2
> 上行六格（Y42→36、pose0→2）。ACT100 亦由 caller `0x323f5`/id100 live 驗為 slot2 下行十格
>（Y8→18、pose0）。不要以舊 map0/window 推論覆蓋此 provenance。

> **2026-07-16 第十九次 Codex 更新（戰後資料不再在城鎮邊界遺失）**：全30戰勝利流盤點發現
> `battle_ch04..10,12,14,16,18,19,22..26,28,29` 共19條非終局路徑直接進 town/preparation；但
> `enterNode` 會清掉 completed battle state，因此實際丟失等級、HP、戰利品與裝備。已在每條插入可編輯
> `postbattle_chNN_persist`：`sync_party → set_chapter → 原本 town/preparation`，不改城鎮／商店／教會
> ／整備去向。campaign regression 逐條追蹤 chapter1..29 正常路徑，強制在第一個戰間節點前恰一次 sync；
> chapter30 保留 direct ending。下一個商店切片必須使用同一 `partyRoster` numeric inventory，不能再寫入舊
> 的名稱字串清單。

> **2026-07-16 第二十次 Codex 更新（商店資料與原版收件規則）**：`shops.json`、demo 與 full
> campaign 的337筆商品皆已保存 EXE 原版 unsigned-byte `id`，逐筆以 `item.json` 的價格交叉驗證；
> generator 會拒絕缺 ID 商品。原版購買順序已 RE 為「確認→金錢檢查→選收件者→8格容量→插入首空槽→
> 裝備品詢問立即裝備→最後扣錢」；滿包／取消／無可裝備者均不扣錢。`0x1c1c3` 是純 class×item.type
> 六欄白名單，EXE file `0x55689`、stride7、首 byte 常數、後六 byte 才是 type；已匯出
> `docs/data/exe_tables/class_equip_types.json` 並加入 exporter selftest。尚未完成的是把此規則接進
> UI 的收件者選單、裝備狀態與能力重算；不得再把購買寫入 `Game.items []string` 當作真實 inventory。

> **2026-07-20 第二十二次 Codex 更新（商店收件者 UI）**：`main.go` 已接入 runtime eligibility assets，
> 商品 Enter 後進入第二段收件者清單，順序使用 `partyJoinOrder`、資料使用 persistent `partyRoster`；裝備品
> 依 class×item.type 篩選，消耗品列全隊，購買成功後以 map copy 寫回 roster 並保留 numeric inventory。
> Escape 可從收件者返回商品清單，商品頁再回原 town。已編譯驗證並推送 `39817dc`。尚待原版「要裝備上去嗎？」
> prompt、equipped flag／能力重算，以及賣出功能。

> **2026-07-16 第二十一次 Codex 更新（商店核心可測、等待 UI 接線）**：`battle.Unit` 已保留 FDFIELD
> numeric `ClassID`；`campaign.BuyGood` 已實作指定收件者的原子購買（成功才入8格 inventory 並扣金，滿欄／
> 缺錢完全不變）；`CanEquip` 已固定原版 class×item.type predicate，`LoadShopEligibility` 讀打包 runtime
> assets。`remake/assets/data/item.json` 與 `class_equip_types.json` 已強制納入 build（不是只放 docs），並有
> campaign regression。**下一輪第一件事**：在 `main.go` 將舊 `g.items []string` 商店分支換成「確認→合格
> 收件者→BuyGood→裝備詢問」UI state machine；裝備 flag、能力重算與賣出仍未實作。

> **2026-07-15 第二次 Codex 更新**：全 60 handler 重新抽取後 unknown 146→133。完整 callee body
> 已把 0x32975/0x32999/0x134e4/0x12d7b 定性成 deactivate_unit/spawn_intro/reset_pose/focus_unit；
> ch00 的 13 個缺漏 FDTXT calls 與 5 個 PAN 已接上；ACT99/100、兩段 scroll_step 與 focus_unit
> 也已 lower 並有 regression。`ch00_pre` 現可完整編譯為 editable beats，**0 unresolved issues**。

> **2026-07-15 第三次 Codex 更新**：`campaign_full.json` 預設入口已切到
> `story_ch00_handler → bindings/ch00_pre.json`，不再預設走手寫分幕。headless GUI smoke 已實際跑過
> 王座、草地、map31 全段並進入 map0 第一段對白；frame220 抓圖亦確認 ACT99+scroll 後索爾在
> `(8,21)` 正常顯示「兒臣索爾，晉見父王陛下。」。完整 runtime/unit tests 與 106-entry exporter check 全綠。

> **2026-07-15 第四次 Codex 更新（external overlay 排查）**：依使用者建議，重新追所有 DOS file
> open/read/seek 與 LE object mapping。結論是 handler/acting **不在外部 DAT，也沒有載入 text section**：
> handler code 在 EXE 跳表；acting directory `[0x627d8]` 是 EXE LE object #3 的 initialized data
>（file+`0x565d8`），payload bank=file+`0x53e00`。`0x111ba` 只把 FDTXT/FDFIELD/FDSHAP/美術資源讀進
> malloc heap。另以 DOSBox-X `-log-fileio` 實跑到 map32 草地對話，acting 期間沒有 FDOTHER/ANI/
> FIGANI/FD2.TMP read；`FD2.TMP` 只有 207360-byte write，無 read-back。詳證補在 `doc50`。

> **2026-07-15 第五次 Codex 更新（戰後 persistent roster）**：`0x11506` 的 **24 個戰後
> caller** 已由完整 body 定案，不是查詢函式。它以角色 ID 配對 runtime battle array 與 persistent
> roster，將完整 `0x50`-byte unit **由 runtime 複製回 persistent**；隨即清 persistent `+0x22..+0x27`
> 六 bytes 與 transient flags，存活 active 者 HP 回滿、全員 MP 回滿，死亡／inactive 者保留零 HP，再呼叫 `0x1145a`
> 依裝備重算衍生值。ch00 post 已 editable lower 成 `dialog → sync_party → set_chapter(1)`，由
> `story_ch02` 的 `bindings/ch00_post.json` 接入；`partyRoster` 會在下一戰 materialize 時覆蓋持久能力
> 值，且已納入 remake JSON save/load。全量 handler `unknown` 因此由 **133 降至 109**。完整位元組流程
> 與欄位證據（包含 ID 0 inactive/dead 時原版會跳過 copy 的特例）見 `doc50 §3.2`。

> **2026-07-15 第六次 Codex 更新（戰後獎勵物品）**：`0x1c220(item_id)` 已由完整 body 與
> `0x1bb8c` 定案為「按 runtime slot 找第一個我方且 8-slot inventory 有空位的角色，放入 item」。
> 兩個 caller 是 ch01 `0xC6` 力量藥水與 ch20 `0x64` 天空之鑰；已 lower 成 editable
> `grant_item`，角色 `Inventory` 會經 `sync_party` 與 save/load 跨章保留，handler unknown 109→107。
> 此更新當時另發現 slots 5..10 存活分支與 FDTXT_002 缺 8 句等 11 issues，不能把 #6/#7 兩條
> 路徑直線串播；分支已由下一筆更新解決，其餘 binding 問題仍待處理。

> **2026-07-16 第七次 Codex 更新（handler control-flow，已校正 bit 方向）**：ch01 post 的 diamond 已從原版
> 指令形狀復原成 editable `if any_unit_inactive(slots 5..10)`。任一村民死亡只播 #7；全員存活才播 #6
> 並送 `0xC6`，之後共同 continuation 只執行一次。compiler 會先 resolve 兩臂、runtime roster 不完整
> 時 fail closed；dialogue binding/unknown diagnostics 亦會遞迴 branch。詳見 `doc50 §3.4`。

> **2026-07-16 第八次 Codex 更新（FDTXT_002 完整化）**：`ch02.json` 已由 53 補到原版 61
> logical utterances，#6/#7 互斥獎勵已拆開，#5 與 #11..16 亦保留在獨立資料位置；ch01 post
> 的五個 dialog call-sites 全部取得精確 mapping，compile issues 11→6。並修正 `FFED operand`
> 不是角色 ID 而是 runtime slot：村民 slots5..10 以 `speaker_slot` 動態解析 DATO134/133，缺 slot
> 時 fail closed。詳見 `doc50 §3.5`。

> **2026-07-16 第九次 Codex 更新（ch01 post 完整接線）**：ch02 battle 已恢復原版 runtime
> constructor 順序：party0..4、村民5..10、group2=11..20、turn3 group3=21..26，戰後 SPAWN4
> 才 append 希莉亞為 slot27；group255 不再預佔 runtime array。`ch01_post.json` binding 以明確
> postbattle context 驗證 slot frontier，PAN 定案為 `(336,48)/(336,24)`，ACT14/15/16 直接作用於
> canonical battle state，compiler 為 0 issues。campaign 已接成 battle→post→下一章 town/preparation，完整測試、
> build 與 Xvfb branch/PAN/SPAWN/ACT14 截圖均通過。戰後演出中因 save 尚不序列化 battle array，
> F5 會明確拒絕，下一節點恢復可存。詳見 `doc50 §3.6`。

> **2026-07-16 第十次 Codex 更新（shared tail + 第二章戰前）**：修正 exporter 把「下一個 jump-table
> entry」誤當 CFG 絕對終點的問題；原版多支 handler 會跳到邊界外／較低位址的共用尾段。60 支
> handler 重新機械輸出後 top-level beats **624→701**，unknown **107→108**；兩個合成 CFG 測試固定
> external/backwards shared-tail 順序。`ch01_pre` 現完整包含尾端 `FDTXT_002 #3` 與 focus(slot0)，
> 四段原版字串展開 20 句、compiler 0 issues；兩段 PAN 依 `0x135dd` 改為 X-first、每次一 tile 的
> `tile_step`。另由 battle-event caller `0x341e6 push 1; call 0x112a5` 定案哈諾在 turn3 JOIN，
> persistent party 順序為 `[索爾0,悠妮9,亞雷斯4,蓋亞30,哈諾1]`，再 materialize group3；campaign
> 已接成 `ch00_post → ch01_pre → battle_ch02`。同時撤回舊的 `ch05_pre=玩家第五章` 假設：它是
> 零起算 table index5，實際選 map5/FDTXT_006（玩家第六章），其 shared dialog 與後期 JOIN chronology
> 尚未閉合，所以不再冒充 campaign complete consumer。詳見 `doc50 §3`。

> **2026-07-16 第十一次 Codex 更新（第三章戰前 + FDTXT_003）**：`ch02_pre` 16 source beats 已
> 完整 lower 成26 runtime beats、0 issues：六人 JOIN-order party `[0,9,4,30,1,8]`，三段 X-first
> tile PAN，ACT18→SPAWN1九人→ACT17/19，以及跨 handler shared dialog/reset/focus。map2 battle 同步
> 改為 party-first runtime append，group255 不再汙染 slots。更重要的是回原始 FDTXT_003 找回舊
> `ch03.json` 真正漏掉的六句 turn3 葛雷／卡蘿硬編碼對話，全文由33補成39，索引重生後達39/39
> count-aligned（generated contexts 81→83、skipped 89→87）。campaign `story_ch03` 已由章標 stub
> 改接 authored ch02_pre。後續已以 constructor/death/revive 完整 body 解開 slot6 bit0 方向，
> 下一筆更新與 `doc50 §3.7` 為現行定案。

> **2026-07-16 第十二次 Codex 更新（bit0 全域翻案 + ch03 條件）**：完整反組譯與 live
> dump 交叉證實 `unit+5 bit0=0` 才是 active/alive，`1` 是死亡／隱藏／inactive；bit7 才是
> acted。有效 constructor `0x10eed` 寫0，HP0 路徑 `0x1dc61/0x1dd4c` 寫1，復活 `0x30f9c`
> 清0。exporter/runtime 因此改名 `unit_inactive`、`any_unit_inactive`、`deactivate_unit`，60 支 handler
> 已由原 EXE 重生。ch01 post 現為六村民全存活才 #6+item198；ch03 turn3 新增
> `unit_slot_active:6`，鐵諾死亡時不再誤生 group2。`0x11506` 也同步校正為存活者 HP 回滿、
> 死亡者保留零 HP。`ch02_post` 真 CFG 已釘死為 `sync → inactive?#6:(layout+#7+JOIN2) → chapter3`；
> 下一優先是 single-slot diamond、`0x233c6 layout_units` 與 15/27-slot runtime frontier。

> **2026-07-16 第十三次 Codex 更新（ch02 post 完整閉合）**：extractor 已以通用指令形狀
> 復原 single-slot diamond，`ch02_post` 現為 `sync_party → if any_unit_inactive([6])`；死亡臂只播
> #6 五句，存活臂執行 `layout_units`、#7 十句並 JOIN2，共同 `set_chapter(3)` 只保留一次。
> `0x233c6` binding 保存 slots0..6 絕對 X/Y/pose、camera `(48,0)`、redraw/fade/delay200；
> post runtime 只接受 15/27 slots，對應 turn3 援軍未生／已生兩種真實 frontier。campaign 已接
> `battle_ch03 → story_ch03_post → town_ch04 → preparation_ch04 → story_ch04`，compiler 0 issues。同輪把全 post handlers 的
> `inc [chapter]` 保留成 editable `set_chapter`，15 個 `0x233c6` caller 改為已命名、待逐章 binding 的
> `layout_units`；全 60 支 manifest 為 **725 top-level beats / 93 unknown calls**。詳見 `doc50 §3.8`。

> **2026-07-16 第十四次 Codex 更新（戰後 town/preparation 全戰役契約）**：原版 victory
> driver 已重追為 `post[current] @0x25e23 → intermission 0x2cad7 → pre[next] @0x25e3a`，
> 不是 post 後直接下一戰。`byte[chapter+0x526b9]` 的零起算章表是 0..21 town、
> 22..24 preparation、25..26 town、27..29 preparation；這與商店只存在玩家章
> 2..22、26、27 相符，也證明 shops.json 的章數是「下一場」，舊 campaign 整體 off-by-one 已修正。
> remake 新增 editable `town`、`preparation`、`church` 節點；town 保留酒店／武器店／出口／
> 道具店／教會五設施與 hidden secret shop，各設施離開後回 hub，出口才進可存檔的隊伍整備；
> 原版無 town 章也依然有「要記錄戰況嗎？」與 sortie preparation。
> `TestCampaignFullPostBattleTownContractMatchesOriginalShopChapters` 已對全戰役固定 shop 章集合、
> post→town/prep→next pre、facility 回 hub、無 town 仍有 prep 及最終 ending；詳證見 `doc50 §3.9`。
> 尚未閉合的原版分支是玩家第27章戰後：天空之鑰 `0x64` 存在才增章進第28章，
> 無鑰匙則 `0x2545d → 0x2bce5` 壞結局；這個 handler/inventory condition 仍需後續接線。

> **2026-07-16 第十五次 Codex 更新（ch03 turn3 通用 battle-event sequencing）**：新增與
> campaign BeatRunner 分離的 `battleEventRun`；`Scenario.TriggerActions` 保存 JSON action 原序，
> runtime 完整播放 `SPAWN2 → PAN(3,0) → 800ms → PAN(3,17) → 200ms → FDTXT_003 #4 七句`。
> map2 24px tile 使鏡頭精確到 `(72,0)/(72,408)`，等待為48/12 ticks；事件最後一句前 Turn 與
> status 都不 tick，finishTurn 重入不重複觸發。battle event 同時改用原版320×200（13×8格）
> viewport；完整 Go tests 與 Xvfb frame120 實畫均通過。詳見 `doc50 §3.7`。

> **2026-07-16 第十六次 Codex 更新（第27章天空之鑰 gate）**：campaign 新增非玩家選擇的
> editable `inventory_gate`，`battle_ch27` 勝利後以 item `0x64` 分成兩臂。有鑰匙才執行
> `sync_party → set_chapter(27)` 並停在 `preparation_ch28`，缺鑰匙進獨立壞結局；Load/runtime
> 對 item/兩臂 fail closed，測試固定原版 `0x24b14` 只掃 runtime slots0..15、無 camp/active filter，
> persistent roster fallback 則明記為 save/load projection。另已釘死真正取得路徑在零起算
> ch20_post（玩家第21章戰後）：必須集齊 `0xD1..0xD6` 六素材，成功才移除六件並 grant `0x64`；
> 目前 `battle_ch21` 還沒接這個 diamond，所以正常實玩仍拿不到鑰匙，下一批要接成
> `battle_ch21 → ch20_post → town_ch22`，不可無條件發鑰匙或跳過城鎮。

> **2026-07-16 第十七次 Codex 更新（玩家第21章戰後鑄造）**：已以完整 disassembly 更正
> 「六種各一件」簡化；原版其實計算 `D1..D6 × runtime slots0..15` 的 `(item,slot)` 命中組合，
> 總數必須**恰為6**，因此 duplicate 分散角色會改變結果。通用 editable `inventory_recipe`
> 現 byte-exact 保存這個怪癖、成功 pair-ordered 移除與 grant `0x64`，失敗不改 inventory。
> campaign 已接 `battle_ch21 → #5十句 → recipe → crafted #7..#10全16句 / insufficient #6全4句`，
> 兩臂共同 JOIN24/JOIN23、sync、chapter21，最後都回 `town_ch22`。layout/ACT63/64/`0x24336`
> 鑄造動畫仍待 lower，且更早章節尚無 D1..D6 正常取得路徑；文字／物品／持久化／城鎮流已接，
> 但不可宣稱這支視覺演出或 true-ending 實玩取得鏈已完整。Xvfb 已以真實 battle_ch21 context
> 實畫 #5 與 #6；#6 畫面仍會露出未 layout 的黑區，這是明列待辦，不以手寫鏡頭假裝還原。

> **2026-07-16 第十八次 Codex 更新（六素材正常取得鏈第一個可玩垂直切片）**：D1 已由
> EXE 人物 defaults 證實在索菲亞 `[36,A7,D1]`，並接入 ch11 party；D2/D6/D4 已由 FDFIELD
> composition terrain flag + slot + control reward 精確接成 map10 `(18,37)`、map12 `(38,18)`、
> map19 hidden `(30,7)` 的可編輯寶物。原版只在站上該格選「休息／待機」時取，背包滿不開箱，
> 敵我皆可取；runtime 已按此實作。D3/D5 不是泛用 inventory 搬運：特殊死亡 id39/id41 的 EXE
> handlers 明確 lower 為單一 `D3/D5` reward，已接 once-only death reward 與跨戰 party sync。
> ch11/13/15/17/20 勝利現在都先經 editable `postbattle_chNN_persist` 再回
> town12/14/16/18/21，沒有為保存素材跳過城鎮／商店／整備。尚未完成的是 D2/D6 獸人主動搶箱與
> 逃離 AI、普通寶箱 opened terrain+1 視覺、物品滿欄時原版互動轉移 UI；詳證見 `doc50 §3.10`。

## 0. 目前焦點(接手就做這裡)
`ch00_pre`、`ch00_post`、`ch01_pre`、`ch01_post` 已成為前四個 campaign 實際 consumer；ch01 post 的 branch、
reward、61-utterance FDTXT_002、dynamic speaker slots、PAN、SPAWN4、ACT14..16、JOIN/sync/chapter tail
與第二、第三章戰前／戰後 handler 均已完整 lower 且 compiler **0 issues**；ch03 turn3 的
slot6 active 條件、SPAWN2、兩段 PAN、800/200ms 與 FDTXT_003 #4 七句也已完整；第27章戰後
天空之鑰→第28章整備／壞結局 gate，以及玩家第21章戰後的六素材 recipe／完整分支文字／
共同 JOIN/sync/town22 均已接；D1 人物 default、D2/D6/D4 寶箱、D3/D5 特殊死亡 reward 與五個
關鍵戰後 persistence→town 節點也已完成第一個可玩垂直切片。下一個具體焦點是 D2/D6 獸人搶箱／
逃離 AI 與普通寶箱 opened 換圖，或 lower ch20_post 的 layout/ACT63/64/鑄造動畫，再選下一支
`0x233c6` post caller 依原版 arrays 補 binding。下方「草地深層未解」是 2026-07-06 歷史記錄，已被 2026-07-15 direct table 修正推翻，
不得再當目前 blocker。

## 1. 這段 session 做完的事
- **王座傳位幕**:走位 (8,42)→**(8,21)**第一次對話→**(8,8)**最終(對原版截圖+FDFIELD 守衛地標實測);
  守衛 dir=0(面向玩家);對話切分 line0 / line1-18;對話框修 4 項(文字不蓋頭像/上下框移入畫面/漸層/**長對白分頁**)。
- **草地幕(palace_path)**:亞雷斯 2 段進場(13,47→11,47→8,46 面向索爾)、進場句用**上框**、對話後索爾走到旁邊。
  ⚠ **「兩人一起走離+淡出」(結尾)先前試做又還原了**(見 §3 待辦)。
- **debug 工具**(cmd/fd2/main.go):`FD2_UNIT_LABELS=1`(sprite 標 `[idx]f<fig>(x,y)dDir`)、
  `FD2_CUTSCENE_LOG=1`(過場 node/beat/走位逐步印 stderr)。
- **文件集中化**:`doc50`=過場機制唯一主檔;新增 `scene-decode/ch1-throne.md`+`ch1-meadow.md`(每幕原始資料×解讀)。

## 2. 已驗證的 RE 定論(耐用真相,別再翻案)
- **走位來源 = step 家族 + 路徑走位 + acting normal frame**：`0x12eaa`下/`0x1300d`左/`0x13185`上/`0x13315`右(各推一格+捲鏡頭);
  通用 `0x13488(單位idx, 方向陣列, 步數)` 走任意路徑。王座是「全上」特例(直接 0x13185×15/13)。單位結構 +0X/+1Y/+3pose/+4tick/+8角色ID。
- **此 handoff 的 acting「只設面向」結論已於 2026-07-15 推翻**：normal frame 依 pose 每拍移一格，
  special frame 才原地顯示。格式與證據以 `doc50 §1.2` 為唯一準據。
  bit7 不改變 (unit,pose) 意義。normal frame 的低7位拍數=移動格數；special frame 的拍數才是
  原地顯示節奏。+4 tick 配繪製公式 `0x127e0=格+tick×f(pose)` 做每一格內的平滑內插。
- **map32 roster(dosbox dump `task_f/slots0_20_dialogue.bin`)**:slot0王/1后/**2=王座索爾**/**3=草地索爾(4,46)**/**4=草地亞雷斯(13,47)**/5-20守衛。
- **面向規則(全劇本)**:dir 預設 0(下/面向玩家);FDFIELD 不存面向;非0僅「走位者面向移動方向」或「劇情主角對看」。

## 3. ~~最大開放問題:草地主角走位~~（2026-07-15 已解）
- 錯表 decoder 才把 ACT101..105 誤讀成守衛16/17。direct resources 實際操作 slot3/4：ACT101/102
  讓亞雷斯接近，ACT103/104 原地定向，ACT105 讓索爾與亞雷斯離場。handler 顯式 ACT 已完整解釋影片，
  不存在額外走位機制或森林 context table。
- 正確機械輸出由 `tools/export_acting_resources.py` 直接讀 EXE 106-entry bank；舊本機 dump 僅考古。
- **方法論(使用者定)**:證據(截圖/影片)+ 已知機制 → 可「由上而下」回原版資料找出處,不必每次 RE 到底。

## 4. 其他待辦(worklist doc91;不急)
- ~~草地結尾兩人一起走~~：已由 direct ACT105 承接，不再用手寫 `exit_walks`。
- **對話分頁捲動動畫**(原版有「文字往上捲」;自寫平滑捲動即可,速度自訂非 RE)。
- **自動結束回合**(全員行動完自動換陣營,免手動 Tab)。
- **狀態欄位置**(HUD 擋單位,doc51)、**哈諾父子死亡→暴走**驗證、**export_units.py 全 33 章敵人數值**套合成公式。

## 5. 鐵則 / 紀律(這段 session 使用者立的,務必遵守)
- **[HARD] 禁臆測**:每個進 code 的值要有 RE 依據(反組譯/dosbox/青衫/影片/FDFIELD);拿不到→標「待RE」→外推前先問使用者。
  驗收=對 reference 實測(原版截圖/影片),不是「測試綠」或「看起來像」。(記憶 `fd2-goal-and-no-speculation-rule`)
- **[HARD] 知識集中一份 markdown**:動手新增文件前先查既有→擴展它;其他檔只引用不複製。過場機制=doc50。
- **[HARD] sonnet 只做 coding;比較/判斷/驗證/反組譯語意 一律旗艦親自做**:sonnet 反組譯猜錯 6/7 原語、
  截圖判讀也會幻覺(回報「視覺達標」實測沒有)。派 sonnet 實作後,「像不像/算不算完成」旗艦親自截圖親看。
- **dosbox 不萬能**:heavy-debug 下執行類斷點卡死;採樣率跟不上快變值會誤判;headless 截圖 fps≠60 送鍵易對不上。
  優先靜態 RE + 原版截圖(靜止參照);Go 測試(確定性)驗邏輯、截圖驗版面。
- **我這 session 自己犯又修的錯(別重犯)**:①「15呼叫=15格→row27」線性外推錯(→21);②「(8,8)改(8,14)」誤判(→8);
  ③ 此處「acting 只設面向」的舊判讀已撤回；後續請以 doc50 的 2026-07-15 更正為準。

## 6. 關鍵檔案地圖
- **機制主檔**:`doc50-cutscene-script-system-design.md`(過場原語/走位/acting/handler/DSL)。
- **每幕原始資料×解讀**:`scene-decode/ch1-throne.md`(含 acting byte 反組譯附錄)、`scene-decode/ch1-meadow.md`。
- **handler 逐 beat 轉錄**:`doc47`(§3 三段/§7 機械抽取/§9 走位實驗/§10-11 step 公式)。
- **草地影片量測**:`doc55`;**remake↔文件溯源+落差**:`doc44 §5`;**dosbox 實測**:`doc48`。
- **remake 對話框渲染規則**:`doc09`。**戰鬥演出**:`doc35`。
- **原版 dump**(本機,gitignore):`extracted/dosbox_dump/`(acting_decoded/、task_e|f/slots、out/);
  **原始 .DAT 解包**:`extracted/raw/`(FDFIELD/FDTXT/FDOTHER…);**原版錄影**:`video/fd2-ch1.mp4`。
- **工具**:`tools/disasm_le.py`(反組譯,docker `fd2-cap`)、`tools/parse_field.py`(FDFIELD)、
  `tools/export_acting_resources.py`（由 FD2.EXE direct bank 產生／檢查 106-entry editable JSON）。
  `extracted/.../decode_acting.py` 與舊 transcript 是 gitignore 考古物，不得作 canonical input。
- **remake**:`remake/`(build:`cd remake && ./build.sh` docker;跑:`./play.sh`;headless 截圖:見 play.sh --shot 或 FD2_SHOT env)。

## 7. 環境速記
- 反組譯:`docker run --rm -v /home/anr2/cht/fd2:/w -w /w fd2-cap sh -c "python3 tools/disasm_le.py 'org_game/炎龍騎士團/FLAME2/FD2.EXE' range 0xA 0xB"`
- headless 截圖:`xvfb-run -a -s "-screen 0 1280x800x24" env LIBGL_ALWAYS_SOFTWARE=1 GALLIUM_DRIVER=llvmpipe FD2_MUTE=1 FD2_CAMPAIGN=... FD2_CAMP_NODE=<node> FD2_SHOT=out.png FD2_SHOT_FRAME=N ./fd2-linux`
- EXE 位址:tool-linear = 檔內位址(disasm 直接吃);執行期位址另有 loader 偏移(見 doc48 §5)。
- **每輪做完 commit + push**(CLAUDE.md 要求)。素材/dump/org_game/references 一律 gitignore 不入庫。

## 2026-07-20 Codex shop transaction slice
- 商店購買已拆成原版順序：選收件者後先插入未裝備物品，裝備類進入「要裝備上去嗎？」；Enter 裝備、ESC 保留未裝備，兩者最後才扣款。
- `battle.Unit` 新增與 Inventory 對齊的 `Equipped []bool`，並在 persistent sync、recipe 移除、死亡獎勵、寶箱領取時維持欄位對齊；`ClassID` 亦納入 persistent overlay。
- `campaign.ReserveGood`/`FinalizeGood` 保留兩階段交易原語；`BuyGood` 維持既有一次完成 API。
- 已以官方 Go 1.22 容器跑 `go test ./internal/campaign ./internal/battle ./cmd/fd2 -count=1`；能力值重算與換裝（覆蓋同類舊裝）仍待下一輪 RE/實作。
- 本輪補上 `campaign.SellGood` 純交易核心（原價 3/4、先驗證再移除欄位）；尚未接 UI。裝備數值暫不臆測：現有 FDFIELD/character dump 丟失原版 inventory slot flag，且 scenario AP/DP/HIT/EV 是有效值，必須先補 provenance 才能安全重算。
- 2026-07-20 後續：商店已接賣出 UI（Tab 切換、角色、指定 inventory slot、ESC 返回），以 `item.json.price` 載入原價並呼叫 `SellSlot`；duplicate item ID 不會賣錯欄位。能力重算仍刻意保留待 RE。
- 2026-07-20 RE 補證：以 `tools/disasm_le.py` 反組譯原版 `0x1145a` 與 `0x1c142`。`0x1145a` 明確掃 8 個 `+0x0a+slot*2`，檢查第一 byte `bit 0x40`，從 item record `+1/+5/+3/+7` 累加 AP/DP/HIT/EV；`0x1c142` 的換裝規則是 item ID `<0x80` 與 `>=0x80` 分兩類，清除同類已裝備 flag，再將新 slot 第一 byte 寫 `0x40`。
- 2026-07-20 provenance slice：`parse_field.py`/`export_units.py`/`dump_exe_tables.py` 現在保留固定長 `inventory_slots`（FDFIELD 8 bytes、character defaults 6 bytes 後補兩個 `0xff`），不再只存 compact inventory；33 張 map units 與 ch11 Sophia 已帶入。`Unit.AddInventoryItem`/`RemoveInventoryIndex`、寶箱、死亡獎勵、配方、商店、persistent sync 都同步維護 raw slots。
- remake 已新增 `BaseAP/BaseDP/BaseHIT/BaseEV` 與 `RecomputeEquipment`。進一步確認原版 spawn `0x10f06..0x10f31`：source inventory 前兩 bytes 直接寫成 flag `0x40`，後續 bytes 為 `0x80` held；remake 現在 materialize 前兩欄 equipped，並由 `InitializeEquipmentBase` 從 authored effective 值扣回一次，避免 double-count。raw `inventory_slots` 已保留原始 `0xff` 空槽位置，並由新增/移除/同步流程維護。
- 2026-07-20 materialization 修正：原始 `inventory_slots` 是 FDFIELD source bytes，不是 runtime 欄位。依 `0x10f06..0x10f31` 的分支，source[0] 為 `0xff` 時 source[1] 會壓入 runtime slot0；否則 source[0]/source[1] 分別進 runtime slot0/1，source[2..7] 保留原位。`Load` 與 `PartyUnits` 現在先 materialize 成 8 格 runtime slots，再建立 compact `Inventory` 與對齊的 `Equipped`；商店、寶箱、配方、死亡獎勵以 runtime slot 操作，避免內部空槽時錯移裝備。核心測試與 `go test -c ./cmd/fd2` 已通過。
- 2026-07-20 town/preparation audit：`campaign_full.json` 的 ch01→town02、ch02..21→postbattle→town、ch22..24 的連戰 preparation、ch25→town26、ch26→town27、ch27→prep28、ch28→prep29、ch29→prep30、ch30→ending 串接均已盤點；town 的 shop/rumor/church 返回原 town。尚缺的是 `main.go` 的 preparation 編成與 church 行為仍為 placeholder（Enter/ESC 直接 Advance），下一輪需先建立可編輯的 party/deploy/equipment 整備節點與 persistent roster/gold 不丟失測試。
- 2026-07-20 item range RE：`0x14237` generic attack target path 讀 item `+0x0c range_min` 並傳給 `0x14818` 的 Manhattan geometry cutoff；`+0x0b atk_rate` 在該路徑未再讀，`+0x0d range_max` 只在特殊 item/effect 路徑出現，不能臆測成通用 `AtkMax`。`0x1df3f` 另以 `atk_rate` 做特殊限制，`0x1ed6a` 將 `atk_attr` 帶入攻擊效果分支。remake 繼續使用已驗證的 `weapon_range.json`，完整 item multiplier/effect 仍待 direct callsite。
- 2026-07-20 preparation slice：反組譯 `0x318ad` 證實整備畫面建立 30-byte 勾選表；一般章節 cap=`0x0f` (15)，late route cap=`0x13` (19)，方向鍵移動、Enter 切換角色，達 cap 後自動離開。remake 新增 `Node.party_limit`（ch28–30=19，其他 preparation 以 direct fallback=15）、`Game.partyDeploy` 暫時出擊名冊與 preparation UI；永久 `partyMembers` 不被改寫，下一場 battle 只以 `partyDeploy` filter，避免 JOIN/save roster 遺失。核心測試與 GUI compile 已通過。
- 2026-07-20 church RE：`0x3072f` 讀教會服務選擇並分派 `0..3` 到 `0x2ffa5/0x2f8ea/0x30dc3/0x31385`；`0x30dc3` 的 `0x24c` 是「無須復活」訊息，存在死亡候選才用 `0x24d` 選人，確認費用後 `0x2d516` 扣款、清 `[unit+5]` 死亡 flag、把 `[unit+0x42]` 複製到 `[unit+0x40]` 恢復 HP。`0x31385` 的 `0x24f/0x250/0x252` 分別是無候選／選人／確認轉職。教會沒有免費一般治療分支；下一步先資料化 revive/class-change service nodes。
- 2026-07-20 revive core slice：直接讀出 `0x52669 + class*2` 的 29 筆 u16 class fee table，新增 `docs/data/exe_tables/revive_fee_rates.json` 與 editable runtime copy `remake/assets/data/revive_fee_rates.json`。`campaign.ReviveUnit` 依已證實公式 `feeRate * level` 先驗金額，再原子寫回 HP=MaxHP、清除死亡投影 `OnField`；不足金或非死亡候選不改狀態。尚未把 church selector 接到 UI，也尚未追完 class-change 寫回能力表。
- 2026-07-20 class-change candidate slice：`campaign.CanChangeClass`／`ClassChangeCandidates` 已接上 `0x31793` 的 exact filter：Lv>=20、portrait<0x12 且 portrait!=7，保留 JOIN order；尚未實作 `0x31860` 道具分支與 `0x2a2e8` class/portrait/能力寫回。
- 2026-07-20 church selector slice：`main.go` church 節點已從直接返回 town 改成四項服務選單；第3項接 `campaign.ReviveUnit` 與 EXE fee table，第4項顯示 exact class-change candidates但保留 item/能力寫回待接。xvfb 實機截圖已存 `docs/figures/church-selector.png`。
- 2026-07-20 class-change RE continuation：`0x3151a..0x3152d` 依 portrait 查轉職道具（portrait 0x34→item 0x5a，其餘 promoted portrait→`0x526a7+portrait` byte），`0x31860` 掃 8 個 inventory slot；成功後 `0x1b8e7` 移除 item、`0x2a2e8` 重算、`0x31571..0x3157a` 寫 class(+0x20)/portrait(+7)。目前只接 candidate/UI，mapping table 尚待完整導出。
- 2026-07-20 class target tables：已導出 `0x615fe` portrait→(class,mobility increment) pairs 與 `0x526a7` raw item bytes 至 `docs/data/exe_tables/class_change_targets.json`；portrait 0x34 的 item 0x5a special override 已明列。`0xff` raw item 代表該 target branch 尚不能直接視為可用道具，runtime 尚不接猜測性 class mutation。
- 2026-07-20 class target table correction：原表把 `0x526a7` 誤標成 target portrait index；依 `0x31793` 實際指令，現在拆成 `current_portraits`（current portrait 0..0x11，default=current+0x20、optional=current+0x32，raw item `[0x526a7+current]`）與 `target_portraits`（`0x615fe` 的 class/mobility increment pairs）。raw `0xff` 不建立 optional target；current portrait 9 的 item 0x5a→target 0x34 special branch 保留。新增 `class_change_table_test.go` 驗證 18/34 rows 與 index 對齊。
- 2026-07-20 `0x31602` stat-reset 定案（更正）：`0x4e4d1(portrait)=0x620a1+portrait*0x0b` 的 11-byte 成長列，五組 row pairs 經 `0x1e529` **加到既有** unit words `+0x37(AP),+0x39(DP),+0x3e(DX/HIT-EV base),+0x42(MaxHP),+0x46(MaxMP)`；`0x1e529` 尾端是 `add word [target], ax`，不是覆寫。`+0x40/+0x44` 由後段回填 current HP/MP；`0x4e48d(new portrait)+1` 的 mobility increment 加到 raw `+0x3b`。流程清 raw EXP `+0x3c`，**未寫 level byte，故保留原 Lv**，HP/MP 全滿。row random 是 pair 的 `[min,max)` 取值。
- 2026-07-20 class mutation core slice：`campaign.ApplyClassChange` 依 `0x31602` 寫回 target portrait/class、AP/DP/DX/MaxHP/MaxMP、MV(+0x3b)、Lv=1/Exp=0/HP=MaxHP/MP=MaxMP，並移除 branch item；invalid range/item 失敗不改動 unit。新增成功與 atomic rollback tests；尚未把 UI/JSON growth rows 接上，也尚未呼叫 equipment recompute（避免猜測舊 Base*）。
- 2026-07-20 class-change editable bridge：新增 `LoadClassChangeTable` 解析 current/target portrait maps，`ClassChangeTargets` 依原版順序產生 default/optional/special branches 與 compact inventory index，並以 `LoadClassChangeGrowth` 將 `growth.json` idx32..67 映射至 target portrait 0x20..0x43；已驗證 18/34 target rows、36 growth rows 與道具存在條件。
- 2026-07-20 church class target UI slice：church 選轉職角色後進入 target branch 選單，依 default/optional/special 顯示 target portrait/class；Enter 會用 shared RNG 執行 `ApplyClassChange`、移除 branch item、設定 class name、重建 equipment base/recompute 並寫回 `partyRoster`。新增 runtime `class_change_targets.json`/`class_change_growth.json`（force-added，assets 預設 ignore）；GUI compile 與 campaign/battle tests 通過。尚待 xvfb 實機操作驗證與 race/multiplier raw bytes 接線。
- 2026-07-20 `0x1b750` synthesis continuation（校正）：`0x1b750` 讀 raw `+0x37/+0x39/+0x3e` 與 item table 23-byte row 的 `+1/+3/+5/+7`，寫 derived `+0x48/+0x4a/+0x4c/+0x4e`；它是 class path 後的 equipment/stat synthesis，不是 screen-only projection。`+0x22/+0x23/+0x24` 雖會影響該 routine 的 transient branches，但 constructor 先清零且 `0x31602` 不寫它們，不能當成 class growth source。`campaign.RecomputeAfterClassChange` 與 double-count regression test 已保留。
- 2026-07-20 xvfb fixture hook：`FD2_CAMP_CLASS_FIXTURE=1` 僅供 headless oracle，注入一名 Lv20 portrait9（索爾顯示名）並帶 item 0x58/0x5a；在 `FD2_CAMP_NODE=church_ch02` 下可用 xdotool `Down×3 Enter` 進轉職、再 `Enter` 選唯一候選，target branch 畫面應列 default portrait 0x29、optional 0x3b、special 0x34。此 hook 不在正常啟動路徑。
- 2026-07-20 xvfb class-target proof：以 fixture、`FD2_CAMPAIGN=assets/scenarios/campaign_full.json`、church_ch02 與 xdotool 減速按鍵實機操作，成功截得三分支畫面 [`docs/figures/church-class-targets.png`](../figures/church-class-targets.png)；畫面文字實際顯示「基本轉職 → portrait 29h / class 13」、「道具 58h → portrait 3Bh / class 22」、「特殊道具 5Ah → portrait 34h / class 21」。
- 2026-07-20 battle progression slice：campaign `Node.Protect` 已資料化，`checkResult` 依 battle node 的 protect 欄位判定敗北，空值維持索爾相容預設；新增 campaign test。另修正升級：原版 DX 是 HIT/EV 共用 raw base，`GainExp` 在已有 equipment base 時同步更新 BaseHIT/BaseEV 與有效 HIT/EV，保留裝備加成並新增 regression test。
- 2026-07-20 AI low-damage slice：依 `docs/knowledge-base/11-enemy-ai.md` 的 `0x15140` 證據，AI 候選目標若預估 `dmg≤2` 直接略過，不會為了微小傷害發動攻擊；若沒有合格目標則保留接近／待命計畫。`aiActUnit` 與 `NextAIPlan` 共用同一套候選篩選，並以固定場景測試邊界值 2/3，避免兩條執行路徑漂移。情境加成、狀態倍率及敵方施法入口仍待後續 RE。
- 2026-07-20 AI spell-entry audit：臨時 capstone 容器 direct disasm `0x15470..0x15618`，並查到呼叫點 `0x13E39`、`0x14F9B`。`0x1548E` 才是函式入口；`0x154D1` 位於其本體，實際流程可見 `0x14B78` 路徑／移動與 `0x12D7B` 演出狀態呼叫，沒有 `Cast` dispatch 證據。已撤回「0x154D1 是施法入口」舊註記；敵方 AI 施法仍待從法術函式反向找真正 callsite。
- 2026-07-20 AI spell dispatch proof：direct disasm `0x15688..0x15880` 與 `0x14F80..0x15220` 證實原版 AI 會枚舉並執行法術命令：`0x1579A–0x157B5` 將 `command>0x0F` 轉為 `spell_id=command-0x10` 呼叫 `0x149F8` 評分；選中後 `0x150D3–0x150F1` 重算同一 spell，`0x15168→0x28784` 播放施法演出。`0x154D1` 仍只是移動函式中段。remake 下一步需把 SpellID／command inventory 與攻擊、治療目標優先級接到 `NextAIPlan`。
- 2026-07-20 AI spell data bridge：remake `battle.State` 新增可注入 `SpellBook`，`AIPlan.SpellID` 以 `-1` 明確表示目前物理／待命計畫不施法；`loadGame` 將已載入的 EXE spell table 複製進 state，並新增 regression test 防止物理 AI 偷生 spell command。刻意未加入猜測性的 spell ranking、治療目標或施法座標；這些要等 command inventory 對映與 `0x15880/0x15B77` 語意定案。
- 2026-07-20 AI spell-family scoring：direct disasm `0x15B77..0x15DA1` 證實法術目標選擇不是通用物理評分：spell `0..12` 掃攻擊目標並累加 8/0x18 等優先分數，`13..16` 掃補血目標，`17..19` 走增益分支，`20..22`、`26`、`27` 走狀態／毒麻分支，部分條件由 `0x1C269` 檢查。依 `03-exe-and-data-structures.md`，`unit+0x22..+0x26` 是 M1–M5 習得 bitfield、`+0x27` 起為 RA/CL/LV，已修正文檔避免稱為狀態旗標；remake 仍不猜接線。
- 2026-07-20 class-change fidelity correction：使用者實測指出轉職結果與原版差距巨大；direct disasm `0x1E529` 尾端確認是 `add word [target], ax`，PTT 實測表亦吻合「舊能力 + 新職 growth row」而非絕對重設。已修正 `ApplyClassChange`：AP/DP/DX/MaxHP/MaxMP 改為累加、Lv 保留、EXP 清零、HP/MP 回滿；campaign/battle 測試通過。外部旁證：[PTT 實測表](https://www.ptt.cc/bbs/Dynasty/M.1185344950.A.91B.html)、[FD2 轉職攻略](https://jaceju-favorite-games.gitbooks.io/fd2/content/walkthrough/INDEX.html)。
- 2026-07-20 外部流程盤點：攻略頁逐章列出武器店／道具店／教會／神秘店，至少第4、7、9、14、16、18、19、21章有明確整備設施與隱藏商店證據（[第4章](https://jaceju-favorite-games.gitbooks.io/fd2/content/walkthrough/4.html)、[第7章](https://jaceju-favorite-games.gitbooks.io/fd2/content/walkthrough/7.html)、[第16章](https://jaceju-favorite-games.gitbooks.io/fd2/content/walkthrough/16.html)）。頁面未明文保證「勝利後立即進入」，故只作 campaign town/shop 節點的外部交叉證據，不取代 EXE branch/table；`campaign_full.json` 仍須保留 postbattle→town/preparation→next battle 的可編輯順序。
- 2026-07-20 class-change equipment correction：發現 `ApplyClassChange` 先改有效 AP/DP/MV、再呼叫 `RecomputeAfterClassChange` 時，已有裝備會被重算兩次。現以既有 equipped item 貢獻反推 raw base，再套用已確認的 `0x1b750` stat/equipment synthesis；新增回歸測試證明 AP/DP 不會由 18/12 錯變 21/14，campaign/battle 測試通過。
- 2026-07-20 handoff reconciliation：本檔較早的 church/class-change 條目是歷史快照；其中「Lv=1」、「尚未接 UI／能力寫回」、「church 仍是 placeholder」等描述已由後續 RE 與實作更正。現行權威狀態是：保留 Lv、清 EXP、五組成長累加、target/item/UI/persistent roster 已接；仍待的是 `+0x22/+0x23/+0x24` transient writer 的完整來源、原版實機數值回歸與完整 GUI 轉職操作截圖。
- 2026-07-20 class raw-field audit：`0x1b750` 的 AP/DP/HIT/EV synthesis 會讀 `+0x22/+0x23/+0x24` 的非零旗標分支，但 spawn constructor `0x10f6b..0x10fa5` 先把 `+0x22..+0x27` 清零，且 `0x31602` class path 不寫這些 bytes；它們是後續 transient/effect writer 的欄位，不是 M1–M5 spell bitfield，也不是可直接從 class growth row 匯出的 modifier。remake 暫不猜測接線。
- 2026-07-20 raw-unit pointer/schema 對齊：spawn constructor `0x10f6b..0x10fa5` 直接證實 FDFIELD b13..b16 的 `magic_raw` 複製到 runtime `unit+0x1a..+0x1d`；runtime `+0x22..+0x27` 另以 memset 清零，後續才由能力流程寫入 modifier flags。因此 `+0x22/+0x23/+0x24` 不是 spells，且目前沒有可從 FDFIELD 直接匯出的非零值；已同步修正 `03-exe-and-data-structures.md` 與 `11-enemy-ai.md`，避免錯誤 schema 繼續污染 remake。
- 2026-07-20 AI command inventory slice：item EXE row 是 23 bytes，K4 command 位於 raw byte `0x11`（item 79 的 `0x1f`→spell 15）；新增 `campaign.LoadAICommandSpellMap` 與 `State.AICommandSpell`，只資料化 command `>=0x10`，不猜測 AI ranking／治療目標。campaign/battle 核心測試通過。
- 2026-07-20 AI available-spell slice：新增 `State.AIAvailableSpells(unit)`，依 unit inventory 順序把 command map 解析出的 spell IDs 對到 EXE `SpellBook`，去重且忽略未知 spell；此層只重現 command inventory，不改 `NextAIPlan` 的目標評分或施法執行。
- 2026-07-20 AI spell-family candidate slice：新增 `State.AISpellCandidates`，依 direct `0x15B77` family 分支提供 attack(0..12)、heal(13..16)、buff(17..19)、cure(20 解毒／21 祛麻，僅掃對應己方狀態)、status(22/26/27) 的 live/camp 候選掃描；保留 runtime order，不猜原版分數與施法執行。
- 2026-07-20 story script fallback slice：`campaign.Runner.NodeID()` 暴露目前 editable node key；`main.enterNode` 對精確 `story_chNN` generic node 自動載入 `assets/story/chNN.json`，因此 ch04–30 等已有完整可編輯劇本不再只播兩句節點 fallback。named/pre/post cutscene 不套用，避免整章重播；Xvfb GUI package test 通過。
- 2026-07-20 ch02/ch03 handler audit：`ch02_pre` 的四組 dialogue index 已由 `count-aligned.json` 對到 `ch03.json` scene0 lines 0–13，並有 act18/17/19、spawn/pan/layout overrides；`ch02_post` 的 Tino 分支對到 scene1 lines1–5，else 分支對到 lines6–15、JOIN char2、sync/set_chapter3。`ch03_post` 僅有一段已證實對到 `ch04.json` scene3 lines0–3。進一步以 jump-table index3、`load_chapter` 的 FDTXT(章節+1) 規則及 direct push index 證實 `ch03_pre` 的 idx0/idx1 分別是 `FDTXT_004` string #0/#1，新增 `bindings/ch03_pre.json`（scene0 lines0–3、scene1 lines0–4、map3/acting20），並將 `story_ch04` 接回 handler；campaign regression 通過。
- 2026-07-20 ch04_pre slice：同一 FDTXT(章節+1) 規則與 `count-aligned` 證實 handler `0x33049` 的 idx0/1/2 對 `FDTXT_005` → `ch05.json` 的 scene0 lines0–5、scene1 lines0–8；新增 `bindings/ch04_pre.json`（map4 50-slot frontier、pan 3,3/8,14、acting22/21），`story_ch05` 現在實際執行可編輯 pre-handler，不再空 cutscene。campaign/battle 全套 regression 通過。
- 2026-07-20 cross-scene dialogue adapter：`HandlerDialog.Segments[]` 現在保留一個 native FDTXT lookup 的 scene-target 順序，compiler 逐 segment→line flatten 成普通 dialog beats；runtime 每拍依明確 Script/Scene/SceneIndex 載入，沒有文字猜測或跨 scene Count。`FDTXT_006 #0` 的 18 句已通過 scene0(1)→scene1(3)→scene2(5)→scene3(9) regression，`ch05_pre` binding 完整，`story_ch06` 接回 editable handler。
- 2026-07-20 ch06_pre slice：`FDTXT_007` index0/1 都是單 scene mapping（2+6句），handler `0x33169` 的 map6/40-slot、pan 8,1→8,0、acting28/29 已新增 binding；`story_ch07` 接回原版 pre-handler，campaign/battle regression 通過。
- 2026-07-20 ch07_pre slice：`FDTXT_008` index0（跨兩 scene、15句）與 index1（2句）由 segments adapter 展開；handler `0x33219` 的 map7/60-slot、pan 7,32→7,23、acting31/32 已新增 binding，`story_ch08` 接回 pre-handler，campaign/battle regression 通過。
- 2026-07-20 ch08_pre slice：`FDTXT_009` index0/1（2+5句，單 scene）與 handler `0x3327d` map8/60-slot、pan 6,0、acting35 已新增 binding；`story_ch09` 接回原版 pre-handler，campaign/battle regression 通過。
- 2026-07-20 ch09_pre slice：`FDTXT_010` index0 跨 scene0/1 共12句，handler `0x3332b` map9/60-slot、pan 10,0 已新增 binding；segments adapter 維持 6+6 line 順序，`story_ch10` 接回 pre-handler，campaign/battle regression 通過。
- 2026-07-20 ch10_pre slice：`FDTXT_011` index0 跨 scene0/1/2（4+6+2句），index1/2 延續 scene2；handler `0x33367` map10/40-slot、pan 10,7、acting38/39 已新增 binding，`story_ch11` 接回 pre-handler，campaign/battle regression 通過。
- 2026-07-20 ch11_pre slice：`FDTXT_012` index0 跨 scene0/1（2+9句），handler `0x333f5` map11/60-slot、pan 4,4→11,40、acting40/41 已新增 binding；`story_ch12` 接回 pre-handler，campaign/battle regression 通過。
- 2026-07-20 ch12_pre slice：`FDTXT_013` index0 單 scene 6句，handler `0x3346b` map12/70-slot、loadch/ch13 script 已新增 binding；`story_ch13` 接回 pre-handler，campaign/battle regression 通過。
- 2026-07-20 ch13_pre slice：`FDTXT_014` index0 單 scene 4句，handler `0x3347c` map13/70-slot、pan 20,20、loadch/ch14 script 已新增 binding；`story_ch14` 接回 pre-handler，campaign/battle regression 通過。
- 2026-07-20 ch14/ch15 boundary：`ch14_pre` 含已證實的 `roster_has(12)`，其 EBX/EAX 動態 text index 尚待 direct control-flow mapping，暫不猜接線。下一個無動態分支的 `ch15_pre` 已完成：FDTXT_016 index0 16句、map15/60-slot、ch16 script，`story_ch16` 接回 pre-handler，campaign/battle regression 通過。
- 2026-07-20 ch17_pre slice：`FDTXT_018` index0/1/2（7+4+13句，segments 保留 scene 邊界），handler `0x335da` map17/70-slot、pan 16,4、acting54/55 已新增 binding；`story_ch18` 接回 pre-handler，campaign/battle regression 通過。
- 2026-07-20 ch18_pre slice：handler `0x33475` 的實際 pre 呼叫只有 FDTXT_019 index0（8句，scene0），已新增 map18/70-slot 與 ch19 script binding；`story_ch19` 接回 pre-handler，campaign/battle regression 通過。其餘 FDTXT_019 strings 不在此 handler 呼叫，未擅自播完整章節。
- 2026-07-20 ch19_pre slice：handler `0x33475` 的 FDTXT_020 index0（17句，scene0）已新增 map19/70-slot、ch20 script binding；`story_ch20` 接回 pre-handler，campaign/battle regression 通過。
- 2026-07-20 ch20_pre slice：handler `0x33475` 的 FDTXT_021 index0（17句，scene0）已新增 map20/80-slot、ch21 script binding；`story_ch21` 接回 pre-handler，campaign/battle regression 通過。
- 2026-07-20 save durability slice：save JSON 改以同目錄暫存檔後 `rename` 原子替換，避免戰後 town／商店／整備節點存檔時因程序中斷留下半份 JSON；新增完整內容與暫存檔清理 regression test。campaign/battle 核心測試通過；GUI package 測試在目前容器缺少 ALSA/X11 headers，需用含圖形依賴的驗證容器重跑。
- 2026-07-20 external flow cross-check（非 EXE 硬證據）：GameFAQs、PTT 與中文攻略逐章列出 Town of Rod、Sara Village、武器店／道具店／教會／旅館／神秘商店，以及戰後角色加入與下一段旅程；這支持保留 postbattle→town/shop/church/preparation 的可編輯節點，但精確順序仍以 `campaign_full.json` 與 direct disassembly 為準。參考：[GameFAQs walkthrough](https://gamefaqs.gamespot.com/pc/582384/flame-dragon-2/faqs/31054)、[第4章攻略](https://jaceju-favorite-games.gitbooks.io/fd2/content/walkthrough/4.html)、[第16章攻略](https://jaceju-favorite-games.gitbooks.io/fd2/content/walkthrough/16.html)。
- 2026-07-20 ch21_pre slice：handler 的 FDTXT_022 index0 實際為 11 句、scene0；binding 已補上 map21/70-slot、pan(16,28)、acting67 與 ch22 script/party scenario，`story_ch22` 改接 editable pre-handler。新增 compiler regression，確認段落順序、載入、鏡頭與演出資源；campaign/battle tests 通過。
- 2026-07-20 web survey（僅作外部交叉證據）：公開資源頁確認原版以外部 `FDFIELD.DAT`（含 mod 目錄替換）提供場景資料，故後續 loader 應保留 DAT provider/override layer，不把所有內容假定在 EXE。攻略資料亦明載章節間先進戰鬥準備，可購買／換裝、教會復活、存讀檔後才進下一章；campaign graph 必須維持 battle→postbattle/town/preparation→next battle。參考：[FD2 資源頁](https://chiuinan.github.io/game/game/intro/ch/c31/fd2.htm)、[準備畫面介紹](https://leoandvc.pixnet.net/blog/posts/13079662050)、[第七章商店觸發](https://jaceju-favorite-games.gitbooks.io/fd2/content/walkthrough/7.html#L55-L58)。尚未找到可靠公開 DAT binary 格式，格式結論仍以本地檔案與反組譯為準。
- 同輪補充：GitBook 的 [FD2.EXE 修改表](https://jaceju-favorite-games.gitbooks.io/fd2/content/modify/FD2_EXE.html) 可作低成本行為 oracle（入隊 ID、行動後再動、隨時存檔、等級上限、寶箱持久化）；只採其可對照的行為線索，不把社群 byte patch 當 loader 格式證據。
- 2026-07-20 ch22_pre control-flow slice：`0x336b5` 的 `EBX` 不是 roster_has，而是 `repeat_hint(limit=16, loop_back=0x336b4)` 的固定清理迴圈；compiler 現在把 `unit_slot_expr:"ebx"` 明確展開成 slots 0..15，並以 active loadch slot_count 驗證。這解開 ch22 的動態索引阻塞；`0x24618` 視覺效果與 `0x11df2` palette/fade 仍保留 unknown，故 story_ch23 尚未猜接。
- 2026-07-20 palette/transition RE correction：`0x11df2(start,end,delta)` 對 `[0x53a65]` 每色 RGB 加 delta、clamp 0..0x3f 後逐項寫 VGA DAC，是一次性 `palette_update`；`0x11d40` 才是減亮 fade-out。compiler 已將 native `0x11df2` immediate calls lower 成 `palette_update`（ch22 呼叫皆 delta=0，runtime 保留順序，不誤當黑幕 fade）。`0x24618` 定案為 `(x,y,palette_delta,step)` 固定 9-frame transition/reveal（每幀 present、5ms、尾端 delay500ms），仍待 indexed transition renderer，未猜接。
- 2026-07-20 ch23_pre slice：handler `0x338ce` 的 FDTXT_024 index0/index1 共 14 句（scene0，5+9），map23/70-slot、四段 pan(0,4→0,22→26,24→26,2)、spawn group1 與 ch24 script 已新增 binding；`story_ch24` 接回 editable pre-handler，compiler/campaign/battle regression 通過。
- 2026-07-20 ch24 transition/audio slice：`0x24b4d(count)` 完整 RE 確認為先 terrain/main redraw，再以兩個 `0x1c8` buffer 交替 present、每幀 20ms；ch24 calls 為 20/20/20/60（400ms/1.2s）。compiler/runtime 已新增 `transition_reveal`，binding 將 `load_res` FDOTHER#88、四次 `play_sfx(priority=1,index=1)` 接到 `battle_88_01.wav`，並把 `0x1d50a` 的 index=-1 stop 與 `0x1a80a` release 接回 handle；`story_ch25` 已切回 editable handler。剩餘差異只在 indexed double-buffer visual adapter（目前保留 exact count/timing，PNG renderer 每幀重繪）。
- 2026-07-20 ch25_pre slice：FDTXT_026 全章因後續分支／raw utterance 與 authored line 數不一致，未宣稱全量 count-aligned；但 handler 實際呼叫的 string0 可直接對到 `ch26.json` scene0 12 lines。已新增 ch25_pre binding（map25/70-slot、pan 9,39、acting76、dialog line0/count12/scene0）並將 `story_ch26` 接回 editable handler；後續 FDTXT_026 分支仍待條件控制流 mapping。
- 2026-07-20 ch26_pre slice：逐字解析 FDTXT_027 證實 handler 的 idx0/3/4/5/6/7 分別對到 `ch27.json` scene0 的 lines 0–3、4、5–6、7、8–12、13–21；新增 `bindings/ch26_pre.json` 與六組 direct line/count overrides，`story_ch27` 接回 editable pre-handler。`0x24b14(100)` 依既有 LE disasm 是天空之鑰 item `0x64` 的 16-slot inventory gate，仍保留 unresolved branch/effect，不把 gate 猜成自動跳轉。
- 2026-07-20 ch27_post slice：`FDTXT_028` 已 count-aligned，handler idx7 (`0x231e5`) 精確對到 `ch28.json` scene1 lines 11–15；新增 `bindings/ch27_post.json`，掛在天空之鑰 present branch 後、進 preparation_ch28 前。sync_party/set_chapter 原語保留原順序。
- 2026-07-20 ch28_pre audit→resolved：FDTXT_029 idx7/idx8 分別精確對到 `ch29.json` scene1 lines 5–12（8句）與 scene2 lines 0–5（6句），pan(9,56)→(216,1344)、acting86 已建 binding。Capstone direct disasm 證實 `0x35822(x,y,group)` 為 pan→spawn→delay300→palette(0,255,0)→delay200→palette(0,255,0)→redraw；compiler 已 lower 且 `story_ch28` 已接回 handler，無 unresolved issues。
- 2026-07-20 ch26_post gate audit：`0x25186` 後 `cmp eax,-1 / JE 0x25348` 證實 item `0x64` 缺失會進 FDTXT_027 idx13–16 離別支線，命中才繼續 idx9–12 正線並 `sync_party/set_chapter(27)`；campaign gate 現已承載缺匙對話 scene→ending，ch26_post 的大量 visual/effect unknown 仍待拆解。
- 2026-07-20 missing-key branch slice：新增 `ch27.json` 可編輯場景「缺少天空之鑰的離別(分支)」，收錄 FDTXT_027 idx13–16 的 17 句離別對話；`inventory_gate_ch27_sky_key.if_missing` 現在先進該 scene，再接 `ending_ch27_no_sky_key`，不再用 generic ending 吞掉原版對白。未解析的 0x25052/0x24618/0x1c2da/0x22253 視覺／系統效果仍刻意保留為待辦。
- 2026-07-20 isolated RE toolchain：新增 `tools/docker/fd2-cap.Dockerfile`，建立本機 `fd2-cap-local` image（Python 3.12 + capstone 5.0.3）；所有後續 `disasm_le.py` 以 repo read-only mount 執行，不污染 host Python。實際 Capstone disasm 確認 `0x35822(x,y,group)` 是 pan→spawn→delay300→palette(0,255,0)→delay200→palette(0,255,0)→redraw；compiler 已 lower，ch28_pre binding 無 unresolved issues，`story_ch28` 已接回 editable handler。
- 2026-07-20 dialogue pagination slice：對話長句翻頁現在以 10 幀可編輯平滑上捲呈現（舊頁上移、新頁由底部進入，框內 clip），動畫期間 Enter 不會跳過頁面；新增 `dlg_test.go` 狀態 regression。核心 campaign/battle 測試通過；GUI package 實測仍受容器缺 ALSA/X11 headers 限制，待圖形依賴容器重跑。
- 2026-07-20 ch29_post audit：Capstone 直接確認 `0x12cea` 是 X-first/Y-second focus(22,23)；`0x24618` 是 palette_delta=10、step=8 的 9-frame alternating-buffer transition；`0x25089` 是 persistent roster cleanup（清 transient、回填 HP/MP），`0x11df2` 是 dynamic 0x3e→0、delta0 palette loop，`0x17aa9` 是 tick busy-wait，`0x2bce5` 是專用 ending renderer。新增 staged `bindings/ch29_post.json` 與四組精確對白 mapping（FDTXT_029→ch29 scene2 lines6–7；FDTXT_030→ch30 scene0 lines0–14），但因 layout/load-text/focus/transition/ending native ops 尚未全部 lower，暫不接 campaign runtime。
- 2026-07-20 focus lowering slice：`0x12cea(x,y)` 的 direct Capstone 證據已接入 compiler，保留 X-first/Y-second 語意並 lower 成 tile-step pan；新增 regression，ch29_post staged binding 的 focus unknown 已可解析，其餘 transition/roster/ending native ops 仍 fail-closed。
- 2026-07-20 ch29 focus slice：`0x12cea` 已依 direct LE ABI（handler PUSH 順序為 y,x）lower 成 tile-step camera pan(22*24,23*24)，並有 staged handler regression；其餘 ch29 post native cleanup/transition/ending 仍待完成，故 campaign 尚不啟用整段 handler。
- 2026-07-20 persistent roster cleanup slice：`0x25089` 已保留為獨立 editable `reset_persistent_roster_state` beat，不與 `sync_party` 混用；runtime 會清除 transient 行動／位置／buff／中毒／麻痺／封印狀態並以 MaxHP/MaxMP 回填。這是 postbattle 進 town/shop/preparation 前的持久隊伍整備基礎，仍需補 direct handler binding 與 runtime regression。
- 2026-07-20 `0x22253` correction：Docker/Capstone 追到 wrapper 內部實際為 6 次 render+present、每次 10ms，尾端兩次 tick(1)；不可再稱 11-frame 或用 `layout_units` 代替。尚無 DSL 等價 primitive，先 fail-closed，待建立 `native_22253` place/present adapter。
- 2026-07-20 `0x17aa9` timing correction：Docker/Capstone 確認它讀 DOS BIOS tick counter（約 54.9ms/tick），不是 60Hz 單幀；compiler 以每 native tick 3 個 remake display frames（約50ms）保留等待邊界，並加 regression，避免把 ch29 尾端 busy-wait 壓成 16.7ms。
- 2026-07-20 `0x22253` renderer audit：Docker/Capstone 追到 `0x22547` 的實作不是單純 placement：先以 FDOTHER `#0x51` 的 LLLLLL sub1 載入（已實測 9782 bytes、可導出約887ms PCM），再呼叫 `0x22046` 做 indexed off-screen blit；loop 為 6 次 render/present、每次 10ms，尾端兩次 BIOS tick。現有 PNG story renderer 沒有 `0x22046` 的 indexed buffer／resource adapter，故 `unit_present` 仍 fail-closed，禁止降級成 layout 或 generic redraw。
- 2026-07-20 `0x2bce5` ending renderer audit：Docker/Capstone 確認它載入 FDOTHER `#0x36`，建立 320×200 雙 buffer，先 ANI/圖像 compositing，再做 0→63 的 palette fade（每步4ms）、2000ms停留、依 chapter 26/29 分支繪製不同 ending text/figures，最後以 200×4ms fade-out 與 1000ms delay 收尾；不能以 generic `ending` 或普通 fade 取代，需建立 evidence-backed ending_renderer adapter。
- 2026-07-20 external town-flow survey（subagent，非 EXE 硬證據）：中文攻略逐章列出羅德鎮、塞拉村、普里茲港等戰間武器店／道具店／神秘商店／教會與整備；第2章明載保住村民後戰後獎勵力量藥水，第6章明載戰後貝克威加入。這強力支持 battle→postbattle→town/shop/preparation→next battle 的可編輯圖，但攻略無法單獨證明「勝利後自動進城」的程式級轉移；精確順序仍以 `campaign_full.json` 與 direct disassembly 為準。參考：[青衫 FD2 攻略](https://chiuinan.github.io/game/game/intro/ch/c31/fd2/fd2/fd2.htm)、[PTT 攻略轉載](https://www.ptt.cc/man/Old-Games/D9EE/D31B/D56E/M.1099301522.A.DE5.html)、[GitBook 第7章](https://jaceju-favorite-games.gitbooks.io/fd2/content/walkthrough/7.html)。
- 2026-07-20 ch29 cleanup slice：`0x25089` 已 lower 成可編輯 `reset_persistent_roster_state`，runtime 依 direct disassembly 清 persistent roster transient/acted 欄位並將 HP/MP 回填 MaxHP/MaxMP；與 `sync_party` 分離，避免把戰後投影誤當終盤清理。campaign/cmd regression 已補上；GUI package 仍受容器缺 ALSA/X11 headers 限制。
- 2026-07-20 external town-flow survey（subagent，非 EXE 硬證據）：GameFAQs 明載第14章對話「途中有小鎮，先休息」，直接支持 battle→town/rest；第22章明載至第26章前沒有 rest/buy/sell，故 ch23–25 不得強插 town/shop。其餘第2、5、6、7、9–22、26–27章有旅館／教會／武器店／道具店／秘密店旁證；攻略無法單獨證明 handler 觸發時機，精確順序仍以 `campaign_full.json` 與 EXE/資產為準。
- 2026-07-20 ch29 tick slice：`0x17aa9(1)` direct RE 的全域 tick busy-wait 已 lower 成 editable `delay(frames=1)`，保留 ch29 redraw loop 的每次 60Hz 邊界；compiler regression 通過，`0x24618`/`0x2bce5` 仍維持 fail-closed。
- 2026-07-20 ch29 palette-loop slice：`0x11df2(EBX,255,0)` 的 direct loop（EBX=0x3e..0、每次後接 4ms wait）已 materialize 為 63 組 editable `palette_update` + `delay(ms=4)`，不再把 register expression 靜默丟失；其餘 ch29 unresolved 降至 layout/0x24618/load-text/pan/0x2bce5。
- 2026-07-20 0x24618 indexed-transition audit：Capstone 確認 9→1 frame、每幀 descriptor/double-buffer copy、5ms tick，尾端 500ms；之後 32 次 `0x11df2(start=0,end=255,delta=0..62 step2)` + 4ms，這是整張 VGA palette brightness ramp，不是 `(0,255,index)`。新增 `HandlerIndexedTransition` editable metadata 與 explicit binding resolver/compiler test；PNG renderer 仍 fail-closed，尚未接 campaign。
- 2026-07-20 0x24618 schema completion：metadata 另保留 fixed `source_y=0`、`blit_width=0xc0`、clip `0x138×0xc0`，以及 tile/source step；compiler 只接受完整 9-frame/500ms/palette timing，避免把 descriptor copy 簡化成普通 fade。
- 2026-07-20 ch29 post `0x1088d` correction：先前將 `0x25870 → 0x1088d` 縮窄 lower 成 `load_ch_text(ch30.json)` 已撤回，因 Docker/Capstone 證實本體不只載 FDTXT：它依 chapter 載三個 FDFIELD resource、讀 map control、重建 `0x1e00` runtime unit buffer、從 persistent `[0x53bf7]` 複製 own records、套 own-deploy coordinates 並 spawn groups。現已以既有完整 `loadch` state（chapter30/map29/roster70/ch30 story+scenario）重新 lower，compiler regression 明確鎖住不得退回文字-only；handler 仍因 layout、transition、ending 等 unresolved ops fail-closed，尚未接 campaign。internal chapter 29 是最終戰 handler，`0x2bce5` 返回後自迴圈，不會進 generic preparation；現行 final battle→generic ending 暫時略過此 handler。
- 2026-07-20 ch29 post layout audit：Docker/Capstone 證實 `0x257b4 → 0x233c6` 使用三個固定 20-byte arrays（slots 0..19 的 X/Y/pose）與 camera `(16,18)`；數值已可重取，但 remake 尚未證實 20-slot `handlerUnitAt(slot)` 身分等同 native runtime array，且 campaign 未接這個 native post handler。因此不建立猜測性 `layout_units` binding，維持 fail-closed；先需補 roster frontier/identity evidence。
- 2026-07-20 terminal-flow reconciliation：`0x25970 call 0x2bce5` 之後的 `EB FE` 是 `jmp 0x25975` self-loop，證實 `0x25757` 不會返回通用戰後/整備流程。它對應 internal chapter 29（玩家面向的 map29 最終戰）；`preparation_ch30` 仍是最終戰**之前**的既有節點，不能把此 post handler 接到 map28 的 `battle_ch29` 勝利。
- 2026-07-20 map29 final roster provenance：`0x1088d` 證實 `[0x53a45]+slot*0x50` slots 0..19 先由 persistent `[0x53bf7]` ordinal 0..19 複製，再寫 map29 own-deploy ordinal positions；`0x233c6` 只覆寫該同一 buffer 的 x/y/pose。`0x1b750` 是裝備／衍生能力 synthesis，不改 identity。進一步 direct `0x112a5` 證實 JOIN 以 `[0x53bfb]*0x50` append 一筆 persistent record 後遞增 count，故正常遊戲 ordinal 就是首次 JOIN chronology；remake `partyJoinOrder`／`reorderScenarioParty` 的角色順序方向正確。map JSON row order 仍不得替代此 persistent order。
- 2026-07-20 final layout materialization：將 `0x257b4 → 0x233c6` 的三個 native 20-byte arrays 完整寫入 editable `layout_units` binding（slot0..19、camera 16×24/18×24）；compiler regression 鎖定首兩筆、末筆、camera。這只保存已證實資料，不能繞過終局 handler 的 runtime array/renderer gate；其餘 unresolved ops 仍阻止 campaign playback。
- 2026-07-20 final camera pan：`0x25933 push 12; 0x25935 push 11; call 0x135dd` 依 native x-first/y-second ABI lower 為 editable tile-step pan `(264,288)`；compiler regression 鎖定此 final-map camera target。它不影響仍 fail-closed 的 transition/ending path。
- 2026-07-20 final indexed-transition callsite：`0x233c6` 先初始化 viewport origin `(16,18)`、focus `(22,23)` 經 `0x11bfa/0x11b9b` 將 scroll offsets 寫為 `(6,5)`，故 `0x25848` 的 dynamic args `[0x53ab9], [0x53abd]+1` 精確為 `(6,6)`。完整 9-frame descriptor/palette metadata 已以 editable `indexed_transition(tile=6,6; source=10,step8)` binding 保存；runtime adapter 尚未具備 indexed descriptor renderer，仍 fail-closed。
- 2026-07-20 ending asset audit：`0x2bce5` 載入 FDOTHER `#36`；raw asset 可讀（31008B、408×138），但它是供 `0x2935b` sprite/RLE compositing 的圖源，不能拉伸成一張 320×200 ending PNG。ANI `#2` 已有 `internal/afm` decoder（26 frames），缺的是 player-provided FDOTHER runtime loader、`0x2935b` frame/transparent/placement adapter、以及 palette/branch compositor；未補齊前維持 fail-closed。
