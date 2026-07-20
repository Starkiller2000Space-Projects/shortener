// Package service implements the core URL shortening business logic.

package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/models"
	"github.com/max-marek-projects/shortener/internal/repository"
	"github.com/max-marek-projects/shortener/internal/requests"
	"github.com/max-marek-projects/shortener/internal/utils"
	"go.uber.org/zap"
)

// Service defines the business logic interface for URL shortening.
//
//go:generate mockery --name=Service --output=../handlers --outpkg=handlers --filename=mock_service.gen._test.go --with-expecter --structname=MockService
//go:generate mockery --name=Service --output=../server --outpkg=server --filename=mock_service.gen._test.go --with-expecter --structname=MockService
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

// NewEndpointService creates a new service implementation with the given storage,
// base URL for short links (showAddr), and ID size (number of characters).
func NewEndpointService(storage repository.Storage, showAddr string, idSize int) Service {
	return &endpointService{storage: storage, showAddr: showAddr, idSize: idSize}
}

// endpointService is the concrete implementation of Service.
type endpointService struct {
	storage  repository.Storage
	showAddr string
	idSize   int
}

// getURLFromID constructs a full short URL (with scheme and host) from a short ID.
// Uses showAddr if configured, otherwise builds from scheme://host.
func (service *endpointService) getURLFromID(id, scheme, host string) (string, error) {
	if scheme == "" {
		scheme = "http"
	}
	var base string
	if service.showAddr != "" {
		base = service.showAddr
	} else {
		base = scheme + "://" + host
	}
	base = strings.TrimRight(base, "/")
	return base + "/" + id, nil
}

// add url into storage and return generated id
func (service *endpointService) CreateShortURL(ctx context.Context, original, scheme, host string) (string, error) {
	userID, ok := requests.GetUserIDFromContext(ctx)
	if !ok {
		return "", fmt.Errorf("userID not found in context")
	}
	original = strings.TrimSpace(original)
	if original == "" {
		logger.Log.Error("Url is Empty")
		return "", ErrEmptyURL
	}
	id := utils.GenerateID(service.idSize)
	err := service.storage.Add(ctx, repository.Row{ID: id, OriginalURL: original, UserID: userID})
	if err != nil {
		var existsErr *repository.ErrAlreadyExists
		if errors.As(err, &existsErr) {
			shortURL, err := service.getURLFromID(existsErr.ExistingID, scheme, host)
			if err != nil {
				return "", fmt.Errorf("failed to create short url: %w", err)
			}
			return shortURL, ErrDuplicate
		}
		logger.Log.Error("Url addition error", zap.String("message", err.Error()))
		return "", fmt.Errorf("failed to create short url: %w", err)
	}
	shortURL, err := service.getURLFromID(id, scheme, host)
	if err != nil {
		return "", fmt.Errorf("failed to create short url: %w", err)
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
		return "", fmt.Errorf("failed to get original url: %w", err)
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
		return nil, ErrEmptyBatch
	}

	items := make([]repository.Row, 0, len(req))
	resp := make([]models.BatchShortenResponse, 0, len(req))

	for _, r := range req {
		if r.OriginalURL == "" {
			return nil, ErrEmptyURL
		}

		id := utils.GenerateID(service.idSize)
		shortURL, err := service.getURLFromID(id, scheme, host)
		if err != nil {
			return nil, fmt.Errorf("failed to add batch to storage: %w", err)
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
		return nil, fmt.Errorf("failed to add batch to storage: %w", err)
	}

	return resp, nil
}

func (service *endpointService) GetUserURLs(ctx context.Context, scheme, host string) ([]models.UserURL, error) {
	userID, ok := requests.GetUserIDFromContext(ctx)
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
		shortURL, err := service.getURLFromID(row.ShortURL, scheme, host)
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
