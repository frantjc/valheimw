import type { MetaFunction } from "@remix-run/node";
import React from "react";
import { BsClipboard, BsClipboardCheck } from "react-icons/bs";
import { getSteamapp, Steamapp, SteamappSummary } from "~/client";
import { CodeModal } from "~/components/code_modal";
import { useSteamapps } from "~/hooks";

export const meta: MetaFunction = () => {
  const title = "Sindri";
  const description = "Read-only container registry for Steamapp images.";

  let url;
  try {
    if (typeof process !== "object") {
      url = new URL(window.location.href);
    } else {
      const port = process.env.PORT || 3000;
      const base = process.env.URL || `http://localhost:${port}/`;
      url = location && new URL(location.pathname, base);
    }
  } catch (_) { /**/ }

  return [
    { charSet: "utf-8" },
    { name: "viewport", content: "width=device-width,initial-scale=1" },
    { property: "og:site_name", content: title },
    { title },
    { property: "og:title", content: title },
    { property: "twitter:title", content: title },
    { name: "description", content: description },
    { property: "og:description", content: description },
    { property: "twitter:description", content: description },
    { property: "og:type", content: "website" },
    { property: "twitter:card", content: "summary" },
    ...((url && [
      { property: "og:url", content: url.toString() },
      { property: "twitter:domain", content: url.hostname },
      { property: "twitter:url", content: url.toString() },
    ]) ||
      []),
  ];
};

const defaultTag = "latest";
const defaultBranch = "public";

export default function Index() {
  const [steamapps, err, hasMore, more, loading] = useSteamapps();
  const [index, setIndex] = React.useState(0);

  const [selectedSteamapp, setSelectedSteamapp] = React.useState<Steamapp | null>(null);
  const [modalOpen, setModalOpen] = React.useState(false);

  async function handleModal(summary: SteamappSummary) {
    setModalOpen(true);

    getSteamapp(summary.app_id, summary.branch).then(setSelectedSteamapp).catch((err) => {
      alert(`Error: ${err}.`);
    });
  }

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
  const tag = steamapp && steamapp.branch || defaultBranch;
  const branch = tag === defaultTag ? defaultBranch : tag;
  const command = steamapp && `docker run sindri.frantj.cc/${steamapp.app_id.toString()}:${tag}`

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
    <div className="grid grid-cols-1 gap-4 pb-8">
      {loading ? (
        <div className="flex h-24 pt-8 justify-center items-center">
          <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-blue-500" />
        </div>
      ) : !!steamapp && (
            <>
            <p className="text-3xl pt-8">Run the...</p>
              <p className="text-xl">
                  <a className="font-bold hover:underline" href={`https://steamdb.info/app/${steamapp.app_id}/`} target="_blank" rel="noopener noreferrer">
                    {steamapp.name}
                  </a>
                  {tag !== defaultTag && (
                    <span>
                      &#39;s {branch} branch
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
      )}
      <p className="py-4">
        Sindri is a read-only container registry for images with Steamapps installed on them.
      </p>
      <p className="pb-4">
        Images are based on <code className="font-mono bg-black rounded text-white p-1">debian:stable-slim</code> and are nonroot for security purposes.
      </p>
      <p className="pb-4">
        Images are built on-demand, so the pulled Steamapp is always up-to-date. To update, just pull the image again.
      </p>
      <p className="pb-4">
        Steamapps commonly do not work out of the box, missing dependencies, specifying an invalid entrypoint or just generally not being container-friendly.
        Sindri attemps to fix this by crowd-sourcing configurations to apply to the images before returning them. To contribute such a configuration,
        check out Sindri&#39;s <a className="font-bold hover:underline" href="/api/v1" target="_blank" rel="noopener noreferrer">API</a>.
      </p>
      <p className="pb-4">
        Image references are of the form <code className="font-mono bg-black rounded text-white p-1">sindri.frantj.cc/{"<steamapp-id>:<steamapp-branch>"}</code>.
        If you do not know your Steamapp&#39;s ID, find it on <a className="font-bold hover:underline" href="https://steamdb.info/" target="_blank" rel="noopener noreferrer">SteamDB</a>.
        There is a special case for the default tag, <code className="font-mono bg-black rounded text-white p-1">:{defaultTag}</code>, which gets mapped to the default Steamapp branch, {defaultBranch}.
        Supported Steamapps can be found below.
      </p>
      {!!steamapps.length && (
        <>
          <table>
            <thead>
              <tr>
                <th className="border-gray-500" />
                <th className="border-gray-500 font-bold">Steamapp</th>
                <th className="border-gray-500 font-bold">Image</th>
                <th className="border-gray-500 font-bold">Definition</th>
              </tr>
            </thead>
            <tbody>
              {steamapps.map((steamapp, key) => {
                return (
                  <tr key={key} className="border-t border-gray-500">
                    <td className="p-2 border-gray-500 flex justify-center items-center">
                      <img
                        src={steamapp.icon_url}
                        alt={`${steamapp.name} icon`}
                        className="size-8 rounded object-contain"
                      />
                    </td>
                    <td className="border-gray-500">
                      <a className="font-bold hover:underline" href={`https://steamdb.info/app/${steamapp.app_id}/`} target="_blank" rel="noopener noreferrer">{steamapp.name}</a>{steamapp.branch && steamapp.branch !== defaultBranch ? `'s ${steamapp.branch} branch` : ""}
                    </td>
                    <td className="border-gray-500">
                      <code className="font-mono">sindri.frantj.cc/{steamapp.app_id}{steamapp.branch ? `:${steamapp.branch}` : `:${defaultTag}`}</code>
                    </td>
                    <td className="border-gray-500">
                      <button
                      onClick={() => handleModal(steamapp)}
                      className="bg-blue-400 hover:bg-blue-600 text-white font-bold p-2 rounded flex items-center"
                    >
                      View
                    </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
          <CodeModal
            open={modalOpen}
            onClose={() => {
              setModalOpen(false)
              setSelectedSteamapp(null);
            }}
            steamapp={selectedSteamapp}
            lines={16}
          />
          {hasMore && (
            <div className="flex justify-center items-center py-4">
              {loading ? (
                <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-blue-500" />
              ) : (
                <button
                  onClick={more}
                  className="bg-blue-400 hover:bg-blue-600 text-white font-bold py-2 px-4 rounded"
                >
                  Load More
                </button>
              )}
            </div>
          )}
        </>
      )}
    </div>
  );
}
