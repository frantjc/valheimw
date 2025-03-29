#!/bin/sh

wget "http://stoker:5050/api/v1/steamapps/896660" \
    --header="Content-Type: application/json" \
    --post-data='{
      "base_image": "",
      "apt_packages": ["ca-certificates"],
      "launch_type": "server",
      "platform_type": "",
      "execs": [
        "rm -r /home/steam/docker /home/steam/docker_start_server.sh /home/steam/start_server_xterm.sh /home/steam/start_server.sh",
        "ln -s /home/steam/linux64/steamclient.so /usr/lib/x86_64-linux-gnu/steamclient.so"
      ],
      "entrypoint": ["/home/steam/valheim_server.x86_64"],
      "cmd": []
    }' -O -