package db

import (
	"context"
	"fmt"
	"gowhatsmeow/internal/config"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func NewSQLContainer(cfg *config.Config, logger waLog.Logger) (*sqlstore.Container, error) {
	dbLog := logger.Sub("Database")
	container, err := sqlstore.New(context.Background(), cfg.Database.Type, cfg.Database.URL, dbLog)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database store: %w", err)
	}
	return container, nil
}
