# Dockerfile
FROM alpine:3.24.2
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/doggo /usr/bin/doggo
ENTRYPOINT ["/usr/bin/doggo"]
