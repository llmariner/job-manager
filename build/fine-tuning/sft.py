# The original file is located at https://github.com/huggingface/trl/blob/main/examples/scripts/sft.py
#
# The code is slightly modified to remove the logic for using a rich progress bar.

import logging
import os
import argparse

import torch

from datasets import load_dataset
from transformers import (
    AutoTokenizer,
    AutoModelForCausalLM,
    BitsAndBytesConfig,
)

from peft import LoraConfig

from tqdm.rich import tqdm

from trl import (
    SFTTrainer,
    SFTConfig,
    get_kbit_device_map,
)

# For progress bars.
tqdm.pandas()

# --------------------------------------------------------------------------------
# Helpers
# --------------------------------------------------------------------------------

# Create preprocessor to ensure training data is well-formed
def build_chat_preprocessor(tokenizer):
    eos_id = tokenizer.eos_token_id

    # Create a closure with tokenizer and eos to preprocess the dataset on the specific tokenizer
    def _preprocess(example):
        """Turn a list‑of‑messages into input_ids + single EOS."""
        text = tokenizer.apply_chat_template(
            example["messages"],  # expects list[dict]
            add_generation_prompt=False,
            tokenize=False,
        )
        ids = tokenizer(text, add_special_tokens=False)["input_ids"]
        ids.append(eos_id)
        return {"input_ids": ids}

    return _preprocess

# --------------------------------------------------------------------------------
# Main
# --------------------------------------------------------------------------------
if __name__ == "__main__":
    parser = argparse.ArgumentParser("sft.py", description="A script to train a model using SFT.")
    parser.add_argument("--model", help="Model path.", type=str)
    parser.add_argument("--dataset", help="Dataset path.", type=str)
    parser.add_argument("--output", help="Output path.", type=str)
    parser.add_argument("--report_to", help="The integration to report the results and logs to.", default="none", type=str)
    parser.add_argument("--wandb_project", help="Name of W&B project.", type=str)

    parser.add_argument("--use_bnb_quantization", help="Use BitesAndBytes quantization", default=False, type=bool)

    # TODO(kenji): Revisit the default values.
    parser.add_argument("--learning_rate", help="Learning rate.", default=2e-4, type=float, nargs="?")
    parser.add_argument("--num_train_epochs", help="Number of training epocs.", default=3, type=int, nargs="?")
    parser.add_argument("--per_device_train_batch_size", help="Batch size per training.", default=2, type=int, nargs="?")

    args = parser.parse_args()

    if args.report_to == "wandb":
        os.environ["WANDB_PROJECT"] = args.wandb_project

    # ------------------------------------------------------------------
    # Tokenizer – set distinct PAD and right-pad. This isn't strictly necessary for all models, but
    # it is for some models (e.g. Llama). It also sets the padding side to "right" for all models to be consistent
    # ------------------------------------------------------------------
    tokenizer = AutoTokenizer.from_pretrained(args.model, use_fast=True)
    if tokenizer.pad_token is None:
        tokenizer.pad_token = "<|finetune_right_pad_id|>"
    tokenizer.padding_side = "right"

    # ------------------------------------------------------------------
    # Load model (with optional 4‑bit quant)
    # ------------------------------------------------------------------
    quantization_config = None
    if args.use_bnb_quantization:
        quantization_config = BitsAndBytesConfig(
            load_in_4bit=True,
            bnb_4bit_compute_dtype=torch.float16,
            bnb_4bit_quant_type="nf4"
        )

    model = AutoModelForCausalLM.from_pretrained(
        args.model,
        attn_implementation=None,
        # Override the default `torch.dtype` and load the model under this dtype. If `auto` is passed,
        # the dtype will be automatically derived from the model's weights."
        torch_dtype="auto",
        # Setting this to False as `use_cache=True` is incompatible with gradient checkpointing.
        use_cache=False,
        device_map=get_kbit_device_map(),
        quantization_config=quantization_config,
    )
    # Ensure the model and tokenizer agree on the pad token.
    model.config.pad_token_id = tokenizer.pad_token_id

    raw_datasets = load_dataset(args.dataset)
    train_dataset = raw_datasets["train"]
    eval_dataset = raw_datasets["test"] if "test" in raw_datasets else None

    preprocess_fn = build_chat_preprocessor(tokenizer)
    num_proc = min(4, os.cpu_count() or 1)

    # Preprocess the datasets with eos_id at the end so even if a model doesn't do it by default we can still train it
    # and it will learn to stop at the right time.
    train_dataset = train_dataset.map(preprocess_fn, remove_columns=train_dataset.column_names, num_proc=num_proc)
    if eval_dataset is not None:
        eval_dataset = eval_dataset.map(preprocess_fn, remove_columns=eval_dataset.column_names, num_proc=num_proc)

    # TODO(kenji): Revisit these parameters.
    training_args = SFTConfig(
        output_dir=args.output,
        overwrite_output_dir=True,
        num_train_epochs=args.num_train_epochs,
        # batch size per device during training
        per_device_train_batch_size=args.per_device_train_batch_size,
        # number of steps before performing a backward/update pass
        gradient_accumulation_steps=2,
        # use gradient checkpointing to save memory
        gradient_checkpointing=True,
        # The use_reentrant parameter need be passed explicitly. use_reentrant=False is recommended.
        gradient_checkpointing_kwargs={"use_reentrant": False},
        # save checkpoint every epoch
        save_strategy="epoch",
        logging_steps=10,
        # learning rate, based on QLoRA paper
        learning_rate=args.learning_rate,
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
        report_to=args.report_to,

        # The name of the text field of the dataset, in case this is passed by a user,
        # the trainer will automatically create a `ConstantLengthDataset` based on the `dataset_text_field` argument.
        dataset_text_field=None,
        # Used only in case `dataset_text_field` is passed. This argument is used by the `ConstantLengthDataset`
        # to pack the sequences of the dataset.
        packing=False,
        # The maximum sequence length to use for the `ConstantLengthDataset`
        # and for automatically creating the Dataset.
        # Defaults to min of the smaller of the `tokenizer.model_max_length` and `1024`.
        max_seq_length=1024,
    )

    # TODO(kenji): Revisit these parameters.
    peft_config = LoraConfig(
        r=16,
        lora_alpha=32,
        lora_dropout=0.05,
        bias="none",
        task_type="CAUSAL_LM",
        target_modules=["q_proj", "o_proj", "k_proj", "v_proj", "gate_proj", "up_proj", "down_proj"],
        modules_to_save=None,
    )

    trainer = SFTTrainer(
        model=model,
        args=training_args,
        train_dataset=train_dataset,
        eval_dataset=eval_dataset,
        tokenizer=tokenizer,
        peft_config=peft_config,
        callbacks=None,
    )

    trainer.train()

    trainer.save_model(args.output)