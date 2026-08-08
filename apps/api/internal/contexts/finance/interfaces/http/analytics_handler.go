package http

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/jjspscl/my/internal/contexts/finance/application"
	"github.com/jjspscl/my/internal/contexts/finance/domain"
	"github.com/jjspscl/my/internal/platform/timeutil"
	"github.com/jjspscl/my/internal/shared/middleware"
	"github.com/jjspscl/my/internal/shared/response"
)

type AnalyticsHandler struct {
	svc   *application.AnalyticsService
	clock *timeutil.Clock
}

func NewAnalyticsHandler(svc *application.AnalyticsService, clock *timeutil.Clock) *AnalyticsHandler {
	return &AnalyticsHandler{svc: svc, clock: clock}
}

func (h *AnalyticsHandler) Routes(r chi.Router) {
	r.Get("/spending", h.SpendingSummary)
	r.Get("/cash-flow", h.CashFlowSummary)
	r.Get("/category-trend", h.CategoryTrend)
	r.Get("/budget-health", h.BudgetHealth)
	r.Get("/goal-health", h.GoalHealth)
	r.Get("/savings-rate", h.SavingsRate)
}

// --- Response types ---

type dateRangeResponse struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type currencySpendingResponse struct {
	Currency          string           `json:"currency"`
	TotalExpenseCents int64            `json:"totalExpenseCents"`
	ByClassification  map[string]int64 `json:"byClassification"`
	UnclassifiedCents int64            `json:"unclassifiedCents"`
}

type spendingSummaryResponse struct {
	DateRange            dateRangeResponse          `json:"dateRange"`
	Currencies           []currencySpendingResponse `json:"currencies"`
	UnclassifiedSharePct float64                    `json:"unclassifiedSharePct"`
	Assumptions          []string                   `json:"assumptions"`
}

type monthlyCashFlowResponse struct {
	Month        string `json:"month"`
	Currency     string `json:"currency"`
	IncomeCents  int64  `json:"incomeCents"`
	ExpenseCents int64  `json:"expenseCents"`
	NetCents     int64  `json:"netCents"`
}

type currencyCashFlowResponse struct {
	Currency     string                    `json:"currency"`
	IncomeCents  int64                     `json:"incomeCents"`
	ExpenseCents int64                     `json:"expenseCents"`
	NetCents     int64                     `json:"netCents"`
	Monthly      []monthlyCashFlowResponse `json:"monthly"`
}

type cashFlowSummaryResponse struct {
	DateRange   dateRangeResponse          `json:"dateRange"`
	Currencies  []currencyCashFlowResponse `json:"currencies"`
	Assumptions []string                   `json:"assumptions"`
}

type categoryTrendPointResponse struct {
	Month       string `json:"month"`
	AmountCents int64  `json:"amountCents"`
}

type categoryTrendResponse struct {
	Category    string                       `json:"category"`
	Currency    string                       `json:"currency"`
	Months      []categoryTrendPointResponse `json:"months"`
	SampleSize  int                          `json:"sampleSize"`
	Sufficient  bool                         `json:"sufficient"`
	Assumptions []string                     `json:"assumptions"`
}

type budgetHealthCategoryResponse struct {
	Category       string `json:"category"`
	AllocatedCents int64  `json:"allocatedCents"`
	SpentCents     int64  `json:"spentCents"`
	RemainingCents int64  `json:"remainingCents"`
}

type budgetHealthResponse struct {
	Month                string                         `json:"month"`
	Currency             string                         `json:"currency,omitempty"`
	HasBudget            bool                           `json:"hasBudget"`
	TotalAllocatedCents  int64                          `json:"totalAllocatedCents"`
	TotalSpentCents      int64                          `json:"totalSpentCents"`
	TotalRemainingCents  int64                          `json:"totalRemainingCents"`
	UnbudgetedSpentCents int64                          `json:"unbudgetedSpentCents"`
	Categories           []budgetHealthCategoryResponse `json:"categories"`
	Assumptions          []string                       `json:"assumptions"`
}

