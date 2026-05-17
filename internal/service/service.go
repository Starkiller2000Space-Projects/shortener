package service

import (
	"context"
	"strings"

	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/repository"
	"go.uber.org/zap"
)

type EndpointService interface {
	CreateShortURL(ctx context.Context, original, scheme, host string) (string, error)
	GetOriginalURL(ctx context.Context, short string) (string, error)
	Ping(ctx context.Context) error
}

type endpointService struct {
	storage  repository.StorageInterface
	showAddr string
}

// add url into storage and return generated id
func (service *endpointService) CreateShortURL(ctx context.Context, original, scheme, host string) (string, error) {
	if scheme == "" {
		scheme = "http"
	}
	original = strings.TrimSpace(original)
	if original == "" {
		logger.Log.Error("Url is Empty")
		return "", ErrorEmptyUrl
	}
	shortID, err := service.storage.Add(ctx, original)
	if err != nil {
		logger.Log.Error("Url is Empty", zap.String("message", err.Error()))
		return "", err
	}
	var shortURL string
	if service.showAddr != "" {
		shortURL = service.showAddr + "/" + shortID
	} else {
		shortURL = scheme + "://" + host + "/" + shortID
	}
	return shortURL, nil
}

// get url by id from storage if exists
func (service *endpointService) GetOriginalURL(ctx context.Context, short string) (string, error) {
	original, err := service.storage.Get(ctx, short)
	if err != nil {
		logger.Log.Error("Error retrieving data from storage", zap.String("message", err.Error()))
		return "", err
	}
	return original, nil

}

func (service *endpointService) Ping(ctx context.Context) error {
	return service.storage.Ping(ctx)
}

func NewEndpointService(storage repository.StorageInterface, showAddr string) EndpointService {
	return &endpointService{storage: storage, showAddr: showAddr}
}
