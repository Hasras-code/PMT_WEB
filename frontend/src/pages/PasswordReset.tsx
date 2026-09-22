import { useState } from 'react';
import { Link } from 'react-router-dom';
import { api, errMsg } from '../api/client';
import toast from 'react-hot-toast';
import { Field, inputCls, PrimaryButton } from '../components/ui';

export default function PasswordReset() {
  const [email, setEmail] = useState('');
  const [token, setToken] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);

  const request = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await api.post('/v1/auth/forgot-password', { email });
      toast.success('If eligible, a reset email was sent.');
    } catch (err) {
      toast.error(errMsg(err, 'Request failed'));
    } finally {
      setLoading(false);
    }
  };

  const reset = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await api.post('/v1/auth/reset-password', { token, password });
      toast.success('Password changed! Sign in with your new password.');
    } catch (err) {
      toast.error(errMsg(err, 'Reset failed'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-obsidian-dark p-8 relative">
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-96 h-96 bg-amber-500/10 blur-[100px] pointer-events-none rounded-full" />
      <div className="w-full max-w-md bg-charcoal-card backdrop-blur-xl rounded-2xl border border-line p-8 space-y-8 shadow-2xl relative z-10">
        <div>
          <p className="text-[28px] tracking-tight font-bold text-gold-gradient">PMT Family</p>
          <h1 className="mt-3 text-2xl font-semibold text-ink">Reset password</h1>
        </div>
        <form onSubmit={request} className="space-y-4">
          <h2 className="text-sm font-semibold text-ink">Step 1 — request a reset email</h2>
          <Field label="Email">
            <input type="email" required autoComplete="email" className={inputCls} value={email} onChange={(e) => setEmail(e.target.value)} />
          </Field>
          <PrimaryButton type="submit" disabled={loading}>
            Send reset email
          </PrimaryButton>
        </form>
        <form onSubmit={reset} className="space-y-4">
          <h2 className="text-sm font-semibold text-ink">Step 2 — set a new password</h2>
          <Field label="Reset token">
            <input required autoComplete="one-time-code" className={inputCls} value={token} onChange={(e) => setToken(e.target.value)} />
          </Field>
          <Field label="New password (min 12 characters)">
            <input type="password" required minLength={12} autoComplete="new-password" className={inputCls} value={password} onChange={(e) => setPassword(e.target.value)} />
          </Field>
          <PrimaryButton type="submit" disabled={loading}>
            Change password
          </PrimaryButton>
        </form>
        <p className="text-sm text-muted">
          <Link to="/login" className="text-primary font-medium hover:underline">
            Back to sign in
          </Link>
        </p>
      </div>
    </div>
  );
}
