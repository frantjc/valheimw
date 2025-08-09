import React from "react";
import { IoMdAdd, IoMdClose } from "react-icons/io";
import { FiSend } from "react-icons/fi";
import { SteamappUpsert } from "~/client";
import { DockerfilePreview } from "./dockerfile-preview";
import { DivIfProps } from "./div-if-props";

type Invalid = {
  field: keyof SteamappUpsert;
  reason: string;
};

export type SteamappFormProps = Omit<
  React.DetailedHTMLProps<React.HTMLAttributes<HTMLDivElement>, HTMLDivElement>,
  "onChange" | "onSubmit"
> & {
  steamapp: SteamappUpsert;
  editing?: boolean;
  onChange: (_: SteamappUpsert) => void;
  onSubmit: (_: SteamappUpsert) => void;
};

export function SteamappForm({
  steamapp,
  editing = false,
  onChange,
  onSubmit,
  ...rest
}: SteamappFormProps) {
  function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    onSubmit(steamapp);
  }

  function validate(steamapp: SteamappUpsert): Invalid[] {
    let inv: Invalid[] = [];

    if (steamapp.app_id <= 0) {
      inv = inv.concat({
        field: "app_id",
        reason: "Steamapp IDs must be greater than zero",
      });
    }

    if (steamapp.app_id % 10 !== 0) {
      inv = inv.concat({
        field: "app_id",
        reason: "Steamapp IDs must be divisible by 10",
      });
    }

    if (isBetaPasswordRequired() && !steamapp.beta_password?.trim()) {
      inv = inv.concat({
        field: "beta_password",
        reason: "Non-public Steamapp branches must have beta passwords",
      });
    }

    return inv;
  }

  function isBetaPasswordRequired() {
    const branch = steamapp?.branch?.trim();
    return !!branch && branch !== "public";
  }

  const invalids = validate(steamapp);
  const isFormValid = !invalids.length;

  return (
    <DivIfProps {...rest}>
      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <StringInput
          steamapp={steamapp}
          onChange={onChange}
          title="App ID *"
          field="app_id"
          type="number"
          required
          disabled={editing}
          invalidReasons={invalids
            .filter((invalid) => invalid.field === "app_id")
            .map((invalid) => invalid.reason)}
        />
        <StringInput
          steamapp={steamapp}
          onChange={onChange}
          title="Base Image"
          field="base_image"
          invalidReasons={invalids
            .filter((invalid) => invalid.field === "base_image")
            .map((invalid) => invalid.reason)}
        />
        <StringArrayInput
          steamapp={steamapp}
          onChange={onChange}
          title="APT Packages"
          field="apt_packages"
          placeholder="curl"
        />
        <div>
          <div className="flex flex-col gap-2">
            <label htmlFor="platform_type" className="text-sm font-medium">
              Platform Type
            </label>
            <select
              id="platform_type"
              className="grow p-2 bg-zinc-100 dark:bg-zinc-700 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              value={steamapp.platform_type || ""}
              onChange={(e) =>
                onChange({
                  ...steamapp,
                  platform_type: e.target.value,
                })
              }
            >
              <option value="">Select platform...</option>
              <option value="linux">Linux</option>
              <option value="windows">Windows</option>
              <option value="macos">macOS</option>
            </select>
          </div>
        </div>
        <StringArrayInput
          steamapp={steamapp}
          onChange={onChange}
          title="Executables"
          field="execs"
          placeholder="ln -s /home/steam/linux64/steamclient.so /usr/lib/x86_64-linux-gnu/steamclient.so"
        />
        <StringArrayInput
          steamapp={steamapp}
          onChange={onChange}
          title="Entrypoint"
          field="entrypoint"
          placeholder="/home/steam/start_server.sh"
        />
        <StringArrayInput
          steamapp={steamapp}
          onChange={onChange}
          title="Command"
          field="cmd"
        />
        <StringInput
          steamapp={steamapp}
          onChange={onChange}
          title="Branch"
          field="branch"
          disabled={editing}
          invalidReasons={invalids
            .filter((invalid) => invalid.field === "branch")
            .map((invalid) => invalid.reason)}
        />
        <StringInput
          steamapp={steamapp}
          onChange={onChange}
          title={`Beta Password${isBetaPasswordRequired() ? " *" : ""}`}
          field="beta_password"
          required={isBetaPasswordRequired()}
          disabled={!isBetaPasswordRequired()}
          invalidReasons={invalids
            .filter((invalid) => invalid.field === "beta_password")
            .map((invalid) => invalid.reason)}
        />
        <button
          type="submit"
          disabled={isFormValid}
          className={`flex justify-center items-center gap-2 p-2 w-32 mx-auto rounded border-2 ${
            !isFormValid
              ? "border-gray-400 text-gray-400 cursor-not-allowed"
              : "text-black dark:text-white border-black dark:border-white hover:border-gray-500 hover:text-gray-500 cursor-pointer"
          }`}
        >
          <FiSend />
          Submit
        </button>
      </form>
    </DivIfProps>
  );
}

