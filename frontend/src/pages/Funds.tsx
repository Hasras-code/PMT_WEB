import { useCallback, useEffect, useMemo, useState } from 'react';
import type { Dispatch, FormEvent, SetStateAction } from 'react';
import {
  ArrowDownTrayIcon,
  ArrowPathIcon,
  ArrowsRightLeftIcon,
  BanknotesIcon,
  CalendarDaysIcon,
  DocumentArrowUpIcon,
  PlusIcon,
  UserGroupIcon,
} from '@heroicons/react/24/outline';
import toast from 'react-hot-toast';
import { api, errMsg, toList, uploadFile } from '../api/client';
import { Badge, Card, CardTitle, Empty, Field, fmtDate, fmtDateTime, inputCls, statusTone } from '../components/ui';
import { useAppStore } from '../store/app';
import { useAuthStore } from '../store/auth';
import { useCan } from '../hooks/useRole';
import type {
  BirthdayContribution,
  BirthdayPeriod,
  BirthdaySummary,
  AccessContext,
  FundAttachment,
  FundManager,
  FundSummary,
  FundTransaction,
  FundTransfer,
  LmsEvent,
  Member,
} from '../types';

type Tab = 'funds' | 'managers' | 'transfers' | 'birthday';
type TransactionFormState = {
  type: string;
  amount: string;
  description: string;
  category: string;
  source_type: string;
  source_name: string;
  reference: string;
  status: string;
};
type FundingEventFormState = {
  title: string;
  description: string;
  location: string;
  starts_at: string;
  ends_at: string;
  visibility: string;
};

const money = new Intl.NumberFormat('en-LK', { style: 'currency', currency: 'LKR', minimumFractionDigits: 2 });
const formatMoney = (minor: number | null | undefined) => money.format((minor || 0) / 100);
const toMinor = (value: string) => Math.round(Number(value) * 100);
const human = (value: string) => value.replaceAll('_', ' ').toLowerCase().replace(/^./, (c) => c.toUpperCase());
const positiveTypes = new Set(['CASH_IN', 'TRANSFER_IN', 'REVERSAL_IN']);

