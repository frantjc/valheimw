import type { LoaderFunctionArgs, MetaFunction } from "@remix-run/node";
import { useLoaderData } from "@remix-run/react";
import React from "react";
import { BsClipboard, BsClipboardCheck } from "react-icons/bs";
import { getSteamapp, getSteamapps, Steamapp, SteamappSummary } from "~/client";
import { AddModal } from "~/components/AddModal";
import { CodeModal } from "~/components/CodeModal";

export const meta: MetaFunction = () => {
  const title = "Sindri";
  const description = "Read-only container registry for Steamapp images.";

  return [
    { charSet: "utf-8" },
    { name: "viewport", content: "width=device-width,initial-scale=1" },
    { property: "og:site_name", content: title },
    { title },
    { property: "og:title", content: title },
    { name: "description", content: description },
    { property: "og:description", content: description },
    { property: "og:type", content: "website" },
  ];
};

export function loader(args: LoaderFunctionArgs) {
  const host =
    args.request.headers.get("Host") ||
    `localhost:${process.env.PORT || "3000"}`;

  return getSteamapps()
    .then(({ token, steamapps }) => {
      return {
        host,
        steamapps,
        token,
      };
    })
    .catch(() => {
      return { host, steamapps: [], token: "" };
    });
}

const defaultTag = "latest";
const defaultBranch = "public";

