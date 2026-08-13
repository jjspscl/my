package http

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/jjspscl/my/internal/contexts/finance/application"
	"github.com/jjspscl/my/internal/contexts/finance/domain"
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
	r.Get("/emergency-fund", h.EmergencyFund)
	r.Get("/affordability", h.Affordability)
	r.Get("/digest", h.Digest)
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

type emergencyFundResponse struct {
	Currency              string   `json:"currency"`
	LiquidBalanceCents    int64    `json:"liquidBalanceCents"`
	MonthlyEssentialCents int64    `json:"monthlyEssentialCents"`
	MonthsOfRunway        float64  `json:"monthsOfRunway"`
	TargetRangeMonths     [2]int   `json:"targetRangeMonths"`
	ShortfallToMinCents   int64    `json:"shortfallToMinCents"`
	ShortfallToMaxCents   int64    `json:"shortfallToMaxCents"`
	Assumptions           []string `json:"assumptions"`
}

type affordabilityResponse struct {
	Currency               string   `json:"currency"`
	AmountCents            int64    `json:"amountCents"`
	LiquidBalanceCents     int64    `json:"liquidBalanceCents"`
	MonthlyEssentialCents  int64    `json:"monthlyEssentialCents"`
	UpcomingBillsCents     int64    `json:"upcomingBillsCents"`
	MonthlyObligationCents int64    `json:"monthlyObligationCents"`
	RunwayMonthsBefore     float64  `json:"runwayMonthsBefore"`
	RunwayMonthsAfter      float64  `json:"runwayMonthsAfter"`
	Assumptions            []string `json:"assumptions"`
}

type digestCashFlowResponse struct {
	Present    bool                       `json:"present"`
	Summary    string                     `json:"summary"`
	Currencies []currencyCashFlowResponse `json:"currencies"`
}

type digestSpendingResponse struct {
	Present              bool                       `json:"present"`
	Summary              string                     `json:"summary"`
	Currencies           []currencySpendingResponse `json:"currencies"`
	UnclassifiedSharePct float64                    `json:"unclassifiedSharePct"`
}

type digestSavingsResponse struct {
	Present bool                  `json:"present"`
	Summary string                `json:"summary"`
	Rates   []savingsRateResponse `json:"rates"`
}

type digestRecurringResponse struct {
	Present bool                      `json:"present"`
	Summary string                    `json:"summary"`
	Charges []recurringChargeResponse `json:"charges"`
}

type digestAnomaliesResponse struct {
	Present   bool              `json:"present"`
	Summary   string            `json:"summary"`
	Anomalies []anomalyResponse `json:"anomalies"`
}

type digestEmergencyResponse struct {
	Present bool                   `json:"present"`
	Summary string                 `json:"summary"`
	Status  *emergencyFundResponse `json:"status,omitempty"`
}

type monthlyDigestResponse struct {
	Month       string                  `json:"month"`
	CashFlow    digestCashFlowResponse  `json:"cashFlow"`
	Spending    digestSpendingResponse  `json:"spending"`
	SavingsRate digestSavingsResponse   `json:"savingsRate"`
	Recurring   digestRecurringResponse `json:"recurring"`
	Anomalies   digestAnomaliesResponse `json:"anomalies"`
	Emergency   digestEmergencyResponse `json:"emergency"`
	Omitted     []string                `json:"omitted"`
	Assumptions []string                `json:"assumptions"`
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

func (h *DerivedAnalyticsHandler) EmergencyFund(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	currency := r.URL.Query().Get("currency")
	targetMonths := 0
	if v := r.URL.Query().Get("targetMonths"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			response.WriteError(w, r, http.StatusBadRequest, "targetMonths must be an integer", err)
			return
		}
		targetMonths = n
	}

	status, err := h.svc.GetEmergencyFund(r.Context(), email, currency, targetMonths)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{Data: emergencyFundResponse{
		Currency:              status.Currency,
		LiquidBalanceCents:    status.LiquidBalanceCents,
		MonthlyEssentialCents: status.MonthlyEssentialCents,
		MonthsOfRunway:        status.MonthsOfRunway,
		TargetRangeMonths:     status.TargetRangeMonths,
		ShortfallToMinCents:   status.ShortfallToMinCents,
		ShortfallToMaxCents:   status.ShortfallToMaxCents,
		Assumptions:           status.Assumptions,
	}})
}

func (h *DerivedAnalyticsHandler) Affordability(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	currency := r.URL.Query().Get("currency")
	amountCents, err := strconv.ParseInt(r.URL.Query().Get("amountCents"), 10, 64)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "amountCents must be an integer", err)
		return
	}

	model, err := h.svc.GetAffordability(r.Context(), email, currency, amountCents)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{Data: affordabilityResponse{
		Currency:               model.Currency,
		AmountCents:            model.AmountCents,
		LiquidBalanceCents:     model.LiquidBalanceCents,
		MonthlyEssentialCents:  model.MonthlyEssentialCents,
		UpcomingBillsCents:     model.UpcomingBillsCents,
		MonthlyObligationCents: model.MonthlyObligationCents,
		RunwayMonthsBefore:     model.RunwayMonthsBefore,
		RunwayMonthsAfter:      model.RunwayMonthsAfter,
		Assumptions:            model.Assumptions,
	}})
}

