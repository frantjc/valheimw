FROM steamcmd/steamcmd:debian-12
WORKDIR /
ENTRYPOINT ["/sindri"]
COPY sindri /
