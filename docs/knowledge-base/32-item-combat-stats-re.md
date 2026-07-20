# 32 — 物品 / 戰鬥數值系統反組譯(進行中)

> 目標:反組譯「裝備如何加成 AP/DP」「物品使用效果」「轉職」,供 M1 戰鬥結算用。
> 本篇記錄**已確認**與**待續**(誠實標註,rulebook 62/63)。本輪深度有限,物品/轉職機制需後續多輪。

## 1. 物品表結構 [驗](EXE `0x540AC`,23B/item,215 筆)

`dump_exe_tables.py` → `docs/data/exe_tables/item.json`:
```
-- TY AP AP HT HT DP DP EV EV S1 S2 R1 R2 K1..K6 MM MM ...
   type  ap(u16) hit(u16) dp(u16) ev(u16) atk_attr atk_rate range[2] K[6] price(u16)
```
例:item#64 `type7 ap80 hit95 price1200`(武器,攻擊力+80)。→ **物品帶 ap/dp/hit/ev 加值**,裝備時加到單位。

## 2. 傷害計算鏈 [驗]

```
攻擊執行(大函式 0x15xxx,含演出+結算)
   ├ 算攻方 AP → 全域暫存 [0x53c27](0x15aff 寫)
   ├ 算守方 DP → 全域暫存 [0x53c2b](0x15b08 寫)
   └ 傷害公式 0x15356(doc 11):
        dmg = AP×地形AP%[地形]/100 − DP×地形DP%[地形]/100   (地形表 0x1A12 / 0x1A2A)
        dmg≤2 → 跳過(AI);擊殺加成見 doc 11
```
即 **傷害讀的是 `[0x53c27]`(攻方AP)/`[0x53c2b]`(守方DP)全域暫存**,在攻擊執行時先算好填入,非直接讀 unit 欄位。

## 3. 已知操作(攻略/doc 13)

- 物品選單(移動後「右」):**使用 / 給予 / 裝備 / 丟棄**(notes.md)。
- 裝備自帶法術不耗 MP、但施放無經驗(item.md)。賣價 = 原價 75 折。
- 單位 roster 26B 含 **物品×8 + 法術×8**(parse_field;即每單位 8 裝備欄 + 8 法術欄)。

## 3.5 主角隊起始武器 + AP/HIT/EV 合成公式 [驗](worklist 第8輪後,對 orig_07 截圖逐位吻合)

**人物出場屬性表**(modify2 §4,EXE `0x55BA1`,anchor `01 01 01 2A`,24B/人物,順序同 growth 表 §5):
`RA(1) CL(1) LV(1) HP(u16) MP(u16) MV(1) MG(4) IT(6) AP(u16) DP(u16) DX(u16)`。IT[0]=起始武器 id、
IT[1]=起始防具 id(FDFIELD 出場人物資訊同款慣例:前兩個固定武器+防具)。此表 `dump_exe_tables.py` 尚未收錄
(anchor 早已定義,缺 `dump_char()`),本輪用臨時腳本直讀驗證,未來若需其他角色可補上該函式。

索爾(idx0)/亞雷斯(idx4)/悠妮(idx9)/蓋亞(idx30,索引與 growth 表 §5 一致)起始武器/防具:

| 角色 | 職業 | 武器 id | 武器 | 防具 id | 防具 |
|---|---|---|---|---|---|
| 索爾 | 劍士 | 0x00 | 短劍(AP10 HIT95) | 0x84 | 皮甲(DP8) |
| 亞雷斯 | 騎士 | 0x14 | 刺矛(AP20 HIT90 射程1-2) | 0x80 | 布衣(DP2) |
| 悠妮 | 法師 | 0x34 | 長棍(AP8 HIT85) | 0xA4 | 長袍(DP5) |
| 蓋亞 | 機兵 | 0x48 | 威力手臂(AP15 HIT90) | 0xB2 | 戰鬥裝甲(DP8) |

**合成公式(對 `extracted/remake_shots/orig_07_unit_status.png` 索爾逐位吻合,LV·01 DX·002 HIT·097 AP·016 EV·002 DP·012)**:

