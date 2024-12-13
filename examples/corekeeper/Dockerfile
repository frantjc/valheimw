# Default tag is "latest" which gets mapped to the Steam app branch "public"
# FROM boil.frantj.cc/1963720
FROM localhost:8080/1963720
USER root
RUN apt-get update -y \
    && apt-get install -y --no-install-recommends \
        # Corekeeper's launch script doesn't check for these,
        # but it does depend on it and SEGFAUTs without it.
        ca-certificates \
        curl \
        locales \
        # Corekeeper's launch script does check for these,
        # but it's better to cache it here.
        libxi6 \
        xvfb \
    && rm -rf /var/lib/apt/lists/*
# Corekeeper SEGFAUTs without this.
RUN ln -s /boil/steamapp/linux64/steamclient.so /usr/lib/x86_64-linux-gnu/steamclient.so
# Something keeps setting this back to root.
# Force it back for now, otherwise Corekeeper
# can't create ~/.config.
RUN chown -R boil:boil
USER boil
# launch.sh just execs _launch.sh with gnome,
# so avoid the extra dependency on gnome.
ENTRYPOINT ["/boil/steamapp/_launch.sh"]
