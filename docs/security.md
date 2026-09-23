# Security model

Identity, membership, permissions, public position and platform administration are separate concepts. JWTs authenticate a user/session; database lookups authorize current operations. Account/session revocation is checked on every authenticated request. Membership and role changes therefore apply without waiting for JWT expiry.

## Permission bundles

All active members receive STUDENT. Additional columns below describe additive bundles; a representative remains a student.

| Role | Permissions |
|---|---|
| STUDENT | batch.view, membership.view, semester.view, module.view, announcement.view, resource.view, kuppi.view, link.view, event.view, position.view, complaint.create, complaint.view_own, feedback.create, fund.view |
| BATCH_REP | batch.manage, batch.profile.manage, membership.manage, role.assign, semester.manage, module.manage, announcement.create/update/delete/publish, resource.create/update/delete/publish, kuppi.view/create/update/publish/archive, link.manage, event.manage, position.manage, feedback.view/manage, audit.view, fund.create/update/close, fund.transaction.create/reverse, fund.manager.assign/remove, fund.transfer.create/reverse, birthday_fund.manage, birthday_contribution.record |
| CONTENT_MANAGER | announcement.create/update/delete/publish, resource.create/update/delete/publish, link.manage, event.manage, position.manage, batch.profile.manage |
| ACADEMIC_REP | semester.manage, module.manage, announcement.create/update/delete/publish, resource.create/update/delete/publish, kuppi.view/create/update/publish/archive, link.manage |
| COMPLAINT_MANAGER | complaint.view_all/respond/resolve |
| PLATFORM_ADMIN | batch.create, batch.archive, platform_user.manage, gallery.manage, platform_audit.view |

`x/y` notation expands to separate permission codes. Catalog definitions and scope constraints live in migration 000002. There is no numeric role precedence. Only batch-assigned roles satisfy batch permissions. Public titles grant no permissions. Role assignment allows only BATCH roles and never accepts platform promotion from a batch endpoint.

Fund-manager assignments are scoped capabilities outside the role catalog. They allow an active member to operate only the referenced fund and do not alter that member's roles. A two-fund transfer requires authority for both funds. Contribution administration remains restricted to its explicit batch permissions because it exposes individual payment status.

## Isolation and transactions

Private repositories include both route batch and entity identifiers. Composite foreign keys prevent cross-batch modules, semesters, resources, versions, positions, complaint messages, funds, financial transactions, managers, transfers, contribution obligations and receipts. Batch write operations lock their cohort to serialize membership/role mutations with protected domain changes. Financial posting additionally locks affected fund rows, in stable UUID order for two-fund operations, before deriving available balances. Unrelated cohorts do not share those locks.

The STUDENT invariant is checked at transaction commit. Assignment/removal and their audits commit together. Verification consumes the token with the account update. Refresh rotation locks account then session; replay revocation is committed before returning an authentication failure. Login rechecks the password hash under the account lock to prevent a concurrent reset issuing an old-password session.

Database RLS is not enabled in V1. Application authorization, scoped SQL and constraints are the primary controls. Future RLS must use transaction-local context, fail closed with missing context, and run without owner/superuser/BYPASSRLS privileges. Never leave user context on a pgxpool connection.

## Files and request security

The filesystem root is private and accessed using Go os.Root. Keys are UUID-based opaque names, not client paths. Creation uses a temporary file and atomic link; the final key cannot be overwritten. Size and signature checks finish before the UPLOADED state can be consumed. Only metadata resides in PostgreSQL. SQL is parameterized and table/column choices are fixed in code.

Signed file URLs are bearer capabilities: redact them from external request logs, retain their five-minute lifetime, and keep the signing secret private. Public images are checked against current publication state rather than exposing a static directory. API responses use no-store, nosniff and no-referrer. HSTS is emitted only in production mode.

CORS uses exact origins. Cookie login/refresh/logout also validate Origin; CORS is not a substitute for CSRF protection. Forwarded IP headers are ignored unless the immediate connection belongs to an explicitly configured trusted CIDR; the chain is processed from the trusted end. Rate limits are per process with bounded memory, not globally distributed. Limits restart when the process restarts.

Anonymous complaint/feedback writes persist no submitter UUID and produce no identity-bearing submission audit. The request logger omits these writes. Complaint contents, passwords and bearer credentials never enter the application access log. Upstream infrastructure must follow the same privacy policy. Anonymous conversation recovery is intentionally absent.

## Retention and deployment boundaries

Users, batches and published content use archive semantics. Audit updates/deletes are rejected by a database trigger. Resource versions and archived files are retained; the jobs command does not destroy historical content. A separate institutional retention policy should govern eventual deletion.

Local Compose credentials are development-only. This configuration is intended for a single API instance with a persistent local volume, not ephemeral Cloud Run storage. Before an internet-facing deployment, replace local SMTP, use externally managed secrets and restricted database users, back up both database and files, and introduce storage/CDN adapters appropriate to that environment.
