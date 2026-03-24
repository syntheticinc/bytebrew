interface EmptyStateProps {
  icon?: string;
  message: string;
  description?: string;
  action?: {
    label: string;
    onClick: () => void;
  };
}

export default function EmptyState({ icon, message, description, action }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 px-6">
      {icon && (
        <div className="text-4xl mb-4 opacity-60">{icon}</div>
      )}
      <p className="text-base font-medium text-zinc-500 dark:text-zinc-400 mb-1">{message}</p>
      {description && (
        <p className="text-sm text-brand-shade3 mb-4 text-center max-w-sm">{description}</p>
      )}
      {action && (
        <button
          onClick={action.onClick}
          className="mt-2 px-4 py-2 bg-brand-accent text-brand-light rounded-btn text-sm font-medium hover:bg-brand-accent-hover transition-colors"
        >
          {action.label}
        </button>
      )}
    </div>
  );
}
