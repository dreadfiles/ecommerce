package handler

import (
	"fmt"
	"net/http"

	importservice "ecommerce/internal/importer/service"
	httptransport "ecommerce/internal/transport/http"
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

func (h *ProductImportHandler) Import(
	w http.ResponseWriter,
	r *http.Request,
) {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxImportRequestSize,
	)

	if err := r.ParseMultipartForm(
		maxImportRequestSize,
	); err != nil {
		httptransport.WriteError(
			w,
			http.StatusBadRequest,
			fmt.Sprintf(
				"invalid multipart form: %v",
				err,
			),
		)
		return
	}

	file, _, err := r.FormFile(importFileField)
	if err != nil {
		httptransport.WriteError(
			w,
			http.StatusBadRequest,
			"csv file is required",
		)
		return
	}
	defer file.Close()

	if err := h.importService.Import(
		r.Context(),
		file,
	); err != nil {
		httptransport.WriteError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
