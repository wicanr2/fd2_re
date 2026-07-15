# 53 — START→開場→第一關第一回合:逐項 RE 來源對照表(禁推測驗收表)

> 目標(記憶 fd2-goal):START→開場→第一關第一回合我方行動,**每一步跟原版一模一樣**。
> 鐵則:**禁推測/外推**,每個進 code 的值要有 RE 來源。本表把整段流程拆成最小元素,逐項標:
> **來源**(handler=doc47 §7 反組譯 / dosbox=doc48 / FDFIELD 直讀 / 青衫 / 影片)、**remake 現況**、
> **判定**:✅可寫(有RE) / ⚠須換(現況是外推/錯值) / ❓待RE(多輪未果,外推前先問使用者)。
>
> 用法:實作前逐列比對;⚠ 先換成 RE 值;❓ 先窮盡靜態/影片,真的多輪失敗才問使用者是否外推。

## A. START(標題→選單)

| 元素 | 原版 | 來源 | remake 現況 | 判定 |
|---|---|---|---|---|
| 開場 ANI 過場(爪痕/龍/騎士/「2」) | ANI.DAT AFM 全解 | doc39 | title.go cutScript 已接 | ✅ |
| 標題畫面 + logo「2」縮放 | FDOTHER #7/#0x45-49 | doc23 §2 | title.go 已接 | ✅ |
| 選單 START/LOAD/CONTINUE | 選 START→進遊戲 | doc23 | title.go titleSel,START→titlePhase="" | ✅ |

## B. 開場 Part 1:王座廳 + 草地(handler 章節0x20=map32,doc47 §7)

handler 序列(反組譯,✅來源確定):
`pan(3,34)→act(0x63)→scroll(2)×15→txt#0→scroll(2)×13→txt#1→bgm停→act(0x64)→pan(0,43)→bgm(11)→palfade→act(0x65)→txt#2→act(0x66)→txt#3→act(0x67)→txt#4→act(0x68)→txt#5→act(0x69)`

