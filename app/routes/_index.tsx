import type { MetaFunction } from "@remix-run/node";
import React from "react";
import { BsClipboard, BsClipboardCheck } from 'react-icons/bs';
import { useSteamapps } from "~/hooks";

export const meta: MetaFunction = () => {
  return [
    { title: "Sindri" },
  ];
};

export default function Index() {
  const [steamapps, err, _, loading] = useSteamapps();
  const [index, setIndex] = React.useState(0);

  React.useEffect(() => {
    if (steamapps.length && steamapps.length > 1) {
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
    <div className="grid grid-cols-1 gap-4">
      <div className="h-24 flex items-end">
        <p className="text-3xl">Run the...</p>
      </div>
      {steamapp ? (
        <>
          <div className="flex">
            <p className="text-xl">{steamapp.name}</p>
          </div>
          <pre
            className="bg-black flex p-2 px-4 rounded items-center justify-between w-full border"
          >
            <code className="font-mono">
              <span className="pr-2">$</span>
              {command}
            </code>
            <button
              onClick={handleCopy}
              className="bg-blue-500 hover:bg-blue-700 text-white font-bold p-2 rounded flex items-center"
            >
              {copied ? <BsClipboardCheck className="h-5 w-4" /> : <BsClipboard className="h-4 w-6" />}
            </button>
          </pre>
        </>
      ) : (<></>)}
    </div>
  );
}
