import { ChevronLeft, ChevronRight } from 'lucide-react';
import { useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';

export const DEFAULT_LIMIT = 20;
const MAX_LIMIT = 100;

// Страница живёт в query-строке, чтобы ссылка открывалась тем же видом.
export function usePage(prefix = '') {
  const [params, setParams] = useSearchParams();
  const offKey = `${prefix}offset`;
  const limKey = `${prefix}limit`;
  const limit = Math.min(Math.max(Number(params.get(limKey)) || DEFAULT_LIMIT, 1), MAX_LIMIT);
  const offset = Math.max(Number(params.get(offKey)) || 0, 0);

  const setOffset = useCallback(
    (next: number) =>
      setParams(
        (p) => {
          const copy = new URLSearchParams(p);
          if (next > 0) copy.set(offKey, String(next));
          else copy.delete(offKey);
          return copy;
        },
        { replace: true },
      ),
    [offKey, setParams],
  );

  return { limit, offset, setOffset };
}

// Общего количества API не отдаёт: «Вперёд» гаснет, если пришло меньше limit.
export function Pagination({
  limit,
  offset,
  count,
  onChange,
}: {
  limit: number;
  offset: number;
  count: number | undefined;
  onChange: (offset: number) => void;
}) {
  const hasPrev = offset > 0;
  const hasNext = (count ?? 0) >= limit;
  if (!hasPrev && !hasNext) return null;
  return (
    <div className="flex items-center justify-center gap-1 border-t px-4 py-3 text-sm">
      <button
        className="inline-flex items-center gap-1 rounded-md px-3 py-1.5 text-muted hover:bg-input hover:text-text disabled:pointer-events-none disabled:opacity-40"
        disabled={!hasPrev}
        onClick={() => onChange(Math.max(offset - limit, 0))}
      >
        <ChevronLeft size={16} /> Назад
      </button>
      <span className="px-2 text-xs text-muted num">
        {offset + 1}–{offset + (count ?? 0)}
      </span>
      <button
        className="inline-flex items-center gap-1 rounded-md px-3 py-1.5 text-muted hover:bg-input hover:text-text disabled:pointer-events-none disabled:opacity-40"
        disabled={!hasNext}
        onClick={() => onChange(offset + limit)}
      >
        Вперёд <ChevronRight size={16} />
      </button>
    </div>
  );
}
