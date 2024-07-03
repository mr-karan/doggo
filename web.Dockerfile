# Dockerfile
FROM ubuntu:24.04
# Install ca-certificates
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
RUN ls -alht
COPY doggo-web.bin .
COPY config-api-sample.toml config.toml
COPY docs/dist /app/dist/
CMD ["./doggo-web.bin"]
