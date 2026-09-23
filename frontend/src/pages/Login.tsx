import { useState, useEffect } from 'react';
import { useLocation, useNavigate, Link } from 'react-router-dom';
import { api, errMsg } from '../api/client';
import { useAuthStore } from '../store/auth';
import toast from 'react-hot-toast';
import { Field, inputCls, PrimaryButton } from '../components/ui';

export default function Login() {
  const navigate = useNavigate();
  const location = useLocation();
  const { setUser } = useAuthStore();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    document.title = 'Sign In - PMT Family Portal';
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');
    // Drop any stale session so a dead token can never block a fresh sign-in.
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    try {
      const res = await api.post('/v1/auth/login', { email, password });
      localStorage.setItem('access_token', res.data.access_token);
      if (res.data.refresh_token) localStorage.setItem('refresh_token', res.data.refresh_token);
      const me = await api.get('/v1/me');
      setUser(me.data);
      toast.success('Welcome back!');
      const requestedPath = (location.state as { from?: string } | null)?.from;
      const from = requestedPath?.startsWith('/') && !requestedPath.startsWith('//') ? requestedPath : '/portal';
      navigate(from, { replace: true });
    } catch (err) {
      const msg = errMsg(err, 'Login failed');
      setError(msg);
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex bg-obsidian-dark">
      <div className="hidden lg:flex w-[42%] bg-charcoal-card border-r border-line flex-col justify-between p-12 relative overflow-hidden">
        <div className="absolute top-0 left-0 w-full h-full bg-gradient-to-br from-amber-500/10 to-transparent pointer-events-none" />
        <p className="text-[40px] tracking-tight leading-tight font-bold text-gold-gradient relative z-10">
          PMT Family
        </p>
        <div className="relative z-10">
          <h2 className="text-4xl font-bold leading-tight text-ink">Welcome to the PMT Batch Trip Portal.</h2>
          <p className="mt-4 text-muted text-lg">Manage your memories, journeys, and community — all in one place.</p>
        </div>
        <p className="text-muted text-sm relative z-10">PMT Batch Trip</p>
      </div>
      <div className="flex-1 flex items-center justify-center p-8 relative">
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-96 h-96 bg-amber-500/10 blur-[100px] pointer-events-none rounded-full" />
        <div className="w-full max-w-md bg-charcoal-card backdrop-blur-xl rounded-2xl border border-line p-8 shadow-2xl relative z-10">
          <h1 className="text-2xl font-semibold text-ink">Sign in</h1>
          <p className="mt-1 text-sm text-muted">Welcome back to your learning platform.</p>
          <form onSubmit={handleSubmit} className="mt-6 space-y-4">
            {error && (
              <p className="px-4 py-3 rounded-xl bg-red-50 text-red-700 text-sm font-medium">{error}</p>
            )}
            <Field label="Email">
              <input type="email" required autoComplete="email" placeholder="you@example.com" className={inputCls} value={email} onChange={(e) => setEmail(e.target.value)} />
            </Field>
            <Field label="Password">
              <input type="password" required autoComplete="current-password" placeholder="••••••••••••" className={inputCls} value={password} onChange={(e) => setPassword(e.target.value)} />
            </Field>
            <PrimaryButton type="submit" disabled={loading}>
              {loading ? 'Signing in…' : 'Sign in'}
            </PrimaryButton>
          </form>
          <div className="mt-5 flex items-center justify-between text-sm">
            <Link to="/register" className="text-primary font-medium hover:underline">
              Create account
            </Link>
            <span className="flex gap-4">
              <Link to="/verify" className="text-muted hover:text-primary">
                Verify email
              </Link>
              <Link to="/password" className="text-muted hover:text-primary">
                Forgot password
              </Link>
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}
