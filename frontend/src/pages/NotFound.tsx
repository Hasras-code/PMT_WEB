import { Link } from 'react-router-dom';

export default function NotFound() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-obsidian-dark p-8">
      <div className="w-full max-w-md text-center rounded-2xl border border-line bg-charcoal-card p-8 shadow-2xl">
        <p className="text-[40px] tracking-tight leading-tight font-bold text-gold-gradient">404</p>
        <h1 className="mt-2 text-xl font-semibold text-ink">Page not found</h1>
        <p className="mt-2 text-sm text-muted">The page you asked for doesn&apos;t exist or was moved.</p>
        <Link
          to="/"
          className="mt-6 inline-block w-full px-5 py-3 rounded-xl bg-gradient-to-r from-amber-500 via-yellow-500 to-amber-600 text-slate-950 text-[15px] font-bold"
        >
          Back to home
        </Link>
      </div>
    </div>
  );
}
