FROM python:3.12-slim

WORKDIR /build

RUN \
  apt update && \
  apt install -y \
    ca-certificates \
    curl

COPY sft.py .
COPY convert-lora-to-ggml.py .

COPY ggml-adapter-model.bin .
COPY adapter_model.safetensors .
COPY tokenizer_config.json .
COPY special_tokens_map.json .
COPY tokenizer.json .
COPY training_args.bin .
COPY adapter_config.json .
