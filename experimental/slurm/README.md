# Slurm

This directory contains experimental code related to Slurm.

## Building a Slurm cluster

Follow https://github.com/llmariner/llmariner/tree/main/hack/slurm to
use Slinky to build a Slurm cluster on top of Kubernetes.

## Server

This is a HTTP server that implements Slurm Rest endpoint. It uses the OpenAI spec in
the `slurm-client` repo ([link](https://github.com/SlinkyProject/slurm-client/blob/main/api/v0040/oapi-codegen-config.yaml))
and uses [`oapi-codegen`](https://github.com/oapi-codegen/oapi-codegen) to generate the Go code.


To run the server,

```bash
make build-server
./bin/server run --config hack/config.yaml
```

The servert will start at port 8080. You can hit the endpoint with `curl`.

```bash
curl http://localhost:8080/slurm/v0.0.40/jobs/
```
