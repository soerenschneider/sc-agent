package mqtt

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/config"
)

const (
	defaultPrefix = "sc-agent"
	groupPrefix   = "group"
)

type MessageHandler interface {
	Handle(ctx context.Context, topic string, payload []byte) (any, error)
}

// TopicRouter routes messages to appropriate handlers based on topic patterns
type TopicRouter struct {
	responsePublisher *ResponsePublisher
	handlers          map[string]MessageHandler
	client            mqtt.Client
	clientId          string
	mu                sync.RWMutex

	topicHostPrefix    string
	topicGroupPrefixes []string
	globalPrefix       string
}

func NewTopicRouter(client mqtt.Client, conf *config.Mqtt) (*TopicRouter, error) {
	if client == nil {
		return nil, errors.New("nil client passed")
	}

	if conf == nil {
		return nil, errors.New("nil config passed")
	}

	return &TopicRouter{
		handlers:           make(map[string]MessageHandler),
		client:             client,
		clientId:           conf.ClientId,
		topicHostPrefix:    conf.TopicHostPrefix,
		topicGroupPrefixes: conf.TopicGroupPrefixes,
		responsePublisher: &ResponsePublisher{
			client: client,
		},
	}, nil
}

func getTopicForGroup(globalPrefix, groupName, topicPattern string) string {
	globalPrefix = cmp.Or(globalPrefix, defaultPrefix)
	return path.Join(globalPrefix, groupPrefix, groupName, topicPattern)
}

func (tr *TopicRouter) MustRegisterHandler(topicPattern string, handler MessageHandler) {
	if err := tr.RegisterHandler(topicPattern, handler); err != nil {
		log.Fatal().Err(err).Msg("could not register handler")
	}
}

func (tr *TopicRouter) RegisterHandler(topicPattern string, handler MessageHandler) error {
	if strings.TrimSpace(topicPattern) == "" {
		return errors.New("empty topic pattern provided")
	}

	if handler == nil {
		return errors.New("empty handler provided")
	}

	tr.mu.Lock()
	defer tr.mu.Unlock()

	for _, topicGroupPrefix := range tr.topicGroupPrefixes {
		if topicGroupPrefix != "" {
			topic := getTopicForGroup(tr.globalPrefix, topicGroupPrefix, topicPattern)
			_, alreadyExists := tr.handlers[topic]
			if alreadyExists {
				return fmt.Errorf("topic %q already exists", topic)
			}
			log.Debug().Str("kind", "group").Msgf("Registering handler for topic %s", topic)
			tr.handlers[topic] = handler
		}
	}

	globalPrefix := cmp.Or(tr.globalPrefix, defaultPrefix)
	topic := path.Join(globalPrefix, tr.topicHostPrefix, topicPattern)
	_, alreadyExists := tr.handlers[topic]
	if alreadyExists {
		return fmt.Errorf("topic %q already exists", topic)
	}
	log.Debug().Str("kind", "host").Msgf("Registering handler for topic %s", topic)
	tr.handlers[topic] = handler

	return nil
}

// Subscribe subscribes to all registered topic patterns
func (tr *TopicRouter) Subscribe() error {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	for topicPattern := range tr.handlers {
		token := tr.client.Subscribe(topicPattern, 1, tr.messageHandler)
		if token.WaitTimeout(3*time.Second) && token.Error() != nil {
			return fmt.Errorf("failed to subscribe to topic %s: %w", topicPattern, token.Error())
		}
		log.Debug().Str("component", "mqtt").Msgf("Subscribed to topic %s", topicPattern)
	}

	return nil
}

// messageHandler is the callback function that routes messages to appropriate handlers
func (tr *TopicRouter) messageHandler(client mqtt.Client, msg mqtt.Message) {
	ctx := context.Background()
	topic := msg.Topic()
	payload := msg.Payload()

	log.Debug().Str("component", "mqtt").Msgf("Received message on topic %q", topic)

	// Find matching handler
	handler := tr.findHandler(topic)
	if handler == nil {
		log.Error().Str("component", "mqtt").Msgf("No handler found for topic: %q", topic)
		return
	}

	// Handle the message
	response, err := handler.Handle(ctx, topic, payload)
	if err != nil {
		log.Printf("Error handling message on topic %s: %v", topic, err)
		responseTopic := getResponseTopic(topic, tr.clientId)
		_ = tr.responsePublisher.PublishError(responseTopic, err, "")
	} else {
		responseTopic := getResponseTopic(topic, tr.clientId)
		_ = tr.responsePublisher.PublishSuccess(responseTopic, response, "")
	}
}

func getResponseTopic(requestTopic, clientID string) string {
	// Convert request topic to response topic
	// replication/create -> replication/response/create
	// replication/user/create -> replication/user/response/create
	parts := strings.Split(requestTopic, "/")
	if len(parts) >= 2 {
		// Insert "response" before the last part
		result := make([]string, 0, len(parts)+1)
		result = append(result, parts[:len(parts)-1]...)
		result = append(result, "response")
		result = append(result, parts[len(parts)-1])

		// If client ID is provided, make it client-specific
		if parts[1] == groupPrefix {
			clientID = strings.ReplaceAll(clientID, fmt.Sprintf("%s-", config.DefaultMqttClientIdPrefix), "")
			result = append(result, clientID)
		}

		return strings.Join(result, "/")
	}

	return path.Join(parts[0], "response")
}

// findHandler finds the appropriate handler for a given topic
func (tr *TopicRouter) findHandler(topic string) MessageHandler {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	// Direct match first
	if handler, exists := tr.handlers[topic]; exists {
		return handler
	}

	// Pattern matching for wildcards
	for pattern, handler := range tr.handlers {
		if tr.matchTopic(pattern, topic) {
			return handler
		}
	}

	return nil
}

// matchTopic checks if a topic matches a pattern (supports + and # wildcards)
func (tr *TopicRouter) matchTopic(pattern, topic string) bool {
	patternParts := strings.Split(pattern, "/")
	topicParts := strings.Split(topic, "/")

	return tr.matchParts(patternParts, topicParts)
}

func (tr *TopicRouter) matchParts(patternParts, topicParts []string) bool {
	i, j := 0, 0

	for i < len(patternParts) && j < len(topicParts) {
		if patternParts[i] == "#" {
			return true // # matches everything remaining
		}
		if patternParts[i] == "+" || patternParts[i] == topicParts[j] {
			i++
			j++
		} else {
			return false
		}
	}

	// Check if we consumed all parts
	if i < len(patternParts) && patternParts[i] == "#" {
		return true
	}

	return i == len(patternParts) && j == len(topicParts)
}
