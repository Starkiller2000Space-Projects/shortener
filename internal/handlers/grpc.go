package handlers

import (
	"context"
	"errors"
	"strings"

	"github.com/max-marek-projects/shortener/api"
	"github.com/max-marek-projects/shortener/internal/audit"
	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/models"
	"github.com/max-marek-projects/shortener/internal/repository"
	"github.com/max-marek-projects/shortener/internal/service"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHandler handles grpc endpoints for URL shortening and redirection.
type GRPCHandler struct {
	api.UnimplementedShortenerServiceServer
	service service.GRPCService
}

// NewGRPCHandler creates a new GRPCHandler.
func NewGRPCHandler(service service.GRPCService) *GRPCHandler {
	return &GRPCHandler{service: service}
}

// ShortenURL creates a short URL from the bytes body.
// The body should contain the original URL.
// On success, returns the short URL in body.
func (h *GRPCHandler) ShortenURL(
	ctx context.Context,
	req *api.URLShortenRequest,
) (*api.URLShortenResponse, error) {
	if req == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"request is nil",
		)
	}
	originalURL := strings.TrimSpace(req.GetUrl())
	if originalURL == "" {
		return nil, status.Error(
			codes.InvalidArgument,
			"Empty body",
		)
	}
	auditData, ok := audit.GetAuditDataFromContext(ctx)
	if !ok {
		logger.Log.Error("No audit data in context")
		return nil, status.Error(
			codes.Internal,
			"Internal error",
		)
	}
	auditData.Action = models.AuditShorten
	auditData.URL = originalURL
	result, err := h.service.CreateShortURL(ctx, originalURL, "", "") // unable to get scheme or host with grpc
	if err != nil {
		if errors.Is(err, service.ErrDuplicate) {
			return nil, status.Error(
				codes.AlreadyExists,
				err.Error(),
			)
		}
		if errors.Is(err, service.ErrEmptyURL) {
			return nil, status.Error(
				codes.InvalidArgument,
				err.Error(),
			)
		}
		logger.Log.Error("Failed create short URL", zap.Error(err))
		return nil, status.Error(
			codes.Internal,
			"Internal error",
		)
	}
	response := &api.URLShortenResponse{}
	response.SetResult(result)
	return response, nil
}

// ExpandURL handles returns  original URL.
// It also records an audit follow action.
func (h *GRPCHandler) ExpandURL(
	ctx context.Context,
	req *api.URLExpandRequest,
) (*api.URLExpandResponse, error) {
	if req == nil || strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	id := strings.TrimSpace(req.GetId())
	originalURL, err := h.service.GetOriginalURL(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "url not found")
		}
		if errors.Is(err, repository.ErrGone) {
			return nil, status.Error(codes.NotFound, "url has been deleted")
		}
		logger.Log.Error("Error retrieving original url", zap.Error(err))
		return nil, status.Error(codes.Internal, "Internal error")
	}
	auditData, ok := audit.GetAuditDataFromContext(ctx)
	if !ok {
		logger.Log.Error("No audit data in context")
		return nil, status.Error(codes.Internal, "Internal error")
	}
	auditData.Action = models.AuditFollow
	auditData.URL = originalURL
	response := &api.URLExpandResponse{}
	response.SetResult(originalURL)
	return response, nil
}

// ListUserURLs returns all URLs created by the authenticated user.
func (h *GRPCHandler) ListUserURLs(
	ctx context.Context,
	req *api.UserURLsRequest,
) (*api.UserURLsResponse, error) {
	urls, err := h.service.GetUserURLs(ctx, "", "") // unable to get scheme or host with grpc
	if err != nil {
		logger.Log.Error("failed to get user URLs", zap.Error(err))
		return nil, status.Error(codes.Internal, "Internal error")
	}
	pbUrls := make([]*api.URLData, len(urls))
	for i, u := range urls {
		pbUrls[i] = &api.URLData{}
		pbUrls[i].SetOriginalUrl(u.OriginalURL)
		pbUrls[i].SetShortUrl(u.ShortURL)
	}
	response := &api.UserURLsResponse{}
	response.SetUrls(pbUrls)
	return response, nil
}
