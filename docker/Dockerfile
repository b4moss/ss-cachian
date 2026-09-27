# syntax=docker/dockerfile:1
# ss-cachian dev image: Go 1.26 + act. Firestore Emulator runs as a compose service.

ARG DEBIAN_VERSION=bookworm
FROM debian:${DEBIAN_VERSION}-slim

ARG GO_VERSION=1.26.7
ARG ACT_VERSION=0.2.82

ENV DEBIAN_FRONTEND=noninteractive \
    LANG=C.UTF-8 \
    GOPATH=/go \
    PATH=/usr/local/go/bin:/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

WORKDIR /workspace

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        git \
        build-essential \
        python3 \
        unzip \
    && rm -rf /var/lib/apt/lists/*

# Go
RUN arch="$(dpkg --print-architecture)" \
    && case "$arch" in \
         amd64) goarch=amd64 ;; \
         arm64) goarch=arm64 ;; \
         *) echo "unsupported arch: $arch" >&2; exit 1 ;; \
       esac \
    && curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${goarch}.tar.gz" \
         -o /tmp/go.tgz \
    && rm -rf /usr/local/go \
    && tar -C /usr/local -xzf /tmp/go.tgz \
    && rm /tmp/go.tgz \
    && ln -sf /usr/local/go/bin/go /usr/local/bin/go \
    && ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt

# nektos/act (local Actions smoke)
RUN arch="$(dpkg --print-architecture)" \
    && case "$arch" in \
         amd64) actarch=x86_64 ;; \
         arm64) actarch=arm64 ;; \
         *) echo "unsupported arch: $arch" >&2; exit 1 ;; \
       esac \
    && curl -fsSL "https://github.com/nektos/act/releases/download/v${ACT_VERSION}/act_Linux_${actarch}.tar.gz" \
         -o /tmp/act.tgz \
    && tar -C /usr/local/bin -xzf /tmp/act.tgz act \
    && rm /tmp/act.tgz \
    && act --version

RUN printf '%s\n' \
      'export GOPATH=/go' \
      'export PATH="/usr/local/go/bin:/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"' \
      'export FIRESTORE_EMULATOR_HOST="${FIRESTORE_EMULATOR_HOST:-firestore:8080}"' \
      > /etc/profile.d/ss-cachian-path.sh \
    && chmod 644 /etc/profile.d/ss-cachian-path.sh

RUN useradd --create-home --shell /bin/bash --uid 1000 ubuntu \
    && mkdir -p /go \
    && chown -R ubuntu:ubuntu /workspace /go /home/ubuntu \
    && printf '%s\n' '. /etc/profile.d/ss-cachian-path.sh' >> /home/ubuntu/.bashrc

USER ubuntu
ENV HOME=/home/ubuntu \
    PATH=/usr/local/go/bin:/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin \
    FIRESTORE_EMULATOR_HOST=firestore:8080

CMD ["sleep", "infinity"]
