# 交接歷史日誌 — 2026-07-06 起

> **文件定位（2026-08-12 稽核）**：本檔累積了多個 session 的當時工作樹、下一步與中間結論，
> 因而是可追溯的歷史日誌，不是目前交接指令、位址主證據或進度真值。不要依本檔的「下一輪」
> 「目前」、working-tree 描述或單一較晚段落執行工作。整體 RE／資料／執行期／E2
> 狀態看 [`58-fd2-exe-re-coverage.md`](58-fd2-exe-re-coverage.md)，系統契約看
> [`56`](56-fd2-remake-sdd.md)，UI 看 [`57`](57-ui-evidence-matrix.md)，有效下一步
> 只看 [`91`](91-worklist.md) 檔首；具體 RE 證據回到 `docs/data/ida/`、
> `docs/data/fd2_*` 與各專題文件。
> 保留本檔僅為提交與判讀 provenance，已被後續文件推翻的斷言須以主證據與上述
> 現況文件為準。搜尋命中本檔不能單獨觸發重新反組譯。

## 2026-07-30 CONTINUE saved runtime roster adapter

新增 `campaign.MaterializeNativeContinueRuntimeUnits`，承接已驗證且已安裝
live field boundary 的 saved `0x50` runtime records。整批先驗證 native
camp、class、active raw presentation 與原順序 first-seen selector slots，
成功後才原子替換 `State.Units`；失敗不改 roster／cache。完整保存 raw
presentation、`+5/+6/+7/+8`、command/transient、inventory、race/class、
`+34..+36`、`+42/+46` 與 stats，且不採信 saved `+2`。

同時修正高風險命名：runtime `+8` 不是全域角色 identity。FDFIELD scripted
constructor 會把 `b1` 寫入 `+7/+8`；只有 camp2 player record 依 persistent
契約提升 `NativeIdentity`，其他 record 使用 `NativeRecordByte8` 保持 raw。
`NativeItemPanelRecordForUnit` 優先使用 raw +8，舊 player-only identity
fallback 僅作相容。使用者 checksum-valid 原版 chapter0 current snapshot
已在唯讀 Docker 測試 materialize 12 筆 records；索爾、悠妮、亞雷斯、
蓋亞與 enemy `+8=96` 分界正確。**此處是 2026-07-30 當時狀態；較晚的
2026-08-02 紀錄已閉合 map timing 與 future-group transaction。** 正式
CONTINUE 後續已新增 chapter0 未改寫排程的 pending roster 適配器；但原版
handler 會改寫 live turn byte，其他章節亦有動態 group formula。因此通用
pending-group binding 與 `Game`／controller handoff 仍未接，保持失敗即關閉。

版本化資料亦同步更正：33份 `map*_units.json` 的 scripted
`native_identity` 全部遷移為 `native_record_byte8`，數值不變；
`sync_native_selector_fields.py --check` 驗證零 pending。scenario party
仍保留真正的 persistent `native_identity`。AI fixture 改讀 raw +8，
避免測試本身把錯誤名稱重新固化。

## 2026-07-30 future group constructor input preservation

官方 IDA Docker 先行嘗試，但私有 home 本輪仍回報 EULA 未接受與
IDAPython target 未啟用，故沒有產生新的 IDA pseudocode／database；
沒有改跑 host IDA。改以 `fd2-cap-local` 完整覆核
`0x10B4E..0x11018`，證實 group row order、6-byte position record、
`b2→+0x3D`、`b13..16→+0x1A..1D`、`b17..19→+0x34..36`、
`b22..24→+0x31..33`，且 b3/b20/b25 在 constructor 無 reader。

33份 map assets 新增 exact position record、raw +3D、death triple、
未讀 source bytes 與 b1-selected constructor record；loader 與
CONTINUE runtime projection 保存對應欄位。直接反組譯 `0x145CD` 證實
兩次呼叫合併標記所有 active runtime unit 所在格為 `0x40`，四鄰
`0x80`；placement 只避開 `0x40`，不是地形通行判斷。
`NativeFutureGroupPlacement` 已轉寫 raw gate、全圖 row-major Manhattan
及同距離後者勝出；`DecodeNativeFutureConstructorBase` 已轉寫兩條 table
分支，33圖1,885筆 race/class/HP/MP 投影驗證通過。table dump 現自帶
FD2.EXE size/MD5/SHA-256，sync 會強制對照 reference
manifest；Docker 重生檔與版本化 JSON 逐位元組相同。`0x1B750` equipment
recompute、per-call gate binding 與 append transaction 尚未接，owner
維持失敗即關閉。證據：
`docs/data/fd2_future_group_constructor_capstone.txt`。

## 2026-07-27 raw byte5 handler bridge

`Scenario.PartyUnits`／`battle.Load` 對 HP>0 的 constructor materialization 保存 `NativeRecordByte5=0`；已知 damage/death
與 church revive writer 也會在 raw provenance 存在時設／清 bit0；
`syncPartyFromBattle`／`applyPersistentStats` 亦會保存 byte5/6，戰後 HP refill 在有 raw provenance 時依 bit0 branch；
本輪再補 FDFIELD b0→runtime `+6` 的 parser/exporter schema 與 LOAD/Party materialization（只保存 raw source，不命名 camp/phase）；
`cmd/fd2` 的 `any_unit_inactive` 在有 raw provenance 時優先讀 `+5&1`，`deactivate_unit` 同步保留
`0x32975` 的整 byte overwrite=1。舊 authored JSON 缺 raw 時仍保留 `OnField/Alive` 相容 projection，
因此這是 E1 partial、不是 native parity；下一輪需補 death writer／LOADCH raw propagation，並在缺 raw 的
strict binding fail-closed。

## 2026-07-25 resume note — SDD first

依使用者要求本輪先建立 `56-fd2-remake-sdd.md`，暫停新增 handler／renderer。盤點確認目前已有 Ebiten battle/story/cutscene、shop、church、preparation、save 與部分 native ending primitive，但原版 menu dispatch、完整 postbattle town/rest flow、weapon reach、item UI、native indexed presentation 仍未達 remake。`42`/`51` 只作 baseline。

working tree 的 `native_2c548.json`、`internal/ending`、`internal/figani` figure-fade 變更是上一輪未提交工作，已保留未覆蓋；待 SDD gate 後另行 Docker regression。`/tmp/fd2cap` 不存在。**本句關於 `~/.codex/knowledge-base` 無可讀檔案是當時接手環境的歷史觀察，已由 2026-08-09 勘誤取代；不可作為現況斷言。**

下一項：UI evidence matrix 已建立於 `57`，逐章 battle→postbattle→town/shop/church/preparation/ending graph
已收進 SDD `56 §5.1`；後續要以原版 handler/DOSBox 補 E0/E2，而非把 existing Ebiten shell 或 legacy
`CastArea` 叫做 remake complete。保持 fail-closed。

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

> **2026-07-16 第二十次 Codex 更新（商店資料與原版收件規則；數量斷言已修正）**：`shops.json`、demo
> 與 full campaign 目前保存的是 215 個 EXE 原版 unsigned-byte `id`（0..214），逐筆以 `item.json` 的價格交叉驗證；
> 舊版「337筆商品」說法與現行 fixture 不一致，撤回；runtime `0x602ad` table 邊界仍未證實。
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
> 改接 authored ch02_pre。後續以 constructor/death/revive 完整 body 補足各 writer，
> 但 caller-specific bit0 高階語意仍由後續 audit 收斂，不能把該輪筆記當成全域定案。

> **2026-07-16 第十二次 Codex 更新（bit0 writer 交叉檢查 + ch03 條件）**：後續 caller audit
> 撤回把 `unit+5 bit0` 全域命名為 active/alive/dead；目前只保留 `0x3453e` 的 raw `&1` predicate，
> 與 constructor、HP、revive、`0x32975` 等獨立 writer。exporter/runtime 的 `unit_inactive` 是
> caller-specific projection，不能當成 native 欄位全域語意。ch03 turn3 的 branch 仍以其直接
> CFG predicate 維持；`0x11506` 的 raw HP/roster 行為另由對應測試驗證。`ch02_post` 真 CFG 已釘死為
> `sync → raw predicate #6:(layout+#7+JOIN2) → chapter3`；
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
- 2026-07-20 item range RE（2026-07-27 勘誤）：撤回把 `0x14237` 的 item `+0x0c` 命名為通用 `range_min`、把 `+0x0b/+0x0d` 命名成 `atk_rate/range_max` 的說法。官方 IDA/Capstone 目前只閉合 caller-specific raw geometry：`0x14237` 將 `+0x0b/+0x0c` 以 `a5/a4` 傳入 `0x14818`，`mode<0x10` 排除 inner marker、`mode>=0x10` 走 cross；不能與 normalized weapon range 一一對位。remake 繼續使用獨立驗證的 `weapon_range.json`，完整 item multiplier/effect 仍待 direct producer。
- 2026-07-20 preparation slice（歷史記錄，已由 2026-07-29 外層呼叫端勘誤）：當時只讀 `0x318ad`，尚未發現小名冊會由 `0x2d093` 完全略過選人，且「達 cap 後自動離開」漏了最終確認。30-byte 勾選表與 cap15/19 原始事實保留，其餘流程以文末 2026-07-29 結論為準。
- 2026-07-20 church RE：`0x3072f` 讀教會服務選擇並分派 `0..3` 到 `0x2ffa5/0x2f8ea/0x30dc3/0x31385`；`0x30dc3` 的 `0x24c` 是「無須復活」訊息，存在死亡候選才用 `0x24d` 選人，確認費用後 `0x2d516` 扣款、清 `[unit+5]` 死亡 flag、把 `[unit+0x42]` 複製到 `[unit+0x40]` 恢復 HP。`0x31385` 的 `0x24f/0x250/0x252` 分別是無候選／選人／確認轉職。教會沒有免費一般治療分支；下一步先資料化 revive/class-change service nodes。
- 2026-07-20 revive core slice：直接讀出 `0x52669 + class*2` 的 29 筆 u16 class fee table，新增 `docs/data/exe_tables/revive_fee_rates.json` 與 editable runtime copy `remake/assets/data/revive_fee_rates.json`。`campaign.ReviveUnit` 依已證實公式 `feeRate * level` 先驗金額，再原子寫回 HP=MaxHP、清除死亡投影 `OnField`；不足金或非死亡候選不改狀態。尚未把 church selector 接到 UI，也尚未追完 class-change 寫回能力表。
- 2026-07-20 class-change candidate slice：`campaign.CanChangeClass`／`ClassChangeCandidates` 已接上 `0x31793` 的 exact filter：Lv>=20、portrait<0x12 且 portrait!=7，保留 JOIN order；尚未實作 `0x31860` 道具分支與 `0x2a2e8` class/portrait/能力寫回。
- 2026-07-20 church selector slice：`main.go` church 節點已從直接返回 town 改成四項服務選單；第3項接 `campaign.ReviveUnit` 與 EXE fee table，第4項顯示 exact class-change candidates但保留 item/能力寫回待接。xvfb 實機截圖已存 `docs/figures/church-selector.png`。
- 2026-07-20 class-change RE continuation：`0x3151a..0x3152d` 依 portrait 查轉職道具（portrait 0x34→item 0x5a，其餘 promoted portrait→`0x526a7+portrait` byte），`0x31860` 掃 8 個 inventory slot；成功後 `0x1b8e7` 移除 item、`0x2a2e8` 重算、`0x31571..0x3157a` 寫 class(+0x20)/portrait(+7)。目前只接 candidate/UI，mapping table 尚待完整導出。
- 2026-07-20 class target tables（2026-07-28 語意勘誤）：已導出 `0x615fe` portrait→(class,mobility increment) pairs 與 `0x526a7` raw item bytes 至 `docs/data/exe_tables/class_change_targets.json`；portrait 9 持 item 0x5a 時的 special override→target 0x34 已明列。這些是單一 target resolution inputs，不是同畫面可選分支。
- 2026-07-20 class target table correction（2026-07-28 補充）：原表把 `0x526a7` 誤標成 target portrait index；依 `0x31793` 實際指令，現在拆成 `current_portraits`（current portrait 0..0x11，default=current+0x20、optional=current+0x32，raw item `[0x526a7+current]`）與 `target_portraits`（`0x615fe` 的 class/mobility increment pairs）。raw `0xff` 不啟用 optional override；current portrait 9 的 item 0x5a 最後覆寫為 target 0x34。新增 `class_change_table_test.go` 驗證 18/34 rows 與 index 對齊。
- 2026-07-20 `0x31602` stat-reset 定案（更正）：`0x4e4d1(portrait)=0x620a1+portrait*0x0b` 的 11-byte 成長列，五組 row pairs 經 `0x1e529` **加到既有** unit words `+0x37(AP),+0x39(DP),+0x3e(DX/HIT-EV base),+0x42(MaxHP),+0x46(MaxMP)`；`0x1e529` 尾端是 `add word [target], ax`，不是覆寫。`+0x40/+0x44` 由後段回填 current HP/MP；`0x4e48d(new portrait)+1` 的 mobility increment 加到 raw `+0x3b`。流程清 raw EXP `+0x3c`，**未寫 level byte，故保留原 Lv**，HP/MP 全滿。row random 是 pair 的 `[min,max)` 取值。
- 2026-07-20 class mutation core slice：`campaign.ApplyClassChange` 依 `0x31602` 寫回 target portrait/class、AP/DP/DX/MaxHP/MaxMP、MV(+0x3b)、Lv=1/Exp=0/HP=MaxHP/MP=MaxMP，並移除 branch item；invalid range/item 失敗不改動 unit。新增成功與 atomic rollback tests；尚未把 UI/JSON growth rows 接上，也尚未呼叫 equipment recompute（避免猜測舊 Base*）。
- 2026-07-20 class-change editable bridge（2026-07-28 已取代）：`LoadClassChangeTable` 與 `LoadClassChangeGrowth` 保留；錯誤的 `ClassChangeTargets` 三分支 API 已由 `NativeClassChangeTarget` 取代。`0x31793` 先給 default，再由 optional item 覆寫，最後由 portrait9/item0x5a special 覆寫，每位候選只留下唯一 target。
- 2026-07-28 official IDA class-change correction：合法 IDA 9.4 重讀 `0x31385/0x31793/0x311DC/0x31019/0x19953`。流程是候選建立→三列可見角色清單（↑/↓ bounded，捲動）→唯一 target→左右 Yes/No 確認；不是 target branch 選單。`0x31019` 每列畫 portrait、角色名、目前 class、FDTXT593 與 target class，選中／未選 palette index 為 201/205。remake 已改為此 state/input contract；精確 indexed assets/compositor 仍 fail-closed。
- 2026-07-20 `0x1b750` synthesis continuation（校正）：`0x1b750` 讀 raw `+0x37/+0x39/+0x3e` 與 item table 23-byte row 的 `+1/+3/+5/+7`，寫 derived `+0x48/+0x4a/+0x4c/+0x4e`；它是 class path 後的 equipment/stat synthesis，不是 screen-only projection。`+0x22/+0x23/+0x24` 雖會影響該 routine 的 transient branches，但 constructor 先清零且 `0x31602` 不寫它們，不能當成 class growth source。`campaign.RecomputeAfterClassChange` 與 double-count regression test 已保留。
- 2026-07-20 xvfb fixture hook（2026-07-28 修正）：`FD2_CAMP_CLASS_FIXTURE=1` 僅供 headless oracle，注入一名 Lv20 portrait9 並帶 item 0x58/0x5a；正確預期是自動解析到 special target 0x34，再顯示 Yes/No 確認。此 hook 不在正常啟動路徑。
- 2026-07-28 screenshot retraction：舊 `church-class-targets.png` 只證明 remake 曾畫出自創三分支選單，與原版不符，已從 README、worklist 與 repo 刪除，不得再作 native visual proof。
- 2026-07-28 native class-list renderer：Docker Capstone 補足 `0x31385→0x311dc` 真實三參數 ABI，並固定 `0x311dc` 的 FDOTHER#14 LMI1 entry16（實檔310×86）、`0x31019` row 座標（FDICON x14/y117+26i；name x40、current class x130、FDTXT593 x175、target class x239，文字 y121+26i，selected/unselected fg201/205、shadow76）。`0x1974c` opening 是六次 y=`177,164,151,138,125,112` 的310-wide crop。新增 atomic indexed compositor、原版資源 regression、runtime bridge與 [`native-class-list-indexed.png`](../figures/native-class-list-indexed.png)；六幀 runtime lifecycle 只在每個 Draw acknowledgement 後前進。
- 2026-07-28 native class confirmation renderer：Docker Capstone/IDA 固定 `0x19953` 使用 FDOTHER#2 raw cells16/17，四幀 spread4/8/12/16；互動態以 `3*16=48`、`3*17=51` 為 normal，選中 pulse再+1，screen落點 x=`248±16`, y168。FDTXT594 的 leading `FFFC` 由 caller 寫入的 current portrait+1 name string取代，不是可忽略控制碼。`0x197e5` 四幀 choice closing spread12/8/4/0，最後 #17依 native順序覆蓋重疊的#16。後續 official IDA caller 重核修正早先不完整斷言：候選 list 必須先 `0x2d31b` 五幀 closing＋source restore才進 confirmation；confirmation 的 choice closing 後還有 dialogue closing 五幀＋source restore，才 mutation／返回。runtime 與原版資源 regression 已補齊完整 Draw-ack lifecycle，並重生含教會 scene/dialogue box 的 [`native-class-list-indexed.png`](../figures/native-class-list-indexed.png)／[`native-class-confirm-indexed.png`](../figures/native-class-confirm-indexed.png)。BIOS low-word delta>=2 時 pulse counter=(counter+1) mod4，選中 cell variant=counter/2。
- 2026-07-28 church raw service0 closure：instruction decode 固定 `0x17aed` 所有 `0x17e0b/0x1c269/0x1ceed` call 均重用單一 stack actor，Hex-Rays 第二參數是 artifact；流程是唯讀 item/status→key wait→有 command 時切 command/MP→key wait→關閉，無 persistent writer。remake 已接 caller `0x2e6b8/0x2ea90` 兩欄六人 roster（FDOTHER#14 entry16、FDICON/name exact coordinates、201/205＋shadow76、stateful bounded ±1/±2、6-open/5-close+restore）及完整 `0x17aed` presentation lifecycle：status 12-open、bottom 7-close/7-open、`0x1ceed` FDTXT441+ID/cell92/MP digit-base42 overlay、12-close+restore後重開名冊。`NativeItemPanelRecordForUnit` 亦補存 `+0x1a..+0x1e` command mask。新增 [`native-status-command-indexed.png`](../figures/native-status-command-indexed.png)；不把唯讀服務誤接為 gameplay mutation。
- 2026-07-28 church raw service1 correction/renderer：FDTXT510「要給誰呢？」、511「沒東西了！」、512「誰的東西呢？」與`0x2f8ea→0x1b8e7→0x1bb8c`閉合此服務為物品轉交。刪除舊「`0x2f8ea`經`0x2f4c6/0x2d516` amount path」混線斷言：完整 caller無這兩個call且不改gold。`0x2dc55(mode1)` exact layout已接：FDOTHER#14 entry16/cell15、兩欄六項、item FDTXT181+ID、cells59..67/frame41、五位數`3/4×word[row+19]`、stateful viewport、6-open/5-close+restore；來源／目的 roster亦使用原版 compositor。成功與取消回來源角色 loop，不再錯回主選單。續核 destination-full branch：目的 roster 關閉後若`0x1b8a6(dest)==8`，caller寫`dword_53ad9=unit[8]+1`，hub selector4由`word_5265f`取FDTXT506，6-open後在`(12,119)`展開leading FFFC姓名、`0x16c57(1)`等待、5-close，再回來源。runtime已依原始八格flags與native identity fail-closed接入，並新增[`native-transfer-full-indexed.png`](../figures/native-transfer-full-indexed.png)。
- 2026-07-28 church revive renderer/semantic correction：official IDA `0x30dc3/0x309ff/0x30c22/0x30a47/0x3453e` 固定候選為 roster raw byte `+5 bit0`，不是 `HP<=0` projection；fee為`word_52669[raw class +0x20] × raw level +0x21`，Lv0不得自行夾成1。三列 stateful selector exact畫出 `(14,117+26i)` sprite、name/race/class、currency cell15與五位fee；FDTXT590 的 FFFC/FFFA/FFFE已接動態名字、十進位fee與19px軟換行。第二次 caller 重核撤回本條早先「確認固定choice4+dialogue5關閉」斷言：`0x197e5`只關YES/NO四幀；不足金接著在仍開著的question第三行 `(12,157)` 寫FDTXT504並`0x16c57(1)`等待，才`0x2d31b`五幀關框；無候選走FDTXT588與`0x16c57(0)`。runtime與原版資源 regression 已接兩條 message lifecycle，新增[`native-revive-empty-indexed.png`](../figures/native-revive-empty-indexed.png)／[`native-revive-insufficient-indexed.png`](../figures/native-revive-insufficient-indexed.png)。成功演出由下一條續核。
- 2026-07-28 revive success `0x2f4c6` closure：church從hub selector4進入，故成功時固定走case4。FDOTHER#14 entries23..31依序透明疊到VGA `(147,32)`且每幀`0x17aa9(2)`；其後baseline DAC delta 0→62與62→0各步2／4ms。`0x17aa9(10/5)`沿用上次動畫／wait latch，實際剩餘等待須扣除前一段32×4ms，不能錯加十／五個完整ticks。最後`sub_16559(0)`於`(118,4)`恢復DATO mode0。remake已接原始九幀、dynamic palette與monotonic timeline，final restore經Draw後才返回名單；新增[`native-revive-success-indexed.png`](../figures/native-revive-success-indexed.png)／[`native-revive-success-flash-indexed.png`](../figures/native-revive-success-flash-indexed.png)。後續IDA+Capstone撤回本條最初「`sub_25977(17/11)`是PCM SFX」斷言：`0x25977(track,loop_count)`直接載FDMUS.DAT並呼叫`AIL_set_sequence_loop_count`；成功動畫前為track17/count1，完成後track11/count1。runtime新增one-shot BGM path，未碰FDOTHER#31 UI SFX。
- 2026-07-28 church main-menu renderer correction：合法 IDA 9.4重讀`0x2d669/0x2d85f/0x2d9fe`並以新`tools/ida_dump_bytes.py`核對data。FDOTHER#14 entries3/5/7/9是四個normal cell，pairs3/4、5/6、7/8、9/10皆為24×20；base `(240,169)`，`0x526da/0x526ea`均為`[-39,-13,13,39]`。四pass opening divisors4/3/2/1、closing1/2/3/4，每pass由palette74的104×20 cleared source重建。撤回舊「opening尾端restore」斷言：只有closing第四幀後restore source。steady selected cell由兩BIOS tick gate、counter mod4、variant=counter/2驅動。新增campaign indexed primitive與原版資源 regression；`0x3072f` entry1／DATO／FDTXT585/586完整背景尚未合成，故未猜測性接runtime。
- 2026-07-28 church main-menu runtime closure：官方IDA續讀`0x3072f/0x1956b/0x16559/0x187d6/0x168b6`，固定stable scene為FDOTHER#5 raw cells1..17 dialogue grid、#5 four-mode digit frames31..40、FDOTHER#14 entry1、DATO resource131 frame0（不是商店店員DATO75）、FDTXT585初始／586服務返回；currency以八位zero-pad畫在`(16,99)`。新增mixed-codec compositor與original-resource regression，runtime以獨立draw-ack lifecycle排程四幀open、steady BIOS pulse、四幀close及cleared-source restore，服務dispatch/Escape均等close完成。deterministic oracle保存[`native-church-menu-indexed.png`](../figures/native-church-menu-indexed.png)；raw service0語意仍不猜。
- 2026-07-28 class fixture identity correction：`FD2_CAMP_CLASS_FIXTURE` 舊值把 portrait9 標成索爾；依原始 ch01 unit/故事 identity9=悠妮，已改 party key/native identity/map key/fig 全為9並顯示悠妮。vertical trace 同步用 roster key9，避免錯名 oracle 再進 README。
- 2026-07-20 battle progression slice：campaign `Node.Protect` 已資料化，`checkResult` 依 battle node 的 protect 欄位判定敗北，空值維持索爾相容預設；新增 campaign test。另修正升級：原版 DX 是 HIT/EV 共用 raw base，`GainExp` 在已有 equipment base 時同步更新 BaseHIT/BaseEV 與有效 HIT/EV，保留裝備加成並新增 regression test。
- 2026-07-20 AI low-damage slice（歷史實作，已撤回）：早期曾依未核實的 `0x15140` 假說加入 `dmg≤2` 篩選；2026-07-27 canonical Docker recheck 已確認該地址不是可證實的 AI entry，因此這條規則與對應測試不再代表原版語意。normalized approximation 可保留作現況行為，但不得當作 native parity；後續以 `0x13A9F/0x14EF0/0x149F8/0x15B77` raw evidence 重建。
- 2026-07-20 AI spell-entry audit（後續補證）：臨時 capstone 容器 direct disasm `0x15470..0x15618`，並查到呼叫點 `0x13E39`、`0x14F9B`。`0x1548E` 才是函式入口；`0x154D1` 位於其本體，實際流程可見 `0x14B78` 呼叫與 `0x12D7B` 演出狀態呼叫，沒有 `Cast` dispatch 證據。當時尚未閉合 `0x14B78`，不可僅以這句宣稱完整 path ABI；2026-07-29 才由 `0x14B78→0x4E1A6→0x13488` 直接證據補完。已撤回「0x154D1 是施法入口」舊註記。
- 2026-07-20 AI spell dispatch proof（2026-07-29 再勘誤）：原版確會由
  item-command 導向施法演出，但此段曾錯把 `0x1567E` 候選寫成送入
  `0x15B77`。直接指令證實兩支候選都送 `0x15880`；`0x15B77` 屬另一條
  `0x1598A` command-mask producer。`[0x53C3F]` 也不是 command，而是
  inventory slot；`0x1507C→0x1B722` 先由槽解 item，`0x150C2` 才讀
  row `+0x10` command。舊句不得再作現況依據。
- 2026-07-20 AI spell data bridge：remake `battle.State` 新增可注入 `SpellBook`，`AIPlan.SpellID` 以 `-1` 明確表示目前物理／待命計畫不施法；`loadGame` 將已載入的 EXE spell table 複製進 state，並新增 regression test 防止物理 AI 偷生 spell command。刻意未加入猜測性的 spell ranking、治療目標或施法座標；這些要等 command inventory 對映與 `0x15880/0x15B77` 語意定案。
- 2026-07-20 AI raw-score topology（後續勘誤）：direct disasm `0x15B77..0x15DA1` 證實 command IDs 走不同 raw scoring branches：0..12 讀候選 raw HP/`+0x08`，13..16 讀 `+0x40/+0x42/+0x34`，17..19 進另一 helper，20/21/26/27 讀 `+0x25/+0x26`，22 gate `+0x27` 後呼 `0x1C269`。早期「增益／狀態／毒麻」是未證實的 gameplay 命名，已撤回；現行欄位結論仍以 constructor trace 為準：magic raw=`unit+0x1a..+0x1d`，`+0x22..+0x24` 是 raw transient/modifier bytes。
- 2026-07-20 class-change fidelity correction：使用者實測指出轉職結果與原版差距巨大；direct disasm `0x1E529` 尾端確認是 `add word [target], ax`，PTT 實測表亦吻合「舊能力 + 新職 growth row」而非絕對重設。已修正 `ApplyClassChange`：AP/DP/DX/MaxHP/MaxMP 改為累加、Lv 保留、EXP 清零、HP/MP 回滿；campaign/battle 測試通過。外部旁證：[PTT 實測表](https://www.ptt.cc/bbs/Dynasty/M.1185344950.A.91B.html)、[FD2 轉職攻略](https://jaceju-favorite-games.gitbooks.io/fd2/content/walkthrough/INDEX.html)。
- 2026-07-20 外部流程盤點：攻略頁逐章列出武器店／道具店／教會／神秘店，至少第4、7、9、14、16、18、19、21章有明確整備設施與隱藏商店證據（[第4章](https://jaceju-favorite-games.gitbooks.io/fd2/content/walkthrough/4.html)、[第7章](https://jaceju-favorite-games.gitbooks.io/fd2/content/walkthrough/7.html)、[第16章](https://jaceju-favorite-games.gitbooks.io/fd2/content/walkthrough/16.html)）。頁面未明文保證「勝利後立即進入」，故只作 campaign town/shop 節點的外部交叉證據，不取代 EXE branch/table；`campaign_full.json` 仍須保留 postbattle→town/preparation→next battle 的可編輯順序。
- 2026-07-20 class-change equipment correction：發現 `ApplyClassChange` 先改有效 AP/DP/MV、再呼叫 `RecomputeAfterClassChange` 時，已有裝備會被重算兩次。現以既有 equipped item 貢獻反推 raw base，再套用已確認的 `0x1b750` stat/equipment synthesis；新增回歸測試證明 AP/DP 不會由 18/12 錯變 21/14，campaign/battle 測試通過。
- 2026-07-20 handoff reconciliation：本檔較早的 church/class-change 條目是歷史快照；其中「Lv=1」、「尚未接 UI／能力寫回」、「church 仍是 placeholder」等描述已由後續 RE 與實作更正。現行權威狀態是：保留 Lv、清 EXP、五組成長累加、target/item/UI/persistent roster 已接；仍待的是 `+0x22/+0x23/+0x24` transient writer 的完整來源、原版實機數值回歸與完整 GUI 轉職操作截圖。
- 2026-07-20 class raw-field audit：`0x1b750` 的 AP/DP/HIT/EV synthesis 會讀 `+0x22/+0x23/+0x24` 的非零旗標分支，但 spawn constructor `0x10f6b..0x10fa5` 先把 `+0x22..+0x27` 清零，且 `0x31602` class path 不寫這些 bytes；它們是後續 transient/effect writer 的欄位，不是 M1–M5 spell bitfield，也不是可直接從 class growth row 匯出的 modifier。remake 暫不猜測接線。
- 2026-07-20 raw-unit pointer/schema 對齊：spawn constructor `0x10f6b..0x10fa5` 直接證實 FDFIELD b13..b16 的 `initial_command_mask` 複製到 runtime `unit+0x1a..+0x1d`；它是 runtime 五位元組 command bitset 的初始四位元組，不是 spell table。runtime `+0x22..+0x27` 另以 memset 清零，後續才由能力流程寫入 raw transient/modifier bytes。因此 `+0x22/+0x23/+0x24` 不是 spells，也不自動等於 named stats/status；individual command ID 的玩法語意仍待對照。已同步修正 `03-exe-and-data-structures.md` 與 `11-enemy-ai.md`，避免錯誤 schema 繼續污染 remake。
- 2026-07-20 AI command inventory slice：item EXE row 是 23 bytes，K4 command 位於 raw byte `0x11`（item 79 的 `0x1f`→spell 15）；新增 `campaign.LoadAICommandSpellMap` 與 `State.AICommandSpell`，只資料化 command `>=0x10`，不猜測 AI ranking／治療目標。campaign/battle 核心測試通過。
- 2026-07-20 AI available-spell slice：新增 `State.AIAvailableSpells(unit)`，依 unit inventory 順序把 command map 解析出的 spell IDs 對到 EXE `SpellBook`，去重且忽略未知 spell；此層只重現 command inventory，不改 `NextAIPlan` 的目標評分或施法執行。
- 2026-07-20 normalized candidate slice（非 native 命名）：`State.AISpellCandidates` 只是 normalized live/camp candidate approximation，保留 runtime order；它依既有 normalized spell IDs 選候選，不代表 native ID17..27 的 status/family 名稱或分數，後續 raw adapters 仍未接 AI runtime。
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
- 2026-07-20 ch22_pre control-flow slice：`0x336b5` 的 `EBX` 不是 roster_has，而是 `repeat_hint(limit=16, loop_back=0x336b4)` 的固定清理迴圈；compiler 現在把 `unit_slot_expr:"ebx"` 明確展開成 slots 0..15，並以 active loadch slot_count 驗證。這解開 ch22 的動態索引阻塞；`0x24618` 視覺效果與 `0x11df2` palette/fade 仍保留 unknown，故 story_ch23 尚未猜接。此為當日狀態，兩者已由 2026-07-26 的 indexed transition 與 palette fade 直接證據勘誤取代。
- 2026-07-20 palette/transition RE correction：`0x11df2(start,end,delta)` 對 `[0x53a65]` 每色 RGB 加 delta、clamp 0..0x3f 後逐項寫 VGA DAC，是一次性 `palette_update`；`0x11d40` 才是減亮 fade-out。compiler 已將 native `0x11df2` immediate calls lower 成 `palette_update`（ch22 呼叫皆 delta=0，runtime 保留順序，不誤當黑幕 fade）。`0x24618` 定案為 `(x,y,palette_delta,step)` 固定 9-frame transition/reveal（每幀 present、5ms、尾端 delay500ms），仍待 indexed transition renderer，未猜接。
- 2026-07-20 ch23_pre slice：handler `0x338ce` 的 FDTXT_024 index0/index1 共 14 句（scene0，5+9），map23/70-slot、四段 pan(0,4→0,22→26,24→26,2)、spawn group1 與 ch24 script 已新增 binding；`story_ch24` 接回 editable pre-handler，compiler/campaign/battle regression 通過。
- 2026-07-20 ch24 transition/audio slice：`0x24b4d(count)` 完整 RE 確認為先 terrain/main redraw，再以兩個 `0x1c8` buffer 交替 present、每幀 20ms；ch24 calls 為 20/20/20/60（400ms/1.2s）。compiler/runtime 已新增 `transition_reveal`，binding 將 `load_res` FDOTHER#88、四次 `play_sfx(priority=1,index=1)` 接到 `battle_88_01.wav`，並把 `0x1d50a` 的 index=-1 stop 與 `0x1a80a` release 接回 handle；`story_ch25` 已切回 editable handler。剩餘差異只在 indexed double-buffer visual adapter（目前保留 exact count/timing，PNG renderer 每幀重繪）。
- 2026-07-20 ch25_pre slice：FDTXT_026 全章因後續分支／raw utterance 與 authored line 數不一致，未宣稱全量 count-aligned；但 handler 實際呼叫的 string0 可直接對到 `ch26.json` scene0 12 lines。已新增 ch25_pre binding（map25/70-slot、pan 9,39、acting76、dialog line0/count12/scene0）並將 `story_ch26` 接回 editable handler；後續 FDTXT_026 分支仍待條件控制流 mapping。
- 2026-07-20 ch26_pre slice：逐字解析 FDTXT_027 證實 handler 的 idx0/3/4/5/6/7 分別對到 `ch27.json` scene0 的 lines 0–3、4、5–6、7、8–12、13–21；新增 `bindings/ch26_pre.json` 與六組 direct line/count overrides，`story_ch27` 接回 editable pre-handler。`0x24b14(100)` 依既有 LE disasm 是天空之鑰 item `0x64` 的 16-slot inventory gate，仍保留 unresolved branch/effect，不把 gate 猜成自動跳轉。
- 2026-07-20 ch27_post slice：`FDTXT_028` 已 count-aligned，handler idx7 (`0x231e5`) 精確對到 `ch28.json` scene1 lines 11–15；新增 `bindings/ch27_post.json`，掛在天空之鑰 present branch 後、進 preparation_ch28 前。sync_party/set_chapter 原語保留原順序。
- 2026-07-20 ch28_pre audit→resolved：FDTXT_029 idx7/idx8 分別精確對到 `ch29.json` scene1 lines 5–12（8句）與 scene2 lines 0–5（6句），pan(9,56)→(216,1344)、acting86 已建 binding。2026-08-02 重新以 IDA／Capstone 直接指令勘誤：`0x35822` 是 pan→spawn→delay300→`palette(0,255,255)` 全白→delay200→`palette(0,255,0)` 基準恢復（baseline restore）→redraw；舊記的兩次 delta0／無作用（no-op）已撤回。來源 `PUSH` 順序是 `(group,y,x)`，不是更早的 `(x,y,group)`。compiler 已 lower 且 `story_ch28` 已接回 handler，無 unresolved issues。
- 2026-07-20 ch26_post gate audit：`0x25186` 後 `cmp eax,-1 / JE 0x25348` 證實 item `0x64` 缺失會進 FDTXT_027 idx13–16 離別支線，命中才繼續 idx9–12 正線並 `sync_party/set_chapter(27)`；campaign gate 現已承載缺匙對話 scene→ending，ch26_post 的大量 visual/effect unknown 仍待拆解。
- 2026-07-20 missing-key branch slice：新增 `ch27.json` 可編輯場景「缺少天空之鑰的離別(分支)」，收錄 FDTXT_027 idx13–16 的 17 句離別對話；`inventory_gate_ch27_sky_key.if_missing` 現在先進該 scene，再接 `ending_ch27_no_sky_key`，不再用 generic ending 吞掉原版對白。`0x25052/0x24618/0x1c2da` 視覺／系統效果與 `0x22253` 的 runtime adapter 仍刻意保留為待辦。
- 2026-07-20 isolated RE toolchain：新增 `tools/docker/fd2-cap.Dockerfile`，建立本機 `fd2-cap-local` image（Python 3.12 + capstone 5.0.3）；所有後續 `disasm_le.py` 以 repo read-only mount 執行，不污染 host Python。實際 Capstone disasm 確認 `0x35822` 的演出序列；2026-08-02 由 IDA／Capstone 直接指令更正來源 `PUSH` 順序為 `(group,y,x)`。compiler 已 lower，ch28_pre binding 無 unresolved issues，`story_ch28` 已接回 editable handler。
- 2026-07-20 dialogue pagination slice：對話長句翻頁現在以 10 幀可編輯平滑上捲呈現（舊頁上移、新頁由底部進入，框內 clip），動畫期間 Enter 不會跳過頁面；新增 `dlg_test.go` 狀態 regression。核心 campaign/battle 測試通過；GUI package 實測仍受容器缺 ALSA/X11 headers 限制，待圖形依賴容器重跑。
- 2026-07-20 ch29_post audit：Capstone 直接確認 `0x12cea` 是 X-first/Y-second focus(22,23)；`0x24618` 是 palette_delta=10、step=8 的 9-frame alternating-buffer transition；`0x25089` 是 persistent roster cleanup（清 transient、回填 HP/MP），`0x11df2` 是 dynamic 0x3e→0、delta0 palette loop，`0x17aa9` 是 tick busy-wait，`0x2bce5` 是專用 ending renderer。新增 staged `bindings/ch29_post.json` 與四組精確對白 mapping（FDTXT_029→ch29 scene2 lines6–7；FDTXT_030→ch30 scene0 lines0–14），但因 layout/load-text/focus/transition/ending native ops 尚未全部 lower，暫不接 campaign runtime。
- 2026-07-20 focus lowering slice：`0x12cea(x,y)` 的 direct Capstone 證據已接入 compiler，保留 X-first/Y-second 語意並 lower 成 tile-step pan；新增 regression，ch29_post staged binding 的 focus unknown 已可解析，其餘 transition/roster/ending native ops 仍 fail-closed。
- 2026-07-20 ch29 focus slice：`0x12cea` 已依 direct LE ABI（handler PUSH 順序為 y,x）lower 成 tile-step camera pan(22*24,23*24)，並有 staged handler regression；其餘 ch29 post native cleanup/transition/ending 仍待完成，故 campaign 尚不啟用整段 handler。
- 2026-07-20 persistent roster cleanup slice：`0x25089` 已保留為獨立 editable `reset_persistent_roster_state` beat，不與 `sync_party` 混用；runtime 會清除 transient 行動／位置／buff／中毒／麻痺／封印狀態並以 MaxHP/MaxMP 回填。這是 postbattle 進 town/shop/preparation 前的持久隊伍整備基礎，仍需補 direct handler binding 與 runtime regression。
- 2026-07-20 `0x22253` correction：早期「6 次 render+present、每次10ms、尾端兩 ticks」只涵蓋 `0x22547` 中段，已由完整 caller trace 撤回。完整 wrapper 另有前段 `0x22470` 11 次 LMI present/tick 和後段 `0x22656` 10 次 remap present/tick，總計 27 次 present；不可再用 `layout_units` 或 6-frame metadata 代替，維持 fail-closed。
- 2026-07-20 `0x17aa9` timing correction：Docker/Capstone 確認它讀 DOS BIOS tick counter（約 54.9ms/tick），不是 60Hz 單幀；compiler 以每 native tick 3 個 remake display frames（約50ms）保留等待邊界，並加 regression，避免把 ch29 尾端 busy-wait 壓成 16.7ms。
- 2026-07-20 `0x22253` renderer audit correction：Docker/Capstone 確認 immediate `0x51` 是 FDOTHER **十進位 #81** 的 nested `LLLLLL` entry（outer 18710 bytes、directory first-word `0x12`；nested payload #1=9782 bytes），不是 #51 音訊或 PCM 資源。**後續 2026-07-26 已更正**：stack-slot trace 證實此 allocation 未傳 `0x22470/0x22547/0x22656`、只在尾端 free，因此不得再叫它 renderer source；`0x11eee` 不負責 nested selection。`0x22547` 使用 boot 載入的 FDOTHER#3 `0x53a6d` descriptor table，另有 FDOTHER#6-derived local pointer，做 indexed `0x22046` blit/present，loop 為 6 次、每次 10ms，尾端兩次 BIOS tick。現有 PNG story renderer 沒有 indexed buffer／resource adapter，故 `unit_present` 仍 fail-closed，禁止降級成 layout 或 generic redraw。
- 2026-07-20 `0x2bce5` ending renderer audit：Docker/Capstone 確認它載入 FDOTHER `#0x36`，建立 320×200 雙 buffer，先 ANI/圖像 compositing，再做 0→63 的 palette fade（每步4ms）、2000ms停留、依 chapter 26/29 分支繪製不同 ending text/figures，最後以 200×4ms fade-out 與 1000ms delay 收尾；不能以 generic `ending` 或普通 fade 取代，需建立 evidence-backed ending_renderer adapter。
- 2026-07-20 external town-flow survey（subagent，非 EXE 硬證據）：中文攻略逐章列出羅德鎮、塞拉村、普里茲港等戰間武器店／道具店／神秘商店／教會與整備；第2章明載保住村民後戰後獎勵力量藥水，第6章明載戰後貝克威加入。這強力支持 battle→postbattle→town/shop/preparation→next battle 的可編輯圖，但攻略無法單獨證明「勝利後自動進城」的程式級轉移；精確順序仍以 `campaign_full.json` 與 direct disassembly 為準。參考：[青衫 FD2 攻略](https://chiuinan.github.io/game/game/intro/ch/c31/fd2/fd2/fd2.htm)、[PTT 攻略轉載](https://www.ptt.cc/man/Old-Games/D9EE/D31B/D56E/M.1099301522.A.DE5.html)、[GitBook 第7章](https://jaceju-favorite-games.gitbooks.io/fd2/content/walkthrough/7.html)。
- 2026-07-20 ch29 cleanup slice：`0x25089` 已 lower 成可編輯 `reset_persistent_roster_state`，runtime 依 direct disassembly 清 persistent roster transient/acted 欄位並將 HP/MP 回填 MaxHP/MaxMP；與 `sync_party` 分離，避免把戰後投影誤當終盤清理。campaign/cmd regression 已補上；GUI package 仍受容器缺 ALSA/X11 headers 限制。
- 2026-07-20 external town-flow survey（subagent，非 EXE 硬證據）：GameFAQs 明載第14章對話「途中有小鎮，先休息」，直接支持 battle→town/rest；第22章明載至第26章前沒有 rest/buy/sell，故 ch23–25 不得強插 town/shop。其餘第2、5、6、7、9–22、26–27章有旅館／教會／武器店／道具店／秘密店旁證；攻略無法單獨證明 handler 觸發時機，精確順序仍以 `campaign_full.json` 與 EXE/資產為準。
- 2026-07-20 ch29 tick slice：`0x17aa9(1)` direct RE 的全域 tick busy-wait 已 lower 成 editable `delay(frames=1)`，保留 ch29 redraw loop 的每次 60Hz 邊界；compiler regression 通過，`0x24618`/`0x2bce5` 仍維持 fail-closed。
- 2026-07-20 ch29 palette-loop slice：`0x11df2(EBX,255,0)` 的 direct loop（EBX=0x3e..0、每次後接 4ms wait）已 materialize 為 63 組 editable `palette_update` + `delay(ms=4)`，不再把 register expression 靜默丟失；其餘 ch29 unresolved 降至 layout/0x24618/load-text/pan/0x2bce5。
- 2026-07-20 0x24618 indexed-transition audit：Capstone 確認 9→1 frame、每幀 descriptor/double-buffer copy、5ms tick，尾端 500ms；之後 32 次 `0x11df2(start=0,end=255,delta=0..62 step2)` + 4ms，這是整張 VGA palette brightness ramp，不是 `(0,255,index)`。新增 `HandlerIndexedTransition` editable metadata 與 explicit binding resolver/compiler test；PNG renderer 仍 fail-closed，尚未接 campaign。
- 2026-07-27 official IDA `0x24618` body recheck：固定 `0x11eee` 13×8 terrain staging、九次 descriptor `9..1` pass、每次 `0x22046`→320×192 viewport copy→5ms、500ms hold、palette delta `0..62` step2/4ms。新增 `fdother.BuildNativeIndexedTransitionSchedule` 保存 outer loop raw cadence；descriptor/double-buffer/presentation 仍 fail-closed。
- 2026-07-27 stale dialogue assertion cleanup：重跑 `0x15f84/0x168b6/0x1956b` 後，`09`、`01`、`18` 改為 operand provenance 模型：`-17/-18` 先經 `0x12c60` identity lookup（必要時 direct-DATO fallback），`-19/-20` 走 runtime unit `+7`，`FFFA/FFFB` 是遞迴名稱／數值插入碼。刪除「每個 operand 都是固定肖像 ID／特效」斷言，未改動 story JSON。
- 2026-07-20 0x24618 schema completion：metadata 另保留 fixed `source_y=0`、`blit_width=0xc0`、clip `0x138×0xc0`，以及 tile/source step；compiler 只接受完整 9-frame/500ms/palette timing，避免把 descriptor copy 簡化成普通 fade。
- 2026-07-20 ch29 post `0x1088d` correction：先前將 `0x25870 → 0x1088d` 縮窄 lower 成 `load_ch_text(ch30.json)` 已撤回，因 Docker/Capstone 證實本體不只載 FDTXT：它依 chapter 載三個 FDFIELD resource、讀 map control、重建 `0x1e00` runtime unit buffer、從 persistent `[0x53bf7]` 複製 own records、套 own-deploy coordinates 並 spawn groups。現已以既有完整 `loadch` state（chapter30/map29/roster70/ch30 story+scenario）重新 lower，compiler regression 明確鎖住不得退回文字-only；handler 仍因 layout、transition、ending 等 unresolved ops fail-closed，尚未接 campaign。當時把 map29／LOADCH context 暫記為 internal chapter 29 終局候選；這不是已閉合的玩家戰次或 campaign owner。`0x2bce5` 返回後自迴圈，不會進 generic preparation；「現行 final battle→generic ending 暫略過此 handler」只描述當時狀態，已由本檔後續 2026-08-12 的近似終局前綴、蒙太奇、尾段與持續隊伍勘誤取代。
- 2026-07-20 ch29 post layout audit：Docker/Capstone 證實 `0x257b4 → 0x233c6` 使用三個固定 20-byte arrays（slots 0..19 的 X/Y/pose）與 camera `(16,18)`；數值已可重取，但 remake 尚未證實 20-slot `handlerUnitAt(slot)` 身分等同 native runtime array，且 campaign 未接這個 native post handler。因此不建立猜測性 `layout_units` binding，維持 fail-closed；先需補 roster frontier/identity evidence。
- 2026-07-20 terminal-flow reconciliation：`0x25970 call 0x2bce5` 之後的 `EB FE` 是 `jmp 0x25975` self-loop，證實 `0x25757` 不會返回通用戰後／整備流程。當時依同函式的 map29 FDFIELD／LOADCH context 暫列為 internal chapter 29 終局候選；2026-08-09 caller-table 勘誤補充：不可只由 table index 29 命名玩家戰次，也不能把候選當成玩家第29戰的正式 owner。`preparation_ch30` 仍是終局**之前**的既有節點，不能把此 post handler 接到 map28 的 `battle_ch29` 勝利。
- 2026-07-20 map29 final roster provenance：`0x1088d` 證實 `[0x53a45]+slot*0x50` slots 0..19 先由 persistent `[0x53bf7]` ordinal 0..19 複製，再寫 map29 own-deploy ordinal positions；`0x233c6` 只覆寫該同一 buffer 的 x/y/pose。`0x1b750` 是裝備／衍生能力 synthesis，不改 identity。進一步 direct `0x112a5` 證實 JOIN 以 `[0x53bfb]*0x50` append 一筆 persistent record 後遞增 count，故正常遊戲 ordinal 就是首次 JOIN chronology；remake `partyJoinOrder`／`reorderScenarioParty` 的角色順序方向正確。map JSON row order 仍不得替代此 persistent order。
- 2026-07-20 final layout materialization：將 `0x257b4 → 0x233c6` 的三個 native 20-byte arrays 完整寫入 editable `layout_units` binding（slot0..19、camera 16×24/18×24）；compiler regression 鎖定首兩筆、末筆、camera。這只保存已證實資料，不能繞過終局 handler 的 runtime array/renderer gate；其餘 unresolved ops 仍阻止 campaign playback。
- 2026-07-20 final camera pan：`0x25933 push 12; 0x25935 push 11; call 0x135dd` 依 native x-first/y-second ABI lower 為 editable tile-step pan `(264,288)`；compiler regression 鎖定此 final-map camera target。它不影響仍 fail-closed 的 transition/ending path。
- 2026-07-20 final indexed-transition callsite：`0x233c6` 先初始化 viewport origin `(16,18)`、focus `(22,23)` 經 `0x11bfa/0x11b9b` 將 scroll offsets 寫為 `(6,5)`，故 `0x25848` 的 dynamic args `[0x53ab9], [0x53abd]+1` 精確為 `(6,6)`。完整 9-frame descriptor/palette metadata 已以 editable `indexed_transition(tile=6,6; source=10,step8)` binding 保存；runtime adapter 尚未具備 indexed descriptor renderer，仍 fail-closed。
- 2026-07-20 ending asset audit correction：`0x2bce5` 的 `push 0x36` 是十六進位 immediate，實際由 `0x111ba` 載入 FDOTHER index **54**，即 `FDOTHER_054.bin`（263655B、111-frame table），不是 `FDOTHER_036.bin`（31008B、408×138 的無關資源）。`0x111ba` 已直接確認只讀 archive entry、沒有解壓/轉碼；`0x2935b` 因而直接吃 raw #054。frame descriptor 的 `+0/+2`=destination dx/dy，`+9/+11`=real width/height，payload `+9` 交給 `0x4e63d`。新增純 Go `internal/fdother` fail-closed parser/blitter：透明 skip 與 dither 都保留既有 indexed destination，不能像 PNG exporter 寫 index0；合成 RLE 與玩家素材 #054 frame geometry regression 均已覆蓋。其 `DecodeResource(FDOTHER.DAT, 0x36)` archive loader 另以 raw #054 byte-for-byte 驗證。ANI `#2` 已有 `internal/afm` decoder（26 frames），但完整 branch/text/ANI schedule與 runtime bridge 尚未接入，故 ending 仍 fail-closed。
- 2026-07-20 ending #054 schedule audit：`0x2bce5` 的可證實部分為 frame0→offscreen/copy、1000ms、palette `(0,255,63)`、frame9、`63..0`×4ms fade、2000ms hold；不透明文字 helper branch 後有三輪 `63..0`×4ms +200ms，frame12..108 每幀20ms；之後第一段 40 次 640-stride frame-pair composite，第二段 200 次 frame-pair/VGA composite（每次20ms，final 64 次的 `0x11d40` first arg 才由0改1）。已將 #054 全111幀以原版透明分支實際 decode 回歸。`0x2c39b` 只證實兩個 caller args 會轉交 `0x1956b`，未證實為字串 ID 或位置，故 timeline 只能將其 opaque 保存、不能猜成 editable dialogue；ANI/後段戰鬥動畫 bridge 仍 fail-closed。
- 2026-07-20 editable terminal-prefix IR：新增版控 `remake/assets/endings/native_2bce5.json`（僅 RE choreography，沒有原版美術）及 `internal/ending` loader。它將所有已釘死 prefix calls 存為可編輯 JSON，including #054 frame0/9/12..108、palette ramps、兩個 `0x2c39b` chapter branch 的 **opaque arg pairs**、兩段 exact loop bound/formula。status 固定 `recovered_prefix_only_fail_closed`，loader 的 `Ready()` 只在明確 `ready` 才可真值；現階段 regression 鎖住不得被 campaign 誤播。下一輪要先釘死 0x2c39b 的字串/位置與 buffer/palette helper，再談 renderer bridge。
- 2026-07-20 ending text-helper slice：`0x2c39b` 在保存 EBX 後將 caller arg1 交給 `0x1956b`，再以 caller arg2 呼叫 `0x15f84([0x53a79], idx, ...)`；`0x15fb9` 的 FDTXT offset-table lookup 證實 arg2 是 **current FDTXT string index**。終局 loader 已選 `FDTXT_030`，故 final-route else branch `(37,2),(21,3),(26,4),(105,5),(32,6)` 精確 count-align 至 `ch30.json` scene1 lines `0..5,6..7,8..9,10..11,12`；後段 `(45,7)` 對應 line13。timeline 現把它們做 editable `else_dialogue` blocks；arg1 僅記為 `visual_resource_index`，因 `0x51a70` archive 類型尚未直接命名，不能猜成 portrait。chapter==26 的另一臂 indexes17..20 仍依當時 current FDTXT 待另行 mapping；ending status 不變。
- 2026-07-20 chapter26 ending-text closure：`0x2545d → 0x2bce5` 在 `ch26_post` 內沒有後續 LOADCH，故 `0x2c39b` chapter==26 branch 的 current table 是 `FDTXT_027`。Docker-isolated raw string decode 證實 idx17/18/19/20 依序正是 `ch27.json` appendix scene（index3）的 lines1/2/3/4：「看！是黃金城」到「一個沒有人找得到的地方」。timeline 的 `then_dialogue` 已保存這四個可編輯 blocks，並為每個 exact visual-resource index、FDTXT string、scene/line/count 建 regression；不再把 bad ending 只寫成 generic text。`0x51a70` resource type和 renderer bridge仍未證實，維持 fail-closed。
- 2026-07-20 ending portrait-type closure：既有 doc14 的 direct trace 已明載 `0x51a70="DATO.DAT"`；本輪核對 `0x2c39b`，其 arg1→`0x1956b`→`0x111ba(0x51a70,arg1)`→DATO decoder `0x4e8e1`。因此 ending timeline 的第一參數從保守 `visual_resource_index` 正名為 `portrait_id`，而非猜測的背景／figure id；所有 final/bad-ending dialogue blocks 的 DATO IDs 均保留。這解除文字 helper 的兩個 ABI 語意，但仍未提供 320×200 ending compositor／palette/ANI bridge，status 維持 fail-closed。
- 2026-07-20 `0x2935b` decoder contract：ending callsites 證實 source 是 frame-table container；frame `n` 的 descriptor 位址來自 `base + uint32[base+8+n*4]`，`u16 width/u16 height` 位於 descriptor `+0/+2`，RLE payload 由 `+9` 以 transparent `-1` blit 至指定 320/640 stride buffer。`0x2bce5` 已釘出 frame0→offscreen、frame9→screen fade、12..108 timed sequence與後續 double-buffer interleave；仍須把這些 calls/branch text變成資料化 schedule後才可接 runtime。
- 2026-07-20 ending prefix player slice（2026-07-28 DAC／runtime audit勘誤）：新增 `internal/ending.Player`，以 presentation millisecond clock 執行 `frame0→offscreen/copy→1000ms→ANI#2`（ANI first frame immediate、每後續幀100ms）。Docker `fd2-cap-local` read-only disasm 直接證實第一 ramp loop 是 `0x11df2(0,255,EBX)`、EBX=63..0、每步4ms；2026-07-28重讀callee確認它從immutable `[0x53a65]`重算range RGB，不是對current DAC additive。後續frame12..108、40/200-pass composite與phase0其實也已有executor；舊「integration只走到第一text」是preview lifecycle落後，不是RE gate。修正後玩家`FDOTHER/FDTXT/ANI` integration可走完整recovered prefix並精確停`0x2c548`。`FD2_ENDING_PREFIX=1`仍未接campaign terminal handler。
- 2026-07-20 native ending dialogue bridge（2026-07-28 lifecycle修正）：`Player.BlockedDialogue(chapter)` 只對 `native_text_branch_opaque` 交出 timeline 的 exact then/else blocks；preview 以 `FD2_ENDING_CHAPTER=26|29` 明確選擇 native branch，使用 `loadStoryScriptAt(chNN.json, scene_index)` 的 line/count slice，並用 timeline `portrait_id` 覆寫 transcript speaker，符合 `0x2c39b` arg1=DATO、arg2=FDTXT index 的 direct ABI。preview bypass map loading 時仍載 DATO portraits/font，Enter/Space 可逐頁／逐句。刪除「對話清完仍停同一gate」錯誤斷言：native helper返回後必續行；runtime現在只resume已完成的text gate並重設queue latch，使第二段`0x2bf1c`可排入，其他opaque gate仍不可跳過。
- 2026-07-20 native text resume boundary：preview 在最後一頁最後一句被確認後才呼叫 `Player.ResumeBlockedDialogue()`；此 API 只接受 `native_text_branch_opaque`，會清 block、segment+1 回到 running。任何其他 opaque op（含 composite）都不能 resume，確保 UI 不會以「按完對話」跳過尚未反組譯的 renderer。
- 2026-07-20 text-post palette repeat：timeline 的 `palette_ramp_repeat`（native 3× EBX=63..0、4ms/step、每輪後200ms）已在 `NewPlayer` 展開成可時鐘驅動的三組 `palette_ramp`/`delay_ms`，保留每個 DAC mutation 與 hold，不降級為 fade。完成後下一 gate 是已知但尚未 player 實作的 frame12..108 20ms sequence。
- 2026-07-20 timed ending frames：`blit_frame_sequence` 現由 player 展開 frame12..108 的逐幀 transparent VGA blit + 20ms delay，完成後可到第二個 `native_text_branch_opaque`；composite loop 的文字欄位改名 `first_frame_formula`，避免與可執行 sequence `first_frame:int` 同名造成 JSON unmarshal ambiguity。`0x2bf60` 的 640-stride composite 已有精確 adapter，但後續 `0x2c172` montage 仍 blocked。
- 2026-07-20 ending composite correction：`0x2bf60` 和 `0x2c0c5` 都以 `(i%4)+1`、`(i%4)+5` 循環取兩組 frame；先前資料中的 `floor(i/4)+1` 是轉錄錯誤，已修正並以 timeline regression 鎖定。200-pass loop 完成後仍只允許到 `0x2c172` gate，絕不把未恢復的 finale montage 當作完成。
- 2026-07-20 composite helper closure：Docker-only Capstone 確認 `0x11eb0(dest,destStride,src,srcStride,bytes,rows)` 是逐列 copy。第一個40-pass loop 已完整釘死：`EBX`=320×200 background、`ESI`=640×200 work；每輪先 EBX→Work viewport x=160，再將 frames `(i%4)+1`／`(i%4)+5` 以 stride640 blit 到 primary/secondary origins，20ms 後 present Work[x=160..479]。primary 初值290，i0..24每輪-4、i25..39每輪-2；secondary 初值80、每輪+2。完成後 second loop 幾何也已知為 Work 320-stride pair；唯一未證實的是其 `0x11d40` palette helper 語意。下一步可安全實作 `BlitAt`／`CopyRect`，不需再猜 buffer layout。
- 2026-07-20 composite scheduler：`Player` 已接 first 40-pass `0x2bf60` loop，逐輪 `Composite40(i)` 後等待20ms，完成才落至第二個 text gate；source 不是 `0x2bf60` 的 composite 仍必定 blocked。完整 internal ending regression 已通過。
- 2026-07-20 `0x11d40` closure：Docker Capstone 直接確認它讀 `[0x53a65]` 的基準 RGB DAC、逐分量做 `base-delta` clamp 到0，再寫 VGA DAC；不是對目前 palette 累加/減。第二200-pass caller 的 EDI 前136輪為0、最後64輪為1。實作第二 loop 必須保留 baseline palette snapshot，不能誤用破壞式 `0x11df2`/`PaletteDelta`。
- 2026-07-20 finale `0x2c405` phase-0 control-flow closure（Docker-only Capstone）：handler 先 `load_ch_text(30)`，alloc/clear `0x36b00` bytes staging buffer，並由 `0x15f84` 將 native text composite 寫至 `staging+0x12c30`；接著正好500次 loop。每次以 `0x11d40(0,255,BL)` 套 baseline palette，從 `staging + iteration*320` 複製 320×200 到 VGA，並 wait 1ms。`BL` 初始40；iteration≤300 時每5 tick 在非零時減1，之後每5 tick 加1。staging glyph path 與 ABI 已恢復，但可執行範圍仍只到後續 `0x2c548` montage gate，不能以空白 scroll 或 generic ending 取代。
- 2026-07-20 finale phase-0 script correction：先前把 `0x2c469` 的 `0x2c` 誤投射到 FDTXT_030 logical #44，已撤回。`0x2c405` 先 `load_ch_text(30)`，而 `0x1088d` 的 resource rule 是 chapter+1，所以 current table 是 **FDTXT_031**；其 raw offset table 有46項，`0x15f84` arg2=`0x2c` 正是合法實體 string #44。raw bytes decode 為「在亞克斯王國宮廷中…各奔前程…」的後日談跑馬燈前言，已對位跨資源重用的 editable `ch32.json` scene0 line0。`native_2c405.json` 與 raw-resource regression 已改為 `FDTXT_031/#44`；完整 finale montage 未復原前，`Ready()` 仍固定 false。
- 2026-07-20 `0x4ea2a` glyph ABI closure：Docker-only Capstone 顯示 arg1=1bpp font base、arg2=glyph index（`index*32`）、arg3=destination、arg4=stride、arg5=foreground、arg6=shadow、arg7=optional background。每 set bit 寫 foreground，並寫左一格／下一列 shadow；arg7 非零時才填整個16×16 cell。`0x15f84` 的 four shifting `push [esp+0x50]` 還原為 caller arg4..7；finale `0x2c469` 因此是 stride320、foreground `0xCD`、shadow `0x4C`、background0。`internal/fdtxt.BlitNativeGlyph` 已逐 pixel regression；仍不可把此 primitive 當作完整 `0x15f84` control-code renderer。
- 2026-07-20 finale phase-0 glyph composition correction：FDTXT_031 physical #44 的130 words 含121個 glyph + 9個 `FFFE` soft line breaks（末尾 FFFF terminator 不在 parser words）。`0x15f84` 的 FFFE path 是 `destination + (++line)*arg8*stride`；本 caller arg8=`0x19`，故每行跳 `25×320` bytes，而非自然 buffer wrap。`ComposePhase0Text` 現精確套此規則並以 FDOTHER #4 / CD/4C/0 style blit；其他 FFxx 仍拒絕，phase readiness false。
- 2026-07-20 finale phase-0 bridge：Docker/Capstone 重驗 `0x2c405` 會在 `0x2c4b4` 前保留前段 indexed presentation 的 DAC，首 pass 呼叫 `0x11d40(0,255,40)`，並在 `0x2c172` 先清 VGA。`Player.EnableRecoveredPhase0` 因此不接外部便利 palette：僅當同一 compositor 已由 `PresentANI` 捕獲原版 palette 才能 hand-off；後續 500 pass 完成會停在 `0x2c548` 的獨立 montage gate。無 baseline、其他 source、或 phase asset 不完整都仍 fail-closed。
- 2026-07-20 finale phase-0 scheduler closure：`0x2c4d6` 的 `[esp+0x20]` 因前面三個 pushes 實際回指 loop local `[base+0x14]`，即 i；source 是 `ESI+i*320`。i=0 先 present palette delta40 / row0，之後若 i<200 且 i%5=0、delta非零便減；`i<=300` 直接 wait1/inc，i>300 後 i%5=0 才加。`Phase0Player` 以此 500×1ms cadence 實作且測試 i0/i1、i195、i305、i499；它結束時不跳進 `0x2c548`。
- 2026-07-20 post-composite boundary（2026-07-28 runtime接線更新）：`0x2c172` 後不是 handler return：先 free work、clear/present、呼叫 `0x2c405`，再載 FDOTHER #60/#58/#57/#59、呼叫 full battle renderer `0x28a6c` 與更多 palette/tick choreography。preview現從同一FDOTHER讀font#4、從FDTXT.DAT讀resource31，啟用既有phase0 bridge並跑完500-pass scroll；因此實際gate前進到`0x2c548`。不得把此狀態宣稱完整native ending或接campaign terminal route。
- 2026-07-20 finale montage first slice：`0x2c405` 先 `loadch(30)`、alloc/clear `0x36b00` staging buffer，執行500 native ticks 的 baseline palette/scroll choreography；再 alloc 0x1f400 + two 0xfa00 buffers、載 FDOTHER #56，並以 native unit records 載 FIGANI/DATO resources、`0x29164`/`0x2b9a1`/`0x28a6c` battle render path 作多輪 320×200 present。這是獨立 finale-montage phase，不可降級重用一般 battle scene；目前只記錄資料邊界，仍 fail-closed。
- 2026-07-20 finale `0x2c548` first party-cycle closure（Docker-only Capstone；2026-07-28 timing correction）：0x2c551/560/573 依序 alloc 0x1f400（320×400 staging）、0xfa00、0xfa00；`0x111ba(TAI.DAT,#3)`、`0x111ba(FDOTHER.DAT,#56)` 後先貼 backdrop。更正 TAI#3：raw bytes 恰為 `0A00 0300 C9C9C9`，即 10×3 全 transparent RLE placeholder，**不是**可見 platform，故 `0x29164` 中該參數的 renderer role 尚未猜定。迴圈 index 由 `[0x53bfb]-1` 向下，但 native 選 unit 不是 identity：`i==0→slot1、i==1→slot0、else→sloti`，再以 `[0x53a45]+slot*0x50` byte `+7` group 形成 FIGANI `group*3+1` 和 `group*3`。`0x29164` 後先有 `0x2b9a1(F0,-1)` 20×1 **BIOS tick** loop（不是1ms），再有 `0x2935b(F1,frame)` 逐 descriptor `+6` tick delay loop，才進 portrait/text/input branch。以上結構進 `assets/endings/native_2c548.json`；完整 scheduler仍不可由既有RGBA battle renderer代替。
- 2026-07-20 finale `0x2c548` portrait/text closure（Docker-only Capstone；2026-07-28 executable update）：後段 DATO resource=`unit[+7]`；current `[0x53a79]` 是 FDTXT_031，#10「姓名：」→`staging+0x16e9`、#11「職業：」→`+0x2fe9`；permanent `[0x53a7d]` 是 FDTXT_000，`unit[+8]+1`→`+0x171b`、`unit[+0x20]+0x96`→`+0x301b`。新直接反組譯固定 portrait destination=`staging+0x0c88`，countdown初值0；零時設 `(random&31)+40` 且不立即減，非零先減，結果`<2`選DATO frame3、否則frame0。loop index非0跑220 ticks；index0跑440 ticks，因此只有swap後slot1在tick220後以FDTXT_031 #45取代一般`unit[+8]+0x0c` epilogue。`ComposeMontagePortraitFrame` 已以玩家原始 FDOTHER/FDTXT/DATO regression 驗證 normal/mouth與special text合成、restore不變；未知FFxx仍fail-closed。`0x10620` kbhit只令outer counter=1，完整FIGANI→portrait→input phase尚待組裝。
- 2026-07-20 indexed FIGANI decoder：新增 `remake/internal/figani`，以通用 LLLLLL reader 讀 FIGANI resource，嚴格驗 frame offset table、13-byte header（signed dx/dy、`+6` delay、`+9/+11` real geometry），完整 4-mode RLE 轉成 indexed `Pixels+Mask`；transparent/dither span 保持 native destination-preserve，不把它化成 PNG alpha 或 palette0。`Frame.BlitAt` 可供 native 320/640 stride surface 使用，synthetic codec 與 player-provided `FIGANI.DAT` #13 regression 都通過。尚未宣稱 `0x29164` 完成：仍需 TAI #3、640-stride layout、8-step palette fade 和 specific slot renderer state。
- 2026-07-20 `0x29164` argument/fade correction（Docker-only Capstone）：`0x2c663` 的最後 push（前方已有6 pushes）才是 arg1=party loop unit index；callee 以 arg1×0x50 讀 `[0x53a45]+6` 決定兩條 path。因此 TAI#3 是尾端 aux argument，且其7-byte all-transparent raw **不能**是 `0x2935b` frame table。兩 path 都從 `esi=8` 倒數到0，逐輪 `0x11d40(0,255,esi*6)`，即 delta 48,42,36,30,24,18,12,6,0，合計9次 320×200 present（不是先前籠統寫的8-step fade）。640-stride figure/platform geometry 與 aux role仍待下輪，禁止接 RGBA renderer。
- 2026-07-20 `0x29164` final-caller non-mirrored geometry：`0x2c663` 呼叫 mode=1、使 finale party records 走 `unit+6!=0` path。每 stage 8→0 以 `work640 + stage*10` 為 byte origin，對 TAI#3 以 (164,157) 做 transparent no-op，對 **secondary** FIGANI（group×3）以 `0x2935b(frame0)` blit，再從 work left viewport copy 320×200 至 VGA；DAC delta=stage×6。這一條已資料化成 `Montage.PlanFigureFade(1)` 的九 pass，且 test 明確拒絕 unitSide0 mirrored path。restore buffer的實際初始來源／後段 primary FIGANI仍未 lower，故 schedule 本身不是可播放 renderer。
- 2026-07-20 `0x29164` restore/compositor closure（Docker-only Capstone）：`A=0x1f400` work、`B=first 0xfa00`、`C=second 0xfa00`；FDOTHER#56 先 blit 到 B。每 non-mirror fade pass 的直接 `0x11eb0` 是 **B→A**（dstStride640/srcStride320、320×200），再 secondary FIGANI→`A+stage*10`、A left viewport→VGA；`RenderFigureFadePass` 已用 indexed buffer 實作這個完整且窄的 primitive，要求 TAI#3 exact transparent bytes。primary FIGANI animation 後 `0x373c4(C,B,0xfa00)`，故 C 是 portrait/text loop 的 frozen restore base（後段每 tick C→A 都為320 stride）。這解除 restore 來源，但未 lower primary animation/DATO text 或 mirrored path，player integration 仍封閉。
- 2026-07-20 finale #56 format closure：玩家 raw `FDOTHER_056.bin` 為13609 bytes，前4 bytes直接是 little-endian 320×200，後接**單一** `0x4e63d` payload、沒有 #54 的 frame-table。新增 player-asset regression 證實以 `Frame{Width:320,Height:200,Pixels:data}.Blit(...,320,-1)` 成功 decode；可重用既有 transparent RLE grammar，但不可使用 `ParseFrames` offset table parser。
- 2026-07-25 item-action provenance correction：Docker/Capstone 重檢 `0x1bbdc` 與 `0x20c6f`。`0x1bb8c` 僅是把 item 插入第一個空 inventory slot，不是 item effect；case1 為 transfer（插入+`0x1b8e7` removal），case2 為 `0x1bffe` equip。case0 依 item `+0xd` type 進 `0x20c6f`，再分派至多個原生 effect routines；各 callee 數值語意仍未閉合，remake 維持 fail-closed。已同步更新 battle-menu、item RE、UI evidence、worklist，刪除錯誤 effect/consume 斷言。
- 2026-07-25 non-mirrored figure-fade implementation：將已證實的 `0x29164` 窄 primitive（B→A 320→640 restore、secondary FIGANI frame0 置於 `stage*10`、A left viewport→VGA、baseline DAC `stage*6`）整理成 `RenderFigureFadePass`，`BlitAtBase` 支援 native work-surface origin，並以 exact TAI#3 transparent bytes 與像素 regression 鎖定。這只完成 evidence-backed primitive；primary FIGANI、DATO text、mirrored branch 與完整 ending player 仍 fail-closed。
- 2026-07-25 type-0x17 item callee audit（後續已 superseded）：當時只確認
  `0x2218a` 依 target `+0x20/+0x21` 更新 raw accumulator、呼
  `0x22253` 並寫 target `+0/+1`；完整 MP debit、identity/maxMP gate、
  relocation transaction 已於本文件尾端閉合。
- 2026-07-25 nested FDOTHER boundary：`FDOTHER.DAT#0x51` raw entry 已由 Docker-backed regression 證實是 nested `LLLLLL`（first-word `0x12`、18710 bytes）；新增 `fdother.ArchiveEntry` 只做 raw boundary validation，不把 nested payload 猜成 frame table。**2026-07-26 更正**：`0x11eee` 是背景 redraw、不是 nested data selector；#81 allocation 只 free、不傳 renderer，真正仍 fail-closed 的是 FDOTHER#6-derived local pointer decoder/call semantics 與 `0x22253` indexed runtime adapter。
- 2026-07-25 acting decoder correction：Docker/Capstone 直接重檢 `0x1366a`。`bit7=0` 每格固定 7 tick，逐 tick 寫 unit `+3=pose`、`+4=1..7`，第 7 tick 才依 pose 更新 X/Y；`bit7=1` 不搬格。`bit7=1/low7=0` 仍是有效 frame，原版含 delay(1)+重繪+delay(2)，remake 以三 tick 保留時序並每 tick 重寫 pose；新增 zero-special pose/timing regression。direct 106-entry acting bank 與 slot ABI 維持已解定，renderer/presentation 尚不猜測接入。
- 2026-07-25 command-label table closure：Docker-only Capstone 的 `0x1ceed` 直接呼叫 `0x15f84([0x53a7d], 0x1b9+commandID, ...)`；既有 permanent-table trace 對齊 `[0x53a7d]` 為 FDTXT_000。從 raw offset table strict decode 的 physical strings #441..#480 已匯出到 `docs/data/command_labels.json`（含空 slot 與系統訊息）。這使 label 成為 editable evidence；它**不**證明每個 command ID 可達或定義 effect／target，後者仍 fail-closed。
- 2026-07-25 raw command-mask pipeline：FDFIELD roster 的 `b13..b16`（constructor `0x10f7f` copy 到 runtime `+0x1a..+0x1d`）不再被 parser 丟棄。`parse_field.py`／`export_units.py` 輸出 `initial_command_mask`，battle `Unit.NativeCommandMask` materialize 為 5-byte ABI，strictly 支援 native order enumeration 與 0x1d7fb-style bounded OR；malformed source 拒絕載入。legacy `Spells` list 仍是另一條 normalized gameplay approximation，不能當作這個 raw mask。
- 2026-07-25 command-record table identity：Docker read-only table comparison 證實 `0x4e516(id)=0x619fd+7*id` 的 IDs 0..35 與 EXE spell table 完全同 bytes；這正名 `+3/+4/+5/+6=dist/range/mp/target`，且 MP gate 有獨立 selector trace。33 maps FDFIELD + 32 character defaults 的 initial masks 實測只出現 IDs 0..30。36..39 的 pointer-adjacent 7-byte data 對應 FDTXT 空／系統訊息 labels，未證實可達，保持 fail-closed。
- 2026-07-25 level-up command learning closure：`0x1e292` 在 EXP threshold 後 increment runtime level，經 portrait growth row `+0x0a=learn_idx`、`0x4e4a2(idx)=0x626b3+12*idx` 掃最多六組 `(required_level,command_id)`；命中後唯一 caller `0x1d79c` OR 5-byte mask，並印 FDTXT_000 #587「學會了！」。20 rows 已由 `dump_exe_tables.py` 導出 `command_learn.json`，FF/FF sentinel 保留。尚無 portrait→growth row 的完整 runtime provenance，不能以 legacy Spells 取代。
- 2026-07-25 learning runtime bridge correction（2026-07-26 最終收斂）：`0x4e4d1(unit[+7])=0x620a1+unit[+7]*11` 直接證實 raw selector→growth row；完整 stack trace 再閉合 constructor FDFIELD `b1→unit+7`，撤回先前因 stack slot 混淆而說 source 未解的記錄。這不使它和 map `unit+2` alias。`State.GainExp` 以 injected `CommandLearn` table 在每次 level increment 後精確比對 raw selector row／new level、OR native command bit，並回傳 learned IDs；standalone legacy `GainExp` 不自動造資料，保持相容與 fail-closed。
- 2026-07-25 learning runtime asset binding：generated `command_learn.json` 已加入 `remake/assets/data`，Game 在 bootstrap 讀它並在 `resetBattle` 的每個新 State bind 同一 table；缺檔不回退 legacy Spells。Docker internal regression 全綠。
- 2026-07-25 UI test container closure：`tools/docker/fd2-go-test.Dockerfile` 集中 Ebiten Linux 所需 ALSA/X11/GL `pkg-config` development headers，並在 image build 時預抓 `go.mod` dependencies；`fd2-go-test-local` 已以 `--network=none` 實跑 `go test ./cmd/fd2 ./internal/... -count=1`、exit 0。這取代先前散落的「GUI compile 被缺 ALSA/X11 容器阻塞」說法；它是可重現的 source regression，不等同完成原版 UI 實機對照。
- 2026-07-25 UI action wrapper correction（Docker-only Capstone）：`0x18d8c` 是 action dispatch wrapper。撤回把 `0x1b83d` 稱為「前序選擇」的錯誤說法：它掃 unit 八個 inventory slots，找 `bit0x40` equipped 且 item ID `<0x80` 的第一格；找不到才設 output `+0=1`，命中才以該 item record 的 `+0xb/+0xc` 建前序 target state。`0x1b8a6==0` 精確等於八格皆 empty（bit0x80 全 set），故設 output `+8=1`；`0x1c269==0` 或 unit `+0x27!=0` 設 `+4=1`。`0x177fc` 回 `-1` 是 selector cancel；僅其後依 `[0x53c57]` 分派 attack／`0x1cff0` command／`0x1bbdc` item／wait-field。flags 對應的可見 action/icon 仍未閉合，保持 raw，沒有接 renderer。
- 2026-07-25 command gate scope correction（Docker-only Capstone）：`0x1598a` 在列舉 command record、MP 或 target grid **之前**，只要 `0x1c269` count 為零或 `unit+0x27!=0` 就直接 zero return；`+0x27` 因此是整個 native command submenu gate，而不是僅 wrapper 局部 flag。全 code scan 另兩個 `+0x27` 命中就是此處與 wrapper read；`0x1eb64` 只是 UI resource frame index。寫入者／status 名稱未閉合，禁止稱作沉默、封魔或接入 remake effect。
- 2026-07-25 native action overlay/input closure（官方 IDA 9.4 + Docker Capstone）：`0x173e7` 由四個 availability words 選第一個零值；`0x177fc` 只允許 availability=0 的 ↑/←/→/↓ 選 action 0/1/2/3，Enter/Space confirm、Esc 回 -1 cancel。`0x18d8c` battle wrapper 的 state table 是固定 `[0,1,2,3]`，故 enabled cells=`[0,2,4,6]`、disabled=`[3,5,7,9]`；舊 handoff 將 `0x1728c` 的另一個巢狀 menu state 誤套於 battle overlay，已撤回。`0x1741c` 以 `[0x53a89]` relative asset table 經 `0x4e9e4` 畫四張 state image，從 shared `+0x390` 做四 frame 十字 slide：up `-0x8e8`、left `-6`、right `+6`、down `+0x8e8`；`0x175a9`/`0x17643` 以 72×72 (`0x1440`) indexed backup restore，`0x176b4` close sequence 為獨立初值／delta（非簡單 reverse）。`0x11bfa/0x11c59` 證實 anchor 使用 visible cursor `[0x53ab9]/[0x53abd]`：共同 byte address=`framebuffer+0x8088+0x18*column+(0x18*0x1c8)*row`，而 `[0x53aa9]/[0x53aad]` 才是 camera scroll。resource provenance／實機 skin 外觀仍未閉合，不能將 remake ring 宣稱成原版皮膚。
- 2026-07-25 action asset provenance/decoder closure：`0x25c97..0x25cac` 直接將 `FDOTHER.DAT#2` 載至 `[0x53a89]`；該 raw 是 untagged 78-cell offset bank（first `u32=0x138` directory end），每 cell `{u16 w,u16 h,w*h raw indexed pixels}`，不是 LMI1 或 ending frame-table。`0x4e9e4` 精確逐列 blit、index0 preserve。新增 strict `fdother.ParseRawCellBank`／`RawCell.BlitAt` 和 player-asset regression；實測 74×24×20 加 4×24×16 cells。relative table→action cell index、anchor 與 runtime renderer 未閉合，保持 fail-closed。
- 2026-07-25 terminology correction：README、SDD、doc14 與 worklist 不再把原版 action UI 斷言成「radial 指令環」；E0 證據是 FDOTHER#2 四張 indexed asset 的十字 overlay 加獨立 command grid。現行 `ringInput` 只保留為 provisional interaction，不能再當 original skin/mechanism 的證明。
- 2026-07-25 title screenshot oracle：新增 repo-maintained `tools/docker/fd2-dosbox-screenshot.*`，以 SVGA/Xvfb/xdotool 對 `/tmp` FD2 sandbox 跑可編輯 timeline；原始 FLAME2 不掛載。連續 Escape 跳 opening 的真實 320×200 crop 已加為 `docs/figures/title-original-dosbox.png`，直接證實 title START／LOAD／CONTINUE 及 cursor。它只提升 UI-01 畫面 E2，不代表 title input、save/load 或 remake title renderer complete。
- 2026-07-25 empty LOAD oracle：原版 title→LOAD 在 empty sandbox 直接得到 `docs/figures/load-empty-original-dosbox.png`，證實 4-slot 縱列、空記錄文字與 row1 outline；僅為 UI-12 empty-state E2，沒有有效 FD2.SAV，故不對 save record、覆寫確認或成功 load 做任何斷言。
- 2026-07-25 ch01 dialogue oracle：原版 START flow 的 `docs/figures/ch01-dialogue-original-dosbox.png` 固定一種 lower/left DATO portrait、blue dialogue box、兩行文字及 bottom-center page indicator。只提升 UI-05 該 anchor E2；upper/right/control code/pagination 仍未閉合。
- 2026-07-25 native action overlay remake slice：remake 可在使用者明確提供 `FD2_ORIGINAL_FDOTHER`（或本機未版控 `assets/original/FDOTHER.DAT`）時，strict 解析 FDOTHER #0 的 6-bit VGA palette 與 #2 的 raw action cells，畫出已證實的最終 opening frame 十字 skin；檔案缺失或格式異常一律回退既有 placeholder。這是靜態 renderer 垂直切片，availability 仍只是目前 remake 的保守近似，尚未聲稱為原版 gate；opening/closing animation、原版 input dispatch 與實機畫面對照仍待完成。
- 2026-07-25 overlay gate tightening（官方 IDA 9.4 asm re-read）：`sub_1B83D(unit,0)` 的 attack precondition 確為 inventory slot 的 `bit0x40` 已裝備且 ID `<0x80`，不是「有任一物品」；`sub_1C269` 的 5-byte low-bit-first command inventory 也可直接作 command availability。remake native skin adapter 已接這兩個狹義前提；當 editable legacy scenario 根本沒有 raw mask 時才保留 normalized Spells fallback。`+0x27`、原版 target geometry 與 item effect 沒有被命名或接入。
- 2026-07-26 native action overlay screenshot closure：Docker/Xvfb 對 player-provided FDOTHER.DAT 的 read-only mount 實跑，截圖 [`action-overlay-native-remake.png`](../figures/action-overlay-native-remake.png) 可見 final-open cross skin。過程修正 screenshot frame scheduler（不能假設 exact frame）與一個實際 renderer bug：`drawRing`/`drawSpellMenu` 不可被 optional Chinese font gate 包住。截圖只證明 remake loader/render path，不取代原版 DOSBox side-by-side visual diff，也不宣稱 gate 或 animation 已完成。
- 2026-07-26 action animation timing correction（official IDA 9.4）：`sub_1741C` 與 `sub_176B4` 都是四輪 `0x4e9e4` cell blit→`0x11eb0` present→backup restore，迴圈中沒有明確 delay/wait；不可由 offset 個數推導每幀毫秒數。保留 open/close geometry E0，presentation cadence 需從 outer loop／實機 trace 另行取得。
- 2026-07-26 command-grid renderer closure（official IDA 9.4）：`0x1d51d→0x1ceed` 確為 320×200 的四列 command grid。第 i command 的 label 位置=`(0x12+0x64*(i/4),0x67+0x16*(i%4))`，FDTXT_000 index=`0x1b9+commandID`；選中/未選中 palette index 分別 `0xc9/0xcd`，record `+5` 的 MP numeric 位於右側。↑↓ wrap、←/→±4 且水平 bounded，confirm 再比較 unit `+0x44` 與 record `+5`。這是 layout/input ABI，不是 effect 或 command-name 斷言；remake command-grid renderer 仍待接入。
- 2026-07-26 native command-grid primitive：新增 `battle.NativeCommandGrid`／`NativeCommandGridMove`，以官方 IDA 確認的四列座標與 0x1d51d navigation 實作成無 effect 語意的純資料層；Docker regression 覆蓋第5項換欄、上下 wrap 與水平 boundary。這為 renderer/input 共用 ABI，尚未把 legacy spell menu 冒充完整原版 command runtime。
- 2026-07-26 command-label bridge：`cmd/fd2` 現可選擇性讀玩家 editable 的 `assets/data/command_labels.json`，以 `command_id` 覆蓋同 ID EXE spell row 的顯示名稱；缺檔／壞檔安全退回 normalized spellNames。這只接 FDTXT label provenance，不改 selection layout、MP gate 或 effect。
- 2026-07-26 native command-grid runtime slice：有 player FDOTHER palette+editable labels 時，ring command branch 現以 raw `NativeCommandMask` 顯示 official-ID A recovered four-row grid，直接使用 palette `0xc9/0xcd` text entries；native navigation ABI 已接。confirm 僅對有 EXE spell record 的 IDs 接既有 target path，其他 ID fail-closed；缺資產保留 legacy spell UI，沒有宣稱 native frame/effect completed。
- 2026-07-26 scenario command-mask audit：command-grid screenshot fixture 盤點顯示 default chapter scenario party lacks raw masks；原因是 party override schema 沒有 FDFIELD `initial_command_mask`，不是 renderer 可任意以 normalized Spells 補齊。runtime 因此正確 fallback；下一個 RE/remake bridge 是把 proven raw field 帶經 exporter→scenario→party materialization，並逐章測試。
- 2026-07-26 scenario command-mask bridge：`PartyMember.initial_command_mask` 現已接通並在 `LoadScenario` 只接受空值或精確四 bytes（malformed fail-closed）；`PartyUnits` materialize 至 native five-byte runtime mask。`gen_campaign.py` 從 EXE `character_defaults.json` 依角色 index 合併 raw source 至 ch01..ch30，保留既有手工 scenario 欄位；ch01 悠妮直接為 `[1,0,0,0]`，不是從 legacy spells 推導。另修正 postbattle persistent projection：完整 `NativeCommandMask` 現跨 town/preparation 保留，故 native level-up `0x1d7fb` 的 fifth-byte OR 不會遺失。已覆蓋 materialization、malformed schema 與 persistent copy regression；仍待每章真機 availability / command effect 對照。
- 2026-07-26 command-confirm dispatch correction（official IDA 9.4 DB/ASM）：`0x1cff0` 在 target confirm 後，IDs `0..8`、`0x18`、`>=0x1c` 直入 `0x2a6bd(unit,id,target,scratch)`；IDs `0x09..0x17`、`0x19..0x1b` 才先跑 `0x1d6c8(id)` 四輪 palette flicker 再呼叫 `funcs_1541f[id]`。撤回舊文「record +4 是 MP/cost」：direct MP gate 為 `+5`。因此悠妮 raw command0 已知進 generic pipeline，仍不能命名成 normalized spell 或接猜測性效果。
- 2026-07-26 command0 compositor boundary（official IDA 9.4 DB/ASM）：`0x2a6bd` 的 ID0 不走 `>=0x20` 或 `0x18..0x1b` special early branch；它使用 generic presentation defaults，`funcs_2ac25[0]=0x26152` 反覆合成 320×200 buffers/FIGANI/FDOTHER 並 present/tick。這只能證明 command0 的 battle renderer path，不能將它誤當 damage、MP 或 status formula；效果 state writer 待後續 dataflow。
- 2026-07-26 command post-resolution boundary：`0x1b6b7` 掃 runtime roster，僅把符合 `+5/+0x31/+0x40` 條件者的 source `+0x31` 起三 bytes 複製至 caller buffer；`0x1cff0` 再傳給 `0x1aa1d`。故這條是後處理（訊息／掉落／互動）資料流，不是 command0 damage/status calculator。三個 raw bytes 的遊戲語意仍未命名。
- 2026-07-26 command0 damage closure re-verified（official IDA 9.4 ASM）：`0x2a6bd` 的 final-target loop 以 `arg_C[var_34]` 取 target slot，並直接 `call 0x1c75e(targetSlot, ebp=commandID)`；該函式在同一 direct path 前呼叫 `sub_2b659`，其 frame event 以 actor／command ID 呼叫 `0x1ca89`。先前一次 grep 漏掉 loop 內 call 而錯誤撤回，現已恢復 ID0 record/hit/HP/MP executor；`+0x40/+0x42` 的 HP 意義亦由此與其他 handlers 交叉支持。
- 2026-07-26 command class-multiplier closure（official IDA 9.4 DB/ASM）：constructor `0x10f7f/0x11399` direct copy source race/class/level 至 runtime `+0x1f/+0x20/+0x21`；故 `0x1c75e` 的 `word_51f96[unit+0x20]` 是 target class-ID-indexed damage multiplier。撤回 doc03 將 race/class/level 放在 `+0x27` 的舊標記；個別 multiplier 的玩法名稱仍待 table/data 對照。
- 2026-07-26 numeric damage resistance closure（official IDA 9.4 DB/ASM + EXE bytes）：`word_51f96` 的 loaded-data file mapping 是 `0x51d96`，即既有職業魔抗 table 的 4-byte rows；其 low byte=`resist_raw`（法師=7 即 30% magic resistance）。已閉合 numeric route 的 base 為 `record.dmg*resist_raw[target class]/10`，並以 `NativeCommandDamage` resolver（hit draw、damage draw、HP clamp）及 fail-closed editable-table loader 實作；不可再把它專屬歸因給 ID0。
- 2026-07-26 command0 target-array closure（official IDA 9.4 ASM）：一般 command record 的 `+3/+6` 進 `0x14818` 產生 candidate unit-index stack array；`0x115b6(mode=+6,count,array)` 做 cursor/confirm，之後第二階段 effect array 才傳 `0x2a6bd(unit,id,count,array)`，其 final loop 可達 `0x1c75e`。撤回把 native command0 當單格 legacy cast 的暗示；尚未命名 target-code/geometry 值域。
- 2026-07-26 `0x14818` target geometry closure（2026-07-28 predicate correction）：`record+3<0x10` 用 `0x4e555` map/reach mask；`>=0x10` 為 horizontal/vertical cross，radius=`+3-0x10`。scan roster 跳 inactive/mask `0xff`；direct Capstone修正 record `+6` predicate為 0:`unit+6==0`, 1:`!=0`, 2:`==1`, 3:`==2`。舊code2 `!=1`是分支方向抄反，已撤回。
- 2026-07-26 target reach closure correction（official IDA 9.4 ASM/data）：`0x4e555` 是 20-byte cost-row helper，mask producer 是 `0x4e040`。它做 cardinal flood-fill，grid bit40 block、bit80 cost=0；但 command selector 固定取 row0，而 `word_61646` row0 的 20 bytes 全=1。因此 command target 不用 terrain-weight，但會受 blockers 限制；撤回「需要 class/tile cost 才能接 native command range」及「無條件 Manhattan」兩種錯誤暗示。
- 2026-07-26 camp offset correction（2026-07-28 target-code correction）：FDFIELD unit `b0` 直接寫 runtime `unit+6`，值 0敵/1友/2己。撤回 docs 把 runtime `+0x0e` 寫成 camp 的舊表；`0x14818` target codes正確為0=enemies、1=non-enemies、2=allies、3=own。
- 2026-07-26 native command MP transaction closure（official IDA 9.4 ASM）：generic route `0x21227` 在 candidate array 建立後、逐 target effect 前呼叫 `0x1CA89(actor,commandID)`；後者由 `0x4e516(commandID)` 取 record `byte+5`，直接扣 runtime current MP `unit+0x44`。selector 的 `currentMP >= byte+5` gate 在此前，因此 remake `SpendNativeCommandMP` 只表達已確認交易、invalid/不足不寫入，且不以 normalized `Spell` 冒充 raw command。target confirm/effect/UI 仍不接。
- 2026-07-26 native command two-stage target correction（official IDA 9.4 ASM）：撤回「`0x115b6` confirm 後把 first `record+3` candidate array 直接交給 `0x2a6bd`」的錯誤捷徑。一般 `0x1cff0` 先以 actor cell/`+3`/`+6` 建可選中心並交 `0x115b6`；confirm 後以 cursor cell/`+4`/`+6` **再**建 final-effect array，僅第二 array/count 傳入 `0x2a6bd`。`NativeCommandTargetCells` 只表達一次 `0x14818` primitive，caller 必須明確選 stage；UI 繼續 fail-closed。
- 2026-07-26 raw command UI fail-closed correction：撤回 native command grid「有同 ID `spells.json` row 就可進 legacy `CastArea`」的暫接。這會跳過已證實的 actor `+3`→cursor confirm→cursor `+4` target pipeline，且 table identity 不證明 effect equivalence。grid confirm 現清楚顯示未驗證並回 action overlay；legacy spell menu 仍是獨立、可編輯的 approximation。
- 2026-07-26 verification note：Docker `go test ./internal/battle` 通過；同 image 的 `cmd/fd2` compile/test 在此 runner 會停於 Ebiten/CGO build（`CGO_ENABLED=0` 則明確報 cglfw build constraints），故此次 UI guard 僅以 gofmt、battle regression 與 diff check 驗證。不可把該環境限制解讀為 UI behavior 測試已通過；應在 Xvfb-capable image 補 UI smoke。
- 2026-07-26 command completion flag closure（official IDA 9.4 ASM）：`0x18d8c` 的 native command branch 僅在 `0x1cff0` success（非 0/non-cancel）後呼叫 `0x13512(unitSlot)`；後者唯一寫 `runtime unit+5 |= 0x80`。這直接證實 native command 成功耗用行動、失敗/取消不耗用。只可套用於已閉合 handler；不能據此替 ID0 猜測 mutation。
- 2026-07-26 ID0 UI vertical slice re-enabled：direct `0x2a6bd→sub_2b659/0x1c75e` dataflow 已逐行確認後，raw native grid 的 ID0 恢復 actor `+3` candidate highlight、confirmed cursor `+4` final effect、state-bound record/resistance/flags core 與 ESC 回 grid；仍不包括原版 compositor/post-resolution/SFX or screenshot oracle。
- 2026-07-26 shared damage IDs0..8 correction：wrapper 可達性與 player dispatch 是兩個不同問題；雖 IDs0..8 不經 `sub_21227`，它們的 direct `0x2a6bd` final-target loop 同樣達 `0x1c75e`，並藉 `sub_2b659` event 扣 MP。故 engine 仍 bounded 支援 IDs0..12；renderer/effect visual 不推論相同。
- 2026-07-26 native command IDs13..16 healing closure（official IDA 9.4 ASM）：jump-table `0x21AD9/0x21B99/0x2211C/0x22153` 只換 ID `13/14/15/16` 與演出參數後跳共同 `0x21B18`；它對 generic final target array 先跑專用 indexed presentation `0x1C4CC/0x1C2DA`，再 `0x1CA89(actor,id)` 扣 MP；每個 target 的 `0x1C8ED→0x1C916` 以 record `u16+0` 算 `floor(amount*0.9)+floor(rand()%100*amount/1000)`，將 runtime `+0x40` 加至最多 `+0x42`，並以 `0x1E0DB(...,0x69,target)` 顯示數字。IDs13..16 因此是 per-target HP restore，不是 IDs0..9 numeric damage route；renderer/UI/SFX 及 remake resolver 未接，維持 fail-closed。
- 2026-07-26 native commands 17..19 modifier-writer closure（official IDA 9.4 ASM）：ID17 `0x226EA→0x22721` 以 `+0x22` gate，ID18 `0x2282F→0x22866` 以 `+0x23` gate，兩者在零值時設 `rand()%4+2` duration，對 `+0x48/+0x4a` 加 `__CHP(value*0.15+1)` 的 **toward-zero** increment；`0x377A4` 暫存 control word、設 RC=11b 後 `frndint` 再 restore。ID19 `0x22960→0x22997` 以 `+0x24` gate，設 duration後對 `+0x4c/+0x4e` 各加 15。這與 `0x1b750` 的 derived AP/DP/HIT/EV writers 一致，故撤回 doc35 將 `+0x48..+0x4e` 誤稱螢幕座標／bounding box 的 assertion；status names、tick/decrement、UI、remake integration 未閉合。
- 2026-07-26 native commands 20..21 flag-clear/restore closure（official IDA 9.4 ASM）：ID20 `0x22A85` 與 ID21 `0x22BC6` 只換 command ID/flag offset，均進 `0x22AA8→0x22AF6`；MP debit 用 command20/21 record，final target 的 `+0x25/+0x26` 為零則失敗 display，非零時 `0x1C916(target,10)` 回復 HP 並清該 flag。兩 status 名稱、UI 與 integration 未閉合；ID22 是不同 `+0x27` application route，不可混稱 cure。
- 2026-07-26 native command 22 application closure（official IDA 9.4 ASM）：`0x22BE1→0x22CDA→0x22D1B` 對 final target 先檢查 `+0x27==0`、class `+0x20` 非 `0x19/0x1a` 及 `rand()%100<0x32`；三者成立才以 `0x1C81F(target,10)` 固定扣 10 HP、display damage、寫 `rand()%4+2` 至 `+0x27`，否則只失敗 display。未命名 status、tick/UI/remake integration，禁止以 raw offset 臆測。
- 2026-07-26 transient command duration lifecycle correction（official IDA 9.4 ASM）：`0x1A30B` 傳 raw selector 給 `0x1A866`；第二段 sweep gate 是 `record+6 == selector` 且 `(record+5 & 1)==0`，不可宣稱 active/alive/same-camp normalized equivalence。通過 gate 後對 `+0x22..+0x27` 六個 duration byte 各自 decrement，任一歸零才顯示 expiry feedback 並呼 `0x1B750` 重算 derived AP/DP/HIT/EV。故 command17..22 寫入的 `rand()%4+2` 是 camp-phase duration，非每 action/frame timer；status names/icons/UI/remake state未接。
- 2026-07-26 native command 23 relocation boundary（official IDA 9.4 ASM）：ID23 走 `0x1CFF0` command-`0x17` special selector；`0x2218A` 用 record23 扣 MP，對 selected unit 兩次呼 `0x22253`。C stack ABI 釘住第一次 direct write unit `+0/+1=0xff/0xff`（原座標離場演出），第二次 direct write `+0/+1=0x51CF9/0x51CFD` cursor globals（入場演出）。這證實直接格座標 relocation、非 path movement；落點 legality/UI/camera/renderer 未閉合。
- 2026-07-26 native commands 25..27 closure（後續 RNG/damage 勘誤）：
  ID25 清 raw bit；ID26/27 復用 ID22 `0x22d1b` 到 `+0x25/+0x26`。
  當時把 argument10 誤寫成 fixed-10 damage；後續完整 trace 證實它是
  `0x1c81f` base amount，另耗 damage RNG，實際9 HP，再耗第三 RNG
  寫 marker。status names/UI 仍未接。
- 2026-07-26 raw transient data-layer correction：`battle.Unit.NativeTransient[6]` 現保存原始 `+0x22..+0x27`，並以 optional `NativeRecordByte5/6` 保存 gate provenance；`State.TickNativeTransientsRaw(selector)` 缺 raw gate 時 fail-closed。舊 normalized `TickNativeTransients(camp)` 不再猜測 selector 映射；不混用 legacy normalized Buff/Poison/Seal/Paralyze timers，也尚未自動做 `0x1B750` equipment/stat recompute。
- 2026-07-26 native command25 engine slice：`State.ExecuteNativeCommand25` 依 `0x22C04` 只接 generic two-stage targets、record25 MP debit 與 final target `Acted` clear-if-set；wrapper-success 後才設 actor acted。缺 raw book/flags、invalid confirmation 或 MP不足均在 mutation 前拒絕。UI/renderer/message feedback 未接，且不重用 normalized CastArea。
- 2026-07-26 native commands22/26/27 engine slice（後續已修正）：
  `ExecuteNativeCommandApplication` 接 two-stage targets/MP；marker/class/
  50% gate 通過後，現正確消耗 damage RNG 並套 base10→actual9 HP，
  再消耗第三 RNG 寫 marker。失敗 target 不 mutation，command transaction
  仍耗 MP/完成 actor；不映射 legacy status 名稱。
- 2026-07-26 native commands20/21 engine slice：`State.ExecuteNativeCommandClearRestore` 只接受 IDs20/21，依各 record generic targets/MP debit；final target 的 `+0x25/+0x26` 非零時，以 **record10** amount 走 `0x1C916` formula/HP cap 後清同 raw byte，empty flag仍是 successful command completion。`ApplyNativeCommandRestore` 分離 rolled display amount 與 actual HP delta；不映射 legacy named status，UI/renderer未接。
- 2026-07-26 native commands13..16 engine slice：`State.ExecuteNativeCommandHeal` 只接受 IDs13..16，使用自己的 raw record generic targets/MP debit/`0x1C916` restore-cap/actor completion；與 IDs20/21 借 record10 的 clear/restore route 分開。專用 indexed animation、SFX/message/UI未接，保持 fail-closed。
- 2026-07-26 native damage route correction for IDs0..12（official IDA 9.4 ASM）：`0x21548` 開頭 `0x1CA89(actor,id)` 扣 MP，尾端 final target loop 直接 `0x1C75E(target,id)`；ID10 `0x21527`、ID11 `0x2185F`、ID12 `0x21A9E` 皆可閉合 numeric core，ID9 亦 direct `1CA89→1C75E`，IDs0..8 則是 `2A6BD→2B659/1C75E` direct family。`ExecuteNativeCommandDamage` bounded range 恢復 0..12；presentation/SFX 不推論相同。
- 2026-07-26 official IDA xref export unblocked：使用者合法 `ida-pro-9.4-ver2` local Docker image 以 `/tmp/fd2-ida-analysis` 保存 IDA sidecar、repo 只接收 address-only report。IDA 9.4/Hex-Rays batch 已完成並輸出 `docs/data/ida/fd2_xrefs.json`；`tools/ida_export_fd2_xrefs.py` 移除 IDA 9.4 不存在的 `ida_xref.get_xref_type` 呼叫。repo Dockerfile 仍不含 license；為 IDAPython 安裝的授權 image overlay 只留本機 Docker cache、不可提交。report 僅驗 call graph，不能獨立命名 handler／status 語意。
- 2026-07-26 modifier MP ABI correction（official IDA 9.4 ASM + raw rows）：jump-table ID17 `0x226EA` 與 ID18 `0x2282F` 都呼叫 `0x1CA89(actor,0x12)`；records17/18 的 seven-byte raw row 完全相同（`00000004020501`），所以可觀察 debit 相同，但不可泛化成 handler 一律傳自身 command ID。ID19 `0x22960` 明確傳 `0x13`。modifier writers／duration 的既有結論不變；這只撤回會污染後續 executor 的 transaction 便利假設。
- 2026-07-26 ID24 player/AI dispatcher split（official IDA 9.4 ASM + Docker Capstone）：撤回「`funcs_1541f[24]=0x22153` 因此玩家 ID24 是 ID16 heal alias」的錯誤捷徑。該 table 只在 `0x15311` AI／自動 dispatcher 使用，確實會把 ID16 傳入共同 heal tail；玩家 `0x1cff0` 則對 `0x18` 直入 `0x2a6bd→0x276ec`，不經此 table。後者的 `0x2b659` 明確以 `0x1ca89(actor,0x18)` 扣 record24 MP，再以 `trunc(actor.+0x48*15/10)-target.+0x4a` 呼 `0x1c81f`；多段畫面先暫存並復原 HP，最後等份扣至相同 delta。`ExecuteNativeCommand24` 已只接 final non-UI state slice；multi-hit/presentation/SFX/UI 仍未接。同步刪除 doc37 將 `0x2a6bd` 說成武器專用、與 command 無關的舊斷言。
- 2026-07-26 0x276EC family expansion（official IDA 9.4 ASM）：同一 player handler 對 ID28 選倍率20、ID29 選12、ID31（default）選18，均沿一般 two-stage selector 與 `0x2b659→0x1ca89(actor,id)`；`ExecuteNativeCommandDerivedStrike` 因此將 24/28/29/31 限定接成 state-only final delta。ID30 改走 `0x149f8`，但後續完整 push/arg trace 已關閉其 special selector：先 normal candidate confirm，再從 saved cursor 朝 confirmed cursor 走 `record+3-0x10`（ID30=4）格，X-first／同 X 才 Y、僅 native camp0；最後 `0x2a6bd→0x276ec` default倍率18。`ExecuteNativeCommand30` 已接 explicit-cursor state slice；UI、multi-hit、SFX/indexed renderer 仍未接。IDs32..35 走 `0x27fc9`。
- 2026-07-26 IDs32..35 compound-handler closure（superseded wording）：`0x27fc9` 對 ID32→`0x2111a→0x1c75e`、ID33 對每 final target 清 `+0x25..+0x27` 後以固定 800 進 `0x211a4→0x1c916`、ID34 順序呼三個 modifier writers `0x22721/0x22866/0x22997`、ID35 順序以 IDs26/22/27 呼 application helper `0x22d1b` 寫 `+0x25/+0x27/+0x26`。早期「wrapper 未見 `0x1ca89`」文字已由後續 `0x2b659` resource-gated callsite correction supersede；不可聲稱免費施放或由 remake 猜扣 MP，四 ID 仍 fail-closed。
- 2026-07-27 compound raw plan：canonical Docker Capstone 重新固定唯一 caller `0x2a7ce`（由 `0x2a6bd` selector `>=0x20` 進入），新增 `battle.NativeCompoundCommandPlan` 保存 ID32..35 helper order、ID33 inline clear offsets 與 amount `0x320`。這是 editable evidence-only plan；不執行 MP/target/transaction/UI，也不替 raw offsets 命名效果。
- 2026-07-27 expansion-document assertion audit：撤回 `17-scenario-expansion-evaluation.md` 的「原版評分式 AI 已還原、可照搬」；現以 `11` raw dispatcher/candidate/score slices、完整 AI runtime 未閉合為準。另將 `50` 的 persistence claim 限定為 remake 自有 JSON projection，明確不等於原版 `FD2.SAV` byte identity。
- 2026-07-26 legacy-status audit correction：`42` 不再把 `TickStatus`／`BuffTurns` 標成原版 status 完成。它們是可玩的 normalized approximation；native `0x1a866` 對 `+0x22..+0x27` 每 camp、逐 byte遞減、歸零才 `0x1b750` 重算，且 raw bytes 的玩法名稱/UI仍未證實。`99` 早期「反組譯研究主體完成」也降為資產 codec 的歷史快照，現況以 SDD/worklist 為準。
- 2026-07-26 ID30 cursor provenance correction：`0x115b6` confirm loop 直接以 `0x53ab1/0x53ab5` 比對 unit `+0/+1`，方向鍵 helpers亦改寫這對值，故它們是 target cursor cell、非 camera scroll。`0x1d339` 的 args 使 saved/new cursor、count 和 selector 可完整對位；同軸 `<=` 的 +Y 不是未知座標系，而是原始 equal-cursor fallback。先前「axis 未關閉、待動態 trace」已撤回；仍待的是 UI/presentation 的 E2 視覺差分，不是 state selector。
- 2026-07-26 `0x22253` renderer provenance correction：`0x11eee` 不是 FDOTHER#81 frame-selection loader，而是背景/tile redraw。boot `0x25c97..0x25cac` 明確 `0x111ba(FDOTHER,#3)`→`0x53a6d`，`0x22547` 以此 base 固定 descriptor 5→0 做六次 blit/present。再由 full stack-slot trace 排除 #81：它不進 renderer callee、只 free；待解的是 FDOTHER#6-derived local pointer decoder，而非 nested payload。`unit_present` 保持 fail-closed。
- 2026-07-26 `unit_present` compiler correction：完整 `0x22253` trace 推翻舊 six-frame schema（實際 11+6+10 presents），故 `7a3b15d` 令 handler compiler 明確拒絕該 op；不再只依賴 GUI runtime 的 error string 防止不完整 handler 被標為可編譯。Docker `go test ./internal/campaign -count=1` 通過。下一步是將三個 LMI/remap phase 的 destination/clip ABI 資料化後再設計新 schema。
- 2026-07-26 native LUT compositor / x87 closure（Docker Capstone）：撤回「`0x22046` 三次呼 `0x219ad`」及它只屬 `0x22253` 的說法。`calls 0x22046` 有六個 caller（`0x21f4c/21fe0/22388/225f6/226a1/246c7`）；其本體只兩次呼 `0x219ad`，各以同一 LUT 做圓形 scanline in-place remap，接著自身做一段矩形同 LUT remap。`0x219ad` 的 span 是 `sqrt(radius²-dy²)*scale/10`。`0x377a4`（`__CHP`）直接 `fnstcw`、將 temporary control word RC 設為 `11b`、`frndint` 後 restore，故其結果確定 toward-zero；同步修正 modifier17/18 的 increment 描述。個別 visual semantics 仍未證實，不能據此接 PNG/Ebiten approximation。
- 2026-07-26 native radial-LUT primitive：在 Docker Capstone 逐指令核對 `0x4DB9C(lut,count,in_place_pixels)` 後，新增 `fdother.ApplyRadialLUTRemap`。它只實作 `0x219AD` 的 strict `CenterY±Radius` scanline、`sqrt(radius²-dy²)*scale/10` toward-zero span、0x138 visible clip 與 256-entry in-place LUT，並有 identity/boundary/clip/fail-closed regression；不含 descriptor source、double-buffer/present、timing 或 UI，故沒有解除任何 `unit_present`／transition handler gate。
- 2026-07-26 native centered-rectangle LUT primitive：`0x22046` final pass 的 x-span/visible clip/EndY-exclusive in-place map 已獨立為 `fdother.ApplyCenteredRectLUTRemap` 並 regression。它與 `ApplyRadialLUTRemap` 刻意分開，因 native 兩個 radial calls 中間有 `0x127a9` redraw，尚未證實可合併或略過；此 commit 只完成已證實 pixel writer，沒有讓 native handler runnable。
- 2026-07-26 redraw mutation closure（Docker Capstone）：`0x127a9` 掃 object roster，對可見 entry 走 `0x127e0`；後者以 camera-relative coordinates／`0x53a61` sprite descriptor 呼 `0x4deda` 或 `0x4de56`，destination 是 `0x53a49` 的同一 indexed buffer。故它不是無害 delay 或 bookkeeping，而是 `0x22046` 兩段 radial LUT pass 中間的 framebuffer mutation；兩個 pure remap helper 必須保持分離，full compositor/UI 仍 fail-closed。
- 2026-07-26 redraw ABI expansion（Docker Capstone）：`0x4deda` 是 24×24 raw indexed RLE blitter；`0x4de56` 為其 palette-remap variant。`0x127a9` 尾端另呼 `0x129ec`，後者仍對 `0x53a49` 做 map/object overlay，不是安全可省略的 maintenance。故 full adapter 的下一個資料邊界是 `0x53a61` descriptor／兩 RLE streams／layer order，不能以兩個 LUT primitive 假冒完整 native frame。
- 2026-07-26 FDICON pointer-table / cycle correction（Docker Capstone）：撤回把 `0x53a61` 留為未知 descriptor 格式的待辦，也撤回「unit+4 直接是 frame」。`0x11019(key,resource)` 配置／填入每 raw key 的 12-pointer block，回傳 slot；`0x127e0` 的 renderer index 為 `slot×12 + pose×3 + cycle`。unit+4 僅乘 pose direction offset 進 screen placement；+4=0 選 global idle `0x3c0b`，非零選 moving `0x3c07`，cycle3→1，`+0x26!=0` 強制0並加入全域繪製偏移。這正是既有 FDICON 24×24 map sprite path；palette-remap selector 已閉合為 `unit+5 bit7`，而 `0x129ec` layer order 仍不可誤連到 FIGANI battle archive。
- 2026-07-26 FDICON codec correction：新增 `internal/fdicon`，以 24×24 B24 offset table + four-mode RLE decode 保存 indexed pixels/transparent mask；`BlitAt` 是 raw `0x4deda`，`BlitPaletteBand` 是 `0x4de56` 的固定 `(index&7)+0x18`。撤回「0x4de56=256-entry LUT」錯誤；Docker regression 覆蓋 fixture 的 run/dither/transparent/raw/band 與原始 `FDICON.B24` 1680 entries。這是 asset/compositor primitive，未主張 native roster scheduling 或 UI 已完成。
- 2026-07-26 FDICON selector correction：`Bank.SpriteFor(key,pose,cycle)` 實作一個已解析 raw key 的 `key×12 + pose×3 + cycle` lookup；native renderer 本身先以 `unit+2` cache slot 選 12-pointer block。它拒絕越界 pose/cycle；`NativeFrameIndex` 實作已證實的 global idle/moving cycle selector。撤回先前「runtime +4 frame」錯誤，+4 是 placement offset；現有 battle `Fig`/`Dir` 不因此自動構成完整 native renderer state。
- 2026-07-26 FDICON blit-branch closure：`0x127e0` 的 `test unit+5,0x80` 直接決定 0x4deda（bit clear raw）或 0x4de56（bit set palette band）；不是 camp/LUT selector。`fdicon.BlitForNativeFlags` 以同一 raw bit 接線並 regression。layer order、unit+0x26 的玩法名稱及完整 renderer schedule 仍待。
- 2026-07-26 FDICON placement closure（Docker Capstone）：`0x127e0` destination 的已閉合 byte equation 為 `0x75d8 + (Y-cameraY)*24*0x1c8 + (X-cameraX)*24 + unit[+4]*dirByteOffset`，其中 pose 0/1/2/3 的 offsets 為 `+0x720/-4/-0x720/+4`。僅 `unit+0x26!=0` 時再加 native toggled 0/1 pixel shift。新增 `fdicon.NativePlacementOffset` regression；它是 buffer ABI，不是未證實的 GUI coordinate/layer adapter。
- 2026-07-26 foreground occlusion closure（Docker Capstone `0x129ec..0x12c0c`）：`0x127a9` 的 unit sprites 完成後，`0x129ec` 對每個可見 runtime unit 固定重畫 `(x,y)` 與 `(x,y-1)` field overlay；`unit+4!=0` 再依 pose 補畫鄰格（0:`y+1`、1:`x-1`、2:`y-2`、其餘:`x+1`）。`0x12ac6` bounds-check 後由 field entry/descriptor flags 判定 bit7，將 tile 寫到 `0x53a49+0x8088+(y-cameraY)*24*0x1c8+(x-cameraX)*24`；entry byte+3=`0xff` 走 raw `0x4deda`，其他值走 `0x4dd52`。後者已由 direct trace 關閉為同一 24×24 four-mode RLE 的 caller-supplied 256-entry indexed-LUT blitter；因此它是 foreground-terrain occlusion、非 bookkeeping。尚未命名 tile descriptor/source/palette-table selection 或接 GUI。
- 2026-07-26 FDSHAP foreground provenance（Docker Capstone `0x10937..0x1097f`）：map loader 讀 selected map resource selector 後，先以 `2*selector` 呼 `0x111ba` 填 `0x53a5d`，再以 `2*selector+1` 填 `0x53a69`；它們正是 `0x12ac6` 用的 image descriptor base／tile flag table。故 foreground source/flag 來自 selected FDSHAP pair，不是 FDICON 或臨時 renderer allocation。`0x12ac6` 的 alternate palette pointer 仍由 FDOTHER#3 descriptor base `0x53a6d` 的 map-selected entry 取得；exact descriptor offset/entry meaning 尚待。
- 2026-07-26 foreground LUT selector correction：撤回「0x12ac6 palette entry selection 未知」。direct trace `0x12b80..0x12b98` 同樣以 `0x51a97[0x53c1f]` 選 FDOTHER#3 descriptor entry，與 `0x11eee` 的 default 20-phase terrain LUT selector 相同；已由 `NativeTerrainLUTIndex`／LUT bank loader 表達。尚待的是 FDSHAP descriptor offset 的高階命名與 complete scheduler，不是 LUT choice。
- 2026-07-26 foreground descriptor index closure：`0x12ac6` 對 terrain control bit `0x80` 才畫；bit `0x08` 加 `2*flip`，再取 FDSHAP offset-table **`index+1`**（`base+0x0a`，不同於 terrain base+0x06）。新增 `fdicon.NativeForegroundFrameIndex` regression，故撤回 descriptor offset 未閉合說法；完整 foreground call schedule/UI 仍待。
- 2026-07-26 foreground LUT blitter closure：`0x4dd52` 與 terrain `0x4dcc6` 不可混用：兩者皆 LUT-map source，但 `0x4dd52` 的 mode3 保留 destination，`0x4dcc6` 才 LUT-map destination。新增 `fdicon.Sprite.BlitLUTTransparent` regression 固定前者語意。
- 2026-07-26 foreground cell compositor：新增 `Bank.BlitNativeForegroundCell`，組合 `0x12ac6` bit80 gate／index+1 selector 與 raw 或 `0x4dd52` transparent-LUT branch；fixture 驗 source LUT-map 與 mode3 preservation。完整 unit-adjacent scheduler 仍待。
- 2026-07-26 FDSHAP codec correction（Docker read-only scan）：以 native `0x4deda` 的 24×24 four-mode token ABI 掃 `FDSHAP_000`，288/288 tiles 恰好解至 576 pixels，實際 modes 為 `[0,2,3]`。因此撤回「FDSHAP 僅有不透明 bg-RLE」：mode3 是 transparent span，舊 decoder 在其後可能偏移。`tools/render_map.py` 已改用相同 four-mode decoder 並把 transparent 保留為 index0；這不等於已關閉 native 多層 foreground composition。
- 2026-07-26 FDSHAP alpha bridge correction（Docker Capstone `0x11eee/0x4dcc6`）：codec 改回傳 pixels 與獨立 opacity mask；`export_engine_assets.py` 的 RGBA sheet 只表達 raw `0x4deda` preview（map0 alpha extrema=`(0,255)`，opaque palette index0 仍 alpha255）。撤回「mode3 一律是 alpha」：terrain `entry+3==0xff` 才 raw；其他值走 `0x4dcc6`，其 opaque source 經 caller LUT，而 mode3 讀既有 destination 再 LUT-map。已把 FDFIELD event high byte 匯出為 `native_tile_blit_modes`，故 future indexed adapter 有 raw selector、不能以 alpha 冒充 LUT compositing；`0x129ec` scheduler 未完成。
- 2026-07-26 native terrain frame selector（Docker Capstone `0x121cb..0x12261`）：`0x11eee` 對 visible composition cell 的 tile ID（word mask `0x3ff`）讀 FDSHAP control byte，按優先序選 frame：flag `0x08` 加 `2*[0x53a40]`；否則 flag `0x10` 加 truncating `[0x53c0b]/2`；否則 flag `0x04` 加 `[0x53a40]`；否則 base。然後 descriptor `+6+index*4`、entry byte+3 分 raw `0x4deda` 或 LUT `0x4dcc6`。只記 raw bits，禁止猜水流／火焰名稱。
- 2026-07-26 terrain frame selector primitive：新增 `fdicon.NativeTerrainFrameIndex`，嚴格收 tile `0..0x3ff`、flip `0/1`，保留 bit8→bit10→bit4 priority 與 signed toward-zero `/2`；Docker regression 覆蓋 negative cycle 與越界 reject。它只輸出 descriptor index，沒有偷接 map scheduler。
- 2026-07-26 native terrain cell compositor：新增 `fdicon.Bank.BlitNativeTerrainCell`，組合 `0x11eee` descriptor selector 和 entry byte+3 raw `0x4deda`／LUT `0x4dcc6` branch；Docker fixture 直接驗 raw 與 LUT mode3 destination remap。它不持有 camera loop、phase selector 或 foreground redraw，故完整 renderer 仍 fail-closed。
- 2026-07-26 foreground redraw schedule primitive：`fdicon.NativeForegroundRedrawCells` 將 Docker trace 的 `0x129ec` call order 固定為 `(x,y)`、`(x,y-1)`，且僅移動時補 pose 0→down、1→left、2→two-up、其他→right；off-map 座標刻意不濾除，因 native `0x12ac6` 才作 bounds/visibility gate。regression 覆蓋 stationary、四個 branch、default branch 與負座標；尚未猜接 roster 或 GUI。
- 2026-07-26 foreground roster-gate correction（Docker Capstone）：撤回「每個 visible unit 都進 `0x129ec`」的過度簡化。loop 先以 `0x1f183(slot)` 過濾，後以 `0x3453e(slot)` inactive 過濾；`0x1f183` 的 raw predicate 是 **`unit+7==0x1c`** 放行，否則 class `0x13` 或 race `4/5` 回 true 而跳過。亦撤回剛才誤稱 `unit+7` 為 group 的說法：map sprite group 是 `unit+2`。`NativeForegroundRedrawEligible` 已 regression 固定兩 gate；數值暫不命名，runtime roster／GUI adapter 繼續 fail-closed。
- 2026-07-26 scripted foreground caller closure（official IDA 9.4 ASM）：`0x129ec` 還有 `0x1366a` caller，故撤回它只屬於 `0x127a9` steady redraw 的隱含說法。`0x1366a` 在 step loop 的 base `0x11eee`、逐 slot `0x127e0` 後呼 `0x129ec`，之後才 `0x11eb0` 和 present/redraw；loop 會由 editable acting frame bytes 改寫 runtime `unit+3`。**更正**：106-entry step input 格式早已由 `export_acting_resources.py` 解出；未接的是其 native indexed presentation adapter，GUI 仍 fail-closed。
- 2026-07-26 native viewport-copy closure（2026-07-28 correction）：`0x11eb0` 是 row-by-row `memmove`；重讀標準 `0x11cac` caller `0x11d12..0x11d36` 的 exact args 為 dst `0xA0504`／stride320、src `0x53a49+0x8088`／stride456、**width312**、height192。舊 width320 斷言已撤回；VGA placement 是 `(4,4)` 四邊4px。
- 2026-07-26 map selected-unit HUD correction（Docker Capstone）：`0x1acf3` 在 normal redraw 的 `0x127a9` 後、`0x11eb0` 前，以 `0x12c0d(0x53ab1,0x53ab5)` 尋 active unit；兩 raw display gate `0x51aab/0x51aac` 皆需非零。它有獨立 resource/raw blit/digit pipeline，anchor 為 row157，並在 `0x53abd>5 && 0x53ab9<3` 加 `0xf2`。撤回 map HUD 等同 FDOTHER#5 full-screen battle-panel frame 的未證實說法；當時 GUI 尚保留 approximation，後續 ch01 strict indexed bridge 已取代這個歷史狀態。
- 2026-07-26 map HUD first-frame closure：`0x1acf3` 對 FDOTHER#5 base 取 `base+0x20e`，即 LMI1 descriptor #130；以 player-provided `FDOTHER_005` decode 實測為 **69×34**。這與 battle panel #22 的149×42不同，直接證實 map HUD 不是復用 full-screen panel；後續 icon/digit descriptor 與 layout 仍待。
- 2026-07-26 map HUD terrain resolver correction（Docker Capstone）：撤回將 `0x12e38` 暫稱為 unit visual resolver 的說法。它是 FDFIELD cursor-cell decoder：輸出 tile word `&0x3ff`、event byte `&0x1f` 和 selected FDSHAP 的四 control bytes；`0x1acf3` 先用此 terrain info，後才 `0x12c0d` optional lookup active unit。新增 `fdicon.NativeTerrainCursorInfoForCell` regression，完整 HUD icon/digit layout 仍待。
- 2026-07-26 native terrain AP/DP closure：重新讀 `0x1acf3` 後撤回 `0x51a12/0x51a2a` 為 figure direction/pose 微調的錯誤斷言；它們由 `0x12e38` 輸出的 FDSHAP control byte+1 索引，值為 0→(+5,0)、1/5→(0,0)、2/3→(-5,+10)、4→(-5,-5)。`battle.Load` 現只在完整 raw map export 驗證後導出逐格 `NativeTerrainMoveCodes`，`TerrainAPDPPct` 優先使用這個原版來源；`Cost` 僅保留給舊／不完整 map 的相容 fallback。回歸覆蓋全部六碼和 loader。
- 2026-07-26 map HUD geometry closure（Docker Capstone）：`0x1acf3(buffer,stride)` 的 #130(69×34) frame 在 `buffer+stride*157+x`，terrain descriptor raw blit 在 `+6`；AP／DP 分別在 `stride*8+0x2b`、`stride*19+0x2b` 經 `0x1aeb1` 繪製。後者依值正負選 LMI1 directory decimal #83／#84、取絕對值，然後走 `0x187d6` native digit path；兩 resource 的 artwork semantic 尚未閉合，不能稱 sign。`x` 是初值1的 raw static anchor，已見條件分支只選 0xf2 或1；暫不替 globals 命名。optional unit icon/HP 與 Ebiten adapter 仍待，現有 #22 map HUD 仍是 approximation。
- 2026-07-26 map HUD codec closure：追 `0x4e63d` 確認它是 four-mode transparent RLE，而 #130／decimal #83／#84 正由 `0x1acf3/0x1aeb1` 走此 path；不能使用 `ParseLMI1` 的 `0x4e916` codec。新增 strict `fdother.ParseLMI1FrameEntry`／`DecodeLMI1FrameResource`，archive regression 對三 entry 驗 geometry（69×34、44×12、45×12）並在 row157 destination decode。LMI1 同容器 codec 混用維持顯式，未推論其他 entry。
- 2026-07-26 HUD LMI1 index correction correction（Docker Capstone `0x1aeb1`）：撤回剛才把 native `mov ebx,0x83/0x84` 誤讀為 decimal index 的修正。它們是 literal hex immediates，真實 #5 entries 是131=`6×7`、132=`6×5`；因此 Go four-mode decoder 對原 `0x83/0x84` 可通過，44×12/45×12 的 decimal entries 與此 caller 無關。SDD/test 已回復 native index，並刪除「HUD decoder gap」的錯誤說法。
- 2026-07-26 map HUD optional-unit closure：`0x1ae4d` 的 icon index=`unit+2*12+rawState`，唯 rawState 3 改為1；`0x4deda` blit destination=`buffer+stride*5+6`。接續 `unit+0x40/+0x42`（current/max HP）送 `0x1875d`，destination=`buffer+stride*21+9`、mode3。新增 `fdicon.NativeMapHUDUnitFrameIndex` regression；state/global 的高階名稱未命名。
- 2026-07-26 strict map HUD destination closure：新增 `fdicon.NativeMapHUDLayoutFor`，以 `0x1acf3` 的 native 456 stride＋raw anchor 計算 frame(row157)、terrain(+6)、AP(row8,col43)、DP(row19,col43)、unit(row5,col6)、HP(row21,col9) 六 byte offsets；只接受 frame 完整落在 320-pixel viewport 的 anchor。它是 renderer input contract，不自動接到 Ebiten。
- 2026-07-26 spell-presentation scope correction：撤回「`0x28784` 不讀 spell id，因此 remake 角色施法動畫已完整等同原版、不需特效」的過強結論。可證的只有沒有 spell-id→另一段 FIGANI selector；`0x2a6bd` 的 command-specific DATO/indexed presentation、SFX、命中與多段畫面分支尚未閉合。doc37／worklist 改以局部 FIGANI hand-gesture adapter 描述，保持其他 presentation fail-closed。
- 2026-07-26 generic command BG selector closure（Docker Capstone）：`0x2a6bd` 以 final target count/array 呼 `0x2b5e1`；後者倒序掃 target unit、`0x12e38` decode 後，僅 raw `0x1f183` gate 不通或累積 selector=0 時採 control byte+2，最後交 `0x111ba("BG.DAT", selector)`。新增 `fdicon.NativeCommandBackgroundSelector` regression 固定規則。撤回 generic command ID 直接選 BG resource 的暗示；raw selector 的高階場景／地形語意仍待。
- 2026-07-26 BG archive codec closure：原始 `BG.DAT` 為 LLLLLL archive，#0/#1/#2 各是 320×100、直接交 `0x4e63d` 的 four-mode single frame（非 frame-directory）。新增 `fdother.DecodeArchiveSingleFrame`，player archive regression 對三層 geometry 與 transparent indexed decode 通過；selector/layer schedule 不從 decoder 推論。
- 2026-07-26 battle-animation assertion audit：修正 doc06 舊摘要。`0x29164` 不是狀態欄而是 figure+TAI 台座淡入；狀態欄是 `0x2a289→0x18c6d`。撤回「全螢幕繪圖已完整反組譯」與 `figure X=unit+0x40`；後者是 current HP。保留已證實 anchors/codec，command presentation、layer schedule 維持未閉合；當時尚未閉合的 figure displacement 已由下方 `0x2935b` 條目取代。
- 2026-07-26 combat-result resolver correction（Docker Capstone）：撤回將 `0x29f72` 稱為 lunge／接近內插器的錯誤。它讀攻 `+0x48/+0x4c`、守 `+0x4a/+0x4e` derived stats、守方 HP、item record，並在 raw gate 下套 `0x51a12/0x51a2a` terrain AP/DP、RNG，輸出 hit/crit/damage flags（`out+0..+0x10`）與 final damage（`out+0x14`）、`0x53ec8` presentation value。沒有已證實 screen-coordinate output；figure displacement 需另追 `0x2935b` caller/frame metadata。
- 2026-07-26 figure displacement closure（Docker Capstone）：`0x2935b` 以 `frameIndex*4+8` 取 animation descriptor 的 relative frame pointer，讀 frame header `u16 +0/+2` 作 X/Y，payload `+9` 交 `0x4e63d`。故逐幀 swing／前傾／後仰是 frame metadata，不是 `0x29f72` 或 unit runtime stat；(164,157) 僅是特定 figure/台座 caller anchor。worklist 將 displacement source 收為完成，保留 caller schedule／攻守配對為未閉合。
- 2026-07-26 FIGANI placement export regression：新增 `cmd/fd2/TestFIGANIMetaMatchesNativeFrameHeaders`，逐一 decode player `FIGANI.DAT` 中 `meta.json` 的 22 個 resources，對每個 frame 比對 header signed X/Y。實跑通過；runtime PNG/meta bridge 因此可驗證為 native placement data，非手動 layout。
- 2026-07-26 generic command renderer schedule closure（Docker Capstone）：`0x523b9` 的 `funcs_2ac25` 是 command-indexed handler bank，ID0=`0x26152`。`0x2a6bd` mode0 取 handler step count，640-stride loop per step 依序 mode2、`0x11eb0` 320×200 present、`0x17aa9(1)` tick、mode1，cleanup再 mode4。`0x2b9a1` 讀 frame descriptor byte+6 delay，推 `0x540fc/0x540fd` subframe counters/reset。這閉合 generic schedule，command-specific artwork/effect 意義仍待。
- 2026-07-26 IDs32..35 player MP-path closure（Docker Capstone + archive）：`0x27fc9` 唯一由 `0x2a6bd` 呼叫，並在 `0x28189` 進共同 `0x2b659`；後者唯一 `0x1ca89` site 以 FIGANI container header `byte+4==1` gate。command-learn row19 的 IDs32..35 可由 player group4..7 optional class19 與 group20 initial class19 得到；其 `group*3+1` resources #13/#16/#19/#22/#61 的 byte4=`2/2/2/5/5`，archive regression 固定此結果。故這些玩家可達路徑繞過**已知** debit sink（仍有 selector MP gate），撤回「每個 spell gate 都代表 deducted」暗示；但 AI／其他 group、未知 writer、compound effects/UI仍未閉合，engine 維持 fail-closed。
- 2026-07-26 visual-selector field split（Docker Capstone）：`0x127e0` 直接讀 `unit+2`，以 `×12 + pose×3 + cycle` 選 FDICON map sprite；`0x287b5` 則讀 `unit+7`，`0x2884c` 以其 `×3` 載 FIGANI。撤回 runtime 註解中「unit+7 就是 FDICON group／同一欄」的錯誤；已驗證 roster 中兩 visual ids 可相等，並不使 raw ABI 等價。SDD/worklist 將 map／battle selector 明確分開，待 exporter 保存兩欄後才能宣稱所有 runtime unit 的 battle visual 完整。
- 2026-07-26 battle visual selector bridge correction（final）：完整 stack-slot trace 更正上一輪的誤讀：`0x10d7f` 將 FDFIELD `b1` 放入 local `+8`，`0x10e3a` 以它呼 `0x4e4d1`，`0x10ef8` 再直接寫 runtime `unit+7/+8`；故 FDFIELD `b1→unit+7` 已閉合。`export_units.py` 恢復輸出 `battle_fig=b1`，舊 JSON 才 fallback `fig`。map `unit+2` 的 source 仍未閉合，兩 selector 不可 alias。
- 2026-07-26 visible terrain pass ABI closure（Docker Capstone `0x11cac`）：standard redraw 以 `0x11eee(buffer+0x8088,0x1c8,13,8,camX,camY)` 先畫地形，接 `0x122dc` range overlay、`0x127a9` unit/foreground layers。新增 `fdicon.Bank.BlitNativeTerrainRegion` 作純 row-major terrain pass，輸入 raw cells/control table/origin/LUT、bounds fail-closed；不將後續 overlay schedule 偷接入。
- 2026-07-26 terrain renderer data export：`export_engine_assets.py` 新保留 paired FDSHAP terrain raw records 為 `native_terrain_control`，並與 FDFIELD event high-byte `native_tile_blit_modes` 同出。map0 實跑輸出 576 modes、1200 control bytes，正好供 native terrain region adapter；normalized `cost` 不再被誤作 renderer input。
- 2026-07-26 terrain runtime data bridge：`battle.Load` 現將 map JSON native terrain modes/control 保存至 `State`，僅在 dimensions、cell count、4-byte control alignment、tiles 的 10-bit/control bounds 全吻合時接受；不完整輸入 fail-closed 留 nil。Docker battle regression 覆蓋 accepted data；現行 PNG/Ebiten draw 未被偷換。
- 2026-07-26 LUT tile blitter primitive：`0x4dcc6` exact loop 已落在 `fdicon.Sprite.BlitLUT`。Decoded Sprite 新保留 mode3 `RemapMask`；source opaque pixel 寫 `lut[source]`，mode3 讀 destination 再 `lut[destination]`，mode1 dither holes 不寫。Docker regression 覆蓋 fixture、原始 FDICON decode、short LUT reject；這是 renderer building block，map LUT selector/frame scheduler/foreground schedule 未接。
- 2026-07-26 FDOTHER#3 LUT input closure：新增 `fdother.ParseLUTBank`／`DecodeLUTResource`，把 #3 LMI1 directory 解析為 23 個獨立 256-byte tables，而非 width/height UI LMI1 cells。Docker regression 讀 player-provided `FDOTHER.DAT` #3、驗證 count=23 與全表長度；可供 `0x4dcc6` primitive 使用，但 selector/palette schedule 不由 loader 猜測。
- 2026-07-26 terrain LUT selector closure（Docker Capstone data `0x51a97`）：`0x11eee` default runtime phase `0x53c1f` 0..19 查表取得 FDOTHER#3 LUT index `[0,1,2,3,4,5,6,7,8,9,10,9,8,7,6,5,4,3,2,1]`。新增 `fdother.NativeTerrainLUTIndex` bounded regression。`0x1215a` 的 explicit override 仍只保存 raw value，未命名／未視覺化。
- 2026-07-26 map-selector assertion audit：完整 `0x10c50→0x11019` trace 顯示 `unit+2` 是 resource-aware FDICON pointer-cache 的回傳 slot；它並未證明 `unit+2=character id/portrait`。撤回 doc31 將少數素材/roster 值相同升格為全域 identity、以及由此推導敵方與轉職 map group 的說法。保留 `unit+2×12+pose×3+cycle` 公式和 `b1→unit+7` battle path；`fig` 保持 compatibility approximation，map source 待閉合。
- 2026-07-26 map-selector cache ABI refinement：`0x10c50` 先取 FDFIELD `b0`，以其為 `0x11019` key、並傳 caller resource；`0x11019` 對 `(key, resource)` 找／建 12-pointer FDICON block，回傳 cache slot 寫 `unit+2`。故 map selector 不是 FDFIELD 的直接欄位 copy；剩餘問題是 resource identity 與 key 對 archive index 的解碼，不是再猜角色 id。
- 2026-07-26 player-party FDICON source split（Docker Capstone）：`0x1088d` 的 player loop 在 `0x10a77` 先把 `[0x53bf7]` 的 persistent 0x50-byte record 複製到 battle roster，隨即讀**已複製 record 的 `+7`** 作 `0x11019` key，並於 `0x10aa2` 寫回傳 slot 到 runtime `unit+2`。這與 scripted spawn 的 `0x10c50`／FDFIELD `b0` 是兩條 source path；只能共用 cache ABI，不能以 legacy `fig` 或任一方欄位替代另一方。doc31 已刪除把錯誤結構 offset、FDICON/portrait identity 混入 runtime roster 的舊表。
- 2026-07-26 official IDA bootstrap topology：授權 IDA 9.4 的 address-only report 定義 `sub_10010=0x10010..0x10620`（callers `0x1a251`、`0x26130`）與 `sub_1088d=0x1088d..0x10b4e`（callers `0x205ff`、`0x25870`、`0x2c437`）；`0x10a77` 正在後者內。這交叉驗證 save/battle/finale 的 roster bootstrap 共用同一 selector-initialization routine，但不增加 `record+7` 的未證實高階名稱。
- 2026-07-26 player map-key writer closure（Docker Capstone）：`JOIN` constructor `0x112a5(join_id)` 建 persistent `[0x53bf7]+count*0x50` 時，依序寫 `+7=join_id`、`+8=join_id`；`0x33499` 已以 `+8` 做 character-ID roster lookup。再接 `0x10a77` 的 copied `+7→0x11019`，可證 fresh joined player 的 raw FDICON key 等於其 character ID。範圍僅限這個 writer；絕不可倒推 FDFIELD、NPC、portrait 或 general `fig` 的 identity。
- 2026-07-26 player map-key mutation audit（Docker Capstone）：撤回「JOIN equality 後續未見 mutation」的隱含說法。class-change UI 在 `0x314a7` 以 selected slot 算 `0x53a45+slot*0x50`，`0x31576..0x3157a` 將 UI-selected raw target 寫到該 live record `+7`（同段重算／寫 `+0x20`）。故 fresh player 的 `+7=+8=join_id` 只是初始化；target 的高階名稱仍保持 raw。persistence ABI 則已閉合：24 個 post caller 的 `0x11506` 以 `+8` 配對並完整 copy 0x50 bytes runtime→persistent，所以經 `sync_party` 的 `+7` mutation 必保存；class-change 是否在當下立即走該 flow 待追。
- 2026-07-26 explicit map-key materialization bridge（歷史狀態；caller 尚未接線的末句已由同日較晚的 runtime selector-order bridge 取代）：`battle.Unit` 新增 optional `MapSelectorKey`／JSON `map_selector_key`，與 slot 分離；`MaterializeNativeMapSelectorSlots` 只接受 caller 給定的同 resource、native source order，先全批 preflight 0..255 raw keys，再以 `fdicon.NativeSelectorCache` first-seen 規則寫 slots。缺 key／invalid key 拒絕且不污染 cache 或 units，永不由 `Fig`／portrait 推導。Docker battle+fdicon regression 通過；此時 player/scripted caller 接線刻意仍封閉。
- 2026-07-26 player selector persistence repair（2026-07-28 用詞校正）：現行 remake 的 stable `Fig` 是 JOIN/+8 identity，不能再把它一般化為 mutable map/battle selector。fresh PartyMember 僅因 native JOIN writer 而以 Fig 初始化 `BattleFig`／raw key；church `ApplyClassChange` 對應 `0x3157a` 改 `0x31793` resolved target 的 `BattleFig`／raw key、清舊 cache slot、不動 Fig；persistent overlay 亦保留這些 split fields。Docker `battle`／`campaign`／`cmd/fd2` regression 通過；indexed renderer 仍 fail-closed。
- 2026-07-26 scripted map-key export closure（Docker Capstone + real export）：完整 `0x10d7f..0x10efc` trace 確認 FDFIELD record `b0` 先作 `0x11019` raw key、回傳 slot→`unit+2`，再同值寫 `unit+6` camp；`b1` 才寫 `+7/+8` battle selector。`parse_field.py` 保留 `native_map_selector_key=b0`，`export_units.py` 輸出 `map_selector_key`，map0 實跑30筆 keys `[0,1,2]` 逐筆等於 camp raw code。這撤回「scripted map key source 未拆出」的舊說法；尚待的是 player-first + scripted group 的共用 slot allocation order，不是角色 identity mapping。
- 2026-07-26 state selector-order bridge（歷史狀態；Scenario 尚未接線已由同日較晚紀錄取代）：新增 `State.AppendNativeMapSelectorBatch`，一個 State 持一個 per-resource `NativeSelectorCache`，只有 full raw-key preflight 成功才 allocation+append。regression 以 native order party `[9,4]`→scripted `[0,2,0]` 固定 slots `[0,1,2,3,2]`，missing-key batch 不改 state/cache。此時 Scenario 尚不自動接，等待所有 versioned map assets 含 raw keys 後才啟用，避免舊 JSON 被 Fig fallback 污染。
- 2026-07-26 versioned scripted-selector data closure：全量 `export_units.py` 比對證實會覆寫既有人工校正數值／掉落，故未採用。新增 `tools/sync_native_selector_fields.py`，它只從 FDFIELD roster 同步已閉合的 `b0→map_selector_key` 與 `b1→battle_fig`，並拒絕 roster count 或既存值衝突。已對 33 份 `remake/assets/maps/map*/map*_units.json` 寫入 3774 欄位，`--check` 全數通過；其餘 asset 欄位未由本步重產。Scenario 仍不自動 materialize native slots：尚待 mixed player/scripted order 與 resource identity 的直接接線。
- 2026-07-26 selector cache scope correction（Docker Capstone）：撤回「`0x11019` per-resource／resource-aware lookup」斷言。完整 tail 顯示它只在全域 `0x53b17[slot]` 比較 raw key、以 `0x53bdf` 計數；archive pointer 僅在新 key 時複製十二 pointers。`0x10a25`（player）及 `0x10b69`（scripted）都以 `rb` 開啟同一 `FDICON.B24`。因此 cache 是 battle construction session 的 global raw-key cache，非 `(key,resource)` cache；SDD/worklist/code comment 已改正。尚待的是將已證實 party-first→scripted order 接到 GUI indexed renderer，不是 resource identity。
- 2026-07-26 runtime selector-order bridge：`spawn_party` 現以 `AppendNativeMapSelectorBatchOrLegacy` materialize fresh-player keys；`AppendGroup` 在 cache 已啟用時以同一 batch API materialize FDFIELD groups，故 party-first→initial/reinforcement group 的 first-seen slot order 被實際保留。若可編輯來源缺 key，State 保留 legacy unit append 並記錄 `NativeMapSelectorError`，全場 native key lookup fail-closed。戰場 PNG/Ebiten draw 可由 validated `slot→key` 選 FDICON group；story actors 保持 legacy Fig。這只閉合 selector adapter，並未接 native indexed buffer/palette/layer/HUD。
- 2026-07-26 steady indexed unit-layer closure（Docker Capstone `0x127a9..0x12975`）：新增 `fdicon.BlitNativeUnitLayer`。它先套 `0x3453e` inactive gate、native camera bounds、`+4` pose direction offset／`+0x26` pixel shift、idle/moving cycle，再由 global cache slot 解 raw key 選 12-frame B24 block，依 `+5 bit7` 做 raw/palette-band blit。API 在全層 selector/destination preflight 後才寫 indexed buffer；foreground `0x129ec`、HUD、viewport present 仍為分離 adapter，未宣稱 full renderer。
- 2026-07-26 steady indexed foreground-layer closure（Docker Capstone `0x129ec..0x12c08`）：新增 `fdicon.BlitNativeForegroundLayer` 與 pointer-style `BlitLUTTransparentAtOffset`。它保留 eligible roster 順序、本格／上格／motion-neighbour schedule、`0x12ac6` camera/foreground descriptor／`0x8088` destination、raw vs `0x4dd52` LUT-transparent branch；先預檢再寫。supplied map 外座標明確 fail-closed skip，不能解讀為原版 memory 語意。scripted `0x1366a`、range、HUD、VGA/Ebiten adapter 尚未接。
- 2026-07-26 ch14 dynamic dialogue closure（Docker Capstone `0x33499..0x3359f`）：`0x33499(id)` 明確掃永久 `[0x53bf7]` 名冊的 record `+8` 並回傳 1/0；`0x334f3..0x334f7` 將 `roster_has(12)` 反相後乘 3，所以 FDTXT_015 base 是「有 char12→0，無→3」，後續 `+1/+2` 給出兩條固定三段對話 `0/1/2` 與 `3/4/5`。新增 editable `if roster_has(char_id:12)`、map14/80-slot/FDTXT_015/acting48 binding 與 compiler/runtime regression。runtime 只查 permanent `partyMembers`；缺永久名冊直接報錯，不從暫時場上 actors 猜分支。舊 2026-07-20「待 control-flow mapping」條目已由此取代。
- 2026-07-26 ch14 post dynamic closure（Docker Capstone `0x239bd..0x23a09`）：同一 `roster_has(12)` 後 `xor al,1; add al,0xc`，故 FDTXT_015 post string 是有 char12→#12（凱麗支線）、無→#13（貝克威支線），各 12 句。`ch14_post` 現為 editable branch，並保留原順序 `dialog → sync_party → JOIN15 → set_chapter15`；campaign 的 postbattle_ch15 接它後仍進 town_ch16，不把戰後補給跳成下一戰。
- 2026-07-26 ch16 pre conditional-spawn closure（Docker Capstone `0x335aa..0x335d5`）：`0x33499(18)` 後 `test eax,eax; jne shared_tail`，僅 absent arm 呼 `0x10b4e(1)`。`ch16_pre` 已改為 `if roster_has(char_id:18) { } else { spawn group1 }`，並接 map16/60-slot/FDTXT_017 到 story_ch17。compiler 的 branch lowering 現帶入前置 LOADCH slot frontier，讓這類 conditional spawn 可驗證；merge 後仍只保守採用分支前 slots，不能暗示新增 slot 必存在。
- 2026-07-26 stale native-global assertion correction（Docker Capstone `0x10322`／`0x13d00`／ch25 post）：撤回剛才「runtime-unit pointer」的過度描述。`[0x53ad5]` 是 pointer to 0x20-byte battle-local state table：初始化複製 32 bytes，event path 以 index 寫 byte，ch25 讀 entry #12 形成 FDTXT base+5/+8。這仍推翻 opened-treasure slot table 斷言，但不授權替 table 命名。`OpenedTreasure` 僅是 remake-owned editable treasure state。
- 2026-07-26 table entry12 writer closure（Docker Capstone `0x356bc..0x35821`）：entry12=0 時，handler 依 actor class 查 item `0xd0`；無 item 顯示 #2，成功則移除 item、跑 presentation、設 entry12=1、`spawn group1`、`JOIN31`、播 #4。這是 ch25 post `base+5/base+8` 的直接 state source；仍不可把 index12泛稱 treasure 或一般 roster flag，兩條 post 演出/布局未閉合前不接 campaign。
- 2026-07-26 official IDA layout topology audit：授權 IDA 9.4 重建 `0x233c6` 為 `0x233c6..0x2345b`，並輸出 15 個 callers（含 `0x23c04` ch16 post、`0x24f11` ch25 post、`0x257b4` final post）。這交叉驗證它是共用 address-keyed layout primitive，不能把 ch16 的兩個 16-byte table 或其 special-slot/focus arguments generalize 成其他 caller ABI。SDD 明訂完整 materialized slots+camera 才可播放，缺欄維持 fail-closed。
- 2026-07-26 ch17 initial group correction：ch16 pre direct CFG 證明 group1 僅在 char18 absent arm append，而 group3 首見於 post handler `0x23bde/0x23c55`。Scenario 的 `initial_groups` 修為只 group0，新增 data-driven `initial_groups_if_party_absent:{char18,group1}`；這只修正當前 roster-backed battle state 的 OnField 初始可見性，不把它誤稱 native slot append reconstruction。
- 2026-07-26 entry12 dispatcher scope（已於 2026-07-29 被直接資料勘誤）：當時 IDA 只見八個 generic indirect xrefs，故無法證實 map25 caller；現在 FDFIELD map25 field slot2 已直接固定 selector1、座標 `(1,46)` → event61/`0x356B7`。舊「未證實 map25-local caller」只保留為歷史過程，不再是現況。
- 2026-07-26 ch27 transition/acting separation（official IDA 9.4 + isolated Capstone）：`sub_24618=0x24618..0x24754` 的 callers 包括 `0x33af1/0x33c9d` post handlers。它先以 13×8 terrain region 建 offscreen buffer，固定做九次 strip composite，最後 palette index 0..62 每步2、delay4ms 收束；四 args 是 tile/strip geometry 與 progression。撤回把 ch27 的 `0x24618` unknown 暗示為 acting 的可能，renderer 必須新增已驗證 transition adapter，否則 handler fail-closed。
- 2026-07-26 `0x24618` ABI naming correction（isolated Capstone full stack-slot trace）：撤回 binding schema 的 `source_x/source_step`、`remap_scale/remap_scale_step`、`source_y/blit_width` 命名。`0x24618` 每 pass 將 third/fourth args 作 radial radius／radius step；`0x22046` 對兩次 `0x219ad` 固定傳 scale=16，最後用 `trunc(radius*1.6)` 作矩形 pass。其常數 row range 是 `[start_y,end_y)=[0,192)`，不是 image source 或 width。schema 改為 `radial_radius`、`radial_radius_step`、`start_y`、`end_y` 並有 compiler/binding regression；indexed adapter 仍 fail-closed。
- 2026-07-26 ch26 sky-key gate closure（isolated Capstone `0x24b14/0x250cc`）：`0x24b14(item)` 固定掃 runtime slots 0..15，透過 `0x31860` 找 exact item、found 回 1／missing 回 -1，不篩 camp/activity。ch26 post 的 `0x24b14(0x64)` success arm 沒有 `0x1b8e7`，故天空之鑰不消耗；其後才 `sync_party→increment chapter→0x25089 cleanup`。missing arm 進 FDTXT_027 idx13–16/ending 演出，不能以成功流的 town/preparation 或 generic ending 取代。
- 2026-07-26 ch26 palette-ramp closure（isolated Capstone `0x25052`）：helper `(start,delay_ms)` 從 `start` 遞減至 0（inclusive），每步 `0x11df2(0,255,delta)` 後 `0x375b2(delay_ms)`；ch26 成功臂的 5/4/3/2、80ms calls 可安全 lower 成 ordinary editable palette_update/delay beats。compiler 僅收 immediate start 0..63 與非負 delay，並固定 zero step，不能把它換成 general fade；其後 `0x24618` indexed adapter 仍 fail-closed。
- 2026-07-26 ch26 palette-ramp export regression：`CompileHandlerScript(ch26_post.json)` 現逐一驗證六個 native call-site `0x25244/0x25277/0x25290/0x252a9/0x252bf/0x252d5` 都不再 unresolved，且各自輸出完整 descending delta、80ms delay sequence；這防止 compiler 只在 synthetic helper test 成功卻漏掉實際 handler IR。
- 2026-07-26 stale `0x1f882` sync assertion correction（isolated Capstone）：展開其 shared tail `0x1f503..0x1f524`：`ebx=0..63`，每步 `0x11d40(0,255,ebx)`＋2ms，故它是 native palette fade-out，非 vsync／同步 helper。doc12/doc23/doc39 的舊說法已撤回；它走 `0x11d40` darkening path，不能與已 lower 的 `0x25052→0x11df2` delta ramp 混用，indexed DAC adapter 前維持 fail-closed。
- 2026-07-26 native palette-fade IR boundary：`0x1f882` 已 lower 為 exact `native_palette_fade_out{start:0,end:63,delay_ms:2}`，不再只留 unknown；compiler 拒絕任何 argument-bearing variant，runtime regression 固定在 indexed DAC adapter 未完成時報明確 fail-closed error，禁止退化為 storyFade。
- 2026-07-26 native palette-pulse closure（isolated Docker Capstone `0x35e5a`）：無參數 helper 完整 body 固定執行 `0x11df2(0,255,delta)` 的 inclusive 0→63、8ms each，400ms hold，然後 inclusive 62→0、8ms each。它不是 generic fade 或 `0x1f882` darkening。新增 exact editable `native_palette_pulse` IR，compiler 拒絕帶參數變體，PNG runtime 尚無 indexed DAC adapter 前明確 fail-closed；官方 IDA 9.4 xref export 已加入 `0x35e5a` 與其 `0x33f78` staging-wrapper 鄰接證據。
- 2026-07-26 ch29 staging-wrapper closure（Docker Capstone + official IDA 9.4）：`sub_33F78=0x33f78..0x33faf` 有七個 `sub_33E3C` callers。full stack trace 固定 exporter 的 native push order `[y,x,slot]`，wrapper 先 `0x12cea(slot,x)`，再呼 `0x22253(slot,x,y,x,y)`。七個 ch29 pre call-sites 已 lower 成 exact `native_staging_present` payload；`0x22253` 的 11+6+10 indexed choreography 尚未有 renderer adapter，runtime 保持 fail-closed，不可猜成 spawn 或 camera pan。
- 2026-07-26 `0x22253` schedule contract：將既有 direct trace 資料化為 `fdother.NativeUnitPresentSchedule`，嚴格列出 27 個 present boundary：#6 LMI `0x72..0x7c` 每次後1 tick，#3 LUT `5..0` 每次後10ms且 entry0 後才2 ticks，#3 LUT `0..9` 每次後1 tick。測試拒絕早期 six-frame shortcut與移位 tail ticks；geometry/buffer redraw 與 GUI adapter 仍未完成，維持 fail-closed。
- 2026-07-26 `0x22470` LMI phase origin closure（Docker Capstone）：destination 是 `buffer+0x8088 + 24*(x-camX) + 24*456*(y-camY) + 456`，隨後才 `0x127a9`、320×192 present、tick1。新增 raw `NativeUnitPresentByteOrigin` regression；offscreen clip 不在這個位址公式中，仍由 native caller/未來 renderer adapter 處理。
- 2026-07-26 `0x22470→0x4e85b` first executable primitive：Capstone 直接確認 `0x4e85b` header width/height 後每 pixel 呼 `0x4e916`，只在 decoded value 非零時寫 destination。新增 `BlitNativeUnitPresentLMI`，以 `LMI1Entry.BlitAt` 的相同 preserve-zero semantics 寫 verified origin，offscreen origin fail-closed；尚未把 unit redraw/present/tick 或 27-step schedule 接 GUI。
- 2026-07-26 `0x22470` intro phase executor：新增 `RunNativeUnitPresentLMIIntro`，以 #6 `0x72..0x7c` 的原序跑完整11次；每次 blit 後必透過 callback 執行 native 的 redraw/present/tick boundary，缺 callback／entry table 短缺即拒絕。這是 indexed phase primitive，尚非 Ebiten renderer。
- 2026-07-26 `0x22253→0x22547` stack-slot mapping（2026-07-27更正）：
  `arg1/arg2`仍證實是source map座標；但重新映射`0x22046`六參數後，
  舊「middle phase dynamic radius」錯誤。literal arg3=11才是radius，
  scale固定16；`trunc((24*[0x53abd]+15)/5)*LUTIndex`是first radial與
  final rectangle的startY。第二radial從centerY開始，三者end/split
  rows已由`NativeUnitPresentLUTPass`固定。
- 2026-07-27 `0x22547/0x22656` geometry plan：刪除
  `NativeUnitPresentContractRadius`，改為
  `NativeUnitPresentContractStartY`；新增完整6 contract +10 release
  `NativeUnitPresentLUTFrames`，每frame保存fixed radius11/scale16、
  first/second radial與horizontal radius17 rectangle。indexed buffer
  snapshot、object redraw callback、present adapter仍待。
- 2026-07-27 unit-present buffer transaction：
  `RunNativeUnitPresentLUTFrame`保存callee每frame的`0x25680` memcpy
  restore，然後在`+0x8088` viewport依序跑first radial→完整buffer
  object redraw callback→second radial→rectangle→present callback。
  strict size/LUT/callback gates與regression已補；剩餘runtime blocker是
  Ebiten側exact indexed map snapshot/object redraw/present scheduling。
- 2026-07-27 unit-present indexed frame composers：`indexedmap`新增
  `ComposeNativeUnitPresentTerrainSnapshot`（只跑`0x11eee`）、
  `RedrawNativeUnitPresentObjects`（只跑`0x127a9/0x129ec`）、
  `CopyNativeUnitPresentViewport`（312×192,456→320 stride）與atomic
  intro/LUT frame composers。測試固定terrain→LMI→objects→viewport
  layer order及右側8px不覆寫。下一個真實runtime blocker是Game尚未保存
  raw unit+3 pose、unit+4 motion與native selector cycles，以及release
  snapshot provenance；禁止由normalized PNG/Dir猜造。
- 2026-07-26 native map redraw range-layer closure（Docker Capstone）：`0x11cac` 的順序是 terrain `0x11eee`、range `0x122dc`、unit/foreground `0x127a9`、HUD `0x1acf3`、viewport copy `0x11eb0`。**後續修正**：只有 raw modes1..5 以固定 offset/descriptor index 呼叫 `0x126f7`；mode6直接清 cell byte+3，7+ return。`0x126f7` 做 camera bounds 後以 raw `0x4deda` 寫 `0x53a49+0x8088`。
- 2026-07-26 `0x122dc` exact range-table closure（isolated Docker Capstone）：modes 1..5 的所有 `0x126f7(x,y,descriptor)` 已逐指令轉成 `fdother.NativeRangeOverlayPlacements`（call counts 1/1/5/13/21）。保留 mode3 centre descriptor 14 與 mode5 的重複座標、不同 descriptor，不能簡化為推測的菱形或移動範圍。mode6 沒有 blit：其 raw expression 是 `4*(cursorX + cursorY*[0x53ac1])+7`，對 `[0x53a51]` 指向資料寫 0；drawable API 拒絕它，另以 `NativeRangeOverlayMode6ByteAddress` 保存 checked arithmetic。`0x53a4d` descriptor-bank loader、RLE asset binding、camera clip 與 indexed renderer remain fail-closed.
- 2026-07-26 range descriptor-bank closure（isolated Docker Capstone + real FDOTHER）：撤回 doc36「FDOTHER#1 用途未確認」的舊斷言。`0x25c7d..0x25c92` 載 #1 至 `[0x53a4d]`；真實 header `{24,24,20,u32 offsets[]}`，`0x126f7` 按 `base+6+4*descriptor` 取 24×24 four-mode RLE stream 再 `0x4deda`。modes 1..5 使用 descriptors #0..18。新增 `DecodeNativeRangeOverlayBank`（嚴格 20 entries）及 `BlitNativeRangeOverlay`，保留 `0x8088`／stride456／24-pixel camera-relative destination、native pre-blit camera clip，且以實檔 decode+blit regression 驗證。mode6 raw grid mutation、native buffer lifetime 和 Ebiten adapter 仍維持 fail-closed。
- 2026-07-26 `0x122dc` mode6 storage closure（isolated Docker Capstone）：`0x108f0..0x10932` 將 FDFIELD composition 讀至 `[0x53a51]`，前四 bytes 為 signed width/height；`0x4dbfc` 從 `base+4` 每 4 bytes 初始化 cell byte+3=`0xff`、並 mask byte+2=`&0x1f`。因此 mode6 的 `base+4*(x+y*width)+7` 精確清除 selected composition cell 的 byte+3（export 中的 raw event-high/blit-mode byte）。新增 `ClearNativeRangeOverlayMode6FieldByte` 作 checked in-place primitive，bounds error 不得 mutation；不推論清零的遊戲／視覺語意。
- 2026-07-26 steady indexed map-frame scheduler closure（2026-07-28 corrected）：`internal/indexedmap.ComposeFrame` 把 `0x11cac` normal layers串成 terrain→range→unit→foreground→mandatory HUD→`0x11eb0`。copy現固定 source `+0x8088`、312×192、dst VGA `+0x504`／stride320並保留4px border。
- 2026-07-26 native map HUD panel subpass：新增 `indexedmap.BlitNativeMapHUDPanel`，嚴格要求 `0x1acf3` 的兩個 raw display gates、FDOTHER#5 LMI1 #130 geometry=69×34，並以 `NativeMapHUDLayoutFor(anchorX,456).Frame` transparent blit。gate 關閉不寫、entry geometry mismatch 拒絕且不留 partial write。這是 panel-only callback building block，terrain/unit icons 和 AP/DP/HP digits 仍未接，不能宣稱完整 HUD/Ebiten equivalence。
- 2026-07-26 native HUD sign/digit boundary：新增 `indexedmap.BlitNativeMapHUDSignedNumber`，將 `0x1aeb1` 的 LMI #0x83（value≥0，6×7）／#0x84（value<0，6×5）、absolute value 與 decimal draw origin `+8` 固化。digits 改由 mandatory callback 提供，primitive clone/commit，所以 callback fail 不留下 sign；未猜字型、數值來源或 AP/DP/HP語意。
- 2026-07-26 HUD mixed-codec correction：撤回剛才 HUD primitive 對 #130/#0x83/#0x84 使用一般 `LMI1Entry`/`0x4e916` 的錯誤。direct `0x1acf3/0x1aeb1` 都呼 `0x4e63d`；改為 `NativeMapHUDFrames`＋`DecodeLMI1FrameResource`/`ParseLMI1FrameEntry` four-mode `Frame`，真實 FDOTHER#5 regression 讀回 #130=69×34、#0x83=6×7、#0x84=6×5 並實際 blit。panel/sign APIs 已改用 Frame，仍不把相鄰 LMI directory entries泛化為同 codec。
- 2026-07-26 HUD two-digit closure（Docker Capstone + real FDOTHER）：`0x1aeb1` 對 `0x187d6` 固定傳 glyph base `0x1f`、digits=2；callee從 `%0.5d` 改 width char 為2，依字元以 `base+digit`、6-pixel advance 呼 `0x16886→0x4e63d`。新增 `BlitNativeMapHUDTwoDigitNumber`；真實 #5 #0x1f..#0x28 regression 固定 digit1=#0x20 是5×8、其餘6×8，仍每位 advance6。adapter 對0..99外 fail-closed；不命名字表值或AP/DP/HP語意。
- 2026-07-26 HUD terrain-icon closure（Docker Capstone）：`0x12e38` 先把 cursor cell 的 masked terrain descriptor 放 local word0；`0x1acf3` 立即以它索引 selected FDSHAP `0x53a5d` offset directory，`0x4deda` raw blit 至 panel origin+6。新增 `BlitNativeMapHUDTerrainIcon`，只接受原始10-bit descriptor／supplied bank、anchor and bounds；不將 exported PNG 或 terrain標籤冒充 native icon source。
- 2026-07-26 HUD unit-icon closure（Docker Capstone）：`0x1acf3` 在 `0x12c0d(cursor)` 成功後，讀 unit+2、乘12加 global raw state（3改1）從 selector-cache pointer block 取 FDICON，`0x4deda` blit 至 panel `stride*5+6`。新增 `BlitNativeMapHUDUnitIcon`，以 validated `NativeSelectorCache` 重現 slot→raw key→frame，不命名 slot/fig/portrait identity；invalid state/cache fail-closed。
- 2026-07-26 HUD terrain AP/DP closure：`0x1acf3` 對 `0x12e38` local control byte+1 查兩張 raw tables，六碼 pair 固定為 0→(+5,0)、1/5→(0,0)、2/3→(-5,+10)、4→(-5,-5)，隨後對 layout AP/DP 各呼 `0x1aeb1`。新增 `NativeMapHUDTerrainAPDP`／`BlitNativeMapHUDTerrainAPDP`，覆蓋每碼、正負 sign／two-digit renderer 與 invalid-code no-mutation；control byte高階語意未命名，HP ratio仍另一條 `0x1875d`。
- 2026-07-26 HUD persistent-anchor closure（Docker Capstone）：直接重讀 `0x1ad2a..0x1ad5f`，確認 anchor global `0x51a0c` 並非每幀純函數：唯 `0x53abd>5 && 0x53ab9<3` 寫 `0xf2`，唯 `0x53abd>5 && 0x53ab9>9` 寫 `1`，中間／左側保留前值。新增 `indexedmap.AdvanceNativeMapHUDAnchor` regression，保留 raw state 而不猜測 globals 的高階名稱。
- 2026-07-26 HUD HP closure（Docker Capstone + real FDOTHER）：`0x1ae8e` 以 zero-extended unit `+0x40/+0x42` 送 `(dest,stride,current,max,mode3)` 到 `0x1875d`；它只在 current==max 選 glyph base #0x1f，否則 #0x2a，`0x187d6` 對 **current** `%0.3d`、6px advance。current>999 直接 blit base+10，真實 FDOTHER#5 #0x29/#0x34 都是18×8；兩 base digit 1為5×8、其餘6×8。新增 `indexedmap.BlitNativeMapHUDHP` atomic adapter，不把 unequal branch 或數值命名成未證實 gameplay 語意。
- 2026-07-26 full map-HUD assembly closure（Docker Capstone）：重讀 `0x1ad72..0x1aea9`，順序精確為 panel→terrain→AP→DP→`0x12c0d` optional unit gate→icon→HP。新增 `BlitNativeMapHUD` raw `NativeMapHUDInput` transaction；`OptionalUnit=nil` 只能代表原版的任一 skip path，unit `+7/+0x1f/+6` gate 不以猜測性角色模型重演。closed display gates no-op；任何 subpass失敗不寫 destination。
- 2026-07-26 HUD optional-unit gate closure（Docker Capstone）：`0x1ae2a..0x1ae47` 的兩條 skip 已固定為 raw `unit+7==0x79`，或（前者不成立時）`unit+0x1f==0x0a && unit+6==1`。新增 pure `NativeMapHUDOptionalUnitEligible` regression，讓 caller 對 `0x12c0d` result 建立 OptionalUnit／nil，未替三 byte 猜測語意。
- 2026-07-26 strict indexed map-frame closure：新增 `ComposeNativeFrame`，直接把 verified Frame layers 和完整 `BlitNativeMapHUD` resources/input 綁定為 `NativeFrameInput`，不讓呼叫端以任意 callback 取代 `0x1acf3`。回歸確認 panel/terrain/unit/HP bytes 均進 456-stride work 並被 `0x11eb0` copy 到 VGA。當時 Ebiten 仍是 PNG-backed legacy path；後續 ch01 已接 native indexed→Ebiten strict bridge，ch26 亦已資料化 event61 所需 view/HUD，但不可泛化為全章 E2。
- 2026-07-26 FDSHAP archive bridge：新增 `fdother.DecodeSpriteBankResource`，用既有 LLLLLL loader 將指定 FDSHAP even image resource直接交 `fdicon.Parse`，player archive #0 實測/回歸為288個24×24 four-mode frames。控制表仍要求 caller 明確取相鄰 resource，避免 image/control 猜配。
- 2026-07-26 FDSHAP map-pair closure：新增 `DecodeMapTerrainResources(N)`，固定讀 FDSHAP #`2N` image/#`2N+1` four-byte control table並做 capacity validation；原版 map0 regression為288/1200。runtime map binding必須明示 N，禁止由 tile count/cost猜配。
- 2026-07-26 production native asset gate：`Game.loadMap` 現嘗試組合完整 native bundle（FDOTHER HUD frames、explicit FDSHAP pair、FDICON.B24、palette），只有全數成功才寫入 `nativeMapAssets`；PNG draw path 尚未切換，避免半套 indexed 資源改變可玩路徑。
- 2026-07-26 indexed-to-Ebiten partial HUD bridge：`drawNativeMapHUD` 已將 verified 456-stride buffer crop 為 320×200 paletted Ebiten surface，實際呈現 panel/terrain/AP/DP；invalid raw tile/control或render error完全回退 legacy panel。optional unit/HP仍 fail-closed，等待 runtime unit-record admission bytes。
- 2026-07-26 unit constructor provenance（Docker Capstone + official IDA 9.4）：`0x10d7f..0x10efc` 證實 runtime `unit+6=FDFIELD b0`、`unit+7/+8=FDFIELD b1`，現有 `map_selector_key`/`battle_fig` 可沿用。高 class branch 以 `FDFIELD b1-0x44` 索引 `0x4e4ff → 0x61af9` 的 10-byte records，`unit+0x1f/+0x20` 是 record byte0/1；lower branch 則是 `0x4e4e8 → 0x61da1` 的 24-byte records，加上 `0x4e4d1 → 0x620a1` 的 11-byte records。撤回先前未證實的「DATO-derived」說法，也撤回把 `0x619fd` 接入 constructor 的說法（它屬於另一個 `0x4e516` helper）；兩 branch selector/table 尚未完整 export 前，optional unit/HP 仍 fail-closed。
- 2026-07-26 indexed compositor regression correction：`ComposeNativeFrame` 的 native source 是 `work+0x8088`（456 stride，viewport 會跨到 row319），測試 fixture 原先錯配 200 rows／range bank容量並以錯誤 viewport 座標驗證；已改成 320-row、完整 range descriptor bank 與實際 `(x=64,y=85)` copy 位置，Docker `go test ./internal/indexedmap` 通過。這是 test contract 修正，不改 renderer 語意。
- 2026-07-26 full Docker regression audit：`go test ./...` 實際執行；`internal/indexedmap` 等套件通過，但 `cmd/fd2` 的 Ebiten init 在無 `DISPLAY` 環境 panic，campaign 的 ch14/ch16 tests 仍暴露 generated branch context／line mapping 不一致。這些不是本輪 static-table 或 compositor 變更造成；在重新核對 handler CFG 前維持 fail-closed，不以改測試或猜測 branch 語意消除失敗。
- 2026-07-26 regression closure：使用 `fd2-go-test-local` 內建 Xvfb，完整 `GOMAXPROCS=1 GOFLAGS=-p=1 xvfb-run -a -s "-screen 0 1280x1024x24" go test ./...` 已全數通過（cmd/fd2、afm、battle、campaign、ending、fdicon、fdother、fdtxt、figani、indexedmap）。`assetPath` 增加 cwd ancestor lookup 解決 package-test 路徑；ch16 只在已 LOADCH 的 branch 內允許 SPAWN，ch14 line assertion 改為 FDTXT count-aligned continuation，不放寬 native 語意。
- 2026-07-26 native unit table raw export：新增 `tools/extract_native_unit_tables.py`，Docker 對實際 FD2.EXE 驗證 `0x61af9` 68×10、`0x61da1` 32×24、`0x620a1` 68×11 records；IDA stack trace 進一步閉合 selector 為 FDFIELD roster b1/portrait（高 branch 先減 `0x44`）。輸出只含 selector/helper provenance 與 `bytes_hex`，不把 record byte 命名成 DATO/gameplay/class。此 fixture 尚未接 runtime unit/HUD admission，維持 fail-closed。
- 2026-07-26 editable constructor raw field：`export_units.py` 可選讀 raw-table JSON，依已證實 portrait selector 輸出 `native_constructor` branch/index/record/aux_record；`battle.Load` 嚴格驗證 record 尺寸並保留舊 JSON 相容。Docker battle regression 通過；renderer/gameplay 尚未讀 raw bytes，避免過早解除 HUD gate。
- 2026-07-27 constructor `+0x42` projection closure：Docker Capstone 重讀 `0x10db4..0x10e58`，確認 high=`u16(record+2)*level`、lower=`u16(record+3)+aux_byte6*(level-1)`，兩者都由 constructor `0x10fe9` 原樣寫入 runtime `+0x40/+0x42`。新增 `native_record_word42_for_portrait`、標準庫 regression 與 `native_unit_tables.json` raw fixture。此段「只保存來源、不覆寫 hp」已被 2026-07-29 的完整 HP／MP 建構器證據取代；勿再當作現況。
- 2026-07-26 HUD raw-state closure（2026-07-27 名稱更正）：Docker
  Capstone/IDA確認 `sub_11cac→sub_1297d` 在 `0x1acf3` 前更新
  `[0x53c0b]`；只有 `BIOS-tick-last`<0 或 >4 才3→0 advance並更新
  `[0x53c0f]`。`[0x46c]`不是scanline；完整timer證據與moving half見本檔
  2026-07-27 raw pose/motion/cycle lifecycle。
- **歷史、已撤回（2026-07-26）** ch29 post campaign wiring：當時曾把 `campaign_full.json` 的 `postbattle_ch29_persist` 改接 `assets/cutscenes/bindings/ch29_post.json`、再進 `preparation_ch30`；後續索引稽核證實 raw `ch29_post` 不屬玩家第29戰，且 `0x2bce5` renderer 未閉合。現況不得把這段歷史當成正式接線；`postbattle_ch29_persist` 仍失敗即關閉，正確 raw owner 尚待重新索引。
- 2026-07-26 official IDA `0x2c548` gate recheck：`0x2c405` 完成 500-pass phase-0 後，`0x2c548` free staging、配置 `0x1f400 + 0xfa00 + 0xfa00` 三個 indexed buffers，呼叫 `sub_111ba("TAI.DAT",3)` 與 `sub_111ba("FDOTHER.DAT",0x38)`，並以 `sub_4e63d(FDOTHER#0x38, transparent=-1, stride=0x140, destination=first 0xfa00 buffer)` 建立 backdrop；其後才從 `[0x53bfb]-1` 反向進 party-cycle。這與既有 `native_2c548.json` first-cycle mapping 一致，但不代表 montage renderer 已完成，PNG/generic fade 仍禁止接入。
- 2026-07-26 `0x29164` mirror-branch ABI：official IDA 釘出 `unit[+6]==0` 會進 `0x2927e..0x29357`，仍做 9 次 stage `8..0`、palette `stage*6`，但 primary FIGANI frame pointer 改為 `staging+0x140-stage*10`；`arg4==0` 才執行額外 TAI#3 與 secondary FIGANI blit。新增 `native_2c548.json` `mirror_branch` schema 與 loader regression；此為 evidence-only，未解除 indexed renderer gate。
- 2026-07-26 mirror fade planner：新增 `Montage.PlanMirrorFigureFade(unitSide,sideFlag)`，輸出 9 個 exact primary offsets/palette deltas，並將 `arg4==0` secondary/platform gate 變成可測試純資料；Docker full regression 通過，未解除 renderer gate。
- 2026-07-26 mirror indexed primitive：新增 `RenderMirrorFigureFadePass`，按 `0x292ad` 使用 caller-preseeded `work+0x140` right viewport，執行 primary `+0x140-stage*10`、`arg4==0` secondary 與兩次 present；TAI#3 只驗證原始透明 bytes。Docker ending/full regression 通過；後續 `MontageCycle` 已將證實的 mirror pass 與 DATO/FDTXT phase 串接，輸入／campaign handoff 仍 fail-closed。
- 2026-07-26 correction：`[0x53a81]` 經 `14-text-control-codes.md`/IDA loader 確認是 `FDOTHER.DAT#5` dialogue-frame bank，不是 DATO；`0x2c773→0x168b6` 的 5×7×5×5 args 先建 dialogue grid，`[0x53a85]` 才是後續 DATO portrait source。撤回 `dato_layout` 錯誤命名，改為 `dialogue_frame_layout`。
- 2026-07-26 DATO decoder foundation：新增 `internal/dato`，按 official `0x4e8af→0x4e916` 高值-run codec 解四幀、保留 opaque zero、提供 indexed blit；synthetic 與玩家 DATO#37 regression 通過。只完成資源/像素 primitive，`0x168b6` grid、mouth cadence 與 ending UI 仍 fail-closed。
- 2026-07-26 dialogue-frame grid transcription：新增 `Montage.PlanDialogueFrameGrid()`，按 `0x168b6` exact arithmetic 保存 49 次 `FDOTHER#5` raw resource/destination placements（固定12、兩組3×2 loop、5×5 grid）；不猜 cell 高階語意，DATO portrait/mouth renderer 仍 fail-closed。
- 2026-07-26 FDOTHER#5 codec correction：official IDA 的 `0x1685c→0x4e9bb` 是 width/height 後逐 row `rep movsb`，不套 DATO 的 `0x4e916` high-run；新增 `fdother.ParseLMI1RawEntry/DecodeLMI1RawEntry`，真實 #5 entry1 3×3 literal bytes regression 通過，撤回 dialogue frame 使用 LMI1 RLE 的可能誤讀。
- 2026-07-26 dialogue-frame raw compositor：新增 `RenderDialogueFrameGrid`，依 49 個 verified placements 將 `FDOTHER#5` raw cells 以 opaque `rep movsb` 寫入 C buffer，保留 overlap/zero semantics；Docker ending/fdother regression 通過，DATO/text/input 仍 fail-closed。
- 2026-07-26 dialogue-frame resource-backed compositor：新增 `RenderDialogueFrameGridResource`，實際載入 player `FDOTHER.DAT#5` entries 1..17 並執行 native placement；缺檔 fail-closed，Docker ending regression 的 asset path 通過。
- 2026-07-26 DATO opaque paste：新增 `RenderDATOFrameAt`／`dato.Frame.BlitAtOffset`，按 `0x4e8af` stride320 opaque paste；destination offset 必須 caller 明確傳入（ending call site 為 `staging+[0x53c67]`），不猜固定 anchor、不接 mouth cadence。Docker dato/ending regression 通過。
- 2026-07-27 mouth cadence adapter：依官方 IDA `0x16D00`／`0x16C57` 證據新增 `remake/internal/dato.MouthState` 與單元測試，接入對話更新迴圈（每 2 frame tick；open→m0 並設 `rand()%30+2`；倒數到零→m3 一 tick）。只閉合節拍狀態機，不宣稱 DATO/grid/speaker layout 全部 parity。
- 2026-07-27 dialogue wording second pass：重掃 `14-text-control-codes.md`，刪除殘留的「`-17/-18` 直接肖像 ID」組合／步驟斷言，改回 `0x12C60` identity lookup、record `+7` 與 direct-DATO fallback 的可證 provenance；故事資產未被猜測性改寫。
- 2026-07-27 indexed transition buffer boundary：IDA 重跑 `0x24618/0x22046/0x11EB0` 確認 staging offset `32904`、stride `456`、viewport `312×192`、present stride `320`；新增 `fdother.CopyNativeTransitionBuffer`／`CopyNativeTransitionViewport` 與 bounds regression。只閉合 raw copy ABI，未把 descriptor/LUT/compositor 猜接進 runtime。
- 2026-07-27 indexed transition LUT selector：IDA `0x4DB9C` 確認是 `(lut,count,pixels)` in-place byte remap，`0x24618` 以 FDOTHER#3 directory offsets `i=9..1` 傳入 LUT pointer；新增 `NativeIndexedTransitionLUT` 與 strict 256-byte/bank-range regression。仍未把其接入 runtime compositor。
- 2026-07-27 indexed transition pass ABI：由 `0x22046` pseudocode 固定 raw args 的 scale16、第二次 `0x219AD` 從 `a2` 開始、final rectangle `EndY=a2` alias；新增 `BuildNativeIndexedTransitionPass` 與 regression，避免把 `a2` 正規化成猜測的 screen geometry。
- 2026-07-27 indexed transition compositor slice：新增 `indexedmap.ComposeNativeTransitionFrame`，以既有 raw banks/controls 與 `ApplyIndexedTransitionPass` 組合 terrain→unit/foreground→LUT pass→312×192 viewport，採 clone/preflight/atomic commit；只完成純 indexed frame primitive，campaign asset binding、palette timing、Ebiten presentation 仍 fail-closed。
- 2026-07-27 native map-cell bridge：`cmd/fd2.MapData` 現讀取 exporter 已產出的 `native_tile_blit_modes`／`native_terrain_control`（optional，舊 PNG map 不自動升級），`indexedmap.BuildNativeTerrainCells` 以 10-bit tile/high-byte pair strict materialize；未接缺欄位 fallback。
- 2026-07-27 worklist count audit：重算 `91-worklist.md` 勾選項後，README 與 worklist header 已由舊的 `366/92/66` 修正為 `[x]=372`、`[~]=93`、`[ ]=65`；數字只代表工程項目數，不宣稱遊戲完成百分比。
- 2026-07-27 native map LUT bundle：`loadNativeMapAssets` 現以 `DecodeLUTResource(FDOTHER#3)` 要求 transition LUT entries 1..9；缺 bank 時整個 native bundle 拒絕暴露，避免把 partial map assets 接成 renderer。
- 2026-07-26 FIGANI scheduler closure：官方 IDA `0x2b9a1` 確認 `arg4==0` 只清 subframe、不 render；非零路徑先 render 目前 frame，再以 descriptor `+6` delay 決定 subframe/frame rollover，frame count 到尾端 wrap。新增 `internal/figani.NativeScheduler.Step` 與 regression；保持 renderer/presentation 由 caller 提供，未猜測 ending 語意。
- 2026-07-26 戰場進入分割滑動閉合（2026-07-29 分類更正）：官方 IDA `0x1f42d/0x1f1cc` 確認 FDOTHER#5 LMI1 `#0x52` 的五步 `100,75,50,25,0`，左右位置為 `(85-offset,82)`／`(165+offset,81)`、列距456，並在每步呈現後還原。2026-07-29 直接重讀呼叫者 `0x1a30b`，確認其操作戰場記錄與 456 列距戰場緩衝，不是 `0x318ad` 選人視窗動畫。程式介面改名為 `fdother.NativeBattleEntrySplitSlideSteps`／`RunNativeBattleEntrySplitSlide`；第一步負 x 的邊緣裁切仍由回歸測試固定。MAP/TURN 與行軍輸入未猜測接入。
- 2026-07-26 preparation record gate closure：Docker Capstone 重讀 `0x1a866`，確認 raw offsets `+0x25/+0x05/+0x06/+0x40/+0x42`、selector/bit gate 與 `+0x42/10` toward-zero integer division、`+0x40` clamp。新增 `fdother` raw record parser/gate/normalizer regression；保留欄位 offset，不宣稱角色生命／部署／座標語意。
- 2026-07-26 preparation dispatch closure：Docker Capstone 重讀 `0x1a813`，確認 16 slots 的 `base+3*i` 重疊 layout、`+3/+5` gate、`+4` function-table index 與 zero-arg call；新增 `FindNativePreparationDispatch` raw planner/regression。只回傳 raw matches，不猜測 callback 的 MAP/TURN/事件名稱。
- 2026-07-26 preparation timer closure：Docker Capstone 重讀 `0x1a941`，確認每個 0x50-byte record 的 `+6` selector、`+5 bit0` gate、六個 `+0x22..+0x27` counter，只有 1→0 觸發 `0x1e1+counterIndex` downstream source。新增 `TickNativePreparationTimers` 與 regression；不命名 counter 的 gameplay 語意。
- 2026-07-26 preparation input closure：Docker Capstone 重讀 `0x19953`，確認 `E0/52/1C/39` return1、`01/53` return-1、`4B/4D` 更新 raw cursor `[0x53c57]` 0/1，unknown key continue；新增 `ApplyNativePreparationInput` regression，不命名 YES/NO。
- 2026-07-26 correction：重新 decode `extracted/raw/FDTXT/FDTXT_000.bin` physical `0x1b0..0x1b3`，內容是敵人掉落／道具滿／金錢訊息；因此撤回把 `0x1ac62` 稱為 preparation command stream。raw layout 保留但改名 `ParseNativePostResolutionCommands`，明確不接 D8/UI-11。
- 2026-07-26 D8 scope correction：官方 `0x1a30b` 重新確認沒有 `0x15f84` text call；raw record gate 為 `+6==2`、`+5&0x81==0`、`+0x25/+0x26==0`，`+0x40` 向 `+0x42` 每次加 `max/5` 並 clamp。新增 `NativeBattleEntryStep`；MAP/TURN labels 與 YES/NO input 不得由此函式或 resource #0x52 推導。
- 2026-07-26 shared-caller audit：官方 xrefs 顯示 `0x1a30b` callers=`0x135c5/0x17154/0x17272`；後兩者與 FDTXT_000 `0x19c/0x1a4`、`0x1728c` selector interaction 同路徑。撤回任何 D8-only 命名，raw transition 保持共用 primitive。
- 2026-07-26 raw action-bit closure：官方 Capstone `0x13512`/`0x13536` 確認 0x50-byte record 的 `+5 bit7` set/all-clear；新增 battle raw helpers/regression，保留 byte-level contract，不覆寫高階 turn 語意。
- 2026-07-26 inventory cell correction（2026-07-29 再勘誤）：官方 Capstone `0x1b8a6` 重讀確認 bit7 **clear** 只增加 occupied count；它不驗證 occupied prefix。caller 另以 count 掃 raw slots `0..count-1`，所以 hole 可暴露 stale item byte。刪除錯誤的 free-slot／prefix 斷言，`NativeInventoryOccupiedCount` 只保存 exact count。
- 2026-07-26 inventory reservation writer closure：官方 Capstone `0x1bb8c` 確認第一個 reserved (flag bit7) cell 被清 flag 並寫 item byte，成功回1、無槽回-1；新增 `AssignNativeReservedItem` raw atomic helper/regression。
- 2026-07-26 preparation shell screenshot：用 `fd2-go-test-local` Docker image build（`-buildvcs=false`）與 Xvfb 實跑 `FD2_CAMPAIGN=assets/scenarios/campaign_full.json FD2_CAMP_NODE=preparation_ch02 FD2_SHOT_FRAME=30`，新增 [`docs/figures/preparation-remake.png`](../figures/preparation-remake.png)。畫面可見地圖與「出戰整備」overlay；這是 remake artifact，不是原版 visual oracle，D8 MAP/TURN/YES-NO 仍 partial。
- 2026-07-26 native phase-dispatch raw gate：worktree resume 時保持乾淨；`/tmp/fd2cap` 仍不存在，Capstone 只在 `fd2-cap-local` 執行。官方 Capstone 重讀 `0x1d80b`，新增 `fdother.FindNativePhaseDispatchCandidates`：0x50 stride、count `[0x53beb]`、只接受 `+6==1`、`+5&0x81==0`、`+0x26==0`，回傳 raw unit/selector。未呼叫 `0x13a9f`、event/chapter tables，未猜事件語意；caller-level integration 留待後續。
- 2026-07-26 inventory removal ABI closure（official IDA 9.4）：decompiler 直接給出 `sub_1B8E7(int unit,int slot)`，以 `memmove(record+0x0a+2*slot, record+0x0c+2*slot, 2*(7-slot))` 左移後續 raw cells，再寫 `record+0x18=0x80`；新增 `battle.RemoveNativeInventorySlot` 與 slot0/2/7、stale-tail-byte、bounds regression。刪除先前「第三個 stack value 未閉合」的錯誤斷言；caller 仍需決定移除原因，但 raw removal 已可接 persistent inventory。
- 2026-07-26 unit mode dispatch raw boundary：`0x13a9f` 在 `record+5&5==0` 後讀 `+0x34&0x0f`、`+0x35`、`+0x36`、`+0x3d`；新增 `fdother.PlanNativeUnitMode` 保存 raw mode/args，短 record 與 gate fail-closed。mode-specific callee、event/AI/item 語意尚未命名，也未接 runtime。
- 2026-07-26 item effect dispatch topology（後續已有 typed closures）：
  Docker Capstone 固定 row `+0x0d/+0x0e` 與 type→callee/subcommand raw
  map。當時只保存 topology；後續 5/13 與 8–10 已分別閉合 HP restore
  consume/retain、永久 base AP/DP/DX，其餘 branch 仍依個別證據處理。
- 2026-07-26 type-6/7 mutation boundary（official IDA 9.4）：`0x22af6(a1..a5)` 逐 `a4` unit-index byte list，清除 `record+a5` 的 nonzero byte，raw score=`4*(record+0x21 + (record+0x20∈9..24 ? 30 : 0))`；新增 `battle.ClearNativeUnitByte`，renderer/global write 不接，未命名 gameplay effect。
- 2026-07-26 item word-delta closure（official IDA 9.4）：`0x21082` 從 `a6` 讀一個 unit index，對 `record+a3` 16-bit word 加 `low16(a2)`，再呼叫已閉合的 `0x1b8e7(a1,a4)`；新增 `battle.ApplyNativeWordDeltaAndRemove`，明示 target/removal units、raw offset 與 wrap semantics，未命名 HP/MP/能力效果。
- 2026-07-26 RNG marker correction：重讀 `0x4e893` 與 `0x22721/0x22866/0x22997`，確認 state=`rol16(state+0x9014,3)`，`idiv 4` 後使用 EDX remainder `+2`，不是 quotient。新增 `fdother.NativeRNGStep/NativeRNGMarker`，growth raw marker 保持未命名。
- 2026-07-26 raw word growth closure（official IDA 9.4）：`0x22721` 對 index byte list 逐筆 gate `record+0x22==0`；成功才 RNG→marker `+0x22`、`trunc(word+0x48*0.15+1)` 寫回 `+0x48`、score `2*effective(+0x21)`。新增 `battle.ApplyNativeRawWordStep`，保留 marked skip、RNG state、16-bit word wrap 與 preflight bounds；未接 renderer/`0x1317d` tail，未命名 gameplay stat。
- 2026-07-26 0x22866 family closure：Capstone 確認同一 growth arithmetic 只換 marker `+0x23`／word `+0x4a`；`ApplyNativeRawWordStepAtOffsets` 共用 raw adapter，variant regression 通過，不猜 gameplay 欄位。
- 2026-07-26 0x22997 pair mutation closure：Capstone 固定 marker `+0x24`、成功後 `+0x4c/+0x4e` 各加 `0x0f`、score `2*effective(+0x21)`；新增 `battle.ApplyNativeRawPairStep`，16-bit wrap 與 marked skip regression 通過，renderer/tail 不接。
- 2026-07-26 `0x22d1b` branch audit（後續 RNG 勘誤）：marker/class gate
  與第一 RNG 50% 正確；但 `0x1c81f(unit,10)` 內部另耗第二 RNG，
  marker 是第三 RNG，不是第二。base amount10 的整數 damage 為9。
- 2026-07-26 command-23 coordinate write closure（official IDA 9.4）：`0x22253` 最終將 supplied `a13/a14` 寫回 unit raw `+0/+1`；`0x2218a` 先傳 `0xff/0xff`，再用 cursor pair。新增 `battle.SetNativeUnitCoordinateBytes`，只接 raw writer，不接 renderer/pathfinding。
- 2026-07-27 persistent identity lookup closure（Docker Capstone）：`0x24bde` 逐 persistent record（caller count、上限32、stride `0x50`）比較 raw `+0x08` unsigned byte，命中回1、缺失回0；新增 `battle.FindNativePersistentIdentity` first-index/read-only/bounds regression。此 raw field 仍只稱 identity lookup，不命名 portrait/Fig/NPC。
- 2026-07-27 raw identity plumbing：`PartyMember.native_identity`（可選 JSON）與 `Unit.NativeIdentity`/presence flag 已加入；`syncPartyFromBattle` 對明確 raw key 走 persistent `+0x08` matching，未知 key fail-closed，缺欄位才沿用 Fig projection。新增 loader range、不得由 Fig 推導、Fig 不同仍能同步的 regression。全 roster/save/export 尚未攜帶 raw record，`SYNC-PARTY-RAW-IDENTITY-GATE` 維持 `[~]`。
- 2026-07-27 UI-12 四槽 selector slice：title LOAD 不再直接讀單一 JSON；新增 native `0x30550` 對應的四槽 bounded selector（0..3、不 wrap、Enter/Space、Esc），slot 1 保留舊 `fd2_save.json`、slot 2–4 使用 `fd2_save_1..3.json`，空槽顯示及 slot-path bounds regression 通過。這只閉合 UI contract，尚未宣稱 native `FD2.SAV` metadata/roster/checksum 相容。
- 2026-07-27 native save envelope adapter：新增 `remake/internal/fdsave`，把已證實的 `0x59cb` size、`0x4dbd8` rolling-XOR、`0x4dbb9` checksum、四槽 bounds 與 metadata `+0/+1/+2..+5` 暴露為 raw/verified API；opaque roster 與其他 metadata 仍只保留 bytes，未接自有 JSON campaign save，避免語意猜測。
- 2026-07-27 official IDA `0x30012` save-write closure：confirmed selector 後以 2560-byte runtime roster 寫入 slot record 起點、metadata `+0..+9` 寫入 verified globals，再算 `0x59c7` byte-sum、套 rolling-XOR 並寫完整 `0x59cb`。`fdsave.WriteSlot` 已保存 opaque replacement boundary 與 regression；native FD2.SAV 尚未接入 campaign runtime。
- 2026-07-27 `0x1145a` raw equipment adapter：新增 `battle.ApplyNativeEquipmentRecalc`，依八個 `[flag,item]` cell、`0x40` equipped gate、`0x4e56c` row stride 0x17、signed base words 與四個 raw destination 完整執行；bounds preflight atomic、16-bit wrap regression 通過。normalized equipment 與 row 欄位語意仍分離，未宣稱 gameplay/stat byte identity。
- 2026-07-27 UI-03 command availability：新增 `battle.NativeCommandAvailable`／`NativeAvailableCommandIDs`，嚴格保存 `0x159fa` 的 raw bit + `0x4e516` 0..35 record + MP cost gate；unknown physical IDs 36..39、malformed book、負 cost 均 fail-closed，不碰 `+0x27` action-direction gate。
- 2026-07-27 assertion audit：修正 worklist 與 boot/UI evidence 中仍稱「單一 JSON save」及「四槽 UI 待建」的過時文字；第10/11輪的「全30章一條龍」改標歷史快照，保留 provenance 但不再當現況 parity 證據；`RE-EQUIPMENT-RECALC-1145A` 改為已具 raw adapter、normalized campaign 仍 projection-only。
- 2026-07-27 `0x22046` indexed inner sequence：Docker Capstone 確認第一 `0x219ad` LUT pass→`0x127a9` mutation redraw→第二 `0x219ad`→centered rectangle LUT；新增 `fdother.ApplyIndexedTransitionPass`，所有 geometry 先 preflight，缺 middle redraw 或 malformed second pass fail-closed 且不改 buffer。尚未接 descriptor/double-buffer/Ebiten presentation。
- 2026-07-27 `0x11506` persistent-copy mutation closure（Docker Capstone）：selected pair 先 runtime→persistent copy `0x50`，清 persistent `+0x22..+0x27`、`+0x05 &= 1`，非1時 `+0x40=+0x42`，固定 `+0x44=+0x46`；新增 `battle.ApplyNativePersistentRecordCopy` regression。`0x3453e` gate 與 `0x1145a` tail 未接，sync runtime 仍 projection-only。
- 2026-07-27 `0x3453e` raw predicate closure（Docker Capstone）：回傳 selected runtime record `+0x05 & 1`，新增 `battle.NativeRecordByte5Bit0` mask/bounds regression；不命名成 acted/alive/active。
- 2026-08-02 ch15 post IDA 直接指令勘誤：合法 IDA Pro 9.4 確認 `sub_3453E` 將參數乘 `0x50` 後讀 `[dword_53A45+index*0x50+5]&1`，所以 handler 傳入的 `0x42..0x49` 確為 runtime slots66..73。另確認 `0x23b1f` 直接跳到 `0x23b52`，故 dialog2→ACTING49→dialog3 路徑跳過 JOIN18；只有兩個前置條件皆不成立且 slot0 `+0x42>=0x140` 時，才走 dialog4→JOIN18。candidate 已把 JOIN18 移入該 nested arm，撤回共同尾端的錯誤斷言。
- 2026-08-02 ch15 runtime shape 假說與撤回：一度以16名 persistent加map14 group0的58筆推導74 slots、再以turn7 group1四筆推導78；提交前反證審查發現map14 group0已含一筆`fig=15` ally，且準備介面最多選15名，無法判定原版是重建、提升既有map record，或保留未部署record。production scenario改動與過度樂觀測試已撤回；candidate binding僅保存74／78假說，不提升為強推論或E2，也不解除campaign binding。
- 2026-08-02 ch15 runtime shape 後續勘誤：上一條只在誤把 raw `ch15_post` 視為玩家第15戰時成立。主迴圈直接指令及 battle1–3 既有測試證實玩家戰鬥 N 使用 raw `ch(N-1)_post`；因此 raw `ch15_post` 實屬玩家第16戰，應對照 map15，不是 map14。合法 IDA Pro 9.4 直接指令另證實 `sub_320FC` 只重排完整 persistent records而不改總數；`sub_1088D` 先複製16個 persistent slots，再由 `sub_10B4E(0)` 逐列附加 map15 的60筆 group0，沒有 fig15 的略過、替換或提升分支，故入口固定76 slots。candidate 已改為76；`postbattle_ch14→ch13_post`、`postbattle_ch15→ch14_post` 已同步 campaign 與回歸，raw ch15 的 production owner `postbattle_ch16` 仍因四分支、JOIN18持續狀態及town17/save路徑未閉合而保持失敗即關閉。
- 2026-08-02 postbattle 索引系統性稽核：`audit_postbattle_binding_gates.py` 現會把 active binding 與 `battle N→raw ch(N-1)_post` 比對，不再以欄位非空代表正確；現行 ch04/05/08/09/10/11/12/13/18/19/24/25/29 共13個同號 binding 被標為 `active_index_mismatch`。本輪只移接已直接複核的 ch14/ch15；其餘下一輪逐章驗證，不能未經 gate 整批平移。
- 2026-08-02 postbattle 索引清理完成：scenario `chNN.json` 的 raw map `NN-1`、所有 raw post handler 的 `set_chapter(N+1)` 與主迴圈 dispatch 交叉確認零起算 owner。13個同號錯接中，ch04/05/09/11/12/19/25 已移接到完整 authored 前一號 binding；ch08/10/13/18/24/29 因未知呼叫、缺 mapping 或可疑來源位址撤回並由 runtime guard 失敗即關閉。新增 raw ch03 authored binding與全24個標準postbattle owner回歸；稽核現為9 active、13 blocked、2 mapping complete但未啟用，無 `active_index_mismatch`。較早聲稱 raw ch29／ch11／ch13／ch24／ch25 接在同號 player node 的歷史紀錄均由本條取代。
- 2026-08-02 postbattle ch26/ch28 後續勘誤：合法 IDA Pro 9.4 直接讀取主表與函式範圍，確認 index25=`0x24e80` 屬玩家第26戰，完整保存 entry12 兩分支、ACT77–80、sync與chapter26尾段，現接 `postbattle_ch26_persist→town_ch27`。index27=`0x25464` 則屬玩家第28戰；它準備 FDTXT_028 index7 後 `jmp 0x231df`，沿真實共享尾段執行 dialog／sync／chapter increment，現接 `postbattle_ch28_persist→preparation_ch29`。2026-07-20 將 raw ch27 binding 掛在玩家第27戰天空之鑰成功分支的紀錄已失效；該成功分支只保留 raw ch26 已證實的 `sync_party→set_chapter(27)`。另發現 ch06/07/16/17/20/22/23 七個未綁定節點仍有泛用 inline beats，會繞過 runtime guard；現已移除。稽核更新為11 active／13 blocked，沒有 mapping-complete、index-mismatch或 inline-bypass。直接證據見 `docs/data/fd2_post26_28_dispatch_ida.txt`；本批為E1，尚非一般玩家E2。
- 2026-08-02 raw ch05 post owner closure：合法 IDA Pro 9.4 直接確認 post table index5=`0x23296`；handler 依序 JOIN13、spawn group3、pan raw `(5,14)`、ACT27，再由 `0x232e3 jmp 0x231df` 進真實共享 dialog／sync／chapter increment 尾段。Capstone 對同範圍與 table bytes 交叉一致。舊 `postbattle_ch05→town_ch06` 同號接法已失效；authored `ch05_post` 現接正確的 `postbattle_ch06→town_ch07`，並以 ordered-tail 與 owner regression 固定。稽核更新為12 active／12 blocked；這是E1，仍缺 `0x2cad7` 戰間 outcome及一般玩家E2。直接證據見 `docs/data/fd2_ch05_post_dispatch_ida.txt`。
- 2026-08-02 raw ch12 post owner closure：post table index12 的原始 bytes `9f 38 01 00` 固定入口 `0x2389f`；IDA 將它歸入較大 `sub_237D5` 只屬導覽，不得覆蓋真實 interior entry。handler 由 FDTXT_013 index9 對話開始，address context 依 count-aligned map 展開 ch13 scene3／4各6句，再走 sync→`0x238d7 jmp 0x237c8`→JOIN3→`0x237d0 jmp 0x231f2`→chapter13。authored binding 現接正確的 `postbattle_ch13→town_ch14`；跨 scene與共享尾段均有 compiler／campaign regression。稽核更新為13 active／11 blocked；本批仍為E1，`0x2cad7` outcome及一般玩家E2尚缺。直接證據見 `docs/data/fd2_ch12_post_dispatch_ida.txt`。
- 2026-07-27 `0x1145a` equipment recalculation audit（Docker Capstone）：signed base `+0x37/+0x39/+0x3e`，八格中 flag bit `0x40` 才查 item `+1` 的 `0x4e56c` effect words `+1/+5/+3/+7`，寫 raw `+0x48/+0x4a/+0x4c/+0x4e`；normalized `campaign.RecomputeEquipment` 保持 projection-only，raw item table adapter 未接。
- 2026-07-27 `0x4e56c` item-row address closure（Docker Capstone）：函式只做 `0x602ad + item*0x17` 指標算術，row stride 為 23 bytes；新增 `battle.NativeItemEffectRowOffset`，只暴露 table-relative offset 並限制 byte-sized selector。row 欄位／table 長度與 normalized equipment 接線仍未證實，維持 fail-closed。
- 2026-07-27 item dispatcher route slice（official IDA 9.4）：`0x20c6f` pseudocode 固定 item row `+0x0d` type、`+0x0e` word 與各 branch 的 raw callee/argument（含 type22→`0x22d1b(arg2=22,arg5=39)`；撤回舊 evidence map 將它誤寫成 `0x21082/0x27`）。新增 `battle.NativeItemEffectRouteForType` 與 route regression；只保存 call topology，不執行 callee、不命名 potion/status/damage/equipment effect，target/mutation/UI 仍 fail-closed。
- 2026-07-27 `0x211a4` item-effect ABI closure（後續補上 gameplay scope）：
  偽碼固定 `(actor,count,targetBytes,amount)`；`0x20c6f` 傳 `a3/a4`
  count/list 與 row `+0x0e` amount。後續 dispatcher-tail 複核已證實這兩條
  item branch 寫 current/max HP：type5 restore 後消耗來源 slot，type13
  保留來源。共享 callee 仍不可整體命名成專用 potion routine，但 item caller
  的 HP restore/consumption contract 已閉合。
- 2026-07-27 `0x1bbdc` item target closure：官方 IDA/Capstone 固定 case0 first stage=`row +0x10` mode、`+0x15` target code、type0x17 inner marker1；`0x115b6` 確認後 second stage從 confirmed cell用 `+0x12/+0x15`、inner0 建 final list，再呼 `0x20c6f(actor,slot,count,list)`。新增 `NativeItemTargetPlanFromRow`／`NativeItemEffectTargets` 與 regression；完整 runtime row producer、renderer/gameplay mapping 未閉合，UI仍 fail-closed。
- 2026-07-27 `0x1c916` HP mutation closure（Docker Capstone）：新增
  `battle.ApplyNativeRawHPRestore`，依 16-bit RNG 與原生 arithmetic 寫
  current HP `+0x40`、cap max HP `+0x42`，保存 score gate。當時保持
  shared primitive；後續 type5/13 caller 已閉合 item scope，UI 仍未接。
- 2026-07-27 `0x1c9dd` MP mutation closure（後續已閉合 item caller）：
  新增 `battle.ApplyNativeRawMPRestore`，同一 RNG/amount arithmetic 寫
  current MP `+0x44`、cap max MP `+0x46`；score 只用 `+0x21`。後續
  type11 caller 已閉合為 consumable MP restore，見本檔尾端。
- 2026-07-27 type-21 `0x2111a` 舊結論已撤回：當時把相鄰的
  `0x1ca89` command-MP debit 誤算成 `0x1cac7`；後續 official IDA/Capstone
  重核與正確 closure 見文件尾端。
- 2026-07-27 raw subtract adapter address correction：既有
  `ApplyNativeRawWordSubtract` arithmetic 對應 `0x1ca89`，不是
  `0x1cac7`；type21 不呼叫此 helper。
- 2026-07-27 official IDA `0x1c4cc/0x1c2da` item presentation closure：Hex-Rays 閉合兩 caller 的 `(actor, raw subcommand, target count, target-byte list)` ABI；`0x1c4cc` 逐 frame 取三張 33-byte frame table、對 camera-visible targets 做 456-stride indexed redraw、312×192 present、subcommand/frame SFX branch 與 BIOS tick，`0x1c2da` 依 `12*visual+cycle` pointer bank 做 target blit，再五次 restore/present。此證據只加入 SDD/worklist 的 presentation ordering，不替 item type、row word、SFX 或 frame asset 命名，也不解除 item runtime/UI gate。
- 2026-07-27 official IDA `0x1cd17/0x1c1c3` item closure：Hex-Rays 閉合 type20/24 的十幀 presentation loop（30-byte remap table、saved-buffer restore、camera-visible target redraw、`7-(frame%8)` blend arg、312×192 present、BIOS tick）與 selector compatibility predicate（actor class 的 six-byte raw table 對 item row `+0`）。兩者只寫入 opaque ABI/evidence，不命名 status/damage/equipment，也不解除 item runtime/UI gate。
- 2026-07-27 official IDA `0x4e53e` table provenance：Hex-Rays 固定 class compatibility row pointer=`0x6188a+class*7`；`0x1c1c3` 只讀 row+0..+5，row+6 保持 opaque。新增 `battle.NativeClassCompatibilityRowOffset`／`NativeClassItemCompatible` 與 Docker Go regression；不接 normalized class/equipment。
- 2026-07-27 item UI shell（後續已更正）：當時 remake以八個 raw位置
  顯示空洞且只接↑↓／Enter／ESC；後續 official IDA已證明 original
  selector會 compact occupied prefix、兩欄四列並支援←/→±4，見文件尾端。
- 2026-07-27 item type 8/9/0xa route（後續已閉合欄位）：
  Docker Capstone 固定三個 branch 以 item row `+0xe` word 寫 target
  `+0x37/+0x39/+0x3e`，並帶 presentation selectors
  `0x11/0x12/0x13`。當時 offset 尚保持 opaque；同日後續的全表
  cross-check 已定案三者為 base AP/DP/DX，見本檔尾端 type8–10
  gameplay closure。presentation selectors 仍未命名。
- 2026-07-27 item RE wording correction：`32-item-combat-stats-re.md` 舊「case 0 的 0x20c6f 仍待解碼」已改為「type dispatch 與數條 raw mutation route 已閉合；target-list producer、presentation、玩法語意仍待」，避免抹除已完成的 raw evidence，也避免宣稱 item-use 已完成。
- 2026-07-27 official IDA `0x25de5` campaign-loop closure：主迴圈在 phase2 先 stop BGM、呼叫 `funcs_25e23[chapter]` post-handler，再呼 `0x2cad7` gate；只有 gate 回傳 zero 才呼叫 `funcs_25e3a[chapter]`、切換 `0x51e63[chapter]` battle BGM 並回到下一戰。phase1 另走固定 `0x22e5c` interlude。這只閉合 call order，未把 `0x2cad7` 命名成 town/shop/menu，也不解除各章 transition E0/E2 gate。
- 2026-07-27 official IDA `0x2d093/0x318ad` town-preparation closure：`[0x5412b]` option0→`0x2fc85` hotel、1/3→`0x2e341` shop family、4→`0x3072f` church、2→save/confirm→`0x318ad` preparation；facility return 後恢復 BGM10。`0x526b9` next-index table 固定 town/preparation-only 章節集合。只閉合 native branch/order，不把 option label、cursor art 或逐章 E2 視覺宣稱完成。
- 2026-07-27 raw gate-table verification：Docker `data 0x526b9 0x30` 讀出 `byte_526b9[22..24]`、`[27..29]`=`1`，其餘 town 範圍 `[0..21]`、`[25..26]`=`0`；配合 `0x2cad7` branch 明確證實 nonzero 是 preparation-only，zero 才進 selectable town hub。此修正只補 table provenance，不替 raw gate 命名高階劇情。
- 2026-07-27 official IDA `0x2e341/0x2fc85` hub subscene closure：shop raw selection/resource branches dispatch `0x2f0b0/0x2f642/0x2f883/0x2f8ea`，hotel choices dispatch `0x2ffa5/0x30012/0x301f4/0x197e5` path；Hex-Rays 固定各子場景完成後 indexed fade 回 hub。未知 callee 僅留 address-level，未命名 normalized service。
- 2026-07-27 `0x22af6` raw flag restore closure（後續 ABI 勘誤）：
  當時 adapter 誤建 caller-owned parallel flags；完整 trace 證實 marker 是
  target `record+a5`。API 已改成 `(records,targets,markerOffset,rng)`，
  清除同一 record byte。
- 2026-07-27 `0x22d1b` raw application closure（後續修正）：adapter
  現保存 gate/damage/marker 三 RNG、base10→actual9 HP、marker與
  accumulator；normalized command executor 同步修正。status 名稱不猜。
- 2026-07-27 README／markdown scope audit：根 README 改為資產／RE／引擎切片／原版差距四欄狀態表，加入 title/dialogue/battle/preparation/church/overlay 圖片；`remake/README`、`00-index`、`20`、`22`、`90`、`51` 與 worklist 均降級過強的「全 30 章／只剩整合」宣稱。專題 RE 文件不合併，保留 address-level provenance；完成度以 README→SDD56→gap42→worklist91 路線判讀。
- 2026-07-27 `0x24d22` branch audit（Docker Capstone）：非零 arg 僅寫 global `0x51a10` low byte 後返回；零 arg 才配置／copy stride `0x138` indexed buffers 並走 `0x37416` free。global 與 copy loop 保持 raw/evidence-only，不命名成 fade、cursor 或 UI renderer。
- 2026-07-27 `0x24e80` raw marker rewrite closure（Docker Capstone）：runtime slot `0x10..count-1` 中，只有 record `+0x07==0x1f` 才寫 `+0=0x10/+1=0x06`；新增 `battle.RewriteNativeMarker1F` regression。欄位保持未命名，不接 renderer/identity。
- 2026-07-27 caller `0x24838` audit（Docker Capstone）：`0x24b14(0x64)` success→text#8→`join(0x16)`；`0x24bde(0x12)` hit→text#10/acting#0x48/`0x32975(0x11)`，miss 再依 `0x53bef<0x0f`→text#13/`join(0x13)` 或 text#12/`0x32975(0x11)`；共用 sync/presentation。只留 raw call order，不命名 immediate 語意，campaign binding 維持 fail-closed。
- 2026-07-27 raw record byte5 closure（Docker Capstone）：`0x32975(index)` 直接把 selected `0x50` record 的 `+0x05` 整 byte 覆寫為 `1`；新增 `battle.SetNativeRecordByte5One` overwrite/bounds regression。與 bit7 writer 分開，byte5 仍不命名成 acted/turn/action。
- 2026-07-27 command 17–19 raw dispatcher：新增
  `battle.ApplyNativeCommandModifier`，嚴格映射 ID17→`0x22721`、
  ID18→`0x22866`、ID19→`0x22997`，保留 branch-specific result/RNG/
  accumulator。後續 equipment cross-check 已定案 destinations 為
  derived AP/DP/HIT/EV；command MP、target、presentation transaction
  仍須依各 caller 證據閉合。
- 2026-07-27 AI spell score raw slice（Docker Capstone，後由 Hex-Rays 校正）：attack IDs0..12 為 `HP < spellValue → 24`、否則8，raw `+0x08==0` 時乘 `1.5` toward-zero；recovery IDs13..16 為 `<max/3→8`、`<max/2→3`、否則0，再乘 `+0x34 bit0`。新增 `battle.ScoreNativeAISpellAttack`／`ScoreNativeAISpellRecovery`；ID10..12 要求 caller-supplied `0x1f183` gate。未接 AI runtime、command inventory、target UI 或效果名稱。
- 2026-07-27 AI flag-score slice（Hex-Rays 校正）：ID20/21 逐候選讀 raw `+0x25/+0x26`，nonzero 各累加6；ID26/27 讀同兩 offsets，zero 各累加4；新增 `battle.ScoreNativeAISpellFlag`／`ScoreNativeAISpellZeroFlag`。只保存 score/read ABI，不清除、不命名 flag，也不接施法 runtime。
- 2026-07-27 AI ID22 score slice（Docker Capstone）：`0x15d30` 對候選先要求 raw `+0x27==0`，再呼 `0x1c269(unit,nil)` 掃 `+0x1a..+0x1e` 五 bytes 的 bitset；任一 bit set 時累加6。新增 `battle.ScoreNativeAISpell22`，不命名欄位、不接 ID22 effect/status runtime。
- 2026-07-27 Hex-Rays `0x15b77` score correction（合法 IDA Docker）：pseudocode 關閉先前三個 raw score 誤讀：recovery IDs13..16 是 `<max/3→8`、`<max/2→3`、否則0，再乘 `+0x34 bit0`；ID20/21 的 nonzero `+0x25/+0x26` 各加6；ID26/27 的 zero `+0x25/+0x26` 各加4；ID17/18/19 的 zero `+0x22/+0x23/+0x24` 各加3。新增 `ScoreNativeAISpellZeroFlag`，並修正既有 recovery/flag adapters 與 tests；仍不命名欄位、不接 AI runtime。
- 2026-07-27 Hex-Rays `0x1598a` dispatcher closure（合法 IDA Docker）：pseudocode 固定 `unit+0x27==0`→`0x1c269` command list→command record `+5 <= unit+0x44`→`0x4e040`/`0x14818` target candidates→`0x15b77` score；最大 score 優先，平手比較 command record `+0`，再保存 raw `(x,y,command)`。新增 `battle.SelectNativeAISpellCandidate`，只接 score/tie-break，不接 MP、target resolver、UI 或施法執行。
- 2026-07-27 native AI command gate adapter：新增 `battle.NativeAvailableAICommandIDs`，在既有 0..35 bounded command/MP scan 前保存 `0x1598a` 的 raw `+0x27==0` gate；第五 command byte 中的 36..39 仍被省略，避免未知 physical IDs 偽裝成 executable commands。
- 2026-07-27 native AI planner bridge 的「只保存 command IDs
  `>=0x10`」結論已由 2026-07-29 直接指令勘誤取代；不得再依此段接線。
- 2026-07-26 command-23 caller scope correction（Docker Capstone）：`0x250cc` chapter-ending/post handler 也在 `0x1c2da` 後呼叫 `0x22253`（unit=1、pre-render `0xff/0xff`、record raw `+0/+1`），接著才進 `0x25089` cleanup 與 `0x2bce5` ending renderer；因此 `0x22253` 不得命名為 command-23 專屬。raw writer 已閉合，但 ending layout/renderer 與 campaign semantics 仍 fail-closed。
- 2026-07-26 `0x250cc` ending branch audit（Docker Capstone）：`0x25348` 依序呈現 FDOTHER `#0x0d/#0x0e/#0x0f`、呼 `0x1c2da`、共用 `0x22253` 寫 unit=1 raw `+0/+1`、再呈現 `#0x10`，最後 `0x25089→0x2bce5` self-loop。只保存 call order；`0x24b14` 回傳與 frame 語意未命名，不能拿此終局分支接一般戰後 town/shop。
- 2026-07-26 raw inventory gate closure（2026-07-29 再勘誤）：`0x24b14(item)` 掃 unit `0..15`，`0x31860` 先取 `0x1b8a6` 的 bit7-clear count，再以 `0x1b722` 比對 raw slots `0..count-1`；不驗證 prefix compactness。`battle.FindNativeInventoryItemInUnit`／`FindNativeInventoryItem` 保存此 read-only count-sized scan。成功不移除 item，ch26 後續分支仍須獨立證據。
- 2026-07-27 worklist assertion cleanup：移除歷史 WBS 中「`0x22e5c` 尚待反組譯、負責 event_id→group 增援」的現況暗示。當時曾把它誤稱為「章1專屬固定中場」；2026-07-29 的 IDA 優先複核只保留固定 `FDOTHER.DAT` #79 呈現邊界。真正增援鏈是 `0x1a813` 的 turn/camp filter → `0x51b91` handler table → spawn 原語。當時仍誤把全表寫成 58 entries；2026-07-29 已更正為全域 90 entries、FDFIELD 子集合 0..57。
- 2026-07-27 constructor inventory flags：官方 IDA `0x10c50` 釘死八格 flag 初始化與 `0x2f8ea` signed-byte gate；`0x40` equipped 與 `0x00` ordinary 都可進 caller list，只有 `0x80` reserved 排除。新增 raw flags adapter/regression，Load/PartyUnits 在有 `inventory_slots` 時保留 flags；撤回「church transfer 只接受未裝備物品」的錯誤斷言。
- 2026-07-27 attack overlay predicate：官方 IDA `0x1b83d` 釘死 `flag&0x40`＋item `<0x80`/`>=0x80` 分支與 first-slot return；新增 `NativeEquippedInventorySlot`，overlay 有 raw constructor flags 時不再用 `Equipped` projection 冒充 attack precondition。
- 2026-07-27 item availability gate：`0x1b8a6` 的八格 signed flag count 已接 `NativeInventoryAvailableCount`，action overlay 有 raw flags 時不再以 compact `len(Inventory)` 冒充 item availability；legacy 無 provenance 才 fallback。
- 2026-07-27 attack geometry caller：官方 IDA `0x14237→0x14818` 釘死 item row `+0x0b/+0x0c` 傳入 raw `a5/a4`，`mode<0x10` 再排除 Manhattan `<a5` marker cells，cross mode 不套 marker；新增 `NativeAttackCandidates` 與 regression，未命名欄位或宣稱完整射程/LOS。
- 2026-07-27 README/KB review：補上原版與 remake 對話成果圖，並把 README 的跨平台、EXE 表完整性、SDL2 runtime 等強斷言降級為已驗證切片／長期目標；`08-text-and-font-format.md` 兩個圖片連結修正為 `../figures/...`。文件不做大合併：`56`/`57`/`42`/`91` 是現行裁決入口，`90`/`30`/`51`/本 handoff 保留歷史與 address-level provenance。
- 2026-07-27 assertion audit correction：刪除／降級 `11-enemy-ai` 將 `0x149F8` 稱為傷害／命中評分、將 `+0x22..+0x24` 命名為 AP/DP/HIT 的說法；改為 candidate-builder 與 raw transient/modifier bytes。`13-battle-menu-system` 的 AA／回合摘要標為歷史摘要，`32-item-combat-stats-re` 的 M1 裝備公式標為 normalized-only；README 撤回 `0x29164`＝可見台座素材斷言，與 SDD56 的 TAI#3 opaque boundary 對齊。
- 2026-07-27 transient helper scope correction（supersedes earlier `0x1A30B` wording）：canonical Docker Capstone confirms direct `0x1A866` callers `0x1A4D1→selector 1`、`0x1A55E→0`、`0x1A797→2`; `0x1A30B` is a separate internal selector-2 raw transition/sweep and is not interchangeable with those callers. `0x1A7BD` allocates the `0x111BA(0x1A4D,0,0x40)` resource handle and `0x1A7F1` releases it. Keep selector/campaign semantics fail-closed.
- 2026-07-27 `0x211A4` shared-caller correction：canonical Docker
  Capstone confirms `0x20CE0` item path 與 `0x285ED` opaque selector
  `0x21` path 共用 helper，故函式本身不能命名成 type5/13 專屬 routine。
  後續 dispatcher caller 已獨立閉合 type5/13 的 HP restore/consumption；
  `0x285ED` 上層語意與 renderer 仍 fail-closed。
- 2026-07-27 postbattle hub route adapter：依 `0x2cad7/0x2d093` 與 `0x526b9` raw table 新增 `fdother.ResolveNativePostbattleRoute`，保存 nonzero preparation-first 與 zero-table hub selector→`0x2fc85/0x2e341/0x318ad/0x3072f` address mapping；不執行 scene、不命名 option 語意，invalid input fail-closed。
- 2026-07-27 shop subscene raw plan：Docker Capstone 重核 `0x2e341`，固定 hub variant `3→FDOTHER#29`、`5→#63`、其他→`#12`，service selector `0..3→0x2f0b0/0x2f642/0x2f883/0x2f8ea`；新增 `fdother.ResolveNativeShopServiceRoute`，只保存 resource/callee address，不命名服務或接 runtime UI。
- 2026-07-27 hotel/preparation subscene raw plan：Docker Capstone 重核 `0x2fc85`，固定 FDOTHER resource `13`、selector `0/1/2→0x2ffa5/0x30012/0x301f4`，selector3→`0x19953→0x197e5`；新增 `fdother.ResolveNativeHotelServiceRoute`，只保存 raw order，不命名服務或接 runtime UI。
- 2026-07-27 preparation cap correction：Docker Capstone 完整 body 固定 `0x318ad` 以 raw global `[0x53c03] <= 0x1a` 選 cap15、`>0x1a` 選 cap19；新增 `fdother.NativePreparationPartyLimit`，輸入保持 native index，避免把 late route 任意轉成人類章號。
- 2026-07-27 preparation preview scope correction：Docker Capstone 完整 trace 固定 `0x31e80` 是 selection-table preview consumer：讀 30-byte flags、經 `0x320ce` 計數、依 flag 走 `0x4deda/0x4de56` indexed blit；未見 table/roster writer。撤回把此 body 當 Enter/toggle mutation 的風險，`partyDeploy` mutation 仍與 renderer 分離。
- 2026-07-27 preparation input loop closure（已由 2026-07-29 勘誤）：當時誤記 `0xe0/0x52` 原樣回傳；其餘呼叫端與位元組邊界記錄保留。
- 2026-07-27 progress-stagnation audit：重檢近期對話與 commit 後確認，反覆挖 offset 的主要原因不是工具 blocker，而是把 E0 raw slice／文件勘誤當成玩家進度；`main.go` 仍集中 scene/input/rules/Draw，UI 缺同一 input trace 的 state＋screenshot gate，30 章 postbattle graph 未逐章驗收。後續新增 RE 必須同輪指定 caller/data contract/runtime consumer/regression，UI 還要 E2 artifact；下一里程碑改為 title→dialog→battle→postbattle hub→preparation/town 垂直鏈，未達前停止無 consumer 的孤立 offset 擴張。
- 2026-07-27 UI-01 title trace：將 `titleUpdate` 的三項主選單與 native `0x30550` 四槽 selector 抽成純 `TitleMenuState`／`TitleSlotState`，保留 wrap、24-tick confirm flash、load/cancel 與 bounded no-wrap 行為；Ebiten runtime 與 Docker/Xvfb regression 共用同一 transition。這只閉合 input/state trace，不宣稱原版逐幀畫面或 FD2.SAV 相容。
- 2026-07-27 UI-07/08 campaign menu trace：新增 `campaign.MenuState`，把 `choice/town` hub 的 bounded cursor、空選項 fail-closed 與 confirm→`optN` transition 接到 `campInput`；internal/campaign 與 Docker/Xvfb focused regression 通過。這只閉合可重播 state contract，不命名 town service、不跳過 postbattle handler，逐章 route/E2 仍未完成。
- 2026-07-27 current town screenshot：發現 repo `remake/fd2-linux` 是 7/16 舊 binary；改在 `fd2-go-test-local` 以目前 source rebuild，再用 `FD2_CAMP_NODE=town_ch02` frame30 產生 `docs/figures/town-hub-remake.png`，並加入 README。此 artifact 可證明 GitHub 現在有最新 town hub 畫面，但不代表原版 E2 visual parity。
- 2026-07-27 campaign/UI vertical trace：`Game.stepCampaignMenu` 現由 `campInput` 共用，`TestCampaignTownPreparationInputTrace` 以 `down,down,enter(opt2)` 驗證 `town_ch02→preparation_ch02→story_ch02_pre→battle_ch02`；新增 editable `docs/data/ui-traces/town-preparation-ch02.json`，並用目前 source rebuild 產生 `preparation-current-remake.png`。完整 Docker/Xvfb regression 通過；這是第一個可重播 state closure，仍不代表逐章 E2/native visual parity。
- 2026-07-27 shop vertical trace：新增 `Game.leaveShop` 與 `TestCampaignTownShopPurchaseReturnTrace`，驗證 `town_ch02→shop_ch02_weapon→town_ch02`，reserve 不先扣金、finalize 後扣款，Escape 依 shop node 的 editable `next` 返回；新增 `town-shop-ch02.json` 與 source rebuild `shop-current-remake.png`。完整 Docker/Xvfb regression 通過；native service callee 與原版 E2 visual parity 仍 fail-closed。
- 2026-07-27 church/revive vertical trace：新增 `Game.reviveChurchUnit`／`Game.leaveChurch` 與 `TestCampaignTownChurchReviveReturnTrace`，驗證 `town_ch02→church_ch02→revive→town_ch02`，class1/Lv3/fee7 將 gold 100→79 並 restore HP/OnField；新增 `town-church-revive-ch02.json` 與 source rebuild `church-current-remake.png`。完整 Docker/Xvfb regression 通過；未知 native callee 與原版 E2 visual parity 維持 fail-closed。
- 2026-07-27 church/class-change vertical trace（2026-07-28 校正）：`Game.applyChurchClassChange` 與 `TestCampaignTownChurchClassChangeReturnTrace` 驗證悠妮（identity/portrait09）依 special>optional>default 解析唯一 target portrait34/class21、Yes 確認、MV 5→7、Exp reset、item `0x5a` 消耗與 Escape 回 `town_ch02`；`town-church-class-change-ch02.json` 已同步。candidate 與 confirmation indexed renderer 均已接；`0x1974c` list opening 六幀、`0x19953` confirmation opening 四幀與 `0x197e5` closing 四幀皆由 Draw acknowledgement 推進，Yes mutation 與 No／Escape 返回延至 close 完成。confirmation pulse 依 BIOS low-word delta>=2 令 counter mod4 前進，選中 cell variant=counter/2。`0x2d669` church 主選單 indexed transition、FD2.SAV 相容性仍 fail-closed。
- 2026-07-27 save/load boundary vertical trace：新增 `TestCampaignSaveLoadRestoresTownBoundaryAndParty`，驗證 town 節點 F5 存檔→清除 transient runtime→F9 讀檔後恢復 campaign cursor、gold、items、party roster/deploy/join order/chapter，並由 `enterNode` 清除 battle/shop/church state；新增 `save-town-boundary-ch02.json`。這只閉合 remake JSON boundary，native `FD2.SAV` byte compatibility 仍 fail-closed。
- 2026-07-27 hotel raw route vertical trace：新增 `hotel` node、`Game.applyHotelServiceSelection`／`Game.leaveHotel` 與 `TestCampaignTownHotelRawRouteReturnTrace`，驗證 `town_ch02→hotel_ch02→town_ch02`，selector 0/1/2/3 保留 resource13 與 `0x2ffa5/0x30012/0x301f4/0x19953→0x197e5` order；未知服務名稱與 party/gold mutation 均不猜測，unknown selector fail-closed。新增 `town-hotel-raw-return-ch02.json`。
- 2026-07-27 assertion recheck：重新讀取 `campaign_full.json`，修正 worklist 過時的 `83/0` story-script 統計為 121 個 story/cutscene：9 個 direct script、33 個 handler-bound、79 個 fallback 缺口；撤回「修法已全章接通」的暗示。campaign route regression 維持通過。
- 2026-07-27 story coverage tool recheck：再細分 fallback role；實際為 121 個 story/cutscene、9 個 direct script、33 個 handler-bound、30 個 retreat、23 個 rumor、22 個 unbound postbattle、4 個 generic story fallback。不能把 authored retreat/rumor 或 handler-bound 節點誤報成同一種 unresolved；工具禁止依章號猜測 scene mapping。
- 2026-07-27 generated binding audit：22 個 unbound postbattle 都有 `bindings/generated/*_post.json` skeleton（全節點共 24 個，另 2 個是 active handler 的對照檔），但尚未通過 active override/compile gate；coverage tool 現列出 skeleton 欄位，明確禁止把 generated 檔當成已接 handler。
- 2026-07-27 unbound postbattle guard：`Game.enterNode` 現對沒有 active handler 的 `postbattle_*` cutscene fail-closed，拒絕空 beats auto-advance；新增 `TestUnboundPostbattleCutsceneFailsClosed`，避免未完成 persistence/reward handler 被誤當成直接回 town。
- 2026-07-27 postbattle save guard：`saveGameToSlot` 現對所有 `postbattle_*` 節點拒絕 F5，新增 `TestSaveRejectsUnboundPostbattleBoundary`；未完成 persistence handler 不會建立假 save。
- 2026-07-27 postbattle binding gate audit：新增唯讀 `tools/audit_postbattle_binding_gates.py`，用 handler 原始 call-site address 對 generated binding 的 `loadch/pan/dialog/act/layout` 欄位做缺口盤點；原 22 個 unbound 中，ch09/ch10/ch12/ch18 的 mappings 已通過 Docker compiler regression 並提升為 authored active bindings，ch09 resource37 由 Docker exporter 解碼為一幀 slot5/pose2，剩餘 18 個仍 blocked，沒有猜測 renderer/handler 語意。
- 2026-07-27 ch19 acting resource audit：Docker acting exporter 解出 resource59/60/61/62；resource59 讀 slots53–60、60讀83、61/62讀1。因 ch19 post 還有 `spawn(group 1)`，而 `map18_units.json` group1 僅一筆，尚無 FDFIELD group cardinality／slot identity 證據，沒有把 resource59 硬接成八-slot handler，維持 fail-closed。
- 2026-07-27 ch19 post closure：Docker Capstone 完整 trace 證實 `0x23ec8..0x23f1f` 先 materialize raw records slots0..15、52..60；`0x10b4e(group1)` 以 FDFIELD raw byte+21 掃描，map18 group1=1，且下一個 acting resource60 直接命中 slot83，故入口 frontier=83、spawn group1 後=84。resource59/60/61/62 已由 Docker exporter 解碼並接 authored `ch19_post` binding，runtime context 嚴格保存 `slot_count=83/spawn_groups[1]=1`。
- 2026-07-27 round accounting：Git audit 顯示 `91` worklist 與 `99` reflection log 各有 14 個命名 round；repo 共 1,025 commits，7/25 起 499、7/27 單日 130，故 commit 數不能冒充玩家功能 round。舊畫面集中於 6/28–7/2，最近新增 title/dialogue、overlay、preparation 與本輪 source-rebuilt town hub；後續以垂直鏈 artifact 而非 commit 數衡量進度。
- 2026-07-27 ch04 postbattle closure：Docker-only Capstone 重讀 `0x231f9..0x23480`，確認 `0x2324c→0x233c6` 的三個 7-byte arrays：X=`[12,11,13,10,10,14,14]`、Y=`[11,11,11,9,10,9,10]`、pose=`[2,2,2,3,3,1,1]`；scalar ABI 與既有 ch02/ch29 caller 對照後定案 slots0–6、special slot41 `(12,8,pose0)`、camera raw `(6,4)`→remake pixels `(144,96)`。map4 raw `enemy_ally_total=50`，FDTXT_005 index9 是 ch05 scene5 lines0–16 加 scene6 lines0–1（17+2句）。新增 authored `ch04_post` binding、campaign `postbattle_ch04_persist→town_ch05` 接線與 compiler regression；Docker campaign package test 通過。這只閉合 handler/state/text mapping，未宣稱 renderer parity。
- 2026-07-27 ch05 postbattle closure：Docker acting exporter 解碼 resource27 為 3 beats、slot34/pose2；Docker `parse_field.py extracted/raw 5` 證實 map5 `enemy_ally_total=40`、group3 僅一筆，故 `0x10b4e(3)` 保存為 `spawn_groups[3]=1`，不把 spawn 誤當新角色 JOIN。`0x232b8` raw pan `(5,14)` 依既有 24px tile ABI 轉為 `(120,336)`；FDTXT_006 index6 count-aligned 對 ch06 scene6 lines0–18。新增 authored `ch05_post` acting/binding、campaign `postbattle_ch05_persist→town_ch06` 接線與 compiler regression；internal/campaign Docker test 通過，renderer parity 仍未宣稱。
- 2026-07-27 ch08 postbattle closure：Docker acting exporter 解碼 resource36 為 5 beats、slot47/pose0；Docker `parse_field.py extracted/raw 8` 與 editable map8 交叉確認 `enemy_ally_total=60`、group4 僅一筆，故 `spawn_groups[4]=1`。`0x235d8` raw pan `(6,1)` 依既有 24px tile ABI 轉為 `(144,24)`；FDTXT_009 index4 count-aligned 對 ch09 scene4 lines0–4。新增 authored `ch08_post` acting/binding、campaign `postbattle_ch08_persist→town_ch09` 接線與 compiler regression；renderer parity 仍未宣稱。
- 2026-07-27 ch11 postbattle closure：Docker Capstone 證實 `0x2382b→0x233c6` 複製三組 14-byte arrays：X=`[10,11,9,12,8,10,11,9,12,8,8,12,8,12]`、Y=`[4,4,4,4,4,5,5,5,5,5,3,3,2,2]`、pose=`[2,2,2,2,2,2,2,2,2,2,3,1,3,1]`；scalar ABI 定案 slots0–13、special slot2 最終覆寫 `(10,4,pose0)`、camera raw `(14,0)`→remake `(336,0)`。map11 editable/raw frontier=60。Docker acting exporter 解碼 resource45 為 slot8 special frame0與6；FDTXT_012 index3/4 對 ch12 scene3 lines0–2、3–9。新增 authored `ch11_post` binding/acting、campaign `postbattle_ch11_persist→town_ch12` 接線與 compiler regression；renderer parity 仍未宣稱。
- 2026-07-27 ch13 postbattle closure：Docker Capstone 證實 `0x23942→0x233c6` 複製三組 16-byte arrays：X=`[18,17,19,18,17,19,16,20,16,15,15,16,20,21,21,20]`、Y=`[15,15,15,16,16,16,15,15,15,16,16,16,15,15,12,13]`、pose=`[2,2,2,2,2,2,2,2,2,2,3,3,3,3,1,1]`；scalar ABI 定案 slots0–15、special slot0 最終覆寫 `(0,0,pose0)`、camera raw `(12,10)`→remake `(288,240)`。map13 70-slot frontier、group1 一筆。Docker acting exporter 解碼 resource47 為 slot67/pose2 4 beats；FDTXT_014 index2/3 對 ch14 scene0 lines8–17、scene1 lines0–6。新增 authored `ch13_post` binding/acting、campaign `postbattle_ch13_persist→town_ch14` 接線與 compiler regression；renderer parity 仍未宣稱。
- **歷史、已撤回（2026-07-27；2026-08-09 再收窄）** ch24 postbattle slice：當時曾把 raw `ch24_post`（實際屬玩家第25戰）誤接到 `postbattle_ch24_persist→town_ch25`，後來又把玩家第25戰的 owner 寫成已正式閉合。2026-08-09 的 IDA／Capstone 複核保留「玩家第25戰候選」判定，但 table index 不直接等於玩家戰次；`postbattle_ch25_persist→town_ch26` 尚未具備完整 E2，因此不得視為正式接線。`postbattle_ch24_persist` 仍失敗即關閉，原始 handler 的 dialog/PAN/spawn/ACT 與兩個 raw append 呼叫仍保留；`0x1a/0x0e` 不得命名成 JOIN，不能把歷史 owner 敘述當成現況。
- 2026-07-27 ch25 post evidence：Docker Capstone 固定 `0x24e80` layout caller 的 16-slot arrays（X=`[14,15,15,14,16,14,15,16,13,14,15,16,17,14,15,16]`、Y=`[6,9,6,9,9,10,10,10,11,11,11,11,11,12,12,12]`、pose=`[0,2,0,2,2,2,2,2,2,2,2,2,2,2,2,2]`）、scalar camera raw `(9,5)`→`(216,120)`；map25 raw frontier=70。Docker acting exporter 解碼 resource77/78/79/80，並將 FDTXT_026 string5–11 對到 ch26 scene2/3/4。因 FDTXT_026 整體 raw 61 utterances 與 authored ch26 63 lines 不一致，沒有建立假 count-aligned mapping；只新增 generated layout/acting evidence 與 fail-closed regression，未接 campaign。
- **歷史、已撤回（2026-07-27；後續索引稽核取代）** ch25 post activation：當時新增 `dialogue_overrides` address+text-index schema，讓同一 native dialog call-site 的條件 text index 可各自指向 editable scene/line；不偽造 FDTXT_026 全量 count alignment。ch25 string5/6→scene2 branch、7→scene2 lines9–10、8/9→scene2/3 branch、10/11→scene4 曾通過 compiler regression，當時曾把 `postbattle_ch25_persist` 接到 `town_ch26`。後續 raw owner／一般玩家 E2 與戰間持續隊伍證據不足，該正式接線已撤回；目前 `postbattle_ch25_persist` 維持失敗即關閉，不得把本歷史句子當成現況。
- 2026-07-27 ch25 camera assertion correction：重讀完整 `0x233c6` scalar ABI 後撤回 `(5,9)→(120,216)`；caller push 順序最後兩個 scalar 是 `cam_x=9, cam_y=5`，正確像素為 `(216,120)`。binding、test、SDD、worklist 已同步修正。
- 2026-07-27 ch06 post branch recheck：Docker Capstone 固定 `[0x53ad5]+0x11==1` 才檢查 `unit_inactive(43)`；inactive 走 dialog #5，active 才走 `0x233c6` 9-slot layout（X=`[12,11,13,10,14,10,14,9,15]`、Y=`[4,4,4,5,5,6,6,7,7]`、pose=`[0,0,0,3,1,3,1,3,1]`）、special slot43=`(12,7,pose2)`、camera raw `(6,2)`，再 dialog #4/JOIN12。map6 只有 40 editable units，native slot43/96-slot buffer 尚未有 runtime model，故維持 fail-closed。
- 2026-08-09 ch23 post handler recheck 勘誤：合法 IDA Pro 9.4 與 Docker Capstone 固定 raw table index23→`0x24c1e` 的兩段 loop：第一段每 stage 30 次 `0x11cac(1)→0x17aa9(1)`，第二段每 stage 12 次 raw PUSH `[ESI,255,0]→0x11d40→0x11cac(0)→0x17aa9(1)`；五個 stage 合計 60 次，`ESI=0..59`，不可縮窄成 `0..11`；形式參數是 `0x11d40(0,255,ESI)`。`0x17aa9` 是 BIOS tick wait；`0x11d40` 是全 256-entry DAC 減法／夾零；`0x4dfcc` 則由 IDA `BYTE1(v2)=-32` 與 Capstone `mov ah,0xe0` 固定寫入 DAC `0xe0..0xef`，撤回舊低位索引說法；`0x24d22` 在此 handler 只走非零 setter，寫 raw `0x51a10`。沿 `0x11cac→0x11eee` 追查後確認 case 23 在 BIOS tick 變化時會間接呼叫 `0x24d22(0)`，其 `0x138` bytes row copy 是 312-byte staging 列旋轉消費端；IDA 另固定 `0x11eee→0x122dc→0x127a9→0x1acf3→0x11eb0` 的共用 indexed 消費鏈，直接交叉參照包含 ch23 的 `0x24c63`／`0x24cd3`。因此舊的「沒有消費端」說法已撤回；新證據只關閉靜態 E1 consumer chain，不代表重製已有 raw state/latch adapter。重製端已加入 `RotateNativeCh23Rows`／`ApplyNativeCh23PaletteCycle`／`RunNativeCh23Loop` 原語，executor 僅消費精確 staging 與 raw callback，失敗即回復 buffer，仍不接 campaign。固定版 raw seed 已由 IDA／Capstone 證實為 `0x01`，但入口 latch 的執行期值、`dword_53C03` 生命週期、raw state mapping 與一般玩家 E2 仍未閉合；tick gate 已證實為 `[0x46c] != [0x539f8]` 的 BIOS tick 變化條件，故不命名泛用 renderer，也不接 `postbattle_ch23_persist`。證據見 [`fd2_ch23_post_ida.txt`](../data/ida/fd2_ch23_post_ida.txt)。
- **歷史快照（已由 2026-08-11 勘誤取代，不可作現況證據）** 2026-08-09 ch24 post raw handler recheck：合法 IDA Pro 9.4 與 Docker Capstone 共同固定 table index24 的 raw entry `0x14df2→0x24df2`，以及獨立相鄰 index25 `0x14e80→0x24e80`。`sub_24DF2` 順序是 FDTXT_025 index6、PAN raw `(4,16)`、raw `0x10b4e(2)`、ACTING75、FDTXT_025 index7、`0x112a5(0x1a)` append、`0x11506`，再以 raw `0x1d` 跳入共享 `0x237c8` 尾段；共享尾段另有 index3、`0x11506`、`0x112a5(0x0e)`。初次記錄只保留 raw append 與未知的隊伍／章節用途；當時暫記角色表索引26「聖寇拉斯」與14「珊」，但後續 Capstone 已證實中途跳入會略過 direct-entry 的 `push 0x0e`，實際消費29，故舊的14解讀已撤回。table index不直接等於玩家戰次；map24／文字／ACTING75 只支持玩家第25戰候選，先前 `postbattle_ch24→town_ch25` 同號接線已撤回，`postbattle_ch24_persist` 仍 fail-closed。完整固定版雜湊、IDA 函式範圍與 raw table bytes 見 [`fd2_ch24_post_ida.txt`](../data/ida/fd2_ch24_post_ida.txt)。
- 2026-08-09 ch24 `0x10b4e` materializer recheck：IDA Pro 9.4 固定 `sub_10B4E` 會讀取 FDFIELD current field、以 row `+0x15` 比對 group，並逐筆呼叫 `sub_10C50` 建立 0x50-byte runtime record。原始 map24 resource073（1951 bytes，MD5 `d64cca11484662bd45ab6c34aeb63ff9`）共有70筆列，group分布 `0=46/1=8/2=1/255=15`，group2唯一列索引54與 authored `map24_units.json` 第54筆一致；因此候選 binding 的 `spawn_groups["2"] = 1` 已獲得列數閉合。這不改變 `0x1a/0x1d/0x0e` 未知、玩家第25戰僅為候選、`postbattle_ch24_persist` 與戰間城鎮／商店／整備／存檔仍 fail-closed 的判定。證據與 SHA-256 見 [`fd2_ch24_post_ida.txt`](../data/ida/fd2_ch24_post_ida.txt)。
- 2026-08-09 ch24 immediate identity refinement：`sub_112A5` 的固定 32 列角色表與 `characters.json`／`native_character_catalog.json` 對照，已證實 raw `0x1a`（26）是聖寇拉斯、raw `0x0e`（14）是珊的建構器角色索引；仍未知的是該次 append 是否永久 JOIN、臨時演出或其他隊伍操作，`0x1d` 也未獲得章節／分支語意。這將舊記錄的角色身分範圍收窄，但不解除 `postbattle_ch24_persist` 或戰間城鎮／商店／整備／存檔的 fail-closed gate。
- 2026-08-09 ch29 terminal body recheck：合法 IDA Pro 9.4／Docker Capstone 完整固定 `sub_2BCE5(0x2bce5..0x2c39b)` 的兩個 caller（`0x2545d`、`0x25970`）與 `sub_2C405(0x2c405..0x2c9ec)` 的獨立邊界。前綴 raw 順序為 FDOTHER `#0x36` frame0/9、`0x11df2` ramp、三輪重複 ramp、frame `0x0c..0x6c`、40／200 次 indexed composite；`0x2c172` 後呼叫 `0x2c405`，後者先 `0x1088d(0x1e)`、500次 raw staging，再進 `0x2c548` montage。`0x10620` 只比較 `word[0x41a]`／`word[0x41c]`，`0x4e031` 只複製前者，沒有已證實按鍵映射；`0x25975` self-loop 與 `0x28a64` 清理尾端均不是 campaign 返回。完整 fixed hash／raw body 見 [`fd2_ch29_terminal_body_ida.txt`](../data/ida/fd2_ch29_terminal_body_ida.txt)。indexed owner、一般玩家 E2、輸入事件與終局 campaign handoff 仍 fail-closed。
- 2026-08-09 ch28 post／玩家第29戰 candidate recheck：raw table index28 bytes `8c 54 01 00`→線性 `0x2548c`，是獨立於 index29 `0x25757` 的 `sub_2548C`。IDA／Capstone 固定 FDTXT_029 string10..15、raw `0x35bba(20)`、`0x10b4e(9)`、PAN `(9,8)`、`0x12cea(10,15)`、`0x22253` 五個 raw push、`0x24b4d` 三段過場、`0x35e5a` 三次、兩段 palette loop、`0x11506` 與 `inc [0x53c03]` 返回；count-aligned map 將文字保留在 ch29 scene2..4。這與現行 battle_ch29/map28 支持玩家第29戰 post 候選，但未知 indexed renderer／E2 仍未閉合，`postbattle_ch29_persist→preparation_ch30` 不接線。完整雜湊與 raw body 見 [`fd2_ch28_post_ida.txt`](../data/ida/fd2_ch28_post_ida.txt)。
- 2026-08-09 ch29 `0x35bba(20)` raw boundary：IDA／Capstone 固定它從
  runtime index20 起逐筆清除 0x50-byte record 的 `+0x40`，再呼叫有七個
  其他 caller 的共用 `0x1db65`。`0x1db65` 讀取 raw `+0/+1/+5/+0x40` 並進入
  共用 indexed 呈現／更新鏈；這只刪除「0x35bba 完全未知」的斷言，不命名
  `+0x40` 成 HP／狀態，不把 `0x1db65` 接成 ch29 renderer 或 campaign owner。
  indexed frame、一般玩家 E2、town／shop／整備／save handoff 及
  `postbattle_ch29_persist` 仍保持失敗即關閉。完整 raw 位址與雜湊見
  [`fd2_ch28_post_ida.txt`](../data/ida/fd2_ch28_post_ida.txt)。
- 2026-07-27 native item-row producer closure：逐 byte 比對 EXE 證實
  `0x4e56c` 的 linear `0x602ad` 對應 file `0x540ad`，比既有 normalized
  `item.json` 的 `0x540ac` 起點向後一 byte；stride 同為 23，故 raw row
  尾 byte 會跨入下一 normalized row 的前導 byte。`dump_exe_tables.py`
  現另產 `native_item_effect_rows.json`，docs/remake 保存 215 個已知 ID 的
  byte-exact prefix；Python shift regression 與 Go consecutive-ID/23-byte
  loader regression 已補。這不證明 native table 在 ID214 結束，未知欄位
  仍不接 normalized gameplay。
- 2026-07-27 equipment word cross-check：以新 raw fixture 對 215 個已知
  selector 全表驗證，runtime row `+1/+3/+5/+7` little-endian words 與
  normalized `item.json` AP/HIT/DP/EV 全數一致；`0x1145a` 的 native
  accumulation order 因此定案為 AP/DP/HIT/EV。Go regression 固定兩份
  fixture contract；未命名 effect bytes 與 table 終點仍 fail-closed。
- 2026-07-27 type8–10 gameplay closure：結合 `0x21082` mutation/removal、
  `0x1145a` base/equipment data flow 與全表 cross-check，定案 item effect
  type8/9/0xa 分別把 row `+0xe` 永久加到 base AP/DP/DX
  (`+0x37/+0x39/+0x3e`)，再重算並移除來源 slot；raw IDs198/199/200
  amounts=9/9/7。route 已型別化；presentation selectors、名稱與共用
  callee 的 type17–19 不在此項證據範圍。後續已由獨立欄位
  producer/consumer 閉合，見本文件尾端 type17–19 項目。
- 2026-07-27 type5/13 HP item transaction：Docker Capstone 重讀
  `0x20c6f` 尾端，確認兩者都以 row `+0xe` 經 `0x211a4→0x1c916`
  恢復 target-list current HP、cap max HP；type5 隨後跳 `0x1b8e7`
  消耗來源 slot，type13 直接 cleanup 並保留來源。新增
  `NativeItemHPRestoreRoute`／`ApplyNativeItemHPRestore`，保存 sequential
  RNG 與 target/source atomic preflight；IDs192/195/211 fixture regression
  固定 40/999/200 amounts 與 consumption branch。道具顯示名稱與 renderer
  不猜測。
- 2026-07-27 type11 MP item transaction：Docker Capstone 重讀
  `0x20db3..0x20e4c` 與 `0x1c9dd`，固定 row `+0xe` amount、current/max
  MP `+0x44/+0x46`；max MP 零的 target 直接走 `0x1e1dc`，不呼
  `0x1c9dd`、不前進 RNG，其他 target 依 list 順序 restore，最後跳
  `0x1b8e7` 消耗來源。新增 `NativeItemMPRestoreRoute`／
  `ApplyNativeItemMPRestore` atomic transaction；IDs206/207 fixture
  amounts=80/200。renderer/名稱仍 fail-closed。
- 2026-07-27 type12 HIT/EV item route：dispatcher 對 type12 呼
  `0x22997` 後直接 cleanup，沒有 `0x1b8e7`；callee 以 marker
  `+0x24` gate，成功才前進 RNG、寫 `(rng%4)+2`，並把已定案的
  derived HIT/EV `+0x4c/+0x4e` 各加15。新增
  `NativeItemHITEVStepRoute`／`ApplyNativeItemHITEVStep`，ID210
  fixture regression 固定 retained-source contract；marker UI 名稱不猜。
- 2026-07-27 type15/16 AP/DP item routes：dispatcher 分別走
  `0x22866/0x22721` 並直接 cleanup，來源保留；type15 marker
  `+0x23`／derived DP `+0x4a`，type16 marker `+0x22`／derived AP
  `+0x48`，成功增加 `trunc(current×0.15+1)`，marked target 不耗 RNG。
  新增 typed route/executor 與 IDs213/214 fixture regression。
- 2026-07-27 type14/22 marker application + correction：完整 Docker
  trace 證實 marker/class gate 後依序消耗 gate RNG、`0x1c81f` damage
  RNG、marker RNG；base amount10 實際減9 HP，而非舊斷言 fixed10。
  type14/22 markers=`+0x26/+0x27` 且來源保留。新增
  `NativeItemMarkerApplicationRoute`／executor 與 IDs212/57 regression；
  `ExecuteNativeCommandApplication` 亦修成三 draws。status/UI 名稱未知。
- 2026-07-27 type6/7 marker-clear item + ABI correction：Docker 重讀
  `0x22af6` 證實 marker 是 target `record+a5`，不是舊 adapter 的 detached
  flags。type6/7 用 `+0x25/+0x26`；nonzero 時 base10 經 RNG 實際恢復
  9 HP、清 record marker，dispatcher 再消耗來源。修正
  `ApplyNativeRawFlagRestore` API，新增 atomic typed transaction 與
  IDs196/197 fixture regression；status/presentation 名稱不猜。
- 2026-07-27 type17–19 capacity/MV closure：dispatcher 與
  `0x21082` 固定 type17/18 將 row amount20 加到 max HP `+0x42`／
  max MP `+0x46`。type19 以 amount1 對 word `+0x3b` 相加，但 caller
  在呼叫前保存 `+0x3c`、返回後恢復；既有 class-change producer 已將
  `+0x3b/+0x3c` 定案為 MV/EXP，因此 net effect 是 MV byte +1、EXP
  不變。三條都由 callee `0x1b8e7` 消耗來源；新增 typed route/executor、
  source preflight 與 IDs94/95/96 fixture regression。
- 2026-07-27 type21 command-damage item + address correction：official
  IDA 9.4 固定 `0x20c6f` 將 row word 傳進 `0x2111a`；後者以該 word
  作 `0x1c75e(target,commandID)` 的 command ID。Docker Capstone 證實
  `0x1cac7` 是 allocation／`0x1cb94` drawing／四輪 320×192 present，
  不是 MP subtract；真正的 subtract helper 是相鄰 `0x1ca89`，而 type21
  不呼叫它。dispatcher 亦不呼 `0x1b8e7`，故來源保留。IDs29/38/51/99
  對 commands6/1/7/6；新增 typed executor、全 target preflight 與 fixture
  regression，未命名道具顯示文字或宣稱 presentation parity。
- 2026-07-27 type20/24 command-damage extension：official IDA 9.4 重讀
  `0x20c6f` 與 `0x1cd17`，確認兩 type 在十幀 indexed presentation 後，
  同樣把 row word 當 command ID 逐 target 呼 `0x1c75e`；動畫本體只做
  saved-buffer restore、target redraw、312×192 present 與 BIOS tick。
  dispatcher 不扣 MP、不移除來源。type20 IDs11/56/60→commands2/0/2，
  type24 ID79→command3；共用 typed executor 已擴充並新增 fixture regression。
- 2026-07-27 type23 item relocation closure + gate correction：official
  IDA `0x1bbdc` 證實 actor gate 是 runtime identity `+8==24` 與 max MP
  `+0x46>=20`，撤回舊「class/level gate」。Docker Capstone 展開
  `0x2218a`：只取 first target，呼 `0x1ca89(actor,23)` 依 command23
  cost20 對 current MP 做 16-bit subtract；target class9..24 時 level
  額外+30，再乘10加 raw accumulator。兩次 `0x22253` 依序寫
  `0xff/0xff` 與 destination cursor bytes；dispatcher 不移除 item ID101，
  row word1 未被 handler 使用。新增 typed relocation executor、MP wrap、
  identity/maxMP/target atomic preflight 與 fixture regression；mode6
  indexed UI/renderer integration仍獨立待辦。
- 2026-07-27 type23 mode6 destination legality：official IDA
  `0x115b6` 與 Docker raw table cross-check 固定 other-unit occupancy
  (`same x/y && +5 bit0==0`)、target-derived selector（class；`+7==0x1c`
  override1；class `0x13`／race4,5 override19）與
  `0x4e555(selector)[terrainIndex]==20`。`0x61646..0x61889` 精確為
  29×20 bytes，下一 byte `0x6188a` 已屬 class compatibility table；
  exporter 新增 editable `native_movement_cost_rows.json`，Go strict loader
  與 `NativeRelocationDestinationAllowed` regression完成。這閉合 legality
  predicate，不等於 cursor/indexed renderer 已接入 GUI。
- 2026-07-27 original item selector input/layout correction：official IDA
  `0x1b932→0x1b9de→0x184c0` 證實 signed flag非負的 raw slots會 compact
  顯示；native inventory writer維持 occupied prefix。layout為兩欄四列，
  ↑/↓在 prefix linear wrap、←/→±4；battle-use mode Enter/Space只接受
  selected item row effect type非零，Esc cancel。display index n 的 label
  `(42+150*(n/4),103+22*(n%4))`，FDTXT index=`itemID+181`，
  color201/205；category icons59/60/61、equipped+3，stat icons
  64/65/66/67/41。新增 pure input/layout adapters與真實 row regression；
  舊 raw-hole shell明確降級為provenance/debug UI，非original parity。
- 2026-07-27 item selector panel schedule：official IDA閉合
  `0x17e0b/0x1b932→0x18409`，opening=11→0、closing=0→11，每幀先
  restore saved 64000-byte framebuffer。三個 raw copy region為left
  `src(5,7) 86×86`（frame6後向左clip16px，frame11寬11）、
  upper `src(92,7) 223×86`（frame3後向上clip16px，frame9起off）、
  bottom `src(5,94) 310×102`（dest y=94+16f，frame6起off）。
  `NativeItemPanelSchedule` 保存exact rectangles/reverse ordering；
  尚未宣稱 indexed source/buffer 已接 Ebiten。
- 2026-07-27 item panel source/data closure：official IDA 9.4 重核
  `0x17eef/0x17fc0`。base先以 `0x168b6(dst,320,5,7,5,5)` 建
  `(5,7)` 的5×5框；unit record `+7` 選 DATO貼 `(8,10)`；
  FDOTHER#5 LMI1 entries20/21（directory offsets `+86/+90`）貼
  `(92,7)`／`(5,94)`。後續2 bar、4 compared-number、8 raw-number、
  3 FDTXT與4組icon的 destination/record-offset schedule均已資料化為
  `NativeItemPanelBaseLayoutFor`／`NativeItemPanelDataPlanFor` 並有
  regression。doc35 舊「`[0x53a81]` loader待確認」已刪；boot
  `0x25c97` 明確載 FDOTHER.DAT #5。尚未接 indexed→Ebiten renderer。
- 2026-07-27 `0x168b6` planner重大更正與 item base compositor：舊
  ending-only formula 漏掉 `v6=dst+stride*a4+a3` 的 `a3=5`，且數個
  placement混用 byte/stride，故撤回「exact arithmetic」斷言。共用
  `fdother.PlanNativeDialogueFrameGrid` 修正首 offsets為2245/2328、
  portrait-overwritten grid origin為3208、尾格23752；ending已改用它。
  `RenderNativeItemPanelBaseResources` 以玩家 archive執行 corrected
  49-cell grid→DATO frame0→FDOTHER#5 entries20/21並 atomic commit。
  另由 `0x4e8af` store loop確認 index0也是 opaque，新增
  `LMI1Entry.BlitOpaqueAt`，刪除 UI matrix舊 transparent斷言。
  dynamic `0x17fc0` overlay與 Ebiten bridge尚待。
- 2026-07-27 item text-helper correction：合法 IDA Docker重核
  `0x15f84/0x16559`。一般 word由
  `0x4ea2a([0x53a75],glyph,dst,stride,fg,shadow,bg)` 使用
  FDOTHER#4 16×16 1bpp font；`0x16559`只從目前 DATO
  `[0x53a85]` offset table取 mouth frame作 portrait animation。
  doc14/doc35 舊「`[0x53a75]`文字區底圖／`[0x53a85]`字模容器」
  斷言已刪。item panel call style為205/76/0。
- 2026-07-27 complete item-panel indexed compositor：
  `RenderNativeItemPanelData/Resources` 現執行完整 `0x17fc0`，包含
  `0x18795→0x17d6f` bars、`0x1875d/0x187d6` digits/overflow、
  entries53–57 icons與三段FDTXT#0/FDOTHER#4文字；三種 codec不混用，
  record `+7` 直接選 DATO，整張 base+overlay atomic commit。
  control-bearing text拒絕。玩家資產 oracle由
  `cmd/fd2-item-panel-oracle` 產生
  [`item-panel-native-indexed.png`](../figures/item-panel-native-indexed.png)。
  尚未接 Ebiten item input與12-frame presentation。
- 2026-07-27 item rows與Ebiten runtime closure：
  `RenderNativeItemPanelRows` 完成 `0x184c0` compact rows之 raw/frame
  icons、FDTXT `ID+181`、selected201/unselected205與stat values；
  oracle PNG已更新有兩筆真實item rows。新增 strict
  `NativeItemPanelRecordForUnit`，raw `+6/+8/+0x1f/+0x20`、DATO與八格
  inventory缺一即拒絕。`cmd/fd2` 現使用完整 indexed panel、
  `AdvanceNativeItemSelector` compact四方向input，以及opening11→0/
  closing0→11 clipped Ebiten player；缺archive/provenance才fallback。
  Xvfb玩家archive test跑完整開關。後續實際資產audit曾發現正常campaign
  主角沒有constructor provenance，因此當時正式流程仍fallback；下一條
  JOIN lifecycle closure已修正此限制。Enter effect transaction仍封閉。
- 2026-07-27 JOIN/item-panel lifecycle closure：Docker Capstone重核
  `0x112a5`，證實join id直接查lower record `0x4e4e8`與growth
  `0x4e4d1`，record byte0/1寫runtime `+0x1f/+0x20`；class-change
  `0x31571..0x3157a`只改`+0x20`與`+7`，不改race。新增獨立raw
  race/class欄位，30章scenario由32筆characters/lower-table逐筆交叉核對後
  輸出JOIN identity/race/class與8格defaults；persistent overlay保存raw
  identity/class/transient/inventory flags，轉職寫回raw class。正常ch01
  scenario加玩家archives已通過原版Ebiten item panel integration test。
- 2026-07-27 first complete item Enter family：Capstone補核
  `0x1bbdc` success在`0x20c6f`後呼`0x13512`、回傳1，caller退出action；
  IDs198/199/200的type8/9/10及IDs94/95/96的type17/18/19 rows皆為
  selection/effect mode0、target code1。新增Unit-level `0x21082`
  projections，經兩段target planner驗證self-target後，處理raw base
  AP/DP/DX或MaxHP/MaxMP/MV（MV保存EXP）、`0x1b8e7` compact移除來源、
  必要的equipment recompute並設raw `+5 bit7`/normalized Acted。
  缺base provenance原子拒絕；RNG及非self效果仍封閉。
- 2026-07-27 native RNG lifecycle與回復道具接線：Docker解析LE object
  table、fixups及`0x627b8` image bytes，確認RNG word初值0，唯一references
  是`0x4e893`的load/store，無FD2.SAV或chapter reset，因此是process-wide
  非persistent state。Ebiten現以獨立`uint16`串接types5/13 HP與type11 MP：
  item row先進兩階段target planner，確認後才atomic建立所有runtime raw
  records、按target list推進RNG、同步HP/MP與compact inventory，最後設
  raw `+5 bit7`並結束action。缺任何raw record即拒絕；effect renderer
  與其餘typed item families仍未接。
- 2026-07-27 targeted item batch 2與command RNG correction：types6/7、
  12、14–16、20/21/24已接同一兩階段target runtime，保存marker
  clear/application、conditional HP、AP/DP/HIT/EV、retained/consumed
  source與raw Unit同步。接type20/21/24前重讀`0x1c75e/0x1c81f`，
  發現舊`NativeCommandDamage`錯用Go math/rand；Capstone證實
  `0x1c7ed`及`0x1c869`皆call `0x4e893`。現player command0及item
  command damage共用process-wide uint16 state（miss 1 step/hit 2）。
  indexed effect presentation仍未接。
- 2026-07-27 type23 runtime closure：item101 first-target確認後現進獨立
  destination cursor，不把角色確認誤當目的地確認。每格以raw roster
  occupancy、exact `NativeTerrainMoveCodes`與editable 29×20 rows走
  `NativeRelocationDestinationAllowed`；合法確認才由
  `ApplyNativeItemRelocation`扣command23 MP、寫target `+0/+1`，來源保留、
  actor結束action。Escape回target selector；缺terrain/raw provenance
  fail-closed。`0x22253`的27-present indexed renderer仍未接。
- 2026-07-27 raw pose/motion/cycle lifecycle：Docker Capstone確認玩家
  materializer `0x10a77..0x10aad` 與 FDFIELD constructor 都初始化
  runtime `+3/+4=0/0`。四向move entries先固定pose，逐格寫motion
  `1..6`，第七拍改X/Y後清0並保留最終pose；`0x1366a` normal同規則，
  special只改pose。另補齊`0x1297d`：moving `[0x53c07]`每call都循環，
  idle `[0x53c0b]`才受timer delta gate。`[0x46c]`再由`0x17aa9`
  的0x10000 wrap busy-wait及`0x16d00` two-tick gate證實為low 16-bit
  BIOS timer tick，不是VGA scanline。新增完整pure cycle helper/regression；
  doc54刪除舊錯位acting dump影片斷言。GUI仍需統一的battle-local raw
  presentation state及monotonic BIOS-tick/call timing才可接。
- 2026-07-27 `0x22253` snapshot ownership：整個routine只有一塊
  `0x25680` allocation。intro每frame restore terrain-only snapshot；
  `0x22547` entry把final LMI `#0x7c`寫入同一snapshot後，6 contract與
  10 release都restore它。中間coordinate mutation/strip-copy不改snapshot。
  新增atomic `ComposeNativeUnitPresentLUTSnapshot` regression；撤回兩phase
  可使用不同未知snapshot的舊註解。完整runtime scheduler仍待raw roster
  materialization與bridge/present call timing。
- 2026-07-27 `0x22253` direct-VGA bridge：撤回「合計27 presents」作為
  完整schedule的說法；它只計11+6+10個`0x11eb0`。contract後helper回傳
  FDOTHER #3 entry0 pointer+1，caller以此做bridge-only `0x22046`後，
  逐row由work stride456 direct memmove 24 bytes到VGA stride320，每row
  delay10ms。targetY==cameraY寫18 rows；否則由target上方6 pixels起寫
  24 rows。新增exact layout、progressive visibility與preflight bounds
  regression；`ComposeNativeUnitPresentStripBridge`將restore→額外LUT/
  object redraw→direct rows接成transaction，並驗證不會誤做full viewport
  copy。完整可見序列為27 full presents + 18/24 direct row writes。
- 2026-07-27 bridge shifted-LUT closure：真實FDOTHER #3 directory
  offsets為`0x66,0x166,0x266...`。`0x22547`經shared epilogue回傳entry0
  pointer+1，`0x22046`仍取256 bytes，故bridge LUT必為
  `entry0[1:256]+entry1[0]`，不等於aligned LUT0/1。新增
  `NativeUnitPresentBridgeLUT`與真實archive regression；runtime不得以
  release LUT0替代。
- 2026-07-27 `0x22253` caller ABI：五個call sites證實參數順序為
  `(unit,newX,newY,visualX,visualY)`。visual pair先供intro/contract，
  之後record `+0/+1`才改成new pair並進bridge/release。command23以
  `ff/ff,current`消失再以`destination,destination`出現；ending只做
  unit1消失；script helper兩pair相等。新增
  `PlanNativeUnitPresentCall` regression，禁止用模糊source/destination
  名稱交換兩對座標。
- 2026-07-28 native map runtime-state bridge：新增battle-local
  `NativeMapPresentationState`，只在selector slot成功materialize時建立
  raw `+0/+1/+3/+4=(X,Y,0,0)`。一般battle walk與`decode_acting`
  現共享原版來源格motion `1..6`、第七拍提交目的格並清motion的狀態
  序列；placement／pose writers同步raw state。`NativeUnitLayerEntry`
  要求presentation、selector slot與record byte5完整才輸出
  `0x127e0` entry，否則fail-closed。State亦開始擁有
  `[0x53c0b]/[0x53c07]/[0x53c0f]` cycle globals。聚焦
  fdicon/indexedmap/battle/campaign/cmd tests已在
  `fd2-go-test-local`+Xvfb通過。限制仍是實時間距：七拍目前跟Ebiten
  Update走，尚未接monotonic約18.2Hz BIOS clock，也尚未把raw roster
  組成production `NativeFrameInput`；不得稱完整原版UI renderer。
- 2026-07-28 official IDA scheduler recheck：合法
  `fd2-ida-authorized-local`以容器內`idapyswitch`選Python3.12後成功
  Hex-Rays `0x11cac/0x122dc`；repo的`ida_dump_pseudocode.py`新增
  `FD2_IDA_OUTPUT/FD2_IDA_ADDRESSES`，避免該entrypoint誤解析帶參數
  `-S`。新證據是`0x122dc` default branch確實no-op，且bootstrap
  `0x10483`寫`[0x51a83]=0`後立即`0x11cac(1)`；修正renderer錯把
  mode0當invalid的阻塞。mode6仍是field byte mutation，不是drawable。
- 2026-07-28 raw frame-roster admission：`Unit`新增`HasBattleFig`以區分
  explicit FDFIELD `unit+7`與舊JSON的`Fig` fallback；
  `NativeMapFrameRoster`要求每個unit具備raw presentation、selector、
  byte5、unit+7、race、class，原子化輸出unit/foreground arrays與cycles。
  任一缺失整批fail-closed。fdother/indexedmap/battle/cmd聚焦
  Docker/Xvfb tests通過；production frame仍待clock/camera/HUD globals。
- 2026-07-28 production native-map bundle補齊已證實的FDOTHER #1
  20-entry range-overlay bank；`nativeMapAssetsAvailable`缺此bank即整包
  fail-closed，不能只載terrain/unit後宣稱完整steady compositor資源。
  完整Docker/Xvfb `go test ./... -count=1`通過。這仍未改畫面輸出；
  原版320×200相機與remake 640×400視野、terrain phase、HUD globals
  尚未對齊前不做猜測性切換。
- 2026-07-28 terrain/global timing closure：官方IDA data-xref與Hex-Rays
  加Docker Capstone指令級確認，`0x11eee`只有在
  `[0x51a93]==-1`且signed BIOS low-word delta>2或wrap時才將
  `[0x53c1f]` modulo20前進；override0..19直接寫phase、不更新
  `[0x539f4]`。terrain flip `[0x53a40/0x53a00]`與unit pixel shift
  `[0x53a04/0x53a08]`是兩組獨立new-tick toggle，不能與phase或
  sprite cycles混用。新增`NativeTerrainPhaseState`、
  `NativeBinaryTickState`及battle ownership/regression。
- 2026-07-28 strict `NativeFrameInput` admission：
  `buildNativeMapFrameInput`現原子化結合original banks、raw map cells、
  selector cache、unit/foreground roster、selected LUT及所有已物化
  cycles/flips；editable control bytes與FDSHAP不一致即拒絕。
  tile-space camera、compositor可畫的selector0..5、cursor、HUD input
  仍須caller明示；這不是battle raw dword的完整domain，
  不從現行640×400 camera或normalized highlight猜值。這使下一個工作
  明確收斂到native camera/HUD/monotonic clock runtime，而非繼續堆
  disconnected compositor primitives。
- 2026-07-28 HUD assertion/runtime correction：刪除doc14將
  `[0x53ab9]/[0x53abd]`稱為對話框寬高、並據此命名
  `0x165ac/0x16b43`為對話框zoom的舊斷言；direct cursor writers證實
  兩者是camera-relative visible cursor column/row。原始data固定
  anchor初值1、gateA/B初值1，但`0x10010`會從native-save plaintext
  `0x30d2`覆寫gateA，故不能hardcode。新增
  `NativeMapHUDRuntimeState`與anchor retention tests；
  `drawNativeMapHUD`現在要求raw state及完整optional unit/HP provenance，
  移除舊true/true/1 partial path。
- 2026-07-28 native save current-runtime map：`tools/fd2save.py`現在列出
  plaintext `0x30c3..0x30d4`的turn counter、runtime count、
  persistent count、chapter、
  camera XY、absolute cursor XY、visible cursor XY、raw globals與
  HUD gateA。player FD2.SAV真實decode/checksum驗證通過；這提供後續
  camera/HUD exporter的byte-level來源，不把slot垃圾資料當active state。
  2026-07-29 IDA 勘誤：header `+0` 是 `[0x53bef]` turn counter，
  `+1` 才是 `[0x53beb]` runtime count，`+9` 才是 `[0x53bfb]`
  persistent count；舊工具把 `+0` 顯示成 persistent count 是錯誤斷言。
- 2026-07-28 camera/cursor executable bridge：direct四個cursor helpers
  `0x11b48..0x11cac`已落成`NativeMapViewState`。它保存
  `visible=absolute-camera`，以原版上/左 `<2`、下 `>5`、右 `>10`
  threshold及13×8 viewport捲動；keyboard與touch都經同一state machine，
  不再由640×400 normalized camera覆蓋。`battle_ch01`新增可編輯
  `native_map_view/native_map_hud`，內容取自玩家FD2.SAV：
  camera `(1,13)`、cursor `(8,17)`、visible `(7,4)`、HUD `(1,1,1)`。
  campaign loader要求兩組欄位成對且raw範圍合法；其餘章節在取得
  chapter-specific來源前維持未materialize，禁止複製ch01值。
- 2026-07-28 steady native frame production milestone：Capstone重讀
  `0x11d12..0x11d36`撤回width320舊斷言，exact copy是work
  `+0x8088`／stride456的312×192→VGA `+0x504`／stride320，
  即320×200畫面四邊4px。compositor、atomic regression與Ebiten bridge
  均已修正。ch01舊map JSON缺576個blit modes/1200-byte controls，
  已以合法原版和現行exporter重建；scenario改成party-first再append
  initial groups的native constructor lifecycle。玩家DAT integration test
  與Docker/Xvfb production screenshot均通過。
- 2026-07-28 BIOS/VGA palette correction：`nativeBIOSClock`以
  `1193182/65536≈18.2065Hz` battle-local monotonic tick驅動一次
  `0x1297d`及terrain/binary latches，保留signed-low-word wrap。
  `ParseVGAPalette`不再錯把DAC index0全域透明；完整VGA palette全opaque，
  只有zero-source透明blitter自行clone alpha0。artifact：
  `docs/figures/native-map-ch01-remake.png`。此段已被下方 selector-lifecycle
  correction 取代；動態target/flash仍未閉合，不宣稱完整battle UI parity。
- 2026-07-28 `[0x51a83]` domain correction：合法 IDA 9.4 完整
  data xrefs 保存為`docs/data/ida/fd2_51a83_xrefs.txt`。撤回舊
  `0..5` bound、`flagA`與doc37「戰鬥訊息索引」斷言。完整xref後逐一
  用Capstone核對variable writers：非零來源是zero-extended
  command/item record byte `+2`，其餘eax/esi writers在控制流上均為0；
  原表range 5/7/9
  實際產生selector 7/9/11。`0x122dc`對>6 no-draw，不代表無作用：
  `0x115b6`仍把selector-1送入`0x14742` target legality。
  battle state改保留observed `0..0x101` domain；此處campaign selector0
  斷言已被下方 lifecycle correction 撤回。
- 2026-07-28 `0x115b6→0x14742` cursor-confirm closure：合法IDA與
  Docker Capstone證實`0x14742`唯一caller是`0x1175f`。Enter/Space時
  code5先拒絕；cell byte+3=`0xff`亦在code4前拒絕；非`0xff` code4接受；
  codes0..3才以selector（>1減1）作strict Manhattan `<radius` raw
  roster count，count非零才確認。code6維持既有relocation branch。
  新增fail-closed `NativeCursorConfirmationAllowed`與identity regression。
  同次逐跳修正`0x14818/0x14742` code2為camp `==1`；舊`!=1/non-allies`
  斷言與Go resolver已撤回。
- 2026-07-28 interactive selector lifecycle correction：Docker Capstone
  固定setup `0x10483`先寫0並呼`0x11cac(1)`，`0x105eb`呼`0x11cac(0)`，
  `0x1060c`再寫persistent selector1；故0只是transient opening no-op，
  campaign/runtime steady state已改為1。`0x1cff0`另證實target entry寫
  `record+4+2`，cancel/effect期間暫寫0，exit恢復1。remake已接此狀態轉移
  與exact cursor-confirm gate，並由FDOTHER#1 descriptor0呈現原生steady
  cursor；白色方框已移除。動態target overlay/flash與indexed effect renderer
  仍維持fail-closed／playable fallback。
- 2026-07-28 drawable target-overlay production bridge：`composeNativeMapFrame`
  與Draw modal gate已開放exact `0x122dc` call-table存在的selectors1–5；
  `nativeCommand0Targeting`即使保留actor `g.sel`也能由原生indexed frame覆蓋
  normalized橘色highlight。regression逐一要求selectors2–5相對selector1
  產生不同VGA bytes。selector0 transient、selector6 field mutation及7+
  no-draw values仍不進production compositor；不宣稱target flash／effect完成。
- 2026-07-28 runtime target-grid／palette correction：Docker Capstone重讀
  `0x4dbfc/0x14818/0x4e040/0x4e16e`，撤回兩個錯誤斷言。serialized
  FDFIELD byte+3不是live renderer state：`0x4dbfc`先全填`0xff`，
  `0x14818`才寫target remaining-budget grid；原先直接使用archive zeroes
  讓整張steady map錯走LUT branch，實機畫面偏灰。`battle.Load`現保留
  archive欄位作provenance但runtime初始化`0xff`，command target entry建立
  first-stage grid，cancel/success reset。另bit`0x80`不是zero-cost：
  `0x4e19a`會把已扣cost的destination budget強制歸零，因此該格是可達終點，
  不能形成zero-cost chain。focused regression與更新後
  `docs/figures/native-map-ch01-remake.png`證實正常飽和indexed terrain。
- 2026-07-28 item target／type23 selector lifecycle correction：direct
  `0x1bbdc` caller evidence固定item entry將global selector寫成
  `row[+0x12]+2`，first target field則以`row[+0x10]`、type23 inner marker
  與`row[+0x15]`建立。第一次`0x115b6`返回後立即`0x4dbfc` reset並恢復
  selector1；second/final target list建立後再reset。type23 destination呼叫
  `0x115b6(6,...)`的6是literal target code，global selector仍為1；不得再把
  它與`0x122dc` selector6混為一談。first-target cancel與destination cancel
  都直接回caller-owned item panel，舊destination→first-target行為已刪除。
  remake現保存panel/effect-table以支援取消後重新確認，並在success/cancel
  reset field與selector。`ComposeFrame`另依已證實scheduler在terrain後、
  foreground前清selector6 selected cell，僅full-frame成功才commit，並不
  猜接其仍未知的production owner。focused Docker/Xvfb
  `go test ./internal/indexedmap ./cmd/fd2 -count=1`通過。
- 2026-07-28 HUD production-base correction：文件稽核先發現UI matrix仍把
  ch01誤寫成只畫full-screen #22 approximation；實際
  `BlitNativeMapHUD→ComposeNativeFrame`早已接入完整已證實subpasses。
  但原版錄影frame oracle又揭露production截圖下緣沒有HUD。Docker-only
  Capstone重讀`0x11cfa..0x11d0a`固定caller傳入
  `[0x53a49]+0x8088`與stride456；舊adapter卻把row157等offset套在work
  allocation base，連unit test都鎖住錯位。`ComposeNativeFrame`現傳
  `work[0x8088:]`，regression驗work與VGA `(anchor+4,161)`位置，HUD重新
  出現在左下角。新增原版錄影oracle
  `docs/figures/native-map-ch01-original-video.png`，並以Docker/Xvfb重建
  `native-map-ch01-remake.png`。兩者證實lower-edge placement，不宣稱
  same-state pixel parity；ch02+ raw view/gates/anchor仍不可複製ch01值。
- 2026-07-28 same-camera/cursor HUD oracle correction：原始錄影是
  1440×1080，game viewport實為置中的`1408×880@(16,100)`；先前直接把
  整張影片縮成320×200會混入黑邊並改變比例，已撤回。新增
  `tools/extract_fd2_video_frame.sh`保存crop/scale命令與「非palette-byte
  oracle」邊界。PSNR候選搜尋定位434.5秒frame；其camera `(1,13)`、
  cursor `(8,15)`／visible `(7,2)`可由terrain與cursor格交叉驗證。
  同時發現`FD2_SHOT_CUR`只改normalized`curX/curY`，production native
  frame仍使用舊`NativeMapViewState`；現改以`MoveNativeMapCursor`逐格驅動
  camera、visible cursor與HUD anchor，並加regression。重建remake後兩圖
  同為tree HUD icon與`A -05/D +10`。roster/event差異仍存在，不把此
  camera/cursor/HUD alignment提升成完整pixel parity。
- 2026-07-28 pre-handler→battle runtime handoff correction：唯讀roster
  audit確認remake圖只見一個party不是`NativeMapFrameRoster`漏畫；direct
  `battle_ch01`從deploy Y `[20,22,21,23]`開始，在cameraY13只有第一筆
  落在下緣。正常原版`ch00_pre`先LOADCH map0，在同一runtime array以
  JOIN順序`0,9,4,30`把scenario UI順序的deploy cells重排為Y
  `[20,21,22,23]`，再由ACT0六個normal beats把slots0..3移至
  `[14,15,16,17]`，接著SPAWN
  initial groups；原版錄影的可見位置符合此序列。舊`resetBattle`的
  「清pre cutscene units避免疊入」註解與行為錯誤：它丟掉已ACT/SPAWN
  records並重播`on_battle_start`。現在handler LOADCH保存exact roster與
  party-scenario paths；只有兩者與下一battle node完全相等時才carry
  runtime array、重建selector slots、保留remaining roster/pending groups
  並consume opening event。direct start、retry、mismatch與非-runtime-append
  scenario仍重建。focused tests覆蓋carry座標、direct部署、opening不重播與
  turn-group事件。後續完整runtime regression更進一步實際compile並執行整份
  `ch00_pre`，逐一排空blocking ACT／PAN／SPAWN／dialog／fade，成功進入
  `battle_ch01`；採用狀態精確為12 slots（party4、group1四筆、group2四筆），
  party座標`0:(7,14),9:(10,15),4:(8,16),30:(11,17)`，slot9的raw
  record `+5` whole-byte overwrite為1，pending groups 3..7完整保留。
  這是end-to-end runtime證據，不再把先前compiler「0 unresolved issues」
  誤當作已跑通handler→battle。
- 2026-07-28 native action-overlay lifecycle closure：原本 production
  `drawNativeActionOverlay` 永遠硬編 `ActionOverlayFrameOffsets(3,false)`，
  所以只畫 final-open skin，雖然 open/close offsets 早已破解。現新增
  caller-owned opening／open／closing state；opening與獨立closing都依序呈現
  frame0..3，期間攔截輸入，且 attack/command/item/spell/wait/cancel mutation
  延至close frame3完成後才提交。`0x1741c/0x176b4` loop沒有delay call，
  因此remake只採「每個Ebiten present一幀」的顯示政策，不把它冒稱原版毫秒值。
  focused lifecycle與全套Docker/Xvfb regression通過；以read-only玩家
  `FDOTHER.DAT`產生的[`action-overlay-open-close-remake.png`](../figures/action-overlay-open-close-remake.png)
  保存上排open0..3、下排close0..3。這是remake runtime artifact，原版DOSBox
  side-by-side及72×72 indexed backup/restore仍未完成。
- 2026-07-28 `0x24618` runtime closure：原先 BeatRunner 即使已有完整
  schedule/LUT/pass/compositor primitives仍固定報「renderer adapter未完成」。
  現在 beat start 先原子驗證 FDFIELD cells、tile-aligned camera、全部
  story actor raw provenance、selector cache、FDOTHER#3 LUT 1..9、
  0x25680 work／320×200 VGA以及FDOTHER#0的768-byte six-bit DAC，成功後才
  publish transition job。九個LUT pass依9→1逐一合成並要求真實Draw ack；
  500ms tail換算30個60Hz ticks；後續0→62 step2的32個`0x11df2` delta皆
  從immutable baseline DAC重算range並upper-clamp63，每步也需Draw ack，indexed pixels不隨palette
  phase改變。缺資源／錯geometry保持原buffer不變。由於60Hz host不可能逐次
  顯示原版5ms/4ms寫入，remake採每狀態至少一個present，不宣稱DOS wall-clock
  parity。focused與全`cmd/fd2` Docker/Xvfb regression通過；ch29 terminal
  handler仍因後段`0x2bce5` ending renderer維持campaign fail-closed。
- 2026-07-28 visual-parity audit：使用者要求先量化「原版畫面 vs remake
  畫面」，故暫停 `0x29164` scheduler。逐項查看 repo DOSBox／錄影 oracle、
  source-rebuild screenshots、indexed fixtures，並以網路原版 church／shop／
  battle UI畫面交叉確認。結論分三層：asset/codec約75–85%、可操作state
  flow約50–55%，但完整操作界面視覺 parity僅約35–40%。town、shop、
  preparation、loadslots、generic ending仍是`drawCampaignUI`現代半透明框；
  ch01 field與church slices最好，但未有同save/roster/tick pixel diff。
  README已撤回把`title.png`／`dialogue.png` raw decode稱作remake runtime
  對照；doc57加入12界面分數／證據、doc91新增UI-VIS-TOWN/SHOP/PREPARATION/
  LOAD/DIFF-HARNESS。此批是文件勘誤，尚未達單獨重大commit門檻。
  其中「loadslots 仍是現代半透明框」已於 2026-07-29 由原版 indexed
  compositor 取代；此歷史子句已失效，不可再當成現況。
- 2026-07-28 native shop stable-scene vertical slice：Docker Capstone 重讀
  `0x2e341/0x1956b/0x4e8af/0x4e8e1/0x2d669`，確認 hub variant 選
  FDOTHER#12/#29/#63，`0x52659` 再選 DATO 128–132/0；`0x1956b`
  先組 FDOTHER#5 dialogue grid＋variant portrait，再以六段水平帶揭幕。
  stable frame 後續貼 entry1、在 `(16,99)` 畫八位 gold、於 `(12,119)`
  畫 FDTXT#440（variant1為#501）。新增 mixed LLLLLL/LMI1 scene parser、
  original-resource compositor regression與
  [`native-shop-scene-indexed.png`](../figures/native-shop-scene-indexed.png)。
  `0x2d669` 服務 entries 使用另一個 `0x4e9e4` primitive，因此保持 raw；
  production shop、圖示 transition/pulse、商品/recipient child panels與
  DOSBox E2 仍未完成，不把 stable fixture 宣稱為完整商店。
- 2026-07-28 native shop service-menu E1 closure：Docker Capstone 固定
  `0x4e9e4` 是 width×height raw literal、palette index0透明；`0x2d669`
  以 entries3/5/7/9 畫四個normal icons，base `0xd430` 加
  `[-39,-13,13,39]/(4-step)`，形成四步spread。`0x2d85f→0x2d9fe`
  對selected option使用相鄰entry，phase/2在normal/selected間切換，
  左右鍵selection 0..3循環。新增 strict raw-cell parser、opening/steady
  compositor與三個FDOTHER variants真實資源 regression；成果圖已更新為
  含四個icons與selected phase2。四個dispatch callee的玩法名稱、
  production owner、child panels與E2仍保持partial。
- 2026-07-28 native shop four-callee semantic closure：平行Docker Capstone
  重讀四個完整body。`0x2f0b0` purchase：商品list→確認→gold/price檢查
  →recipient→`0x1bb8c`插入→optional `0x1c142` equip→`0x2d516`
  debit；`0x2f642` sell：來源item→`floor(3*price/4)`→確認→
  `0x2d3ff`credit→`0x1b8e7`remove→`0x1b750`recalc；
  `0x2f883` equip：角色→item selector→`0x1c1c3`compatibility→
  `0x1c142`same-type replacement→recalc，無gold write；`0x2f8ea`
  為既已閉合的source/destination transfer。`0x2dc55` mode0顯示
  full row+19 price，nonzero顯示trunc(3/4)，新增typed shared renderer。
  `ResolveNativeShopServiceRoute`不再錯稱四項全未知。variant-specific
  FDTXT tables、child panel lifecycle與production integration仍待。
- 2026-07-28 native purchase child-panel E1：續讀`0x2e0bd`證實
  shop resource entry16經`0x4e8af`貼在`(5,112)`，再呼`0x2dc55`。
  新增`ComposeNativeShopItemListFrame`，使用同一shop variant的entry15
  price cell與entry16 panel；mode0顯示full price，nonzero顯示3⁄4。
  真實FDOTHER/FDTXT及tracked item rows regression通過，新增
  [`native-shop-purchase-list-indexed.png`](../figures/native-shop-purchase-list-indexed.png)
  （item0–5 deterministic fixture）。這閉合stable child target，不代表
  product stock table、opening/closing、recipient/equip prompt或production已完成。
- 2026-07-28 native purchase confirmation E1：`0x2f0c5..0x2f0f4`
  的question／insufficient／no-recipient／equip四組六variant FDTXT表已原樣
  保存；question展開商品名／價格後接`0x19953` Yes/No。重要生命週期勘誤：
  insufficient不是fresh overlay。後續指令順序重核修正本條早先錯誤：
  `0x2f2a9`先完成`0x197e5`四幀choice closing，接著
  `0x19913..0x1994c`恢復保存的question region；`0x2f2d3`才在
  `0xac44c=(12,157)`追加第三行；並非保留steady choices，也不是第四個
  inward frame。API目前以deterministic recomposition取得相同question pixels
  並fail-closed，尚未實作generic saved-buffer restore owner。新增
  `native-shop-purchase-confirm-indexed.png`與
  `native-shop-purchase-insufficient-indexed.png`、真實資源regression。
  下一個精確gate是recipient selector／full-inventory feedback與
  optional-equip→success→debit lifecycle，尚未接production owner。
- 2026-07-28 native purchase confirmation production：商品list關閉後接
  原版四幀confirmation opening、bounded左右Yes/No與pulse；No/Escape先
  choice-close四幀，再dialogue-close五幀並重開商品list。Yes且gold不足
  依上述更正後framebuffer追加FDTXT504/438，等待Enter/Escape後同樣關框
  返回。gold足夠時因recipient owner尚未接，只顯示fail-closed訊息並返回，
  不插入物品、不扣款。成果圖`native-shop-purchase-insufficient-indexed.png`
  已依更正後framebuffer重生。
- 2026-07-28 native purchase recipient production：gold足夠後先關dialogue，
  type≥`0x20`進兩欄六人`0x2e6b8`，type<`0x20`依raw class whitelist進
  三列`0x2e8cf→0x2ebe0` AP/DP/HIT/EV比較；兩者均有stateful bounded
  cursor、6-open/5-close、Escape返回。選到八個raw inventory flags均occupied
  的角色會關recipient、開FDTXT506＋`unit[+7]+1`姓名訊息，等待後關閉返回，
  不insert/debit。integration時發現generic item-panel record未填
  `+0x37/+0x39` base AP/DP；新增專用adapter要求`EquipmentBaseSet`後明確
  填入，刪除把零值preview當正確的捷徑。非滿欄選擇目前仍fail-closed返回，
  下一步才接insert→optional equip→success→debit。
- 2026-07-28 native purchase transaction production：非滿欄選擇先在深拷貝
  unit上`ReserveGood`，consumer直接進success；equipment開FDTXT507
  Yes/No，Yes才`EquipItem`，No/Escape保留未裝備新物。Docker Capstone
  重核`0x1c142`確認替換分類是新舊item ID同為`<0x80`或同為`>=0x80`，
  removed raw flag精確寫0、新slot精確寫`0x40`；compact/raw equipped現同步。
  `0x2f4c6` production timeline依variant1/3/5保留5×2、pre1+post8、
  7×2 BIOS ticks（總10/9/14）與portrait restore差異。staged unit在timeline
  啟動前才publish，gold保持不變；final frame呈現後callback才debit並6幀
  重開product list。完整transaction regression鎖定insert-before-animation、
  debit-after-animation。剩餘purchase缺口縮為no-eligible message與E2。
- 2026-07-28 native sell production：Docker Capstone完整重核
  `0x2f642..0x2f87c`。source使用`0x2e6b8`兩欄 roster；empty顯示六variant
  全為FDTXT509＋`unit[+7]+1`；item list用mode1顯示
  `floor(3*row[+0x13]/4)`；question table為
  `{508,508,508,659,508,508}`。Yes順序為dialogue close→variant success
  →credit→`0x1b8e7` remove→recalc，No/Escape直接回source roster而非item list。
  production已接完整loop並保留actor selection。新增`SellNativeSlot`，因舊
  generic `SellSlot`在raw layout留hole，違反`0x1b8e7` shift。display與commit
  price共用effect row；commit先在deep copy preflight、success完成後才publish。
  注意高階Unit為可再次插入而把native ignored stale tail item canonicalize
  成`0xff`，只保證語意與flag/shift，不宣稱FD2.SAV byte parity。
- 2026-07-28 purchase recipient E1：`0x2f30a`證實type≥`0x20`才走
  `0x2e6b8`兩欄六人；type<`0x20`走filtered `0x2e8cf→0x2ebe0`三列
  AP/DP/HIT/EV比較，故新增consumable-only compositor並拒絕裝備type。
  `0x2f36d`滿欄分支以`word_5265f`與`unit[+7]+1`展開動態姓名、mode1
  wait後回商品loop，無insert/debit。新增
  `native-shop-purchase-recipient-indexed.png`與
  `native-shop-purchase-recipient-full-indexed.png`。裝備比較面板與
  success/equip/debit/production lifecycle仍待。
- 2026-07-28 equipment recipient E1：`0x2e8cf/0x2ebe0/0x2ef8f/
  0x2efb7`已完成strict實作。type<`0x20`顯示三列compatible actors；
  `0x2efb7`依raw base AP/DP/DX、candidate item及另一裝備類別計算
  AP/DP/HIT/EV，對current derived words用digit banks31/42/119表示
  equal/increase/decrease。shop entries16/18..22、FDICON/FDTXT、三列
  geometry與6-open/5-close已有原版資源regression，新增
  `native-shop-purchase-equipment-recipient-indexed.png`。尚待成功
  insert→optional equip→animation→debit與production/E2。
- 2026-07-28 shop purchase success E1：共享`0x2f4c6`已按hub variant
  拆開，不能誤用church case4。variant1/#12為5幀`(169,45)`＋portrait
  restore；variant3/#29為單幀`(148,39)`、pre1/post8 ticks＋restore；
  variant5/#63為7幀`(131,28)`且無restore。三variant原版資源regression
  與`native-shop-purchase-success-indexed.png` contact sheet已補。
  caller順序為insert→optional equip/recalc→success→debit→product loop；
  production timeline owner與DOSBox E2仍待。此為當輪狀態；production owner
  與bare/debit修正已由本文件後續2026-07-29段落取代。
- 2026-07-28 native shop production slice：campaign schema新增
  `native_hub_variant`，69個原版weapon/item/secret shop節點分別明確
  指定1/3/5；0保留自訂戰役generic fallback，其他值load時fail-closed。
  `cmd/fd2`現以原版FDOTHER/DATO/FDTXT indexed assets擁有stable scene、
  四格service menu與purchase list，並實作service 4-frame spread/
  contract、purchase 6-frame open/5-frame close＋stable restore及
  selected pulse。production regression會確認原版節點被native owner接管、
  custom節點不被誤接。尚未接confirmation→recipient→success→debit，
  其他三個service也仍fail-closed；不得把本切片描述成完整可交易商店。
  「尚未接」同樣是歷史狀態，後續段落已接confirmation/recipient/success/debit；
  完整E2限制仍有效。

## 2026-07-28 standalone shop equip production closure

- Docker Capstone重新閉合`0x2f883→0x1bffe`：service2不是purchase item
  list或裝備收件者面板。`0x2f883`只以`0x2e6b8`選角色並關閉roster；
  `0x1bffe`透過`0x17e0b`保存當前shop VGA、以11→0開完整
  `0x17eef+0x184c0` item/status panel，循環`0x1b9de(actor,0)`。
- Enter後`0x1c1c3`不相容直接回selector且沒有feedback owner；相容才走
  `0x1c142` same-ID-category equip、`0x1b750`重算並原地重畫panel。Escape
  或空inventory確認才以0→11關閉並restore shop，再回角色roster。
- production已接service2→equip roster→native item panel→strict mutation→
  in-place redraw→restore lifecycle。`EquipNativeCompactSlot`先證明raw
  occupied cells與compact Inventory/Equipped同序，保留ignored raw hole與
  stale item byte；不一致時原子fail-closed。
- `TestNativeShopProductionOwner*`已用玩家原版FDOTHER/FDTXT/DATO與Docker/
  Xvfb通過；`campaign`另有raw-hole與divergence atomic regression。這是E1/
  production closure，不是DOSBox同save/tick E2。
- 下一個shop高價值缺口是`0x2f8ea` inventory transfer production owner；
  standalone equip不要再重用purchase price/recipient/success fields。

## 2026-07-28 shop transfer production closure

- direct caller交叉重核：`0x2f8ea`同時由shop `0x2e341` service3與church
  `0x3072f` raw1呼叫；不是任一設施專屬。subagent一度只看到shop caller而
  建議刪church接線，已由`0x308a5..0x308b2`直接call證據否決，未採用。
- exact loop為FDTXT512→source全party roster；空來源FDTXT511＋姓名，否則
  `0x2e0bd/0x2df6b(mode1)` item list；選item後FDTXT510→destination全party
  roster；滿八格走variant FDTXT506＋姓名，成功走
  `0x1b722→0x1b8e7→0x1bb8c→0x1b750`，無gold/confirm。
- destination roster不排除source本人。未滿八格的self-transfer會literal
  remove→append，item移到尾端且清equipped；滿八格則remove前先走506。
  remake已撤回排除source的高階假設，shop與church兩個caller都保存此行為。
- shop production已接intro/empty/destination/full messages、source/item/
  destination panels、所有cancel/success回512 loop及source-cancel回service
  menu。`ValidateNativeInventoryProjection`要求raw signed cells與compact
  Inventory/Equipped同序，目的full只看raw flags。
- Docker/Xvfb玩家原版資源production regression與battle self/full/divergence
  regression通過；仍缺DOSBox同save/tick E2。

## 2026-07-28 native secret-shop entry gate

- Docker Capstone重讀`0x2cad7..0x2d301`。`0x4e4b9(chapter)`回傳
  `0x6238d+(chapter-1)*0x1f`；town loop在`0x2cde0..0x2cef7`同時比較
  record `+1`與目前selection、record `+2`與BIOS scan，完全命中才把
  `[0x5412b]`寫5。`0x2d28c`的非0/2/3/4分支再呼`0x2e341`，即variant5。
- scan `0x54..0x5d`、`0x5e..0x67`、`0x68..0x71`分別是
  Shift/Ctrl/Alt-F1..F10。23筆有town的raw records對應玩家章2..22、26、27，
  已資料化為`native_secret_gate {selection,scan_code,to}`。
- runtime只有selection與chord完全命中才把hub selector揭露為5；原版會先
  重畫第六組icon/label，後續Enter才進secret shop，不要求也不寫persistent
  unlock flag。modified F2/F3/F5/F9不再同時觸發remake全域快捷鍵。
- 撤回「`found_secret_ch*`先解鎖並永久顯示第六項」與legacy
  `SecretIf/ShopGoods()`等同原版gate的斷言；兩者只保留editable擴充相容。
- `gen_campaign.py`目前會把已人工整合的`campaign_full.json`退回舊
  `story_ch01`拓撲；本輪測出後已恢復權威檔，只保留23筆小型gate差異。
  後續不可無審核直接以生成器輸出覆蓋campaign。

## 2026-07-28 native town indexed production owner

- Docker Capstone閉合`0x2cd16/0x2cd46/0x2cf71/0x11eb0`：town record byte0
  選FDOTHER#11/#61/#62；#10的62×26 label畫在scene `(244,162)`，
  FDTXT `0x1ef+selection`從`(252,168)`開始，FDICON pulse依
  `0,1,2,1`及`0x52635/0x52647`三組六項座標畫入；最後只把312×192
  viewport貼到VGA `(4,4)`。
- 23個town節點新增editable raw `native_town_variant` 0/1/2；
  `gen_campaign.py`同步保存variant table，但生成器整體拓撲仍落後權威
  `campaign_full.json`，不可直接覆蓋。
- production `drawCampaignUI`在資源與schema完整時改走原版indexed owner；
  右鍵selection遞減、左鍵遞增、0..4 wrap，selection5可重畫。pulse依signed
  BIOS low-word delta≥4遞增。缺資源／非法資料仍fail closed回generic custom UI。
- `AdvanceNativeTownSecret`的「chord立即dispatch」API與斷言已刪除，改成
  `MatchNativeTownSecret`揭露、`ConfirmNativeTownSecret`確認兩階段，並以
  production-owner、真實資源、hidden redraw與signed wrap regression覆蓋。
- [`town-hub-remake.png`](../figures/town-hub-remake.png)已由目前source在
  `fd2-go-test-local`＋Xvfb重拍，為ch02 variant0/selection0 runtime畫面；
  當時尚無 DOSBox E2；下節已補同章／同selection pair/diff，剩餘 gate 是
  同 pulse tick、其他 selection 與 variant。

## 2026-07-28 ch02 town hub DOSBox E2 slice

- `FD2.SAV` 的 current-runtime ch00 battle 以 `/tmp/fd2-town-e2f` 可寫副本執行；
  原始遊戲目錄未掛入容器、未修改。sandbox 的 FD2.EXE 只 patch 共享玩家控制器
  `0x117f3→call 0x205be`、`0x117f8→jmp 0x1187a`，以及 victory helper
  `0x205d5→jmp 0x206c3`，用途是略過人工打完整場；戰後 handler、campaign
  gate、town renderer/resource 都仍走原版。不可把此 route patch 描述成原版
  正常輸入或勝利語意。
- title timeline 必須把四次 Escape 分散在 opening phases：
  `wait2, Esc, wait4, Esc, wait4, Esc, wait4, Esc, wait6`；舊的連續 Escape
  寫法會因按鍵落在錯誤 phase 而不穩定。CONTINUE 後共有20次戰後對話確認，
  第21次確認後得到 ch02 variant0／selection0 原版 town hub。
- 新增 DOSBox runner `repeat:count,key,delay_ms`，可重現長對話而不手寫大量
  timeline steps；參數不合法即 exit 2。
- 保存 [`town-hub-original-dosbox.png`](../figures/town-hub-original-dosbox.png)、
  [`town-hub-original-vs-remake.png`](../figures/town-hub-original-vs-remake.png)
  與 [`town-hub-pixel-diff.png`](../figures/town-hub-pixel-diff.png)。初次 capture
  只證明遮罩外一致；下一節已修正 glyph shadow 並取得 selection0/1 整幀相同，
  因此本段的 masked hash 不再是目前最高證據。

## 2026-07-28 town glyph shadow correction and two-state exact E2

- DOSBox selection1 pulse 階梯顯示角色 sprite 可與 remake 完全相同，但中文
  「武器店」仍殘缺，推翻「剩餘差異只是未同步 pulse」的斷言。Docker Capstone
  重讀 `0x4ea2a`：set bit 先寫 foreground 到 `edi`，shadow 寫到
  `edi+(stride-1)` 與 `edi+stride`；現有 `BlitNativeGlyph` 第一個 shadow
  誤寫成同列 `edi-1`，會覆蓋相鄰 foreground。
- `BlitNativeGlyph` 已修成下一列左下／正下，新增 adjacent source bits
  regression，防止 shadow 再覆蓋連續筆畫。這是所有使用此共用 primitive 的
  town/shop/church indexed text 修正，不只是一張 town 圖的特例。
- 新增 strict screenshot-only `FD2_SHOT_TOWN_STATE=selection,pulse`；
  selection 必須0..5、pulse必須0..3，且目前必須是有 native variant 的 town
  node，否則 Update 回錯且不產生錯狀態 artifact。它只固定 oracle state，
  不改正常玩家輸入。
- `0x2ce7a/0x2ceac` 方向鍵只改 `[0x5412b]`，`0x2cef7` secret chord 也只寫5；
  三者都不寫 pulse counter `[0x54133]`。remake 已刪除移動／reveal 時強制
  reset pulse0 的錯誤行為；只有 screenshot hook 為下一次 Draw 固定指定 phase。
- ch02 variant0 selection0/pulse2 原版與 remake 320×200 raw RGB MD5 均為
  `8a6a4b03946d1958d3af95fd4bd775c3`，更新後
  [`pixel diff`](../figures/town-hub-pixel-diff.png) 全黑。Left 後
  selection1/pulse2 的 [`pair`](../figures/town-hub-selection1-original-vs-remake.png)
  亦整幀相同，MD5 均為 `60a4791d60b32fd6efc82864afd63525`。
  尚未閉合 variant1/2、selection2–5、Right/wrap、secret reveal/confirm。

## 2026-07-28 town variant0 six-selection/input E2 closure

- 固定 `repeat:20` 不可靠：原版打字／翻頁狀態會讓同一 Return 有時只完成文字，
  有時才換頁，造成 capture 少一頁或多按進酒店。新增 runner action
  `waittown0:key,delay_seconds,max_tries`；每次送 key 前先以 variant0 三個固定
  背景像素 `(10,10)=(56,77,16)`、`(160,130)=(170,142,101)`、
  `(300,190)=(117,138,138)` 判斷 full-colour town ready，命中即停止。
  參數錯誤 exit2、上限未命中 exit3。名稱刻意是 `waittown0`，不能拿它宣稱
  variant1/2 同步完成。
- 原版 route patch sandbox 在多輪執行後可能更新 current-runtime/save 狀態；
  本輪另建 `/tmp/fd2-town-e2g`，從原始 FLAME2 全新複製後只重套三個已驗證
  上游 patch。Docker Capstone再次確認`0x117f3 call 0x205be`、
  `0x117f8 jmp 0x1187a`、`0x205d5 jmp 0x206c3`。不得重用已改變狀態的
  sandbox 又假設 CONTINUE 仍從同一戰場開始。
- ch02 variant0 六項 E2 raw RGB MD5：
  selection0=`8a6a4b03946d1958d3af95fd4bd775c3`；
  selection1=`60a4791d60b32fd6efc82864afd63525`；
  selection2=`10017309d3c833c8e323e8739d624f8b`；
  selection3=`0e1db5b95951230b3c13d1f0309296d2`；
  selection4=`1577fc5749410221497f512b52a12dbe`；
  selection5=`e695d6cf391c45ccf4d2cf70096eb9bf`。每個原版幀都能與
  `FD2_SHOT_TOWN_STATE=selection,pulse` 的某個 production runtime 幀整張
  320×200 相同；不是只比較 crop 或遮罩。
- 原版 input trace另外實測 Right `0→4`、Left `4→0`，以及 ch02
  Shift+F1 (`scan 0x54`) `0→5` 顯示「???」。新增
  `revealNativeTownSecret` production helper與 regression，預置非零 pulse、
  lastTick、hasTick後 reveal只改selection5，三個 clock state完全不變。
- 新 artifact
  [`town-hub-six-selections-original-vs-remake.png`](../figures/town-hub-six-selections-original-vs-remake.png)
  上排原版、下排remake，selection0..5由左至右。variant0 steady redraw、
  normal wrap與hidden reveal E2已閉合；仍待variant1/2及selection5後續
  Enter→secret shop transition。

## 2026-07-28 ch02 secret-shop service0 exact E2

- 由同一 `/tmp/fd2-town-e2g` sandbox 以 `waittown0` 抵達 ch02 town，
  Shift+F1 只揭露 selection5，再送一次 Return；原版約0.15秒仍是town、
  0.5秒為淡入、1秒後為variant5/resource63/DATO portrait `0x84` 商店。
  上游 route patch 邊界未變，town/shop handler、resource、input均未修改。
- 新增 strict screenshot-only
  `FD2_SHOT_SHOP_STATE=service,pulse,gold`。只接受已由production
  `setupNativeShop` claim 的stable native shop menu、service/pulse 0..3與
  gold 0..99999999；非法node/mode/variant/resource全部fail closed。gold是
  同save畫面輸入，不應用來推導或改寫正常campaign的初始金額。
- 第一輪同gold比較顯示phase0整幀相同，但四個requested pulse都輸出phase0。
  原因是 `ComposeNativeShopServiceSteadyFrame` 已執行 `phase/2`，runtime又先
  傳 `nativeShopUIPulse/2`。已只把service-menu caller改傳完整phase，並新增
  production consumer regression；Yes/No等接受0/1 variant的compositor不改。
- 高頻原版取樣證明service0在兩個selected states交替。gold0時phase0
  original/remake raw RGB MD5均為
  `12fad3c03096aae48098c8f9074370c7`，phase2均為
  `e5654e8ed03d1e4fd30b2c76106bb7a1`，兩組320×200 AE均為0。
  早期4秒樣本剩53像素差全在portrait mouth `(151..180,50..53)`，是獨立嘴型
  animation，不能歸因於service pulse；後續同嘴型樣本已取得全幀相同。
- 新圖
  [`secret-shop-ch02-original-vs-remake.png`](../figures/secret-shop-ch02-original-vs-remake.png)
  左為原版DOSBox、右為目前source-built remake selected phase。下一個 precise
  gate 原為service1–3與Escape return；下節已閉合。variant1/3也已由後續
  ch02 trace閉合，不再是待辦。

## 2026-07-28 ch02 secret-shop four-service and return E2

- 原版 stable menu 實送 Right `0→1→2→3→0`、Left `0→3`；六張畫面各自
  與 `FD2_SHOT_SHOP_STATE=service,pulse,0` 某一phase的320×200全幀
  AE=0。這同時驗證四項service icon anchors、wrap與input後pulse reset結果。
- Escape後0.1/0.35/0.85秒為closing/fade，3.85秒已回ch02 town且仍顯示
  hidden selection5；該幀與deterministic town selection5/pulse1全幀AE=0。
- 這推翻production `leaveShop`已等價的斷言：`enterNode()`返回town時會把
  `campSel`重設0。現在先保存native shop variant，只有next node確為town且
  variant為1/3/5時恢復同值；custom/non-native shop仍為0。variant5有本輪
  DOSBox直接證據，variant1/3沿用已閉合的town option→shop variant dispatch。
  `TestNativeShopReturnRestoresDispatchingTownSelection`覆蓋1/3/5及custom0。
- 新圖
  [`secret-shop-ch02-services-return-original-vs-remake.png`](../figures/secret-shop-ch02-services-return-original-vs-remake.png)
  上排原版、下排remake，從左到右service0/1/2/3與returned town selection5，
  每一格全幀AE=0。下一gate原改為ordinary shop variant1/3；下節已閉合，
  現改為四個service child panel的DOSBox同狀態E2。

## 2026-07-28 ch02 shop variants1/3/5 menu E2 closure

- 同一原版town route由selection1進weapon variant1/resource12、selection3
  進item variant3/resource29；各取10個stable樣本。variant1十張、variant3
  排除首張transition後九張都在phase0/2間交替，且每張與
  `FD2_SHOT_SHOP_STATE=0,pulse,0` production frame全幀AE=0。
- selected phase raw RGB MD5：variant1
  `69003be54f47c221916c1ed89cf1d26f`；variant3
  `dd5d80bb761cc87980dff066773f6763`；variant5
  `e5654e8ed03d1e4fd30b2c76106bb7a1`。每一值的原版與remake成對相同。
- 新圖
  [`shop-variants-1-3-5-original-vs-remake.png`](../figures/shop-variants-1-3-5-original-vs-remake.png)
  上排原版、下排remake，左至右weapon/item/secret。ch02三種shop
  stable service-menu的background、portrait、text、gold、icon layout與selected
  pulse gate已閉合；不能把此證據推廣成purchase/sell/equip/transfer child
  panel E2。下一個 precise gate 是service0 Enter後的purchase list。

## 2026-07-28 ch02 weapon purchase-list E2 closure

- 新增 screenshot-only
  `FD2_SHOT_SHOP_PURCHASE_STATE=selection,start,gold`。它只接受目前已由
  production claim 的 native shop menu，且goods selection合法、window start
  為正規化偶數值；variant/resource/mode/window任一不一致即fail closed。正常
  campaign/input不讀此hook。
- 原版ch02 weapon variant1由service0 Enter進購買清單，四筆可見商品為
  布衣50／旅行裝500／皮甲300／法師袍750，與editable campaign goods一致。
  實送Right `0→1`、Down `1→3`、Left `3→2`，證實兩欄input mapping。
- 最初短等待的original/remake比較只差screen `(175,90)/(176,90)` 兩像素；
  延長取樣後先出現portrait animation差異，再進入持續全幀相同的stable state。
  因此不能把opening/lifecycle transient當成steady renderer誤差。四個stable
  raw RGB MD5為selection0
  `1589cee3c068936f0beb6058cfd63991`、selection1
  `7480dbb0284b033b4e9ad8c8c7a8b78e`、selection2
  `48d6182e261ebce574b08c4778b8a072`、selection3
  `3c0a2c935260b8ca80432b25b3600111`；每一對320×200 AE=0。
- 新圖
  [`shop-purchase-ch02-selections-original-vs-remake.png`](../figures/shop-purchase-ch02-selections-original-vs-remake.png)
  上排原版、下排remake，順序0/1/3/2。這只關閉purchase list steady/input；
  purchase confirm、recipient、success/failure與其他三個service仍須E2。

## 2026-07-28 repo-wide Markdown assertion audit

- 掃描66個Markdown後，已修正會直接誤導production的現行敘述：攻略目標不能
  取代逐章battle/postbattle handler；`battle_events.json`與
  `gen_campaign.py`只可作candidate/editable scaffold，不能直接當原版route
  oracle；Beat DSL仍有unknown post handlers，不能稱33關機械完整抽取。
- 撤回跨文件殘留的「character id＝DATO portrait＝FDICON sprite group」
  全域恆等。現行ABI以doc31為準：`unit+2`是`0x11019` cache slot，
  `unit+7`與persistent identity有各自writer；remake單一角色表只是extension
  authoring schema，不是原版數值alias。
- README、42、56、57、91已同步目前E2範圍：ch02 town variant0、三種shop
  menu、secret return及weapon purchase list已有窄切片exact證據；不可外推到
  23 towns／69 shops。歷史handoff/reflection保留，但current authority與
  evidence tier優先。
- 名稱稽核未發現衝突：角色正名仍為「悠妮」；`DATO_075`為商店店員NPC，
  不得進party/JOIN。

## 2026-07-28 ch02 weapon purchase-confirm E2 closure

- 原版由已閉合的weapon purchase list selection0再送Return，經list close與
  question/choice opening進入「布衣／50元／要不要啊？」；預設Yes，Right到No，
  Left回Yes。這是真實input route，不是indexed fixture。
- 新增 strict screenshot-only
  `FD2_SHOT_SHOP_CONFIRM_STATE=good,choice,pulse,gold`。它只在production已
  claim native shop、shared class/shop assets與variant一致且good確實存在時接受
  choice0/1、pulse0..3、gold0..99999999；清除screenshot lifecycle job但不執行
  交易，正常campaign/input不讀此hook。
- 原版高頻sample證實normal與selected pulse交替。selected Yes raw RGB MD5
  `7a07b1c064ca2c431bc97c798dcfd51e`，selected No
  `56f6ffb003e87cbc63d7a915ac4b5dd0`，normal
  `b8cce25df13447e73e1750a8b2edaf0f`；每一對production 320×200全幀AE=0。
- 新圖
  [`shop-purchase-confirm-ch02-original-vs-remake.png`](../figures/shop-purchase-confirm-ch02-original-vs-remake.png)
  上排原版、下排remake，左Yes右No的可見selected pulse。當時下一個gate含
  insufficient；該分支已由下節閉合。recipient／no-recipient／full／success
  等其餘transaction結果仍不得由confirmation E2外推。

## 2026-07-28 ch02 weapon insufficient-gold E2 closure

- 新增 strict screenshot-only
  `FD2_SHOT_SHOP_INSUFFICIENT_STATE=good,gold`。它只接受production已claim、
  shared assets完整、good存在且`gold<editable price`的native shop；不扣金、
  不改recipient，final compositor admission失敗會原子回復，正常input不讀hook。
- 原版gold0由布衣good0確認Yes後顯示「錢不夠！」及mode-one等待標記。最初
  production誤把`0x197e5`第四個inward choice frame當return framebuffer，
  導致整幀AE=563與紅色重疊。Docker Capstone重核`0x197e5`證實四次present後
  `0x19913..0x1994c`恢復保存的310×86 question region；`0x16c57(1)`先畫
  FDOTHER#5 cell18，依BIOS delta在cell18/19間循環，最後以cell13清理。
- remake以ch02限定的deterministic recomposition取得restore後question的相同
  pixels，並在variant1 caller-owned `(143,181)` anchor畫cell18後，原版與
  production 320×200 raw RGB MD5皆為
  `6babcedfe2017a7457924c4df65ba7dc`，整幀AE=0。新增
  [`shop-purchase-insufficient-ch02-original-vs-remake.png`](../figures/shop-purchase-insufficient-ch02-original-vs-remake.png)。
- 原版gold1000已取得裝備收件者候選「索爾、悠妮、亞雷斯」與三列能力比較
  畫面，但目前screenshot啟動路徑沒有同一份production persistent party raw
  projection。不得硬編姓名或以fixture冒充runtime；recipient/no-recipient/
  full/success仍是下一批E2 gate。

## 2026-07-28 ch02 equipment-recipient stable E2 closure

- 真正阻塞不是renderer入口，而是`FD2_CAMP_NODE=shop_ch02_weapon`只進node、
  不建立`partyRoster/partyJoinOrder`。新增screenshot-only
  `FD2_SHOT_PARTY_BINDING`：只接受compile無issue且唯一LOADCH同時提供
  `PartyScenario+PartyOrder`，依該binding記錄的order materialize scenario；
  hook本身不重新證明JOIN來源位址。這只是screenshot bootstrap，不代表正常
  campaign persistence或native FD2.SAV載入。
  初始化equipment base，逐員要求identity/selector/race/class/byte6/raw八格
  inventory與base provenance。姓名、職業、順序均不在code硬編。
- 新增
  `FD2_SHOT_SHOP_EQUIPMENT_RECIPIENT_STATE=good,selection,start,cycle,gold`；
  只接受可負擔的真實equipment good、正規三列window、cycle0..2與完整final
  compositor，失敗原子回復。這是bounded screenshot adapter，正常input不讀。
- 第一張production已讓「索爾、悠妮、亞雷斯」順序正確，但E2抓出候選HIT/EV
  少DX projection。`PartyMember`現顯式保存`dx`，ch01四人值2/2/1/2由
  visible derived HIT/EV與已知equipment rows交叉約束；不是獨立raw
  `+0x3e` dump，也不再用零值補projection。
- 像素diff另固定原版geometry：AP/DP current/arrow/result offsets為
  `+15/+35/+43`，HIT/EV為`+18/+38/+46`，所有arrow Y=`row+dy+1`。
  高頻DOSBox樣本證實FDICON idle cycle1時，good0/selection0/start0/gold1000
  原版與production raw RGB MD5皆
  `28258fb3ce5bc42eb1c701a7792d193b`，整幀AE=0。新增
  [`shop-equipment-recipient-ch02-original-vs-remake.png`](../figures/shop-equipment-recipient-ch02-original-vs-remake.png)。
- 這只關閉stable selection0/cycle1，不證明recipient方向鍵／scroll、
  native FD2.SAV相容、no-recipient/full/success或交易debit。

## 2026-07-28 normal campaign JOIN→LOADCH persistent-party bootstrap

- recipient調查抓出正常流程斷鏈：JOIN只建立`partyMembers/partyJoinOrder`，
  首次LOADCH未建立`partyRoster`；而帶native identity的`syncPartyFromBattle`
  只匹配既有typed record，故開局成員可能全部被skip。
- `applyLoadCH`現於已有JOIN chronology時，從同一typed party scenario只補
  缺少的persistent records；既有battle/shop/class-change進度不覆蓋。
  direct/debug LOADCH沒有JOIN history時維持無persistent state，未把
  screenshot bootstrap偷接進production。
- regression以ch00真實scenario/order `[0,9,4,30]`驗證raw inventory、
  selector、identity與equipment base均保留，ch02布衣equipment recipients
  為`[0,9,4]`，首次native-identity sync亦能命中既有record。
- 這是remake typed runtime lifecycle修正，不證明native FD2.SAV相容或完整
  title→ch00→battle→postbattle→town→shop input/E2 playthrough。

## 2026-07-28 equipment-recipient selection1／暫停點

- 原版由已閉合的裝備收件者selection0按Down可到selection1，按Up回selection0；
  Left/Right不改selection。production目前也是bounded Up/Down、Left/Right no-op，
  三列window由`campaign.NativeThreeRowWindow`提供，但還沒有直接覆蓋
  `handleNativeShopInput`的input-level regression，不能只以renderer測試代替。
- 目前selection1高頻取樣與同FDICON cycle的remake逐像素比較，只剩商店人物區
  `(175,90)`、`(176,90)`兩點不同：原版兩點皆`(138,158,158)`，remake分別為
  `(101,121,121)`／`(117,138,138)`，總計6個RGB bytes。recipient panel、
  selected-yellow row、stats、sprites與window其餘像素一致。selection0的不同
  原版採樣也曾在同兩點於兩種值間切換，因此這是尚未同步的原版環境／portrait
  相位證據，不得遮罩，也不得把selection1寫成全幀AE=0。
- `tools/docker/fd2-dosbox-screenshot.sh`新增
  `waitpixel:x,y,r,g,b,delay_seconds,max_tries`，可在送下一個input前以原版畫面
  像素簽章同步。已通過`bash -n`、Docker image重建、非法參數exit2、
  未命中exit4與命中exit0；沒有改裝host Python或Capstone環境。
- 下一輪先建立乾淨的disposable native save/game sandbox，再以`waitpixel`
  同步上述環境相位後送Down，取得selection1與後續scroll E2。若相位在乾淨流程
  仍不可達，應保留partial並追caller-owned portrait/palette presentation，
  不能以固定sleep、忽略兩像素或猜測renderer語意閉合。
- 文件稽核提醒：裝備收件者selection0 E2依賴screenshot-only
  `FD2_SHOT_PARTY_BINDING`，DX `2/2/1/2`是可見HIT/EV與已知裝備列交叉約束，
  LOADCH PartyOrder則是已記錄provenance加compiler validation；三者都不是
  native FD2.SAV完整campaign trace或獨立raw欄位dump。

## 2026-07-28 authority-document assertion audit

- `29-remake-extensible-event-system.md`是早期擴充DSL構想，不是native event
  ABI。已撤回「hard-coded handler只管勝負、動作都在FDFIELD」；目前逐章
  evidence已直接看到SPAWN/JOIN/PAN/ACT/dialogue/LOADCH，忠實模式必須逐支
  轉錄，不能由攻略或schema補完。
- 同文件撤回`record +5 bit0=alive/dead`全域命名，改為caller-specific raw
  predicate；第一章主角隊修正為索爾／亞雷斯／悠妮／蓋亞，示例中的妮雅改為
  悠妮。`spawn_march`仍只留schema示意，不代表原版進場owner已證實。
- worklist早期round標題曾把單一codec/fixture成果寫成「核心全完成／通用
  1:1／像素級收官／魔法SFX補完」，已改成歷史scope標記；項目內較新的
  partial/fail-closed敘述才是現況。
- SDD整體UI視覺估計已統一到doc57的40–45%；shop recipient的filtered list、
  open/close jobs與full/cancel branch只算E1 production implementation，
  input/scroll、no-recipient/full/success及lifecycle timing仍需DOSBox E2。
- ch29 `0x25089` cleanup已有binding/compiler/runtime regression；刪除同一
  worklist後段仍稱「需補binding、測試、接town/shop/preparation」的過時重複項。

## 2026-07-28 repo-wide historical Markdown assertion audit

- repo入口`00`與關卡整理`28/53`已把攻略統一降為E3 authored/player-visible
  reference；攻略可建立candidate與E2案例，但不能命名raw bit、result code、
  event id或取代逐章handler/postbattle/town/preparation/persistence evidence。
- `19/29/30`中的「自動生成忠實原版33關」「ConditionRegistry/ActionRegistry
  一行擴充」「所有差異已在資料」均改為設計目標或authored scaffold。production
  目前沒有這兩個Registry，仍使用typed battle events、campaign beats與部分
  chapter-specific adapter。
- battle/renderer舊文件修正三個會直接造成錯畫的衝突：`35`摘要不再把TAI
  台座稱為BG層，`[0x53ec8]`不再命名成縮放figure X，我方背影／台座與敵方
  正面只限已驗fixture；`39`明確指出離線AFM PNG不能省略caller schedule、
  scroll、palette與input/skip。
- `44`撤回「序章完全沒有單位逐幀移動」；map0 direct deployment與此前
  `0x13185`/ACT走位是不同階段。group10/11只可說在已審ch00/turn-event路徑
  未見啟用，不能宣稱全遊戲死資料。舊「兩行文字直接開戰」標成歷史snapshot，
  現行palace/forest/march仍以partial/E2 gate描述。
- `47/50`把dir0限定為ch00與已觀測initial state；其他handler/ACT writer可
  覆寫pose。`TestCampaignFullPostBattleTownContractMatchesOriginalShopChapters`
  只驗authored graph contract，不證明全戰役原版route或DOSBox E2。
- `99`的「資產格式全解」已收窄成當時列舉base codecs/exports；mixed-resource
  entries、caller binding與scene composition仍以doc56/57為現況權威。

## 2026-07-28 equipment-recipient production input contract

- `handleNativeShopInput`的equipment recipient分支不再inline修改selection；
  production與tests共用純`advanceNativeShopEquipmentRecipient`。它保存
  bounded Up/Down、horizontal no-op、同tick Up後Down的原有更新順序，並只由
  `campaign.NativeThreeRowWindow`擁有stateful三列viewport。
- direct regression覆蓋原版已觀測trace
  `selection0→Down→selection1→Up→selection0`、頂／底界線、六候選window
  `start 0→1→2→2→1`與helper-level invalid count/selection/start拒絕。
  production caller會在索引recipient前把拒絕路由回purchase list。
- focused與完整`go test ./...`均在`fd2-go-test-local` Docker image及手動管理
  Xvfb lifecycle下通過。先前直接使用`xvfb-run`時go test已退出但wrapper未
  收掉Xvfb，故該次被明確中止，不算驗證結果。
- 這只關閉E1 production input contract。selection1仍有商店人物區兩像素
  ambient phase差，scroll／opening／closing timing也尚無DOSBox E2；不得升級
  UI-09為完整recipient lifecycle parity。此句是當輪狀態，selection1兩像素
  差異已由下一節的相位同步E2取代；scroll與完整lifecycle限制仍有效。

## 2026-07-29 equipment-recipient selection1 phase-synchronized E2

- 從`/tmp/fd2-town-e2d`取得未跑過的原始SAV/TMP基線，只覆蓋
  `/tmp/fd2-town-e2g`的verified三處route-patched EXE，建立新的disposable
  sandbox；原始FLAME2未掛載、未修改。SAV只在副本以既有`tools/fd2save.py`
  decode/encode及native checksum設定current runtime raw gold為1000。
- Docker DOSBox先以`waittown0`到ch02 town，再進weapon→purchase→good0→Yes。
  在selection0送Down前以`waitpixel:175,90,101,121,121,0.05,400`同步商店人物
  動畫／palette相位；五張selection1高頻原版crop與remake三個cycle逐一做
  ImageMagick raw RGB `compare -metric AE`。
- 結果不是兩像素renderer缺陷：0.05／0.20秒樣本對cycle1為AE=0，0.10／0.80秒
  樣本對cycle2為AE=0，0.40秒樣本（timeline名`040`）對cycle0為AE=0。
  比較全程未遮罩像素。新增
  `docs/figures/shop-equipment-recipient-selection1-original-vs-remake.png`
  保存其中一組原版／remake 320×200左右對照。
- selection0↔1 input與三cycle畫面可升為E2；四人以上三列window scroll、
  opening/closing timing、交易提交與正常campaign/native save lifecycle仍未閉合，
  不得把本結果外推成完整recipient或shop parity。

## 2026-07-29 purchase success bare framebuffer and gold odometer

- 從同一乾淨route實際選recipient0後，原版先顯示「要裝備上去嗎？」；Yes關閉
  confirmation/dialogue後才呼`0x2f4c6`。高頻抓圖證明success effect底下沒有
  藍色問句框。舊production與`native-shop-purchase-success-indexed.png`把
  greeting/question framebuffer帶入success，是錯誤斷言與renderer狀態。
- production新增`ComposeNativeShopBareScene`，success timeline只在bare
  background＋portrait＋decoration＋gold上套variant effect。修正後原版
  0.03/0.08/0.16/0.26秒四張依序對source-built frame0/1/2/3；每張AE=2，
  差異只在portrait `(175,90)/(176,90)`：原版`138,158,158`、deterministic
  compositor `101,121,121`／`117,138,138`。未遮罩，故success E2仍partial。
- Docker Capstone完整重讀`0x2d516..0x2d620`：先format舊八位數，在
  `0x2d551`立即sub balance，再format新八位數；每個不同digit同步decrement，
  `0→9`，設counter9。每phase以`9*digit+counter-1`索引current FDOTHER
  resource entry2的6x99 strip，`0x2d620`逐9列opaque copy6到
  `0xa7a90+6*position`、stride320，最後`0x375b2(10)`。重複直到新值。
- `ComposeNativeGoldDebitFrames`與production兩段timeline已接上述contract：
  success完成後才commit新gold並開始roll，roll完成才六幀重開product list。
  `1000→950`=45 phases，`1234→1134`=9 phases；wrap、borrow cascade、bounds、
  10ms per phase與mutation boundary均有regression。60Hz timeline依wall-clock
  elapsed取樣，卡頓時可能略過中間10ms phase；這保留總時長而非「每phase必present」，
  文件不得宣稱逐phase display E2。
- 更新原本錯誤的success fixture，並新增
  `shop-purchase-success-ch02-original-vs-remake.png`四幀上下對照。下一gate是
  caller-owned portrait saved-buffer phase與扣款phase E2，不是再猜instant debit。
  此句是當輪狀態；扣款由下一節閉合，success人物兩點則由本檔稍後的
  「購買成功動畫兩像素」章節閉合，不再是現行限制。

## 2026-07-29 DOSBox internal-capture debit E2 and one-row correction

- 外部ImageMagick `import`會擾動10ms roll；DOSBox 0.74內建Ctrl+Alt+F5在此
  Debian build未產AVI，但Ctrl+F5可直接保存320×200 paletted framebuffer。
  `xdotool key --repeat 80 --delay 10 ctrl+F5`在optional-equip Yes後得到80張，
  無X11 crop／scale；0..25涵蓋success、26..46涵蓋debit、47後為product list。
- 首次將debit oracle與80張逐一比對時，所有gold差異集中在y98..107。重核
  `0x2d5f7 push 0x140; push 0xa7a90; call 0x2d620`：literal destination
  `0xa7a90-0xa0000=0x7a90=(16,98)`，而stable digit compositor是`(16,99)`。
  production此前錯用stable offset，整個roll低一列；新增獨立
  `NativeShopGoldRollOffset=98*320+16`，保留stable `NativeShopGoldOffset`。
- 修正後debit captures 26/27/28/31/32/33/34/35/37/38/39/40/41/44/45/46
  分別找到source-built candidate，整幀`compare -metric AE=0`，覆蓋45-phase
  odometer的早／中／末段。29/30/36/42/43為DOSBox hotkey中斷
  `0x2d620→0x373c4`逐列memmove的partial writes（AE 24/9/6/9/8），不能要求
  atomic remake製造同樣tearing。
- 新增`shop-purchase-debit-ch02-original-vs-remake.png`，選五個AE=0 phase
  作上原版／下remake對照；unit regression明確鎖定roll比stable高一列。
  debit compositor可升E2。當時success仍為AE=2；此差異已由稍後章節修正為
  25個原子樣本整幀AE=0。整條route仍使用三處verified battle-skip patch，
  不是正常玩家playthrough gate，這項限制仍有效。

## 2026-07-29 durable agent memory and Docker cleanup

- 新增repo根目錄`AGENTS.md`作跨session唯一操作契約：保存文件權威順序、
  E0–E3、fail-closed、normal-player-path gate、已知人物／DATO更正、
  Docker-only Capstone、重大更新才commit與Codex身份。`CLAUDE.md`保留專案
  目標但明確指向`AGENTS.md`，避免兩份規則漂移；`~/.codex/AGENTS.md`另保存
  不限FD2的Docker生命周期與image清理鐵則。
- 稽核時發現四個`fd2-go-test-local` container已持續21小時。四個均已
  `docker stop`，因原本是`--rm` container而停止後自動刪除；再次
  `docker ps -a`確認無FD2 container。以後one-shot一律`--rm`、Xvfb用trap，
  每批工具／測試後立即查running/stopped container。
- image盤點刪除已由合法authorized workflow取代、repo無引用的
  `fd2-ida-local`（3.6GB）。保留目前仍有明確用途的
  `fd2-cap-local`、`fd2-go-test-local`、`fd2-dosbox-screenshot-local`與
  `fd2-ida-authorized-local`各一份；不得global prune其他專案資源。
- repo-wide斷言續審確認多數2026-07-28高風險句已收窄；補修
  `39-ani-afm-format.md`，不再把九個AFM resources直接稱為九段已完成、
  可獨立播放的開場流程。caller／章節／title sequence仍須逐一驗證。

## 2026-07-29 購買成功動畫兩像素的畫面緩衝區沿革

- 依新Docker鐵則，以單次`docker run --rm --network=none`及唯讀repo掛載，
  在`fd2-cap-local`重讀購買呼叫者；命令結束後以`docker ps -a`確認沒有
  FD2容器殘留。
- `0x2f426 call 0x15f84`先建立選擇裝備的詢問與背景存底，
  `0x2f455 call 0x16559(0)`才畫第0幀人物；Yes分支在
  `0x2f4a1 call 0x2d31b`完成關框與恢復後，立刻
  `0x2f4a6 call 0x2f4c6`播放購買成功動畫，輔助函式到`0x2f543`才再次
  呼叫`0x16559(0)`。這固定了成功動畫前後的人物寫入順序。
- `0x1956b`指令進一步證實它先配置三份`0xfa00`緩衝區，把當下VGA完整保存
  到`[0x53c5f]`，再建立對話格與DATO基底；`0x2d31b`則以
  `0x373c4(...,0xfa00)`完整恢復該存底，並非兩像素複製邊界。
  FDOTHER商店背景的`(175,90)/(176,90)`本來就是190/190，DATO#128第0幀
  才是96/191。`ComposeNativeShopBareScene`先前名為裸畫面卻提前覆蓋DATO
  第0幀，等同把原版`0x2f543`的尾端人物恢復移到成功動畫之前。
- 已移除該提前覆蓋，沒有寫死任何像素。真實資源回歸鎖定裸畫面190/190、
  success首幀190/190及尾端恢復96/191。重生來源影格後，26張DOSBox內建
  success抓圖中25張各有未遮罩整幀AE=0；第15張只在`0x16886`效果寫入途中
  差`(184,47)/(184,49)`兩點，下一張同一來源影格即AE=0，故不列為原子畫面。
  `shop-purchase-success-ch02-original-vs-remake.png`已改用四組AE=0影格。
  成功動畫合成切片可升E2；正常未修改玩家路徑與其他商店子面板仍未閉合。
- 2026-07-29 整備輸入與最終確認勘誤：Docker Capstone 重讀
  `0x318ad..0x321c8`，確認 `0x32004` 的 `0xe0/0x52` 會直接改寫成
  `0x1c`，`[0x53a8d]==0x20` 亦優先改寫成 `0x1c`；只有未走前述
  分支且 `[0x53a8e]==0x53` 時才改寫為 `1`。同一呼叫端證實選取數量達上限時，
  `0x31a68` 先令 `edi=1` 並由 `0x320fc` 依旗標重排 0x50-byte 隊伍
  記錄，之後 `0x31d3c..0x31db4` 仍顯示最終確認；不是選滿即直接進
  戰場。程式已修正位元組正規化；當時暫接的「小隊不足仍進確認」已由
  下一段外層呼叫端證據撤回。確認框文字與外觀仍是重製介面，不宣稱原版視覺一致。
- 2026-07-29 整備外層呼叫端勘誤（取代上一段「小隊不足仍進最終確認」）：
  Docker Capstone 重讀 `0x2d093..0x2d190`，確認城鎮 option2 先顯示
  FDTXT `0x201`「要進入戰場嗎？」；早期 `[0x53bfb] <= 0x10`、後期
  `[0x53bfb] <= 0x14` 時完全不呼叫 `0x318ad`，直接離開城鎮。
  `0x318ad` 又只處理 `[0x53bfb]-1` 筆可選記錄，故門檻正是15／19人。
  真正進入選人時，`0x318c7` 會把30個旗標全清為零；舊重製的預先全選、
  每章固定顯示勾選面板、以Escape讓不足上限的小隊出發，三者均已撤回。
  23個城鎮整備節點新增可編輯 `cancel→town_chNN`；流程改為出發確認→
  小名冊直接出發／大名冊全零選人→選滿後最終確認，任何取消回城。
- 同一正常路徑回歸另抓到第1章第3回合哈諾 `join_party` 只更新 membership
  與 chronology、未建立 `partyRoster`，導致戰後 native-identity sync
  封閉略過。現改為只從同一戰場中唯一、已成我方且身分相符的真實記錄建立
  持久快照；缺失或重複仍停止猜造。序章四人→哈諾加入→戰後五筆同步→
  羅德鎮→整備的連續回歸已通過。`preparation-current-remake.png` 亦從目前
  原始碼重建為第一階段出發詢問；它仍是重製成果圖，不是原版實機差分。
- 同日再重讀 `0x2cad7..0x2cd04`，確認無城鎮 gate 才使用 FDTXT `0x19a`
  「要記錄戰況嗎？」：肯定分支呼叫 `0x30012(0)`，否定分支略過存檔，
  兩者都進 `0x318ad`；選人取消則重新進全零選取流程。生成器與
  `campaign_full.json` 已把23個 town-backed prompt 改成「要進入戰場嗎？」，
  其餘 preparation-only 節點保留記錄詢問，不再混成同一語意。

## 2026-07-29 原版檔案版本基準與戰鬥人工智慧專題

- 所有既有 `FD2.EXE` 位址現在明確綁定大小 `357074` 位元組、MD5
  `b97caf2239a27a896069d03549d96e1e`、SHA-256
  `222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f`。
  `docs/data/fd2-reference-files.json` 另列目前解析器使用的 12 個資產檔；
  `tools/hash_fd2_reference.py` 可對玩家自備原版唯讀重算。不同雜湊不可直接
  沿用位址。存檔與暫存檔因內容可變，刻意排除。
- `11-enemy-ai.md` 已升為戰鬥人工智慧專題入口。現有可信鏈是 phase-specific
  scan `0x1A4EB/0x1A58F→0x1D80B/0x1D8BA→0x13A9F`，再到
  `0x14EF0` 候選分派及 `0x15AD8→0x15B77` 原始評分邊界。舊
  `0x15140/0x15356` 的逐落點、傷害門檻、擊殺加倍完整公式已刪除，不得接入
  runtime。下一步先閉合 `0x1548E` 與固定存檔動態 trace。
- 後續同日已閉合 `0x14237..0x145CC`：它才是物理候選評分迴圈，保存
  候選格×目標列舉、雙方 `+0x48/+0x4A` 地形修正、差值`<=2`拒絕、
  嚴格超過target `+0x40`時分數×2/priority18、`0x1DEBE`／raw `+8`
  調整及 priority→score→先出現者同分規則。新增
  `ScoreNativePhysicalAttackCandidate` raw-only adapter/regression。
- `0x1548E` 本身已更正為「選擇結果執行」：消費
  `0x53C43/47/4B`，經 `0x14B78` 移動／呈現後依 `0x53AF9` 選地圖鏈或
  `0x28A6C(actor,target)`，最後固定回1。它沒有呼叫 `0x4EE40/0x4F355`，
  不得再稱 pathfinder。後續已閉合
  `0x145CD→0x4E040→0x146D1→0x14B16`：另一 selector 組的中心格
  `0x40` 阻擋、相鄰格 `0x80` 為扣成本後 budget 歸零的終點，移動成本由
  FDFIELD tile 高位分組→FDSHAP `+1` move code→`0x4E555` cost row 取得；
  同組佔用格事後排除，最後按 Y/X row-major 枚舉。新增 fail-closed
  `NativeAIPhysicalDestinations` 與 regression。下一步追 `0x14B78`
  最後一個參數及無攻擊方案 fallback。
- 後續已閉合 `0x14B78→0x4E1A6→0x13488`：方向碼
  `0/1/2/3=下/左/上/右`，使用原版地形成本列與 `0x40/0x80` gate；
  目的地不可直接到達時會用 budget 28/mode 1 取得引導路徑，再依
  Manhattan 距離→XY 軸差→`0x14B16` 逐列先後選實際落點。第四參數是
  `0x145CD/0x146D1` 的零／非零分組選擇值。
- 舊「`0x15192` 是接近最近敵人」斷言已撤回；該位址屬於施法演出。
  真正無 action 備援由 `0x13A9F` 證實：一般 mode 0 先呼 `0x14121`
  以 `0x4E1A6` mode 2 搜尋另一組的 `0x40` blocked cell；仍失敗才呼
  `0x13E9C`，依 runtime record 順序與 Manhattan 距離選另一組座標。
  新增 `NativePathDirections`、`NativePathBlockedCoordinate`、
  `SelectNativeMovementDestination`、
  `SelectNativeNearestOppositeCoordinate` 與 regression；尚未猜接
  `NextAIPlan`。

## 2026-07-29 原版整備選人主畫面第一個生產切片

- 只以 `fd2-cap-local`、唯讀掛載與無網路容器重讀
  `0x318ad..0x32004`。`0x31a7c..0x31b08` 證實輸入為左／右一格與
  上／下一列十格，原先重製版「只有上下逐項移動」已撤回。
- `0x3195f..0x319be` 建立 320×200 畫面：FDOTHER #5 第20項
  223×86 放 `(92,7)`、第21項 310×99 放 `(5,94)`；第137項 86×86
  經 `0x16886→0x4e63d` 放 `(5,7)`。第31～40項供
  `0x187d6` 兩位數計數，位置為 `(61,35)` 與 `(61,73)`。
- `0x31e80` 的角色格是 10 欄：`x=23+28*(i%10)`、
  `y=100+30*(i/10)`。FDOTHER #1 descriptor0 游標先畫在 `y+4`；
  已選 FDICON pose0/cycle 以 `0x4deda` 畫在 `y+3`，未選以
  `0x4de56` 映到色盤 24～31 畫在原 y。原始 selector key 缺失時，
  生產路徑退回既有介面，不用角色編號猜圖像。
- 新增 `NativePreparationAssets`、原子合成器、四向游標規則與
  `cmd/fd2-preparation-oracle`。真實原版資源回歸和遊戲層針對性測試已通過；
  [`preparation-roster-compositor-partial.png`](../figures/preparation-roster-compositor-partial.png)
  以原始圖像索引 0～19 顯示 20 格，只是資源／版面局部證據，不是原版
  DOSBox、正常戰役名冊或晚期存檔。
- 後續重讀 `0x17fc0..0x182ab`，確認它就是既有教會／物品面板使用的同一
  個 0x50 位元組角色記錄合成器。整備生產路徑現先畫背景／名冊，再把游標
  角色交給 `NativeItemPanelRecordForUnit→RenderNativeItemPanelData`，
  重現姓名、種族、職業、HP／MP、等級、經驗、移動與能力值；缺任何原始
  記錄欄位即整張退回，不從正規化名稱或角色編號補猜。成果圖以 ch01 索爾
  的真實可編輯情境記錄填入右上狀態。
- `0x1297d` 待機週期已接：整備活躍時由獨立 `nativeBIOSClock` 取有號低字，
  重用 `fdicon.AdvanceNativeMapSpriteCycles` 的 `delta>4 || delta<0` gate，
  再用 `NativeFrameIndex` 將 raw state3 正規化為圖幀1。邊界回歸固定
  tick 0/4/5/10/15/20 的 raw `0/0/1/2/3/0` 與可見
  `0/0/1/2/1/0`，另覆蓋 signed wrap；非整備或非選人狀態不消耗時鐘。
- `0x31d3c` 穩定最終確認已接。Docker Capstone 固定呼叫順序為
  `0x1956b(0x4b)`、FDTXT `0x292` 寫至 `0xa951f=(95,119)`、
  `0x16559(0)`、`0x19953`；`0x1956b` 的非 `0x80..0x84` 分支把
  DATO #75 放在偏移 `0x9017`，而最後 `0x16559(0)` 會以第0幀覆蓋文字
  重疊區，因此 E1 穩定圖只露出右側「戰場嗎？」。正式路徑使用相同
  FDOTHER #5 對話框、DATO #75 商店店員、FDTXT 與 FDOTHER #2 Yes／No；
  缺資源即退回。保存
  [`preparation-confirmation-compositor-partial.png`](../figures/preparation-confirmation-compositor-partial.png)，
  明確不是 DOSBox 截圖或正常玩家路徑。
- 最終確認生命週期已接正式路徑：`0x1956b` 六個對話框帶狀幀之後才跑
  `0x19953` 四個選項展開幀；穩定態沿用兩個 BIOS tick 的二位元脈動。
  確認、取消或 Escape 都先跑 `0x197e5` 四幀、`0x2d31b` 五幀與原畫面
  還原，只有 Draw 確認還原幀後才轉場或重選。真實資源測試固定
  10 個開啟幀、9 個關閉幀、還原幀與延後 continuation；保存
  [`preparation-confirmation-lifecycle.png`](../figures/preparation-confirmation-lifecycle.png)。
- 兩個 `0x318ad` 呼叫端已用 Docker Capstone 重讀：城鎮出發路徑
  `0x2d16b..0x2d17c` 在回傳 0 時跳 `0x301ec`，退出本次出發；直接整備
  `0x2ccd6..0x2cce7` 在回傳 0 時跳 `0x2cccc` 再次呼叫選人。因此正式路徑
  保留「城鎮取消／直接整備重選」，不是依介面文案猜測結果。
- 整備前置提示已接原版索引色生命週期。`0x2d0d1` 城鎮路徑以
  `0x1956b(0x4b)` 開框、FDTXT `0x201` 寫至 `0xa951f=(95,119)`；
  runtime 在離開 town 前保存該幀，關框最後還原同一畫面。`0x2cc04`
  無城鎮路徑先把 64000-byte VGA 清為0，再於 `0xa9524=(100,119)` 寫
  FDTXT `0x19a`；肯定結果只在 4＋5＋還原完成後呼叫既有存檔流程。
  兩者都使用 DATO #75、FDOTHER #5／#2、十幀開啟、兩 tick 脈動及九幀
  關閉。保存
  [`preparation-record-prompt-lifecycle.png`](../figures/preparation-record-prompt-lifecycle.png)
  與
  [`preparation-departure-prompt-ch02-lifecycle.png`](../figures/preparation-departure-prompt-ch02-lifecycle.png)；
  後者是 ch02 variant0／selection2 的 E1 原始資源合成，不是 DOSBox 擷取。
- 尚未完成：跨戰場／城鎮的行程全域初始相位，以及合法晚期 `FD2.SAV`
  的同狀態原版／重製差分。`0x1f42d` 已更正為戰場進入演出，不再列為
  選人視窗缺口。這些缺口使 UI-11 維持部分完成。

## 2026-07-29：敵方回合行動模式來源閉合

- `0x10FB6` 證實 FDFIELD 名冊 `b17/b18/b19` 分別建立 runtime
  `record+0x34/+0x35/+0x36`；`b2` 建立 `+0x3D`。
- 同版 FDFIELD.DAT 的 33 張圖共 1887 筆名冊已重解析；低四位初始值只出現
  0、1、2、3、4、5、7、8、9、10，完整分布保存於
  `docs/data/fdfield_native_ai_modes.json`。高四位另有用途，不可把整 byte
  稱為單一模式。
- `0x3419C` 是保留高四位、替換低四位的 inclusive range writer；
  `0x13D20` 與數個章節處理器另會整 byte 覆寫。重製新增對應的 raw writer、
  provenance materializer、資料匯出與回歸測試。
- doc11 已新增敵方回合專題矩陣：原版依 mode 選擇候選攻擊、指定座標／單位
  移動或無候選備援，不是單純選最近角色。矩陣只記可觀察呼叫，不替
  0/1/2/3/4/5/7/8/9/10/11 猜玩法名稱。
- 當時的重要缺口：`NextAIPlan` 尚未消費這些 raw mode；ch01 的
  `set_ai:berserk` 只寫 inert 事件標記，沒有規劃器讀取。模式 2／11
  與已知 writer 已由下一節繼續閉合；其餘模式仍不能猜值硬接。

## 2026-07-29：模式 2／11 與休息回復閉合

- 修正 mode 2 控制流誤讀：`0x14EF0` 失敗後是
  `0x14237→0x13FD4`；`0x14237` 在 `0x145C3` 固定回傳 0，因此不會走
  `0x13E9C` 最近座標備援。
- mode 11 有兩個獨立 signed `>=6` gate：`[0x53C23]` 控制
  `0x15311`，但第一段後仍無條件跑 `0x14237`；`[0x53C4F]` 控制
  `0x1548E`，不足才走 `0x14121→失敗時 0x13FD4`。
- `0x13FD4` 已閉合為 raw `+0x25/+0x26` 皆零時的 HP 回復：
  `min(currentHP+floor(maxHP/5),maxHP)`。新增 state-only adapter，
  玩家休息正式路徑也移除「至少回復 1」錯誤近似並接 raw transient gates。
- 初始 FDFIELD 沒有 mode 11；唯一已知 writer 是全域 event 82 handler
  `0x35F92` 內
  `[0x53AD5]+0x10==4 → 0x3419C(20,20,11)`。IDA 確認函式邊界與 generic
  dispatcher xrefs，但一般玩家觸發仍未知；33 張 FDFIELD 格子事件表沒有
  event 82，禁止命名成踩格、特定人物、章節或狂暴。

## 2026-07-29：全域事件表與格子事件更正

- `0x51B91` 的重定位連續到 `0x51CF5`，是 event 0..89 共 90 entries；
  先前依 FDFIELD turn-events 最大值截成 58 entries 的說法已撤回。
- `0x35F92` 是 slot 82 指向的全域 event handler，不是另一張 30-entry
  function table 的 entry 22。
- `0x13A44` 證實 FDFIELD 控制段 `+0x33` 是 16×2
  `(event_id,selector)` 格子事件表。地圖 event-word low5 是 1-based
  slot，且 FDSHAP control byte0 的 `0x20/0x40` 皆零才走此路徑。
- 33 張 `map.json` 已同步 `native_field_event_slots` 與
  `native_field_events`，重製端只提供失敗即關閉的 selector 查詢，尚未
  dispatch 未解 handler。全圖沒有 event 82，這是排除「mode 11 writer
  由踩格觸發」的直接資料證據。

## 2026-07-29：後處理列與全圖寶物資料更正

- `0x1AA1D` 的列是 `{kind:u8,payload:u16le}`；kind0/1 給物品／金錢，
  kind2 以 payload dispatch `0x51B91`，kind3 走另一呈現分支。
- `0x10FA8..0x10FB2` 只把 FDFIELD b22 與 b23..24 複製到 runtime
  `+0x31..+0x33`。舊解析器把 b23..25 合成 24-bit value 是錯誤斷言；
  b25 在全 1887 筆皆零，但仍只作未命名來源 byte 保存。
- `0x190AC` 的寶物控制列同樣是三位元組：type0=item、type1=gold，
  其餘 type 以 value dispatch 全域 event。新增選擇性同步器，把 33 圖
  寶物格、hidden flag、16 槽 type/value 寫入既有資產而不覆蓋人工單位校正。
- map25 有 14 個普通箱圖塊，其中 slots0..4 是 type2/event58。重製端已能
  看見這些 editable event treasure，但在 event58 handler 尚未 lower 前
  取得會失敗且不設 opened，避免舊資料錯給 item58。
- event82 仍未出現在 turn／field／treasure／unit effect 資料；四個 EXE
  硬編碼 `0x1AA1D` 單列亦只有 kind0 的 D3、D5、0x65、0x0B。暫列
  「無已知資料 producer」，仍待 runtime effect writer 稽核，不宣稱 dead code。

## 2026-07-29：map25 event58 五選一寶物閉合

- `0x354FE` 先查行動單位八格 inventory；滿欄只顯示 FDTXT_000 `0x1E0`，
  不改寶物狀態。
- 有空欄時以 `0x12E38` 取得 slot0..4，查 EXE `0x5274E` 五 bytes
  `[1D,2B,33,3D,47]`，由 `0x1BB8C` 寫入對應物品。
- 成功後 `[0x53AD5]+0..4` 全設 1，因此五箱只能擇一，不是五箱各自可拿。
- 新 extractor 輸出含 FD2.EXE 雜湊與位址的 editable rule；map25 asset
  嵌入規則，runtime 以 typed data 執行並共同標記 slots0..4 opened。
  沒有規則的特殊 event treasure 仍失敗即關閉。

## 2026-07-29：map25 field events 59／60／61 閉合

- event59：selector0、y36 全列；觸發單位 runtime `+6 !=0` 時把單位
  39..44 的低四位模式設0。
- event60：selector0、y22 全列；同 gate，把 23..24、53..56 設 mode0。
- event61：selector1、`(1,46)`；entry12 未設且觸發單位持有 D0 才移除
  物品、播放 resource45×59 frames、設 entry12、spawn group1、JOIN31，
  並使用 FDTXT 3/4；缺 D0 使用 FDTXT2 且不改狀態。
- 新 `native_field_event_rules.json` 綁定 EXE 雜湊，map25 asset 嵌入三條
  editable rules。這是當時的資料化狀態；selector0/1 後續接線進度以本檔
  較晚的段落為準。
- `ApplyNativeFieldModeEvent` 已執行 event59/60 的完整 provenance
  preflight 與原子 mode-range 寫入；上層尚未證實 selector 時機，所以沒有
  自動掛到任意 walk completion。event61 仍只載入 typed core rule。

## 2026-07-29：selector0 向左格步驟時機閉合

- 合法 IDA 9.4 固定 `0x13488` 依路徑 byte 分派：只有 byte1 呼叫
  `0x1300D`。後者七拍動畫提交 runtime `+0`／`[0x53AB1]` 的 `x-1`
  後，才以新座標呼叫 `0x13A44(...,0)`。
- `0x12E38` 以 `x + mapWidth*y` 定址，交叉確認第一參數是 x；所以
  selector0 不是泛稱移動結束，而是每一個向左格步驟提交後。
- `stepBattleWalk` 已在相同提交點執行 event59/60。map25 實檔測試固定
  第1..6拍不改狀態、第7拍原子改 39..44，以及向右跨過同格不觸發。
- selector1 雖已定位到 `0x13A9F` 共用行動收尾與 `0x18890` 多個成功臂，
  仍不能簡化成 walk completion。這是當時尚未接入呈現的狀態；後續
  wait 成功臂接線與尚未達 E2 的限制見較晚段落。

## 2026-07-29：FDTXT_026 計數與 event61 初始名冊勘誤

- 舊「FDTXT_026 raw 61／authored 63」不是原始資料差異，而是
  `count_logical_utterances()` 只計算以開引號 glyph 起始的對話。
  string2／3 是 `0x15F84` 直接顯示的未加引號、無頭像訊息，應各計一次；
  修正後全資源 63 個顯示單位與 `ch26.json` 63 lines 完全對齊。
- event61 的 FDTXT string2／3／4 現可精確映射到 ch26 scene0 line10、
  scene0 line11 與 scene1 lines0–9。2026-07-20／07-27 的舊 mismatch
  敘述只保留歷史過程，以本節勘誤為準。
- presentation 規則補齊為 `FDOTHER.DAT` resource45、59 frames、
  destination offset48356、stride320、transparent -1、每幀 delay2。
- `ch26.json` 已切換至 runtime append construction，initial group 只保留
  group0；group1 的 char31 渥德留在 pending roster，等待 event61 成功臂。
  測試固定事件前 active Wold=0、pending Wold=1。
- 當時尚未完成的正式路徑是 selector1 成功收尾、59 幀畫面擁有權、D0 原始
  inventory 原子移除、entry12、append group1 與永久 JOIN31；後續接線
  狀態與仍未達一般玩家 E2 的限制，以本節較晚的勘誤為準。
- 同輪 battle core 新增 `PlanNativeFieldEvent61`／
  `CommitNativeFieldEvent61`。前者無突變地預檢 selector1、typed rule、
  entry12、八格原始 inventory projection 與 pending group1/char31；缺 D0
  只回傳 FDTXT2。後者要求精確 59 frames，重驗 D0 後以 `0x1B8E7`
  對應 adapter 移除、設 entry12、`AppendGroup(1)`，並回傳 char31 供
  campaign owner 持久化。58 frames 與中途 inventory 變更回歸均固定為
  零部分寫入。這是核心層完成、UI job／永久 JOIN 尚未接線時的歷史狀態；
  現況見下一項。
- 後續接線勘誤：UI job 與永久 JOIN31 adapter 已完成；當時只在明示
  materialized native runtime 測試中成立。job 依真實 FDOTHER.DAT 跑
  FDTXT3→59 frames×2 BIOS ticks→commit→persistent JOIN31→FDTXT4；
  缺 D0 只播 FDTXT2。這個限制已由下一項的 pre-handler/cursor 資料流
  解除；仍不得把 production E1 冒稱同狀態畫面 E2。
- 同輪發現 repository-wide UI 根因：只有 map0 的 map.json 保存
  `native_tile_blit_modes/native_terrain_control`，map1–32 均為空。
  新同步器先對 FDFIELD.DAT／FDSHAP.DAT 做 size+MD5+SHA-256 驗證，再
  只補 composition byte+3 與 terrain control。33 圖 check、loader
  regression 與完整 Docker/Xvfb Go suite 均通過。這只是全圖 E0
  renderer input completeness，ch02+ dynamic view/HUD 仍待逐章證據。

## 2026-07-29：current-runtime loader 與 ch26 event61 production view 勘誤

- 舊文件把 `main→0x25ebb→0x10010` 稱為一般 cutscene→戰場線性鏈是錯誤
  斷言。直接展開 `0x25ec8..0x26151`：新遊戲與四槽讀檔分支各自呼叫
  pre-handler 後返回；只有第三主選單分支在 `0x26130` 呼叫 `0x10010`。
  main 於 `0x25dbd` 呼叫 `0x25ebb`，回傳0才在 `0x25dce` 進 `0x117e7`。
- `0x10010` 開啟 FD2.SAV、驗 rolling-XOR/checksum，`esi=buffer+0x30c3`；
  `+0/+1/+2/+9` 分別恢復 turn counter、runtime count、chapter、
  persistent count；`+3..+8` 恢復 camera/cursor/visible cursor，
  `+15` 恢復 HUD gate A。plaintext `0x08a3` 的固定 `0xa00` bytes
  載入 persistent roster，`0x12a3` 則依 runtime count 載入 runtime
  roster；其後另載 field units、event table。因此它是 current-runtime
  snapshot loader，不是一般章節 loader。兩個真 caller
  `0x1a251/0x26130` 的 callgraph 結論仍有效，但不能推導跨分支 fallthrough。
- 寫入端 `0x19f7a..0x1a136` 把相同 runtime blobs 與18-byte header逐欄
  回寫 FD2.SAV。gate A 另由系統選單 `0x173ba` XOR 1，且保存於四槽
  metadata `+6`；它是持久 raw option，不是 renderer hardcode。
- 一般 ch26 view 改由 pre-handler 證據建立：ch25 pre 最後
  `PAN(9,39)→FOCUS_UNIT(0)`，slot0 部署於 `(15,46)`，得到初始
  camera `(9,39)`、cursor `(15,46)`、visible `(6,7)`。沿原版 cursor
  state machine 到 event61 `(1,46)` 後為 camera `(0,39)`、visible `(1,7)`，
  HUD anchor 同時確定寫 `0xf2`。`battle_ch26` 已資料化初始 view/HUD；
  event61 真實資產 regression 不再手工 materialize。這是 production E1，
  尚缺同 roster/event/tick 的 DOSBox 像素比較，未升 E2。
- 2026-07-30 native persistent party narrow bridge：新增綁定參考
  FD2.EXE SHA-256 的 `native_character_catalog.json` 與
  `MaterializeNativePersistentPartyRecord`。它只投影 record 已證實欄位；
  portrait／sprite／map selector／座標／章節不猜接。後續合法
  IDA Pro 9.4 重核證實 `0x30B07..0x30B17` 與
  `0x310B5..0x310C9` 直接使用 `FDTXT[150+record.class]`；固定雜湊
  FDTXT 的 class 27 是兩個全形空格、class 28 是「？？？」。因此 catalog
  已擴至 0–28；原先 `cls28`、`?`、fallback「職業28」造成的「來源衝突」
  是重製端錯誤斷言，不是原版不確定性。Docker 唯讀原版 `FD2.SAV` 的可選
  integration test 已由 current snapshot 實得索爾、悠妮、亞雷斯、蓋亞；
  此橋尚未成為正式 CONTINUE owner。

- 2026-07-30 current snapshot owner／漏列 raw 區域勘誤：合法
  IDA Pro 9.4 重讀 `0x10010..0x10620`，Capstone 獨立核對，確認舊證據漏列
  plaintext `0x0000..0x08A2` FDFIELD 控制映像（複製後呼 `0x10652`）與
  `0x30A3..0x30C2` battle-local event state（複製至 `[0x53AD5]`）。`InspectCurrentSnapshot`
  現原樣保存兩區，並測試不與呼叫端緩衝區共用底層資料。`0x10010` 自己載資源、
  建 runtime selector、恢復畫面。**2026-08-02 勘誤**：`0x10616` 呼叫的
  `0x4E031` 只複製 BIOS 鍵盤緩衝 word，不是戰鬥驅動；
  `0x1061B→0x22BBE` 經共享 epilogue 返回 main 後，才由 `0x25DCE` 呼叫
  `0x117E7` 共享戰鬥控制器。重製端真正缺的是已知 current-runtime 狀態與
  chapter pending groups 的嚴格綁定，以及正式 `Game`／controller handoff。
  四槽 LOAD 的 `0x2CAD7→pre-handler`
  仍是另一條 ABI。

## 2026-07-29：selector1 全成功動作擁有權接線

- Docker Capstone 固定玩家外層 `0x18890` 的 selector1 呼叫點為
  `0x18AEF/0x18B0C/0x18B66`：它們都在 `0x18D8C` action handler 成功
  返回後；返回 `-1` 的取消／復原臂不呼叫 selector1。AI 另在
  `0x13E77` 的動作收尾呼叫相同 selector。
- 重製端新增共同成功閘門 `finishSuccessfulUnitAction`，由待命、攻擊、
  一般法術、原始指令、即時／指向／移位物品與 AI 共用。各 executor 必須
  先成功完成 mutation；目標不合法、使用者取消及錯誤返回不會觸發。
- 攻擊與攻擊型法術不能在結算資料寫入時立刻進事件，否則會遮蔽全螢幕
  FIGANI。`atkAnim.after` 現保存 selector1 owner，演出結束後才啟動
  event61；沒有全螢幕演出的成功動作則立即進閘門。
- 真實 ch26／FDOTHER 資產回歸新增兩條玩家路徑：攻擊在演出結束前不得
  出現 FDTXT2，結束後才出現；即時物品必須先成功消耗，再進 FDTXT2，
  對話完成後才清理選取狀態。完整 Docker/Xvfb `go test ./...` 通過。
- 這關閉重製端 selector1 的 E1 動作擁有權，不提升為 E2；仍須以未修改
  原版、相同名冊／事件狀態／tick 的一般玩家路徑比較。

## 2026-07-29：AI 上層兩遍掃描控制流補證

- 以 `fd2-cap-local`、無網路、一次性 Docker 容器重生三份內含 EXE
  大小與 MD5/SHA-256 的完整指令產物：
  `fd2_ai_phase_setup_disasm.txt`、`fd2_ai_unit_scan_disasm.txt`、
  `fd2_ai_mode_dispatch_disasm.txt`。
- `0x1D80B` 只處理 raw `record+6==1` 的合格記錄，呼叫
  `0x13A9F(unit,1)`。`0x1D8BA` 對 raw `+6==0` 不是單遍：第一遍先跑
  `0x1598A/0x1567E`，只有 `[0x53C23]` 或 `[0x53C33]` 的有號分數至少6
  才進 `0x13A9F(unit,0)`；第二遍 `0x1D988` 對相同 raw gates 直接進
  `0x13A9F(unit,0)`。
- 每筆 mode 在 `0x13E77` 固定執行 selector1、`0x13512` bit7、
  `0x134E4` 與重畫，再回掃描器；掃描器其後依序執行可選的90-entry
  全域事件表、章節戰場事件 handler 表，並讀 `[0x53ECC]` pending 碼
  決定是否離開。這三者先前已閉合，不是本輪新增的未知 callback。
- `[0x53BEF]` 的增加在第二掃描返回後才發生。交叉核對既有
  `FDFIELD b0→record+6` constructor writer 與 `0x14818` target-code
  consumer，raw camp code 敵0／友1／己2 已閉合；因此 `0x1D80B` 是友軍
  單遍，`0x1D8BA/0x1D988` 是敵軍預選＋第二遍。真正未閉合的是敵軍為何
  需要兩遍及兩個分數的玩法語意，不再把 raw `+6` 列為未知。
- `fdother.PlanNativePhaseUnitScans` 已將 selector1 單遍、selector0
  分數預選與 selector0 第二遍分離輸出；預選只在 caller-supplied signed
  `[0x53C23]` 或 `[0x53C33]` 至少6時標記 action。三遍不攤平，以保留
  每筆 callback 後 pending 碼可能提早退出的邊界；缺 score provenance
  會失敗即關閉。這仍是 E0 規則，不接 normalized `NextAIPlan`。
- 同時撤回 AI 專題中「完全沒有攻擊方案時接近路線仍未知」的舊句：
  mode0 的 `0x14121→0x13E9C` 備援已在較早證據閉合；未知的是其他 mode
  的完整玩法命名與上層 raw phase。
- 兩遍目的也已由既有 producer 證據閉合：`0x1598A` 的法術候選最佳分數
  寫 `[0x53C23]`，`0x1567E` 的 item-command 候選最佳分數寫
  `[0x53C33]`。第一遍只有任一 signed score 至少6才讓敵軍進入 dispatcher；收尾
  `0x13512` 設 bit7，第二遍 `+5 & 0x81` 會排除它，所以不是雙動。
  typed score 欄位已改用 producer＋地址名稱，仍不猜特定 spell/item。
- 2026-07-29 AI command-index 勘誤：Docker Capstone 交叉核對
  `0x1C269`、`0x1598A` 與 `0x15311`，證實 command mask 的 set-bit
  索引直接交給 `0x4E516`、`0x15B77` 及選中結果執行端，這條路徑沒有
  `command-0x10`。舊 `NativeAvailableAISpellCommandIDs` 將索引限制為
  `>=0x10`，誤混入 `0x1567E` item-command 規則，已改為
  `NativeAvailableAIScoredCommandIDs` 並保留已知 `0..35`；
  `AIPlan.NativeScoredCommands` 仍只帶 evidence，不執行效果。完整帶
  FD2.EXE 雜湊的三段指令保存於
  `docs/data/fd2_ai_command_index_disasm.txt`。
- 同輪建立 `battle.NativeAIScoringRecords`：完整 roster 必須具備 native
  map presentation、battle fig、identity、race/class、inventory、
  `+5/+6/+34..+36` provenance，才會產生 detached `0x50` records；
  缺一欄整批失敗即關閉。map0 真實匯出資產 regression 固定第一筆座標
  `(1,3)`、`+5=0`、enemy code 0、mode byte 0 與 HP 28。這只是 E0
  score input；尚未接候選建立、完整 score 或 production `NextAIPlan`。
- 接續完成 `battle.NativeAIScoredCommandCandidateGroups`：command record
  `+3` 經 `0x4E040→0x14B16` 建 row-major destinations，`+4` 經
  raw `0x14818` target predicate 建 roster-ordered indices；selector0
  的 target-code 轉換、`+5 bit0` inactive gate、空 target skip 皆有
  deterministic regression。map0 真實 roster＋command #0＋movement row0
  驗證 identity103 actor 在 `(23,14)` 命中 ally index。仍未接
  `0x15B77` score、best selection、presentation 或 production planner。

## 2026-07-29：AI 地圖原始輸入與群組評分勘誤

- 檢查實際資產後發現，33 張地圖共 1887 個單位原本全都沒有
  `initial_command_mask`；先前「命令遮罩已在地圖資產」的理解是錯誤
  斷言。`sync_native_selector_fields.py` 現由 FDFIELD b13..b16 同步五位元組
  遮罩，重新檢查為 33 張圖、0 缺漏。263 筆非零遮罩中，261 筆在命令表與
  原始 MP 成本閘門下至少有一個可用命令。
- Docker Capstone 直接重讀 `0x10d7f..0x1100c`，閉合建構器的 MP 公式：
  高階分支 `high[+4]*level`，低階分支
  `u16(lower[+5])+lower_aux[+8]*(level-1)`；結果同寫 runtime
  `+0x44/+0x46`。新增 `native_record_word46` 投影，1885 筆地圖單位具備
  完整來源；map32 兩筆未覆蓋 selector 保持缺值與失敗即關閉。
- 載入器對具備來源的 scripted roster，以 `word42` 初始化
  `HP/MaxHP`、以 `word46` 初始化 `MP/MaxMP`。這修正原先只保存來源、
  卻讓 AI 消費錯誤正規化數值的管線缺口；舊式編輯列缺欄位時仍沿用原值。
  map19 unit55 固定 identity92、mask `[4,0,0,8,0]`、MP288、可用命令
  `[2,27]`，其 detached runtime `+0x44/+0x46` 皆為288。
- 新增 `ScoreNativeAIScoredCommandGroups`，依 `0x15B77` 的完整 ID 家族
  分派攻擊、恢復、旗標與零分 scorer。map0 command0 的四個友軍目標各得24、
  合計96；IDs10..12 缺 `0x1F183` caller gate 時拒絕執行。
- `[0x53C23]` 數值最大值可由零開始比較，但函式區域命令字在全零分時的初值
  尚未由 caller／prologue 證實。因此下一步可以接非負數值最大值與非零勝者，
  不能猜測零分命令，也不能直接接正式 `NextAIPlan`。
- 提交 `5e6a013` 已將上述地圖輸入、HP／MP 建構器、群組評分與文件勘誤推送
  至 `origin/main`，完整 Docker／Xvfb `go test ./...`、Python exporter 測試及
  33-map 同步檢查均通過。
- 提交後的下一批新增 `ScoreNativeAI1598A`：逐一枚舉可用命令、建立候選群組、
  評分並只保存大於零的 strict winner；actor mask／MP 與 detached record
  不一致時失敗即關閉。map0 command0 得分96、目的地 `(23,14)`；全零分只回
  `MaxScore=0`。這批目前只完成 targeted Docker battle regression，尚未形成
  下一個重大提交。
- 後續 `0x1567E` 稽核又撤回兩個錯誤斷言：此路徑候選交 `0x15880`，
  不是 `0x15B77`；`[0x53C3F]` 保存 inventory slot，不是 command。
  帶 FD2.EXE 雜湊的完整窗口已保存為
  `docs/data/fd2_ai_item_preselection_disasm.txt`，並同步修正 SDD、AI 專題、
  gap audit 與 worklist。
- 新增 `ScoreNativeAIItemCommandTargets` 保存 `0x15880` 的 type5／0x0D
  HP 三段分數、raw `+0x34 bit7` 倍率，以及 type0x14／0x15／0x18 的
  threshold 分支；不為 raw type 指派效果名稱。targeted Docker battle
  regression 已通過，尚待把 inventory slot 枚舉與 caller-specific geometry
  接成完整 `[0x53C33]` producer。

## 2026-07-29：`[0x53C33]` item-command producer 閉合

- Docker Capstone 保存 `docs/data/fd2_ai_item_candidate_disasm.txt`，涵蓋
  `0x14818..0x14B78` 並綁定參考 FD2.EXE 雜湊。第一個 `0x14818`
  由 actor、row command 與 high-command marker 建目的地 field，
  `0x14B16` 依 row-major 匯出座標。
- 逐目的地時，低 command 以 row `+0x12` 與 selector-dependent
  row `+0x11` target code 再呼 `0x14818`；高 command 的 `0x149F8`
  從 actor 朝目的地走 `command-0x10` 步，此 caller 固定 selector0，
  因而只收 raw camp0。兩條 target list 都交 `0x15880`。
- 新增 `ScoreNativeAI1567E`，精確保存 `0x1B8A6` count-sized raw slot scan、
  slot→row-major destination→roster target 順序與 strict `score>best`。
  map0 roster／constructor low5 基底加 tracked item79 的 E0 fixture 固定
  score8、`(19,15)`、slot0；不宣稱一般玩家 map0 原本持有 item79。
- 同輪撤回 `0x1B8A6`「回傳 occupied prefix length」的過度斷言：它只數
  八格中 bit7-clear cells，caller 才掃 slots `0..count-1`；函式不驗 compact，
  malformed hole 可能讓 stale item byte 進掃描。程式註解、SDD、worklist 與
  舊 handoff 條目均已更正。

## 2026-07-29：AI 雙預選分數接入三遍掃描診斷

- 新增 `battle.BuildNativeAIPhaseDiagnosticPlan`，依 `0x1D8BA` 的直接指令
  順序，對每筆 raw `+6==0` 合格記錄先跑 `ScoreNativeAI1598A(unit,0)`，
  再跑 `ScoreNativeAI1567E(unit,0)`，將兩個最大值轉成 signed dword 後
  交給 `fdother.PlanNativePhaseUnitScans` 的 `>=6` 門檻。
- 每筆合格單位都必須有唯一、呼叫者提供的移動成本列及可選
  `0x1F183` 閘門；缺少、重複、越界或替不合格單位提供輸入都會在任何正式
  動作前失敗即關閉。回傳只含兩個 producer 結果與三遍掃描計畫，不呼叫
  `0x13A9F`、兩張回呼表或處理 `[0x53ECC]`，亦不修改戰鬥記錄。
- map0 修改狀態的 E0 交叉夾具沿用真實名冊、構成格 low5 基底、地形、命令表、
  物品列與成本列，排除其他 selector-zero 單位後，替 index23 注入
  command0 與已追蹤 item79；結果固定為 `[0x53C23]=96`、
  `[0x53C33]=8`，優先遍通過且第二遍仍保留。這不代表一般玩家 map0
  原始狀態。
- 同輪完整 `0x1088D→0x111BA→0x4DBFC` 重讀再撤回一層錯誤命名：
  FDFIELD 構成格 `+2` 是封存 event low byte，不是完整 live target flags。
  `0x111BA` 釋放舊指標、配置精確資源大小並讀入 `[0x53A51]`；`0x4DBFC`
  隨即執行 `+2 &=0x1F`、`+3=0xFF` 與 tile high byte `&=3`。
  `0x145CD→0x14625/0x146A7` 才依 selector／roster 加 `0x40/0x80`。
  合法 IDA Pro 9.4 優先確認這些函式邊界及交叉參照，並補正
  `0x4E040` 是搜尋入口、`0x4E0DC` 是遞迴器、`0x4E16E` 才是
  `0x40/0x80` 直接消費端；Capstone 再逐指令覆核。
  資產鍵已改成 `native_composition_event_bytes`，不再由 JSON 自動填入
  一份假性的執行期旗標（live flags）。完整指令產物為
  `docs/data/fd2_field_composition_lifecycle_disasm.txt`。
- map19 取得1600格 event bytes，其中7格非零；真實 unit55（identity92、遮罩
  `[4,0,0,8,0]`、MP288）直接執行兩個 producer 都得到零分，且不創造
  勝者。下一個證據門檻仍是固定原版動態 trace，以及逐單位回呼／pending
  code 的共同驗證；正式 `NextAIPlan` 維持未接。
- 同輪續以合法 IDA Pro 9.4 優先重讀
  `0x1D8BA/0x1598A/0x1567E/0x1CFF0/0x1BBDC`，再由 Capstone 覆核：
  每個 `0x4E040/0x14818` 候選生命週期後都呼叫 `0x4DBFC`，這些呼叫端（caller）
  都不呼叫 `0x145CD`。因此刪除 `State.NativeTargetFlags`；玩家命令與兩個
  AI producer 每次從 `NativeCompositionEventBytes` 重建獨立低五位切片（low5 slice），
  缺來源即失敗即關閉。跨呼叫 mutation 與命令家族回歸已加入，直接指令見
  `docs/data/fd2_ai_composition_flag_lifetime_disasm.txt`。

## 2026-07-29：`0x205B4/0x205BE` 三值結果規則與函式邊界勘誤

- Docker Capstone 以參考 FD2.EXE 重新確認：`0x205B4` 的共享入口
  `0x205BE` 先寫
  `[0x53ECC]=2`，掃描 `[0x53A45]` 的 `[0x53BEB]` 筆 0x50-byte records；
  任一 `raw +6==0 && (+5&1)==0` 寫 code0，最後 record0 `+5 bit0`
  可覆寫 code1。新增 `NativeBattleResultCode205B4`，保存原始順序及數值，
  不為 camp、bit 或 code 指派勝敗名稱；map0 真實 roster 錨點得到 code0。
- 舊文件把 `0x205BE` 寫成「設2後清0、呼 `0x1088D` 載章」是函式邊界
  混線。`0x205D5` 會直接跳至 `0x2067E`，不落入相鄰的 `0x205DA`；
  `0x205DA` 有另外28個 direct callers，才是清全域、呼 loader 的入口。
  直接指令與 caller 清單保存於
  `docs/data/fd2_battle_result_205be_disasm.txt`。
- 合法 IDA 9.4／Hex-Rays 另將查詢位址 `0x205BE` 歸入函式
  `0x205B4`，輸出相同迴圈與覆寫順序；交叉證據保存於
  `docs/data/ida/fd2_205b4_pseudocode.txt`。
- 已同步修正 doc24／25／26／28、SDD 與 worklist：不得再把
  `0x205C9..0x20C64` 稱為單一事件解譯器，也不得單靠 default handler
  把 raw code0/1/2 直接命名成殲滅／勝利／失敗。正式 campaign transition
  仍需逐章 handler 與外層 consumer／玩家路徑證據。

## 2026-07-29：AI 逐筆回呼表與 pending 邊界的 IDA 優先閉合

- 合法 IDA Pro 9.4 將 `0x1D80B..0x1D8BA` 與
  `0x1D8BA..0x1DA16` 固定為兩個函式，`0x1D988` 只是後者內的第二遍。
  Capstone 的既有逐指令產物再獨立覆核相同呼叫順序。
- 原始資料表邊界已由相鄰位址及 raw pointer 固定：
  `0x51B19..0x51B91` 是 30 筆章節 handler 表；
  `0x51B91..0x51CF9` 是 90 筆全域事件表。在 IDA 直接資料交叉參照中，
  `0x13A44` 是 `[0x51A8F]` 唯一非重設寫入端。
- 三段掃描對每一筆 record 都先做可選全域事件呼叫，再重新讀
  `[0x53C03]` 無條件呼叫章節 handler，最後才檢查 `[0x53ECC]`。
  即使 record 未通過 action gate 也會走尾段；即使全域 handler 已設
  pending，也不會跳過章節 handler。
- 新增 `fdother.ExecuteNativePhaseUnitScans`，保存三遍逐筆動態重判、
  第一遍 bit7 mutation 對第二遍 admission 的影響、90／30 表界與
  pending 提前退出。缺任一回呼或索引越界即失敗關閉；handler 效果仍由
  呼叫端提供，不代表正式 AI runtime 已完成。
- 重新檢查 `main 0x25BF4` 與 `0x22E5C` 後，撤回把所有 raw code1
  稱作「世界地圖／中場」及所有 code2 稱作「勝利／直接下一章」的斷言。
  code1 只證實固定載入 `FDOTHER.DAT` #79 並呈現；code2 只證實先走
  章節索引戰後表，再進 `0x2CAD7` gate，回傳零才走第二張章節表。
- 可重現命令、函式表指標與位址證據保存於
  `docs/data/fd2_ai_phase_callback_tables_ida.txt`；已同步修正 AI 專題、
  SDD、doc23–26、差距稽核與工作清單。

## 2026-07-29：`0x2CAD7` raw 回傳契約與舊名稱稽核

- 重新盤點後確認 30-byte gate、`0x2D093` selector→callee、
  `0x318AD` 整備及 town/shop/church UI 已有大量 E0／E1 工作，不再重複
  反組譯既有範圍；本輪只補真正缺少的 `0x2CAD7` raw return。
- 合法 IDA Pro 9.4 固定：直接整備 `0x318AD` 回傳0時在
  `0x2CAD7` 內重複，非零時 gate 回傳0；selectable hub 的
  `0x2D093` 回傳0也重複，option2 的非零結果使 gate 回傳0，
  option0／1／3／4 的非零結果使 gate 回傳1。
- 新增 `NativePostbattleRoute.DirectPreparation` 以區分 gate-table
  直接整備與 hub option2，並以
  `fdother.ResolveNativePostbattleOutcome` 保存內部重複及 raw 0／1。
  偽造 callee／selector 組合一律失敗關閉。
- 機械式文件稽核找到 `tools/event_handler_dump.py` 仍把 `0x2CAD7`
  標成「結局判定?」，已改為戰後 raw gate。通用 `0x51DE9`
  也由「戰後／勝利」降為「phase-2 戰後」；逐章玩家結果仍須 E0／E2。
- SDD 已更正 `0x2D093` 不是外層戰役迴圈直接 callee，而是
  `0x2CAD7` 的 selectable-hub 分支；酒店／商店／教會名稱雖有文字、
  資源及 mutation writer 旁證，具型別路由仍以 raw option→address 為主。
- 直接證據保存於
  `docs/data/fd2_postbattle_gate_outcome_ida.txt`，並同步更新介面證據矩陣、
  工作清單與戰役／腳本文件。

## 2026-07-30：原生四槽 LOAD 戰間還原擁有者

- 合法 IDA Pro 9.4 固定 `0x30012..0x301F4` writer 只有
  `0x2CCB6`（`0x2CAD7` 直接整備）與 `0x2FD93`（酒店）兩個直接
  caller。writer 與 `0x2602C..0x26098` reader 對稱處理固定
  `0xA00` roster 和 metadata `+0..+9`；`+10..+39` 只可稱為這三條
  已查路徑未生產／消費，不宣稱全程 unused。
- 由 writer caller 與 LOAD 再進 `0x2CAD7` 可知，選槽後應回到已完成
  postbattle 特殊處理的戰間入口。raw `[0x53C03]` 為 zero-based，
  `0x526B9[22..24,27..29]=1` 進 preparation，其餘可存 raw1..21、
  25..26 進 town；ch21 recipe 與 ch27 item gate 不可重播。
- 新增以參考 EXE SHA-256／`0x526B9` 綁定的
  `native_intermission_gate.json`、`BuildNativeChapterSlotRestorePlan`
  與 production title owner。完整驗證 chapter、node type、catalog、
  active roster 及 identity 唯一性後，才一次套用 fresh flags、gold、
  typed party、raw chapter 與 metadata `+6..+9` 保存值。
- Docker regression 已覆蓋完整 `campaign_full` raw1..29、ch21/ch27
  不重播、合成有效槽成功 restore，以及錯誤 route 不部分 mutation。
  使用者未修改原版四槽目前皆空，因此仍只有 E1，尚無一般玩家有效槽 E2；
  `0x10010` CONTINUE current battle 仍是另一條未接路徑。

## 2026-07-30：CONTINUE `+0x30A3` event-state 邊界閉合

- 合法 IDA Pro 9.4 的首次 xref 掃描暴露方法錯誤：`0x53AD9` 之後是
  相鄰 globals，不是 `[0x53AD5]` table 欄位。main
  `0x25D33..0x25D3D` 實際以 `malloc(0x20)` 建 buffer，再把 pointer
  寫入 `[0x53AD5]`；因此必須追 pointer 後的 indexed access。
- current-save writer `0x1A03D..0x1A04C` 把 pointer 指向的32 bytes
  完整寫到 plaintext `+0x30A3`；CONTINUE `0x10319..0x10328` 對稱
  讀回。Capstone 5.0.3 以相同 EXE 雜湊獨立覆核。
- `0x190F1..0x190FC` 依 event index 測 table byte，
  `0x19246..0x1924B` 成功時寫1；既有 runtime 另有 ch25 index12、
  ch06 index17、AI index16 的 caller-specific raw predicates。因此
  `CurrentSnapshot.Raw30A3` 已更名為 `NativeEventState[32]`，與
  `battle.State.NativeEventState` 對齊但不替其他 index 命名。
- CONTINUE 仍不接 production；plaintext `0x0000..0x08A2` 已由新章
  FDFIELD 載入來源、對稱 current-save／CONTINUE copy，以及
  `0x1A813/0x13A44/0x10B4E` consumers 閉合為固定容量
  `NativeFieldControl[0x8A3]`，不再稱為泛用 raw battle state。
  **此為當時狀態；較晚的 2026-08-02 交易已閉合 control、runtime unit 與
  timing 的具型別轉寫，未改寫 live turn/event 的 chapter0 排程也已有嚴格
  pending-roster consumer。** 現行剩餘主缺口是 handler-mutated turn slots／
  group formulas 的通用 pending-group binding，與 `battle.State` 到正式
  `Game`／controller handoff 的嚴格一致性。
  直接證據保存於 `docs/data/fd2_current_event_state_ida.txt`。
  控制映像證據保存於 `docs/data/fd2_current_field_control_ida.txt`。
- 2026-07-30 `0x10652` 勘誤：合法 IDA Pro 9.4 將函式固定為
  `0x10652..0x1088d`，直接 callers 只有 `0x101d7/0x108a6/0x24a9a`；
  Docker Capstone 獨立覆核。它先釋放 `[0x53aff]/[0x53b03]`，只對 raw
  chapter `9/17/21–25/27–29` 準備特定 FDOTHER 輔助圖形，不是完整章節
  背景 loader。exporter 與 ch22 post editable IR 已由 `load_ch_bg`
  改為 `prepare_chapter_aux_graphics`，compiler regression 固定無
  runtime lowering 時 fail-closed。全量重生曾將 ch14/ch16/ch25 已人工
  閉合的 structured branches 降回暫存器形式，已撤回該批意外差異；
  generator 補齊前不可全量覆寫 canonical handler assets。直接證據：
  `docs/data/fd2_chapter_aux_graphics_10652_ida.txt`。
- 同輪 exporter consistency：目前 `PRIM[0x1088D]` 與 canonical
  ch29 post 都輸出完整 `loadch`；刪除 compiler 對錯誤舊名
  `load_ch_text` 的相容降階。新 regression 要求舊名即使提供完整 binding
  也必須失敗，避免文字-only 斷言再度混入可執行路徑。
- 2026-07-30 CONTINUE selector／preflight：合法 IDA Pro 9.4 重讀
  `0x10010/0x11019`，Capstone 獨立覆核。`0x102f3..0x10316` 先複製
  current runtime records，`0x1035c` 清 cache count，
  `0x1036a..0x1039c` 再依 runtime list 順序取每筆 `+7` 呼叫
  `0x11019`，並覆寫 `+2`。故撤回把新章 persistent→FDFIELD construction
  order 套到 CONTINUE，以及要求存檔舊 `+2` 等於 slot 的錯誤模型。
  新 `BuildContinueRuntimeInput` 原子驗證 resource context、counts、
  FDFIELD 80-unit capacity、13×8 view identity、active raw presentation
  與 first-seen slots；輸出深複製。後續 IDA 9.4 data xrefs 與 Capstone
  又固定標題 caller：`0x10483` 設 opening range mode `0`，
  `0x1060C` 在返回 main 的共享控制器 `0x117E7` 前設 interactive mode `1`；資料映像
  gate B／anchor seed 均為 `1`，而 anchor 只依 restored visible cursor
  的 `<3`／`>9`、Y `>5` 分支推進。`ContinueMapPresentation` 已保存
  這些值；戰場內 `0x1A251` caller 不在此結論範圍。**當時**明列 map timing、
  runtime-unit projection、future-group constructor、battle driver 四個
  unresolved owners；較晚的 2026-08-02 證據已將它們縮為 pending-group
  binding 與 `battle_controller_handoff` 兩項。
  `ReadyForContinue()` 目前必為 false，不接 production。證據：
  `docs/data/fd2_continue_selector_rebuild_ida.txt`、
  `docs/data/fd2_continue_map_presentation_ida.txt`。
- 2026-07-30 CONTINUE map timing seed：IDA data xrefs 與 Capstone raw
  data 證實 `[0x53C07]/[0x53C0B]/[0x53C1F]/[0x539F4]` 及
  `[0x53A40]/[0x53A00]/[0x53A04]/[0x53A08]` 資料初值全零，
  `[0x51A93]=-1`。唯 sprite last tick `[0x53C0F]` 由 main
  `0x25D83..0x25D8B` 在標題入口擷取 signed BIOS low word；
  它不在 FD2.SAV。`ContinueRuntimeContext.TitleTimerTick` 現明示此
  外部狀態，`ContinueMapTimingSeed` 保存首次 redraw 前完整種子。
  但 `0x10494`／`0x105ED` 的 redraw 及中間 delay 仍須 runtime clock
  逐點推進，故 map timing owner 尚未解除。證據：
  `docs/data/fd2_continue_map_timing_seed_ida.txt`。
- 同輪 FDFIELD control typed boundary：`ContinueFieldControlView` 現依
  `[0x53A55]` 固定 layout 唯讀拆出 raw header、16×3 turn events、
  16×2 field events、16×3 chest controls 及 count-delimited 26-byte
  unit rows；原始 `0x8A3` 仍完整深複製。勘誤：這不是 `[0x53A51]`
  composition grid，CONTINUE 另載 resource `3N` 並以 `0x4DBFC`
  初始化 live cell byte `+3`。typed decoder 已完成，但
  `battle.State` atomic adapter 當時尚未完成；後續 field boundary
  consumer 已補，但 runtime unit projection 與 future group constructor
  仍維持 fail-closed。證據：`docs/data/fd2_current_field_control_ida.txt`。
- 後續 unit-count 邊界覆核：合法 IDA Pro 9.4 的
  `0x10BC7..0x10BF6` 與 Docker Capstone 都顯示
  `cmp ebx,[0x53BE3]; jge` 發生在 row load 前，故 control raw `+2`
  是排他的有效筆數。使用者 checksum-valid current snapshot 的 chapter0
  控制前 937 bytes 與 FDFIELD resource1 全同，資源有 31 列空間但
  `+2=30`；typed view 因而只輸出 30 列，第31列及固定容量尾端只保留
  raw。已補 `count=30` 邊界回歸，未因檔案長度猜測額外 live unit。
- 後續 live control 生命週期：IDA 完整 data xrefs 與 Capstone 覆核
  `0x19357` 的 chest value writer、`0x34AB4/0x34AC5` 與多個 chapter
  handler 的 turn-event writer。故 `[0x53A55]` 不是載入後不變的
  FDFIELD resource 副本；current snapshot 的 `0x8A3` 映像是唯一 live
  來源，原始資源／map JSON 不得覆寫。control rows 只用於後續
  `0x10B4E→0x10C50` group append，現有單位必須採 saved runtime
  record order。doc26 已撤回把 party 與 FDFIELD constructor 拼成單一路徑
  及錯套 `+0/+1/+2/+6` 的舊表。證據：
  `docs/data/fd2_current_field_control_mutations_ida.txt`。
- **當時狀態，已由 2026-08-02 的 runtime-unit、map-timing 與
  future-group transaction 紀錄取代**。同輪 live field boundary consumer：新增
  `campaign.MaterializeNativeContinueFieldBoundary`。它會從公開 input
  重建 snapshot、重跑完整 preflight 並逐欄比對，不能只靠可沿用的 marker
  接受建構後竄改；再要求 asset raw chapter、dimensions 與 field-event
  topology 全相符，才一次安裝 exact control、
  turn/field/chest/future-unit rows、event state、raw round、
  camera/cursor、HUD 與 opening range mode0；拒絕前不改 State，輸出
  深複製。它不動 saved runtime Units、timing、interactive mode1 或當時尚未
  辨識的 battle-controller handoff。原本籠統的
  `ContinueOwnerFieldRuntimeBridge` 當時改拆成
  `ContinueOwnerRuntimeUnitProjection` 與
  `ContinueOwnerFutureGroupConstructor`；這些是歷史 owner 名稱，後續嚴格
  consumer 完成後已撤出未解清單。exact runtime records／rebuilt
  slots 已深複製進 State，但尚未投影成 `battle.Unit`。聚焦
  fdsave/battle/campaign Docker regression 通過。
## 2026-07-29：讀檔空槽 production／E2 閉合

- 依 IDA Pro 第一順位規則重查 `0x30550/0x30437`。`0x25F48..0x25F5D`
  證實 LOAD 分支將 FDOTHER #13 載入 `[0x54147]`；舊的 FDOTHER #5
  對話框類推不適用。`0x305CF..0x305E8` 使用 #13 entry16（310×86）
  置於 indexed `(5,112)`。
- `0x30437` 四列固定在 `y=119+19*row`：FDTXT `0x225`＋FFFA 槽號於
  `x=10`；empty `0x202` 於 `x=88`；有效槽使用
  `0x202+chapter`／`0x226+chapter`。selected foreground `0xC9`，
  normal `0xCD`，shadow `0x4C`。
- 新增 `campaign.ComposeNativeLoadSlotsFrame` 與 production
  `drawNativeLoadSlots`。使用 FDOTHER #13、FDTXT #0、FDOTHER #4
  點陣字型與 FDOTHER #0 palette；缺資源或不可映射 JSON metadata 時
  失敗即關閉。
- `TestComposeNativeLoadSlotsFrameMatchesEmptyDOSBoxOracle` 對
  `load-empty-original-dosbox.png` 做 320×200 全幀 RGB 比較並通過；
  `load-empty-remake.png` 是目前 source 以一次性 Docker/Xvfb 建置與
  strict screenshot state 產生的 2× production artifact。
- 有效 native FD2.SAV restore、剩餘 metadata、roster ABI、刪除／覆寫
  尚未閉合，UI-VIS-LOAD 維持部分完成。詳細直接指令見
  [`fd2_load_slots_ui_ida.txt`](../data/fd2_load_slots_ui_ida.txt)。
- 原始 FD2.SAV 四槽皆空。另在 `/tmp/fd2-load-valid` 複本只把 slot0
  已證實 metadata 改為 chapter1／count1／currency1000，重新計算 native
  checksum／rolling XOR；分段 Escape 後進 title→LOAD，但未確認槽位。
  原版顯示「第二章／羅德鎮」，與 chapter1 JSON production 逐像素精確
  2× 比較為零差異。新增 original/remake 圖與 regression；此項明確標成
  修改路徑，不可當成成功 restore 或正常玩家存檔。
- 合法 IDA Pro 9.4 重核 `0x25ebb..0x26152`，Capstone 獨立核對
  `0x25f81..0x26120`：選槽確認後 `0x2602c..0x26056` 先複製固定
  `0xa00` roster，`0x2605e..0x26098` 再載 metadata `+0..+9`；
  chapter `0xff` 回到 selector。非空槽關框後先呼 `0x2cad7`，只有
  gate 回 0 才呼 chapter pre-handler。這條四槽路徑不呼 `0x10010`，
  後者仍是第三個標題選項的目前戰鬥續戰入口。
- 新增 `fdsave.InspectChapterSlot`：保留 32×`0x50` 完整 raw records、
  全部 `0x28` metadata 與已證實 header，空槽／count > 32 失敗即關閉；
  不把 opaque record 猜成 `battle.Unit`。專注 regression 通過；一般玩家
  native restore 與 roster 正規化仍未完成。證據見
  [`fd2_native_chapter_slot_restore_ida.txt`](../data/fd2_native_chapter_slot_restore_ida.txt)。
- production title 新增唯讀 `FD2_NATIVE_SAVE` 來源：先驗 checksum，再用
  已證實 metadata 呈現四槽；tamper regression 會拒絕整份檔案。確認空槽
  會留在 selector；確認有效 native 槽也會明示 roster restore 尚未完成，
  不會錯誤轉入自有 JSON loader。這是 metadata→UI／gate 垂直切片，不是
  successful native restore。
- 合法 IDA Pro 9.4 再重核 persistent roster：`0x112A5` 是完整 record
  constructor，`0x1145A` 由 base／equipped items 重算 `+48..+4E`，
  `0x17EEF→0x17FC0` 直接顯示 level/EXP/MV/HP/MP/AP/DP/DX/HIT/EV 與
  identity/race/class。新增 `fdsave.PersistentRecord.View` 保存已證實
  offsets、signed words 與全部 raw key 分界；尚不解析名稱、class label、
  sprite 或 normalized party。證據見
  [`fd2_persistent_roster_ida.txt`](../data/fd2_persistent_roster_ida.txt)。

## 2026-08-01：`0x53AFA` 逐呼叫配置旗標接入 handler

- official IDA Pro 9.4 完整資料交叉參照固定 `[0x53AFA]` 只有
  `0x10CB7` 一個 reader，以及11組成對的 literal-one／literal-zero
  writer；Docker Capstone 從每個正確指令邊界獨立重生
  `set1 → push group → call 0x10B4E → reset0`。三組是章節 handler，
  八組是 global event call；`0x32999` wrapper 及 caller 無 writer，
  故其內部 call 明確讀零。版本與完整表見
  [`fd2_future_group_raw_gate_ida.txt`](../data/fd2_future_group_raw_gate_ida.txt)。
- `dump_chapter_beats.py`、`export_handler_scripts.py` 與
  `extract_event_id_groups.py` 現按 source call-site 產生
  `raw_placement_gate`，不從 group/camp 推論。版本化資料共25筆 handler
  spawn（3筆為1）及34筆 global event call（8筆為1）；generator 測試與
  全表 assertion 已在 `fd2-cap-local` 通過。
- handler compiler 對原版 `spawn`／`spawn_intro` 要求 explicit byte；缺失
  即 compile issue。runtime Beat 將欄位送入
  `AppendGroupWithNativePlacement`，完整 batch 先預演 position row、兩次
  occupancy writer 與逐列新占用，錯誤時不改 roster／units；成功才依原
  FDFIELD row order append。campaign/battle Go regression 與 `cmd/fd2`
  Xvfb regression 通過。
- **當時狀態，已由 2026-08-02 的 future-group 建構交易紀錄取代**。
  此提交當時只接 handler path；global turn-event scenario action 在下方
  「global turn-event 配置來源接入 production action」批次才補上
  source/via/gate。在此時間點，完整 `0x10C50` table/inventory projection
  與 `0x1B750` recompute 尚未完成；下一個 2026-08-02 建構交易紀錄已將
  此缺口閉合，正式 CONTINUE 則仍維持失敗即關閉。
- 發現既有 `docs/data/event_id_groups.json` 是 root-owned；只針對該檔
  替換後修回目前 UID/GID。專案與 `~/.codex/AGENTS.md` 已新增規則：可寫
  容器須明示目前 UID/GID、寫前檢查目標 ownership，歷史 root-owned 只能
  狹義修復，禁止對 repository／HOME 做遞迴 `chown`。

## 2026-08-01：global turn-event 配置來源接入 production action

- `gen_campaign.py` 現把45筆可解析 schedule 合併為46個 editable
  `spawn_group` action；ch01/event0 的 group 3、7 因人工演出拆成兩個 action，
  仍保留同一 `native_event_id=0` 與原 call order。46/46 action 都有 event id，
  46/46 call 都有 source/via/`raw_placement_gate`。排程可達的 gate=1 只有
  ch01/e3、ch02/e6、ch05/e15、ch07/e25、ch12/e35、ch13/e7；event37 未在
  turn schedule，沒有猜測性塞入關卡。
- 人工 scenario 合併遇同一回合多個 spawn schedule 或既有 action 已綁不同
  event id 時失敗即關閉。新增 Python 回歸固定 ch01 split order、歧義拒絕與
  禁止改綁；Go loader 要求完整 event provenance，並固定46 actions／六筆
  gate=1 集合。
- `cmd/fd2` battle-event runner 改用 `ExecuteActionChecked`。具
  `runtime_append_groups` 與完整 roster 時逐 call 執行
  `AppendGroupWithNativePlacement`；缺 roster 直接回報錯誤，不靜默退回。
  ch02 turn3/event6 的版本化 action 已驗證六名 group3 友軍逐筆採 gate=1
  原始位置列（position row）；錯誤時不呼叫回合完成回呼（continuation），
  不會漏生後仍前進。尚未遷移情境維持明示的正規化相容路徑
  （normalized compatibility path）；
  當時 `spawn_group_with_intro` 的 acting/reveal/present、完整 table/inventory
  projection 與 equipment recompute 仍未接；下節已進一步勘誤 wrapper 與
  caller 的責任邊界，不能宣稱 constructor 完成。
- 再次實測 `gen_campaign.py` 預設總表輸出會把權威 `campaign_full.json`
  由299節點降回293並遺失已整合欄位。本輪已精準恢復權威檔；新增
  `--scenarios-only`，回歸以相同 git blob hash 證明重生逐章資料不改總表。
- 依使用者指定的另一專案經驗，專案與 `~/.codex/AGENTS.md` 新增 IDA
  非破壞性註記規則：原始名稱／位址／偏移必須保留，語意只能附加，且每項
  語意都要帶已證實／強推論／假說／未知與來源；工具內改名不能取代 raw
  bytes、xref 及讀寫端證據。

## 2026-08-01：`0x32999` wrapper 與呼叫端 acting 責任閉合

- 合法 IDA Pro 9.4 固定 `sub_32999` 範圍 `0x32999..0x32D18`，直接 caller
  只有 `0x3289B`、`0x328BB`、`0x342CE`、`0x34336`。本體沒有
  `0x1366A`；四個 caller 返回後分別於 `0x328A5/0x328C5/0x342E7/0x3434F`
  執行 ACTING(1/2/3/4)。先前把 acting 寫成 wrapper 內部行為的斷言已撤回。
- wrapper 載入 FDOTHER #95/#9、保存 `0x25680` 位元組工作畫面與舊單位數，
  呼叫 `0x10B4E(group)` 後固定走 #9 的12個 `LMI1` 項目。每個 pass 只合成
  新增且在攝影機視窗內的單位，再呈現312×192區域；第6、7、8次另有不同
  背景／前景重建順序。這是12次索引合成／呈現，不等於等待12個重製畫面。
- `event_id_groups.json`、scenario generator 與 Go `NativeSpawnCall` 現保存
  global event1/2 的 `following_acting` resource/source。loader 要求 intro
  呼叫具完整後續 provenance，普通 spawn 則拒絕夾帶它。
- **當時狀態，已由下方 2026-08-01 後續紀錄取代**：正式 battle action 與
  原版來源的 handler `spawn_intro` 當時都在 roster 變更前失敗即關閉；舊的
  12-tick 路徑只可留給無原版 provenance 的擴充內容相容模式。
- 直接證據見 [`fd2_spawn_intro_32999_ida.md`](../data/fd2_spawn_intro_32999_ida.md)
  與 [`fd2_spawn_intro_32999_capstone.txt`](../data/fd2_spawn_intro_32999_capstone.txt)。

## 2026-08-01：`0x32999` handler 的12次索引呈現接入正式路徑

- `fdother.NativeSpawnIntroSchedule` 保存原始 pass 0..11、FDOTHER #9 項目索引、
  pass1 的原始資源 #95，以及 pass6/7/8 的重建模式與 `-8/-5` 像素列位移；
  原始 pass、資源號與指標位移皆保留，語意標籤只作附加。
- `indexedmap.ComposeNativeSpawnIntroPass` 以 `0x25680`、stride `0x1C8` 的工作
  畫面運作，先呈現312×192視窗，再按原順序更新下一張 snapshot。舊／新增
  單位以 constructor 前的 unit count 分界；列位移只作用於 framebuffer base，
  不改世界座標。任何驗證失敗都不修改 caller buffers。
- handler runtime 先在私有 FDFIELD／roster 副本執行 placement 與12個 pass
  預檢，全部成功後才發布新增群組。每個 pass 必須經 `Draw` 確認才前進；完成
  後才執行 caller 的獨立 ACTING beat。缺 FDOTHER #9、#95、selector、原始位置
  或 framebuffer 幾何時，在 roster 變更前失敗即關閉。
- 真實 ch00 handler 回歸以使用者合法 `FDOTHER.DAT` 驗證兩次增援各呈現12幀，
  並繼續到 `battle_ch01`、戰後同步、`town_ch02` 與 `preparation_ch02`。這是
  重製端決定性路徑證據，不提升成與未修改 DOSBox 同狀態逐像素 E2。
- **當時狀態，已由下方 2026-08-02 紀錄取代**：低階
  `battle.Scenario.ExecuteActionChecked` 沒有畫面／音訊與 following acting
  擁有者，因此 event1/2 當時仍失敗即關閉。

## 2026-08-02：ch01 global event1/2 接通12次呈現與 ACTING(3/4)

- 非同步 owner 固定在 `Game.advanceBattleEvent`：它在呼叫低階 action executor
  前，只辨識 `0x342CE→0x342E7/ACTING(3)` 與
  `0x34336→0x3434F/ACTING(4)` 兩個已證實 caller。低階
  `ExecuteActionChecked` 仍拒絕 intro，避免沒有 `Draw`／音訊／continuation 的
  同步呼叫繞過畫面責任。
- `Scenario.native_acting_resources` 明示可編輯的
  `assets/cutscenes/acting/map32.json`；runtime 在任何 constructor commit 前驗證
  resource3／4 的 frame timing、pose、slot frontier 與重複 slot。原始
  source address／resource id 保留，語意只是附加。
- battle candidate 由目前 State 的 units／roster、selector cache、terrain／unit
  animation phase 與原始 composition bytes 私有複製；因此 turn4／5 不會被錯誤
  重設成章節開場動畫相位。group4／5 placement、12個 pass 與 following acting
  全部預檢後，才一次發布新增 units、剩餘 roster 與 selector cache。
- 真實資產回歸先跑 ch01 turn3 建立14槽 frontier；turn4 event1 發布 slots14–17，
  各12次 Draw 後 ACTING(3) 才移動；turn5 event2 發布 slots18–22，各12次 Draw
  後 ACTING(4) 才移動，speaker71 對話最後才出現。缺 acting resource 的反例
  驗證 units／roster／selector cache／turn continuation 全不變。
- 這是重製端 E1 決定性機制證據；尚未取得同 camera／roster／pass 的 DOSBox
  event1/2 逐幀比較，因此不提升為 E2。本節記錄的當時狀態是
  `0x10C50` table／inventory projection 與 `0x1B750` equipment recompute
  尚未閉合；緊接的下一節已以合法 IDA 與 Capstone 將該交易閉合。

## 2026-08-02：`0x10C50→0x1B750` future-group 建構交易閉合

- 合法 IDA Pro 9.4 固定 `sub_1B750` 範圍 `0x1B750..0x1B83D` 與12個直接
  caller；Capstone 再核原始指令。它讀即時名冊 `[0x53A45]+index*0x50`，以
  signed `+0x37/+0x39/+0x3E` 為基礎，掃八格 `0x40` 裝備，讀 item row
  `+1/+5/+3/+7` 後寫 `+0x48/+0x4A/+0x4C/+0x4E`。
- `+0x22/+0x23` 非零時，AP／DP 乘 0x5018D 的 binary64 1.15，再由
  `0x377A4` 設 x87 朝零捨入；因此 20→22、100→114。`+0x24` 非零時 HIT／EV
  各加15。這證明 `0x1B750` 不等同於沒有 transient modifier 的 persistent
  `0x1145A`；只有 `0x10C50` 先清 `+0x22..+0x27` 的 caller 可化約為共同核心。
- 新增 `ApplyNativeRuntimeEquipmentRecalc` 的 exact ratio／朝零與 atomic bounds
  regression；`campaign.RecomputeEquipment` 降回明確的正規化投影，不再冒充
  `0x1B750/0x1145A` byte-equivalent transaction。
- `MaterializeNativeFutureConstructor` 把 table base、原始八格 inventory 與
  effective-stat 重算套在私有候選。`AppendGroupWithNativePlacement` 只有在所有
  candidate 的 constructor、位置、selector 與 presentation 預檢成功後，才發布
  units／roster／cache；失敗不修改來源名冊。ch01/ch02/ch03 與 battle-event／beat
  runner 回歸均改用完整 item rows 與 selector state。
- 這關閉 future-group append 的具型別交易，不宣稱完整 0x50-byte 逐位元組一致，
  也不把其他 `0x1B750` caller、transient expiry、轉職、商店、戰後 persistent
  sync 或 DOSBox E2 自動標成完成。直接證據見
  [`fd2_runtime_equipment_recalc_1b750_ida.txt`](../data/fd2_runtime_equipment_recalc_1b750_ida.txt)。

## 2026-08-02：CONTINUE 返回控制器與地圖計時勘誤

- 合法 IDA Pro 9.4 以唯讀 IDC 稽核固定 `sub_4E031` 僅有
  `0x41A→0x41C` word copy；它沒有單位、回合或事件 dispatch。把兩個 word
  視為 BIOS head/tail、進而解釋為丟棄待處理輸入，最多是外部 ABI 的強推論；
  直接證據只提升到 word copy，重製端不可把它當成已證實按鍵 consume。
  `0x10010` 經 `0x1061B→0x22BBE` epilogue 返回後，main 才於
  `0x25DCE` 呼叫並循環重入 `0x117E7` 共享戰鬥控制器。Capstone 以相同
  FD2.EXE（357074 bytes；MD5 `b97caf2239a27a896069d03549d96e1e`；
  SHA-256 `222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f`）
  獨立重生三段直接指令。證據與可重跑腳本為
  [`fd2_continue_controller_117e7_ida.txt`](../data/fd2_continue_controller_117e7_ida.txt)
  及 `tools/ida_probe_continue.idc`。
- 標題第三選項不再錯讀重製 JSON slot 0；它現在明確產生 native CONTINUE
  action，完整原生交易接通前留在標題並失敗即關閉。四槽 LOAD 的
  `FD2_NATIVE_SAVE` 路徑保持獨立，不受此勘誤改寫。
- `MaterializeNativeContinueMapTiming` 現要求 field boundary 與 runtime-unit
  projection 都已成功，才原子安裝原版 timing seed。地圖週期不再於每次
  `Game.Update` 無條件推進，而是由一次成功的 `0x11CAC` 等價 compositor
  transaction 取樣並同時發布 timing／pixels；失敗不改 timing、clock 或 VGA。
- `BuildContinueRuntimeInput` 的未解 owner 已縮為兩項：chapter asset 的
  `pending_group_binding`，以及已知 `0x117E7` 對應的
  `battle_controller_handoff`。runtime-unit、map-timing 與 future-group
  constructor 不再冒充未解 owner；正式 CONTINUE 仍未接 production，不能
  宣稱一般玩家原生續戰或 E2 已完成。
- 同輪全庫斷言稽核同步修正 SDD、介面證據矩陣、工作清單、控制流與 data
  evidence；歷史段落保留時間序列，但明示已被較晚結論取代。Docker 驗證結果：
  Python 31 項測試通過，Xvfb 下 `go test -count=1 ./...` 全部通過；原版
  `0x4E031`、`0x105ED..0x1061B`、`0x25DBD..0x25DCE` 範圍亦以 Capstone
  實際重生核對。

## 2026-08-02：CONTINUE 目前回合與 chapter0 pending roster

- 合法 IDA Pro 9.4 固定 `0x117E7` 的系統選單分支：`0x16F55` 在 raw
  selector0 呼叫會寫 FD2.SAV 的 `0x19DF7` 後直接返回，沒有先執行
  `0x1A30B`。玩家行動後的 `0x13565` 只有在無合格 raw camp2 列時才呼叫
  `0x1A30B`；它以 selector1/0 掃描目前回合，接著才在 `0x1A5B9` 增加
  `[0x53BEF]`，後段再掃 selector2。Capstone 以固定雜湊原版獨立重生。
- 因此 current snapshot 的 saved turn 不能合併判定：selector1/0 尚未被
  下一次 phase 收束掃描，selector2 已在上一輪增加回合後的尾端掃描。
  pending roster 必須採 `turn > saved_turn`，或 saved turn 且 selector0/1；
  不能無條件使用 `>=`。新增
  [`fd2_continue_pending_schedule_ida.txt`](../data/fd2_continue_pending_schedule_ida.txt)
  保存輸入雜湊、IDA 版本、原始位址、直接指令與推論等級。
- `MaterializeNativeContinuePendingGroups` 只在 chapter/map/dimensions、
  FDFIELD row count、native append provenance 及 live `(turn,event_id)` 全部
  精確相符時，深複製目前與未來事件引用的 FDFIELD rows、PendingGroups 與
  item row prefix。map25/event61 另要求 editable rule、地圖 slot、資產
  selector1 與存檔 live event 相符，再由 once-state12 判斷 group1 是否待命；
  失敗不修改 current Units、selector、live control 或 roster。
- 未改寫 live schedule 的窄切片以 checksum-valid 原版 chapter0 FD2.SAV
  於唯讀 Docker 完整跑過 field、
  runtime unit 與 pending-group adapters：12筆 current runtime 保持原順序，
  只綁 groups3..7 共15筆 future rows；groups1/2 已出場，10/11 未排程。
  同時刪除測試內 map0=31×24 的錯誤註解，版本化 map0 資產為24×24。
- 全章機械盤點顯示46個 `spawn_group` action，但只有 ch01/ch02/ch03/ch26
  已宣告 native runtime append；event27/54/57 使用目前回合作 group，
  event47/49 另有 formula，且多個 handler 會改寫 live turn byte。這些仍未
  資料化，故 `pending_group_binding` owner 不移除；正式 CONTINUE 與 E2
  仍維持失敗即關閉。

## 2026-08-02：map26 event62 休眠列與 staging helper 勘誤

- 合法 IDA Pro 9.4 與 Docker Capstone 直接指令共同固定：event62
  `0x35898..0x358C6` 由 `0x358BA` 寫 live row0 turn=`[0x53BEF]+1`，並以
  state byte17 防止重複；map26 原始 row0 是 `ff 3f 00`。舊證據表把
  `inc dl` 的 `0x358B5` 誤列成 store，同批其他 writer 也多數停在 pointer
  load／算術指令，已逐一更正為實際 `mov [memory],...` 位址。
- 原 parser 只輸出 `turn != 0xff`，使 handler 後續會啟用的 row0/event63 與
  row1/event65 消失。新增全33圖×16列的
  `remake/assets/maps/native_turn_event_controls.json`；策展過的
  `docs/data/turn_events.json` 保持獨立，禁止用純 raw exporter 覆寫。
- IDA Pro 9.4 把共用戰鬥初始化固定在 `sub_205DA`，29個章節開場呼叫者
  在 `0x2066E` 將 `[0x53BEF]` 寫成1；Docker Capstone 獨立確認相同指令。
  完整回合目錄現同時鎖定 FD2.EXE 雜湊、寫入端與初始值，新戰鬥載入值1；
  CONTINUE 快照仍優先恢復即時值，缺出處的手工狀態維持0。載入器並鎖定
  目錄 SHA-256 與 map 0–32 唯一集合；控制列、寫入端或地圖身分遭竄改即拒絕。
- `NativeTurnActivation` 與 `ApplyNativeFieldTurnActivationEvent` 現只接受
  map26 selector0、event62、once-state17、slot0/event63/raw-camp0/delta1；
  完整列或 CONTINUE raw image 不一致時不作部分 mutation。Go battle/campaign
  與 Python parser/sync 回歸均涵蓋休眠列、原子失敗及重複觸發。
- 正式 `Game.stepBattleWalk` 已在原版限定的向左一步第七拍提交後 dispatch
  event62；前六拍 row/state 不動，第七拍才以 native round+1 啟用 row0。
  其他方向不泛化，malformed rule 走 `loadErr` 停止。Xvfb Docker 測試通過，
  但尚無同狀態 DOSBox 玩家實驗，因此只列 E1。
- 更正 `0x35822` 的來源參數：export 的 raw args 是 `PUSH`
  `(group,y,x)`，callee 才分派 pan `(x,y)` 與 spawn `group`。因此 ch27
  `[6,16,0]` 是 group6@(0,16)，ch28 `[8,19,9]` 是 group8@(9,19)；舊
  `(x,y,group)` 文件、compiler 與整合測試已同步修正。
- `extract_event_id_groups.py` 現先驗證 FD2.EXE size／MD5／SHA-256，再辨識
  direct call 與 event63 的 `0x35318` 共用 tail。版本化 event63 不再是
  `spawns: []`，而以 `staging_helper_0x35822` 保存 group1@(3,27) 與
  group2@(15,27)；這個 via 未被一般 spawn consumer 接受，因此尚未閉合畫面
  owner 時仍保持失敗即關閉。同輪也機械閉合 event64／66／68／70／72 的
  group、座標與 call-site；只列為固定原始 E0 資料，不提升觸發條件或 E2。
- 2026-08-02 event63 dynamic runner：IDA Pro 9.4 與 Docker Capstone 5.0.3
  共同證實 `0x1A554→sub_1A813(0)` 位於 `0x1A58F→0x1D8BA` 敵軍 AI
  前，`0x1A5B9` 才增加 native round；raw camp0 不再錯稱敵軍回合結束後。
  ch27 新增可編輯 `native_turn_events`，group1／2 從 initial groups 移到
  pending roster。正式 `Game.endTurn` 先匹配 live row，再依兩個 `0x35822`
  呼叫執行 pan→native append→300ms→delta255 全白→200ms→delta0 restore→
  redraw；全部完成才啟動 AI。兩批 constructor 先在 private state 預演，
  第二批錯誤時第一批不會部分發布；未知同回合 row 亦在 AI 前失敗即關閉。
- event63 目前是重製端 E1，不是 E2（本段 HUD 缺口已由檔末較晚勘誤取代）：
  下方較晚勘誤已閉合並接線 ch27 精確 native view 與 selector0；當時尚未閉合
  persistent HUD gate A／anchor，所以不建立 `native_map_hud`。缺完整 HUD 時
  indexed DAC 失敗即關閉，
  一般路徑保留既有 RGB 戰場的數學等價全白覆蓋。下一個切片是
  `MAP26-EVENT63-E2-PLAYER-PATH`：未修改 DOSBox 的 event62→跨回合→event63
  同狀態逐幀比較，以及 CONTINUE 邊界。

## 2026-08-02：ch27 戰前 view 與 HUD 持續邊界

- 綁定參考 `FD2.EXE` 雜湊的 IDA Pro 9.4 主判讀與 Docker Capstone 覆核，
  已閉合 ch26_pre 返回 battle_ch27 時的 camera `(9,49)`、absolute cursor
  `(14,54)`、visible cursor `(5,5)` 與 selector 0。原始位址、acting 位元組
  與推論等級保存在
  [`fd2_ch27_pre_view_ida.txt`](../data/fd2_ch27_pre_view_ida.txt)。
- `battle_ch27.native_map_view` 已資料化並由正式 runtime 原子載入；schema
  現允許具直接證據的 view 在 HUD 未閉合時獨立存在，節點入口只接受已證實的
  selector 0／1。HUD 仍不可脫離 view 單獨宣告。
- 不建立 ch27 `native_map_hud`（歷史狀態；已由檔末「HUD 持續擁有者閉合」
  取代）：main 返回後 gate B 已知為1，但 gate A 可由
  選項與存檔延續，anchor 在本處理器 visible row 最大為5時也保留既值。兩者
  尚缺跨節點／存檔擁有者，猜成1會製造錯誤原版斷言。
- 此垂直切片是 E1，不是 E2；event63 的 indexed DAC 仍要求完整 HUD，缺值時
  走既有 RGB 戰場的數學等價全白覆蓋。下一門檻仍是未修改 DOSBox 同狀態逐幀
  比較、persistent HUD state owner 與 CONTINUE 邊界。

## 2026-08-02：HUD 持續擁有者閉合與 JOIN raw record 勘誤

- IDA Pro 9.4 主判讀與 Docker Capstone 覆核已把 HUD 三個來源分開：gate A
  `0x51AAB` 由 custom save 與 native chapter slot metadata `+6` 保存／恢復；
  anchor `0x51A0C` 是程序內持續狀態，只由 `0x1ACF3` 的兩條已證實邊界分支
  改寫；gate B `0x51AAC` 是 controller-owned 暫態值，戰鬥入口只採已證實的1，
  不寫入上述持續欄位。位址、雜湊與推論分級見
  [`fd2_hud_persistence_ida.txt`](../data/fd2_hud_persistence_ida.txt)。
- `NativeMapHUDPersistentState` 已成為 `Game` 擁有者；custom save 與 native
  chapter restore 保存 gate A，anchor 跨節點保留，`battle_ch01`、ch26、ch27
  以 `native_map_hud_inherited` 原子物化完整 runtime HUD。固定 `(1,1,1)` 只
  保留在明確 snapshot fixture，不再當章節常數。
- persistent roster overlay 現同時保存 raw `+0x42` 與 `+0x46`，不由正規化
  HP／MP 反推；並修正 `MapSelectorSlot` 被跨戰複製的錯誤。該 slot 是
  `0x11019` 每場戰鬥重建的 `unit+2` cache result，持續層只能保存其 raw `+7`
  key。
- event63 production regression 現使用明確帶有 persistent raw `+0x42` 的
  凱麗 fixture，能走 inherited HUD 與 indexed DAC；它不以 ch27 近似 `hp=90`
  冒充原版欄位。`JOIN 0x112A5` 的精確表格公式另證實凱麗 id12 fresh record
  `+0x42=151`。此值只證明 fresh JOIN，不代表 ch27 一般玩家時點的 current
  raw record；完整公式與限制見
  [`fd2_join_constructor_word42_ida.txt`](../data/fd2_join_constructor_word42_ida.txt)。
- 本切片仍是 E1。下一門檻是把 JOIN default/growth table 轉為具型別 asset，
  走正常 JOIN→LOADCH→sync/save 建立 raw record，並取得未修改 DOSBox
  event62→event63／CONTINUE 的同狀態逐幀 oracle；完成前不得提升為 E2。

## 2026-08-02：JOIN `0x112A5` raw record 正式接線

- 上段所列具型別資產缺口已完成：`tools/sync_native_join_constructor.py`
  從 32×0x18 character defaults、32×0x0B growth 與 reference manifest
  產生 `native_join_constructor.json`，保留每列 file offset、raw bytes、
  FD2.EXE size／MD5／SHA-256 與「已證實」等級。
- `campaign.NativeJoinConstructorTable` 逐列驗證 row order、`0x55BA1+id*0x18`、
  `0x55EA1+id*0x0B`、stride 與 EXE identity；錯誤版本或缺列失敗即關閉。
- 正常 beat JOIN 仍先保存 membership／chronology，第一次 LOADCH 建 persistent
  roster 時才以 scenario unit 提供姓名等顯示 base，再由 raw table 覆寫已閉合的
  identity/key、race/class、level/MV、command mask 與 `+0x42/+0x46`。場內
  `scenario join_party` 使用相同 materializer，不再有兩套首次建檔公式。
- 修正舊 camp gate：永久 JOIN 不依賴角色當下已是 Own；許多招募角色仍是 Ally。
  既有實作因只搜尋 Own 而未在 JOIN 當下建檔，過去由 legacy Fig sync 偶然補回。
  現保留角色唯一性檢查，但不以陣營顏色拒絕 persistent record。
- 凱麗 event63 regression 現由同一 table 得到 fresh `+0x42=151`，不再手填
  `0x123`，亦不採 ch27 approximate `hp=90`。這仍只證 fresh JOIN；ch27 真實
  玩家時點可因升級／轉職而不同。
- `sub_1145A` 裝備重算仍未閉合。本 materializer 故意不覆寫 AP／DP／DX 等
  scenario base，也不宣稱完成整個 0x50-byte record；下一 RE 切片是八物品格
  到 `+0x48/+0x4A/+0x4C/+0x4E` 的直接資料流與物品旗標消費端。

## 2026-08-02：JOIN `0x112A5→0x1145A` 裝備交易接線勘誤

- 上段「`sub_1145A` 尚未閉合」是過時斷言。更早已有
  `battle.ApplyNativeEquipmentRecalc`、215 列 item row cross-check 與 raw
  regression 證實八格 `flag&0x40`、row `+1/+5/+3/+7`、signed 16-bit
  accumulation 及 `+0x48/+0x4A/+0x4C/+0x4E` writers；真正未完成的是 JOIN
  production consumer。
- `NativeJoinConstructorTable.MaterializePersistentUnit` 現以局部 0x50-byte
  transaction record 精確寫入 `0x112A5` 已證實的 inventory、command、
  race/class/level、base AP／DP／DX、HP／MP 欄位，再呼叫 raw `0x1145A`
  helper。缺少任一 equipped item row 時不發布部分角色。
- 凱麗 id12 fresh JOIN 現固定 base AP／DP／DX `80/69/10`，inventory
  `0x3E/0xAC`，derived AP／DP／HIT／EV `100/79/110/15`；`+0x42=151` 不變。
  這些是 fresh JOIN 的 E1 transaction regression，不是 ch27 一般玩家時點
  或 DOSBox 同狀態 E2。
- 同輪修正兩項 raw 邊界：native persistent restore 現保留原始
  `+0x42/+0x46` provenance；商店 equipment recipient 的 current
  `+0x48..+0x4E` 改依已證實 signed word 解碼，負值不再顯示成巨大正數。
- Docker regression 已覆蓋 32 個 JOIN rows、凱麗 exact fixture、缺 item row
  原子失敗、signed shop current stats 與 native save restore；
  `go test ./... -count=1`、JOIN asset `--check` 及 247 個本地 Markdown
  檔案／圖片連結均通過。

## 2026-08-02：玩家第7戰 event25／戰後 JOIN12 垂直切片

- IDA Pro 9.4 找到 map6 state17 的直接 writer `0x34924`：enemy turn10 的
  event25 依序執行 spawn group2、pan `(16,10)`、ACTING30、FDTXT_007
  index2，最後 `0x34996` 才寫 state17=1。原始位址與推論等級見
  [`fd2_ch06_post_event25_ida.txt`](../data/ida/fd2_ch06_post_event25_ida.txt)。
- 後續勘誤：event25 不是「第10回合必定執行」。IDA Pro 9.4 固定 map6
  field-event slot0＝event26／selector0，六個格子座標為 `(9,13)`、
  `(10,14)`、`(11,14)`、`(12,15)`、`(13,15)`、`(14,15)`；
  `0x3499B` 只在觸發單位 raw `+6 != 0` 時呼叫 `0x3419C(9,27,0)`，
  再寫 state16=1。event25 `0x34924` 先要求 state16==1，否則直接返回。
  重製已接既有向左一步第七拍 selector0 owner，並以未踏格反例回歸防止
  第10回合錯誤增援。
- 撤回 2026-07-27「slot43 需要96-slot空白 frontier」的錯誤斷言。40 是
  FDFIELD constructor row 數，不是 active runtime count；正確順序是 party9、
  group1共25（34 slots），event25再 append group2共10（slots34..43）。
  ACTING30 的直接 targets 也正是34..43，slot43為group2最後一筆凱麗。
- `ch07.json` 已改用 runtime-append，完整資料化 event26 gate 與 event25 的
  spawn→pan→ACTING→九句對話→raw state write；不再把 Ally 凱麗隨整批
  覆寫成 Enemy。
- raw ch06 post 已用結構化雙層 CFG 接正確 owner
  `postbattle_ch07_persist`：只有 state17==1且runtime精確44 slots才讀
  slot43 byte5 bit0；active arm執行layout→index4→JOIN12，其他路徑播index5。
  `JOIN 0x112A5` 現在於 beat 當下建立 persistent record，不再延遲到下一次
  LOADCH，故 town8／save 邊界可立即看見凱麗。
- 決定性回歸已覆蓋 event26 踏格／未踏格反例、event25 state commit timing、
  ACTING30、34／44 frontier、
  post雙分支、JOIN12 persistent record與 `town_ch08`。目前仍是E1；未修改
  DOSBox第10回合至戰後城鎮的一般玩家錄影／逐幀比較仍待完成。

## 2026-08-02：玩家第8戰 raw ch07 戰後與初始 roster 勘誤

- 較早把 raw `ch08_post` 稱為「ch08 postbattle closure」的說法已失效：
  玩家戰鬥 N 使用 raw `ch(N-1)_post`，所以玩家第8戰的正確 owner 是
  `ch07_post`／`0x234BB`；raw `ch08_post` 屬玩家第9戰。
- IDA Pro 9.4 與 Docker Capstone 共同固定開場 constructor：`0x1088D`
  只呼叫 `0x10B4E(0)`，raw ch07 pre `0x33219` 也沒有其他 spawn。
  因此正常 runtime 是 party10＋group0十九筆，共29 slots。舊
  `ch08.json initial_groups=[0,1,8,9,10]` 讓沒有 producer 的四組提前出場，
  並破壞 slot28 的 raw `+8==5` 身分，現已撤回為 runtime append＋group0 only。
- event27 `0x349D9` 在回合2..7才依目前 group 追加 groups2..7，每組兩筆；
  所以戰後 handler 合法 frontiers 是29、31、33、35、37、39、41。
  event28 `0x34A0E` 只對 slots10..27 執行 raw `+0x34 &= 0x80`，不是生怪；
  其正式回合接線仍列後續工作。
- `0x234BB` 的 X/Y tables、slot28 special placement、ACTING33／34 與
  FDTXT_008 index3／4 已做 address-keyed binding。`0x23599` 的
  `0x11D40(0,255,64)` 會將全部六位元 DAC 分量夾成0，且呼叫者緊接
  `memset(0xA0000,0,0xFA00)`；因此只對此 call site 實作精確全黑，其他
  `0x11D40` 使用仍失敗即關閉，亦不再誤稱 HP 條。
- 正式 `postbattle_ch08_persist` 現執行 layout→對話／acting→全黑→
  `JOIN5→sync_party→chapter8`，之後保留原有 `town_ch09` 節點。決定性測試
  覆蓋29-slot開場、event27 29→31、錯誤初始 groups 缺席、call-site／參數
  負向拒絕、洛娜 persistent record與進城；目前是E1，不是DOSBox E2。
- 完整非破壞性位址證據見
  [`fd2_ch07_post_ida.txt`](../data/ida/fd2_ch07_post_ida.txt)。

## 2026-08-02：玩家第10戰 raw ch09 戰後與精確 DAC 淡入勘誤

- IDA Pro 9.4 主判讀與 Docker Capstone 覆核固定 `sub_235F9`
  （`0x235F9..0x23790`）。`0x1F882` 是 delta 0→63 共64次
  `0x11D40(0,255,delta)` 淡出；`0x1F525` 是 delta 64→0 共65次淡入，
  兩者每次都等待2ms。較早將後者 lower 成通用36幀 RGBA `fade` 的說法已撤回。
- `0x2362D..0x236E2` 不經 helper 的位置、pose、raw `+5/+0x26` 與 view
  寫入已轉成 address-keyed `direct_record_patch`。執行期先驗證全部 slot、
  值域、原始欄位 provenance、重複 offset 與 view，再一次提交；後段非法值
  或短 frontier 不會留下前段部分修改。raw offset 不從局部清零推導 gameplay
  名稱。
- 第10戰正常 group0 四十二筆，回合5 event32 再 append group1 八筆；永久隊伍
  是否含凱麗 id12，形成60／61兩種戰後 frontier。兩者已有 deterministic
  regression，但仍是 producer 交叉支持的強推論，尚非未修改 DOSBox E2。
- 正式 `postbattle_ch10_persist` 現依序執行淡出、`+5 bit7` 清除、sparse
  patch、重繪、淡入、FDTXT_010 index4、ACTING37、index5、sync、JOIN11、
  JOIN6、chapter10，之後保留 `town_ch11`，不直接跳下一戰。
- `syncPartyFromBattle` 會在 typed identity 尚未物化時消費 raw `+8`，並保留
  到 persistent snapshot，修正 JOIN 後被 legacy `Fig` 覆寫的身分缺口。
- ch00 `0x3241F` 因 map32 runtime roster 尚缺 raw FDICON key，只保留 exact
  call-site 的 RGBA E1 近似；它不是 `0x1F525` parity 或其他 handler fallback。
- 非破壞性位址、位元組、hash 與推論等級見
  [`fd2_ch09_post_ida.txt`](../data/ida/fd2_ch09_post_ida.txt)。postbattle 稽核
  現為16 active／8 blocked；本切片仍缺一般玩家同狀態 DOSBox 逐幀比較。
- Docker 驗證通過：583個 JSON 可解析、handler extractor 重新讀得
  `0x236F6→0x1F525` 零參數、postbattle／story coverage 稽核一致、變更文件
  的本地連結與圖片無缺漏，且 `go test ./... -count=1` 全數通過。Ebiten 測試
  另實際呼叫 indexed palette Draw path，確認六位元 DAC palette 與呈現
  acknowledgement；不使用遊戲迴圈外不合法的 GPU pixel readback。

## 2026-08-02：玩家第20戰 raw ch19 戰後與整備 record0 勘誤

- IDA Pro 9.4 主判讀與 Docker Capstone 覆核已固定 `sub_23E74`
  （`0x23E74..0x240FA`）的直接寫入與控制流程。四張原始 byte table 對 runtime
  slots `0..15`、`52..60` 寫座標與 pose，並把 camera／cursor 設為 `(26,31)`；
  重製只允許來源 `0x23EC4` 的 `direct_record_patch`，未列欄位保持原值。
- `0x23FFE` 比較 round counter 與15，`0x24005 jg 0x240C6` 證實
  `round > 15` 會略過 group1、ACTING60–62、FDTXT index14–16與JOIN28；
  JOIN25及index13位於共同路徑。舊 handler 無條件執行 JOIN28 的線性圖已撤回，
  改為可編輯的 `native_round_gt(15)` 分支。
- `sub_320FC` 另證實整備 selection table 只描述 persistent records
  `1..count-1`：目的 index 從1開始，record0固定且不消耗 quota。舊重製把
  record0放入 `prepIDs`，使一般／後期最多只上場15／19人；現修正為固定
  record0加選15／19人，總數16／20。
- 玩家第20戰的16筆玩家區加 map19 group0的67筆，形成83-slot入口；低回合
  group1一筆後為84，且ACTING60直接消費slot83。這項frontier有靜態 producer／
  consumer交叉支持，仍維持E1／強推論，尚未宣稱未修改一般玩家DOSBox E2。
- 完整回歸曾揭露舊 `ch20.json` 缺少 runtime append，會把全部70筆FDFIELD
  records與16名玩家同時保留，錯誤形成86 slots。原始position資源059的header
  86其實是70筆單位座標加16格部署座標，不是runtime count；scenario現只在
  開場追加group0的67筆，group1留給戰後分支。
- 本段同時勘誤較早把 `ch19_post` 配到 `map18` 的記錄：玩家第20戰的 raw
  owner與戰場分別是 `ch19_post`／map19；較早的 `16 active／8 blocked` 統計
  也已由本輪實際稽核的17 active／7 blocked取代。
- 正式 `postbattle_ch20_persist` 已綁定 raw `ch19_post`，兩條回合路徑都保留
  chapter20並進入 `town_ch21`，不直接跳下一戰。決定性回歸覆蓋整備固定筆、
  83→84 frontier、round15／16、JOIN25／28及城鎮邊界。
- 非破壞性位址、原始位元組、雜湊與推論等級見
  [`fd2_ch19_post_ida.txt`](../data/ida/fd2_ch19_post_ida.txt) 與
  [`fd2_preparation_fixed_record_ida.txt`](../data/ida/fd2_preparation_fixed_record_ida.txt)。
  Docker驗證已通過882個JSON解析、17 active／7 blocked戰後稽核、故事覆蓋
  稽核、235個本地文件／圖片目標與`go test ./... -count=1`完整回歸；舊
  `map18`及16／8統計只保留在時間序列歷史段，並由本段明確勘誤。

## 2026-08-02：本週暫停點與下輪恢復順序（歷史；2026-08-09 勘誤見下節）

- 已推送基線為`030f1d2`（`feat: 閉合第20戰戰後與整備名冊`），本機HEAD與
  `origin/main`相同；本節寫入前工作樹乾淨。第20戰切片的完整回歸與文件稽核
  結果以上一節為準。
- （歷史，2026-08-02）可稽核現況為24個標準戰後節點中17 active／7 blocked；
  第16戰當時仍 blocked。後續的第16戰與第18戰 production E1 勘誤已取代這組
  統計；目前數字只以本檔最末的 2026-08-09 最新節為準。
- story/cutscene覆蓋稽核為121節點、9個獨立script、48個handler binding、
  64個inline／generic fallback。後一數字包含retreat、rumor與generic節點，
  不能直接換算為「65段原版劇情遺失」，但也不能當成已具原版handler證據。
- 第16戰 raw `ch15_post` 已完成本輪切片：四條分支、76-slot runtime provenance、
  四條 raw 分支均已通過 Docker/Xvfb E1 回歸，包含 JOIN18 arm。
  正式 campaign 已接入 production binding，不再維持舊 candidate-only gate。
  詳細位址證據仍見 `fd2_ch15_post_ida.txt`；76-slot runtime provenance、JOIN18
  persistent record、`town_ch17` 與 save/load 均已驗證；未修改一般玩家 DOSBox E2
  仍是獨立 gate，下一順位為玩家第17、18、22、23、24、29戰。
- UI-01至UI-12仍全部是partial，只有城鎮／商店的部分ch02狀態達E2。完整目標
  仍包括同狀態逐幀比較、剩餘硬編碼對話／演出資料化、敵方AI與戰鬥機制、
  以及無debug的30章一般玩家可破關鏈；目前不能宣稱重製接近完成。
- 當時只確認本輪自己啟動的一次性容器與暫存資料已清理；不可把這句當成後續
  交接時的即時容器狀態。最新狀態以本節後段的 `docker ps` 檢查為準。

## 2026-08-09：第16戰 raw ch15 post production E1 勘誤（歷史；下節已更新現況）

- 正式接入 `postbattle_ch16_persist`：`ch15_post.json`、binding 與 `ch16.json`
  的 `runtime_append_groups=true` 均已加入；16 個 persistent party records
  先於 map15 group0 的60筆，入口固定76 slots。
- Docker/Xvfb 真實測試通過四條 raw branch：round>18、inactive count>4、
  slot0 word+0x42<0x140、以及唯一 raw +8=18 的 JOIN18 arm。四路都進
  `town_ch17`，JOIN18 路徑另通過 town boundary save/load。
- 當時稽核：24 個標準 postbattle 節點為18 active／6 blocked；這組數字與
  玩家第18戰的 raw `ch17_post` 尚未納入，已由下節最新稽核取代。未修改一般
  玩家 DOSBox E2 仍未完成。
- 本輪實際驗證：`go test ./... -count=1` 與 `go test ./internal/campaign` 相關
  binding／coverage 測試均在 `fd2-go-test-local`＋Xvfb 通過，包含
  `TestChapter15PostFourRawBranchesJoin18Town17AndSaveBoundary`。本輪一次性
  測試／稽核容器均使用 `--rm`；交接檢查時另有其他代理持有的 FD2 抓圖與 IDA
  容器，未予停止；`/tmp/fd2cap` 不存在。

## 2026-08-09：raw ch17 post 接入玩家第18戰（勘誤後最新）

- 初始候選曾把 `ch17_post` 接到 `postbattle_ch17_persist`；真實
  `audit_postbattle_binding_gates.py` 回報 `active_index_mismatch`。直接比對
  `battle_chNN→raw ch(N-1)_post` 後已撤回：玩家第17戰仍 blocked、raw ch17
  正確 owner 是玩家第18戰的 `postbattle_ch18_persist`。
- IDA Pro 9.4 Docker 直接證據已保存於 [`fd2_ch17_post_ida.txt`](../data/fd2_ch17_post_ida.txt)：
  `sub_23CD5`、sub_233C6 的 slots 0..16＋special slot17、camera `(432,96)`、
  FDTXT_018 index7/8/9/10 與 acting 56/57/58 的原始位址、檔案偏移及固定
  FD2.EXE 雜湊均保留；可編輯演出稿為
  [`ch17_post.json`](../../remake/assets/cutscenes/acting/ch17_post.json)。
- FDTXT index10 以兩段 scene mapping 保存 scene3 line9 與 scene4 lines0..10，
  不再把跨場景字串誤當一段。map17 group0 37筆＋18個 persistent party records
  形成55-slot runtime；JOIN21／7 改由明示的強推論
  [`native_join_base_units.json`](../../remake/assets/data/native_join_base_units.json)
  提供 map17 raw +8 base，未知角色仍 fail-closed。
- Docker/Xvfb 真實回歸通過 `battle_ch18→postbattle_ch18→town_ch19`、演出、
  JOIN21／7、town save/load，以及 `TestNativeJoinBaseTable*`。（歷史快照；後續
  `ch16_post` 接入已取代）當時的舊 postbattle 覆蓋統計不再作為現況依據；story
  稽核為121節點、49個 handler binding、63個 fallback。未修改一般玩家
  DOSBox E2 仍未完成。
- 本輪所有測試、IDA、Capstone、JSON／Markdown 稽核均在一次性 Docker 容器內；
  `/tmp/fd2cap` 不存在。工作結束時不得留下本輪 FD2 容器；其他專案的既有
  `modest_hermann` IDA 容器不屬本輪，未予停止。

## 2026-08-09：raw ch16 post 接入玩家第17戰（最新）

- 前一節的 raw `ch17_post→玩家第18戰` 判讀維持有效；本節補上正確的下一個
  owner：`battle_ch17` 使用 raw `ch16_post`，handler `sub_23B5F` 位於
  `0x23b5f..0x23cd5`。任何把 raw ch17 接到 `postbattle_ch17_persist` 的舊候選
  均不得恢復。
- 合法 IDA Pro 9.4 Docker 與 Capstone Docker 已保存固定 FD2.EXE 雜湊、原始
  table／call site／FDTXT_017 index5–8、ACTING50–53、JOIN16 及
  `sub_233C6` 的 layout／PAN 參數於 [`fd2_ch16_post_ida.txt`](../data/fd2_ch16_post_ida.txt)。
  共享 ACTING call site 以 `source_addr#acting_id` 作用域覆寫保存 immediate
  分支，不改寫原始位址；新增資源稿與 binding 都是可編輯資料。
- Docker/Xvfb 回歸以真實 `ch16_pre`／`ch17`／`ch16_post` 資料驅動兩條
  `roster_has(18)` branch：有18由60進61 slots，無18由61進62 slots；兩路都
  進 `town_ch18`，JOIN16 路徑另驗證持久隊伍與 town save/load。編譯器只在靜態
  分支採 slot 上界，runtime 仍拒絕不精確 count 或缺 raw provenance。
- 目前稽核為24個標準 postbattle 節點 **19 active／5 blocked**；story/cutscene
  為121節點、9個獨立 script、49個 handler binding、63個 fallback。剩餘 blocked
  為玩家第22、23、24、25、29戰；未修改一般玩家 DOSBox E2、完整30章無除錯路徑及
  UI／AI缺口仍不得以本批 E1 測試宣稱完成。所有本批一次性容器使用 `--rm`，
  `/tmp/fd2cap` 不存在；交接前仍須再檢查 FD2 容器狀態。

## 2026-08-09：raw ch21 loader frontier 勘誤（玩家第22戰；仍未接入）

- 前段曾把「map21 authored 70 筆不足以知道戰後 runtime slots」寫成完全
  未知；這個說法已由新的 IDA Pro 9.4 loader 證據收窄，不再作為現況斷言。
  `sub_1088D` 讀 map21 `FDFIELD_064` header `15 10 46`，明確得到 16 個
  持久名冊槽與 70 筆控制列；`sub_10B4E(0)` 依 raw group byte+0x15 呼叫
  `sub_10C50` 追加 runtime record。positions `FDFIELD_065` 的 header 86
  是 70 筆列位置加 16 部署格，不是 86 個 runtime slots。
- 全域 runtime buffer 交叉證據顯示 `[0x53a45]` 配置 96 個 `0x50`-byte
  槽，而 `[0x53beb]` 是追加 count；因此 66 不是原版物理容量上限。這只
  讓「候選已 materialize 數量」與「配置容量」分開，沒有證明短前沿的
  slot72 record 已具備可供 renderer 消費的內容。
- map21 raw group 分布 group0=50、group1=6、group2=1、group3=6、group255=7，
  對照 `ch22.json` 的 group0 初始及回合3／5／7 追加後，候選 native frontier
  為 **66→72→73→79**。這是**強推論**，不是已驗證一般玩家 runtime trace；
  證據與固定版雜湊見 [`fd2_ch21_post_ida.txt`](../data/ida/fd2_ch21_post_ida.txt)。
- `postbattle_ch22_persist` 仍保持失敗即關閉：`0x233c6` 的 16 格 layout、
  special slot72、raw camera 與 acting 65/66 已由 IDA／Docker exporter 轉成
  E1 可編輯候選，並有 compiler regression；各 frontier 的同狀態 raw record、
  一般玩家 runtime trace 與 indexed 畫面狀態仍未閉合。`0x24618` 的九段 LUT
  9→1、5ms／段、500ms 尾端、0..62／4ms 調色盤序列與固定目的地 `0xA0504`
  已由 IDA／Capstone 閉合；其 raw
  相對游標 globals（`0x53ab9/0x53abd`）與 ch21 呼叫點 `0x245ce` 的 Y+3
  變換已由 IDA 證實；重製端已補上依呼叫位址核對、帶 provenance 的 fail-closed
  動態欄位橋接，但正式 binding 仍未建立。不得只因 frontier 已可估算就建立
  production binding 或接 `town_ch23`；城鎮／整備邊界仍保留在 campaign graph。

## 2026-08-09：raw ch22 pre 可編輯候選（玩家第23戰戰前；尚未接入）

- 合法 IDA Pro 9.4 與 Docker Capstone 以固定雜湊的 `FD2.EXE` 重讀
  `0x336a0..0x338c0`。已證實 `0x336ab→0x205da` 載入 context、
  `0x336b5→0x32975` 的 `EBX=0..15` 16 次迴圈、PAN `(14,32)`、
  `(14,29)`、`(14,13)`、`0x336e5→0x24618` 的 raw push
  `[0x53abd]+5`／`[0x53ab9]+6`／`10`／`8`，以及 redraw、palette、
  FDTXT_023 0..4、ACTING 68..70、group1 spawn、reset/focus 順序。完整輸入
  雜湊、工具版本與位址基準見 [`fd2_ch22_pre_ida.txt`](../data/ida/fd2_ch22_pre_ida.txt)。
- 新增可編輯 handler 與研究 binding：
  [`ch22_pre.json`](../../remake/assets/cutscenes/handlers/ch22_pre.json) 及
  [`ch22_pre_candidate.json`](../../remake/assets/cutscenes/bindings/ch22_pre_candidate.json)。
  compiler regression 通過 map22／70 slots／group1 24 rows、五個 FDTXT source、
  ACTING 68..70 與原始 call-site 保留；candidate 不會被正式 campaign 自動載入。
- `0x24618` runtime bridge 現依呼叫位址核對 raw cursor：ch21 `0x245ce` 只接受
  Y+3，ch22 pre `0x336e5` 只接受 Y+5；未知來源或偏移失敗即關閉。這修正了
  不能把所有 handler 都套用 Y+3 的過度概括，但不宣稱兩者已完成 indexed renderer。
- 【歷史快照，現況以 `58-fd2-exe-re-coverage.md` 為準】當時記錄的
  `story_ch23`、`postbattle_ch22_persist→town_ch23` 節點名稱已由後續勘誤改為
  `postbattle_ch23_persist→preparation_ch24`；一般玩家 DOSBox E2、
  戰後城鎮／商店／整備／存檔與完整戰役鏈仍保持 blocked；本批只達 E1 的
  靜態證據與資料消費候選。未來接入前仍需 raw runtime trace、同狀態畫面及
  town boundary 回歸。

## 2026-08-09：raw ch22 post `0x2189a` 原語勘誤（玩家第23戰戰後）

- 舊交接中把 `0x2189a` 只列為 `unknown`，已被本輪 IDA Pro 9.4／Capstone
  交叉證據收窄：`0x24754` 在 `0x24978`、`0x249c4`、`0x24a10` 三處呼叫同一
  個十次索引呈現 helper；`0x21914`、`0x21955`、`0x2195d`、`0x21986`、
  `0x219a3` 的巢狀 call-site、raw push 形狀、`work+0x8088`、stride 456、
  13×8、312×192 與呈現 stride 320 已保留。原始 handler 控制流程與
  `0x24b14(100)`／`0x24bde(18)` 的 raw 條件則另行保留，沒有替未知 record
  建立角色或玩法名稱。
- 重製端新增 `Native2189ALoop`／`native_2189a_loop`，三個 JSON beat 及
  compiler／fail-closed regression 已完成；runner 仍會在沒有 indexed state
  adapter 時停止，沒有猜接 renderer 或 campaign。完整證據與固定檔案雜湊見
  [`fd2_ch22_post_ida.txt`](../data/ida/fd2_ch22_post_ida.txt)。
- 本輪 Docker 回歸：`go test ./internal/campaign ./cmd/fd2 ./internal/battle`
  通過；這是 E1 可編輯資料消費切片，不是一般玩家 E2。`postbattle_ch22_persist`
  仍 blocked，下一節點不可假設為 `town_ch23`；城鎮／商店／整備／存檔邊界仍需
  raw runtime trace、indexed adapter 與未修改玩家路徑驗證。

## 2026-08-09：raw ch22 post 分支條件轉成 editable CFG

- 直接指令核對固定 `0x247c6 cmp eax,-1; je 0x247fb`、`0x24840 test eax,eax;
  je 0x248b5`，以及 `0x248b5 cmp dword [0x53bef],0xf; jl 0x24902`。因此
  `0x24b14(100)` 找到時才走 text8／JOIN22，`0x24bde(18)` 找到時走
  text10／ACT72／deactivate17／text11；未找到 18 時再依 raw counter `<15`
  選 text13／JOIN19 或 text12／ACT72／deactivate17。
- `ch22_post.json` 現保存三層巢狀 `if`，條件名稱僅描述 byte-level predicate：
  `native_inventory_item_present`、`native_persistent_identity_present`、
  `native_round_lt`。compiler 與 BeatRunner regression 通過；執行器缺少
  完整 raw inventory／persistent record／round provenance 時失敗即關閉，不使用
  normalized inventory 或角色名稱替代。
- 這是 E1 控制流資料化，不是畫面或一般玩家 E2；`postbattle_ch22_persist`、
  indexed renderer 與戰後 town／shop／整備／save 邊界仍保持 blocked。
- 同步清理 SDD 中一條過時的「active ch29-post binding 可執行
  `0x24618`」句子：目前 transition adapter 只可在完整候選資料下隔離執行，
  `postbattle_ch29_persist` 沒有正式 campaign binding，仍等待 `0x2bce5`
  terminal renderer 與正確 owner index；這避免把候選 adapter 誤讀成戰役已接通。

## 2026-08-09：Markdown 斷言清理與現況入口校正

- `99-reflections-log.md` 第 7 輪的「9 項必要能力全具備／重製只剩工程整合」及
  `91-worklist.md` 第 11 輪的「全 30 章一條龍可玩」都是當時的規劃或 authored
  快照，已改成明確的歷史限定語；沒有刪除資產、raw 對映或方法證據。
- `26-per-chapter-event-handlers.md` 的標題改為「原版 handler 對照（raw 對映已驗；
  不等於重製流程完成）」；`56` 頂端日期與現況同步到 2026-08-09，列出 19 active／5
  blocked 及玩家第 22、23、24、25、29 戰的失敗即關閉狀態。
- 這次只修正會污染記憶的完成度句子，沒有把任何未證實 renderer、handler 或戰後
  town／shop／整備／存檔語意接入正式 campaign。

## 2026-08-09：ch23 tick gate 交叉驗證

- Capstone Docker 逐指令核對 `0x11eee`：raw chapter `0x17`（十進位 23）在
  `0x120a7` 比較 `[0x46c]` 與 `[0x539f8]`，只有 tick 改變才由
  `0x120af→0x24d22(0)` 旋轉 staging，並在 `0x120b9` 更新 last tick。
  「tick gate 完全未知」已撤回；它是 BIOS tick 變化閘門，不是固定影格率。
- 同一分支的 `0x120c6..0x120fe` 保留 `0x53aff`、`0x138` row stride、
  `0xc0` rows 與 `0x11eb0` present 呼叫形狀；IDA／Capstone 又固定
  `0x11CAC` 尾端以 `sub_11EB0(0xA0504,0x140,work+0x8088,0x1C8,0x138,0xC0)`
  將共用 indexed work buffer 複製到固定線性目的地 `0xA0504`。後續 `0x10652`
  載入器證據已證實 raw staging 的建立／擁有者；固定版 raw seed 為 `0x01`，但
  `0x53AED`／`0x53AF5`／`0x53AF1` 中間偏移生命週期或 handler 入口 latch 的
  執行期值仍未知，因此 `postbattle_ch23_persist` 仍
  fail-closed，沒有把 `nativeMapWork`／PNG framebuffer 猜成原版 staging。
- SDD、worklist 與 `fd2_ch23_post_ida.txt` 已同步；這是 E1 反組譯證據收窄，
  不是 production renderer 或一般玩家 E2 完成。

## 2026-08-09：ch23 raw staging loader owner 交叉驗證

- Docker Capstone 以同一固定版 `FD2.EXE` 重讀 `0x10652..0x1088d`。在
  `0x107d4`，只有 raw chapter `[0x53c03]==0x17` 進入分支；
  `0x107dd→0x107ea` 呼叫 `0x36d16(0xea00)` 並把指標寫入 `[0x53aff]`。
  `0xea00` 恰為 `0x138×0xc0`，所以 raw ch23 staging 的配置大小與 owner
  pointer 已有直接證據。
- `0x107ef..0x10804` 保留 `push 0x2a; push [0x53b03]; push 0x1a4d;
  call 0x111ba` 的原始參數形狀；`0x10809..0x10820` 保留
  `push -1; push 0x138; push [0x53aff]; push 0; push 0; push eax;
  call 0x4e63d`；`0x10823..0x10831` 呼叫 `0x37416` 後清零 `[0x53b03]`；
  `0x1083b→0x10842` 最後呼叫 `0x24d22(0)`。同函式 IDA 資料表已將這個
  分支對到 `FDOTHER.DAT` #42；這收窄 loader 的 raw decode／handle cleanup
  邊界，但不把 #42 命名成背景、轉場或 UI。
- 重製端新增 `fdother.DecodeNativeCh23Stage`／`BlitNativeCh23Stage`，以真實
  `FDOTHER.DAT` #42 regression 固定 312×192、`0x138` stride、`0xea00`
  staging surface 與透明 `0x4e63d` blit。這是可執行的 E1 原語，不是完整
  indexed state/latch renderer；`native_ch23_loop` 仍保持失敗即關閉。
- 既有「staging 建立／擁有者未知」已修正為「raw loader owner 已知；固定版 raw
  seed 為 `0x01`，共用尾端目的地 `0xA0504` 的位址／複製 ABI 已知，但
  `0x53AED`／`0x53AF5`／`0x53AF1` 中間偏移生命週期、入口 latch 的執行期值與
  raw state adapter 仍未知」。
  沒有因此新增 renderer、campaign 或 `postbattle_ch23_persist` binding；
  城鎮／商店／整備／存檔邊界與一般玩家 E2 仍保持失敗即關閉。

## 2026-08-09：ch29 `0x2c548` party montage standalone executor

- 合法 IDA Pro 9.4 與 Capstone 5.0.3 以固定雜湊 `FD2.EXE` 重讀
  `0x2c548` 所在的 `0x2c405`、`0x29164`、`0x2935b`、`0x2b9a1`、
  `0x168b6`、`0x4e8af` 與 `0x10620`。新增證據檔
  [`fd2_ch29_montage_ida.txt`](../data/ida/fd2_ch29_montage_ida.txt)，保留
  原始位址、slot swap、resource index、descriptor `+6` tick 與五個文字
  destination，不把未知欄位命名成角色或按鍵。
- 新增 `internal/ending.LoadMontageCycleAssets`：唯讀載入
  `FDOTHER#56`、`TAI#3`、`FDOTHER#5` grid、`FIGANI`、`DATO`、
  `FDTXT_031/FDTXT_000` 與 FDOTHER#4 font；來源不完整或雜湊／格式不符時
  失敗即關閉。
- 新增 `internal/ending.MontageCycle`：實際逐步執行九段
  mirror/non-mirror fade、20 次 secondary intro、primary FIGANI descriptor
  `+6` delay、DATO/FDTXT portrait 220／440 loop，以及 `0x1f882` 的 64 段
  palette 收尾。Docker 以原始資源跑完兩個 unit side branch 的完整 cycle
  regression；這是獨立 indexed renderer，不是 campaign 接線。
- 明確勘誤：舊「party scheduler／dedicated renderer 完全不存在」只適用於
  本節之前的歷史快照，現況已由 standalone executor 取代；`0x10620→0x4e031`
  的 raw word 比較／複製已以 `NativeBIOSKeyboardState` 保存，但按鍵映射與
  外層輸入事件仍未知；`0x28a64` 已校正為共用清理 epilogue，不是後續 owner。
  一般玩家 E2、`0x2c194..0x2c39a` handoff 與
  `postbattle_ch29_persist` campaign terminal 仍保持失敗即關閉。完整證據見
  [`fd2_ch29_input_cleanup_ida.txt`](../data/ida/fd2_ch29_input_cleanup_ida.txt)。

## 2026-08-09：ch29 montage 後續 raw tail schedule

- 合法 IDA Pro 9.4／Docker Capstone 固定 `0x2c194..0x2c39a` 不會直接回傳
  campaign：先清 VGA、載入 FDOTHER #60，接著載入 #58／#57，跑 20 次 raw
  table loop，最後載入 #59 並釋放。`0x2c253..0x2c33b` 的三組 20-byte
  table、unit raw offsets、`0x28a6c`／`0x11d40`／`0x2935b`／`0x1f882`／
  `0x375c0` 呼叫與 20／78 tick 已輸出到
  [`fd2_ch29_post_montage_tail_ida.txt`](../data/ida/fd2_ch29_post_montage_tail_ida.txt)。

## 2026-08-12：20 組終局尾段近似播放器（E1；忠實模式仍失敗即關閉）

本節是同日較早「近似模式只驗證資源後直接呈現 #59」及「20 段重製轉接器整體
尚不存在」的追加勘誤。真正阻礙玩家可見成果的缺口不是再擴張
`MontageTailLoaderBaseline`，而是缺少不冒充精確 `0x28a6c` 的有界播放器；本輪
因此停止擴張載入器，完成以下垂直切片：

- `MontageTail.PlanVisualResources` 保留每輪 TAI／BG 與 record0／record1 的四項
  FIGANI 原始選擇算術；合法 IDA Pro 9.4 交叉參照重新確認 record1 也同時載入
  `3*record1[+7]+1`，先前少列該資源的文件已勘誤。
- `LoadMontageTailVisualSets` 在播放前一次驗證 20 組 TAI／BG／FIGANI；任一晚期資源
  不可解碼時不建立半套播放器。`MontageTailPlayer` 不寫入戰役、戰鬥、隊伍或存檔，
  只在 `FD2_APPROXIMATE=1` 的已通過來源驗證終局節點執行。
- 近似合成依已證實幾何放置 BG 與 TAI，先播放 record0 auxiliary／record1 base，
  再播放 record1 auxiliary／record0 base，接著保持兩個 base、疊上 FDOTHER#58 的
  對應影格；20 組完成後才呈現並保持 FDOTHER#59。這不補寫未知的狀態欄、滑動、
  聲音、效果或呼叫時 runtime records。
- FIGANI frame descriptor `+4/+5/+6/+7` 已更正為四個獨立位元組；`+6` 才是延遲。
  舊解碼器把 `+6/+7` 合成 little-endian u16，會把正常延遲誤讀成數千 ticks，已由
  真實資源回歸鎖定。未證實的 `+4/+5/+7` 只保留 raw 名稱。
- `FDMUS_018` 現在於近似尾段開始時接線，而非先跳到 #59 後才播放；精確停曲、
  呼叫間隔與畫面同步仍不宣稱等同原版。

決定性回歸使用玩家自備原版 archive，驗證全部 20 組交易、超過 100 次索引畫面
呈現、來源延遲、相同輸入的相同輸出、缺件失敗即關閉，以及最後畫面等於獨立解碼的
#59。可審查的 20 組總覽見
[`ending-tail-20-segments-approximate-remake-e1.png`](../figures/ending-tail-20-segments-approximate-remake-e1.png)，
其雜湊、產生命令與限制見
[`ending-tail-20-segments-approximate-remake-e1.json`](../data/ui-traces/ending-tail-20-segments-approximate-remake-e1.json)。

現況應統一讀成：近似模式已有玩家可見的 20 組原版資源尾段；忠實模式仍缺
`0x2c2a6` 呼叫當下完整 records/globals、精確 `0x28a6c` 狀態欄／滑動／聲音／效果
renderer、原版輸入與時序，以及未修改一般玩家終局 E2。較早把「精確轉接器缺口」
寫成「所有 20 段 consumer 都不存在」的句子只代表當時狀態，不再是現況斷言。

- 新增 `remake/assets/endings/native_2c194_tail.json` 與
  `ending.MontageTail.Plan`。它只驗證資源索引、raw tables、位址契約並產生
  20 筆 raw entry；沒有寫入 battle state、沒有猜動畫／角色名稱，也沒有把
  `postbattle_ch29_persist` 接到下一城鎮。indexed resource owner、輸入事件、
  campaign／town／shop／整備／save handoff 與一般玩家 E2 仍失敗即關閉。

## 2026-08-09：ch29 終局 caller table 校正

- IDA／Capstone 固定 `0x25e23` 以 `[0x53c03]` 消費資料映像 `0x51de9`：
  index26→`0x250cc`、index29→`0x25757`。`0x25757` 的
  `0x25970→0x2bce5→0x25975` 是 self-loop；`0x250cc` 依
  `0x24b14(0x64)` 分成 success exit 與 ending self-loop。這些是 raw
  caller／控制流證據，不是玩家戰次名稱或 town／preparation owner。
- 已新增 [`fd2_ch29_terminal_callers_ida.txt`](../data/ida/fd2_ch29_terminal_callers_ida.txt)。
  這次只刪除／校正「table index 可直接命名戰次」與「self-loop 可直接回城」
  的不必要斷言；`postbattle_ch29_persist`、完整 ending owner、輸入事件、
  一般玩家 chapter provenance 仍保持失敗即關閉。

- 2026-08-09 ch23 staging owner xref：IDA／Capstone 追查固定 `0x53aff` 只由
  raw `0x10652..0x1088d` loader 配置／清理並交給 `0x11eee`／`0x24d22`，
  `0x51a10` 只在 `0x24d22` 讀寫，`0x539f8` 只在 `0x11eee` tick gate
  比較／寫回；`0x53aed`、`0x53af1`、`0x53af5` 的寫入則分散於
  `0x12eaa`、`0x1300d`、`0x13185`、`0x13315`。這只關閉 raw offset-state
  的靜態 owner 邊界，不把欄位改名成 camera／framebuffer／`nativeMapWork`，
  也不解除 `postbattle_ch23_persist` 的 indexed／campaign fail-closed gate。

- 2026-08-09 ch22 `0x2189a` global xref：IDA data-xref 固定 caller 只讀取
  共用 `0x53a49`／`0x53aa9`／`0x53aad`／`0x53a6d`，這些位址同時被
  `0x11cac`、`0x11eee`、`0x127a9`、`0x21eb1`、`0x22046`、`0x24618` 等
  多個呈現 caller 消費；`0x53b03` 是 raw resource loader handle，
  `0x53b07`／`0x53b0b` 由共用呈現函式寫入。這只收窄 owner 邊界，不把工作區
  欄位命名成 camera／portrait／effect／framebuffer；`native_2189a_loop`、
  `postbattle_ch22_persist` 與戰後城鎮／整備／存檔仍保持失敗即關閉。

## 2026-08-09：DOSBox 原版抓圖工具校正

- `tools/docker/fd2-dosbox-screenshot.sh` 的 `shot:` 已改為擷取實際 DOSBox
  用戶端視窗（client window）；Xvfb 根視窗只用於既有固定座標就緒探測（readiness probe）。Docker／DOSBox
  實測產生 320×200 的標題選單與 `FD2.SAV` CONTINUE 後戰場對話畫面，確認先前
  1024×768 根視窗截圖只是左上角局部，不再把它當成 UI 證據。
- 既有 `title-original-dosbox.png` 與新的原版標題畫面逐像素相同；新的原版戰場
  畫面與 `ch01-dialogue-original-dosbox.png` 只有動畫時序差異，因此沒有新增原版
  重複圖。另保存重製端 Docker／Xvfb 執行期畫面
  [`title-remake-runtime.png`](../figures/title-remake-runtime.png)，供 README 與
  UI 矩陣標示 E1 差距。原版一般玩家 E2、戰後節點與 AI 正式執行仍未因抓圖工具
  修正而解除失敗即關閉。

## 2026-08-09：重製端對話框執行期證據

- 以 `fd2-go-test-local` Docker／Xvfb 執行 `FD2_CAMPAIGN=1`，從可編輯的序章腳本
  產生 640×400 重製端對話框畫面
  [`dialogue-remake-runtime.png`](../figures/dialogue-remake-runtime.png)。
  圖檔 SHA-256 為
  `6e66e18ed66ac018c69a29d7e2f880444c96d68d4e4ad021626663b4cd914061`。
- 將該畫面縮放至原版 DOSBox 320×200 對話框 oracle
  [`ch01-dialogue-original-dosbox.png`](../figures/ch01-dialogue-original-dosbox.png)
  後，實測平均絕對誤差（AE）為 `60414`。這證明資料腳本→肖像／文字→重製端
  渲染器（renderer）的 E1 消費鏈已存在，但不是同狀態 E2，也沒有解除上／右肖像位置、
  控制碼、裁切、分頁或一般玩家路徑缺口。
- 本次只新增可回查的執行期圖與現況統計勘誤；沒有把對話 renderer 的未證實語意
  接入戰役正式流程。暫存擷取目錄在 Docker 內清理，主機 `/tmp/fd2cap` 仍不存在。

## 2026-08-09：歷史統計與 ch21 未知語意勘誤

- SDD 中 2026-08-02 的 13 active／11 blocked 已明確標為歷史快照；目前有效稽核
  仍是 19 active／5 blocked，玩家第22、23、24、25、29戰均失敗即關閉。
- 工作清單的 2026-08-02 區段已標為歷史檢查點，並補回漏列的玩家第25戰；這只
  修正文件斷言，不把任何候選處理器（handler）提升為正式戰役接線。
- 合法 IDA Pro 9.4 Docker 重跑 `0x17aa9`、`0x24d22`、`0x11d40`、`0x24618`、
  `0x22046`、`0x25a96`、`0x1f882` 與 `0x11df2` 後，結果與
  [`fd2_ch21_post_ida.txt`](../data/ida/fd2_ch21_post_ida.txt) 一致；ch21 的
  執行期前沿（frontier）／索引畫面狀態（indexed runtime）仍不足，
  `postbattle_ch22_persist` 維持失敗即關閉。
- 同次 IDA 直接程式碼交叉參照另固定 `0x24618` 的 8 個直接呼叫者（`0x245ce`、
  `0x252ee`、`0x25848`、`0x336e5`、`0x33bb9`、`0x33c09`、`0x33c66`、
  `0x33ce2`）與 `0x22046` 的 6 個直接呼叫者；這只是共享 indexed
  消費端（consumer）的**已證實**邊界，不能把函式改名成單一戰後角色演出或通用淡出。
  完整原始位址、函式範圍及「不提升為 campaign owner」限制已追加到
  [`fd2_ch21_post_ida.txt`](../data/ida/fd2_ch21_post_ida.txt)。
- 同輪 IDA 又固定 ch22 `0x24a92→0x4dbfc` 的完整 raw 遮罩：依
  `u8[base+0]*u8[base+2]` 決定筆數，逐格寫 `+3=0xff`、遮罩 `+2=0x1f`、
  `+1=0x03`；該函式共有34個共享直接呼叫者（callers）。因重製狀態沒有完整 cell
  `+1/+2/+3` buffer，沒有把這項證據誤降成只重設 `byte+3`；ch22 的 unknown
  與 `postbattle_ch23_persist` 失敗即關閉邊界保持不變。完整證據見
  [`fd2_ch22_post_ida.txt`](../data/ida/fd2_ch22_post_ida.txt)。

## 2026-08-09：AI `0x14ef0` raw 尾端分派契約

- 合法 IDA Pro 9.4（9.4.0.260610）與 Docker Capstone 5.0.3 以固定
  FD2.EXE 雜湊重跑 `0x14ef0..0x15055`；函式先固定呼叫
  `0x14237→0x1598a→0x1567e`，再讀 `[0x53c4f]`／`[0x53c23]`／`[0x53c33]`
  signed 分數、record `+0x34 & 0x40`、actor `+0x48`、target `+0x4a`，
  tie branch 另呼 `0x4e516([0x53c2f])`。原始路由與六個 direct callers
  已保存於 [`fd2_ai_14ef0_dispatch_ida.txt`](../data/ida/fd2_ai_14ef0_dispatch_ida.txt)。
- 重製端新增 `battle.SelectNativeAI14EF0Tail` 與 raw provenance regression，
  只保存 `0x1548e`／`0x15311`／`0x15055`／共用收尾路由；缺欄位即失敗關閉，
  不執行 producer／尾端函式、不接 `NextAIPlan`。六個 caller 的 turn/camp
  語意、完整交易、UI、一般玩家路徑與 E2 仍未閉合。

## 2026-08-09：mode 2／`0x13fd4` 控制流勘誤

- 合法 IDA 9.4 直接輸出與既有 Capstone `0x13a9f` 指令共同證實：mode 2 的
  `0x13b4d→0x14ef0` 失敗後只呼叫 `0x14237`，再跳入 `0x13b1e→0x13c06`
  共用收尾；沒有 `0x13fd4` 或 `0x13e9c`。`sub_13FD4` 的 IDA direct callers
  是 `0x13bc5`、`0x13c0f`、`0x13c84`、`0x13d58`、`0x13e52`、`0x19082`，
  沒有 `0x13b5f`。完整勘誤見
  [`fd2_ai_mode_dispatch_ida.txt`](../data/ida/fd2_ai_mode_dispatch_ida.txt)。
- 已修正 `PlanNativeUnitMode2`、測試、`11` AI dossier、SDD、gap audit 與
  worklist；`0x13fd4` 的 raw HP 回復與 mode 11／其他已證實 caller 不受影響。

## 2026-08-09：action overlay 72×72 索引快照原語

- 依官方 IDA／Capstone 已保存的 `0x175a9`／`0x17643` 證據，原版開啟前與每幀
  還原固定使用 `72×72 = 0x1440` indexed bytes。重製端在
  `remake/internal/fdother/action_overlay.go` 新增
  `CaptureActionOverlaySnapshot`／`RestoreActionOverlaySnapshot`；兩者都只接受
  caller 明確提供的矩形左上角，尺寸、stride、邊界或快照長度不符時於寫入前
  失敗。`action_overlay_test.go` 以非均質 100×100 indexed fixture 驗證完整
  區域還原、外部像素保留與 malformed/out-of-bounds fail-closed。
- 這只關閉可重生的資料原語，不推論 native private-buffer owner，也沒有把 API
  接進目前每幀整幅重畫的 Ebiten adapter；因此 UI-03 仍是 partial，正式 renderer
  的快照生命週期與 DOSBox 同狀態視覺差分仍是後續 gate。`56` SDD、`57` UI evidence
  matrix 與 `91` worklist 已同步這個邊界，避免把「有單元測試」誤寫成原版畫面完成。

## 2026-08-09：IDA 重核 action overlay 快照 owner

- 官方 IDA Pro 9.4 與 Docker Capstone 5.0.3 重新讀取 `0x175a9..0x176b4`，固定
  `0x175a9` 先配置 `0x1440` bytes，從
  `framebuffer+0x8088+0x18*(dword_53ab9-1)+0x18*0x1c8*(dword_53abd-1)`
  開始，逐 `0x1c8` stride 複製 `0x48` bytes 共 `0x48` 列；`0x17643` 以同一
  來源逐列還原。原始位址與指令、雜湊及工具版本已保存於
  [`fd2_action_overlay_snapshot_ida.txt`](../data/ida/fd2_action_overlay_snapshot_ida.txt)。
- `fdother.ActionOverlaySnapshotOrigin` 現已保存「可視游標各減一個 24-pixel cell」
  的 raw byte address，並以 regression 拒絕零／負 cursor；這修正了先前只知道
  72×72 大小、卻沒有 owner 的文件邊界。現行 Ebiten adapter 仍沒有消費 private
  snapshot，因而不提升 UI-03 的 E2 狀態。

## 2026-08-09：ch23 runner 移除未證實的自動列旋轉

- 重新對照 `0x24d22`：ch23 handler 的非零 `stage=2..14` 呼叫只寫入
  `byte_0x51a10`；`arg==0` 的 312-byte 列複製是在共享 `0x11eee` 的 BIOS tick
  變化閘門下間接發生。先前 `RunNativeCh23Loop` 在每個 stage 開始時直接呼叫
  `RotateNativeCh23Rows`，把 handler 排程誤當成已發生的 renderer 副作用，已移除。
- `NativeCh23LoopHooks` 現要求 `Latch(rawStage)`、`Draw`、`Tick`，palette 段另要求
  raw `ESI` 回呼（callback）；runner 只重現原始呼叫順序並在回呼失敗時回復 staging／DAC，
  不替 caller 命名或猜測 tick 時序。Docker `go test ./internal/fdother -run NativeCh23`
  與 `go test ./internal/...` 均通過；`postbattle_ch23_persist`、indexed adapter
  與一般玩家 E2 維持失敗即關閉。

## 2026-08-09：ch21 多 frontier layout 閘門勘誤

- `ch21_post_candidate.json` 的候選 runtime frontier 仍保留原始證據推導的
  **66→72→73→79**；其 layout 也保留 special slot72，沒有把其中一項猜測
  改寫成事實或刪除。
- 編譯器原先只用最大 slot count（79）檢查 `layout_units`，因此會錯誤產生
  看似可執行的候選；現改為同時追蹤最小「已 materialize」frontier。slot72
  在最小 66-slot 入口尚未取得 record provenance 時，候選以 layout issue
  失敗即關閉，不會產生 `runtime_context`。這是候選資料契約的保守閘門，
  不是宣稱原版 96-slot buffer 不存在或 slot72 必然越界。
  分支指定的 `required_slot_count`、`loadch` 與已證實的 `spawn` 會同步更新
  最小與最大 frontier，既有 ch02/ch06/ch07/ch16 等 branch regression 均通過。
- 新回歸名稱為
  `TestCompileChapter21PostCandidateFailsClosedBelowMinimumRuntimeFrontier`；
  Docker 完整 `go test ./internal/campaign -count=1` 通過。這是編譯期拓撲
  安全閘門，不代表 slot72 已找到原版 materialize 時機；`postbattle_ch22_persist`
  與 `town_ch23` 仍保持失敗即關閉，indexed runtime／一般玩家 E2 仍待。

## 2026-08-09：共用反組譯／GUI 方法論勘誤

- 重新盤點後確認 `~/.codex/knowledge-base/fd2/reverse-engineering-gui-restoration.md`
  與 `sources/claude/retro/ida-pro-9.4.md` 實際可讀；早期「本環境沒有可讀
  知識庫」只屬當時環境觀察，已不再作為現況結論。
- 本輪採用其中的 composition graph、scene owner、E0/E1/E2 分層、IDA xref
  優先、非破壞性斷言索引與 Docker-only 工具分工。這些是方法論，不提升任何
  FD2 地址、資源或玩法語意；真正證據仍必須落在本儲存庫的固定雜湊、IDA／
  Capstone 匯出、資源 regression 或 DOSBox 實驗。

## 2026-08-09：AI 物理候選資料橋（E0，未接正式回合）

- 新增 `battle.BuildNativeAIPhysicalAttackCandidates`，把已閉合的
  `0x145cd→0x4e040→0x146d1→0x14b16` row-major 落點、caller 明示的
  `0x14818` target geometry、raw `+5/+6` 目標篩選與 detached `0x50`-byte
  actor／target record snapshots 串成候選資料鏈。
- 地形百分比修正、`0x1debe` 結果、target `+8` 與 command record 來源仍由
  `NativeAIPhysicalAttackScoreResolver` 明示提供；缺 provenance 或回呼失敗
  即關閉。既有 `SelectNativePhysicalAttackCandidate` 仍負責 priority→score→
  先出現者同分選擇。
- Docker `go test ./internal/battle -run 'NativeAIPhysicalAttackCandidates|NativeAIPhysicalAttackCandidate'`
  通過；這只證明 E0 資料鏈與失敗即關閉，不接 `NextAIPlan`、正式敵方回合、
  movement／battle effect、UI 或一般玩家 E2。下一步應取得固定存檔的
  `0x1b83d→0x1b722→0x4e56c` actor 物品列來源已閉合；下一步只追實際選中目標
  trace，再考慮更窄的 runtime consumer；不得用 normalized `aiTargets` 補缺資料。

## 2026-08-09：戰鬥 FIGANI 幀延遲呈現橋（E1）

- 從固定雜湊版本的玩家 `FIGANI.DAT` 逐幀擷取 descriptor `+6`，新增
  `remake/assets/figani/delays.json`；現有 22 個 PNG 動畫的幀數與延遲均由
  `cmd/fd2/TestFIGANIDelaysMatchNativeFrameHeaders` 交叉比對。
- 新增 `internal/figani.DisplayScheduler`，依官方 `NativeScheduler` 的先呈現後
  累加規則，把 raw delay 轉成明示 `FD2_BATTLE_FPT` 顯示倍率；全螢幕攻擊演出
  不再以固定 15 幀補猜，PNG／延遲表不一致即不建立演出。
- Docker/Xvfb 實跑：`internal/figani` scheduler tests、`cmd/fd2` FIGANI
  placement/delay regression 與 `TestNewAtkAnimRequiresNativeDelayPairing` 通過。
  這只關閉幀順序與停留時間的 E1 呈現邊界；命中幀、傷害、音效、台座、完整
  原版畫面與一般玩家 E2 仍未閉合，不能將本項誤寫成完整戰鬥演出。

## 2026-08-09：戰後城鎮／存檔與 command grid 防禦性回歸

- `TestInventoryRecipeInsufficientReturnsToTownAndSaveLoad` 補上天空之鑰材料不足
  分支的實際可編輯（editable）路徑：recipe→insufficient→town，清除戰場暫態後在
  town22 等價節點存檔，再清空暫態並讀檔，驗證金幣、持續隊伍成員與加入順序。
  成功分支原有回歸保持不變；這不是 raw `ch21_post` handler 或一般玩家 E2 證據。
- `TestNativeCommandGridMissingInputsRemainInert` 驗證空 ID、負值／越界 selected
  不會改指令格（command grid）長度或選取狀態，也不擴大 command effect。兩項測試均在 Docker
  中通過；未修改 renderer、command whitelist 或原版語意。

## 2026-08-09：原版 command grid E2 可達性盤點

- 目前沒有可安全重播的原版戰術指令格時間線。`tools/docker/fd2-dosbox-screenshot.sh`
  只提供通用按鍵／像素同步原語；既有原版時間線只閉合 title、空槽 LOAD、修改路徑
  town／shop 與 ch02 town hub，沒有「進入戰場→選取我方單位→開啟指令格」的已證實前置狀態。
- 原版 `FD2.SAV` 四槽目前皆空；`/tmp/fd2-load-valid` 歷史複本只驗證修改 metadata 後的
  LOAD／城鎮畫面，交接已明確標為修改路徑，不能當一般玩家戰鬥狀態。不能把
  `FD2_SHOT_RING`、command-mask fixture 或重製截圖當成原版 oracle。
- 因此 UI-03 先維持 E0 layout／input contract 與重製端 E1，缺原版存檔時失敗即關閉；
  下一個解除條件是取得合法、未修改、含已知戰鬥前／中狀態的存檔，再在 Docker 可寫副本
  以 `waitpixel`／逐步 `shot` 建立原版與重製同狀態比較。

## 2026-08-09：ch02 城鎮 variant2 正常選項 E2（修改 LOAD 路徑）

- 由固定雜湊的原始 `FD2.EXE`／`FD2.SAV` 建立全新 Docker `/tmp` 副本；slot0
  只複製目前持久隊伍資料（current persistent roster），metadata raw chapter 設為 `5`、roster count
  `4`、currency `0`。原始檔與修改副本的 MD5／SHA-256、欄位範圍及限制集中記錄在
  [`native_town_variant2_e2.json`](../data/native_town_variant2_e2.json)。副本不可視為
  未修改一般玩家存檔，完成後不加入儲存庫。
- 以 Docker DOSBox 重播標題→LOAD→slot0→城鎮，觀察到原版 variant2。實際選項時間線
  先以 selection0 截圖，再以 Left 每次間隔 `0.15` 秒取得 selection1–4；不是把方向鍵
  循環回 selection0 誤當成隱藏 selection5。原版截圖先裁為 320×200，再用 ImageMagick
  `-scale` 整數放大至 640×400；不可用插值縮放製造差異。
- 重製端使用 `FD2_CAMP_NODE=town_ch06`、`FD2_SHOT_TOWN_STATE=selection,pulse`。
  原版 selection0–4 分別與重製 pulse `0,1,2,1,0` 對齊，五組 640×400 未遮罩整幀
  `compare -metric AE` 均為 `0`。對照圖
  [`town-variant2-five-selections-original-vs-remake.png`](../figures/town-variant2-five-selections-original-vs-remake.png)
  只作審查導覽，不取代 JSON 中的 raw frame hash。
- 這一輪只關閉 variant2 的正常五項畫面消費端 E2；variant1、variant2 selection5
  的該章 BIOS 掃描碼／後續 Enter、未修改一般玩家城鎮／戰後路徑與其他城鎮仍維持
  fail-closed。不得把這個修改 LOAD 結果寫成原生 `FD2.SAV` 完整相容或 23 個城鎮完成。

## 2026-08-09：ch02 城鎮 variant1 正常選項 E2（修改 LOAD 路徑）

- 以固定雜湊的原始 `FD2.EXE`／`FD2.SAV` 建立 Docker `/tmp` 研究副本；slot0
  只複製目前持久隊伍資料，將 raw chapter 設為 `0x0b`、roster count 設為 `4`、
  currency 設為 `0`。`tools/fd2save.py` 在 `fd2-cap-local` 中解碼出
  plaintext `0x59cb`、checksum `0x0025b9a9` 與 slot0 欄位；原始與研究副本雜湊及
  限制集中於 [`native_town_variant1_e2.json`](../data/native_town_variant1_e2.json)。
- 原版候選截圖由 `fd2-dosbox-screenshot-local` 取得，內容裁為 320×200，再以
  ImageMagick `-scale` 整數放大至 640×400；重製端以
  `FD2_CAMP_NODE=town_ch12`、`FD2_SHOT_TOWN_STATE=selection,pulse` 產生相同尺寸畫面。
- 五個正常選項的 exact-pixel 配對為 selection/pulse `0/1`、`1/2`、`2/1`、`3/0`、`4/1`，
  每組未遮罩整幀 `compare -metric AE` 均為 `0`。這些 pulse 只記錄本次擷取的相位，
  不推論原版計時器映射；並列圖見
  [`town-variant1-five-selections-original-vs-remake.png`](../figures/town-variant1-five-selections-original-vs-remake.png)。
- 本項只關閉 variant1 的五項畫面消費端 E2；selection5 的 BIOS 掃描碼／後續 Enter、
  未修改一般玩家城鎮／戰後路徑與其他城鎮仍 fail-closed。重新嘗試時若按鍵節奏未進入
  城鎮，不得把標題／載入畫面誤寫成原版證據。

## 2026-08-09：工作清單 AI 完成度斷言勘誤

- 盤點 `docs/knowledge-base/91-worklist.md` 時發現歷史快照把正規化
  `NextAIPlan`／`aiStep` 寫成「AI 行走+敵攻我演出」已完成，容易被誤讀為原版
  敵方 AI 已接通。該條現明示為 **normalized approximation、非 native parity**；
  現況 AI 段落也明確寫成「尚未接入原版 native AI 的 production planner」。
- `docs/knowledge-base/11-enemy-ai.md`、`docs/knowledge-base/42-re-vs-remake-gap-audit.md`
  與 `docs/knowledge-base/56-fd2-remake-sdd.md` 原本已保留相同的 fail-closed 限定，
  本勘誤不改變任何位址、候選橋或執行器語意；下一個真正解除閘門的證據仍是固定
  存檔中的 command／目標 trace 及正式回合消費端回歸。

## 2026-08-09：`0x14237` actor 物品來源閉合（E0）

- 合法 IDA Pro 9.4 在固定雜湊 `FD2.EXE` 上確認：`0x14288→0x1B83D(unit,0)`
  回傳 equipped 且 item `<0x80` 的 raw slot；`0x1429C→0x1B722(unit,slot)` 讀
  runtime record `+0x0B` 的 item ID；`0x142AA→0x4E56C(item)` 計算
  `0x602AD + item*0x17`。`0x142B2..0x142BE` 讀 item row `+0x0B/+0x0C`，
  這兩個 raw byte 才是 `0x14818` 的 actor-side caller input。完整非破壞性證據見
  [`fd2_ai_physical_item_source_ida.txt`](../data/ida/fd2_ai_physical_item_source_ida.txt)。
- 這條來源不是 `0x4E516` command-record table；先前工作清單把
  「`0x1B83D command record 來源`」寫得過寬，現已改成 item-source closure。
  新增 `battle.ResolveNativeAIPhysicalItemSource`，以完整 raw runtime record 與
  bounded item-row table 產生 detached row snapshot；無 equipped low item、缺 row
  或 bounds 不符時失敗即關閉。這仍不接 `0x14818`、物理 score、選中目標、移動、攻擊
  或 `NextAIPlan`，下一道 gate 仍是固定存檔的實際目標 trace。

## 2026-08-09：mode 2／`0x13FD4` 第二次控制流勘誤

- 重新以合法 IDA Pro 9.4 `idat -A -B`（DOS LE 線性位址）與 Docker Capstone
  交叉檢查 `sub_13A9F`、`sub_14237`。`0x13B5F` 呼叫 `0x14237` 後跳到
  `0x13B1E→0x13C06`；`0x13C06` 的零回傳分支在 `0x13C0F` 呼叫 `0x13FD4`。
- `sub_14237` 的函式尾端 `0x145C3` 以 `xor eax,eax` 回傳 0，無 equipped low item
  也直接跳到同一尾端。因此現有證據支持的 mode 2 失敗路徑是
  `0x14EF0→0x14237→0x13FD4`，不走 `0x13E9C`；不能用 `0x13FD4` 的 direct-xref
  清單沒有 `0x13B5F`，推論執行期不會抵達共用 `0x13C0F`。
- 先前「mode 2 不呼叫 `0x13FD4`」的勘誤本身已撤回；`PlanNativeUnitMode2`、測試、
  AI dossier、SDD、gap audit 與工作清單均改為保存這條共用基本區塊路徑。完整原始
  指令與 IDA／Capstone 工具版本、雜湊仍見
  [`fd2_ai_mode_dispatch_ida.txt`](../data/ida/fd2_ai_mode_dispatch_ida.txt)。

## 2026-08-09：mode 0／1 備援控制流資料化（E0）

- 依固定雜湊 `FD2.EXE` 的既有 IDA／Capstone `0x13A9F` 輸出，新增
  `fdother.PlanNativeUnitMode0`／`PlanNativeUnitMode1`。mode 0 保留
  `0x14EF0→0x14121→0x13E9C` 的巢狀失敗路徑，`0x13E9C` 回傳 0 才列入
  `0x13FD4`；mode 1 只列 `0x14EF0→0x14121`，零回傳才列 `0x13FD4`。
- 測試只比較位址級 action 順序與 caller-supplied 回傳旗標；不命名
  `0x14121`／`0x13E9C` 的玩法，也沒有接入 `NextAIPlan`。這是 E0 raw
  控制流切片，正式回合交易、畫面與一般玩家路徑仍未閉合。

## 2026-08-09：mode 3／4／5／7／9／10 raw 分支資料化（E0）

- 依固定雜湊 `FD2.EXE` 的既有 IDA／Capstone `0x13A9F` 指令，新增
  `PlanNativeUnitMode3/4/5/7/9/10`。mode 3／9 保留 `0x12C60` 的
  `-1` 與索引分支；mode 4／7／10 保留 `0x51A83` 清零、`0x12D7B`、
  `0x14B78`、`0x13FD4` 與 `0x32975` 的 raw 順序。
- mode 5 另以 `NativeUnitMode5Inputs` 保存 `[0x53AD5+ebp]`、
  `0x15DF3`／`0x14B78` 回傳、座標比對與 `0x53A55` type byte；抵達時只
  依 raw `<2`／`==0` gate 列出 `+0x31/+0x32`、`0x1BB8C`、
  `[0x53AD5+ebp]=1`、`0x25B45`、`0x12263`、`+0x34=7` writes/calls。
- 這批 helper 與 regression 只保存 E0 位址級 CFG，不接 `NextAIPlan`、
  正式回合交易或畫面；一般玩家的事件資料、動態回傳與完整 AI orchestration
  仍是下一道閘門。

## 2026-08-09：玩家近戰選單接入型別結算（E1 重製端）

- 檢查戰場 action menu 後，發現近戰確認仍呼叫舊的 `State.Attack`；該介面會
  消耗程序全域亂數，且丟棄完整 `AttackResult`。現已新增
  `Game.resolvePlayerPhysicalAttack`，改由注入的遊戲亂數呼叫
  `AttackWithRNG`，並將未命中／暴擊／傷害／經驗保留到訊息與演出橋。
- 缺少亂數來源時會在任何狀態變更前失敗即關閉；Docker 回歸涵蓋確定命中、缺少
  亂數不 mutation，以及訊息保留未命中／暴擊／經驗。這只補重製端「選單→型別
  結算→演出」垂直切片，不宣稱原版攻擊 ABI、完整劍技／命中表或一般玩家 E2。

## 2026-08-09：正規化敵方演出共用遊戲亂數邊界

- `aiStep` 原本仍呼叫 `State.Attack` 的程序全域亂數；現改與玩家近戰共用
  `resolvePhysicalAttack`／`AttackWithRNG`，固定 `FD2_SEED` 時敵方正規化演出也可
  重現，並保留未命中／暴擊／經驗訊息。
- 這只是重製端的一致性修正；`NextAIPlan`、目標選擇與回合 orchestration 仍是
  normalized approximation，沒有把這次改動描述成原版 native AI 接線。亂數來源
缺失時 `aiStep` 停止且不標記單位已行動，維持失敗即關閉。

## 2026-08-09：完整 native 戰場 frame 疊加 action overlay（E1 重製端）

- Docker／Xvfb 以目前 source、唯讀 `FDOTHER.DAT` 與 `battle_ch01` 實際抓圖，發現
  ring／native command grid 原本被 `drawNativeMapFrame` admission 排除；在短地圖上
  只畫 normalized map，畫面下方出現大片黑帶。
- 現已讓 ring／native command grid 在完整 native map frame 成功時疊加，缺資源仍
  使用既有回退。新增 admission regression 與
  [`action-overlay-native-remake-fullframe.png`](../figures/action-overlay-native-remake-fullframe.png)
（SHA-256 `4402adbb6f1ddff94639ae594b392bf96c68a264f9f05126f3e4f022d74a7852`）。這是
重製端 E1 renderer 接線，不是原版 DOSBox 逐像素差分。

## 2026-08-09：戰後結果輸入至城鎮的 runtime 邊界（E1 重製端）

補上 `confirmBattleResult` production seam，並由 Docker／Xvfb 真實回歸驗證：
`endTurn → aiStep → finishTurn → completeTurn → checkResult` 完成敵方回合後，
結果仍停在 battle；玩家按 Enter 才進入含 `sync_party` 的 postbattle cutscene，
淡出完成後才進 town。測試也確認持續隊伍快照已寫入且不殘留舊結果覆蓋。

這只是可編輯流程的 E1 垂直切片，沒有把 fixture 的空敵方計畫當成原版 AI 證據，
也沒有解除逐章 handler、一般玩家 DOSBox E2 或完整 town/shop/preparation/save
驗收門檻。

## 2026-08-09：AI `0x1DEBE` caller-specific raw gate 證據補檔（E0）

以使用者合法 IDA Pro 9.4 Docker 與 `fd2-cap-local` Capstone 重新核對固定版
`FD2.EXE`：`sub_1DEBE` 位於 `0x1debe..0x1df58`，唯一直接 caller 是
`0x14496`。IDA／Capstone 共同確認 `+0x26==0`、候選座標曼哈頓距離恰為一、
`sub_1B83D(a1,0)` 找到 equipped raw slot，且 item row `+0x0b<=1` 時回傳 1；
caller 才把兩個 caller-owned raw word 差值加入 score。原始運算元與未知高階語意
均保留，未接 normalized `NextAIPlan`。完整輸入指紋、工具版本與直接指令見
[`fd2_ai_physical_score_ida.txt`](../data/ida/fd2_ai_physical_score_ida.txt)。

## 2026-08-09：戰場 GUI 對照圖差異勘誤（E1；未解除）

重新檢視 GitHub 首頁上的戰場圖片後，確認
`native-map-ch01-original-video.png`（320×200，原版參考）與
`native-map-ch01-remake.png`（640×400，重製端 E1）不是同一個戰場狀態；原版
可見的單位與左下 HUD，在重製圖中的像素內容仍不同。較早 E1 已保存 raw
相機／游標欄位，但不等於整幀畫面一致。這兩張圖只能
作為資源消費與排版參考，不能再被描述成「相近相機已對齊」或原版 E2。

尺寸、雜湊、可見觀察與下一道驗收門檻已保存於
[`battle-visual-gap-ch01.json`](../data/ui-traces/battle-visual-gap-ch01.json)，
並同步修正 README、SDD、介面證據矩陣與工作清單。下一步必須用同一
`FD2.SAV`、相機／游標／回合／單位狀態，取得原版 DOSBox 與重製端逐幀配對，
再分辨狀態橋接、相機、單位合成、HUD 或調色盤的實際差異；三平台套件與推廣影片
發布閘門在此之前保持未解除。

## 2026-08-10：正式序章 handler 戰場截圖橋接（E1；未解除）

為釐清 GitHub 戰場圖中的「只剩一名角色」問題，新增僅限
`FD2_SHOT=...` 且 `FD2_SHOT_FAST_FORWARD=1` 的截圖快速時鐘。它仍由
`story_ch00_handler` 的既有 BeatRunner 執行 73 拍，逐拍保留 LOADCH、兩次
`spawn_intro`、四次 JOIN、`reset_pose`、`focus_unit` 與最後的 `battle_ch01`
handoff；對白按頁消費，原生 present job 也逐步執行，不改一般玩家路徑。

Docker／Xvfb 真實命令產生新的正式 handler 截圖：
[`native-map-ch01-remake-handler.png`](../figures/native-map-ch01-remake-handler.png)，
640×400，MD5 `851781caf1bcff736e24a0bc57d39372`、SHA-256
`da9cae2827d027f933e47ab2c4e846c990babc2f856c449f699fdba0794d524b`。它確實顯示
完整多單位戰場；但與儲存庫的原版 320×200 錄影片格仍有單位位置、游標、HUD 與
尺度差異。舊 `native-map-ch01-remake.png` 保留為直接節點除錯入口的歷史證據，
不再作正式比較圖。

目前仍只能判定為 E1 消費端與狀態橋接證據；同一 `FD2.SAV`、相機／游標／回合／
單位狀態的原版 DOSBox 與重製逐幀配對尚未取得。三平台套件與推廣影片閘門維持
未解除。

## 2026-08-10：FDICON map-selector 原始來源勘誤（已證實；取代先前 b0 說法）

使用固定版 `FD2.EXE`（大小 357074、MD5
`b97caf2239a27a896069d03549d96e1e`、SHA-256
`222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f`）在合法
`fd2-ida-authorized-local:latest` Docker 映像以 IDA Pro 9.4／Hex-Rays
`9.4.0.260610` 重新判讀。`0x10c50` 的直接欄位關係是：FDFIELD row
`b1` 傳給 `0x11019`，回傳 cache slot 寫入 runtime `unit+2`；FDFIELD
`b0` 另寫入 runtime `+6`；同一 b1 再寫入 `+7/+8`。`0x1088d` 的玩家迴圈
則以 persistent record `+7` 傳給 `0x11019`。完整逐位址摘要見
[`fd2_fdicon_selector_constructor_ida.txt`](../data/ida/fd2_fdicon_selector_constructor_ida.txt)。

因此先前「scripted FDFIELD b0→`0x11019`→`unit+2`」及「map0 keys
`[0,1,2]` 等於陣營碼」是錯誤斷言；保留該歷史記錄，但不再作現況依據。
`tools/parse_field.py`、`export_units.py` 與 selector synchronizer 已改為 b1，
並以明確 `--rewrite-map-selector-key` 只遷移 33 張可編輯地圖的 1886 筆 stale
欄位。Docker `--check` 與 1887/1887 原始列比對通過，所有檔案仍為目前使用者
UID/GID。這修正可解釋 GitHub 戰場圖中敵人被畫成鍵 0 藍色圖像的高可信度原因；
仍須重新抓取同狀態原版／重製畫面，不能因此宣稱逐像素一致或解除三平台發布閘門。

## 2026-08-10：戰場截圖以原版資源重抓（E1；取代上一節舊圖雜湊）

上一節後，重新檢查截圖命令發現若沒有設定
`FD2_ORIGINAL_FDOTHER`，正式序章會在 `spawn_intro` 的原生基準輸入失敗；舊截圖
鉤子仍會把只剩地形的錯誤狀態寫成 PNG。現已在 Docker／Xvfb 中以唯讀
`org_game/炎龍騎士團/FLAME2/FDOTHER.DAT`（同目錄 `FDSHAP.DAT`、`FDICON.B24`）
重抓，並套用 b1 selector 遷移後的 1887/1887 地圖資料；截圖命令另設定
`FD2_SHOT_DETERMINISTIC=1`，以固定 60 Hz 虛擬時鐘鎖定動畫幀，兩次 Docker 執行
的 PNG 已做位元組級比對相同。

目前 GitHub 圖片 [`native-map-ch01-remake-handler.png`](../figures/native-map-ch01-remake-handler.png)
的 MD5 為 `6e200d3bbd48782f4248a6a8ced48343`、SHA-256 為
`8f2e948995350d64c53267c55e71a858eef34db9e6cb9f22577327007af8813c`（640×400）。
這張圖已不再把舊 b0 映射的敵軍誤畫成友軍圖像，但仍與 320×200 原版參考不是同一
狀態；上一節的 `851781...`／`da9cae...` 雜湊保留為歷史產物，不再代表目前首頁
圖片。截圖鉤子也改為錯誤狀態不產生證據，避免再次把失敗即關閉的畫面誤列為正式圖。
這仍是 E1，不解除同狀態 DOSBox／重製逐幀配對或三平台發布閘門。

## 2026-08-10：map HUD terrain icon row-5 位址勘誤（已證實；修正戰場差異）

以同一固定版 `FD2.EXE` 在合法 IDA Pro 9.4 Docker 重新讀取 `sub_1ACF3`
（0x1acf3..0x1aeb1）。0x1adb5..0x1adc4 直接把 `edi`（native stride）乘 5，
加上 panel base `ebp` 與 6，再呼叫 `0x4DEDA`；因此 FDSHAP terrain icon 的 raw
目的地是 `base + stride*5 + 6`。0x1ae74..0x1ae86 對 optional FDICON unit
icon 使用相同目的地。舊版重製端只把 terrain icon 寫在 `base+6`，造成 GitHub
戰場圖左下圖示向上偏移約 5 個原生列。

已修正 `fdicon.NativeMapHUDLayoutFor`、layout regression 與 SDD／worklist，
完整 IDA 指令與檔案指紋保存於 [`fd2_map_hud_geometry_ida.txt`](../data/ida/fd2_map_hud_geometry_ida.txt)。
這是位址已證實的 renderer bridge 修正，不等於其他章節或整體 GUI E2 完成。

同輪以 Docker DOSBox/Xvfb 從 `FD2.SAV`（大小 22987、MD5
`409795ccebc2af340d5c74152c2d471c`、SHA-256
`6d14f2c22562cabca83725084f1a9d6539a1d4066da5c1debcdadb446812691f`）的 CONTINUE
狀態取得原版 320×200 oracle，加入
[`native-map-ch01-original-fd2sav.png`](../figures/native-map-ch01-original-fd2sav.png)。
修正後重製圖 MD5 `faa28a76c93146684c40eee55ac82a3e`、SHA-256
`10cd82b22aa3637fa230f1fcd12146ecae1b8d93fe4055cdefaae85c05bcaeb4`；最近鄰縮放
到 320×200 後用 Docker ImageMagick `compare` 得 AE=22、RMSE=41.2403，差異只
剩左下畫布邊界。這個結果是 ch01 單一 FD2.SAV 狀態的 E2 範圍候選，不能外推
到其他章節、一般玩家 CONTINUE 或完整操作界面；完整欄位與方法見
[`battle-visual-gap-ch01.json`](../data/ui-traces/battle-visual-gap-ch01.json)。

## 2026-08-10：戰鬥狀態欄姓名改接 FDOTHER#4（E1；未解除）

重新檢查 GitHub 戰鬥演出圖後，確認另一項可重現的差異：重製端狀態欄姓名仍走
現代 28 像素字型，原版 `0x18c6d→0x15f84→0x4ea2a` 則消費
FDOTHER#4 的 16×16 1bpp glyph。現已以可版控的 `FDOTHER_004.bin` 與
`unicode_to_glyph.json` 接入重製端，保留未知字元的既有 fallback，不猜測 glyph
索引；測試也固定確認前景／陰影寫入與未知字元拒絕 native 路徑。

Docker／Xvfb 產生的 E1 證據圖為
[`battle-native-name-remake.png`](../figures/battle-native-name-remake.png)，完整
檔案指紋、原版雜湊、呼叫鏈與命令見
[`battle-name-glyph-ch01.json`](../data/ui-traces/battle-name-glyph-ch01.json)。
這只關閉狀態欄姓名字形的重製端消費差異；攻擊幀時序、傷害／閃紅、台座、完整
指令介面及一般玩家同狀態 DOSBox E2 仍未驗收，三平台套件與推廣影片閘門維持
未解除。

## 2026-08-10：戰場全畫面紅罩勘誤與命中色盤證據

針對 GitHub 戰鬥演出圖的「命中時整個背景與狀態欄泛紅」偏差，先以合法 IDA Pro
9.4 重新追 `0x28a6c→0x2939d`，再用 Docker `fd2-cap-local` Capstone 5.0.3
交叉驗證。原版只有在 frame `+4 == 1`、傷害步進完成、`0x29f72` 兩個未命名
原始輸出欄位非零時，才寫入索引 0 的短暫 DAC 脈衝（第一段 RGB
`1,0x20,0`；第二段 `0x3f,0x3f,0x3f`，各含 20/40 毫秒等待），每個子幀另有
`0x17aa9(1)` 等待。完整證據與雜湊見
[`fd2_battle_impact_pulse_ida.txt`](../data/ida/fd2_battle_impact_pulse_ida.txt)。

目前重製端沒有這些 raw 欄位的可追溯來源，已移除不受證據支持的 RGBA 全畫面
紅罩；另依 `orig_05_attack_03_impact.png` 保留守方紅剪影、讓攻方維持 FIGANI
原色。這修正可見的戰場畫面偏差，但不宣稱已完成
DAC runtime bridge、攻擊全時序、逐像素一致或一般玩家 E2。逐幀截圖鉤子另已修正
為 `FD2_SHOT_SERIES` 單獨啟用時也建立攻擊演出，供後續比較使用。修正後代表性
畫面為 [`battle-impact-no-global-tint.png`](../figures/battle-impact-no-global-tint.png)，
擷取雜湊與條件見 [`battle-impact-no-global-tint.json`](../data/ui-traces/battle-impact-no-global-tint.json)。

## 2026-08-10：命中幀 HP 邊界與戰場對照圖更新（E1；未解除）

使用 Docker／Xvfb 以目前 source 重新抓取 `FD2_SHOT_ATTACK=4` 的逐幀序列，並與
原版 `orig_05_attack_03_impact.png` 對照。影像證實原版在 impact 開始時立即顯示
post-hit HP；重製端原先 8 tick 緩降造成中間值，現以 `battleImpactHP` 在同一邊界
切換 `defHP0→defHP1`，並補 `TestBattleImpactHPCommitsPostHitValueAtImpactBoundary`。
守方 E1 剪影色值由原先 `(208,16,16)` 改為原版擷取主色 `(190,0,0)`；這仍不是
`0x2939d` DAC runtime adapter。

目前首頁已改展示
[`battle-impact-compare-20260810.png`](../figures/battle-impact-compare-20260810.png)：
左原版正規化、中重製、右逐 RGB 差異遮罩。原版 640×480 以垂直 2.4 倍取樣、重製
640×400 以 2 倍最近鄰縮小為 320×200，固定 fixture 仍有 3933 個差異像素；舊
`battle_restore.gif` 保留為歷史證據，不再代表目前 source。`battle-impact-no-global-tint.png`
已更新為修正後 frame 77，MD5 `15b4e22f09a732abcb0caf32daaa565d`、SHA-256
`58423664e32ebddff09768b191a059b15c3d0595462ee8ec4feb1584d7b599fe`。
實際 targeted/full regression、Markdown/JSON/影像雜湊與 Docker 清理仍是提交前必要驗證；
原始 DAC 欄位、完整攻擊時序及一般玩家 E2／三平台發布閘門維持未解除。

同輪另修正守方 FIGANI 待機幀：原先的固定 `(prog/6)` 不是原版可證實的時序，
現以每個守方資源 descriptor `+6` 與 `FD2_BATTLE_FPT` 的純排程橋選幀；攻守任一
延遲表不完整即失敗即關閉。這個選幀管線修正未改變已記錄的 frame 77 雜湊，故
現有 3933 差異像素對照仍有效；它不解除 raw DAC、完整攻擊時序或一般玩家 E2。

## 2026-08-10：戰場首頁比較圖改為同狀態三欄證據（E2 範圍候選）

針對 GitHub 首頁仍容易把「不同狀態歷史參考圖」誤讀成渲染器錯位的問題，保留原
兩張歷史圖片不刪除，但新增 [`battle-field-ch01-scoped-compare-20260810.png`](../figures/battle-field-ch01-scoped-compare-20260810.png)。
左欄是同一 `FD2.SAV` 原版 320×200，中央是同狀態重製端 640×400 最近鄰縮小結果，
右欄是差異遮罩；Docker ImageMagick 驗證兩個來源面板均為 AE=0，遮罩只包含 22
個低幅度像素，座標集中在左下畫布邊界 `x=4..45、y=185..195`。因此目前可證實
的是 ch01 單一狀態的內容層已對齊，邊界差異成因仍未知；其他章節、一般玩家
CONTINUE、操作覆蓋層與完整戰場 E2 仍未解除。

### 同日組版勘誤

初版三欄 PNG 的中欄曾因組版命令錯誤而重複左欄；這是證據圖片問題，不是重製端
渲染器結論。已在 Docker ImageMagick 內重新產生，現在左欄與原版來源 AE=0，中欄
與 640×400 重製端最近鄰縮放 AE=0，原版／重製內容比較仍為 AE=22，右欄紅色遮罩
精確 22 像素。修正版檔案 MD5 為 `0506c67ad58728c6fcaf45be6f12a432`、SHA-256
為 `3b3ab913fd0630ce37a80f78476edba0bcacb8e0636cc919dc90e7a09ddd1174`；後續引用
應以此雜湊與 `battle-visual-gap-ch01.json` 為準。

## 2026-08-10：敵方 AI mode 2 窄執行期切片、祕密商店矩陣與終局音訊邊界

本輪先按優先順序處理敵方人工智慧。合法 IDA Pro 9.4／Docker Capstone 已固定
`0x13A9F` 的 mode dispatch 與 `0x14237` 物理候選消費邊界；重製端新增
`battle.State.BindNativeMovementCostRows`，在每個 battle state 綁定版本化
`0x4e555` 29×20 原始移動列（raw movement rows），並讓 `NextAIPlan` 僅在 mode 2、
原始選擇碼（selector）、完整 FDFIELD 地形／組成來源、物品幾何、`0x1DEBE` 與
`0x14237` 評分輸入全部存在時產生物理路徑／目標。缺任何 provenance 會回傳
`NativeError`，`Game.aiStep` 停止且不消耗行動，不再把缺資料靜默換成另一套
正規化（normalized）目標選擇。Docker battle regression 已涵蓋成功候選、路徑與缺移動表
失敗即關閉；這是 E1 窄切片，不是完整敵方 AI。`0x14EF0` 前置選擇、其他 mode、
命令／法術／物品交易、`0x1548E` 原生演出與一般玩家 E2 仍待下一輪 raw trace。

祕密商店不是共用按鍵。`campaign_full.json` 現保存 23 個城鎮的可編輯
`native_secret_gate`（可見選項、BIOS 掃描碼（scan code）、祕密商店節點）；
ch02 與其餘章節的 selection／scan 差異以
`TestCampaignFullNativeTownSecretGatesAreChapterSpecific` 固定，Runner 仍維持
「精確組合鍵揭露 selection 5 → 再次確認」兩步邊界。這解除「ch02 祕密店入口未
資料化」的過時說法，但還不等於 23 章未修改一般玩家 DOSBox E2。

音訊接線改以原版證據為準：戰鬥節點以 `0x51e63` 30-entry chapter table，城鎮／
商店以 `FDMUS_010`；新增 campaign regression 防止曲目退回單一猜測值。沒有
已證實終局曲目的 generic `ending` 會經既有 `play_bgm(-1)` 停止前一曲，
`campaign_full.json` 的終局文字也移除生成器狀態尾註，改為可編輯的玩家可見結語。
原版 `0x2BCE5` indexed renderer、文字閘門後 montage、確切終局音樂與正式
campaign handoff 仍失敗即關閉；`FD2_ENDING_PREFIX` 預覽不可當成通關完成。

## 2026-08-10：AI 其他模式、0x14EF0 消費端與結局精確音訊補證

本次接手保留上一節的時間序列，不覆寫舊結論；以下是新證據及其消費邊界。

- 新增 [`fd2_ai_mode5_event_ida.txt`](../data/ida/fd2_ai_mode5_event_ida.txt)：
  IDA 直接固定 `0x15DF3` 對 mutable `0x53A51` 的寬高／四位元組 cell
  row-major 搜尋、terrain flag `0x20`、event low5 first match；並固定
  mode 5 field-control row、`+0x31..+0x33`、inventory writer、`0x53AD5`
  state、`0x12263` map update 與整個 `+0x34=7` 寫入。重製端由 map JSON
  的 raw tile／event／blit bytes 重建同形狀 buffer，且在任何 state mutation
  前驗證整張 grid，避免 malformed later cell 留下半套交易。
- 新增 [`fd2_ai_mode11_13fd4_ida.txt`](../data/ida/fd2_ai_mode11_13fd4_ida.txt)：
  固定 `0x13FD4` current/max HP、`max/5`、`+0x25/+0x26` raw gate、
  `0x51A83` 暫存寫入，以及 mode 11 的 `[0x53C23] >= 6`、
  `[0x53C4F] >= 6` 雙 gate。mode 11 的雙動作 transaction 與 `0x13FD4`
  presentation owner 尚未證實，因此重製端仍 fail-closed，不以正規化回血或
  下一步攻擊代替。
- `0x14EF0` runtime consumer 現已把 `0x14237` 的 raw priority（8／18）
  與獨立 score word 分開，依 raw `+0x34` bit、actor／target `+0x48/+0x4A`
  及 command word 走 `0x1548E`／`0x15311`／`0x15055`；mode 3／9 的
  `+0x35→+0x08` lookup 與 mode 5 state tail 有 focused Docker regression。
  未知 command／item relocation、mode 11、mode 5 raw sample audio、一般玩家
  回合 E2 仍未關閉；本節較早的「indexed presentation」暫稱已由末尾勘誤撤回。
- 結局音訊證據 `fd2_ending_audio_ida.txt` 已加入三筆精確事件：
  `0x2C5CF→FDMUS_004`、`0x2C1AC→play_bgm(-1)`、`0x2C1F5→FDMUS_018`。
  `internal/ending` timeline／預覽器會依序驗證並消費這三筆；這不等於完整
  `0x2BCE5` indexed montage、輸入交接或正式 campaign ending 已完成。

本輪聚焦 Docker 測試通過：mode 5 raw grid／state tail、mode 3／9 raw lookup、
0x14EF0 command／item route、所有 raw tail table cases。未修改原版 DOSBox
動態 AI trace 與終局逐幀／逐音訊比對仍是下一個證據門檻。

## 2026-08-10：mode 11 雙閘門的純 E0 選擇器補證

在上一節已提交的窄消費端之外，新增
`remake/internal/battle/native_ai_mode11.go` 與對應測試。它保存 IDA 已證實的
獨立條件：`[0x53C23] >= 6` 才選 `0x15311`，`0x14237` 之後無條件保留第二階段，
再由 `[0x53C4F] >= 6` 選 `0x1548E`，否則選 `0x14121`。`0x14121` 使用獨立
mode 11 路由型別，明確表示它不是 `0x14EF0` 尾端。

Docker focused regression 已通過四種分數組合與缺少任一 raw score 的失敗即關閉。
這個選擇器沒有寫入戰鬥狀態，也沒有假接 transaction owner、命令／法術／物品
執行、`0x13FD4` 回復演出或 mode 5 raw sample audio；較早的
`0x25B45`／`0x17AA9` indexed presentation 暫稱已由末尾勘誤撤回；
因此仍只提升 mode 11 的 E0 靜態路由證據，不解除完整敵方回合或原版一般玩家 E2。

## 2026-08-10：0x13FD4 state-only recovery decision 補證

將既有 `ApplyNativeAIIdleRecovery` 拆成兩層：
`PlanNativeAIIdleRecovery` 先在 detached raw record 上讀取 `+0x40/+0x42`、
`+0x25/+0x26`，只輸出接受／拒絕、目前／最大／下一個 HP 與原始 gate bytes，
不改寫輸入；提交 wrapper 再以同一快照寫回 `+0x40`。Docker regression 已
驗證 current==max、任一 raw gate 非零、`max/5` 與封頂，以及純函式不變更
record。這補的是 `0x13FD4` state-only E0，仍沒有把 `0x12D7B` 等候、畫面
演出、mode 2／11 caller handoff 或結局／AI 一般玩家路徑接入正式執行器。

## 2026-08-10：mode 11 完整直接控制流重檢（E0）

本輪以合法 IDA Pro 9.4 優先、Docker Capstone 5.0.3 交叉驗證目前雜湊固定的
`FD2.EXE`（357074 位元組、MD5 `b97caf2239a27a896069d03549d96e1e`、
SHA-256 `222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f`）。
完整位址、工具與輸出位置見
[`fd2_ai_mode11_full_ida_20260810.txt`](../data/ida/fd2_ai_mode11_full_ida_20260810.txt)。

- `0x13A9F` 的 mode 11 分支從 `0x13E02` 直接呼叫 `0x1598A`，不是先經
  `0x14EF0`；之後無條件呼叫 `0x14237`。
- `[0x53C23] >= 6` 只控制 `0x15311`；`[0x53C4F] >= 6` 只控制
  `0x1548E`，不足才走 `0x14121`，且其零回傳才呼叫 `0x13FD4`。
- IDA direct callers 固定 `0x1598A`（`0x13E09`、`0x14F15`、`0x1D91A`）、
  `0x14237`（`0x13B5F`、`0x13E26`、`0x14F0B`）、`0x15311`（`0x13E1C`、
  `0x14FDB`）與 `0x1548E`（`0x13E39`、`0x14F9B`）；這些 caller 不可互併成
  單一 dispatcher。
- 重製端目前只保存兩個 raw gate 的無副作用 E0 路由，尚未假接 transaction
  owner、指令／法術／物品效果或演出；仍採失敗即關閉。`0x13FD4` 的 HP 算術
  仍只是 state-only 契約，後續會單獨補證其 presentation 邊界。

## 2026-08-10：`0x13FD4` 完整 presentation 邊界重檢（E0）

合法 IDA Pro 9.4 與 Docker Capstone 固定 `0x13FD4..0x14120`：接受路徑在
`max/5` HP 寫回前依序執行三次 `0x17AA9(1)`、兩次 `0x1DA16`（raw 24×24
解碼）與兩次 `0x11EB0`（312×192、來源步幅456／目的步幅320）拷貝；第一段
`0x1DA16` 的 raw 參數是 `2,0xFD`，第二段是 `0,0`。`0x12D7B` 只把 actor
record 前兩個座標位元組交給 `0x12CEA`，後者依 `[0x53AB1]`／`[0x53AB5]`
更新座標並受 `[0x51A83]` 等候閘門影響，尚不足以命名成移動或休息。

IDA direct callers 仍是 `0x13BC5`、`0x13C0F`、`0x13C84`、`0x13D58`、
`0x13E52` 與 `0x19082`；後者只有在未命名原始 `a3==0` 時呼叫。完整 raw
指令、helper 邊界與雜湊見
[`fd2_ai_13fd4_full_ida_20260810.txt`](../data/ida/fd2_ai_13fd4_full_ida_20260810.txt)。
重製端只保留 `PlanNativeAIIdleRecovery`／`ApplyNativeAIIdleRecovery` 的
state-only 契約，尚未把影格、色彩、音效或 renderer 接入正式路徑；未知部分
維持失敗即關閉。

## 2026-08-10：mode 5 音訊邊界勘誤與完整 raw tail（E0）

重新以合法 IDA Pro 9.4／Docker Capstone 核對 `0x13C19..0x13D24`、
`0x15DF3..0x15E6C` 與 `0x25B45..0x25BF3` 後，撤回先前把
`0x25B45／0x17AA9` 稱為「mode 5 indexed presentation」的說法：

- mode 5 在抵達事件格後直接呼叫 `0x25B45([0x53EE8], 12, 1)`；IDA 的
  `0x391D1`、`0x39344`、`0x3975E`、`0x39448` 字串／呼叫鏈證實這是
  AIL sample 的 stop、init、address、loop-count、start 序列。重製端現以
  `NativeAIMode5AudioCue` 將同一 FDOTHER #31 導出的 `sfx_12.wav` 接到
  raw 播放邊界，缺樣本時失敗即關閉；sample 的遊戲名稱仍未知。
- `0x17AA9` 不在 mode 5 的 direct caller 清單；它在 `0x13FD4` 的三段
  等候與其他 raw presentation 路徑出現，不能與 mode 5 音訊併稱。
- `0x15DF3` 的 return `0` 才代表第一個 row-major 事件格命中，`-1` 代表
  無命中；`0x12263` 只做整張 mutable map 的 state word 增加與 event byte 清零。
- `PlanNativeUnitMode5`／`ApplyNativeAIMode5Event` 已保留事件格、field-control
  row、`0x1BB8C`、`0x53AD5`、`0x12263`、完整 `record+0x34=7` state tail，
  並在 callback 中消費 raw sample tuple；缺樣本時正式執行路徑仍失敗即關閉。
  完整證據見
  [`fd2_ai_mode5_full_ida_20260810.txt`](../data/ida/fd2_ai_mode5_full_ida_20260810.txt)。

## 2026-08-11：mode 11／`0x13FD4` 交易邊界與未修改原版玩家回合

本輪接續上一輪中斷的 AI 工作，先核對 worktree 與固定雜湊，沒有覆蓋既有
未提交變更。新增內容如下：

- `remake/internal/battle/native_ai_mode11.go` 的
  `NativeAIMode11Stage`／`Stages`／`ExecuteNativeAIMode11Transaction` 保存
  `0x13E02` 之後的兩段 raw route 順序。第一段由 `[0x53C23] >= 6` 選
  `0x15311`；第二段無條件執行，由 `[0x53C4F] >= 6` 選 `0x1548E`，否則選
  `0x14121`。回呼失敗會停止後續 stage；未替 route 猜 command、物理、法術、
  道具或移動語意。
- `remake/internal/battle/native_ai_idle_recovery.go` 的
  `NativeAIIdleRecoveryPresentation` 保存 `0x13FD4` 的 sample／解碼／拷貝／
  wait raw 參數；`ApplyNativeAIIdleRecoveryWithPresentation` 只有在 callback
  成功、record 未被改動且重新 preflight 相同時才寫入 `+0x40`，其餘情況回復
  快照並失敗即關閉。完整位址與參數仍見
  [`fd2_ai_13fd4_full_ida_20260810.txt`](../data/ida/fd2_ai_13fd4_full_ida_20260810.txt)。
- Docker focused regression 實際通過：
  `go test ./internal/battle -run 'NativeAIMode11|NativeAIIdleRecovery' -count=1`。
  這只是 raw transaction E0／E1，不是正式敵方 AI 回合完成；缺少
  `0x15311`／`0x1548E`／`0x14121` 的完整消費 owner、indexed framebuffer、
  音訊 sample 與 mode 11→`0x13FD4` caller handoff 時，正式路徑仍停止。
- 未修改原版的一般玩家驗證已在 Docker DOSBox 沙箱完成：保持
  `FD2.EXE`／`FD2.SAV` 原始雜湊，從標題與開場對話走到第一戰第一個我方單位的
  玩家指令格，保存 320×200 PNG、輸入時間線與雜湊於
  [`native-player-turn-original.json`](../data/ui-traces/native-player-turn-original.json)。
  這是玩家回合 E2 原版錨點；尚未取得同一 raw 狀態的重製端對照，且敵方 AI
  回合／攻擊演出仍未完成。

下一輪應先用同一 raw runtime 狀態建立重製端玩家指令格對照，再取得可回查的
mode 11 實際 producer／consumer trace；在此之前不可把本輪交易契約升格成完整
AI 或逐像素 UI parity。Docker 工作皆使用一次性容器；本輪結束要確認 FD2
容器、臨時 DOSBox 沙箱與 `/tmp/fd2cap` 均已清理。

## 2026-08-11 追加勘誤：本輪已完成 E1 owner

上段是中斷前的交接狀態，不能覆蓋本輪實作結果。現況如下：

- `NextAIPlan` 已在 raw mode 11 且來源完整時建立兩段 stage；
  `native_ai_mode11_execute.go` 以 continuation 實際消費 `0x15311`（command／
  item）與 `0x1548E`（physical／FIGANI），`0x14121` 無 blocked cell 時才交給
  `0x13FD4`，中間不重新選下一個單位。
- `native_ai_idle_recovery.go` 已成為 `0x13FD4` 的 indexed／音訊 owner，驗證
  `[0x53EEC]` index `4`／loop `1`、FDICON、DAC、312×192 copy 與三次 Draw
  確認後才提交 HP；任一資源、raw tuple 或 record 變動會停止並回復 range。
- 這只是 E1 原始證據到可見窄切片的完成，不是原版逐幀／逐音訊一致，也不是
  一般玩家敵方回合 E2。index 4 高階音色、decode mode 玩法語意、未知命令／法術／
  物品演出與其他 caller 的一般玩家 trace 仍需後續補證。對應工作清單已更新於
  [`91-worklist.md`](91-worklist.md)。

## 2026-08-11：玩家第22戰 ch21 post 正式 E1 戰役邊界

本輪先稽核工作清單，撤回 mode 11／`0x13FD4`、主選單與敵方 AI runtime
仍完全未接的過時字句；未具備原版證據或消費端的項目仍保留未完成。接著完成
玩家第22戰戰後到整備的最小戰役切片：

- `remake/assets/cutscenes/bindings/ch21_post.json` 以 raw `0x244b6`／
  `0x24512`／`0x245ce`／`0x24618`／`0x1f882` 為定位，正式接入三段對話、
  ACTING 65／66、兩次 PAN、indexed transition、`sync_party` 與
  `set_chapter(22)`；只接受能物化 raw layout 所需 slot72 的 73 或79槽。
- `ch22.json` 啟用 `runtime_append_groups`，以 group0 起始並依原始 map21
  group1／2／3 形成 66→72→73→79 frontier；缺少 group2 或尚未物化的
  66／72-slot 狀態會停止，不以猜測補齊演出。
- `campaign_full.json` 的 `postbattle_ch22_persist` 現正式指向該 binding，
  完成後進入既有 `preparation_ch23`，不是直接跳下一場戰鬥。Docker／Xvfb
  回歸實際消費 group1+2 與 group1+2+3、持久隊伍同步及整備節點。

這是原始證據到重製 runtime 的 E1 窄切片，不是未修改一般玩家 DOSBox E2，
也不代表第23／24／25／29戰已完成。下一步應從最小可重現的戰役節點繼續，
並維持城鎮／商店／整備／存檔邊界與失敗即關閉規則。

## 2026-08-11：ch22_pre LOADCH 視圖來源已補證並接入正式戰役

本輪完成玩家第23戰戰前 `ch22_pre` 的原始游標／視圖來源窄切片。先以合法
IDA Pro 9.4 Docker 重新讀取固定雜湊 `FD2.EXE`，再以 Docker＋Xvfb 執行真實
回歸；沒有使用主機 Capstone、主機 Python 或未鎖定虛擬環境。

### 已證實

- `0x205da` 在 `0x1088d` 後將 `[0x53AA9]`／`[0x53AAD]`、
  `[0x53AB1]`／`[0x53AB5]`、`[0x53AB9]`／`[0x53ABD]` 全部清為零，
  再呼叫 `0x11CAC(1)`。
- `0x135dd` 只同步更新鏡頭與絕對游標；沒有寫入可見游標。故第一次
  PAN `(14,32)` 後，`0x336e5` 的 `[0x53AB9]+6`／`[0x53ABD]+5`
  是 indexed tile `(0,5)`，不是 `(6,5)`。
- `Game` 現以場景專用 `storyNativeMapView` 保存六個原始視圖全域，
  `syncStoryNativeMapPanView` 每個 tile-step 驗證鏡頭／絕對游標身份與地圖邊界。
  `handlerRuntimeSlotCount` 會在 runtime_context 位於 LOADCH 前時驗證下一個
  LOADCH 的 70 筆宣告，不拿前一戰 66 槽誤判。
- 正式 `story_ch23` 已使用 `ch22_pre.json`；正式 regression 驗證 70 筆 roster、
  目前整備選出的 16 人、16 次停用、三段 PAN、indexed transition，並到達
  `battle_ch23`。因 `ch23.json` 沒有 `runtime_append_groups` 明確契約，
  handoff 只走正式戰場重建，沒有猜測 handler 陣列與 battle state 共享。

### 實際驗證

在 `/home/anr2/cht/fd2` 使用一次性 Docker＋Xvfb（原版 `FLAME2` 唯讀掛載）執行：

```text
go test -v ./cmd/fd2 -run TestChapter22PreHandlerReachesBattle23WithLoadCHView -count=1
go test -v ./cmd/fd2 -run 'TestChapter22PreLoadCHUsesSelectedPartyAndRawViewReset|TestResolveNativeIndexedTransition' -count=1
```

兩組均通過；完整證據見
[`fd2_ch22_pre_view_reset_ida.txt`](../data/ida/fd2_ch22_pre_view_reset_ida.txt)。

### 尚未完成

- 這是 E1，不是未修改一般玩家 DOSBox 同狀態 E2，也不宣稱逐像素／逐音訊一致。
- `postbattle_ch23_persist` 及其戰後城鎮／商店／整備／存檔仍失敗即關閉。
- `ch23.json` 的 runtime append handoff 尚無原始證據，暫不把部分 handler 陣列
  猜接成戰鬥 runtime；後續需先取得原版一般玩家路徑與 raw slot trace。

## 2026-08-11：未修改原版 CONTINUE／command grid E2 錨點補證

在確認「後續補證」範圍時，重新檢查 `org_game/炎龍騎士團/FLAME2/FD2.SAV`，並以
Docker `fd2-cap-local` 解碼其 current-runtime header。固定雜湊快照是 chapter0、
12 筆 runtime records、camera `(1,13)`、cursor `(8,17)`、visible `(7,4)`。

接著以一次性 Docker DOSBox（原版目錄唯讀掛載、複本寫入 tmpfs、沒有 route patch 或
debug shortcut）實際走：開場 Escape → 標題 Down、Down、Return（CONTINUE）→
戰場游標單位 Enter。保存兩張 320×200 client crop：

- [`native-continue-current-runtime-original-dosbox.png`](../figures/native-continue-current-runtime-original-dosbox.png)
- [`native-continue-current-command-original-dosbox.png`](../figures/native-continue-current-command-original-dosbox.png)

完整輸入、輸入檔雜湊與 PNG 雜湊見
[`native-continue-current-runtime-e2.json`](../data/ui-traces/native-continue-current-runtime-e2.json)。
這補上原版一般玩家 UI-02／UI-03 的 chapter0 E2 錨點，但不是 ch22／ch23 的
同狀態證據，也不解除重製端 CONTINUE 的 pending-group／controller handoff gate。
本段當時的第23／24／25／29戰阻擋清單是歷史快照；第25戰已由2026-08-13尾端
勘誤解除runtime E1，現況仍阻擋第23／24／29戰。

## 2026-08-11：未修改原版敵方回合 E2 錨點

接續同一份固定雜湊的 `FD2.EXE`／`FD2.SAV`，本輪在 Docker DOSBox 走完：
開場 Escape → 標題 `CONTINUE` → 戰場 Return 開啟 command grid → Down 選
`END` → Return 開啟確認 → Return 選 `YES`。約 1 秒擷取到明確的 `ENEMY PHASE`，
約 10 秒仍在敵方回合，約 20 秒回到玩家操作畫面。三張 320×200 圖片與完整雜湊／
限制見 [`native-enemy-turn-original-e2.json`](../data/ui-traces/native-enemy-turn-original-e2.json)。

這補上原版一般玩家敵方回合的 E2 輸入與畫面錨點，但不宣稱已解出敵方目標選擇、
移動評分、命令／法術／道具效果，也不代表重製端已完成同狀態 parity。下一步仍須
把同一 raw runtime roster、camera、cursor、tick 接到重製端，並完成戰役戰後→城鎮／
商店／整備／存檔的正常玩家 E2；未知 AI 分支保持失敗即關閉。

## 2026-08-11：mode 2 `aiStep` 消費端 E1 已驗證

在上述原版 E2 錨點之外，重製端現已用完整原始移動／地形／道具來源證據（raw
provenance）跑通 mode 2 的 `NextAIPlan`→`aiStep`→行走→FIGANI 攻擊演出擁有者（owner）
→回合完成；另以缺少移動成本資料列（movement-cost rows）的夾具確認執行器會失敗即
關閉，且不把單位標成已行動。測試
檔為 `remake/cmd/fd2/native_ai_consumer_test.go`，這是重製端 E1 遊戲層消費證據，
不是原版目標選擇語意、所有 AI mode 或一般玩家 E2。戰役主線仍須逐節驗證戰鬥→戰後
→城鎮／商店／整備／存檔，不能只以進入下一場戰鬥判定完成。

另以同一個明確截圖模式實際跑通 `story_ch00_handler` 到 `battle_ch01`，保存
[`native-battle-ch01-remake-e1.png`](../figures/native-battle-ch01-remake-e1.png) 與
[`native-battle-ch01-remake-e1.json`](../data/ui-traces/native-battle-ch01-remake-e1.json)。
這是重製端 E1 產物，並非一般玩家輸入或原版同狀態 E2；下一輪仍應優先把原版
current-runtime 的 roster／鏡頭／游標／tick 與重製端逐幀配對，再處理完整敵方 AI
目標選擇與所有戰役戰後節點。

## 2026-08-11：ch24 raw post 共享尾段角色參數勘誤（E1，未正式接線）

以合法 IDA Pro 9.4 Docker 與 `fd2-cap-local` 重新核對固定版 `FD2.EXE`
（357074 bytes、MD5 `b97caf2239a27a896069d03549d96e1e`、SHA-256
`222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f`）。
`0x23790..0x237d5` 的直接指令是 `0x237c6 push 0x0e`、`0x237c8 call
0x112a5`；因此 ch24 raw handler 在 `0x24e7b push 0x1d` 後跳到 `0x237c8`
時，會跳過固定的 14，實際把 29 傳給 `0x112a5`。ch12 的 `push 3→jmp 0x237c8`
與 ch14 的同型資料是交叉證據。原先把共享尾段記成角色14的說法已勘誤，
不可再回填到現況文件。

map24 的 authored／raw 對照也閉合了兩個建構器輸入：group0 唯一我方 raw unit
是26「聖寇拉斯」，group2 唯一敵方 raw unit 是29「亞奇梅吉」；後續 ch26–ch30
party 同時列出26／29。`remake/assets/cutscenes/handlers/ch24_post.json`
因此只做非破壞性資料修正：保留 `source.addr`，將兩個已分級操作編輯為
`join 26`（強推論，`0x24e6c`）與 `join 29`（強推論，`0x24e7b→0x237c8`），
沒有把 handler 接進正式 campaign。

實際 Docker runtime regression 暴露目前真正的 handoff gate：`ch25.json`
重建後有86筆 battle runtime units，但候選 ch24 post binding 仍宣告70筆
map-control frontier，且 `0x10b4e(2)` 的 typed consumer 要求 `Roster`／selector
provenance；直接把候選 binding 接到 battle State 會在 `runtime_context` 或
`native future group 2: runtime roster unavailable` 失敗即關閉。這是保護行為，
不是可繞過的測試問題。`postbattle_ch25_persist→town_ch26` 仍未達一般玩家 E2，
後續必須先取得未修改玩家路徑的70→86 roster handoff與存檔／城鎮邊界證據，
再決定是否升級正式 binding。詳細 IDA／Capstone 指令與雜湊見
[`fd2_ch24_post_ida.txt`](../data/ida/fd2_ch24_post_ida.txt)。

> **2026-08-13 勘誤：本段的86筆 runtime 與70→86 handoff 結論已失效。**
> 86來自重製端錯誤地預載全部70筆控制列後又追加16名隊員；原版可追溯拓撲為
> 62→70→71。正式 binding、JOIN26／29、`town_ch26` 與存讀檔已達 E1；完整
> 證據與仍缺的一般玩家 E2 見本檔2026-08-13條目。

## 2026-08-11：CONTINUE 戰場交接發布契約（E1）

本輪完成的是重製端的最後一段「已驗證快照如何安全交給戰場控制器」契約，
不是把尚未取得的標題／一般玩家證據猜接成完成：

- `remake/internal/campaign/native_continue_handoff.go` 的
  `MaterializeNativeContinueInteractiveBoundary` 只在欄位控制（field control）、
  執行期單位（runtime units）、待處理群組（pending groups）、地圖計時（map timing）、
  視圖（view）、HUD 與開場範圍模式（opening range mode）都存在時，原子安裝
  互動範圍模式（interactive range mode）`1`；`ValidateNativeContinueBattleHandoff`
  再驗證章節／地圖（chapter/map）、尺寸、`runtime_append_groups`、保存的視圖／回合／
  HUD、選擇器快取（selector cache）、隊伍名冊（roster）與待處理群組。
- `remake/cmd/fd2/native_continue_handoff.go` 的
  `publishNativeContinueBattle` 只接受通過驗證的 state，複製 campaign runner，
  一次發布 `Game.st`／`Game.sc`／current node，清除殘留 story、dialog、transition、
  battle event、AI／map 暫存，最後同步保存鏡頭／游標；它不呼叫 `resetBattle` 或
  `Scenario.Setup`，因此不會把 CONTINUE 快照誤重播成新戰鬥。
- Docker 測試 `TestNativeContinueBattlePublicationFromRealCurrentSnapshot` 實際
  讀取 `org_game/炎龍騎士團/FLAME2/FD2.SAV`，其 MD5 為
  `409795ccebc2af340d5c74152c2d471c`、SHA-256 為
  `6d14f2c22562cabca83725084f1a9d6539a1d4066da5c1debcdadb446812691f`；另有
  `TestMaterializeNativeContinueInteractiveBoundaryInstallsControllerMode` 及
  `TestValidateNativeContinueBattleHandoffRejectsIncompleteAdapters`。三項均在
  Docker 通過。

測試刻意由呼叫端（caller）提供 `TitleTimerTick=0` 只驗證資料契約；不代表
`0x25D83..0x25D8B` 帶符號 BIOS 計時值（signed BIOS tick）、`0x10494/0x105ED` 重繪／
延遲（redraw/delay）、標題正式呼叫端、逐像素 E2 或一般玩家 E2 已接通。多章節待處理群組寫入器／公式（pending-group writer／formula）、
戰後城鎮／商店／整備／存檔全路徑、以及 mode 11／`0x13FD4`／mode 5 的完整
目標與指令／法術／道具 AI 仍維持 fail-closed。下一輪應先補正式 caller 的時鐘／
pending producer 證據，再以同一 raw roster／camera／cursor／tick 做原版與重製
逐幀比較；不能以本輪 E1 publication contract 宣稱 CONTINUE 或完整 remake 完成。

## 2026-08-11 勘誤：CONTINUE 標題呼叫端已接 E1

上一段「標題正式呼叫端尚未接通」已由本輪窄切片更新：

- `TitleMenuContinue` 現只接受 `FD2_NATIVE_SAVE` 與 `FD2_NATIVE_TITLE_TICK`；它從
  可編輯戰役圖唯一解析 `scenario.chapter` 相符的 battle node，於私有 state 完成
  field/runtime/pending/timing/view/HUD adapters 後才發布 `Game`。缺存檔、signed BIOS
  tick、資產或章節對映含糊時，標題保持不動並失敗即關閉。
- Docker／Xvfb 以 Escape×8、Down×2、Enter 走過標題續戰，成功發布真實 chapter0
  `FD2.SAV` current-runtime；重製端畫面與條件見
  [`native-continue-current-runtime-remake-e1.png`](../figures/native-continue-current-runtime-remake-e1.png)
  與 [`native-continue-current-runtime-remake-e1.json`](../data/ui-traces/native-continue-current-runtime-remake-e1.json)。
- 這只關閉重製端 E1 publication／輸入邊界。`FD2_NATIVE_TITLE_TICK=0` 是明確的
  deterministic 夾具，不是原版 BIOS 時鐘逐幀 E2；action 選取擁有者、status/equipment
  panel、同狀態原版／重製逐幀比較、戰後 town/shop/preparation/save 全路徑，以及
  敵方 AI 正式 caller／目標／命令／法術／物品決策仍維持 fail-closed。

## 2026-08-11：戰役 intermission、E2 配對與指令環座標勘誤

本輪可交接成果如下：

- `TestCampaignFullChapter16BattlePreservesTownShopPreparationBoundary` 以實際
  `campaign_full.json` 驗證 `battle_ch16→postbattle_ch16_persist→town_ch17`，走過
  武器店返回、整備取消、再次整備確認，才進 `battle_ch17`。這是戰後城鎮／商店／
  整備的可編輯垂直切片，不是 30 章一般玩家 E2。
- `native-continue-current-runtime-remake-e2.json` 保存同一份固定 `FD2.SAV` 的
  原版 CONTINUE E2 與重製端普通 X11 輸入 E1 配對；原版 320×200 放大至 640×400
  比較 AE `164`、RMSE `50.2631`。重製端因 `FD2_NATIVE_TITLE_TICK=0` 仍是計時夾具，
  不得寫成重製端 E2 或逐像素 parity。
- `drawRing` 的 `actionOverlayAnchor` 已在完整 native map frame 時套用 2×呈現縮放；
  `TestNativeActionOverlayAnchorUsesPresentationScale` 固定驗證 `(336,144)` native
  anchor。這只修正座標偏移，不替圖示可用性或 command／spell／item 語意猜測接線。

原版一般玩家敵方回合的 E2 錨點仍是
[`native-enemy-turn-original-e2.json`](../data/ui-traces/native-enemy-turn-original-e2.json)；
重製端同一 raw 狀態的敵方回合畫面、目標選擇與完整 mode 2／3／5／9／11 消費端仍未閉合。
目前已驗證的 mode 2 遊戲層 owner、mode 11 兩段 owner、`0x14EF0` raw route 與
`0x13FD4` indexed／音訊 owner 必須繼續以 Docker 回歸；未知欄位保持失敗即關閉。

## 2026-08-11：遊戲層模式 11 回歸與可玩近似戰後邊界

- `TestAIStepConsumesVerifiedMode11StagesInNativeOrder` 在 Docker／Xvfb 內由
  `NextAIPlan` 實際建立兩段 raw stage，經 `aiStep` 先消費 `0x15311` 指令，再
  沿 continuation 消費 `0x1548E` 物理／FIGANI；`TestAIStepStopsMode11WithoutVerifiedProducerTables`
  證明缺 command book 或 item／movement producer 時會停止且不消耗行動。這是
  重製端 E1 遊戲層 owner，不是原版一般玩家 AI E2。
- 新增 `FD2_APPROXIMATE=1` 可玩近似模式。未綁定的 `postbattle_*` 只同步已物化
  戰場隊伍，顯示戰後整理提示，按 Enter 後沿 authored `next` 進城鎮或整備；
  不猜 JOIN／獎勵／章節／原版分支。未設定時仍維持忠實模式失敗即關閉。
  `TestApproximatePostbattlePreservesAuthoredIntermissionBoundary` 驗證 town 與
  preparation 兩條邊界，並確認戰場狀態已清除；新增
  `TestApproximateCampaignFullUnboundPostbattleBoundaries` 與
  `TestCampaignFullUnboundPostbattleDefaultsFailClosed`，以 `campaign_full.json` 的
  第 23、24、25、29 戰實際節點驗證四條當時尚未綁定的 authored 戰間邊界。
  這是2026-08-11歷史測試範圍；第25戰已於2026-08-13改由正式binding回歸取代。

- 實際 Docker／Xvfb 擷取已保存為
  [`postbattle-approximate-remake.png`](../figures/postbattle-approximate-remake.png)，
  SHA-256 `dfdd3248f0d653a97c95ac1ea17cb8e884436fc4a473a78f2b9bb3d5b4967abe`；這張圖
  只證明近似模式提示可見，不代表原版戰後演出等價。
- 本輪遊戲測試報告由獨立子代理在 Docker 產出，見
  [`game-test-2026-08-11.md`](../reports/game-test-2026-08-11.md)；報告中的 DOSBox／
  重製命令、輸入時間線、資產雜湊與畫面限制已由主代理重新檢查，仍維持 DOSBox
  同 raw 敵方回合未閉合的限制。

## 2026-08-11：ch01 生產戰鬥結果確認回歸

`TestCh00CompiledHandlerCarriesItsExactRuntimeRosterIntoChapterOne` 不再直接呼叫
`Runner.Advance("win")` 進入戰後；測試在已完成的 battle fixture 上走正式
`battle.State.Result`、`Game.checkResult` 與 `Game.confirmBattleResult`，再沿編譯後
戰後節點進入 `town_ch02` 與整備。為保持測試的可重現性，夾具明確清除敵方與 pending
spawn groups，且保留「索爾」保護角色；這證明結果確認的生產消費端，不是未修改原版
一般玩家 E2，也沒有替未知戰後 handler 增加語意。

## 2026-08-11：ch22 生產戰鬥結果→整備→存檔回歸

新增 `TestChapter22BattleResultPreparationSaveLoadUsesProductionBoundaries`，以
map21 的真實 ch22 runtime fixture、73-slot group1／2 frontier 與原版資產唯讀掛載，
先走正式 `Game.confirmBattleResult`，再讓既有 `ch21_post` binding 自然消費到
`preparation_ch23`。測試確認 `postbattle_ch22_persist` 不會被結果確認直接跳過，
進整備時 battle array 已清除且持久隊伍已同步；在隔離 XDG 目錄保存後，再經
`loadGameFromSlot` 還原 chapter=22、節點、隊伍名冊、加入順序與部署狀態。

這是重製端 E1 的正式戰役／存檔邊界，不是未修改一般玩家 DOSBox E2，也不替
66／72-slot frontier、未綁定 ch23／24／25／29 handler 或未知演出增加語意。若資產
或 raw view provenance 缺失，測試會停止而不繞過 handler。

## 2026-08-11：四個未綁定戰後節點的正式結果→近似戰間回歸

新增 `TestApproximateCampaignFullResultConfirmationKeepsUnboundIntermissions`，以
`campaign_full.json` 的 `battle_ch23/24/25/29` 分別建立已完成的勝利結果，走正式
`Game.confirmBattleResult`，確認未綁定的 `postbattle_ch23/24/25/29` 會先停在
明示的近似整理提示；只有 `continueApproximatePostbattle` 確認後，才沿資料檔的
`next` 進入 `preparation_ch24`、`preparation_ch25`、`town_ch26`、
`preparation_ch30`。這補上直接把 Runner 游標放在 postbattle 的舊測試與玩家結果
消費端之間的邊界，仍不猜 JOIN、獎勵或未知 handler 語意。

此為重製端 E1／可玩近似流程，不是未修改原版 E2；忠實模式仍對同四節點失敗即關閉。

## 2026-08-11：敵方回合多單位 loop 與 DOSBox 時間線勘誤

新增 `TestAIStepConsumesTwoVerifiedMode7ActorsBeforeFinishingTurn`／
`TestAIStepStopsTwoMode7ActorsWithoutMovementProvenance`，以兩名 raw mode-7 actor
驗證正式 `NextAIPlan→aiStep` 依序提交兩個行動，最後只完成一次 `Turn++`；缺 movement
provenance 時第一名即停止且沒有部分寫入。這是重製端 E1 編排，不是原版目標選擇 E2。

遊戲測試子代理另查明先前 DOSBox `enemy10`／`enemy20` 相同並非原版沒有敵方回合，
而是輸入時間線少了標題 `Down×2→Return`、戰場 `Return`、`Down→Return→Return`。
補足完整 `Escape→CONTINUE→command grid→END→YES` 後，`enemy1`、`enemy5`、
`enemy10`、`enemy20` 四個畫面雜湊均不同；既有
[`native-enemy-turn-original-e2.json`](../data/ui-traces/native-enemy-turn-original-e2.json)
仍是原版章節0 E2 錨點。這只證實原版敵方回合進入／持續／返回，重製端尚未以同一
raw roster／camera／cursor／tick 配對，故 AI parity 與目標選擇仍保持失敗即關閉。

## 2026-08-11：mode 5 遊戲層消費端回歸

`TestAIStepConsumesVerifiedMode5EventPlan` 與
`TestAIStepStopsMode5WithoutMovementProvenance` 已在 Docker／Xvfb 通過。測試由
`NextAIPlan` 提供完整 raw event grid、field-control row 與 movement-cost rows，
再由 `aiStep` 完成事件格移動、event state `0→1`、map event 清除、raw `+0x34=7`
與回合提交；缺少 movement provenance 時不建立 walk／attack、不改寫事件或回合，
維持失敗即關閉。`FD2_MUTE=1` 只是隔離測試容器缺少 AIL
sample 的聲音環境，不是替未知 mode 5 語意接線。這是重製端 E1 消費端證據，不升格
為原版目標／物品／法術決策或一般玩家 E2；相關限制與測試入口已同步至
[`91-worklist.md`](91-worklist.md) 與 [`56-fd2-remake-sdd.md`](56-fd2-remake-sdd.md)。

## 2026-08-11：mode 7 遊戲層消費端回歸

`TestAIStepConsumesVerifiedMode7DestinationPlan` 與
`TestAIStepStopsMode7WithoutMovementProvenance` 已在 Docker／Xvfb 通過。測試以
raw `+0x35/+0x36` 目的地建立 movement-only 計畫，抵達後提交 raw `+0x05=1` 與
map-range provenance；缺少 movement rows 時不建立 walk／attack、不改寫 raw byte、
位置或回合，維持失敗即關閉。這是 `0x32975` writer 的重製端 E1 owner，不命名
mode 7 高階玩法，也不升格為原版一般玩家 E2；工作清單與 SDD 已同步。

## 2026-08-11：mode 3／9 遊戲層消費端回歸

`TestAIStepConsumesVerifiedMode3AndMode9RawTargetPlans` 與
`TestAIStepStopsMode3AndMode9WithoutMovementProvenance` 已在 Docker／Xvfb 通過。
兩者依 raw `+0x08` 首筆查找建立 movement-only 路徑；mode 3 僅提交 raw map-range
write，mode 9 保留不寫入的分支，均不執行攻擊。缺少 movement rows 時不建立 walk／
attack、不改寫位置、回合或 map-range，維持失敗即關閉。這是 `0x12C60` lookup 的
重製端 E1 owner，不命名高階玩法，也不升格為原版一般玩家 E2。

## 2026-08-11：mode 4／10 遊戲層消費端回歸

`TestAIStepConsumesVerifiedMode4AndMode10DestinationPlans` 與
`TestAIStepStopsMode4AndMode10WithoutMovementProvenance` 已在 Docker／Xvfb 通過。
兩者依 raw `+0x35/+0x36` 目的地完成 movement-only 行走並提交 map-range write，
不寫入 raw `+0x05`、不建立攻擊；缺少 movement rows 時不改寫位置、回合或 map-range，
維持失敗即關閉。這是 E1 raw consumer，不命名 mode 4／10 高階玩法，也不升格為
原版一般玩家 E2。

## 2026-08-11：mode 0／8 遊戲層消費端回歸

`TestAIStepConsumesVerifiedMode0NearestFallback`、
`TestAIStepStopsMode0WithoutMovementProvenance` 與
`TestAIStepConsumesVerifiedMode8Completion` 已在 Docker／Xvfb 通過。mode 0 依 raw
nearest fallback 完成 movement-only 行走且不寫入 map-range；mode 8 走共同的 raw
行動完成分支；缺少 mode 0 movement rows 時不改寫位置、回合或 raw 狀態，維持失敗即
關閉。mode 1 的 blocked-coordinate 遊戲層消費端已由下一節補上；原版語意與一般玩家
E2 仍不以近似值冒充。

## 2026-08-11：mode 1 blocked-coordinate 遊戲層消費端回歸

`TestAIStepConsumesVerifiedMode1BlockedCoordinate` 與
`TestAIStepStopsMode1WithoutMovementProvenance` 已在 Docker／Xvfb 通過。正向測試以
raw `0x14121` blocked-cell、完整 runtime record、selector、地形／組成與 movement-cost
rows 建立三格戰場：`NextAIPlan` 只接受唯一 raw 落點，`aiStep` 走到該落點前一格，
以 movement-only 完成回合；沒有攻擊、nearest fallback 或 map-range 寫入。負向測試
缺少 movement provenance 時，位置、回合、行動旗標與 raw 狀態均不變，維持失敗即關閉。

這是 `0x14EF0→0x14121` 備援的重製端 E1 消費證據，不是完整 mode 1 producer、高階
玩法或原版一般玩家 E2；未知 command／法術／物品與完整敵方回合仍保持未閉合。

## 2026-08-11：`0x14EF0` command route 遊戲層回歸

`TestAIStepConsumesVerified14EF0CommandRoute` 已在 Docker／Xvfb 通過。正向 fixture
提供完整 raw command book、command mask、selector、runtime record、地形／組成、
movement-cost 與 class resistance；`NextAIPlan` 實際選出 `0x14EF0→0x15311`，
`aiStep` 完成 raw destination movement 後，交給已驗證 command 0 numeric owner，
確認 MP 扣除、目標 HP 變更與回合完成。這是 raw route→遊戲層的 E1 證據，不把
synthetic command tuple 升格為原版法術名稱、完整演出或一般玩家 E2。

`0x15055` 未知 item／relocation、未知 command／spell 演出與缺少 producer 時的完整
回合編排仍維持失敗即關閉；已核對的 type-5 item 窄交易另見下一節。

## 2026-08-11：`0x14EF0` type-5 item route 遊戲層回歸

`TestAIStepConsumesVerified14EF0ItemRoute` 已在 Docker／Xvfb 通過。正向 fixture 直接
使用資產 item 192 的 raw type-5 row，提供 command book、selector、runtime record、
地形／組成、movement-cost 與 tile-overlay provenance；`NextAIPlan` 實際選出
`0x14EF0→0x15055`，`aiStep` 交由既有 item owner 完成 `0x211A4` HP 回復、來源欄位
消耗與回合提交。`TestAIStepStops14EF0ItemRouteWithoutItemRows` 缺少 item table 時
保持失敗即關閉，沒有改變 HP、背包、行動旗標或回合。

這是已核對 raw row 的重製端 E1 垂直切片，不是 item 192 的玩法命名，也不是所有
物品、relocation、未知 command／spell 演出或原版一般玩家 E2。後續應優先把這個
窄路由放進同一份 AI regression，而不是複製沒有 producer／consumer 的反組譯筆記。

## 2026-08-11：可編輯 AI 法術後備（fallback）勘誤與 E1 消費端

本檔較早在 2026-07-20 所記的「法術僅建立候選、尚未接入 `NextAIPlan`／正式施放」
是當時工作樹的歷史現況，不能再當作目前限制。現已加入受限的可編輯後備：只有原始
AI 路徑正常未處理時，`NextAIPlan` 才從 `SpellBook`／`Spells` 選擇已知法術，並由
`aiStep → CastArea` 消費至數值效果與回合提交。原始 AI 路徑回報缺少來源
（provenance）或其他 `NativeError` 時仍在狀態變更前停止，不會以後備掩蓋原始資料缺口。

正向回歸涵蓋治療優先於物理備援、攻擊法術朝可施放格移動，以及正式遊戲 loop 的治療
與攻擊消費；缺少決定性亂數時保持 MP、位置、HP、行動與回合不變。這是重製端 E1 的
正規化（normalized）近似，沒有推翻先前對原版法術評分、`0x1598A`、命令格、演出與
一般玩家 E2 尚未閉合的結論。

## 2026-08-11：CONTINUE 空游標命令面板索引勘誤與 E1 畫面切片

先前 2026-07-25 記錄的 `0x1741c` cell index
`3*availabilityWord + 2*directionState` 是把兩個乘數顛倒的錯誤斷言。這次以固定
雜湊 `FD2.EXE` 在合法 IDA Pro 9.4／Hex-Rays Docker 讀取
`0x16f55`、`0x1741c`、`0x18d8c`、`0x177fc`，再用 Docker Capstone 5.0.3 逐條覆核
`0x174f9..0x1752b`，直接指令固定為：

```text
index = 3 * firstArgumentWord + 2 * secondArgumentWord
```

`0x18d8c` 的第一表是 `[0,1,2,3]`，所以 second word 0 對應
`[0,3,6,9]`，word 1 對應 `[2,5,8,11]`；舊的 `[0,2,4,6]`／`[3,5,7,9]`
不可再作 battle skin 或 gate 依據。完整原始位址、表 bytes、工具版本與推論等級見
[`fd2_continue_action_overlay_ida.txt`](../data/ida/fd2_continue_action_overlay_ida.txt)。

同一份未修改原版 `FD2.SAV` 的 normal title `CONTINUE→Return` 畫面，與
`0x16f55` 初始兩表 `[7,5,6,4]`／`[0,0,0,0]` 對應 cells `[21,15,18,12]` 相符。
把該 normal snapshot 歸到 `0x16f55` 仍是**強推論**：本輪沒有向原版程序植入 trace；
cell formula、tables、`0x117e7` 的直接 call site 和 renderer resource 則是**已證實**。

重製端因此改為完整讀取 FDOTHER #2 的 78 格，並在這個狹義 current-runtime 空游標
狀態使用 caller-owned raw state。Docker/Xvfb 以普通 X11 鍵盤事件重播
`Escape→Down→Down→Return→Return`，旁車記錄 `battle_ch01`、cursor `(8,17)`、
selection=false、overlay=true。結果圖與原版／重製／差異比較為
[`native-continue-current-command-remake-e1.png`](../figures/native-continue-current-command-remake-e1.png)
及 [`native-continue-current-command-compare-e1.png`](../figures/native-continue-current-command-compare-e1.png)；
詳細輸入、雜湊與限制見
[`native-continue-current-command-remake-e1.json`](../data/ui-traces/native-continue-current-command-remake-e1.json)。
最近鄰比較 AE 為 8932，故這只是重製端 E1 的可見／輸入切片，不宣稱逐像素 E2。

四格的動作 owner、名稱、目標、確認效果與後續 handler 仍未由這項畫面證據閉合；
重製端在此狀態只允許方向、取消與顯示，Enter 保持失敗即關閉。focused
`internal/fdother`／`cmd/fd2` Docker/Xvfb regression 已通過；完整回歸與文件連結
檢查仍須在提交前重跑。

## 2026-08-12：`0x2BCE5` 已還原前綴的近似戰役接線（E1）

這次只接一個已能閉合證據的垂直切片，沒有把原版終局尾段猜接為可玩流程。
固定版 `FD2.EXE` 的 `0x2BCE5` 前綴、`0x2C405` 的 500-pass scroll、以及
`0x2C548` 邊界仍以
[`fd2_ch29_terminal_body_ida.txt`](../data/ida/fd2_ch29_terminal_body_ida.txt)
為原始定位；`0x2C5CF→FDMUS_004` 的直接事件則見
[`fd2_ending_audio_ida.txt`](../data/ida/fd2_ending_audio_ida.txt)。

- `campaign.Node` 新增嚴格 `native_ending_prefix` 合約：唯一接受
  `assets/endings/native_2bce5.json`、handler `0x2bce5`、chapter `26/29` 與
  `recovered_prefix_only_fail_closed`。錯誤 node type、位址、章節或 mode 都會在
  載入時拒絕；記憶體直接建構的 Campaign 也再次檢查。
- `campaign_full.json` 的最終 `ending` 節點只在 `FD2_APPROXIMATE=1` 啟動該前綴。
  預設忠實模式維持原有可編輯靜態結語與停曲行為；這不是 raw `ch29_post` 的
  campaign owner，也不解除其 handler 仍未綁定的 gate。
- timeline 只允許 `after_gate=0x2c548` 的唯一 audio cue。到達精確的
  `native_finale_montage_opaque/0x2c548` 時才消費 `FDMUS_004`；此處舊有的
  「立刻確認回退」描述已由同日後段勘誤取代：近似模式先播放原資源 party montage，
  cycle 完成或 provenance admission 失敗後才接受確認並顯示可編輯結語。
  `play_bgm(-1)`、`FDMUS_018`、`0x2c194` tail owner、精確按鍵對映與一般玩家 E2
  都沒有 consumer，仍失敗即關閉。
- Docker／Xvfb 以玩家自備 `FDOTHER.DAT`、`FDTXT.DAT`、`ANI.DAT` 直接進入
  最終節點，取得第一個原版文字閘門 `0x2BE44` 的重製端 E1 擷取：
  [`ending-prefix-approximate-remake-e1.png`](../figures/ending-prefix-approximate-remake-e1.png)。
  它的輸入、固定檔案雜湊、狀態旁車與限制記於
  [`ending-prefix-approximate-remake-e1.json`](../data/ui-traces/ending-prefix-approximate-remake-e1.json)；
  這是直接節點，不是第 30 戰一般玩家路徑，且 `FD2_MUTE=1`，不能當音訊或 E2 證明。
- 回歸 `TestApproximateCampaignFinalNodeConsumesRecoveredPrefixThenStops` 使用真實原版
  資產，完整走過兩個已還原文字閘門到 `0x2C548`，檢查唯一 cue 的消費與回退；同日
  後段新增 `TestApproximateCampaignMontageStartsFromPersistentLoadCHOrder`、raw record
  拒絕與 portrait input-loop regression，鎖定 montage admission。`TestDirectEndingPreviewCannotUseApproximateCampaignFallback`、不合法合約與
  截圖狀態旁車測試則防止 direct preview、猜測性設定或截圖被錯誤升格。

較早記錄「`0x2BCE5` 只存在於獨立 preview、未接 campaign」描述的是當時狀態；現在
應讀成「預設忠實模式與 raw terminal owner 仍未接，只有上述明確近似切片可用」。

## 2026-08-12：終局 montage input／nonzero 分支勘誤與近似接線

本次重新以合法 IDA Pro 9.4 Docker 資料庫追蹤 `0x2c405`，再以 Docker Capstone
5.0.3 對固定雜湊 `FD2.EXE` 交叉驗證。這是對較早「`j=1` 的玩家可見意義未知」
記錄的**追加勘誤**，不刪除舊的 raw-word 證據：

- `0x2918a..0x29191` 直接為 `movzx unit[+6]`、`test ebx,ebx`、`je 0x2927e`。
  所以 `0x29164` 是 `+6==0`／非零分支；舊 `MontageCycle` 只允許 `0/1` 的限制
  錯拒了 persistent party constructor 實際會產生的 `+6=2`，已修正為全 nonzero。
- `0x2c946..0x2c96a` 直接為 `0x17aa9(1)→0x10620→test→j=1→0x4e031→inc edi`；
  `0x2c5e3..0x2c5ec` 在當前 portrait loop 結束後遞減 outer counter。因此 raw word
  差異的已證實效果是「完成當前 portrait，下一 outer loop 走 `j=0`」，而不是立刻
  中止畫面。仍**未知**的是造成 word change 的實際 BIOS key code／佇列規則；不能
  反寫成原版 Enter、Space 或 Escape。
- `0x17aa9` 已直接確認以 wrap-aware `word[0x46c]` delta 等待。近似 runtime 的
  55ms tick 是 BIOS 頻率的強推論，不是逐毫秒或一般玩家 E2。

實作只在 `FD2_APPROXIMATE=1` 的 final node 使用 persistent JOIN roster 的 deployed
order 與 raw `+6/+7/+8/+0x20` 建立原資源 `MontageCycle`；new input 只在 portrait
phase 轉為上述 raw-change effect。素材或 raw provenance 缺失時不猜補 renderer，仍回到
可編輯結語。`0x2c194` tail、精確輸入 owner、`FDMUS_018`／停曲、raw terminal owner
與未修改一般玩家 E2 仍是未解除 gate。完整現況同步至 SDD、UI matrix、worklist 和
`fd2_ch29_input_cleanup_ida.txt`／`fd2_ch29_montage_ida.txt`。

同日再確認：`ch29_post` 的 `0x25970→0x2bce5` 仍留下明確編譯問題（compile issue）；
截圖隊伍輔助程式會拒絕其不完整的 LOADCH，不能把它當作上述持續隊伍的一般玩家來源。

## 2026-08-12：終局尾段資源角色與近似定格勘誤

這是對本檔較早「#58／#57 都是 loop frame source」及「party montage 完成後一律回到
可編輯結語」的**追加勘誤**；舊記錄保留其當時的發現過程，不再作為現況接口。

- 固定雜湊的 IDA Pro 9.4／Capstone 證據現在明確分開：FDOTHER #60（`0x2c1be`）與
  #59（`0x2c357`）是 320×200 單影格；#58（`0x2c220`）是交給
  `0x2935b` 的 20-entry frame table；#57（`0x2c234`）是 768-byte 256×3 VGA
  調色盤，於 `0x2c2b6` 指派 `[0x53a65]` 後交給 `0x11d40`。`0x2935b(resource_57,...)`
  的舊文字是 stack resource index 錯置，已在
  [`fd2_ch29_post_montage_tail_ida.txt`](../data/ida/fd2_ch29_post_montage_tail_ida.txt)
  勘誤為 `resource_58`。
- `FD2_APPROXIMATE=1` 的已 admission 終局節點完成 `MontageCycle` 後，現在會驗證
  #57/#58/#60/#59 的來源形狀、呈現並保持 #59 靜態圖；只有素材或 raw provenance
  admission 失敗才回到可編輯結語。原始 `FDMUS_018` 位於未還原 20-entry loop 前，
  所以重製端在定格取得後接同一曲目只能標作**近似時序**。
- 第 30 戰外部片尾錄影顯示黑底金色 `THE END` 持續畫面。依 raw 順序將 #59 對應成
  該定格是**強推論／外部視覺旁證**，不是原版一般玩家 E2；方法與限制見
  [`ch30-ending-youtube-visual-side-evidence.json`](../data/ui-traces/ch30-ending-youtube-visual-side-evidence.json)。
- 終局定格是預設。Enter／空白鍵才啟動可選的隊伍最終狀態重播循環，Enter／空白鍵／Esc
  回到 #59。它是重製版延伸，未映射為 `0x10620` 的原版 BIOS 輸入，也不推定原版 self-loop
  有同樣功能。

`0x28a6c` 的 20-entry renderer、#60 的可見 owner、原始 palette/wait 時序、raw
`0x25970→0x2bce5` 一般玩家 owner 與完整終局 E2 仍未閉合；它們不能因近似定格而被
接到正式 campaign handoff。

## 2026-08-12：`0x2c194→0x28a6c(0,1)` 雙 record 排程勘誤（E0；仍失敗即關閉）

本段是對較早尾段筆記中「首筆 `unit+6=0`、其餘為 2」與把
`[0x53a45]+0x56/+0x57` 視為同一筆 record 高位欄位的**追加勘誤**。舊筆記保留作為
錯誤形成過程，不可再當作重製輸入合約。

固定雜湊 `FD2.EXE` 以合法 IDA Pro 9.4 Docker 資料庫檢視 `sub_2BCE5`，再以 Docker
Capstone 對 `0x2c253..0x2c33b` 逐指令覆核，現已確認固定的 60 bytes 分別是：

- LE 線性位址 `0x525dc..0x525ef`（object 2 檔案偏移 `0x523dc..0x523ef`）：record 0 的 `+7` selector（20 筆）；
- LE 線性位址 `0x525f0..0x52603`（object 2 檔案偏移 `0x523f0..0x52403`）：record 1 的 `+7` selector（20 筆）；
- LE 線性位址 `0x52604..0x52617`（object 2 檔案偏移 `0x52404..0x52417`）：每輪寫入 `[0x540ff]` 的值（20 筆）。

runtime record stride 是 `0x50`，故 `0x53a45+0x56 = 0x53a95 = record 1 + 6`、
`+0x57 = record 1 + 7`。每輪都以各自的 selector 即時計算
`+6 = selector < 0x4c ? 2 : 0`，再呼叫 `0x28a6c(0,1)`；`+6` 不是另有兩張表，
也不是「第 0 筆特例」。例如第 0 輪為 record0 `( +6=2, +7=0x33 )`、record1
`( +6=0, +7=0x67 )`；第 19 輪則為 record0 `(0,0x7e)`、record1 `(2,0x32)`。

因此 `native_2c194_tail.json` 升為 schema 2，拒絕舊的欄位轉置；
`MontageTail.Plan` 只輸出兩筆 record 的 raw `+6/+7` 與 `[0x540ff]`，不寫入
`battle.State`、不建立 renderer adapter、也不把 record 0/1 命名成角色或攻守。FDOTHER
#58 的 20 筆 frame descriptor 幾何已有玩家提供檔案 regression，但它只證實原始
frame-table layout；`0x28a6c` 對這些輸入的可見輸出、20 段時序、輸入 owner、一般玩家
E2 與 raw campaign handoff 仍未閉合，全部維持失敗即關閉。詳見
[`fd2_ch29_post_montage_tail_ida.txt`](../data/ida/fd2_ch29_post_montage_tail_ida.txt)。

> **同日後續勘誤：**上述「20 段重製轉接器」缺口應限縮為**精確**
> `0x28a6c` renderer。明確近似模式已有 `MontageTailPlayer` 消費 20 組原版
> TAI／BG／FIGANI 與 FDOTHER#58、完成後保持 #59；完整說明、回歸與 E1 擷取見
> 本檔較前方「20 組終局尾段近似播放器」一節。呼叫時 records/globals、狀態欄、
> 滑動、聲音、效果、原版輸入時序及一般玩家 E2 仍維持失敗即關閉。

## 2026-08-12：終局 loader 基線與 `0x28a6c` 非零視覺鏈勘誤（E0）

本段追加更正兩項會阻礙後續實作的舊斷言。固定雜湊 `FD2.EXE` 先以合法
IDA Pro 9.4 Docker 資料庫追蹤交叉參照與資料流，再用 Docker Capstone 5.0.3
逐指令覆核；FDFIELD／FDICON 也綁定 reference manifest 的大小與雜湊。

- `0x1088d` 的終局 caller 是 `0x2c435 push 0x1e`、`0x2c437 call`，不是
  `0x2c469`。selector `0x1e` 使用 FDFIELD #90/#91/#92，建立 31 筆 deployment
  runtime records：active prefix 完整複製 persistent `0x50` record 後做位置、
  selector cache、狀態及 `0x1b750` 覆寫，其餘 records 只標 inactive。
  `MontageTailLoaderBaseline` 現以真實玩家素材測試這個形狀，且只回傳值拷貝。
- `[0x540ff]!=0` 的 `0x28a6c` 並非沒有畫面。它仍載入 TAI／FIGANI／BG，經
  `0x29164→0x2939d` 合成、frame loop、palette／VGA consumer 輸出；非零分支
  略過的是 `[0x540ff]==0` 才呼叫的 `0x29f72` 一般戰鬥結果解析器。

這兩項只把原版 loader baseline 與可見消費鏈提升到 E0。`0x2c548` 位於 loader
與 `0x2c2a6` 之間，可能觀察或改動同一 runtime image；因此 post-loader baseline
不可冒充呼叫當下 snapshot。尚缺完整 records/globals、精確 20 段重製轉接器、
精確輸入與一般玩家 E2；忠實模式仍維持失敗即關閉。近似播放器的後續勘誤見上節。
完整證據與工具／位址基準見
[`fd2_ch29_post_montage_tail_ida.txt`](../data/ida/fd2_ch29_post_montage_tail_ida.txt)。

## 2026-08-12：第 29／30 戰終局持續隊伍邊界（近似 E1）

本段不新增原版 handler 語意；它修正重製端既有戰役圖的資料遺失，並保留先前
raw owner 與一般玩家 E2 的失敗即關閉結論。

- 第 29 戰 `battle_ch29 → postbattle_ch29_persist → preparation_ch30` 現新增正式
  勝利確認、近似戰後同步、整備節點隔離 JSON 存檔／讀檔回歸。`ch28_post` 未閉合的
  renderer 與原版一般玩家路徑仍未完成。
- 第 30 戰原本由 `battle_ch30.on_win` 直接進 `ending`；`confirmBattleResult` 不同步
  戰場，而結局輪播只讀 persistent `partyRoster`，因此最後一戰的等級、經驗與數值
  會遺失。campaign 節點新增 `approximate_sync_party_on_win`：僅
  `FD2_APPROXIMATE=1` 且勝利邊直接指向 ending 時，在移動游標前呼叫既有
  `syncPartyFromBattle`。同步失敗或零筆持續隊伍身分（persistent identity）符合即
  停在原節點並保留勝利結果；忠實模式不執行。
- campaign loader、近似成功／失敗與忠實模式皆有回歸。這個欄位是重製端 E1
  資料邊界，不能用來宣稱 `0x25757→0x2bce5` owner、精確終局 handler 或一般玩家
  E2 已閉合。

## 2026-08-12：玩家第 28／29 戰前置處理器錯一章勘誤（E1）

本段追加推翻本檔 2026-07-20 條目中「`ch28_pre` 已接 `story_ch28`」的舊結論；
舊條目保留作為錯誤形成過程，不可再當作現況。

- 合法 IDA Pro 9.4 Docker 直接讀 `0x51D71` 前置表：raw index27=`0x33C9D`、
  index28=`0x33DBA`、index29=`0x33E3C`。`0x205DA→0x1088D(chapter)` 同時載
  `FDTXT_(chapter+1)` 與 `FDFIELD_(3*chapter+0/1/2)`；所以玩家第28戰必須用
  `ch27_pre`／map27／FDTXT_028，玩家第29戰才用 `ch28_pre`／map28／FDTXT_029。
- campaign 現改為 `story_ch28→ch27_pre.json`、`story_ch29→ch28_pre.json`；
  後者 binding 由錯誤的 map27／slot70／ch28 scenario 修成 map28／slot76／
  ch29 scenario。兩條均由原始資產 Docker／Xvfb 回歸走到相應戰鬥節點。
- `ch27_pre` 的 slots0..19 隱藏後，只在 current HP word `+0x40!=0` 時清 byte
  `+5`；現以 call-site 限定、固定20筆、全批 preflight 的可編輯原語保存。
  `0x33CE2→0x24618` 另固定 relative cursor X+6／Y+5。缺 raw byte來源、位置、
  indexed 資產或群組筆數時仍失敗即關閉。
- `0x1088D` 的部署迴圈從 `2+6*enemy_ally_total` 起讀 control 宣告的
  `own_deploy` 筆 X/Y，不讀第三 word。exporter 已停止用 `raw_key==0` 掃全表：
  map28 own_deploy 16→20；map31／map32 1／2→0。
- raw index29 `0x33E3C` 仍受 `0x22253` 完整 indexed 演出阻擋，沒有因相鄰 owner
  closure 而猜測接入第30戰。上述僅是 E1，不是一般玩家 E2 或逐像素一致。

完整位址、雜湊、FDFIELD 資源與推論等級見
[`fd2_ch27_ch28_pre_owner_ida.txt`](../data/ida/fd2_ch27_ch28_pre_owner_ida.txt)。
本檔較早引用的
[`fd2_ch27_pre_view_ida.txt`](../data/fd2_ch27_pre_view_ida.txt) 實際存在，且回答
raw ch26 pre→玩家第27戰的視窗／HUD 來源；新證據回答 raw ch27／ch28 pre 的
owner 與部署尾端。兩者主題不同、都保留，撤回先前「舊檔不存在／已被取代」的說法。

## 2026-08-12：IDA 全函式清冊與 handler unknown 三態勘誤

- 合法 IDA Pro 9.4 在一次性無網路 Docker 對固定雜湊 `FD2.EXE` 重跑；DOS LE
  loader 建立 `0x10000` 起始 code segment，Watcom 9.x FLIRT 生效，辨識1305個
  函式。受版控精簡清冊為 [`fd2_function_inventory.json`](../data/ida/fd2_function_inventory.json)：
  產品27、runtime170、未知1108；這是第一輪函式分類，不是 remake 百分比。
- [`fd2_semantic_index.json`](../data/ida/fd2_semantic_index.json) 保存28筆非破壞性
  位址語意、等級、scope 與 evidence。匯出器拒絕雜湊不符或註記未落函式起點；
  不改 IDA 名稱、型別、註解或資料庫。完整逐 call-site xref 留在一次性產物，版控
  清冊保留函式邊界與直接 caller 函式。
- 60份 handler 的歷史83個 `unknown` 已分成78個已證實窄 `native_call`、4個
  已知 callee 但 caller/runtime 未閉合的 `unresolved_native_call`
  （`0x22253`、`0x2BCE5`）與唯一真正未知 `0x24336`。每筆具名呼叫保存原始
  source／PUSH順序、推論等級與 evidence；Go compiler 仍依 exact caller/binding
  失敗即關閉，不因名稱自動接入 campaign。
- 全檔 caller 清冊同時勘誤 `0x28A6C`：直接 caller 包含正式戰鬥、人工智慧／事件
  與終局 `0x2C2A6`，所以它是共用雙 runtime-record renderer，不是終局專屬函式。
  終局缺口限縮為該 caller 當下 records／globals、精確輸出、輸入與 campaign handoff。

## 2026-08-12：文件權威與重複反組譯治理

- 新增 [`58-fd2-exe-re-coverage.md`](58-fd2-exe-re-coverage.md)，把每個子系統拆成
  原版證據、可編輯資料、正式執行期、一般玩家 E2 與下一缺口；不再用單一 `[x]`、
  文件數或沒有分母的百分比宣稱完成。
- README、`00`、`56`、`57`、`91`、`99` 與歷史計畫入口已統一責任；本檔明確降為
  時間序列證據，`91` 只有檔首有效佇列能決定下一步。舊40–45%介面估計、舊19 active／
  5 blocked 現況與「資料格式全破」已撤回或標為有日期的歷史快照。
- 已閉合位址只有雜湊不同、直接指令／跳表反證、同狀態執行矛盾，或主證據缺少其
  聲稱的 writer／consumer 時才能重開。raw exporter 仍標 unknown 或搜尋命中本檔，
  都不能單獨觸發重新反組譯。
- 全庫 Markdown 本地連結／圖片以受版控檢查器在一次性 Docker 檢查682項，0項
  斷鏈；攻略 Markdown 的36張圖片已修回 `../html/`。完整 Go 回歸、Capstone image
  內41項工具測試及
  玩家第28／29戰前置 owner／部署尾段的唯讀原版資產重點回歸均通過。

## 2026-08-13：`0x24336` 閉合與文件現況勘誤

本段追加更正上一節的歷史快照；上一節的78／4／1與產品27／runtime170／未知1108
保留為當時狀態，不可再當現況。

- 合法 IDA Pro 9.4 Docker 重生固定雜湊 `FD2.EXE` 清冊後，受版控語意索引為32筆，
  現況是產品31、runtime170、未知1104。60份 handler 則為79筆 `native_call`、
  4筆 `unresolved_native_call`、0筆真正 `unknown`；4筆未閉合項仍是
  `0x22253`／`0x2BCE5` 的 caller／執行期問題。
- `0x242C9→0x24336` 已由 IDA 函式／交叉參照與 Capstone 直接指令閉合為玩家
  第21戰戰後天空之鑰固定演出。它依序消費 `FDOTHER.DAT #34` 101幀、
  `ANI.DAT #0` 96幀、`0x11DF2` 調色盤變換與 `0x4DFCC` 高色階相對相位循環；
  詳見
  [`fd2_ch20_sky_key_sequence_ida.txt`](../data/ida/fd2_ch20_sky_key_sequence_ida.txt)。
- 重製端新增精確來源／目標／零參數的具型別指令，任何被編輯的 payload 或缺少
  原始資產都失敗即關閉。正式戰役回歸已由玩家第21戰勝利走過六素材配方、動畫、
  JOIN24／23、隊伍同步、`town_ch22` 與存檔／讀檔。
- map20 另由固定雜湊 FDFIELD 資源證實75→79→83→87→91的群組追加前沿；
  `ch21.json` 已改用 `runtime_append_groups`，群組255不物化，撤回舊重製端把80筆
  控制列與16名部署隊伍併成錯誤初始拓撲的可能性。
- 目前只達 E1。未修改一般玩家同狀態 E2、`0x4DFCC` 第一個程序內相位，以及相鄰
  `layout_units`／ACT63／64仍待補；不得因此重新反組譯已閉合的 `0x24336` 本體。

## 2026-08-13：玩家第25戰 raw ch24 post 的86-slot斷言勘誤與 E1 接線

本段追加更正本檔較早「70-slot story 對86-slot battle」與
`postbattle_ch25_persist`仍阻擋的說法；歷史條目保留錯誤形成原因，不可再當現況。

- 固定版 FDFIELD map24 控制列仍是70筆：group0=46、group1=8、group2=1、
  group255=15。錯誤來自重製端先預載全部70筆，再加16名隊員而形成86；原版
  `0x10B4E` 是按 group 追加，而不是預建整份槽位穩定名冊。
- `ch25.json` 現採 `runtime_append_groups:true`、`initial_groups:[0]`：開戰先建立
  party16＋group0(46)=62；第6回合 event56 依既有 `0x3549F→0x10B4E(1)` 證據
  追加group1(8)成70；raw ch24 post再以`0x24E31→0x10B4E(2)`追加唯一group2
  成71。ACTING resource75操作slot70，與此順序精確吻合；group255沒有初始或
  事件materializer，不進runtime。
- 正式`postbattle_ch25_persist`已綁定`ch24_post.json`。Docker／Xvfb決定性回歸
  從`battle_ch25`勝利確認進入戰後，走過PAN、spawn、ACT75、JOIN26、sync、
  JOIN29、chapter25，最後進`town_ch26`；另驗證JOIN順序、原始身分持續紀錄與
  town node-boundary save/load。
- 【2026-08-13 歷史快照；現況統計以 `58-fd2-exe-re-coverage.md` 為準】戰後稽核
  當時為24節點中21 active／3 blocked；當時剩餘玩家第23、24、29戰。
  本切片只提升為runtime E1，仍缺未修改一般玩家原版同狀態E2與逐像素比較。

## 2026-08-20：第25戰後 `town_ch26` 祕密商店正常戰間 E1

本輪沒有重開已閉合的祕密商店反組譯，而是補上玩家可見消費鏈。既有
`0x2cd16→0x2cef7` 與23筆章節表已證實每個城鎮使用不同的normal selection／BIOS
scan；`town_ch26` 是 selection4＋Shift+F5（`0x58`），不是ch02的Shift+F1（`0x54`）。

- 第25戰正式回歸在62→70→71、JOIN26／29、`town_ch26`存讀檔之後繼續執行；
  錯誤`0x54`不改selector，`0x58`只將selection4揭露成selection5並以原版城鎮資源
  重畫，仍停在town。
- 下一次確認才進`shop_ch26_secret`；正式原版資源消費端啟用variant5四幀開場，
  商品ID固定為195／207／40。四幀離店完成後回`town_ch26`並保留selection5、
  JOIN26／29及持續隊伍。
- Docker／Xvfb focused regression通過。這是重製端正常戰間`RUNTIME-E1`，仍缺
  未修改原版第25戰勝利後相同狀態的`PLAYER-E2`與逐幀畫面／音訊比較。

## 2026-08-20：chapter0 `CONTINUE→END→YES` 玩家入口接入敵方回合

本輪沿用既有未修改原版 E2 證據，不重開已閉合反組譯。固定原版存檔的輸入時間線
已證實空游標面板按 Down 選 END、Return 開確認、Return 選 YES；重製端因此只為
current-runtime direction3 建立正式 owner，其餘三格仍失敗即關閉。

- action overlay 的四個 closing present 完成前不發布確認，也不啟動敵方回合。
- YES 經既有 `endTurn→beginEnemyPhase→aiStep→finishTurn` 顯示 `ENEMY PHASE`，
  敵方無可行動計畫時完成回合並返回 `PLAYER PHASE`；取消不改寫回合。
- Docker／Xvfb focused regression 覆蓋正向鏈、四幀發布邊界、取消與 direction0..2
  反例。確認提示暫用重製文字層；原版 indexed 確認 renderer 及同狀態重製 E2 尚缺。

## 2026-08-21：END 靜態閉合與 command 13 演出 owner

本輪以合法 IDA Pro 9.4／Hex-Rays 的一次性 tmpfs 資料庫為主、Docker Capstone 為
第二套驗證。專案 IDA 映像的 GUI `ida` 缺 X11 library，改用同一映像內可用且授權
正常的 headless `idat`；沒有改用 host、Ghidra 或建立重複 image。

- `0x16F55` 的 `[0x53C57]==3` 直接分支顯示 FDTXT `0x1A3`，YES choice0 後顯示
  `0x1A4`，呼叫 `delay(0xC8)` 再進 `0x1A30B`；`0x117E7` 為直接 caller。
  所以 direction3→END 已由旁證提升為直接指令證實。
- 重製端使用解碼原文「要結束本回合的行動嗎？」與含換行的肯定句，並以十二個
  60 Hz幀近似200 ms；延遲完成前不啟動 AI，之後才進敵方回合並返回玩家回合。
- command 13–16 的 wrapper／共同演出 owner 已固定於
  `0x21AD9/0x21B99/0x2211C/0x22153→0x21B18`。command 13 正式玩家交易回歸
  已通過；indexed 畫格／調色盤／音效仍未移植，不用文字結果冒稱演出完成。
- 主證據：[`fd2_end_turn_command13_owner_ida.txt`](../data/ida/fd2_end_turn_command13_owner_ida.txt)。

## 2026-08-21：command 13–16 `0x21EB1` indexed 演出接入

本輪沒有重解已閉合的 `0x22046` compositor，而是用合法 IDA Pro 9.4／Hex-Rays
與 Docker Capstone 追查尚未分型的 caller `0x21EB1`。第一次 IDA 啟動曾因錯誤覆寫
專案授權 entrypoint 而被 license gate 拒絕；沒有產生資料庫或儲存庫變更。更正後
沿用 `fd2-ida-authorized-local`，原始EXE唯讀、資料庫只在tmpfs。

- 四個 wrapper 對 `0x21EB1` 的 raw `(start,step)` 是13=`(1,2)`、14=`(2,4)`、
  15=`(8,4)`、16=`(6,6)`，並共同播放 sample index11。
- `0x21EB1` 以鏡頭相對可見游標建立中心，從 boot 載入的FDOTHER #3取LUT，先走
  descriptor9→1的九張擴張，每張5 ms；等待200 ms後以最後半徑走3→9七張收束，
  再`0x11CAC(0)`與200 ms尾停。完整snapshot restore、object redraw與viewport copy
  復用既有strict `0x22046` adapter。
- 玩家 command 13–16 已接此16張排程；開始前一次驗證全部raw map／LUT／palette／
  visible cursor／normal baseline。缺任一項不扣MP、不改HP、不提交acted；演出完成後
  才進既有交易與range reset。敵方owner、後段`0x1C4CC/0x1C2DA→0x1E0DB`及
  同狀態原版逐幀／逐音訊E2仍待。
- 主證據：[`fd2_command13_21eb1_presentation_ida.txt`](../data/ida/fd2_command13_21eb1_presentation_ida.txt)。
- 提交前 Docker 驗證：`go test ./... -count=1` 全套通過；工具測試55項、JSON 917份、
  戰後節點稽核21 active／3 blocked均通過；本批文件579個具副檔名本地目標無斷鏈。
  原版每張5 ms在60 Hz重製端採至少一個可見幀的E1近似，沒有升格成逐毫秒E2。

## 2026-08-21：敵方 mode 11 command 13–16 target array 與演出消費端

接線前回歸先證明玩家 `NativeCommandTargetMatches(1)` 的絕對陣營 predicate 不能
直接套到 Enemy；該嘗試未提交。回查既有 IDA 主證據後，`0x15311` 已明確在移動完成
後用 winner 座標、row `+4/+6` 再呼叫 `0x14818`，因此重製端新增 AI 專用 typed
target builder：由 raw `+6` selector、當下 presentation 座標、完整 runtime records
與 composition flags 重建 target array，不猜測翻轉 `Camp`。

- `ExecuteNativeAICommandHeal` 只接受 command 13–16 與完整 raw provenance；缺 selector、
  roster、baseline、LUT 或 palette 時不扣 MP、不改 HP、不提交 acted。
- mode 11 的 `0x15311` consumer 現先播放同一 `0x21EB1` 16張演出，完成後才交易；
  `aiStep` 在演出存在時不重新規劃，避免同一敵人重複行動。
- 這是重製端 `RUNTIME-E1`；後段數字佇列、同狀態原版逐幀／逐音訊與一般玩家
  mode 11 E2 仍缺。主證據沿用
  [`fd2_ai_mode11_full_ida_20260810.txt`](../data/ida/fd2_ai_mode11_full_ida_20260810.txt) 與
  [`fd2_command13_21eb1_presentation_ida.txt`](../data/ida/fd2_command13_21eb1_presentation_ida.txt)。
- Docker／Xvfb `go test ./... -count=1` 全套、55項工具測試、917份JSON、戰後
  21 active／3 blocked稽核均通過；本批582個具副檔名本地文件目標無斷鏈。

## 2026-08-21：raw ch23／玩家第24戰戰後 adapter 勘誤與 E1 接入

本條取代2026-08-09至本日前所有「入口 latch、三個 transient offset 或第一幀
來源仍阻擋 `postbattle_ch24_persist`」的舊現況敘述，但保留那些歷史段落作為錯誤
形成過程。合法 IDA Pro 9.4 直接證實 `sub_24C1E` 在每個 inner loop draw 前都先
呼叫 `sub_24D22(stage)`；offset globals 的四個 producer 又全在自己的共同尾端清零，
且不由此 handler 呼叫。`sub_11EEE` case23 的 row rotation 只由 BIOS tick snapshot
差異觸發，`sub_4DFCC` 的高位 palette window 則另有兩拍 gate。

- 知識庫主證據先更新於
  [`fd2_ch23_post_ida.txt`](../data/ida/fd2_ch23_post_ida.txt)，再於
  [`56-fd2-remake-sdd.md`](56-fd2-remake-sdd.md)建立 adapter 規格，最後才接
  production runtime，遵守「RE→規格→實作」順序。
- 正式 adapter 消費 FDOTHER #42、312×192 staging、240＋60次 draw、ESI 0..59、
  相對 BIOS tick與兩拍 palette gate；任何 compose／buffer／palette 錯誤都原子
  回復且不執行隊伍同步或章節遞增。
- 正常戰役回歸從 `battle_ch24` 的戰果確認出發，保留 map23 的70筆 FDFIELD＋
  16筆 LOADCH runtime，完成兩段演出與 `sync_party` 後進 `preparation_ch25`，
  並在該節點驗證存檔／讀檔。因此戰後稽核現為22 active／2 blocked；剩餘是
  玩家第23、29戰。
- 本批只提升為 `RUNTIME-E1`。handler 入口程序相位、未修改原版同狀態逐幀／
  時序仍缺 `PLAYER-E2`，不得宣稱逐像素或 DOS BIOS 時鐘一致。

## 2026-08-21：raw ch22／玩家第23戰戰後正式接線（RUNTIME-E1）

本條取代較早「`postbattle_ch23_persist` 仍失敗即關閉」及22 active／2 blocked
的現況敘述；舊段落保留為歷史形成過程。工作順序先補
[`fd2_ch22_post_ida.txt`](../data/ida/fd2_ch22_post_ida.txt) 的 IDA Pro 9.4 證據，
再更新 [`56-fd2-remake-sdd.md`](56-fd2-remake-sdd.md) 契約，最後才接正式執行期。

- 原版 `0x1088D` 先建立16筆 persistent records，再由 `0x10B4E` 追加 map22
  的70筆 records；重製端 ch23 scenario 現以相同順序建立86-slot frontier。
  現有 authored `initial_groups=0..9` 仍是全群組已在場的近似，event52 在
  rounds13／15／18／22 的精確追加時序沒有被冒稱已還原。
- 正式 `ch22_post` binding 固定86 slots、18筆 layout、ACT71／72／73、三項
  resource reload 與兩個原始 indexed adapter；compiler 零 issue。palette loop
  依 caller 固定為 `0x24A24` 的0、2、…、62，共32步，且每步只消費一次4 ms。
- 正常回歸由 `battle_ch23` 的玩家戰果確認進入 handler，不直接呼叫處理器；
  完成隊伍同步後抵達 `preparation_ch24`，再驗證存檔／讀檔。戰後稽核因此為
  24節點中23 active／1 blocked，只剩玩家第29戰。
- 本批只提升為 `RUNTIME-E1`。event52精確時序、高階畫面名稱與未修改原版
  同狀態 `PLAYER-E2` 仍缺；不得宣稱 DOS 時序或逐像素一致。

## 2026-08-21：raw ch28 `0x25535` 窄 E1 與文件斷言勘誤

本條取代舊交接中將所有 `0x22253` 概括為「renderer／GUI adapter 未完成」及
handler 尚有4筆 unresolved 的現況說法；舊段落保留為歷史形成過程。固定版
caller 現已證實 `0x25535→0x22253([0x53BEB]-1,15,10,15,10)`，但只證實動態
最後 materialized slot，不證實固定 slot93。

- 先更新 [`fd2_ch28_post_ida.txt`](../data/ida/fd2_ch28_post_ida.txt) 的 caller／
  topology 證據，再於 [`56-fd2-remake-sdd.md`](56-fd2-remake-sdd.md) 建立 typed
  presenter 契約，最後才接正式執行期，遵守 RE→規格→實作。
- `native_unit_present` 只接受來源 `0x25535` 與動態最後槽位，完整預先產生
  11＋6＋18/24＋10個 indexed frame／row；mutation 只發生在 bridge 邊界，
  缺 asset／raw provenance 時零修改，執行中錯誤回復 unit／work／VGA。
- `native_palette_pulse` 另精確保存 `0x35E5A` 的0..63、400 ms peak hold、
  62..0，共127次 DAC 寫入；它不是 RGBA fade 或純 delay。
- source comment／Markdown 稽核已區分上述 battle-state E1、仍 blocked 的
  `0x33F78` story/focus wrapper與無 caller ABI的 legacy `unit_present`。另修正
  handler manifest 產生器每次執行重複插入 JSON key 的缺陷並加冪等回歸。
- Docker 驗證：Go `go test ./...` 全套、Python工具56項、918份JSON（含重複鍵）、
  722個 Markdown 本地目標皆通過；分類為80 classified／3 unresolved／0 unknown，
  戰後節點仍23 active／1 blocked。
- 玩家第29戰仍失敗即關閉：map28 groups1..9 materialize 拓撲、group9後的實際
  runtime frontier、dialog／pan binding、持續隊伍、`preparation_ch30` save/load
  與未修改一般玩家 E2 尚未閉合。

**2026-08-21 勘誤（取代上一點的 map28 groups1..9 概括）：** 一般入口已證實
為20筆持續隊伍＋group8(56)=76，不會把groups1..9全部物化。event75 selector1
與event74 groups4..7已成為可編輯資料及正式 `RUNTIME-E1`；event76／79、post
group9、持續隊伍、`preparation_ch30` save/load與一般玩家E2仍待完成。map28
事件格依實際寬31重算為(15,21)；暫算(28,22)已撤回。

**2026-08-21 event76 續接：** `sub_35D60` 已從主證據與SDD lower成可編輯
progression，正式runtime在round increment後dispatch raw-camp2。repeat branch
原子寫slot1 raw `+5` bit7／state17／row1；final branch在index2後私下建構group1
三筆、寫state21 base與event79 row，再依序播放六次`0x35E5A`、兩次額外400ms及
indices3..6。event79、post binding、save/load與E2仍未完成。

**2026-08-21 event79 續接：** `sub_35EE6` 已lower成typed pair mutation；正式
raw-camp0 owner以process-wide `nativeRNGState`前進一次，從state21 base的三筆
group1選兩個循環相鄰slots，原子設定raw `+5` bit7並reschedule row2。post
binding、save/load與E2仍未完成。

**2026-08-21 ch28 post raw prelude 補證：** 合法 IDA Pro 9.4 直接重讀
`sub_2548C`、`sub_35BBA`與`sub_1DB65`。現已固定slots20..tail raw word
`+0x40`清除、slot20 `+7/+8=0x7E`，並新增只接受76／78／80／82／84／87的
frontier validator與原子raw transaction。`0x1DB65`另有13＋6＋6次indexed呈現及
raw `+3/+5` writes，不是generic redraw；原資源presenter、group9完整transaction、
正式binding、持續隊伍／`preparation_ch30` save/load與E2仍待完成。

## 2026-08-21：玩家第29戰 raw ch28 post 正式 E1 閉合

- 依「RE→規格→實作」先在
  [`fd2_ch27_ch28_pre_owner_ida.txt`](../data/ida/fd2_ch27_ch28_pre_owner_ida.txt)
  固定第29戰入口 camera `(9,56)`、absolute `(15,63)`、visible `(6,7)`、
  selector0 與 controller gate B=1；再在 SDD 建立戰後 pan/focus 必須連續
  同步六欄 `NativeMapViewState` 的契約。
- `battle_ch29` 現以可編輯 `native_map_view` 與
  `native_map_hud_inherited` 物化視圖、range selector 與持續 HUD。beat pan
  不再只改 Ebiten camera，focus 也不再用 `absolute-camera` 覆蓋原版
  故意保留的 visible cursor。
- 正式 `postbattle_ch29_persist` binding 消費 `0x35BBA→0x1DB65`
  13＋6＋6呈現、group9、`0x25535→0x22253`、`0x24B4D`、`0x35E5A`、
  隊伍同步與 chapter increment，最後進 `preparation_ch30`。此節點的存檔／
  讀檔回歸保留 roster、join order 與 chapter29。
- Docker／Xvfb 聚焦回歸
  `TestChapter29BattleResultRunsCh28PostToPreparation30AndSaveLoad`、視圖同步與
  campaign 資料測試全數通過。postbattle audit 現為24 active／0 blocked。
- 這是 `RUNTIME-E1`，不是未修改原版一般玩家 `PLAYER-E2`，也不命名
  FDOTHER #5 entries `0x44..0x4F` 或 UI SFX sample index3的未證實玩家語意。
- 本輪再以唯讀子代理程式稽核 source comments 與 Markdown；程式註解無新的
  高信心錯誤，權威現況文件中的23／1、未綁定、presenter未接等斷言已修正；
  時間序列歷史則保留並明示由本條取代。

## 2026-08-21：第30戰正式終局來源約束 E1

- 依 RE→規格→實作順序，先新增
  [`fd2_ch29_tail_nonzero_renderer_ida.txt`](../data/ida/fd2_ch29_tail_nonzero_renderer_ida.txt)，
  用 IDA Pro 9.4 與固定雜湊證明 `0x2C2A6` 的20次 `0x28A6C(0,1)` 均走
  `[0x540FF]!=0` 視覺分支、兩次固定配對與非戰鬥結果消費邊界。
- SDD 已建立全有或全無的20段 renderer 契約；`figani.Animation` 開始保存 raw
  header byte1。現有 `MontageTailPlayer` 仍是來源約束 E1 視覺橋接，
  尚未實作精確 header byte1 工作區、raw `+4..+7`與兩次配對合成。
- `battle_ch30` 現以 `ending_party_snapshot_on_win` 原子同步最後隊伍，再進入
  `source_bound_e1_terminal_hold`；正式 campaign 不再需要 `FD2_APPROXIMATE=1`。
  必要資產或 raw roster provenance 不足時，整批拒絕並回到可編輯結語；
  成功時播放前綴、角色蒙太奇、20段尾段並停在 FDOTHER #59。
- 程式註解與 Markdown 稽核同步撤回「`postbattle_ch29_persist` 仍失敗即關閉」、
  「終局只在 `FD2_APPROXIMATE=1` 消費」與「`0x2C194` 不是 campaign binding」
  等舊現況斷言；歷史段落保留勘誤原因。原版 owner、精確 renderer／音訊／
  輸入與一般玩家 E2 仍未閉合。

## 2026-08-22：終局20段可達配對 renderer E1

- 本節取代上一節「尚未實作精確 header byte1 工作區、raw `+4..+7` 與兩次
  配對合成」的當時狀態；舊段保留作為時間序列，不再代表目前現況。
- 合法 IDA Pro 9.4 重讀 `0x29164`、`0x2939D`、`0x29C90`、`0x29DED`、
  `0x4E63D` 與 `0x4E916`，並以 Docker Capstone 交叉核對 operands／tables。
  輸入仍綁定 `docs/data/fd2-reference-files.json` 的 `FD2.EXE` 雜湊；非破壞性
  證據追加於 `fd2_ch29_tail_nonzero_renderer_ida.txt`。
- FIGANI header 現確認 byte0 是總 frame 數、byte1 是前段分支旗標、byte2 是
  旗標非零時的前段 frame 數，不能再解析為 little-endian `u16`。終局20組
  選到的80個 FIGANI 實檔全部為 byte1=0、byte2=byte0，因此只接原版實際可達
  的 header-zero 主迴圈；非零前段仍失敗即關閉。
- `MontageTailPlayer` 現逐 descriptor raw `+6` 執行 inner present，每次推進配對
  base scheduler；raw `+7 bit0` 決定 auxiliary／base 層序，raw `+4` 重設
  位移 phase 並讓第一個 inner present 的不透明像素改為 palette index33，第二次
  配對在最後一個 raw `+4!=0` frame 後結束。兩次配對保持 record0 auxiliary＋
  record1 base，再 record1 auxiliary＋record0 base。raw `+5` 只保留 marker，
  因聲音 owner globals 尚未閉合而不猜接音訊。
- `figani.Frame.BlitTranslated` 新增原子裁切與不透明像素覆寫；測試涵蓋 header
  byte1 不得污染 frame count、真實資源、raw inner 次數、base scheduler、層序、
  palette33／位移、最後 effect 終止與非零前段拒絕。Docker／Xvfb 全套
  `go test ./... -count=1` 通過。
- 現況仍是來源約束 `RUNTIME-E1`，不是 `PLAYER-E2`。未閉合項縮小為
  `0x2C548→0x2C2A6` call-time records／globals 動態連續性、原版3% RNG重播、
  精確音訊與終端輸入、`0x2BCE5` 原版 owner，以及未修改一般玩家第30戰驗證。

## 2026-08-22：共用空游標 END 正式 E1

- 先以合法 IDA Pro 9.4 重讀 `sub_117E7`，再用 Docker Capstone 交叉核對
  `0x118B3..0x118CC`：main `0x25DCE` 的共用玩家控制器先呼叫 `sub_12C0D`，
  回傳 `-1` 才進 `sub_16F55`，非負 index 則走單位分支。這推翻把空游標
  系統面板限制為 chapter0 CONTINUE 一次性特例的舊實作邊界。
- 新主證據 `fd2_117e7_empty_cursor_system_overlay_ida.txt` 先建立，SDD 再定義
  完整 FDOTHER #2、四幀 opening／closing、direction3-only END、取消零修改與
  十二幀近似延遲的失敗即關閉契約，之後才修改 runtime。
- `nativeSystemCursorOverlay` 現由所有正式 battle 的空格確認共用；缺任一
  `[21,15,18,12]` 原始圖格時不建立不可見熱區。Down→END 仍沿已閉合的
  FDTXT `0x1A3／0x1A4`、YES、`0xC8` 邊界與既有敵方回合；其他三格不猜 owner。
- chapter0 CONTINUE 的 `nativeContinueOpeningConfirm` 只保留其存檔第一個 Return
  證據；面板與 END 狀態已改用共用命名。聚焦 Docker／Xvfb 回歸涵蓋一般戰場
  開啟、缺圖格拒絕、CONTINUE handoff、四幀 close、確認／取消、敵方回合與返回。
- 同批清除 README／FIGANI 格式／終局註解、歷史「只能按 Tab」與商店收件者
  input仍未接等錯誤現況斷言。這仍是 `RUNTIME-E1`；確認框 indexed renderer、
  其餘三格 owner與逐章一般玩家 E2 尚未閉合。

## 2026-08-22：END 確認框原版索引生命週期接通

- `RE-CLOSED`：合法 IDA Pro 9.4 與未修改原版 chapter0 一般玩家擷取，固定
  `0x16F55` 的 `0x1956B(0x4B)→0x15F84(FDTXT 0x1A3)→0x19953→
  0x197E5`，以及接受 `0x1A4`／取消 `0x19C`、`0xC8` ms、`0x196CB` 與
  接受分支 `0x1A30B`。主證據為
  [`fd2_end_turn_confirmation_ui_ida.txt`](../data/ida/fd2_end_turn_confirmation_ui_ida.txt)。
- `RUNTIME-E1`：正式 battle 現以原版 DATO #75、FDOTHER #5/#2、FDTXT 與目前
  320×200 indexed map source，跑完6個對話展開、4個選項展開、YES／NO、4個選項
  收合、十二幀回應、5個對話收合及來源復原。缺任一資產時在命令框關閉前拒絕；
  只有接受分支在來源復原後進敵方回合，取消保持回合零修改。
- 文件勘誤：舊「確認仍是泛用文字層」已標成歷史快照；`56` 內早期終局
  standalone／campaign blocked 段落也加上歷史標籤，避免覆蓋目前 E1 現況。
- 尚未提升：同一 raw 戰況的原版／重製逐幀差分、精確 DOS tick 與音訊 owner
  仍缺，因此不是完整 UI E2。

## 2026-08-22：END 問句同狀態比較與 `0x9017` 勘誤

- 一般玩家鍵盤路徑實跑揭露兩個真實缺陷：question／accepted／canceled 共用底層
  slice，造成三段文字疊印；DATO #75 又把`0x9017`錯當左端，肖像遮住問句。
- 固定雜湊原版的`0x4E8E1`直接指令顯示每列由傳入的`EDI`右端向左寫入；未修改
  DOSBox 擷取也顯示80像素肖像位於左側，故舊「`0x9017`是左端／只露出右側
  『戰場嗎？』」解釋已被否定。歷史段落保留，但不得再作目前規格。
- 重製端已讓三個畫面各自擁有不可變 backing slice，並新增共用DATO右往左blit。
  同一存檔問句畫面的肖像與文字子區域差異均為0；全幀仍有1692個差異像素，
  來自選項脈動與戰場動畫相位。因此原版是`PLAYER-E2`錨點，重製仍是決定性
  `RUNTIME-E1`，不冒稱逐像素完整一致。證據見
  [`native-end-turn-confirmation-original-vs-remake-e1.json`](../data/ui-traces/native-end-turn-confirmation-original-vs-remake-e1.json)。

## 2026-08-22：END 接受／取消逐字回覆正式接通

- 未修改原版 chapter0 CONTINUE 一般玩家路徑的密集擷取直接顯示：接受回覆由
  「好」逐步長成完整句，取消回覆由「是」逐步長成完整句；這推翻把 `0x15F84`
  所有 caller 都概括為整句瞬間出現的舊說法。原始位址、雜湊與限制已追加到
  `fd2_end_turn_confirmation_ui_ida.txt`，歷史過場觀察則保留並縮小適用範圍。
- 依先建立的 SDD 契約，`NativeBattleEndTurnResponseFrames` 現對每個普通字形發布
  一個獨立 indexed frame，`0xFFFE` 只換行；正式 action overlay runtime 必須先
  跑完這批回覆畫面，才開始十二個60 Hz畫格的近似等待與五幀對話框收合。
- 重製以同一份合法 `FD2.SAV`、`campaign_full.json` 及普通 X11 鍵盤事件完成兩條
  正常路徑：接受完整回覆後進敵方回合，取消完整回覆後回玩家戰場。兩張接觸表
  的上半為原版、下半為重製，證據與雜湊見
  [`native-end-turn-response-progressive-original-vs-remake-e1.json`](../data/ui-traces/native-end-turn-response-progressive-original-vs-remake-e1.json)。
- 這批結果關閉逐字形內容順序與重製正常輸入可達性，未提升成逐像素／逐毫秒
  parity；精確 DOS tick、音訊與收合的同動畫相位 E2 仍保留待驗。

## 2026-08-22：教會轉職跨 town 存讀檔邊界

- 先重讀既有 `0x31793` 目標表、`0x314A7..0x3157A` 寫入、職業顯示與裝備
  消費證據，不重做已閉合反組譯；SDD 隨後定義正式轉職 mutation 必須經
  `leaveChurch` 回 town，才可寫入重製自有 JSON 存檔。
- `TestCampaignTownChurchClassChangeReturnTrace` 現由 town 選單實際進教會，解析
  悠妮的特殊轉職分支，呼叫 `applyChurchClassChange`，返回 town 後存檔；清空記憶
  狀態再讀檔，逐項核對 portrait、class／raw class、battle figure、map selector、
  成長值、EXP、HP／MP、背包、裝備旗標與重算基礎、membership、join order、
  deployment、金幣、道具及 campaign cursor。
- 首次回歸揭露 `loadGameFromSlot` 只清戰場，未清讀檔前的 `churchMode`、候選與
  indexed 工作；正式讀檔邊界現先清除全部教會暫態，再進入存檔節點。這是
  `RUNTIME-E1` 的重製 JSON 持久化閉合，不外推成原版四槽 `FD2.SAV` 或 DOSBox E2。

## 2026-08-22：原始碼註解與首頁完成度斷言勘誤

- 交叉核對 `56`／`57`／`58`、測試與正式消費端後，撤回 README 的「原生
  `FD2.SAV` 四槽 LOAD 已能還原到具型別隊伍及城鎮／整備邊界」說法。現有證據是
  原版 envelope／metadata 驗證，加上合成有效槽 fixture 的 typed restore
  `RUNTIME-E1`；尚未以未修改原版的有效槽完成一般玩家 LOAD E2。
- `growth_test.go` 舊註解所稱「轉職系統尚未實作」已被正式教會轉職 owner 與
  跨 town JSON 存讀檔回歸推翻；該測試只負責驗證多職業成長列。
- `native_ai_runtime.go` 的 mode 2 無候選分支仍未綁到 `0x13FD4` owner，但 mode 11
  的 `0x14121→0x13FD4` 正式畫面／音訊消費端已達 E1。註解現已限定分支，不再把
  局部缺口誤寫成整個 `0x13FD4` 尚未接入。

## 2026-08-22：商店裝備／轉移跨 town 存讀檔邊界

- 先重讀既有 `0x2F883→0x1C142→0x1B750` 獨立裝備（standalone-equip）與
  `0x2F8EA` 轉移證據，SDD 再定義正式 mutation 必須經 `leaveShop` 回 town，
  才能寫入重製 JSON；沒有重開已閉合 IDA 位址。
- `TestNativeShopProductionOwnerDrawsOriginalMenuAndPurchaseList` 現沿正式 indexed
  擁有者（owner）完成裝備與自身轉移（self-transfer），再穿越離店、town、槽位3存檔、清空記憶狀態與
  冷讀檔。回歸核對緊密背包投影（compact inventory）／equipped、八格 `InventorySlots`／
  `NativeInventoryFlags`、AP、equipment base、membership、join order、deployment、
  chapter 與 campaign cursor。
- 首次回歸直接重現讀檔污染：`nativeShopMode="transfer_full"`、variant5、舊 transfer
  source 與 item panel 全部殘留。`loadGameFromSlot` 現在集中清除購買／賣出／裝備／
  轉移選擇器、待提交交易（pending transaction）、索引工作（indexed job）、item panel 與一般 shop 暫態後才
  進入存檔 town。這是重製 JSON `RUNTIME-E1`，不外推成原版 `FD2.SAV` 或商店 E2。

## 2026-08-22：商店購買／賣出跨 town 存讀檔邊界

- SDD 先固定既有正式提交點：購買在物品 staged publish 與 `0x2F4C6` 成功演出後，
  才由 `0x2D516` 金幣滾動擁有者扣款；賣出在成功演出後才依
  `0x2D3FF→0x1B8E7→0x1B750` 發布金幣、移除物品與重算。沒有重做 RE。
- 原始資產 production 測試現於購買 `1234→1134` 及賣出 `100→137` 的 callback
  完成後，各自實際穿越 `leaveShop→town→JSON save/load`。購買保留第一個 raw
  空格中的未裝備 item；賣出保留左移後緊密背包、尾格 `0x80` flag／`0xFF` item、
  裝備投影與能力，並同時核對 membership、join order、deployment 與 chapter。
- 這關閉商店四種正式 mutation 的重製 JSON `RUNTIME-E1` 持久化；未修改原版
  `FD2.SAV`、recipient scroll、no-recipient/full及sell/equip/transfer子面板E2仍待。

## 2026-08-22：ch02 賣出角色／物品子面板同狀態 E2

- 先讀既有 `0x2F642..0x2F87C` 賣出證據並在 SDD 固定 E2 契約，沒有重開已閉合
  RE。原版使用固定雜湊檔案的可丟棄副本，只套用已驗證三處戰鬥略過 patch；
  由 ch02 town 經正常 Left／Enter／Right／Enter 進入賣出名冊，再以 Enter 進索爾
  物品清單。故證據是 route-patched E2，不是未修改一般玩家戰後路徑。
- 高頻原版樣本揭露賣出名冊也消費 FDICON 三個 cycle；舊 production compositor
  寫死 cycle0。新增 `FD2_SHOT_SHOP_SELL_STATE` 受限 adapter，cycle成為明確
  renderer state；它只接受完整 typed/raw binding party，拒絕無效 unit、window、
  cycle或缺少資源的狀態，且失敗原子回復。未證實的正常 runtime cadence仍不猜。
- 以 `waitpixel:175,90,101,121,121,0.05,400` 同步人物相位後，角色名冊
  selection0/cycle1與索爾物品selection0分別取得完整320×200 `AE=0`；raw RGB
  MD5為`62307f5918f1de723055f951a7e6dc6a`與
  `f5ff2d3650575a93bcc0d795fae7c4ea`。證據見
  [`shop-sell-child-ch02-e2.json`](../data/ui-traces/shop-sell-child-ch02-e2.json)及
  [`shop-sell-child-ch02-original-vs-remake.png`](../figures/shop-sell-child-ch02-original-vs-remake.png)。
- 本切片只關閉兩個 stable child states；selection1、問句、取消／成功 lifecycle、
  未修改campaign/native save、equip／transfer與recipient scroll仍待。

## 2026-08-22：ch02 賣出 selection1 與 Yes／No E2

- 沿用同一固定雜湊與三處 route patch，原版從賣出名冊 selection0 實際 Right 到
  悠妮，再 Left 回索爾；selection1/cycle1與production完整320×200 `AE=0`。
- 選索爾、短劍後原版問句顯示37元。Yes selected與Right後No selected高頻樣本
  分別對production choice0/pulse2、choice1/pulse2整幀`AE=0`。新增受限
  `FD2_SHOT_SHOP_SELL_CONFIRM_STATE=unit,item,choice,pulse,gold`；只接受真實active
  raw item，價格仍由effect row的75%計算，非法狀態會原子回復。
- 三組raw RGB MD5依序為`d63d213c835f59c1a60428ef6a14d7ad`、
  `38bd7527570c0ddbf819f19eeea71685`、`6168c00b515ffe33e14a658bd7932d42`。
  證據見[`shop-sell-selection-confirm-ch02-e2.json`](../data/ui-traces/shop-sell-selection-confirm-ch02-e2.json)
  與[`shop-sell-selection-confirm-ch02-original-vs-remake.png`](../figures/shop-sell-selection-confirm-ch02-original-vs-remake.png)。
- 本批關閉selection0↔1及Yes／No stable/input E2；success timeline、提交後返回名冊、
  未修改campaign/native save仍待。

## 2026-08-22：原始碼與商店完成度斷言第二輪勘誤

- `growth_test.go` 雖已撤回「轉職系統尚未實作」，`growth.go` 的資料表註解仍殘留
  同一句舊狀態。本輪改為限定該表只擁有升級成長；正式教會轉職由 campaign／UI
  owner 實作，不能由 battle package 的表格註解反向判定整個功能不存在。
- `Beat` 與 cutscene `Node` 的型別註解原稱「一比一」承接原版 handler，範圍過廣。
  現改為「可逐拍承接已有直接證據的序列」；型別可表達某個 op，不等於每個章節
  handler 已完整閉合或達到一般玩家 E2。
- UI-09 原狀態欄把購買成功動畫／扣款原子影格 E2 與賣出名冊／問句 E2 串成
  `sell ... success/debit stable E2`，同列又把 sell success／return 列為待辦，容易
  形成互相矛盾的記憶。現明確限定 success／debit 原子影格屬購買路徑；賣出目前只
  關閉名冊、物品與 Yes／No stable state，成功時間線與返回名冊仍待驗證。
  **本句是當時狀態，已由緊接的「ch02 賣出成功、向上金幣滾動與返回名冊
  E2」取代；不得再以本句重開 sell success／return。**

## 2026-08-22：ch02 賣出成功、向上金幣滾動與返回名冊 E2

- 先在 SDD 固定成功→加款→背包發布／返回三個邊界，再沿固定雜湊原版的可丟棄
  route-patched 副本，以正常標題、城鎮、武器店與賣出輸入按下短劍 Yes；未修改
  `FD2.EXE` 與原始資產仍保持唯讀，因此證據是 route-patched E2，不是完整戰役 E2。
- 密集擷取證實成功演出期間金額維持0，之後可見向上滾動 `0→11→33→34→35→37`，
  約0.8秒後返回索爾名冊且selection不重設；重新進入時短劍已移除，皮甲與藥草保留。
- 合法 Docker 內 IDA Pro 9.4 以既有 `.i64` 非破壞性續讀：`0x2D3FF` 先增加
  `dword_53BF3`，每個不同digit再畫phase1..9並十進位wrap；`0x2D620`只是九列×六像素
  blitter。證據綁定固定雜湊、IDA線性位址及caller／consumer，見
  [`fd2_shop_gold_credit_ida.txt`](../data/ida/fd2_shop_gold_credit_ida.txt)。
- 新增具型別 `ComposeNativeGoldCreditFrames` 與正式三段owner：成功動畫只持有
  staged transaction；第一個callback發布gold並開始10 ms credit timeline；最後
  callback才發布移除後背包並返回原actor名冊。資源或projection不足時整筆失敗即關閉。
- 五個成功影格、gold11／36與返回cycle0／1共九組原版／重製完整320×200 RGB
  `AE=0`；證據、雜湊與受限截圖入口見
  [`shop-sell-success-return-ch02-e2.json`](../data/ui-traces/shop-sell-success-return-ch02-e2.json)
  及[`對照圖`](../figures/shop-sell-success-return-ch02-original-vs-remake.png)。
- 本批沒有證明未修改戰役、原版存檔、成功影格的精確停留時間、recipient scroll、
  no-recipient/full、equip/transfer子面板或其他章節；這些仍依 `57` 保持 partial。

## 2026-08-22：原始碼註解與 Markdown 現況斷言第三輪勘誤

- 先逐項重核前輪候選：轉職、postbattle fallback、戰鬥音效E1近似及mode2
  `0x13FD4`註解都已限定實際路徑，現況正確，未為了縮短文件而刪除有效限制。
- `drawNativeBattleStatus` 使用原版6×8 digit cell與7px advance，但舊註解把局部素材
  消費直接寫成「100%還原」。現改為只聲明素材與局部幾何；完整狀態欄仍依UI矩陣
  的同狀態畫面與E2驗收，不可由程式註解自行升格。
- `18-font-modernization-utf8-ttf-plan.md` 同樣把FDOTHER#4 16×16 atlas寫成
  「100%還原1995畫面」。現更正為忠實取用原版字模；文字布局、陰影、裁切、分頁
  與整幀結果仍逐狀態驗收。1824／1824 glyph mapping與round-trip等有明確分母的
  資料覆蓋數字保留，不與畫面完成度混為一談。
- 商店賣出「成功時間線與返回仍待」只剩交接檔上一輪的歷史狀態，後方已有本日
  新E2勘誤；依時間序列追溯規則保留原句，不把歷史改寫成當時已完成。

## 2026-08-22：商店裝備收件者正常輸入 E1

- 先重讀既有 `0x2F0B0` 購買、三列收件者、滿欄／無合適角色及交易owner；這些
  RE與compositor早已閉合，本輪沒有重解位址。SDD先固定正常輸入、UI job發布
  邊界、原子交易與E1／E2限制，再抽出正式收件者typed input consumer。
- `Game.Update`在`recipient_equipment`／`recipient_consumable`／`recipient_full`／
  `no_recipient`仍讀Ebiten普通鍵盤，但同拍Enter／Escape／方向鍵現在收斂成不可變
  值後交給同一consumer；非收件者mode會拒絕，測試不再直接改mode冒充輸入。
- 新production regression以六名具identity、selector、class與八格raw背包provenance
  的typed party，從正式`setupNativeShop`走`menu→purchase→Yes→recipient`。
  Down三次閉合selection3／start1，水平鍵不動；非滿欄角色再走equip confirmation、
  success與`0x2D516`扣款，最後以1134元、已插入且裝備的item0返回purchase。
- 第二、三條路徑分別以第四名八格滿欄及全員不相容驗證`recipient_full`與
  `no_recipient`；opening／closing／restore逐幀經正式Draw acknowledgment，兩條
  feedback返回後gold、roster與pending transaction均原子不變。
- 第四條路徑由recipient直接按Escape，五幀收合與六幀purchase重開後，金額、
  六人roster與pending transaction同樣不變。聚焦四條Docker／Xvfb測試均通過。
  這是synthetic typed party的production-input `RUNTIME-E1`；測試直接驅動正式
  consumer，未注入OS鍵盤，也未從完整campaign抵達商店，因此不是未修改原版一般
  玩家E2。原版四人以上scroll／full／no-recipient同狀態畫面與equip/transfer child
  panel仍待。

## 2026-08-22：ch02 獨立裝備名冊／索爾面板 partial E2

- 先以 SDD 固定 `0x2F883→0x1BFFE→0x17E0B` 既有 owner 的同狀態契約，
  本輪未重解已閉合位址。新增的 screenshot-only adapter 只能由
  `FD2_SHOT_PARTY_BINDING` 建立的完整 typed/raw party 選取角色與 occupied
  item slot，並呼叫正式名冊／item-panel owner；非法 window 或資料失敗時原子拒絕。
- 固定雜湊原版的可拋棄 route-patched 副本，以正常標題、城鎮、武器店、
  `Right×2→service2→索爾` 輸入取得名冊 selection0 與索爾完整item/status
  panel；名稱、職業、金額、能力、背包與裝備效果均與重製正式 renderer 相符。
- 完整320×200 RGB比較未達AE=0：名冊 `AE=1389`集中在四名角色動畫精靈，
  索爾面板 `AE=1433`包含呈現相位／面板邊緣。嘗試補抓高頻相位時，前一次
  可拋棄 sandbox 已無法重現同一城鎮簽章，因此有界停止，沒有猜測修改 renderer。
- 本批是 `PLAYER-E2 route-patched partial`，不證明mutation／restore、service3 transfer、
  原版存檔或未修改完整campaign。證據見
  [`shop-standalone-equip-ch02-e2.json`](../data/ui-traces/shop-standalone-equip-ch02-e2.json)
  與[` 對照圖`](../figures/shop-standalone-equip-ch02-original-vs-remake.png)。

## 2026-08-22：原始碼註解與 Markdown 現況斷言第四輪勘誤

- 以 `58→56→57→91` 現況順序重核程式註解、README、歷史 gap audit 與交接檔。
  `growth_test.go` 與 mode2 `0x13FD4` 已有正確路徑限定，不採用子代理程式的
  過時行號回報，也不刪除有效的失敗即關閉註記。
- `atkAnim` 與 `0x16D00` 嘴型註解原本分別以「對照原版」與「忠實原版」
  概括尚未閉合的 DAC／調色盤／逐幀時序。現明寫為重製 E1 與已知節奏，
  不得由註解自行提升為完整 E2。
- `42` 保留早期 HIT／EV 來源缺口，但撤回「remake 尚無裝備系統」；
  現況是裝備與 shop equip owner 已存在，尚缺完整 native stat source 與戰鬥整合。
- README 改為明寫「賣出子面板圖本身」不包成功／返回，而該後續現已有
  九組 route-patched E2；歷史 `91` 與本檔舊「仍待 sell success」句均追加取代關係，
  避免單純搜尋把已閉合切片重開。未修改一般玩家路徑仍是獨立缺口。

## 2026-08-22：ch02 物品轉移五狀態 partial E2

- 先重讀既有 `0x2F8EA` RE 與正式 transfer owner，於 SDD 固定同狀態契約後才
  實作截圖入口；本輪沒有重解已閉合函式。入口只接受
  `intro|source|items|dest_prompt|dest,source,item,selection,start,gold`，由
  `FD2_SHOT_PARTY_BINDING` 的完整 typed/raw party 推導角色與物品，非法 window、
  raw provenance 或 compositor 失敗時原子拒絕，且不執行任何背包 mutation。
- 固定雜湊原版的可拋棄 route-patched 副本，由正常標題、城鎮、武器店與
  `Right×3→service3` 鍵盤輸入取得 FDTXT512、來源名冊、索爾物品、FDTXT510
  與目的名冊；目的名冊依原版保留來源索爾。
- 五組320×200整幀 AE 依序為 `88／1391／2／88／321`。提示差異是翻頁箭頭
  相位，名冊差異是角色小圖相位；文字、名冊、物品、效果、價格與幾何一致。
  這是 `PLAYER-E2 route-patched partial`，不是 AE=0 或未修改一般玩家 E2。
- 證據見
  [`shop-transfer-ch02-e2.json`](../data/ui-traces/shop-transfer-ch02-e2.json)與
  [`對照圖`](../figures/shop-transfer-ch02-original-vs-remake.png)。本批不證明
  mutation、empty/full、cancel/restore、原版存檔、church caller 或其他章節。

## 2026-08-22：原始碼註解與 Markdown 現況斷言第五輪勘誤

- 以 `README→58→56→57→91→handoff` 與實際消費端重核高風險措辭。
  `title.go` 舊稱 cutScript 為「忠實／反組譯真值」，但 `39 §10` 已明確推翻
  舊幕序，現改標 E1 並保留交錯捲動、完整幕序與 DAC 時序缺口；啟動註解也不再
  把有 ANI.DAT 說成完整原版開場。
- 天空之鑰測試直接以 ch21 scenario 建立隊伍投影，現明寫不等價於已執行上一個
  整備或一般玩家路徑。戰鬥畫面註解只保留已證實的 figure／狀態欄繪製順序，
  不再稱動畫完整；storyZoom、FDICON 單位、TTF 字形與 SETSOUND 註解也撤回
  「常數即還原、所有繁中字必有 glyph、預錄 OGG 等於原版驅動」等過度概括。
- `42/56/57` 現況摘要與 `91` 歷史搜尋入口已同步：service3提示／來源名冊／物品／
  目的提示／名冊五狀態已有route-patched partial E2，不再列為 transfer child panel
  未做。仍缺的是動畫相位、service3 mutation／empty／full、church caller、其他章節、
  原版存檔與未修改一般玩家路徑；沒有因文件勘誤把這些真實缺口刪掉。

## 2026-08-22：ch02 物品轉移成功交易 partial E2

- 先以 SDD 固定成功 transaction、返回 loop、雙方清單與 JSON 邊界，再沿用已閉合
  `0x2F8EA`／`0x1B8E7→0x1BB8C→0x1B750`；沒有重解 callee。新增截圖入口只在
  私有 party map 呼叫 production owner，任一索引／raw projection／compositor
  拒絕時公開 roster 原子不變。
- 固定雜湊原版 route-patched 副本由正常鍵盤選索爾、短劍與悠妮。Enter 交易後
  自動回 FDTXT512；再選索爾只見皮甲／藥草。從該清單按 Escape 返回 loop 後改選
  悠妮，可見長棍／長袍／未裝備短劍，金幣保持0。
- 目的悠妮、返回提示、索爾結果、悠妮結果四組320×200 AE依序為
  `1391／82／2／286`；內容與幾何一致，差異為角色、翻頁箭頭或物品選取脈動相位。
  證據見
  [`shop-transfer-success-ch02-e2.json`](../data/ui-traces/shop-transfer-success-ch02-e2.json)
  與[`對照圖`](../figures/shop-transfer-success-ch02-original-vs-remake.png)。
- 正式回歸另把同一跨角色交易穿越`leaveShop→town_ch02→JSON save/load`，冷清空
  記憶後仍保存雙方 compact/raw 背包、裝備、AP、金幣、隊伍順序與節點，且清除
  transfer暫態。這是重製JSON `RUNTIME-E1`；self／empty／full／destination-cancel、
  church caller、原版存檔與未修改一般玩家路徑仍未外推。

## 2026-08-22：ch02 目的取消與自我轉移正式生命週期

- 先以 SDD 固定目的名冊取消與 self-transfer 契約，沒有重解已閉合的
  `0x2F8EA`／`0x1B8E7→0x1BB8C→0x1B750`。目的 Escape 現由
  `beginNativeShopTransferDestinationCancel` 單一擁有：五幀名冊收合完成後，
  才以既有來源 loop 建立六幀 FDTXT512；合成失敗不再略過動畫繼續前進。
- 原始資產整合回歸證實取消前後完整 unit 與 gold 不變，錯誤 mode 不能啟動；
  取消返回後重新走 source→items→destination 選來源本人，仍由正式 raw
  remove→append／重算得到未裝備尾項。完整 Docker／Xvfb `go test ./...` 通過。
- 這批只提升重製端 `RUNTIME-E1`，沒有新增 DOSBox 畫面；因此 self／取消的原版
  同狀態 E2，以及 empty／full、church caller、原版存檔與未修改玩家路徑仍保留。
- README 新增以玩家交付門檻計算的自我評估：目前是中後段整合期，保守估計仍有
  30–40% 的產品整合與驗收工作；數字不是 EXE 反組譯率或像素相似率。

## 2026-08-22：標題 LOAD 到原版章節槽正式確認 owner

- 先以 SDD 固定四槽 selector 的原子確認契約，未重解既有
  `0x2602C..0x26098` reader、`0x30012` writer、envelope 或 chapter gate。
  `confirmTitleLoadSlot` 現是 Enter／Space 共用 owner；只有完整 native／JSON
  restore 成功才離開 `loadslots`。
- checksum-valid 合成原版槽由正式 selector 還原到 `town_ch02`，一次發布悠妮
  typed/raw record、join order、789 金幣、chapter1 與 HUD gate，並清除舊 battle
  state／selection。竄改 envelope 保持 selector、campaign、gold、party、battle
  state 與 handler chapter 原子不變，錯誤訊息也不再把 checksum 失敗說成空槽。
- 聚焦 Docker／Xvfb 回歸通過。這是合成 fixture 的 `RUNTIME-E1`，不是由未修改
  原版酒店／整備寫槽後再經標題 LOAD 的一般玩家 E2；該 gate 仍保留。

## 2026-08-22：service2 裝備交易原子發布

- 先依既有 `0x2F883→0x1BFFE→0x1C142→0x1B750` 證據在 SDD 固定原子契約，
  沒有重解 callee。稽核發現 `applyNativeShopEquipSelection` 原先先寫
  `partyRoster`，之後才重建 item/status panel；深層 renderer 失敗會留下已改裝
  角色與舊畫面，違反失敗即關閉。
- 正式 owner 現在私有 unit 上完成 raw／compact 驗證、裝備與能力重算，再完整
  建立候選 panel，最後才一次發布 roster。整合回歸刻意在最後移除 palette：函式
  失敗時 unit、裝備旗標、能力、selection、既有 panel image／buffers 均不變；恢復
  palette 後同一輸入才成功，既有收合、名冊重開與 JSON round-trip 仍通過。
- 原版 route-patched 擷取有界重跑後確認：目前 `FD2.SAV／TMP` 的 CONTINUE 只進
  戰場角色名冊／狀態面板，Return 在兩者間循環，Escape 回標題，無法抵達舊 ch02
  城鎮簽章。停止重播，不以直接注入替代；service2 mutation／restore 仍是原版 E2
  缺口，本批裁決只到 `RUNTIME-E1`。

## 2026-08-22：command 17–19 原子交易與敵方 mode 11 消費端

- 先重讀既有 IDA 證據與 SDD family matrix，沒有重解
  `0x226EA／0x2282F／0x22960` 或其 writer。規格明確保留 ID17 使用 record17
  selector、但由 record18 debit MP；ID18／19 分別使用自己的 debit record。
- 新增 `ExecuteNativeCommandModifier`／`ExecuteNativeAICommandModifier`：final targets
  先投影到私有 `0x50`-byte records，包含 raw class／level、六個 transient bytes 與
  derived words；整批 `ApplyNativeCommandModifier` 成功後才一次發布 target、MP與
  acted。缺record、selector、raw provenance、有效target、MP或16-bit word時零修改。
- 敵方 `executeNativeAIAction` 現正式消費 IDs17–19，更新process-lifetime 16-bit RNG，
  再走既有成功動作／AI continuation 邊界。玩家grid、status名稱、專用indexed演出、
  SFX與phase-expiry caller沒有被猜測接入，仍失敗即關閉。
- `internal/battle` 與 `cmd/fd2` 完整 Docker／Xvfb 套件回歸通過；本批提升的是窄
  `RUNTIME-E1`，不是同狀態原版逐幀／逐音訊 E2。

## 2026-08-22：command 17–19 玩家色盤演出與正式指令入口

- 先以授權 Docker 內 IDA Pro 9.4 重新定位尚未閉合的玩家演出 owner，而非重解
  已閉合的 modifier writer。固定雜湊 `FD2.EXE` 的 `0x1D6C8..0x1D79C`
  先播放 `FDOTHER #88` 子音效0，再由三張36-byte六位元 DAC 表依 command ID
  取色，執行四輪「command color→black」；唯一直接 caller 位於函式
  `0x1CFF0`。原始位址、表格、caller與推論分級保存在
  [`fd2_command_modifier_palette_ida.txt`](../data/ida/fd2_command_modifier_palette_ida.txt)。
- 三張原始 DAC 表已轉成可編輯
  `native_command_palette_flash.json`。正式玩家 command 17–19 入口現先驗證
  framebuffer、baseline DAC、typed table、原始 records、目標與精確音效；任一缺漏
  都不扣 MP、不改 target、不標 acted。每個 phase 必須實際經過 Draw，八幀完成後
  才原子發布既有 modifier transaction；敵方 mode 11 不誤套玩家 palette owner。
- 聚焦回歸覆蓋缺 framebuffer、缺 sample、八幀發布邊界與 ID17 的
  record18 MP debit。此切片提升玩家與敵方 command 17–19 至 `RUNTIME-E1`；尚未
  證實的 phase-expiry正式 caller、status UI、精確 DOS tick及同狀態逐幀／逐音訊
  `PLAYER-E2` 仍保留，不以近似行為冒稱原版一致。

## 2026-08-22：command 20–22 共用玩家色盤 owner

- 既有 IDA `0x1CFF0→0x1D6C8` 範圍直接涵蓋 command 20–22，故不重解
  palette loop；證據檔擴充標題與 DAC 值，20／21為`(0x3F,0x28,0x1E)`，22為
  `(0x23,0,0)`。三者與17–19同樣先播放`FDOTHER #88`子音效0，再走四輪
  command-color／black。
- 玩家正式入口不再於游標確認時立即執行20／21 clear／restore或22 application。
  新增非破壞 preflight，先驗證record、final targets、MP、RNG與HP bounds；八個
  phase均經Draw後才交易並清理selection／range。深層target失敗時MP、acted、HP與
  raw interval皆保持不變；AI仍不套用玩家palette owner。
- Docker／Xvfb聚焦測試走過command20的`+0x25`清除及command22 application，完整
  `go test ./... -count=1`亦通過。本批為玩家`RUNTIME-E1`；status名稱、expiry UI、
  精確DOS tick與同狀態逐音訊`PLAYER-E2`仍未閉合。

## 2026-08-22：command 25–27 玩家色盤 owner與raw bit7勘誤

- 既有`0x1CFF0`直接分支`0x19..0x1B`證實25–27同樣先進`0x1D6C8`；沿用
  已閉合的#88 sub0與八個DAC phases，不重解palette loop。25／26／27色值分別為
  `(0x3F,0x3D,0x2E)`、`(0x0A,0x1F,0)`、`(0x23,0x19,0)`。
- 程式稽核發現`ExecuteNativeCommand25`雖在註解宣稱清raw `unit+5 bit7`，實作卻
  錯改Go `Unit.Acted`。現已要求target具`NativeRecordByte5` provenance，只清
  `0x80`且保留target `Acted`；缺raw欄位時在sample、palette與MP debit前拒絕。
- 玩家25–27現均完整preflight，八個Draw phases完成後才執行25 clear或26／27
  application，再共用range／selection cleanup。核心與玩家聚焦回歸通過；本批仍只到
  `RUNTIME-E1`，status名稱、expiry UI、精確tick與逐音訊E2未外推。

## 2026-08-22：command 23 雙 `0x22253` 玩家生命週期

- 本條取代舊交接中「command23只接座標交易、27-present renderer仍未接」的現況斷言；
  原始`0x2218A`與`0x22253`證據本身不變，也不重解callee。
- mode-6目的地確認現先在私有records驗證command23 MP／座標交易，並預建離場及
  入場兩個完整`0x22253`工作；任一raw record、terrain、palette、sample、indexed
  asset或兩段renderer前置條件缺失時，都在第一個sample／frame前失敗。
- 正式順序是`0x1D6C8`的#88 sub0與八個Draw-ack palette phases →
  `0x22253(target,0xff,0xff,currentX,currentY)` →
  `0x22253(target,destX,destY,destX,destY)` → 原子發布MP、座標、raw action bit及
  cleanup；item101維持不消耗。兩段共用目的地確認前的unit／work／VGA rollback。
- Docker／Xvfb整合回歸證實缺renderer不啟動、八段palette前不變更、可觀察
  `0xff/0xff`中間狀態及第二段完成後才交易，提升為`RUNTIME-E1`。原版同狀態
  camera、逐幀、逐音訊及一般玩家`PLAYER-E2`仍待驗收。

## 2026-08-22：command 24 selector32學習鏈與FIGANI演出

- 授權Docker內IDA Pro 9.4閉合`0x276EC`的正常selector32路徑，主證據保存於
  [`fd2_command24_presentation_ida.txt`](../data/ida/fd2_command24_presentation_ida.txt)。
  FIGANI資源98的15幀raw schedule中，frame4發布MP並播放FDOTHER #53 sub3，
  frame10一次發布完整傷害並播放sub2，尾幀才發布acted；command24分母為1，撤回
  舊「等分multi-hit」說法。
- 同輪程式稽核修正升級學習索引：原版是`unit+7→11-byte growth row byte10
  learn_idx→12-byte command row`，不是以portrait直接查command table。selector32
  經row4在Lv4授予command24；optional selector50使用不同row10。缺raw selector或
  任一資料表時維持失敗即關閉。
- 正式玩家owner現以私有damage plan預建交易，只有對應幀實際Draw後才依序發布MP、
  HP與acted；缺FIGANI／target idle／palette／panel／sample時零修改。畫面仍沿用既有
  battle background／TAI，尚未接`0x29C90`背景轉場，所以只列`RUNTIME-E1 partial`。
  未修改原版「轉職→Lv4學會→施放」連續E2、背景逐幀及精確音訊仍待驗收。

## 2026-08-22：command24 `0x29C90`兩段背景滑動

- 合法Docker內IDA Pro 9.4補閉合`sub_29C90`與caller：第一段`i=9..0`依
  BG resources `i%3`累積貼左半，再present `work640+32*i`；第二段預建target
  BG／status／idle後，依`(j+2)%3`貼右半並present同一組offset。兩段各10次，
  函式內沒有BIOS wait。command24的`0x52363`初值是0，故single target必取
  target格raw FDSHAP control byte2。證據保存於
  [`fd2_command24_background_transition_ida.txt`](../data/ida/fd2_command24_background_transition_ida.txt)。
- 新增獨立`battlepresent` pure compositor，固定640×200 work、20個320×200
  viewport frames、BG 0/1/2順序與32-byte位移；任一base、layer或idle frame無效時
  不回傳部分結果。正式command24 owner從MapData raw tile/control載入玩家自備
  BG.DAT，預建全部frames後才允許演出與交易。
- 目前source base使用actor terrain BG＋resource98 frame8，target base使用target
  terrain BG＋target idle，狀態欄仍由既有RGBA renderer疊加；因此提升玩家可見
  滑動但仍列`RUNTIME-E1 partial`。尚缺`sub_29164/sub_2B659/sub_2A289`完整
  indexed base與一般玩家E2，不宣稱逐像素相同。

## 2026-08-22：command24 indexed狀態欄接入

- 依RE→規格→實作順序，以合法Docker內IDA Pro 9.4閉合
  `0x2A289→0x18C6D`。raw `unit+6==0`選`(0,154)`、非零選`(171,4)`，
  raw chapter24且unit index17強制下方；框是FDOTHER#5 LMI1 entry22，
  HP／MP bar使用raw entries23..30，數字使用31..52／93，姓名使用
  FDTXT resource0的`unit+8+1`與FDOTHER#4字模。主證據見
  [`fd2_battle_status_panel_ida.txt`](../data/ida/fd2_battle_status_panel_ida.txt)。
- 新增全有或全無的`RenderNativeBattlePanel`與LMI1 opaque-entry decoder；
  raw `+6/+8`、panel、bar、digits、FDTXT或font任一缺失時不發布部分畫面。
  command24的actor／target indexed base現正式消費此compositor，20張背景滑動
  不再額外疊兩塊RGBA panel；MP／HP／Acted仍保持原Draw邊界。
- 本切片是`RUNTIME-E1 partial`。`0x29164／0x2B659`完整source base、精確
  palette／音訊與未修改一般玩家「轉職→Lv4學會→施放」E2仍待，不以現有
  effect frame8近似冒稱逐像素一致。

## 2026-08-22：command24 actor來源畫面與target idle重設勘誤

- 本段取代上一節把`0x2B659`完整來源畫面列為未接的現況斷言；狀態欄本身的
  原始證據與位址不變。合法Docker內IDA Pro 9.4重新固定`0x2B659`在command24
  actor階段只播放frame0..header byte2-1（實檔為0..8），且command24不符合
  `command<10 || command==28`，所以actor階段不畫target idle。
- frame8後的轉場來源快照現由actor地形BG、扣MP後原生狀態欄、raw `unit+6!=0`
  時的TAI平台及resource98 frame8組成；raw `+4` marker造成的MP扣除會先反映於
  私有record，再組成來源畫面。任一BG、TAI、FIGANI、FDTXT、字模、狀態欄或raw
  provenance缺失時，在任何交易與畫面發布前失敗即關閉。
- `sub_2B9A1(targetIdle,0)`已證實會把全域待機frame／repeat同時清為0且不繪圖；
  正式owner在兩段20張背景滑動完成後套用同一重設，再由target base與idle frame0
  進入尾段。`sub_2B9A1`已加入非破壞性語意索引；1305函式現況為產品35、
  Watcom runtime170、未知1100。
- 此批仍是`RUNTIME-E1 partial`。尚未接的是`0x29164`九段figure prelude、精確
  palette／音訊，以及未修改一般玩家「轉職→Lv4學會→施放」E2；不得把已接的
  `0x2B659`來源畫面再次列為反組譯缺口。

## 2026-08-22：command24 `0x29164`九段前導正式接入

- 本段取代上一節把`0x29164`列為未接的現況斷言。合法Docker內IDA Pro 9.4
  重新匯出`0x276EC`與`0x29164`，固定command24實參為
  `(actorIndex,1,actorIdle,firstTargetIdle,work640,actorBase320,actorTAI)`；第二
  參數固定1，所以兩支raw side分支都不畫firstTargetIdle。
- raw `unit+6==0`以右viewport讓actor idle frame0由`-80`滑到0；raw非零以左
  viewport讓actor idle與實際TAI由`+80`滑到0。兩支都做stage8..0九次present，
  每張以FDOTHER#0原始6-bit DAC基線套`0x11D40`的`stage*6`減算；函式內無delay。
- 新增全有或全無的`BuildNativeCommand24PreludeFrames`與正式Ebiten owner。九張
  畫面會在effect frame0前逐Draw發布，且期間MP、HP、acted均不變；缺base、
  actor idle、raw DAC或非零分支TAI時在建立job前失敗即關閉。聚焦回歸同時覆蓋
  左右兩支像素位移、48→0 DAC、缺件拒絕與九張Draw生命週期。
- `0x29164`已加入非破壞性語意索引；1305函式現況為產品36、Watcom runtime170、
  未知1099。command24仍列`RUNTIME-E1 partial`，但剩餘缺口已縮為精確音訊、
  同狀態逐幀／palette驗證及未修改一般玩家「轉職→Lv4學會→施放」E2。

## 2026-08-23：`sub_17FC0` transient 狀態面板勘誤

- 本段訂正較早把「status icon／UI 全部未接」列為缺口的說法。授權 Docker 內
  IDA Pro 9.4 以非破壞性 offset probe 固定 `sub_17FC0`：raw
  `+0x22/+0x23/+0x24` 非零時分別把 AP、DP、HIT／EV 的 digit base 從
  `0x2A` 切到 `0x77`，並不另畫三個圖示；raw `+0x25/+0x26/+0x27` 非零時才
  分別畫 FDOTHER #5 entries `0x37/0x38/0x39` 到三個固定位置。主證據見
  [`fd2_status_panel_transient_indicators_ida.txt`](../data/ida/fd2_status_panel_transient_indicators_ida.txt)。
- 既有 `NativeItemPanelDataPlanFor→RenderNativeItemPanelData→prepareNativeChurchStatus`
  已精確保存上述 offsets、colors、entries、位置與十二幀角色狀態面板 owner；
  新增原始資產 production regression，驗證六個 raw 欄位都會改變各自的玩家可見
  區域。因此本項提升為 `RE-CLOSED`／`RUNTIME-E1`，不再列 remake 阻擋。
- 尚未證實的是三個圖示與前三個 color indicator 的高階玩家名稱、精確 tick／音訊
  與未修改一般玩家同狀態 E2；不得再造六個猜測圖示，也不得重解 `sub_17FC0`。

## 2026-08-23：commands 28／29／31 caller分歧與command28數值勘誤

- 授權Docker內IDA Pro 9.4重新匯出`sub_276EC`、`sub_29164`、`sub_2B659`、
  `sub_29C90`。command28使用first-target BG／panel、`sub_29164` mode0、actor
  phase target idle並略過`sub_29C90`；29／31使用actor base、mode1且逐final
  target執行背景轉場。這否定直接複製command24固定selector32 presenter的做法。
- `0x27C6D..0x27D4D`對command28固定分母8，函式尾端沒有補差。固定雜湊
  FIGANI.DAT的409筆全檔掃描又證實所有可達effect每個target tail都只有一個
  `raw+4==1` marker，因此舊`ExecuteNativeCommandDerivedStrike`對28扣完整roll
  是重製端錯誤，已修正為roll/8；29／31維持分母1並補獨立倍率回歸。
- 主證據見[`fd2_command28_29_31_presentation_ida.txt`](../data/ida/fd2_command28_29_31_presentation_ida.txt)。
  command29已有growth selector34→resource104資產鏈；28／31的一般玩家actor
  selector及三者正式玩家／敵方indexed owner仍未閉合，繼續失敗即關閉而不猜資源。

## 2026-08-23：command29 玩家多目標 indexed owner

- 本段取代上一段「command29 正式玩家 indexed owner 未閉合」的現況斷言；原始
  caller 分歧與 selector34→resource104 證據不變。系統設計先固定原子契約，正式
  玩家 confirm 才接入獨立 command29 owner。
- owner 預建九段 mode1 前導、一次 actor phase、FDOTHER #50 samples1／4，以及
  每個 final target 各自的 indexed BG／panel／idle 與 20 張 `0x29C90` 轉場。
  actor marker 只扣一次 MP；每個 target 都從 idle frame0 重設，並在自己的 marker
  發布 HP。全部 target 完成後才設定 `Acted` 並由 callback 發放死亡獎勵。
- 多目標回歸證實發布順序與單次 MP；第二個目標 marker 前刻意改變狀態時，正式
  owner 會把 actor MP／`Acted` 及所有 target HP／raw `+5` 整批回復。selector 非34
  或任一原始資產／raw provenance 缺失時保持零修改。
- 本切片為玩家 `RUNTIME-E1`，不是一般玩家原版 E2。敵方 caller、command28／31
  actor selector與演出仍未閉合，不得把此 owner 外推共用。

## 2026-08-23：command28／31 正常取得路徑裁決

- 本段勘誤前文把「尋找28／31一般玩家actor selector」列為持續待辦的現況。
  授權Docker內IDA Pro 9.4窄探針固定runtime mask OR writer `sub_1D79C`
  (`0x1D79C..0x1D80B`)只有`0x1E39A`一筆direct caller，位於level-up
  `sub_1E292`；caller依raw `+7`→growth `+10`→12-byte learn row掃六組pair。
- 固定command-learn表沒有ID28／31，32筆player constructor defaults也都沒有
  byte3 bit4／bit7。故「未修改一般玩家目前沒有已證實取得來源」提升為強推論，
  不再把兩者renderer列為remake交付阻擋；但因間接／未分類writer仍可能存在，
  不宣稱死碼。
- selector18雖是全FIGANI僅剩的impact-effect候選，沒有producer／consumer可綁到
  28或31，不得排除猜配。只有動態玩家路徑出現bit、找到新mask writer，或同actor
  raw `+7`與bit成對時才重開。證據見
  [`fd2_command28_31_reachability_ida.txt`](../data/ida/fd2_command28_31_reachability_ida.txt)。

## 2026-08-23：ID34 class19 玩家原子 state transaction

- 先依既有`0x27FC9`主證據寫規格，再接runtime；沒有重解helper。只接受raw
  `NativeRecordClass==19`、`BattleFig`為4／5／6／7／20、record34 MP gate與
  完整final-target raw provenance。這五條可達FIGANI resource已證實繞過唯一
  `0x1CA89` debit sink，所以MP需至少28但成功後保持不變；不外推其他visual group。
- 新交易以record34兩階段geometry建立私有`0x50` records，依
  `0x22721→0x22866→0x22997`套用17／18／19；三段全成才一次發布raw
  `+0x22..+0x24`、derived `+0x48..+0x4E`與actor `Acted`。缺任一stage零修改。
- 正式command whitelist現開放ID34；command grid→target confirm→三段交易測試與
  selector／target fail-closed測試通過。此為`RUNTIME-E1` state slice；
  `0x27FC9` indexed presentation、EXP、ID32／33／35、AI與E2仍未完成。

## 2026-08-23：command33 受限玩家 raw clear＋HP restore transaction

- `RE-CLOSED`：授權 IDA Pro 9.4 以固定雜湊 `FD2.EXE` 的一次性資料庫重新固定
  `0x27FC9` 唯一 caller `0x2A7CE`，以及ID33分支`0x285AC..0x285ED`：逐final
  target清`+0x25..+0x27`，再以固定`0x320`呼叫`0x211A4`。新主證據為
  [`fd2_command33_transaction_ida.txt`](../data/ida/fd2_command33_transaction_ida.txt)。
- `RUNTIME-E1`：正式玩家command grid→cursor confirm現只對raw class19與BattleFig
  4／5／6／7／20開放；record33的52 MP只作gate，不扣這五條已證實繞過debit
  sink的來源。交易在私有records完成三byte clear與完整target-list回復後才一次發布
  HP／raw transient／RNG／`Acted`，任一raw provenance或後段失敗均零修改。
- 聚焦Docker回歸通過3個battle原子交易案例，以及Xvfb正式ID33／34 confirm與
  whitelist。`0x27FC9` indexed presentation、score／EXP、ID32／35、AI、其他
  BattleFig與一般玩家E2仍失敗即關閉。

## 2026-08-23：README／有效佇列同步與後續 polish 入口

- README評估快照更新為2026-08-23，補入ID33／34正式class19玩家狀態交易；仍明示
  `0x27FC9` indexed畫面、score／EXP、ID32／35、敵方owner與E2未完成，沒有把綠色
  交易測試寫成畫面等價。文化保存段落、AFM署名、DOS中文、音源與圖片圖說未刪改。
- `91`有效佇列日期同步；下一個玩家可見工作回到戰場介面美化，只採已有原版
  座標／截圖證據且能建立決定性回歸的有界差異，不因「polish」猜補renderer。

## 2026-08-23：標題畫面有界 polish 與執行期重擷取

- 正式標題選單移除原版不存在的 `♪ F2 <source>` 常駐提示，但保留 F2 音源切換
  功能；未改動原版按鍵路由或 CONTINUE 語意。為避免截圖靠時間碰運氣，新增只在
  同時指定 `FD2_SHOT` 時生效的 `FD2_SHOT_TITLE_MENU=1` 有界畫面 oracle；缺少輸出
  或標題資產時失敗即關閉，不會影響一般玩家入口。
- Docker／Xvfb 重新產生 [`title-remake-runtime.png`](../figures/title-remake-runtime.png)，
  SHA-256 為 `13ff759213d9ccf1c72f687553d131554a8e360a0e5adb40a7cfcaaccc06266d`。
  新 oracle 的必要輸出／選單狀態測試及既有標題循環、CONTINUE 選項測試皆通過。
- 戰場 ch01 既有 exact-state 比較只剩22個邊界像素差，但目前證據不足以唯一歸因；
  本輪沒有為了消除數字而猜改 renderer。logozoom仍是近似，維持後續非阻擋美化。