func (h *DerivedAnalyticsHandler) Digest(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	month := r.URL.Query().Get("month")
	if month == "" {
		month = h.clock.CurrentMonth()
	}

	digest, err := h.svc.GetMonthlyDigest(r.Context(), email, month)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	out := monthlyDigestResponse{
		Month:       digest.Month,
		Omitted:     digest.Omitted,
		Assumptions: digest.Assumptions,
	}
	if digest.CashFlow.Present {
		out.CashFlow = digestCashFlowResponse{
			Present:    true,
			Summary:    digest.CashFlow.Summary,
			Currencies: toCurrencyCashFlowResponses(digest.CashFlow.Currencies),
		}
	}
	if digest.Spending.Present {
		out.Spending = digestSpendingResponse{
			Present:              true,
			Summary:              digest.Spending.Summary,
			Currencies:           toCurrencySpendingResponses(digest.Spending.Currencies),
			UnclassifiedSharePct: digest.Spending.UnclassifiedSharePct,
		}
	}
	if digest.SavingsRate.Present {
		out.SavingsRate = digestSavingsResponse{
			Present: true,
			Summary: digest.SavingsRate.Summary,
			Rates:   toSavingsRateResponses(digest.SavingsRate.Rates),
		}
	}
	if digest.Recurring.Present {
		out.Recurring = digestRecurringResponse{
			Present: true,
			Summary: digest.Recurring.Summary,
			Charges: toRecurringChargeResponses(digest.Recurring.Charges),
		}
	}
	if digest.Anomalies.Present {
		out.Anomalies = digestAnomaliesResponse{
			Present:   true,
			Summary:   digest.Anomalies.Summary,
			Anomalies: toAnomalyResponses(digest.Anomalies.Anomalies),
		}
	}
	if digest.Emergency.Present {
		out.Emergency = digestEmergencyResponse{
			Present: true,
			Summary: digest.Emergency.Summary,
			Status:  toEmergencyFundResponse(digest.Emergency.Status),
		}
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{Data: out})
}

// --- Response mappers ---

func toCurrencyCashFlowResponses(in []domain.CurrencyCashFlow) []currencyCashFlowResponse {
	out := make([]currencyCashFlowResponse, 0, len(in))
	for _, c := range in {
		out = append(out, currencyCashFlowResponse{
			Currency:     c.Currency,
			IncomeCents:  c.IncomeCents,
			ExpenseCents: c.ExpenseCents,
			NetCents:     c.NetCents,
		})
	}
	return out
}

func toCurrencySpendingResponses(in []domain.CurrencySpending) []currencySpendingResponse {
	out := make([]currencySpendingResponse, 0, len(in))
	for _, c := range in {
		byClass := make(map[string]int64, len(c.ByClassification))
		for k, v := range c.ByClassification {
			byClass[string(k)] = v
		}
		out = append(out, currencySpendingResponse{
			Currency:          c.Currency,
			ByClassification:  byClass,
			UnclassifiedCents: c.UnclassifiedCents,
		})
	}
	return out
}

func toSavingsRateResponses(in []domain.SavingsRate) []savingsRateResponse {
	out := make([]savingsRateResponse, 0, len(in))
	for _, r := range in {
		out = append(out, savingsRateResponse{
			Currency:    r.Currency,
			RatePercent: r.RatePercent,
			ZeroIncome:  r.ZeroIncome,
		})
	}
	return out
}

func toRecurringChargeResponses(in []domain.RecurringCharge) []recurringChargeResponse {
	out := make([]recurringChargeResponse, 0, len(in))
	for _, c := range in {
		out = append(out, recurringChargeResponse{
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
	return out
}

func toAnomalyResponses(in []domain.Anomaly) []anomalyResponse {
	out := make([]anomalyResponse, 0, len(in))
	for _, a := range in {
		out = append(out, anomalyResponse{
			Category:    a.Category,
			Currency:    a.Currency,
			Month:       a.Month,
			AmountCents: a.AmountCents,
			MedianCents: a.MedianCents,
			Ratio:       a.Ratio,
			Explanation: a.Explanation,
		})
	}
	return out
}

func toEmergencyFundResponse(in *domain.EmergencyFundStatus) *emergencyFundResponse {
	if in == nil {
		return nil
	}
	return &emergencyFundResponse{
		Currency:              in.Currency,
		LiquidBalanceCents:    in.LiquidBalanceCents,
		MonthlyEssentialCents: in.MonthlyEssentialCents,
		MonthsOfRunway:        in.MonthsOfRunway,
		TargetRangeMonths:     in.TargetRangeMonths,
		ShortfallToMinCents:   in.ShortfallToMinCents,
		ShortfallToMaxCents:   in.ShortfallToMaxCents,
		Assumptions:           in.Assumptions,
	}
}
