package controller

import (
	"encoding/json"
	"net/http"

	observe_service "github.com/BMokarzel/rhaxis-code.git/internal/module/observe/service"
	logger "github.com/BMokarzel/rhaxis-code.git/pkg/log"
	"github.com/go-chi/chi/v5"
)

type Controller struct {
	log *logger.Log
	svc observe_service.Service
}

func New(log *logger.Log, svc observe_service.Service) *Controller {
	return &Controller{log: log, svc: svc}
}

func (c *Controller) RegisterRoutes(r chi.Router) {
	r.Route("/observe", func(r chi.Router) {
		r.Get("/metrics", c.stats)
	})
}

func (c *Controller) stats(w http.ResponseWriter, r *http.Request) {
	stats, err := c.svc.Stats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}
