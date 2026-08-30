FROM python:3.11-slim-bookworm

# 固定轉換與推論執行庫；模型 revision 由產生器固定，並以外掛快取供應。
RUN pip install --no-cache-dir --index-url https://download.pytorch.org/whl/cpu \
      "torch==2.2.2" \
 && pip install --no-cache-dir \
      "numpy<2" \
      "transformers==4.44.2" \
      "sentencepiece==0.2.0" \
      "ctranslate2==4.8.1"
