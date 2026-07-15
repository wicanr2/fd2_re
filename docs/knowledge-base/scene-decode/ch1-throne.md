# ch1 皇宮傳位(story_ch01_palace_throne)— 原始資料 + 解讀註解

> 目的(使用者要求 2026-07-06):從**資料面**檢查 `campaign_full.json` 的 throne 節點是怎麼從原版資料
> 產生的、我怎麼解讀每個 opcode/byte、哪裡可能沒解乾淨。**原始 binary(hex+ascii)+ 我的註解並列**,供人工覆核。
> 機制總論見 `doc50`(過場機制主檔);本檔只做「這一幕」的原始資料 × 解讀對照。

## 名詞(先讀這個,避免看不懂)
- **beat**:過場動作序列裡的「一步 / 一個 cue」——移動鏡頭、顯示一句對白、播一段演出、走一步。
  handler 就是照順序把這些 beat 跑完(像分鏡腳本一格一格)。「§1 beat 序列」= 這一幕依序做的動作清單。
- **演出(acting)**:由 `0x1366a` 播放；正常 frame 依 pose 每拍走一格，特殊 frame 只做原地姿態。
- **拍數**:正常 frame 為格數；特殊 frame 為顯示節奏。完整機制見 `doc50 §1.2`。
- **pose(姿態/方向)**:`0=下 1=左 2=上 3=右`。
- **slot / unit 索引**:單位陣列 `[0x53a45]` 裡第幾格(每單位 0x50=80 bytes;+0=X格 +1=Y格 +3=pose +4=tick +8=角色ID)。

## 0. 對映
- **campaign 節點**:`story_ch01_palace_throne`(map32,傳位對話)
- **原版來源**:EXE handler **`0x3231b`** Part1 前段(暫借章節 `0x20`=32 → 載 map32 + FDTXT_033)
- **走位機制**:step 家族 `0x13185`(往上一步,計數迴圈)——**不是 acting**(見 doc50 §1.1)

## 1. Handler beat 序列(反組譯 `0x3231b`~`0x323f5`,逐 call)

| 位址 | 反組譯 | 原語 | 參數 | → campaign beat |
|---|---|---|---|---|
| 0x32326 | `mov [0x3c03],0x20` | 設章節=32 | | (map: map32) |
| 0x32330 | `call 0x205da` | LOADCH | 載 map32+FDTXT_033 | node.map |
| 0x32335/9 | `push 0x22; call 0x135dd` | PAN | (col=3,row=0x22=34) | `pan (72,816)`=(3,34)×24 |
| 0x32341/3 | `push 0x63; call 0x1366a` | ACT | 演出 0x63(見 §2) | — (索爾進場,實走靠 step) |
| 0x32351 | `call 0x13185` ×15 | STEP↑ | cmp eax,0xf=15 步 | `walk (8,21)` |
| 0x32382 | `…push 0x13/0x4a/0x4c/0xcd/0x140/0xa0000; call 0x15f84` | 對白 | txt#0 | `dialog line0` |
| 0x3239a | `call 0x13185` ×13 | STEP↑ | cmp eax,0xd=13 步 | `walk (8,8)` |
| 0x323cb | `…; call 0x15f84` | 對白 | txt#1 | `dialog line1(count18)` |
| 0x323e1 | `call 0x25977` | BGM | 停/切 | node.bgm |
| 0x323f3 | `push 0x64; call 0x1366a` | ACT | 演出 0x64(退朝,見 §2)| →接草地段(見 ch1-meadow) |

> 對白 0x15f84 前那串固定 push(0x13,0x4a,0x4c,0xcd,0x140,0xa0000)= 對話框繪製參數(0xa0000=VGA記憶體、
> 0x140=320寬),**不是走位資料**;真正的文字索引在最後一個 push(這裡是章文本游標 [0x53a79])。

## 2. 演出(acting)原始資料 + 解碼

> **2026-07-15 live 更正：舊 dump 從錯誤的 `0x207718` 位置起讀，造成 ID 錯移 48 entries。**
> 下列舊 unit60/61 bytes 已撤回；getter 實際使用 direct table `0x2077d8`，完整 106 entries。

> 格式(decode_acting.py):`u8 幀數 + 幀×{ u8 拍數(bit7=模式旗標/低7位=真拍數), u8 N, N×(u8 單位idx, u8 姿態) }`。
> pose:0下/1左/2上/3右。normal/special 完整語意見 `doc50 §1.2`。

### 演出 0x63 / ACT99（handler `0x32343`）

Live entry stack 回扣 caller `0x32343`，getter `table[99]=0x208493`，resource bytes：

```text
01 06 01 02 02  -> 1 frame, beats=6, slot2, pose2(up)
```

完整 unit buffer 前後只有 slot2 改變：Y `42→36`、pose `0→2`。隨後 handler 的兩段
`0x13185(slot2)` 分別再上移 15、13 格，形成 `Y36→21→8` 的兩個對白停位。

