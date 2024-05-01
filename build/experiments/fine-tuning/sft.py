# The original file is located at https://github.com/huggingface/trl/blob/main/examples/scripts/sft.py
#
# The code is slightly modified to remove the logic for using a rich progress bar.

# flake8: noqa
# Copyright 2023 The HuggingFace Inc. team. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import logging
import os
import argparse

import torch

from datasets import load_dataset
from transformers import AutoTokenizer, TrainingArguments, BitsAndBytesConfig
from peft import LoraConfig

from tqdm.rich import tqdm

from trl import (
    SFTTrainer,
    get_kbit_device_map,
)

# For progress bars.
tqdm.pandas()

if __name__ == "__main__":
    parser = argparse.ArgumentParser("sft.py", description="A script to train a model using SFT.")
    parser.add_argument("--model", help="Model path.", type=str)
    parser.add_argument("--dataset", help="Dataset path.", type=str)
    parser.add_argument("--output", help="Output path.", type=str)
    args = parser.parse_args()
    print(args)

    quantization_config = BitsAndBytesConfig(
        load_in_4bit=True,
        bnb_4bit_compute_dtype=torch.float16,
        bnb_4bit_quant_type="nf4"
    )

    model_kwargs = dict(
        # The specific model version to use (can be a branch name, tag name or commit id)."
        # revision="main",
        # Trust remote code when loading a model.
        # trust_remote_code=False,
        # Which attention implementation to use; you can run --attn_implementation=flash_attention_2,
        # in which case you must install this manually by running `pip install flash-attn --no-build-isolation`
        attn_implementation=None,
        # Override the default `torch.dtype` and load the model under this dtype. If `auto` is passed,
        # the dtype will be automatically derived from the model's weights."
        torch_dtype='auto',
        # use_cache=False,
        device_map=get_kbit_device_map(),
        quantization_config=quantization_config,
    )

    # TODO(kenji): Revisit these parameters.
    training_args = TrainingArguments(
        output_dir=args.output,
        overwrite_output_dir=True,
        num_train_epochs=3,
        # batch size per device during training
        per_device_train_batch_size=2,
        # number of steps before performing a backward/update pass
        gradient_accumulation_steps=2,
        # use gradient checkpointing to save memory
        gradient_checkpointing=True,
        # save checkpoint every epoch
        save_strategy="epoch",
        logging_steps=10,
        # learning rate, based on QLoRA paper
        learning_rate=2e-4,
        # warmup ratio based on QLoRA paper
        warmup_ratio=0.03,
        # max gradient norm based on QLoRA paper
        max_grad_norm=0.3,
        # use constant learning rate scheduler
        lr_scheduler_type="constant",
        # use bfloat16 precision
        bf16=True,
        # use tf32 precision
        tf32=True,
        report_to="none",
    )

    raw_datasets = load_dataset(args.dataset)
    train_dataset = raw_datasets["train"]
    eval_dataset = raw_datasets["test"] if "test" in raw_datasets else None

    tokenizer = AutoTokenizer.from_pretrained("./base-model", use_fast=True)
    tokenizer.pad_token = tokenizer.eos_token

    # TODO(kenji): Revisit these parameters.
    peft_config = LoraConfig(
        r=16,
        lora_alpha=32,
        lora_dropout=0.05,
        bias="none",
        task_type="CAUSAL_LM",
        target_modules=None,
        modules_to_save=None,
    )

    trainer = SFTTrainer(
        model=args.model,
        model_init_kwargs=model_kwargs,
        args=training_args,
        train_dataset=train_dataset,
        eval_dataset=eval_dataset,
        # The name of the text field of the dataset, in case this is passed by a user,
        # the trainer will automatically create a `ConstantLengthDataset` based on the `dataset_text_field` argument.
        dataset_text_field=None,
        # Used only in case `dataset_text_field` is passed. This argument is used by the `ConstantLengthDataset`
        # to pack the sequences of the dataset.
        packing=False,
        # The maximum sequence length to use for the `ConstantLengthDataset`
        # and for automatically creating the Dataset.
        # Defaults to min of the smaller of the `tokenizer.model_max_length` and `1024`.
        max_seq_length=None,
        tokenizer=tokenizer,
        peft_config=peft_config,
        callbacks=None,
    )

    trainer.train()

    trainer.save_model(args.output)
