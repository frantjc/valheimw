#!/bin/sh

curl -X PUT "http://stoker:5050/api/v1/steamapps/896660" \
     -H "Content-Type: application/json" \
     -d '{
          "datecreated": "2025-03-26T12:34:56Z",
          "baseimage": "",
          "aptpackages": ["ca-certificates"],
          "launchtype": "server",
          "platformtype": "",
          "execs": [
            "rm -r /home/steam/docker /home/steam/docker_start_server.sh /home/steam/start_server_xterm.sh /home/steam/start_server.sh",
            "ln -s /home/steam/linux64/steamclient.so /usr/lib/x86_64-linux-gnu/steamclient.so"
          ],
          "entrypoint": ["/home/steam/valheim_server.x86_64"],
          "cmd": []
        }'