export default function Funds() {
  const { currentBatchID, currentBatch } = useAppStore();
  const user = useAuthStore((state) => state.user);
  const setAccess = useAuthStore((state) => state.setAccess);
  const canCreateFund = useCan('fund.create', currentBatchID) === true;
  const canManageEvents = useCan('event.manage', currentBatchID) === true;
  const canCreateTransaction = useCan('fund.transaction.create', currentBatchID) === true;
  const canCloseFund = useCan('fund.close', currentBatchID) === true;
  const canAssignManager = useCan('fund.manager.assign', currentBatchID) === true;
  const canTransfer = useCan('fund.transfer.create', currentBatchID) === true;
  const canReverseTransfer = useCan('fund.transfer.reverse', currentBatchID) === true;
  const canManageBirthday = useCan('birthday_fund.manage', currentBatchID) === true;
  const canRecordContribution = useCan('birthday_contribution.record', currentBatchID) === true;

  const [tab, setTab] = useState<Tab>('funds');
  const [funds, setFunds] = useState<FundSummary[]>([]);
  const [transfers, setTransfers] = useState<FundTransfer[]>([]);
  const [birthday, setBirthday] = useState<BirthdaySummary | null>(null);
  const [periods, setPeriods] = useState<BirthdayPeriod[]>([]);
  const [selectedFundID, setSelectedFundID] = useState('');
  const [selectedPeriodID, setSelectedPeriodID] = useState('');
  const [transactions, setTransactions] = useState<FundTransaction[]>([]);
  const [managers, setManagers] = useState<FundManager[]>([]);
  const [members, setMembers] = useState<Member[]>([]);
  const [events, setEvents] = useState<LmsEvent[]>([]);
  const [contributions, setContributions] = useState<BirthdayContribution[]>([]);
  const [myContribution, setMyContribution] = useState<BirthdayContribution | null>(null);
  const [attachments, setAttachments] = useState<Record<string, FundAttachment[]>>({});
  const [loading, setLoading] = useState(false);
  const [busy, setBusy] = useState(false);
  const [showCreateFund, setShowCreateFund] = useState(false);
  const [showTransaction, setShowTransaction] = useState(false);
  const [showTransfer, setShowTransfer] = useState(false);
  const [includeDrafts, setIncludeDrafts] = useState(false);
  const [transactionType, setTransactionType] = useState('');
  const [fundForm, setFundForm] = useState({ name: '', description: '', type: 'OTHER', event_link: 'none', event_id: '' });
  const [eventForm, setEventForm] = useState<FundingEventFormState>({ title: '', description: '', location: '', starts_at: '', ends_at: '', visibility: 'MEMBERS_ONLY' });
  const [transactionForm, setTransactionForm] = useState<TransactionFormState>({ type: 'CASH_IN', amount: '', description: '', category: '', source_type: '', source_name: '', reference: '', status: 'POSTED' });
  const [transferForm, setTransferForm] = useState({ from_fund_id: '', to_fund_id: '', type: 'TRANSFER', amount: '', description: '' });
  const [periodForm, setPeriodForm] = useState({ year: String(new Date().getFullYear()), month: String(new Date().getMonth() + 1), amount: '200' });

  const selectedFund = funds.find((item) => item.id === selectedFundID) || null;
  const selectedPeriod = periods.find((item) => item.id === selectedPeriodID) || null;
  const assignedManager = managers.some((item) => item.user_id === user?.id);
  const canManageSelected = Boolean(selectedFund?.status === 'ACTIVE' && (canCreateTransaction || assignedManager));

  const loadOverview = useCallback(async () => {
    if (!currentBatchID) return;
    setLoading(true);
    try {
      const [fundResponse, transferResponse, birthdayResponse, periodResponse] = await Promise.all([
        api.get(`/v1/batches/${currentBatchID}/funds`, { limit: 100 }),
        api.get(`/v1/batches/${currentBatchID}/fund-transfers`, { limit: 100 }),
        api.get(`/v1/batches/${currentBatchID}/birthday-fund`),
        api.get(`/v1/batches/${currentBatchID}/birthday-fund/periods`, { limit: 100 }),
      ]);
      const nextFunds = toList<FundSummary>(fundResponse.data);
      setFunds(nextFunds);
      setTransfers(toList<FundTransfer>(transferResponse.data));
      setBirthday(birthdayResponse.data as BirthdaySummary);
      const nextPeriods = toList<BirthdayPeriod>(periodResponse.data);
      setPeriods(nextPeriods);
      setSelectedFundID((current) => nextFunds.some((item) => item.id === current) ? current : nextFunds[0]?.id || '');
      setSelectedPeriodID((current) => nextPeriods.some((item) => item.id === current) ? current : nextPeriods[0]?.id || '');
    } catch (err) {
      toast.error(errMsg(err, 'Could not load fund information'));
    } finally {
      setLoading(false);
    }
  }, [currentBatchID]);

  const loadFund = useCallback(async () => {
    if (!currentBatchID || !selectedFundID) return;
    const params: Record<string, unknown> = { limit: 100 };
    if (includeDrafts) params.include_drafts = true;
    if (transactionType) params.type = transactionType;
    try {
      const [transactionResponse, managerResponse] = await Promise.all([
        api.get(`/v1/batches/${currentBatchID}/funds/${selectedFundID}/transactions`, params),
        api.get(`/v1/batches/${currentBatchID}/funds/${selectedFundID}/managers`),
      ]);
      setTransactions(toList<FundTransaction>(transactionResponse.data));
      setManagers(toList<FundManager>(managerResponse.data));
    } catch (err) {
      if (includeDrafts) setIncludeDrafts(false);
      else toast.error(errMsg(err, 'Could not load the fund ledger'));
    }
  }, [currentBatchID, includeDrafts, selectedFundID, transactionType]);

  const loadPeriod = useCallback(async () => {
    if (!currentBatchID || !selectedPeriodID) return;
    const own = api.get(`/v1/batches/${currentBatchID}/birthday-fund/periods/${selectedPeriodID}/my-contribution`)
      .then((response) => setMyContribution(response.data as BirthdayContribution))
      .catch(() => setMyContribution(null));
    const matrix = canManageBirthday
      ? api.get(`/v1/batches/${currentBatchID}/birthday-fund/periods/${selectedPeriodID}/contributions`, { limit: 100 }).then((response) => setContributions(toList<BirthdayContribution>(response.data)))
      : Promise.resolve(setContributions([]));
    await Promise.all([own, matrix]);
  }, [canManageBirthday, currentBatchID, selectedPeriodID]);

  useEffect(() => { setSelectedFundID(''); setSelectedPeriodID(''); setAttachments({}); loadOverview(); }, [loadOverview]);
  useEffect(() => { loadFund(); }, [loadFund]);
  useEffect(() => { loadPeriod(); }, [loadPeriod]);
  useEffect(() => {
    let cancelled = false;
    api.get('/v1/me/access').then((response) => {
      if (!cancelled) setAccess(response.data as AccessContext);
    }).catch(() => {});
    return () => { cancelled = true; };
  }, [currentBatchID, setAccess]);
  useEffect(() => {
    if (!canAssignManager || !currentBatchID) return;
    api.get(`/v1/batches/${currentBatchID}/members`, { limit: 100 }).then((response) => setMembers(toList<Member>(response.data))).catch(() => {});
  }, [canAssignManager, currentBatchID]);
  useEffect(() => {
    if (!canCreateFund || !currentBatchID) {
      setEvents([]);
      return;
    }
    api.get(`/v1/batches/${currentBatchID}/events`, { limit: 100 }).then((response) => setEvents(toList<LmsEvent>(response.data))).catch(() => setEvents([]));
  }, [canCreateFund, currentBatchID]);

  const totals = useMemo(() => ({
    balance: funds.reduce((sum, item) => sum + item.balance_minor, 0),
    income: funds.reduce((sum, item) => sum + item.total_cash_in_minor, 0),
    expense: funds.reduce((sum, item) => sum + item.total_expense_minor, 0),
    loans: funds.reduce((sum, item) => sum + item.outstanding_loans_payable_minor, 0),
  }), [funds]);

  const run = async (task: () => Promise<unknown>, message: string, refresh: 'all' | 'fund' | 'period' = 'all'): Promise<boolean> => {
    setBusy(true);
    try {
      await task();
      toast.success(message);
      if (refresh === 'fund') await Promise.all([loadOverview(), loadFund()]);
      else if (refresh === 'period') await Promise.all([loadOverview(), loadPeriod()]);
      else await loadOverview();
      return true;
    } catch (err) {
      toast.error(errMsg(err, 'The operation could not be completed'));
      return false;
    } finally {
      setBusy(false);
    }
  };

  const createFund = (event: FormEvent) => {
    event.preventDefault();
    const body: Record<string, unknown> = { name: fundForm.name, description: fundForm.description, type: fundForm.type };
    if (fundForm.type === 'EVENT' && fundForm.event_link === 'existing') {
      if (!fundForm.event_id) return toast.error('Select an event to link');
      body.event_id = fundForm.event_id;
    }
    if (fundForm.type === 'EVENT' && fundForm.event_link === 'new') {
      if (!eventForm.title.trim() || !eventForm.starts_at) return toast.error('Enter the event title and start time');
      const startsAt = new Date(eventForm.starts_at);
      const endsAt = eventForm.ends_at ? new Date(eventForm.ends_at) : null;
      if (Number.isNaN(startsAt.getTime()) || (endsAt && Number.isNaN(endsAt.getTime()))) return toast.error('Enter valid event dates');
      body.event = {
        title: eventForm.title.trim(),
        description: eventForm.description,
        location: eventForm.location,
        starts_at: startsAt.toISOString(),
        ...(endsAt ? { ends_at: endsAt.toISOString() } : {}),
        visibility: eventForm.visibility,
      };
    }
    run(() => api.post(`/v1/batches/${currentBatchID}/funds`, body), fundForm.type === 'EVENT' && fundForm.event_link === 'new' ? 'Event and fund created' : 'Fund created').then((succeeded) => {
      if (!succeeded) return;
      setFundForm({ name: '', description: '', type: 'OTHER', event_link: 'none', event_id: '' });
      setEventForm({ title: '', description: '', location: '', starts_at: '', ends_at: '', visibility: 'MEMBERS_ONLY' });
      setShowCreateFund(false);
    });
  };

  const createTransaction = (event: FormEvent) => {
    event.preventDefault();
    const amount_minor = toMinor(transactionForm.amount);
    if (!Number.isSafeInteger(amount_minor) || amount_minor <= 0) return toast.error('Enter a valid positive amount');
    const body: Record<string, unknown> = { type: transactionForm.type, amount_minor, description: transactionForm.description, status: transactionForm.status };
    for (const key of ['category', 'source_type', 'source_name', 'reference'] as const) if (transactionForm[key].trim()) body[key] = transactionForm[key].trim();
    run(() => api.post(`/v1/batches/${currentBatchID}/funds/${selectedFundID}/transactions`, body), transactionForm.status === 'DRAFT' ? 'Draft saved' : 'Transaction posted', 'fund').then((succeeded) => {
      if (!succeeded) return;
      setTransactionForm({ type: 'CASH_IN', amount: '', description: '', category: '', source_type: '', source_name: '', reference: '', status: 'POSTED' }); setShowTransaction(false);
    });
  };

  const createTransfer = (event: FormEvent) => {
    event.preventDefault();
    const amount_minor = toMinor(transferForm.amount);
    if (!Number.isSafeInteger(amount_minor) || amount_minor <= 0) return toast.error('Enter a valid positive amount');
    run(() => api.post(`/v1/batches/${currentBatchID}/fund-transfers`, { ...transferForm, amount_minor }), transferForm.type === 'LOAN' ? 'Loan recorded' : 'Transfer completed').then((succeeded) => {
      if (!succeeded) return;
      setTransferForm({ from_fund_id: '', to_fund_id: '', type: 'TRANSFER', amount: '', description: '' }); setShowTransfer(false);
    });
  };

  const createPeriod = (event: FormEvent) => {
    event.preventDefault();
    const amount_minor = toMinor(periodForm.amount);
    run(() => api.post(`/v1/batches/${currentBatchID}/birthday-fund/periods`, { year: Number(periodForm.year), month: Number(periodForm.month), amount_minor }), 'Contribution period created', 'period');
  };

  const assignManager = (membershipID: string) => {
    if (!membershipID) return Promise.resolve(false);
    return run(() => api.post(`/v1/batches/${currentBatchID}/funds/${selectedFundID}/managers`, { membership_id: membershipID }), 'Fund manager assigned', 'fund');
  };

  const openFundForm = (type: 'OTHER' | 'EVENT') => {
    setTab('funds');
    setFundForm({ name: '', description: '', type, event_link: type === 'EVENT' ? 'new' : 'none', event_id: '' });
    setEventForm({ title: '', description: '', location: '', starts_at: '', ends_at: '', visibility: 'MEMBERS_ONLY' });
    setShowCreateFund(true);
  };

  const reverseTransaction = (transaction: FundTransaction) => {
    if (!window.confirm(`Reverse ${formatMoney(transaction.amount_minor)} — ${transaction.description}?`)) return;
    run(() => api.post(`/v1/batches/${currentBatchID}/funds/${selectedFundID}/transactions/${transaction.id}/reverse`), 'Transaction reversed', 'fund');
  };

  const repay = (transfer: FundTransfer) => {
    const value = window.prompt(`Repayment amount (maximum ${formatMoney(transfer.outstanding_minor)})`);
    if (!value) return;
    const amount_minor = toMinor(value);
    run(() => api.post(`/v1/batches/${currentBatchID}/fund-transfers/${transfer.id}/repayments`, { amount_minor }), 'Loan repayment recorded');
  };

  const recordContribution = (contribution: BirthdayContribution) => {
    const value = window.prompt(`Payment from ${contribution.display_name || 'student'} (outstanding ${formatMoney(contribution.outstanding_minor)})`);
    if (!value) return;
    run(() => api.post(`/v1/batches/${currentBatchID}/birthday-fund/periods/${selectedPeriodID}/contributions/${contribution.membership_id}/payments`, { amount_minor: toMinor(value) }), 'Contribution payment recorded', 'period');
  };

  const uploadReceipt = async (transaction: FundTransaction, file?: File) => {
    if (!file || file.type !== 'application/pdf') return toast.error('Choose a PDF receipt');
    await run(async () => {
      const init = await api.post(`/v1/batches/${currentBatchID}/funds/${selectedFundID}/transactions/${transaction.id}/attachments/uploads`, { file_name: file.name, mime_type: 'application/pdf', size_bytes: file.size });
      await uploadFile(init.data.upload_url, file);
      await api.post(`/v1/batches/${currentBatchID}/funds/${selectedFundID}/transactions/${transaction.id}/attachments`, { upload_id: init.data.upload_id, visibility: 'MEMBERS' });
      await loadAttachments(transaction.id);
    }, 'Receipt attached', 'fund');
  };

  const loadAttachments = async (transactionID: string) => {
    try {
      const response = await api.get(`/v1/batches/${currentBatchID}/funds/${selectedFundID}/transactions/${transactionID}/attachments`);
      setAttachments((current) => ({ ...current, [transactionID]: toList<FundAttachment>(response.data) }));
    } catch (err) { toast.error(errMsg(err, 'Could not load receipts')); }
  };

  const downloadAttachment = async (transactionID: string, attachmentID: string) => {
    try {
      const response = await api.get(`/v1/batches/${currentBatchID}/funds/${selectedFundID}/transactions/${transactionID}/attachments/${attachmentID}/download`);
      window.open(response.data.url, '_blank', 'noopener,noreferrer');
    } catch (err) { toast.error(errMsg(err, 'Could not download receipt')); }
  };

  if (!currentBatchID) return <Card className="p-10"><Empty message="Select a cohort to view its funds." /></Card>;

  return (
    <div className="space-y-7">
      <div className="flex flex-col gap-5 xl:flex-row xl:items-end xl:justify-between">
        <div className="flex items-center gap-4">
          <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-primary-light text-primary"><BanknotesIcon className="h-8 w-8" /></div>
          <div><h1 className="text-2xl font-semibold text-ink">Cohort Funds</h1><p className="mt-1 text-[15px] text-muted">Transparent finances for {currentBatch()?.name || 'your cohort'}.</p></div>
        </div>
        <div className="flex flex-wrap gap-2">
          {canCreateFund && <button onClick={() => openFundForm('OTHER')} className="inline-flex items-center gap-2 rounded-xl border border-primary px-4 py-2.5 text-sm font-bold text-primary"><PlusIcon className="h-5 w-5" /> New general fund</button>}
          {canCreateFund && canManageEvents && <button onClick={() => openFundForm('EVENT')} className="inline-flex items-center gap-2 rounded-xl bg-primary px-4 py-2.5 text-sm font-bold text-slate-950"><CalendarDaysIcon className="h-5 w-5" /> Create event fund</button>}
          {canAssignManager && <button onClick={() => setTab('managers')} className="inline-flex items-center gap-2 rounded-xl border border-line px-4 py-2.5 text-sm font-medium text-ink"><UserGroupIcon className="h-5 w-5" /> Manage fund managers</button>}
          <button onClick={loadOverview} className="inline-flex items-center gap-2 rounded-xl border border-line px-4 py-2.5 text-sm text-muted hover:text-ink"><ArrowPathIcon className="h-5 w-5" /> Refresh</button>
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Metric label="Combined cash balance" value={formatMoney(totals.balance)} tone="text-primary" />
        <Metric label="Cash received" value={formatMoney(totals.income)} tone="text-emerald-400" />
        <Metric label="Expenses" value={formatMoney(totals.expense)} tone="text-rose-400" />
        <Metric label="Outstanding borrowing" value={formatMoney(totals.loans)} tone="text-violet-400" />
      </div>

      <div className="flex gap-2 overflow-x-auto border-b border-line pb-3">
        {([['funds', 'Funds & ledger'], ...(canAssignManager ? [['managers', 'Fund managers']] : []), ['transfers', 'Transfers & loans'], ['birthday', 'Birthday Fund']] as [Tab, string][]).map(([value, label]) => <button key={value} onClick={() => setTab(value)} className={`whitespace-nowrap rounded-xl px-4 py-2.5 text-sm font-medium ${tab === value ? 'bg-primary text-slate-950' : 'bg-charcoal-card text-muted hover:text-ink'}`}>{label}</button>)}
      </div>

      {showCreateFund && canCreateFund && <CreateFundForm form={fundForm} setForm={setFundForm} events={events} canManageEvents={canManageEvents} eventForm={eventForm} setEventForm={setEventForm} onSubmit={createFund} busy={busy} />}
      {loading ? <Card className="p-10 text-center text-sm text-muted">Loading financial records…</Card> : tab === 'funds' ? (
        <FundsTab funds={funds} selectedFund={selectedFund} setSelectedFundID={setSelectedFundID} managers={managers} canAssignManager={canAssignManager} openManagers={() => setTab('managers')} canManageSelected={canManageSelected} canCloseFund={canCloseFund} transition={(target) => run(() => api.post(`/v1/batches/${currentBatchID}/funds/${selectedFundID}/${target}`), `Fund ${target}d`)} showTransaction={showTransaction} setShowTransaction={setShowTransaction} transactionForm={transactionForm} setTransactionForm={setTransactionForm} createTransaction={createTransaction} includeDrafts={includeDrafts} setIncludeDrafts={setIncludeDrafts} transactionType={transactionType} setTransactionType={setTransactionType} transactions={transactions} postTransaction={(id) => run(() => api.post(`/v1/batches/${currentBatchID}/funds/${selectedFundID}/transactions/${id}/post`), 'Transaction posted', 'fund')} reverseTransaction={reverseTransaction} attachments={attachments} loadAttachments={loadAttachments} downloadAttachment={downloadAttachment} uploadReceipt={uploadReceipt} busy={busy} />
      ) : tab === 'managers' && canAssignManager ? (
        <FundManagersTab funds={funds} selectedFundID={selectedFundID} setSelectedFundID={setSelectedFundID} managers={managers} members={members} assignManager={assignManager} removeManager={(membershipID) => run(() => api.delete(`/v1/batches/${currentBatchID}/funds/${selectedFundID}/managers/${membershipID}`), 'Fund manager removed', 'fund')} busy={busy} />
      ) : tab === 'transfers' ? (
        <TransfersTab funds={funds} transfers={transfers} canTransfer={canTransfer} canReverse={canReverseTransfer} show={showTransfer} setShow={setShowTransfer} form={transferForm} setForm={setTransferForm} submit={createTransfer} repay={repay} reverse={(id) => run(() => api.post(`/v1/batches/${currentBatchID}/fund-transfers/${id}/reverse`), 'Transfer reversed')} busy={busy} />
      ) : (
        <BirthdayTab summary={birthday} periods={periods} selected={selectedPeriod} select={setSelectedPeriodID} own={myContribution} contributions={contributions} canManage={canManageBirthday} canRecord={canRecordContribution} form={periodForm} setForm={setPeriodForm} create={createPeriod} close={() => run(() => api.post(`/v1/batches/${currentBatchID}/birthday-fund/periods/${selectedPeriodID}/close`), 'Contribution period closed', 'period')} record={recordContribution} busy={busy} />
      )}
    </div>
  );
}

