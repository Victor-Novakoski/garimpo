// Command garimpo collects, filters and scores Brazilian tech job postings.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
)

var version = "dev"

const usage = `garimpo: peneira as vagas de tecnologia do Brasil.

Uso:
  garimpo <comando>

Comandos:
  collect   coleta as vagas e mostra a lista do dia
  mark      marca o que você fez com uma vaga: garimpo mark 12 apliquei
  note      anota algo numa vaga: garimpo note 12 "falei com a Ana"
  list      mostra as vagas que você marcou (vou aplicar, apliquei, entrevista)
  show      mostra uma vaga e o histórico dela: garimpo show 12
  version   mostra a versão
  help      mostra esta ajuda

Use "garimpo collect -h" para ver as opções da coleta.
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
	case "mark":
		return runMark(context.Background(), args[1:], stdout, stderr)
	case "note":
		return runNote(context.Background(), args[1:], stdout, stderr)
	case "list":
		return runList(context.Background(), args[1:], stdout, stderr)
	case "show":
		return runShow(context.Background(), args[1:], stdout, stderr)
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
