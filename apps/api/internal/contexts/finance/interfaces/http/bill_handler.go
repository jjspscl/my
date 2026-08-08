package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/jjspscl/my/internal/contexts/finance/application"
	"github.com/jjspscl/my/internal/contexts/finance/domain"
	"github.com/jjspscl/my/internal/shared/middleware"
	"github.com/jjspscl/my/internal/shared/response"
)

type BillHandler struct {
	svc *application.BillService
}

func NewBillHandler(svc *application.BillService) *BillHandler {
	return &BillHandler{svc: svc}
}

func (h *BillHandler) Routes(r chi.Router) {
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Get("/upcoming", h.Upcoming)
	r.Post("/{id}/pay", h.MarkPaid)
}

// --- Request types ---

type createBillRequest struct {
	Name         string  `json:"name"`
	Category     string  `json:"category"`
	AmountCents  int64   `json:"amountCents"`
	Currency     string  `json:"currency"`
	Frequency    string  `json:"frequency"`
	DayOfMonth   int     `json:"dayOfMonth"`
	StartDate    string  `json:"startDate"`
	EndDate      *string `json:"endDate,omitempty"`
	AutoMatch    bool    `json:"autoMatch"`
	MatchPattern *string `json:"matchPattern,omitempty"`
}

type updateBillRequest struct {
	Name         string  `json:"name"`
	Category     string  `json:"category"`
	AmountCents  int64   `json:"amountCents"`
	Currency     string  `json:"currency"`
	Frequency    string  `json:"frequency"`
	DayOfMonth   int     `json:"dayOfMonth"`
	StartDate    string  `json:"startDate"`
	EndDate      *string `json:"endDate,omitempty"`
	AutoMatch    bool    `json:"autoMatch"`
	MatchPattern *string `json:"matchPattern,omitempty"`
}

type markPaidRequest struct {
	DueDate       string  `json:"dueDate"`
	TransactionID *string `json:"transactionId,omitempty"`
}

// --- Response types ---

type billResponse struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Category     string  `json:"category"`
	AmountCents  int64   `json:"amountCents"`
	Currency     string  `json:"currency"`
	Frequency    string  `json:"frequency"`
	DayOfMonth   int     `json:"dayOfMonth"`
	StartDate    string  `json:"startDate"`
	EndDate      *string `json:"endDate,omitempty"`
	AutoMatch    bool    `json:"autoMatch"`
	MatchPattern *string `json:"matchPattern,omitempty"`
	CreatedAt    string  `json:"createdAt"`
	UpdatedAt    string  `json:"updatedAt"`
}

type upcomingBillResponse struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Category        string  `json:"category"`
	AmountCents     int64   `json:"amountCents"`
	Frequency       string  `json:"frequency"`
	DayOfMonth      int     `json:"dayOfMonth"`
	DueDate         string  `json:"dueDate"`
	Status          string  `json:"status"`
	PaidAmountCents *int64  `json:"paidAmountCents,omitempty"`
	PaidDate        *string `json:"paidDate,omitempty"`
	AutoMatch       bool    `json:"autoMatch"`
}

// --- Helpers ---

func toBillResponse(b *domain.RecurringBill) billResponse {
	var endDate *string
	if b.EndDate != nil {
		s := b.EndDate.Format("2006-01-02")
		endDate = &s
	}
	return billResponse{
		ID:           b.ID,
		Name:         b.Name,
		Category:     b.Category,
		AmountCents:  b.AmountCents,
		Currency:     b.Currency,
		Frequency:    string(b.Frequency),
		DayOfMonth:   b.DayOfMonth,
		StartDate:    b.StartDate.Format("2006-01-02"),
		EndDate:      endDate,
		AutoMatch:    b.AutoMatch,
		MatchPattern: b.MatchPattern,
		CreatedAt:    b.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    b.UpdatedAt.Format(time.RFC3339),
	}
}

// --- Handlers ---

