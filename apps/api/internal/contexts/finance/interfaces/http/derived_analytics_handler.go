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
