import { Component } from 'react';
import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
}

/** Catches render crashes per outlet so one broken page can't blank the app. */
export default class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false };

  static getDerivedStateFromError(): State {
    return { hasError: true };
  }

  componentDidCatch(): void {
    // Intentionally silent: no error-tracking backend is wired yet.
  }

  render(): ReactNode {
    if (!this.state.hasError) return this.props.children;
    return (
      <div className="min-h-[50vh] flex items-center justify-center p-8" role="alert">
        <div className="w-full max-w-md text-center rounded-2xl border border-line bg-charcoal-card p-8">
          <h1 className="text-xl font-semibold text-ink">Something went wrong</h1>
          <p className="mt-2 text-sm text-muted">This section hit an unexpected error. Your session is intact.</p>
          <div className="mt-6 flex gap-3">
            <button
              type="button"
              onClick={() => window.location.reload()}
              className="flex-1 px-5 py-2.5 rounded-xl bg-primary text-slate-950 text-sm font-bold hover:opacity-90"
            >
              Reload
            </button>
            <Link
              to="/"
              className="flex-1 px-5 py-2.5 rounded-xl border border-line text-sm font-medium text-ink hover:bg-surface text-center"
            >
              Go home
            </Link>
          </div>
        </div>
      </div>
    );
  }
}
