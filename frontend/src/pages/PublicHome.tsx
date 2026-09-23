import { useState, useEffect } from 'react';
import { motion, AnimatePresence, useMotionValue, useTransform, animate } from 'framer-motion';
import { 
  ArrowRightIcon, 
  CalendarIcon, 
  UserGroupIcon, 
  LightBulbIcon, 
  AcademicCapIcon, 
  TrophyIcon,
  XMarkIcon
} from '@heroicons/react/24/outline';
import { Skeleton, fmtDate } from '../components/ui';

interface HeroSlide { id: string; title: string; image_url: string; }
interface Achievement { id: string; title: string; count: number; icon: string; }
interface EventData { id: string; title: string; date: string; image_url: string; status: 'Upcoming' | 'Past'; url?: string; }
interface GalleryImage { id: string; event_name: string; image_url: string; }
interface Rep { id: string; name: string; role: string; batch: string; image_url: string; linkedin?: string; }

const API_URL = import.meta.env.VITE_BACKEND_API_URL || 'http://localhost:8080';

// Animated Counter Component
function AnimatedCounter({ value }: { value: number }) {
  const count = useMotionValue(0);
  const rounded = useTransform(count, Math.round);

  useEffect(() => {
    const animation = animate(count, value, { duration: 2, ease: "easeOut" });
    return animation.stop;
  }, [value, count]);

  return <motion.span>{rounded}</motion.span>;
}

