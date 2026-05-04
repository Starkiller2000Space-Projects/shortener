package service

import (
	"context"
	"strings"

	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/models"
	"github.com/max-marek-projects/shortener/internal/repository"
	"github.com/max-marek-projects/shortener/internal/utils"
	"go.uber.org/zap"
)

type EndpointService interface {
	CreateShortURL(ctx context.Context, original, scheme, host string) (string, error)
	GetOriginalURL(ctx context.Context, short string) (string, error)
	Ping(ctx context.Context) error
	CreateShortURLsBatch(
		ctx context.Context,
		req []models.BatchShortenRequest,
		scheme string,
		host string,
	) ([]models.BatchShortenResponse, error)
}

func NewEndpointService(storage repository.StorageInterface, showAddr string, idSize int) EndpointService {
	return &endpointService{storage: storage, showAddr: showAddr, idSize: idSize}
}

type endpointService struct {
	storage  repository.StorageInterface
	showAddr string
	idSize   int
}

// add url into storage and return generated id
func (service *endpointService) getUrlFromId(id, scheme, host string) string {
	if scheme == "" {
		scheme = "http"
	}
	var shortURL string
	if service.showAddr != "" {
		shortURL = service.showAddr + "/" + id
	} else {
		shortURL = scheme + "://" + host + "/" + id
	}
	return shortURL
}

// add url into storage and return generated id
func (service *endpointService) CreateShortURL(ctx context.Context, original, scheme, host string) (string, error) {
	original = strings.TrimSpace(original)
	if original == "" {
		logger.Log.Error("Url is Empty")
		return "", ErrorEmptyUrl
	}
	id := utils.GenerateId(service.idSize)
	err := service.storage.Add(ctx, id, original)
	if err != nil {
		logger.Log.Error("Url is Empty", zap.String("message", err.Error()))
		return "", err
	}
	shortURL := service.getUrlFromId(id, scheme, host)
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

// create short url for batch
func (service *endpointService) CreateShortURLsBatch(
	ctx context.Context,
	req []models.BatchShortenRequest,
	scheme string,
	host string,
) ([]models.BatchShortenResponse, error) {
	if len(req) == 0 {
		return nil, ErrorEmptyBatch
	}

	items := make([]repository.Row, 0, len(req))
	resp := make([]models.BatchShortenResponse, 0, len(req))

	for _, r := range req {
		if r.OriginalURL == "" {
			return nil, ErrorEmptyUrl
		}

		id := utils.GenerateId(service.idSize)
		shortURL := service.getUrlFromId(id, scheme, host)

		items = append(items, repository.Row{
			ID:          id,
			OriginalURL: r.OriginalURL,
		})

		resp = append(resp, models.BatchShortenResponse{
			CorrelationID: r.CorrelationID,
			ShortURL:      shortURL,
		})
	}

	if err := service.storage.AddBatch(ctx, items); err != nil {
		return nil, err
	}

	return resp, nil
}
