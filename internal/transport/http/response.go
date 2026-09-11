package http

import (
	"encoding/json"
	nethttp "net/http"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func WriteJSON(
	w nethttp.ResponseWriter,
	status int,
	data any,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}

func WriteError(
	w nethttp.ResponseWriter,
	status int,
	message string,
) {
	WriteJSON(
		w,
		status,
		ErrorResponse{
			Error: message,
		},
	)
}
