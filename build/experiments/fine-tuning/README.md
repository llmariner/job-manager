This is an experimetnal Docker image that can be used to run a fine-tuning job.

Here is an example pod spec:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: gpu-test
spec:
  restartPolicy: OnFailure
  containers:
  - name: main
    image: sft:latest
    imagePullPolicy: Never
    command: ["accelerate"]
    args:
    - launch
    - --config_file=./single_gpu.yaml
    - --num_processes=1
    - ./sft.py
    - --model_name=google/gemma-2b
    - --dataset_name=OpenAssistant/oasst_top1_2023-08-25
    - --per_device_train_batch_size=2
    - --gradient_accumulation_steps=1
    - --max_steps=1000
    - --learning_rate=2e-4
    - --save_steps=20_000
    - --use_peft
    - --lora_r=16
    - --lora_alpha=32
    - --lora_target_modules
    - q_proj
    - k_proj
    - v_proj
    - o_proj
    - --load_in_4bit
    - --output_dir=gemma-finetuned-openassistant
    env:
    - name: HUGGING_FACE_HUB_TOKEN
      value: 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
    resources:
      limits:
        nvidia.com/gpu: 1
```
