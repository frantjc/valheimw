#!/usr/bin/env sh

wget $1/api/v1/steamapps/896660 \
    --header="Content-Type: application/json" \
    --post-data='{
      "apt_packages": [
        "ca-certificates"
      ],
      "launch_type": "server",
      "execs": [
        "rm -r /home/steam/docker /home/steam/docker_start_server.sh /home/steam/start_server_xterm.sh /home/steam/start_server.sh",
        "ln -s /home/steam/linux64/steamclient.so /usr/lib/x86_64-linux-gnu/steamclient.so"
      ],
      "entrypoint": ["/home/steam/valheim_server.x86_64"]
    }' -O-

wget $1/api/v1/steamapps/896660/public-test?betapassword=yesimadebackups \
    --header="Content-Type: application/json" \
    --post-data='{
      "apt_packages": [
        "ca-certificates"
      ],
      "launch_type": "server",
      "execs": [
        "rm -r /home/steam/docker /home/steam/docker_start_server.sh /home/steam/start_server_xterm.sh /home/steam/start_server.sh",
        "ln -s /home/steam/linux64/steamclient.so /usr/lib/x86_64-linux-gnu/steamclient.so"
      ],
      "entrypoint": [
        "/home/steam/valheim_server.x86_64"
      ]
    }' -O-

wget $1/api/v1/steamapps/1963720 \
    --header="Content-Type: application/json" \
    --post-data='{
      "apt_packages": [
        "ca-certificates",
        "curl",
        "locales",
        "libxi6",
        "xvfb"
      ],
      "launch_type": "server",
      "execs": [
        "ln -s /home/steam/linux64/steamclient.so /usr/lib/x86_64-linux-gnu/steamclient.so"
      ],
      "entrypoint": [
        "/home/steam/_launch.sh",
        "-logfile", "/dev/stdout"
      ]
    }' -O-

wget $1/api/v1/steamapps/2394010 \
    --header="Content-Type: application/json" \
    --post-data='{
      "apt_packages": [
        "ca-certificates",
        "xdg-user-dirs"
      ],
      "launch_type": "default"
    }' -O-
