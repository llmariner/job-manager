# job-manager

Job Manage manages fine-tuning jobs.

# TODO
- Design integratation with the model registry
- Design dataset management
- Design GPU management & scheduling

# Running Dispatcher Locally

You can run `dispatcher` locally.

```bash
make build-dispatcher
./bin/dispatcher run --config config.yaml
```

`config.yaml` has the following content:

```yaml
jobPollingInterval: 10s
jobNamespace: default
job:
  image: llm-operator/experiments-fake-job
  version: latest
  numGpus: 0

debug:
  kubeconfigPath: /Users/kenji/.kube/config
  standalone: true
  sqlitePath: /tmp/job_manager.db
```

You can then connect to the DB and create a job.

```bash
sqlite3 /tmp/job_manager.db
# Run the query inside the database.
insert into jobs
  (job_id, message, state, tenant_id, version, created_at, updated_at)
values
  ('my-job', '', 'queued', 'my-tenant', 0, time('now'), time('now'));
```
