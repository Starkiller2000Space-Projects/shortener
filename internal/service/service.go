package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/models"
	"github.com/max-marek-projects/shortener/internal/repository"
	"github.com/max-marek-projects/shortener/internal/utils"
	"go.uber.org/zap"
)

//go:generate mockery --name=Service --output=../handlers/mocks --with-expecter
type Service interface {
	CreateShortURL(ctx context.Context, original, scheme, host string) (string, error)
	GetOriginalURL(ctx context.Context, short string) (string, error)
	Ping(ctx context.Context) error
	CreateShortURLsBatch(
		ctx context.Context,
		req []models.BatchShortenRequest,
		scheme string,
		host string,
	) ([]models.BatchShortenResponse, error)
	GetUserURLs(ctx context.Context, scheme, host string) ([]models.UserURL, error)
	DeleteUserURLs(ctx context.Context, userID string, shortIDs []string) error
}

func NewEndpointService(storage repository.Storage, showAddr string, idSize int) Service {
	return &endpointService{storage: storage, showAddr: showAddr, idSize: idSize}
}

type endpointService struct {
	storage  repository.Storage
	showAddr string
	idSize   int
}

// add url into storage and return generated id
func (service *endpointService) getUrlFromId(id, scheme, host string) (string, error) {
	if scheme == "" {
		scheme = "http"
	}
	var shortURL string
	var err error
	if service.showAddr != "" {
		shortURL, err = url.JoinPath(service.showAddr, id)
	} else {
		shortURL, err = url.JoinPath(scheme+"://"+host, id)
	}
	if err != nil {
		return "", fmt.Errorf("Failed to create short url: %w", err)
	}
	return shortURL, nil
}

// add url into storage and return generated id
func (service *endpointService) CreateShortURL(ctx context.Context, original, scheme, host string) (string, error) {
	original = strings.TrimSpace(original)
	if original == "" {
		logger.Log.Error("Url is Empty")
		return "", ErrorEmptyUrl
	}
	userID, ok := utils.GetUserIDFromContext(ctx)
	if !ok {
		return "", fmt.Errorf("userID not found in context")
	}
	id := utils.GenerateId(service.idSize)
	err := service.storage.Add(ctx, repository.Row{ID: id, OriginalURL: original, UserID: userID})
	if err != nil {
		var existsErr *repository.ErrAlreadyExists
		if errors.As(err, &existsErr) {
			shortURL, err := service.getUrlFromId(existsErr.ExistingID, scheme, host)
			if err != nil {
				return "", fmt.Errorf("Failed to create short url: %w", err)
			}
			return shortURL, ErrorDuplicate
		}
		logger.Log.Error("Url addition error", zap.String("message", err.Error()))
		return "", fmt.Errorf("Failed to create short url: %w", err)
	}
	shortURL, err := service.getUrlFromId(id, scheme, host)
	if err != nil {
		return "", fmt.Errorf("Failed to create short url: %w", err)
	}
	return shortURL, nil
}

// get url by id from storage if exists
func (service *endpointService) GetOriginalURL(ctx context.Context, id string) (string, error) {
	original, err := service.storage.Get(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrGone) {
			return "", err
		}
		logger.Log.Error("Error retrieving data from storage", zap.String("message", err.Error()))
		return "", fmt.Errorf("Failed to get original url: %w", err)
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
		shortURL, err := service.getUrlFromId(id, scheme, host)
		if err != nil {
			return nil, fmt.Errorf("Failed to add batch to storage: %w", err)
		}

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
		return nil, fmt.Errorf("Failed to add batch to storage: %w", err)
	}

	return resp, nil
}

func (service *endpointService) GetUserURLs(ctx context.Context, scheme, host string) ([]models.UserURL, error) {
	userID, ok := utils.GetUserIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("userID not found in context")
	}
	rows, err := service.storage.GetUserURLs(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]models.UserURL, 0, len(rows))
	for _, row := range rows {
		// create full short url
		shortURL, err := service.getUrlFromId(row.ShortURL, scheme, host)
		if err != nil {
			return nil, err
		}
		result = append(result, models.UserURL{
			ShortURL:    shortURL,
			OriginalURL: row.OriginalURL,
		})
	}
	return result, nil
}

func (service *endpointService) DeleteUserURLs(ctx context.Context, userID string, shortIDs []string) error {
	if len(shortIDs) == 0 {
		return nil
	}
	return service.storage.DeleteBatch(ctx, userID, shortIDs)
}