func (h *BillHandler) Create(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())

	var req createBillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid startDate format, use YYYY-MM-DD", err)
		return
	}

	var endDate *time.Time
	if req.EndDate != nil {
		parsed, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			response.WriteError(w, r, http.StatusBadRequest, "invalid endDate format, use YYYY-MM-DD", err)
			return
		}
		endDate = &parsed
	}

	bill, err := h.svc.Create(r.Context(), email, application.CreateBillInput{
		Name:         req.Name,
		Category:     req.Category,
		AmountCents:  req.AmountCents,
		Currency:     req.Currency,
		Frequency:    domain.Frequency(req.Frequency),
		DayOfMonth:   req.DayOfMonth,
		StartDate:    startDate,
		EndDate:      endDate,
		AutoMatch:    req.AutoMatch,
		MatchPattern: req.MatchPattern,
	})
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, apiResponse{OK: true, Data: toBillResponse(bill)})
}

func (h *BillHandler) List(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())

	bills, err := h.svc.List(r.Context(), email)
	if err != nil {
		response.WriteError(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}

	resp := make([]billResponse, 0, len(bills))
	for _, b := range bills {
		resp = append(resp, toBillResponse(b))
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{Data: resp})
}

func (h *BillHandler) Update(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")

	var req updateBillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid startDate format, use YYYY-MM-DD", err)
		return
	}

	var endDate *time.Time
	if req.EndDate != nil {
		parsed, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			response.WriteError(w, r, http.StatusBadRequest, "invalid endDate format, use YYYY-MM-DD", err)
			return
		}
		endDate = &parsed
	}

	bill, err := h.svc.Update(r.Context(), email, application.UpdateBillInput{
		ID:           id,
		Name:         req.Name,
		Category:     req.Category,
		AmountCents:  req.AmountCents,
		Currency:     req.Currency,
		Frequency:    domain.Frequency(req.Frequency),
		DayOfMonth:   req.DayOfMonth,
		StartDate:    startDate,
		EndDate:      endDate,
		AutoMatch:    req.AutoMatch,
		MatchPattern: req.MatchPattern,
	})
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{OK: true, Data: toBillResponse(bill)})
}

func (h *BillHandler) Delete(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), id, email); err != nil {
		response.WriteError(w, r, http.StatusNotFound, "bill not found", err)
		return
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{OK: true})
}

func (h *BillHandler) Upcoming(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	daysStr := r.URL.Query().Get("days")
	days := 30
	if daysStr != "" {
		if d, err := parseInt(daysStr); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	upcoming, err := h.svc.GetUpcoming(r.Context(), email, days)
	if err != nil {
		response.WriteError(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}

	resp := make([]upcomingBillResponse, 0, len(upcoming))
	for _, u := range upcoming {
		var paidDate *string
		if u.PaidDate != nil {
			s := u.PaidDate.Format("2006-01-02")
			paidDate = &s
		}
		resp = append(resp, upcomingBillResponse{
			ID:              u.Bill.ID,
			Name:            u.Bill.Name,
			Category:        u.Bill.Category,
			AmountCents:     u.Bill.AmountCents,
			Frequency:       string(u.Bill.Frequency),
			DayOfMonth:      u.Bill.DayOfMonth,
			DueDate:         u.DueDate.Format("2006-01-02"),
			Status:          string(u.Status),
			PaidAmountCents: u.PaidAmountCents,
			PaidDate:        paidDate,
			AutoMatch:       u.Bill.AutoMatch,
		})
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{Data: resp})
}

func (h *BillHandler) MarkPaid(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")

	var req markPaidRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	dueDate, err := time.Parse("2006-01-02", req.DueDate)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid dueDate format, use YYYY-MM-DD", err)
		return
	}

	payment, err := h.svc.MarkPaid(r.Context(), id, email, dueDate, req.TransactionID)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	var paidDate *string
	if payment.PaidDate != nil {
		s := payment.PaidDate.Format("2006-01-02")
		paidDate = &s
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]interface{}{
		"id":       payment.ID,
		"billId":   payment.BillID,
		"dueDate":  payment.DueDate.Format("2006-01-02"),
		"status":   string(payment.Status),
		"paidDate": paidDate,
	}})
}

func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
