# syntax=docker/dockerfile:1

FROM steamcmd/steamcmd AS steamcmd
RUN steamcmd \
	+force_install_dir /mnt \
	+login anonymous \
	+@sSteamCmdForcePlatformType windows \
	+app_update 2857200 \
	+quit

FROM debian:stable-slim AS wine
ADD https://dl.winehq.org/wine-builds/winehq.key /tmp/
ADD https://dl.winehq.org/wine-builds/debian/dists/trixie/winehq-trixie.sources /mnt/sources.list.d/
RUN apt-get update -y \
	&& apt-get install -y --no-install-recommends \
		gnupg \
	&& mkdir -p /mnt/keyrings \
	&& cat /tmp/winehq.key | gpg --dearmor -o /mnt/keyrings/winehq-archive.key -

FROM debian:stable-slim
RUN apt-get update -y \
	&& apt-get install -y --no-install-recommends \
		ca-certificates \
	&& dpkg --add-architecture i386
COPY --from=wine /mnt /etc/apt
RUN groupadd --system steam \
	&& useradd --system --gid steam --shell /bin/bash --create-home steam \
	&& apt-get update -y \
	&& apt-get install -y --no-install-recommends \
		winehq-stable \
	&& rm -rf /var/lib/apt/lists/* \
	&& apt-get clean
USER steam
COPY --from=steamcmd --chown=steam:steam /mnt /home/steam
ENTRYPOINT ["wine", "/home/steam/AbioticFactor/Binaries/Win64/AbioticFactorServer-Win64-Shipping.exe", "-useperfthreads", "-NoAsyncLoadingThread"]
