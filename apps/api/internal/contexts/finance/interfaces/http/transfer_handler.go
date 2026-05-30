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

type TransferHandler struct {
	svc *application.TransferService
}

func NewTransferHandler(svc *application.TransferService) *TransferHandler {
	return &TransferHandler{svc: svc}
}

func (h *TransferHandler) Routes(r chi.Router) {
	r.Get("/", h.List)
	r.Post("/", h.Create)
}

// --- Request types ---

type createTransferRequest struct {
	FromWalletID string `json:"fromWalletId"`
	ToWalletID   string `json:"toWalletId"`
	AmountCents  int64  `json:"amountCents"`
	Description  string `json:"description"`
	TransferDate string `json:"transferDate"`
}

// --- Response types ---

type transferResponse struct {
	ID           string `json:"id"`
	FromWalletID string `json:"fromWalletId"`
	ToWalletID   string `json:"toWalletId"`
	AmountCents  int64  `json:"amountCents"`
	Description  string `json:"description"`
	TransferDate string `json:"transferDate"`
	CreatedAt    string `json:"createdAt"`
}

func toTransferResponse(t *domain.WalletTransfer) transferResponse {
	return transferResponse{
		ID:           t.ID,
		FromWalletID: t.FromWalletID,
		ToWalletID:   t.ToWalletID,
		AmountCents:  t.AmountCents,
		Description:  t.Description,
		TransferDate: t.TransferDate.Format("2006-01-02"),
		CreatedAt:    t.CreatedAt.Format(time.RFC3339),
	}
}

// --- Handlers ---

func (h *TransferHandler) Create(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())

	var req createTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	transferDate, err := time.Parse("2006-01-02", req.TransferDate)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid transferDate format, use YYYY-MM-DD", err)
		return
	}

	transfer, err := h.svc.Create(r.Context(), email, application.CreateTransferInput{
		FromWalletID: req.FromWalletID,
		ToWalletID:   req.ToWalletID,
		AmountCents:  req.AmountCents,
		Description:  req.Description,
		TransferDate: transferDate,
	})
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, apiResponse{OK: true, Data: toTransferResponse(transfer)})
}

func (h *TransferHandler) List(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())

	transfers, err := h.svc.List(r.Context(), email, application.TransferFilter{})
	if err != nil {
		response.WriteError(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}

	resp := make([]transferResponse, 0, len(transfers))
	for _, t := range transfers {
		resp = append(resp, toTransferResponse(t))
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{Data: resp})
}
