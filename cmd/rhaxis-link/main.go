// rhaxis-link resolve CallHTTP não-linkadas contra Endpoints já no grafo.
//
// Deve rodar DEPOIS de todos os rhaxis-load (um por serviço) — só então
// existem Endpoints alvo pra casar. Idempotente: uma segunda passada só
// atua sobre o que continuar unresolved.
//
// Uso:
//   rhaxis-link --neo4j-uri neo4j://localhost:7687 \
//               --neo4j-user neo4j --neo4j-pass secret \
//               [--neo4j-db codegraph] [--verbose]
//
// Exit codes:
//   0 sucesso, mesmo com ambiguidades reportadas
//   1 erro de I/O ou Neo4j
//   2 uso inválido
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	neo4jinfra "github.com/BMokarzel/rhaxis-code.git/infra/neo4j"
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
		uri     = flag.String("neo4j-uri", envOr("NEO4J_URI", "neo4j://localhost:7687"), "Neo4j URI")
		user    = flag.String("neo4j-user", envOr("NEO4J_USER", "neo4j"), "Neo4j username")
		pass    = flag.String("neo4j-pass", envOr("NEO4J_PASS", ""), "Neo4j password")
		db      = flag.String("neo4j-db", envOr("NEO4J_DB", ""), "Neo4j database (empty = default)")
		timeout = flag.Duration("timeout", 5*time.Minute, "overall operation timeout")
		verbose = flag.Bool("verbose", false, "print per-call decisions")
	)
	flag.Parse()

	if *pass == "" {
		fmt.Fprintln(os.Stderr, "usage: rhaxis-link --neo4j-pass <pass> [flags]")
		flag.PrintDefaults()
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	drv, err := neo4jinfra.Open(ctx, neo4jinfra.Config{
		URI: *uri, Username: *user, Password: *pass, Database: *db,
	})
	if err != nil {
		die(err)
	}
	defer drv.Close(ctx)

	linker := neo4jinfra.NewLinker(drv, *db)
	stats, ambiguous, err := linker.Link(ctx)
	if err != nil {
		die(err)
	}

	fmt.Fprintf(os.Stderr,
		"linked: scanned=%d resolved=%d ambiguous=%d no-match=%d self-loop=%d already=%d depends-updated=%d\n",
		stats.CallsScanned, stats.Resolved, stats.Ambiguous, stats.NoMatch,
		stats.SelfLoopSkip, stats.AlreadyLinked, stats.DependsUpdated,
	)

	if len(ambiguous) > 0 {
		fmt.Fprintln(os.Stderr, "ambiguities (not resolved, review manually):")
		for _, a := range ambiguous {
			fmt.Fprintf(os.Stderr, "  %s %s → candidates: %v\n", a.Method, a.PathPattern, a.Candidates)
			_ = a.CallURN
		}
	}

	if *verbose && stats.CallsScanned == 0 {
		fmt.Fprintln(os.Stderr, "no unresolved CallHTTP nodes in graph")
	}
}

func die(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
