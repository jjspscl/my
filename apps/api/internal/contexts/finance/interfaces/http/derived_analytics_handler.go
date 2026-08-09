package http

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/jjspscl/my/internal/contexts/finance/application"
	"github.com/jjspscl/my/internal/platform/timeutil"
	"github.com/jjspscl/my/internal/shared/middleware"
	"github.com/jjspscl/my/internal/shared/response"
)

// DerivedAnalyticsHandler exposes the derived analytics: spending anomalies,
// recurring-charge summary, bill reconciliation, emergency-fund status,
// affordability, and the monthly digest. Every response carries named
// assumptions and human-readable explanations.
type DerivedAnalyticsHandler struct {
	svc   *application.DerivedAnalyticsService
	clock *timeutil.Clock
}

func NewDerivedAnalyticsHandler(svc *application.DerivedAnalyticsService, clock *timeutil.Clock) *DerivedAnalyticsHandler {
	return &DerivedAnalyticsHandler{svc: svc, clock: clock}
}

func (h *DerivedAnalyticsHandler) Routes(r chi.Router) {
	r.Get("/anomalies", h.Anomalies)
	r.Get("/recurring-charges", h.RecurringCharges)
	r.Get("/bill-reconciliation", h.BillReconciliation)
}

// --- Response types ---

type anomalyResponse struct {
	Category    string  `json:"category"`
	Currency    string  `json:"currency"`
	Month       string  `json:"month"`
	AmountCents int64   `json:"amountCents"`
	MedianCents int64   `json:"medianCents"`
	Ratio       float64 `json:"ratio"`
	Explanation string  `json:"explanation"`
}

type anomalyReportResponse struct {
	Currency    string            `json:"currency"`
	Months      int               `json:"months"`
	Sufficient  bool              `json:"sufficient"`
	Anomalies   []anomalyResponse `json:"anomalies"`
	Assumptions []string          `json:"assumptions"`
}

type recurringChargeResponse struct {
	Category        string `json:"category"`
	Currency        string `json:"currency"`
	Occurrences     int    `json:"occurrences"`
	DistinctMonths  int    `json:"distinctMonths"`
	MedianCents     int64  `json:"medianCents"`
	Status          string `json:"status"`
	BillName        string `json:"billName,omitempty"`
	BillAmountCents int64  `json:"billAmountCents,omitempty"`
	Explanation     string `json:"explanation"`
}

type recurringChargesSummary struct {
	Currency    string                    `json:"currency"`
	Months      int                       `json:"months"`
	Charges     []recurringChargeResponse `json:"charges"`
	Assumptions []string                  `json:"assumptions"`
}

type billReconciliationItemResponse struct {
	BillID                      string `json:"billId"`
	Name                        string `json:"name"`
	Category                    string `json:"category"`
	Currency                    string `json:"currency"`
	ExpectedCents               int64  `json:"expectedCents"`
	PaidCents                   int64  `json:"paidCents"`
	VarianceCents               int64  `json:"varianceCents"`
	PaidCount                   int    `json:"paidCount"`
	PaidWithoutTransactionCount int    `json:"paidWithoutTransactionCount"`
	Explanation                 string `json:"explanation"`
}

type billReconciliationResponse struct {
	Month       string                           `json:"month"`
	Items       []billReconciliationItemResponse `json:"items"`
	Assumptions []string                         `json:"assumptions"`
}

// --- Handlers ---

func (h *DerivedAnalyticsHandler) Anomalies(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	currency := r.URL.Query().Get("currency")
	months := 6
	if v := r.URL.Query().Get("months"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			response.WriteError(w, r, http.StatusBadRequest, "months must be an integer", err)
			return
		}
		months = n
	}

	report, err := h.svc.GetMonthlyAnomalies(r.Context(), email, currency, months)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	out := anomalyReportResponse{
		Currency:    report.Currency,
		Months:      report.Months,
		Sufficient:  report.Sufficient,
		Assumptions: report.Assumptions,
		Anomalies:   make([]anomalyResponse, 0, len(report.Anomalies)),
	}
	for _, a := range report.Anomalies {
		out.Anomalies = append(out.Anomalies, anomalyResponse{
			Category:    a.Category,
			Currency:    a.Currency,
			Month:       a.Month,
			AmountCents: a.AmountCents,
			MedianCents: a.MedianCents,
			Ratio:       a.Ratio,
			Explanation: a.Explanation,
		})
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{Data: out})
}

func (h *DerivedAnalyticsHandler) RecurringCharges(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	currency := r.URL.Query().Get("currency")
	months := 6
	if v := r.URL.Query().Get("months"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			response.WriteError(w, r, http.StatusBadRequest, "months must be an integer", err)
			return
		}
		months = n
	}

	summary, err := h.svc.GetRecurringCharges(r.Context(), email, currency, months)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	out := recurringChargesSummary{
		Currency:    summary.Currency,
		Months:      summary.Months,
		Assumptions: summary.Assumptions,
		Charges:     make([]recurringChargeResponse, 0, len(summary.Charges)),
	}
	for _, c := range summary.Charges {
		out.Charges = append(out.Charges, recurringChargeResponse{
			Category:        c.Category,
			Currency:        c.Currency,
			Occurrences:     c.Occurrences,
			DistinctMonths:  c.DistinctMonths,
			MedianCents:     c.MedianCents,
			Status:          string(c.Status),
			BillName:        c.BillName,
			BillAmountCents: c.BillAmountCents,
			Explanation:     c.Explanation,
		})
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{Data: out})
}

func (h *DerivedAnalyticsHandler) BillReconciliation(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	month := r.URL.Query().Get("month")
	if month == "" {
		month = h.clock.CurrentMonth()
	}

	recon, err := h.svc.GetBillReconciliation(r.Context(), email, month)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	out := billReconciliationResponse{
		Month:       recon.Month,
		Assumptions: recon.Assumptions,
		Items:       make([]billReconciliationItemResponse, 0, len(recon.Items)),
	}
	for _, it := range recon.Items {
		out.Items = append(out.Items, billReconciliationItemResponse{
			BillID:                      it.BillID,
			Name:                        it.Name,
			Category:                    it.Category,
			Currency:                    it.Currency,
			ExpectedCents:               it.ExpectedCents,
			PaidCents:                   it.PaidCents,
			VarianceCents:               it.VarianceCents,
			PaidCount:                   it.PaidCount,
			PaidWithoutTransactionCount: it.PaidWithoutTransactionCount,
			Explanation:                 it.Explanation,
		})
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{Data: out})
}
