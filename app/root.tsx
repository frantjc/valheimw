import {
  Links,
  Meta,
  Outlet,
  Scripts,
  ScrollRestoration,
} from "@remix-run/react";
import type { LinksFunction } from "@remix-run/node";
import { FaGithub } from 'react-icons/fa';
import { TbApi } from 'react-icons/tb';

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
        <header className="fixed inset-x-0 top-0 z-10 border-b border-black/5 dark:border-white/10 bg-white dark:bg-gray-950">
          <nav className="flex h-14 items-center justify-between gap-8 px-4 sm:px-6">
            <a className="text-2xl font-bold" aria-label="Home" href="/">Sindri</a>
            <ul className="flex items-center gap-6 max-md:hidden">
              <li>
                <a href="https://github.com/frantjc/sindri" target="_blank" rel="noopener noreferrer" className="text-white hover:text-gray-400">
                  <FaGithub className="h-6 w-6" />
                </a>
              </li>
              <li>
                <a href="/api/v1" target="_blank" rel="noopener noreferrer" className="text-white hover:text-gray-400">
                  <TbApi className="h-6 w-6" />
                </a>
              </li>
            </ul>
          </nav>
        </header>
        <main className="isolate">
          {children}
        </main>
        <ScrollRestoration />
        <Scripts />
      </body>
    </html>
  );
}

export default function Main() {
  return <Outlet />;
}
