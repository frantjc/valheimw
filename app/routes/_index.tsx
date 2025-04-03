import type { MetaFunction } from "@remix-run/node";
import React from "react";
import { getSteamapps, SteamappSummary } from "~/client";
import { BsClipboard, BsClipboardCheck } from 'react-icons/bs';

export const meta: MetaFunction = () => {
  return [
    { title: "Sindri" },
  ];
};

export default function Main() {
  const [steamapps, setSteamapps] = React.useState<Array<SteamappSummary>>();
  const [index, setIndex] = React.useState(0);

  React.useEffect(() => {
    getSteamapps()
      .then(res => setSteamapps(res.steamapps))
      .catch(() => setSteamapps([
        {
          app_id: 896660,
          name: "Valheim Dedicated Server",
          branch: "public-test",
          icon_url: "https://cdn.cloudflare.steamstatic.com/steamcommunity/public/images/apps/896660/1aab0586723c8578c7990ced7d443568649d0df2.jpg",
          date_created: new Date(),
          locked: false
        },
        {
          app_id: 896660,
          name: "Valheim Dedicated Server",
          icon_url: "https://cdn.cloudflare.steamstatic.com/steamcommunity/public/images/apps/896660/1aab0586723c8578c7990ced7d443568649d0df2.jpg",
          date_created: new Date(),
          locked: false
        }
      ]));
  }, [setSteamapps])

  React.useEffect(() => {
    if (steamapps?.length && steamapps.length > 1) {
      const timeout = setInterval(
        () => setIndex(i => (i+1)%steamapps.length),
        2000,
      );

      return () => clearTimeout(timeout);
    }
  }, [steamapps, setIndex]);

  const tag = steamapps && steamapps.length > 0 && steamapps[index].branch || "latest";
  const branch = tag === "latest" ? "public" : tag;
  const command = steamapps && steamapps.length > 0 && `docker run sindri.frantjc.cc/${steamapps[index].app_id.toString()}:${tag}`

  const [copied, setCopied] = React.useState(false);

  const handleCopy = () => {
    if (command) {
      navigator.clipboard.writeText(command);
      setCopied(true);
      const timeout = setTimeout(() => setCopied(false), 1331);
      return () => clearTimeout(timeout);
    }
  };

  return (
    <main className="isolate">
      <div className="max-w-screen overflow-x-hidden">
        <div className="grid min-h-dvh grid-cols-1 grid-rows-[1fr_1px_auto_1px_auto] justify-center pt-14.25 [--gutter-width:2.5rem] md:-mx-4 md:grid-cols-[var(--gutter-width)_minmax(0,var(--breakpoint-2xl))_var(--gutter-width)] lg:mx-0">
          <div className="grid gap-24 pb-24 text-white sm:gap-40 md:pb-40">
            {steamapps?.length && (
              <div className="bg-gray-800 text-white p-4 rounded-lg flex items-center">
                {steamapps[index].icon_url && (
                  <img
                    src={steamapps[index].icon_url}
                    alt={`${steamapps[index].name} logo`}
                    className="w-16 h-16 object-contain mr-4"
                  />
                )}
                <div className="flex flex-col">
                  <p className="text-lg mb-2">Run the {steamapps[index].name}: (:{tag} uses the {branch} branch of the Steamapp)</p>
                      <pre
                        className="bg-gray-900 p-2 rounded mt-2 flex items-center justify-between w-full"
                      >
                        <code className="font-mono">{command}</code>
                        <button
                          onClick={handleCopy}
                          className="ml-4 bg-blue-500 hover:bg-blue-700 text-white font-bold py-1 px-2 rounded flex items-center"
                        >
                          {copied ? <BsClipboardCheck className="h-5 w-4" /> : <BsClipboard className="h-5 w-4" />}
                        </button>
                      </pre>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </main>
  );
}
