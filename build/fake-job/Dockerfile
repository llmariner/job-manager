FROM python:3.12-slim

WORKDIR /build

RUN \
  apt update && \
  apt install -y \
    ca-certificates \
    curl

COPY accelerate /bin/accelerate
COPY train.py .
COPY convert-lora-to-ggml.py .
COPY ft-output ft-output
