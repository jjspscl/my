package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/jjspscl/my/internal/contexts/intelligence/application"
	"github.com/jjspscl/my/internal/contexts/intelligence/domain"
	"github.com/jjspscl/my/internal/shared/middleware"
	"github.com/jjspscl/my/internal/shared/response"
)

// IntelligenceHandler exposes provider/connector settings (write-only
// credentials) and the import-analysis endpoints.
type IntelligenceHandler struct {
	settings *application.SettingsService
	analysis *application.AnalysisService
}

func NewIntelligenceHandler(settings *application.SettingsService, analysis *application.AnalysisService) *IntelligenceHandler {
	return &IntelligenceHandler{settings: settings, analysis: analysis}
}

type apiResp struct {
	OK    bool        `json:"ok,omitempty"`
	Error string      `json:"error,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

func (h *IntelligenceHandler) Routes(r chi.Router) {
	r.Route("/providers", func(r chi.Router) {
		r.Get("/", h.ListProviders)
		r.Post("/", h.CreateProvider)
		r.Put("/{id}", h.UpdateProvider)
		r.Delete("/{id}", h.DeleteProvider)
		r.Put("/{id}/credential", h.SaveProviderCredential)
		r.Post("/{id}/test", h.TestProvider)
	})
	r.Route("/connectors", func(r chi.Router) {
		r.Get("/", h.ListConnectors)
		r.Post("/", h.CreateConnector)
		r.Put("/{id}", h.UpdateConnector)
		r.Delete("/{id}", h.DeleteConnector)
		r.Put("/{id}/credential", h.SaveConnectorCredential)
		r.Post("/{id}/test", h.TestConnector)
	})
}

// AnalysisRoutes mount under /finance/imports/analyses (inside the
// authenticated + CSRF group).
func (h *IntelligenceHandler) AnalysisRoutes(r chi.Router) {
	r.Post("/", h.CreateAnalysis)
	r.Get("/{id}", h.GetAnalysis)
	r.Delete("/{id}", h.CancelAnalysis)
}

// ---- providers ----

type providerResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ProviderType string `json:"providerType"`
	BaseURL      string `json:"baseUrl,omitempty"`
	Model        string `json:"model"`
	Enabled      bool   `json:"enabled"`
	Priority     int    `json:"priority"`
	MaxTokens    int    `json:"maxTokens,omitempty"`
	TimeoutMS    int    `json:"timeoutMs"`
	AllowLocal   bool   `json:"allowLocal"`
	HasCredential bool  `json:"hasCredential"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type createProviderRequest struct {
	Name         string `json:"name"`
	ProviderType string `json:"providerType"`
	BaseURL      string `json:"baseUrl,omitempty"`
	Model        string `json:"model"`
	MaxTokens    int    `json:"maxTokens,omitempty"`
	TimeoutMS    int    `json:"timeoutMs,omitempty"`
	AllowLocal   bool   `json:"allowLocal,omitempty"`
	APIKey       string `json:"apiKey,omitempty"`
}

type updateProviderRequest struct {
	Name         string `json:"name"`
	ProviderType string `json:"providerType"`
	BaseURL      string `json:"baseUrl,omitempty"`
	Model        string `json:"model"`
	MaxTokens    int    `json:"maxTokens,omitempty"`
	TimeoutMS    int    `json:"timeoutMs,omitempty"`
	AllowLocal   bool   `json:"allowLocal,omitempty"`
	Enabled      bool   `json:"enabled"`
}

type credentialRequest struct {
	Value string `json:"value"`
}

func (h *IntelligenceHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	profiles, err := h.settings.ListProviders(r.Context(), email)
	if err != nil {
		response.WriteError(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	resp := make([]providerResponse, 0, len(profiles))
	for _, p := range profiles {
		hasCred := h.settings.HasCredential(r.Context(), "provider", p.ID)
		resp = append(resp, providerResponse{
			ID: p.ID, Name: p.Name, ProviderType: p.ProviderType, BaseURL: p.BaseURL,
			Model: p.Model, Enabled: p.Enabled, Priority: p.Priority, MaxTokens: p.MaxTokens,
			TimeoutMS: int(p.Timeout.Milliseconds()), AllowLocal: p.AllowLocal,
			HasCredential: hasCred, CreatedAt: p.CreatedAt.Format(time.RFC3339), UpdatedAt: p.UpdatedAt.Format(time.RFC3339),
		})
	}
	response.WriteJSON(w, http.StatusOK, apiResp{Data: resp})
}

func (h *IntelligenceHandler) CreateProvider(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	var req createProviderRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}
	p, err := h.settings.CreateProvider(r.Context(), email, application.CreateProviderInput{
		Name: req.Name, ProviderType: req.ProviderType, BaseURL: req.BaseURL, Model: req.Model,
		MaxTokens: req.MaxTokens, TimeoutMS: req.TimeoutMS, AllowLocal: req.AllowLocal, APIKey: req.APIKey,
	})
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, apiResp{OK: true, Data: p.ID})
}

func (h *IntelligenceHandler) UpdateProvider(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")
	var req updateProviderRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}
	p, err := h.settings.UpdateProvider(r.Context(), email, application.UpdateProviderInput{
		ID: id, Name: req.Name, ProviderType: req.ProviderType, BaseURL: req.BaseURL, Model: req.Model,
		MaxTokens: req.MaxTokens, TimeoutMS: req.TimeoutMS, AllowLocal: req.AllowLocal, Enabled: req.Enabled,
	})
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}
	response.WriteJSON(w, http.StatusOK, apiResp{OK: true, Data: p.ID})
}

