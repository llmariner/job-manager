set -euo pipefail
set -x

# Download the model and the training file.
mkdir base-model

{{ range $path, $url := .BaseModelURLs }}
mkdir -p $(dirname base-model/{{ $path }})
curl {{ $.CurlFlags }} --output base-model/{{ $path }} "{{ $url }}"
{{ end }}

mkdir dataset/
curl {{ .CurlFlags }} --output dataset/train.json "{{ .TrainingFileURL }}"
{{if .ValidationFileURL }}
curl {{ .CurlFlags }} --output dataset/test.json "{{ .ValidationFileURL }}"
{{ end }}

mkdir output

accelerate launch \
  --mixed_precision=no \
  --num_processes={{ .NumProcessors }} \
  --num_machines=1 \
  --num_cpu_threads_per_process=1 \
  --dynamo_backend=no \
  ./train.py \
  --model=./base-model \
  --method={{ .Method }} \
  --dataset=./dataset \
  --output=./output {{ .AdditionalSFTArgs }}

python ./convert-lora-to-ggml.py ./output

# We don't need the checkpoint files.
rm -rf output/checkpoint-*

# Upload all files under the "output" directory.
find output -type f -exec curl {{ .CurlFlags }} --request POST {{ .OutputModelPresignFlags }} -F file=@{} "{{ .OutputModelURL }}" \;
