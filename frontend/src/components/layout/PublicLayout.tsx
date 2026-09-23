import { useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { Bars3Icon, XMarkIcon } from '@heroicons/react/24/outline';
import { useAuthStore } from '../../store/auth';
import { PrimaryButton } from '../ui';

const NAV_LINKS = [
  { label: 'About', href: '#about' },
  { label: 'Achievements', href: '#achievements' },
  { label: 'Events', href: '#events' },
  { label: 'Gallery', href: '#gallery' },
  { label: 'Committee', href: '#committee' },
  { label: 'Community', href: '#community' },
];

export default function PublicLayout({ children }: { children: React.ReactNode }) {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const { isAuthenticated } = useAuthStore();
  const { pathname } = useLocation();
  const isHome = pathname === '/';

  const handleNavClick = (href: string) => {
    setMobileMenuOpen(false);
    if (isHome) {
      const el = document.querySelector(href);
      if (el) {
        // Account for sticky header
        const y = el.getBoundingClientRect().top + window.scrollY - 80;
        window.scrollTo({ top: y, behavior: 'smooth' });
      }
    }
  };

  return (
    <div className="min-h-screen bg-obsidian-dark flex flex-col text-ink font-sans">
      {/* Header */}
      <header className="fixed top-0 inset-x-0 z-50 backdrop-blur-xl bg-slate-950/80 border-b border-amber-500/20">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-20">
            {/* Logo */}
            <div className="flex-shrink-0 flex items-center">
              <Link to="/" onClick={() => window.scrollTo(0, 0)} className="text-2xl font-bold text-gold-gradient tracking-tight">
                PMT Family
              </Link>
            </div>

            {/* Desktop Nav */}
            <nav className="hidden md:flex space-x-8">
              {NAV_LINKS.map((link) => (
                isHome ? (
                  <button
                    key={link.label}
                    onClick={() => handleNavClick(link.href)}
                    className="text-sm font-medium text-slate-300 hover:text-amber-500 transition-colors"
                  >
                    {link.label}
                  </button>
                ) : (
                  <Link
                    key={link.label}
                    to={`/${link.href}`}
                    className="text-sm font-medium text-slate-300 hover:text-amber-500 transition-colors"
                  >
                    {link.label}
                  </Link>
                )
              ))}
            </nav>

            {/* CTA */}
            <div className="hidden md:flex items-center space-x-4">
              <Link to="/portal">
                <PrimaryButton className="!py-2.5 !px-5 !text-sm !w-auto">
                  {isAuthenticated ? 'Go to Portal' : 'Portal Login'}
                </PrimaryButton>
              </Link>
            </div>

            {/* Mobile menu button */}
            <div className="flex md:hidden items-center">
              <button
                type="button"
                className="text-slate-300 hover:text-amber-500 transition-colors"
                onClick={() => setMobileMenuOpen(true)}
              >
                <Bars3Icon className="h-7 w-7" />
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Mobile Menu Drawer */}
      {mobileMenuOpen && (
        <div className="fixed inset-0 z-[60] flex md:hidden">
          <div className="fixed inset-0 bg-black/60 backdrop-blur-sm transition-opacity" onClick={() => setMobileMenuOpen(false)} />
          <div className="relative flex-1 flex flex-col max-w-[280px] w-full bg-charcoal-card border-l border-amber-500/20 ml-auto p-6 shadow-2xl animate-in slide-in-from-right duration-300">
            <div className="flex justify-between items-center mb-8">
              <span className="text-xl font-bold text-gold-gradient tracking-tight">Menu</span>
              <button onClick={() => setMobileMenuOpen(false)} className="text-slate-400 hover:text-white transition-colors">
                <XMarkIcon className="h-7 w-7" />
              </button>
            </div>
            <nav className="flex flex-col space-y-6">
              {NAV_LINKS.map((link) => (
                isHome ? (
                  <button
                    key={link.label}
                    onClick={() => handleNavClick(link.href)}
                    className="text-lg font-medium text-slate-300 hover:text-amber-500 text-left transition-colors"
                  >
                    {link.label}
                  </button>
                ) : (
                  <Link
                    key={link.label}
                    to={`/${link.href}`}
                    onClick={() => setMobileMenuOpen(false)}
                    className="text-lg font-medium text-slate-300 hover:text-amber-500 transition-colors"
                  >
                    {link.label}
                  </Link>
                )
              ))}
              <div className="pt-6 border-t border-amber-500/20">
                <Link to="/portal" onClick={() => setMobileMenuOpen(false)}>
                  <PrimaryButton className="w-full">
                    {isAuthenticated ? 'Go to Portal' : 'Portal Login'}
                  </PrimaryButton>
                </Link>
              </div>
            </nav>
          </div>
        </div>
      )}

      {/* Main Content */}
      <main className="flex-grow pt-20">
        {children}
      </main>

      {/* Footer */}
      <footer className="bg-[#0b0c10] border-t border-amber-500/20 py-16 px-4 sm:px-6 lg:px-8 mt-auto">
        <div className="max-w-7xl mx-auto grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-12 lg:gap-8">
          <div className="col-span-1 md:col-span-2 lg:col-span-1">
            <span className="text-2xl font-bold text-gold-gradient tracking-tight">PMT Family</span>
            <p className="mt-4 text-sm text-slate-400 leading-relaxed">
              Creating memories, building connections. The official Pure Mathematics Community at the University of Sri Jayewardenepura.
            </p>
          </div>
          <div>
            <h3 className="text-sm font-semibold text-white tracking-wider uppercase mb-5">Quick Links</h3>
            <ul className="space-y-3">
              {NAV_LINKS.slice(0, 4).map(link => (
                <li key={link.label}>
                  <a href={`/${link.href}`} className="text-sm text-slate-400 hover:text-amber-500 transition-colors">{link.label}</a>
                </li>
              ))}
            </ul>
          </div>
          <div>
            <h3 className="text-sm font-semibold text-white tracking-wider uppercase mb-5">Community</h3>
            <ul className="space-y-3">
              <li><a href="https://chat.whatsapp.com/KWvoKLIsvc9EFxZveo4qQy" target="_blank" rel="noopener noreferrer" className="text-sm text-slate-400 hover:text-amber-500 transition-colors">WhatsApp Official</a></li>
              <li><a href="https://www.facebook.com/share/1HgVyRGC1a/" target="_blank" rel="noopener noreferrer" className="text-sm text-slate-400 hover:text-amber-500 transition-colors">Facebook Page</a></li>
              <li><a href="https://www.instagram.com/pmtfamily_usj?stkn=MTJudm5vNnY1MzI5bQ%3D%3D&utm_source=qr" target="_blank" rel="noopener noreferrer" className="text-sm text-slate-400 hover:text-amber-500 transition-colors">Instagram</a></li>
              <li><a href="https://youtube.com/@pmtfamily-s2d?si=S7a2HGd3BuW2cdzP" target="_blank" rel="noopener noreferrer" className="text-sm text-slate-400 hover:text-amber-500 transition-colors">YouTube Channel</a></li>
            </ul>
          </div>
          <div>
            <h3 className="text-sm font-semibold text-white tracking-wider uppercase mb-5">Legal</h3>
            <ul className="space-y-3">
              <li><Link to="/portal" className="text-sm text-slate-400 hover:text-amber-500 transition-colors">Student & Admin Portal</Link></li>
            </ul>
          </div>
        </div>
        <div className="max-w-7xl mx-auto mt-16 pt-8 border-t border-amber-500/10 flex flex-col md:flex-row justify-between items-center">
          <p className="text-sm text-slate-500">&copy; {new Date().getFullYear()} PMT Family - University of Sri Jayewardenepura.</p>
        </div>
      </footer>
    </div>
  );
}
