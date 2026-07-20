# Isolated local reverse-engineering toolchain; never install capstone on host.
FROM python:3.12-slim

RUN python -m pip install --no-cache-dir --disable-pip-version-check capstone==5.0.3

WORKDIR /w
ENTRYPOINT ["python3"]
