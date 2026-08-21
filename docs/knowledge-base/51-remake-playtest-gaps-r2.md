# 51 — remake 試玩落差 R2（歷史快照；戰棋機制+UI，2026-07-04 使用者實測)

> 使用者玩 `remake/play.sh` 回報 6 項與原版落差。本篇=記錄+對策分析,**尚未實作**。
> **注意**：部分條目後來已實作或被新 RE 推翻；不要把本篇的「尚未實作」直接當成今日狀態，
> 目前分層狀態看 [`58`](58-fd2-exe-re-coverage.md)，UI 狀態看
> [`57`](57-ui-evidence-matrix.md)，有效下一步只看 [`91`](91-worklist.md) 檔首；
> `42` 同樣是歷史落差快照。
> 工具邊界(2026-07-04 使用者釐清,規則 62):**靜態資料表(物品/武器/攻擊範圍/法術/成長)= 靜態 code + 青衫
> 修改紀錄即可還原,不需 debugger**。dosbox-x debugger(fd2-dosbox-x,doc48;normal core + BP exec trace 可攔截)
> **只在三類「非它不可」時才用**:①執行期才生成且無可離線解碼的靜態來源(如 acting 資源)②靜態讀出誤導/臆測屢錯
> (如走位機制,靠 runtime 差分才定案)③湧現/時序/耦合行為(如鏡頭追焦同步)。下列 6 項多屬靜態或純呈現,
> debugger 至多當 fallback。
> ⚠ 實作協調:①②③④⑤ 多數動 `cmd/fd2/main.go` / `internal/battle/*`,**須等 beat-runner 落地後**再改,避免衝突。

## 1. 結束回合［2026-08-22 已關閉原症狀］

- **歷史症狀**：當時只能靠 Tab，沒有可發現的正常入口；這個現況斷言已失效。
- **已證實原版行為**：共用玩家控制器 `0x117E7` 在 `0x12C0D` 回傳 `-1` 時
  呼叫空游標系統面板 `0x16F55`；direction3 是 END，接著顯示 FDTXT
  `0x1A3／0x1A4`、YES、等待 `0xC8` ms 再進 `0x1A30B`。
- **現行實作**：所有正式戰場都可在空格按確認開啟原始 FDOTHER #2 四格面板，
  向下選 END；四幀關閉、取消、兩段原文、十二個60 Hz近似幀、敵方回合與返回
  玩家回合均有回歸。缺原始圖格時不建立不可見熱區；其餘三格 owner 仍失敗即關閉。
- Tab 只保留為重製端快速鍵。自動判定全員行動完畢仍是可選改善，不再是基本玩法
  blocker，也不可用它取代已還原的原版 END 路徑。
- 主證據：[`fd2_117e7_empty_cursor_system_overlay_ida.txt`](../data/ida/fd2_117e7_empty_cursor_system_overlay_ida.txt)。

## 2. 亞雷斯（騎士）無法兩格攻擊 — 攻擊距離取決於武器

- **症狀**:所有單位只能打相鄰格。
- **根因**:`internal/battle/move.go:59` `InAttackRange` 寫死 `dx+dy == 1`(僅相鄰),無「武器射程」概念。
- **原版行為**:攻擊距離由**裝備武器**決定(使用者明示)。長槍/騎槍類=2 格,弓=遠程,劍=1 格等。
  doc32 已反組譯物品表 23B 結構；舊 `0x15356` 傷害鏈地址已由 canonical recheck 降級為未證實，武器射程欄位仍**未定位**。
- **對策**:
  1. 單位加 `AtkMin/AtkMax`(或 `Reach`)欄,來源=裝備武器的射程屬性。
  2. `InAttackRange` 改 `AtkMin <= dist <= AtkMax`;移動可達+攻擊可達的高亮要一起改(move.go reach 計算)。
  3. 武器→射程資料。
- **RE**:低,**純靜態解(規則 62,不需 debugger)**——射程是物品表(doc32 已解 23B 結構)的一個靜態欄位。三條靜態路任一即可:
  ①先查青衫武器表有無「攻擊範圍」欄(最快);②byte-diff:已知不同射程的兩把武器(騎槍 vs 劍)物品 bytes 對比,差異欄=射程;
  ③沿目前已證實的 `0x13A9F/0x14EF0/0x15B77` action/score boundaries 追射程檢查，看它讀物品結構哪個 byte。→ 更新 doc32。
  dosbox BP trace 僅在上述三路都對不上(靜態臆測屢錯)時才 fallback,不是首選。