function Metric({ label, value, tone }: { label: string; value: string; tone: string }) {
  return <Card className="p-5"><p className="text-xs font-medium uppercase tracking-wider text-muted">{label}</p><p className={`mt-2 text-2xl font-semibold ${tone}`}>{value}</p></Card>;
}

type FundFormState = { name: string; description: string; type: string; event_link: string; event_id: string };

function CreateFundForm({ form, setForm, events, canManageEvents, eventForm, setEventForm, onSubmit, busy }: { form: FundFormState; setForm: Dispatch<SetStateAction<FundFormState>>; events: LmsEvent[]; canManageEvents: boolean; eventForm: FundingEventFormState; setEventForm: Dispatch<SetStateAction<FundingEventFormState>>; onSubmit: (e: FormEvent) => void; busy: boolean }) {
  return <Card className="p-6"><div><CardTitle>Create a fund</CardTitle><p className="mt-1 text-sm text-muted">Create a standalone fund, link an existing event, or create the event and its fund together.</p></div><form onSubmit={onSubmit} className="mt-5 grid gap-4 md:grid-cols-3"><Field label="Fund name"><input required className={inputCls} value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></Field><Field label="Type"><select className={inputCls} value={form.type} onChange={(e) => setForm({ ...form, type: e.target.value, event_link: e.target.value === 'EVENT' ? form.event_link : 'none', event_id: e.target.value === 'EVENT' ? form.event_id : '' })}><option value="OTHER">Other</option><option value="EVENT">Event fund</option></select></Field>{form.type === 'EVENT' && <Field label="Event relationship"><select className={inputCls} value={form.event_link} onChange={(e) => setForm({ ...form, event_link: e.target.value, event_id: '' })}><option value="none">No linked event</option><option value="existing">Link existing event</option>{canManageEvents && <option value="new">Create a new event</option>}</select></Field>}{form.type === 'EVENT' && form.event_link === 'existing' && <div className="md:col-span-3"><Field label="Existing event"><select required className={inputCls} value={form.event_id} onChange={(e) => setForm({ ...form, event_id: e.target.value })}><option value="">Select an event…</option>{events.filter((item) => item.status !== 'ARCHIVED').map((item) => <option key={item.id} value={item.id}>{item.title} · {fmtDateTime(item.starts_at)}</option>)}</select></Field>{events.length === 0 && <p className="mt-2 text-xs text-muted">No available events. Choose “Create a new event” instead.</p>}</div>}{form.type === 'EVENT' && form.event_link === 'new' && <div className="md:col-span-3 grid gap-4 rounded-2xl border border-line bg-surface p-5 md:grid-cols-2"><div className="md:col-span-2"><p className="font-medium text-ink">New funding event</p><p className="mt-1 text-xs text-muted">The event and fund are saved together. If either fails, neither is created.</p></div><Field label="Event title"><input required className={inputCls} value={eventForm.title} onChange={(e) => setEventForm({ ...eventForm, title: e.target.value })} /></Field><Field label="Location / meeting link"><input className={inputCls} value={eventForm.location} onChange={(e) => setEventForm({ ...eventForm, location: e.target.value })} /></Field><Field label="Starts at"><input required type="datetime-local" className={inputCls} value={eventForm.starts_at} onChange={(e) => setEventForm({ ...eventForm, starts_at: e.target.value })} /></Field><Field label="Ends at"><input type="datetime-local" className={inputCls} value={eventForm.ends_at} onChange={(e) => setEventForm({ ...eventForm, ends_at: e.target.value })} /></Field><Field label="Visibility"><select className={inputCls} value={eventForm.visibility} onChange={(e) => setEventForm({ ...eventForm, visibility: e.target.value })}><option value="MEMBERS_ONLY">Members only</option><option value="PUBLIC">Public</option></select></Field><div className="md:col-span-2"><Field label="Event description"><textarea rows={2} className={inputCls} value={eventForm.description} onChange={(e) => setEventForm({ ...eventForm, description: e.target.value })} /></Field></div></div>}<div className="md:col-span-3"><Field label="Fund description"><textarea rows={2} className={inputCls} value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} /></Field></div><button disabled={busy} className="rounded-xl bg-primary px-5 py-2.5 text-sm font-bold text-slate-950 disabled:opacity-50">{form.type === 'EVENT' && form.event_link === 'new' ? 'Create event and fund' : 'Create fund'}</button></form></Card>;
}

