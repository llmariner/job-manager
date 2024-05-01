FROM python:3.12-slim

WORKDIR /build

RUN \
  apt update && \
  apt install -y \
    ca-certificates \
    curl

COPY sft.py .
COPY ggml-adapter-model.bin .
COPY convert-lora-to-ggml.py .
