import { AddForm } from "./AddForm";

type AddModalProps = {
  open: boolean;
  onClose: () => void;
};

export function AddModal({ open, onClose }: AddModalProps) {
  return (
    <div
      className={`fixed inset-0 flex items-center justify-center bg-black bg-opacity-50 z-50 px-4 ${
        open ? "block" : "hidden"
      }`}
      onClick={onClose}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.target === e.currentTarget) {
          if (e.key === "Escape" || e.key === "Enter" || e.key === " ") {
            onClose();
          }
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
        <AddForm onClose={onClose} />
      </div>
    </div>
  );
}
