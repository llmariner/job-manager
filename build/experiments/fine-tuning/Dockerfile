FROM python:3.12-slim

WORKDIR /build

RUN \
  apt update && \
  apt install -y \
    ca-certificates \
    gcc \
    curl

COPY requirements.txt .
RUN pip install -r requirements.txt

COPY sft.py .
COPY single_gpu.yaml .
COPY convert-lora-to-ggml.py .
