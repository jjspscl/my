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

type FinanceHandler struct {
	svc *application.TransactionService
}

func NewFinanceHandler(svc *application.TransactionService) *FinanceHandler {
	return &FinanceHandler{svc: svc}
}

func (h *FinanceHandler) Routes(r chi.Router) {
	r.Get("/transactions", h.List)
	r.Post("/transactions", h.Create)
	r.Get("/transactions/today-total", h.TodayTotal)
	r.Delete("/transactions/{id}", h.Delete)
}

type createTransactionRequest struct {
	AmountCents     int64  `json:"amountCents"`
	Category        string `json:"category"`
	Description     string `json:"description"`
	Type            string `json:"type"`
	TransactionDate string `json:"transactionDate"`
}

type transactionResponse struct {
	ID              string `json:"id"`
	AmountCents     int64  `json:"amountCents"`
	Currency        string `json:"currency"`
	Category        string `json:"category"`
	Description     string `json:"description"`
	Type            string `json:"type"`
	TransactionDate string `json:"transactionDate"`
	CreatedAt       string `json:"createdAt"`
}

type apiResponse struct {
	OK    bool        `json:"ok,omitempty"`
	Error string      `json:"error,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

func toTransactionResponse(t *domain.Transaction) transactionResponse {
	return transactionResponse{
		ID:              t.ID,
		AmountCents:     t.AmountCents,
		Currency:        t.Currency,
		Category:        t.Category,
		Description:     t.Description,
		Type:            string(t.Type),
		TransactionDate: t.TransactionDate.Format("2006-01-02"),
		CreatedAt:       t.CreatedAt.Format(time.RFC3339),
	}
}

func (h *FinanceHandler) Create(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())

	var req createTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if req.AmountCents <= 0 {
		response.WriteError(w, r, http.StatusBadRequest, "amountCents must be positive", nil)
		return
	}
	if req.Category == "" {
		response.WriteError(w, r, http.StatusBadRequest, "category is required", nil)
		return
	}
	if req.Type != "expense" && req.Type != "income" {
		response.WriteError(w, r, http.StatusBadRequest, "type must be expense or income", nil)
		return
	}

	txDate := time.Now()
	if req.TransactionDate != "" {
		parsed, err := time.Parse("2006-01-02", req.TransactionDate)
		if err != nil {
			response.WriteError(w, r, http.StatusBadRequest, "invalid transactionDate format, use YYYY-MM-DD", err)
			return
		}
		txDate = parsed
	}

	tx, err := h.svc.Create(r.Context(), email, application.CreateTransactionInput{
		AmountCents:     req.AmountCents,
		Category:        req.Category,
		Description:     req.Description,
		Type:            domain.TransactionType(req.Type),
		TransactionDate: txDate,
	})
	if err != nil {
		response.WriteError(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, apiResponse{OK: true, Data: toTransactionResponse(tx)})
}

func (h *FinanceHandler) List(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	from := time.Now().AddDate(0, -1, 0)
	to := time.Now()

	if fromStr != "" {
		parsed, err := time.Parse("2006-01-02", fromStr)
		if err == nil {
			from = parsed
		}
	}
	if toStr != "" {
		parsed, err := time.Parse("2006-01-02", toStr)
		if err == nil {
			to = parsed
		}
	}

	txs, err := h.svc.List(r.Context(), email, application.TransactionFilter{
		From: from,
		To:   to,
	})
	if err != nil {
		response.WriteError(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}

	resp := make([]transactionResponse, 0, len(txs))
	for _, tx := range txs {
		resp = append(resp, toTransactionResponse(tx))
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{Data: resp})
}

func (h *FinanceHandler) TodayTotal(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())

	total, err := h.svc.GetTodayTotal(r.Context(), email)
	if err != nil {
		response.WriteError(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{Data: total})
}

func (h *FinanceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), id, email); err != nil {
		response.WriteError(w, r, http.StatusNotFound, "transaction not found", err)
		return
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{OK: true})
}
