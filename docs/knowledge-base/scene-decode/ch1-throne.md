# ch1 皇宮傳位(story_ch01_palace_throne)— 原始資料 + 解讀註解

> 目的(使用者要求 2026-07-06):從**資料面**檢查 `campaign_full.json` 的 throne 節點是怎麼從原版資料
> 產生的、我怎麼解讀每個 opcode/byte、哪裡可能沒解乾淨。**原始 binary(hex+ascii)+ 我的註解並列**,供人工覆核。
> 機制總論見 `doc50`(過場機制主檔);本檔只做「這一幕」的原始資料 × 解讀對照。

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

> 格式(decode_acting.py):`u8 幀數 + 幀×{ u8 拍數(bit7=模式旗標/低7位=真拍數), u8 N, N×(u8 單位idx, u8 姿態) }`。
> pose:0下/1左/2上/3右。**bit7 特殊模式語意未完全確認**(見 §5)。

### 演出 0x63(handler beat「索爾進場」)
```
原始 bytes: 02 05 01 3d 00 84 01 3d 00
  +00: 02              幀數=2
  +01: 05 01 3d 00     frame0: 拍數=5 N=1 (unit=0x3d=61, pose=0)
  +05: 84 01 3d 00     frame1: 拍數=0x84→bit7,真拍數=4 N=1 (unit=61, pose=0)
```
**⚠ 可疑**:unit=**61** 超出 roster(只有槽 0-20)=**無效槽**。所以 0x63 沒有真的動任何單位;
索爾長廊走入的**位移是 0x13185(step)做的**,0x63 疑為**節拍/鏡頭 setup**(bit7 分支會重繪地形,見 §5)。

### 演出 0x64(handler beat「退朝」)
```
原始 bytes: 02 05 01 3c 00 84 01 3c 00
  +00: 02              幀數=2
  +01: 05 01 3c 00     frame0: 拍數=5 N=1 (unit=0x3c=60, pose=0)
  +05: 84 01 3c 00     frame1: 拍數=4[bit7] N=1 (unit=60, pose=0)
```
**⚠ 可疑**:unit=**60** 也是無效槽。同 0x63,疑節拍/鏡頭而非單位動作。

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

## 5. 可疑點 / 解碼完整性疑慮(供使用者檢查)
1. **無效槽 u60/u61**:0x63/0x64 解出的 unit 索引超出 roster。要嘛(a)這些演出本就是節拍/鏡頭 dummy、
   單位位移交給 step;要嘛(b)**bit7 模式下 (unit,pose) 兩 byte 的語意根本不是 (單位,姿態)** ← 若是後者,則我的解碼在 bit7 幀是錯的。
2. **bit7 特殊模式**:decode_acting.py 自標「語意待實測、不臆測」。反組譯線索:bit7 分支(0x1370b)會
   `push 鏡頭座標 [0x53aa9]/[0x53aad]; call 0x11eee(地形重繪)`——**疑 bit7 幀是「鏡頭/捲動」相關而非單位姿態**。
   若成立,0x63/0x64 的 frame1(bit7,u61/u60)其實在做鏡頭事,不是動 u61/u60。
3. **這一幕的走位不在 acting**(是 0x13185 step),所以 acting 解碼是否完整**不影響傳位這幕的還原**;
   但它是「bit7 到底在做什麼」的線索來源(草地那幕才卡在這,見 ch1-meadow §5)。
