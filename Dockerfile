FROM scratch
ARG TARGETARCH
WORKDIR /app
COPY keyvaluedatabaseweb-${TARGETARCH} /usr/bin/
COPY keysindex.html /app
COPY namespacesindex.html /app
COPY certificates /
ENTRYPOINT [\"keyvaluedatabaseweb\"]