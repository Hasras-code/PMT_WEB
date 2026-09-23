import { useState, useEffect } from 'react';
import { api, errMsg } from '../api/client';
import toast from 'react-hot-toast';
import { useAccess } from '../hooks/useRole';
import { Tabs, Card, Field, inputCls, PrimaryButton } from '../components/ui';

export default function PublicContent() {
  const access = useAccess();
  // Ensure strict RBAC: only PLATFORM_ADMIN acts as super_admin
  const isSuperAdmin = access?.platform_roles.includes('PLATFORM_ADMIN') ?? false;

  const tabIds = ['hero', 'events', 'gallery'];
  if (isSuperAdmin) {
    tabIds.push('stats', 'committee', 'social');
  }

  const [tab, setTab] = useState<string>(tabIds[0] || 'hero');

  const labels: Record<string, string> = {
    hero: 'Hero Slides',
    events: 'Events',
    gallery: 'Gallery',
    stats: 'Stats & Achievements',
    committee: 'Executive Committee',
    social: 'Social Links',
  };

  // --- Hero Slides Manager ---
  const [slides, setSlides] = useState<any[]>([]);
  const loadSlides = () => api.get('/v1/admin/public/hero-slides').then(res => setSlides(res.data || [])).catch(() => {});
  
  const handleSaveSlide = async (e: any) => {
    e.preventDefault();
    const data = new FormData(e.target);
    try {
      await api.post('/v1/admin/public/hero-slides', Object.fromEntries(data.entries()));
      toast.success('Hero slide saved');
      e.target.reset();
      loadSlides();
    } catch (err) {
      toast.error(errMsg(err, 'Failed to save slide'));
    }
  };

  const deleteSlide = async (id: string) => {
    try {
      await api.delete(`/v1/admin/public/hero-slides/${id}`);
      toast.success('Hero slide deleted');
      loadSlides();
    } catch (err) {
      toast.error(errMsg(err, 'Failed to delete slide'));
    }
  };

  // --- Events Manager ---
  const [events, setEvents] = useState<any[]>([]);
  const loadEvents = () => api.get('/v1/admin/public/events').then(res => setEvents(res.data || [])).catch(() => {});
  
  const handleSaveEvent = async (e: any) => {
    e.preventDefault();
    const data = new FormData(e.target);
    try {
      await api.post('/v1/admin/public/events', Object.fromEntries(data.entries()));
      toast.success('Event saved');
      e.target.reset();
      loadEvents();
    } catch (err) {
      toast.error(errMsg(err, 'Failed to save event'));
    }
  };

  const deleteEvent = async (id: string) => {
    try {
      await api.delete(`/v1/admin/public/events/${id}`);
      toast.success('Event deleted');
      loadEvents();
    } catch (err) {
      toast.error(errMsg(err, 'Failed to delete event'));
    }
  };

  // --- Gallery Manager ---
  const [gallery, setGallery] = useState<any[]>([]);
  const loadGallery = () => api.get('/v1/admin/public/gallery').then(res => setGallery(res.data || [])).catch(() => {});
  
  const handleSaveGallery = async (e: any) => {
    e.preventDefault();
    const data = new FormData(e.target);
    try {
      await api.post('/v1/admin/public/gallery', {
        title: data.get('title'),
        category: data.get('category'),
        image_url: data.get('image_url'),
        visibility: data.get('visibility') === 'on' ? 'Public' : 'Internal'
      });
      toast.success('Gallery item saved');
      e.target.reset();
      loadGallery();
    } catch (err) {
      toast.error(errMsg(err, 'Failed to save gallery item'));
    }
  };

  const deleteGallery = async (id: string) => {
    try {
      await api.delete(`/v1/admin/public/gallery/${id}`);
      toast.success('Gallery item deleted');
      loadGallery();
    } catch (err) {
      toast.error(errMsg(err, 'Failed to delete gallery item'));
    }
  };

  // --- Stats Manager (Super Admin) ---
  const [stats, setStats] = useState<any[]>([]);
  const loadStats = () => api.get('/v1/admin/public/achievements').then(res => setStats(res.data || [])).catch(() => {});
  
  const handleSaveStat = async (e: any) => {
    e.preventDefault();
    if (!isSuperAdmin) return toast.error("Unauthorized");
    const data = new FormData(e.target);
    try {
      await api.post('/v1/admin/public/achievements', {
        title: data.get('title'),
        count: parseInt(data.get('count') as string, 10),
        icon: data.get('icon')
      });
      toast.success('Stat saved');
      e.target.reset();
      loadStats();
    } catch (err) {
      toast.error(errMsg(err, 'Failed to save stat'));
    }
  };

  const deleteStat = async (id: string) => {
    if (!isSuperAdmin) return toast.error("Unauthorized");
    try {
      await api.delete(`/v1/admin/public/achievements/${id}`);
      toast.success('Stat deleted');
      loadStats();
    } catch (err) {
      toast.error(errMsg(err, 'Failed to delete stat'));
    }
  };

  // --- Committee Manager (Super Admin) ---
  const [reps, setReps] = useState<any[]>([]);
  const loadReps = () => api.get('/v1/admin/public/reps').then(res => setReps(res.data || [])).catch(() => {});
  
  const toggleRepFeatured = async (id: string, currentFeatured: boolean) => {
    if (!isSuperAdmin) return toast.error("Unauthorized");
    try {
      await api.post(`/v1/admin/public/reps/${id}/toggle-featured`, { isFeaturedOnHome: !currentFeatured });
      toast.success('Committee feature status updated');
      loadReps();
    } catch (err) {
      toast.error(errMsg(err, 'Failed to update feature status'));
    }
  };

  // --- Social Links Manager (Super Admin) ---
  const [socialLinks, setSocialLinks] = useState({ whatsapp: '', facebook: '', instagram: '', youtube: '' });
  const loadSocials = () => api.get('/v1/admin/public/socials').then(res => setSocialLinks(res.data || {})).catch(() => {});
  
  const handleSaveSocials = async (e: any) => {
    e.preventDefault();
    if (!isSuperAdmin) return toast.error("Unauthorized");
    const data = new FormData(e.target);
    try {
      await api.post('/v1/admin/public/socials', Object.fromEntries(data.entries()));
      toast.success('Social links updated');
      loadSocials();
    } catch (err) {
      toast.error(errMsg(err, 'Failed to update social links'));
    }
  };

  useEffect(() => {
    if (tab === 'hero') loadSlides();
    if (tab === 'events') loadEvents();
    if (tab === 'gallery') loadGallery();
    if (tab === 'stats' && isSuperAdmin) loadStats();
    if (tab === 'committee' && isSuperAdmin) loadReps();
    if (tab === 'social' && isSuperAdmin) loadSocials();
  }, [tab, isSuperAdmin]);

  const renderTab = () => {
    switch (tab) {
      case 'hero':
        return (
          <div className="space-y-6">
            <Card className="p-6">
              <h3 className="text-lg font-bold mb-4">Add Hero Slide</h3>
              <form onSubmit={handleSaveSlide} className="grid gap-4 md:grid-cols-2">
                <Field label="Title"><input name="title" required className={inputCls} /></Field>
                <Field label="Subtitle"><input name="subtitle" className={inputCls} /></Field>
                <Field label="CTA Text"><input name="cta_text" className={inputCls} defaultValue="Explore Events" /></Field>
                <Field label="Image URL"><input name="image_url" required className={inputCls} /></Field>
                <Field label="Priority Order"><input name="priority" type="number" defaultValue="0" className={inputCls} /></Field>
                <div className="col-span-full pt-4">
                  <PrimaryButton type="submit">Save Slide</PrimaryButton>
                </div>
              </form>
            </Card>
            <div className="grid gap-4 md:grid-cols-2">
              {slides.map(s => (
                <Card key={s.id} className="p-4 flex flex-col gap-2 relative">
                  <img src={s.image_url} alt={s.title} className="w-full h-32 object-cover rounded-md" />
                  <h4 className="font-bold">{s.title}</h4>
                  <button onClick={() => deleteSlide(s.id)} className="absolute top-2 right-2 bg-red-500/20 text-red-500 rounded p-1">Delete</button>
                </Card>
              ))}
            </div>
          </div>
        );

      case 'events':
        return (
          <div className="space-y-6">
            <Card className="p-6">
              <h3 className="text-lg font-bold mb-4">Add Public Event</h3>
              <form onSubmit={handleSaveEvent} className="grid gap-4 md:grid-cols-2">
                <Field label="Event Title"><input name="title" required className={inputCls} /></Field>
                <Field label="Date"><input name="date" type="date" required className={inputCls} /></Field>
                <Field label="Image URL"><input name="image_url" required className={inputCls} /></Field>
                <Field label="Status">
                  <select name="status" className={inputCls}>
                    <option value="Upcoming">Upcoming</option>
                    <option value="Past">Past</option>
                  </select>
                </Field>
                <Field label="External URL"><input name="url" className={inputCls} placeholder="https://..." /></Field>
                <div className="col-span-full pt-4">
                  <PrimaryButton type="submit">Save Event</PrimaryButton>
                </div>
              </form>
            </Card>
            <div className="grid gap-4 md:grid-cols-2">
              {events.map(ev => (
                <Card key={ev.id} className="p-4 flex gap-4 relative">
                  <img src={ev.image_url} alt={ev.title} className="w-20 h-20 object-cover rounded-md" />
                  <div>
                    <h4 className="font-bold">{ev.title}</h4>
                    <span className="text-sm text-slate-400">{ev.status}</span>
                  </div>
                  <button onClick={() => deleteEvent(ev.id)} className="absolute top-2 right-2 text-red-500 text-sm">Delete</button>
                </Card>
              ))}
            </div>
          </div>
        );

      case 'gallery':
        return (
          <div className="space-y-6">
            <Card className="p-6">
              <h3 className="text-lg font-bold mb-4">Add Gallery Image</h3>
              <form onSubmit={handleSaveGallery} className="grid gap-4 md:grid-cols-2">
                <Field label="Caption / Event Name"><input name="title" required className={inputCls} /></Field>
                <Field label="Category">
                  <select name="category" className={inputCls}>
                    <option value="Academic">Academic</option>
                    <option value="Workshops">Workshops</option>
                    <option value="Social">Social</option>
                    <option value="Sports">Sports</option>
                  </select>
                </Field>
                <Field label="Image URL"><input name="image_url" required className={inputCls} /></Field>
                <Field label="Visibility">
                  <label className="flex items-center gap-2 mt-2">
                    <input type="checkbox" name="visibility" defaultChecked className="rounded border-line bg-surface text-primary focus:ring-primary/20" />
                    <span className="text-sm">Make Public (Show on Homepage)</span>
                  </label>
                </Field>
                <div className="col-span-full pt-4">
                  <PrimaryButton type="submit">Save to Gallery</PrimaryButton>
                </div>
              </form>
            </Card>
            <div className="grid gap-4 grid-cols-2 md:grid-cols-4">
              {gallery.map(img => (
                <div key={img.id} className="relative group">
                  <img src={img.image_url} alt={img.title} className="w-full h-32 object-cover rounded-md" />
                  <button onClick={() => deleteGallery(img.id)} className="absolute top-2 right-2 bg-black/60 text-red-500 rounded p-1 opacity-0 group-hover:opacity-100 transition-opacity">Delete</button>
                </div>
              ))}
            </div>
          </div>
        );

      case 'stats':
        if (!isSuperAdmin) return null;
        return (
          <div className="space-y-6">
            <Card className="p-6">
              <h3 className="text-lg font-bold mb-4">Add Achievement Stat</h3>
              <form onSubmit={handleSaveStat} className="grid gap-4 md:grid-cols-2">
                <Field label="Title (e.g. Students, Events)"><input name="title" required className={inputCls} /></Field>
                <Field label="Count (e.g. 500)"><input name="count" type="number" required className={inputCls} /></Field>
                <Field label="Icon name"><input name="icon" defaultValue="TrophyIcon" className={inputCls} /></Field>
                <div className="col-span-full pt-4">
                  <PrimaryButton type="submit">Save Stat</PrimaryButton>
                </div>
              </form>
            </Card>
            <div className="grid gap-4 md:grid-cols-3">
              {stats.map(st => (
                <Card key={st.id} className="p-4 relative">
                  <div className="text-2xl font-bold">{st.count}+</div>
                  <div className="text-slate-400">{st.title}</div>
                  <button onClick={() => deleteStat(st.id)} className="absolute top-2 right-2 text-red-500 text-sm">Remove</button>
                </Card>
              ))}
            </div>
          </div>
        );

      case 'committee':
        if (!isSuperAdmin) return null;
        return (
          <div className="space-y-6">
            <div className="grid gap-4 md:grid-cols-3 lg:grid-cols-4">
              {reps.map(rep => (
                <Card key={rep.id} className="p-4 text-center">
                  <img src={rep.image_url || `https://ui-avatars.com/api/?name=${rep.name}`} className="w-16 h-16 mx-auto rounded-full mb-4" />
                  <h4 className="font-bold">{rep.name}</h4>
                  <div className="text-sm text-slate-400 mb-4">{rep.role}</div>
                  <button onClick={() => toggleRepFeatured(rep.id, rep.isFeaturedOnHome)} className={`w-full px-3 py-2 rounded-lg text-xs font-bold transition-all ${rep.isFeaturedOnHome ? 'bg-amber-500 text-slate-900' : 'bg-white/10 text-white hover:bg-white/20'}`}>
                    {rep.isFeaturedOnHome ? 'Featured on Home' : 'Set as Featured'}
                  </button>
                </Card>
              ))}
            </div>
          </div>
        );

      case 'social':
        if (!isSuperAdmin) return null;
        return (
          <Card className="p-6 max-w-2xl">
            <h3 className="text-lg font-bold mb-4">Global Social Media Links</h3>
            <p className="text-sm text-slate-400 mb-6">These links map to the global footer and homepage Community Hub.</p>
            <form onSubmit={handleSaveSocials} className="space-y-4">
              <Field label="WhatsApp Community Link">
                <input name="whatsapp" defaultValue={socialLinks.whatsapp} className={inputCls} placeholder="https://chat.whatsapp.com/..." />
              </Field>
              <Field label="Facebook Page">
                <input name="facebook" defaultValue={socialLinks.facebook} className={inputCls} placeholder="https://facebook.com/..." />
              </Field>
              <Field label="Instagram Profile">
                <input name="instagram" defaultValue={socialLinks.instagram} className={inputCls} placeholder="https://instagram.com/..." />
              </Field>
              <Field label="YouTube Channel">
                <input name="youtube" defaultValue={socialLinks.youtube} className={inputCls} placeholder="https://youtube.com/..." />
              </Field>
              <div className="pt-4">
                <PrimaryButton type="submit">Update Social Links</PrimaryButton>
              </div>
            </form>
          </Card>
        );

      default: return null;
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Public Content</h1>
          <p className="text-slate-400 text-sm mt-1">Manage what guests see on the platform homepage.</p>
        </div>
      </div>
      
      <Tabs tabs={tabIds} active={tab} onChange={setTab} labels={labels} />
      
      <div className="pt-4 animate-in fade-in slide-in-from-bottom-4 duration-500">
        {renderTab()}
      </div>
    </div>
  );
}
