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
RUN useradd -r -g valheim -m -d /home/valheim -s /bin/bash valheimw
USER valheimw
WORKDIR /home/valheimw
ENTRYPOINT ["valheimw"]
COPY --from=build /valheimw /usr/local/bin

FROM scratch AS boil
COPY --from=build /$tool /
ENTRYPOINT ["/boil"]

FROM scratch AS mist
COPY --from=build /$tool /
ENTRYPOINT ["/mist"]

FROM scratch AS sindri
COPY sindri /sindri
ENTRYPOINT ["/sindri"]
