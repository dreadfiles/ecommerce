package handler

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockImportService struct {
	importFunc func(context.Context, io.Reader) error
	called     bool
}

func (m *mockImportService) Import(
	ctx context.Context,
	reader io.Reader,
) error {
	m.called = true

	if m.importFunc == nil {
		return errors.New("import mock not configured")
	}

	return m.importFunc(ctx, reader)
}

func createMultipartRequest(
	t *testing.T,
	content string,
) *http.Request {
	t.Helper()

	var body strings.Builder

	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile(
		"file",
		"products.csv",
	)
	if err != nil {
		t.Fatalf(
			"failed to create multipart file: %v",
			err,
		)
	}

	if _, err := part.Write(
		[]byte(content),
	); err != nil {
		t.Fatalf(
			"failed to write multipart content: %v",
			err,
		)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf(
			"failed to close multipart writer: %v",
			err,
		)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/products/import",
		strings.NewReader(body.String()),
	)

	request.Header.Set(
		"Content-Type",
		writer.FormDataContentType(),
	)

	return request
}

func createMultipartRequestWithoutFile(
	t *testing.T,
) *http.Request {
	t.Helper()

	var body strings.Builder

	writer := multipart.NewWriter(&body)

	if err := writer.WriteField(
		"name",
		"test",
	); err != nil {
		t.Fatalf(
			"failed to write form field: %v",
			err,
		)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf(
			"failed to close multipart writer: %v",
			err,
		)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/products/import",
		strings.NewReader(body.String()),
	)

	request.Header.Set(
		"Content-Type",
		writer.FormDataContentType(),
	)

	return request
}

func TestNewProductImportHandler(t *testing.T) {
	mockService := &mockImportService{}

	handler := NewProductImportHandler(mockService)

	if handler == nil {
		t.Fatal("expected handler, got nil")
	}

	if handler.importService != mockService {
		t.Fatal("expected import service to be assigned")
	}
}

func TestProductImportHandler_Import(t *testing.T) {
	tests := []struct {
		name           string
		request        *http.Request
		serviceError   error
		expectedStatus int
		expectedError  string
		expectCalled   bool
	}{
		{
			name: "imports csv successfully",
			request: createMultipartRequest(
				t,
				"name,sku,description,category,price,stock,weight_kg\n"+
					"Laptop,LAP-001,Description,Computers,10.00,10,1.250\n",
			),
			expectedStatus: http.StatusCreated,
			expectCalled:   true,
		},
		{
			name: "maps service error",
			request: createMultipartRequest(
				t,
				"name,sku,description,category,price,stock,weight_kg\n"+
					"Laptop,LAP-001,Description,Computers,invalid,10,1.250\n",
			),
			serviceError:   errors.New("invalid csv"),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid csv",
			expectCalled:   true,
		},
		{
			name: "rejects invalid multipart form",
			request: httptest.NewRequest(
				http.MethodPost,
				"/api/v1/products/import",
				strings.NewReader(""),
			),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid multipart form",
			expectCalled:   false,
		},
		{
			name:           "rejects missing file",
			request:        createMultipartRequestWithoutFile(t),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "csv file is required",
			expectCalled:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockImportService{
				importFunc: func(
					context.Context,
					io.Reader,
				) error {
					return tt.serviceError
				},
			}

			handler := NewProductImportHandler(mockService)

			response := httptest.NewRecorder()

			handler.Import(
				response,
				tt.request,
			)

			if response.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d; body=%q",
					tt.expectedStatus,
					response.Code,
					response.Body.String(),
				)
			}

			if tt.expectedError != "" &&
				!strings.Contains(
					response.Body.String(),
					tt.expectedError,
				) {
				t.Errorf(
					"expected body containing %q, got %q",
					tt.expectedError,
					response.Body.String(),
				)
			}

			if mockService.called != tt.expectCalled {
				t.Errorf(
					"expected service called=%v, got %v",
					tt.expectCalled,
					mockService.called,
				)
			}
		})
	}
}

func TestProductImportHandler_Import_RequiresFile(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/products/import",
		strings.NewReader("not multipart"),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	mockService := &mockImportService{}

	handler := NewProductImportHandler(mockService)

	response := httptest.NewRecorder()

	handler.Import(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status 400, got %d",
			response.Code,
		)
	}

	if mockService.called {
		t.Fatal("service should not be called")
	}
}
