export interface User {
  id: string;
  student_number: string;
  combination: string | null;
  first_name: string;
  last_name: string;
  display_name: string;
  email: string;
  phone_number: string | null;
  status: string;
}

export interface AuthTokens {
  access_token: string;
  refresh_token: string;
}

export interface AccessMembership {
  membership_id: string;
  batch_id: string;
  batch_name: string;
  batch_slug: string;
  entry_year: number;
  status: string;
  roles: string[];
  permissions: string[];
}

export interface AccessContext {
  platform_roles: string[];
  platform_permissions: string[];
  memberships: AccessMembership[];
}

export interface Batch {
  id: string;
  name: string;
  slug: string;
  entry_year: number;
  graduation_year: number | null;
  description: string;
  status: string;
}

export interface BatchProfile {
  headline: string | null;
  about_text: string | null;
  mission_text: string | null;
  contact_email: string | null;
}

export interface Semester {
  id: string;
  batch_id: string;
  semester_number: number;
  name: string;
  academic_year: string;
  starts_at: string | null;
  ends_at: string | null;
  is_current: boolean;
  created_at: string;
  updated_at: string;
}

export interface Module {
  id: string;
  batch_id: string;
  semester_id: string;
  module_code: string;
  name: string;
  description: string;
  lecturer_name: string | null;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface Lesson {
  id: string;
  batch_id: string;
  module_id: string;
  title: string;
  description: string;
  youtube_video_id: string;
  lesson_date: string | null;
  duration_seconds: number | null;
  status: string;
  published_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface LinkItem {
  id: string;
  batch_id: string;
  module_id: string | null;
  title: string;
  url: string;
  description: string;
  category: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface Resource {
  id: string;
  batch_id: string;
  module_id: string;
  module_code: string;
  module_name: string;
  type: string;
  title: string;
  description: string;
  academic_year: string | null;
  exam_type: string | null;
  status: string;
  published_at: string | null;
  created_at: string;
  updated_at: string;
  version_id: string;
  version_number: number;
  file_name: string;
  mime_type: string;
  size_bytes: number;
  original_size_bytes: number;
  compression_profile: string;
  page_count: number | null;
}

export type ResourceType =
  | 'LECTURE_NOTE'
  | 'HANDWRITTEN_NOTE'
  | 'PAST_PAPER'
  | 'TUTORIAL'
  | 'ASSIGNMENT'
  | 'REFERENCE'
  | 'OTHER';

export interface Announcement {
  id: string;
  batch_id: string;
  module_id: string | null;
  title: string;
  body: string;
  priority: string;
  pinned: boolean;
  expires_at: string | null;
  status: string;
  published_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface LmsEvent {
  id: string;
  batch_id: string;
  title: string;
  description: string;
  location: string | null;
  starts_at: string;
  ends_at: string | null;
  visibility: string;
  status: string;
  published_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface Member {
  id: string;
  user_id: string;
  status: string;
  joined_at: string;
  display_name: string;
  combination: string | null;
  roles: string[];
}

export interface RoleCatalogEntry {
  code: string;
  name: string;
}

export interface AdminUser {
  id: string;
  student_number: string;
  combination: string | null;
  first_name: string;
  last_name: string;
  display_name: string;
  email: string;
  status: string;
  created_at: string;
  platform_roles: string[];
  batch_roles: AdminBatchRole[];
}

export interface AdminBatchRole {
  batch_id: string;
  batch_name: string;
  code: string;
}

export interface PlatformRole {
  code: string;
  name: string;
  scope: 'PLATFORM' | 'BATCH';
}

export interface AdminBatch {
  id: string;
  name: string;
  slug: string;
  entry_year: number;
}

export interface NotificationItem {
  id: string;
  batch_id: string | null;
  type: string;
  title: string;
  body: string;
  entity_id: string | null;
  read_at: string | null;
  created_at: string;
}

export interface AuditLog {
  id: string;
  batch_id: string | null;
  actor_user_id: string | null;
  action: string;
  entity_type: string;
  entity_id: string | null;
  new_values: Record<string, unknown> | null;
  created_at: string;
}

export interface GalleryImage {
  id: string;
  title: string;
  caption: string;
  alt_text: string;
  width: number;
  height: number;
  taken_at: string | null;
  published_at: string | null;
  display_url: string;
  thumbnail_url: string;
  status?: string;
}

export interface Complaint {
  id: string;
  submitted_by: string | null;
  category: string;
  subject: string;
  message: string;
  is_anonymous: boolean;
  status: string;
  assigned_to: string | null;
  resolved_at: string | null;
  created_at: string;
}

export interface FeedbackItem {
  id: string;
  category: string;
  message: string;
  is_anonymous: boolean;
  rating: number | null;
  status: string;
  created_at: string;
}

export interface Position {
  id: string;
  title: string;
  description: string;
  sort_order: number;
  is_public: boolean;
}

export interface PublicBatch {
  id: string;
  name: string;
  slug: string;
  description: string;
  hero_image_url: string | null;
  entry_year: number;
  graduation_year: number | null;
  headline: string | null;
  about_text: string | null;
  mission_text: string | null;
  contact_email: string | null;
}

export interface Session {
  id: string;
  created_at: string;
  last_used_at: string;
  expires_at: string;
  user_agent: string;
}

export interface ErrorResponse {
  error: {
    code: string;
    message: string;
    request_id: string;
  };
}

export interface FundSummary {
  id: string;
  batch_id: string;
  name: string;
  description: string;
  type: 'BIRTHDAY' | 'EVENT' | 'OTHER';
  event_id: string | null;
  currency: 'LKR';
  status: 'ACTIVE' | 'CLOSED' | 'ARCHIVED';
  balance_minor: number;
  total_cash_in_minor: number;
  total_expense_minor: number;
  total_transfer_in_minor: number;
  total_transfer_out_minor: number;
  outstanding_loans_payable_minor: number;
  created_at: string;
  closed_at: string | null;
}

export interface FundTransaction {
  id: string;
  fund_id: string;
  type: 'CASH_IN' | 'EXPENSE' | 'TRANSFER_IN' | 'TRANSFER_OUT' | 'REVERSAL_IN' | 'REVERSAL_OUT';
  amount_minor: number;
  description: string;
  category: string | null;
  source_type: string | null;
  source_name: string | null;
  reference: string | null;
  transaction_date: string;
  status: 'DRAFT' | 'POSTED';
  transfer_id: string | null;
  reversal_of_transaction_id: string | null;
  reversed_by_transaction_id: string | null;
  created_at: string;
  posted_at: string | null;
}

export interface FundManager {
  id: string;
  membership_id: string;
  user_id: string;
  display_name: string;
  assigned_at: string;
}

export interface FundTransfer {
  id: string;
  from_fund_id: string;
  from_fund_name: string;
  to_fund_id: string;
  to_fund_name: string;
  type: 'TRANSFER' | 'LOAN' | 'LOAN_REPAYMENT' | 'REVERSAL';
  amount_minor: number;
  description: string;
  parent_transfer_id: string | null;
  reversal_of_transfer_id: string | null;
  reversed_by_transfer_id: string | null;
  outstanding_minor: number | null;
  created_at: string;
}

export interface BirthdaySummary {
  fund_id: string;
  name: string;
  currency: 'LKR';
  balance_minor: number;
  total_expected_minor: number;
  total_collected_minor: number;
  outstanding_minor: number;
}

export interface BirthdayPeriod {
  id: string;
  year: number;
  month: number;
  amount_minor: number;
  status: 'OPEN' | 'CLOSED';
  member_count: number;
  expected_minor: number;
  collected_minor: number;
  created_at: string;
  closed_at: string | null;
}

export interface BirthdayContribution {
  id: string;
  membership_id?: string;
  user_id?: string;
  display_name?: string;
  expected_minor: number;
  paid_minor: number;
  outstanding_minor: number;
  status: 'PAID' | 'PARTIAL' | 'UNPAID';
}

export interface FundAttachment {
  id: string;
  file_name: string;
  mime_type: string;
  size_bytes: number;
  visibility: 'MEMBERS' | 'MANAGERS';
  created_at: string;
}
