FROM debian:stable-slim
RUN apt-get update -y \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        lib32gcc-s1 \
    && rm -rf /var/lib/apt/lists/*
ENTRYPOINT ["boiler"]
COPY boiler /usr/local/bin
