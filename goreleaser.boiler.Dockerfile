FROM debian:stable-slim
RUN apt-get update -y \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        lib32gcc-s1 \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system boiler \
    && useradd --system --gid boiler --shell /bin/bash --create-home boiler
USER boiler
ENTRYPOINT ["/usr/local/bin/boiler"]
COPY boiler /usr/local/bin
