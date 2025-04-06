import type { MetaFunction } from "@remix-run/node";
import React from "react";
import { getSteamapps, SteamappSummary } from "~/client";
import { BsClipboard, BsClipboardCheck } from 'react-icons/bs';

export const meta: MetaFunction = () => {
  return [
    { title: "Sindri" },
  ];
};

export default function Index() {
  const [steamapps, setSteamapps] = React.useState<Array<SteamappSummary>>();
  const [index, setIndex] = React.useState(0);

  React.useEffect(() => {
    getSteamapps()
      .then(res => setSteamapps(res.steamapps));
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
  const steamapp = steamapps && steamapps.length > 0 && steamapps[index];
  const command = steamapp && `docker run sindri.frantjc.cc/${steamapp.app_id.toString()}:${tag}`

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
    <div className="max-w-screen overflow-x-hidden">
      <div className="grid min-h-dvh grid-cols-1 grid-rows-[1fr_1px_auto_1px_auto] justify-center pt-14.25 [--gutter-width:2.5rem] md:-mx-4 md:grid-cols-[var(--gutter-width)_minmax(0,var(--breakpoint-2xl))_var(--gutter-width)] lg:mx-0">
        <div className="grid gap-24 pb-24 sm:gap-40 md:pb-40">
          {steamapp && (
            <div className="bg-gray-800 text-white p-4 rounded-lg flex items-center">
              <img
                src={steamapp.icon_url}
                alt={`${steamapp.name} logo`}
                className="w-16 h-16 object-contain mr-4"
              />
              <div className="flex flex-col">
                <p className="text-lg mb-2">Run the {steamapp.name}: (:{tag} uses the {branch} branch of the Steamapp)</p>
                    <pre
                      className="bg-gray-900 p-2 pl-4 rounded mt-2 flex items-center justify-between w-full"
                    >
                      <code className="font-mono">{command}</code>
                      <button
                        onClick={handleCopy}
                        className="bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-2 rounded flex items-center"
                      >
                        {copied ? <BsClipboardCheck className="h-5 w-4" /> : <BsClipboard className="h-4 w-6" />}
                      </button>
                    </pre>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
