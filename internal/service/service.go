package service

import (
	"context"
	"strings"

	"github.com/Starkiller2000Space-Projects/shortener/internal/storage"
)

type EndpointService interface {
	CreateShortURL(ctx context.Context, original, scheme, host string) (string, error)
	GetOriginalURL(ctx context.Context, short string) (string, error)
}

type endpointService struct {
	storage  storage.StorageInterface
	showAddr string
}

// add url into storage and return generated id
func (service *endpointService) CreateShortURL(ctx context.Context, original, scheme, host string) (string, error) {
	if scheme == "" {
		scheme = "http"
	}
	original = strings.TrimSpace(original)
	if original == "" {
		return "", ErrorEmptyUrl
	}
	shortID, err := service.storage.Add(ctx, original)
	if err != nil {
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
		return "", err // кастомная ошибка
	}
	return original, nil

}

func NewEndpointService(storage storage.StorageInterface, showAddr string) EndpointService {
	return &endpointService{storage: storage, showAddr: showAddr}
}
