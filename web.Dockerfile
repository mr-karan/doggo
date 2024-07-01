# Dockerfile
FROM ubuntu:24.04
WORKDIR /app
RUN ls -alht
COPY doggo-web.bin .
COPY config-api-sample.toml config.toml
COPY docs/dist /app/dist/
CMD ["./doggo-web.bin"]
