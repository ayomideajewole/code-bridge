package services

import "code-bridge/internal/code_translator"

// Services holds all application services
type Services struct {
	CodeTranslatorService *code_translator.CodeTranslatorService
}

// NewServices creates and initializes all services
func NewServices(translatorService *code_translator.CodeTranslatorService) *Services {
	return &Services{
		CodeTranslatorService: translatorService,
	}
}
