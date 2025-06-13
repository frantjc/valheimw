import { createRequestHandler } from "@remix-run/express";
import compression from "compression";
import express from "express";
import proxy from "express-http-proxy";

const date_created = new Date("1970-01-01T00:00:00.000Z");

const dummySteamapps = {
  896660: {
    name: "Valheim",
    branch: "public",
    icon_url: "https://placehold.co/64x64",
    date_created,
    locked: false,
    base_image: "debian:stable-slim",
    apt_packages: [
      "ca-certificates",
    ],
    launch_type: "server",
    platform_type: "linux",
    execs: [
      "rm -r /home/steam/docker /home/steam/docker_start_server.sh /home/steam/start_server_xterm.sh /home/steam/start_server.sh",
      "ln -s /home/steam/linux64/steamclient.so /usr/lib/x86_64-linux-gnu/steamclient.so"
    ],
    entrypoint: [
      "/home/steam/valheim_server.x86_64",
    ],
    ports: [
      {
        port: 2456,
      }
    ]
  },
}

const dummySteamappsKeys = Object.keys(dummySteamapps);

function dummyGetSteamapps(req, res) {
  const cont = req.query.cont || 0;
  const limit = req.query.limit || 10;
  const end = cont + limit;

  if (dummySteamappsKeys.length > end) {
    res.setHeader("X-Continue-Token", end)
  }

  res.json(
    dummySteamappsKeys.slice(cont, limit).map((app_id) => ({
      app_id,
      ...dummySteamappsKeys[app_id],
    })),
  );
}

function dummyGetSteamapp(req, res) {
  const app_id = req.params.app_id;
  const branch = req.params.branch;
  const steamapp = dummySteamapps[app_id];

  if (steamapp) {
    res.json({
      app_id,
      ...steamapp,
      branch: branch || steamapp.branch || "public",
    });
    return;
  }

  res.status(404).json({
    error: `Steamapp with ID ${app_id} not found`,
  });
}

const viteDevServer =
  process.env.NODE_ENV === "production"
    ? undefined
    : await import("vite").then((vite) =>
        vite.createServer({
          server: { middlewareMode: true },
        })
      );

const remixHandler = createRequestHandler({
  build: viteDevServer
    ? () => viteDevServer.ssrLoadModule("virtual:remix/server-build")
    : await import("./build/server/index.js"),
});

const app = express();

app.use(compression());

// http://expressjs.com/en/advanced/best-practice-security.html#at-a-minimum-disable-x-powered-by-header
app.disable("x-powered-by");

try {
  const stokerUrl = new URL(process.env.STOKER_URL);
  const scheme = stokerUrl.protocol.slice(0, -1);
  switch (scheme) {
  case "http":
  case "https":
    app.all("/api/*", proxy(stokerUrl.toString(), {
      proxyReqOptDecorator: (proxyReqOpts) => {
        proxyReqOpts.headers["Host"] = stokerUrl.hostname;
        return proxyReqOpts;
      },
    }));
    break;
  case "dummy":
    app.get("/api/v1/steamapps", dummyGetSteamapps);
    app.get("/api/v1/steamapps/:app_id", dummyGetSteamapp);
    app.get("/api/v1/steamapps/:app_id/:branch", dummyGetSteamapp);
    break;
  default:
    throw new Error(`unsupported scheme: ${scheme}`);
  }
} catch (_) { /**/ }

// Handle asset requests.
if (viteDevServer) {
  app.use(viteDevServer.middlewares);
} else {
  // Vite fingerprints its assets so we can cache forever.
  app.use(
    "/assets",
    express.static("build/client/assets", { immutable: true, maxAge: "1y" })
  );
}

// Everything else (like favicon.ico) is cached for an hour. You may want to be
// more aggressive with this caching.
app.use(express.static("build/client", { maxAge: "1h" }));

// Handle SSR requests.
app.all("*", remixHandler);

const port = process.env.PORT || 3000;
app.listen(port);
