# 工作歷程

目前分層狀態以 [58](docs/knowledge-base/58-fd2-exe-re-coverage.md)為準，完整匹配範圍依 [118](docs/goal/118-goal-fd2-matching-decompilation-20261008.md)。既有時間序列保留於[工作清單歷史](docs/knowledge-base/91-worklist-history.md)與該頁引用的研究紀錄。

## 2026-10-09 原始 ESI 輸入的完整 C 匹配

- 新增 [C 來源](tools/fd2_matching_game_class_stack.c)與[回歸](tools/test_fd2_matching_input_esi.py)，登錄於既有編譯驅動器的 `game_class_stack`。`sub_12DAC` 完整 140 bytes 匹配。輸入 ESI 在原始指令先讀後寫，保留其未知初值，不補零。
- 14 個表示模型各編譯 3s／4s／5s 與預設／space／speed，共 126 份候選，獨立重建兩輪。所有 OMF、完整報告及產碼相同，126 份另經 objcopy 核對。五份匹配只增加一個原始函式；31385／2DF6B 的模型仍未匹配。
- 全檔組合保留舊 75 份來源報告的順序，新報告最後加入。舊 185 段 C 出處逐項相同；SDK 19 函式的出處保持，SDK 原生連結與全檔收據另重建兩輪。完整 357074-byte EXE 的 SHA-256 與固定輸入相同。其餘原版 code 尚未還原，#198 與 Goal 保持開啟。
- 回歸核對完整原始函式、ESI 先讀後寫、EBX／ESI 保存與 16-bit 有號讀取。缺來源、來源內容與來源雜湊錯誤均於輸出前拒收。既有符號／LE 反例回歸首次誤用未含 190AC 的新物件，改以原有 `treasure-fixed-owner-r1` 乾淨重跑通過，未改守衛。
- 原作參數宣告、輸入 ESI 的來源與實際 GS 修改仍未知。`parm`／`modify exact` 僅為相容產碼表示。已知位址由既有 IDA 9.4 匯出機械核對，不重開既有隊伍／面板語意。正式 Go／Ebiten 與章驗收等級保持。

提交前核對 102 份來源雜湊、1205 條本地連結、191 條教訓與 76 個守衛。真正主機 Issue 清單經官方 pull／render／verify，16 條仍開啟，8 條需人工判定，可能已完成 0 條。原始 IDA 資料庫雜湊保持，變更檔與新產物 UID／GID 為 1000；歷史 root-owned 共 2811 項保持，沒有 `.md` 目錄。本批一次性 FD2 容器均已退出。提交與真正遠端 HEAD 結果回填 [#198](https://github.com/wicanr2/fd2_re/issues/198)。

原始位址、雜湊、工具版本、caller、精確旗標、完整候選與驗證結果保存在[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_class_stack_matches`、`class_stack_original_evidence`、`validation.class_stack` 與 `bootstrap`。公開提交保存自行重寫的 C、工具及整理證據；完整 EXE、SDK 物件與工具二進位留在本機 `work/matching-full-20261008/`。

編譯與連結沿用[驅動器](tools/fd2_matching_game_restore.py)：在既有 Docker 工具鏈內執行 `compile --source-key game_class_stack --output /work/class-stack-final-r1`，再執行 `link --objects /work/class-stack-final-r1 --evidence /work/ida-full-image.json --output /work/class-stack-final-linked-r1`。容器與唯讀編譯器掛載方式沿用主收據的工具鏈紀錄。第二輪使用新的 `r2` 輸出，不覆寫第一輪。

## 2026-10-10 五個 DOS／檔案流程的完整 C 匹配

- 新增 [C 來源](tools/fd2_matching_game_dos_file.c)與[完整函式回歸](tools/test_fd2_matching_dos_file.py)，登錄於既有驅動器的 `game_dos_file`。361CC／36284／36344／36900／36955 完整 137／109／311／85／173 bytes 匹配，共 815 bytes。
- 36284 保留兩個分開的無號條件。36344 保留 204-byte 局部布局、早期錯誤返回及下一次讀取位置。其餘三項以相容 callee 宣告保留原始 EBX 跨呼叫存活，沒有 inline code 或位元組補丁。原始 unknown 分類、作者宣告與完整間接副作用集合仍分開記錄。
- 14 個模型共 126 份候選，獨立編譯、連結兩輪，OMF、完整報告與產碼相同；126 份全部由 objcopy 獨立核對。17 份正例只計五個原始函式；其餘模型保持負例。
- 舊 76 份來源報告順序保持，新報告最後加入。舊 186 段 C 出處逐項相同；SDK 19 函式／1200 bytes 出處保持，在新 C 基準實際重跑原生連結兩輪，全檔與收據相同。完整 EXE 的大小及 SHA-256 與固定輸入相同，其餘原版 code 尚未還原，#198 與 Goal 保持開啟。
- 回歸核對完整 815 bytes、原始無號分支、堆疊布局與指標跨 close 存活。十一項原始名稱／CDECL 修飾別名錯址及一項原始 LE 矛盾均拒收，CLI 矛盾在輸出前停止。既有符號、明示位移與 LE 守衛通過。
- 第一輪連結曾缺少 CDECL 前置底線別名。直接檢查 COFF relocation 後補上受原始 caller／LE 核對的別名，未放寬守衛。將錯誤返回改回原始布局後，完整函式才收件。已知位址沿既有 IDA 9.4 匯出機械核對，沒有重開玩家語意或提升正式 Go／Ebiten 與 PLAYER-E2。

精確正例旗標、原始函式與 caller、來源雜湊及驗證結果見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_dos_file_matches`、`dos_file_original_evidence`、`validation.dos_file` 與 `bootstrap`。編譯與連結使用[既有驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_dos_file --output /work/dos-file-final-r1` 與 `link --objects /work/dos-file-final-r1 --evidence /work/ida-full-image.json --output /work/dos-file-final-linked-r1`，第二輪使用新的 `r2` 輸出。公開保存重寫 C、工具與整理證據；完整原版／SDK 二進位留本機。

提交前核對 104 份來源雜湊、1212 條本地連結、192 條教訓與 77 個守衛。真正主機 Issue 清單經官方 pull／render／verify，16 條仍開啟，8 條需人工判定，可能已完成 0 條。正式 IDA 資料庫雜湊保持，變更檔與新產物 UID／GID 為 1000；歷史 root-owned 共 2811 項保持，沒有 `.md` 目錄。本批 FD2 容器已全部退出。提交、推送及真正遠端 HEAD 核對結果回填既有 #198。
