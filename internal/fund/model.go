package fund

import (
	"time"

	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/event"
)

var (
	ErrClosed          = apperror.WithCode(apperror.ErrConflict, "fund_closed")
	ErrInsufficient    = apperror.WithCode(apperror.ErrConflict, "insufficient_funds")
	ErrReversed        = apperror.WithCode(apperror.ErrConflict, "transaction_already_reversed")
	ErrInvalidTransfer = apperror.WithCode(apperror.ErrInvalid, "invalid_transfer")
	ErrRepayment       = apperror.WithCode(apperror.ErrConflict, "repayment_exceeds_outstanding")
	ErrPeriodClosed    = apperror.WithCode(apperror.ErrConflict, "birthday_period_closed")
)

type FundInput struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Type        string       `json:"type"`
	EventID     *string      `json:"event_id"`
	Event       *event.Input `json:"event,omitempty"`
}

type FundUpdate struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type TransactionInput struct {
	Type            string     `json:"type"`
	AmountMinor     int64      `json:"amount_minor"`
	Description     string     `json:"description"`
	Category        *string    `json:"category"`
	SourceType      *string    `json:"source_type"`
	SourceName      *string    `json:"source_name"`
	Reference       *string    `json:"reference"`
	TransactionDate *time.Time `json:"transaction_date"`
	Status          string     `json:"status"`
}

type TransactionUpdate struct {
	AmountMinor     *int64     `json:"amount_minor"`
	Description     *string    `json:"description"`
	Category        *string    `json:"category"`
	SourceType      *string    `json:"source_type"`
	SourceName      *string    `json:"source_name"`
	Reference       *string    `json:"reference"`
	TransactionDate *time.Time `json:"transaction_date"`
}

type TransferInput struct {
	FromFundID  string `json:"from_fund_id"`
	ToFundID    string `json:"to_fund_id"`
	Type        string `json:"type"`
	AmountMinor int64  `json:"amount_minor"`
	Description string `json:"description"`
}

type RepaymentInput struct {
	AmountMinor int64  `json:"amount_minor"`
	Description string `json:"description"`
}

type PeriodInput struct {
	Year        int   `json:"year"`
	Month       int   `json:"month"`
	AmountMinor int64 `json:"amount_minor"`
}

type PaymentInput struct {
	AmountMinor int64  `json:"amount_minor"`
	Description string `json:"description"`
}

type ManagerInput struct {
	MembershipID string `json:"membership_id"`
}
type AttachmentInput struct {
	UploadID   string `json:"upload_id"`
	Visibility string `json:"visibility"`
}
