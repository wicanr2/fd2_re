# 24 — Call-graph 逐步反組譯紀錄(釘死 doc 23 受阻項)

> doc 23 §7 留了一個受阻項:cutscene `ret` → 進戰場 `0x10010` 的「精確呼叫鏈」用**線性 sweep**(disasm_le 的 `range`/`calls`、le_xref 的 raw-`0xe8` 掃描)釘不死——線性反組譯會在資料區/跳表漂移,產生偽指令與偽 caller。
> 本篇改用**遞迴可達性反組譯**(`tools/callgraph_le.py`)逐步釘死,並**每一步與先前結論/記憶比對**,主動排除殘留的錯誤結論(rulebook 62/63)。
> 結論先講:**進戰場鏈 = `main 0x25bf4 → driver 0x25ebb → 0x10010`,cutscene(章節跳表)與進戰場 call 同在 0x25ebb driver 內,線性串接,不經神秘相位機。**

## 方法:為何遞迴可達 > 線性 sweep

線性 sweep 從某位址逐位元組解碼,遇到資料(跳表、字串、對齊填充)會把它當指令,產生「偽 call」。
遞迴可達反組譯只從**種子函式**出發,跟隨真正會執行到的 `call`/`jcc`/`jmp`(立即數)目標,標記「可達指令集」。
於是「誰呼叫 X」只回報落在可達集內的 call —— **資料偽命中自動被排除**(它們不可達)。
工具:`callgraph_le.py`,子命令 `reach`/`callers`/`rpath`/`funcof`/`edges`/`jtab`。

## 步驟 1 — 建可達集

```
$ callgraph_le.py FD2.EXE reach 0x25bf4
種子 ['0x25bf4']:可達指令 46782,direct call 點 4218
```
從真 main(0x25bf4,doc 23 已驗證)出發,46782 條可達指令、4218 個直接 call 點。涵蓋面足以做全域 caller 分析。

## 步驟 2 — 釘死 0x10010(進戰場)的真 caller

```
$ callgraph_le.py FD2.EXE callers 0x10010
可達 caller of 0x10010:
  call @ 0x01a251
  call @ 0x026130
```

**比對(關鍵):**
| 來源 | 對 0x10010 caller 的說法 | 裁決 |
|---|---|---|
| `le_xref calls`(raw 0xe8 線性掃描) | **0 處** | ✗ 漏報(相對位移計算/範圍限制) |
| doc 23 撰寫時某 agent 猜測 | 0x1b051 / 0x26f30 | ✗ **偽命中**(落在漂移/資料區,不可達) |
| **callgraph(可達)** | **0x1a251、0x26130** | ✅ 採信(兩者皆在可達集內) |

→ doc 23 §7 的受阻項,根因正是「線性工具的偽命中」;遞迴可達一步解決。

## 步驟 3 — 兩個 caller 的語境(disasm 佐證)

```
caller B 0x26130:                       caller A 0x1a251:
  0x26124 push 0; push -1               0x1a245 push 0; push -1
  0x26128 call 0x25977  ; play_bgm(-1)   0x1a249 call 0x25977 ; play_bgm(-1)
  0x26130 call 0x10010  ; 進戰場          0x1a251 call 0x10010 ; 進戰場
  0x26135 play_bgm([0x53c03]→[0x51e63])  0x1a256 jmp 0x1a193  ; 回模組迴圈
```
兩者同模式(先停曲再進戰場);B 之後放「該章 BGM」(章節曲表 0x51e63),A 之後跳回所屬模組迴圈。

## 步驟 4 — 反向追到 main + 函式歸屬

```
$ callgraph_le.py FD2.EXE rpath 0x10010
0x25bf4 → 0x25ebb → 0x10010

$ callgraph_le.py FD2.EXE funcof 0x260f5   → 0x25ebb   (章節跳表 call = cutscene)
$ callgraph_le.py FD2.EXE funcof 0x26130   → 0x25ebb   (進戰場 call)
$ callgraph_le.py FD2.EXE funcof 0x1a251   → 0x19df7   (讀檔/過場子模組)
```

