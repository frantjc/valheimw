# Mist

Mist is a CLI tool for use in `Dockerfile`s to install Steamapps, Steam Workshop items, and [thunderstore.io](https://thunderstore.io/) mods. It uses [GoCloud's URL concept](https://gocloud.dev/concepts/urls/) to expose installing the content from the different sources using a similar command.

The following `Dockerfile` builds a container image for a modded Valheim server and provides an excellent example for how to use Mist:

```Dockerfile
FROM debian:stable-slim
COPY --from=ghcr.io/frantjc/mist /mist /usr/local/bin
RUN apt-get update -y \
    && apt-get install -y --no-install-recommends \
        # So that mist can make a trusted TLS connection
        # to download `steamcmd`.
        ca-certificates \
        # `mist` installs `steamcmd`, but we still have to
        # satisfy `steamcmd`'s dependencies.
        lib32gcc-s1 \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* \
    # Install the Valheim server to /root/valheim.
    # "896660" refers to the Steamapp ID of the Valheim server.
    && mist steamapp://896660 /root/valheim \
    # Install BepInEx to /root/valheim.
    && mist thunderstore://denikson/BepInExPack_Valheim /root/valheim \
    # Install EquipmentAndQuickSlots to /root/valheim/BepInEx/plugins.
    && mist thunderstore://RandyKnapp/EquipmentAndQuickSlots /root/valheim/BepInEx/plugins \
    # Cleanup.
    && mist --clean \
    && rm /usr/local/bin/mist \
    && apt-get remove -y \
        ca-certificates \
        lib32gcc-s1
WORKDIR /root/valheim/
ENTRYPOINT ["/root/valheim/start_server.sh"]
```
