package handlers

import (
	"context"
	"fmt"
	"testing"

	"github.com/max-marek-projects/shortener/api"
	"github.com/max-marek-projects/shortener/internal/audit"
	"github.com/max-marek-projects/shortener/internal/models"
	"github.com/max-marek-projects/shortener/internal/repository"
	"github.com/max-marek-projects/shortener/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestGRPCHandler_ShortenURL(t *testing.T) {
	fixedID := "test1234"

	type want struct {
		code    codes.Code
		status  string
		success bool
		body    string
	}
	type serviceData struct {
		value string
		err   error
	}
	tests := []struct {
		name    string
		service *serviceData
		request string
		audit   bool
		want    want
	}{
		{
			name: "positive test",
			service: &serviceData{
				value: fmt.Sprintf("http://example/%s", fixedID),
				err:   nil,
			},
			request: "https://www.example0.com/",
			audit:   true,
			want: want{
				status:  "",
				success: true,
				body:    fixedID,
			},
		},
		{
			name:    "empty body test",
			request: "",
			audit:   true,
			want: want{
				status:  "Empty body",
				code:    codes.InvalidArgument,
				success: false,
				body:    "",
			},
		},
		{
			name:    "no audit",
			request: "https://www.example0.com/",
			audit:   false,
			want: want{
				code:    codes.Internal,
				status:  "No audit data",
				success: false,
				body:    "",
			},
		},
		{
			name: "empty url",
			service: &serviceData{
				value: "",
				err:   service.ErrEmptyURL,
			},
			request: "https://www.example0.com/",
			audit:   true,
			want: want{
				code:    codes.InvalidArgument,
				status:  "url is empty or contains only whitespace characters",
				success: false,
				body:    "",
			},
		},
		{
			name: "duplicate",
			service: &serviceData{
				value: "",
				err:   service.ErrDuplicate,
			},
			request: "https://www.example0.com/",
			audit:   true,
			want: want{
				code:    codes.AlreadyExists,
				status:  "url already exists",
				success: false,
				body:    "",
			},
		},
		{
			name: "unknown error",
			service: &serviceData{
				value: "",
				err:   fmt.Errorf("unknown error"),
			},
			request: "https://www.example0.com/",
			audit:   true,
			want: want{
				code:    codes.Internal,
				status:  "Internal error",
				success: false,
				body:    "",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockSvc := NewMockService(t)
			handler := NewGRPCHandler(mockSvc)
			if test.service != nil {
				mockSvc.EXPECT().CreateShortURL(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(test.service.value, test.service.err)
			} else {
				mockSvc.AssertNotCalled(t, "CreateShortURL", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			}

			ctx := context.Background()
			if test.audit {
				ctx = audit.SetAuditDataToContext(ctx, &models.AuditData{})
			}
			req := &api.URLShortenRequest{Url: test.request}
			resp, err := handler.ShortenURL(ctx, req)
			if test.want.success {
				assert.Contains(t, resp.Result, test.want.body)
				assert.NoError(t, err)
				return
			}
			assert.Nil(t, resp)
			assert.Error(t, err)
			st, _ := status.FromError(err)
			assert.Equal(t, test.want.code, st.Code())
			assert.Equal(t, test.want.status, st.Message())
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestGRPCHandler_ExpandURL(t *testing.T) {
	fixedID := "test1234"
	existingURL := "https://example.com/"

	type want struct {
		code    codes.Code
		message string
		result  string
	}
	type serviceData struct {
		value string
		err   error
	}
	tests := []struct {
		name    string
		req     *api.URLExpandRequest
		service *serviceData
		audit   bool
		want    want
	}{
		{
			name: "positive test",
			req:  &api.URLExpandRequest{Id: fixedID},
			service: &serviceData{
				value: existingURL,
				err:   nil,
			},
			audit: true,
			want: want{
				code:    codes.OK,
				message: "",
				result:  existingURL,
			},
		},
		{
			name:  "empty id test",
			req:   &api.URLExpandRequest{Id: ""},
			audit: true,
			want: want{
				code:    codes.InvalidArgument,
				message: "id is required",
				result:  "",
			},
		},
		{
			name: "missing id (not found)",
			req:  &api.URLExpandRequest{Id: fixedID},
			service: &serviceData{
				value: "",
				err:   repository.ErrNotFound,
			},
			audit: true,
			want: want{
				code:    codes.NotFound,
				message: "url not found",
				result:  "",
			},
		},
		{
			name: "gone test",
			req:  &api.URLExpandRequest{Id: fixedID},
			service: &serviceData{
				value: "",
				err:   repository.ErrGone,
			},
			audit: true,
			want: want{
				code:    codes.NotFound,
				message: "url has been deleted",
				result:  "",
			},
		},
		{
			name: "no audit",
			req:  &api.URLExpandRequest{Id: fixedID},
			service: &serviceData{
				value: existingURL,
				err:   nil,
			},
			audit: false,
			want: want{
				code:    codes.Internal,
				message: "id is required", // из-за сообщения в коде
				result:  "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := NewMockService(t)
			if tt.service != nil {
				mockSvc.EXPECT().GetOriginalURL(mock.Anything, fixedID).Return(tt.service.value, tt.service.err)
			} else {
				mockSvc.AssertNotCalled(t, "GetOriginalURL", mock.Anything, mock.Anything)
			}
			handler := NewGRPCHandler(mockSvc)

			ctx := context.Background()
			if tt.audit {
				ctx = audit.SetAuditDataToContext(ctx, &models.AuditData{})
			}
			resp, err := handler.ExpandURL(ctx, tt.req)

			if tt.want.code == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.want.result, resp.Result)
			} else {
				assert.Nil(t, resp)
				assert.Error(t, err)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.want.code, st.Code())
				assert.Equal(t, tt.want.message, st.Message())
			}
			mockSvc.AssertExpectations(t)
		})
	}
}
func TestGRPCHandler_ListUserURLs(t *testing.T) {
	type want struct {
		code    codes.Code
		message string
		hasData bool
		count   int
	}
	type serviceData struct {
		value []models.UserURL
		err   error
	}
	tests := []struct {
		name    string
		service *serviceData
		want    want
	}{
		{
			name: "positive test",
			service: &serviceData{
				value: []models.UserURL{
					{ShortURL: "https://example.com/12345", OriginalURL: "https://example.com/very/long/"},
					{ShortURL: "https://example.com/67890", OriginalURL: "https://google.com/"},
				},
				err: nil,
			},
			want: want{
				code:    codes.OK,
				hasData: true,
				count:   2,
			},
		},
		{
			name: "empty list",
			service: &serviceData{
				value: []models.UserURL{},
				err:   nil,
			},
			want: want{
				code:    codes.OK,
				hasData: false,
				count:   0,
			},
		},
		{
			name: "unknown error",
			service: &serviceData{
				value: nil,
				err:   fmt.Errorf("unknown error"),
			},
			want: want{
				code:    codes.Internal,
				message: "failed to get user urls",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := NewMockService(t)
			if tt.service != nil {
				mockSvc.EXPECT().GetUserURLs(mock.Anything, "", "").Return(tt.service.value, tt.service.err)
			} else {
				mockSvc.AssertNotCalled(t, "GetUserURLs", mock.Anything, mock.Anything, mock.Anything)
			}
			handler := NewGRPCHandler(mockSvc)
			ctx := context.Background()
			resp, err := handler.ListUserURLs(ctx, &emptypb.Empty{})

			if tt.want.code == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				if tt.want.hasData {
					assert.Len(t, resp.Urls, tt.want.count)
					// проверяем поля первого элемента для уверенности
					if tt.want.count > 0 {
						assert.Equal(t, "https://example.com/12345", resp.Urls[0].ShortUrl)
						assert.Equal(t, "https://example.com/very/long/", resp.Urls[0].OriginalUrl)
					}
				} else {
					assert.Empty(t, resp.Urls)
				}
			} else {
				assert.Nil(t, resp)
				assert.Error(t, err)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.want.code, st.Code())
				assert.Equal(t, tt.want.message, st.Message())
			}
			mockSvc.AssertExpectations(t)
		})
	}
}
