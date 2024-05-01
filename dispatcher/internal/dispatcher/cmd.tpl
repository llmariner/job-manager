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
  --mixed_precision=no \
  --num_processes=1 \
  --num_machines=1 \
  --num_cpu_threads_per_process=1 \
  --dynamo_backend=no \
  ./sft.py \
  --model=./base-model \
  --dataset=./dataset \
  --output=./output {{ .AdditionalSFTArgs }}


python ./convert-lora-to-ggml.py ./output
{{ end }}

curl --request PUT --upload-file output/ggml-adapter-model.bin "{{ .OutputModelURL }}"
