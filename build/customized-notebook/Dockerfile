FROM mirror.gcr.io/cschranz/gpu-jupyter:v1.7_cuda-12.3_ubuntu-22.04_python-only

WORKDIR /build

USER root

RUN curl --silent "https://awscli.amazonaws.com/awscli-exe-linux-x86_64-2.11.20.zip" -o "awscliv2.zip" && \ 
    unzip awscliv2.zip && \
    ./aws/install --bin-dir /usr/local/bin --install-dir /usr/local/aws-cli --update && \
    rm -rf awscliv2.zip aws

RUN curl --silent https://llmariner.ai/get-cli | bash && \
    mv llma /usr/local/bin/llma
