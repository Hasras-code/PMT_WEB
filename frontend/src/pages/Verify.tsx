import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { api, errMsg } from '../api/client';
import toast from 'react-hot-toast';
import { Field, inputCls, PrimaryButton } from '../components/ui';

export default function Verify() {
  const navigate = useNavigate();
  const [token, setToken] = useState('');
  const [email, setEmail] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await api.post('/v1/auth/verify-email', { token });
      toast.success('Email verified! You can sign in now.');
      navigate('/login');
    } catch (err) {
      toast.error(errMsg(err, 'Verification failed'));
    } finally {
      setLoading(false);
    }
  };

  const resend = async () => {
    if (!email) return;
    setLoading(true);
    try {
      await api.post('/v1/auth/resend-verification', { email });
      toast.success('If eligible, a new verification email was sent.');
    } catch (err) {
      toast.error(errMsg(err, 'Could not request another email'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-obsidian-dark p-8 relative">
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-96 h-96 bg-amber-500/10 blur-[100px] pointer-events-none rounded-full" />
      <div className="w-full max-w-md bg-charcoal-card backdrop-blur-xl rounded-2xl border border-line p-8 shadow-2xl relative z-10">
        <p className="text-[28px] tracking-tight font-bold text-gold-gradient">PMT Family</p>
        <h1 className="mt-3 text-2xl font-semibold text-ink">Verify your email</h1>
        <p className="mt-1 text-sm text-muted">Paste the one-time token from your email.</p>
        <form onSubmit={handleSubmit} className="mt-6 space-y-4">
          <Field label="Verification token">
            <input required placeholder="Token from email" className={inputCls} value={token} onChange={(e) => setToken(e.target.value)} />
          </Field>
          <PrimaryButton type="submit" disabled={loading}>
            {loading ? 'Verifying…' : 'Verify email'}
          </PrimaryButton>
        </form>
        <div className="mt-6 pt-6 border-t border-line space-y-3">
          <Field label="Need another email?">
            <input type="email" placeholder="Your registration email" className={inputCls} value={email} onChange={(e) => setEmail(e.target.value)} />
          </Field>
          <button type="button" disabled={loading || !email} onClick={resend} className="text-sm text-primary font-medium hover:underline disabled:opacity-50">
            Resend verification email
          </button>
        </div>
        <p className="mt-5 text-sm text-muted">
          <Link to="/login" className="text-primary font-medium hover:underline">
            Back to sign in
          </Link>
        </p>
      </div>
    </div>
  );
}
