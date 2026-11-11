// Command garimpo-api serves the job flow over HTTP for the panel: list,
// detail, mark and note, with the same rules as the command line.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/victor-novakoski/garimpo/internal/api"
	"github.com/victor-novakoski/garimpo/internal/tracker"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stderr))
}

func run(ctx context.Context, args []string, stderr io.Writer) int {
	var addr, state string
	fs := flag.NewFlagSet("garimpo-api", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&addr, "addr", "127.0.0.1:8080", "endereço onde a API escuta (o padrão só atende esta máquina)")
	fs.StringVar(&state, "state", "", tracker.StateHelp)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "argumento inesperado: %q\n", fs.Arg(0))
		return 2
	}
	if state == "off" {
		fmt.Fprintln(stderr, "-state off não faz sentido aqui: a API serve o que você marcou")
		return 2
	}
	auth, err := api.NewToken(os.Getenv(api.TokenEnv))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	store, done, err := tracker.Open(ctx, state)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer done()

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if host, _, _ := net.SplitHostPort(listener.Addr().String()); !net.ParseIP(host).IsLoopback() {
		fmt.Fprintln(stderr, "aviso: a API está aberta para fora desta máquina e não fala HTTPS. Ponha um proxy com HTTPS na frente, senão o token viaja em texto puro.")
	}
	srv := &http.Server{
		Handler:           api.New(store, auth),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	fmt.Fprintf(stderr, "garimpo-api em http://%s (Ctrl+C para parar)\n", listener.Addr())

	failed := make(chan error, 1)
	go func() { failed <- srv.Serve(listener) }()
	select {
	case err := <-failed:
		fmt.Fprintln(stderr, err)
		return 1
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdown); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
