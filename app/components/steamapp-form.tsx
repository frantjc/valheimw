import React from "react";
import { IoMdAdd, IoMdClose } from "react-icons/io";
import { FiSend } from "react-icons/fi";
import { SteamappUpsert } from "~/client";
import { DockerfilePreview } from "./dockerfile-preview";

enum ValidationReason {
  APP_ID_ZERO = "app_id_zero",
  APP_ID_NOT_DIVISIBLE = "app_id_not_divisible", 
  BETA_PASSWORD_REQUIRED = "beta_password_required"
}

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
  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    onSubmit(steamapp);
  }

  const getFormValidation = (): { isValid: boolean; reason: ValidationReason | null } => {
    if (steamapp.app_id <= 0) {
      return { isValid: false, reason: ValidationReason.APP_ID_ZERO };
    }
    if (steamapp.app_id % 10 !== 0) {
      return { isValid: false, reason: ValidationReason.APP_ID_NOT_DIVISIBLE };
    }
    
    const hasBranch = !!steamapp.branch?.trim() && steamapp.branch.trim() !== "public";
    const hasBetaPassword = !!steamapp.beta_password?.trim();
    
    if (hasBranch && !hasBetaPassword) {
      return { isValid: false, reason: ValidationReason.BETA_PASSWORD_REQUIRED };
    }
    
    return { isValid: true, reason: null };
  };

  const isFormValid = () => {
    return getFormValidation().isValid;
  };

  const isBetaPasswordRequired = () => {
    const branch = steamapp?.branch?.trim();
    return !!branch && branch !== "public";
  };

  return (
    <div {...rest}>
      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <StringInput
          steamapp={steamapp}
          onChange={onChange}
          title="App ID *"
          field="app_id"
          type="number"
          required
          disabled={editing}
          getFormValidation={getFormValidation}
        />
        <StringInput
          steamapp={steamapp}
          onChange={onChange}
          title="Base Image"
          field="base_image"
        />
        <StringArrayInput
          steamapp={steamapp}
          onChange={onChange}
          title="APT Packages"
          field="apt_packages"
          placeholder="curl"
        />
        <StringInput
          steamapp={steamapp}
          onChange={onChange}
          title="Base Image"
          field="base_image"
          placeholder="default"
        />
        <div>
          <div className="flex flex-col gap-2">
            <label htmlFor="platform_type" className="text-sm font-medium">
              Platform Type
            </label>
            <select
              id="platform_type"
              className="grow p-2 bg-zinc-100 bg-dark:bg-zinc-700 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              value={steamapp.platform_type || ""}
              onChange={(e) =>
                onChange({
                  ...steamapp,
                  platform_type: e.target.value,
                })
              }
            >
              <option value="">Select plat...</option>
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
        />
        <StringInput
          steamapp={steamapp}
          onChange={onChange}
          title={`Beta Password${isBetaPasswordRequired() ? " *" : ""}`}
          field="beta_password"
          required={isBetaPasswordRequired()}
          disabled={!isBetaPasswordRequired()}
          getFormValidation={getFormValidation}
        />
<button
  type="submit"
  disabled={!isFormValid()}
  className={`flex justify-center items-center gap-2 p-2 w-32 mx-auto rounded border-2 transition-all duration-200 ${
    !isFormValid() 
      ? "border-gray-400 text-gray-400 cursor-not-allowed" 
      : "border-gray-600 dark:border-gray-400 hover:border-gray-800 dark:hover:border-gray-200 hover:shadow-md"
  }`}
>
  <FiSend />
  Submit
</button>
      </form>
    </div>
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
  getFormValidation,
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
  getFormValidation?: () => { isValid: boolean; reason: string | null };
} & Omit<
  React.DetailedHTMLProps<React.HTMLAttributes<HTMLDivElement>, HTMLDivElement>,
  "onChange"
>) {
  const getValidationMessage = () => {
    if (!getFormValidation) return null;
    
    const validation = getFormValidation();
    
    if (!validation.isValid) {
      switch (validation.reason) {
        case ValidationReason.APP_ID_ZERO:
          if (field === "app_id") return "App ID must be greater than 0";
          break;
        case ValidationReason.APP_ID_NOT_DIVISIBLE:
          if (field === "app_id") return "App ID must be divisible by 10";
          break;
        case ValidationReason.BETA_PASSWORD_REQUIRED:
          if (field === "beta_password") return "Beta password is required when branch is set";
          break;
      }
    }
    
    return null;
  };

  const validationMessage = getValidationMessage();

  return (
    <div {...rest}>
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
                field === "app_id" ? parseInt(e.target.value) : e.target.value,
            })
          }   
        />
        {validationMessage && (
          <span className="text-red-500 text-xs">{validationMessage}</span>
        )}
      </div>
    </div>
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
    <div {...rest}>
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
    </div>
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
    <div {...rest}>
      <div className="grid grid-cols-1 xl:grid-cols-2 gap-8">
        <SteamappForm
          steamapp={steamapp}
          editing={editing}
          onSubmit={onSubmit}
          onChange={onChange}
        />
        <DockerfilePreview steamapp={steamapp} />
      </div>
    </div>
  );
}
