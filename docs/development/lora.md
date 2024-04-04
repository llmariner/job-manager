# Examples of Fine-Tuning with LoRA or QLoRA

## Google Gemma Example

Link: https://huggingface.co/blog/gemma/?utm_source=agd&utm_medium=referral&utm_campaign=view-on-huggingface&utm_content=

### Set up

```bash
sudo apt install python3
sudo apt install python3-pip
sudo apt install python3.10-venv

python3 -m venv .venv
source .venv/bin/activate
pip install -U "transformers==4.38.1" --upgrade
pip install torch
pip install bitsandbytes
pip install accelerate
```

### Check if a model runs

```bash
export HUGGING_FACE_HUB_TOKEN=<Hugging Face API key>
```

Run the following Python code:

```python
from transformers import AutoTokenizer, pipeline
import torch

model = "google/gemma-2b-it"
device_map='cuda' # Change this to `mps` to run on Apple Sillicon.

tokenizer = AutoTokenizer.from_pretrained(
   model,
   device_map=device_map,
)

pipeline = pipeline(
    "text-generation",
    model=model,
    model_kwargs={
        "torch_dtype": torch.float16,
        "quantization_config": {"load_in_4bit": True}
    },
    device_map=device_map,
)

messages = [
    {"role": "user", "content": "Who are you? Please, answer in pirate-speak."},
]
prompt = pipeline.tokenizer.apply_chat_template(messages, tokenize=False, add_generation_prompt=True)
outputs = pipeline(
    prompt,
    max_new_tokens=256,
    do_sample=True,
    temperature=0.7,
    top_k=50,
    top_p=0.95
)
print(outputs[0]["generated_text"][len(prompt):])
```

### Run a fine-tuning

```bash
pip install trl peft
git clone https://github.com/huggingface/trl
cd trl

max_steps=-1 # or some small number to finish quickly

accelerate launch --config_file examples/accelerate_configs/single_gpu.yaml --num_processes=1 \
    examples/scripts/sft.py \
    --model_name google/gemma-2b \
    --dataset_name OpenAssistant/oasst_top1_2023-08-25 \
    --per_device_train_batch_size 2 \
    --gradient_accumulation_steps 1 \
    --max_steps=${max_steps} \
	--learning_rate 2e-4 \
    --save_steps 20_000 \
    --use_peft \
    --lora_r 16 --lora_alpha 32 \
    --lora_target_modules q_proj k_proj v_proj o_proj \
    --load_in_4bit \
    --output_dir gemma-finetuned-openassistant
```

## MLX Example

Link: https://github.com/ml-explore/mlx-examples/tree/main/lora

```bash
python3 -m venv .venv
source .venv/bin/activate

git clone https://github.com/ml-explore/mlx-examples.git
cd mlx-examples/lora
pip install -r requirements.txt
pip install torch

# Edit util.py and add the `token` parameter to the call of `fetch_from_hub()` and pass the API key.

# Change load_weights in .venv/lib/python3.11/site-packages/mlx/nn/layers/base.py
# to ignore "lm_head.weight" from missing parameter.
# See https://github.com/vllm-project/vllm/issues/3323 and https://github.com/vllm-project/vllm/pull/3553/files.

python convert.py --hf-path google/gemma-2b -q

python lora.py \
  --model google/gemma-2b \
  --data data \
  --train \
  --batch-size 1 \
  --lora-layers 1 \
  --iters 10
```

## Local AI Example

Link: https://localai.io/docs/advanced/fine-tuning/

```bash
git clone http://github.com/mudler/LocalAI
cd LocalAI/examples/e2e-fine-tuning

# Install axolotl and dependencies
git clone https://github.com/OpenAccess-AI-Collective/axolotl && pushd axolotl && git checkout 797f3dd1de8fd8c0eafbd1c9fdb172abd9ff840a && popd #0.3.0
pip install packaging
pushd axolotl && pip install -e '.[flash-attn,deepspeed]' && popd

# https://github.com/oobabooga/text-generation-webui/issues/4238
pip install https://github.com/Dao-AILab/flash-attention/releases/download/v2.3.0/flash_attn-2.3.0+cu117torch2.0cxx11abiFALSE-cp310-cp310-linux_x86_64.whl

pip install accelerate

accelerate config default

# Optional pre-tokenize (run only if big dataset)
python -m axolotl.cli.preprocess axolotl.yaml

# Fine-tune
accelerate launch -m axolotl.cli.train axolotl.yaml

# Merge lora
python3 -m axolotl.cli.merge_lora axolotl.yaml --lora_model_dir="./qlora-out" --load_in_8bit=False --load_in_4bit=False

# Convert to gguf
git clone https://github.com/ggerganov/llama.cpp.git
pushd llama.cpp && make LLAMA_CUBLAS=1 && popd

# We need to convert the pytorch model into ggml for quantization
# It creates 'ggml-model-f16.bin' in the 'merged' directory.
pushd llama.cpp && python convert.py --outtype f16 \
    ../qlora-out/merged/pytorch_model-00001-of-00002.bin && popd

# Start off by making a basic q4_0 4-bit quantization.
# It's important to have 'ggml' in the name of the quant for some
# software to recognize it's file format.
pushd llama.cpp &&  ./quantize ../qlora-out/merged/ggml-model-f16.gguf \
    ../custom-model-q4_0.bin q4_0
```
