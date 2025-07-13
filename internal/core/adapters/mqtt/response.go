package mqtt

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/domain"
)

type ErrorResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	RequestID string    `json:"request_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// SuccessResponse represents successful operation response
type SuccessResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

type ResponsePublisher struct {
	client mqtt.Client
}

func NewResponsePublisher(client mqtt.Client) *ResponsePublisher {
	return &ResponsePublisher{
		client: client,
	}
}

// PublishSuccess publishes a success response
func (rp *ResponsePublisher) PublishSuccess(responseTopic string, data any, requestID string) error {
	response := SuccessResponse{
		Success:   true,
		Data:      data,
		RequestID: requestID,
		Timestamp: time.Now(),
	}

	return rp.publishResponse(responseTopic, response)
}

// PublishError publishes an error response
func (rp *ResponsePublisher) PublishError(responseTopic string, err error, requestID string) error {
	var message string
	switch {
	case errors.Is(err, domain.ErrNotImplemented):
		message = "not implemented"
	case errors.Is(err, domain.ErrComponentDisabled):
		message = "component disabled"
	case errors.Is(err, domain.ErrPermissionDenied):
		message = "not allowed"
	default:
		// don't leak sensitive information
		message = "internal server error"
	}

	response := ErrorResponse{
		Success:   false,
		Message:   message,
		RequestID: requestID,
		Timestamp: time.Now(),
	}

	return rp.publishResponse(responseTopic, response)
}

func (rp *ResponsePublisher) publishResponse(topic string, response interface{}) error {
	payload, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	token := rp.client.Publish(topic, 1, false, payload)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish response: %w", token.Error())
	}

	log.Debug().Str("component", "mqtt").Msgf("Published response to topic: %s", topic)
	return nil
}