export default function PublicHome() {
  const [heroSlides, setHeroSlides] = useState<HeroSlide[]>([]);
  const [achievements, setAchievements] = useState<Achievement[]>([]);
  const [events, setEvents] = useState<EventData[]>([]);
  const [gallery, setGallery] = useState<GalleryImage[]>([]);
  const [reps, setReps] = useState<Rep[]>([]);
  const [loading, setLoading] = useState(true);
  const [eventTab, setEventTab] = useState<'Upcoming' | 'Past'>('Upcoming');
  const [selectedImage, setSelectedImage] = useState<GalleryImage | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        const [slidesRes, achRes, evRes, galRes, repsRes] = await Promise.allSettled([
          fetch(`${API_URL}/api/v1/public/hero-slides`),
          fetch(`${API_URL}/api/v1/public/achievements`),
          fetch(`${API_URL}/api/v1/public/events?limit=6`),
          fetch(`${API_URL}/api/v1/public/gallery`),
          fetch(`${API_URL}/api/v1/public/featured-reps`)
        ]);

        if (slidesRes.status === 'fulfilled' && slidesRes.value.ok) setHeroSlides(await slidesRes.value.json());
        if (achRes.status === 'fulfilled' && achRes.value.ok) setAchievements(await achRes.value.json());
        if (evRes.status === 'fulfilled' && evRes.value.ok) setEvents(await evRes.value.json());
        if (galRes.status === 'fulfilled' && galRes.value.ok) setGallery(await galRes.value.json());
        if (repsRes.status === 'fulfilled' && repsRes.value.ok) setReps(await repsRes.value.json());
      } catch (err) {
        console.error("Failed to fetch public data:", err);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  const safeHeroSlides = heroSlides.length ? heroSlides : [
    { id: '1', title: 'Empowering the next generation of Mathematicians', image_url: 'https://images.unsplash.com/photo-1509228468518-180dd4864904?auto=format&fit=crop&q=80' }
  ];
  
  const filteredEvents = events.filter(e => e.status === eventTab);

  return (
    <div className="bg-obsidian-dark text-ink w-full overflow-hidden">
      {/* Lightbox Modal */}
      <AnimatePresence>
        {selectedImage && (
          <motion.div 
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-[100] flex items-center justify-center bg-black/90 p-4 backdrop-blur-sm"
            onClick={() => setSelectedImage(null)}
          >
            <button 
              className="absolute top-6 right-6 text-white/70 hover:text-white transition-colors"
              onClick={() => setSelectedImage(null)}
            >
              <XMarkIcon className="w-8 h-8" />
            </button>
            <motion.img 
              initial={{ scale: 0.9, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.9, opacity: 0 }}
              transition={{ type: "spring", damping: 25, stiffness: 300 }}
              src={selectedImage.image_url} 
              alt={selectedImage.event_name} 
              className="max-w-full max-h-[90vh] object-contain rounded-lg shadow-2xl shadow-black/50"
              onClick={(e) => e.stopPropagation()}
            />
            <motion.div 
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              className="absolute bottom-8 left-1/2 -translate-x-1/2 bg-black/60 backdrop-blur-md px-6 py-3 rounded-full border border-white/10 text-white font-medium shadow-xl"
            >
              {selectedImage.event_name}
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* 1. Hero Section */}
      <section className="relative w-full h-[600px] lg:h-[800px] flex items-center justify-center pt-20">
        <div className="absolute inset-0 z-0 overflow-hidden">
          {/* Animated Mathematical Grid Backdrop */}
          <motion.div 
            animate={{ opacity: [0.1, 0.2, 0.1] }}
            transition={{ duration: 4, repeat: Infinity, ease: "easeInOut" }}
            className="absolute inset-0 bg-[linear-gradient(to_right,#80808012_1px,transparent_1px),linear-gradient(to_bottom,#80808012_1px,transparent_1px)] bg-[size:24px_24px]"
          />
          
          {loading ? (
            <Skeleton className="w-full h-full rounded-none" />
          ) : (
            <>
              <div className="absolute inset-0 bg-gradient-to-t from-obsidian-dark via-slate-950/80 to-transparent z-10" />
              <img 
                src={safeHeroSlides[0]?.image_url} 
                alt="PMT Hero" 
                className="w-full h-full object-cover opacity-30 object-top"
              />
            </>
          )}
        </div>
        
        <motion.div 
          initial={{ opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8 }}
          viewport={{ once: true }}
          className="relative z-20 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 text-center"
        >
          <span className="inline-block py-1 px-3 rounded-full border border-amber-500/30 bg-amber-500/10 text-amber-500 text-sm font-semibold tracking-wider mb-6 shadow-[0_0_15px_rgba(245,158,11,0.2)]">
            UNIVERSITY OF SRI JAYEWARDENEPURA
          </span>
          <h1 className="text-5xl md:text-7xl font-bold tracking-tight text-white mb-8 leading-tight">
            <span className="block text-gold-gradient">Pure Mathematics</span>
            <span className="block text-slate-200 text-4xl md:text-5xl mt-2">Subject Community</span>
          </h1>
          <p className="max-w-2xl mx-auto text-xl text-slate-400 mb-10">
            {loading ? <Skeleton className="h-6 w-3/4 mx-auto" /> : safeHeroSlides[0]?.title}
          </p>
          <div className="flex flex-col sm:flex-row justify-center items-center gap-4">
            <motion.a 
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
              href="https://chat.whatsapp.com/KWvoKLIsvc9EFxZveo4qQy"
              target="_blank"
              rel="noopener noreferrer"
              className="w-full sm:w-auto rounded-xl bg-gradient-to-r from-green-500 to-emerald-600 px-8 py-4 text-base font-bold text-white shadow-lg hover:shadow-green-500/30 transition-all"
            >
              Join WhatsApp Community
            </motion.a>
            <motion.a 
              whileHover={{ scale: 1.05, backgroundColor: 'rgba(245,158,11,0.1)' }}
              whileTap={{ scale: 0.95 }}
              href="#events"
              className="w-full sm:w-auto rounded-xl border border-amber-500/30 bg-charcoal-card/80 backdrop-blur-sm px-8 py-4 text-base font-bold text-amber-500 transition-all"
            >
              Explore Events
            </motion.a>
          </div>
        </motion.div>
      </section>

      {/* 2. About & Pillars */}
      <section id="about" className="py-24 relative">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <motion.div 
            initial={{ opacity: 0, y: 30 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-16"
          >
            <h2 className="text-3xl md:text-4xl font-bold text-white mb-4">Our Foundation</h2>
            <p className="text-slate-400 max-w-2xl mx-auto">Built upon three core pillars to elevate the academic and social experience of Pure Mathematics undergraduates.</p>
          </motion.div>
          
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            {[
              { title: "Academic Excellence", desc: "Fostering a culture of rigorous mathematical inquiry and providing resources for academic success.", icon: AcademicCapIcon },
              { title: "Technological Innovation", desc: "Bridging the gap between abstract mathematics and modern computational technologies.", icon: LightBulbIcon },
              { title: "Strong Community", desc: "Creating lasting memories and building connections that extend beyond university life.", icon: UserGroupIcon },
            ].map((pillar, i) => (
              <motion.div 
                key={pillar.title}
                initial={{ opacity: 0, y: 30 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true, margin: "-50px" }}
                transition={{ delay: i * 0.15 }}
                whileHover={{ y: -6, scale: 1.02 }}
                className="bg-charcoal-card border border-white/5 rounded-2xl p-8 backdrop-blur-md shadow-2xl hover:border-amber-500/30 hover:shadow-[0_0_30px_rgba(245,158,11,0.1)] transition-all group"
              >
                <div className="w-14 h-14 bg-amber-500/10 rounded-xl flex items-center justify-center mb-6 group-hover:bg-amber-500/20 transition-colors">
                  <pillar.icon className="w-7 h-7 text-amber-500" />
                </div>
                <h3 className="text-xl font-bold text-white mb-3">{pillar.title}</h3>
                <p className="text-slate-400 leading-relaxed">{pillar.desc}</p>
              </motion.div>
            ))}
          </div>
        </div>
      </section>

      {/* 3. Milestones & Achievements */}
      <section id="achievements" className="py-24 bg-[#0a0c13] relative border-y border-amber-500/10">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-8">
            {loading ? (
              Array(4).fill(0).map((_, i) => <Skeleton key={i} className="h-32" />)
            ) : achievements.length > 0 ? (
              achievements.map((ach, i) => (
                <motion.div 
                  key={ach.id} 
                  initial={{ opacity: 0, scale: 0.9 }}
                  whileInView={{ opacity: 1, scale: 1 }}
                  viewport={{ once: true, margin: "-50px" }}
                  transition={{ delay: i * 0.1 }}
                  className="text-center p-6"
                >
                  <div className="w-12 h-12 mx-auto bg-amber-500/20 rounded-full flex items-center justify-center mb-4 shadow-[0_0_15px_rgba(245,158,11,0.15)]">
                    <TrophyIcon className="w-6 h-6 text-amber-500" />
                  </div>
                  <div className="text-4xl font-bold text-white mb-2">
                    <AnimatedCounter value={ach.count} />+
                  </div>
                  <div className="text-sm font-medium text-slate-400 uppercase tracking-wider">{ach.title}</div>
                </motion.div>
              ))
            ) : (
              // Fallback
              [
                { count: 500, label: "Students" },
                { count: 50, label: "Events Hosted" },
                { count: 20, label: "Workshops" },
                { count: 10, label: "Awards" },
              ].map((fallback, i) => (
                <motion.div 
                  key={fallback.label}
                  initial={{ opacity: 0, scale: 0.9 }}
                  whileInView={{ opacity: 1, scale: 1 }}
                  viewport={{ once: true }}
                  transition={{ delay: i * 0.1 }}
                  className="text-center p-6"
                >
                  <div className="text-4xl font-bold text-white mb-2"><AnimatedCounter value={fallback.count} />+</div>
                  <div className="text-sm font-medium text-slate-400 uppercase tracking-wider">{fallback.label}</div>
                </motion.div>
              ))
            )}
          </div>
        </div>
      </section>

      {/* 4. Latest Events Grid */}
      <section id="events" className="py-24 relative">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <motion.div 
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="flex flex-col md:flex-row justify-between items-end mb-12"
          >
            <div>
              <h2 className="text-3xl md:text-4xl font-bold text-white mb-4">Latest Events</h2>
              <p className="text-slate-400">Discover what's happening in the PMT Family.</p>
            </div>
            
            <div className="mt-6 md:mt-0 flex bg-charcoal-card rounded-lg p-1 border border-white/10 shadow-inner">
              <button 
                onClick={() => setEventTab('Upcoming')}
                className={`px-6 py-2 rounded-md text-sm font-medium transition-colors ${eventTab === 'Upcoming' ? 'bg-amber-500 text-slate-950 shadow-md shadow-amber-500/20' : 'text-slate-400 hover:text-white'}`}
              >
                Upcoming
              </button>
              <button 
                onClick={() => setEventTab('Past')}
                className={`px-6 py-2 rounded-md text-sm font-medium transition-colors ${eventTab === 'Past' ? 'bg-amber-500 text-slate-950 shadow-md shadow-amber-500/20' : 'text-slate-400 hover:text-white'}`}
              >
                Past
              </button>
            </div>
          </motion.div>

          <motion.div layout className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
            <AnimatePresence mode="popLayout">
              {loading ? (
                Array(3).fill(0).map((_, i) => <Skeleton key={i} className="h-80" />)
              ) : filteredEvents.length > 0 ? (
                filteredEvents.map(event => (
                  <motion.div 
                    key={event.id}
                    layout
                    initial={{ opacity: 0, scale: 0.9 }}
                    animate={{ opacity: 1, scale: 1 }}
                    exit={{ opacity: 0, scale: 0.9 }}
                    whileHover={{ y: -6, scale: 1.02 }}
                    className="bg-charcoal-card border border-white/5 rounded-2xl overflow-hidden group hover:border-amber-500/30 hover:shadow-[0_0_30px_rgba(245,158,11,0.15)] transition-all cursor-pointer"
                  >
                    <div className="relative h-48 overflow-hidden bg-slate-900">
                      <img src={event.image_url} alt={event.title} className="w-full h-full object-cover group-hover:scale-110 transition-transform duration-700" />
                      <div className="absolute top-4 right-4 px-3 py-1 bg-black/60 backdrop-blur-md rounded-full text-xs font-semibold text-amber-500 border border-amber-500/30">
                        {event.status}
                      </div>
                    </div>
                    <div className="p-6">
                      <div className="flex items-center text-sm text-slate-400 mb-3">
                        <CalendarIcon className="w-4 h-4 mr-2" />
                        {fmtDate(event.date)}
                      </div>
                      <h3 className="text-xl font-bold text-white mb-4 line-clamp-2 group-hover:text-amber-500 transition-colors">{event.title}</h3>
                      {event.url && (
                        <a href={event.url} target="_blank" rel="noopener noreferrer" className="inline-flex items-center text-sm font-medium text-amber-500 hover:text-amber-400 transition-colors">
                          View Details <ArrowRightIcon className="w-4 h-4 ml-1" />
                        </a>
                      )}
                    </div>
                  </motion.div>
                ))
              ) : (
                <motion.div 
                  initial={{ opacity: 0 }} animate={{ opacity: 1 }}
                  className="col-span-full py-12 text-center text-slate-500 border border-dashed border-white/10 rounded-2xl"
                >
                  No {eventTab.toLowerCase()} events found.
                </motion.div>
              )}
            </AnimatePresence>
          </motion.div>
        </div>
      </section>

      {/* 5. Photo Gallery */}
      <section id="gallery" className="py-24 bg-[#0a0c13] relative border-y border-amber-500/10">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <motion.div 
            initial={{ opacity: 0, y: 30 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-16"
          >
            <h2 className="text-3xl md:text-4xl font-bold text-white mb-4">Capturing Memories</h2>
            <p className="text-slate-400 max-w-2xl mx-auto">Moments of excellence, joy, and unity from our past events.</p>
          </motion.div>
          
          <div className="columns-1 md:columns-2 lg:columns-3 gap-6 space-y-6">
            {loading ? (
              Array(6).fill(0).map((_, i) => <Skeleton key={i} className="h-64 break-inside-avoid" />)
            ) : gallery.length > 0 ? (
              gallery.map((img, i) => (
                <motion.div 
                  key={img.id} 
                  initial={{ opacity: 0, y: 20 }}
                  whileInView={{ opacity: 1, y: 0 }}
                  viewport={{ once: true, margin: "-50px" }}
                  transition={{ delay: (i % 3) * 0.1 }}
                  whileHover={{ scale: 1.02 }}
                  onClick={() => setSelectedImage(img)}
                  className="relative group rounded-xl overflow-hidden break-inside-avoid shadow-lg bg-slate-900 border border-white/5 cursor-pointer hover:shadow-[0_0_20px_rgba(245,158,11,0.2)] hover:border-amber-500/40 transition-all"
                >
                  <img src={img.image_url} alt={img.event_name} className="w-full h-auto object-cover group-hover:scale-105 transition-transform duration-700" loading="lazy" />
                  <div className="absolute inset-0 bg-gradient-to-t from-black/90 via-black/30 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300 flex items-end p-6">
                    <span className="text-white font-medium drop-shadow-md translate-y-4 group-hover:translate-y-0 transition-transform duration-300">{img.event_name}</span>
                  </div>
                </motion.div>
              ))
            ) : (
              // Fallback
              Array(3).fill(0).map((_, i) => (
                <div key={i} className="h-64 bg-charcoal-card border border-white/5 rounded-xl break-inside-avoid flex items-center justify-center text-slate-600">
                  Image Placeholder
                </div>
              ))
            )}
          </div>
        </div>
      </section>

      {/* 6. Executive Committee */}
      <section id="committee" className="py-24 relative">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <motion.div 
            initial={{ opacity: 0, y: 30 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-16"
          >
            <h2 className="text-3xl md:text-4xl font-bold text-white mb-4">Executive Committee</h2>
            <p className="text-slate-400 max-w-2xl mx-auto">The dedicated team leading the Pure Mathematics subject community.</p>
          </motion.div>
          
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-8">
            {loading ? (
              Array(4).fill(0).map((_, i) => <Skeleton key={i} className="h-72" />)
            ) : reps.length > 0 ? (
              reps.map((rep, i) => (
                <motion.div 
                  key={rep.id} 
                  initial={{ opacity: 0, scale: 0.9 }}
                  whileInView={{ opacity: 1, scale: 1 }}
                  viewport={{ once: true }}
                  transition={{ delay: i * 0.1 }}
                  whileHover={{ y: -6 }}
                  className="bg-charcoal-card rounded-2xl p-6 text-center border border-white/5 hover:border-amber-500/30 hover:shadow-[0_0_30px_rgba(245,158,11,0.1)] transition-all group"
                >
                  <div className="relative w-28 h-28 mx-auto mb-6">
                    <div className="absolute inset-0 bg-gold-gradient rounded-full blur opacity-20 group-hover:opacity-50 group-hover:scale-110 transition-all duration-500" />
                    <img src={rep.image_url} alt={rep.name} className="relative w-full h-full object-cover rounded-full border-2 border-amber-500/50" />
                  </div>
                  <h3 className="text-lg font-bold text-white group-hover:text-amber-500 transition-colors">{rep.name}</h3>
                  <p className="text-amber-500 text-sm font-medium mb-2">{rep.role}</p>
                  <span className="inline-block px-2.5 py-0.5 bg-white/5 rounded text-xs text-slate-400 border border-white/10">{rep.batch}</span>
                </motion.div>
              ))
            ) : (
              <div className="col-span-full py-8 text-center text-slate-500">
                Committee profiles will be updated soon.
              </div>
            )}
          </div>
        </div>
      </section>

      {/* 7. Social Media & Community Hub */}
      <section id="community" className="py-24 bg-[#0a0c13] relative border-t border-amber-500/10">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <motion.div 
            initial={{ opacity: 0, scale: 0.95 }}
            whileInView={{ opacity: 1, scale: 1 }}
            viewport={{ once: true }}
            className="bg-charcoal-card border border-amber-500/20 rounded-3xl p-8 md:p-16 text-center shadow-[0_0_40px_rgba(245,158,11,0.05)] relative overflow-hidden"
          >
            {/* Ambient Background Glow */}
            <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[800px] h-[300px] bg-amber-500/10 blur-[100px] rounded-full pointer-events-none" />

            <h2 className="relative text-3xl md:text-5xl font-bold text-white mb-6">Join the PMT Family</h2>
            <p className="relative text-xl text-gold-gradient font-medium italic mb-12">"Creating memories, building connections."</p>
            
            <div className="relative flex flex-col md:flex-row justify-center items-stretch gap-6 max-w-4xl mx-auto">
              {/* WhatsApp Main Card */}
              <motion.a 
                whileHover={{ scale: 1.03, y: -5 }}
                whileTap={{ scale: 0.97 }}
                href="https://chat.whatsapp.com/KWvoKLIsvc9EFxZveo4qQy" 
                target="_blank" 
                rel="noopener noreferrer"
                className="flex-1 bg-gradient-to-br from-slate-900 to-emerald-950 border border-emerald-500/30 rounded-2xl p-8 flex flex-col items-center justify-center shadow-xl shadow-emerald-500/10 group"
              >
                <div className="w-16 h-16 bg-emerald-500/20 rounded-2xl flex items-center justify-center mb-4 group-hover:scale-110 group-hover:bg-emerald-500/30 transition-all">
                  <svg className="w-8 h-8 text-emerald-500" fill="currentColor" viewBox="0 0 24 24"><path d="M12.031 6.172c-3.181 0-5.767 2.586-5.768 5.766-.001 1.298.38 2.27 1.019 3.287l-.582 2.128 2.182-.573c.978.58 1.711.927 3.149.929 3.177 0 5.767-2.587 5.769-5.766s-2.59-5.771-5.769-5.771zm3.141 8.243c-.156.44-1.026.85-1.428.903-.393.052-.77.108-2.395-.536-1.95-1.002-3.21-2.99-3.328-3.145-.118-.155-.794-1.057-.794-2.016s.508-1.433.693-1.628c.184-.195.399-.244.53-.244s.262.001.378.005c.115.006.273-.044.428.33.155.374.529 1.29.575 1.384.047.094.078.203.02.316-.059.114-.089.184-.176.287-.088.103-.186.223-.263.315-.088.102-.18.214-.078.39.102.175.457.753 1.053 1.282.595.53 1.157.701 1.332.793.176.094.279.078.383-.041.103-.119.444-.515.565-.691.121-.176.241-.146.4-.088.159.059 1.002.472 1.174.558.172.086.287.129.329.201.043.072.043.418-.113.858z"/></svg>
                </div>
                <h3 className="text-2xl font-bold text-white mb-2 group-hover:text-emerald-400 transition-colors">WhatsApp</h3>
                <p className="text-emerald-400/80 text-sm font-medium mb-6">Official Student Group</p>
                <div className="px-6 py-2 bg-emerald-500 text-white rounded-full text-sm font-bold shadow-lg shadow-emerald-500/20 group-hover:bg-emerald-400 group-hover:shadow-emerald-400/40 transition-all animate-pulse group-hover:animate-none">
                  Join Now
                </div>
              </motion.a>
              
              <div className="flex-1 flex flex-col gap-6">
                {/* Facebook & Instagram */}
                <div className="flex gap-6 h-full">
                  <motion.a 
                    whileHover={{ scale: 1.05, y: -4 }}
                    whileTap={{ scale: 0.95 }}
                    href="https://www.facebook.com/share/1HgVyRGC1a/" 
                    target="_blank" 
                    rel="noopener noreferrer" 
                    className="flex-1 bg-slate-900 border border-blue-500/20 rounded-2xl p-6 flex flex-col items-center justify-center hover:bg-slate-800 hover:border-blue-500/40 hover:shadow-[0_0_20px_rgba(59,130,246,0.15)] transition-colors group"
                  >
                    <svg className="w-8 h-8 text-blue-500 mb-3 group-hover:scale-110 transition-transform" fill="currentColor" viewBox="0 0 24 24"><path d="M22.675 0h-21.35c-.732 0-1.325.593-1.325 1.325v21.351c0 .731.593 1.324 1.325 1.324h11.495v-9.294h-3.128v-3.622h3.128v-2.671c0-3.1 1.893-4.788 4.659-4.788 1.325 0 2.463.099 2.795.143v3.24l-1.918.001c-1.504 0-1.795.715-1.795 1.763v2.313h3.587l-.467 3.622h-3.12v9.293h6.116c.73 0 1.323-.593 1.323-1.325v-21.35c0-.732-.593-1.325-1.325-1.325z"/></svg>
                    <span className="text-white font-semibold group-hover:text-blue-400 transition-colors">Facebook</span>
                  </motion.a>
                  
                  <motion.a 
                    whileHover={{ scale: 1.05, y: -4 }}
                    whileTap={{ scale: 0.95 }}
                    href="https://www.instagram.com/pmtfamily_usj?stkn=MTJudm5vNnY1MzI5bQ%3D%3D&utm_source=qr" 
                    target="_blank" 
                    rel="noopener noreferrer" 
                    className="flex-1 bg-slate-900 border border-pink-500/20 rounded-2xl p-6 flex flex-col items-center justify-center hover:bg-slate-800 hover:border-pink-500/40 hover:shadow-[0_0_20px_rgba(236,72,153,0.15)] transition-colors group"
                  >
                    <svg className="w-8 h-8 text-pink-500 mb-3 group-hover:scale-110 transition-transform" fill="currentColor" viewBox="0 0 24 24"><path d="M12 2.163c3.204 0 3.584.012 4.85.07 3.252.148 4.771 1.691 4.919 4.919.058 1.265.069 1.645.069 4.849 0 3.205-.012 3.584-.069 4.849-.149 3.225-1.664 4.771-4.919 4.919-1.266.058-1.644.07-4.85.07-3.204 0-3.584-.012-4.849-.07-3.26-.149-4.771-1.699-4.92-.058-1.265-.07-1.644-.07-4.849 0-3.204.013-3.583.07-4.849.149-3.227 1.664-4.771 4.919-4.919 1.266-.057 1.645-.069 4.849-.069zm0-2.163c-3.259 0-3.667.014-4.947.072-4.358.2-6.78 2.618-6.98 6.98-.059 1.281-.073 1.689-.073 4.948 0 3.259.014 3.668.072 4.948.2 4.358 2.618 6.78 6.98 6.98 1.281.058 1.689.072 4.948.072 3.259 0 3.668-.014 4.948-.072 4.354-.2 6.782-2.618 6.979-6.98.059-1.28.073-1.689.073-4.948 0-3.259-.014-3.667-.072-4.947-.196-4.354-2.617-6.78-6.979-6.98-1.281-.059-1.69-.073-4.949-.073zm0 5.838c-3.403 0-6.162 2.759-6.162 6.162s2.759 6.163 6.162 6.163 6.162-2.759 6.162-6.163c0-3.403-2.759-6.162-6.162-6.162zm0 10.162c-2.209 0-4-1.79-4-4 0-2.209 1.791-4 4-4s4 1.791 4 4c0 2.21-1.791 4-4 4zm6.406-11.845c-.796 0-1.441.645-1.441 1.44s.645 1.44 1.441 1.44c.795 0 1.439-.645 1.439-1.44s-.644-1.44-1.439-1.44z"/></svg>
                    <span className="text-white font-semibold group-hover:text-pink-400 transition-colors">Instagram</span>
                  </motion.a>
                </div>
                
                {/* YouTube */}
                <motion.a 
                  whileHover={{ scale: 1.02, y: -4 }}
                  whileTap={{ scale: 0.98 }}
                  href="https://youtube.com/@pmtfamily-s2d?si=S7a2HGd3BuW2cdzP" 
                  target="_blank" 
                  rel="noopener noreferrer" 
                  className="bg-slate-900 border border-red-500/20 rounded-2xl p-6 flex items-center justify-center gap-4 hover:bg-slate-800 hover:border-red-500/40 hover:shadow-[0_0_20px_rgba(239,68,68,0.15)] transition-colors group"
                >
                  <svg className="w-8 h-8 text-red-500 group-hover:scale-110 transition-transform" fill="currentColor" viewBox="0 0 24 24"><path d="M19.615 3.184c-3.604-.246-11.631-.245-15.23 0-3.897.266-4.356 2.62-4.385 8.816.029 6.185.484 8.549 4.385 8.816 3.6.245 11.626.246 15.23 0 3.897-.266 4.356-2.62 4.385-8.816-.029-6.185-.484-8.549-4.385-8.816zm-10.615 12.816v-8l8 3.993-8 4.007z"/></svg>
                  <span className="text-white font-semibold group-hover:text-red-400 transition-colors">Subscribe on YouTube</span>
                </motion.a>
              </div>
            </div>
          </motion.div>
        </div>
      </section>
    </div>
  );
}
