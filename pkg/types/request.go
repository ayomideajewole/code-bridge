package types

type TranslateRequest struct {
	Code           string `json:"code" binding:"required"`
	TargetLanguage string `json:"target_language" binding:"required"`
	SourceLanguage string `json:"source_language"`
}
