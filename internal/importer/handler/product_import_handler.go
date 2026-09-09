package handler

import (
	"fmt"
	"net/http"

	importservice "ecommerce/internal/importer/service"
)

const (
	importFileField      = "file"
	maxImportRequestSize = 10 * 1024 * 1024
)

type ProductImportHandler struct {
	importService importservice.ProductImportService
}

func NewProductImportHandler(
	importService importservice.ProductImportService,
) *ProductImportHandler {
	return &ProductImportHandler{
		importService: importService,
	}
}

func (h *ProductImportHandler) Import(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxImportRequestSize,
	)

	if err := r.ParseMultipartForm(maxImportRequestSize); err != nil {
		http.Error(
			w,
			fmt.Sprintf("invalid multipart form: %v", err),
			http.StatusBadRequest,
		)
		return
	}

	file, _, err := r.FormFile(importFileField)
	if err != nil {
		http.Error(
			w,
			"csv file is required",
			http.StatusBadRequest,
		)
		return
	}
	defer file.Close()

	if err := h.importService.Import(r.Context(), file); err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