export default function Index() {
  const {
    host,
    steamapps: initialSteamapps,
    token: initialToken,
  } = useLoaderData<typeof loader>();

  const [steamapps, setSteamapps] =
    React.useState<Array<SteamappSummary | Steamapp>>(initialSteamapps);
  const [token, setToken] = React.useState(initialToken);
  const [err, setErr] = React.useState<Error>();

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const handleError = React.useCallback(
    (err: any) => {
      if (err instanceof Error) {
        setErr(err);
      } else if (err instanceof Response) {
        setErr(new Error(`${err.status}: ${err.statusText}`));
      } else {
        setErr(new Error(err));
      }
    },
    [setErr]
  );

  const more = React.useCallback(
    (token?: string) => {
      return getSteamapps({ token })
        .then((res) => {
          setSteamapps((s) => [
            ...s,
            ...res.steamapps.filter(
              (app) =>
                !s.some(
                  (existing) =>
                    existing.app_id === app.app_id &&
                    existing.branch === app.branch
                )
            ),
          ]);
          setToken(res.token);
        })
        .catch(handleError);
    },
    [setSteamapps, setToken, handleError]
  );

  React.useEffect(() => {
    if (steamapps.length === 0) {
      more();
    }
  }, [more, steamapps]);

  const [prefetchIndex, setPrefetchIndex] = React.useState(0);

  React.useEffect(() => {
    if (steamapps.length && steamapps.length > 1) {
      const timeout = setInterval(
        () => setPrefetchIndex((i) => (i + 1) % steamapps.length),
        2000
      );

      return () => clearTimeout(timeout);
    }
  }, [steamapps, setPrefetchIndex]);

  const getSteamappDetails = React.useCallback(
    (index: number) => {
      const steamapp = steamapps[index];

      if (steamapp && !(steamapp as Steamapp).base_image) {
        return getSteamapp(steamapp.app_id, steamapp.branch).then((s) => {
          setSteamapps((ss) => {
            const newSteamapps = [...ss];
            newSteamapps[index] = s;
            return newSteamapps;
          });

          return s;
        });
      }

      return Promise.resolve(steamapp as Steamapp);
    },
    [steamapps, setSteamapps]
  );

  const [dockerRunIndex, setDockerRunIndex] = React.useState(0);

  React.useEffect(() => {
    if (steamapps.length > prefetchIndex && prefetchIndex >= 0) {
      getSteamappDetails(prefetchIndex)
        .then(() => {
          setDockerRunIndex(prefetchIndex);
        })
        .catch(() => {
          /**/
        });
    }
  }, [prefetchIndex, getSteamappDetails, setDockerRunIndex, steamapps]);

  const [selectedSteamapp, setSelectedSteamapp] = React.useState<number>(-1);

  React.useEffect(() => {
    if (err) {
      alert(`Error: ${err}.`);
    }
  }, [err]);

  const steamapp =
    steamapps &&
    steamapps.length > 0 &&
    (steamapps[dockerRunIndex] as Steamapp).base_image &&
    (steamapps[dockerRunIndex] as Steamapp);
  const tag = (steamapp && steamapp.branch) || defaultBranch;
  const branch = tag === defaultTag ? defaultBranch : tag;
  const command =
    steamapp &&
    "docker run"
      .concat(
        steamapp.ports
          ? steamapp.ports
              .map((port) => ` -p ${port.port}:${port.port}`)
              .join("")
          : ""
      )
      .concat(` ${host}/${steamapp.app_id.toString()}:${tag}`);

  const [copied, setCopied] = React.useState(false);

  const handleCopy = () => {
    if (command) {
      navigator.clipboard.writeText(command);
      setCopied(true);
      const timeout = setTimeout(() => setCopied(false), 2000);
      return () => clearTimeout(timeout);
    }
  };

  const [addModalOpen, setAddModalOpen] = React.useState(false);

  return (
    <div className="grid grid-cols-1 gap-4 pb-8">
      {!!steamapp && (
        <>
          <p className="text-3xl pt-8">Run the...</p>
          <p className="text-xl">
            <a
              className="font-bold hover:underline"
              href={`https://steamdb.info/app/${steamapp.app_id}/`}
              target="_blank"
              rel="noopener noreferrer"
            >
              {steamapp.name}
            </a>
            {tag !== defaultTag && <span>&#39;s {branch} branch</span>}
          </p>
          <pre className="bg-black flex p-2 px-4 rounded items-center justify-between w-full border border-gray-500">
            <code className="font-mono text-white overflow-auto pr-4">
              <span className="pr-2 text-gray-500">$</span>
              {command}
            </code>
            <button
              onClick={handleCopy}
              className="bg-blue-400 hover:bg-blue-600 text-white p-2 rounded flex items-center"
            >
              {copied ? (
                <BsClipboardCheck className="h-4 w-8" />
              ) : (
                <BsClipboard className="h-4 w-8" />
              )}
            </button>
          </pre>
        </>
      )}
      <p className="py-4">
        Sindri is a read-only container registry for images with Steamapps
        installed on them.
      </p>
      <p className="pb-4">
        Images are based on{" "}
        <code className="font-mono bg-black rounded text-white p-1">
          debian:stable-slim
        </code>{" "}
        and are nonroot for security purposes.
      </p>
      <p className="pb-4">
        Images are built on-demand, so the pulled Steamapp is always up-to-date.
        To update, just pull the image again.
      </p>
      <p className="pb-4">
        Steamapps commonly do not work out of the box, missing dependencies,
        specifying an invalid entrypoint or just generally not being
        container-friendly. Sindri attemps to fix this by crowd-sourcing
        configurations to apply to the images before returning them. To
        contribute such a configuration, check out Sindri&#39;s{" "}
        <a
          className="font-bold hover:underline"
          href="/api/v1"
          target="_blank"
          rel="noopener noreferrer"
        >
          API
        </a>
        .
      </p>
      <p className="pb-4">
        Image references are of the form{" "}
        <code className="font-mono bg-black rounded text-white p-1">
          {host}/{"<steamapp-id>:<steamapp-branch>"}
        </code>
        . If you do not know your Steamapp&#39;s ID, find it on{" "}
        <a
          className="font-bold hover:underline"
          href="https://steamdb.info/"
          target="_blank"
          rel="noopener noreferrer"
        >
          SteamDB
        </a>
        . There is a special case for the default tag,{" "}
        <code className="font-mono bg-black rounded text-white p-1">
          :{defaultTag}
        </code>
        , which gets mapped to the default Steamapp branch, {defaultBranch}.
        Supported Steamapps can be found below.
      </p>
      <div className="flex justify-start items-center py-4">
        <button
          onClick={() => setAddModalOpen(true)}
          className="bg-blue-400 hover:bg-blue-600 text-white font-bold py-2 px-5 rounded inline-flex items-center text-lg"
        >
          Add
        </button>
      </div>
      {!!steamapps.length && (
        <>
          <table>
            <thead>
              <tr>
                <th className="border-gray-500" />
                <th className="border-gray-500 font-bold">Steamapp</th>
                <th className="border-gray-500 font-bold">Image</th>
              </tr>
            </thead>
            <tbody>
              {steamapps.map((steamapp, i) => {
                return (
                  <tr key={i} className="border-t border-gray-500">
                    <td className="p-2 border-gray-500 flex justify-center items-center">
                      <img
                        src={steamapp.icon_url}
                        alt={`${steamapp.name} icon`}
                        className="size-8 rounded object-contain"
                      />
                    </td>
                    <td className="border-gray-500">
                      <a
                        className="font-bold hover:underline"
                        href={`https://steamdb.info/app/${steamapp.app_id}/`}
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        {steamapp.name}
                      </a>
                      {steamapp.branch && steamapp.branch !== defaultBranch
                        ? `'s ${steamapp.branch} branch`
                        : ""}
                    </td>
                    <td className="border-gray-500">
                      <code className="font-mono">
                        {host}/{steamapp.app_id}
                        {steamapp.branch
                          ? `:${steamapp.branch}`
                          : `:${defaultTag}`}
                      </code>
                    </td>
                    <td className="border-gray-500">
                      <button
                        onClick={() =>
                          getSteamappDetails(i)
                            .then(() => setSelectedSteamapp(i))
                            .catch(setErr)
                        }
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
            open={steamapps.length > selectedSteamapp && selectedSteamapp >= 0}
            onClose={() => setSelectedSteamapp(-1)}
            steamapp={steamapps[selectedSteamapp] as Steamapp}
            lines={16}
          />
          <AddModal
            open={addModalOpen}
            onClose={() => setAddModalOpen(false)}
          />
          {!!token && (
            <div className="flex justify-center items-center py-4">
              <button
                onClick={() => more(token)}
                className="bg-blue-400 hover:bg-blue-600 text-white font-bold py-2 px-4 rounded"
              >
                Load More
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
