# cre-simulator: runs the real workflow code through `cre workflow simulate` against the local gateway.
# Started only by `docker compose --profile cre up`; never deploys, logs in or spends funds (docs/cre/SPEC.md section 10).
FROM oven/bun:1-debian

RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates bash && rm -rf /var/lib/apt/lists/*

# The CRE CLI from a pinned GitHub release, verified against the SHA-256 published in that release's checksums.txt
# (smartcontractkit/cre-cli v1.37.0). Bump both the version and the checksums together.
ARG CRE_CLI_VERSION=v1.37.0
ARG CRE_CLI_SHA256_AMD64=1e660e955be607bca3ae683d5f264bb354252f6af657f0d86e56108438536503
ARG CRE_CLI_SHA256_ARM64=8454d872386a1633e9f1792d593b13b3f1dd7f8101bc09cbb6f67069edcb5dfa
ARG TARGETARCH
RUN set -eu; \
    arch="${TARGETARCH:-amd64}"; \
    case "$arch" in \
      amd64) sum="${CRE_CLI_SHA256_AMD64}" ;; \
      arm64) sum="${CRE_CLI_SHA256_ARM64}" ;; \
      *) echo "unsupported TARGETARCH ${arch}" >&2; exit 1 ;; \
    esac; \
    url="https://github.com/smartcontractkit/cre-cli/releases/download/${CRE_CLI_VERSION}/cre_linux_${arch}.tar.gz"; \
    curl -fsSL "$url" -o /tmp/cre.tar.gz; \
    echo "${sum}  /tmp/cre.tar.gz" | sha256sum -c -; \
    mkdir -p /opt/cre && tar -xzf /tmp/cre.tar.gz -C /opt/cre && rm /tmp/cre.tar.gz; \
    # The release tarball holds one file named after the version and platform, e.g. cre_v1.37.0_linux_arm64.
    bin="/opt/cre/cre_${CRE_CLI_VERSION}_linux_${arch}"; \
    if [ ! -f "$bin" ]; then echo "expected $bin in the release tarball; found:" >&2; ls -la /opt/cre >&2; exit 1; fi; \
    install -m 0755 "$bin" /usr/local/bin/cre; \
    rm -rf /opt/cre

WORKDIR /workspace
COPY run.sh /usr/local/bin/cre-simulate
RUN chmod +x /usr/local/bin/cre-simulate

ENTRYPOINT ["cre-simulate"]
