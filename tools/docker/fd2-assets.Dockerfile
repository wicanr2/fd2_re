# 可重現的 FD2 素材解碼／清冊工具鏈。
# 原版檔案與輸出均由 runtime mount 提供，不會烘進映像。
FROM python:3.12-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends vorbis-tools \
    && rm -rf /var/lib/apt/lists/*

RUN python -m pip install --no-cache-dir --disable-pip-version-check Pillow==11.3.0

WORKDIR /repo
ENTRYPOINT ["python3"]
