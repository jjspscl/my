package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/jjspscl/my/internal/contexts/finance/application"
	"github.com/jjspscl/my/internal/contexts/finance/domain"
	"github.com/jjspscl/my/internal/shared/middleware"
	"github.com/jjspscl/my/internal/shared/response"
)

type FinanceHandler struct {
	svc             *application.TransactionService
	defaultCurrency string
}

func NewFinanceHandler(svc *application.TransactionService, defaultCurrency string) *FinanceHandler {
	return &FinanceHandler{svc: svc, defaultCurrency: defaultCurrency}
}

func (h *FinanceHandler) Routes(r chi.Router) {
	r.Get("/transactions", h.List)
	r.Post("/transactions", h.Create)
	r.Patch("/transactions/{id}", h.Update)
	r.Post("/transactions/bulk", h.BulkUpdate)
	r.Post("/transactions/bulk-delete", h.BulkDelete)
	r.Get("/transactions/today-total", h.TodayTotal)
	r.Delete("/transactions/{id}", h.Delete)
}

type createTransactionRequest struct {
	AmountCents     int64  `json:"amountCents"`
	Category        string `json:"category"`
	Description     string `json:"description"`
	Type            string `json:"type"`
	WalletID        string `json:"walletId"`
	TransactionDate string `json:"transactionDate"`
	IdempotencyKey  string `json:"idempotencyKey,omitempty"`
}

type updateTransactionRequest struct {
	AmountCents     *int64  `json:"amountCents"`
	Category        *string `json:"category"`
	Description     *string `json:"description"`
	Type            *string `json:"type"`
	WalletID        *string `json:"walletId"`
	TransactionDate *string `json:"transactionDate"`
}

type bulkTransactionItemRequest struct {
	ID       string `json:"id"`
	Revision int    `json:"revision"`
}

// bulkUpdateRequest is the collection patch: one shared patch applied to many
// transactions, each with its own revision precondition.
type bulkUpdateRequest struct {
	Items []bulkTransactionItemRequest `json:"items"`
	Patch struct {
		Category *string `json:"category"`
		Type     *string `json:"type"`
		WalletID *string `json:"walletId"`
	} `json:"patch"`
}

type bulkDeleteRequest struct {
	Items []bulkTransactionItemRequest `json:"items"`
}

type transactionResponse struct {
	ID              string `json:"id"`
	AmountCents     int64  `json:"amountCents"`
	Currency        string `json:"currency"`
	Category        string `json:"category"`
	Description     string `json:"description"`
	Type            string `json:"type"`
	WalletID        string `json:"walletId,omitempty"`
	WalletName      string `json:"walletName,omitempty"`
	TransactionDate string `json:"transactionDate"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt,omitempty"`
	Revision        int    `json:"revision"`
	Imported        bool   `json:"imported"`
	ImportProvider  string `json:"importProvider,omitempty"`
}

type apiResponse struct {
	OK    bool        `json:"ok,omitempty"`
	Error string      `json:"error,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

func toTransactionResponse(t *domain.Transaction) transactionResponse {
	resp := transactionResponse{
		ID:              t.ID,
		AmountCents:     t.AmountCents,
		Currency:        t.Currency,
		Category:        t.Category,
		Description:     t.Description,
		Type:            string(t.Type),
		WalletID:        t.WalletID,
		WalletName:      t.WalletName,
		TransactionDate: t.TransactionDate.Format("2006-01-02"),
		CreatedAt:       t.CreatedAt.Format(time.RFC3339),
		Revision:        t.Revision,
		Imported:        t.Imported,
		ImportProvider:  t.ImportProvider,
	}
	if !t.UpdatedAt.IsZero() {
		resp.UpdatedAt = t.UpdatedAt.Format(time.RFC3339)
	}
	return resp
}

// parseIfMatch parses an If-Match header in the form `"3"` and returns the
// revision. missing reports whether the header was absent.
func parseIfMatch(h http.Header) (revision int, present bool, err error) {
	v := h.Get("If-Match")
	if v == "" {
		return 0, false, nil
	}
	v = strings.TrimSpace(v)
	if !strings.HasPrefix(v, `"`) || !strings.HasSuffix(v, `"`) {
		return 0, true, fmt.Errorf("If-Match must be a quoted integer, e.g. \"3\"")
	}
	n, err := strconv.Atoi(strings.Trim(v, `"`))
	if err != nil || n <= 0 {
		return 0, true, fmt.Errorf("If-Match must be a quoted positive integer, e.g. \"3\"")
	}
	return n, true, nil
}

func (h *FinanceHandler) Create(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())

	var req createTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
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
		WalletID:        req.WalletID,
		TransactionDate: txDate,
		IdempotencyKey:  req.IdempotencyKey,
	})
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
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

	total, err := h.svc.GetTodayTotal(r.Context(), email, h.defaultCurrency)
	if err != nil {
		response.WriteError(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{Data: total})
}

