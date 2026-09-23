import { useEffect, useState } from 'react';
import { api, toList, errMsg } from '../api/client';
import toast from 'react-hot-toast';
import type { NotificationItem } from '../types';
import { Card, Badge, Empty, timeAgo } from '../components/ui';

export default function Notifications() {
  const [items, setItems] = useState<NotificationItem[]>([]);
  const [markingAll, setMarkingAll] = useState(false);

  const load = () => api.get('/v1/me/notifications', { limit: 100 }).then((res) => setItems(toList<NotificationItem>(res.data))).catch(() => {});
  useEffect(() => {
    load();
  }, []);

  const markRead = (id: string) =>
    api.patch(`/v1/me/notifications/${id}`).then(load).catch((e) => toast.error(errMsg(e, 'Failed')));

  const readAll = () => {
    if (markingAll) return;
    setMarkingAll(true);
    api.post('/v1/me/notifications/read-all').then(() => { toast.success('All caught up'); load(); }).catch((e) => toast.error(errMsg(e, 'Failed'))).finally(() => setMarkingAll(false));
  };

  const unread = items.filter((n) => !n.read_at).length;

  return (
    <div className="max-w-3xl mx-auto space-y-5">
      <div className="flex items-center justify-between">
        <p className="text-muted text-[15px]">{unread} unread notification{unread === 1 ? '' : 's'}</p>
        {unread > 0 && (
          <button onClick={readAll} disabled={markingAll} className="text-sm text-primary font-medium hover:underline disabled:opacity-50">{markingAll ? 'Marking…' : 'Mark all as read'}</button>
        )}
      </div>
      {items.length === 0 ? (
        <Card className="p-8"><Empty message="No notifications." /></Card>
      ) : (
        items.map((n) => (
          <Card key={n.id} className={`p-5 ${n.read_at ? '' : 'ring-1 ring-primary/30'}`}>
            <div className="flex items-start justify-between gap-4">
              <div>
                <p className="font-medium text-ink">{n.title}</p>
                <p className="text-sm text-muted mt-0.5">{n.body}</p>
                <p className="text-xs text-muted mt-1.5">{n.type.replaceAll('_', ' ')} · {timeAgo(n.created_at)}</p>
              </div>
              {!n.read_at ? (
                <button onClick={() => markRead(n.id)} className="shrink-0"><Badge tone="blue">Unread</Badge></button>
              ) : (
                <Badge tone="gray">Read</Badge>
              )}
            </div>
          </Card>
        ))
      )}
    </div>
  );
}
