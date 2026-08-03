// rhaxis-load consome um payload JSON produzido pelo extrator e escreve
// no Neo4j via repository.Writer.
//
// Uso:
//   rhaxis-load --payload path/to/payload.json \
//               --neo4j-uri neo4j://localhost:7687 \
//               --neo4j-user neo4j --neo4j-pass secret \
//               [--neo4j-db codegraph]
//
// Idempotente: rodar duas vezes sobre o mesmo payload não duplica.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	neo4jinfra "github.com/BMokarzel/rhaxis-code.git/infra/neo4j"
	"github.com/BMokarzel/rhaxis-code.git/loader"
)

func main() {
	var (
		payloadPath = flag.String("payload", "", "path to extraction payload JSON")
		uri         = flag.String("neo4j-uri", "neo4j://localhost:7687", "Neo4j URI")
		user        = flag.String("neo4j-user", "neo4j", "Neo4j username")
		pass        = flag.String("neo4j-pass", "", "Neo4j password")
		db          = flag.String("neo4j-db", "", "Neo4j database (empty = default)")
		timeout     = flag.Duration("timeout", 60*time.Second, "overall operation timeout")
	)
	flag.Parse()

	if *payloadPath == "" || *pass == "" {
		fmt.Fprintln(os.Stderr, "usage: rhaxis-load --payload <file> --neo4j-pass <pass> [flags]")
		flag.PrintDefaults()
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	p, err := loader.ReadPayloadFile(*payloadPath)
	if err != nil {
		die(err)
	}

	drv, err := neo4jinfra.Open(ctx, neo4jinfra.Config{
		URI: *uri, Username: *user, Password: *pass, Database: *db,
	})
	if err != nil {
		die(err)
	}
	defer drv.Close(ctx)

	w := neo4jinfra.NewWriter(drv, *db)
	stats, err := loader.LoadInto(ctx, p, w)
	if err != nil {
		die(err)
	}

	fmt.Fprintf(os.Stderr,
		"loaded service=%s nodes=%d edges=%d (skipped nodes=%d edges=%d)\n",
		p.Service.Name, stats.NodesUpserted, stats.EdgesUpserted, stats.NodesSkipped, stats.EdgesSkipped,
	)
}

func die(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
