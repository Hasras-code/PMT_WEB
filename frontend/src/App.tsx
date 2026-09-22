import { Suspense, lazy, useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate, useLocation } from 'react-router-dom';
import { Toaster } from 'react-hot-toast';
import { useAuthStore } from './store/auth';
import { useElevated } from './hooks/useRole';
import { lazy, Suspense, type JSX } from 'react';
import Layout from './components/Layout';
import StudentLayout from './components/StudentLayout';
import ErrorBoundary from './components/ErrorBoundary';

const Login = lazy(() => import('./pages/Login'));
const Register = lazy(() => import('./pages/Register'));
const Verify = lazy(() => import('./pages/Verify'));
const PasswordReset = lazy(() => import('./pages/PasswordReset'));
const Dashboard = lazy(() => import('./pages/Dashboard'));
const StudentDashboard = lazy(() => import('./pages/StudentDashboard'));
const Users = lazy(() => import('./pages/Users'));
const Courses = lazy(() => import('./pages/Courses'));
const CourseDetail = lazy(() => import('./pages/CourseDetail'));
const Assessments = lazy(() => import('./pages/Assessments'));
const Communication = lazy(() => import('./pages/Communication'));
const Meetings = lazy(() => import('./pages/Meetings'));
const Monitoring = lazy(() => import('./pages/Monitoring'));
const System = lazy(() => import('./pages/System'));
const Administration = lazy(() => import('./pages/Administration'));
const Gallery = lazy(() => import('./pages/Gallery'));
const PublicBatch = lazy(() => import('./pages/PublicBatch'));
const Profile = lazy(() => import('./pages/Profile'));
const Notifications = lazy(() => import('./pages/Notifications'));
const NotFound = lazy(() => import('./pages/NotFound'));

function PageFallback() {
  return (
    <div className="min-h-[50vh] flex items-center justify-center" role="status" aria-live="polite">
      <div className="w-10 h-10 rounded-full border-2 border-line border-t-primary animate-spin" aria-hidden="true" />
      <span className="sr-only">Loading page…</span>
    </div>
  );
}

function ScrollToTop() {
  const { pathname } = useLocation();
  useEffect(() => {
    window.scrollTo(0, 0);
  }, [pathname]);
  return null;
}

const TITLES: Array<[RegExp, string]> = [
  [/^\/login$/, 'Sign in'],
  [/^\/register$/, 'Create account'],
  [/^\/verify$/, 'Verify email'],
  [/^\/password$/, 'Reset password'],
  [/admin\/dashboard$/, 'Admin dashboard'],
  [/student\/dashboard$/, 'Student dashboard'],
  [/\/users$/, 'Users'],
  [/\/courses\/[^/]+$/, 'Course detail'],
  [/\/courses$/, 'Courses'],
  [/\/assessments$/, 'Assessments'],
  [/\/communication$/, 'Communication'],
  [/\/meetings$/, 'Meetings'],
  [/\/monitoring$/, 'Monitoring'],
  [/\/system$/, 'System'],
  [/\/administration$/, 'Administration'],
  [/\/gallery$/, 'Gallery'],
  [/\/public\/batches\//, 'Public batch'],
  [/\/profile$/, 'Profile'],
  [/\/notifications$/, 'Notifications'],
];

function DocumentTitle() {
  const { pathname } = useLocation();
  useEffect(() => {
    const match = TITLES.find(([re]) => re.test(pathname));
    document.title = match ? `${match[1]} · PMT LMS` : 'PMT LMS';
  }, [pathname]);
  return null;
}

const Funds = lazy(() => import('./pages/Funds'));

function FundsPage() {
  return <Suspense fallback={<p className="p-8 text-sm text-muted">Loading funds…</p>}><Funds /></Suspense>;
}

function RequireAuth({ children }: { children: JSX.Element }) {
  const { isAuthenticated } = useAuthStore();
  const location = useLocation();
  if (!isAuthenticated) return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  return children;
}

/** Keeps signed-in users out of auth pages (they land in their portal). */
function GuestOnly({ children }: { children: JSX.Element }) {
  const { isAuthenticated } = useAuthStore();
  if (isAuthenticated) return <Navigate to="/" replace />;
  return children;
}

function RequireAdmin({ children }: { children: JSX.Element }) {
  const elevated = useElevated();
  if (elevated === null)
    return (
      <div className="p-8" role="status" aria-live="polite">
        <div className="w-10 h-10 rounded-full border-2 border-line border-t-primary animate-spin" aria-hidden="true" />
        <span className="sr-only">Checking access…</span>
      </div>
    );
  if (!elevated) return <Navigate to="/student/dashboard" replace />;
  return children;
}

/** Sends signed-in users to their portal, guests to login. */
function Landing() {
  const { isAuthenticated } = useAuthStore();
  const elevated = useElevated();
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  if (elevated === null)
    return (
      <div className="p-8" role="status" aria-live="polite">
        <div className="w-10 h-10 rounded-full border-2 border-line border-t-primary animate-spin" aria-hidden="true" />
        <span className="sr-only">Loading…</span>
      </div>
    );
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
      <Toaster position="top-right" toastOptions={{ duration: 4000 }} />
      <ScrollToTop />
      <DocumentTitle />
      <ErrorBoundary>
        <Suspense fallback={<PageFallback />}>
          <Routes>
            <Route
              path="/login"
              element={
                <GuestOnly>
                  <Login />
                </GuestOnly>
              }
            />
            <Route
              path="/register"
              element={
                <GuestOnly>
                  <Register />
                </GuestOnly>
              }
            />
            <Route
              path="/verify"
              element={
                <GuestOnly>
                  <Verify />
                </GuestOnly>
              }
            />
            <Route
              path="/password"
              element={
                <GuestOnly>
                  <PasswordReset />
                </GuestOnly>
              }
            />
            <Route path="/" element={<Landing />} />

            {/* Public batch profiles need no login. */}
            <Route path="/public/batches/:slug" element={<PublicBatch />} />

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
              <Route path="profile" element={<Profile />} />
              <Route path="notifications" element={<Notifications />} />
              <Route path="funds" element={<FundsPage />} />
            </Route>

            {LEGACY.map((p) => (
              <Route key={p} path={`/${p}`} element={<Navigate to="/" replace />} />
            ))}
            <Route path="*" element={<NotFound />} />
          </Routes>
        </Suspense>
      </ErrorBoundary>
    </BrowserRouter>
  );
}

export default App;
