import { SteamappUpsert } from "~/client";

type Instruction =
  | "FROM"
  | "RUN"
  | "COPY"
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
  const isBeta = steamapp.branch && steamapp.branch !== "public";
  const betaBranch = isBeta ? ` -beta ${steamapp.branch}` : "";
  const betaPassword = isBeta ? ` -betapassword ${steamapp.beta_password}` : "";

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
        `\t\t@sSteamCmdForcePlatformType ${steamapp.platform_type} \\\n` +
        `\t\t+app_update ${steamapp.app_id || 0}${betaBranch}${betaPassword} \\\n` +
        `\t\t+quit`,
    ),
    new Directive(
      "FROM",
      steamapp.base_image?.startsWith("docker.io/library/")
        ? steamapp.base_image.slice(18)
        : steamapp.base_image || "debian:stable-slim",
    ),
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
