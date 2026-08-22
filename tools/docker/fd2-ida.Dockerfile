# Local FD2 reverse-engineering overlay for the user-authorized IDA Pro image.
#
# Build only after `/home/anr2/ida_94_official/dist` has loaded the private
# `ida-pro-9.4-ver2:civ1-py312-v1` image.  The private base contains the user's
# license and working IDAPython selection; this public Dockerfile contains no
# installer, game binary, database, or license and must never be published as
# a built image.
FROM ida-pro-9.4-ver2:civ1-py312-v1

USER root
COPY --chmod=0755 tools/docker/fd2-ida-entrypoint.sh /usr/local/bin/fd2-ida-entrypoint

# IDA 9.4 的批次模式仍會載入 Qt 平台程式庫。私人基底只帶 IDA 本體，若缺少
# 這些執行期函式庫，連 `-A -S...` 都會在進入 IDAPython 前失敗。只安裝
# headless/offscreen 所需的共享函式庫；不在映像內加入原版檔案或授權內容。
RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        libdbus-1-3 \
        libegl1 \
        libfontconfig1 \
        libfreetype6 \
        libgl1 \
        libglib2.0-0 \
        libopengl0 \
        libx11-6 \
        libxext6 \
        libxkbcommon0 \
        libxtst6 \
    && rm -rf /var/lib/apt/lists/*

# The private base keeps its seed files under /root/.idapro.  Permit traversal
# only so the non-root entrypoint can copy the already-readable seed files into
# its ephemeral HOME.  Runtime analysis and bind-mount outputs remain UID/GID
# 1000:1000; the private seed directory is never mounted into the repository.
RUN chmod 0711 /root \
    && chmod 0711 /root/.idapro

USER 1000:1000
ENTRYPOINT ["/usr/local/bin/fd2-ida-entrypoint"]
CMD ["bash"]
