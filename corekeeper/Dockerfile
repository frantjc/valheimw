# Default tag is "latest" which gets mapped to the Steam app branch "public"
FROM localhost:5000/1963720
USER root
RUN apt-get update -y \
    && apt-get install -y --no-install-recommends \
        # The Core Keeper server's launch script doesn't check for these,
        # but it does depend on it and SEGFAUTs without it.
        ca-certificates \
        curl \
        locales \
        # Core Keeper's launch script does check for these,
        # but it's better to cache it here.
        libxi6 \
        xvfb \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*
    # The Core Keeper server SEGFAUTs without this.
    && ln -s /home/boil/steamapp/linux64/steamclient.so /usr/lib/x86_64-linux-gnu/steamclient.so \
    # Something keeps setting this back to root.
    # Force it back for now, otherwise the Core Keeper
    # server cannot create ~/.config.
    && chown -R boil:boil /home/boil
USER boil
# launch.sh just execs _launch.sh with gnome,
# so avoid the extra dependency on gnome.
# Also, log to stdout like a container.
ENTRYPOINT ["/home/boil/steamapp/_launch.sh", "-logfile", "/dev/stdout"]
