# 102 — 戰鬥結果字串沒有原版出處（2026-09-09）

第 6 項的後半：「不應該在戰鬥後顯示傷害多少」。

## 兩項證據

**一、字串出自重製端自己的原始碼。** `remake/assets/locales/zh-Hant/pack.json`
裡五筆 `battle.attack.*` 的 `source_string_id` 全部指向重製端的 Go 檔：

| 條目 | source_string_id |
|---|---|
| `battle.attack.miss` | `legacy.go.remake.cmd.fd2.main.l5973-c22` |
| `battle.attack.hit` | `legacy.go.remake.cmd.fd2.main.l5975-c25` |
| `battle.attack.critical_suffix` | `legacy.go.remake.cmd.fd2.main.l5977-c14` |
| `battle.attack.exp_suffix` | `legacy.go.remake.cmd.fd2.main.l5980-c26` |
| `battle.attack.choose_target` | `legacy.go.remake.cmd.fd2.main.l5193-c13` |

**二、原版 FDTXT 沒有這種模板。** 掃過分離出的全部 FDTXT 資源，「傷害」
「造成」「點傷」只出現五次，全部在劇情對白句子裡（FDTXT_010／018／025／027／
030），沒有任何「造成 N 傷害」型的戰鬥結果模板。

原版的傷害數字出現在全螢幕戰鬥演出**之內**（例如 command 24 在第 10 幀發布
單一完整傷害），不是戰鬥結束後留在地圖底部的一行字。

## 處置

字串本身保留——它同時是語言包完整性的前置檢查，缺條目要在改動 HP 之前就
失敗即關閉。改的是顯示：`publishPhysicalAttackMessage` 只在 F3 診斷模式寫進
`g.msg`，一般玩家路徑不顯示。玩家、AI 一般路徑與 native AI mode 11 三個發布
點都改走它。

回歸 `TestPhysicalAttackMessageStaysOutOfThePlayerPath`。

## 尚未涵蓋

- 第 6 項的前半（全螢幕戰鬥演出退化）沒有取到原版收據。本輪用
  `CONTINUE→END→YES` 推進敵方回合三次，敵方只有移動沒有交戰；要拍到攻擊
  演出得先讓雙方進入接戰距離。
- 原版演出內部的傷害數字版面與時序。