type goalHealthItemResponse struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	Currency             string `json:"currency"`
	TargetAmountCents    int64  `json:"targetAmountCents"`
	CurrentAmountCents   int64  `json:"currentAmountCents"`
	RemainingAmountCents int64  `json:"remainingAmountCents"`
	ProgressPercent      int    `json:"progressPercent"`
	Status               string `json:"status"`
	RequiredMonthlyCents *int64 `json:"requiredMonthlyCents,omitempty"`
}

type goalHealthResponse struct {
	Goals       []goalHealthItemResponse `json:"goals"`
	Assumptions []string                 `json:"assumptions"`
}

type savingsRateResponse struct {
	Currency     string   `json:"currency"`
	IncomeCents  int64    `json:"incomeCents"`
	ExpenseCents int64    `json:"expenseCents"`
	NetCents     int64    `json:"netCents"`
	RatePercent  float64  `json:"ratePercent"`
	ZeroIncome   bool     `json:"zeroIncome"`
	Assumptions  []string `json:"assumptions"`
}

// --- Handlers ---

func (h *AnalyticsHandler) SpendingSummary(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	from, to, ok := h.parseRange(w, r)
	if !ok {
		return
	}

	summary, err := h.svc.GetSpendingSummary(r.Context(), email, from, to)
	if err != nil {
		var insufficient *domain.ErrInsufficientClassification
		if errors.As(err, &insufficient) {
			response.WriteError(w, r, http.StatusUnprocessableEntity, insufficient.Error(), err)
			return
		}
		response.WriteError(w, r, http.StatusInternalServerError, "failed to compute spending summary", err)
		return
	}

	out := spendingSummaryResponse{
		DateRange:            dateRangeResponse{From: summary.DateRange.From, To: summary.DateRange.To},
		UnclassifiedSharePct: summary.UnclassifiedSharePct,
		Assumptions:          summary.Assumptions,
		Currencies:           make([]currencySpendingResponse, 0, len(summary.Currencies)),
	}
	for _, c := range summary.Currencies {
		byClass := make(map[string]int64, len(c.ByClassification))
		for k, v := range c.ByClassification {
			byClass[string(k)] = v
		}
		out.Currencies = append(out.Currencies, currencySpendingResponse{
			Currency:          c.Currency,
			TotalExpenseCents: c.TotalExpenseCents,
			ByClassification:  byClass,
			UnclassifiedCents: c.UnclassifiedCents,
		})
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{Data: out})
}

func (h *AnalyticsHandler) CashFlowSummary(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	from, to, ok := h.parseRange(w, r)
	if !ok {
		return
	}

	summary, err := h.svc.GetCashFlowSummary(r.Context(), email, from, to)
	if err != nil {
		response.WriteError(w, r, http.StatusInternalServerError, "failed to compute cash flow summary", err)
		return
	}

	out := cashFlowSummaryResponse{
		DateRange:   dateRangeResponse{From: summary.DateRange.From, To: summary.DateRange.To},
		Assumptions: summary.Assumptions,
		Currencies:  make([]currencyCashFlowResponse, 0, len(summary.Currencies)),
	}
	for _, c := range summary.Currencies {
		cc := currencyCashFlowResponse{
			Currency:     c.Currency,
			IncomeCents:  c.IncomeCents,
			ExpenseCents: c.ExpenseCents,
			NetCents:     c.NetCents,
			Monthly:      make([]monthlyCashFlowResponse, 0, len(c.Monthly)),
		}
		for _, m := range c.Monthly {
			cc.Monthly = append(cc.Monthly, monthlyCashFlowResponse{
				Month:        m.Month,
				Currency:     m.Currency,
				IncomeCents:  m.IncomeCents,
				ExpenseCents: m.ExpenseCents,
				NetCents:     m.NetCents,
			})
		}
		out.Currencies = append(out.Currencies, cc)
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{Data: out})
}

func (h *AnalyticsHandler) CategoryTrend(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	category := r.URL.Query().Get("category")
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

	trend, err := h.svc.GetCategoryTrend(r.Context(), email, category, currency, months)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	out := categoryTrendResponse{
		Category:    trend.Category,
		Currency:    trend.Currency,
		SampleSize:  trend.SampleSize,
		Sufficient:  trend.Sufficient,
		Assumptions: trend.Assumptions,
		Months:      make([]categoryTrendPointResponse, 0, len(trend.Months)),
	}
	for _, m := range trend.Months {
		out.Months = append(out.Months, categoryTrendPointResponse{Month: m.Month, AmountCents: m.AmountCents})
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{Data: out})
}

