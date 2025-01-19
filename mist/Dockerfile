FROM scratch
COPY --from=ghcr.io/frantjc/mist /mist /
RUN /mist steamapp://896660 / \
    && /mist thunderstore://denikson/BepInExPack_Valheim / \
    && /mist thunderstore://RandyKnapp/EquipmentAndQuickSlots /BepInEx/plugins \
    && /mist --clean
