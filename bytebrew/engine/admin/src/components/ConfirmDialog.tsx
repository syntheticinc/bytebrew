import Modal from './Modal';

interface ConfirmDialogProps {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  message: React.ReactNode;
  confirmLabel?: string;
  loading?: boolean;
  variant?: 'danger' | 'default';
}

export default function ConfirmDialog({
  open,
  onClose,
  onConfirm,
  title,
  message,
  confirmLabel = 'Confirm',
  loading,
  variant = 'default',
}: ConfirmDialogProps) {
  const btnClass =
    variant === 'danger'
      ? 'bg-red-600 hover:bg-red-700 text-white'
      : 'bg-brand-accent hover:bg-brand-accent-hover text-brand-light';

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={title}
      className="max-w-sm"
      footer={
        <div className="flex justify-end gap-3">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm text-brand-shade2 border border-brand-shade3/30 rounded-btn hover:bg-brand-dark hover:text-brand-light transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            disabled={loading}
            className={`px-4 py-2 text-sm rounded-btn font-medium disabled:opacity-50 transition-colors ${btnClass}`}
          >
            {loading ? 'Processing...' : confirmLabel}
          </button>
        </div>
      }
    >
      <div className="text-sm text-brand-shade2">{message}</div>
    </Modal>
  );
}