## 3. 法術路徑 partial（歷史試玩快照已修正）

- **歷史症狀**:法術打不出來。現行 code 已接 `spellOpen`/`castSp`/指令環 case 1 法術/`drawSpellMenu`/`InCastRange`，但 native command/effect 仍未閉合。
- **根因(修正)**:這段舊試玩結論已過期。`spellOpen`/`castSp`/case 1/`drawSpellMenu`/`InCastRange` 與
  normalized `CastArea` 結算皆已存在；剩下的是 native command target/effect ABI 與原版演出，不可再寫成
  「選完目標不會結算」。
- **原版行為**:doc13 `Get_EasyMagic`/`0x1CFF0` 的 command selector 已部分反組譯；只有 raw
  command inventory、record `+5` MP gate、部分 target/effect routes 已閉合。`spell.json` 是可編輯
  normalized data，**不是**原版全量法術數值／範圍／效果表的完成證明。
- **對策**:保留 normalized `CastArea` 為可玩的 editable approximation；native grid 僅在有 E0 data 的部分使用。
  command 0 的 single-target numeric resolver 已獨立閉合，但 target geometry／完整 effect family 尚未閉合，
  因此不得把 legacy cast path 宣稱成原版 command runtime。
- **RE**:低-中。效果公式青衫已有;施法射程/AOE 形狀若不確定 → dosbox 觀察。

## 4. 狀態欄隨游標移位（舊近似已撤回）

- **勘誤**：舊版以「左下象限／畫面半屏」描述換角，並主張無需反向工程；
  這是缺少來源的近似，已被直接指令推翻，不可再作實作依據。
- **已證實原版行為**：`0x1AD2A..0x1AD5F` 只在 visible cursor Y
  `[0x53ABD]>5` 且 X `[0x53AB9]<3` 時寫 anchor `0xF2`，同一列且
  X `>9` 時寫 `1`；X 邊界 `3..9`、Y `<=5` 都保留既有 anchor。
- **現行實作**：`indexedmap.AdvanceNativeMapHUDAnchor` 與
  `fdsave.ContinueMapPresentation` 保存 persistent raw anchor；不得改回
  以半屏或被選單位位置推測。

## 5. 對話視窗:頭像溢出 + 右側頭像被文字覆蓋

- **症狀 a**:人物頭像沒完全放進對話框,凸出框外。
- **症狀 b**:頭像在右側(上框/對方 NPC)時,文字畫到頭像上。
- **根因**:`main.go` 對話繪製段——頭像 `ps=2.1`(80×80→168px)接近框高但定位讓它凸出框頂;
  文字換行寬度 `perLine` 用 `bx+620-16-tx` 未在「頭像在右」時扣掉右側頭像佔寬,故文字壓到頭像。
- **原版行為**:頭像**收在框內**(orig_02_dialog 量測:我方下框左頭像、對方上框右頭像,都在框邊界內);
  文字區與頭像不重疊。
- **對策**:
  a. 頭像縮放/定位改成「完全落在框矩形內」(降 ps 或改 anchor 讓頭底貼框底、頭頂不超框頂)。
  b. 上框(頭像在右)時,文字 `perLine` 與右緣改成 `頭像左緣 - 邊距`,把文字擠回左側,不與頭像重疊;
     或文字起點 tx 與寬度都避開頭像列。
- **RE**:低。對照既有 `extracted/remake_shots/orig_02_dialog*.png`(已有量測)即可,不需新 RE。

## 歷史實作優先序（不可當現行佇列）

1. **#1 結束回合**與 **#5 對話框**的原始症狀後續均已有實作；現況看 `57／58／91`。
2. **#2 武器射程**(核心戰棋規則,需 dosbox 定位武器射程欄)+ **#3 法術結算**(補完既有路徑)
3. **#4 狀態欄移位**(UX 打磨)

> 這是2026-07-04的歷史排序；beat runner 與多項後續切片均已落地，不得再以本句阻擋現行工作。
