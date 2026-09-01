# FD2 四語全量內容資料

本目錄保存四語內容產生所需的受控術語表。實際翻譯內容位於
`remake/assets/locales/<locale>/content.json`，每包各有 5,176 筆玩家內容出現位置。

狀態區分：

- `zh-Hant` 是從目前 canonical／legacy 受控資料投影的來源文字。
- `zh-Hans`、`en`、`ja` 以可重現的機器初稿起始，逐筆審校後標為 `reviewed`；
  尚未能由原版證據閉合的項目保留 `machine_draft` 或 `blocked`。

官方對話阻擋清冊見 [`review-blockers.json`](review-blockers.json)，目前列出 11 筆：
來源短句截斷、說話者身分衝突及專名衝突。清冊以繁中來源 `string_id` 與原文
綁定；對話本文的嚴格測試不允許清冊外殘留未審校項目。其他介面、實體名稱與
原始來源碎片仍須依各自範圍審校，不可從本清冊推論為已完成。模型產物
只能作為後續譯者的全量初稿，不得在發行說明冒稱已完成官方人工翻譯。NLLB 模型來源及
授權見 [Hugging Face 模型卡](https://huggingface.co/facebook/nllb-200-distilled-600M)（CC BY-NC
4.0）；Argos 模型格式與安裝方法見
[Argos Translate](https://github.com/argosopentech/argos-translate)。模型權重不加入本儲存庫或發行包。

主驗證入口是從儲存庫根目錄執行 `python tools/validate_full_locale_content.py`；阻擋清冊
測試是 `python -m unittest tools.test_locale_review_blockers`。它預設
驗證四個官方內容目錄，證明四包身分、變數與來源完整，並
阻擋已知的英文／日文污染及退化模式；它不能取代譯者的語意、語氣與專名審查。
