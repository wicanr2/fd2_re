# FD2 四語全量內容資料

本目錄保存四語內容產生所需的受控術語表。實際翻譯內容位於
`remake/assets/locales/<locale>/content.json`，每包各有 5,176 筆玩家內容出現位置。

狀態區分：

- `zh-Hant` 是從目前 canonical／legacy 受控資料投影的來源文字。
- `zh-Hans` 是 OpenCC `t2s` 可重現初稿。
- `en` 是 Argos `translate-zt_en-1_9` 離線模型初稿。
- `ja` 是 `facebook/nllb-200-distilled-600M` 固定 revision
  `f8d333a098d19b4fd9a8b18f94170487ad3f821d` 的繁中直譯初稿。

英文與日文的每筆 `status` 都是 `machine_draft`，不是人工校譯文字。模型產物
只能作為後續譯者的全量初稿，不得在發行說明冒稱已完成官方人工翻譯。NLLB 模型來源及
授權見 [Hugging Face 模型卡](https://huggingface.co/facebook/nllb-200-distilled-600M)（CC BY-NC
4.0）；Argos 模型格式與安裝方法見
[Argos Translate](https://github.com/argosopentech/argos-translate)。模型權重不加入本儲存庫或發行包。

主驗證入口是從儲存庫根目錄執行 `python tools/validate_full_locale_content.py`。它預設
驗證四個官方內容目錄，證明四包身分、變數與來源完整，並
阻擋已知的英文／日文污染及退化模式；它不能取代譯者的語意、語氣與專名審查。
