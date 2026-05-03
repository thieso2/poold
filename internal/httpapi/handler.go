package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pooly/services/poold/internal/pool"
)

type API struct {
	service *Service
	token   string
}

func New(service *Service, token string) http.Handler {
	api := &API{service: service, token: token}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", api.handleWebUI)
	mux.HandleFunc("GET /health", api.handleHealth)
	mux.HandleFunc("GET /status", api.handleStatus)
	mux.HandleFunc("GET /observations", api.handleObservations)
	mux.HandleFunc("GET /observations/stream", api.handleObservationStream)
	mux.HandleFunc("GET /events", api.handleEvents)
	mux.HandleFunc("GET /events/stream", api.handleEventStream)
	mux.HandleFunc("GET /desired-state", api.handleGetDesiredState)
	mux.HandleFunc("PUT /desired-state", api.handlePutDesiredState)
	mux.HandleFunc("GET /plans", api.handleGetPlans)
	mux.HandleFunc("PUT /plans", api.handlePutPlans)
	mux.HandleFunc("POST /commands", api.handleCommands)
	return api.auth(mux)
}

func (a *API) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/" {
			next.ServeHTTP(w, r)
			return
		}
		if a.token == "" {
			next.ServeHTTP(w, r)
			return
		}
		value := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(value, prefix) ||
			subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(value, prefix)), []byte(a.token)) != 1 {
			writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.service.Health(r.Context()))
}

func (a *API) handleStatus(w http.ResponseWriter, r *http.Request) {
	status, err := a.service.RefreshStatus(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (a *API) handleEvents(w http.ResponseWriter, r *http.Request) {
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	var (
		events []pool.Event
		err    error
	)
	if truthy(r.URL.Query().Get("latest")) {
		events, err = a.service.LatestEvents(r.Context(), limit)
	} else {
		events, err = a.service.Events(r.Context(), after, limit)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (a *API) handleObservations(w http.ResponseWriter, r *http.Request) {
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	var (
		observations []pool.Observation
		err          error
	)
	if truthy(r.URL.Query().Get("latest")) {
		observations, err = a.service.LatestObservations(r.Context(), limit)
	} else {
		observations, err = a.service.Observations(r.Context(), after, limit)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"observations": observations})
}

func (a *API) handleEventStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		events, err := a.service.Events(r.Context(), after, 100)
		if err != nil {
			fmt.Fprintf(w, "event: error\ndata: %q\n\n", err.Error())
			flusher.Flush()
			return
		}
		for _, event := range events {
			body, _ := json.Marshal(event)
			fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", event.ID, event.Type, body)
			after = event.ID
		}
		flusher.Flush()
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func (a *API) handleObservationStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		observations, err := a.service.Observations(r.Context(), after, 100)
		if err != nil {
			fmt.Fprintf(w, "event: error\ndata: %q\n\n", err.Error())
			flusher.Flush()
			return
		}
		for _, observation := range observations {
			body, _ := json.Marshal(observation)
			fmt.Fprintf(w, "id: %d\nevent: observation\ndata: %s\n\n", observation.ID, body)
			after = observation.ID
		}
		flusher.Flush()
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func (a *API) handleGetDesiredState(w http.ResponseWriter, r *http.Request) {
	desired, err := a.service.DesiredState(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, desired)
}

func (a *API) handlePutDesiredState(w http.ResponseWriter, r *http.Request) {
	var desired pool.DesiredState
	if err := json.NewDecoder(r.Body).Decode(&desired); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := a.service.SaveDesiredState(r.Context(), desired); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = a.service.EnforceLatest(r.Context())
	writeJSON(w, http.StatusOK, desired.WithHardwareConstraints())
}

func (a *API) handleGetPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := a.service.Plans(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"plans": plans})
}

func (a *API) handlePutPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := decodePlans(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := a.service.SavePlans(r.Context(), plans); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = a.service.EnforceLatest(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"plans": plans})
}

func (a *API) handleCommands(w http.ResponseWriter, r *http.Request) {
	var command pool.CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&command); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	record, err := a.service.ExecuteCommand(r.Context(), command)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, record)
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func decodePlans(r *http.Request) ([]pool.Plan, error) {
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, err
	}
	var plans []pool.Plan
	if err := json.Unmarshal(raw, &plans); err == nil {
		return plans, nil
	}
	var wrapper struct {
		Plans []pool.Plan `json:"plans"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Plans, nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func truthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
