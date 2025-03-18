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
        <header className="w-full bg-gray-800 p-4">
          <nav className="flex justify-between items-center">
            <a href="/" className="text-2xl font-bold">Sindri</a>
            <ul className="flex space-x-4">
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
        {children}
        <ScrollRestoration />
        <Scripts />
      </body>
    </html>
  );
}

export default function Main() {
  return <Outlet />;
}