// Update handles PATCH /transactions/{id}. The If-Match header carries the
// revision the client last saw; a mismatched revision yields 412 so stale
// edits never silently clobber newer state.
func (h *FinanceHandler) Update(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")

	revision, present, err := parseIfMatch(r.Header)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}
	if !present {
		response.WriteError(w, r, http.StatusPreconditionRequired, "If-Match header is required for PATCH", errors.New("missing If-Match"))
		return
	}

	var req updateTransactionRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	input := application.UpdateTransactionInput{ExpectedRevision: revision}
	if req.AmountCents != nil {
		input.AmountCents = req.AmountCents
	}
	if req.Category != nil {
		input.Category = req.Category
	}
	if req.Description != nil {
		input.Description = req.Description
	}
	if req.Type != nil {
		t := domain.TransactionType(*req.Type)
		input.Type = &t
	}
	if req.WalletID != nil {
		input.WalletID = req.WalletID
	}
	if req.TransactionDate != nil {
		parsed, err := time.Parse("2006-01-02", *req.TransactionDate)
		if err != nil {
			response.WriteError(w, r, http.StatusBadRequest, "invalid transactionDate format, use YYYY-MM-DD", err)
			return
		}
		input.TransactionDate = &parsed
	}

	tx, err := h.svc.Update(r.Context(), email, id, input)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrStaleRevision):
			response.WriteError(w, r, http.StatusPreconditionFailed, "transaction was modified elsewhere; reload and retry", err)
		case err.Error() == "transaction not found":
			response.WriteError(w, r, http.StatusNotFound, "transaction not found", err)
		default:
			response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		}
		return
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{OK: true, Data: toTransactionResponse(tx)})
}

// BulkUpdate handles POST /transactions/bulk: applies one category/type/wallet
// patch to many transactions atomically. Collection ops use POST because
// If-Match cannot express per-row preconditions.
func (h *FinanceHandler) BulkUpdate(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())

	var req bulkUpdateRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	items := make([]application.BulkItem, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, application.BulkItem{ID: it.ID, Revision: it.Revision})
	}

	patch := application.BulkPatch{Category: req.Patch.Category, WalletID: req.Patch.WalletID}
	if req.Patch.Type != nil {
		t := domain.TransactionType(*req.Patch.Type)
		patch.Type = &t
	}

	txs, err := h.svc.UpdateMany(r.Context(), email, items, patch)
	if err != nil {
		writeBulkError(w, r, err)
		return
	}

	resp := make([]transactionResponse, 0, len(txs))
	for _, tx := range txs {
		resp = append(resp, toTransactionResponse(tx))
	}
	response.WriteJSON(w, http.StatusOK, apiResponse{
		OK:   true,
		Data: map[string]any{"updated": len(resp), "transactions": resp},
	})
}

// BulkDelete handles POST /transactions/bulk-delete: removes many transactions
// atomically with per-row revision preconditions.
func (h *FinanceHandler) BulkDelete(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())

	var req bulkDeleteRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	items := make([]application.BulkItem, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, application.BulkItem{ID: it.ID, Revision: it.Revision})
	}

	deleted, err := h.svc.DeleteMany(r.Context(), email, items)
	if err != nil {
		writeBulkError(w, r, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{
		OK:   true,
		Data: map[string]any{"deleted": deleted},
	})
}

// writeBulkError maps bulk operation failures onto HTTP status codes.
func writeBulkError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrStaleRevision):
		response.WriteError(w, r, http.StatusPreconditionFailed, "one or more transactions changed elsewhere; reload and retry", err)
	case err.Error() == "transaction not found":
		response.WriteError(w, r, http.StatusNotFound, "transaction not found", err)
	default:
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
	}
}

func (h *FinanceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")

	var err error
	if revision, present, perr := parseIfMatch(r.Header); perr != nil {
		response.WriteError(w, r, http.StatusBadRequest, perr.Error(), perr)
		return
	} else if present {
		// Browser/API clients send the revision for safe deletes; the MCP
		// path deletes without a precondition.
		err = h.svc.DeleteAtRevision(r.Context(), id, email, revision)
	} else {
		err = h.svc.Delete(r.Context(), id, email)
	}

	if err != nil {
		switch {
		case errors.Is(err, domain.ErrStaleRevision):
			response.WriteError(w, r, http.StatusPreconditionFailed, "transaction was modified elsewhere; reload and retry", err)
		case err.Error() == "transaction not found":
			response.WriteError(w, r, http.StatusNotFound, "transaction not found", err)
		default:
			response.WriteError(w, r, http.StatusInternalServerError, "internal error", err)
		}
		return
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{OK: true})
}
