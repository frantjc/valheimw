FROM debian:stable-slim AS valheimw
RUN apt-get update -y \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        lib32gcc-s1 \
        libatomic1 \
        libpulse-dev \
        libpulse0 \
    && rm -rf /var/lib/apt/lists/*
RUN groupadd -r valheimw
RUN useradd -r -g valheimw -m -s /bin/bash valheimw
USER valheimw
ENTRYPOINT ["/usr/local/bin/valheimw"]
COPY valheimw /usr/local/bin
