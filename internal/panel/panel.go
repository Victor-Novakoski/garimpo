// Package panel serves the panel: the React app of web/, built by Vite into
// dist/ and embedded in the binary, so one command serves everything.
package panel

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

// dist holds the build of web/ (npm run build). The repository keeps only a
// placeholder there: the build is made on the machine and never committed.
//
//go:embed all:dist
var dist embed.FS

// notBuilt is answered while the binary was compiled without the build.
const notBuilt = `<!doctype html>
<html lang="pt-BR">
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Garimpo</title>
<body>
<h1>O painel ainda não foi montado</h1>
<p>O Garimpo está no ar, mas este binário foi compilado sem as telas. Na pasta do projeto, rode:</p>
<pre>npm --prefix web ci
npm --prefix web run build</pre>
<p>Precisa de um Node recente: 22.22 em diante, ou 24.15 em diante.</p>
<p>Depois pare o Garimpo (Ctrl+C) e rode o <code>garimpo serve</code> de novo.</p>
</body>
</html>
`

// Handler serves the panel from the embedded build.
func Handler() http.Handler {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err) // the directory is embedded: it cannot be missing
	}
	return handler(sub)
}

func handler(files fs.FS) http.Handler {
	_, err := fs.Stat(files, "index.html")
	built := err == nil
	static := http.FileServerFS(files)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		// The page only loads what this server gives it: no script, style
		// or request from anywhere else, and no other page can frame it.
		// The text of a posting comes from the internet; this is the second
		// lock behind React escaping it.
		h.Set("Content-Security-Policy", "default-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'; object-src 'none'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		// The links to the postings must not tell the companies' sites
		// where the click came from.
		h.Set("Referrer-Policy", "no-referrer")
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "método não permitido", http.StatusMethodNotAllowed)
			return
		}
		if !built {
			h.Set("Content-Type", "text/html; charset=utf-8")
			h.Set("Cache-Control", "no-store")
			w.WriteHeader(http.StatusServiceUnavailable)
			if r.Method == http.MethodGet {
				w.Write([]byte(notBuilt))
			}
			return
		}
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			// Vite names these files by their content.
			h.Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			h.Set("Cache-Control", "no-cache")
		}
		static.ServeHTTP(w, r)
	})
}
