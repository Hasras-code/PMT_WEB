import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { api, errMsg } from '../api/client';
import { useAuthStore } from '../store/auth';
import toast from 'react-hot-toast';
import { Field, inputCls, PrimaryButton } from '../components/ui';

export default function Login() {
  const navigate = useNavigate();
  const { setUser } = useAuthStore();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

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
      navigate('/', { replace: true });
    } catch (err) {
      const msg = errMsg(err, 'Login failed');
      setError(msg);
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex bg-surface">
      <div className="hidden lg:flex w-[42%] bg-primary flex-col justify-between p-12 text-white">
        <p className="text-3xl font-semibold leading-snug">
          Learning
          <br />
          Management
          <br />
          System
        </p>
        <div>
          <h2 className="text-4xl font-bold leading-tight">Monitor and manage your learning platform.</h2>
          <p className="mt-4 text-white/80 text-lg">Courses, users, meetings and content — in one place.</p>
        </div>
        <p className="text-white/60 text-sm">PMT University LMS</p>
      </div>
      <div className="flex-1 flex items-center justify-center p-8">
        <div className="w-full max-w-md bg-white rounded-2xl border border-line p-8">
          <h1 className="text-2xl font-semibold text-ink">Sign in</h1>
          <p className="mt-1 text-sm text-muted">Welcome back to your learning platform.</p>
          <form onSubmit={handleSubmit} className="mt-6 space-y-4">
            {error && (
              <p className="px-4 py-3 rounded-xl bg-red-50 text-red-700 text-sm font-medium">{error}</p>
            )}
            <Field label="Email">
              <input type="email" required placeholder="you@example.com" className={inputCls} value={email} onChange={(e) => setEmail(e.target.value)} />
            </Field>
            <Field label="Password">
              <input type="password" required placeholder="••••••••••••" className={inputCls} value={password} onChange={(e) => setPassword(e.target.value)} />
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