### 演出 0x64 / ACT100（handler `0x323f5`）

Live entry stack 回扣 caller `0x323f5`，resource bytes：

```text
01 0A 01 02 00  -> 1 frame, beats=10, slot2, pose0(down)
```

播放後 slot2 Y `8→18`、pose=`0`。舊 slot60 結論同樣是錯表位移造成。

## 3. 單位陣列(roster,dosbox dump `task_f/slots0_20_dialogue.bin`,對話中快照)

| slot | 角色(charID) | X | Y | pose | 說明 |
|---|---|---|---|---|---|
| 0 | 國王(48) | 7 | 5 | 0 | 王座 |
| 1 | 王后(66) | 10 | 5 | 0 | 王座 |
| **2** | **索爾(0)** | 8 | 21 | 2 | **傳位這幕走王座的索爾**(此快照在 (8,21)=第一次對話位置) |
| 3 | 索爾(0) | 4 | 46 | 0 | (草地那幕的索爾,見 ch1-meadow) |
| 4 | 亞雷斯(4) | 13 | 47 | 0 | (草地那幕) |
| 5-20 | 守衛(68×8/69×8) | 5/6/11/12 | 14~40 | 0 | 長廊儀隊 |

> 傳位這幕動的是 **slot2**(索爾),0x13185 每呼叫 push 2 = 對 slot2 步進。

## 4. campaign_full.json 對映(現值)
```json
"story_ch01_palace_throne": {
  "map":"assets/maps/map32", "cam_x":0,"cam_y":0,"cam_max_y":808,
  "actors":[ {"fig":48...國王}, {"fig":66...王后}, ...16守衛 dir:0..., {"fig":0,"x":8,"y":42,"dir":2}索爾 ],
  "beats":[ pan(72,816), walk(8,21) follow, dialog line0, walk(8,8) follow, dialog line1 count18 ] }
```
- 守衛座標 = §3 roster(逐筆吻合);守衛 dir=0(原版面向玩家,FDFIELD 不存面向=zero-init 預設)。
- walk 停位 (8,21)/(8,8):對原版截圖 + FDFIELD 守衛地標實測(doc50 §1.1)。
- 對話切分 line0 / line1-18:依 §1 的 call 序列(STEP×15→對話→STEP×13→對話)。

## 5. 本幕限定解讀(2026-07-15)

本幕走位已閉環：ACT99 上六格，STEP×15 到第一次對話，STEP×13 到王座前，ACT100 再向下十格。
不存在 slot60/61 或「acting 只當節拍器」的未解假說。完整 decoder 與 runtime 規則見 doc50 §1.2。

## 附錄:acting bytes 怎麼讀(反組譯播放器 `0x1366a` 證明)

> 格式不是猜的,是反組譯**「讀這些 byte 的程式」= 演出播放器 `0x1366a`** 得來。
> 正確 bytes 由 `FD2.EXE file+0x565d8` 的 106-entry directory + `file+0x53e00` data bank 重建；
> `tools/export_acting_resources.py` 可重跑。舊 `acting_resources_0x50_0x99_throne.bin` 僅保留作錯 context
> 的考古樣本，不得再餵給 binding。格式仍是 `u8 幀數 + 幀×{u8 拍數,u8 N,N×(u8 slot,u8 pose)}`。

| 欄位 | 讀取指令(0x1366a) | 用途證據(它拿這個值幹嘛) |
|---|---|---|
| **byte[0] = 幀數** | `0x13687 mov dl,[eax]` → `[esp+0x48]` | `0x13837 cmp,[esp+0x48]; 0x1383e jge 結束` = 當**幀迴圈上限** |
| **拍數**(每幀 byte0) | `0x13844 mov al,[ebp]` → `[esp+0x54]` | `0x13825 cmp,拍數; 0x1382c jl 0x13803` = 當「顯示幾個 tick」上限(0x13803=`call 0x11cac 重繪`+`0x17aa9 等一tick`)= **幀持續時間** |
| **N**(每幀 byte1) | `0x1384b mov al,[ebp+1]` → `[esp+0x44]` | `0x136b3 cmp,N; 0x136ba jl 讀對` = 當「讀幾對」上限 = **這幀設幾個單位** |
| **(unit,pose)**×N | `0x1369a` 對迴圈 | `0x13882 a×0x50`+`0x13891 [0x53a45]基址`+`0x1389a mov [單位[a]+3], b` = **寫 單位[a] 的 +3(pose)=b** |

本附錄的舊 bit7 使用碼解讀曾在 `0x13918` 截斷，故其「不寫格座標」結論已失效；正確尾段與實機寫入
證據集中在 `doc50 §1.2`。本檔的原始 bytes 仍可作為本幕資源資料佐證。
