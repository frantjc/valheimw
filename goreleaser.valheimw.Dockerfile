FROM debian:stable-slim AS valheimw
RUN apt-get update -y \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        lib32gcc-s1 \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system valheimw \
    && useradd --system --gid valheimw --shell /bin/bash --create-home valheimw
USER valheimw
ENTRYPOINT ["/usr/local/bin/valheimw"]
COPY valheimw /usr/local/bin
