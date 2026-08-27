# 09 — 劇情 / 對話結構與抽取

> 怎麼把《炎龍騎士團2》的劇本從 `FDTXT.DAT` 還原成可讀、可翻譯的文字。
> ⚠ **完整劇本是遊戲著作權內容,只保留在本機 `extracted/story/`,不入版控**;
> 本文件只記錄「格式 / 方法 / 極短樣例」。

## 文本在哪、怎麼編碼

- `FDTXT.DAT`(LLLLLL 容器)共 35 個資源 ≈ **章節**;合計約 1016 條字串、58000 字。
- 文字**不是 Big5**,而是**自製字型的 uint16 字模索引**(見 `08-text-and-font-format.md`):
  glyph 0–0x71F 為 1824 個字模(0–9/A–Z 為英數,其餘為漢字),字串以 `0xFFFF` 結束。

## 對話段落結構(第 3 輪解出)

每個資源內的字串可含**多段對白**,每段格式:

```
[對話控制碼 0xFFxx] [身分 operand / 場景索引] [0x22D『] [對白 glyph…] [』]
```

- **對話控制碼**(`0xFFEE` / `0xFFED` / `0xFFEF` …):段落 / 換行 / 對話框類型。`0xFFFE` 是已驗證的軟換行，不再標成「疑為」。
- **第二個 operand 不是一律直接 DATO ID**：`0x15F84` 的 `-17/-18` 先經 `0x12C60` 嘗試以場景表／persistent record 的 `+8` 找身分，再由 record `+7` 取得 DATO；未命中的 direct-DATO fallback 與 `-19/-20` 的 runtime-unit index 路徑必須分開保存。不能把數字直接當作跨章固定角色／肖像表索引。
- `0x22D` ≈ 開引號 `『`。

### 控制碼語意(已反組譯文本渲染器 `0x15F84`，外層嘴型／輸入在 `0x16D00+`)

控制碼以 sign-extend int8 比較分派(`cmp eax, imm8`)。各碼語意:

| 碼 | int8 | 語意 |
|---|---|---|
| `0xFFEF` | -17 | **開上框 + 載入 DATO**；operand 先走 `0x12C60` 身分查找，未命中時才可作 direct-DATO fallback |
| `0xFFEE` | -18 | **開下框 + 載入 DATO**；operand／record provenance 由 `0x12C60` 路徑決定，不命名成固定肖像 ID |
| `0xFFED` | -19 | **開上框 + runtime unit lookup**；operand 是 unit index，最後讀該 record `+7` 作 DATO selector |
| `0xFFEC` | -20 | **開下框 + runtime unit lookup**；operand 是 unit index，最後讀該 record `+7` 作 DATO selector |
| `0xFFFE` | -2 | **句內換行**;每框最多 3 行,滿則 `0x17C24` 捲動 / 等待 |
| `0xFFFD` | -3 | **換行 + 等待按鍵翻頁**(`0x17A57(1)` 清框續下一頁) |
| `0xFFFC` | -4 | 遞迴插入 `[0x53AD9]` 字串 |
| `0xFFFA`/`0xFFFB` | -6/-5 | 遞迴插入 `[0x53AE1]`／`[0x53ADD]` 名稱或數值字串，不是特效 |
| `0xFFFF` | -1 | **字串結束** |

對話框位置全域 `[0x3C67]`:`0x728`=上框、`0x9017`=下框。文字起點 `0xA0B4F`(上)/`0xA951F`(下)。
**重要副產物**:四種開框碼最終都會在其各自 provenance 路徑讀取 `DATO.DAT`；`DATO.DAT` 是人物頭像／立繪資源，但 operand 數字本身不一定就是 DATO archive index。

> 對重製:框開碼=換說話者(本工具已據此分行並標說話者名);`0xFFFE`=軟換行;`0xFFFD`=「按鍵翻頁」點。
> 翻譯改寫文字長度時,要保留框開碼與其 raw identity operand／查找 provenance,並注意每框 3 行的排版上限。

## 抽取方法

1. `tools/decode_text.py` 解析字串為 glyph 索引序列。
2. `tools/render_story.py` 用自製字型把整章渲染成**可讀 PNG**(分頁,控制碼當換行)。
3. 對照 PNG 人工 / 多模態轉錄為文字,說話者依肖像 ID 標註 → 存 `extracted/story/`(本機)。

```bash
# 把全部 35 章渲染成可讀 PNG(需自備原版,先 unpack_dat + extract_all)
python3 tools/render_story.py --all extracted/raw/FDTXT \
        extracted/raw/FDOTHER/FDOTHER_004.bin extracted/story
```

## 極短樣例(序章開場,示意格式)

> 完整劇本見本機 `extracted/story/`,此處僅示意說話者標註格式:

```
索爾  :累死了,大家休息一下吧!
亞雷斯:聽說再越過這片海洋就到馬拉大陸了…
悠妮  :嗯,還好。海風吹起來真舒服啊…
```

