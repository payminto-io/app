# cre-simulator: runs the real workflow code through `cre workflow simulate` against the local gateway.
# Started only by `docker compose --profile cre up`; never deploys, logs in or spends funds (docs/cre/SPEC.md section 10).
FROM oven/bun:1-debian

RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates bash && rm -rf /var/lib/apt/lists/*

# Official installer (docs.chain.link/cre/getting-started/cli-installation); pin with CRE_CLI_VERSION when it supports one.
RUN curl -sSfL https://cre.chain.link/install.sh | bash
ENV PATH="/root/.cre/bin:${PATH}"

WORKDIR /workspace
COPY run.sh /usr/local/bin/cre-simulate
RUN chmod +x /usr/local/bin/cre-simulate

ENTRYPOINT ["cre-simulate"]
