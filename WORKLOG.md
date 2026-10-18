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

## 2026-10-10 狀態群組與原始載入位元組守衛

- 新增 [C 候選](tools/fd2_matching_game_state_group.c)及[完整群組／載入位元組回歸](tools/test_fd2_matching_state_group.py)。Watcom 10.0a 與 9.5 各 261 份候選均未完整匹配，獨立編譯與連結兩輪，OMF、完整報告及產碼相同；522 份全部由 objcopy 核對。原有 C 覆蓋、191 段出處及 19 段 SDK 出處保持。
- 22AF6 的原始欄位比較使用 EAX，基準候選使用 ESI，差異限於三個編碼位元組。固定 C 的 3s 預設四函式群組中，9.5 保留原始條件跳躍及 RET；10.0a 額外共用 22AA7 的 RET。最後 27 bytes 局部相同仍不收件，不推定原作精確版本、作者宣告或新玩法。
- 隔離反例只改匯出的三條 `loaded_bytes`，舊比較器會誤報 22AF6 匹配。原版與正式匯出保持不變，偽造輸出不計覆蓋。新增守衛從原始 `file_bytes` 及真正 LE fixup 重算每條載入指令；正式 75,427 條指令與 7,003 個 CODE 重定位位置全部相同，三項改寫與 CLI 偽造在輸出前拒收。
- 舊有 815-byte 五函式正例重新連結及回歸通過，報告除新驅動器雜湊外完全相同。既有符號、LE 與明示位移守衛通過。[56](docs/knowledge-base/56-fd2-remake-sdd.md)、[57](docs/knowledge-base/57-ui-evidence-matrix.md)與[58](docs/knowledge-base/58-fd2-exe-re-coverage.md)同步說明修正範圍，正式 Go／Ebiten 與 PLAYER-E2 保持。
- 261 份 9.5 候選兩輪曾在固定 90 秒內只完成 217／215 份物件，屬批次時限不足。驅動器依 `max(90, min(600, 30 + trial_count))` 設定有界時限，寫入 `runner_timeout_seconds`，同 C 與同候選的兩輪 291 秒批次均完成。失敗目錄保留，不算 C 不匹配。

來源、精確旗標、原始函式、修正前後反例與驗證結果見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_state_group_attempts`、`state_group_original_evidence`、`validation.state_group`、`validation.loaded_image_guard` 及 `validation.compiler_batch_bounds`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_state_group --compiler-version 10.0a` 或 `9.5`，各用新的輸出目錄，再以固定原檔及 IDA 匯出執行 `link`。全檔 C 反編譯尚未完成，#198 與 Goal 保持開啟。

提交前核對 106 份來源雜湊、1229 條本地連結、194 條教訓與 79 個守衛。真正主機清單經官方 pull／render／verify，16 條仍開啟，8 條需人工判定，可能已完成 0 條。正式 IDA 匯出與資料庫雜湊保持，變更檔與新產物 UID／GID 為 1000；歷史 root-owned 共 2811 項保持，沒有 `.md` 目錄。本批 FD2 容器全部退出。提交、推送及真正遠端 HEAD 核對結果回填既有 #198。

## 2026-10-10 開場與戰後較大函式完整 C 匹配

- 新增 [C 來源](tools/fd2_matching_game_long_sequences.c)與[完整序列回歸](tools/test_fd2_matching_long_sequences.py)。3231B 完整 1626 bytes 及 2548C 完整 715 bytes 匹配，共新增兩個原始函式2341 bytes。既有開場／戰後語意沿用原證據，原始分類、作者宣告、正式 Go／Ebiten 與 PLAYER-E2 保持。
- 六個模型各比較三種 CPU 與三種成本策略，共54份候選，獨立編譯、連結兩輪，OMF、完整報告及產碼相同，全部由 objcopy 核對。25份正例只計兩個原始函式；32-bit開場計數控制未匹配。
- 開場保留 BL 的8-bit計數、15／13次迴圈、所有呼叫與32／31／0全域寫入順序。戰後保留35BBA後讀取指標、原始欄位寫入與兩個淡出／淡入迴圈。初輪指標模型曾提前讀取，已依原始順序修正並新增回歸；缺少32975的外部綁定已按直接call補齊。宣告及volatile只作相容產碼表示，延遲呼叫參數不證明硬體wall-clock。
- 舊77份來源報告順序保持，新報告最後加入。舊191段C出處及SDK19函式／1200 bytes出處逐項相同；新C基準實際原生連結兩輪，全檔與收據相同，完整EXE大小及SHA-256與固定輸入相同。其餘原版code未還原，#198與Goal保持開啟。
- 176列原始具名參照、兩個完整IDA函式及caller保存；五項新符號錯址、既有LE／位移與75427筆載入記錄／偽造拒收回歸通過。較早六份9.01單輪候選均未匹配，另經objcopy核對，不納雙輪小計。
- 發布前核對發現IDA的caller清單來自間接派送關係，沒有E8直接呼叫。改以原始LE表項及四個scale4間接call位置保存consumer；初版文件產生器的直接caller假設在寫入主收據前已拒收，沒有提升錯誤證據。

精確旗標、原始函式、來源與驗證結果見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_long_sequences_matches`、`long_sequences_original_evidence`、`validation.long_sequences` 與 `bootstrap`。單輪9.01比較在 `early_901_comparisons`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_long_sequences`，各輪用新的輸出目錄，再以固定原檔及IDA匯出執行 `link`。公開保存C、工具及整理證據，完整原版／SDK二進位留本機。

提交前核對108份來源雜湊、1229條本地連結、195條教訓與80個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出及資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 演出／介面C與堆疊實參核對

- 新增[C候選](tools/fd2_matching_game_animation_panel.c)、[實參與保存回歸](tools/test_fd2_matching_animation_panel.py)及[IDA導覽匯出器](tools/ida_export_fd2_local_navigation.py)。28784完整744 bytes與2EBE0完整943 bytes的13個模型，由10.0a／9.5各編譯117份，兩輪OMF、完整報告與產碼相同，234份全由objcopy核對，均未完整匹配。C覆蓋、193段C出處及19段SDK出處保持。
- 原始288E5的`[esp+24h]`與288E9的`[esp+28h]`在不同PUSH之後，都讀caller局部基準+18h。初版C誤傳saved指標的位置已修正為兩個figure來源，原始運算元與有效偏移分列，不用運算元表面差異推定不同來源。
- 演出原始EDI在迴圈初次判斷前沒有局部初始化；來源保留未知，不補成零值。29164兩個已核對返回尾端均還原EDI。三個graphics callee也實際保存EBX；候選pragma允許修改EBX只作保守compiler契約，不能說成實際clobber或原作者宣告。

零值初始化只作`ANIM_LOCAL_2`的未匹配對照，不當作原始初值。
- 原始36／56-byte局部配置、16-bit容量讀取、三個真正E8 caller窗口、107列原始參照與十項錯址拒收回歸保存。既有LE／位移及75427筆載入記錄／偽造拒收回歸通過。兩份WPP前端單輪對照也未匹配，明示前端及固定包裝，另列且不納C前端雙輪小計。
- IDA9.4沿既有py312映像，以正式i64唯讀掛載後的一次性複本輸出導覽。受版控工具重生原檔／DB SHA一致、1305函式、schema及UID核對的JSON；Hex-Rays型別、變數名與推測stack位置只作導覽，不當原作者或位元組證據。技能入口的符號連結指向工具專案，已沿真正權威路徑讀取，沒有另建映像。
- 空的`parm []`後不能再列EDI的編譯語法失敗已修正為EDI輸入在先、堆疊參數在後，機器輸入位置保持。未完成編譯與其後缺收據的診斷不計負例。較早2A2E8的旗標依賴尚未建立C候選，沒有猜補或改列原作組語。

完整來源與結果見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_animation_panel_attempts`、`animation_panel_original_evidence`、`validation.animation_panel` 與 `animation_panel_cpp_supplement`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_animation_panel --compiler-version 10.0a` 或 `9.5`，各輪用新的輸出目錄，再以固定原檔及IDA匯出執行 `link`。公開保存C、工具與整理證據，完整原版、i64／導覽及SDK物件留本機；#198與Goal保持開啟。

提交前核對111份來源雜湊、1238條本地連結、196條教訓與81個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 戰後對話與資源切換完整 C 匹配

- 新增[C來源](tools/fd2_matching_game_post_dialogue.c)及[備份／副作用與派送回歸](tools/test_fd2_matching_post_dialogue.py)，24754完整960 bytes匹配。三個C表示都重現三個17-byte局部備份、分支與章節INC在context推入之後／CALL之前的原始順序，不能由相同產碼推定原作者副作用寫法。
- 27份候選獨立編譯、連結兩輪，OMF、完整報告與產碼相同，全部由objcopy核對。3s預設／speed的六份正例只計一個原始函式；其餘CPU／成本策略按完整區間拒收。
- 舊78份來源報告順序保持，新報告最後加入。舊193段C出處及SDK19函式／1200 bytes出處保持；新C基準實際重跑SDK原生連結兩輪，全檔與收據相同，357074-byte EXE的SHA-256與固定輸入相同。原版其餘code尚未還原，#198與Goal保持開啟。
- 72列原始具名參照、完整IDA函式與LE表項22／25E23的scale4間接consumer保存。原始60-byte局部配置、4×MOVSD＋MOVSB三次複製、fade步進2／上限64、六項錯址、既有LE／位移與75427筆載入記錄／偽造拒收回歸通過。既有戰後語意、原始分類、正式Go／Ebiten、PLAYER-E2及硬體時序保持。
- 另查237D5為含多個入口與區間外共用尾端的owner，沿既有覆蓋與完整區間政策保留限制，沒有將入口片段裁切成完整C匹配。

精確旗標、來源與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_post_dialogue_matches`、`post_dialogue_original_evidence`、`validation.post_dialogue` 與 `bootstrap`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_post_dialogue`，各輪用新的輸出目錄，再以固定原檔及IDA匯出執行 `link`。公開保存C、工具與整理證據，原版、i64及SDK物件留本機。

提交前核對113份來源雜湊、1245條本地連結、196條教訓與81個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 六模式分派完整 C 匹配

- 新增[C來源](tools/fd2_matching_game_overlay_dispatch.c)及[條件邊／記錄寫入回歸](tools/test_fd2_matching_overlay_dispatch.py)，122DC完整1051 bytes匹配。if鏈及逐分支返回兩種表示都在3s預設成本策略下匹配；原始分類、型別與作者宣告未知，既有語意保持。
- 18份候選獨立編譯、連結兩輪，OMF、完整報告及產碼相同，全部由objcopy核對。初次switch探針在CODE前置24-byte跳表，函式入口不在起始，工具拒收；該未完成探針不列雙輪總數，也不增加覆蓋。
- 核對六個原始模式條件邊、37個明示helper CALL、兩個返回位置及第五模式共用CALL目標。模式6保留原始四位元組記錄的+7寫入，不把C型別或offset推成新語意。125列原始參照、完整IDA函式及11CF0／18BF7兩個E8 caller窗口保存；六項錯址、既有LE／位移與75427筆載入記錄／偽造拒收回歸通過。
- 舊79份來源報告順序保持，新報告最後加入。舊194段C出處與SDK19函式／1200 bytes出處保持；新C基準實際重跑SDK原生連結兩輪，全檔與收據相同。357074-byte EXE的SHA-256與固定輸入相同，其餘原版code尚未還原，#198與Goal保持開啟。
- 篩選時另外核對32D18、244B6、234BB等完整owner；多入口、額外chunk與區間外共用尾端不裁切計數。這些已知布局限制未改動既有分類，也未重開戰役語意或正式Go／Ebiten。

精確旗標、來源與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_overlay_dispatch_matches`、`overlay_dispatch_original_evidence`、`validation.overlay_dispatch` 與 `bootstrap`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_overlay_dispatch`，各輪用新的輸出目錄，再以固定原檔及IDA匯出執行 `link`。公開保存C、工具與整理證據，原版、i64及SDK物件留本機。

提交前核對115份來源雜湊、1252條本地連結、196條教訓與81個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 音效過場、事件尾呼叫與對話備份候選

- 新增[過場C](tools/fd2_matching_game_sound_transition.c)、[事件C](tools/fd2_matching_game_tail_calls.c)與[完整回歸](tools/test_fd2_matching_tail_and_backups.py)。3396A／34C76／35487完整324／61／86 bytes匹配，三個E9均到完整callee入口，由C自然產生。15份正例只計三個原始函式，共471 bytes。
- 另保存[對話備份C](tools/fd2_matching_game_dialogue_backups.c)的九種表示。165AC完整688 bytes的81份候選均未匹配；參數槽、求值順序與暫存器配置仍有差異，聚合及volatile表示不證明作者宣告。後續先核對callee保存契約，再決定相容宣告，不裁切相同片段或提高原始分類。
- 三組108份候選完整編譯、連結兩輪，OMF、完整報告及產碼相同，全由objcopy核對。早期探索來源只留本機，正式矩陣由目前受版控來源乾淨重跑，未完成探針不列入小計。
- 核對三筆LE表項：3396A為51D71表項24，由25E3A消費；34C76／35487為51B91表項35／56，由19511消費。表項檔案偏移與IDA線性位址分列。過場只載入#88 selector1，保留一般玩家指令#80的既有證據；三次delay實參600與20／20／20／60等待值不推成硬體wall-clock。
- 原始42列對話參照、兩個E8 caller窗口、兩次有號除法與sum零值旁路、五次26668-byte配置、五次備份／繪製及返回53A18保持。七項備份錯址、三個錯誤尾目標、既有LE／位移及75427筆載入記錄／偽造拒收回歸通過。
- 舊80份來源報告順序保持，兩份正例報告最後加入；195段舊C與SDK19函式／1200 bytes出處保持。新C基準實際重跑SDK原生連結兩輪，全檔與收據相同。357074-byte EXE的SHA-256仍等於固定輸入；其餘原版code尚未還原，#198與Goal保持開啟。

來源、旗標與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_sound_transition_matches`、`game_tail_event_matches`、`game_dialogue_backup_attempts`、`tail_and_backups_original_evidence`、`validation.tail_and_backups` 與 `bootstrap`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_sound_transition`、`game_tail_events` 或 `game_dialogue_backups`，各輪用新的輸出目錄，再以固定原檔及IDA匯出執行 `link`。公開保存C、工具與整理證據，原版、i64及SDK物件留本機。

提交前核對119份來源雜湊、1265條本地連結、196條教訓與81個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 START／LOAD完整主流程與對話保存契約

- 新增[主流程C](tools/fd2_matching_game_start_load.c)及[欄位／重試回歸](tools/test_fd2_matching_start_load.py)，25EBB完整663 bytes匹配。槽位址的LEA排列與零實參表示共同重現原始產碼；不能由相同產碼推定作者C表示或型別。
- 22987-byte緩衝、12587-byte header、2600-byte槽、2560-byte記錄複製與七個原始欄位讀取保持。+2欄位為未對齊4-byte載入，其餘指定欄位為byte；255轉selected零值後回26016，原始51D71間接表與25DBD直接caller保留。
- [對話C](tools/fd2_matching_game_dialogue_backups.c)新增五個宣告候選，共45份完整負例，原81份歷史負例保持。原始4E031只改AX並還原ESI；4E96F以PUSHA／POPA保存通用暫存器；15E71保存EBX，15E9E經22BBE共享收尾還原EBX。保守pragma允許修改與實際callee保存分列，不宣稱原作者ABI。
- 兩組90份候選完整編譯、連結兩輪，OMF、完整報告及產碼相同，全由objcopy核對。七項錯址、既有LE／常數位移及75427筆載入記錄／偽造拒收回歸通過。
- 首輪pack(push,1)／pack(pop)被10.0a以E1054／E1009拒收，未產生完整編譯收據；不計成產碼負例。改用pack(1)／pack()後乾淨重跑，sizeof斷言確認10／2600 bytes。教訓與守衛寫入既有[台帳](docs/data/fd2-lessons.json)。
- 舊82份來源報告順序保持，新正例最後加入。198段舊C與SDK19函式／1200 bytes出處保持；新C基準實際重跑SDK原生連結兩輪，全檔與收據相同。357074-byte EXE的SHA-256保持；其餘原版code未還原，#198與Goal保持開啟。

來源、旗標與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_start_load_matches`、`game_dialogue_backup_contract_attempts`、`start_load_original_evidence`、`validation.start_load` 與 `bootstrap`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_start_load`；對話補充用 `game_dialogue_backups --cases DIALOGUE_BACKUP_9 DIALOGUE_BACKUP_10 DIALOGUE_BACKUP_11 DIALOGUE_BACKUP_12 DIALOGUE_BACKUP_13`。各輪用新的輸出目錄，再以固定原檔及IDA匯出執行 `link`。公開保存C、工具與整理證據，原版、i64及SDK物件留本機。

提交前核對121份來源雜湊、1274條本地連結、197條教訓與82個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 終局對話完整C與1718-byte owner候選

- 新增[對話C](tools/fd2_matching_game_terminal_dialogue.c)、[owner C](tools/fd2_matching_game_terminal_body.c)與[完整回歸](tools/test_fd2_matching_terminal.py)。2C39B完整106 bytes匹配，保留A9514h目的位址、原始呼叫順序、EBX保存及2BCE5內12個直接caller。兩種保守宣告的四份正例只計一個函式。
- 2BCE5完整1718 bytes仍未匹配。八種表示保留三份20-byte表、84-byte局部布局、40／200／20迴圈、有號除法、原始欄位寫入及兩個外層caller。局部相同不增加覆蓋；原有終局語意、分類與一般玩家驗收保持。
- 兩組99份候選獨立編譯、連結兩輪，OMF、完整報告及產碼相同，全由objcopy核對。72份owner候選均負例，預設對話宣告103 bytes也按完整區間拒收。
- 4E031實際只改AX，1956B及4E63D保存EBX的原始指令另存；保守pragma允許修改不當作實際clobber或作者ABI。九項錯址、既有LE／常數位移與75427筆載入記錄／偽造拒收回歸通過。
- 初次連結因sub_20421未登錄被ld拒收，已依原始E8目標補齊，不計成產碼負例。回歸初稿假設預設對照104 bytes，實際物件為103 bytes；已改讀實際收據，完整位元組負例檢查保持。兩者屬工具／驗證問題，不列為產品缺陷。
- 舊83份來源報告順序保持，新正例最後加入。199段舊C與SDK19函式／1200 bytes出處保持；新C基準實際重跑SDK原生連結兩輪，全檔與收據相同。357074-byte EXE的SHA-256保持，其餘原版code未還原，#198與Goal保持開啟。

來源、旗標與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_terminal_dialogue_matches`、`game_terminal_body_attempts`、`terminal_original_evidence`、`validation.terminal` 與 `bootstrap`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_terminal_dialogue` 或 `game_terminal_body`，各輪用新的輸出目錄，再以固定原檔及IDA匯出執行 `link`。公開保存完整C候選、工具與整理證據，原版、i64及SDK物件留本機。

提交前核對124份來源雜湊、1283條本地連結、197條教訓與82個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 原廠SDK完整群組與額外來源

- 先核對剩餘分布及固定SDK證據，未將未知函式按名稱或相似片段升格。新增[來源工具](tools/fd2_matching_library_extension.py)與[拒收回歸](tools/test_fd2_matching_library_extension.py)，只補既有保留函式庫的來源。
- 原生重建ioalloc／setvbuf／unlink完整119／120／36 bytes，import由既有runtime定位及原始參照逐筆核對。另重建amodf整個302-byte CODE，兩個公開入口0／144及三個完整IDA函式144／75／83 bytes共同驗證。沒有裁切、遮罩、補opcode或改原始名稱。
- 新增六個來源、577 bytes；SDK來源由19函式／1200 bytes增至25函式／1777 bytes。C199函式／29424 bytes、另146-byte未歸屬區塊、175個保留函式庫及931個待還原／分類函式保持。這些來源原本已列為函式庫，不增加C覆蓋。
- 兩輪實際WLINK／WDIS重建的全部CODE、完整報告、收據及357074-byte EXE相同。舊19個SDK來源、200段C出處與完整函式台帳逐筆保持；原始固定EXE SHA-256相同，Goal與#198保持開啟。
- 十項未知／C目標、C出處重疊、裁切邊界、移動或歧義公開入口、偽造載入位元組、跨chunk與物件來源變動均在輸出前拒收。原作精確版本與高階函式語意仍未知。
- 首輪錯把RAW的OFFSET對齊基準當作容器起點，長度守衛拒收。原生map證實唯一CODE從4D7BC起，RAW本身就是完整302 bytes；已依既有工具鏈契約修正並從新目錄重跑，整段比較未放寬。

