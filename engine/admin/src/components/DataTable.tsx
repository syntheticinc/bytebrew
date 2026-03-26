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
      <table className="min-w-full divide-y divide-brand-shade3/15">
        <thead>
          <tr className="bg-brand-dark">
            {columns.map((col) => (
              <th
                key={col.key}
                className={`px-4 py-3 text-left text-[11px] font-semibold text-brand-shade3 uppercase tracking-[0.1em] ${col.className ?? ''}`}
              >
                {col.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="bg-brand-dark-alt divide-y divide-brand-shade3/10">
          {data.map((row) => {
            const rowKey = String(row[keyField]);
            const isActive = activeKey != null && String(activeKey) === rowKey;

            return (
              <tr
                key={rowKey}
                onClick={() => onRowClick?.(row)}
                className={[
                  'transition-all duration-150',
                  onRowClick ? 'cursor-pointer' : '',
                  isActive
                    ? 'bg-brand-accent/8'
                    : '',
                  !isActive && onRowClick ? 'hover:bg-[#1a1a1a]' : '',
                ]
                  .filter(Boolean)
                  .join(' ')}
              >
                {columns.map((col) => (
                  <td key={col.key} className={`px-4 py-3 text-sm text-brand-light ${col.className ?? ''}`}>
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