func (h *AnalyticsHandler) BudgetHealth(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	month := r.URL.Query().Get("month")
	if month == "" {
		month = h.clock.CurrentMonth()
	}

	health, err := h.svc.GetBudgetHealth(r.Context(), email, month)
	if err != nil {
		response.WriteError(w, r, http.StatusInternalServerError, "failed to compute budget health", err)
		return
	}

	out := budgetHealthResponse{
		Month:                health.Month,
		Currency:             health.Currency,
		HasBudget:            health.HasBudget,
		TotalAllocatedCents:  health.TotalAllocatedCents,
		TotalSpentCents:      health.TotalSpentCents,
		TotalRemainingCents:  health.TotalRemainingCents,
		UnbudgetedSpentCents: health.UnbudgetedSpentCents,
		Assumptions:          health.Assumptions,
		Categories:           make([]budgetHealthCategoryResponse, 0, len(health.Categories)),
	}
	for _, c := range health.Categories {
		out.Categories = append(out.Categories, budgetHealthCategoryResponse{
			Category:       c.Category,
			AllocatedCents: c.AllocatedCents,
			SpentCents:     c.SpentCents,
			RemainingCents: c.RemainingCents,
		})
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{Data: out})
}

func (h *AnalyticsHandler) GoalHealth(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())

	health, err := h.svc.GetGoalHealth(r.Context(), email)
	if err != nil {
		response.WriteError(w, r, http.StatusInternalServerError, "failed to compute goal health", err)
		return
	}

	out := goalHealthResponse{
		Assumptions: health.Assumptions,
		Goals:       make([]goalHealthItemResponse, 0, len(health.Goals)),
	}
	for _, g := range health.Goals {
		out.Goals = append(out.Goals, goalHealthItemResponse{
			ID:                   g.ID,
			Name:                 g.Name,
			Currency:             g.Currency,
			TargetAmountCents:    g.TargetAmountCents,
			CurrentAmountCents:   g.CurrentAmountCents,
			RemainingAmountCents: g.RemainingAmountCents,
			ProgressPercent:      g.ProgressPercent,
			Status:               string(g.Status),
			RequiredMonthlyCents: g.RequiredMonthlyCents,
		})
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{Data: out})
}

func (h *AnalyticsHandler) SavingsRate(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	from, to, ok := h.parseRange(w, r)
	if !ok {
		return
	}

	rates, err := h.svc.GetSavingsRate(r.Context(), email, from, to)
	if err != nil {
		response.WriteError(w, r, http.StatusInternalServerError, "failed to compute savings rate", err)
		return
	}

	out := make([]savingsRateResponse, 0, len(rates))
	for _, rate := range rates {
		out = append(out, savingsRateResponse{
			Currency:     rate.Currency,
			IncomeCents:  rate.IncomeCents,
			ExpenseCents: rate.ExpenseCents,
			NetCents:     rate.NetCents,
			RatePercent:  rate.RatePercent,
			ZeroIncome:   rate.ZeroIncome,
			Assumptions:  rate.Assumptions,
		})
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{Data: out})
}

// parseRange reads from/to as YYYY-MM-DD, defaulting to the current month
// (half-open). On parse failure it writes the error response and returns ok=false.
func (h *AnalyticsHandler) parseRange(w http.ResponseWriter, r *http.Request) (time.Time, time.Time, bool) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	if fromStr == "" && toStr == "" {
		from, to, err := h.clock.MonthRange(h.clock.CurrentMonth())
		if err != nil {
			response.WriteError(w, r, http.StatusInternalServerError, "failed to resolve month range", err)
			return time.Time{}, time.Time{}, false
		}
		return from, to, true
	}

	from, err := h.clock.ParseDate(fromStr)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid from date, use YYYY-MM-DD", err)
		return time.Time{}, time.Time{}, false
	}
	to, err := h.clock.ParseDate(toStr)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid to date, use YYYY-MM-DD", err)
		return time.Time{}, time.Time{}, false
	}
	if !to.After(from) {
		response.WriteError(w, r, http.StatusBadRequest, "to must be after from", nil)
		return time.Time{}, time.Time{}, false
	}
	return from, to, true
}
