package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/jjspscl/my/internal/contexts/finance/application"
	"github.com/jjspscl/my/internal/shared/middleware"
	"github.com/jjspscl/my/internal/shared/response"
)

type BudgetHandler struct {
	svc *application.BudgetService
}

func NewBudgetHandler(svc *application.BudgetService) *BudgetHandler {
	return &BudgetHandler{svc: svc}
}

func (h *BudgetHandler) Routes(r chi.Router) {
	r.Get("/", h.GetSummary)
	r.Put("/", h.Upsert)
}

// --- Request types ---

type budgetCategoryRequest struct {
	Category        string `json:"category"`
	AllocatedCents  int64  `json:"allocatedCents"`
	RolloverEnabled bool   `json:"rolloverEnabled"`
}

type upsertBudgetRequest struct {
	Month      string                  `json:"month"`
	Categories []budgetCategoryRequest `json:"categories"`
}

// --- Response types ---

type budgetCategorySummaryResponse struct {
	Category        string `json:"category"`
	AllocatedCents  int64  `json:"allocatedCents"`
	SpentCents      int64  `json:"spentCents"`
	RemainingCents  int64  `json:"remainingCents"`
	RolloverEnabled bool   `json:"rolloverEnabled"`
}

type budgetSummaryResponse struct {
	Month               string                          `json:"month"`
	TotalAllocatedCents int64                           `json:"totalAllocatedCents"`
	TotalSpentCents     int64                           `json:"totalSpentCents"`
	TotalRemainingCents int64                           `json:"totalRemainingCents"`
	Categories          []budgetCategorySummaryResponse `json:"categories"`
}

// --- Handlers ---

func (h *BudgetHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	month := r.URL.Query().Get("month")
	if month == "" {
		response.WriteError(w, r, http.StatusBadRequest, "month query param required (YYYY-MM)", nil)
		return
	}

	summary, err := h.svc.GetSummary(r.Context(), email, month)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	cats := make([]budgetCategorySummaryResponse, 0, len(summary.Categories))
	for _, c := range summary.Categories {
		cats = append(cats, budgetCategorySummaryResponse{
			Category:        c.Category,
			AllocatedCents:  c.AllocatedCents,
			SpentCents:      c.SpentCents,
			RemainingCents:  c.RemainingCents,
			RolloverEnabled: c.RolloverEnabled,
		})
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{
		OK: true,
		Data: budgetSummaryResponse{
			Month:               summary.Month,
			TotalAllocatedCents: summary.TotalAllocatedCents,
			TotalSpentCents:     summary.TotalSpentCents,
			TotalRemainingCents: summary.TotalRemainingCents,
			Categories:          cats,
		},
	})
}

func (h *BudgetHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())

	var req upsertBudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	cats := make([]application.BudgetCategoryInput, 0, len(req.Categories))
	for _, c := range req.Categories {
		cats = append(cats, application.BudgetCategoryInput{
			Category:        c.Category,
			AllocatedCents:  c.AllocatedCents,
			RolloverEnabled: c.RolloverEnabled,
		})
	}

	budget, err := h.svc.UpsertBudget(r.Context(), email, application.UpsertBudgetInput{
		Month:      req.Month,
		Categories: cats,
	})
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]string{
		"id":    budget.ID,
		"month": budget.Month,
	}})
}
