// Command garimpo collects, filters and scores Brazilian tech job postings.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
)

var version = "dev"

const usage = `garimpo: peneira as vagas de tecnologia do Brasil.

Uso:
  garimpo <comando>

Comandos:
  collect   coleta as vagas e mostra a lista do dia
  serve     abre o painel no navegador, só nesta máquina: garimpo serve
  mark      marca o que você fez com uma vaga: garimpo mark 12 apliquei
  note      anota algo numa vaga: garimpo note 12 "falei com a Ana"
  list      mostra as vagas que você marcou (vou aplicar, apliquei, entrevista)
  show      mostra uma vaga e o histórico dela: garimpo show 12
  migrate   cria ou atualiza as tabelas no banco (Postgres)
  import    copia para o banco o que você já marcou no arquivo local
  version   mostra a versão
  help      mostra esta ajuda

Use "garimpo collect -h" para ver as opções da coleta, e "garimpo serve -h"
para as do painel.

O que você marca fica num arquivo na sua máquina. Com a variável
GARIMPO_DATABASE_URL definida, fica no Postgres (veja o README).
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, usage)
		return 0
	}
	switch args[0] {
	case "collect":
		return runCollect(context.Background(), args[1:], stdout, stderr)
	case "serve":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return runServe(ctx, args[1:], stderr)
	case "mark":
		return runMark(context.Background(), args[1:], stdout, stderr)
	case "note":
		return runNote(context.Background(), args[1:], stdout, stderr)
	case "list":
		return runList(context.Background(), args[1:], stdout, stderr)
	case "show":
		return runShow(context.Background(), args[1:], stdout, stderr)
	case "migrate":
		return runMigrate(context.Background(), args[1:], stdout, stderr)
	case "import":
		return runImport(context.Background(), args[1:], stdout, stderr)
	case "version":
		fmt.Fprintln(stdout, version)
		return 0
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "comando desconhecido: %q\n\n%s", args[0], usage)
		return 2
	}
}
