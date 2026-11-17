package tracker

import (
	"context"
	"fmt"
	"os"

	"github.com/victor-novakoski/garimpo/internal/db"
)

// StateHelp describes the -state flag of every command that opens a store.
const StateHelp = `onde fica o que você marcou em cada vaga: um arquivo, "file" (o arquivo padrão), "db" (o banco de ` + db.URLEnv + `) ou "off". Padrão: o banco, se ` + db.URLEnv + ` estiver definida; senão o arquivo na pasta de configuração do usuário`

// Open returns the store chosen by state (see StateHelp) and what closes
// it, or a nil store when tracking is off. The database is chosen by the
// environment, never by a flag: the URL holds the password.
func Open(ctx context.Context, state string) (Store, func(), error) {
	url := os.Getenv(db.URLEnv)
	switch {
	case state == "off":
		return nil, func() {}, nil
	case state == "db" && url == "":
		return nil, nil, fmt.Errorf("-state db precisa da variável %s com a URL do Postgres", db.URLEnv)
	case state == "db" || (state == "" && url != ""):
		pool, err := db.OpenEnv(ctx, true)
		if err != nil {
			return nil, nil, err
		}
		return OpenPostgres(pool), pool.Close, nil
	}
	if state == "" || state == "file" {
		var err error
		if state, err = DefaultPath(); err != nil {
			return nil, nil, fmt.Errorf("não achei a pasta de configuração (use -state): %w", err)
		}
	}
	return OpenFile(state), func() {}, nil
}
