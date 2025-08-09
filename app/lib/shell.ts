import { Steamapp } from "~/client";
import { dockerfileFromSteamapp } from "./dockerfile";

const defaultBranch = "public";

type ImageRefOpts = {
  steamapp: Pick<Steamapp, "app_id" | "branch">;
  registry: string;
};

export function imageRef({ steamapp, registry }: ImageRefOpts): string {
  const branch = steamapp.branch || defaultBranch;
  const tag = branch ? `:${branch}` : "";
  return `${registry}/${steamapp.app_id}${tag}`;
}

type RunCommandOpts =
  | {
      steamapp: Pick<Steamapp, "app_id" | "branch" | "ports">;
      registry: string;
      url?: never | URL | string;
      method: "pull";
    }
  | {
      steamapp: Pick<Steamapp, "app_id" | "branch" | "ports">;
      registry?: never | string;
      url: URL | string;
      method: "build";
    };

export function runCommand({
  steamapp,
  registry,
  url,
  method = "pull",
}: RunCommandOpts): string {
  const branch = steamapp.branch || defaultBranch;

  switch (method) {
    case "build":
      return `curl ${new URL(
        `/${steamapp.app_id
          .toString()
          .concat(branch === defaultBranch ? "" : `/${branch}`)
          .concat("/run.sh")}`,
        url,
      )} | bash`;
    case "pull":
      const ref = imageRef({ steamapp, registry: registry! });

      return "docker run"
        .concat(
          steamapp.ports
            ? steamapp.ports
                .map((port) => ` -p ${port.port}:${port.port}`)
                .join("")
            : "",
        )
        .concat(` ${ref}`);
  }
}

type RunScriptOpts =
  | {
      steamapp: Steamapp;
      inline: true;
      ref: string;
      url?: never | URL | string;
    }
  | {
      steamapp: Pick<Steamapp, "app_id" | "branch" | "ports">;
      inline?: never | false;
      ref: string;
      url: URL | string;
    };

export function runScript({
  steamapp,
  inline,
  ref,
  url,
}: RunScriptOpts): string {
  if (inline) {
    const dockerfile = dockerfileFromSteamapp(steamapp);

    return [
      "#!/bin/bash",
      `cat <<EOF | docker build -f- -t ${ref} .`,
      dockerfile,
      "EOF",
      "docker run"
        .concat(
          steamapp.ports
            ? steamapp.ports
                .map((port) => ` -p ${port.port}:${port.port}`)
                .join("")
            : "",
        )
        .concat(` ${ref}`),
    ]
      .join("\n")
      .concat("\n");
  }

  const branch = steamapp.branch || defaultBranch;

  return [
    "#!/bin/bash",
    `docker build -f ${new URL(
      `/${steamapp.app_id
        .toString()
        .concat(branch === defaultBranch ? "" : `/${branch}`)
        .concat("/dockerfile")}`,
      url,
    )} -t ${ref} .`,
    "docker run"
      .concat(
        steamapp.ports
          ? steamapp.ports
              .map((port) => ` -p ${port.port}:${port.port}`)
              .join("")
          : "",
      )
      .concat(` ${ref}`),
  ]
    .join("\n")
    .concat("\n");
}
