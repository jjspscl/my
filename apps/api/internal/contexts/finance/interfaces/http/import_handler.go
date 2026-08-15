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

type ImportHandler struct {
	svc *application.ImportService
}

func NewImportHandler(svc *application.ImportService) *ImportHandler {
	return &ImportHandler{svc: svc}
}

func (h *ImportHandler) Routes(r chi.Router) {
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Delete("/{id}", h.Rollback)
}

type createImportRowRequest struct {
	SourceReference string `json:"sourceReference"`
	OccurredAt      string `json:"occurredAt"`
	AmountCents     int64  `json:"amountCents"`
	Kind            string `json:"kind"`
	Category        string `json:"category"`
	Description     string `json:"description"`
	Counterparty    string `json:"counterparty,omitempty"`
	CounterWalletID string `json:"counterWalletId,omitempty"`
}

type createImportRequest struct {
	Provider            string                    `json:"provider"`
	FileFingerprint     string                    `json:"fileFingerprint"`
	StatementFrom       string                    `json:"statementFrom"`
	StatementTo         string                    `json:"statementTo"`
	WalletID            string                    `json:"walletId,omitempty"`
	CreateWallet        *createWalletImportRequest `json:"createWallet,omitempty"`
	OpeningBalanceCents int64                     `json:"openingBalanceCents"`
	EndingBalanceCents  int64                     `json:"endingBalanceCents"`
	Reconciliation      string                    `json:"reconciliation"`
	Rows                []createImportRowRequest  `json:"rows"`
}

type createWalletImportRequest struct {
	Name                string `json:"name"`
	OpeningBalanceCents int64  `json:"openingBalanceCents"`
}

type importEntryResponse struct {
	ID              string `json:"id"`
	SourceReference string `json:"sourceReference"`
	OccurredAt      string `json:"occurredAt"`
	AmountCents     int64  `json:"amountCents"`
	Kind            string `json:"kind"`
	Category        string `json:"category"`
	Description     string `json:"description"`
	Counterparty    string `json:"counterparty,omitempty"`
	CounterWalletID string `json:"counterWalletId,omitempty"`
	Outcome         string `json:"outcome"`
	EntityType      string `json:"entityType,omitempty"`
	EntityID        string `json:"entityId,omitempty"`
}

type importBatchResponse struct {
	ID                  string                `json:"id"`
	Provider            string                `json:"provider"`
	FileFingerprint     string                `json:"fileFingerprint"`
	StatementFrom       string                `json:"statementFrom"`
	StatementTo         string                `json:"statementTo"`
	WalletID            string                `json:"walletId"`
	CreatedWalletID     string                `json:"createdWalletId,omitempty"`
	OpeningBalanceCents int64                 `json:"openingBalanceCents"`
	EndingBalanceCents  int64                 `json:"endingBalanceCents"`
	Reconciliation      string                `json:"reconciliation"`
	Status              string                `json:"status"`
	Summary             domain.ImportSummary  `json:"summary"`
	CreatedAt           string                `json:"createdAt"`
	RolledBackAt        string                `json:"rolledBackAt,omitempty"`
	Entries             []importEntryResponse `json:"entries,omitempty"`
}

func toImportBatchResponse(b *domain.ImportBatch, entries []*domain.ImportEntry) importBatchResponse {
	resp := importBatchResponse{
		ID:                  b.ID,
		Provider:            b.Provider,
		FileFingerprint:     b.FileFingerprint,
		StatementFrom:       b.StatementFrom.Format("2006-01-02"),
		StatementTo:         b.StatementTo.Format("2006-01-02"),
		WalletID:            b.WalletID,
		CreatedWalletID:     b.CreatedWalletID,
		OpeningBalanceCents: b.OpeningBalanceCents,
		EndingBalanceCents:  b.EndingBalanceCents,
		Reconciliation:      b.Reconciliation,
		Status:              b.Status,
		Summary:             b.Summary,
		CreatedAt:           b.CreatedAt.Format(time.RFC3339),
	}
	if b.RolledBackAt != nil {
		resp.RolledBackAt = b.RolledBackAt.Format(time.RFC3339)
	}
	if entries != nil {
		resp.Entries = make([]importEntryResponse, 0, len(entries))
		for _, e := range entries {
			resp.Entries = append(resp.Entries, importEntryResponse{
				ID:              e.ID,
				SourceReference: e.SourceReference,
				OccurredAt:      e.OccurredAt.Format(time.RFC3339),
				AmountCents:     e.AmountCents,
				Kind:            e.Kind,
				Category:        e.Category,
				Description:     e.Description,
				Counterparty:    e.Counterparty,
				CounterWalletID: e.CounterWalletID,
				Outcome:         e.Outcome,
				EntityType:      e.EntityType,
				EntityID:        e.EntityID,
			})
		}
	}
	return resp
}

func (h *ImportHandler) Create(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	r.Body = http.MaxBytesReader(w, r.Body, 4<<20)

	var req createImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	from, err := time.Parse("2006-01-02", req.StatementFrom)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid statementFrom format, use YYYY-MM-DD", err)
		return
	}
	to, err := time.Parse("2006-01-02", req.StatementTo)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid statementTo format, use YYYY-MM-DD", err)
		return
	}

	rows := make([]application.ImportRowInput, 0, len(req.Rows))
	for _, row := range req.Rows {
		occurredAt, err := time.Parse(time.RFC3339, row.OccurredAt)
		if err != nil {
			response.WriteError(w, r, http.StatusBadRequest,
				"invalid occurredAt for row "+row.SourceReference+", use RFC3339", err)
			return
		}
		rows = append(rows, application.ImportRowInput{
			SourceReference: row.SourceReference,
			OccurredAt:      occurredAt,
			AmountCents:     row.AmountCents,
			Kind:            row.Kind,
			Category:        row.Category,
			Description:     row.Description,
			Counterparty:    row.Counterparty,
			CounterWalletID: row.CounterWalletID,
		})
	}

	var createWallet *application.CreateWalletForImport
	if req.CreateWallet != nil {
		createWallet = &application.CreateWalletForImport{
			Name:                req.CreateWallet.Name,
			OpeningBalanceCents: req.CreateWallet.OpeningBalanceCents,
		}
	}

	batch, err := h.svc.Create(r.Context(), email, application.CreateImportInput{
		Provider:            req.Provider,
		FileFingerprint:     req.FileFingerprint,
		StatementFrom:       from,
		StatementTo:         to,
		WalletID:            req.WalletID,
		CreateWallet:        createWallet,
		OpeningBalanceCents: req.OpeningBalanceCents,
		EndingBalanceCents:  req.EndingBalanceCents,
		Reconciliation:      req.Reconciliation,
		Rows:                rows,
	})
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, apiResponse{OK: true, Data: toImportBatchResponse(batch, nil)})
}

func (h *ImportHandler) List(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())

	batches, err := h.svc.List(r.Context(), email, application.ImportFilter{})
	if err != nil {
		response.WriteError(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}

	resp := make([]importBatchResponse, 0, len(batches))
	for _, b := range batches {
		resp = append(resp, toImportBatchResponse(b, nil))
	}
	response.WriteJSON(w, http.StatusOK, apiResponse{Data: resp})
}

func (h *ImportHandler) Get(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")

	batch, entries, err := h.svc.Get(r.Context(), id, email)
	if err != nil {
		response.WriteError(w, r, http.StatusNotFound, "import not found", err)
		return
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{Data: toImportBatchResponse(batch, entries)})
}

func (h *ImportHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")

	removed, err := h.svc.Rollback(r.Context(), id, email)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]int{"removedEntities": removed}})
}