```
角色底值(空身無裝備,list.md §7.3 交叉驗證) = char表base + LV×growth_min(AP/DP同理;HP/MP用(LV-1))
有效 AP  = 角色底 AP + 武器.ap                  (索爾 6+10=16 ✓)
有效 DP  = 角色底 DP + 防具.dp                  (索爾 4+8=12 ✓)
有效 HIT = 角色底 DX + 武器.hit                 (索爾 2+95=97 ✓ ←關鍵發現:item表HIT/EV是「增值」,非絕對值)
有效 EV  = 角色底 DX + 防具.ev                  (索爾 2+0=2  ✓;起始4件防具ev皆為0)
```

四人算出:索爾 AP16/DP12/HIT97/EV2/crit5%;亞雷斯 AP26/DP6/HIT92/EV2/crit3%;
悠妮 AP11/DP7/HIT86/EV1/crit3%;蓋亞 AP22/DP14/HIT92/EV2/crit0%(resist_crit.json 依職業)。
已串進 `internal/battle/event.go` PartyMember(新增 HIT/EV/CritPct 欄位 + spawn_party 賦值)與
`assets/scenarios/ch01.json`,修好主角隊 HIT=0 導致 100% miss / 0 傷害的問題。

> 此發現直接解答下方原「[阻] 裝備加成精確公式」——至少對「基礎四圍(AP/DP/HIT/EV)如何疊加裝備」已鎖死
> (DX 是 HIT/EV 的底值來源,item 表 HIT/EV 欄是疊加增量);轉職後 DX 底值/其他角色武器仍待逐一 RE。

## 4. 待續(需後續輪次)[阻]

### 4.1 2026-07-20 direct range-field trace

以 `tools/disasm_le.py` 追 `0x318ad` 與 item pointer helper `0x4e56c` 後，欄位偏移更正如下：

```
+0x0 type, +0x01 AP, +0x03 HIT, +0x05 DP, +0x07 EV
+0x0a atk_attr, +0x0b atk_rate, +0x0c range_min, +0x0d range_max, +0x0e..0x13 K[6]
```

`0x14237` 是戰鬥指令／攻擊目標路徑；它讀 `item+0x0c`，再把該值傳入 `0x14818`。後者在 `<16` 分支以 Manhattan 距離與此參數比較並標記可選格，因此 `range_min` 的確是通用目標幾何 cutoff。相同 caller 讀到的 `item+0x0b` 在該攻擊路徑後未再使用。另一條 `0x18d8c` 是物品使用／效果路徑，雖也把 `+0x0b/+0x0c` 傳給 `0x14818`，不能反推成武器射程。`+0x0d` 在 `0x15723` 的特殊效果分支有幾何用途，也不能直接當通用武器距離。

因此 remake 暫時只沿用已由 `weapon_range.json` 驗證的武器射程；不得把 `atk_attr/atk_rate/range_max` 臆測成 `AtkMax`。這輪只補 provenance 與欄位註解，不改變未證實的戰鬥公式。

- **[阻] 表 base-relative 存取**:item/unit/growth 表(0x540ac…)在 code 中以「obj2 基底(reg)+ offset」讀,
  絕對位址不經 fixup → 不能用 `refs` 直接找讀取點,要追基底暫存載入處。
- **[阻] 物品使用效果碼**:藥水回血量、卷軸效果、轉職道具的程式邏輯未反組譯。
- **[阻] 轉職系統**:攻略層有(Lv20+教會、轉職道具表 58h–60h→英雄/聖者/召喚師…,doc 02 §5.10);反組譯機制(職業數值替換、能力繼承、成長表切換)未做。
- **[阻] 轉職與 sprite**:角色 id = 肖像 = sprite組 恆等(doc 31,memory.md 權威);轉職後換成轉職態肖像編號(memory.md 0x20–0x41),sprite組是否隨之切到另一組**待反組譯轉職碼確認**。⚠ 舊版「凱拉斯組17→49、轉職當機」已作廢(DATO_067 誤判,凱拉斯實為 id16,三者恆等)。

## 5. 對 remake 的暫行做法

數值都在 `item.json` / `unit.json` / `growth.json` + 攻略公式(doc 02 §4),所以 **M1 可先用「base(unit表)+ 裝備(item表 ap/dp)」實作,攻略公式當規則**,反組譯機制(精確累加/使用效果/轉職)當後續校正,不阻塞戰鬥切片。

> 相關:doc 02(數值/公式)· 03(EXE 表)· 11(AI/傷害)· 13(物品選單)· 27(戰鬥規則)。資料:`docs/data/exe_tables/`。