type FundsTabProps = {
  funds: FundSummary[]; selectedFund: FundSummary | null; setSelectedFundID: (id: string) => void; managers: FundManager[]; canAssignManager: boolean; openManagers: () => void; canManageSelected: boolean; canCloseFund: boolean; transition: (target: 'close' | 'archive') => void; showTransaction: boolean; setShowTransaction: (v: boolean) => void; transactionForm: TransactionFormState; setTransactionForm: Dispatch<SetStateAction<TransactionFormState>>; createTransaction: (e: FormEvent) => void; includeDrafts: boolean; setIncludeDrafts: (v: boolean) => void; transactionType: string; setTransactionType: (v: string) => void; transactions: FundTransaction[]; postTransaction: (id: string) => void; reverseTransaction: (t: FundTransaction) => void; attachments: Record<string, FundAttachment[]>; loadAttachments: (id: string) => void; downloadAttachment: (tid: string, aid: string) => void; uploadReceipt: (t: FundTransaction, f?: File) => void; busy: boolean;
};

function FundsTab(props: FundsTabProps) {
  const f = props.selectedFund;
  if (props.funds.length === 0) return <Card className="p-10"><Empty message="No funds are available." /></Card>;
  return <div className="space-y-6">
    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">{props.funds.map((item) => <button key={item.id} onClick={() => props.setSelectedFundID(item.id)} className={`rounded-2xl border p-5 text-left transition ${item.id === f?.id ? 'border-primary bg-primary-light shadow-gold-glow' : 'border-line bg-charcoal-card hover:border-primary/60'}`}><div className="flex items-start justify-between gap-3"><div><p className="font-semibold text-ink">{item.name}</p><p className="mt-1 text-xs text-muted">{human(item.type)}</p></div><Badge tone={statusTone(item.status)}>{item.status}</Badge></div><p className="mt-5 text-2xl font-semibold text-ink">{formatMoney(item.balance_minor)}</p><div className="mt-4 grid grid-cols-2 gap-2 text-xs text-muted"><span>Income {formatMoney(item.total_cash_in_minor)}</span><span>Expenses {formatMoney(item.total_expense_minor)}</span></div></button>)}</div>
    {f && <>
      <Card className="p-6"><div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between"><div><div className="flex items-center gap-3"><CardTitle>{f.name}</CardTitle><Badge tone={statusTone(f.status)}>{f.status}</Badge></div><p className="mt-2 max-w-2xl text-sm text-muted">{f.description || 'No description provided.'}</p><p className="mt-3 text-xs text-muted">Created {fmtDate(f.created_at)}{f.closed_at ? ` · closed ${fmtDate(f.closed_at)}` : ''}</p></div><div className="flex flex-wrap gap-2">{props.canManageSelected && <button onClick={() => props.setShowTransaction(!props.showTransaction)} className="rounded-xl bg-primary px-4 py-2 text-sm font-bold text-slate-950">+ Record transaction</button>}{props.canCloseFund && f.status === 'ACTIVE' && <button onClick={() => props.transition('close')} className="rounded-xl border border-line px-4 py-2 text-sm text-muted hover:text-ink">Close fund</button>}{props.canCloseFund && f.status === 'CLOSED' && <button onClick={() => props.transition('archive')} className="rounded-xl border border-line px-4 py-2 text-sm text-muted hover:text-ink">Archive</button>}</div></div>
        <div className="mt-6 border-t border-line pt-5"><div className="flex items-center justify-between gap-3"><div><div className="flex items-center gap-2"><UserGroupIcon className="h-5 w-5 text-primary" /><p className="text-sm font-medium text-ink">Fund managers</p></div><p className="mt-1 text-xs text-muted">Managers can record and reverse transactions for this fund only.</p></div>{props.canAssignManager && <button onClick={props.openManagers} className="rounded-xl border border-primary px-4 py-2 text-sm font-medium text-primary">Manage assignments</button>}</div><div className="mt-3 flex flex-wrap items-center gap-2">{props.managers.length === 0 && <span className="text-sm text-muted">No fund-specific managers.</span>}{props.managers.map((m) => <span key={m.id} className="inline-flex items-center gap-2 rounded-lg bg-surface px-3 py-2 text-sm text-ink">{m.display_name}</span>)}</div></div>
      </Card>
      {props.showTransaction && props.canManageSelected && <TransactionForm form={props.transactionForm} setForm={props.setTransactionForm} submit={props.createTransaction} busy={props.busy} />}
      <Card className="overflow-hidden"><div className="flex flex-col gap-3 border-b border-line p-5 md:flex-row md:items-center md:justify-between"><CardTitle>Ledger</CardTitle><div className="flex flex-wrap items-center gap-3"><select className={`${inputCls} w-auto min-w-44`} value={props.transactionType} onChange={(e) => props.setTransactionType(e.target.value)}><option value="">All transaction types</option>{['CASH_IN','EXPENSE','TRANSFER_IN','TRANSFER_OUT','REVERSAL_IN','REVERSAL_OUT'].map((v) => <option key={v} value={v}>{human(v)}</option>)}</select>{props.canManageSelected && <label className="flex items-center gap-2 text-sm text-muted"><input type="checkbox" checked={props.includeDrafts} onChange={(e) => props.setIncludeDrafts(e.target.checked)} /> Include drafts</label>}</div></div>
        {props.transactions.length === 0 ? <Empty message="No transactions match this view." /> : <div className="overflow-x-auto"><table className="w-full min-w-[900px] text-left text-sm"><thead className="bg-surface text-xs uppercase tracking-wider text-muted"><tr><th className="px-5 py-3">Date</th><th className="px-5 py-3">Description</th><th className="px-5 py-3">Type</th><th className="px-5 py-3">Source</th><th className="px-5 py-3 text-right">Amount</th><th className="px-5 py-3">Actions</th></tr></thead><tbody>{props.transactions.map((t) => <tr key={t.id} className="border-t border-line"><td className="px-5 py-4 text-muted">{fmtDateTime(t.transaction_date)}</td><td className="px-5 py-4"><p className="font-medium text-ink">{t.description}</p><p className="text-xs text-muted">{t.category || t.reference || '—'}</p></td><td className="px-5 py-4"><Badge tone={t.status === 'DRAFT' ? 'orange' : 'blue'}>{human(t.type)}{t.status === 'DRAFT' ? ' · draft' : ''}</Badge></td><td className="px-5 py-4 text-muted">{t.source_name || (t.source_type ? human(t.source_type) : '—')}</td><td className={`px-5 py-4 text-right font-semibold ${positiveTypes.has(t.type) ? 'text-emerald-400' : 'text-rose-400'}`}>{positiveTypes.has(t.type) ? '+' : '−'}{formatMoney(t.amount_minor)}</td><td className="px-5 py-4"><div className="flex items-center gap-2">{props.canManageSelected && t.status === 'DRAFT' && <button onClick={() => props.postTransaction(t.id)} className="text-xs font-medium text-emerald-400">Post</button>}{props.canManageSelected && t.status === 'POSTED' && !t.transfer_id && !t.reversed_by_transaction_id && !t.reversal_of_transaction_id && <button onClick={() => props.reverseTransaction(t)} className="text-xs font-medium text-rose-400">Reverse</button>}<button onClick={() => props.loadAttachments(t.id)} className="text-xs text-primary">Receipts</button>{props.canManageSelected && <label className="cursor-pointer text-primary" title="Attach PDF receipt"><DocumentArrowUpIcon className="h-5 w-5" /><input type="file" accept="application/pdf,.pdf" className="hidden" onChange={(e) => props.uploadReceipt(t, e.target.files?.[0])} /></label>}</div>{props.attachments[t.id] && <div className="mt-2 space-y-1">{(props.attachments[t.id] ?? []).length === 0 ? <span className="text-xs text-muted">No receipts</span> : (props.attachments[t.id] ?? []).map((a) => <button key={a.id} onClick={() => props.downloadAttachment(t.id, a.id)} className="flex items-center gap-1 text-xs text-primary hover:underline"><ArrowDownTrayIcon className="h-3.5 w-3.5" />{a.file_name}</button>)}</div>}</td></tr>)}</tbody></table></div>}
      </Card>
    </>}
  </div>;
}

