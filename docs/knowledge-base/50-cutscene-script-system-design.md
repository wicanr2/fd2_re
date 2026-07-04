# 50 — 過場腳本系統:原版指令集 → remake Beat DSL(全 33 關通用)

> 結論整理(2026-07-04,doc47/48/49 三線收束)。使用者戰略:**第一關指令破解後,
> 後續 32 關全部機械可解**——因為所有章節 handler 用同一套原語指令集,差別只在參數。
> 本篇定義 remake 腳本系統如何一比一承接。

## 1. 原版過場機制最終結論(全部實證)

三層架構,各層職責與還原狀態:

| 層 | 原版實體 | 還原狀態 |
|---|---|---|
| **編排** | EXE 每章 handler(跳表 0x51d71[章] 戰前 / 0x51de9[章] 戰後),線性呼叫原語 | 序章 0x3231b 已全轉錄(doc47);其餘章可機械抽取(§3) |
| **對白** | FDTXT 章文本,0x15f84(idx) 逐條播 | 35 檔全解+1533 句精校 |
| **演出** | acting 資源(表 0x50~0x99 共 74 筆),0x1366a(id) 播放 | 74 筆全 dump+解碼(dosbox-x) |
| **走位** | **引擎逐格步進單位(+0=X/+1=Y/+4=tick倒數)+鏡頭鎖定跟隨**(doc47 §9 實測) | 機制閉環;remake storyWalks+FollowWalk 同構 ✓ |

原語指令集(= 所有章節 handler 的「組合語言」):

| 原語 | 語意 | remake 對應 |
|---|---|---|
| `LOADCH` (0x205da) | 載章節地圖+文本(章節變數驅動) | 節點 map/script 欄 |
| `PAN(col,row)` (0x135dd) | 平滑鏡頭平移到格 | beat op:pan |
| `TXT(idx)` (0x15f84) | 播章文本第 idx 條(開框/頭像/翻頁) | beat op:dialog |
| `ACT(id)` (0x1366a) | 演出:批次設單位 pose/觸發走位,N 拍 | beat op:act(walk/pose) |
| `SPAWN(g)` (0x10b4e) | 群組 g 登場 | beat op:spawn |
| `JOIN(char)` (0x112a5) | 角色入隊伍名冊 | beat op:join |
| `BGM(track)` (0x25977) | 配樂切換/停止 | beat op:bgm |
| `FADEs` (0x13185)/`PALFADE` (0x1f525) | 淡變步/整幕 palette 淡入 | beat op:fade |
| `DELAY(ms)` (0x375b2) | 延遲 | beat op:delay |
| `REVEAL(n)` (0x32975/0x32999) | 攝影機 reveal 族(內部待展開,先當 pan 近似) | beat op:pan(近似) |

## 2. remake Beat DSL(campaign 節點新形態)

story 節點升級為 **cutscene 節點**:`beats:[{op,args…}]` 順序執行,一比一對映原語:

```json
{ "type": "cutscene", "map": "assets/maps/map32", "script": "assets/story/ch00_palace.json",
  "beats": [
    { "op": "pan",    "x": 3, "y": 34 },
    { "op": "walk",   "fig": 0, "from": [8,42], "to": [8,8], "follow": true },
    { "op": "dialog", "line": 0 },
    { "op": "pan",    "x": 0, "y": 43 },
    { "op": "bgm",    "track": 11 },
    { "op": "dialog", "line": 2 }, { "op": "act", "...": "…" }
  ] }
```

原則:
- beats 序列直接照抄章 handler 轉錄(doc47 §3/§7 即 ch1 的 beats 來源)。
- `walk`=ACT 中含位移的演出(引擎步進+可選 follow,同原版);`act`=純姿態(pose 循環/昏迷/阻擋)。
- 對白×演出**交錯**天然支援(beats 是平面序列,不再「一幕一段」)。
- 舊 story 節點(Lines/Scene/Actors)保留相容,逐步遷移。

## 3. 全 33 關機械破解管線

1. `tools/dump_chapter_beats.py`(待寫):走跳表 0x51d71/0x51de9 30 entry,
   對每支 handler 跑「push/call 配對抽取」(doc47 §7 的方法,已驗證),輸出
   `docs/data/chapter_beats/chNN_{pre,post}.json`(原語+參數序列,機器可讀)。
2. 轉換器:beats JSON + 章文本 + FDFIELD roster → campaign 節點(cutscene beats)。
3. 引擎 BeatRunner:main.go 依序執行 beats(pan/dialog/walk/act/spawn/join/bgm/fade/delay)。
4. 驗收:每章過場對照 dosbox 錄影(規則 65,對 reference 非內部訊號)。

## 4. 版權界線

beats JSON=行為事實轉錄(指令+參數),同 scenarios 屬原創整理可入庫;
acting 資源原始 bytes、dump 檔=版權物,永久留 extracted/(gitignore)。
