package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/jjspscl/my/internal/contexts/habits/application"
	"github.com/jjspscl/my/internal/contexts/habits/domain"
	"github.com/jjspscl/my/internal/shared/middleware"
	"github.com/jjspscl/my/internal/shared/response"
)

type HabitHandler struct {
	svc *application.HabitService
}

func NewHabitHandler(svc *application.HabitService) *HabitHandler {
	return &HabitHandler{svc: svc}
}

func (h *HabitHandler) Routes(r chi.Router) {
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Post("/{id}/toggle", h.Toggle)
	r.Patch("/{id}/archive", h.Archive)
	r.Get("/{id}/completions", h.Completions)
	r.Get("/completions", h.AllCompletions)
}

type createHabitRequest struct {
	Name          string `json:"name"`
	Color         string `json:"color"`
	Frequency     string `json:"frequency"`
	TargetPerWeek int    `json:"targetPerWeek"`
}

type habitResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Color          string `json:"color"`
	Frequency      string `json:"frequency"`
	TargetPerWeek  int    `json:"targetPerWeek"`
	Archived       bool   `json:"archived"`
	CreatedAt      string `json:"createdAt"`
	CompletedToday bool   `json:"completedToday,omitempty"`
	CurrentStreak  int    `json:"currentStreak,omitempty"`
}

type apiResponse struct {
	OK    bool        `json:"ok,omitempty"`
	Error string      `json:"error,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

func toHabitResponse(h *domain.Habit) habitResponse {
	return habitResponse{
		ID:            h.ID,
		Name:          h.Name,
		Color:         h.Color,
		Frequency:     string(h.Frequency),
		TargetPerWeek: h.TargetPerWeek,
		Archived:      h.Archived,
		CreatedAt:     h.CreatedAt.Format(time.RFC3339),
	}
}

func (h *HabitHandler) Create(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())

	var req createHabitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	habit, err := h.svc.Create(r.Context(), email, application.CreateHabitInput{
		Name:          req.Name,
		Color:         req.Color,
		Frequency:     req.Frequency,
		TargetPerWeek: req.TargetPerWeek,
	})
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, apiResponse{OK: true, Data: toHabitResponse(habit)})
}

func (h *HabitHandler) List(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())

	habits, err := h.svc.ListWithStatus(r.Context(), email, time.Now())
	if err != nil {
		response.WriteError(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}

	resp := make([]habitResponse, 0, len(habits))
	for _, hw := range habits {
		hr := toHabitResponse(&hw.Habit)
		hr.CompletedToday = hw.CompletedToday
		hr.CurrentStreak = hw.CurrentStreak
		resp = append(resp, hr)
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{Data: resp})
}

type toggleRequest struct {
	Date string `json:"date,omitempty"`
}

func (h *HabitHandler) Toggle(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")

	var req toggleRequest
	json.NewDecoder(r.Body).Decode(&req)

	dateStr := req.Date
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	toggled, err := h.svc.ToggleCompletion(r.Context(), id, email, dateStr)
	if err != nil {
		response.WriteError(w, r, http.StatusNotFound, err.Error(), err)
		return
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]bool{"completed": toggled}})
}

func (h *HabitHandler) Archive(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.svc.Archive(r.Context(), id, email); err != nil {
		response.WriteError(w, r, http.StatusNotFound, "habit not found", err)
		return
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{OK: true})
}

func (h *HabitHandler) Completions(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	from := time.Now().AddDate(0, -1, 0)
	to := time.Now()

	if fromStr != "" {
		if parsed, err := time.Parse("2006-01-02", fromStr); err == nil {
			from = parsed
		}
	}
	if toStr != "" {
		if parsed, err := time.Parse("2006-01-02", toStr); err == nil {
			to = parsed
		}
	}

	comps, err := h.svc.GetCompletions(r.Context(), id, email, from, to)
	if err != nil {
		response.WriteError(w, r, http.StatusNotFound, err.Error(), err)
		return
	}

	type compResponse struct {
		ID            string `json:"id"`
		CompletedDate string `json:"completedDate"`
	}

	resp := make([]compResponse, 0, len(comps))
	for _, c := range comps {
		resp = append(resp, compResponse{
			ID:            c.ID,
			CompletedDate: c.CompletedDate.Format("2006-01-02"),
		})
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{Data: resp})
}

func (h *HabitHandler) AllCompletions(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	from := time.Now().AddDate(0, -3, 0)
	to := time.Now()

	if fromStr != "" {
		if parsed, err := time.Parse("2006-01-02", fromStr); err == nil {
			from = parsed
		}
	}
	if toStr != "" {
		if parsed, err := time.Parse("2006-01-02", toStr); err == nil {
			to = parsed
		}
	}

	grouped, err := h.svc.GetAllCompletionsGrouped(r.Context(), email, from, to)
	if err != nil {
		response.WriteError(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{Data: grouped})
}
