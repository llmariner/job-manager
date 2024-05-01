# Summary

This describes the design of the fine-tuning jobs and open questions.

# Overview

We use [Supervised Fine-tuning Trainer
(SFT)](https://huggingface.co/docs/trl/en/sft_trainer) for
fine-tuning. SFT provides an easy-use API to train LLM models.

The Python script for SFT is invoked with [Accelerate](https://huggingface.co/docs/accelerate/en/index).

The input to the Python script is an LLM model and trainint
data. Validation data can also be optionally provided.  The script
generates a LoRA adapter, which we convert to the GGUF format with
[the `convert-lora-to-ggml.py`
script](https://github.com/ggerganov/llama.cpp/blob/master/convert-lora-to-ggml.py),
which is provided by
[llama.cpp](https://github.com/ggerganov/llama.cpp).

# Mising Features

- Checkpointing
- Metrics/events
- Distributed training

# Open Questions

SFT and Accelerate requires the following input configuratoins:

- Accelerate configuration (e.g., number of GPUs to be allocated)
- Model configuration (e.g., Attention implementation, Torch dtype)
- Quantization configuration
- Training configuration (e.g., batch size, number of epocs, learning rate, checkpointing)
- LoRA configuration

Some of the hyperparameters can be manually specified by end users, but we also need to provide sane defaults.
The optimal values depend on underlying envrionment, model, etc. and we need to find some more exercise to find
right parameters.
