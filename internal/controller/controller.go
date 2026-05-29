package controller

import (
	"exam-paper/internal/repository"
	"exam-paper/internal/service/questionbank"
)

const MaxUploadSize = 96 << 20

type Controller struct {
	rootDir string
	dataDir string
	store   *repository.Store
	bank    *questionbank.Service
}

func New(rootDir string, dataDir string, store *repository.Store, bank *questionbank.Service) *Controller {
	return &Controller{rootDir: rootDir, dataDir: dataDir, store: store, bank: bank}
}