function FundManagersTab({ funds, selectedFundID, setSelectedFundID, managers, members, assignManager, removeManager, busy }: { funds: FundSummary[]; selectedFundID: string; setSelectedFundID: (id: string) => void; managers: FundManager[]; members: Member[]; assignManager: (id: string) => Promise<boolean>; removeManager: (id: string) => void; busy: boolean }) {
  const [candidate, setCandidate] = useState('');
  const selectedFund = funds.find((fund) => fund.id === selectedFundID);
  const availableMembers = members.filter((member) => member.status === 'ACTIVE' && !managers.some((manager) => manager.membership_id === member.id));

  useEffect(() => { setCandidate(''); }, [selectedFundID]);

  if (funds.length === 0) return <Card className="p-10"><Empty message="Create a fund before assigning fund managers." /></Card>;
  return <div className="grid gap-6 xl:grid-cols-[340px_1fr]"><Card className="p-5"><div className="flex items-center gap-2"><UserGroupIcon className="h-6 w-6 text-primary" /><CardTitle>Fund managers</CardTitle></div><p className="mt-2 text-sm text-muted">Choose a fund, then assign active members from this cohort. Each assignment applies only to that fund.</p><div className="mt-5 space-y-2">{funds.map((fund) => <button key={fund.id} onClick={() => setSelectedFundID(fund.id)} className={`w-full rounded-xl border p-4 text-left ${fund.id === selectedFundID ? 'border-primary bg-primary-light' : 'border-line bg-surface hover:border-primary/60'}`}><div className="flex items-center justify-between gap-3"><span className="font-medium text-ink">{fund.name}</span><Badge tone={statusTone(fund.status)}>{fund.status}</Badge></div><p className="mt-1 text-xs text-muted">{human(fund.type)}</p></button>)}</div></Card><div className="space-y-6"><Card className="p-6"><CardTitle>Assign manager to {selectedFund?.name || 'fund'}</CardTitle><p className="mt-2 text-sm text-muted">The selected member will be able to manage transactions for this fund without receiving authority over other funds.</p><div className="mt-5 flex flex-col gap-3 sm:flex-row"><select aria-label="Select fund manager" className={inputCls} value={candidate} onChange={(event) => setCandidate(event.target.value)}><option value="">Select an active cohort member…</option>{availableMembers.map((member) => <option key={member.id} value={member.id}>{member.display_name}{member.roles.length ? ` · ${member.roles.map(human).join(', ')}` : ''}</option>)}</select><button type="button" disabled={!candidate || busy} onClick={async () => { if (await assignManager(candidate)) setCandidate(''); }} className="whitespace-nowrap rounded-xl bg-primary px-5 py-2.5 text-sm font-bold text-slate-950 disabled:opacity-50">Add fund manager</button></div>{availableMembers.length === 0 && <p className="mt-3 text-xs text-muted">Every active cohort member is already assigned to this fund.</p>}</Card><Card className="overflow-hidden"><div className="border-b border-line p-5"><CardTitle>Current managers</CardTitle></div>{managers.length === 0 ? <Empty message="No managers are assigned to this fund." /> : <div className="divide-y divide-line">{managers.map((manager) => <div key={manager.id} className="flex items-center justify-between gap-4 p-5"><div><p className="font-medium text-ink">{manager.display_name}</p><p className="mt-1 text-xs text-muted">Assigned {fmtDate(manager.assigned_at)}</p></div><button disabled={busy} onClick={() => removeManager(manager.membership_id)} className="rounded-lg border border-rose-500/40 px-3 py-2 text-xs font-medium text-rose-400 disabled:opacity-50">Remove manager</button></div>)}</div>}</Card></div></div>;
}

