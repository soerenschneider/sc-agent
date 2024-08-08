package domain

type WolWakeupRequest struct {
	Alias string `json:"alias" validate:"required"`
}
