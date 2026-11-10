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
  collect   coleta as vagas e lista no terminal
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