**判讀:**
- **caller B(0x26130)在 driver `0x25ebb` 內**,與「呼叫章節跳表(cutscene)的 call」(0x25f10 新遊戲分支 / 0x260f5 讀檔分支,funcof 同為 0x25ebb)**同屬一個 driver 區段**。
  → **cutscene 與進戰場是同一 driver 內的線性流程**:driver 先 `call [章節*4 + 0x51d71]`(章節 0 = cutscene 0x3231b),該 handler `ret` 後,driver 繼續走到 `0x26130 call 0x10010` 載入並進入戰場。中間**沒有玩家選擇點**,即「自動過場」。
- **caller A(0x1a251)在子模組 `0x19df7`**(讀檔子選單 / 過場,doc 23 §1)——是另一條獨立的進戰場入口(讀檔續戰 / 過場後進場),非新遊戲開場路徑。

→ doc 23 的「新遊戲 cutscene → 自動進第一場戰場」鏈,至此**端到端釘死**:
```
main 0x25bf4
  └ driver 0x25ebb
       ├ [0x53c03]=0                         ; 新遊戲歸零章節
       ├ call [0*4 + 0x51d71] = 0x3231b      ; 開場 cutscene(與前代主角對話)→ ret
       └ call 0x10010                        ; 進戰場(地圖 = 章節*3+2)
```

## 步驟 5 — 獨立驗證章節跳表(修了一個工具 bug)

第一次 `jtab 0x51d71` 回報 `raw 0x0`:**發現工具 bug** —— 跳表 0x51d71 在 **obj2 data 段(0x50000+)**,而 fixup 解析與讀取只涵蓋了 obj1 code 段。修正 `fixup_map` 改掃**全部 object 的 fixup pages**、`page_base_linear` 依 page 所屬 object 換算 linear 後:

```
跳表 A 0x51d71(章節→戰前/劇情):  [0]0x3231b [1]0x32d18 [2]0x32e8c [3]0x32fb2 [4]0x33049 …
跳表 B 0x51de9(章節→戰後/勝利):  [0]0x22ef6 [1]0x22f37 [2]0x230f2 [3]0x231bc [4]0x231f9 …
驗 [0]:0x32326 mov [0x53c03],0x20 ; call 0x205da   → 確為 cutscene
```

**比對:** 與 doc 23 §4/§5 中由 agent 給出的跳表內容**完全一致**——但這次是修好 data 段 fixup 後**獨立重現**,不是照搬。agent 的結論於此升級為「已自驗」。

## 與先前結論/記憶的比對總表(避免殘留錯誤)

| 項目 | 先前狀態 | 本輪裁決 |
|---|---|---|
| 進戰場 0x10010 caller | doc 23 §7 標 **[阻]**(線性工具釘不死) | ✅ **已解**:0x1a251、0x26130(可達驗證) |
| 0x1b051 / 0x26f30 是 caller | 某 agent 推測 | ❌ 偽命中,刪除 |
| cutscene→戰場是否經相位機 | doc 23 用「相位機接手」描述 | 修正:**同 driver 0x25ebb 內線性串接**;相位變數 `[0x53ecc]` 是**戰後**分支用(1 事件/2 勝利),非 cutscene→戰場中介 |
| 章節跳表 0x51d71/0x51de9 內容 | agent 給出 | ✅ 獨立驗證一致 |
| main = 0x25bf4 | doc 23 已驗證 | ✅ rpath 再確認(0x10010 經 0x25ebb 上溯到 0x25bf4) |
| 記憶層(5 個 fd2-* 記憶) | — | ✅ 無一含被推翻的結論(都是耐用事實/教訓) |

## 本輪受阻 / 待續(誠實標註)

- **[阻]** `0x25ebb` 是個大 driver,內部「新遊戲分支 / 讀檔分支 / 戰役續打迴圈」的精確邊界與各分支條件未逐條展開(只確認三者匯流到 0x26130 進戰場)。`funcof` 用「最近 call target」近似函式宿主,對「非被 call、靠 fall-through 進入」的相鄰函式入口會歸併——描述以「driver 區段」為準,未強斷 C 函式邊界。
- **[阻]** 相位變數 `[0x53ecc]` 的完整戰後狀態圖(1=事件 0x22e5c、2=勝利跳表 0x51de9)未逐分支追完。
- **工具**:`callgraph_le.py` 的 `edges`/`path` 為近似版;間接呼叫(跳表)需手動以 `jtab` 解出再分析。

> 相關:doc 23(開場流程,本篇釘死其受阻項)· doc 12(BGM)· doc 19/21(重製對映)。工具:`tools/callgraph_le.py`、`tools/disasm_le.py`、`tools/le_xref.py`。
