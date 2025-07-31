import type { LoaderFunctionArgs, MetaFunction } from "@remix-run/node";
import { useLoaderData } from "@remix-run/react";
import React from "react";
import { BsClipboard, BsClipboardCheck } from "react-icons/bs";
import { HiMagnifyingGlass } from "react-icons/hi2";
import { IoMdAdd } from "react-icons/io";
import { MdExpandMore, MdOutlineEdit } from "react-icons/md";
import {
  getSteamapp,
  getSteamapps,
  Steamapp,
  SteamappSummary,
  SteamappUpsert,
  upsertSteamapp,
} from "~/client";
import {
  DockerfilePreview,
  Modal,
  SteamappFormWithDockerfilePreview,
} from "~/components";

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

const defaultAddForm: SteamappUpsert = {
  app_id: 0,
  base_image: "docker.io/library/debian:stable-slim",
  apt_packages: [],
  launch_type: "",
  platform_type: "linux",
  execs: [],
  entrypoint: [],
  cmd: [],
  branch: defaultBranch,
  beta_password: "",
};

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

  const [activity, setActivity] = React.useState<
    "adding" | "editing" | "viewing" | undefined
  >(undefined);
  const [addForm, setAddForm] = React.useState<SteamappUpsert>(defaultAddForm);
  const [editForm, setEditForm] =
    React.useState<SteamappUpsert>(defaultAddForm);
  const [activityAppID, setActivityAppID] = React.useState<number | undefined>(
    undefined,
  );

  React.useEffect(() => {
    function parseInitialFragment(): {
      activity: "adding" | "editing" | "viewing" | undefined;
      appId: number | undefined;
    } {
      const fragment = window.location.hash.slice(1);

      if (fragment === "add") {
        return { activity: "adding" as const, appId: undefined };
      } else if (fragment.startsWith("edit/")) {
        const appId = parseInt(fragment.split("/")[1]);
        if (!isNaN(appId) && appId >= 0) {
          return { activity: "editing" as const, appId: appId };
        }
      } else if (fragment.startsWith("view/")) {
        const appId = parseInt(fragment.split("/")[1]);
        if (!isNaN(appId) && appId >= 0) {
          return { activity: "viewing" as const, appId: appId };
        }
      }

      return { activity: undefined, appId: undefined };
    }

    const { activity: parsedActivity, appId } = parseInitialFragment();
    if (parsedActivity) {
      setActivity(parsedActivity);
      setActivityAppID(appId);
    }
  }, []);

  const handleErr = React.useCallback(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (err: any) => {
      if (err instanceof Error) {
        setErr(err);
      } else if (err instanceof Response) {
        setErr(new Error(`${err.status}: ${err.statusText}`));
      } else {
        setErr(new Error(err));
      }
    },
    [setErr],
  );

  const getMoreSteamapps = React.useCallback(
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
                    existing.branch === app.branch,
                ),
            ),
          ]);
          setToken(res.token);
        })
        .catch(handleErr);
    },
    [setSteamapps, setToken, handleErr],
  );

  React.useEffect(() => {
    if (steamapps.length === 0) {
      getMoreSteamapps();
    }
  }, [getMoreSteamapps, steamapps]);

  const [prefetchIndex, setPrefetchIndex] = React.useState(0);

  React.useEffect(() => {
    if (steamapps.length && steamapps.length > 1 && !activity) {
      const timeout = setInterval(
        () => setPrefetchIndex((i) => (i + 1) % steamapps.length),
        2000,
      );

      return () => clearTimeout(timeout);
    }
  }, [steamapps, setPrefetchIndex, activity]);

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
    [steamapps, setSteamapps],
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

  React.useEffect(() => {
    if (err) {
      alert(`${err}.`);
    }
  }, [err]);

  React.useEffect(() => {
    if (
      (activity === "editing" || activity === "viewing") &&
      activityAppID &&
      steamapps.length > 0
    ) {
      const steamappIndex = steamapps.findIndex(
        (s) => s.app_id === activityAppID,
      );
      if (steamappIndex >= 0) {
        getSteamappDetails(steamappIndex)
          .then(() => {
            if (activity === "editing") {
              const steamapp = steamapps[steamappIndex] as SteamappUpsert;
              setEditForm(steamapp);
            }
          })
          .catch(handleErr);
      }
    }
  }, [steamapps, activityAppID, activity, getSteamappDetails, handleErr]);

  const steamapp =
    steamapps &&
    steamapps.length > 0 &&
    (steamapps[dockerRunIndex] as Steamapp).base_image &&
    (steamapps[dockerRunIndex] as Steamapp);
  const tag = (steamapp && steamapp.branch) || defaultBranch;
  const branch = tag === defaultTag ? defaultBranch : tag;
  const command =
    !!steamapp &&
    "docker run"
      .concat(
        steamapp.ports
          ? steamapp.ports
              .map((port) => ` -p ${port.port}:${port.port}`)
              .join("")
          : "",
      )
      .concat(` ${host}/${steamapp.app_id.toString()}:${tag}`);

  const [copied, setCopied] = React.useState(false);

  const handleCopy = () => {
    if (!command) {
      // This cannot happen.
      handleErr(new Error("panic"));
      return;
    }

    navigator.clipboard.writeText(command);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const setActivityWithFragment = (
    newActivity: typeof activity,
    appId?: number,
  ) => {
    if (newActivity === "adding") {
      window.location.hash = "#add";
      setActivityAppID(undefined);
    } else if (newActivity === "editing" && appId) {
      window.location.hash = `#edit/${appId}`;
      setActivityAppID(appId);
    } else if (newActivity === "viewing" && appId) {
      window.location.hash = `#view/${appId}`;
      setActivityAppID(appId);
    } else {
      window.location.hash = "";
      setActivityAppID(undefined);
    }
    setActivity(newActivity);
  };

  const openAddModal = () => {
    setAddForm(defaultAddForm);
    setActivityWithFragment("adding");
  };

  const openEditModal = (index: number) => {
    getSteamappDetails(index)
      .then(() => {
        const updatedSteamapp = steamapps[index];
        setEditForm(updatedSteamapp as SteamappUpsert);
        setActivityWithFragment("editing", updatedSteamapp.app_id);
      })
      .catch(handleErr);
  };

  const openViewModal = (index: number) => {
    getSteamappDetails(index)
      .then(() => {
        const updatedSteamapp = steamapps[index];
        setActivityWithFragment("viewing", updatedSteamapp.app_id);
      })
      .catch(handleErr);
  };

  const closeModal = () => {
    setActivityWithFragment(undefined);
  };

  return (
    <div className="flex flex-col gap-8 py-8">
      {!!steamapp && (
        <div className="flex flex-col gap-4">
          <p className="text-3xl">Run the...</p>
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
            <code className="font-mono text-white p-1 overflow-auto pr-4">
              <span className="pr-2 text-gray-500">$</span>
              {command}
            </code>
            <button
              onClick={handleCopy}
              className="text-white hover:text-gray-500 p-2"
            >
              {copied ? <BsClipboardCheck /> : <BsClipboard />}
            </button>
          </pre>
        </div>
      )}
      <p>
        Sindri is a read-only container registry for images with Steamapps
        installed on them.
      </p>
      <p>
        Images are based on{" "}
        <code className="font-mono bg-black rounded text-white p-1">
          debian:stable-slim
        </code>{" "}
        and are nonroot for security purposes.
      </p>
      <p>
        Images are built on-demand, so the pulled Steamapp is always up-to-date.
        To update, just pull the image again.
      </p>
      <p>
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
        </a>{" "}
        or the add button below.
      </p>
      <p>
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
      {!!steamapps.length && (
        <>
          <table>
            <thead>
              <tr>
                <th className="p-2 border-gray-500 flex justify-center items-center">
                  <button
                    onClick={openAddModal}
                    className="hover:text-gray-500 p-2"
                  >
                    <IoMdAdd />
                  </button>
                </th>
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
                    <td className="border-gray-500 text-center">
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
                    <td className="border-gray-500 text-center">
                      <code className="font-mono">
                        {host}/{steamapp.app_id}
                        {steamapp.branch
                          ? `:${steamapp.branch}`
                          : `:${defaultTag}`}
                      </code>
                    </td>
                    <td className="border-gray-500 text-center">
                      <button
                        onClick={() => openViewModal(i)}
                        className="hover:text-gray-500 p-2"
                      >
                        <HiMagnifyingGlass />
                      </button>
                    </td>
                    <td className="border-gray-500 text-center">
                      <button
                        onClick={() => openEditModal(i)}
                        className={`${(steamapp as Steamapp).locked ? "hover:cursor-not-allowed" : "hover:text-gray-500"} p-2`}
                        disabled={(steamapp as Steamapp).locked}
                      >
                        <MdOutlineEdit />
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
          {!!token && (
            <div className="flex justify-center items-center">
              <button
                onClick={() => getMoreSteamapps(token)}
                className="hover:text-gray-500 p-2"
              >
                <MdExpandMore />
              </button>
            </div>
          )}
        </>
      )}
      <Modal open={activity === "adding"} onClose={closeModal}>
        <div className="rounded bg-white dark:bg-gray-950 h-[80vh] w-[90vw]">
          <SteamappFormWithDockerfilePreview
            className="pb-12"
            steamapp={addForm}
            onSubmit={(s) =>
              upsertSteamapp(s)
                .then(() => setActivity(undefined))
                .catch(handleErr)
            }
            onChange={setAddForm}
          />
        </div>
      </Modal>
      <Modal open={activity === "editing"} onClose={closeModal}>
        <div className="rounded bg-white dark:bg-gray-950 h-[80vh] w-[90vw]">
          <SteamappFormWithDockerfilePreview
            editing
            className="pb-12"
            steamapp={editForm}
            onSubmit={(s) =>
              upsertSteamapp(s)
                .then(() => setActivity(undefined))
                .catch(handleErr)
            }
            onChange={setEditForm}
          />
        </div>
      </Modal>
      <Modal open={activity === "viewing"} onClose={closeModal}>
        <div className="rounded bg-white dark:bg-gray-950 h-[80vh] w-[80vw]">
          {activity === "viewing" &&
            activityAppID &&
            (() => {
              const viewingIndex = steamapps.findIndex(
                (s) => s.app_id === activityAppID,
              );
              return (
                viewingIndex >= 0 && (
                  <DockerfilePreview
                    className="pb-12"
                    steamapp={steamapps[viewingIndex] as Steamapp}
                  />
                )
              );
            })()}
        </div>
      </Modal>
    </div>
  );
}
