ARG MIST_TAG=latest
FROM ghcr.io/frantjc/mist:$MIST_TAG AS mist

FROM debian:stable-slim
COPY --from=mist /mist /usr/local/bin
RUN apt-get update -y \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        lib32gcc-s1 \
    && mist steamapp://896660 /root/valheim \
    && mist thunderstore://denikson/BepInExPack_Valheim /root/valheim \
    && mist thunderstore://RandyKnapp/EquipmentAndQuickSlots /root/valheim/BepInEx/plugins \
    && mist --clean
