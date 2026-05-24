package extract

import (
	"fmt"
	"net/http"

	"github.com/BMokarzel/rhaxis-code.git/config"
	"github.com/BMokarzel/rhaxis-code.git/internal/module/extract/controller"
	extract_service "github.com/BMokarzel/rhaxis-code.git/internal/module/extract/service"
	"github.com/BMokarzel/rhaxis-code.git/internal/repository/n4j"
	pkg_http "github.com/BMokarzel/rhaxis-code.git/pkg/http"
	logger "github.com/BMokarzel/rhaxis-code.git/pkg/log"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	cfg    *config.Config
	router *chi.Mux
}

func NewServer(cfg *config.Config) *Server {
	log := logger.New(cfg.Env)
	repo := n4j.New(log)
	svc := extract_service.New(log, repo)
	ctrl := controller.New(log, svc)

	router := pkg_http.NewRouter()
	ctrl.RegisterRoutes(router)

	return &Server{cfg: cfg, router: router}
}

func (s *Server) Name() string         { return "extract" }
func (s *Server) Addr() string         { return fmt.Sprintf(":%d", s.cfg.HTTP.ExtractPort) }
func (s *Server) Routes() http.Handler { return s.router }
