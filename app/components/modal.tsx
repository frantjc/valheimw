import { IoMdClose } from "react-icons/io";

export type ModalProps = React.PropsWithChildren<{
  open: boolean;
  onClose: () => void;
}> &
  React.DetailedHTMLProps<React.HTMLAttributes<HTMLDivElement>, HTMLDivElement>;

export function Modal({ open, onClose, children, ...rest }: ModalProps) {
  return (
    <div
      className={`fixed inset-0 flex items-center justify-center cursor-default bg-black bg-opacity-50 z-50 ${open ? "block" : "hidden"}`}
      onClick={onClose}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === "Escape") {
          onClose();
        }
      }}
    >
      {children && (
        <div
          className="relative bg-white dark:bg-gray-950 rounded shadow-lg overflow-auto p-12"
          onClick={(e) => e.stopPropagation()}
          role="presentation"
          onKeyDown={(e) => {
            if (e.key === "Escape") {
              onClose();
            }
          }}
        >
          <button
            onClick={onClose}
            className="hover:text-gray-500 absolute top-2 right-2 p-2"
            aria-label="Close"
          >
            <IoMdClose />
          </button>
          <div {...rest}>{children}</div>
        </div>
      )}
    </div>
  );
}
