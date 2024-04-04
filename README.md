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