契約先登記於[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `sdk_extension_spec`，驗證後為CONFORMED；來源與結果在 `sdk_library_extension_origin`、`validation.sdk_library_extension` 與 `bootstrap`。後續C基準先沿用既有library_restore／library_native_restore形成19函式SDK基準，再執行來源工具。CLI的 `--baseline` 與 `--receipt` 指向該中間基準，`--objects`、`--library`、`--evidence`、`--original` 指向固定本機輸入，`--output` 每輪用新目錄。原廠物件、完整CODE／map與EXE留本機；公開工具、雜湊、原始定位與有限caller證據。

提交前核對126份來源雜湊、1290條本地連結、197條教訓與82個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 AI物品完整清單與執行流程C匹配

- 新增[C來源](tools/fd2_matching_game_ai_item_execute.c)及[清單／byte／夾限回歸](tools/test_fd2_matching_ai_item_execute.py)，15055完整700 bytes匹配。兩種局部整數重用表示同碼，不推定作者的volatile、pragma或局部變數宣告。
- 保留36-byte局部、byte32讀寫、14818／149F8的完整target list與count，以及交給20C6F的原始順序。小command分支在152E5共同接續；兩個14EF0 caller、八位元減16、signed X／Y夾限、64起始漸暗與1..8閃爍保持原始指令證據，沒有新增效果或E2結論。
- 4E56C實際只改EAX／EDX，4DBFC實際保存EBX／ESI／EDI；保守允許修改的C宣告分開標示。物品欄位與record活躍區間、X暫存值及Y局部重用共同重現原始配置。相同700-byte長度的負例仍按完整區間拒收。
- 90份候選完整編譯、連結兩輪，OMF、完整報告與產碼相同，全由objcopy核對；兩份正例只計一個原始函式。八項錯址、既有LE／常數位移及75427筆載入記錄／偽造拒收回歸通過。
- 舊84份來源報告順序保持，新正例最後加入。200段舊C出處逐筆保持。SDK完整11／19／25來源鏈在新C基準實際重跑兩輪，25函式／1777 bytes出處與原始來源報告保持；十項SDK未知／C／裁切等拒收在新基準通過。擴充輸出的六個來源是相對19函式中間層，與上一輪25來源比較沒有新增SDK覆蓋。
- 357074-byte全檔與收據相同，SHA-256等於固定輸入。累計200個原始函式以C匹配，其餘930個函式待還原或分類；Goal與#198保持開啟，正式Go／Ebiten與既有AI效果驗收不提升。

來源、旗標與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_ai_item_execute_matches`、`ai_item_execute_original_evidence`、`validation.ai_item_execute` 與 `bootstrap`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_ai_item_execute`，各輪用新的輸出目錄，再以固定原檔及IDA匯出執行 `link`。SDK沿既有三層鏈接續重建；公開C、工具與整理證據，原版、i64及SDK物件留本機。

提交前核對128份來源雜湊、1296條本地連結、197條教訓與82個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 AI物品評分候選與REGS完整C匹配

- 新增[評分C](tools/fd2_matching_game_ai_item_select.c)、[REGS C](tools/fd2_matching_game_int31_regs.c)與[完整回歸](tools/test_fd2_matching_ai_item_select_regs.py)。36255完整47 bytes匹配，保留56-byte REGS、第三參數32-bit讀取後16-bit遮罩、兩段相距28 bytes與int386原始呼叫。三種C表示的18份正例只計一個函式；四個owner內八個直接caller另存，原分類unknown保持，不推定高階INT31h服務或原作者宣告。
- 1567E先核對既有game_ai_sequences來源與歷史負例，再依已知callee保存契約建立18種表示。162份完整候選均未匹配。部分候選與原版同為514 bytes，只差actor乘80的七個bytes，仍按完整區間拒收；C++四份單輪完整負例另列，不併入C雙輪統計。
- 72-byte局部、400-byte配置先於count、零count略過free、位置表步幅2、14818／149F8完整清單、15880的完整count與strict-greater寫入53C33／37／3B／3F均核對原始指令。4E56C只改EAX／EDX，4DBFC保存EBX／ESI／EDI；保守pragma與volatile只作相容產碼表示。
- 兩組189份候選獨立編譯、連結兩輪，OMF、完整報告與產碼相同，全由objcopy核對。六項錯址、既有LE／常數位移與75427筆載入記錄／偽造拒收回歸通過。舊85份來源報告順序保持，新正例最後加入，201段舊C出處逐筆保持。
- SDK完整11／19／25來源鏈在新C基準重跑兩輪，25函式／1777 bytes出處與原始來源報告保持；十項未知／C／裁切等拒收通過。首次掛載少一層clib3s.dos而找不到SDK，修正唯讀路徑後用相同命令乾淨重跑。C++初輪malloc隱式轉型E166無完整收據，明確cast後重跑；這些失敗不計產碼負例。
- 357074-byte全檔與兩輪收據相同，SHA-256等於固定輸入。累計201個原始函式以C匹配，其餘929個函式待還原或分類；其餘原版code未還原，Goal與#198保持開啟。正式Go／Ebiten與既有AI效果驗收保持。

來源、旗標與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_int31_regs_matches`、`game_ai_item_select_attempts`、`ai_item_select_regs_original_evidence`、`ai_item_select_cpp_supplement`、`validation.ai_item_select_regs` 與 `bootstrap`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_int31_regs` 或 `game_ai_item_select`，各輪使用新輸出目錄，再以固定原檔及IDA匯出執行 `link`。SDK沿既有三層鏈重建；公開完整C、工具與整理證據，原版、i64及SDK物件留本機。

提交前核對131份來源雜湊、1305條本地連結、197條教訓與82個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 物品效果與命中完整C候選

- 新增[效果C](tools/fd2_matching_game_item_effect.c)、[命中C](tools/fd2_matching_game_command_hit.c)與[完整回歸](tools/test_fd2_matching_item_effect_hit.py)。20C6F完整1043-byte來源接續已匹配的15055，也保留1BBDC內的玩家caller。1C75E完整193-byte來源保留七個E8 caller；IDA匯出的code xref總數8另列，不把兩種口徑混用。既有物品與職業規則不重解。
- 效果來源保留108-byte局部，將偏移14的16位元欄位擴成32位元存於局部偏移104，將record+60的位元組暫存於局部偏移100再還原；完整目標清單與count、各分支消耗邊界及共用收尾保持。命中來源保留116-byte局部、112-byte表複製、record+32原生職業減一索引、有號16位元讀取、有號除10與餘100、原始命中閘門及1C81F呼叫。
- 12種效果表示與10種命中表示在Watcom 10.0a／9.5、三種CPU與三種成本策略，共396份候選完整編譯、連結兩輪。全部OMF、報告與完整產碼相同，全由objcopy核對，全部仍為負例。型別複製已重現原始REP MOVSD，但同長度193-byte候選仍有六個bytes的暫存器差異，沒有補opcode、遮罩、裁切或計入覆蓋。
- 13項錯址、既有LE／常數位移與75427筆載入記錄／偽造拒收回歸通過。1C9DD的兩條退出邊仍經1C916內的1C9C7／1C9D5接22BBE共享收尾；不把172-byte區間單獨當作可返回的完整C函式。保守pragma、volatile、聚合與指標表示只作產碼候選，不推定作者宣告或實際clobber。
- 既有201個原始函式／30171 bytes、另146-byte未歸屬區塊、202段C出處與SDK25函式／1777 bytes逐項保持。沒有新增匹配，沿用既有全檔基準及SDK來源鏈，不把負例放入EXE，不重跑未受影響的SDK重建。固定全檔SHA-256保持，929個函式仍待還原或分類；Goal與#198保持開啟。
- 後續先核對「分開局部變數＋保守的迴圈被呼叫端宣告」及「型別複製＋4E516保守宣告」的交互作用；本批分別驗過各因素，尚未驗證這兩個組合。不重跑本批相同來源矩陣來尋找偶然成功。RND只核對原始產碼，沒有執行或宣稱亂數對拍。

來源、旗標與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_item_effect_attempts`、`game_command_hit_attempts`、`item_effect_hit_original_evidence`、`validation.item_effect_hit` 與 `bootstrap`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_item_effect` 或 `game_command_hit`，版本用 `--compiler-version 10.0a` 或 `9.5`。各輪使用新輸出目錄，再以固定原檔及IDA匯出執行 `link`。公開完整C候選、工具與整理證據，原版、i64及SDK物件留本機。

提交前核對134份來源雜湊、1314條本地連結、197條教訓與82個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 ACTING與標題共享返回段C群組

- 前輪預定的兩個宣告組合已核對。20C6F新增分開局部與保守呼叫宣告、16位元欄位／32位元容器、直接真值等六種表示；1C75E新增型別複製與4E516保守宣告的組合。兩個歷史編譯器共126份新候選均負例，舊396份不重跑。1C75E的原始矩陣仍由78a05a7b提交中的來源與驅動器重生，不改寫舊收據。
- 新增[完整群組C](tools/fd2_matching_game_title_acting_group.c)與[回歸](tools/test_fd2_matching_title_shared_group.py)。1366A完整818 bytes與1F894完整1765 bytes共有92-byte局部框架；1FF74以E9跳13994的八位元組收尾。先核對固定原檔、IDA函式邊界、raw bytes與呼叫端，將契約登記為READY，再實作同一C編譯單元，不把1765-byte函式裁成獨立返回函式。
- 來源保留ACTING兩份32-byte局部陣列、原始欄位及更新邊界；標題保留60-byte事件表、五張147列圖、535至0的捲動、原始插播點、存檔校驗、三種選單數量、鍵盤回捲與四次閃爍。聚合、bitfield、volatile與指標重用只作候選，不推定作者宣告。
- 群組六種表示在兩個歷史編譯器共108份候選仍未匹配。編譯器自然產生跨函式共享收尾，既有COFF工具將完整函式分開，只提升跨區間REL32供ld計算，保留指令寬度。共享結構成立不等於完整位元組或入口一致，也不增加C覆蓋。
- 本批六組共234份候選完整編譯、連結兩輪，OMF、完整報告與產碼相同，全由objcopy核對；群組分別擷取兩個完整PE區段再串接。12項錯址、既有LE／常數位移與75427筆載入記錄／偽造拒收回歸通過。
- 既有201個原始函式／30171 bytes、另146-byte未歸屬區塊、202段C出處與SDK25函式／1777 bytes逐項保持。沒有新增匹配，沿用既有全檔基準與SDK來源鏈，負例不放回EXE。929個函式仍待還原或分類，Goal與#198保持開啟；既有語意與Go／Ebiten驗收不提升。

來源、旗標與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `title_acting_group_spec`、`game_title_acting_group_attempts`、`title_acting_group_original_evidence`、`game_item_effect_mix_attempts`、`game_command_hit_mix_attempts` 與 `validation.title_shared_group`。群組編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_title_acting_group`；新組合分別使用 `game_item_effect --cases ITEM_EFFECT_12 ITEM_EFFECT_13 ITEM_EFFECT_14 ITEM_EFFECT_15 ITEM_EFFECT_16 ITEM_EFFECT_17` 與 `game_command_hit --cases COMMAND_HIT_10`。版本用 `--compiler-version 10.0a` 或 `9.5`，各輪使用新目錄後執行 `link`。原版、i64、SDK物件與原生產物留本機。

下一步只對1C75E測一次32位元actor的volatile讀取。原版以EAX讀actor、EBX計算80-byte記錄偏移，目前同長度候選的六個差異都在這段完整索引；若新表示沒有改善，就轉向尚未試過且有完整返回段的函式本體。這是尚未驗證的產碼假說，不是作者宣告。

提交前核對136份來源雜湊、1322條本地連結、197條教訓與82個守衛。舊396份矩陣的C及driver雜湊另以78a05a7b的實際Git內容核對。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 暫停匹配試驗，先提交現有C候選

使用者要求暫停後先commit及push。本批只核對並保存暫停前的來源與收據，沒有再編譯或新增試驗。

- 新增[完整C候選](tools/fd2_matching_game_record_resolution.c)，目標為1DB65的857 bytes。保留124-byte局部、record+5整位元組寫入、word+64、視口公式、13幀姿態、6＋6幀效果及153216-byte背景。完整原始指令與八個直接E8 caller由既有IDA9.4匯出保存，既有狀態／死亡演出語意不重開。
- 驅動器登記11種表示。目前最終來源只有RECORD_RESOLVE_8／9／10的Watcom 10.0a、-mf -3s單次收據，產碼863／845／852 bytes，首差偏移17／17／32，皆非完整匹配。較早局部試驗的來源已變動，不作當前來源的正式矩陣。其餘版本、CPU、成本策略與獨立第二輪仍未完成。
- COMMAND_HIT_11的32位元actor volatile讀取已在10.0a／9.5各測一次，產碼皆193 bytes，首差偏移31，沒有改善。收據保留當時driver雜湊，並驗證由本提交driver移除三條record-resolution註冊行可重建原雜湊；不把歷史driver寫成當前版本。
- 五份既有OMF、完整產碼與來源雜湊核對通過，objcopy獨立擷取五份完整區間。當前driver的錯址、常數位移、75427筆載入記錄及偽造拒收回歸通過。零筆路徑的EAX未在函式本體定義，35BEB caller仍傳遞EAX；不推定所有caller忽略回傳，C型別與pragma維持產碼假說。
- 本批沒有新增C匹配。201個原始函式／30171 bytes、另146-byte區塊、202段C出處、SDK25函式／1777 bytes及既有全檔基準保持。五份單次收據另列，不併入116組6405份候選／6375份獨立核對／30份布局拒收的雙輪統計，不把負例放回EXE。929個函式仍待還原或分類，Goal維持暫停，#198保持開啟，Go／Ebiten及PLAYER-E2不提升。

來源、旗標、物件與產碼雜湊見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的record_resolution_candidate_spec、record_resolution_original_evidence、record_resolution_single_pass、command_hit_actor_single_pass及validation.record_resolution_commit。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的compile --source-key game_record_resolution；單次試驗須明示--cases RECORD_RESOLVE_8 RECORD_RESOLVE_9 RECORD_RESOLVE_10，再用固定原檔及IDA匯出執行link。此入口只供日後重生現有負例，本輪不接續試驗。原版、i64、SDK物件與編譯產物留本機。

提交前核對137份來源雜湊、1328條本地連結、197條教訓與82個守衛。真正主機Issue清單經官方pull／render／verify，16條開啟、8條需人工判定、可能已完成0條。固定原檔、正式IDA匯出與資料庫雜湊保持；變更檔UID／GID1000，歷史root-owned 2811項保持，沒有`.md`目錄。提交前確認本批FD2容器全部退出；推送後核對真正遠端HEAD，結果回填#198。

## 2026-10-15 重製完成與HD排程，匹配研究再次暫停

使用者先要求接續，再明確要求暫停matching decompilation，改查重製缺口、加速重製與HD，並重新調整worklist及GitHub Issues。依復古重製、玩法稽核及工作清單路由載入任務入口；HD架構取捨依grilling確認，沒有新增production規則或素材格式。

- FD2起點2975ecb6，追蹤檔原先乾淨，既有未追蹤work保留。dosgolem已合併；fetch後main與origin/main一致且工作樹乾淨，從81c85be建立fd2-followup-20261015。尚未用整合後程式重生章oracle，分支建立不算行為驗證。
- 暫停前，既有record-resolution來源在Watcom10.0a編譯99份候選；連結容器缺wdis而退出，屬環境失敗。compile-report位於work/matching-full-20261008/record-resolution-resume-20261015/10.0a-r1，SHA-256為5c72a5002fc467bc87d4d196fc77530d705a5915eaf39af4a9f2da74b896a0d0。沒有完整link報告，不列產碼負例、不加雙輪統計；C來源、driver與主收據未改。使用者暫停後未接續編譯，容器已退出。
- 真正主機gh auth成功。讀取遠端Issue後更新16個既有題目的排程、現況與依賴，新增#199～#204六個執行子項。總項#14／#4不與子項重複計算。#198、#52、#197保持開啟，標P3與狀態:暫停；後續平台及編輯器的前置排除暫停研究，保留原驗收內容。
- 遠端22項逐條核對：P0六項、P1八項、P2五項、P3三項，每項恰有一個P級標籤。官方build_snapshot與pull快照完全相同，父子連結及#142→#203依賴通過。工作清單verify沒有可能已完成項；沒有關閉任何未達驗收題目。
- 唯一分層現況58新增本日狀態；56／57同步研究與重製的邊界，61同步HD執行子項，91由官方快照重生。舊匹配研究與章收據保留。沒有重跑已passed章、提高章分級、調整像素預算或恢復第12章目標追蹤。
- 本機缺少fd2-go-test-local:latest及fd2-assets-local:20260829-sfx，受版控Dockerfile仍在；#203記為環境前置。沿用既有eob-remake-release:1.26.7-ebiten2.9.9-20261008-r2容器的Python／Pillow完成清冊機械核對，不用它替代FD2正式Go／GPU驗收，也未建立重複映像。
- 全域sync_native_turn_events.py --check通過，仍有27筆未完整轉寫。現代catalog的161筆結構與私人檔尺寸／SHA／透明度驗證通過；file/files指向的1153檔全部存在。數量與正式章台帳只記在58；原型素材存在不代表全玩家路徑或高解析呈現已完成。
- 工作清單既有19個測試與現代catalog既有1個私人素材測試通過。高解析HD第一階段範圍已提問，#204維持DRAFT；現有核准風格、靜態嘴型、圖塊與存檔語意未改。

本輪可重跑入口，以下Python及測試命令均在上述既有容器內、repo唯讀掛載時執行；render需另將文件輸出所在工作樹可寫掛載並指定UID/GID1000：

```sh
# 只有官方Issue控制面在真正主機執行。
tools/fd2_worklist_issues.py pull
# 以下在Docker內執行。
python3 tools/fd2_worklist.py render
python3 tools/fd2_worklist.py verify
python3 -m unittest discover -s tools -p 'test_fd2_worklist*.py'
python3 -m unittest discover -s tools -p test_modern_theme_catalog.py
python3 tools/sync_native_turn_events.py --check
python3 tools/validate_modern_theme_catalog.py --verify-private
python3 -c 'import json; d=json.load(open("docs/data/parity-campaign-progress.json")); print([(c["chapter"],c["status"]) for c in d["chapters"]])'
python3 -c 'import json,collections; d=json.load(open("remake/assets/themes/modern/catalog.json")); print(collections.Counter((a["role"],a["status"]) for a in d["assets"]))'
```

本輪只重排工作與修正現況文件，未提交或推送程式變更；GitHub Issues已直接更新。歷史root-owned基線2811項未增加，沒有`.md`目錄，變更檔及新研究輸出UID/GID1000。本批容器均退出，其他專案容器未動。

## 2026-10-15 恢復正式工具鏈，補齊IRQ7並回到第24章

使用者選擇先補完整IRQ7鏈，再繼續章對拍。matching decompilation維持暫停。
依復古重製與平台規格優先路由，使用公開DPMI／DOS4GW契約及dosgolem既有305，
先建立429 READY規格，再接真實IVT、DSP確認、PIC EOI、IRET及caller返回。
沒有逐週期ISR考古、輪詢替代或遊戲規則變更。

- 由既有Dockerfile重建fd2-go-test-local及fd2-assets-local，最小context只含五個受版控輸入。
  Go1.22.12與Pillow11.3.0沿用原工具鏈；原版、私人素材、IDA與work未進build context。
  私人現代素材18項通過、零略過；完整Go19套件2383項通過、52項既有選用略過。
- 合併main81c85be的首輪正式原版，在5780804步／LE0x373FE因IRQ7接線缺口拒收。
  actual IVT0F=67FF:06F3，其他兩種0F向量皆零。原始失敗保持。
- dosgolem共用main保持乾淨，專用worktree位於work/dosgolem-fd2-followup-20261015。
  實作e26fb41及驗證規格2c5c104已推送fd2-followup-20261015，遠端HEAD已核對。
  cpu386／machine／FD2 oracle三套件2021項通過；10項略過皆缺MOO2可選原版。
- 真實FD2 ISR每次76條指令。600萬步3次進場／返回；52516893步到標題等待時504次皆返回。
  固定第三方CONTINUE槽及既有鍵盤計畫兩輪皆594524785步，233份完整狀態、233張PNG與3筆trace相同。
  舊／新嚴格比較仍failed，共446筆差異；完整單位、鍵盤讀取與DOS呼叫相同，view只在aux_phase不同。
  211張PNG不同，舊seq0缺圖；不將全部影像差異歸為動畫，不補圖或增加忽略欄位。
- native准入保留舊／合併未修／IRQ7修正版的完整來源組合，修正版另綁派送與實模式橋接兩檔。
  真實Git bytes測試接受三組與當前來源，拒收六檔單獨修改及兩種新舊混搭。oracle相關108項Python回歸通過。
- 現代靜態宣告盤點保留來源SHA與JSON pointer，不複製台詞。九個非負speaker候選缺頭像，負speaker另列；
  尚未證明正式玩家可達，不直接開產品缺陷或生成素材。高解析範圍仍DRAFT，catalog保持runtime_candidate。
- 真正主機Issue更新#142及#201，依限定工具收據關閉#205與#203。官方pull拉回21項；暫停項保持開啟。
  #142接續第23章實際保存槽，SHA-256為22bd63070189e0e6703e14d4bd4db8dc1f66452a21128dd5154579171ecbea63，
  seed4及祖先AP200／DP0／DX60政策保持。原版第24章首輪已到T3、單次清敵、戰後整備及成功DOS保存。
  重播因漏帶既有舞台trace而拒收，屬本輪驗證參數問題；補回既有0x24D48／0x11EED等入口後乾淨重跑。

唯一主證據為[工具恢復收據](docs/data/fd2-remake-toolchain-recovery-20261015.json)，保存輸入、來源、映像及輸出雜湊。
原始原版收據留在work/remake-recovery-20261015；不提升整章PLAYER-E2或公開私人素材。

可重跑命令均在Docker內，repo與原版唯讀、明確輸出可寫、UID/GID1000；Go及素材測試使用恢復的專用映像，
dosgolem使用dosgolem-go-ebiten:1.24-r1。oracle入口與原版／state掛載依[正式驅動器](tools/dosgolem_oracle.sh)。

```sh
FD2_ASSET_PACK=/out/pack FD2_TEST_MODERN_PACK=/repo/remake/generated-assets/modern-theme-prototypes \
  with-xvfb go test ./cmd/fd2 -run 'Test(LoadModern|ModernTheme)' -count=1 -v
FD2_ASSET_PACK=/out/pack with-xvfb go test ./... -count=1 -json
go test ./internal/cpu386 ./internal/machine ./apps/fd2/cmd/oracle -count=1 -timeout 480s -json
python3 tools/test_oracle_native_source_profiles.py --dosgolem <專用worktree>
python3 -m unittest discover -s tools -p 'test_*oracle*.py'
python3 tools/audit_modern_theme_coverage.py --verify-private --output <現代宣告報告>
python3 tools/verify_oracle_prefix.py \
  --prefix work/remake-recovery-20261015/normal-original-irq7-r1 \
  --full work/remake-recovery-20261015/normal-original-irq7-r2 \
  --compare-frames --compare-trace --out <新版兩輪比較報告>
```

舊／新版比較另用原memory197-command-native-original-r1及完整fa25f6e／e26fb41提交對，
`--compare-frames --compare-trace`不變。第24章使用受版控ch24-sample-r3計畫、native近堆及完整舞台／AI trace，
實際初始SAV另拷貝至state覆蓋層；每次運行有界，原始遊戲檔不寫入。

第24章後續結果：補回舞台trace後，r2重播通過，但seq162 PNG被下一個單步控制覆寫。
行為、節點與完整SAV相同，畫面閘門依缺圖拒收；容器exit143與缺少計畫尾端mark也如實保留。
通用單槽pendingFrame問題登記#206、P1，沒有修正或關閉。r4計畫在T3 mark後正常空等待200萬步，
原版r3完整走到退出存檔列表後的整備mark、exit0，211份狀態與211張PNG齊全。
重製側11.10秒通過，16筆AI入口全數消費、零順序分岔；四gate通過、13張完整RGB皆零差異，
兩側完整SAV SHA-256皆a1e6cc85722fc820cf38f820f4454785cb2138450d9c6000226466a340288406。

本次只驗T1～T3、移動、敵方回合與戰後保存，未抽T4／7／10事件、普通攻擊及其他適用戰鬥路徑。
不因四gate或13張零差異直接提升整章111，#142與正式章台帳保持。生成比較收據內嵌於工具主收據的
`chapter24_followup.bounded_receipt`；r4計畫已掛00索引。#206與缺舞台trace的前輪失敗分開記錄。
現代主題未生成新素材、未改格式或正式繪製路徑，高解析範圍仍DRAFT。

本批最後核對：遠端22個未完成Issue，官方pull／render／verify通過，可能已完成0項。
工作清單19項與oracle相關108項Python測試通過；教訓198條、83個守衛通過。
變更文件2238個本地連結通過，20份來源／輸出SHA全部相符。全知識庫掃描另發現9個既有問題，
分別為歷史本機交付物缺失及舊文件中的括號被簡易解析器當連結；均不在本批變更文件，不宣稱全庫連結通過。
原版及私人素材未改，變更檔UID/GID1000，歷史root-owned仍2811項、`.md`目錄0。
dosgolem專用分支及共用main乾淨；FD2提交只加入整理證據、工具與文件，work及大型擷取不入Git。

## 2026-10-15 第24章流程驗收與事件54補齊

使用者允許章前有界 DP 校準，隨後指定作弊過關、參考攻略與流程一致，不要求每個 frame 相同。已同步111／114、56／57／58及 GitHub #14／#142／#33。匹配式反編譯保持暫停。

- 第24章 event54 原先只有登場增援。本輪重用已核對的 IDA 9.4 匯出，與固定 EXE 及 Capstone 5.0.3 交叉確認125 bytes，保留 sub_3535D、sub_35822共用尾端及全域表consumer。規格 READY 後才補四角鏡頭與等待，九項具型別操作同步四筆正式／canonical回合事件。入口為 [event54主契約](docs/data/ida/fd2_turn_event54_20261015.json)。
- 本地青衫攻略第24章記載四角增援，回合為3／4／7／10；固定控制列與原版實測為2／4／7／10。保留差異，攻略只作玩家旁證。
- 新增同章 boost-only 建槽模式，拒收混用推章、升級、事件與金幣。第一輪沿既有 fresh 槽只加DP1000，seed4，25名冊僅100個DP資料bytes及2個checksum bytes不同；HP／MP／物品／金幣與其他槽相同。原最早ch02基底不存在，直接沿已驗fresh產物，沒有猜補來源。
- 流程驗收仍嚴格檢查行為、節點、交易與完整保存。畫面保留原始結果，advisory模式不參與完成判定。測試可省略圖片輸出，正式Game.Update／Draw完成閘門仍執行。31項Python工具測試與建槽、字串Go回歸通過。
- 原有第24章短鏈重新消費新劇本，16筆AI入口全消費且零順序分歧，13張完整RGB零差異，完整SAV相同；這份回歸只驗T1～T3。
- 官方全域事件同步檢查通過，剩23筆未轉寫，沒有將spawn-only提升為完整事件。
- 字串review與新清冊雜湊原已一致，本輪測試Go變更不改production inventory；summary與四語內容落後既有來源，沿官方遷移器補同步，保留原譯文與審查狀態。10筆新來源用明示候選，外語維持machine_draft。四語結構驗證及14項測試通過，其中1項依原環境條件略過。

環境修正：Python建槽測試在唯讀repo先建立cache失敗，改為先拒收錯誤基底檔名再碰cache，乾淨重跑通過。第一次DP位元組核對漏列既有末尾checksum，按fd2save契約補核對後通過，未放寬資料欄位範圍。全docs連結掃描列9個既有歷史／交付路徑缺件，本輪新增連結另行核對，不把它們寫成產品缺陷。

本輪第一輪已正常至T11，T2／4／7／10增援全觸發。重製端重播通過，280筆AI入口全消費且零順序分岔。清敵一次60筆後，原版正常END於9470673688steps異常跳至0x1666E8，next bytes疑似資料，真因未知。沒有戰後SAV，流程收據保留failed，章台帳不提升。已登記[P0 #207](https://github.com/wicanr2/fd2_re/issues/207)，不直接補opcode或再調DP掩蓋。第23章完整流程三閘門回歸通過。

本輪完整Go回歸19個含測試套件全部通過。原版追蹤實際確認event54入口與四個pan、返回均各四次，控制序號134／192／330／538；失敗流程收據已公開索引，章台帳仍20章passed。

提交前已核對本輪31個變更檔UID／GID1000、2259條本地連結與git diff --check。工作清單由真正主機GitHub拉取後render／verify，23項未完成、15項人工判定、可能已完成0項。199條教訓與83個守衛通過。本輪FD2容器均已退出，既有root-owned基線2811項保持，沒有`.md`目錄。FD2專用映像只保留既有可重現版本；未清理其他專案映像或原有IDA工具鏈。

## 2026-10-15 第24章清敵視野與戰後交接接續

接續b8e7ec37與#207。工作樹只有既有work產物，matching維持暫停。原版對拍／規格閘門／文件職責路由及復古重製、IDA技能已核對；dosgolem遠端main仍81c85be，前輪IRQ7分支尚未合併，沿乾淨2c5c104。主機gh auth成功，開工官方pull為23項。

先轉送既有oracle的trace-tail，原預算末端2000條唯讀EIP，並明示寫入runner。診斷重播到第5回合時，已由既有IDA9.4完整匯出找到更窄證據，因此主動停止並保留部分輸出，沒有當作新的完整原版收據。

原版sub_1DB65完整857 bytes與固定EXE／Capstone5.0.3逐指令及邊界一致。局部124 bytes中，30個畫面指標佔0x00..0x77，malloc指標位於0x78；0x1DC29無數量限制地寫入，0x1DEB6..1DEBD依原順序恢復暫存器並RET。失敗前checkpoint631有40個符合畫面與死亡條件的角色，第31..35項同時解釋四個損壞暫存器及0x1666E8返回位址。五項還原同一畫面基底0x14F018，原生heap事件亦有此153216-byte配置。原始writer／consumer已證實，崩潰關聯列強推論，未宣稱取得實際RET trace。

READY修正只改章測試輸入：第11回合後正常方向鍵移游標至(0,0)，再一次HP-only清敵及正常END。沿相同DP1000槽、seed4及T2／4／7／10流程，不改CPU、原版死亡規則或正式重製。原版r2正在執行，通過前章台帳維持todo。重製端新增對應mark，含最後一個方向鍵重播與原版游標／鏡頭四座標核對，不靠座標注入。證據、規格與命令沿[event54主契約](docs/data/ida/fd2_turn_event54_20261015.json)，[視野稽核工具](tools/fd2_clear_viewport_audit.py)可重算。破壞EBP的反例拒收。

來源准入正反例通過，兩個shell入口語法通過；TRACE_TAIL=2在oracle啟動前拒收，實際診斷runner記錄enabled=true。來源准入首輪未掛git worktree共同.git目錄而失敗，補唯讀掛載後用相同容器重跑通過，未改准入清冊。Capstone首次多傳python3造成entrypoint用法錯誤，改用原Dockerfile既有入口後乾淨重跑；均屬環境／命令問題。

原版r2於第6回合seq303被SIGTERM終止，exit143、oracle.log為空，沒有原版錯誤收據。前301份完整JSON bytes與c1相同，沒有忽略任何欄位。改由Docker detached容器自身的2040秒timeout管理生命週期，以相同來源／槽／計畫重跑r3；保留r2，分類環境中止，不再校準DP。第23章以最終重播程式回歸，行為、節點、交易與完整SAV通過。

等待原版期間先核對下一章：攻略第25章第6回合援軍對應全域event56，固定表入口sub_35487，IDA LE linear35487..354DD共86 bytes。原始呼叫依序為pan(6,40)、spawn group1、ACTING74、文字索引5的對白、跳至134E4重設pose。1366A的既有語意索引已證實為ACTING播放器，不能誤當單位聚焦或把74當角色槽。event55是另一條炎龍劍事件，不混入本回合援軍。此處只準備入口，依同步工具契約等第25章原版收據後才接完整正式劇本，剩餘未轉寫數量保持23。

原版r3已完成710份狀態、兩次成功DOS寫入、戰後四槽保存與退出列表，SAV SHA256 e67cf23b0c5f72e40742b69544cee572cfb155e01a75d7eff65bf3fbdc20ee95。T11改鏡頭前622份完整JSON與c1逐byte相同，原版來源與輸入雜湊在執行期間保持。清敵時實際可見候選只有3個，仍只寫60個敵軍HP，沒有改byte5。

第一輪重製重播在戰後被舊[28,86]前沿白名單拒收，已另列#208。677與685兩個原版快照分別含24C51／24CB2 caller及有效76筆陣列，依既有#145契約限制先寫READY修訂，再僅增加實測76，其他未知count仍拒收。重播再次通過，280筆AI全消費且零順序分岔。行為34點、節點36點、交易與完整SAV均通過；畫面未擷取，frames=false且required=false，不冒稱逐像素一致。

[正式收據](docs/data/ui-traces/parity-ch24.json)與台帳已由官方工具更新，第24章passed，合計21／30，另9章todo。19套件Go回歸通過；其後76槽binding資料修訂再通過正式章與相關Context／Frontier／Binding回歸。123項Python工具測試通過，27項既有Go／字串回歸及第23章完整流程／SAV回歸通過。匹配式反編譯保持暫停。

本批已關閉#142／#207／#208，#199成為下一個P0；正式保存名冊25筆、HP>0為11筆，下一章先驗正常選人並保留原來源。工作清單已由真正主機拉取／render／verify，21項未完成；最後核對18檔UID／GID1000、2262條本地連結零缺件、200條教訓與84個守衛通過。既有root-owned基線2811項保持，`.md`目錄0，本輪FD2容器全部退出。提交及真正遠端HEAD將回填已關閉Issue。

## 2026-10-15 第25章主線流程接續

目前狀態：第24章已passed，台帳21／30；第25章尚未驗收。沿f0c04db3及乾淨dosgolem2c5c104，主機Issue #199為P0。依原版對拍、規格閘門及文件職責路由載入技能與任務入口。使用者詢問必要時啟動matching，現階段先以原版收據定位差異；未啟動編譯試驗。

新輸入是第24章實際保存e67cf23b0c5f72e40742b69544cee572cfb155e01a75d7eff65bf3fbdc20ee95，複製到獨立state。名冊25筆，HP>0為11筆，零額外治療、升級或強化。攻略第25章第6回合己方結束援軍作旁證，依正常鍵盤推至T7後一次明示清敵；炎龍劍支線未列入此主線計畫。畫面advisory，流程三閘門與保存仍必要。

入口：[接續槽清冊](docs/data/parity-slots/ch25-manifest.json)、[主線計畫r1](docs/data/parity-plans/ch25-flow-cheat-r1.jsonl)。原版使用唯一tools/dosgolem_oracle_container.sh，來源與原版唯讀，輸出位於work/ch25-flow-20261015；不覆寫前章。

首輪原版至T5必存角色聖寇拉斯index16／identity26，HP104→0後返回標題，未觸發event56、未保存。保留555份狀態與失敗流程收據，不把正常敗北當執行器缺口。r2沿同SAV／seed，以每回合正常move_unit向(0,43)退避；T1退避前104份完整JSON bytes與r1相同。沒有追加DP或注入HP。

另以原版battle_start seq103建立唯讀前綴供重播診斷。重製原在story_ch25的339F5呼叫24B4D時要求g.st，無法消費LOADCH的故事載體，已開#209。沿既有3396A完整IDA匯出與24B4D／story adapter證據先寫READY，再沿materializeNativeStoryMapState／composeNativeStoryMapFrame組裝場景，輸入組裝完成即恢復g.st=nil。正常四次揭幕參數為20／20／20／60，未略過演出。相同前綴重播通過；18項相關Go回歸及場景成功／缺資產保持非戰鬥狀態通過。字串清冊SHA bba089ef90272e287c20260b037d3a26805aae7effc1a71fe4ee2ef1d6dd4ac3與review相同，不遷移四語。

第25章event56的完整86-byte靜態轉寫已READY並加入producer，尚未降階到正式劇本。既有35487在9份歷史編譯候選中5份完整匹配，C來源SHA相符；無新matching編譯。依規格等本章原版收據後才接正式consumer。唯一主契約為docs/data/ida/fd2_turn_event56_20261015.json，已掛00索引與58。

第二輪仍於T5因聖寇拉斯死亡返標題，沒有觸發event56。退避不是足以通關的結論，不再單靠調走位重跑。原始LOADCH紀錄證實其自帶兩瓶神聖之水raw槽2／3，沿既有type5、amount999、玩家目標與回復交易，先寫READY後補控制器及重播。第5／6回合正常自用，沒有改初始HP、增道具或鎖血。計畫完成閘門要求兩次item_use；125項Python回歸通過。

第三輪在T5實際完成正常物品使用：seq377→378，HP57→600、byte5=128、第二瓶補到槽2。原版無錯誤，控制器誤以為被用掉的原槽一定空掉而拒收。回查既有已證實sub_1B8E7與原版八格bytes，修正為完整左移補位及尾flag0x80、尾item byte保留；第二次仍選當時槽2。保留r3及其控制器SHA，r4沿相同容器／槽／seed乾淨重跑，不計成DP校準或遊戲缺陷。修正後126項Python測試通過，含取自r3控制鏈／八格內容的正常消耗正例。

進度訊息勘誤：先前所稱退避後第4回合仍600 HP，應為第2回合當下值；第4回合起手原版為199 HP，第5回合起手57 HP。以checkpoint及主契約為準，不用舊訊息推定存活。
原版r4以正常鍵盤完成T6 event56、T7、戰後及酒店保存／離開，共601份checkpoint。T5／6正常使用原有神聖之水，HP57→600及6→600；T7正常移鏡頭後一次HP-only清敵35筆。沒有追加道具、強化或鎖HP。event56固定入口與完整五操作實測後，正式producer、scenario及canonical同步。

前綴揭幕修正後，另查到battle_ch25缺原生view及HUD繼承，開#210並依READY補資料與generator。正式AI首個敵軍施法者record21延後，造成12筆順序漂移；診斷找到ScoreNativeAI1598A的三個caller未傳既有1F183閘門，開#211後先寫READY再接線。相同原版r4完整Game r2消費240筆AI、零順序分歧，正常物品、T6演出及戰後隊伍／酒店都到達。這些缺口皆由既有證據定位，本輪沒有新matching編譯。使用者另允許必要時局部匹配，118與#198已明示界線，全面仍暫停。

完整Go初次只敗在舊測試仍期待event56單一spawn；改成核對完整七個lower動作並保持戰後名冊隔離範圍後，19套件乾淨重跑通過。132項Python回歸通過；第24章以最終Game重新重播，280筆AI零順序分岔，行為／節點／交易及整檔SAV相同。全域sync check通過，剩餘未轉寫22筆；字串inventory SHA仍為bba089ef90272e287c20260b037d3a26805aae7effc1a71fe4ee2ef1d6dd4ac3，未遷移四語。

第25章目前未通過。原版SAV SHA0d420b379a38906b208a5e2a88dd74617b1c9a763010205b5c864f0ff102a9cb，重製52984034e916259e2c4a45eafd0bb36f1835edabfd8867e007dee9ad38c1996b。全部明文只有identity26的+0x28..+0x30區間七byte與衍生檢查碼兩byte不同。既有IDA建構器未寫此區，11506整筆複製，本次原版runtime一路保持同值；用途及最初producer仍未知，不能宣稱指標或無用欄位。新#212追蹤，正式失敗收據docs/data/ui-traces/parity-ch25.json、主契約opaque_save_gap保存完整證據。工具先達READY再加--oracle-save，核對實檔SHA與checksum後揭露所有差異，未排除任何byte或放寬交易閘門。台帳第25章blocked，passed仍21／30。

已提出具體九位元組區間驗收例外供使用者決定；尚未收到答覆，完整保存門檻保持，不進下一章、不把oracle常數填進正式runtime。前三項consumer缺陷的驗收範圍與未知存檔題拆開，章驗收由#199／#212繼續追蹤。

提交前官方主機pull／render／verify：22條未完成，可能已完成0條；三項接線缺陷#209／#210／#211已結案，#212與父#199保持。台帳21 passed／1 blocked／8 todo一致，202條教訓及86個守衛通過。40個變更／新檔UID/GID1000；本輪文件2289條本地連結零缺件。全域另有9項缺件／解析誤判，均位於未改動歷史文件，不算本輪新增或產品缺陷。原始輸入及r4控制器／計畫雜湊保持；既有root-owned基線2811項、md目錄0、掃描錯誤0。本輪FD2容器全部退出。提交及真正遠端HEAD核對後回填已結案Issue。

## 2026-10-16 第25章有界保存驗收定案

使用者確認接受逐筆揭露第25章保存差異並繼續。命中原版決定性對拍、規格閘門及文件職責路由，載入既有入口、重製技能與必要reference；開工工作樹只有既有work未追蹤資料，沒有FD2容器。真正主機gh auth成功，官方pull為22項未完成。

先逐byte保留原parity-ch25.json為parity-ch25-flow-r2-save-failed.json，再更新111及event56主契約save_acceptance_policy為READY。正式比較器新增明示--save-policy ch25-opaque-20261016，只接受第25章同流程profile、輸入SAV、計畫、原版EXE／dosgolem、兩側實檔SHA及完整差異清單；七byte與衍生檢查碼兩byte全部保留，bytes_equal=false。未知用途不升格、不回填oracle常數，不改正式遊戲與存檔格式，其他章及預設strict保持。

相同原版r4唯讀，正式Game乾淨重播remake-approved-r3至酒店，240筆AI全消費、零順序分歧，SAV SHA仍為52984034e916259e2c4a45eafd0bb36f1835edabfd8867e007dee9ad38c1996b。使用預設strict同pair仍failed；明示有界政策後行為／節點／交易通過。137項Python工具測試通過，涵蓋不同章／profile／來源／實檔SHA、新增／刪除／改值差異、缺保存writer等拒收；第24章流程與整檔SAV回歸仍passed。此輪沒有Go正式程式修改，未重跑原版或啟動matching編譯。

政策達CONFORMED，第25章按111例外登記PLAYER-E2，官方台帳set／verify更新為22／30。第25章slot_policy明示直接接續實際SAV、零追加升級／強化／event-state改寫，祖先政策獨立保留。58、56、57與索引同步，歷史failed及其SHA保持；#212以使用者接受有界證據限制結案，不宣稱未知語意已解或保存bytes已修成相同。下一個P0為#199第26章。

最終相關回歸141項通過，包含原137項工具與4項台帳測試。真實第25章缺少實檔報告反例及第24章錯套政策反例都交易failed，明示政策沒有一般化為遮罩。原failed收據與前一提交逐byte相同，正式收據來源SHA與原版EXE保持。官方主機pull／render／verify為21條未完成、可能已完成0條；台帳22 passed／8 todo／0 blocked，203條教訓及87個守衛通過。本輪16個變更／新檔UID/GID1000，2294條本地連結零缺件；全域既有9項缺件／解析誤判均在未改動歷史文件。root-owned既有基線2811項、md目錄0、掃描錯誤0，FD2容器全部退出。提交及真正遠端HEAD核對後回填#212及父#199。

## 2026-10-16 第26章原版主線接續

開工1f8fa562，工作樹只有既有work未追蹤資料，dosgolem專用工作樹2c5c104乾淨，沒有舊FD2容器。命中原版決定性對拍、規格閘門與文件職責，載入既有入口及重製技能。主機gh auth成功，官方pull為21項未完成，#199接續第26章。

沿第25章原版實際SAV 0d420b379a38906b208a5e2a88dd74617b1c9a763010205b5c864f0ff102a9cb複製獨立覆蓋層，名冊27筆，不追加治療／升級／強化／道具。悠妮identity9現HP0，先由正常LOAD／選人核對本章必存角色條件；不能章內鎖HP。第25章保存例外不套用本章。

攻略未知的迴廊與原有控制列表對照：event57在敵方T2／4／6／8／10／12／15／16／17，group由當前53BEF round取得。沿既有IDA9.4完整匯出保存sub_354DD 33bytes及353C4..353DA共用尾段；完整操作為spawn當前round group、pan(9,0)、delay400。現行九列只有spawn，先建立READY契約，等原版收據才接正式完整lower，不重開matching編譯。

95個正常鍵盤控制命令，預估19～29分鐘、2核；原版r1由唯一tools/dosgolem_oracle_container.sh執行，原始素材及dosgolem唯讀，獨立state-r1／original-r1，native近堆、40bn步預算、外層2040秒timeout、記錄AI及event57入口。預定T18後正常移鏡頭再一次HP-only清敵、戰後及酒店保存；台帳仍todo。入口docs/data/ida/fd2_turn_event57_20261016.json、docs/data/parity-slots/ch26-manifest.json及docs/data/parity-plans/ch26-flow-cheat-r1.jsonl，均已掛00索引。

第26章r1於cp78取消：原版LOAD回城鎮，獨立整備前綴誤入教會狀態窗。此為控制計畫問題，未進戰鬥，不列產品缺陷。接續原版槽悠妮HP0，復活費4000高於金幣2000。r2沿已授權建構政策，使用第24章slot-c1完整隊伍基底，以`fd2-chapter-slot -target 25 -levels-per-chapter 6 -seed 4 -boost-base-dp 1000`建槽；累計政策及所有27筆狀態見清冊。正常城鎮選人先選14筆，再以down／right選末位亞奇梅吉。輸入與規格來源已登記索引，保存仍嚴格比較。

第26章r2在戰前探訪取消：`town_probe`固定settle後於轉場狀態反覆送esc，建築選擇偏離，未進戰鬥。r3保留同一建構槽與正常選人輸入，移除戰前探訪，先驗九波與酒店保存；五處探訪與買賣未抽樣，不冒稱完成。預估104個邏輯序列，2核上限，單執行緒原版實測約每秒160萬指令，外層90分鐘有界。

第26章原版r3正常至T6。診斷前綴到T5的Game可執行且176筆AI零順序分歧，但比較器揭露站位／HP不同，不能稱通過。#213定位為缺少raw25的required[9,29]及依序front[9,29]資料；沿既有318AD原始caller與31DBE／321C8閉合helper，READY契約先記於第26章主證據preparation_contract，不重新深挖helper。

原版r3在T13正常敵方範圍法術後悠妮HP272→0，回標題；六波已觀測，零清敵，後三波及戰後／保存未驗，不將敗北當完成。依使用者已明示作弊通關方向，本章採章前一次HP／MaxHP+30000，沿既有signed word欄位，不章內鎖HP、不改正式規則。預檢原子性、保持傷勢與HP0、未指定時完全不改的READY契約先記主證據hp_slot_policy，準備受版控建槽旗標。

章前HP政策工具Go回歸通過，slot-r4由同章slot-r2以`-boost-only -target 25 -boost-max-hp 30000 -seed 4`建構，27筆，零handler／升級，SHA f04c6761f4dd4aa0a1047dc5d18fbfdbcfdfac94cccf0fd62169ca9895a645ff。索爾30420、悠妮30272、亞奇梅吉30416 HP；所有數值低於signed word上限。r4控制命令與r3相同，只有來源槽與政策標示不同，原版source仍乾淨2c5c104，90分鐘有界。

r3敗北診斷Game消費601筆AI零分歧，一般比較behavior=true，但實際end為cursor／T14，不能證明敗北終態。最初誤設只適用第15章的DEFEAT_TAIL測試旗標，已按同容器命令移除並乾淨重跑，失敗log保留，不列產品缺陷。新#214實際缺陷為第26章缺20B3C結果列及原生敗北返回設定。固定EXE54bytes、章25結果表51B7D及既有helper契約已核對，READY先存defeat_contract；只補資料，不重解一般敗北呈現或新增matching。

#214補兩列原生結果與敗北返回資料，移除未證retreat_ch26路由及孤兒節點，保留既有flag。同r3正常Game消費601筆AI，零順序分歧；record2悠妮由272承302傷害至0、raw+5 bit0成立，native_round13觸發record2_inactive，完整原生敗北／publisher／開場至title menu selection0，保存未變。有限工具只新增章26的已觀測終態准入，其他章仍拒收。原版9／36tick影像未逐像素比，不提升完整章；r4HP流程繼續。

第26章正式r4三項流程閘門通過，913筆AI全消費、零順序分歧；SAV 22987 bytes整檔相同，SHA 89a8d7a93f8baae11e2315f9d06c9a979ef5d6d608ee1558ec42ffb198cdbbd8。首次全套Go有三項失敗：覆蓋率舊計數、文字審查舊SHA，以及我加入全域FD2_MUTE造成command17缺sample反例失效。來源稽核先重生為117故事／過場、57handler、51fallback及26retreat，再同步斷言。文字清冊重綁98筆處置不變，四語5094筆全以既有id配對、new=0；canonical-root最初誤指父目錄，按工具規格改story後乾淨重跑。三個針對性Go測試通過，完整同映像回歸移除全域靜音重跑。

#216渲染收尾勘誤：原README仍有舊動態標記，首次渲染改寫了區塊，保留SHA斷言拒收該結果。現以穩定現況連結取代動態表，README區塊外所有文字對HEAD逐字相同，文化保存、圖片與技術導覽未變。進度頁的動態表改由工具產生，前言不再無條件稱四gate通過；CLI重跑與--check保持README SHA。

最終19套件Go回歸通過；第25章用目前Game乾淨重播，240筆AI零分歧，既有有界保存政策仍passed。另審查canonical時發現移除一節點使預設初次匯入規則重編121個ID；#217新增選用既有canonical來源，依legacy_key保留其餘318個ID與引用，不改schema或預設初次匯入。刪除／插入、錯文件／重複ID／錯kind回歸及官方CLI精確重生通過。9項相關Python工具及三相關Go套件通過；最終第26章再重播913筆AI零分歧，整份收據bytes與正式收據相同。README另修正驗收政策段落，不再無條件稱四項全過，其餘文化內容保持。下一章27先核對LOAD owner與已閉合318AD的raw26必出／前置資料，不沿用第26章選人鍵或重解helper。

最後渲染審查補正：非必驗畫面明示非必驗，未比像素不顯示0px，第25章保存明示差異已揭露並接受；10項渲染工具測試通過，相關Python合計85項。官方主機pull／render／verify為21條未完成、可能已完成0條，#213～#217已結案，#199接續第27章。台帳23 passed／7 todo／0 blocked，205條教訓及88個守衛通過。本輪49個變更／新檔UID/GID1000，2750條本地連結零新增缺件；全域既有9項缺件／解析誤判均在未改動歷史文件。root-owned既有基線2811項、md目錄0、掃描錯誤0。README僅更新進度入口與驗收政策，其餘文化／圖片／導覽逐byte保持；原EXE及最終正式收據雜湊保持。最終台帳verify初次誤掛唯讀而拒絕寫回，改為明示台帳可寫後同命令乾淨重跑一致，非產品缺陷。本輪FD2容器全部退出，其他專案容器不動。提交並核對真正遠端HEAD後回填各結案Issue。

## 2026-10-16 第27章主線接續

開工bf4bdab8，追蹤檔乾淨，只有既有work資料，FD2專用dosgolem2c5c104乾淨。路由命中原版對拍／規格閘門／文件職責，主機gh auth成功、官方pull21項未完成。攻略第27章及新浪旁證確認階梯增援、隊長死亡增援與天空之鑰後續，正式 ABI 沿既有IDA證據。直接複製第26章原版實際SAV，零追加升級／強化／事件改寫；祖先HP+30000及AP／DP／DX仍明示。正常LOAD與選人、索爾向y39移動觸發62→63，T9後一次清敵再驗64及戰後記錄，原始素材唯讀。必出缺陷只補raw26既有318AD caller，required9／front9的READY先於資料實作，不重解helper或啟動matching編譯。

#218必出／前置資料已依READY補入生成器及正式／canonical，318個ID保持。缺9拒收、單9通過與完整名冊前置測試通過；66筆IDA指令對固定原始file bytes相同，Capstone5.0.3對齊。原版r1漏設AI／成長trace，於開戰前取消，不作正式收據；r2同輸入同槽，補0x13A9F／1E54A後乾淨重跑。

r1取消邊界補正：最後cp74已載入戰場並處揭幕對白，尚無敵方AI；先前『開戰前』應讀為首次敵方行動前，診斷原始資料不改。r2補齊唯讀決策追蹤並保持同輸入，50分鐘、30bn指令上限。

第27章原版r2完整至T9及非城鎮保存返回選人，424筆AI入口、event62入口13次與event63一次；一次HP-only清敵59筆。
勘誤：最初計畫預期清敵觸發64，實際沒有358EA；64與隊長死亡增援仍未驗。原版SAV22987 bytes，SHA003d0394e1c5d9bb609949830450b4658284501f2d1e94456cbf96ed69b11663。

揭幕舊binding混入死亡對白且漏pan／ACT81／三轉場，依原caller先補READY再接原生19句；來源實際80筆、初始runtime63，撤回loader前錯置context。
故事inventory predicate接回已物化raw roster首16筆，不借持續隊伍。#220接250CC共同前綴與持鑰匙成功臂23句、ACT82..84、完整typed色盤／轉場及清理。
兩層direct patch25116白名單、透明inventory gate視圖交接及indexed palette consumer依READY修正。
先前『鏡頭歸零』只是症狀：真正根因為節點交接捕獲過時battle carrier，未保持當前story view；無原版歸零writer。

Game r7走完整戰後且SAV相同，但正式比較揭露90筆AI順序分歧，沒有因保存相同就標完成。
#222既有IDA直接反證1567E高command第一階段應用mode1／inner1，149F8 selector0應收raw+6非零；舊consumer誤收同陣營造成互傷。
READY先於實作，11／56及舊map0 score8靜態斷言同步勘誤；六項scorer回歸通過。
同原版r2重製r8全424筆零分歧，正式行為／節點／交易通過，完整SAV嚴格相同；frames0張且advisory，白／黑VGA memset尚未轉寫如實列限制。
415筆IDA9.4來源指令與固定EXE原bytes、Capstone5.0.3邊界一致；沒有新matching編譯或硬體RE。

故事稽核重生118節點／9直接腳本／59handler／50fallback，26舊撤退；既有318個canonical ID保持，只新增共同前綴節點。
字串重綁98筆處置保持，inventory SHA39cd7d84193c274946b5f30399f8a66a34e0b6151dd9d044a9b5faaf2f2938cd；四語各5094筆，4860依ID、234依signature，new0。
完整Go與前章正常Game回歸收尾中，台帳及GitHub最後核對後提交。下一步第28章，先核對非城鎮LOAD owner與已閉合必出／前置資料。

最終19套件Go全部通過，cmd/fd2 283.154秒；81項Python回歸零失敗／略過。第26章913筆AI零分歧、三流程閘門與整檔SAV保持。最後indexed色盤發布清除legacy白屏旗標的三項測試通過，第27章r9重播424筆零分歧，正式收據與r8逐byte相同。最後Go修改的文字清冊SHA保持，四語文字／role／variables全與HEAD多重集合相同。官方canonical importer逐byte重生，318個既有ID保留，只新增1個節點。

真正主機gh已更新#199至第28章，#218～#222結案，#33保持13筆未轉寫與未抽樣64限制；官方pull／render／verify為21條未完成、可能已完成0。206條教訓及89守衛通過。README SHA保持6fd6f74bc84a17d4f05bd31cd624b7d91306a3a756bd62b2405516dcbc38dd70。首次衛生檢查發現2812項root-owned，定位為本輪oracle掛載時建立的空dos-cache/gomodcache；只修該空目錄至1000:1000，未遞迴變更。後續所有mount來源先確認存在。歷史map0 item79分數與phase96／8已在11／56及時間序列文件追加勘誤。

最終衛生固定工作樹後乾淨通過：51個變更／新檔UID/GID1000，本輪3282條本地連結零新增缺件；全域9個既有缺件／解析誤判保持，其中交接檔兩項舊封包路徑逐字存在HEAD未變前綴。root-owned回到既有2811項，md目錄0、掃描錯誤0。前一掃描與git暫存並行且錯誤明細未記錄，固定後相同掃描零錯誤，不列產品缺陷。FD2本輪容器全部退出，其他專案容器不動。暫存51檔無原始EXE／SAV／資料庫或私人素材，正式收據strict保持。提交推送並核對真正遠端HEAD後回填#218～#222及父#199。

## 2026-10-16 第27章隊長增援再次抽樣

使用者補充鎖定索爾與敵方隊長血量直到增援抵達，並要求重製也有增援。先回第27章，暫不開第28章。初始工作樹只含work，主機gh登入成功、官方pull21項。路由載入原版對拍、規格閘門及文件職責；既有358EA直接指令顯示第二名死亡生三組、第三名死亡清場。採有界E1修改路徑，既有正式收據不替換。新dosgolem分支從目前main81c85be建立，補回先前未合併的IRQ7兩提交，核心來源tuple與已驗基線一致。430及主契約resample_hp_lock_policy先達READY，再實作選擇性控制及雙側重播。

第27章增援r1已驗前波12筆與保護清敵56筆，但朝15,22的曼哈頓候選停於15,32。成本格確認y31中央三格不可通行，r2明示經x13繞行六回合，候選先以同成本格核對。r1控制465另因牆鐘期限未收到checkpoint而結束，最後trace仍完成三隊長AI入口，未判為引擎缺陷；驅動逾時改用單調時鐘並加回歸。

增援診斷追加：重製r1-fixed已實際追加前波12筆，262筆AI零順序分歧。首輪測試鎖血漏接持續角色NativeIdentity的既有+8相容契約，已按NativeItemPanelRecordForUnit修正，沒有改正式Game。原版r2於雙隊長攻擊前主動停止，因已證實29F72負差額普通傷害為0；依使用者既有作弊授權，用既有fd2-chapter-slot boost-only生成同章全名冊AP+1000槽，SHA256 0eb3de0a84fd2907f28d5112380511a2e05d13423b0b04a66db5b96c9b9836ea。解碼差異僅27筆baseAP／effectiveAP兩word與兩byte校驗碼，共110byte；r3兩側使用同槽，原正式SAV及章台帳維持。

第27章增援r3／Game r3完成：正常兩次物理死亡派送358EA，三路21筆確實追加，新增位置／HP／raw+8一致；270筆AI零分歧，索爾30420及第三隊長3200存活。獨立E1收據passed；正式24／30及parity-ch27維持。第一次驗證腳本誤對不同AP槽要求BOOT暫存器相同，已分離同輸入95邊界嚴格回歸與AP槽EBX完整差異；未知EBX語意未猜補。終點view差異另列#224。最終Go19套件282.332s、102控制器加4章槽Python、來源准入及90教訓guard通過。

## 2026-10-16 驗證錄影清理

依使用者要求節省空間，核對既有收據與實檔雜湊後刪除下列五段本機暫存錄影，共10548497 bytes。保留收據、日誌、必要截圖與受版控重跑腳本。歷史收據中的影片路徑代表當時產物，不代表目前仍保存。原版第一章來源錄影與成品推廣片保留；後續章對拍沿流程模式，不產生非必要錄影。

- `dist/private-promo/fd2-v011-palace-remake-complete-raw.mp4`：3090291 bytes，SHA-256 `eccadf9fe99b76c253b907de6b449b141729c79058a7aa7472ddbcf3bf1960d2`。
- `dist/promo/fd2-ch01-march-remake.mp4`：3581208 bytes，SHA-256 `efe8a83313771e5be2e5e0fade67c4b7b534364f3395a548cd1bd2a8b5941bcf`。
- `extracted/title_re/dragonfx_compare/remake_intro_8s.mp4`：1642688 bytes，SHA-256 `b12937910d13a6ca0a8e1baa50bdeeaba358fba3ed33c6fa0e6580fb32d0eba3`。
- `dist/promo/fd2-v0.1.0-live-gameplay.mp4`：1566124 bytes，SHA-256 `c3fb3c0c07ebb8f05613bae712eefe9a37227452b4da2f0a62d674430b832d10`。
- `dist/promo/fd2-v0.1.0-ch01-dosbox-vs-remake.mp4`：668186 bytes，SHA-256 `a5e9b98b167d135a7167def90a07aa1450a700db18e158f7180f65f432f96e3c`。

## 2026-10-16 第28章主線接續

開工a3c8669d，保留上一輪錄影清理兩檔；真正主機gh登入與官方pull22項成功。路由命中原版對拍、規格閘門與文件職責。直接複製第27章正式實際SAV，SHA003d0394不變，原來源唯讀；非城鎮LOAD與19人選擇單獨短探針。舊第28章E1聲稱64-slot包含未啟動增援，沿既有IDA及休眠控制列核對，不重解共用helper。

第28章原版正常LOAD短探針完成，34-slot與19人選擇、非城鎮入口已確認。既有222筆IDA指令對固定file bytes及Capstone邊界相同，READY先於補接；新增66／68／70完整轉寫與65／69原子區域啟動，必出前置沿既有318AD。Game同源短前綴的行為／節點／取樣交易通過，0 AI且未驗戰鬥／保存，不登記整章。首次exporter把RET納入對白區間，被完整指令核對拒收；修正半開區間後全58事件及全域回合同步通過，13筆固定回合待轉寫保持。完整原版r1已啟動，T9後才單次清敵；未錄影。

完整r1於T5被實際敵軍與地圖邊界堵住，exit8／cp312，未清敵。校準槽依既有boost-only一次AP+1000，54個word政策值與原檔110byte差異核對，其他byte保持；首次檢查誤將全檔checksum算進第4槽meta，依fd2save.CHECKSUM_OFFSET更正後通過，非產品缺陷。r2先要求完整64筆再清敵。Game r1另在第一移動揭露缺HUD入口，原版cp71與25E5E共同caller先達READY再補typed資料。聚焦晚期測試的原版locator／語系／首張穩定地圖缺件分開修正fixture，沒有放寬正式Game閘門。

#226定位增援candidate回滾PAN六全域，限定66／68／70保留當前view與135F2的selector0；已閉合135DD／35822另68筆原指令獨立核對。修正後Game完整重播r1至原版堵路邊界，74筆AI全消費且零順序分歧。晚期來源測試fixture改從既有JOIN constructor物化持續名冊，語系與穩定地圖用正式helper補齊；環境／fixture失敗不列產品缺陷。r2原版已經正常觸發67死亡及68第二波，等待第三波與戰後保存。

原版r2及Game完整流程完成：初始34筆，66／68／70三波到40／46／64，第三波T9；T13後seq1035一次清敵34個存活敵人，seq1064成功兩次DOS寫入，seq1073返回選人。305筆AI全消費且零順序分歧；正式比較行為／節點／交易通過，整份22987 bytes SAV含checksum相同，SHA df50919111cccb37a00215dea248a069617aa222545d0f4493ab4b00dc5b819f。未錄影，frames advisory不宣稱逐像素。台帳登記25／30，下一章29，不在本章工作單位開新章。字串重新綁定98筆處置不變；完整Go回歸執行中。

2026-10-17收尾：原版與Game完整收據保持passed，13個正常回合／齊備mark視圖四座標相同。首次完整Go額外提供FDOTHER／DATO，揭露六項先前受來源環境閘門略過的舊fixture缺語系；補正式locale，另補第22章持續名冊後的正式部署、第29章冷讀選人素材與非同步END回復畫面。修正僅測試，不改正式規則，也不提升第29章章驗收。137項Python在專用Capstone映像及明示FD2_EXE後全通過；先前兩次容器分別缺Capstone及原版路徑，失敗日誌保留且分類為環境。全域死亡58事件、回合／死亡程式／33地圖同步check通過，固定回合待轉寫13保持。新增PAN candidate教訓後208條、91 guard通過。

來源fixture r7六項全通過，另外保留原生EXP整數byte限制：舊77.5改為77，不放寬正式sync。完整Go以相同FDOTHER／DATO環境乾淨r2重跑，原始org_game另加唯讀掛載。GitHub #225／#226依完整章收據結案，#199排至第29章；官方pull22項，最終render／verify收尾中。最終清冊與runtime修訂後清冊逐byte相同，review SHA3c58908df73ae0a80f3443892e57d6b5e80f2a0f92177df356c83edc461ebc80，98筆處置保持。固定工作樹47檔擁有權1000:1000，新增失效連結0，歷史封包兩項缺件保持；全樹root-owned2811、md目錄0、掃描錯誤0，README與原EXE雜湊不變。

最終完整Go r2於相同來源環境乾淨通過19套件，cmd/fd2 277.634秒；137項Python無失敗／略過，六項舊source fixture乾淨通過。官方worklist verify為22條、14項人工判斷、可能已完成0。所有FD2本輪容器退出，README與原EXE保持；暫存47檔無SAV／EXE／私人素材。提交推送後以真正遠端HEAD核對，並回填#225／#226、#199及#33。

## 2026-10-17 第29章護送與防衛系統流程

開工633d2e43，tracked工作樹乾淨，主機gh登入與官方pull22項成功，FD2容器無殘留。路由命中復古重製、原版對拍與文件職責，沿既有work研究區。原始來源為第28章正式實際SAV df509191，直接複製至新input及state覆蓋層，不追加強化。攻略第29章說明護送悠妮到石碑、四回合增援及三頭龍；ABI與規則以既有map28 IDA主證據為準。先做正常非城鎮LOAD／19人選擇短探針，再安排實際地形路線；章台帳仍25／30，沒有新matching或錄影。

第29章完整r1在round2／checkpoint192主動停止，未清敵。route_cost把敵軍當全程障礙，遠方封鎖列讓Yunie全無候選；這是驗證驅動錯誤，不是原版規則或Game缺陷。修正READY：未來成本按地形，當回合可達集合仍保留占用及ZOC，再由原版Enter裁決。r1計畫與收據保留。

勘誤：r1問題在checkpoint192辨認；實際停止命令完成於checkpoint238，仍round2、未清敵。r2保留同一AP5000槽，地形潛勢改正、正常接戰助手0與18、最多30回合；r1未計整章。
戰後局部回歸先缺原素材環境，再揭露呈現器舊prefix限制。補齊FD2_FDOTHER／原資產包，READY六筆event72插入契約先行後修改正式validator；post-frontier-r3乾淨通過，21種完整組間插入與錯序原子拒收、93筆正式戰後、舊76筆保存冷讀保持。正常動態順序仍以完整章收據為準。

原版r2在2026-10-16 17:24:26 UTC、checkpoint597／round3以143退出，oracle.log無錯誤，工具工作階段收到SIGTERM。分類為環境中斷，未清敵、不計整章。r3沿相同r2凍結計畫與AP5000槽重跑，改由獨立Docker容器內timeout6000s及退出紀錄管理生命週期。
第72號死亡事件兩次35822與第74號同一135DD／10B4E，READY先加入72後補PAN回交；三項局部回歸全部通過，35C18最後range_one保持。#224不因此結案。

戰後首個fixture失敗的原因勘誤：startNativeUnitDeathPresent只在非靜音且g.sfx缺sample3時拒收。direct-entry fixture未初始化音效，也未設FD2_MUTE；明示靜音後可繼續。先前籠統記為素材環境缺件，不代表FD2_FDOTHER或素材包確實缺檔；音訊聆聽驗收未提升。

原版r3超過12分鐘後穩定運作，前598份checkpoint JSON與r2逐bytes相同，已排除生命周期調整改變狀態。round7／8 Yunie卡16,46，raw26=0；前方17,46／16,45守軍尚在，助手仍選後方近敵。主動停止，未清敵、不計整章。READY修正正常接戰策略：每助手依原生MV，先前方敵軍、再距離與索引，無前敵才退回最近其他敵軍；不改原版規則。

r4原版在第30回合觸發75及首波74，3781／round31／84筆以控制器exit19結束，沒有戰敗、清敵或保存。接管shell時容器已退出，未保留原程序；FD2.TMP為207360 bytes工作檔，非完整SAV。r5延至40輪，新增選用正常貼身攻擊解阻，固定同一AP5000來源與seed4，避免再次等後方守軍。
Game r4在正常石碑wait seq3569因raw8來源缺件拒收，已列#229；MaterializeNativePersistentPartyRecord有來源，applyPersistentStats漏帶raw8，READY後補單行，拒收不放寬。

2026-10-17第29章追加勘誤：r5正常護送於T21到碑，T25物化93筆，T26抽79後一次HP清敵28筆，T27仍留戰場。暫停driver並保留原程序查證，確認結果表基址51B19的raw28表項51B89指向20B72；先前誤把單一表項51B51當基址，已由117E7實際call運算元修正。既有IDA完整131-byte handler與Capstone34指令相同，沒有205BE全滅掃描；state18／19／20須由三龍death73／77／78正常派送，單純HP0不設旗標。r5保留failed並以143停止，不計整章、不寫SAV。#230與chapter_result_contract先達READY，才補typed初始code0、三旗標code2及record0／1敗北優先，string9沿正式文字索引lower。r6沿同一AP5000來源及seed4重跑，全部增援與79後正常攻擊三頭龍，不再清敵；原版戰後35BBA自行清場。

工具檢查補記：186項Python回歸在Capstone容器通過但Pillow測試略過1項，已在素材容器補跑35項收據測試全部通過。新正常三龍控制122項driver測試通過。新增純文字canonical guard首次因掃描器未含.txt而失敗；只補.txt支援後同一檢查211條／94guard乾淨通過，不改證據marker。canonical重匯入先被過嚴的零warning斷言及絕對來源路徑拒收，屬操作問題；保留既有extensions.legacy並沿正式相對來源重跑，319節點ID保持。

r6於工具審查初期停止，finish_ui誤加move_unit已改回engage，新增直接接戰收尾測試後123項全過；r7同一plan r4及AP5000來源重新凍結，driver SHA9298cf88不再修改。Game r5負例前兩輪signal killed保留；較大12GB診斷容器取得實際失敗：seq2753的END在76 pulse期間重入，nativeEndTurnRecovery與pulse互等，峰值8792117248 bytes、cgroup OOM kill0。依READY #231補正式Update／END與重播控制權閘，原有frame budget不變。修後同源能完成，但79在END早跑、隨後13536清位，AI多出90／91；#232先依既有1A30B及35EE6達READY，再把79移到清位後。受控事件seed逐筆取原版入口40262／3223，分別得到27345／58716，metadata揭露，不要求整段內部骰序。負例r6乾淨通過963筆AI、零順序差異；原版仍缺三龍死亡旗標，不計整章。局部新fixture首輪跑入缺完整AI前置的mode2拒收，改成只驗本fixture的清位／pair交接；正式AI順序由同源長程重播驗證。

最後正式Go修正後再次重綁字串盤點，98筆候選與各處置內容／數量保持。原版r7實測單核及約400MiB，先前12GiB診斷容器已退出；改用兩CPU、8GiB、Go軟上限4GiB有界並行完整回歸，總容器上限維持12GiB，避免等原版重跑結束才發現舊fixture問題。

第29章原版r7完成：seq3342／T30，349動作／3343 checkpoint；三龍91／90／92依序於seq2986／3068／3258正常死亡，最後同seq寫20B93並進2548C。章內零state_injections、零HP鎖與零force-enemy-clear。post追加group9至94，record_question3323、正常保存3333有兩次成功DOS寫入，最後返回選人。首次Game完整1065 AI零分歧與整檔SAV相同，但3271多出勝利確認而節點失敗；#233先依main25E02／25E23達READY，再補可編輯auto-advance。auto-r2 production已進正確post，重播工具漏既有post分支而未消費劇情；修正由battle_start的正式on_win承接，沒有放寬gate或縮短演出。auto-r3乾淨通過349行為／351節點與strict保存，22,987 bytes含checksum零差異，SHA33c3d74deec1be3ca87b3754a26f2d5ee03799918e237958017be7518b164a6f。章台帳登記26／30，下一章30，本批不開新章。

完整Go r2十九套件已通過；最後auto-advance修正後完整r3另跑。191項Python零失敗／略過，回合與死亡同步check通過且固定回合待轉寫仍13，214條教訓／97guard通過。新增結果／旗標／phase指令另141筆對原始bytes及Capstone邊界相同，不與先前重疊區間相加作覆蓋率。隔離生成器scenario及battle節點逐值相同；最後盤點與上次清冊逐byte相同，98筆處置保持。沒有新matching編譯或錄影。

最後auto-advance與重播交接修正後，完整Go r3在相同FDOTHER／DATO環境乾淨通過19套件，cmd/fd2 205.181秒、exit0、峰值4468736000 bytes、OOM kill0。隔離生成器再次核對最終第29章scenario與battle節點逐值相同，preparation必出／前置欄位相同；正式執行來源雜湊全部相符。正式章收據與完整SAV一致結果保持。

生成器收尾檢查勘誤：新增整個preparation相等的斷言過廣，正式節點另含既有party_limit19，骨架未產生該欄位。未改正式來源；核對限於本輪必出／前置欄位，scenario與battle仍逐值完整相同。

GitHub #227～#233已依完整章與最後回歸結案，#199接第30章。官方pull／render／verify為22項、14項人工判斷、可能已完成0，標籤相符。53個異動檔擁有權1000:1000，66份canonical輸出雜湊相符，9個新證據／計畫入口與37條新增本地連結均有效。README、原EXE、父槽、章前來源、最終SAV與凍結driver雜湊保持；全樹歷史root-owned2811、md目錄0、掃描錯誤0。本輪無錄影、FD2容器無殘留。工作產物留在work且不加入版控。

## 2026-10-17 第30章正常接續與終局驗收

開工0d60760b，tracked乾淨，官方pull22項與主機gh認證通過，無FD2容器殘留。路由命中復古重製、原版決定性對拍與文件職責，沿既有work。來源第29章實際SAV SHA33c3d74d原樣接續，祖先作弊政策保持，沒有新增強化、matching或錄影。先驗正常LOAD、19人選擇與揭幕，不把短探針計整章。主契約DRAFT與既有終局canonical入口同批登錄00-index；專屬勝負與魔神連鎖先查既有證據，不假定清敵即過關。

第30章原版短探針exit0、94 checkpoint，正常20我方＋group0七筆，共27；入口camera16,17／cursor22,23／visible6,6／selector1。Game初輪提前加groups1..3成33筆，短鏈比較失敗；#234依READY修正資料及生成器，canonical319個節點ID保持，r2短鏈行為／節點通過，不計整章。錯帶--oracle-save到沒有保存動作的短探針屬比較命令問題，移除不適用參數乾淨重跑，正式strict未改。既有IDA9.4匯出299指令、5個表項及FDFIELD_088 ff5200原始bytes／hash核對；81 writer使82從休眠啟動，推翻56全33圖無serialized82的舊斷言。82、80與專屬結果各先立READY，才接正式consumer。




第30章完整原版r2到seq1050／T11／25975永久停駐，1051 checkpoint，四次81／82及三波兩人增援、五魔神正常擊倒、完整25757、selector30、2BCE5、27人蒙太奇與20段尾段均實際走過。章內零注入，DOS create／write零次；兩側SAV整檔與來源33c3d74d相同。Game full-r4消費85筆AI、零順序分岔，行為／獨立節點／交易及終局窄契約全部通過，frames advisory。原版18觀測點失敗及Game guard、名冊consumer、mode11漏回復各失敗樣本保留，READY先於修改；#237／#238由正常流程抓到，沒有新matching。

終局補回FDTXT031遺漏的六段以建立60句完整索引，四個caller共42句逐原始controls／glyphs重生。selector30的1筆模板與31格部署容量明示分開，27active／4原始清零格各有來源；2C548實際讀當前scene records，消費四欄與原版相同。原98項字串審查處置不變，新增1項loadErr契約診斷。完整Go回歸另記實際結果，不把fixture、預覽或綠色單元測試提升為整章。

最後Game full-r5再次通過，正式收據及台帳命令已指向此輪。相同FDOTHER／DATO與素材包環境的完整Go r2通過19套件，cmd/fd2 163.702秒、exit0、峰值4304441344 bytes、OOM kill0；相關Python183項及嚴格四語8項均通過，四包各5106筆結構驗證通過。生成器第30章scenario／battle／post逐值相同，整備必出／前置同為9；全域回合／死亡同步、終局字庫與216條教訓的99條guard均通過。68,022,299 bytes失敗log壓至1,778,761 bytes，解壓SHA相同，保留.gz。

GitHub #234～#238及父#199已結案；#239保留新恢復六句三語人工校譯，#200接第1～3章，HD待辦保持。官方pull／render／verify為22項、14項人工判斷、可能已完成0，遠端狀態標籤相符。66個異動檔擁有權1000:1000，66份canonical輸出SHA相符，22條新增本地連結有效，新證據／計畫入口有00-index路由。README、原EXE、父槽、章前來源及完整輪次凍結driver雜湊保持；全樹歷史root-owned2811、md目錄0、掃描錯誤0。本輪無錄影或matching，無非預期FD2容器殘留；work不加入版控。

## 2026-10-17 第1～3章接續驗收

開工961d9525，tracked乾淨，主機gh認證及官方pull22項通過，無FD2容器殘留。路由命中復古重製、原版決定性對拍、規格閘門與文件職責。#200要求目前正式引擎正常START到首章，既有ch01-clear計畫實際走CONTINUE，不能沿用為START收據。沿用已閉合開場／首章事件證據，先建DRAFT主契約與正常START短計畫並掛入00-index，不啟動matching或生成錄影。

正常START prefix-r2原版exit0、seq191、434516894步；正式Game由完整開場動畫／START及既有序章owner到首章可操作，19組來源call、97句、118次按鍵推進一致。首比較失敗只有record9：原版32917→32975寫raw+5=1而HP仍28，Game已正確停用；比較器與driver只看HP造成假差異。#240依READY補地圖caller接納條件，不賦予bit0全域死亡／存活語意。原失敗保留，prefix-r4三閘門通過但不計整章。187項Python通過，28／29／30既有同源收據由新比較器重比仍passed，不新增章驗收。

原版前綴r1的20M等鍵觀測及完整r1的每次攻擊140M額外settle均主動停止；兩者是測試計畫冗餘等待，沒有戰鬥失敗或重選seed。完整r2固定2M觀測、strike_settle1，仍由正常cursor／dialogue owner收尾，保留START、戰鬥至T7四個事件、111清敵及城鎮正常保存；開始後凍結driver與計畫。

完整Go19套件回歸通過，cmd/fd2 410.878秒、exit0；191項Python工具回歸通過。full-r2進一步發現工具預設MV6與原版raw+3B矛盾：四名角色為4／4／7／4，原先只歸因住宅地形不足，保留假說並由raw證據取代。#241新增明示native_move_budget及candidate_map，僅改測試候選，未選用的歷史計畫排序不變。新r3開跑後再次凍結driver及計畫；已停止r1另用正式Game核對首個物理及反擊，不以局部提升整章。

完整r3的raw預算移動已正常推進，索爾7,14→5,16等被原版Enter接受；先前超距重試問題已消除。但前線策略令索爾T2自然死亡，原版返回標題，exit15，該輪不計完整章。r4保留同一未修改SAV與固定起點，只正常將索爾／悠妮退到東側，再抽一次亞雷斯物理，T2索爾／蓋亞退守後正常END至T7。這是第二個自然戰鬥策略，不重選seed、不鎖HP、不做章內座標／狀態注入。

完整原版full-r4正常到T7／seq938，939點與55筆AI，四回合事件均到達；一次物理與正常END抽樣後依111清敵六筆，無章內HP鎖。Game r4首差是原版phase metadata缺漏，#242先READY後補四事件所有action camp；r5消除T3首差但T4新敵軍漏行動。#243沿原版constructor移除GUI額外Acted覆寫，r6／r8均55筆AI零分岔，行為／節點通過。保存首次形成共同狀態後才確定#244：START未建LOAD raw基底，酒店只寫JSON，原版SAV未變。

測試側DRAFT提案保留其他槽、目前戰況及metadata尾段，沿既有零基底與typed隊伍建構首槽。正式標題LOAD冷讀5人能力／物品、章節、金幣與選項一致；與原版實際SAV差2228 bytes，未使用名冊2160、有效紀錄65及檢查碼3。有效紀錄含未用到的AI／死亡欄位，不能全稱不透明；修正兩處原版新開局全零的錯誤註解。依grilling提出限本收據接受差異或繼續來源追查，等待使用者決定，尚未改production或章gate。191項Python乾淨回歸通過；前一命令列錯兩個模組名保留為環境紀錄。

保存政策決策追加：使用者先選逐byte相同，開始沿既有IDA工具鏈追查配置來源，未啟動新matching或新oracle；之後澄清若屬新增欄位則接受建議，再明確回覆「允許差異存在」。確認本案是同尺寸同ABI的既有欄位殘留值，將已展示2228個差異的限本收據方案改為READY。其他來源與章的嚴格政策保持，正式SAV不得抄oracle未知常數。

依使用者明示要求，matching輔助僅重連結既有Watcom10.0a START物件45變體，固定EXE／IDA9.4原始指令與fixup核對通過，G27的25EBB完整663 bytes相同。酒店30012沿用直接memmove2560及metadata十byte證據，完整C未匹配限制保留；65個有效紀錄差異均從同一首章battle首次人物紀錄原值保留到保存。

正式START初次酒店保存以明示可寫容器建立名冊基底，驗證成功及原子取代後才發布；LOAD仍保留raw來源。新增缺START／LOAD來源、壞容器零發布、非目標區域保留及重存決定性測試。冷讀測試重構少右括號造成r9／Go r3在gofmt即停止，修正後預檢成功並乾淨重跑，不當產品缺陷。Game full-r10 exit0，55筆AI全消費、零分岔，正式LOAD冷讀通過。SAV SHAf3942ff2與提案相同；strict仍拒收，明示首章政策後三必要gate通過，正式parity-ch01及台帳更新為28章passed，第2、3章todo。196項Python工具回歸含政策偽造拒收通過，matching契約另由專用CLI實際驗證，回合／死亡同步與218教訓101guard通過；Go r2曾19套件通過，新正式保存完整Go r4另記實際結果。

完整Go r4的18套件通過，cmd/fd2 325.058秒；字串清冊套件因來源位置ID及全清冊SHA過期拒收。最初猜為新增保存診斷，逐檔案／函式／文字／role／參數及重複次序對比後，證實文字多重集合完全相同，99筆go_review處置不變。只將原處置對應到目前行列ID並重生清冊與摘要，受影響套件另作乾淨回歸；不重跑已通過的18套件。

字串清冊套件修正後0.390秒通過，18＋1共19套件的相關回歸閉合；保留完整r4 exit1及修正套件的獨立log。66份canonical SHA相符，2387條本地文件連結有效，40個原異動檔UID/GID1000:1000，歷史root-owned2811、md目錄0、掃描錯誤0。README、原EXE、原始與章前SAV及凍結driver雜湊保持。未產生錄影；全部一次性容器已退出，工作輸出不加入Git。

首章程式與正式證據已提交並推送ace79b15，遠端HEAD核對相同。GitHub #240～#244以該提交與正式收據結案，#200留言保留第2、3章待驗。官方pull／render／verify更新為22項，狀態標籤相符；正式章台帳28 passed、2 todo。最終快照另提交，沒有新增遊戲變更或重跑已通過的章。完整清冊為ignored本機生成物，只提交可重生工具、摘要與原處置映射。

## 2026-10-17 第2章正常接續驗收

接手b38d96fe，tracked乾淨、無FD2容器殘留；主機gh認證與官方pull22項通過，現況28章passed，第2、3章待驗。路由命中復古重製、原版對拍及文件職責，沿既有work。攻略僅作T3六人增援、村民分支與希莉亞JOIN的旁證。沿第1章原版full-r4實際酒店SAV SHA540c3bc1正常LOAD，保留5人及三名HP0，沒有新增強化、復活、HP鎖或matching；先建DRAFT主契約、READY起手與短計畫，全部掛入00-index。

第2章原版startup-r1正常到seq76、21筆、camera9,13／cursor20,14；Game拒收源於計畫把2D11D出戰提示誤標大量選人後的departure_confirmation，改正新r2計畫並保留舊輪次。原版full-r2到T4、seq622、623點，53筆AI與4筆growth；event6一次，六村民raw35/36全為26,15，slot7自然死亡。正常蓋亞物理抽樣後111清敵，戰後希莉亞JOIN及下一城鎮保存成功，沒有HP鎖或新強化。

#245沿既有IDA9.4完整匯出160原始bytes補event6 PAN、ACT13、九句FDTXT002#4及六筆目的地寫入，先READY後降階；既有record_bytes補+36 typed setter及raw投影。起手實測村民mode4反證DRAFT導覽中的mode11，實際map1控制檔004反證002定位，勘誤均保留。純測試先遭舊spawn-only斷言拒收，再查明同步Scenario consumer不消費GUI原語，改測同一ApplyNativeDeathOp規則；三相關套件r3通過，未改正式adapter求通過。196項Python通過，全域尚未完整轉寫事件12筆。

Game完整r2起手名冊相同，第一次移動組圖因HUD來源缺失拒收。#246由原版cp76、33142 focus0與既有LOAD的26112 writer建立READY，再補battle_ch02 view及HUD繼承、生成器和canonical；未關閉renderer預檢。B=1僅用作已證非零顯示，最終raw B未直接擷取仍保留限制。r3同源重播驗證中，章台帳保持28。

Game full-r3正常重播通過53筆AI，零順序分岔；行為／節點passed，保存剩希莉亞+28..30九byte及兩checksum，共11差異。r4新增可明示啟用的正式保存冷讀驗證，實際SAV經新Game loadGame／confirmTitleLoadSlot通過六人能力、8格物品、金幣、章節及選項；strict仍failed，不提升章台帳。使用者要求以matching輔助釐清原因，新增#247，沒有接受第2章政策。既有scene-flow物件126變體重連結，112A5九個完整C候選皆負例；直接IDA指令顯示建構只清+22長6，將以相同SAV／計畫的唯讀memory-change窗口及正對照writer確認建構前殘留來源。205項Python工具回歸通過。canonical66份SHA相符，320節點ID全部保留；先前摘要319為抄錄錯誤，以實檔核對取代。

第2章來源查核完成：原版同一輸入的有界memory-change先讀到runtime27九個保留值，建構時三個正對照writer均捕獲，九byte無值改變。固定EXE／IDA指令核對及工具驗證passed；同輪最終SAV與正式原版d8d534d5相同。正式保存BuildNativeChapterSlot對同步後新入隊紀錄用零基底，與原版未初始化區保留不同；不是新增save欄位。局部副本補九byte重算checksum可重現整份原版，沒有寫回正式SAV。使用者在證據展示後確認本份例外；新政策逐筆綁定全部11差異，負例strict保留failed，正式parity-ch02三必要gate通過，台帳29章，第3章待驗。210項Python工具回歸通過。

數量勘誤：第2章完整原版最終current為seq625／1,698,395,489 steps，共626個checkpoint JSON；早期seq622是保存後較早一點，不是最終停止邊界。主契約已依直接收據修正。完整Go首次18套件通過，cmd/fd2在預設10m的T.Run邊界逾時，依環境分類只重跑該套件、18m上限及json逐項紀錄；未把逾時列為產品缺陷。死亡同步首次用了不存在的舊工具名，改用sync_native_death_programs.py後同容器乾淨通過。擁有權掃描歷史root-owned2811、md目錄0、錯誤0；2343個本地文件連結有效，README／EXE／來源SAV／凍結driver雜湊保持。沒有新增錄影。

依使用者確認接續第3章：攻略只作T3兩端援軍及鐵諾存活JOIN的旁證；起手沿第2章實際原版SAV d8d534d5、六人及三名HP0，零新強化／復活／HP鎖。兩次Left出口由上一章正常town_ch03 probe seq527佐證，先READY起手及DRAFT整章規格，清冊／短計畫／主契約同批加入索引。

第2章完整Go回歸閉合：前輪18套件已通過，唯一逾時的cmd/fd2同工具鏈乾淨重跑643.058秒、exit0，18＋1共19套件。第3章正常LOAD短前綴原版exit0、seq68、15筆；Game完整章前同輸入7.84秒通過，零AI入口抽樣，僅起手E1，不提升第3章台帳。

第2章已提交並推送ab35f6b9，遠端HEAD相同；#245／#246／#247依正式收據結案，#200仍開，第3章待驗，官方快照22條。開始第3章完整r2：正常索爾兩次向橋中退守、蓋亞接戰及三次END，預期T3援軍後27筆，再依111清敵。鐵諾分支由實際狀態決定，無HP鎖或新強化；既有起手cp42..43提供slot6建構窗口，同輪唯讀記錄九個保留值，避免若保存再有差異時重跑整章。

第3章r2在T3已到27筆、索爾29／鐵諾43HP；之後額外第三次END使索爾敗北，最早標題owner為cp365。driver的ally_alive讀到標題留下的舊unit_count與已重用陣列，不能當戰鬥HP；停止無效等待並保留r2。新r3同一SAV／seed／兩次END，在T3援軍抵達且可操作後依111清敵；無HP鎖或新強化。這是有界操作策略調整，不改正式規則或凍結driver。

#248原因勘誤：r2 driver.log在seq341已把round等待判為離開戰場；實際耗盡600次的是後續await_ui cursor。cp365已1FE60標題仍持續至941，exit15。舊raw確已重用且不能判HP，但先前將整段延遲直接歸因ally_alive不成立。Issue內文依直接log修正，原敗北／舊解釋仍留存。


第3章完整r3原版exit0、seq435／1,315,400,367步、436 checkpoint；正常兩次END後T3事件9已到27筆，交回操作即依111清敵20筆。鐵諾存活JOIN、下一城鎮五類服務、酒店保存及返回城鎮通過。成功原版維持f50bd7d4凍結driver，無HP鎖、新強化或重擲seed。既有event9正分支可消費，全域仍12筆待完整轉寫，不新增matching。

#249依原版cp68及LOAD26112既有writer先READY，再補battle_ch03的view／HUD繼承、生成器與canonical；正式移動及完整Game full-r2通過14.79秒，18筆AI全消費、零分岔，正式LOAD冷讀7人／金幣1000。完整strict保存22987 bytes含checksum相同，SHA b942e356f2babb2d936fe7d2daa461f1270175835619f0e59ea4bf43b4ddcd40。戰後先SYNC再JOIN，沒有第二次SYNC；NativeJoinPersistentPending沿同一LOAD基底第6筆，不能套用第2章新入隊零基底結論。正式parity-ch03三閘門通過，工具更新台帳30／30，畫面仍advisory，不推定所有分支、自然難度或HD完成。

#248在成功原版結束後才補do_await_ui guard：wanted UI先接納，title且kbd_pending為整數0即失敗；有待處理鍵、缺metadata及一般對白沿既有界限。cp365真實負例零送鍵即失敗，r3 cp43..68的25筆正常鍵序保持。首fixture省略EXE hash／eip，三個portrait等待分類失去版本脈絡，補齊原始context後斷言通過，未改UI分類或降低斷言。完整215項工具回歸r3只有worklist現況測試失敗：30章已過但#14仍開，屬結案同步，提交後關閉並更新快照重跑。

提交前驗證：正式第3章比較再次三閘門passed；137項driver回歸、219條教訓／102guard、全域回合／死亡同步均通過。66份canonical輸出SHA相符，30份正式章收據均passed。本批只有資料與Python工具變更，既有19套件完整Go及本輪正式Game全章回歸已覆蓋，不重跑無變更Go。全樹連結檢查發現9筆歷史舊交付目標或程式語法誤判，本輪修改文件及README沒有失效目標；保留原歷史文件。初次衛生腳本因素材image未裝git而停止，改由主機git提供唯讀NUL清單，不在主機執行分析。

最終衛生掃描：20個本批提交檔均UID/GID1000:1000，歷史root-owned仍2811，md目錄0，掃描錯誤0。全樹3844個本地目標中保留前述9筆歷史問題，本輪修改文件零失效目標；README文化保存內容與圖片保持，原EXE／第3章來源SAV／實際保存SHA相符。沒有錄影，沒有非預期FD2容器殘留；work不加入版控。

第3章主批36ea3023已推送，遠端HEAD核對相同；#14戰役總項、#200第1～3章及#248／#249依提交與正式證據結案。官方pull20項，render／verify通過且可能已完成0；相同容器乾淨重跑215項Python工具回歸全部通過，原r3的遠端同步失敗保留。第3章及30章台帳完成，現代素材#201、呈現#202與HD#204維持未完成，完整matching #198保持暫停。


## 2026-10-17 正式第一版三平台發行

使用者明示先發布1.0.0，指定macOS、Windows ZIP及Linux AppImage。路由載入版號、dist-all與Windows ZIP契約，完整識別v.1.0.0-20261017；歷史tag不刪不移。公開包沿既有權利邊界只含引擎與可散布資料，HD及實體桌面缺口明示保留。先建立41的READY契約，再補共同stage／verify／attest／collect、三平台原生CI與最後Release發布job。沒有修改遊戲規則或save格式。

actionlint1.7.7語法檢查及6項封包負例回歸通過。初次素材容器單元測試因沒有git而停止，將Git HEAD視為來源比較測試的外部輸入，真實HEAD仍由流水線核對。Linux候選第一輪授權清單取go list -m all，包含未被程式引用、未下載完整source的圖節點而缺Dir；改由go list -deps -json=Module取得實際linked module來源，不刪除必要授權文字。原候選只含stage，沒有正式封包，清除本輪明確部分輸出後同工具鏈重跑。

Linux本機候選已通過：同一Docker image、447份公開資產、實際AppImage從空白cwd進行本版本自檢，沒有原版／私人素材。候選留work且來源工作樹尚未提交，不作正式Release收據。另補type2-runtime固定版本MIT全文及SHA，macOS兩架構的實際模組授權取聯集；正式CI將從乾淨提交重建。

預檢cf9c5cd2的macOS建置步驟失敗，未建立正式tag或Release。公開API可讀狀態與註記，但job log端點回403，未搬運主機憑證；先在受版控macOS流程增加僅含本步驟stdout／stderr的失敗摘要註記，再由公開API讀直接原因，不猜修SDK或規則。Linux候選實際預設啟動已抽frame120，畫面為story_ch00_handler演出，無關卡注入，未用此單張推定完整START。

macOS直接原因已由39041602077的原生註記取得：兩個slice及lipo create成功，Apple lipo將位於-verify_arch後的檔名當成架構旗標，usage明示lipo <input_file> <command>。修正為輸入在前，維持雙架構及後續Mach-O驗證，沒有改SDK、遊戲或放寬gate。cf9c5cd2的Linux／Windows原生建置、自檢及封包均success；正式三平台同來源收據仍待修正後新提交重跑。重複失敗後回查打包路由與封包四層驗證，分類為驗證命令問題。

正式預檢39041776347全數通過後，main快轉至d8c361a8並推送，建立輕量tag v.1.0.0-20261017。Git metadata在sandbox唯讀，改以已授權的unsandboxed Git控制面寫index／refs，沒有覆寫或移動舊tag。正式run39042142693三平台、聚合及發布全success，22:48:16台北時間公開Release 415858640並設為Latest。

受版控verify_fd2_release.py回下載三包至dist-all/v.1.0.0-20261017/patch，與API digest／manifest大小及SHA完全相同。Windows 6,660,970 bytes、macOS 11,329,216 bytes、Linux 6,765,048 bytes；ZIP CRC、UTF-8／BOM／CRLF、兩種Mach-O架構、GUI PE、授權及各447份公開素材全部通過。實際回下載Linux AppImage解包內容及空白cwd自檢亦通過，沒有原版或私人現代素材混入。正式receipt及重跑工具同批加入00-index與41，58新增發行列。未宣稱HD、自然難度、所有分支、簽署／公證或實體平台完成；#9維持實機抽測範圍。

索引檢查勘誤：41原有入口使用打包標題與頁碼，不能只搜尋完整檔名判定缺入口。核對既有標題入口後，同批補上直接可點的41及正式收據連結，沒有新增第二份職責重複文件。

發行收尾依版號契約將工具regex的數字限定為[0-9]，加入全形數字版號拒收案例，6項回歸通過。已發布的ASCII版號及所有附件不受影響，tag／Release不移動、不覆寫。

最後衛生檢查：異動檔與正式交付均UID/GID1000:1000，1496條異動文件本地連結有效。歷史root-owned仍2811、md目錄0、掃描錯誤0；原FD2.EXE SHA保持。清理本輪候選AppImage及stage共38,335,421 bytes，保存SHA／清理紀錄與必要PNG／文字收據，正式三包保留在dist-all。沒有產生錄影；未留下FD2一次性容器。

發行主證據與可重跑工具已提交6ac5616b並推送main；原始正式tag仍固定d8c361a8。GitHub #9回填第一版及三平台原生／回下載證據，保留實機啟動、存讀檔與音訊驗收條件及開啟狀態。官方pull／render／verify仍20項、13項人工判斷、可能已完成0；HD與完整matching的既有狀態保持。

## 2026-10-18 第一版非HD完整包與實際遊玩推廣片

使用者要求從正常開頭、第一章到城鎮購物並介紹技術，完成後另打包三平台非HD完整版。命中推廣影片、原版素材權利、dist-all及Windows ZIP路由，使用reverse-engineer-retro-game-remake技能；公開版號沿已發布v.1.0.0-20261017，發行日與tag來源d8c361a8保持，不更動公開Release。從main建立promo/full-local-v.1.0.0-20261017工作分支。

沿既有Go／X11鏡像補可重現fd2-promo:20261017-r1，替代缺少的歷史game-video工具。正常開場、普通Space略過可略過動畫及START後，沿序章前進到第一章。X11普通鍵盤選亞雷斯、移動、攻擊及反擊；未強化、鎖血或修改敵人。戰後明示以已驗證首章hotel-save.json透過F9接續，普通商店操作買藥草兩件交給索爾，金幣1000→980、道具192新增兩件，F5保存前後核對。這是媒體／重製玩家操作證據，不是新原版同狀態或整章連續自然通關收據。

成片138.035秒、1280×720／30fps／H.264與AAC，SHA aabbf42724f458bf720d14926eee4523dd0d6bcf90f65210434153f91cd5aa92。技術字幕介紹可編輯流程、dosgolem受控驗證、AFM／1824字模及Go／Ebiten三平台；明示尚未HD及城鎮存檔接續。完整解碼及16份抽樣圖通過，字幕無裁切。未調整AAC版本的戰鬥音效峰值達0dB，降低成片3dB並保留視訊stream copy後，平均−20.7dB、峰值−2.7dB。2.5秒遊戲區黑幕為實際反擊後的返回轉場，對話／清單停留與原作靜音窗逐項記錄，不宣稱零黑幕或人耳音色已驗。

三平台完整包沿已發布的相同執行檔，補齊合法原版素材，assets與pack分開。每包43177份素材，原生Windows三份README保留CRLF差異，其餘43174份三平台相同。Windows ZIP 130497727 bytes、macOS通用ZIP 137579176 bytes、Linux AppImage 99240440 bytes；全部逐檔SHA、ZIP CRC、所有非ASCII UTF-8旗標、README BOM／CRLF、GUI PE、雙架構Mach-O、bundle、啟動器與授權通過。實際完整AppImage及Windows ZIP從空cwd自檢並以預設場景擷取通過；Windows為Wine輔助，兩張PNG SHA相同，不推論原版對拍或平台逐幀一致。macOS完整包未執行，#9實機輸入、存讀檔、音訊維持未完成。

環境與腳本勘誤：首輪Docker build旗標不適用buildx，改用專用Dockerfile.dockerignore及既有base；首次有界image匯出到時停止，快取重跑完成。1024個未壓縮影格緩衝下初輪Xvfb／FFmpeg被SIGKILL，沒有該輪cgroup計數，不確定OOM來源；縮至8幀、限制繪圖執行緒後約1GB且OOM計數零。工作樹README不能覆蓋已發布公開基底；Windows三份原生CRLF以平台SHA覆蓋清單精確登錄，不改Game資料。Wine先預建使用者擁有的prefix，再固定C.UTF-8處理中文BAT參數，實際原ZIP啟動器通過；不把Wine環境錯誤列為產品缺陷。跨bind mount的rename回EXDEV，改為複製、核SHA、刪自有暫存；未改正式視訊內容。

全部交付在dist-all/v.1.0.0-20261017/，原公開SHA256SUMS保持，新增DELIVERY-SHA256SUMS列三份公開包、三份本機完整包及影片。完整版與媒體含原版衍生素材，只留本機，不入Git或公開Release。主收據、工具與入口同批登錄00-index及41，58只更新交付列；HD、完整matching與既有20條worklist狀態保持。

驗證後核實檔SHA，刪19份暫存錄影／剪輯共103752637 bytes；正式影片與必要圖、普通輸入、保存前後及音畫收據保留。完整包暫存另移除604960924 bytes，正式三包及原始來源保留。最終Git差異、所有權、連結與Docker清理在本輪收尾核對。

最終核對：七份正式交付物SHA及UID/GID1000:1000通過，1518條本地連結有效，所有新Python語法與兩支shell語法通過。原FD2.EXE SHA保持222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f。222條教訓、105個guard通過；正式worklist仍20項、13項人工、可能完成0。歷史root-owned仍2811、md目錄0，所有本輪FD2容器已退出；舊IDA映像4ac62de83339不屬本輪，不作全域清理。完整包暫存實際移除604960924 bytes，加錄影共708713561 bytes；正式四個新增成品與三份公開包保留。dist-all被.gitignore排除。

本批工具與交付主證據已提交3ddee599並快轉／推送origin/main，遠端HEAD核對相同；正式v.1.0.0-20261017 tag仍為d8c361a8。GitHub #9回填完整包及Wine／Linux抽測證據，留言6236383549，保留實機驗收條件與開啟狀態。交付收據另綁定工具來源提交，未改任何正式成品bytes。

## 2026-10-18 推廣片節奏、序章頭像與1920×1200切片

使用者要求與DOSBox比較影片卡頓，補四語推廣，接續HD。從main 61d2651a建立promo/hd-intro-20261018。先恢復現況、遠端21項worklist與既有Docker工具鏈；全檔matching保持暫停，本輪未改dosgolem原版執行器。路由載入重製／規格閘門、推廣FFmpeg、文件職責與玩家路徑入口。

推廣片直接擷取已發布Go/Ebiten引擎，dosgolem參數不在這條流水線。單FFmpeg同時讀X11與Pulse的代表片段為16.531至18.225fps；拆成獨立音畫後為35.224至49.351fps。DOSBox-X normal core／固定12000 cycles／nosound的輔助片段59.887fps，維持原版來源SHA。比較為相近騎士構圖，未同步動畫相位，不宣稱same-state或硬體wall-clock。完整量測與限制保存於fd2-promo-cadence-20261018.json。

新片136.035秒、1280×720／CFR60，SHA 478d68ff9a4b56254776101d989a52bb86ece37ca48f9e9e286e384fa49c0c42。正常開頭與START、序章、首章普通亞雷斯移動／攻擊及反擊後，以明示F9接戰後槽至城鎮，兩次藥草購物令金幣1000→980、道具192新增兩件。F4實際顯示繁中、簡中、日文及英文商店。全解碼、音畫時長、平均−20.6dB／峰值−3.0dB通過，無超過門檻的黑幕或靜音；八段低變化停留為AFM、標題、對話／交易等鍵或技術字幕。21張中點與字幕邊界已目視，未宣稱人耳驗收或每秒60個不同畫面。12秒DOSBox並列短片SHA 962ae1c06a8fbc4619007ccc8711367743a861c5aef5d307c34b3618b8835d8f。原正式影片aabbf427與公開Release／tag均保持；DELIVERY清冊另列新兩片。

正式序章consumer稽核97筆，需DATO 0／1／4／9／30／48／66／96；缺48父王、66王后、96強盜。正常現代START先在speaker48拒收，補齊後以原正式第一版引擎正常97拍到首章通過。三張1254母稿及80衍生圖經catalog登錄，私人庫fd93f14f已推送並核遠端。原版身分、母稿、產生提示、正常輸入及GPU來源見fd2-modern-portraits-20261018.json，不把母稿存在稱為高解析完成。

使用者先確認父王手繪風格，之後撤回160方案。Ebiten同場景原型比較1280×800／320與1920×1200／480，使用者明確採1920×1200與480×480。先寫61 READY契約，再從31位既有母稿衍生480圖，保留80圖與舊包。新角色集合、契約、SHA、透明度、幾何或部分集合不符即原子拒收。高解析包啟動時使用Ebiten v2.6.6 FinalScreenDrawer，保留640×400遊戲與輸入座標，直接在最終畫面繪製480圖並快取GPU圖片；逐幀清呈現描述，保留幫助、人物資訊與全黑／白遮罩及淡出。背景、字型、戰場及由忠實啟動後F2動態進入HD仍未完成，#204保留。

錄影首輪在人物資訊仍開著時F9讀酒店，節點變更但舊資訊全畫面殘留，新增#250。LOAD成功發布處清nativePlayerStatus，驗證失敗時原指標保持。原有存讀檔回歸先紅後綠；GPU建構槽普通Home→F9酒店→Escape城鎮通過，限RUNTIME-E1，非正常開頭完整E2或原版對拍。重製正常購物片改在資訊完全關閉後讀檔，沒有以後製掩蓋缺陷。

驗證工具與環境勘誤：Go容器login shell會重置PATH，改bash -c。PNG時間戳不能比字幕內容，ImageMagick的%[signature]在此版本回空值；改%#並要求64位實際像素簽章。FFmpeg concat清單相對路徑會重複目錄，改絕對路徑。未交付新片的載入黑幕已剪除，舊正式影片未動。Bash在執行中讀取來源被修改導致EOF，EXIT trap已合併有效影音；擷取器改啟動時複製不可變副本並記SHA。

GPU單元測試不能在RunGame前ReadPixels；改驗真實Draw提交的幾何與細節倍率，真正像素另以正常視窗擷取對清冊480圖MAE核對。Go測試的浮點矩陣用1e-6誤差，不把浮點捨入當產品幾何缺陷。新GUI工具先遇Pulse設定目錄不可寫，再把原版pack錯當公開assets根目錄；以XDG設定目錄及分開的assets／FD2_ASSET_PACK修正，沒有改遊戲規則。故事line在不同caller重起，只用story／line會取錯角色；取樣鍵改保留原始handler_address並拒收重複。中止的不完整輪次不作玩家通過證據。

相關Go回歸含完整私人包、舊包相容、HD缺件／孤兒／重複／壞SHA／尺寸／透明度、上下錨點、細節倍率、GPU快取與遮罩，最終4.340秒通過。Python清冊195項與1187個file／files通過；31張480候選已按allow-list準備私人同步，待完整序章結果後獨立提交。正常1920視窗已確認索爾、父王及王后480區域MAE=0，完整97拍仍執行中。最終收據、私人提交、媒體清理與公開提交於本節後續追加。

高解析最終正常輪hd-normal-r4通過97拍至battle_ch01，七位頭像480區域MAE=0，含父王、王后與強盜；GPU圖已目視。61提升本切片CONFORMED／PLAYER-E2-LIMITED，正式來源與時序觀測綁fd2-modern-hd-20261018.json。F2由忠實啟動後動態進HD及整體介面／戰場、跨平台效能仍未完成。

核對九份現行交付SHA／所有權與不可變原版EXE通過。本輪40份暫存音畫與剪輯426217972 bytes已依使用者要求刪除，正式三片、PNG、操作、存檔與雜湊清冊保留。新HD流程沒有建立錄影。受影響文件2590條本地連結零失效；無範圍全庫掃描額外包含work歷史樹與已歸檔dist，64項既有失效不作本輪產品缺陷。226條教訓及109個guard通過。

私人HD素材獨立提交3f289e7bad8dae340d803f0176843fb2c3b3ea70，推送origin/main後ls-remote完全相同；33個變更檔只含31張480圖及catalog／manifest。1313份檔案SHA與UID1000核對，舊164筆清冊語意不變。

收尾所有權掃描：歷史root-owned檔案2786個、md目錄0個，本輪新檔與私人同步均UID1000；不修改歷史root檔。兩份開發執行檔驗證後刪除37718792 bytes，來源與雜湊留收據。所有本輪FD2容器已退出，未動其他專案容器或映像。

收尾所有權自檢限本輪工作根work/promo-cadence-20261018：root-owned檔案0、md目錄0。全樹掃描已自行完成，結果為2786個root-owned檔案、md目錄0；這次只計檔案，不把歷史不同範圍的2811數字混用。兩份開發執行檔驗證後移除，雜湊與大小留高解析收據。所有本輪FD2容器已退出，未動其他專案容器／映像。提交前受影響文件2595條連結、正式來源SHA與舊161筆清冊語意核對通過。

本批實作與主證據提交55af16c5ecbc8963b102c8283575aa7837a58c67，從promo/hd-intro-20261018快轉main並推送，ls-remote核對相同。原v.1.0.0-20261017 tag／Release未改。#250已依該提交及GPU／回歸證據關閉；官方pull更新為20項。#201／#202保留其他角色／章節與操作驗證，#204改為1920×1200正式入口及後續HD，未重開已驗序章。

## 2026-10-18 F2動態高解析與忠實縮放

由main b2716f8c建立hd/f2-runtime-20261018，沿使用者已確認1920×1200／480×480接續61的READY切片。
所有啟動主題共用最終呈現器；忠實模式保留鎖定Ebiten v2.6.6的整數、非整數、縮小及關閉濾鏡契約。
改寫檔保留Apache署名、修改說明與完整授權，既有封包器仍保存原模組LICENSE。

F2原子預檢目前已解析speaker，僅替換素材與目前頭像，不重建對話或重設文字進度。
視窗模式切入1920×1200，切出恢復原大小；成功預檢的素材於同一程序快取，往返不重複解碼。
相關Go及完整私人包回歸通過11.251s；文字盤點與多語系分別2.285s、2.333s通過。
99筆既有審校只搬座標，從審校原提交ace79b15重建舊盤點後遷移，處置數保持。

正常忠實START途中F2走完97個對話拍點至首章，七位480區域MAE=0。
實際遮罩、視窗大小還原、1600×1000與1920×1080縮放通過。
同份首章JSON以忠實／HD兩次F9載入，單位結果相同，保存SHA保持。
F5沿既有節點保存契約，沒有改為戰中快照。五種實際GPU濾鏡比較均為全畫面零像素差異。
來源、入口、診斷、輸入、截圖及SHA見[F2主收據](docs/data/fd2-modern-hd-f2-20261018.json)。

前兩輪工具在同步初始化時漏送F2或提前讀取舊收據；第三輪完成97拍，卻誤以章內暫態比較節點LOAD，
且說明截圖早於GPU呈現。工具已改為等待完成按鍵收據與遮罩像素變化，第四輪乾淨通過。
沒有放寬正式遊戲或資料規則。

最終LOAD畫面另確認非原生對話仍取80×80現代圖，未接480最終層，沿#202接續。
說明標題與首列重疊另開[#251](https://github.com/wicanr2/fd2_re/issues/251)。
#204仍追蹤介面／字型／戰場及平台驗收，不以本切片關閉整體HD。
私人素材未變，遠端main仍3f289e7b且乾淨。正式第一版tag／Release不變，本輪沒有錄影。

提交前遠端Issue已pull／render／verify，21項開啟、14項人工判定、0項訊號消失。228條教訓與111守衛通過。變更文件2580條本地連結零缺件，13份已測來源SHA相同；本輪44,155,968 bytes測試執行檔已移除，無錄影、root-owned新檔或.md目錄，無FD2容器遺留。

## 2026-10-18 非原生對話HD與首章JSON冷讀

從main d230cfbf建立hd/dialogue-final-20261018，沿61既有1920×1200與480圖源授權，先寫READY再接Game.Draw非原生穩定對話。保留既有168邏輯區域、下框鏡像與上框方向；在1920視窗實際504區域直接取480圖源，原生480區域不變。舊80包、忠實模式、開收框與最終遮罩保持，沒有變更規則、存檔或私人素材。

完整私人包及HD／主題／存讀檔Go回歸2.381秒通過。最終四語盤點、審校與在地化三套件回歸8.831／0.195／0.252秒通過。從base重建盤點遷移99筆既有審校，四語既有譯文／狀態／變數保持；5106→5107來源條目只新增一個已譯相同文字別名，無新譯文。本輪未宣稱新的四語GUI覆蓋。

受版控GUI工具以此前正常97拍後F5的原樣JSON複本，兩個新程序普通START後F9冷讀首章。normal-r5在63.014秒完成F2雙向、F1遮罩、1600×1000／1920×1080與還原、F9重讀及HD啟動冷讀。六個GPU504區域最大正規化MAE為0.00162285，舊80放大參考為0.0251632；獨立參照以Triangle取樣，與GPU線性取樣差異明示，不宣稱零像素或原版parity。實際HD冷讀、忠實還原及遮罩／黑邊已目視。存檔SHA保持089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2；主證據見docs/data/fd2-modern-hd-dialogue-20261018.json。

工具r1／r2與r4的短按鍵或舊audit判斷造成漏ACK、多確認或GPU上傳時漏按；改為按下等實際ACK、finally釋放並等後續Update，不改產品。r3實際發現首章F5 JSON無正chapter，標題nativeLoadSlotMetadata拒收，同份F9可讀。新增缺陷#252，保留正常存檔而不猜填章號；本輪冷讀限START後F9，不能宣稱標題LOAD通過。#251說明間距維持待修。

61提升本切片CONFORMED限首章下框PLAYER-E2；非原生上框僅幾何E1。58／56／57與索引同步，其他章節、介面、字型、戰場與平台效能仍待驗。沒有新錄影；本輪開發執行檔已核SHA後刪除，大小與來源保留主收據，既有work與正式發布不變。

收尾官方pull／render／verify為22項、15項人工判定、可能完成0；230條教訓與113個守衛通過。2588條異動文件本地連結零缺件，13份正式來源與11張GPU截圖SHA核對通過。本輪18887184 bytes開發執行檔已移除，無新增root-owned檔案或.md目錄。私人素材庫乾淨，遠端仍3f289e7b；未留下FD2一次性容器。

## 2026-10-18 首章JSON標題LOAD修正

由main a24ed3c7建立fix/ch01-title-load-20261018。命中規格閘門、正常玩家存讀檔及文件職責路由，先追查saveGameToSlot、loadJSONGameFromSlot、nativeLoadSlotMetadata與確認owner。首章handlerChapter零值經omitempty省略，F9可正常恢復；標題誤以原生文字框metadata投影作為JSON讀取gate，並忽略void包裝背後的實際失敗結果。證據與READY契約先保存於docs/data/fd2-json-title-load-20261018.json，沒有重開原版SAV逆向或全檔matching。

標題JSON改與F9共用實際還原結果，成功才清titlePhase；非法槽與章號、未知節點及原有隊伍／選項拒收保持在選槽。共用JSON讀取者在發布前拒收超出0..255的章號。首章零值、writer、schema與原生SAV交易保持，原生槽metadata仍只負責既有投影；首章沿現有在地化四槽畫面。

新回歸在原程式重現零章號拒收與未知節點失敗後離開選槽。正例手建Game缺官方語系，改最小場景仍失敗；查看loadErr並依既有attachOfficialLocale補齊測試資料後，最終四套件24.099／0.005／0.531／0.748秒通過。覆蓋JSON保存／拒收、原生槽與檢查碼、HD切換及在地化。文字盤點重綁僅三筆來源座標，99筆審校及四語各5107筆譯文／狀態／變數完全保持；無新增譯文。JSON輸出依既有sort_keys格式，沒有整檔格式重排。

受版控GUI工具新增--load-entry title，normal-r1在178.953秒完成忠實與HD兩個新程序普通標題LOAD、F2往返、F1遮罩、縮放與F9。普通F5確實重新寫入，再由第三個新程序普通標題LOAD冷讀。原樣來源、重新保存及冷讀後完整JSON bytes相同，SHA均089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2；六個HD區域MAE均0.0016226，插值近似限制保持。實際選槽、HD與第三次忠實冷讀PNG已目視；只提升首章JSON玩家路徑，未新增原版對拍或跨平台實機聲明。

56／57／58／61、README與索引同步限定現況，較早只驗F9的收據不回溯提升。#202／#204保留其他玩家路徑及介面、字型、戰場與平台驗收，#251仍待修。沒有新錄影；本輪18887224 bytes開發執行檔核SHA後已刪除，PNG、普通輸入及存檔保留。

提交前2593條本地連結零缺件，14份正式來源與15張GPU截圖SHA通過，來源與重新F5保存完整bytes相同。230條教訓與113個守衛通過，異動檔UID/GID1000:1000，本輪工作根無root-owned新檔或.md目錄。私人素材庫乾淨，遠端main仍3f289e7b，沒有FD2容器遺留。

修正與玩家主收據已提交8f89be9115f57fc0baca347a5610b38045f863b1，快轉推送main後與遠端核對相同。#252依該提交關閉，#202／#204回填限定標題讀取結果；官方pull／render／verify為21項、14項人工判定、可能完成0。生成工作清單9條本地連結通過，較早的#252負例與歷史pending紀錄保留，現行入口已指向新收據。主契約另綁定實際來源提交，不更動已驗程式、存檔或GPU產物。

## 2026-10-18：HD 清晰黑體主路徑與 README 四語截圖

使用者選定 A 清晰黑體。一般介面、原生故事逐字／捲動與商店文字接最終1920×1200字級；商店金幣、價格與效果數字沿原目的座標直接繪字。Noto TTC依已驗證family選TC／SC／JP；忠實模式與舊包保留。原生固定面板、轉場及圖示內點陣字樣尚未接入，#204保持開啟。

逐步字形差異測試揭露捲動快照與後續寫字共用next緩衝區，已改獨立快照，跨FFFD全部步驟測試通過，缺陷登錄#253。說明頁標題間距與寬度適配修正#251。正常START四語首句、F2往返、F9接續旅店後城鎮／商店及單筆購買1000→990、三種視窗尺寸與最大字級通過；首章標題LOAD、F5／F9及完整保存bytes保持。主收據為docs/data/fd2-modern-hd-fonts-20261018.json。

cmd/fd2、campaign、fdother及localization回歸通過。最初未指定素材包的失敗與非標準4GiB程序被終止保留為環境診斷；依既有8GiB／4CPU契約重跑cmd通過。GUI等待與單筆／兩筆購買期待值修正後，同建置完整重跑通過。四語各5107筆內容、譯文、狀態與變數與base相同，僅遷移來源座標。

README新增四語忠實像素商店截圖，明示重製執行期與F9接續來源，不冒稱DOS原版新增語言。2519個本地連結、文化入口與四張圖片SHA核對通過；231條教訓／114個guard通過。未建立錄影，刪除本輪暫時建置與字型副本，保留PNG、收據與輸出雜湊。正式第一版tag保持，私人美術未變。


## 2026-10-18：原生旅館固定文字 HD 接續

從 main ceb1dc5c 建立 feat/hd-native-facility-fonts-20261018。命中規格閘門、正常玩家操作與文件職責路由；旅館提示、四槽及服務回覆沿既有FDTXT字模合成器，61先加入READY契約。字模包裝只用於當次Draw，保留驗證、背景填色、未知與歧義字形回退。槽框覆蓋提示時清除提示文字描述，共享字模及原始素材不變。原生旅館維持既有文字，沒有新增四語翻譯或改服務規則。

最初字模驗證、原字模不可變、旅館共享資源還原與槽框遮蔽測試通過。四語各5107筆以相同身分重綁，沒有新譯文。hotel-gui-r1與最初完整回歸由代理停止，未列通過；較密觀測先在r2、r3揭露連續Escape沒有新JustPressed，等待ACK+2或放鍵後重新取得frame都不足以證明已處理放鍵。重查實機路由與鎖定Ebiten API後，唯讀收據加入JustReleased事件；GUI須拿到同鍵放開收據才繼續。FD2_INPUT_AUDIT_INTERVAL只調整觀測密度，未設定仍120；遊戲時序不變。失敗樣本與既有教訓保留，不列產品缺陷。

旅館r4機械保存檢查通過，但實際GPU保存回覆重疊，拒收並登錄#254。新增去重斷言在saved的[24 238]重現；清除服務回覆重建前的提示描述後，最終focused回歸cmd/fd2 1.950秒、fdtxt 0.026秒通過。r5以新建置完整重跑171.367秒，13張實際視窗通過，保存與讀取回覆已目視無殘留。兩主題保存及來源槽SHA均190b9f8f614c69a40eea7cf3883facf92efd83c32b20dca2a1677289437aa456。

完整API與觀測器回歸cmd/fd2 643.131秒、campaign 4.899秒、fdtxt 0.460秒、localization 1.480秒通過。其來源指紋保留於hotel-regression-r3-source.json，為服務回覆清除三行與新增去重斷言之前；最終修正另以focused及GUI驗證，不把早版全套外推成最終來源全套。四語各5107筆保持。58／56／57／61與README同步限定範圍，README加入實際HD旅館PNG，原四語忠實截圖保持。無新美術、私人素材或正式Release變更，matching維持暫停。最終暫時執行檔核SHA後已刪除，釋出18946288 bytes，未產生錄影。

來源快照註記：完整測試已編譯後才蒐集來源清冊，其間加入去重斷言，故清冊不能證明該斷言進入完整測試程式。逐檔核對生產來源只差native_hotel_ui.go的服務回覆清除三行；新斷言與最終版仍由focused與r5驗證。2618條異動文件本地連結零缺件，README文化入口與原四語圖保持。

提交前官方pull／render／verify為21項、14項人工判定、可能完成0；#254待來源提交後結案。231條教訓與114個守衛通過，更新放鍵事件守衛。新增產物UID/GID1000:1000，工作根無root-owned檔案或.md目錄；FD2一次性容器均已清除。原版EXE SHA保持，私人素材庫及正式tag未修改。

本切片已提交2e0d17897d6fc440c8e65298882d4c46398887c2並快轉推送main，遠端HEAD核對相同。#254已依該來源結案，#204同步已驗旅館範圍與其餘字型待辦。官方pull／render／verify回到20項、13項人工判定、可能完成0；生成清單9條本地連結通過。主收據綁定來源提交，未再修改Go或GPU產物。前段來源快照的生產／測試界線依後續註記勘誤，不能把編譯後蒐集的測試檔當成完整回歸二進位的來源證明。

## 2026-10-18：教會即時面板 HD 字型

從main ad5797b0建立feat/hd-church-fonts-20261018。命中規格閘門、正常玩家操作、文件職責與README路由，61先加入READY契約。教會即時選單、名冊、轉職／復活列表與問句、交付物品及回覆沿既有FDTXT與具型別資料提交高解析文字，金幣與31／42／119數字沿原座標。共享資產只在Draw期間包裝並還原；下框覆蓋前清除文字描述，保留上方金幣。旅館沿共用包裝，商店數字映射抽出共用函式；規則、保存及原始素材未變。

初次文字清冊將新Draw guard的內部模式字串誤列玩家文字，遷移因缺少翻譯拒收。改用具名內部常數後重新盤點與重綁，沒有修改盤點器或猜補翻譯。四語各5107筆譯文、狀態、變數及審校身分保持；5106筆依id、1筆依signature遷移，新增0筆。

初次focused回歸22.356秒通過；最終來源在建置與回歸之前保存960個Go檔案指紋。相關cmd介面回歸83.485秒、campaign 11.168秒、fdtxt 0.331秒、fdother 16.778秒、localization 1.802秒通過。cmd只跑HD、教會、轉職、商店、旅館、JSON、標題LOAD、存槽與語系相關測試，不宣稱完整cmd測試。轉職、空復活名冊、餘額不足、交付目的名冊及滿袋回覆等fixture分支限RUNTIME-E1。

church-gui-r1在251.157秒完成正常START後F9接續既有hotel_ch02，經城鎮進教會；實際使用原槽倒下隊員與物品，沒有新增狀態注入。四語名冊、復活列表／問句與取消、交付物品列表與取消、F2、縮放及說明遮罩通過，15張實際視窗已審查。66次操作均觀察實際按下及放開事件；整理收據時修正xdotool的Return／Left／Right與Ebiten的Enter／ArrowLeft／ArrowRight名稱映射，GUI收據與測試程式未改。名冊F2往返PNG SHA相同。回旅館普通F5／F9，兩主題保存與來源完整bytes相同，SHA均190b9f8f614c69a40eea7cf3883facf92efd83c32b20dca2a1677289437aa456。

58／56／57／61、主字型收據及索引同步限定現況。README新增HD教會復活確認實際畫面，原四語忠實截圖與文化保存入口保持。#204更新教會即時面板進度，教會快取狀態、其他固定面板、預建轉場及圖示文字仍待辦；整款HD與字型尚未完成。無新美術、私人素材或正式Release變更，matching保持暫停。暫時執行檔核SHA後刪除，釋出18960600 bytes；沒有錄影，保留PNG及收據。

提交前官方pull／render／verify為20項、13項人工判定、可能完成0；231條教訓與114個守衛通過。2622條異動文件本地連結零缺件，960個Go檔案來源指紋與15張GUI截圖SHA一致，原版EXE SHA保持。新增檔案UID/GID1000:1000，工作根無root-owned檔案或.md目錄；Go／GUI一次性容器均已退出清除。

教會切片來源已提交48ab3e57dc7ea72f3f01fd4726ff26be59748c23並快轉推送main，遠端HEAD核對相同。#204保持開啟並綁定來源提交；主收據補入source_commit，不修改已驗Go、GUI工具或產物。

## 2026-10-18：角色狀態與命令穩定頁 HD 字型

從main 8d725032建立feat/hd-status-fonts-20261018。載入規格閘門、正常玩家操作、文件分工及README路由，61先列READY。原先教會及戰場狀態在Update預建像素快取；新增開啟時具型別輸入快照，保留80-byte紀錄、無動態文字底圖、素材、物品列與命令資料，供穩定Draw合成HD字模及數字。忠實快取與12／14幀轉場沿原排程，不改規則、存檔、頭像或圖示。

原合成器完整索引bytes、共享素材與快照不可變、live角色／命令表變動不污染已開面板、教會／戰場穩定消費端與轉場無殘留通過。focused 11.905秒；相關cmd介面96.249秒，battle面板／命令合成器0.689秒；campaign 10.050、fdtxt 0.217、fdother 21.737、localization 1.854秒與文字清冊1.952秒通過。未跑完整cmd或battle套件，不外推。Go編輯結束後保存962檔指紋，蒐集與最終建置／回歸並行；期間沒有再改Go，GUI前後逐檔核對相同，不把快照蒐集時點當成編譯來源證明。

來源行號移動後，4864筆依id、243筆依signature遷移，by_text與新增均0。四語各5107筆文字、審校狀態、變數及身分逐項保持。初次patch因main欄位空白錨點不合未套用，讀取實際欄位後修正；不是產品或原版缺陷。

status-gui-r1在246.771秒完成正常START後F9接續既有旅館，經城鎮普通選人進索爾狀態與ID9悠妮命令。24張實際視窗完成審查，78次輸入均有真實按下／放開事件。兩頁F2往返PNG SHA各自相同；縮放／說明遮罩、四語名冊、復活及交付取消、返回旅館F5／F9通過。兩主題保存及來源完整SHA均190b9f8f614c69a40eea7cf3883facf92efd83c32b20dca2a1677289437aa456，未注入角色狀態。戰場消費端與14幀推進另用fixture列RUNTIME-E1，沒有新正常戰場GUI或原版對拍。

56／57／58／61、字型主收據status_followup與索引同步限定範圍；README增加實際HD狀態截圖，文化保存及四語忠實圖保留。#204仍待其他固定面板、預建轉場與圖示字樣，未宣稱完整HD字型。無新美術、私人素材或正式Release變更，matching保持暫停。暫時執行檔核SHA後刪除，釋出18965704 bytes，沒有錄影；收據與PNG保留，Go／GUI容器已退出清除。

提交前官方pull／render／verify為20項、13項人工判定、可能完成0；231條教訓與114個守衛通過。2626條異動文件本地連結零缺件，962個Go檔案、24張GUI截圖、回歸日誌與README圖片SHA通過；文化入口保持，原版EXE SHA未變。新增檔案UID/GID1000:1000，工作根無root-owned檔案或.md目錄，本輪FD2一次性容器已清除。

角色狀態切片已提交70d50d7cf78d9de4770d3e685ee79b347d8f166e並快轉推送main，遠端HEAD核對相同。#204保持開啟並綁定來源提交；主收據補入source_commit，不修改已驗Go、GUI工具或產物。

## 2026-10-18：物品操作與商店裝備穩定頁 HD 字型

從main 7062d21c建立feat/hd-item-fonts-20261018。命中規格閘門、正常操作、文件分工及README路由，61先列READY。開啟／裝備重建時保存無動態文字底圖、紀錄、素材、效果列與raw slot選取快照；成功發布像素畫面才發布選取，失敗還原既有快照，關閉清除。穩定Draw提交HD字模與數字，原12幀開收框保持像素。每個實際拷貝矩形只遮蔽其內文字，框外部分保留，新文字也裁在拷貝區域；規則、名冊、保存、頭像與圖示不變。

focused r1把ReadPixels放在Ebiten繪製迴圈前，引發command queue panic，分類測試環境錯誤。改驗CPU索引底圖、image身分與快照不變，GPU另走正式GUI；r2在3.855秒通過。初次盤點誤把新Text helper的內部portraits路徑當玩家文字，遷移拒收；改具名資源目錄常數後重綁，4865筆依id、242筆依signature，新增與by_text均0。四語各5107筆譯文、審校狀態、變數及身分保持。最終964檔來源指紋於建置及相關回歸之前保存，期間沒有再改Go。

最終cmd相關模式73.341秒、battle物品面板／selector／命令合成器0.313秒、campaign 4.140、fdtxt 0.241、fdother 9.975、localization 0.995與文字清冊1.467秒通過；不是完整cmd或battle套件。覆蓋有洞raw slot、空袋、裝備刷新與失敗還原、矩形遮蔽、戰場物品及死亡交換caller。其他caller未新增正常GUI或原版對拍。

item-gui-r1在城鎮誤送Right，從0走到教會4而等待武器店，分類腳本失敗。回查main的原生城鎮Right=-1／Left=+1與商店獨立delta定義，修正受版控工具的前往／返回方向；遊戲Go與binary未改。item-gui-r2以同建置完整重跑100.147秒，13張實際視窗已審查，53次操作均有實際按下／放開事件。短劍、皮甲、藥草選取，拒絕裝備藥草及重新裝備既有短劍，F2、縮放／說明遮罩、返回與F5／F9通過。角色與金幣保持，F2往返PNG SHA相同，兩主題保存與來源完整SHA均190b9f8f614c69a40eea7cf3883facf92efd83c32b20dca2a1677289437aa456，沒有新增狀態注入。

56／57／58／61、字型主收據item_followup與索引同步限定範圍，README加入實際HD商店裝備截圖，保留四語忠實圖與文化入口。#204保持開啟，接續原生整備、原生標題LOAD、預建轉場與圖示字樣，不宣稱完整HD字型。無新美術、私人素材或正式Release變更，matching保持暫停。暫時執行檔核SHA後刪除，釋出18981432 bytes，無錄影；PNG與收據保留，一次性GUI與回歸容器已清除。

提交前官方pull／render／verify為20項、13項人工判定、可能完成0；231條教訓與114個守衛通過。2630條異動文件本地連結零缺件，964份Go來源、13張GUI圖片、主收據與回歸日誌SHA通過；README文化入口及原四語圖保持，原版EXE SHA未變。新增檔案UID/GID1000:1000，工作根無root-owned檔案或.md目錄，相關一次性容器已清除。

物品操作切片已提交316f3a18a7781082c2eb307959a2b77d5e0bd378並快轉推送main，遠端HEAD核對相同。#204保持開啟並綁定來源提交；主收據補入source_commit，不修改已驗Go、GUI工具或產物。


## 2026-10-18：整備與標題讀檔穩定畫面 HD 字型

從main 02933ee7建立feat/hd-preparation-load-fonts-20261018。命中規格閘門、文件職責與正常操作路由，61先列READY。整備在穩定Draw複製資產並沿原字模／已驗數字bank提交HD文字，繪製後還原；標題四槽使用自身palette，完整畫面清除舊文字。原預建開收框、部署規則、槽metadata、JSON後備及SAV交易保持。

最終965份Go來源在建置與回歸前固定。相關介面回歸53.889秒，campaign 11.900、fdtxt 1.133、fdother 20.849、localization 1.441及清冊2.248秒通過，不宣稱完整cmd回歸。必選訊息初測用了不合法identity0，依既有1..31改為1後通過。初次重構移出Draw的錯誤訊息造成清冊少一筆，保留原入口並由修改前基底重綁，四語各5107筆譯文／狀態／變數／身份逐筆相同；5106筆依id、1筆依signature，無新增翻譯。

JSON普通標題LOAD接續既有hotel_ch02，城鎮出發選NO後返回旅館F5／F9，11張GUI、36次按下與放鍵收據通過，兩主題保存及來源SHA均190b9f8f614c69a40eea7cf3883facf92efd83c32b20dca2a1677289437aa456。第24章原生槽另走普通標題LOAD、保存第二槽、選人15筆、確認取消、F2、縮放與說明遮罩。來源為原版第23章SAV，祖先含攻略建構與清敵；本輪不新增注入，不提升自然難度或整章PLAYER-E2。最終native r3有17張GUI與52次按下／放鍵，兩主題保存SHA均d5f39666ae53f7b55438a1d4658d38b47a301e434c8aaed65c5ed6fac73fae09，後續選人及取消不改保存。

native r1按鍵通過，但縮小視窗後只隔4個Update擷取，GPU仍舊viewport而裁切，不列縮放通過。工具等待20個後續Update並記錄X11幾何。native r2另遇Xvfb尚未就緒，加入getdisplaygeometry有界檢查。相同Go及binary完整r3重跑，實際相隔30幀，1600×1000完整顯示；上方靜態HUD縮放診斷誤差由0.189337降至0.003671，非原版對拍。原失敗日誌保留，兩條可重用教訓與守衛加入清冊。

最終28張實際視窗已審查。56／57／58／61、字型主收據preparation_followup及索引同步；README新增HD整備與標題讀檔圖，保留四語忠實圖及文化內容。剩餘未列固定畫面、預建轉場／金幣滾動、圖示內文字與平台效能由#204追蹤，不宣稱整款HD字型完成。matching暫停，無新美術、私人素材或Release修改。暫時執行檔核SHA後刪除，釋出18990448 bytes；無錄影，PNG與收據保留。

提交前官方pull／render／verify為20項、13項人工判定、可能完成0；233條教訓與116個守衛通過。2634條異動文件本地連結零缺件，965份Go來源與新圖片SHA保持，原版EXE與正式第一版tag未變。README文化入口與四語忠實圖保留；新增檔案UID/GID1000:1000，工作根無root-owned檔案或.md目錄，本輪一次性容器均已清除。

整備與標題讀檔切片已提交cde408b492c4a911965d5df3b0be5f4f3ac59300並快轉推送main，遠端HEAD核對相同。#204保持開啟並綁定來源提交；主收據補入source_commit，不修改已驗Go、GUI工具或產物。


## 2026-10-18：整備問句預建轉場 HD 字型

從main ca028679建立feat/hd-preparation-transitions-20261018。命中規格閘門、正常操作與README路由，使用reverse-engineer-retro-game-remake技能，61先列READY。從既有六個建立端確認六張空框、四張問句選項動畫及收框後caller還原；必選訊息為六張空框後一張文字。為登記階段保存獨立底圖、原字模／文本／選項素材、問句種類與必選身份，在Draw重建HD畫面。原索引快取、Draw承認、輸入鎖定、還原與一次性延續點保持；其他job沿原路徑，不在Update包裝字模，沒有新增原版RE或matching工作。

七種開收配方逐張等同原索引畫格，後續來源變更不改既有轉場；中途主題切換及錯誤回退通過。首輪測試誤把收框第5張算作問句，且錯誤回退fixture缺真實palette，依原四張文字階段與素材契約修正，focused r2為12.504秒。最終967份Go來源在建置與回歸前固定，相關介面回歸77.535秒；campaign 4.106、fdtxt 0.122、fdother 10.141、localization 0.787與清冊0.826秒通過，不宣稱完整cmd回歸。四語各5107筆譯文／狀態／變數／身份逐筆保持，5106筆依id、1筆依signature，沒有新增翻譯。

GUI r1的CPU階段判定不足：preparation-transition-gui-json-r1雖回報PASS，frame7與14筆文字的PNG仍空框，另一CPU-contained圖片仍為穩定頁。此PASS由實際GPU審圖推翻，不作轉場完成證據。native r1反因擷取後CPU離開四張文字階段，拒收已呈現HD字與移動中選項的樣本，CLI失敗。回查正常操作路由後，改用實際PNG問句RGB完全等同HD參照，且選項金色邊界須較穩定頁內移。CPU前後狀態照實保留，不宣稱精確同幀；空框／穩定頁負例保留，新增可重用教訓與守衛。

同一Go與binary乾淨重跑兩條r2。JSON普通標題LOAD後城鎮出發選NO、返回旅館F5／F9，有11張普通畫面、2張有效開收轉場及36次按下／放鍵；兩主題與來源完整SHA均190b9f8f614c69a40eea7cf3883facf92efd83c32b20dca2a1677289437aa456。晚期第24章原生槽普通LOAD、保存第二槽、選人15筆、確認取消、F2與縮放／說明，有17張普通畫面、3張GPU轉場及52次按下／放鍵。記錄問句開收框與確認問句收框通過，兩主題保存SHA均d5f39666ae53f7b55438a1d4658d38b47a301e434c8aaed65c5ed6fac73fae09；後續選人及取消不改保存。晚期槽祖先有攻略建構與清敵，本輪沒有新增注入，不提升自然難度或整章PLAYER-E2；必選訊息限fixture。

最終33張實際視窗已審查，5張轉場問句區RGB相同，選項金色邊界由穩定1392..1727內移至1416..1703或1440..1679。56／57／58／61、字型主收據preparation_transition_followup及索引同步。README加入城鎮開框與整備確認收框實際圖，保留四語忠實圖與文化入口。其餘設施／角色／物品轉場、金幣滾動、圖示內文字及平台效能由#204追蹤，完整字型與HD仍未完成。無新美術、私人素材或Release變更；核對SHA後刪除暫時執行檔，釋出18996616 bytes，沒有錄影，PNG與收據保留。

提交前官方pull／render／verify為20項、13項人工判定、可能完成0；234條教訓與117個守衛通過。2640條異動文件本地連結零缺件，967份Go來源、GUI工具及圖片SHA通過。README文化入口與原四語圖、原版EXE及正式第一版tag保持。新增來源與圖片UID/GID1000:1000，工作根無root-owned檔案或.md目錄，本輪一次性容器已清除。

整備問句轉場切片已提交5f6934373c0c25a7c97c0302fc0e5e57f8a40a6c並快轉推送main，遠端HEAD核對相同。#204保持開啟並綁定來源提交；主收據補入source_commit，不修改已驗Go、GUI工具或產物。

## 2026-10-18 HD 金幣滾動與出售確認

從main的a586bcc9建立feat/hd-gold-roll-20261018。依復古遊戲路由載入規格先行、正常玩家路徑及README／文件職責契約，沿用重製技能。61先登記READY，再將既有扣款／入款的八位獨立滾動抽成共用窗口計畫；原索引合成及HD使用同份計畫。交易建構時保存背景與窗口，HD Draw只替換金額區，原10毫秒步長、金額／物品提交、輸入鎖定及延續點保持，不重開matching或硬體時序RE。

新測試首輪括號錯誤，修正後r2／r3指出素材假設不符。清冊實際記錄數字1為5×8，其餘為6×8，字距仍6；修正精確來源檢查後focused r4通過。清冊遷移第一次誤用editor-canonical根，查明正式來源在story子目錄後重跑，無產品修改。四語各5107筆，只遷移座標與來源雜湊，譯文、狀態、變數與身份保持。

普通START後F9接續既有hotel_ch02，進武器店購買布衣、拒絕立即裝備，再出售同件，餘額1000→950→987。GUI r1保存預期過嚴：既有RemoveNativeCompactInventory會左移全部raw格，包括未使用格的殘值，造成索爾InventorySlots[5]由255變0。原始規則在native_inventory_transfer.go，未因HD修改。修正預期後，HD r2及忠實r1的完整保存SHA同為32c9cd021b3c7112f1b6f16dae8ffa1c753fd45cd0d1fc9d92331e99c83eed66，新程序標題LOAD通過。這是重製兩主題對照，不新增原版SAV差異放行。

逐張審查另發現既有出售確認殘留HD歡迎詞，登錄#255。原生合成器已重建對話區，但HD上層未清除。依61補READY契約，在Draw清除被覆蓋描述，再由同份原生字模擷取問句，不新增翻譯；歡迎詞負例與忠實畫格保持通過。最終Go相關介面回歸73.288秒，campaign／fdtxt／fdother／localization／清冊套件均通過，非完整cmd或battle回歸。最終來源為gold-roll-source-release.json，binary SHA為23091a2a680499906a43820ac9c7fd2c20f599dbad7cd5d48ec2822b542be7b1。

最終GUI r3整張視窗擷取錯過0.45秒扣款階段，只取得入款有效樣本。CPU候選不足以證明GPU，該輪不通過。改抓實際金幣區域後，同份binary的HD r4通過9張正常畫面、兩張有效GPU區域與新程序標題LOAD。忠實r3的買賣與保存通過，但重開程序時舊audit高幀號按鍵被誤認為新程序收據，冷讀工具逾時。工具改為逐程序保存audit，保留失敗紀錄；不調遊戲速度、不重建binary，接續完整重跑。上述環境及驗證腳本失敗不登錄為遊戲缺陷。

最終HD r5與忠實r4使用同份binary／GUI工具完整通過，包含新程序標題LOAD，audit按程序分檔。共22張實際畫面含4張有效HD金幣區域，GPU閘門拒收空框／前一階段／穩定金額，不宣稱精確同幀。兩側完整保存SHA均32c9cd021b3c7112f1b6f16dae8ffa1c753fd45cd0d1fc9d92331e99c83eed66，僅金額與既有未使用格殘值相對來源改變。README加入最終扣款區域及出售確認，原四語圖與文化入口保持；56／57／58／61／索引及字型主收據gold_roll_followup同步。未錄影，核對SHA後刪除暫時binary，釋出19013056 bytes，PNG與收據保留。正式Release／tag、原版檔及私人素材包未修改；其他轉場、成功演出、圖示文字、戰場與平台效能仍待完成。

提交前236條教訓與119個守衛通過；守衛工具略過測試檔，故出售修正守衛綁正式繪字掛勾。2639條本地連結、969份Go來源、GUI工具、README文化入口、圖片與UID/GID核對通過；工作根無root-owned檔案或.md目錄。GitHub #255待來源提交後結案；完整HD仍沿#204接續。

金幣滾動與出售確認切片已提交d19340be3be13ae1e93e7dd5f4bd95b5515e8804，快轉推送main且遠端HEAD核對相同。主收據補綁source_commit；已驗Go、GUI工具、圖片與保存不再修改。

#255已在遠端確認CLOSED，#204維持開啟並更新金幣驗收與剩餘範圍。官方pull／render／verify回到20項、13項人工判定、可能完成0；236條教訓與119個守衛通過。最後只同步來源綁定與工作清單，不修改已驗程式或產物。

## 2026-10-18 角色與物品面板 HD 轉場

上一輪為有效進度：金幣與#255已推送並結案。本輪由12b269f4建立feat/hd-panel-transitions-20261018；先核對Git／容器／遠端#201／#202及官方pull，沿規格先行、正常玩家路徑與文件職責契約，61先READY再實作。教會與戰場狀態開收／命令轉場保存獨立資料與底圖；物品沿既有raw slot快照。像素與文字共用NativeItemPanelRegion，先裁來源，再移動字形及clip，最後裁目的。原12／14張快取、Draw承認、交易與保存保持，不在Update包裝字型。

首輪focused3.821秒通過；其後追加唯讀audit，最終相關cmd回歸48.994秒與五資料／清冊套件通過，非完整cmd或battle回歸。最終來源panel-transition-source-final.json含971份Go與兩份GUI工具，建置與回歸後未再改動。四語各5107筆僅來源／座標遷移，by_id5105、signature2、new0，內容／狀態／變數／身份保持。

同份binary的教會status r1與商店items r1首次完整GUI均通過，共39張正常圖與11張有效GPU區域，50張已逐張審查。區域RGB等同穩定HD參照的移動後裁切，且不同於未移動位置；CPU只定位候選，不宣稱精確同幀。普通START後F9接續既有hotel_ch02，選人／命令、裝備選取／拒絕藥草／重新裝備、F2、縮放／說明遮罩、F5／F9與新程序標題LOAD通過。完整保存與來源均190b9f8f614c69a40eea7cf3883facf92efd83c32b20dca2a1677289437aa456，不增加原版保存例外。戰場caller限E1，尚未新增正常戰場GPU。

另以內建image_gen生成可丟棄的戰場筆觸參照，保存於work/promo-cadence-20261018/hd-fonts/battlefield-hd-style-prototype-v1.png，提示在battlefield-hd-style-prototype-prompt.json，來源與SHA在同名v1.json。實檔1586×992，非1920正式runtime收據，不登錄catalog或私人正式包，不當作完成證據。61既有手繪配色與大頭短身已核准；提問只收集紋理密度偏好，不阻擋既定HD。原型中的裝備與地形仍須依正式來源校對。

下一主切片先以正常首章LOAD驗現代戰場與狀態返回，避免只做孤立字型。合法來源work/promo-cadence-20261018/f2-runtime/normal-r4/user-data/fd2_re/fd2_save.json，node=battle_ch01、SHA089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2。現有Home為正常強制檢視鍵；drawNativePlayerStatus在Game.Draw的戰場／介面之後。須實測完整indexed面板是否蓋回現代底圖，尚不能僅憑程式下結論，再接144角色母稿與地形HD。

README加入實際命令轉場局部連結，文化內容及四語圖保持；56／57／58／61／索引與panel_transition_followup同步。未錄影，核對SHA後刪除暫時binary，釋出19034376 bytes，保留PNG與收據。原版檔、正式第一版tag／Release及私人素材包不變；完整HD尚未完成。

提交前20項工作清單、13項人工判定、可能完成0；236條教訓與119個守衛通過。2657條本地連結、最終Go／GUI來源、README文化入口、圖片及UID/GID核對通過，工作根無root-owned檔案或.md目錄。無新缺陷Issue；本輪只更新#204／#202的具體驗收與未完成範圍。

角色與物品面板轉場已提交ecb006ad2fda7eac97c10d8e3d7c6f364812c4e1，快轉整合並推送main；主收據補綁source_commit，已驗Go、GUI工具與產物保持。#204／#202維持開啟，接續實際戰場與其餘HD範圍。

遠端#204／#202已更新並保持開啟。官方pull／render／verify為20項、13項人工判定、可能完成0；236條教訓與119個守衛通過。來源綁定與工作清單快照同步，未修改已驗程式。

## 2026-10-18 實際戰場角色狀態與現代底圖

前輪面板轉場已推送並同步Issue，屬有效進度。本輪由main e82cb099建立feat/hd-battlefield-panels-20261018。命中正常玩家試玩、規格閘門、文件職責與README路由，沿用重製技能。正常標題LOAD既有普通START／F5首章JSON後，Z聚焦、Home檢視索爾，面板外地圖54000像素有50832被換成原版底圖；由實際PNG及presentNativeClassFrameWithPalette全幀consumer確認，登錄#256，61先READY再實作。

現代戰場只繪原NativeItemPanelRegion目的區域，HD文字沿原來源移動與裁切；HD合成失敗仍使用像素面板，不還原完整原版地圖。忠實與教會路徑保持，原12／14張快取與生命週期不變。focused5.052秒、相關cmd35.408秒、清冊0.226及localization0.188秒通過；不是完整cmd或battle回歸。971份Go在回歸與最終建置之間保持，文字清冊bytes完全相同，四語各5107筆不需遷移。

前測工具錯等索爾命令頁，實際無命令確認會關框；依既有NativeCommandIDs改驗索爾與悠妮。final r1機械通過但冷讀圖片尚在空對話框，不能當可讀完成圖；只修GUI等待後續20個Update並正常結束對話，Go未改，重建binary SHA完全相同。最終r2的14張實際圖逐張審查，六組54000像素面板外地圖零差異，F2、縮放／說明、兩角色返回及新程序標題LOAD通過。完整保存與來源SHA均089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2。沒有戰鬥、座標／血量注入或新原版對拍。

字型主收據battle_status_followup、56／57／58／61及索引同步，README加入實際首章狀態圖片，文化入口與四語忠實圖保留。下一主切片為144×144角色及地形；已盤點94組母稿，48 RGB、45 RGBA、1缺master_file登錄，89組12幀、5組3相。此盤點不當作HDruntime完成，group68須核對及正式登錄來源，不能憑檔名猜補。未新增美術包或私人同步，原版檔與正式Release保持，matching暫停。

核對SHA後刪除暫時binary，沒有錄影；保留PNG、前測失敗與來源收據。#256待本輪來源提交後結案，#202／#204仍待完整戰場與其他HD範圍。

提交前21項工作清單、14項人工判定、可能完成0；238條教訓與120個守衛通過。2575條本地連結、971份Go與GUI來源、README文化入口及原四語圖保持。全work檢查找到三個2026-09-16既有root-owned空dosgolem-cache目錄，確認無FD2容器使用後只對三目錄逐一修復；重查無root-owned或.md目錄。最終圖片及文件UID/GID1000:1000，FD2一次性容器均已退出。

戰場覆蓋修正已提交7dcce7296aeea708b02cd5f27cc9692ade141b5d並快轉整合main，主收據補入source_commit；已驗Go、GUI工具、圖片及完整保存保持。

遠端HEAD已核對7dcce729，#256確認CLOSED；#202／#204保持開啟並更新本輪來源。官方pull／render／verify回到20項、13項人工判定、可能完成0；238條教訓與120個守衛通過。最後僅同步來源綁定與工作清單，已驗程式及產物保持。

## 2026-10-19 HD戰場角色母稿編製

前輪7dcce729與122ebdfb已驗推送，屬有效進度；本輪從main建立feat/hd-battlefield-sprites-20261019。沿既定1920與手繪風格，先以受版控工具從母稿編製144圖，不放大24成品。四組39張樣本包含行走、靜態與重複政策；靜態48母稿本身仍具像素化外觀，不能因尺寸144就宣稱細節提升。全94組候選正在有界容器編製，DRAFT及工具入口已掛61。

本輪已編製94組1083張首輪144候選，hash／尺寸／alpha通過，但固定格線有435筆碰邊診斷，第4組明顯裁切，拒絕進catalog。不能因資料齊備而宣稱美術或runtime完成。工具改全母稿連通輪廓排序及來源alpha保留，拒收不明額外輪廓；0／4／9／48／52五組51張已逐幀審圖，頭部、武器及四方向完整，52保留重複幀。全組r2另有界重編，結果以實際manifest為準。

內建image_gen沿核准手繪風格補FDICON48銀白機械靜態母稿，與DATO48父王分開。新2172×724 RGBA及三個相位已審，來源26f5b0aea38b2665abc8a57099b2dfb88149a50363efa32ed96ac2a09dd82d35，保存於remake/generated-assets/modern-theme-prototypes/fdicon-048-style-a-v2-hd-master.png，完整prompt及來源記錄在work/promo-cadence-20261018/hd-fonts/hd-static48-v2-prompt.json與generation.json。舊稿保持，尚未登錄catalog或同步正式私人包。

同狀態F2比較先發現#257：全域切主題後又被中立戰場F2／Home舊alias消費，前測忠實圖片其實是狀態頁，不能作取景證據。61先READY後只移除戰場F2舊alias，Home保持；相關cmd16.550秒、文字清冊bytes不變，四語各5107筆文件未改。最終普通首章標題LOAD、Z聚焦、F2往返不開狀態、Home、兩角色／命令、縮放／說明、F5／F9與新程序LOAD通過16張實際圖，完整保存與來源SHA均089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2。不是完整cmd／battle回歸或原版對拍。

真正雙主題戰場圖片確認#258：相同units／view仍有現代24邏輯直繪與原生320×200兩倍的取景差異。這是下一個HD世界切片，尚未修正。原生NativeFrameInput／frame／placement及Mask-only單位pass已核對，不重做RE；39／57無來源寫入時保持no-op，不能造人物。完整世界與透明介面分層仍DRAFT，尚未接production。

新HD戰場主收據、61及00索引、56／57／58同步，README補F2／Home既定操作並保留文化與四語圖。核SHA後刪除兩份暫時binary，未錄影；PNG及拒收候選保留。#257待來源提交後結案，#258／#201／#202／#204保持工作範圍；完整HD仍未完成。

完整輪廓r2在第3組拒收，只有7個大輪廓，不能安全宣稱12幀；未調低門檻或挑雜點補數。工具追加--audit-all以一次列完全組拒收，有缺件仍非零退出；r3有界執行中，實際終態與manifest待核對。既有51張樣本的編製函式保持，F2相關Go與GUI來源未改。

提交前22項工作清單、15項人工判定、可能完成0；240條教訓與122個守衛通過。2585條本地連結、F2回歸Go／GUI來源、README文化入口與原四語圖保持，文件與工具UID/GID1000:1000。全組素材audit仍為有界工作，未把執行中或拒收候選當完成。

F2修正與HD候選編製器已提交f5fdb81c1ac86fa969842c0ce6bb578fe63e0d71，快轉整合main；新主收據補綁來源提交，已驗Go／GUI與圖片保持。#257依本提交結案，#258及HD主項保持未完成。

完整輪廓audit r3已完成，實際94組均受檢，84組963張通過hash／尺寸／alpha；10組拒收，exit1為來源不齊的明示結果，不是假產品失敗。拒收group為3,10,12,17,18,20,27,44,49,62。未放寬門檻，未登錄production；全組姿態仍待人工審查。新主收據與61補入實際manifest及拒收清單。所有本輪Docker編製／GUI批次已終止清除；內建image_gen另正補3／10／12母稿，尚未採用。

#257已確認遠端CLOSED，#258及HD主項保持開啟；官方pull／render／verify為21項、14項人工判定、可能完成0，240條教訓與122個守衛通過。三份新母稿已保存，但以非零alpha連通仍拒收。分級診斷顯示64／128／192核心可在三組各分出12人物，真因為低透明度橋接；後續只改拓樸判定並保留原alpha邊緣，不硬放行。group10 frame5方向仍錯，另列補繪。已驗Go／GUI保持。

GitHub #201／#202／#204已回填全組診斷84組963張及10組拒收的實際數字，官方pull／render／verify21項、14項人工判定、可能完成0；原240條教訓及122守衛驗證保持。F2來源971份Go及GUI SHA不變，正式第一版tag保持，所有本輪Docker與內建圖像生成皆已終止，素材仍未登錄。

## 2026-10-19 HD透明度核心與候選校稿

從已核遠端c78072f4建立feat/hd-sprite-alpha-core-20261019，保留接手的核心分離變更。
alpha128只識別人物核心，取回原覆蓋邊緣；單幀明示替換另記來源SHA，重複相位完整沿用canonical來源。
獨立驗證器檢查來源、144尺寸、覆蓋alpha、原墨跡邊界及相位。r6全94組1083幀通過，
48張舊binary-alpha樣本bytes相同，六項拒收／來源守衛通過；只證明候選資料，不是原版對拍或GPU驗收。

內建image_gen補繪來源並保存prompt與實檔SHA，目前22份新母稿明示供28組重用，另3份單幀。
全組候選掃描後，重審29組339張替換幀，修正錯向及第37組花形盾徽。
第19組整稿持盾側與背向錯誤拒收；其單幀只改善朝向，仍像素化。
部分舊母稿比例待校正；第76組中立左向圖顯示不等比填框會把頭盔拉寬，下一步先做框內等比例樣本。
正式catalog、私人包與Go執行期未改，HD世界仍DRAFT，#258保持未修。主收據追加alpha_core_followup_20261019，
61與56／57同步，不以資料齊備宣稱完整HD。來源仍是未整合工作樹，本輪未提交或推送候選。

971份Go及GUI來源共972檔SHA保持，沒有必要重跑未變動的Go回歸；本輪無錄影或遊戲binary。
Python遞迴擁有權掃描逾時屬檢查工具問題，改以同容器GNU find完成；work及本輪可寫目標皆1000:1000。
既有docs研究檔有root-owned檔案，本輪只讀，未做廣域chown。一次性編製／驗證容器均退出。

GitHub #201／#202／#204已同步本機候選進度與未整合限制，官方pull／render／verify維持21項、
14項人工判定、可能完成0；241條教訓與123個守衛通過。正式第一版tag保持d8c361a8，
README及四語忠實截圖未改，沒有新的玩家可見完成聲明。

## 2026-10-19 HD等比例與完整圖庫範圍勘誤

沿既有分支與手繪風格接續。六個縮放原型追到24成品的producer，確認registered ink是現代預覽輪廓，不能當原版FDICON邊界。改用22像素方格等比例、水平置中及已登錄腳底，r9既有94組1083幀資料驗證通過；新增11母稿供14組168幀，19整組取代舊像素化單幀，10及76保留明示方向修正。六頁完整方向掃描及兩頁新稿全尺寸審查已完成；未以此宣稱全部身分或正常GPU完成。

r8序列編製300秒界限留下336張，exit124且沒有完整manifest，保留為工具效能診斷。編製器改同母稿只辨識一次與有界工作程序，r9三程序完整通過；63張r7樣本及336張r8部分輸出bytes相同。正式工具另驗序列／兩程序PNG及manifest完全相同、canonical重複來源、比例、雜湊及拒收等九項守衛。14-v2預覽藍色背景的初次判斷保留勘誤：背景抽查alpha為0，不能稱整份不透明；未選v2，採用v3。

61先READY後增加HD圖源載入、原生單位計畫、前景Mask計畫與候選世界合成器。1568組單位位置及96組前景實際native寫入比較通過，含零色來源、透明洞、位移與原子拒收；候選世界固定四像素邊界並保留既有現代變暗政策。完整Game.Draw／透明UI／HUD與走行仍未接。圖源loader首輪因/pack symlink缺/repo掛載失敗，修正同一repo唯讀掛載後18.497秒通過；後續座標84.983秒、前景0.896秒、世界fixture2.721秒通過。Docker socket沙箱連線失敗未啟動測試，以同一有界隔離命令重跑。980份Go來源快照保持，標準現代主題回歸60.698秒通過，但未給外部候選env而略過該條。

外部候選實測不是綠燈：同一正常首章來源JSON還原後，94組圖源預檢完成，世界因missing group96 frame0拒收，整體回歸39.757秒失敗。沒有用跳過敵人、猜代圖組或像素替代讓它通過，也沒有候選世界圖片、GPU／原版或新E2收據。來源保存SHA仍089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2。

固定B24的實際header與既有1680幀loader直接反證舊96組分母。新受版控工具核對原始size／MD5／SHA、前6 bytes及全部分離Mask：140組中125組1455槽可見，現代清冊漏31組372槽，沒有與較早完整12幀組相同的decoded組合。#259登記本缺陷，56／57／58／61及主收據同步；91歷史只追加勘誤。94組子集資料仍有效，驗證器改明示registered_catalog_complete，另明示native_bank_coverage_verified=false，不把清冊完整當全圖庫。

內建image_gen依正式DAC上色的原版96參照及核准0風格補第一份96母稿。1070×1470 RGBA與12方向已審，來源e4839b40733dd56e905acda6f9096b868147ec3bc334ba1799bddb4014a1e9d5；不推定角色／職業或未知持物。原圖、prompt與來源SHA已保存，尚未編製24父項／144圖或登錄私人正式包。下一閘門先補96及其他實際需求、清冊與loader範圍，再接世界／UI與正常GPU；HD地形與平台仍待完成。

本輪沒有錄影或遊戲binary，未變更原版檔、正式catalog、私人正式包與Release。README文化入口及四語忠實圖保持；主收據追加proportional_world_followup_20261019。

提交前官方Issue pull／render／verify為22項、15項人工判定、可能完成0；242條教訓與124個守衛通過。全遞迴Markdown檢查找到64筆既有歷史／oldtree／舊封包缺失，未誤稱整庫通過；README、00、56／57／58／61、91與91歷史共2912筆本地連結及正對照通過。980份已測Go SHA保持，work／現代素材無root-owned或.md目錄，本輪目標1000:1000；FD2容器全部退出。

HD候選層與圖庫範圍勘誤已提交2537cb0811812e13d20a5da77832857f94ba6b42並快轉整合main，遠端HEAD已核同值。#201／#202／#204／#258已綁來源，#259保留開啟並補96候選SHA。主收據補source_commit；980份Go及已驗素材保持，只有來源綁定與官方工作清單快照接續同步。

## 2026-10-19 完整原生人物圖庫與HD清冊

從main的2d6f62c8建立feat/hd-native-sprite-coverage-20261019。61先READY再修正原生140組邊界，
125組可見、15組無來源寫入與固定原始1680幀Mask逐項核對。舊無HD清冊保持相容，
含HD清冊必須完整125組及同組24父項；不再接受94組子集冒充完整。

依已核准風格與原版彩色資料參照補31份母圖及372張24父項。97首稿右向刀刃缺失拒收，
採v2；新增372張HD切幀已逐頁檢閱。r10全125組1455張來源、比例、alpha與相位驗證通過，
舊94組1083張父項及HD輸出bytes保持。正式351筆清冊保留所選母稿、單幀來源與SHA，
編製器可直接依清冊重生；拒收稿與原版參照不入私人allow-list。

首章同份普通START／F5 JSON的CPU世界候選已通過，原始保存089b077b不變。
正式清冊及私人包另驗16張正常GPU、F2／Home、狀態／命令、縮放／說明、F5／F9與新程序標題LOAD；
兩主題與來源保存bytes相同，六組面板外GPU區域零差異。這只驗既有24呈現，144世界仍未接Game.Draw。
相關Go最終回歸通過，fdicon0.341秒、cmd33.060秒；清冊六項與編製九項守衛通過。

兩次環境失敗保留：Go r2誤用FD2_ASSET_DIR，改FD2_ASSET_PACK同image r3通過；
GUI r1在暫存binary建置完成前啟動，沒有建立畫面，待完成後同image r2通過。
登錄後工具測試的隔離複本漏帶10／76單幀來源，補齊既有全域引用後r5通過；正式拒收規則不變。

私人庫維持private。3207檔逐項SHA與allow-list核對，無額外檔；提交1edc3a3b8e1c6582cceaa31a0e97ab99223851c2
已推送且遠端HEAD相同。runtime-only包3010檔，不帶母稿；正式Release與四語README截圖保持。
新主證據追加HD戰場主收據native_coverage_followup_20261019，56／57／58／61同步。
接續#258正式世界與透明介面，再擴HD地形、其他美術及早中晚期／平台驗證；完整HD仍未完成。

登錄後直接由私人包重生的九項工具守衛通過，runtime-only正式載入11.228秒通過。
242條教訓與124守衛、目前八個文件入口2916個本地連結及新工具索引正對照通過。
work與現代素材無root-owned或.md目錄，全部本輪FD2容器退出；已驗暫存binary與runtime-only重複包移除，
保留manifest、雜湊與可重跑工具。沒有新增錄影，沒有改動正式Release。

完整圖庫來源d8d8df214916d9a1523afdffc271ba88e5bd2d26已快轉整合main並核對origin/main。
#259依資料／loader／既有GUI／私人同步驗收結案；#201／#202／#204／#258已更新目前範圍及來源，仍保持開啟。
官方pull後為21個未完成項，主收據綁定來源；後續提交只同步遠端工作清單與證據，不改已驗程式或素材。

## 2026-10-19 正式HD戰場與戰鬥後F9

144人物已接入1920世界，與原生幀共用一次時鐘／DAC交易；地形種子、透明UI與GPU人物分層，缺件拒收不發布原生狀態。普通／第12章上移caller共用定位與terrain觀測，索引輸出保持。

正常標題LOAD既有START/F5首章槽，34張GPU、4張實際走行、指令環、取消、F2、兩次普通攻擊、狀態／命令、縮放／說明、保存及新程序LOAD通過。兩份穩定人物來源各10678核心像素零差異，一份跨相位不可比；同view的960地形樣本零差異。完整保存SHA保持089b077b，沒有錄影。

r4/r5發現#260：F9載入新非原生章首對話，舊經驗owner先攔輸入且原生等鍵條件不成立。較早把移動計畫稱為攔截原因並不充分；r5唯讀owner欄位才確認因果。有效JSON驗證後丟棄經驗owner／待播佇列與移動計畫，不跑舊續行；拒收保持。修正前回歸失敗，修正後成功／拒收通過，GUI亦越過卡點。

r2/r3為驗證工具未等章首對話及固定Z次數錯誤；r6混用讀檔前後不同鏡頭，改為同view比較後同一遊戲來源與輸入r7通過。首次完整測試僅字串清冊綁定過期，99項人工處置以精確舊SHA清冊遷移，其他套件通過；最後完整測試以隔離/tmp乾淨重跑通過，主程式282.386秒，所有套件通過。

現況、來源綁定與重跑命令沿[HD戰場主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的world_runtime_followup_20261019與[61](docs/knowledge-base/61-modern-theme-runtime-spec.md)。地形仍為原版放大，完整HUD、經驗／死亡／法術／故事演出、早中晚期與平台另驗。私人素材及Release保持，沒有新原版oracle、matching或整章PLAYER-E2。提交、Issue與清理結果於本批最後追加。

完整final回歸的後段全部因TempDir建立時空間不足失敗，沒有其他具體斷言錯誤；磁碟剩1.4GB、inode充足。改用有界3GB記憶體/tmp與8GB容器記憶體，保持同一來源、素材、seed條件及測試命令重跑，不清理其他專案。路由未命中額外Docker支線，沿既有規格與AGENTS的隔離驗證契約。

有界tmpfs先明示exec並以清冊套件驗證可執行，最後全套通過。五份GUI暫存執行檔與本任務Go快取已清除，共釋出957156211 bytes；來源、雜湊、PNG與測試收據保留，沒有錄影。

來源提交`3e1abc519fd6a125ea8854db24eaaa0c0b10b967`已快轉整合main、推送並核對真正遠端HEAD相同。#258與#260依提交及驗收結案，#201／#202／#204已改寫目前HD範圍與下一閘門。官方pull／render／verify為20條開啟、13條人工、可能已完成0；243條教訓及125守衛通過。主要文件連結2924條無缺件，986份來源綁定保持，新程式UID/GID1000，輸出無root-owned或.md目錄，無本批FD2容器殘留。Release與私人素材保持；本次後續提交只同步現況及工作清單。完整HD仍未完成。

## 2026-10-19 計時戰場對話保留HD世界

穩定經驗文字原已沿HD黑體，缺口是完整indexed開收框覆蓋HD人物。新增來源格網覆蓋與FFEC計時對話圖層，首次HD Draw保存世界／UI，原生時鐘照常、忠實路徑保持；F9與對話結束隨處理器丟棄。沒有新美術、規則、格式或翻譯。

CPU完整來源格網、零色、背景快照與原子拒收通過。FFEE文字窗口超出格網，因此僅接已證實FFEC，不猜其餘caller。相關現代／原生對話／升級／存讀檔68.504秒與三資料套件通過，未重跑整套Go。章前只把亞雷斯EXP0改99，實際普通攻擊升Lv2／EXP30；37張主要GPU、6張對話、4張走行、F2與來源／兩主題保存bytes相同、新程序LOAD通過。獨立框角驗證2張開框與2張收框，1張跨階段不可比；HD來源8296核心像素零差異。

r1工具誤等第二次攻擊的經驗，改驗固定第一次，不改seed。r2首張開框仍是前段indexed表面，audit不等於GPU fence。r3等待也套在短收框，因而漏擷取；改只等開框交界，先PPM後讀狀態再PNG。r4全部通過；實際框角分級不要求audit索引與畫面逐幀相同。原始失敗與一張不可比樣本均保留。

[主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的timed_dialogue_followup_20261019保存來源／衍生槽SHA、唯一差異、來源綁定、測試、GPU與重跑命令。完整HD、地形、其他演出與平台仍待完成；沒有原版oracle、matching或整章E2。私人素材及Release保持，沒有錄影。提交與清理結果於本批最後追加。

提交前官方pull／render／verify為20項開啟、13項人工、可能已完成0。244條教訓、126守衛與2934本地連結通過，991份來源綁定保持；新輸出UID/GID1000，無root-owned與.md目錄。兩份暫存執行檔、PPM及本任務快取已清除，釋出293281361 bytes，沒有錄影與本批容器殘留。

來源提交`b086b2c3db01bfb52673e3d9180b6f1d2233aff0`已從獨立分支快轉main，推送後真正遠端HEAD核同值。#201／#202／#204已同步計時對話成果與尚未驗證caller，三項保持開啟。官方快照再次pull／render／verify為20項、13人工、可能已完成0；後續提交僅同步來源及工單。完整HD仍未完成。
