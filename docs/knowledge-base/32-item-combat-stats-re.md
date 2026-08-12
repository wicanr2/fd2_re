# 32 — 物品 / 戰鬥數值系統反組譯(進行中)

> 目標:反組譯「裝備如何加成 AP/DP」「物品使用效果」「轉職」,供 M1 戰鬥結算用。
> 本篇記錄**已確認**與**待續**(誠實標註,rulebook 62/63)。本輪深度有限,物品/轉職機制需後續多輪。

## 1. 物品表的兩種錯位視圖 [驗]（2026-07-27 勘誤）

`dump_exe_tables.py` 的攻略／normalized 視圖從 EXE file `0x540AC` 起，以
23B/item 匯出目前攻略列出的 215 個 ID 到 `docs/data/exe_tables/item.json`：
```
-- TY AP AP HT HT DP DP EV EV S1 S2 R1 R2 K1..K6 MM MM ...
   type  ap(u16) hit(u16) dp(u16) ev(u16) atk_attr atk_rate range[2] K[6] price(u16)
```
例:item#64 `type7 ap80 hit95 price1200`(武器,攻擊力+80)。→ **物品帶 ap/dp/hit/ev 加值**,裝備時加到單位。

這不是 `0x4e56c` 回傳指標的同一個 row 起點。Docker data dump 與 EXE
逐 byte 比對確認：原生 helper 的 linear `0x602AD` 對應 file `0x540AD`，
也就是比上述 normalized view 向後一 byte；stride 同為 `0x17`。因此 runtime
row 0 是 normalized row 0 的 bytes 1..22 再接 normalized row 1 的 byte 0，
不能直接把兩份資料用相同 offset 命名。

匯出器現在另產生 `native_item_effect_rows.json`，保存 215 個已知 ID 的 raw
runtime prefix，並在 docs 與 remake assets 各追蹤一份。這只閉合「已知 prefix
的逐 byte producer」，不證明 native table 正好在 ID 214 結束。

## 2. 傷害計算鏈 [驗]

```
攻擊執行(大函式 0x15xxx,含演出+結算)
   ├ 算攻方 AP → 全域暫存 [0x53c27](0x15aff 寫)
   ├ 算守方 DP → 全域暫存 [0x53c2b](0x15b08 寫)
   └ 舊筆記標為 0x15356 的傷害公式（地址未由 canonical scan 證實）：
        normalized dmg = AP×地形AP%[地形]/100 − DP×地形DP%[地形]/100
        （地形表 0x1A12 / 0x1A2A；`dmg≤2`/擊殺加成不得當作 native AI 證據）
```
即 **normalized/remake 的傷害估值**可使用 `[0x53c27]`/`[0x53c2b]` 作 raw lead；canonical scan 尚未證明 `0x15356` 是 native callee，也不能以此行宣稱完整 attack/AI parity。

## 3. 已知操作(攻略/doc 13)

- 物品選單(移動後「右」):**使用 / 給予 / 裝備 / 丟棄**(notes.md)。
- 裝備自帶法術不耗 MP、但施放無經驗(item.md)。賣價 = 原價 75 折。
- 單位 roster 26B 含 **物品×8 + 法術×8**(parse_field;即每單位 8 裝備欄 + 8 法術欄)。

## 3.5 主角隊起始武器 + AP/HIT/EV 合成公式 [驗]（worklist 第8輪後,對 orig_07 截圖逐位吻合）

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

### 4.0 2026-07-27 item row caller audit（新增證據）

官方 IDA 9.4 重新檢查 `0x4e56c` 的多個呼叫者，確認 raw row 的用途比單一攻擊 caller 更廣：

- `0x1145a` 與 `0x1b750` 都只在 inventory flag `&0x40` 時讀 row
  `+1/+3/+5/+7`；215 個已知 raw rows 已逐筆與 normalized `item.json`
  交叉，四個 little-endian words 全數等於 AP/HIT/DP/EV。這是裝備合成
  資料流，不是 UI 顯示專用欄位。
