# Dockerfile
ARG ARCH
FROM ${ARCH}/alpine
WORKDIR /app
COPY doggo-api.bin .
COPY config-api-sample.toml config.toml
CMD ["./doggo-api.bin"]
