mkdir /models/adapter;
accelerate launch \
  --config_file=./single_gpu.yaml \
  --num_processes=1 \
  ./sft.py \
  --model_name=google/gemma-2b \
  --dataset_name=OpenAssistant/oasst_top1_2023-08-25 \
  --per_device_train_batch_size=2 \
  --gradient_accumulation_steps=1 \
  --max_steps=100 \
  --learning_rate=2e-4 \
  --save_steps=20_000 \
  --use_peft \
  --lora_r=16 \
  --lora_alpha=32 \
  --lora_target_modules q_proj k_proj v_proj o_proj \
  --load_in_4bit \
  --output_dir=./output &&
python ./convert-lora-to-ggml.py ./output &&
cp ./output/ggml-adapter-model.bin /models/adapter/