- `0x14237` 讀 row `+0x0b/+0x0c` 後呼叫 `0x14818`；目前只把它記為 caller-specific geometry inputs，不能把 `+0x0b` 命名為通用射程上限。
- `0x1567e` 會讀 row `+0x0d/+0x10/+0x11/+0x12`，依分支呼叫 `0x14818` 或 `0x149f8`；這證實特殊物品效果共用幾何 routines，但仍不足以命名效果或欄位。
- `0x1bbdc`／`0x20c6f` 以 row `+0x0d` 做 type dispatch，並由不同原生 callee 消費；數值方向、顯示語意與 target ABI 仍未閉合。
- `0x1e0db(value, digitBias, target)` 只在 target 位於 camera bounds 時，把 `value` 轉成四位十進位字元，寫入四組 raw presentation queue（位置碼 `2,7,12,17`、target index 與 digit bytes），最後遞增 queue count。它不是 HP/MP/damage/heal 的命名 renderer；`0x1e1dc` 是相鄰的四 byte raw queue writer。
- `0x1debe(actor, x, y)` 只證實 active gate、曼哈頓相鄰一格與 equipped item row `+0x0b <= 1`；它不能推出 item `+0x0b` 是所有武器的通用最大射程。

因此目前安全結論是：`item.json` 的 normalized AP/HIT/DP/EV 與已驗證的
`weapon_range.json` 可供 remake 使用；raw table base `0x602ad`、stride
`0x17` 與 215 個 ID 的 byte-exact prefix 已另存
`native_item_effect_rows.json`。runtime table 的最終邊界及其餘欄位仍
fail-closed，不能把 215 筆 prefix 宣稱為完整 table。

### 4.1 2026-07-20 direct range-field trace（2026-07-27 勘誤）

以 `tools/disasm_le.py` 追 `0x318ad` 與 item pointer helper `0x4e56c` 後，欄位偏移更正如下：

```
+0x0 type, +0x01 AP, +0x03 HIT, +0x05 DP, +0x07 EV
+0x0a..+0x0d caller-specific raw inputs, +0x0e..0x13 K[6]
```

原先把這四個 byte 命名成 `atk_attr/atk_rate/range_min/range_max` 並把 `0x14237` 的 `+0x0c` 稱為通用 `range_min`，現撤回。已確認的安全描述是：`0x14237` 將 item row `+0x0b/+0x0c` 以 caller-specific 順序傳入 `0x14818` 的 `a5/a4`；`mode<0x10` 時 `a5` 會排除 marker cells，`mode>=0x10` 走 cross branch。另一條 `0x18d8c` 也讀相鄰 raw bytes，`+0x0d` 另有特殊 effect dispatch caller；這些都不足以反推出通用武器射程或 normalized `AtkMin/AtkMax`。

因此 remake 暫時只沿用已由 `weapon_range.json` 獨立驗證的 normalized 武器射程；不得把 raw `+0x0b..+0x0d` 臆測成 `AtkMin/AtkMax`。這輪只修正 provenance 斷言，不改變未證實的戰鬥公式。

- **[阻] 表 base-relative 存取**:item/unit/growth 表(0x540ac…)在 code 中以「obj2 基底(reg)+ offset」讀,
  絕對位址不經 fixup → 不能用 `refs` 直接找讀取點,要追基底暫存載入處。
- **[~] 物品使用效果碼**：`0x1bbdc` 的 selector／transfer／equip branches
  與 observed type5–24 mutation routes已閉合。重要 UI correction：
  `0x1b9de/0x184c0` 不是把八個 raw holes原位顯示，而是依 signed flag
  非負 compact成兩欄四列；native writers 維持 occupied prefix。
  ↑/↓ linear wrap、←/→±4；battle-use Enter拒絕 effect type0。
  `NativeItemSelectorCells`／`AdvanceNativeItemSelector` 已保存 input、
  `(42/192,103+22r)` geometry、category/equipped/stat icon raw IDs。
  現行 remake shell仍保留八個 raw位置，是 provenance/debug UI而非原版
  parity；Enter transaction、indexed animation仍待接。
- Docker Capstone 也已閉合共用 item pointer `0x4e56c(item)`：table base
  `0x602ad`、row stride `0x17`（23 bytes）。EXE file view 已確認從
  `0x540ad` 起，與 normalized `item.json` 的 `0x540ac` 起點相差一 byte；
  215-row raw prefix 已獨立匯出並由 loader regression 固定。table 最終
  長度與其餘未命名欄位仍未證實；只有已全表交叉的 `+1/+3/+5/+7`
  可命名為 AP/HIT/DP/EV。
