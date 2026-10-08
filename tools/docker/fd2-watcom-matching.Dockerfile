# 建置 context 是 fd2_matching_toolchain.py 產生的 toolchain/，不包含原版或 IDA。
# docker build --network none -f tools/docker/fd2-watcom-matching.Dockerfile \
#   -t fd2-watcom-matching:2.0-20261001-r1 work/matching-pilot-20261008/toolchain
FROM python:3.13-bookworm@sha256:933b46a028fd786c9c3d426ebabc237e29a15912231ea8de576e95f0e4f41a4c
COPY --chown=1000:1000 . /opt/watcom/
USER 1000:1000
RUN python -c 'import hashlib,pathlib; p=pathlib.Path("/opt/watcom/binl64"); locks={"wcc386":"86eeb5103035c1bfcaed3a1c93f87d82d74e7a31d9dc43fd8ade8251025ceb11","wdis":"c6d20aa8aeb550e780c4a72a367f4446225ad4ab4b7d4e8f18203c4271805015","wlink":"97ff1fd068c48de2745d25b0f85057e485b3eee4df0fcf6570f1839e5d315fb5","wlib":"67bfcd5fd28fb035d0eebf66a0767af45ebc276bf8f8619e1ec23c0580ce1899"}; assert all(hashlib.sha256((p/n).read_bytes()).hexdigest()==s for n,s in locks.items())'
ENV WATCOM=/opt/watcom
ENV PATH=/opt/watcom/binl64:${PATH}
ENV PYTHONDONTWRITEBYTECODE=1
WORKDIR /work
LABEL org.opencontainers.image.source="https://github.com/open-watcom/open-watcom-v2/releases/tag/2026-10-01-Build"
LABEL org.opencontainers.image.description="FD2 three-function matching experiment; candidate compiler, not identified original toolchain"