function TransactionForm({ form, setForm, submit, busy }: { form: TransactionFormState; setForm: Dispatch<SetStateAction<TransactionFormState>>; submit: (e: FormEvent) => void; busy: boolean }) {
  const change = (key: keyof TransactionFormState, value: string) => setForm({ ...form, [key]: value });
  return <Card className="p-6"><CardTitle>Record cash or expense</CardTitle><form onSubmit={submit} className="mt-5 grid gap-4 md:grid-cols-3"><Field label="Transaction type"><select className={inputCls} value={form.type} onChange={(e) => change('type', e.target.value)}><option value="CASH_IN">Cash in</option><option value="EXPENSE">Expense</option></select></Field><Field label="Amount (LKR)"><input required type="number" min="0.01" step="0.01" className={inputCls} value={form.amount} onChange={(e) => change('amount', e.target.value)} /></Field><Field label="Posting"><select className={inputCls} value={form.status} onChange={(e) => change('status', e.target.value)}><option value="POSTED">Post now</option><option value="DRAFT">Save draft</option></select></Field><div className="md:col-span-2"><Field label="Description"><input required className={inputCls} value={form.description} onChange={(e) => change('description', e.target.value)} /></Field></div><Field label="Category"><input className={inputCls} value={form.category} onChange={(e) => change('category', e.target.value)} placeholder="Transport, printing…" /></Field><Field label="Source type"><input className={inputCls} value={form.source_type} onChange={(e) => change('source_type', e.target.value)} placeholder="Sponsor, donation…" /></Field><Field label="Source name"><input className={inputCls} value={form.source_name} onChange={(e) => change('source_name', e.target.value)} /></Field><Field label="Reference"><input className={inputCls} value={form.reference} onChange={(e) => change('reference', e.target.value)} /></Field><button disabled={busy} className="rounded-xl bg-primary px-5 py-2.5 text-sm font-bold text-slate-950 disabled:opacity-50">Save transaction</button></form></Card>;
}

