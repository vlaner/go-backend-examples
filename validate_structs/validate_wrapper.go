package main

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

type StructValidator struct {
	uni      *ut.UniversalTranslator
	validate *validator.Validate
}

func NewStructValidator() *StructValidator {
	en := en.New()

	sv := StructValidator{
		uni:      ut.New(en, en),
		validate: validator.New(),
	}

	trans, _ := sv.uni.GetTranslator("en")
	en_translations.RegisterDefaultTranslations(sv.validate, trans)

	return &sv
}

func (sv *StructValidator) UseJsonTags() {
	sv.validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]

		if name == "-" {
			return ""
		}

		return name
	})
}

func (sv *StructValidator) ValidateStruct(s any, locale string) (map[string]string, error) {
	err := sv.validate.Struct(s)
	validationErrors, ok := err.(validator.ValidationErrors)
	if validationErrors == nil {
		return nil, nil
	}
	if !ok {
		return nil, fmt.Errorf("error is not of type 'validator.ValidationErrors': %w", err)
	}

	trans, exists := sv.uni.GetTranslator(locale)
	if !exists {
		trans = sv.uni.GetFallback()
	}

	errorMessages := make(map[string]string)
	for _, fe := range validationErrors {
		errorMessages[fe.Field()] = fe.Translate(trans)
	}

	return errorMessages, nil
}
