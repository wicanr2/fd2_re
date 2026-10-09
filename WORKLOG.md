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
