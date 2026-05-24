package controller

import (
	"encoding/json"
	"net/http"

	"github.com/BMokarzel/rhaxis-code.git/internal/entity/node"
	read_service "github.com/BMokarzel/rhaxis-code.git/internal/module/read/service"
	logger "github.com/BMokarzel/rhaxis-code.git/pkg/log"
	"github.com/go-chi/chi/v5"
)

type Controller struct {
	log *logger.Log
	svc read_service.Service
}

type nodeGroups struct {
	prefix string
	Type   node.NodeType
	extra  func(chi.Router)
}

var groups = []nodeGroups{
	{"/service", node.ServiceNode, nil},
	{"/module", node.ModuleNode, nil},
	{"/endpoint", node.EndpointNode, nil},
	{"/function", node.FunctionNode, nil},
	{"/type", node.DataNode, nil},
	{"/variable", node.VariableNode, nil},
	{"/call", node.CallNode, nil},
	{"/schema", node.SchemaNode, nil},
}

func New(log *logger.Log, svc read_service.Service) *Controller {
	return &Controller{log: log, svc: svc}
}

// RegisterRoutes monta as rotas do módulo no router fornecido.
func (c *Controller) RegisterRoutes(r chi.Router) {
	r.Route("/read", func(r chi.Router) {
		for _, g := range groups {
			c.mountNodeGroup(r, g)
		}
	})
}

// mountNodeGroup monta o conjunto CRUD para um grupo de nós sob `prefix`.
// Cada handler recebe `kind` por closure, então a camada de service pode
// filtrar / despachar por tipo de nó.
func (c *Controller) mountNodeGroup(r chi.Router, g nodeGroups) {
	r.Route(g.prefix, func(r chi.Router) {
		r.Get("/", c.findAll(g.Type))
		r.Get("/{id}", c.findByID(g.Type))
		r.Post("/", c.create(g.Type))
		r.Put("/{id}", c.update(g.Type))
		r.Delete("/{id}", c.delete(g.Type))

		switch g.Type {
		case node.ServiceNode:
			{
				r.Get("", func(w http.ResponseWriter, r *http.Request) {})
			}
		}
	})
}

func (c *Controller) findAll(tp node.NodeType) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = tp // TODO: c.svc.FindAll(r.Context(), kind)
	}
}

func (c *Controller) findByID(tp node.NodeType) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		n, err := c.svc.GetNode(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if n == nil {
			http.NotFound(w, r)
			return
		}
		_ = tp // TODO: validar que n.Type == kind
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(n)
	}
}

func (c *Controller) create(tp node.NodeType) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = tp // TODO: c.svc.Create(r.Context(), kind, payload)
	}
}

func (c *Controller) update(tp node.NodeType) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = tp // TODO: c.svc.Update(r.Context(), kind, id, payload)
	}
}

func (c *Controller) delete(tp node.NodeType) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = tp // TODO: c.svc.Delete(r.Context(), kind, id)
	}
}
