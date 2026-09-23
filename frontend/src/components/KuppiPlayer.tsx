import { useState } from 'react';
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline';

interface KuppiPlayerProps {
  embedUrl: string;
  title: string;
}

export default function KuppiPlayer({ embedUrl, title }: KuppiPlayerProps) {
  const [failed, setFailed] = useState(false);

  if (failed) {
    return (
      <div className="flex aspect-video items-center justify-center rounded-xl bg-slate-950/70 px-6 text-center text-sm text-muted">
        <span className="flex items-center gap-2">
          <ExclamationTriangleIcon className="h-5 w-5" aria-hidden="true" />
          This video could not be embedded.
        </span>
      </div>
    );
  }

  return (
    <div className="aspect-video w-full overflow-hidden rounded-xl bg-slate-950">
      <iframe
        className="h-full w-full"
        src={embedUrl}
        title={title}
        loading="lazy"
        allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
        referrerPolicy="strict-origin-when-cross-origin"
        allowFullScreen
        onError={() => setFailed(true)}
      />
    </div>
  );
}
