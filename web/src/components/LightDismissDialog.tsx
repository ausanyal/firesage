import React, { useEffect, useId, useRef } from 'react';

interface LightDismissDialogProps {
  open: boolean;
  onClose: () => void;
  title: string;
  subtitle?: string;
  maxWidth?: string;
  children: React.ReactNode;
}

export const LightDismissDialog: React.FC<LightDismissDialogProps> = ({
  open,
  onClose,
  title,
  subtitle,
  maxWidth = '860px',
  children,
}) => {
  const dialogRef = useRef<HTMLDialogElement | null>(null);
  const titleId = useId();

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;

    if (open && !dialog.open) {
      dialog.showModal();
    } else if (!open && dialog.open) {
      dialog.close();
    }
  }, [open]);

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;

    const handleClose = () => {
      onClose();
    };
    dialog.addEventListener('close', handleClose);

    // Mandatory Safari fallback per modern-web-guidance `light-dismiss-a-dialog`
    let fallbackClickHandler: ((event: MouseEvent) => void) | null = null;
    if (!('closedBy' in HTMLDialogElement.prototype)) {
      fallbackClickHandler = (event: MouseEvent) => {
        if (event.target !== dialog) return;
        const rect = dialog.getBoundingClientRect();
        const isDialogContent =
          rect.top <= event.clientY &&
          event.clientY <= rect.top + rect.height &&
          rect.left <= event.clientX &&
          event.clientX <= rect.left + rect.width;
        if (isDialogContent) return;
        dialog.close();
      };
      dialog.addEventListener('click', fallbackClickHandler);
    }

    return () => {
      dialog.removeEventListener('close', handleClose);
      if (fallbackClickHandler) {
        dialog.removeEventListener('click', fallbackClickHandler);
      }
    };
  }, [onClose]);

  return (
    <dialog
      ref={dialogRef}
      {...({ closedby: 'any' } as Record<string, string>)}
      aria-labelledby={titleId}
      className="fs-dialog"
      style={{ maxWidth }}
    >
      <div className="fs-dialog-inner">
        <header className="fs-dialog-header">
          <div>
            <h2 id={titleId} className="fs-dialog-title">
              {title}
            </h2>
            {subtitle && <p className="fs-dialog-subtitle">{subtitle}</p>}
          </div>
          <form method="dialog">
            <button type="submit" className="fs-btn fs-btn-ghost" aria-label="Close dialog">
              ✕ Close
            </button>
          </form>
        </header>
        <div className="fs-dialog-body">{children}</div>
      </div>
    </dialog>
  );
};
