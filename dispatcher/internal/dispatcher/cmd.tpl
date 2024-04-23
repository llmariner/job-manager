set -euo pipefail
set -x

# Download the model and the training file.
mkdir base-model

{{ range $path, $url := .BaseModelURLs }}
mkdir -p $(dirname base-model/{{ $path }})
curl --output base-model/{{ $path }} "{{ $url }}"
{{ end }}

mkdir dataset/
curl -o dataset/train.json "{{.TrainingFileURL }}"
{{if .ValidationFileURL }}
curl -o dataset/test.json "{{.ValidationFileURL }}"
{{ end }}

mkdir output

{{if .UseFakeJob }}
cp ./ggml-adapter-model.bin ./output/
{{ else }}
accelerate launch \
  --config_file=./single_gpu.yaml \
  --num_processes=1 \
  ./sft.py \
  --model_name=./base-model \
  --dataset_name=dataset \
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
  --output_dir=./output

python ./convert-lora-to-ggml.py ./output
{{ end }}

curl --request PUT --upload-file output/ggml-adapter-model.bin "{{ .OutputModelURL }}"
