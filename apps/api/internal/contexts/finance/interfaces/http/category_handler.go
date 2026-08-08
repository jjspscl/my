package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/jjspscl/my/internal/contexts/finance/application"
	"github.com/jjspscl/my/internal/contexts/finance/domain"
	"github.com/jjspscl/my/internal/shared/response"
)

type CategoryHandler struct {
	svc *application.CategoryService
}

func NewCategoryHandler(svc *application.CategoryService) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

func (h *CategoryHandler) Routes(r chi.Router) {
	r.Get("/", h.List)
	r.Put("/{name}", h.Update)
}

// --- Request types ---

type updateCategoryRequest struct {
	Classification string `json:"classification"`
	Essential      bool   `json:"essential"`
	Active         bool   `json:"active"`
}

// --- Response types ---

type categoryResponse struct {
	Name           string `json:"name"`
	Classification string `json:"classification"`
	Essential      bool   `json:"essential"`
	Active         bool   `json:"active"`
}

func toCategoryResponse(c *domain.Category) categoryResponse {
	return categoryResponse{
		Name:           c.Name,
		Classification: string(c.Classification),
		Essential:      c.Essential,
		Active:         c.Active,
	}
}

// --- Handlers ---

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	categories, err := h.svc.List(r.Context())
	if err != nil {
		response.WriteError(w, r, http.StatusInternalServerError, "failed to list categories", err)
		return
	}

	out := make([]categoryResponse, 0, len(categories))
	for _, c := range categories {
		out = append(out, toCategoryResponse(c))
	}
	response.WriteJSON(w, http.StatusOK, apiResponse{Data: out})
}

func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		response.WriteError(w, r, http.StatusBadRequest, "category name is required", nil)
		return
	}

	var req updateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	category, err := h.svc.Update(r.Context(), name, domain.Classification(req.Classification), req.Essential, req.Active)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{OK: true, Data: toCategoryResponse(category)})
}
