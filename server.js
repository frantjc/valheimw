import { createRequestHandler } from "@remix-run/express";
import compression from "compression";
import express from "express";
import proxy from "express-http-proxy";

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
  app.all("/api/*", proxy(stokerUrl.toString(), {
    proxyReqOptDecorator: (proxyReqOpts) => {
      proxyReqOpts.headers["Host"] = stokerUrl.hostname;
      return proxyReqOpts;
    },
  }))
} catch (_) { /**/ }

// handle asset requests
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

// handle SSR requests
app.all("*", remixHandler);

const port = process.env.PORT || 3000;
app.listen(port);
