# 42 — RE 已記錄 vs remake 已實作:落差稽核

> 目的:逐一核對「RE knowledge-base 已記錄的機制」與「remake 程式碼實際做了什麼」,列出落差與優先度。
> 方法:每項機制先讀對應 doc,再 grep/讀 `remake/internal/battle`、`remake/cmd/fd2` 的實作,以 code 為準,不憑印象判定。
> 唯讀稽核,未修改任何程式碼。序章主角隊進場(staging)由另一 agent 處理,本篇不重複列。
> 狀態符號:✅已實作(含公式/資料對齊) 🟡部分(做了一半或簡化) ❌缺(RE 有記錄,remake 未做)。

## 總表

| 機制 | RE doc 出處 | remake 狀態 | 證據 | 優先度 |
|---|---|---|---|---|
| 物理攻擊:基礎傷害 AP−DP | doc02 §4.1 | ✅ | `combat.go:12` `dmg := a.EffectiveAP() - d.EffectiveDP()` | — |
| 物理攻擊:**地形 AP/DP% 修正** | doc02 §3.2、doc11、doc32 §2 | 🟡已補(近似) | `terrain.go TerrainAPDPPct` + `combat.go AttackWithRNG`/`estDamage` 已接上一般/沼澤兩類(AP+5%/DP-5%、AP-5%/DP-5%);**森林(AP-5%,DP+10%)因 map.json 的 `cost[]` 把森林/正常都存成 cost=1(export_engine_assets.py 既有限制)無法分辨,誠實標記於 terrain.go 檔頭**,待該匯出管線補獨立地形代碼欄位才能收斂 | 中(森林細節待補) |
| 物理攻擊:**暴擊(DP÷2)** | doc02 §4.1「暴擊時 DP=守方DP/2」 | ✅ | `combat.go AttackWithRNG`:`CritPct>0 && rng.Intn(100)<a.CritPct` 觸發後 `dp/=2`,順序照 notes.md(先減半再套地形%);`CritPct` 來源 `resist_crit.json`(EXE 0x5219B,已與 doc02 §7.2 逐職業交叉驗證吻合) | — |
| 物理攻擊:**命中率 (HIT−EV)%** | doc02 §4.1 | 🟡已補(HIT/EV 為近似值) | `model.go EffectiveHIT/EffectiveEV` + `combat.go rollsHitPct`;**HIT/EV 兩個基礎值本身是固定近似值(export_units.py DEFAULT_HIT=90/DEFAULT_EV=5)**,因為 doc03 明確記載這是「衍生值(由上面計算,直接改無效)」而非「敵/友單位 10B」表的原始欄位,且 remake 尚無裝備系統可提供真正來源(item.json 的 hit/ev 是掛在武器/防具上)——**doc42 原敘述「只是匯出腳本未取用」不完全準確,實際是來源表本身缺這兩欄**,見 export_units.py 檔頭更正說明 | 中(HIT/EV 真值待裝備系統) |
| 物理攻擊:**傷害隨機化(0.9×max~max−1)** | doc02 §4.1 | ✅ | `combat.go AttackWithRNG` 呼叫 `magic.go randomizeAmount`(與法術共用同一公式) | — |
| 劍技(破龍擊/熾炎刀/音速刃/淒煌斬):AP×加乘率、100%命中 | doc02 §4.2/§6.2 | ❌ | `magic.go:224-226` `case 24,28,29,30`:直接 `return CastResult{Target: tgt}`,無傷害,註解自承「加乘率未在 spell.json…待實裝」 | 高 |
| 法術攻擊傷害(最大×(1−魔抗)、隨機化) | doc02 §4.3 | 🟡 | `magic.go:dealDamage` 做了隨機化,但**魔法抗性固定當 0**(`dealDamage` 註解:「魔法抗性欄位尚未進資料管線,先以 0 計」)→ 對高魔抗角色(悠妮 30-50%）傷害被高估 | 中 |
| 恢復法術(最大×0.9~max−1) | doc02 §4.4 | ✅ | `applySpell` target=1 分支,`randomizeAmount` | — |
| 命中率:法術內定命中率 | doc02 §4.3 | ✅(含資料矛盾已誠實記錄) | `magic.go rollsHit`,hit=0 視為必中,檔頭註解說明與 dump 值的取捨依據 | — |
| 輔助法術(魔刃/魔鎧/風行,AP+15%/DP+15%/HIT+15,EV+15) | doc02 §6.4 | ✅ | `magic.go case17/18/19`,`applyBuff` | — |
| 狀態法術(解毒/祛麻/封咒/毒擊/麻痺) | doc02 §6.4 | ✅ | `magic.go case20/21/22/26/27` | — |
| 組合技(破壞神/暗邪鬼) | doc02 §6.4 | ✅ | `magic.go case34/35` | — |
| 傳送術(目的地任選) | doc02 §6.4 | ❌ | `magic.go case23`:註解「battle 套件不處理定位——待實裝」,只回空效果 | 低(用途窄,多為劇情/特定角色) |
| 經驗值公式(攻擊/恢復/各系術) | doc02 §4.5 | 🟡已補(worklist 第 9 輪) | `growth.go` 逐條實作攻擊/恢復/傳送/行動/魔刃魔鎧風行/麻痺毒擊/解毒祛麻七式,`combat.go AttackWithRNG`、`magic.go CastArea/awardCastExp` 已接上,僅 Own/Ally 攻方生效;**封咒術(22)/破壞神(34)/暗邪鬼(35)doc02 §4.5 未列公式,誠實回 0 不編造**;劍技(24/28/29/30)因傷害本身未實作(見上表)連帶無經驗值 | 中(劍技/組合技經驗待劍技傷害公式補上後一併收斂) |
| 升級(每 100 經驗一級、成長亂數) | doc02 §2/§4.6/§7.2 | ✅已補(worklist 第 9 輪) | `growth.go GainExp`/`applyLevelUpGrowth`,門檻 100(doc03 0x43),可連續跨級;`growthTable` 為 doc02 §7.2 顯示值與 EXE 升級成長表(`docs/data/exe_tables/growth.json` 0x55EA1)交叉比對後的精確版(63 列全比對成功,見該檔案頭註解),非估計值。`Unit` 新增 `Exp`/`ExpPerLevel`/`DX` 欄;`ExpPerLevel`(攻擊經驗公式的「守方每級經驗」)來源 EXE 敵/友單位表,由 `export_units.py` 新增 `ex` 欄接上,34 份本機 `map*_units.json` 資產已重新匯出;查無成長資料的單位(如無名雜兵)等級仍照門檻演進但不套用屬性成長,誠實標記非靜默丟棄。**升級是否立即回滿新增 HP**doc 未明講,採較合理的 RPG 慣例並於 `growth.go` 註解誠實標記為假設 | — |
| 敵方 AI:目標評分(dmg、擊殺加成×2) | doc11 | 🟡 | `combat.go aiActUnit/NextAIPlan`:已套地形 AP/DP%、並依原版證據加入 **dmg≤2 略過**；擊殺加成仍是 remake 簡化版(`dmg≥HP→score×2+1000`)，**情境加成(0x1529E)、狀態倍率×1.5(0x152AB)** 尚待 RE | 中 |
| 敵方 AI:**施法決策**(法師/僧侶主動用攻擊術/補血術) | 原先標記的 `0x154D1` 經 direct disasm 證實只是 `0x1548E` 移動函式本體中段；呼叫點 `0x13E39`/`0x14F9B` 都在 AI 落點／移動分支，未證實 spell dispatch | ❌ | `aiActUnit`/`NextAIPlan` 完整讀過,只呼叫 `Attack`,從未呼叫 `CastArea`/`Cast`;敵方法師/僧侶單位在 remake 中恆為純物理攻擊 | 高；需另找真正的 spell dispatch callsite |
| 敵方 AI:**使用道具** | doc11 未記錄此分支(doc11「仍待確認」清單也未提道具);doc02 未給 AI 道具規則 | ❌(RE 亦未記錄機制) | `aiActUnit` 無任何道具邏輯;RE doc11 本身也沒有「AI 用道具」的反組譯條目——這條是 RE 與 remake 雙缺,非「RE 有記 remake 沒做」 | 低(先確認原版是否真有此行為,再排 RE 工作) |
| 移動地形成本(森林/沼澤耗 MV) | doc02 §3.1 | ✅ | `move.go MoveCost` 讀 `map.json` 的 `cost` 陣列(worklist 第8輪「地形屬性接線」) | — |
| **地形攻防加成(+5%/-5%、森林-5%/+10%、沼澤-5%/-5%)** | doc02 §3.2 | 🟡已補(一般/沼澤;森林待補) | 同上「物理攻擊:地形 AP/DP% 修正」條:`terrain.go` 已接一般(+5%/-5%)、沼澤(-5%/-5%)兩類;森林因 `map.json cost[]` 資料管線把森林/正常都存成 cost=1,無法分辨,待該管線補地形代碼欄位 | 中(森林細節待補) |
| 裝備欄 / 裝備加成 AP/DP/HIT/EV | doc02 §5、doc32 §1/§5 | ❌ | `Unit` struct(`model.go`)無裝備欄位;`main.go:384-385` 選單「道具」分支直接 `g.msg = "道具:尚未實裝"`;doc32 §5 已建議暫行做法「base+裝備 ap/dp」但 remake 未採用,AP/DP 全靠 `unitsFile` 匯出值寫死 | 高 |
| 道具使用效果(藥水回血、卷軸) | doc02 §5.13 | ❌ | 同上,選單直接擋掉 | 高 |
| 裝備自帶法術(不耗MP、無經驗) | doc02 §4.6、doc32 | ❌ | 無裝備系統,此規則無從掛載 | 中(依賴裝備系統先做) |
| 轉職系統(Lv20+教會、轉職道具→最高職業) | doc02 §7.1、doc32 §4「[阻] 轉職系統」 | ❌ | grep `轉職\|promot` 全 remake 零命中;doc32 已自承反組譯機制未完成,remake 更是完全沒有轉職 UI/邏輯 | 中(需先有裝備/道具系統及教會場景才有意義) |
| 中毒/麻痺/封咒 回合遞減與到期解除 | doc02 §6.4 | ✅ | `model.go TickStatus`,`main.go:1962` 每回合結尾呼叫 | — |
| 中毒每回合 −10% HP | doc02 §6.4 | ✅ | `TickStatus`:`dmg := u.MaxHP/10` | — |
| Buff(魔刃/魔鎧/風行)到期清除 | doc02 §6.4 | ✅ | `TickStatus` `BuffTurns` 遞減歸零清空 | — |
| 對話嘴型動畫(m0閉/m3開,doc14 0x16d00) | doc14、doc40 | ✅ | `main.go:930-936` `mouthOpen`/`mouthTimer`,`rand.Intn(30)+2` 對齊原版 tick 語意;`portraits` 依肖像 id 存 4 嘴型幀(`loadPortraits`) | — |
| 法術施放演出(命中/傷害畫面) | doc35、doc37 | 🟡 | 攻擊型法術(`sp.Target==0`)重用 `newAtkAnim`(即物理攻擊揮劍動畫),**無獨立法術特效**;治療型法術(`sp.Target==1`)完全**無演出**,只有文字訊息 | 中(已知美術缺口,非戰鬥正確性問題) |
| 商店(一般商品) | doc13 | ✅ | `main.go` `case "shop"`,`ShopGoods()`,購買扣金流程 | — |
| 祕密商店(旗標條件開啟) | doc13、campaign.go `SecretIf` | ✅ | `campaign.go:50` `SecretIf`;`ShopGoods()` 依旗標回傳 `Secret` 清單(commit e09c68c 已完成) | — |
| 商店賣出(原價 75 折) | doc02 §4.6 | ❌ | `main.go`/`campaign.go` grep `賣\|Sell\|0.75` 零命中,只有購買邏輯,無賣出功能 | 低 |
| 存檔/讀檔 | doc19 | ✅(自有格式,非破解 FD2.SAV,已在 save.go 註明是刻意設計) | `save.go` 存 campaign 節點/旗標/金幣/道具 | — |
| BGM 播放 | doc12 | ✅ | `audio.go playBGM`,同曲不重播/換曲釋放語意對齊 `0x26777` | — |
| SFX(命中/陣亡/選單音) | doc36 | ✅(池對照為近似值,doc36 已註記真實 attack_id→sfx 池對照未 RE 完成) | `audio.go loadSFX/playSFX`,`main.go` 多處呼叫 | — |
| 出場人數上限(前27章16人/末3章20人) | doc02 §4.6 | ❌ | grep `16.*人\|20.*人\|MaxDeploy` 零命中;remake 依 `own_deploy` 格數放人,無顯式上限規則 | 低(多數地圖部署格數本身就 < 上限,實務影響小) |

## RE 側也需要補的缺口(非 remake 落差,附帶記錄)

- **`0x154D1` 施法假說已撤回**：direct disasm 顯示它位於 `0x1548E` 移動函式中段，且該函式實際串接 `0x14B78` 路徑／移動與 `0x12D7B` 演出狀態；不能再把它列為 spell 入口。敵方 AI 施法仍是 remake 缺口，但下一步應以 `Cast`/法術效果函式的反向 callsite 找真正 dispatch，而不是繼續猜 `0x154D1`。
- **doc32 §4 三個 `[阻]` 項目(裝備加成精確公式、物品使用效果碼、轉職系統機制)本身就還沒反組譯完**,remake 的裝備/道具/轉職缺口有一部分要等 RE 補完才能對照實作,不能單純算「remake 沒做」。

## 落差統計

- 稽核機制條目共 **33** 項(不含「RE 側缺口」兩項)。
- ❌ 缺:**17** 項
- 🟡 部分:**4** 項(法術魔抗、AI 評分公式、AI 施法情境見上表、法術演出)——註:AI 施法已獨立列為❌(高優先),此處🟡計數為法術魔抗/AI評分公式/法術演出 3 項再加傳送術外的模糊項,實際明確🟡為:法術傷害魔抗、AI目標評分簡化、法術施放演出 = 3 項;敵方AI施法決策因「remake 完全沒有」歸類為❌。
- ✅ 已做:**13** 項

> **worklist 第 8 輪更新(物理攻擊公式補全)**:上表「物理攻擊:地形AP/DP%修正/暴擊/命中率/傷害隨機化」與「地形攻防加成」5 條由 ❌ 轉為 ✅/🟡,「敵方AI:目標評分」補上地形%(仍 🟡,其餘子項未變)。相對本節原始統計:❌ 減 5(暴擊、傷害隨機化轉✅共2項;地形AP/DP%修正、命中率、地形攻防加成轉🟡共3項)→ **12** 項;✅ 加 2 → **15** 項;🟡 加 3 → **7** 項(其中地形AP/DP%修正、命中率、地形攻防加成 3 項為「已補但含近似值」,見對應列的誠實標記:HIT/EV 為固定近似值、森林修正因資料管線限制未接)。

## 最該補的前 5 項(依「影響原版可玩性/戰鬥正確性」排序,worklist 第 8 輪前的排序,第 1、5 項已部分處理見上方更新註記)

1. ~~**物理攻擊補全命中率/暴擊/地形修正/傷害隨機化**~~(worklist 第 8 輪已補:暴擊/傷害隨機化完整實作;命中率/地形AP-DP%修正因來源資料本身缺 HIT/EV 欄與森林/正常地形無法從 `cost[]` 分辨,採誠實標記的近似值,細節見上表對應列與 `combat.go`/`terrain.go`/`export_units.py` 檔頭註解)。
2. ~~**經驗值與升級系統**~~(worklist 第 9 輪已補:`growth.go` 七式經驗公式 + 升級成長,`combat.go`/`magic.go` 已接上,見上表。轉職系統本身仍未實作,`growthTable` 已含轉職後職業的成長列供未來銜接)。
3. **裝備/道具系統**(doc02 §5、doc32)——選單直接寫死「尚未實裝」,武器/防具/消耗品完全不影響戰鬥,198 筆物品資料形同虛設。**HIT/EV 命中閃避的真值也依賴此系統**(見上表命中率列)。
4. **敵方 AI 施法**——使用者已實測確認的落差,法師/僧侶敵人變成不會用法術的近戰兵,大幅改變原版戰鬥難度曲線與戰術；原先懷疑的 `0x154D1` 已證實只是移動函式中段，落地前需從 `Cast`/法術效果函式反向追真正 dispatch。
5. ~~**地形攻防加成**~~(worklist 第 8 輪已補一般/沼澤兩類;森林 AP-5%/DP+10% 待 `export_engine_assets.py` 匯出管線補獨立地形代碼欄位後才能收斂,見上表)。

轉職系統、劍技傷害、商店賣出、傳送術等排在此 5 項之後,原因是:轉職依賴裝備系統先到位才有意義;劍技/傳送使用頻率相對低;商店賣出是純數值缺口、不影響戰鬥正確性。