function TransfersTab({ funds, transfers, canTransfer, canReverse, show, setShow, form, setForm, submit, repay, reverse, busy }: { funds: FundSummary[]; transfers: FundTransfer[]; canTransfer: boolean; canReverse: boolean; show: boolean; setShow: (v: boolean) => void; form: { from_fund_id: string; to_fund_id: string; type: string; amount: string; description: string }; setForm: (v: typeof form) => void; submit: (e: FormEvent) => void; repay: (t: FundTransfer) => void; reverse: (id: string) => void; busy: boolean }) {
  return <div className="space-y-6"><div className="flex justify-end">{canTransfer && <button onClick={() => setShow(!show)} className="inline-flex items-center gap-2 rounded-xl bg-primary px-4 py-2.5 text-sm font-bold text-slate-950"><ArrowsRightLeftIcon className="h-5 w-5" /> New transfer</button>}</div>{show && canTransfer && <Card className="p-6"><CardTitle>Transfer between funds</CardTitle><form onSubmit={submit} className="mt-5 grid gap-4 md:grid-cols-2 xl:grid-cols-5"><Field label="From"><select required className={inputCls} value={form.from_fund_id} onChange={(e) => setForm({ ...form, from_fund_id: e.target.value })}><option value="">Select source…</option>{funds.filter((f) => f.status === 'ACTIVE').map((f) => <option key={f.id} value={f.id}>{f.name}</option>)}</select></Field><Field label="To"><select required className={inputCls} value={form.to_fund_id} onChange={(e) => setForm({ ...form, to_fund_id: e.target.value })}><option value="">Select destination…</option>{funds.filter((f) => f.status === 'ACTIVE' && f.id !== form.from_fund_id).map((f) => <option key={f.id} value={f.id}>{f.name}</option>)}</select></Field><Field label="Purpose"><select className={inputCls} value={form.type} onChange={(e) => setForm({ ...form, type: e.target.value })}><option value="TRANSFER">Permanent transfer</option><option value="LOAN">Internal loan</option></select></Field><Field label="Amount (LKR)"><input required type="number" min="0.01" step="0.01" className={inputCls} value={form.amount} onChange={(e) => setForm({ ...form, amount: e.target.value })} /></Field><Field label="Description"><input className={inputCls} value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} /></Field><button disabled={busy} className="rounded-xl bg-primary px-5 py-2.5 text-sm font-bold text-slate-950 disabled:opacity-50">Complete transfer</button></form></Card>}
    <Card className="overflow-hidden">{transfers.length === 0 ? <Empty message="No transfers or loans have been recorded." /> : <div className="overflow-x-auto"><table className="w-full min-w-[850px] text-left text-sm"><thead className="bg-surface text-xs uppercase tracking-wider text-muted"><tr><th className="px-5 py-3">Date</th><th className="px-5 py-3">Movement</th><th className="px-5 py-3">Purpose</th><th className="px-5 py-3 text-right">Amount</th><th className="px-5 py-3 text-right">Outstanding</th><th className="px-5 py-3">Actions</th></tr></thead><tbody>{transfers.map((t) => <tr key={t.id} className="border-t border-line"><td className="px-5 py-4 text-muted">{fmtDate(t.created_at)}</td><td className="px-5 py-4"><p className="font-medium text-ink">{t.from_fund_name} → {t.to_fund_name}</p><p className="text-xs text-muted">{t.description || 'No description'}</p></td><td className="px-5 py-4"><Badge tone={t.type === 'LOAN' ? 'purple' : t.type === 'REVERSAL' ? 'orange' : 'blue'}>{human(t.type)}</Badge></td><td className="px-5 py-4 text-right font-semibold text-ink">{formatMoney(t.amount_minor)}</td><td className="px-5 py-4 text-right text-muted">{t.type === 'LOAN' ? formatMoney(t.outstanding_minor) : '—'}</td><td className="px-5 py-4"><div className="flex gap-3">{canTransfer && t.type === 'LOAN' && (t.outstanding_minor || 0) > 0 && !t.reversed_by_transfer_id && <button onClick={() => repay(t)} className="text-xs font-medium text-emerald-400">Repay</button>}{canReverse && !t.reversed_by_transfer_id && t.type !== 'REVERSAL' && <button onClick={() => reverse(t.id)} className="text-xs font-medium text-rose-400">Reverse</button>}{t.reversed_by_transfer_id && <span className="text-xs text-muted">Reversed</span>}</div></td></tr>)}</tbody></table></div>}</Card></div>;
}

