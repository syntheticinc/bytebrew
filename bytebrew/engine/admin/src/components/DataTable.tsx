import EmptyState from './EmptyState';

interface Column<T> {
  key: string;
  header: string;
  render?: (row: T) => React.ReactNode;
  className?: string;
}

interface DataTableProps<T> {
  columns: Column<T>[];
  data: T[];
  keyField: string;
  onRowClick?: (row: T) => void;
  activeKey?: string | number | null;
  emptyMessage?: string;
  emptyIcon?: string;
  emptyAction?: { label: string; onClick: () => void };
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export default function DataTable<T extends Record<string, any>>({
  columns,
  data,
  keyField,
  onRowClick,
  activeKey,
  emptyMessage = 'No data',
  emptyIcon,
  emptyAction,
}: DataTableProps<T>) {
  if (data.length === 0) {
    return (
      <EmptyState
        icon={emptyIcon}
        message={emptyMessage}
        action={emptyAction}
      />
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="min-w-full divide-y divide-brand-shade1">
        <thead>
          <tr className="bg-brand-light/50">
            {columns.map((col) => (
              <th
                key={col.key}
                className={`px-4 py-3 text-left text-xs font-semibold text-brand-shade3 uppercase tracking-wider ${col.className ?? ''}`}
              >
                {col.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="bg-white divide-y divide-brand-shade1/50">
          {data.map((row) => {
            const rowKey = String(row[keyField]);
            const isActive = activeKey != null && String(activeKey) === rowKey;

            return (
              <tr
                key={rowKey}
                onClick={() => onRowClick?.(row)}
                className={[
                  'transition-colors',
                  onRowClick ? 'cursor-pointer' : '',
                  isActive
                    ? 'bg-brand-accent/5 border-l-2 border-l-brand-accent'
                    : 'border-l-2 border-l-transparent',
                  !isActive && onRowClick ? 'hover:bg-brand-light/70' : '',
                ]
                  .filter(Boolean)
                  .join(' ')}
              >
                {columns.map((col) => (
                  <td key={col.key} className={`px-4 py-3 text-sm text-brand-dark ${col.className ?? ''}`}>
                    {col.render ? col.render(row) : String(row[col.key] ?? '')}
                  </td>
                ))}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

export type { Column, DataTableProps };
