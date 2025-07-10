import {
  isRouteErrorResponse,
  Links,
  Meta,
  Outlet,
  Scripts,
  ScrollRestoration,
  useRouteError,
} from "@remix-run/react";
import type { LinksFunction } from "@remix-run/node";
import { FaGithub } from "react-icons/fa";
import { TbApi } from "react-icons/tb";

import styles from "./tailwind.css?url";

export const links: LinksFunction = () => [
  {
    rel: "stylesheet",
    href: styles,
  },
];

export function Layout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <Meta />
        <Links />
      </head>
      <body>
        <div className="isolate">
          <div className="max-w-screen overflow-x-hidden">
            <header className="border-b border-gray-500">
              <nav className="flex h-14 items-center justify-between px-4">
                <a
                  className="text-2xl font-bold hover:text-gray-500"
                  aria-label="Home"
                  href="/"
                >
                  Sindri
                </a>
                <ul className="flex items-center gap-6">
                  <li>
                    <a
                      href="https://github.com/frantjc/sindri"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="hover:text-gray-500"
                    >
                      <FaGithub className="h-6 w-6" />
                    </a>
                  </li>
                  <li>
                    <a
                      href="/api/v1"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="hover:text-gray-500"
                    >
                      <TbApi className="h-6 w-6" />
                    </a>
                  </li>
                </ul>
              </nav>
            </header>
            <main className="min-h-dvh container mx-auto px-2 tracking-wider">
              {children}
            </main>
          </div>
        </div>
        <ScrollRestoration />
        <Scripts />
      </body>
    </html>
  );
}

export default function Index() {
  return <Outlet />;
}

export function ErrorBoundary() {
  const err = useRouteError();

  return (
    <Layout>
      {isRouteErrorResponse(err)
        ? err.statusText
        : err instanceof Error
          ? err.message
          : "Unknown error"}
    </Layout>
  );
}
