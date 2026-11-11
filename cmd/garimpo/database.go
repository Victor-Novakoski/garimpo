package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/victor-novakoski/garimpo/internal/db"
	"github.com/victor-novakoski/garimpo/internal/tracker"
)

// runMigrate creates or updates the tables. It is safe to run again: what
// was already applied is skipped.
func runMigrate(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		fmt.Fprintln(stderr, "uso: garimpo migrate   (o banco vem de "+db.URLEnv+")")
		return 2
	}
	pool, err := db.OpenEnv(ctx, false)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer pool.Close()
	applied, err := db.Migrate(ctx, pool)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if len(applied) == 0 {
		fmt.Fprintln(stdout, "o banco já está em dia")
		return 0
	}
	fmt.Fprintf(stdout, "migrações aplicadas: %s\n", strings.Join(applied, ", "))
	return 0
}

// runImport copies what was marked in the local file to the database, so
// moving to the database loses nothing. The file is only read.
func runImport(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	var from string
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&from, "from", "", "arquivo de onde importar (padrão: o arquivo na pasta de configuração do usuário)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "argumento inesperado: %q\n", fs.Arg(0))
		return 2
	}
	if from == "" {
		var err error
		if from, err = tracker.DefaultPath(); err != nil {
			fmt.Fprintf(stderr, "não achei a pasta de configuração (use -from): %v\n", err)
			return 1
		}
	}
	entries, err := tracker.OpenFile(from).List()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if len(entries) == 0 {
		fmt.Fprintf(stderr, "nada para importar: %s não existe ou está vazio\n", from)
		return 0
	}
	pool, err := db.OpenEnv(ctx, true)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer pool.Close()
	if err := tracker.OpenPostgres(pool).Import(entries); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	marked := 0
	for _, e := range entries {
		if e.Status != tracker.StatusNew || e.Note != "" {
			marked++
		}
	}
	fmt.Fprintf(stdout, "%d vagas importadas de %s (%d com estado ou anotação). O arquivo continua lá, sem mudança.\n",
		len(entries), from, marked)
	return 0
}