func (h *IntelligenceHandler) DeleteProvider(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")
	if err := h.settings.DeleteProvider(r.Context(), email, id); err != nil {
		response.WriteError(w, r, http.StatusNotFound, "provider not found", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, apiResp{OK: true})
}

func (h *IntelligenceHandler) SaveProviderCredential(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")
	var req credentialRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}
	if req.Value == "" {
		response.WriteError(w, r, http.StatusBadRequest, "value is required", nil)
		return
	}
	if err := h.settings.SaveCredential(r.Context(), email, "provider", id, req.Value); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}
	response.WriteJSON(w, http.StatusOK, apiResp{OK: true})
}

func (h *IntelligenceHandler) TestProvider(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")
	if err := h.settings.TestProvider(r.Context(), email, id); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "test failed: "+err.Error(), err)
		return
	}
	response.WriteJSON(w, http.StatusOK, apiResp{OK: true})
}

// ---- connectors ----

type connectorResponse struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Endpoint  string   `json:"endpoint"`
	Enabled   bool     `json:"enabled"`
	Allowlist []string `json:"allowlist"`
	TimeoutMS int      `json:"timeoutMs"`
	HasToken  bool     `json:"hasToken"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
}

type createConnectorRequest struct {
	Name      string   `json:"name"`
	Endpoint  string   `json:"endpoint"`
	Allowlist []string `json:"allowlist"`
	TimeoutMS int      `json:"timeoutMs,omitempty"`
	Token     string   `json:"token,omitempty"`
}

type updateConnectorRequest struct {
	Name      string   `json:"name"`
	Endpoint  string   `json:"endpoint"`
	Allowlist []string `json:"allowlist"`
	TimeoutMS int      `json:"timeoutMs,omitempty"`
	Enabled   bool     `json:"enabled"`
}

func (h *IntelligenceHandler) ListConnectors(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	connectors, err := h.settings.ListConnectors(r.Context(), email)
	if err != nil {
		response.WriteError(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	resp := make([]connectorResponse, 0, len(connectors))
	for _, c := range connectors {
		hasToken := h.settings.HasCredential(r.Context(), "connector", c.ID)
		resp = append(resp, connectorResponse{
			ID: c.ID, Name: c.Name, Endpoint: c.Endpoint, Enabled: c.Enabled,
			Allowlist: c.Allowlist, TimeoutMS: int(c.Timeout.Milliseconds()), HasToken: hasToken,
			CreatedAt: c.CreatedAt.Format(time.RFC3339), UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
		})
	}
	response.WriteJSON(w, http.StatusOK, apiResp{Data: resp})
}

func (h *IntelligenceHandler) CreateConnector(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	var req createConnectorRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}
	c, err := h.settings.CreateConnector(r.Context(), email, application.CreateConnectorInput{
		Name: req.Name, Endpoint: req.Endpoint, Allowlist: req.Allowlist, TimeoutMS: req.TimeoutMS, Token: req.Token,
	})
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, apiResp{OK: true, Data: c.ID})
}

func (h *IntelligenceHandler) UpdateConnector(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")
	var req updateConnectorRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}
	c, err := h.settings.UpdateConnector(r.Context(), email, application.UpdateConnectorInput{
		ID: id, Name: req.Name, Endpoint: req.Endpoint, Allowlist: req.Allowlist, TimeoutMS: req.TimeoutMS, Enabled: req.Enabled,
	})
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}
	response.WriteJSON(w, http.StatusOK, apiResp{OK: true, Data: c.ID})
}

func (h *IntelligenceHandler) DeleteConnector(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")
	if err := h.settings.DeleteConnector(r.Context(), email, id); err != nil {
		response.WriteError(w, r, http.StatusNotFound, "connector not found", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, apiResp{OK: true})
}

func (h *IntelligenceHandler) SaveConnectorCredential(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")
	var req credentialRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}
	if req.Value == "" {
		response.WriteError(w, r, http.StatusBadRequest, "value is required", nil)
		return
	}
	if err := h.settings.SaveCredential(r.Context(), email, "connector", id, req.Value); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}
	response.WriteJSON(w, http.StatusOK, apiResp{OK: true})
}

func (h *IntelligenceHandler) TestConnector(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")
	if err := h.settings.TestConnector(r.Context(), email, id); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "test failed: "+err.Error(), err)
		return
	}
	response.WriteJSON(w, http.StatusOK, apiResp{OK: true})
}

// ---- analysis ----

type createAnalysisRequest struct {
	ScopeID string `json:"scopeId"`
	Rows    []struct {
		SourceReference string `json:"sourceReference"`
		Description     string `json:"description"`
		AmountCents     int64  `json:"amountCents"`
		Kind            string `json:"kind"`
	} `json:"rows"`
}

type suggestionResponse struct {
	ID         string  `json:"id"`
	RunID      string  `json:"runId"`
	TargetKey  string  `json:"targetKey"`
	Field      string  `json:"field"`
	Value      string  `json:"value"`
	Confidence float64 `json:"confidence"`
	Status     string  `json:"status"`
	Rationale  string  `json:"rationale,omitempty"`
	Evidence   []struct {
		Source string `json:"source"`
		Detail string `json:"detail"`
	} `json:"evidence"`
}

type runResponse struct {
	ID        string `json:"id"`
	Scope     string `json:"scope"`
	ScopeID   string `json:"scopeId"`
	Status    string `json:"status"`
	Model     string `json:"model,omitempty"`
	Error     string `json:"error,omitempty"`
	CreatedAt string `json:"createdAt"`
	CompletedAt string `json:"completedAt,omitempty"`
}

func toRunResponse(r *domain.AgentRun) runResponse {
	resp := runResponse{ID: r.ID, Scope: r.Scope, ScopeID: r.ScopeID, Status: r.Status, Model: r.Model, Error: r.Error, CreatedAt: r.CreatedAt.Format(time.RFC3339)}
	if r.CompletedAt != nil {
		resp.CompletedAt = r.CompletedAt.Format(time.RFC3339)
	}
	return resp
}

func (h *IntelligenceHandler) CreateAnalysis(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	var req createAnalysisRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}
	if req.ScopeID == "" {
		response.WriteError(w, r, http.StatusBadRequest, "scopeId is required", nil)
		return
	}
	rows := make([]application.AnalysisRow, 0, len(req.Rows))
	for _, row := range req.Rows {
		rows = append(rows, application.AnalysisRow{
			SourceReference: row.SourceReference,
			Description:     row.Description,
			AmountCents:     row.AmountCents,
			Kind:            row.Kind,
		})
	}
	result, err := h.analysis.AnalyzeImport(r.Context(), email, req.ScopeID, rows)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, apiResp{OK: true, Data: toAnalysisData(result)})
}

func (h *IntelligenceHandler) GetAnalysis(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")
	result, err := h.analysis.Get(r.Context(), email, id)
	if err != nil {
		response.WriteError(w, r, http.StatusNotFound, "analysis not found", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, apiResp{Data: toAnalysisData(result)})
}

func (h *IntelligenceHandler) CancelAnalysis(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetEmailFromContext(r.Context())
	id := chi.URLParam(r, "id")
	if err := h.analysis.Cancel(r.Context(), email, id); err != nil {
		response.WriteError(w, r, http.StatusNotFound, "analysis not found", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, apiResp{OK: true})
}

func toAnalysisData(result *application.AnalysisResult) map[string]interface{} {
	suggestions := make([]suggestionResponse, 0, len(result.Suggestions))
	for _, s := range result.Suggestions {
		evidence := make([]struct {
			Source string `json:"source"`
			Detail string `json:"detail"`
		}, 0, len(s.Evidence))
		for _, e := range s.Evidence {
			evidence = append(evidence, struct {
				Source string `json:"source"`
				Detail string `json:"detail"`
			}{Source: string(e.Source), Detail: e.Detail})
		}
		suggestions = append(suggestions, suggestionResponse{
			ID: s.ID, RunID: s.RunID, TargetKey: s.TargetKey, Field: s.Field, Value: s.Value,
			Confidence: s.Confidence, Status: s.Status, Rationale: s.Rationale, Evidence: evidence,
		})
	}
	return map[string]interface{}{
		"run":         toRunResponse(result.Run),
		"suggestions": suggestions,
	}
}
