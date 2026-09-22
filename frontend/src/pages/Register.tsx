import { useEffect, useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { api, errMsg, toList } from '../api/client';
import type { Batch } from '../types';
import toast from 'react-hot-toast';
import { Field, inputCls, PrimaryButton } from '../components/ui';

export default function Register() {
  const navigate = useNavigate();
  const [form, setForm] = useState({
    student_number: '',
    combination: 'PMT-ICT',
    batch_id: '',
    first_name: '',
    last_name: '',
    display_name: '',
    email: '',
    password: '',
  });
  const [loading, setLoading] = useState(false);
  const [cohorts, setCohorts] = useState<Batch[]>([]);
  const [cohortsLoading, setCohortsLoading] = useState(true);

  useEffect(() => {
    api.get('/v1/public/batches')
      .then((response) => setCohorts(toList<Batch>(response.data)))
      .catch((err) => toast.error(errMsg(err, 'Could not load cohorts')))
      .finally(() => setCohortsLoading(false));
  }, []);

  const set = (k: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm({ ...form, [k]: e.target.value });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await api.post('/v1/auth/register', form);
      toast.success('Account created! Check your email to verify.');
      navigate('/verify');
    } catch (err) {
      toast.error(errMsg(err, 'Registration failed'));
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
          <h2 className="text-4xl font-bold leading-tight">Join your cohort.</h2>
          <p className="mt-4 text-white/80 text-lg">Register with your student number to get started.</p>
        </div>
        <p className="text-white/60 text-sm">PMT University LMS</p>
      </div>
      <div className="flex-1 flex items-center justify-center p-8">
        <div className="w-full max-w-lg bg-white rounded-2xl border border-line p-8">
          <h1 className="text-2xl font-semibold text-ink">Create account</h1>
          <p className="mt-1 text-sm text-muted">Takes less than a minute.</p>
          <form onSubmit={handleSubmit} className="mt-6 grid grid-cols-2 gap-4">
            <Field label="Student number">
              <input required placeholder="AS2025..." className={inputCls} value={form.student_number} onChange={set('student_number')} />
            </Field>
            <Field label="Display name">
              <input required placeholder="John Doe" className={inputCls} value={form.display_name} onChange={set('display_name')} />
            </Field>
            <div className="col-span-2">
              <Field label="Combination">
                <select className={inputCls} value={form.combination} onChange={(e) => setForm({ ...form, combination: e.target.value })}>
                  <option value="PMT-ICT">PMT-ICT</option>
                  <option value="PMT-CS">PMT-CS</option>
                </select>
              </Field>
            </div>
            <div className="col-span-2">
              <Field label="Cohort">
                <select required disabled={cohortsLoading || cohorts.length === 0} className={inputCls} value={form.batch_id} onChange={(e) => setForm({ ...form, batch_id: e.target.value })}>
                  <option value="">{cohortsLoading ? 'Loading cohorts…' : 'Select your cohort…'}</option>
                  {cohorts.map((cohort) => (
                    <option key={cohort.id} value={cohort.id}>
                      {cohort.name} ({cohort.entry_year})
                    </option>
                  ))}
                </select>
              </Field>
              {!cohortsLoading && cohorts.length === 0 && <p className="mt-2 text-sm text-red-600">No active cohort is available for registration.</p>}
            </div>
            <Field label="First name">
              <input required className={inputCls} value={form.first_name} onChange={set('first_name')} />
            </Field>
            <Field label="Last name">
              <input required className={inputCls} value={form.last_name} onChange={set('last_name')} />
            </Field>
            <div className="col-span-2">
              <Field label="Email">
                <input type="email" required placeholder="you@example.com" className={inputCls} value={form.email} onChange={set('email')} />
              </Field>
            </div>
            <div className="col-span-2">
              <Field label="Password (min 12 characters)">
                <input type="password" required minLength={12} className={inputCls} value={form.password} onChange={set('password')} />
              </Field>
            </div>
            <div className="col-span-2">
              <PrimaryButton type="submit" disabled={loading || cohortsLoading || cohorts.length === 0}>
                {loading ? 'Creating…' : 'Create account'}
              </PrimaryButton>
            </div>
          </form>
          <p className="mt-5 text-sm text-muted">
            Already have an account?{' '}
            <Link to="/login" className="text-primary font-medium hover:underline">
              Sign in
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
}
