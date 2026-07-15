# ch1 草地/王城一隅(story_ch01_palace_path,亞雷斯撞見)— 原始資料 + 解讀註解

> 目的(使用者要求 2026-07-06):從資料面檢查草地幕怎麼從原版產生。**2026-07-15 已確認
> acting decoder 的 table 起點錯位 48 entries；改用 106-entry direct bank 後，本幕主角走位已閉環。**
> 機制總論見 `doc50`。

## 0. 對映
- **campaign 節點**:`story_ch01_palace_path`(map32 底部草地/庭院,亞雷斯撞見索爾)
- **原版來源**:EXE handler **`0x3231b`** Part1 後段(同 map32,鏡頭平移到草地列;FDTXT_033 scene[1])
- **主角**:草地索爾=**slot3**(4,46)、草地亞雷斯=**slot4**(13,47)(dump 確認,見 §3)

## 1. Handler beat 序列(反組譯 `0x323f5`~`0x3251c`)

| 位址 | 反組譯 | 原語 | 參數 | 說明 |
|---|---|---|---|---|
| 0x323f3 | `push 0x64; call 0x1366a` | ACT | ACT100 | slot2 向下十格，退朝 |
| 0x32407/b | `push 0x2b; call 0x135dd` | PAN | (col=0,row=0x2b=43) | 鏡頭平移到草地列(索爾4,46/亞雷斯13,47 在 row46/47) |
| 0x32415/7 | `push 0xb; call 0x25977` | BGM | track 0xb=11 | 草地配樂 |
| 0x32424/6 | `push 0x65; call 0x1366a` | ACT | ACT101 | slot4 左三格 |
| ~0x3245f | `push 0x66; call 0x1366a` | ACT | ACT102 | slot4 左二、上一、左一 |
| ~0x3249a | `push 0x67; call 0x1366a` | ACT | ACT103 | slot4 special 面左 |
| ~0x324d5 | `push 0x68; call 0x1366a` | ACT | ACT104 | slot3 special 面右兩拍 |
| ~0x3251a | `push 0x69; call 0x1366a` | ACT | ACT105 | slots3/4 離場走位 |
| (交錯) | `call 0x15f84` ×4 | 對白 | txt#2~5 | scene[1] 對白 |

> 本段不需要額外 step：ACT101/102/105 的 normal frames 本身就是逐格走位。

## 2. 演出(acting)direct 解碼 ★本檔重點★

正確來源是 `FD2.EXE file+0x565d8` 的 direct-ID directory；editable 行為在
`remake/assets/cutscenes/acting/map32.json`，可由 `tools/export_acting_resources.py --check` 驗證。

| ID | editable frames | 對座標的結果 |
|---:|---|---|
| 101 | normal `slot4 left×3` | 亞雷斯 `(13,47)→(10,47)` |
| 102 | normal `slot4 left×2 → up×1 → left×1` | `(10,47)→(7,46)` |
| 103 | special0 `slot4 left` | 原地定向／轉場 |
| 104 | special2 `slot3 right` | 索爾原地面右 |
| 105 | slot3 `right×2/down×1/right×6`；slot4 `down×1/right×4` | 兩人依劇本離場 |

舊 0x65..0x69 的 slots16/17 bytes 是從錯誤 table base 解到的其他資源，已撤回。

## 3. 單位陣列(roster,dump `task_f/slots0_20_dialogue.bin`)

| slot | 角色 | X | Y | 這一幕角色? |
|---|---|---|---|---|
| 0 | 國王 | 7 | 5 | 否(王座) |
| 1 | 王后 | 10 | 5 | 否(王座) |
| 2 | 索爾 | 8 | 21 | 否(傳位幕的索爾) |
| **3** | **索爾** | **4** | **46** | **✅ 草地索爾** |
| **4** | **亞雷斯** | **13** | **47** | **✅ 草地亞雷斯** |
| 5-15 | 守衛 | … | 14~36 | 否 |
| **16,17** | **守衛(69)** | 12,6 | 16,28 | **否(王座區守衛,草地時在畫面外)** |

## 4. remake 對映

`campaign_full.json` 預設由 `story_ch00_handler` 載入 `bindings/ch00_pre.json`；草地不再走舊手寫
`storyWalk/exit_walk` 節點，而是依 source addresses 直接播放 ACT101..105 editable frames。

## 5. 目前追查狀態(2026-07-15)

已解，無額外 acting caller／ID 待追。normal-core 的 slot4 `(13,47)→(7,46)` 快照正好等於
ACT101+102 累積結果；完整證據鏈與 runtime 規則見 doc50 §1.2。
