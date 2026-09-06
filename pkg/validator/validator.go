package validator

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/invopop/ctxi18n"
	"github.com/invopop/ctxi18n/i18n"
)

type fieldErrorDetail struct {
	field string
	tag   string
	param string
	kind  reflect.Kind
}

// ValidationError represents one or more human-friendly validation errors.
type ValidationError struct {
	details  []fieldErrorDetail
	messages []string
}

func (ve *ValidationError) Error() string {
	return strings.Join(ve.messages, "; ")
}

func (ve *ValidationError) ErrorMessages() []string {
	return ve.messages
}

func (ve *ValidationError) Localize(ctx context.Context) []string {
	if ctx == nil {
		return ve.messages
	}
	loc := ctxi18n.Locale(ctx)
	if loc == nil {
		return ve.messages
	}
	msgs := make([]string, len(ve.details))
	for i, d := range ve.details {
		msgs[i] = formatFieldErrorDetail(loc, d)
	}
	return msgs
}

func (ve *ValidationError) Unwrap() []error {
	errs := make([]error, len(ve.messages))
	for i, msg := range ve.messages {
		var detail fieldErrorDetail
		if i < len(ve.details) {
			detail = ve.details[i]
		}
		errs[i] = &fieldError{msg: msg, detail: detail}
	}
	return errs
}

type fieldError struct {
	msg    string
	detail fieldErrorDetail
}

func (fe *fieldError) Error() string {
	return fe.msg
}

func (fe *fieldError) Localize(ctx context.Context) string {
	if ctx == nil {
		return fe.msg
	}
	loc := ctxi18n.Locale(ctx)
	if loc == nil {
		return fe.msg
	}
	return formatFieldErrorDetail(loc, fe.detail)
}

type CustomValidator struct {
	validator *validator.Validate
}

func New() *CustomValidator {
	v := validator.New()

	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		for _, tag := range []string{"json", "query", "form"} {
			name := strings.SplitN(fld.Tag.Get(tag), ",", 2)[0]
			if name != "" && name != "-" {
				return name
			}
		}
		return fld.Name
	})

	return &CustomValidator{validator: v}
}

func (cv *CustomValidator) Validate(i any) error {
	if cv == nil || cv.validator == nil {
		return nil
	}

	err := cv.validator.Struct(i)
	if err == nil {
		return nil
	}

	valErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}

	details := make([]fieldErrorDetail, 0, len(valErrors))
	messages := make([]string, 0, len(valErrors))
	for _, fe := range valErrors {
		detail := fieldErrorDetail{
			field: fe.Field(),
			tag:   fe.Tag(),
			param: fe.Param(),
			kind:  fe.Kind(),
		}
		details = append(details, detail)
		messages = append(messages, formatFieldErrorDetail(nil, detail))
	}

	return &ValidationError{details: details, messages: messages}
}

func translate(l *i18n.Locale, key string, fallback string, args ...any) string {
	if l != nil && l.Has(key) {
		return l.T(key, args...)
	}
	return fmt.Sprintf(fallback, args...)
}

func formatFieldErrorDetail(loc *i18n.Locale, d fieldErrorDetail) string {
	field := d.field
	param := d.param

	switch d.tag {
	case "required":
		return translate(loc, "validation.required", "%s is required", field)
	case "email":
		return translate(loc, "validation.email", "%s must be a valid email address", field)
	case "min":
		switch d.kind {
		case reflect.Slice, reflect.Array, reflect.Map:
			if param == "1" {
				return translate(loc, "validation.min_items_one", "%s must contain at least 1 item", field)
			}
			return translate(loc, "validation.min_items", "%s must contain at least %s items", field, param)
		case reflect.String:
			if param == "1" {
				return translate(loc, "validation.min_string_empty", "%s cannot be empty", field)
			}
			return translate(loc, "validation.min_string", "%s must be at least %s characters long", field, param)
		default:
			return translate(loc, "validation.min_numeric", "%s must be at least %s", field, param)
		}
	case "max":
		switch d.kind {
		case reflect.Slice, reflect.Array, reflect.Map:
			return translate(loc, "validation.max_items", "%s must contain at most %s items", field, param)
		case reflect.String:
			return translate(loc, "validation.max_string", "%s must be at most %s characters long", field, param)
		default:
			return translate(loc, "validation.max_numeric", "%s must be at most %s", field, param)
		}
	case "gte":
		switch d.kind {
		case reflect.Slice, reflect.Array, reflect.Map:
			return translate(loc, "validation.gte_items", "%s must contain at least %s items", field, param)
		case reflect.String:
			return translate(loc, "validation.gte_string", "%s must be at least %s characters long", field, param)
		default:
			return translate(loc, "validation.gte_numeric", "%s must be at least %s", field, param)
		}
	case "lte":
		switch d.kind {
		case reflect.Slice, reflect.Array, reflect.Map:
			return translate(loc, "validation.lte_items", "%s must contain at most %s items", field, param)
		case reflect.String:
			return translate(loc, "validation.lte_string", "%s must be at most %s characters long", field, param)
		default:
			return translate(loc, "validation.lte_numeric", "%s must be at most %s", field, param)
		}
	case "gt":
		return translate(loc, "validation.gt", "%s must be greater than %s", field, param)
	case "lt":
		return translate(loc, "validation.lt", "%s must be less than %s", field, param)
	case "oneof":
		options := strings.Join(strings.Fields(param), ", ")
		return translate(loc, "validation.oneof", "%s must be one of: %s", field, options)
	case "url":
		return translate(loc, "validation.url", "%s must be a valid URL", field)
	case "uuid":
		return translate(loc, "validation.uuid", "%s must be a valid UUID", field)
	case "eqfield":
		return translate(loc, "validation.eqfield", "%s must match %s", field, param)
	case "gtfield":
		return translate(loc, "validation.gtfield", "%s must be greater than %s", field, param)
	case "gtefield":
		return translate(loc, "validation.gtefield", "%s must be greater than or equal to %s", field, param)
	case "ltfield":
		return translate(loc, "validation.ltfield", "%s must be less than %s", field, param)
	case "ltefield":
		return translate(loc, "validation.ltefield", "%s must be less than or equal to %s", field, param)
	case "nefield":
		return translate(loc, "validation.nefield", "%s cannot be equal to %s", field, param)
	default:
		return translate(loc, "validation.invalid", "%s is invalid", field)
	}
}
