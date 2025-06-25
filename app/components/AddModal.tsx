import React from "react";
import { SteamappDetail } from "../client";

type AddModalProps = {
  open: boolean;
  onClose: () => void;
};

export function AddModal({ open, onClose }: AddModalProps) {
  const [formData, setFormData] = React.useState<SteamappDetail & {app_id: number, branch?: string}>({
    app_id: 0,
    base_image: "",
    apt_packages: [],
    beta_password: "",
    launch_type: "",
    platform_type: "",
    execs: [],
    entrypoint: [],
    cmd: [],
    ports: [],
    volumes: [],
    resources: { cpu: "", memory: "" },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: Submit to API
    console.log("Submitting:", formData);
    onClose();
  };

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 flex items-center justify-center bg-black bg-opacity-50 z-50 px-4"
      onClick={onClose}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          onClose();
        }
      }}
    >
      <div
        className="bg-white rounded shadow-lg w-full max-w-screen-lg overflow-hidden cursor-default"
        onClick={(e) => e.stopPropagation()}
        role="presentation"
      >
        <div className="flex justify-between items-center px-4 py-2 border-b border-gray-700 bg-gray-800">
          <span className="font-bold text-lg text-white">Add Steam App</span>
          <div className="flex gap-2">
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-white font-bold px-2 py-1 rounded"
            >
              Close
            </button>
          </div>
        </div>
        <form
          onSubmit={handleSubmit}
          className="p-6 overflow-y-auto max-h-[calc(90vh-60px)]"
        >
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="space-y-4">
              <div>
                <label
                  htmlFor="app_id"
                  className="block text-sm font-medium mb-1"
                >
                  App ID *
                </label>
                <input
                  id="app_id"
                  type="number"
                  required
                  className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                  value={formData.app_id || ""}
                  onChange={(e) =>
                    setFormData((prev) => ({
                      ...prev,
                      app_id: parseInt(e.target.value) || 0,
                    }))
                  }
                />
              </div>
            </div>
          </div>

          <div className="flex justify-start gap-3 mt-8 pt-4 border-t">
            <button
              type="submit"
              className="px-4 py-2 bg-blue-400 text-white rounded hover:bg-blue-600"
            >
              Submit
            </button>
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-gray-600 border border-gray-300 rounded hover:bg-gray-50"
            >
              Cancel
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
