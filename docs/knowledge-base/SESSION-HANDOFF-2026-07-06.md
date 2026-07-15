# 交接文件 — 給下一個 session(2026-07-06)

> 炎龍騎士團2 RE + Go/Ebiten remake(`/home/anr2/cht/fd2`,repo `wicanr2/fd2_re` main)。
> 記憶檔(`~/.claude/projects/.../memory/`)會自動載入=長期真相;本檔補「這段 session 的當前狀態 + 開放線索」。
> **動手前先讀:記憶索引 MEMORY.md、`docs/knowledge-base/00-index.md`(問題導向路由)、`doc50`(過場機制主檔)。**

> **2026-07-15 Codex 接手更新**：map0 ACT0/1/2/5 已全解。getter 的 relocated disp32 會隨 context
> 從序章 `0x207718` 改為 map0 `0x2077d8`（差 48 entries）；raw directory/data 位於 FD2.EXE
> file+0x565d8 / +0x53e00，不是 LOADCH0 新讀的 DAT。remake 已接 persistent party（JOIN order
> 0,9,4,30）→ FDFIELD group1/2 append → editable acting。詳見 doc50 §1.2/§2.1；本 handoff 以下
> 「草地未解」是 2026-07-06 歷史狀態，不得覆蓋較新的 doc50 定論。注意 ACT 20/20 解碼不代表
> `ch00_pre` 全 handler 已 lower；非 ACT 的 unresolved 原語仍保持 partial/fail-closed。

## 0. 目前焦點(接手就做這裡)
第一章**開場過場**逐幕對齊原版:王座傳位(done)→ 草地/王城一隅亞雷斯撞見(**幾乎 done,差一點**)→ 森林。
目前卡在一個**深層 RE 未解**:**草地主角(索爾/亞雷斯)的走位是用什麼機制驅動**(見 §3)。

## 1. 這段 session 做完的事
- **王座傳位幕**:走位 (8,42)→**(8,21)**第一次對話→**(8,8)**最終(對原版截圖+FDFIELD 守衛地標實測);
  守衛 dir=0(面向玩家);對話切分 line0 / line1-18;對話框修 4 項(文字不蓋頭像/上下框移入畫面/漸層/**長對白分頁**)。
- **草地幕(palace_path)**:亞雷斯 2 段進場(13,47→11,47→8,46 面向索爾)、進場句用**上框**、對話後索爾走到旁邊。
  ⚠ **「兩人一起走離+淡出」(結尾)先前試做又還原了**(見 §3 待辦)。
- **debug 工具**(cmd/fd2/main.go):`FD2_UNIT_LABELS=1`(sprite 標 `[idx]f<fig>(x,y)dDir`)、
  `FD2_CUTSCENE_LOG=1`(過場 node/beat/走位逐步印 stderr)。
- **文件集中化**:`doc50`=過場機制唯一主檔;新增 `scene-decode/ch1-throne.md`+`ch1-meadow.md`(每幕原始資料×解讀)。

## 2. 已驗證的 RE 定論(耐用真相,別再翻案)
- **走位機制 = step 家族 + 路徑走位**(非 acting):`0x12eaa`下/`0x1300d`左/`0x13185`上/`0x13315`右(各推一格+捲鏡頭);
  通用 `0x13488(單位idx, 方向陣列, 步數)` 走任意路徑。王座是「全上」特例(直接 0x13185×15/13)。單位結構 +0X/+1Y/+3pose/+4tick/+8角色ID。
- **此 handoff 的 acting「只設面向」結論已於 2026-07-15 推翻**：normal frame 依 pose 每拍移一格，
  special frame 才原地顯示。格式與證據以 `doc50 §1.2` 為唯一準據。
  bit7 不改變 (unit,pose) 意義。normal frame 的低7位拍數=移動格數；special frame 的拍數才是
  原地顯示節奏。+4 tick 配繪製公式 `0x127e0=格+tick×f(pose)` 做每一格內的平滑內插。
- **map32 roster(dosbox dump `task_f/slots0_20_dialogue.bin`)**:slot0王/1后/**2=王座索爾**/**3=草地索爾(4,46)**/**4=草地亞雷斯(13,47)**/5-20守衛。
- **面向規則(全劇本)**:dir 預設 0(下/面向玩家);FDFIELD 不存面向;非0僅「走位者面向移動方向」或「劇情主角對看」。

## 3. ★最大開放問題:草地主角走位機制未定位★
- handler 草地段(0x3231b Part1 後段)只呼叫 PAN/BGM/**ACT(0x64~0x69)**/對白;演出 0x65~0x69 **只設面向**
  (0x65=units0-15面上、0x66/67/69=守衛16/17、0x68=后1),**沒有一筆動主角 slot3/4、也沒有 step/0x13488 呼叫**。
- 但原版影片(doc55)明明看到索爾/亞雷斯在草地走多格。⟹ **主角走位是「另一個機制」,還沒找到。**
- **候選下一步(scene-decode/ch1-meadow §5.4)**:
  (a) 全 EXE 搜「誰寫 slot3/4 的 +0/+1(X/Y)」= `mov [base+idx*0x50+0/1]`(idx=3/4 或動態);
  (b) 解森林 context acting(`out/acting_resources_forest.bin`+`acting_table_forest.bin`,未解；但草地問題優先直接抓 entry/caller);
  (c) 接受 doc55 影片量測重建(remake 已對齊、可玩),精確原版驅動長期擱置。
- **方法論(使用者定)**:證據(截圖/影片)+ 已知機制 → 可「由上而下」回原版資料找出處,不必每次 RE 到底。

## 4. 其他待辦(worklist doc91;不急)
- **草地結尾「兩人一起走 + 淡出」**(18-08-17):remake 現直接淡出,少一段。exit_walks(多人並行)可做,但先前試做已還原。
- **索爾走不完整**:exit_walk(到 7,47)疑被淡出提前打斷(時機衝突),可修。
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
  `extracted/dosbox_dump/acting_decoded/decode_acting.py`(acting 解碼)。
- **remake**:`remake/`(build:`cd remake && ./build.sh` docker;跑:`./play.sh`;headless 截圖:見 play.sh --shot 或 FD2_SHOT env)。

## 7. 環境速記
- 反組譯:`docker run --rm -v /home/anr2/cht/fd2:/w -w /w fd2-cap sh -c "python3 tools/disasm_le.py 'org_game/炎龍騎士團/FLAME2/FD2.EXE' range 0xA 0xB"`
- headless 截圖:`xvfb-run -a -s "-screen 0 1280x800x24" env LIBGL_ALWAYS_SOFTWARE=1 GALLIUM_DRIVER=llvmpipe FD2_MUTE=1 FD2_CAMPAIGN=... FD2_CAMP_NODE=<node> FD2_SHOT=out.png FD2_SHOT_FRAME=N ./fd2-linux`
- EXE 位址:tool-linear = 檔內位址(disasm 直接吃);執行期位址另有 loader 偏移(見 doc48 §5)。
- **每輪做完 commit + push**(CLAUDE.md 要求)。素材/dump/org_game/references 一律 gitignore 不入庫。
