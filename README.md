# job-manager

Job Manage manages fine-tuning jobs.

The following commands build a binary and a Docker container.

```bash
make build-server
docker build --build-arg TARGETARCH=amd64 -t job-manager-server:latest -f build/server/Dockerfile .
```

TODO(kenji): Just build a binary inside the container.

# TODO
- Implement the `store` package
- Implement the `server` package
- Implement a simple dipatcher/executor for running a fine-tuning job
- Design integratation with the model registry
- Design dataset management
- Design GPU management & scheduling 
