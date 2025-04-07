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
  const [steamapps, err, , loading] = useSteamapps();
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

  React.useEffect(() => {
    if (err) {
      alert(`Error: ${err}.`);
    }
  }, [err]);

  const steamapp = steamapps && steamapps.length > 0 && steamapps[index];
  const tag = steamapp && steamapp.branch || "latest";
  const branch = tag === "latest" ? "public" : tag;
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
      {loading ? (
        <></>
      ) : (
        <>
          <p className="text-3xl pt-8">Run the...</p>
          {steamapp ? (
            <>
              <p className="text-xl">
                  <a className="font-bold hover:underline" href={`https://steamdb.info/app/${steamapp.app_id}/`} target="_blank" rel="noopener noreferrer">
                    {steamapp.name}
                  </a>
                  {tag !== "latest" && (
                    <span>
                      's {branch} branch
                    </span>
                  )}
              </p>
              <pre
                className="bg-black flex p-2 px-4 rounded items-center justify-between w-full border border-gray-500"
              >
                <code className="font-mono text-white">
                  <span className="pr-2 text-gray-500">$</span>
                  {command}
                </code>
                <button
                  onClick={handleCopy}
                  className="bg-blue-400 hover:bg-blue-600 text-white font-bold p-2 rounded flex items-center"
                >
                  {copied ? <BsClipboardCheck className="h-4 w-8" /> : <BsClipboard className="h-4 w-8" />}
                </button>
              </pre>
            </>
          ) : (<></>)}
        </>
      )}
      <p className="py-4">
        Sindri is a read-only container registry for images with Steamapps installed on them.
      </p>
      <p className="pb-4">
        Images are based on <code className="font-mono">debian:stable-slim</code> and are nonroot for security purposes.
      </p>
      <p className="pb-4">
        Images are built on-demand, so the pulled Steamapp is always up-to-date. To update, just pull the image again.
      </p>
      <p className="pb-4">
        Steamapps commonly do not work out of the box, missing dependencies, specifying an invalid entrypoint or just generally not being container-friendly.
        Sindri attemps to fix this by crowd-sourcing configurations to apply to the images before returning them. To contribute such a configuration,
        check out Sindri's <a className="font-bold hover:underline" href="https://steamdb.info/" target="_blank" rel="noopener noreferrer">API</a>.
      </p>
      <p className="pb-4">
        Image references are of the form <code className="font-mono">sindri.frantj.cc/{"<steamapp-id>"}</code>.
        If you do not know your Steamapp's ID, find it on <a className="font-bold hover:underline" href="https://steamdb.info/" target="_blank" rel="noopener noreferrer">SteamDB</a>.
        Supported Steamapps can be found below.
      </p>
    </div>
  );
}