## 對中文化 / 重製的意義

- 已能把劇本還原成可讀文字 + 說話者 → 可做劇情校訂、翻譯、語音對照。
- **glyph索引→Unicode對照表**已完成；後續工作是依caller補齊罕見控制碼語意與
  人工校訂，不能把自動解碼直接視為逐章呈現E2。

## 進度

- ✅ 文本 / 字型格式破解;✅ 對話段落結構(說話者 + 控制碼)解出;✅ 全 35 章渲染成可讀 PNG。
- ✅ **glyph→Unicode 對照表完成(1824/1824,100% 覆蓋)** → `docs/data/glyph_map.json`。
- ✅ **全35章自動轉錄產物完成**：`tools/decode_story_text.py --all`可解碼成含說話者
  候選的UTF-8，輸出本機`extracted/story/full_story_auto.md`（1450句對白）；這不表示
  35章caller mapping、人工校訂、控制碼或一般玩家E2均完成。序章＋第2＋第3章另有人工精校版。
- ⬜ 控制碼語意精確化(換行 / 名稱插入 / 顏色);少數罕用字模可能仍有個別誤判,可隨閱讀校正。

## 劇情大綱(自動解碼後可讀)

開場(序章):索爾、亞雷斯、悠妮渡海抵馬拉大陸遇海盜 → 哈瓦特父子(哈諾)相助入隊。
第2章羅德鎮強盜團、希莉亞登場、揭露悠妮失憶疑為公主。第3章救鐵諾入隊、反派葛雷與卡蘿娜伏線。
中段轉職、世界各地;後段**天空之城機器人篇**(機甲守衛以型號代號 73/C2/A1 等對話,悠妮一度被機械控制)。
結局眾人保住世界、返鄉,希莉亞請父王(王國)辦慶功宴(印證其公主身份)。
完整逐章見本機 `extracted/story/full_story_auto.md`。

## remake 對話框渲染規則(cmd/fd2/main.go;使用者實玩逐項校正 2026-07-05)

原版對照:我方(角色 id 0–31)= **下框、頭像在左**;對方/NPC(id≥32,如父王48/王后66)= **上框、頭像在右**;
頭像側臉朝文字。以下為對原版截圖逐項修正的規則(commit 57c0e30 / dc5ebb1 / b81268d):

- **文字寬 vs 頭像**:文字換行右緣須**止於頭像側之前**,不得覆蓋頭像。下框到框右緣;上框(頭像在右)
  到頭像左緣前 8px(`rightEdge = hx-8`)。原版父王對話文字佔左~60%、碰頭像前換行(orig 18-02-20)。
- **框位置(框高 198,logicalH=400)**:下框 `by=198`(底邊 396 在畫面內);上框 `by=4`(頂邊在畫面內)。
  舊值(下 224/上 -22)使邊線出畫面看不到。頭像垂直置中框內 `hy=by+(198-168)/2`,不凸出框上下邊。
- **框內底色 = 頭像底色漸層**:頭像背景是漸層藍(頂 40,69,138 → 底 56,85,154);框素材 `assets/ui/dialog.png`
  內部均勻 56,85,154(僅等於頭像底部亮色)。故框內疊同一漸層(`g.dlgGrad`),消除頭像↔框接縫色差。
- **[HARD] 長對白分頁,不截斷**:一句 >3 顯示行要**分頁**(每頁 3 行),Enter 有下一頁先翻頁、翻完才換句。
  王座對白多句超 3 行(line6/12/14/15/17…,如 line17 57字=5行=2頁)。抽 `dlgWrap`(繪製與分頁共用換行)
  + `dlgPageCount` + `dlgAdvance`(翻頁/pop),`g.dlgPage` 記頁碼、新對白歸零。**舊碼繪製迴圈 `i<3` 只畫 3 行
  其餘丟棄 → 後半永遠看不到**(使用者實玩抓到)。回歸測試 `cmd/fd2/dlg_test.go`(需 xvfb 跑,ebiten init 要顯示器)。
- **面向**:cutscene 角色 dir 預設 0(下,面向玩家);走位者走完面向 actor 目標 dir(進場走位=`a.Dir`,
  如亞雷斯走到索爾旁面向他);詳見 doc47 §11 面向通用規則。
- **驗證法**:`FD2_CAMPAIGN=… FD2_CAMP_NODE=<node> FD2_SHOT=out.png FD2_SHOT_FRAME=N`(headless,host xvfb+llvmpipe);
  長對白翻頁用 xdotool 送 Enter。⚠ 軟體渲染 fps≠60,`FD2_SHOT_FRAME`(遊戲幀)與 wall-clock 送鍵易對不上,
  優先 Go 測試(確定性)驗邏輯、截圖驗版面(見記憶 [[fd2-goal-and-no-speculation-rule]] 採樣率不準)。
