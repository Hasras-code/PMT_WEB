import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { useAppStore } from '../store/app';
import { usePortalBase, useElevated } from '../hooks/useRole';
import type { Batch } from '../types';
import { Card, Badge, statusTone, Empty, coverColor, initials } from '../components/ui';
import CreateCourse from '../components/CreateCourse';

export default function Courses() {
  const navigate = useNavigate();
  const base = usePortalBase();
  const elevated = useElevated();
  const { batches, setBatches } = useAppStore();
  const [showCreate, setShowCreate] = useState(false);

  const load = () => {
    api.get('/v1/batches').then((res) => setBatches(res.data as Batch[])).catch(() => {});
  };

  useEffect(() => {
    load();
  }, []);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <p className="text-muted text-[15px]">{batches.length} course{batches.length === 1 ? '' : 's'} available</p>
        {elevated && (<button onClick={() => setShowCreate(true)} className="px-5 py-2.5 rounded-xl bg-primary text-white text-[15px] font-medium hover:bg-primary-dark">
          + Create Course
        </button>)}
      </div>
      {batches.length === 0 ? (
        <Card className="p-8">
          <Empty message="No courses yet. Ask a representative to approve your membership." />
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
          {batches.map((b) => (
            <Card key={b.id} className="overflow-hidden cursor-pointer hover:shadow-md transition-shadow" >
              <div onClick={() => navigate(`${base}/courses/${b.id}`)}>
                <div className="h-28 flex items-center justify-center" style={{ background: coverColor(b.slug) }}>
                  <span className="text-white text-3xl font-bold">{initials(b.name)}</span>
                </div>
                <div className="p-5">
                  <div className="flex items-start justify-between gap-3">
                    <h3 className="font-semibold text-ink text-[17px] leading-snug">{b.name}</h3>
                    <Badge tone={statusTone(b.status)}>{b.status}</Badge>
                  </div>
                  <p className="mt-1 text-sm text-muted">
                    {b.slug} · Class of {b.entry_year}
                    {b.graduation_year ? ` → ${b.graduation_year}` : ''}
                  </p>
                  <p className="mt-2 text-sm text-muted line-clamp-2">{b.description || 'No description.'}</p>
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}
      <CreateCourse open={showCreate} onClose={() => setShowCreate(false)} />
    </div>
  );
}
