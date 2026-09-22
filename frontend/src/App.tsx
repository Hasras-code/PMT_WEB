import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { Toaster } from 'react-hot-toast';
import { useAuthStore } from './store/auth';
import { useElevated } from './hooks/useRole';
import { lazy, Suspense, type JSX } from 'react';
import Layout from './components/Layout';
import StudentLayout from './components/StudentLayout';
import Login from './pages/Login';
import Register from './pages/Register';
import Verify from './pages/Verify';
import PasswordReset from './pages/PasswordReset';
import Dashboard from './pages/Dashboard';
import StudentDashboard from './pages/StudentDashboard';
import Users from './pages/Users';
import Courses from './pages/Courses';
import CourseDetail from './pages/CourseDetail';
import Assessments from './pages/Assessments';
import Communication from './pages/Communication';
import Meetings from './pages/Meetings';
import Monitoring from './pages/Monitoring';
import System from './pages/System';
import Administration from './pages/Administration';
import Gallery from './pages/Gallery';
import PublicBatch from './pages/PublicBatch';
import Profile from './pages/Profile';
import Notifications from './pages/Notifications';

const Funds = lazy(() => import('./pages/Funds'));

function FundsPage() {
  return <Suspense fallback={<p className="p-8 text-sm text-muted">Loading funds…</p>}><Funds /></Suspense>;
}

function RequireAuth({ children }: { children: JSX.Element }) {
  const { isAuthenticated } = useAuthStore();
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  return children;
}

function RequireAdmin({ children }: { children: JSX.Element }) {
  const elevated = useElevated();
  if (elevated === null) return <p className="p-8 text-sm text-muted">Loading…</p>;
  if (!elevated) return <Navigate to="/student/dashboard" replace />;
  return children;
}

/** Sends signed-in users to their portal, guests to login. */
function Landing() {
  const { isAuthenticated } = useAuthStore();
  const elevated = useElevated();
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  if (elevated === null) return <p className="p-8 text-sm text-muted">Loading…</p>;
  return <Navigate to={elevated ? '/admin/dashboard' : '/student/dashboard'} replace />;
}

const LEGACY = [
  'dashboard',
  'users',
  'courses',
  'assessments',
  'communication',
  'meetings',
  'monitoring',
  'system',
  'administration',
  'gallery',
  'profile',
  'notifications',
  'support',
  'funds',
];

function App() {
  return (
    <BrowserRouter>
      <Toaster position="top-right" />
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route path="/verify" element={<Verify />} />
        <Route path="/password" element={<PasswordReset />} />
        <Route path="/" element={<Landing />} />

        <Route
          path="/admin"
          element={
            <RequireAuth>
              <RequireAdmin>
                <Layout />
              </RequireAdmin>
            </RequireAuth>
          }
        >
          <Route index element={<Navigate to="/admin/dashboard" replace />} />
          <Route path="dashboard" element={<Dashboard />} />
          <Route path="users" element={<Users />} />
          <Route path="courses" element={<Courses />} />
          <Route path="courses/:batchID" element={<CourseDetail />} />
          <Route path="assessments" element={<Assessments />} />
          <Route path="communication" element={<Communication />} />
          <Route path="meetings" element={<Meetings />} />
          <Route path="monitoring" element={<Monitoring />} />
          <Route path="system" element={<System />} />
          <Route path="administration" element={<Administration />} />
          <Route path="gallery" element={<Gallery />} />
          <Route path="profile" element={<Profile />} />
          <Route path="notifications" element={<Notifications />} />
          <Route path="funds" element={<FundsPage />} />
        </Route>

        <Route
          path="/student"
          element={
            <RequireAuth>
              <StudentLayout />
            </RequireAuth>
          }
        >
          <Route index element={<Navigate to="/student/dashboard" replace />} />
          <Route path="dashboard" element={<StudentDashboard />} />
          <Route path="courses" element={<Courses />} />
          <Route path="courses/:batchID" element={<CourseDetail />} />
          <Route path="assessments" element={<Assessments />} />
          <Route path="communication" element={<Communication />} />
          <Route path="meetings" element={<Meetings />} />
          <Route path="gallery" element={<Gallery />} />
          <Route path="public/batches/:slug" element={<PublicBatch />} />
          <Route path="profile" element={<Profile />} />
          <Route path="notifications" element={<Notifications />} />
          <Route path="funds" element={<FundsPage />} />
        </Route>

        {LEGACY.map((p) => (
          <Route key={p} path={`/${p}`} element={<Navigate to="/" replace />} />
        ))}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
