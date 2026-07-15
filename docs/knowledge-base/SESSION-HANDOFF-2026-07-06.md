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
> canonical battle state，compiler 為 0 issues。campaign 已接成 battle→post→choice，完整測試、
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

## 0. 目前焦點(接手就做這裡)
`ch00_pre`、`ch00_post`、`ch01_pre`、`ch01_post` 已成為前四個 campaign 實際 consumer；ch01 post 的 branch、
reward、61-utterance FDTXT_002、dynamic speaker slots、PAN、SPAWN4、ACT14..16、JOIN/sync/chapter tail
與第二、第三章戰前 handler 均已完整 lower 且 compiler **0 issues**。下一個具體焦點是釘死
ch03 battle turn3 的 slot6 active 條件已接上；尚待補完該 event 的 PAN/delay/dialog #4 演出，接著以
single-slot diamond + `layout_units` 閉合 `ch02_post`。下方「草地深層未解」是 2026-07-06 歷史記錄，已被 2026-07-15 direct table 修正推翻，
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
