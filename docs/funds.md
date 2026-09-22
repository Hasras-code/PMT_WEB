# Fund management

All money is stored as positive integer LKR minor units. A value of `20000` represents Rs. 200.00. Fund balances are never stored directly; they are derived from posted ledger entries.

`CASH_IN`, `TRANSFER_IN`, and `REVERSAL_IN` increase a balance. `EXPENSE`, `TRANSFER_OUT`, and `REVERSAL_OUT` decrease it. Draft transactions have no financial effect. A posted transaction cannot have its financial fields changed or be deleted. Corrections create linked compensating records.

Every fund belongs to one cohort. Database composite foreign keys enforce cohort ownership for funds, managers, transfers, contribution periods, payments, transactions, and attachments. Transfers can only reference funds in the same cohort and create the outgoing and incoming entries atomically.

An event fund may remain standalone, reference an existing event in the same cohort through `event_id`, or create and link a new event by sending an `event` object in the fund creation request. New event-and-fund creation runs in one database transaction, including both audit records, so a failure cannot leave an orphan event.

Outgoing postings lock their fund row with `FOR UPDATE` before deriving the available balance. Transfers lock both fund rows in UUID order. Loan repayment also locks the original loan before deriving its outstanding principal. These locks prevent concurrent requests from spending the same balance or over-repaying a loan.

Active students receive `fund.view` and can read fund summaries, posted transactions, transfers, aggregate Birthday Fund values, and their own contribution obligation. The complete contribution matrix is restricted to Birthday Fund administrators. `BATCH_REP` receives the batch-level financial permissions. A fund manager may mutate only the exact fund assigned to them; operations affecting two funds require authority for both. Platform roles do not bypass cohort financial boundaries.

Each cohort has exactly one Birthday Fund. Creating a period snapshots the cohort's currently active student memberships and does not change when membership later changes. Recording a payment creates both a posted Birthday Fund `CASH_IN` transaction and the linked contribution payment in one database transaction. Partial payments are supported and total payments cannot exceed the obligation.

Receipt authorization creates a short-lived upload capability. Finalization consumes that intent and stores only permanent object metadata. `MEMBERS` receipts are visible to active cohort members with `fund.view`; `MANAGERS` receipts require authority over the exact fund. Download authorization produces a short-lived capability and never exposes a storage key as authorization.
