import { dockerfileFromSteamapp } from "./dockerfile";
import type { Steamapp } from "~/client";

describe("dockerfileFromSteamapp", () => {
  const baseSteamapp: Steamapp = {
    app_id: 896660,
    name: "Valheim",
    branch: "public",
    beta_password: "",
    icon_url: "",
    date_created: Date.now(),
    locked: false,
    base_image: "docker.io/library/debian:stable-slim",
    apt_packages: [],
    launch_type: "server",
    platform_type: "linux",
    execs: [],
    entrypoint: [],
    cmd: [],
  };

  it("generates a Dockerfile for a public branch", () => {
    const dockerfile = dockerfileFromSteamapp(baseSteamapp).toString();
    expect(dockerfile).toContain("FROM steamcmd/steamcmd AS steamcmd");
    expect(dockerfile).toContain("+app_update 896660");
    expect(dockerfile).toContain("FROM debian:stable-slim");
  });

  it("includes apt_packages", () => {
    const dockerfile = dockerfileFromSteamapp({
      ...baseSteamapp,
      apt_packages: ["libfoo", "libbar"],
    }).toString();
    expect(dockerfile).toContain("apt-get install -y --no-install-recommends");
    expect(dockerfile).toContain("libfoo");
    expect(dockerfile).toContain("libbar");
  });

  it("includes execs", () => {
    const dockerfile = dockerfileFromSteamapp({
      ...baseSteamapp,
      execs: ["echo hello there"],
    }).toString();
    expect(dockerfile).toContain("echo hello there");
  });

  it("includes entrypoint and cmd", () => {
    const dockerfile = dockerfileFromSteamapp({
      ...baseSteamapp,
      entrypoint: ["foo", "bar"],
      cmd: ["baz"],
    }).toString();
    expect(dockerfile).toContain('ENTRYPOINT ["foo", "bar"]');
    expect(dockerfile).toContain('CMD ["baz"]');
  });

  it("includes beta flags", () => {
    const dockerfile = dockerfileFromSteamapp({
      ...baseSteamapp,
      branch: "beta",
      beta_password: "pwd",
    }).toString();
    expect(dockerfile).toContain(
      "+app_update 896660 -beta beta -betapassword pwd \\",
    );
  });
});