- `0x20c6f` 的 Docker trace 已確認完整 type dispatch；目前 typed gameplay
  closures 已覆蓋 5/13、8–12、14–16、22。其餘 callee 或 presentation
  語意仍依個別證據 fail-closed，不能再用「全部效果都未完成」的舊斷言
  掩蓋已閉合 routes。
- `0x21082` 已確認是 modifier-word + unit-field-offset、effect display、
  `0x1b750` synthesis、source removal 的共同路徑；`0x22af6` 已確認掃
  target list 並累加全域結果，但後者的 status/gameplay 名稱仍不可猜測。
- `0x20c6f`→`0x21082` 的 type 8/9/0xa 已閉合為永久 base-stat item：
  item row `+0xe` unsigned word 分別加到 persistent `+0x37/+0x39/+0x3e`
  的 base AP/DP/DX，經 `0x1b750` 重算後移除來源 slot。三筆已知 raw item
  IDs 198/199/200 的 amount 分別是 AP+9／DP+9／DX+7。presentation
  selectors `0x11/0x12/0x13` 的顯示名稱仍保持 opaque；共用 callee 的
  type17–19 由下列獨立欄位證據閉合，不套用 base AP/DP/DX 名稱。
- `0x211a4(actor,count,targetBytes,amount)` ABI 已由官方 IDA 9.4 閉合：
  `0x20c6f` 把自己的 `a3/a4` 直接作 count/list，item row `+0x0e` word
  作 HP restore amount；callee 依 list 順序逐筆呼叫
  `0x1c916(target,amount)`，寫 current HP `+0x40` 並 cap max HP
  `+0x42`。dispatcher 尾端進一步證實 type5 restore 後跳
  `0x1b8e7` 消耗來源 slot，type13 則保留來源。`ApplyNativeItemHPRestore`
  保存 sequential RNG、atomic preflight 與這個 consumption 分歧。
  道具顯示名稱、renderer/SFX 仍 fail-closed；共享 `0x211a4` 的非 item
  caller 不改變這兩條 item branch 的已證實語意。
- `0x1bbdc` target transaction 已由官方 IDA 9.4/Capstone 閉合：row `+0x10` 是 first-stage raw mode、`+0x15` 是兩階段共用 target code；只有 type `0x17` 的 first stage 帶 inner marker 1。確認後以 row `+0x12` 從 confirmed cell 建 final list，inner marker 固定 0。`NativeItemTargetPlanFromRow`／`NativeItemEffectTargets` 保存此 ABI、confirmed-candidate gate 與 raw grid flags；不把三個 byte命名為 normalized range/AOE。
- `0x1c916` 的 HP mutation 已新增 `battle.ApplyNativeRawHPRestore`
  regression：RNG step、`amount*9/10 + (rng%100)*amount/1000`、
  current HP `+0x40` cap max HP `+0x42` 與 raw score gate 均保存。
  此 helper 仍是 shared primitive；只有 caller 已閉合的 type5/13 item
  route 可宣稱 item HP restore。
- 相鄰 `0x1c9dd` MP path 亦已新增
  `battle.ApplyNativeRawMPRestore`：同一 arithmetic 寫 current MP
  `+0x44`、cap max MP `+0x46`，score 僅用 `+0x21`、沒有 HP class
  bonus。`0x20c6f` type11 caller 已閉合成 consumable MP restore：
  max MP 為零的 target 跳過且不消耗 RNG，其餘依 list 順序恢復，最後
  移除來源 slot；IDs206/207 amounts=80/200。
- type12 已閉合為 retained HIT/EV modifier：`0x22997` 只處理 marker
  `+0x24==0` 的 target，成功才前進 RNG、寫 `(rng%4)+2`，並把 derived
  HIT/EV `+0x4c/+0x4e` 各加 15；dispatcher 不呼 `0x1b8e7`。tracked
  raw row 是 ID210。marker 的玩家可見名稱仍未知。
- type15/16 已閉合為 retained DP/AP modifier：marker
  `+0x23/+0x22` 為零才前進 RNG並寫 `(rng%4)+2`；derived
  DP `+0x4a`／AP `+0x48` 分別增加 `trunc(current×0.15+1)`。dispatcher
  不移除來源，tracked rows 是 ID213/214。
- type17/18/19 已閉合為 consumable maxHP/maxMP/MV modifier：
  `+0x42/+0x46` 分別加 row amount20；type19 對 word `+0x3b` 加1，
  caller 在 `0x21082` 前後保存/恢復 byte `+0x3c`，故只改 MV byte、
  EXP 不變。三條都由共用 callee `0x1b8e7` 消耗來源；IDs94/95/96。
