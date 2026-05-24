package controller

import (
	"encoding/json"
	"net/http"

	"github.com/BMokarzel/rhaxis-code.git/internal/entity/node"
	extract_service "github.com/BMokarzel/rhaxis-code.git/internal/module/extract/service"
	logger "github.com/BMokarzel/rhaxis-code.git/pkg/log"
	"github.com/go-chi/chi/v5"
)

type Controller struct {
	log *logger.Log
	svc extract_service.Service
}

func New(log *logger.Log, svc extract_service.Service) *Controller {
	return &Controller{log: log, svc: svc}
}

func (c *Controller) RegisterRoutes(r chi.Router) {
	r.Route("/extract", func(r chi.Router) {
		r.Post("/", c.ingest)
	})
}

func (c *Controller) ingest(w http.ResponseWriter, r *http.Request) {
	var n node.Node
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := c.svc.Ingest(r.Context(), &n); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
