FROM debian:stable-slim
RUN apt-get update -y \
    && apt-get install -y --no-install-recommends \
        # So that mist can make a trusted TLS connection
        # to download `steamcmd`.
        ca-certificates \
        # `mist` installs `steamcmd`, but we still have to
        # satisfy `steamcmd`'s dependencies.
        lib32gcc-s1 \
    && rm -rf /var/lib/apt/lists/*
COPY --from=ghcr.io/frantjc/mist /mist /usr/local/bin
# Install the Valheim server to /root/valheim.
RUN mist steamapp://896660 /root/valheim
# Install BepInEx to /root/valheim.
RUN mist thunderstore://denikson/BepInExPack_Valheim /root/valheim
# Install EquipmentAndQuickSlots to /root/valheim/BepInEx/plugins.
RUN mist thunderstore://RandyKnapp/EquipmentAndQuickSlots /root/valheim/BepInEx/plugins
# Cleanup after `mist`.
# A bug in `mist` requires 2 arguments even when cleaning.
RUN mist --clean x x
RUN rm /usr/local/bin/mist
RUN apt-get remove -y \
    ca-certificates \
    lib32gcc-s1
WORKDIR /root/valheim/
ENTRYPOINT ["/root/valheim/start_server.sh"]
