# syntax=docker/dockerfile:1

FROM steamcmd/steamcmd AS steamcmd
RUN groupadd --system steam \
	&& useradd --system --gid steam --shell /bin/bash --create-home steam \
	&& steamcmd \
		+force_install_dir /mnt \
		+login anonymous \
		+@sSteamCmdForcePlatformType linux \
		+app_update 896660 \
		+quit

FROM debian:stable-slim
RUN groupadd --system steam \
	&& useradd --system --gid steam --shell /bin/bash --create-home steam \
	&& apt-get update -y \
	&& apt-get install -y --no-install-recommends \
		ca-certificates \
	&& rm -rf /var/lib/apt/lists/* \
	&& apt-get clean \
	&& rm -r /home/steam/docker /home/steam/docker_start_server.sh /home/steam/start_server_xterm.sh /home/steam/start_server.sh \
	&& ln -s /home/steam/linux64/steamclient.so /usr/lib/x86_64-linux-gnu/steamclient.so
USER steam
COPY --from=steamcmd --chown=steam:steam /mnt /home/steam
ENTRYPOINT ["/home/steam/valheim_server.x86_64"]
