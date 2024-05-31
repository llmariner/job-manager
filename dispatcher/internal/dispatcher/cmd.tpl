set -euo pipefail
set -x

# Download the model and the training file.
mkdir base-model

{{ range $path, $url := .BaseModelURLs }}
mkdir -p $(dirname base-model/{{ $path }})
curl --fail --silent --output base-model/{{ $path }} "{{ $url }}"
{{ end }}

mkdir dataset/
curl --fail --silent --output dataset/train.json "{{.TrainingFileURL }}"
{{if .ValidationFileURL }}
curl --fail --silent --output dataset/test.json "{{.ValidationFileURL }}"
{{ end }}

mkdir output

accelerate launch \
  --mixed_precision=no \
  --num_processes={{ .NumProcessors }} \
  --num_machines=1 \
  --num_cpu_threads_per_process=1 \
  --dynamo_backend=no \
  ./sft.py \
  --model=./base-model \
  --dataset=./dataset \
  --output=./output {{ .AdditionalSFTArgs }}

python ./convert-lora-to-ggml.py ./output

curl --fail --silent --request PUT --upload-file output/ggml-adapter-model.bin "{{ .OutputModelURL }}"
