package http

import (
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	response := httptest.NewRecorder()

	payload := map[string]string{
		"message": "success",
	}

	WriteJSON(
		response,
		nethttp.StatusCreated,
		payload,
	)

	if response.Code != nethttp.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d",
			nethttp.StatusCreated,
			response.Code,
		)
	}

	if contentType := response.Header().Get(
		"Content-Type",
	); contentType != "application/json" {
		t.Errorf(
			"expected Content-Type application/json, got %q",
			contentType,
		)
	}

	var body map[string]string

	if err := json.NewDecoder(
		response.Body,
	).Decode(&body); err != nil {
		t.Fatalf(
			"failed to decode response: %v",
			err,
		)
	}

	if body["message"] != "success" {
		t.Errorf(
			"expected message=success, got %q",
			body["message"],
		)
	}
}

func TestWriteError(t *testing.T) {
	response := httptest.NewRecorder()

	WriteError(
		response,
		nethttp.StatusBadRequest,
		"invalid request",
	)

	if response.Code != nethttp.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			nethttp.StatusBadRequest,
			response.Code,
		)
	}

	if contentType := response.Header().Get(
		"Content-Type",
	); contentType != "application/json" {
		t.Errorf(
			"expected Content-Type application/json, got %q",
			contentType,
		)
	}

	var body ErrorResponse

	if err := json.NewDecoder(
		response.Body,
	).Decode(&body); err != nil {
		t.Fatalf(
			"failed to decode response: %v",
			err,
		)
	}

	if body.Error != "invalid request" {
		t.Errorf(
			"expected error=%q, got %q",
			"invalid request",
			body.Error,
		)
	}
}

func TestWriteError_ReturnsJSONBody(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		message        string
		expectedStatus int
	}{
		{
			name:           "bad request",
			status:         nethttp.StatusBadRequest,
			message:        "invalid request",
			expectedStatus: nethttp.StatusBadRequest,
		},
		{
			name:           "not found",
			status:         nethttp.StatusNotFound,
			message:        "resource not found",
			expectedStatus: nethttp.StatusNotFound,
		},
		{
			name:           "conflict",
			status:         nethttp.StatusConflict,
			message:        "resource already exists",
			expectedStatus: nethttp.StatusConflict,
		},
		{
			name:           "internal server error",
			status:         nethttp.StatusInternalServerError,
			message:        "internal server error",
			expectedStatus: nethttp.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()

			WriteError(
				response,
				tt.status,
				tt.message,
			)

			if response.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d",
					tt.expectedStatus,
					response.Code,
				)
			}

			var body ErrorResponse

			if err := json.NewDecoder(
				response.Body,
			).Decode(&body); err != nil {
				t.Fatalf(
					"expected valid JSON response: %v",
					err,
				)
			}

			if body.Error != tt.message {
				t.Errorf(
					"expected error=%q, got %q",
					tt.message,
					body.Error,
				)
			}
		})
	}
}
