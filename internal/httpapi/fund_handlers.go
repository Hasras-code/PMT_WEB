package httpapi

import (
	"github.com/Hasras-code/PMT_WEB.git/internal/fund"
	"github.com/Hasras-code/PMT_WEB.git/internal/upload"
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (a *API) fundService() fund.Service { return fund.Service{Pool: a.Pool, Uploads: a.Uploads} }
func (a *API) fundRoutes(r chi.Router) {
	r.Get("/funds", a.wrap(a.listFunds))
	r.Post("/funds", a.wrap(a.createFund))
	r.Get("/funds/{fundID}", a.wrap(a.getFund))
	r.Patch("/funds/{fundID}", a.wrap(a.updateFund))
	r.Post("/funds/{fundID}/close", a.wrap(a.closeFund))
	r.Post("/funds/{fundID}/archive", a.wrap(a.archiveFund))
	r.Get("/funds/{fundID}/managers", a.wrap(a.listFundManagers))
	r.Post("/funds/{fundID}/managers", a.wrap(a.assignFundManager))
	r.Delete("/funds/{fundID}/managers/{membershipID}", a.wrap(a.removeFundManager))
	r.Get("/funds/{fundID}/transactions", a.wrap(a.listFundTransactions))
	r.Post("/funds/{fundID}/transactions", a.wrap(a.createFundTransaction))
	r.Get("/funds/{fundID}/transactions/{transactionID}", a.wrap(a.getFundTransaction))
	r.Patch("/funds/{fundID}/transactions/{transactionID}", a.wrap(a.updateFundTransaction))
	r.Post("/funds/{fundID}/transactions/{transactionID}/post", a.wrap(a.postFundTransaction))
	r.Post("/funds/{fundID}/transactions/{transactionID}/reverse", a.wrap(a.reverseFundTransaction))
	r.Post("/fund-transfers", a.wrap(a.createFundTransfer))
	r.Get("/fund-transfers", a.wrap(a.listFundTransfers))
	r.Get("/fund-transfers/{transferID}", a.wrap(a.getFundTransfer))
	r.Post("/fund-transfers/{transferID}/reverse", a.wrap(a.reverseFundTransfer))
	r.Post("/fund-transfers/{loanID}/repayments", a.wrap(a.repayFundLoan))
	r.Get("/birthday-fund", a.wrap(a.birthdayFund))
	r.Get("/birthday-fund/periods", a.wrap(a.listBirthdayPeriods))
	r.Post("/birthday-fund/periods", a.wrap(a.createBirthdayPeriod))
	r.Get("/birthday-fund/periods/{periodID}", a.wrap(a.getBirthdayPeriod))
	r.Post("/birthday-fund/periods/{periodID}/close", a.wrap(a.closeBirthdayPeriod))
	r.Get("/birthday-fund/periods/{periodID}/contributions", a.wrap(a.listBirthdayContributions))
	r.Get("/birthday-fund/periods/{periodID}/my-contribution", a.wrap(a.myBirthdayContribution))
	r.Post("/birthday-fund/periods/{periodID}/contributions/{membershipID}/payments", a.wrap(a.recordBirthdayPayment))
	r.Post("/funds/{fundID}/transactions/{transactionID}/attachments/uploads", a.wrap(a.authorizeFundReceipt))
	r.Post("/funds/{fundID}/transactions/{transactionID}/attachments", a.wrap(a.attachFundReceipt))
	r.Get("/funds/{fundID}/transactions/{transactionID}/attachments", a.wrap(a.listFundReceipts))
	r.Get("/funds/{fundID}/transactions/{transactionID}/attachments/{attachmentID}/download", a.wrap(a.downloadFundReceipt))
}

func (a *API) listFunds(w http.ResponseWriter, r *http.Request) error {
	l, o, e := page(r)
	if e != nil {
		return e
	}
	v, e := a.fundService().List(r.Context(), userID(r), batchID(r), l, o)
	if e != nil {
		return e
	}
	return send(w, 200, v)
}
func (a *API) createFund(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[fund.FundInput](w, r)
	if e != nil {
		return e
	}
	id, e := a.fundService().Create(r.Context(), userID(r), batchID(r), in)
	if e != nil {
		return e
	}
	return created(w, id)
}
func (a *API) getFund(w http.ResponseWriter, r *http.Request) error {
	v, e := a.fundService().Get(r.Context(), userID(r), batchID(r), param(r, "fundID"))
	if e != nil {
		return e
	}
	return send(w, 200, v)
}
func (a *API) updateFund(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[fund.FundUpdate](w, r)
	if e != nil {
		return e
	}
	if e = a.fundService().Update(r.Context(), userID(r), batchID(r), param(r, "fundID"), in); e != nil {
		return e
	}
	return send(w, 204, nil)
}
func (a *API) closeFund(w http.ResponseWriter, r *http.Request) error {
	return a.fundTransition(w, r, "CLOSED")
}
func (a *API) archiveFund(w http.ResponseWriter, r *http.Request) error {
	return a.fundTransition(w, r, "ARCHIVED")
}
func (a *API) fundTransition(w http.ResponseWriter, r *http.Request, target string) error {
	if e := a.fundService().Transition(r.Context(), userID(r), batchID(r), param(r, "fundID"), target); e != nil {
		return e
	}
	return send(w, 204, nil)
}

func (a *API) listFundManagers(w http.ResponseWriter, r *http.Request) error {
	v, e := a.fundService().Managers(r.Context(), userID(r), batchID(r), param(r, "fundID"))
	if e != nil {
		return e
	}
	return send(w, 200, v)
}
func (a *API) assignFundManager(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[fund.ManagerInput](w, r)
	if e != nil {
		return e
	}
	id, e := a.fundService().AssignManager(r.Context(), userID(r), batchID(r), param(r, "fundID"), in.MembershipID)
	if e != nil {
		return e
	}
	return created(w, id)
}
func (a *API) removeFundManager(w http.ResponseWriter, r *http.Request) error {
	if e := a.fundService().RemoveManager(r.Context(), userID(r), batchID(r), param(r, "fundID"), param(r, "membershipID")); e != nil {
		return e
	}
	return send(w, 204, nil)
}

func (a *API) listFundTransactions(w http.ResponseWriter, r *http.Request) error {
	l, o, e := page(r)
	if e != nil {
		return e
	}
	v, e := a.fundService().Transactions(r.Context(), userID(r), batchID(r), param(r, "fundID"), r.URL.Query().Get("type"), r.URL.Query().Get("source_type"), r.URL.Query().Get("date_from"), r.URL.Query().Get("date_to"), r.URL.Query().Get("include_drafts") == "true", l, o)
	if e != nil {
		return e
	}
	return send(w, 200, v)
}
func (a *API) createFundTransaction(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[fund.TransactionInput](w, r)
	if e != nil {
		return e
	}
	id, e := a.fundService().CreateTransaction(r.Context(), userID(r), batchID(r), param(r, "fundID"), in)
	if e != nil {
		return e
	}
	return created(w, id)
}
func (a *API) getFundTransaction(w http.ResponseWriter, r *http.Request) error {
	v, e := a.fundService().Transaction(r.Context(), userID(r), batchID(r), param(r, "fundID"), param(r, "transactionID"))
	if e != nil {
		return e
	}
	return send(w, 200, v)
}
func (a *API) updateFundTransaction(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[fund.TransactionUpdate](w, r)
	if e != nil {
		return e
	}
	if e = a.fundService().UpdateTransaction(r.Context(), userID(r), batchID(r), param(r, "fundID"), param(r, "transactionID"), in); e != nil {
		return e
	}
	return send(w, 204, nil)
}
func (a *API) postFundTransaction(w http.ResponseWriter, r *http.Request) error {
	if e := a.fundService().PostTransaction(r.Context(), userID(r), batchID(r), param(r, "fundID"), param(r, "transactionID")); e != nil {
		return e
	}
	return send(w, 204, nil)
}
func (a *API) reverseFundTransaction(w http.ResponseWriter, r *http.Request) error {
	id, e := a.fundService().ReverseTransaction(r.Context(), userID(r), batchID(r), param(r, "fundID"), param(r, "transactionID"))
	if e != nil {
		return e
	}
	return created(w, id)
}

func (a *API) createFundTransfer(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[fund.TransferInput](w, r)
	if e != nil {
		return e
	}
	id, e := a.fundService().CreateTransfer(r.Context(), userID(r), batchID(r), in)
	if e != nil {
		return e
	}
	return created(w, id)
}
func (a *API) listFundTransfers(w http.ResponseWriter, r *http.Request) error {
	l, o, e := page(r)
	if e != nil {
		return e
	}
	v, e := a.fundService().Transfers(r.Context(), userID(r), batchID(r), l, o)
	if e != nil {
		return e
	}
	return send(w, 200, v)
}
func (a *API) getFundTransfer(w http.ResponseWriter, r *http.Request) error {
	v, e := a.fundService().Transfer(r.Context(), userID(r), batchID(r), param(r, "transferID"))
	if e != nil {
		return e
	}
	return send(w, 200, v)
}
func (a *API) reverseFundTransfer(w http.ResponseWriter, r *http.Request) error {
	id, e := a.fundService().ReverseTransfer(r.Context(), userID(r), batchID(r), param(r, "transferID"))
	if e != nil {
		return e
	}
	return created(w, id)
}
func (a *API) repayFundLoan(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[fund.RepaymentInput](w, r)
	if e != nil {
		return e
	}
	id, e := a.fundService().Repay(r.Context(), userID(r), batchID(r), param(r, "loanID"), in)
	if e != nil {
		return e
	}
	return created(w, id)
}

func (a *API) birthdayFund(w http.ResponseWriter, r *http.Request) error {
	v, e := a.fundService().BirthdaySummary(r.Context(), userID(r), batchID(r))
	if e != nil {
		return e
	}
	return send(w, 200, v)
}
func (a *API) listBirthdayPeriods(w http.ResponseWriter, r *http.Request) error {
	l, o, e := page(r)
	if e != nil {
		return e
	}
	v, e := a.fundService().Periods(r.Context(), userID(r), batchID(r), l, o)
	if e != nil {
		return e
	}
	return send(w, 200, v)
}
func (a *API) createBirthdayPeriod(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[fund.PeriodInput](w, r)
	if e != nil {
		return e
	}
	id, e := a.fundService().CreatePeriod(r.Context(), userID(r), batchID(r), in)
	if e != nil {
		return e
	}
	return created(w, id)
}
func (a *API) getBirthdayPeriod(w http.ResponseWriter, r *http.Request) error {
	v, e := a.fundService().Period(r.Context(), userID(r), batchID(r), param(r, "periodID"))
	if e != nil {
		return e
	}
	return send(w, 200, v)
}
func (a *API) closeBirthdayPeriod(w http.ResponseWriter, r *http.Request) error {
	if e := a.fundService().ClosePeriod(r.Context(), userID(r), batchID(r), param(r, "periodID")); e != nil {
		return e
	}
	return send(w, 204, nil)
}
func (a *API) listBirthdayContributions(w http.ResponseWriter, r *http.Request) error {
	l, o, e := page(r)
	if e != nil {
		return e
	}
	v, e := a.fundService().Contributions(r.Context(), userID(r), batchID(r), param(r, "periodID"), l, o)
	if e != nil {
		return e
	}
	return send(w, 200, v)
}
func (a *API) myBirthdayContribution(w http.ResponseWriter, r *http.Request) error {
	v, e := a.fundService().MyContribution(r.Context(), userID(r), batchID(r), param(r, "periodID"))
	if e != nil {
		return e
	}
	return send(w, 200, v)
}
func (a *API) recordBirthdayPayment(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[fund.PaymentInput](w, r)
	if e != nil {
		return e
	}
	id, e := a.fundService().RecordPayment(r.Context(), userID(r), batchID(r), param(r, "periodID"), param(r, "membershipID"), in)
	if e != nil {
		return e
	}
	return created(w, id)
}

func (a *API) authorizeFundReceipt(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[upload.Input](w, r)
	if e != nil {
		return e
	}
	v, e := a.fundService().AuthorizeReceipt(r.Context(), userID(r), batchID(r), param(r, "fundID"), in)
	if e != nil {
		return e
	}
	return send(w, 201, v)
}
func (a *API) attachFundReceipt(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[fund.AttachmentInput](w, r)
	if e != nil {
		return e
	}
	id, e := a.fundService().Attach(r.Context(), userID(r), batchID(r), param(r, "fundID"), param(r, "transactionID"), in)
	if e != nil {
		return e
	}
	return created(w, id)
}
func (a *API) listFundReceipts(w http.ResponseWriter, r *http.Request) error {
	v, e := a.fundService().Attachments(r.Context(), userID(r), batchID(r), param(r, "fundID"), param(r, "transactionID"))
	if e != nil {
		return e
	}
	return send(w, 200, v)
}
func (a *API) downloadFundReceipt(w http.ResponseWriter, r *http.Request) error {
	o, e := a.fundService().Attachment(r.Context(), userID(r), batchID(r), param(r, "fundID"), param(r, "transactionID"), param(r, "attachmentID"))
	if e != nil {
		return e
	}
	url, e := a.Files.DownloadURL(a.Config.BaseURL, o.Key, o.Name, o.MIME)
	if e != nil {
		return e
	}
	return send(w, 200, map[string]any{"url": url, "expires_in": 300})
}
