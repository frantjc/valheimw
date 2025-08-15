import { SteamappUpsert } from "~/client";
import { defaultBranch } from "./shell";

type Instruction =
  | "FROM"
  | "RUN"
  | "COPY"
  | "ADD"
  | "USER"
  | "ENTRYPOINT"
  | "CMD"
  | "SYNTAX"
  | "ESCAPE";

class Directive {
  instruction: Instruction;
  args: string[];

  constructor(instruction: Instruction, ...args: string[]) {
    this.instruction = instruction;
    this.args = args;
  }

  toString(): string {
    switch (this.instruction.toUpperCase()) {
      case "SYNTAX":
        return `# syntax=${this.args.join(" ")}`;
      case "ESCAPE":
        return `# escape=${this.args.join(" ")}`;
      case "FROM":
        return `FROM ${this.args.join(" ")}`;
      case "RUN":
        return `RUN ${this.args.join(" \\\n\t&& ")}`;
      case "USER":
        return `USER ${this.args.join(" ")}`;
      case "COPY":
        return `COPY ${this.args.join(" ")}`;
      case "ADD":
        return `ADD ${this.args.join(" ")}`;
      case "ENTRYPOINT":
        return `ENTRYPOINT [${this.args.map((arg) => `"${arg}"`).join(", ")}]`;
      case "CMD":
        return `CMD [${this.args.map((arg) => `"${arg}"`).join(", ")}]`;
    }

    return "";
  }
}

class Dockerfile {
  directives: Directive[];

  constructor(...directives: Directive[]) {
    this.directives = directives;
  }

  toString(): string {
    return this.directives
      .map(
        (directive, i) =>
          (directive.instruction === "FROM" && i > 0 ? "\n" : "") +
          directive.toString(),
      )
      .join("\n");
  }
}

const user = "steam";
const groupadd = `groupadd --system ${user}`;
const useradd = `useradd --system --gid ${user} --shell /bin/bash --create-home ${user}`;
const mount = "/mnt";

export function dockerfileFromSteamapp(steamapp: SteamappUpsert): Dockerfile {
  const isBeta = steamapp.branch && steamapp.branch !== defaultBranch;
  const betaBranch = isBeta ? ` -beta ${steamapp.branch}` : "";
  const betaPassword = isBeta ? ` -betapassword ${steamapp.beta_password}` : "";
  const isWine =
    steamapp.platform_type === "windows" &&
    steamapp.apt_packages.some((pkg) =>
      ["winehq-stable", "winehq-devel", "winehq-staging"].includes(pkg),
    );
  let baseImage = steamapp.base_image || "debian:stable-slim";
  if (baseImage.startsWith("docker.io/library/")) {
    baseImage = baseImage.slice(18);
  } else if (baseImage.startsWith("docker.io/")) {
    baseImage = baseImage.slice(10);
  }

  return new Dockerfile(
    new Directive("SYNTAX", "docker/dockerfile:1"),
    new Directive("FROM", "steamcmd/steamcmd", "AS", "steamcmd"),
    new Directive(
      "RUN",
      groupadd,
      useradd,
      "steamcmd \\\n" +
        `\t\t+force_install_dir ${mount} \\\n` +
        `\t\t+login anonymous \\\n` +
        `\t\t+@sSteamCmdForcePlatformType ${steamapp.platform_type} \\\n` +
        `\t\t+app_update ${steamapp.app_id || 0}${betaBranch}${betaPassword} \\\n` +
        `\t\t+quit`,
    ),
    ...(isWine
      ? [
          new Directive("FROM", "debian:stable-slim", "AS", "wine"),
          new Directive(
            "ADD",
            "https://dl.winehq.org/wine-builds/winehq.key",
            "/tmp/",
          ),
          new Directive(
            "ADD",
            "https://dl.winehq.org/wine-builds/debian/dists/trixie/winehq-trixie.sources",
            "/mnt/sources.list.d/",
          ),
          new Directive(
            "RUN",
            "apt-get update -y",
            "apt-get install -y --no-install-recommends \\\n\t\tgnupg",
            "mkdir -p /mnt/keyrings",
            "cat /tmp/winehq.key | gpg --dearmor -o /mnt/keyrings/winehq-archive.key -",
          ),
        ]
      : []),
    new Directive("FROM", baseImage),
    ...(isWine
      ? [
          new Directive(
            "RUN",
            "apt-get update -y",
            "apt-get install -y --no-install-recommends \\\n\t\tca-certificates",
            "dpkg --add-architecture i386",
          ),
          new Directive("COPY", "--from=wine", "/mnt", "/etc/apt"),
        ]
      : []),
    new Directive(
      "RUN",
      ...[groupadd, useradd]
        .concat(
          steamapp.apt_packages?.length
            ? [
                "apt-get update -y",
                "apt-get install -y --no-install-recommends \\\n" +
                  steamapp.apt_packages
                    .map((pkg) => `\t\t${pkg}`)
                    .join(" \\\n"),
                "rm -rf /var/lib/apt/lists/*",
                "apt-get clean",
              ]
            : [],
        )
        .concat(steamapp.execs?.length ? steamapp.execs : []),
    ),
    new Directive("USER", user),
    new Directive(
      "COPY",
      "--from=steamcmd",
      `--chown=${user}:${user}`,
      mount,
      `/home/${user}`,
    ),
    ...(steamapp.entrypoint?.length
      ? [new Directive("ENTRYPOINT", ...steamapp.entrypoint)]
      : []),
    ...(steamapp.cmd?.length ? [new Directive("CMD", ...steamapp.cmd)] : []),
  );
}
