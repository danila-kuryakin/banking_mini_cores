import * as Popover from '@radix-ui/react-popover';
import { useQuery } from '@tanstack/react-query';
import { Bell } from 'lucide-react';
import { useState } from 'react';
import { Link } from 'react-router-dom';
import { notificationsApi } from '../api/notifications';
import { useUser } from '../auth/AuthContext';
import { Spinner } from '../components/Spinner';
import { formatRelative, timeOf } from '../lib/dates';
import { readStorage, writeStorage } from '../lib/storage';
import { qk } from '../features/client/queries';

// Признака «прочитано» в API нет: непрочитанные - новее метки lastSeenAt из localStorage.
function seenKey(userId: string) {
  return `bmc.notifications.seen.${userId}`;
}

export function NotificationsBell() {
  const user = useUser();
  const [lastSeen, setLastSeen] = useState(() => Number(readStorage(seenKey(user.id))) || 0);
  const { data, isLoading } = useQuery({
    queryKey: [...qk.notifications('me'), 'bell'],
    queryFn: async () => (await notificationsApi.list('me', { limit: 20, offset: 0 })).notifications ?? [],
    refetchInterval: 30_000,
  });
  const unread = (data ?? []).filter((n) => timeOf(n.created_at) > lastSeen).length;

  const markSeen = () => {
    const latest = Math.max(Date.now(), ...(data ?? []).map((n) => timeOf(n.created_at)));
    writeStorage(seenKey(user.id), String(latest));
    setLastSeen(latest);
  };

  return (
    <Popover.Root onOpenChange={(open) => open && markSeen()}>
      <Popover.Trigger asChild>
        <button className="relative rounded-lg p-2 text-muted hover:bg-input hover:text-text" aria-label="Уведомления">
          <Bell size={18} />
          {unread > 0 && (
            <span className="absolute right-1 top-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-down px-1 text-[10px] font-semibold text-white">
              {unread > 9 ? '9+' : unread}
            </span>
          )}
        </button>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content
          align="end"
          sideOffset={8}
          className="z-50 w-[min(360px,calc(100vw-16px))] overflow-hidden rounded-xl border border-border bg-elevated shadow-2xl"
        >
          <div className="flex items-center justify-between border-b px-4 py-3">
            <span className="text-sm font-semibold">Уведомления</span>
            <Popover.Close asChild>
              <Link to="/app/notifications" className="text-xs link">
                Все
              </Link>
            </Popover.Close>
          </div>
          <div className="max-h-[420px] overflow-y-auto">
            {isLoading && (
              <div className="flex justify-center p-6 text-muted">
                <Spinner />
              </div>
            )}
            {data?.length === 0 && <p className="p-6 text-center text-sm text-muted">Уведомлений пока нет</p>}
            {data?.map((n) => (
              <div key={n.id} className="border-b border-border/60 px-4 py-3 last:border-0">
                <p className="text-sm">{n.text}</p>
                <p className="mt-1 text-xs text-muted">{formatRelative(n.created_at)}</p>
              </div>
            ))}
          </div>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  );
}
