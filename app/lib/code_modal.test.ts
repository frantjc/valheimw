import { generateContainerDefinition } from "./code_modal";
import type { Steamapp } from "~/client";

describe("generateContainerDefinition", () => {
  const baseSteamapp: Steamapp = {
    app_id: 896660,
    name: "Valheim",
    branch: "public",
    beta_password: "",
    icon_url: "",
    date_created: new Date(),
    locked: false,

    base_image: "debian:stable-slim",
    apt_packages: [],
    launch_type: "server",
    platform_type: "linux",
    execs: [],
    entrypoint: [],
    cmd: [],
  };

  it("generates a definition for a public branch", () => {
    const def = generateContainerDefinition(baseSteamapp);
    expect(def).toContain("FROM steamcmd/steamcmd AS steamcmd");
    expect(def).toContain("+app_update 896660 \\");
    expect(def).toContain("FROM debian:stable-slim");
  });

  it("includes apt packages if present", () => {
    const steamapp = { ...baseSteamapp, apt_packages: ["libfoo", "libbar"] };
    const def = generateContainerDefinition(steamapp);
    expect(def).toContain("apt-get install -y --no-install-recommends libfoo libbar");
  });

  it("includes execs if present", () => {
    const steamapp = { ...baseSteamapp, execs: ["echo hi", "echo bye"] };
    const def = generateContainerDefinition(steamapp);
    expect(def).toContain("RUN echo hi && echo bye");
  });

  it("includes entrypoint and cmd if present", () => {
    const steamapp = { ...baseSteamapp, entrypoint: ["foo", "bar"], cmd: ["baz"] };
    const def = generateContainerDefinition(steamapp);
    expect(def).toContain('ENTRYPOINT ["foo", "bar"]');
    expect(def).toContain('CMD ["baz"]');
  });

  it("throws if beta branch is missing beta_password", () => {
    const steamapp = { ...baseSteamapp, branch: "beta", beta_password: undefined };
    expect(() => generateContainerDefinition(steamapp)).toThrow(/beta_password/);
  });

  it("includes beta flags if beta branch and password are present", () => {
    const steamapp = { ...baseSteamapp, branch: "beta", beta_password: "pwd" };
    const def = generateContainerDefinition(steamapp);
    expect(def).toContain("+app_update 896660 -beta beta -betapassword pwd \\");
  });

  it("returns empty string if steamapp is undefined", () => {
    expect(generateContainerDefinition(undefined)).toBe("");
  });
});