function StringInput({
  title,
  placeholder,
  required = false,
  disabled = false,
  type = "text",
  field,
  steamapp,
  onChange,
  invalidReasons,
  ...rest
}: {
  title: string;
  placeholder?: string;
  required?: boolean;
  disabled?: boolean;
  type?: React.HTMLInputTypeAttribute;
  field: "app_id" | "base_image" | "branch" | "beta_password";
  steamapp: SteamappUpsert;
  onChange: (_: SteamappUpsert) => void;
  invalidReasons?: string[];
} & Omit<
  React.DetailedHTMLProps<React.HTMLAttributes<HTMLDivElement>, HTMLDivElement>,
  "onChange"
>) {
  return (
    <DivIfProps {...rest}>
      <div className="flex flex-col gap-2">
        <label htmlFor={field} className="text-sm font-medium">
          {title}
        </label>
        <input
          id={field}
          type={type}
          required={required}
          disabled={disabled}
          placeholder={placeholder}
          className={`${disabled ? "bg-gray-400 cursor-not-allowed " : "bg-zinc-100 dark:bg-zinc-700 "}${type === "number" ? "[appearance:textfield] " : ""}grow p-2 rounded focus:outline-none focus:ring-2 focus:ring-blue-500`}
          value={steamapp[field] || ""}
          onChange={(e) =>
            onChange({
              ...steamapp,
              [field]:
                field === "app_id"
                  ? parseInt(e.target.value) || 0
                  : e.target.value,
            })
          }
        />
        {!!invalidReasons?.length && (
          <span className="text-red-500 text-xs">
            {invalidReasons.join(", ")}
          </span>
        )}
      </div>
    </DivIfProps>
  );
}

function StringArrayInput({
  title,
  placeholder,
  field,
  steamapp,
  onChange,
  ...rest
}: {
  title: string;
  placeholder?: string;
  field: "apt_packages" | "execs" | "entrypoint" | "cmd";
  steamapp: SteamappUpsert;
  onChange: (_: SteamappUpsert) => void;
} & Omit<
  React.DetailedHTMLProps<React.HTMLAttributes<HTMLDivElement>, HTMLDivElement>,
  "onChange"
>) {
  const [input, setInput] = React.useState("");

  function handleAdd() {
    if (input.trim() && !(steamapp[field] ?? []).includes(input.trim())) {
      onChange({
        ...steamapp,
        [field]: [...(steamapp[field] ?? []), input.trim()],
      });
      setInput("");
    }
  }

  function handleRemove(index: number) {
    onChange({
      ...steamapp,
      [field]: (steamapp[field] ?? []).filter((_, i) => i !== index),
    });
  }

  return (
    <DivIfProps {...rest}>
      <div className="flex flex-col gap-2">
        <label htmlFor={field} className="text-sm font-medium">
          {title}
        </label>
        <div className="flex flex-row gap-2">
          <input
            type="text"
            placeholder={placeholder}
            className="grow p-2 bg-zinc-100 dark:bg-zinc-700 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                e.stopPropagation();
                handleAdd();
              }
            }}
          />
          <button onClick={handleAdd} className="p-2 hover:text-gray-500">
            <IoMdAdd />
          </button>
        </div>
        {(steamapp[field] ?? []).length > 0 && (
          <div className="flex flex-wrap gap-2">
            {(steamapp[field] ?? []).map((val, index) => (
              <span
                key={index}
                className="bg-gray-700 px-3 p-1 rounded-full text-sm flex items-center gap-2"
              >
                {val}
                <button
                  type="button"
                  onClick={() => handleRemove(index)}
                  className="hover:text-gray-500"
                >
                  <IoMdClose />
                </button>
              </span>
            ))}
          </div>
        )}
      </div>
    </DivIfProps>
  );
}

export function SteamappFormWithDockerfilePreview({
  steamapp,
  editing,
  onSubmit,
  onChange,
  ...rest
}: SteamappFormProps) {
  return (
    <DivIfProps {...rest}>
      <div className="grid grid-cols-1 xl:grid-cols-2 gap-8">
        <SteamappForm
          steamapp={steamapp}
          editing={editing}
          onSubmit={onSubmit}
          onChange={onChange}
        />
        <DockerfilePreview steamapp={steamapp} />
      </div>
    </DivIfProps>
  );
}
