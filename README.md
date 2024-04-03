# job-manager

Job Manage manages fine-tuning jobs.

# TODO
- Implement a simple dipatcher/executor for running a fine-tuning job
- Design integratation with the model registry
- Design dataset management
- Design GPU management & scheduling

# Running Dispatcher Locally

You can run `dispatcher` locally.

```bash
make build-dispatcher
kubectl port-forward -n postgres service/postgres 5432:5432 &
DB_PASSWORD=ps_password ./bin/dispatcher run --config config.yaml
```

`config.yaml` has the following content:

```yaml
jobPollingInterval: 10s
jobNamespace: default

debug:
  autoMigrate: true
  kubeconfigPath: /Users/kenji/.kube/config

database:
  host: localhost
  port: 5432
  database: job_manager
  username: ps_user
  passwordEnvName: DB_PASSWORD
```

# An Example of Fine-Tuning with LoRA or QLoRA

See https://github.com/ml-explore/mlx-examples/tree/main/lora.


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