- type20/21/24 已由 official IDA 9.4 caller ABI 與 Docker Capstone
  重核閉合：row word 直接成為 `0x1c75e(target,commandID)` 的 command
  ID。20/24 用 `0x1cd17` 十幀 presentation，21 用
  `0x2111a→0x1cac7`；兩個 presentation helpers 都不做 gameplay
  mutation。dispatcher 沒有 `0x1ca89` MP debit 或 `0x1b8e7` inventory
  removal，故來源保留。type20 IDs11/56/60→commands2/0/2，
  type21 IDs29/38/51/99→6/1/7/6，type24 ID79→command3。
- `battle.ApplyNativeRawWordSubtract` 的 arithmetic core 實際對應
  `0x1ca89`，不是 `0x1cac7`；既有 normalized `SpendNativeCommandMP`
  才保存 verified selector-success MP transaction。type20/21/24
  都不呼叫兩者。
- `0x22af6` 舊 adapter 把 marker 當 caller-owned parallel `flags[]` 是錯的，
  已撤回。正確 ABI 是 target runtime `record+a5`：type6/7 分別用
  `+0x25/+0x26`，nonzero 才以 base amount10 經 `0x1c916` 實際恢復
  9 HP、清該 record marker，最後 `0x1b8e7` 消耗來源。
  `ApplyNativeItemMarkerClearRestore` 保存 record-local mutation、RNG 與
  atomic source preflight；IDs196/197。status/presentation 名稱仍未知。
- `0x22d1b` application branch 的舊「兩次 RNG／固定10 damage」斷言已
  撤回。正確順序是：gate RNG；成功後 `0x1c81f(...,baseAmount=10)`
  再消耗 damage RNG，實際整數結果為 9 HP；第三 RNG 寫 marker。
  type14/22 item callers 分別用 marker `+0x26/+0x27`，來源保留；
  `ApplyNativeItemMarkerApplication` 保存 class exclusion、50% gate、
  三次 RNG、HP mutation、marker與 atomic preflight。status/UI 名稱仍未知。
- type23 `0x2218a` 已閉合為 post-confirm direct relocation：只取第一
  target，以 command23 record cost 對 actor current MP `+0x44` 做
  16-bit subtract；target class `+0x20`／level `+0x21` 形成 raw
  accumulator delta，最後把 destination cursor bytes 寫入 target
  `+0/+1`。actor gate 是 identity `+8==24`、max MP `+0x46>=20`；
  dispatcher 保留來源物品。落點 mode-6 raw legality已定位，但完整
  predicate 現由 `NativeRelocationDestinationAllowed` 保存：排除 other
  raw-active occupant，依 target class/race/unit+7 選 29×20
  `0x4e555` editable cost row，目的地 terrain entry 必須為20。完整
  indexed renderer/Ebiten selector仍未接。
- **[阻] 轉職系統**:攻略層有(Lv20+教會、轉職道具表 58h–60h→英雄/聖者/召喚師…,doc 02 §5.10);反組譯機制(職業數值替換、能力繼承、成長表切換)未做。
- **[阻] 轉職與 sprite**：已撤回「角色 id = DATO 肖像 = FDICON
  sprite組」的全域恆等斷言。`unit+2` 是 `0x11019` 回傳的 FDICON
  raw-key cache slot，`unit+7` 有獨立 constructor／class-change writer；
  DATO portrait 又由場景文字／設施資源選取。轉職後 sprite、portrait 與
  persistent identity 的映射必須分別追 writer/caller，不能以相同數字推導。

## 5. 對 remake 的暫行做法

數值都在 `item.json` / `unit.json` / `growth.json` + 攻略公式(doc 02 §4),所以 **M1 僅可作 normalized vertical slice：暫用「base(unit表)+裝備(item表 ap/dp)」與攻略公式**；這不是 native item/stat source 的證明，也不得接入 native command/effect/UI 或宣稱原版一致。反組譯機制(精確累加/使用效果/轉職)仍需後續校正，不阻塞戰鬥切片。

> 相關:doc 02(數值/公式)· 03(EXE 表)· 11(AI/傷害)· 13(物品選單)· 27(戰鬥規則)。資料:`docs/data/exe_tables/`。
