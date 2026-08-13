package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/jjspscl/my/internal/contexts/finance/application"
	"github.com/jjspscl/my/internal/contexts/finance/domain"
	"github.com/jjspscl/my/internal/shared/middleware"
	"github.com/jjspscl/my/internal/shared/response"
)

type GoalHandler struct {
	svc *application.GoalService
}

func NewGoalHandler(svc *application.GoalService) *GoalHandler {
	return &GoalHandler{svc: svc}
}

func (h *GoalHandler) Routes(r chi.Router) {
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Post("/{id}/contributions", h.AddContribution)
}

// --- Request types ---

type goalRequest struct {
	Name              string  `json:"name"`
	TargetAmountCents int64   `json:"targetAmountCents"`
	TargetDate        *string `json:"targetDate,omitempty"`
	TargetWalletID    string  `json:"targetWalletId"`
}

type contributionRequest struct {
	AmountCents     int64  `json:"amountCents"`
	ContributedAt   string `json:"contributedAt"`
	Note            string `json:"note,omitempty"`
	SourceWalletID  string `json:"sourceWalletId,omitempty"`
	FromAmountCents *int64 `json:"fromAmountCents,omitempty"`
	IdempotencyKey  string `json:"idempotencyKey,omitempty"`
}

// --- Response types ---

type goalResponse struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	TargetAmountCents int64   `json:"targetAmountCents"`
	TargetDate        *string `json:"targetDate,omitempty"`
	TargetWalletID    string  `json:"targetWalletId"`
	CreatedAt         string  `json:"createdAt"`
	UpdatedAt         string  `json:"updatedAt"`
}

type goalSummaryResponse struct {
	ID                   string  `json:"id"`
	Name                 string  `json:"name"`
	TargetAmountCents    int64   `json:"targetAmountCents"`
	TargetDate           *string `json:"targetDate,omitempty"`
	TargetWalletID       string  `json:"targetWalletId"`
	CurrentAmountCents   int64   `json:"currentAmountCents"`
	RemainingAmountCents int64   `json:"remainingAmountCents"`
	ProgressPercent      int     `json:"progressPercent"`
	RequiredMonthlyCents *int64  `json:"requiredMonthlyCents,omitempty"`
	Status               string  `json:"status"`
	CreatedAt            string  `json:"createdAt"`
	UpdatedAt            string  `json:"updatedAt"`
}

// --- Helpers ---

func toGoalResponse(g *domain.SavingsGoal) goalResponse {
	var targetDate *string
	if g.TargetDate != nil {
		s := g.TargetDate.Format("2006-01-02")
		targetDate = &s
	}
	return goalResponse{
		ID:                g.ID,
		Name:              g.Name,
		TargetAmountCents: g.TargetAmountCents,
		TargetDate:        targetDate,
		TargetWalletID:    g.TargetWalletID,
		CreatedAt:         g.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         g.UpdatedAt.Format(time.RFC3339),
	}
}

func toGoalSummaryResponse(s domain.GoalSummary) goalSummaryResponse {
	var targetDate *string
	if s.Goal.TargetDate != nil {
		td := s.Goal.TargetDate.Format("2006-01-02")
		targetDate = &td
	}
	return goalSummaryResponse{
		ID:                   s.Goal.ID,
		Name:                 s.Goal.Name,
		TargetAmountCents:    s.Goal.TargetAmountCents,
		TargetDate:           targetDate,
		TargetWalletID:       s.Goal.TargetWalletID,
		CurrentAmountCents:   s.CurrentAmountCents,
		RemainingAmountCents: s.RemainingAmountCents,
		ProgressPercent:      s.ProgressPercent,
		RequiredMonthlyCents: s.RequiredMonthlyCents,
		Status:               string(s.Status),
		CreatedAt:            s.Goal.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            s.Goal.UpdatedAt.Format(time.RFC3339),
	}
}

// --- Handlers ---

func (h *GoalHandler) Create(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())

	var req goalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	var targetDate *time.Time
	if req.TargetDate != nil {
		parsed, err := time.Parse("2006-01-02", *req.TargetDate)
		if err != nil {
			response.WriteError(w, r, http.StatusBadRequest, "invalid targetDate format, use YYYY-MM-DD", err)
			return
		}
		targetDate = &parsed
	}

	if req.TargetWalletID == "" {
		response.WriteError(w, r, http.StatusBadRequest, "targetWalletId is required", nil)
		return
	}

	goal, err := h.svc.Create(r.Context(), email, application.CreateGoalInput{
		Name:              req.Name,
		TargetAmountCents: req.TargetAmountCents,
		TargetDate:        targetDate,
		TargetWalletID:    req.TargetWalletID,
	})
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, apiResponse{OK: true, Data: toGoalResponse(goal)})
}

func (h *GoalHandler) List(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())

	summaries, err := h.svc.ListSummaries(r.Context(), email)
	if err != nil {
		response.WriteError(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}

	resp := make([]goalSummaryResponse, 0, len(summaries))
	for _, s := range summaries {
		resp = append(resp, toGoalSummaryResponse(s))
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{Data: resp})
}

func (h *GoalHandler) Update(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")

	var req goalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	var targetDate *time.Time
	if req.TargetDate != nil {
		parsed, err := time.Parse("2006-01-02", *req.TargetDate)
		if err != nil {
			response.WriteError(w, r, http.StatusBadRequest, "invalid targetDate format, use YYYY-MM-DD", err)
			return
		}
		targetDate = &parsed
	}

	if req.TargetWalletID == "" {
		response.WriteError(w, r, http.StatusBadRequest, "targetWalletId is required", nil)
		return
	}

	goal, err := h.svc.Update(r.Context(), email, application.UpdateGoalInput{
		ID:                id,
		Name:              req.Name,
		TargetAmountCents: req.TargetAmountCents,
		TargetDate:        targetDate,
		TargetWalletID:    req.TargetWalletID,
	})
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{OK: true, Data: toGoalResponse(goal)})
}

func (h *GoalHandler) Delete(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), id, email); err != nil {
		response.WriteError(w, r, http.StatusNotFound, "goal not found", err)
		return
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{OK: true})
}

func (h *GoalHandler) AddContribution(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")

	var req contributionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	contributedAt, err := time.Parse("2006-01-02", req.ContributedAt)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid contributedAt format, use YYYY-MM-DD", err)
		return
	}

	var note *string
	if req.Note != "" {
		note = &req.Note
	}

	var sourceWalletID *string
	if req.SourceWalletID != "" {
		sourceWalletID = &req.SourceWalletID
	}

	contribution, err := h.svc.AddContribution(r.Context(), email, application.AddContributionInput{
		GoalID:          id,
		AmountCents:     req.AmountCents,
		ContributedAt:   contributedAt,
		Note:            note,
		SourceWalletID:  sourceWalletID,
		FromAmountCents: req.FromAmountCents,
		IdempotencyKey:  req.IdempotencyKey,
	})
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, apiResponse{OK: true, Data: map[string]interface{}{
		"id":            contribution.ID,
		"goalId":        contribution.GoalID,
		"amountCents":   contribution.AmountCents,
		"contributedAt": contribution.ContributedAt.Format("2006-01-02"),
		"note":          contribution.Note,
	}})
}
