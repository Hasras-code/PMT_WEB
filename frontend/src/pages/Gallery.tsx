import { useCallback, useEffect, useState } from 'react';
import { api, errMsg } from '../api/client';
import toast from 'react-hot-toast';
import type { GalleryImage } from '../types';
import { Card, Empty } from '../components/ui';

function monthOptions(): string[] {
  const out: string[] = [];
  const d = new Date();
  for (let i = 0; i < 12; i++) {
    out.push(d.toISOString().slice(0, 7));
    d.setMonth(d.getMonth() - 1);
  }
  return out;
}

export default function Gallery() {
  const [images, setImages] = useState<GalleryImage[]>([]);
  const [cursor, setCursor] = useState<string>('');
  const [month, setMonth] = useState(new Date().toISOString().slice(0, 7));
  const [loading, setLoading] = useState(true);

  const load = useCallback(async (reset: boolean, cur?: string) => {
    setLoading(true);
    try {
      const params: Record<string, unknown> = { month, limit: 12 };
      if (cur) params.cursor = cur;
      const res = await api.get('/v1/public/gallery', params);
      const data: GalleryImage[] = Array.isArray(res.data?.data) ? res.data.data : [];
      setImages((current) => reset ? data : [...current, ...data]);
      setCursor(res.data?.next_cursor || '');
    } catch (err) {
      toast.error(errMsg(err, 'Could not load gallery'));
    } finally {
      setLoading(false);
    }
  }, [month]);

  useEffect(() => {
    load(true);
  }, [load]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <p className="text-muted text-[15px]">Published moments from across the platform</p>
        <select value={month} onChange={(e) => setMonth(e.target.value)} className="px-4 py-2.5 rounded-xl border border-line bg-white text-[15px] text-ink">
          {monthOptions().map((m) => <option key={m} value={m}>{m}</option>)}
        </select>
      </div>
      {loading && images.length === 0 ? (
        <p className="text-sm text-muted">Loading…</p>
      ) : images.length === 0 ? (
        <Card className="p-8"><Empty message="No published images this month." /></Card>
      ) : (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
            {images.map((g) => (
              <Card key={g.id} className="overflow-hidden">
                <img src={g.thumbnail_url || g.display_url} alt={g.alt_text} className="w-full h-52 object-cover" loading="lazy" />
                <div className="p-4">
                  <p className="font-medium text-ink">{g.title || g.alt_text}</p>
                  {g.caption && <p className="text-sm text-muted mt-0.5 line-clamp-2">{g.caption}</p>}
                </div>
              </Card>
            ))}
          </div>
          {cursor && (
            <div className="flex justify-center">
              <button onClick={() => load(false, cursor)} className="px-6 py-2.5 rounded-xl border border-line bg-white text-[15px] font-medium text-ink hover:bg-surface">
                Load more
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
