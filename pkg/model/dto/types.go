package dto

import (
	"errors"
)

// Errors
var (
	ErrNotAuthorized       = errors.New("not authorized")
	ErrInvalidAPIKey       = errors.New("invalid API key")
	ErrNotFound            = errors.New("not found")
	ErrBadRequest          = errors.New("bad request")
	ErrInternalServerError = errors.New("internal server error")
)

// PaginatedResponse represents a paginated API response
type PaginatedResponse[T any] struct {
	Results    []T      `json:"results"`
	Page       int      `json:"page"`
	PerPage    int      `json:"per_page"`
	TotalPages int      `json:"total_pages"`
	TotalCount int64    `json:"total_count"`
	Errors     []string `json:"errors,omitempty"`
	ErrorCodes []string `json:"error_codes,omitempty"`
}

// Response represents a simple API response
type Response[T any] struct {
	Results    T        `json:"results"`
	Errors     []string `json:"errors,omitempty"`
	ErrorCodes []string `json:"error_codes,omitempty"`
}

// AddError adds an error message to the response
func (r *Response[T]) AddError(err ...error) {
	for _, e := range err {
		r.Errors = append(r.Errors, e.Error())
	}
}

// AddError adds an error message to the paginated response
func (pr *PaginatedResponse[T]) AddError(err ...error) {
	for _, e := range err {
		pr.Errors = append(pr.Errors, e.Error())
	}
}

// PaginationParams represents pagination query parameters
type PaginationParams struct {
	Page    int `query:"page"`
	PerPage int `query:"per_page"`
}

// SetDefaults sets default values for pagination
func (p *PaginationParams) SetDefaults() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PerPage < 1 {
		p.PerPage = 20
	}
	if p.PerPage > 100 {
		p.PerPage = 100
	}
}

// GetOffset calculates the database offset
func (p *PaginationParams) GetOffset() int {
	return (p.Page - 1) * p.PerPage
}

// CalculateTotalPages calculates total pages from total count
func (p *PaginationParams) CalculateTotalPages(totalCount int64) int {
	if p.PerPage == 0 {
		return 0
	}
	pages := int(totalCount) / p.PerPage
	if int(totalCount)%p.PerPage > 0 {
		pages++
	}
	return pages
}
