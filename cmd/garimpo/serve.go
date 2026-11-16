package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/victor-novakoski/garimpo/internal/api"
	"github.com/victor-novakoski/garimpo/internal/panel"
	"github.com/victor-novakoski/garimpo/internal/tracker"
)

// The panel runs on the owner's machine and nowhere else: serve always
// listens on the loopback, so there is no flag for the address, only for
// the port.
const loopback = "127.0.0.1"

type serveOptions struct {
	collectOptions
	port int
	// demo serves made-up postings from the repository (demo.go) instead
	// of the owner's record.
	demo bool
}

func parseServe(args []string, stderr io.Writer) (serveOptions, error) {
	var o serveOptions
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.IntVar(&o.port, "port", 8080, "porta do painel, sempre em "+loopback+" (só esta máquina)")
	fs.BoolVar(&o.demo, "demo", false, "mostra o painel com vagas e perfil de exemplo, sem banco, sem rede e sem gravar nada (rode na pasta do projeto)")
	o.register(fs)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "uso: garimpo serve [opções]\n\nAbre o painel em http://"+loopback+":8080. As opções de coleta valem para o botão \"Buscar vagas\" do painel.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	if o.port < 1 || o.port > 65535 {
		return o, fmt.Errorf("-port: porta inválida %d", o.port)
	}
	if o.state == "off" {
		return o, errors.New("-state off não faz sentido aqui: o painel mostra o que você marcou")
	}
	if err := o.check(fs); err != nil {
		return o, err
	}
	if o.demo {
		// The demo has its own postings and profile: an option that points
		// to the real ones would be ignored without a word.
		only := map[string]bool{"demo": true, "port": true}
		var extra string
		fs.Visit(func(f *flag.Flag) {
			if !only[f.Name] && extra == "" {
				extra = f.Name
			}
		})
		if extra != "" {
			return o, fmt.Errorf("-demo não combina com -%s: o modo demo só aceita -port", extra)
		}
	}
	return o, nil
}

// collectRun is the collection the panel's button starts: the same steps of
// "garimpo collect", with the options serve was started with. The profile
// is read again on every run, so editing it needs no restart.
//
// What goes to log is shown in the panel. A store error can name the
// database host, so it goes to the terminal and the panel gets a plain
// sentence.
func collectRun(o collectOptions, store tracker.Store, terminal io.Writer) api.RunFunc {
	return func(ctx context.Context, log io.Writer) error {
		p, hasProfile, err := loadProfile(o.profile)
		if err != nil {
			return fmt.Errorf("perfil inválido: %w", err)
		}
		if !hasProfile {
			return errors.New(noProfile)
		}
		at := now()
		g := gather(ctx, sources(o, at), o.all, log)
		if ctx.Err() != nil {
			return errors.New("coleta interrompida: o Garimpo foi encerrado")
		}
		if len(g.sources) == 0 {
			return errors.New("nenhuma fonte respondeu: confira a internet e tente de novo")
		}
		if _, _, err := rank(ctx, o, p, store, g, at, log); err != nil {
			fmt.Fprintf(terminal, "coleta pelo painel: %v\n", err)
			return errors.New("não consegui gravar a coleta: veja o terminal onde o Garimpo está rodando")
		}
		return nil
	}
}

const noProfile = "sem perfil não há nota nem lista: copie profile.example.yaml para " + defaultProfile + ", ajuste e clique de novo"

// serveHandler is everything serve answers: the API under /api/ and the
// panel on the rest, both only for this machine's own page.
func serveHandler(store tracker.Store, o api.Options) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", api.NewLocal(store, o)))
	mux.Handle("/", api.LocalOnly(panel.Handler()))
	return mux
}

func runServe(ctx context.Context, args []string, stderr io.Writer) int {
	o, err := parseServe(args, stderr)
	if err != nil {
		if err != flag.ErrHelp {
			fmt.Fprintln(stderr, err)
		}
		return 2
	}
	if o.demo {
		store, err := openDemo(ctx, demoJobs, demoProfile, now().Truncate(time.Minute), stderr)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return listen(ctx, o.port, store, api.Options{CollectOff: demoCollectOff}, "", stderr)
	}
	// The profile is checked now only to warn early; each collection reads
	// it again.
	_, hasProfile, err := loadProfile(o.profile)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if o.ai != "" && !hasProfile {
		fmt.Fprintln(stderr, "-ai precisa de um perfil: a IA compara a vaga com ele")
		return 2
	}
	store, done, err := tracker.Open(ctx, o.state)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer done()

	// Collections stop with the server: their context ends on Ctrl+C.
	runner := api.NewRunner(ctx, collectRun(o.collectOptions, store, stderr))
	warning := ""
	if !hasProfile {
		warning = "aviso: sem perfil, o botão de buscar vagas não funciona (copie profile.example.yaml para " + defaultProfile + ")"
	}
	return listen(ctx, o.port, store, api.Options{Collect: runner}, warning, stderr)
}

// listen serves the panel over the store until the context ends. The
// warning, when there is one, is printed after the address.
func listen(ctx context.Context, port int, store tracker.Store, o api.Options, warning string, stderr io.Writer) int {
	listener, err := net.Listen("tcp", net.JoinHostPort(loopback, strconv.Itoa(port)))
	if err != nil {
		fmt.Fprintf(stderr, "não consegui abrir a porta %d (tente outra com -port): %v\n", port, err)
		return 1
	}
	srv := &http.Server{
		Handler:           serveHandler(store, o),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	fmt.Fprintf(stderr, "Garimpo em http://%s (só nesta máquina; Ctrl+C para parar)\n", listener.Addr())
	if warning != "" {
		fmt.Fprintln(stderr, warning)
	}

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
	// A collection in progress saw the context end; let it finish before
	// the store closes under it.
	if o.Collect != nil {
		o.Collect.Wait()
	}
	return 0
}