| 元素 | 原版值/機制 | 來源 | remake 現況(campaign_full.json throne) | 判定 |
|---|---|---|---|---|
| 起始鏡頭 | `pan(3,34)` | handler + PAN body ✅ | binding `pan(72,816)` | ✅ |
| 索爾走入起點 | (8,42) | FDFIELD map32 slot2 ✅ | walk from(隱含) | ✅ |
| 索爾走入=act(0x63) | 引擎逐格步進+pose循環 | handler + doc47§9 ✅ | walk beat(storyWalks) | ✅機制 |
| 走入**速度/幀數**(每格幾 tick) | act#0x63=2幀(拍5+bit7\|4);ticks/格未定 | 部分RE | walk frames=278(外推自450) | ❓待RE(靜態未給明確 ticks/格;影片可數格/秒) |
| 鏡頭跟隨索爾=scroll(2) | **跟隨 slot2 上捲步**,讀unitY改camY | **0x13185 靜態RE定案✅**(本輪) | pan(72,120)/(72,0)外推 | ⚠須換成 scroll-follow(非外推pan) |
| scroll 次數 ×15、×13 | 迴圈自動偵測=15、13 | handler(chapter_beats)✅ | 無(用外推pan替代) | ✅可寫(15/13) |
| txt#0/#1 對白 | FDTXT_033 string0/1(內含多頁0xFFFD) | FDTXT轉錄✅ | dialog line0/1 count1 | ⚠須換:一個 TXT=一整條FDTXT字串(多頁),非「一行」 |
| 王座廳共幾句 | FDTXT_033=**6 條字串**(txt#0-5),非18行 | FDTXT直讀✅ | ch00_palace.json 18行(攤平) | ⚠須對齊 FDTXT 分條 |
| 國王/王后位置 | (7,5)/(10,5) portrait48/66 | FDFIELD slot0/1 ✅ | actors已擺 | ✅ |
| 長廊守衛×16 | portrait68/69 y14-40 | FDFIELD ✅ | actors已擺 | ✅ |
| 王城→草地轉場 | `pan(0,43)`同map32平移(非換圖/淡出) | handler + PAN body ✅ | binding `pan(0,1032)`；explicit PAN 不受 follow cap 裁切 | ✅機制 |
| 草地配樂 | `bgm(track11)` | handler ✅ | — | ⚠須加 |
| 草地幕 act(0x65-69)+txt#2-5 | 演出/對白交錯 | handler ✅ | path節點 | ⚠須照序列 |
| act(0x63-69)**幀內容** | acting表 id0x63-69 已dump解碼 | doc48/acting_decoded ✅ | BeatRunner act=方向近似,**未接解碼資料** | ⚠須接 acting_decoded |

## C. 開場 Part 2:密林(handler 章節0x1f=map31)

| 元素 | 原版 | 來源 | 現況 | 判定 |
|---|---|---|---|---|
| 序列 | `pan(5,42)→spawn(1)→act(0x5a)→txt#0→…→spawn(3)→pan(4,41)→…→reveal75(2)→spawn(5)→…act(0x5e-61)…→act(0x62)` | handler ✅ | forest_duel/discover 部分 | ⚠須照序列重排 |
| 索爾/亞雷斯起點 | (19,46)/(19,47) | FDFIELD map31 ✅ | 已用 | ✅ |
| 蓋亞/悠妮位置 | (5,43)/(5,44) | FDFIELD map31 ✅ | 已用 | ✅ |
| 索爾+亞雷斯走向悠妮蓋亞 | act 內走位(14格外) | handler+FDFIELD ✅ | walk beat | ✅機制 |
| 蓋亞阻擋/悠妮昏迷 staging | act(0x5e-61)幀(11單位複合等) | acting_decoded ✅ | 未接 | ⚠須接 acting_decoded |
| activate/spawn intro | 0x32975(slot)=flags1；0x32999(group)=spawn+12-step present | 完整 callee body ✅ | activate_unit / spawn_intro | ✅ |

## D. 開場 Part 3:入隊 + 海島 + 進戰場(handler 章節0)

| 元素 | 原版 | 來源 | 現況 | 判定 |
|---|---|---|---|---|
| 入隊 | `join(0/9/4/30)`=索爾/悠妮/亞雷斯/蓋亞 | handler ✅ | runtime 保存 membership + JOIN chronology | ✅ |
| 載真戰場 | `loadch(0)`=map0+FDTXT_001 | handler ✅ | story_ch01→map0 | ✅ |
| 海島鏡頭 | `pan(4,12)→pan(0,0)→pan(0,15)`三平移點 | handler + PAN body ✅ | binding `96,288 → 0,0 → 0,360` | ✅ |
| **主角隊／海盜進場移動** | act(0/1/2/5)+reveal；ACT0=party slots0–3 up×6，ACT1/2=spawn groups，ACT5=海盜 slot9 down×4 | 各 call 的 live provenance（舊「map0 getter base」泛化撤回） | editable acting + party→spawn slot pipeline | ⚠逐 call 保留證據 |
| 主角隊 ACT0 起點→停位 | 索爾(7,20)→(7,14) / 悠妮(10,21)→(10,15) / 亞雷斯(8,22)→(8,16) / 蓋亞(11,23)→(11,17) | map0 runtime slot dump + ACT0 解碼 ✅ | ch01 deploy_cells + editable ACT0 | ✅ |
| 海島遇海盜對白 | FDTXT_001 txt#0-2 | FDTXT ✅ | story_ch01 script | ✅ |
| 戰前UI(MAP/TURN+行軍確認) | 有 | 影片 doc46 D8 ✅ | 無 | ⚠須加(低優先) |

## E. 第一關第一回合(系統 B,battle;青衫=事件 ground truth)

| 元素 | 原版 | 來源 | 現況 | 判定 |
|---|---|---|---|---|
| 開局對白 | 「累死了,大家休息一下吧!」(索爾) | 青衫/FDTXT/ch01.json ✅ | on_battle_start event | ✅ |
| 敵方 | LV2盜賊×7+×4+×4、LV3海盜頭目 | 青衫 ✅ | ch01 groups | ✅骨架 |
| 我方名冊 | 索爾/亞雷斯/悠妮/蓋亞(戰1章) | 青衫/ch01 ✅ | party | ✅ |
| 勝/敗 | 敵全滅 / 索爾死 | 青衫 ✅ | ch01 | ✅ |
| 第一回合玩家操作 | 選單:移動(行軍確認YES/NO)/攻擊/待機/END | doc13戰場選單 ✅ | 指令環部分 | ⚠見doc51(結束回合/武器射程/法術/狀態欄) |
| 攻擊距離 | 依武器(騎士騎槍2格) | 物品表靜態(doc32)✅可解 | InAttackRange寫死相鄰 | ⚠須換(靜態RE,不需dosbox) |
| 哈諾父子登場 | **第三回合**己方結束(青衫確認turn3) | 青衫 ✅ | ch01 hano turn3 ✅ | ✅ |

## F. 判定彙總(要動手先解這些)

- **⚠須換(現況是外推/錯值,有RE值可直接換)**:王座廳 pan(3,34)、scroll-follow 取代外推pan、
  對白改「一FDTXT字串=一TXT」對齊6條、王城→草地同圖pan、接 acting_decoded、海島三平移、入隊、武器射程(靜態)。
- **✅ 2026-07-15 補齊**：direct acting bank 共 106 entries；ACT99 live 為 slot2 上行六格，ACT100
  live 為 slot2 下行十格；兩段 scroll_step 實測 `Y36→21→8`，每格七 ticks。map31 ACT90..98 的
  direct slots 皆落在當時已 spawn 的 2→4→5 個 active units 內，不再使用 `timing_only`。
- **✅可直接寫**:START全段、FDFIELD所有座標、handler序列與scroll次數、0x13185機制、青衫事件骨架。

> 下一步建議:先做全部 ⚠(都有RE值,純照抄,無推測);❓ 逐項評估「靜態/影片能不能補」,不能才問你。
