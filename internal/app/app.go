package app

import (
	"exam-paper/internal/api"
	"exam-paper/internal/repository"
	"exam-paper/internal/service/questionbank"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type App struct {
	RootDir string
	DataDir string
	Store   *repository.Store
	Bank    *questionbank.Service
}

type Options struct {
	Host string
	Port int
}

func New() (*App, error) {
	rootDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	dataDir := filepath.Join(rootDir, "data")
	store, err := repository.NewStore(filepath.Join(dataDir, "exam-paper.db"))
	if err != nil {
		return nil, err
	}
	bank := questionbank.New(rootDir, store)
	store.SetQuestionFinder(bank)
	app := &App{RootDir: rootDir, DataDir: dataDir, Store: store, Bank: bank}
	if err := bank.SyncBundledQuestionBank(); err != nil {
		store.Close()
		return nil, err
	}
	if err := store.Checkpoint(); err != nil {
		store.Close()
		return nil, err
	}
	return app, nil
}

func (a *App) Close() {
	if a != nil && a.Store != nil {
		a.Store.Close()
	}
}

func (a *App) Run(opts Options) error {
	engine := api.NewRouter(api.Dependencies{
		RootDir: filepath.Clean(a.RootDir),
		DataDir: filepath.Clean(a.DataDir),
		Store:   a.Store,
		Bank:    a.Bank,
	})
	addr := fmt.Sprintf("%s:%d", opts.Host, opts.Port)
	log.Printf("ExamPaper is running at http://%s", addr)
	return engine.Run(addr)
}
