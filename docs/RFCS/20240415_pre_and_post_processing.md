# Summary

This describes the design of the pre-processing and post-processing of fine-tuning jobs.

# Requirements

Here is the high-level flow of a fine-tuning job.

1. Download a training file and a base model from the object store.
2. Run the training and generate a LoRA adapter.
3. Convert the format of the LoRA adapter to GGML.
4. Upload the generated model to the object store.

Step 2 needs to run on a pod where GPU resources are allocated. Step 1, 3, and 4
can run on the dispatcher or or the fine-tuning job, but accessing the object store
requires the credential of the object store.

One approach is to run all the steps in a single pod, but we want to avoid exposing
the object store credential to the pod if possible. The reason is that eventually
we might allow end users to exec into the fine-tuning pods for debugging. In such a case,
we don't want the users to access the object store with the credential.

# Design

We use pre-signed URLs to allow a pod to access the object store without exposing the credential to the pod.
Here is an example flow:

1. Dispatcher creates pre-signed URLs for a training file, a base model, and an output model. The URLs are injected to a
   fine-tuning pod as commandline arguments.
2. The pod downloads the training file and the model with the pre-signed URLs and stores them locally.
3. The pod runs the training and converts the LoRA adapter to GGML.
4. The pod uses the pre-signed URL to upload the generated model.

The pod can have multiple containers (including init containers) to run the above steps.

# Limitations

We assume that the underlying object store supports pre-signed URLs (e.g., S3, MinIO). If we
need to support an object store that doesn't provide pre-signed URLs, we need to revisit the design (and simply
inject credentials to the pod).
