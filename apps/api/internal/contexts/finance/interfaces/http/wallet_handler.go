package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/jjspscl/my/internal/contexts/finance/application"
	"github.com/jjspscl/my/internal/contexts/finance/domain"
	"github.com/jjspscl/my/internal/shared/middleware"
	"github.com/jjspscl/my/internal/shared/response"
)

type WalletHandler struct {
	svc *application.WalletService
}

func NewWalletHandler(svc *application.WalletService) *WalletHandler {
	return &WalletHandler{svc: svc}
}

func (h *WalletHandler) Routes(r chi.Router) {
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Archive)
}

// --- Request types ---

type createWalletRequest struct {
	Name                string `json:"name"`
	Kind                string `json:"kind"`
	OpeningBalanceCents int64  `json:"openingBalanceCents"`
}

type updateWalletRequest struct {
	Name                string `json:"name"`
	Kind                string `json:"kind"`
	OpeningBalanceCents int64  `json:"openingBalanceCents"`
}

// --- Response types ---

type walletResponse struct {
	ID                    string  `json:"id"`
	Name                  string  `json:"name"`
	Kind                  string  `json:"kind"`
	Currency              string  `json:"currency"`
	OpeningBalanceCents   int64   `json:"openingBalanceCents"`
	BalanceCents          int64   `json:"balanceCents"`
	IncomeCents           int64   `json:"incomeCents"`
	ExpenseCents          int64   `json:"expenseCents"`
	IncomingTransferCents int64   `json:"incomingTransferCents"`
	OutgoingTransferCents int64   `json:"outgoingTransferCents"`
	IsDefault             bool    `json:"isDefault"`
	ArchivedAt            *string `json:"archivedAt,omitempty"`
	CreatedAt             string  `json:"createdAt"`
	UpdatedAt             string  `json:"updatedAt"`
}

func toWalletResponse(w *domain.Wallet) walletResponse {
	var archivedAt *string
	if w.ArchivedAt != nil {
		s := w.ArchivedAt.Format("2006-01-02")
		archivedAt = &s
	}
	return walletResponse{
		ID:                  w.ID,
		Name:                w.Name,
		Kind:                string(w.Kind),
		Currency:            w.Currency,
		OpeningBalanceCents: w.OpeningBalanceCents,
		IsDefault:           w.IsDefault,
		ArchivedAt:          archivedAt,
		CreatedAt:           w.CreatedAt.Format("2006-01-02"),
		UpdatedAt:           w.UpdatedAt.Format("2006-01-02"),
	}
}

func toWalletBalanceResponse(b *domain.WalletBalance) walletResponse {
	var archivedAt *string
	if b.Wallet.ArchivedAt != nil {
		s := b.Wallet.ArchivedAt.Format("2006-01-02")
		archivedAt = &s
	}
	return walletResponse{
		ID:                    b.Wallet.ID,
		Name:                  b.Wallet.Name,
		Kind:                  string(b.Wallet.Kind),
		Currency:              b.Wallet.Currency,
		OpeningBalanceCents:   b.Wallet.OpeningBalanceCents,
		BalanceCents:          b.BalanceCents,
		IncomeCents:           b.IncomeCents,
		ExpenseCents:          b.ExpenseCents,
		IncomingTransferCents: b.IncomingTransferCents,
		OutgoingTransferCents: b.OutgoingTransferCents,
		IsDefault:             b.Wallet.IsDefault,
		ArchivedAt:            archivedAt,
		CreatedAt:             b.Wallet.CreatedAt.Format("2006-01-02"),
		UpdatedAt:             b.Wallet.UpdatedAt.Format("2006-01-02"),
	}
}

// --- Handlers ---

func (h *WalletHandler) Create(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())

	var req createWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	wallet, err := h.svc.Create(r.Context(), email, "PHP", application.CreateWalletInput{
		Name:                req.Name,
		Kind:                domain.WalletKind(req.Kind),
		OpeningBalanceCents: req.OpeningBalanceCents,
	})
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, apiResponse{OK: true, Data: toWalletResponse(wallet)})
}

func (h *WalletHandler) List(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())

	balances, err := h.svc.ListWithBalances(r.Context(), email)
	if err != nil {
		response.WriteError(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}

	resp := make([]walletResponse, 0, len(balances))
	for _, b := range balances {
		resp = append(resp, toWalletBalanceResponse(b))
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{Data: resp})
}

func (h *WalletHandler) Update(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")

	var req updateWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	wallet, err := h.svc.Update(r.Context(), email, application.UpdateWalletInput{
		ID:                  id,
		Name:                req.Name,
		Kind:                domain.WalletKind(req.Kind),
		OpeningBalanceCents: req.OpeningBalanceCents,
	})
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{OK: true, Data: toWalletResponse(wallet)})
}

func (h *WalletHandler) Archive(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.svc.Archive(r.Context(), id, email); err != nil {
		response.WriteError(w, r, http.StatusNotFound, "wallet not found", err)
		return
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{OK: true})
}
