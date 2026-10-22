# Dockerfile
FROM ubuntu:26.04
ARG TARGETPLATFORM
# Install ca-certificates
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY ${TARGETPLATFORM}/doggo-web.bin .
COPY config-api-sample.toml config.toml
COPY docs/dist /app/dist/
CMD ["./doggo-web.bin"]
