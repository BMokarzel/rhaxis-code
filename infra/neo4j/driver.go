// Package neo4j implementa o repository.Reader contra Neo4j 5.x.
//
// Nada deste pacote é exportado além do NewReader — a intenção é que o
// consumidor use apenas repository.Reader e não conheça o driver.
package neo4j

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Config são as credenciais e configurações mínimas para abrir uma sessão.
type Config struct {
	URI      string // "neo4j://localhost:7687"
	Username string
	Password string
	Database string // "" para default
}

// Open cria o driver e valida conectividade. Chamador deve fechar via Close().
func Open(ctx context.Context, cfg Config) (neo4j.DriverWithContext, error) {
	auth := neo4j.BasicAuth(cfg.Username, cfg.Password, "")
	drv, err := neo4j.NewDriverWithContext(cfg.URI, auth)
	if err != nil {
		return nil, fmt.Errorf("neo4j: create driver: %w", err)
	}
	if err := drv.VerifyConnectivity(ctx); err != nil {
		_ = drv.Close(ctx)
		return nil, fmt.Errorf("neo4j: verify connectivity: %w", err)
	}
	return drv, nil
}

// sessionConfig materializa o Database (se especificado) na config do driver.
func sessionConfig(database string) neo4j.SessionConfig {
	cfg := neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead}
	if database != "" {
		cfg.DatabaseName = database
	}
	return cfg
}
