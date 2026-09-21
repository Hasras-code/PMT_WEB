import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { api, toList, errMsg } from '../api/client';
import toast from 'react-hot-toast';
import type { PublicBatch } from '../types';
import { Card, Badge, Empty, fmtDateTime } from '../components/ui';

interface PublicEvent {
  id: string;
  title: string;
  description: string;
  location: string | null;
  starts_at: string;
  ends_at: string | null;
}

export default function PublicBatch() {
  const { slug = '' } = useParams();
  const [batch, setBatch] = useState<PublicBatch | null>(null);
  const [events, setEvents] = useState<PublicEvent[]>([]);
  const [positions, setPositions] = useState<{ title: string; person_name?: string }[]>([]);

  useEffect(() => {
    api.get(`/v1/public/batches/${slug}`).then((res) => setBatch(res.data)).catch(() => toast.error(errMsg({}, 'Cohort not found')));
    api.get(`/v1/public/batches/${slug}/events`).then((res) => setEvents(toList<PublicEvent>(res.data))).catch(() => {});
    api.get(`/v1/public/batches/${slug}/positions`).then((res) => setPositions(toList(res.data))).catch(() => {});
  }, [slug]);

  if (!batch) return <p className="text-sm text-muted">Loading…</p>;

  return (
    <div className="space-y-6 max-w-4xl mx-auto">
      <div className="bg-primary rounded-2xl px-8 py-9">
        <h1 className="text-[26px] font-bold text-white">{batch.name}</h1>
        <p className="mt-2 text-white/85 text-[16px]">{batch.description}</p>
      </div>
      <Card className="p-7">
        <h2 className="text-lg font-semibold text-ink">Public meetings</h2>
        {events.length === 0 ? <Empty message="No public meetings." /> : (
          <div className="mt-3 divide-y divide-line">
            {events.map((e) => (
              <div key={e.id} className="py-3">
                <p className="font-medium text-ink">{e.title}</p>
                <p className="text-sm text-muted">{fmtDateTime(e.starts_at)}{e.location ? ` · ${e.location}` : ''}</p>
              </div>
            ))}
          </div>
        )}
      </Card>
      <Card className="p-7">
        <h2 className="text-lg font-semibold text-ink">Committee</h2>
        {positions.length === 0 ? <Empty message="No public positions." /> : (
          <div className="mt-3 space-y-2">
            {positions.map((p, i) => (
              <div key={i} className="flex items-center justify-between py-2 border-b border-line last:border-0">
                <p className="font-medium text-ink">{p.title}</p>
                <Badge tone="blue">{p.person_name || 'Vacant'}</Badge>
              </div>
            ))}
          </div>
        )}
      </Card>
    </div>
  );
}