function BirthdayTab({ summary, periods, selected, select, own, contributions, canManage, canRecord, form, setForm, create, close, record, busy }: { summary: BirthdaySummary | null; periods: BirthdayPeriod[]; selected: BirthdayPeriod | null; select: (id: string) => void; own: BirthdayContribution | null; contributions: BirthdayContribution[]; canManage: boolean; canRecord: boolean; form: { year: string; month: string; amount: string }; setForm: (v: typeof form) => void; create: (e: FormEvent) => void; close: () => void; record: (c: BirthdayContribution) => void; busy: boolean }) {
  return <div className="space-y-6">{summary && <div className="grid gap-4 md:grid-cols-4"><Metric label="Birthday Fund balance" value={formatMoney(summary.balance_minor)} tone="text-primary" /><Metric label="Expected" value={formatMoney(summary.total_expected_minor)} tone="text-ink" /><Metric label="Collected" value={formatMoney(summary.total_collected_minor)} tone="text-emerald-400" /><Metric label="Outstanding" value={formatMoney(summary.outstanding_minor)} tone="text-rose-400" /></div>}
    {canManage && <Card className="p-6"><div className="flex items-center gap-2"><CalendarDaysIcon className="h-5 w-5 text-primary" /><CardTitle>Create monthly contribution period</CardTitle></div><form onSubmit={create} className="mt-5 grid gap-4 sm:grid-cols-4"><Field label="Year"><input required type="number" min="2000" max="2200" className={inputCls} value={form.year} onChange={(e) => setForm({ ...form, year: e.target.value })} /></Field><Field label="Month"><select className={inputCls} value={form.month} onChange={(e) => setForm({ ...form, month: e.target.value })}>{Array.from({ length: 12 }, (_, i) => <option key={i + 1} value={i + 1}>{new Date(2026, i).toLocaleString(undefined, { month: 'long' })}</option>)}</select></Field><Field label="Expected per student (LKR)"><input required type="number" min="0.01" step="0.01" className={inputCls} value={form.amount} onChange={(e) => setForm({ ...form, amount: e.target.value })} /></Field><button disabled={busy} className="self-end rounded-xl bg-primary px-5 py-2.5 text-sm font-bold text-slate-950 disabled:opacity-50">Create period</button></form></Card>}
    <div className="grid gap-6 xl:grid-cols-[320px_1fr]"><Card className="p-4"><p className="px-2 pb-3 text-sm font-medium text-ink">Contribution periods</p><div className="space-y-2">{periods.length === 0 ? <Empty message="No contribution periods yet." /> : periods.map((p) => <button key={p.id} onClick={() => select(p.id)} className={`w-full rounded-xl border p-4 text-left ${selected?.id === p.id ? 'border-primary bg-primary-light' : 'border-line bg-surface'}`}><div className="flex items-center justify-between"><p className="font-medium text-ink">{new Date(p.year, p.month - 1).toLocaleString(undefined, { month: 'long', year: 'numeric' })}</p><Badge tone={statusTone(p.status)}>{p.status}</Badge></div><p className="mt-2 text-xs text-muted">{p.member_count} students · {formatMoney(p.collected_minor)} collected</p></button>)}</div></Card>
      <div className="space-y-5">{selected ? <><Card className="p-6"><div className="flex items-start justify-between gap-4"><div><CardTitle>{new Date(selected.year, selected.month - 1).toLocaleString(undefined, { month: 'long', year: 'numeric' })}</CardTitle><p className="mt-1 text-sm text-muted">Expected {formatMoney(selected.amount_minor)} per student</p></div>{canManage && selected.status === 'OPEN' && <button onClick={close} className="rounded-xl border border-line px-4 py-2 text-sm text-muted hover:text-ink">Close period</button>}</div><div className="mt-5 grid grid-cols-3 gap-3"><MiniMetric label="Students" value={String(selected.member_count)} /><MiniMetric label="Expected" value={formatMoney(selected.expected_minor)} /><MiniMetric label="Collected" value={formatMoney(selected.collected_minor)} /></div></Card>
        <Card className="p-6"><CardTitle>My contribution</CardTitle>{own ? <div className="mt-4 flex flex-wrap items-center justify-between gap-4"><div><p className="text-2xl font-semibold text-ink">{formatMoney(own.paid_minor)} <span className="text-sm font-normal text-muted">of {formatMoney(own.expected_minor)}</span></p><p className="mt-1 text-sm text-muted">Outstanding {formatMoney(own.outstanding_minor)}</p></div><Badge tone={own.status === 'PAID' ? 'green' : own.status === 'PARTIAL' ? 'orange' : 'red'}>{own.status}</Badge></div> : <p className="mt-4 text-sm text-muted">You are not included in this period's membership snapshot.</p>}</Card>
        {canManage && <Card className="overflow-hidden"><div className="border-b border-line p-5"><CardTitle>Student contribution matrix</CardTitle><p className="mt-1 text-sm text-muted">Visible only to authorized Birthday Fund administrators.</p></div>{contributions.length === 0 ? <Empty message="No contribution obligations in this period." /> : <div className="overflow-x-auto"><table className="w-full min-w-[650px] text-left text-sm"><thead className="bg-surface text-xs uppercase tracking-wider text-muted"><tr><th className="px-5 py-3">Student</th><th className="px-5 py-3 text-right">Expected</th><th className="px-5 py-3 text-right">Paid</th><th className="px-5 py-3">Status</th><th className="px-5 py-3">Action</th></tr></thead><tbody>{contributions.map((c) => <tr key={c.id} className="border-t border-line"><td className="px-5 py-4 font-medium text-ink">{c.display_name}</td><td className="px-5 py-4 text-right text-muted">{formatMoney(c.expected_minor)}</td><td className="px-5 py-4 text-right text-emerald-400">{formatMoney(c.paid_minor)}</td><td className="px-5 py-4"><Badge tone={c.status === 'PAID' ? 'green' : c.status === 'PARTIAL' ? 'orange' : 'red'}>{c.status}</Badge></td><td className="px-5 py-4">{canRecord && selected.status === 'OPEN' && c.outstanding_minor > 0 && <button onClick={() => record(c)} className="text-xs font-medium text-primary">Record payment</button>}</td></tr>)}</tbody></table></div>}</Card>}
      </> : <Card className="p-10"><Empty message="Select a contribution period." /></Card>}</div></div>
  </div>;
}

function MiniMetric({ label, value }: { label: string; value: string }) { return <div className="rounded-xl bg-surface p-4"><p className="text-xs text-muted">{label}</p><p className="mt-1 font-semibold text-ink">{value}</p></div>; }
