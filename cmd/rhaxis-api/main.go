// rhaxis-api serve o codegraph via HTTP/JSON — pensado para Postman/curl
// e para o web front consumir.
//
//	rhaxis-api --addr :8080 \
//	           --neo4j-uri neo4j://localhost:7687 \
//	           --neo4j-user neo4j --neo4j-pass rhaxis-dev
//
// Endpoints (todos GET, retornam application/json):
//
//	GET  /healthz                       liveness (ping no driver)
//	GET  /api/services                  Tela 1 — mapa macro
//	GET  /api/endpoints?service=<urn>   Tela 2 — endpoints do serviço
//	GET  /api/flow?endpoint=<urn>       Tela 3 — árvore inicial do endpoint
//	GET  /api/expand?target=<urn>       expansão lazy de um step
//
// URNs vão como query param (URL-encoded), NÃO no path — evita escapar
// `:` e `/` no roteador.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/BMokarzel/rhaxis-code.git/domain"
	neo4jinfra "github.com/BMokarzel/rhaxis-code.git/infra/neo4j"
	"github.com/BMokarzel/rhaxis-code.git/repository"
)

// envOr lê uma env var; se ausente/vazia, usa o fallback como default do flag.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	var (
		addr = flag.String("addr", envOr("API_ADDR", ":8080"), "HTTP listen address")
		uri  = flag.String("neo4j-uri", envOr("NEO4J_URI", "neo4j://localhost:7687"), "Neo4j URI")
		user = flag.String("neo4j-user", envOr("NEO4J_USER", "neo4j"), "Neo4j username")
		pass = flag.String("neo4j-pass", envOr("NEO4J_PASS", "rhaxis-dev"), "Neo4j password")
		db   = flag.String("neo4j-db", envOr("NEO4J_DB", ""), "Neo4j database (empty = default)")
	)
	flag.Parse()

	// Driver: um só por processo. O ctx aqui é só para verificar conectividade.
	initCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	drv, err := neo4jinfra.Open(initCtx, neo4jinfra.Config{
		URI: *uri, Username: *user, Password: *pass, Database: *db,
	})
	cancel()
	if err != nil {
		log.Fatalf("neo4j: %v", err)
	}
	defer drv.Close(context.Background())

	reader := neo4jinfra.NewReader(drv, *db)

	mux := http.NewServeMux()
	h := &handlers{reader: reader, driver: drv}
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("GET /api/services", h.services)
	mux.HandleFunc("GET /api/endpoints", h.endpoints)
	mux.HandleFunc("GET /api/flow", h.flow)
	mux.HandleFunc("GET /api/expand", h.expand)

	srv := &http.Server{
		Addr:              *addr,
		Handler:           withLog(withCORS(mux)),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Graceful shutdown em SIGINT/SIGTERM.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		log.Printf("rhaxis-api listening on %s (neo4j=%s)", *addr, *uri)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http: %v", err)
		}
	}()
	<-stop
	log.Println("shutting down…")
	shCtx, shCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shCancel()
	_ = srv.Shutdown(shCtx)
}

// ------------------- handlers ------------------------------------------

type handlers struct {
	reader repository.Reader
	driver interface {
		VerifyConnectivity(ctx context.Context) error
	}
}

func (h *handlers) healthz(w http.ResponseWriter, r *http.Request) {
	if err := h.driver.VerifyConnectivity(r.Context()); err != nil {
		writeErr(w, http.StatusServiceUnavailable, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *handlers) services(w http.ResponseWriter, r *http.Request) {
	var filter *repository.ServiceMapFilter
	if p := r.URL.Query().Get("namePrefix"); p != "" {
		filter = &repository.ServiceMapFilter{NamePrefix: p}
	}
	sm, err := h.reader.LoadServiceMap(r.Context(), filter)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, serviceMapView(sm))
}

func (h *handlers) endpoints(w http.ResponseWriter, r *http.Request) {
	svcURN := requireURN(w, r, "service")
	if svcURN == "" {
		return
	}
	el, err := h.reader.ListEndpoints(r.Context(), svcURN)
	if err != nil {
		writeErr(w, statusFromErr(err), err)
		return
	}
	writeJSON(w, http.StatusOK, endpointListView(el))
}

func (h *handlers) flow(w http.ResponseWriter, r *http.Request) {
	epURN := requireURN(w, r, "endpoint")
	if epURN == "" {
		return
	}
	ef, err := h.reader.LoadEndpointFlow(r.Context(), epURN)
	if err != nil {
		writeErr(w, statusFromErr(err), err)
		return
	}
	writeJSON(w, http.StatusOK, endpointFlowView(ef))
}

func (h *handlers) expand(w http.ResponseWriter, r *http.Request) {
	target := requireURN(w, r, "target")
	if target == "" {
		return
	}
	fn, err := h.reader.ExpandFlow(r.Context(), target)
	if err != nil {
		writeErr(w, statusFromErr(err), err)
		return
	}
	writeJSON(w, http.StatusOK, flowNodeView(fn))
}

// ------------------- helpers -------------------------------------------

func requireURN(w http.ResponseWriter, r *http.Request, param string) domain.URN {
	v := r.URL.Query().Get(param)
	if v == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("missing required query param %q", param))
		return ""
	}
	return domain.URN(v)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(body)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": err.Error()})
}

// statusFromErr: mensagens do reader que indicam "não achado" viram 404.
// Simples/pragmático — o reader não tem tipo de erro dedicado ainda.
func statusFromErr(err error) int {
	msg := err.Error()
	for _, needle := range []string{"not found", "does not exist", "no such"} {
		if containsFold(msg, needle) {
			return http.StatusNotFound
		}
	}
	return http.StatusInternalServerError
}

func containsFold(s, sub string) bool {
	// evita import de strings só para isso? preferi manter — legibilidade > alocação.
	return len(sub) > 0 && indexFold(s, sub) >= 0
}

func indexFold(s, sub string) int {
outer:
	for i := 0; i+len(sub) <= len(s); i++ {
		for j := 0; j < len(sub); j++ {
			a, b := s[i+j], sub[j]
			if a >= 'A' && a <= 'Z' {
				a += 32
			}
			if b >= 'A' && b <= 'Z' {
				b += 32
			}
			if a != b {
				continue outer
			}
		}
		return i
	}
	return -1
}

// ------------------- middleware ----------------------------------------

func withLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(lrw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.RequestURI(), lrw.status, time.Since(start))
	})
}

// CORS aberto porque isto é dev. Se sair de dev, restringir Origin.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}
