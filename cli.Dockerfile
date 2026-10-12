# Dockerfile
ARG ARCH
FROM ${ARCH}/alpine:3.24.2
COPY doggo /usr/bin/doggo
ENTRYPOINT ["/usr/bin/doggo"]
