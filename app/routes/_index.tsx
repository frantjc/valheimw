import type { LoaderFunctionArgs, MetaFunction } from "@remix-run/node";
import { useLoaderData } from "@remix-run/react";
import React from "react";
import { BsClipboard, BsClipboardCheck } from "react-icons/bs";
import { HiMagnifyingGlass } from "react-icons/hi2";
import { IoMdAdd } from "react-icons/io";
import { MdExpandMore, MdOutlineEdit } from "react-icons/md";
import {
  getSteamapps,
  Steamapp,
  SteamappUpsert,
  upsertSteamapp,
} from "~/client";
import {
  DockerfilePreview,
  Modal,
  SteamappFormWithDockerfilePreview,
} from "~/components";
import { useErr, useSteamapps } from "~/hooks";

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
  const host = process.env.BOILER_URL
    ? new URL(process.env.BOILER_URL).host
    : args.request.headers.get("Host") ||
      `localhost:${process.env.PORT || "3000"}`;

  const featureFlags = Object.entries(process.env).reduce(
    (acc, [env, value]) => {
      const featureFlagPrefix = "FEATURE_FLAG_";

      if (env.startsWith(featureFlagPrefix)) {
        const key = env
          .slice(featureFlagPrefix.length)
          .toLowerCase()
          .replace(/_([a-z])/g, (_, c) => c.toUpperCase());

        return {
          ...acc,
          [key]: value ?? "",
        };
      }

      return acc;
    },
    {} as Record<string, string>,
  );

  return getSteamapps()
    .then(({ token, steamapps }) => {
      return {
        host,
        steamapps,
        token,
        featureFlags,
      };
    })
    .catch(() => {
      return { host, steamapps: [], token: "", featureFlags };
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

function parseBool(value: string | undefined): boolean {
  return ["1", "true", "yes"].includes(value ?? "");
}

const ActivityAdd = "add";
const ActivityEdit = "edit";
const ActivityView = "view";
const Activities = [ActivityAdd, ActivityEdit, ActivityView] as const;
type Activity = (typeof Activities)[number];

export default function Index() {
  const {
    host,
    steamapps: initialSteamapps,
    token: initialToken,
    featureFlags,
  } = useLoaderData<typeof loader>();

  const handleErr = useErr();

  const { steamapps, getSteamappDetails, getMoreSteamapps } = useSteamapps({
    steamapps: initialSteamapps,
    token: initialToken,
  });

  const [modal, setModal] = React.useState<{
    activity: Activity;
    appID?: number;
    branch?: string;
  }>();

  const closeModal = React.useCallback(() => setModal(undefined), [setModal]);

  React.useEffect(() => {
    const [initialActivity, rawInitialActivityAppID, initalBranch] =
      window.location.hash.slice(1).split("/");

    switch (initialActivity) {
      case ActivityEdit:
      case ActivityView:
        // eslint-disable-next-line no-case-declarations
        const initialActivityAppID = parseInt(rawInitialActivityAppID);
        if (initialActivityAppID) {
          setModal({
            activity: initialActivity,
            appID: initialActivityAppID,
            branch: initalBranch,
          });
        }
        break;
      case ActivityAdd:
        setModal({ activity: initialActivity });
        break;
    }
  }, [setModal]);

  React.useEffect(() => {
    if (modal?.appID) {
      getSteamappDetails({ appID: modal.appID, branch: modal.branch })
        .then((steamapp) => {
          if (modal.activity === ActivityEdit) {
            setEditForm(steamapp);
          }
        })
        .catch(handleErr);
    }
  }, [steamapps, modal, getSteamappDetails, handleErr]);

  const [addForm, setAddForm] = React.useState<SteamappUpsert>(defaultAddForm);

  const [editForm, setEditForm] =
    React.useState<SteamappUpsert>(defaultAddForm);

  const [prefetchIndex, setPrefetchIndex] = React.useState(0);

  React.useEffect(() => {
    if (
      steamapps.length &&
      steamapps.length > 1 &&
      // The following condition is just to pause the "animation" when a modal is open.
      !modal?.activity
    ) {
      const timeout = setInterval(
        () => setPrefetchIndex((i) => (i + 1) % steamapps.length),
        2000,
      );

      return () => clearTimeout(timeout);
    }
  }, [steamapps, setPrefetchIndex, modal]);

  const [dockerRunIndex, setDockerRunIndex] = React.useState(0);

  React.useEffect(() => {
    if (steamapps.length > prefetchIndex && prefetchIndex >= 0) {
      getSteamappDetails({ index: prefetchIndex })
        .then(() => {
          setDockerRunIndex(prefetchIndex);
        })
        .catch(() => {
          // Do not alert the user because this is a background process.
        });
    }
  }, [prefetchIndex, getSteamappDetails, setDockerRunIndex, steamapps]);

  const noBoiler = parseBool(featureFlags.noBoiler);

  const steamapp =
    steamapps &&
    steamapps.length > 0 &&
    (steamapps[dockerRunIndex] as Steamapp).base_image &&
    (steamapps[dockerRunIndex] as Steamapp);
  const branch = (steamapp && steamapp.branch) || defaultBranch;
  const tag = branch === defaultBranch ? "" : `:${branch}`;
  const command = noBoiler
    ? !!steamapp &&
      `docker build --tag ${host}/${steamapp?.app_id.toString()}${tag} .`
        .concat(" && docker run")
        .concat(
          steamapp.ports
            ? steamapp.ports
                .map((port) => ` -p ${port.port}:${port.port}`)
                .join("")
            : "",
        )
        .concat(` ${host}/${steamapp.app_id.toString()}${tag}`)
    : !!steamapp &&
      "docker run"
        .concat(
          steamapp.ports
            ? steamapp.ports
                .map((port) => ` -p ${port.port}:${port.port}`)
                .join("")
            : "",
        )
        .concat(` ${host}/${steamapp.app_id.toString()}${tag}`);

  const [copied, setCopied] = React.useState(false);

  const handleCopy = React.useCallback(
    (text: string) => {
      return navigator.clipboard.writeText(text).then(() => {
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      });
    },
    [setCopied],
  );

  React.useEffect(() => {
    let hash = "";

    if (modal?.activity) {
      hash += modal.activity;
    }

    if (modal?.appID) {
      hash += `/${modal.appID}`;

      if (modal?.branch) {
        hash += `/${modal.branch}`;
      }
    }

    if (hash || window.location.hash) {
      window.location.hash = hash;
    }
  }, [modal]);

  return (
    <div className="flex flex-col gap-8 py-8">
      {noBoiler ? (
        <>
          <p>
            Sindri is a database of Dockerfiles to build container images with
            Steamapps installed on them.
          </p>
          {!!steamapp && (
            <>
              <p>
                Find your Steamapp below, click the
                <button className="hover:cursor-default p-2">
                  <HiMagnifyingGlass />
                </button>
                to preview and download its Dockerfile, open a terminal in the
                download directory, and then run the...
              </p>
              <p className="text-xl">
                <a
                  className="font-bold hover:underline"
                  href={`https://steamdb.info/app/${steamapp.app_id}/`}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  {steamapp.name}
                </a>
                {branch !== defaultBranch && (
                  <span>&#39;s {branch} branch</span>
                )}
              </p>
              <pre className="bg-black flex p-2 px-4 rounded items-center justify-between w-full border border-gray-500">
                <code className="font-mono text-white p-1 overflow-auto pr-4">
                  <span className="pr-2 text-gray-500">$</span>
                  {command}
                </code>
                {command && (
                  <button
                    onClick={() => handleCopy(command).catch(handleErr)}
                    className="text-white hover:text-gray-500 p-2"
                  >
                    {copied ? <BsClipboardCheck /> : <BsClipboard />}
                  </button>
                )}
              </pre>
            </>
          )}
          <p>
            Images are based on{" "}
            <code className="font-mono bg-black rounded text-white p-1">
              debian:stable-slim
            </code>{" "}
            and are nonroot for security purposes.
          </p>
          <p>
            Steamapps commonly do not work out of the box, missing dependencies,
            specifying an invalid entrypoint, or just generally not being
            container-friendly. Sindri attemps to fix this by crowd-sourcing
            configurations to apply to the images. To contribute such a
            configuration, click the
            <button
              onClick={() => setModal({ activity: ActivityAdd })}
              className="hover:text-gray-500 p-2"
            >
              <IoMdAdd />
            </button>
            button.
          </p>
          <p>
            If you do not know your Steamapp&#39;s ID, find it on{" "}
            <a
              className="font-bold hover:underline"
              href="https://steamdb.info/"
              target="_blank"
              rel="noopener noreferrer"
            >
              SteamDB
            </a>
            .
          </p>
        </>
      ) : (
        <>
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
                {branch !== defaultBranch && (
                  <span>&#39;s {branch} branch</span>
                )}
              </p>
              <pre className="bg-black flex p-2 px-4 rounded items-center justify-between w-full border border-gray-500">
                <code className="font-mono text-white p-1 overflow-auto pr-4">
                  <span className="pr-2 text-gray-500">$</span>
                  {command}
                </code>
                {command && (
                  <button
                    onClick={() => handleCopy(command).catch(handleErr)}
                    className="text-white hover:text-gray-500 p-2"
                  >
                    {copied ? <BsClipboardCheck /> : <BsClipboard />}
                  </button>
                )}
              </pre>
            </div>
          )}
          <p>
            Sindri is a read-only container registry for images with Steamapps
            installed on them.
          </p>
          <p>
            Images are built on-demand, so the pulled Steamapp is always
            up-to-date. To update, just pull the image again.
          </p>
          <p>
            Images are based on{" "}
            <code className="font-mono bg-black rounded text-white p-1">
              debian:stable-slim
            </code>{" "}
            and are nonroot for security purposes.
          </p>
          <p>
            Steamapps commonly do not work out of the box, missing dependencies,
            specifying an invalid entrypoint, or just generally not being
            container-friendly. Sindri attemps to fix this by crowd-sourcing
            configurations to apply to the images before returning them. To
            contribute such a configuration, click the
            <button
              onClick={() => setModal({ activity: ActivityAdd })}
              className="hover:text-gray-500 p-2"
            >
              <IoMdAdd />
            </button>
            button.
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
        </>
      )}
      {!!steamapps.length && (
        <>
          <table>
            <thead>
              <tr>
                <th className="p-2 border-gray-500 flex justify-center items-center">
                  <button
                    onClick={() => setModal({ activity: ActivityAdd })}
                    className="hover:text-gray-500 p-2"
                  >
                    <IoMdAdd />
                  </button>
                </th>
                <th className="border-gray-500 font-bold">Steamapp</th>
                {!noBoiler && (
                  <th className="border-gray-500 font-bold">Image</th>
                )}
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
                    {!noBoiler && (
                      <td className="border-gray-500 text-center">
                        <code className="font-mono">
                          {host}/{steamapp.app_id}
                          {steamapp.branch
                            ? `:${steamapp.branch}`
                            : `:${defaultTag}`}
                        </code>
                      </td>
                    )}
                    <td className="border-gray-500 text-center">
                      <button
                        onClick={() =>
                          getSteamappDetails({ index: i }).then((details) =>
                            setModal({
                              activity: ActivityView,
                              appID: details.app_id,
                              branch: details.branch,
                            }),
                          )
                        }
                        className="hover:text-gray-500 p-2"
                      >
                        <HiMagnifyingGlass />
                      </button>
                    </td>
                    <td className="border-gray-500 text-center">
                      <button
                        onClick={() =>
                          getSteamappDetails({ index: i }).then((details) =>
                            setModal({
                              activity: ActivityEdit,
                              appID: details.app_id,
                              branch: details.branch,
                            }),
                          )
                        }
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
          {!!getMoreSteamapps && (
            <div className="flex justify-center items-center">
              <button
                onClick={getMoreSteamapps}
                className="hover:text-gray-500 p-2"
              >
                <MdExpandMore />
              </button>
            </div>
          )}
        </>
      )}
      <Modal open={modal?.activity === ActivityAdd} onClose={closeModal}>
        <div className="rounded bg-white dark:bg-gray-950 h-[80vh] w-[90vw]">
          <SteamappFormWithDockerfilePreview
            className="pb-12"
            steamapp={addForm}
            onSubmit={(s) =>
              upsertSteamapp(s).then(closeModal).catch(handleErr)
            }
            onChange={setAddForm}
          />
        </div>
      </Modal>
      <Modal open={modal?.activity === ActivityEdit} onClose={closeModal}>
        <div className="rounded bg-white dark:bg-gray-950 h-[80vh] w-[90vw]">
          <SteamappFormWithDockerfilePreview
            editing
            className="pb-12"
            steamapp={editForm}
            onSubmit={(s) =>
              upsertSteamapp(s).then(closeModal).catch(handleErr)
            }
            onChange={setEditForm}
          />
        </div>
      </Modal>
      <Modal open={modal?.activity === ActivityView} onClose={closeModal}>
        <div className="rounded bg-white dark:bg-gray-950 h-[80vh] w-[80vw]">
          {modal?.appID && (
            <DockerfilePreview
              className="pb-12"
              steamapp={
                steamapps.find((s) => s.app_id === modal?.appID) as Steamapp
              }
            />
          )}
        </div>
      </Modal>
    </div>
  );
}
