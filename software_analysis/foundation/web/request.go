package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/dimfeld/httptreemux/v5"
	en "github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

// validate holds the settings and caches for validating request struct values
var validate *validator.Validate

// translator is a cache of locale and translation information.
var translator ut.Translator

func init() {

	// Instantiate a validator.
	validate = validator.New()

	// Create a translator for english so the error messages are
	// more human readable than technical.
	translator, _ = ut.New(en.New(), en.New()).GetTranslator("en")

	// Register the english error messages for use.
	en_translations.RegisterDefaultTranslations(validate, translator)

	// Use JSON tag names for errors instead of Go struct names.
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

// Param returns the web call parameters from the request
func Params(r *http.Request) map[string]string {
	return httptreemux.ContextParams(r.Context())
}

// Decode reads the body of an HTTP request looking for a JSON document. The
// body is decoded into the provided value.
//
// If the provided value is a struct then it is checked for validation tags.
func Decode(r *http.Request, val interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(val); err != nil {
		return NewRequestError(err, http.StatusBadRequest)
	}

	if err := validate.Struct(val); err != nil {
		// Use a type assertion to get the real error value
		verrors, ok := err.(validator.ValidationErrors)
		if !ok {
			return err
		}

		var fields []FieldError
		for _, verror := range verrors {
			field := FieldError {
				Field: verror.Field(),
				Error: verror.Translate(translator),
			}
			fields = append(fields, field)
		}
		return &Error{
			Err: errors.New("field validate error"),
			Status: http.StatusBadRequest,
			Fields: fields,
		}
	}

	return nil
}
