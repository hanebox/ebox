package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/services/crypto"
	"ebox/app/services/realtime"
)

const (
	maxAttempts    = 3
	retryDelay     = 2 * time.Second
	historyRetain  = 200
	secretSentinel = "__stored__"
)

// secretKeys are configuration values that are encrypted at rest and never
// returned by the API.
var secretKeys = map[string][]string{
	"email":    {"password"},
	"telegram": {"bot_token"},
	"webhook":  {"secret"},
}

type Input struct {
	Name    string
	Type    string
	Enabled bool
	Config  map[string]string
	Events  []string
}

type Service struct{}

func Default() *Service {
	return &Service{}
}

func (s *Service) List() ([]models.NotificationChannel, error) {
	var channels []models.NotificationChannel
	if err := facades.Orm().Query().Order("name asc").Find(&channels); err != nil {
		return nil, err
	}

	return channels, nil
}

func (s *Service) Find(uuidValue string) (*models.NotificationChannel, error) {
	var channel models.NotificationChannel
	if err := facades.Orm().Query().Where("uuid", uuidValue).FirstOrFail(&channel); err != nil {
		return nil, err
	}

	return &channel, nil
}

// Config decrypts the channel configuration, including secrets.
func (s *Service) Config(channel *models.NotificationChannel) (map[string]string, error) {
	values := map[string]string{}

	if strings.TrimSpace(channel.Config) == "" {
		return values, nil
	}

	payload, err := crypto.Decrypt(channel.Config)
	if err != nil {
		return nil, errors.New("notifications: the stored configuration could not be read")
	}

	if err := json.Unmarshal([]byte(payload), &values); err != nil {
		return nil, errors.New("notifications: the stored configuration is invalid")
	}

	return values, nil
}

// PublicConfig hides secret values so they can be shown in the UI.
func (s *Service) PublicConfig(channel *models.NotificationChannel) map[string]string {
	values, err := s.Config(channel)
	if err != nil {
		return map[string]string{}
	}

	for _, key := range secretKeys[channel.Type] {
		if values[key] != "" {
			values[key] = secretSentinel
		}
	}

	return values
}

// Events returns the subscribed event names. An empty list means every event.
func (s *Service) Events(channel *models.NotificationChannel) []string {
	if strings.TrimSpace(channel.Events) == "" {
		return []string{}
	}

	var events []string
	if err := json.Unmarshal([]byte(channel.Events), &events); err != nil {
		return []string{}
	}

	return events
}

func (s *Service) Create(input Input) (*models.NotificationChannel, error) {
	channel := &models.NotificationChannel{UUID: uuid.NewString()}

	if err := s.apply(channel, input); err != nil {
		return nil, err
	}

	if err := facades.Orm().Query().Create(channel); err != nil {
		return nil, err
	}

	return channel, nil
}

func (s *Service) Update(channel *models.NotificationChannel, input Input) error {
	if err := s.apply(channel, input); err != nil {
		return err
	}

	return facades.Orm().Query().Save(channel)
}

func (s *Service) Delete(channel *models.NotificationChannel) error {
	_, _ = facades.Orm().Query().Where("channel_id", channel.ID).Delete(&models.NotificationDelivery{})

	_, err := facades.Orm().Query().Delete(channel)

	return err
}

func (s *Service) SetEnabled(channel *models.NotificationChannel, enabled bool) error {
	channel.Enabled = enabled

	return facades.Orm().Query().Save(channel)
}

func (s *Service) apply(channel *models.NotificationChannel, input Input) error {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return errors.New("notifications: the name is required")
	}

	if _, ok := senders[input.Type]; !ok {
		return fmt.Errorf("notifications: unsupported channel type %q", input.Type)
	}

	config := input.Config
	if config == nil {
		config = map[string]string{}
	}

	// A secret the client sent back unchanged keeps its stored value.
	if existing, err := s.Config(channel); err == nil {
		for _, key := range secretKeys[input.Type] {
			if config[key] == secretSentinel || config[key] == "" {
				if value, ok := existing[key]; ok {
					config[key] = value
				}
			}
		}
	}

	for _, event := range input.Events {
		known := false
		for _, supported := range models.NotificationEvents() {
			if event == supported {
				known = true
				break
			}
		}
		if !known {
			return fmt.Errorf("notifications: unknown event %q", event)
		}
	}

	payload, err := json.Marshal(config)
	if err != nil {
		return err
	}

	encrypted, err := crypto.Encrypt(string(payload))
	if err != nil {
		return err
	}

	events, err := json.Marshal(input.Events)
	if err != nil {
		return err
	}

	channel.Name = name
	channel.Type = input.Type
	channel.Enabled = input.Enabled
	channel.Config = encrypted
	channel.Events = string(events)

	return nil
}

// Notify delivers an event to every subscribed channel. It never blocks the
// caller: delivery, retries and history happen in the background.
func (s *Service) Notify(event, title, body string, data map[string]any) {
	message := Message{Event: event, Title: title, Body: body, Data: data}

	s.publish(event, title, body)

	go func() {
		channels, err := s.subscribers(event)
		if err != nil || len(channels) == 0 {
			return
		}

		var wait sync.WaitGroup
		for _, channel := range channels {
			wait.Add(1)

			go func(target models.NotificationChannel) {
				defer wait.Done()
				s.deliver(&target, message)
			}(channel)
		}

		wait.Wait()
		s.trimHistory()
	}()
}

// Test sends a message through one channel, returning the delivery error.
func (s *Service) Test(ctx context.Context, channel *models.NotificationChannel) error {
	message := Message{
		Event: "test",
		Title: "Test notification from ebox",
		Body:  "If you can read this, " + channel.Name + " is configured correctly.",
		Data:  map[string]any{"channel": channel.Name},
	}

	config, err := s.Config(channel)
	if err != nil {
		return err
	}

	deliveryErr := Send(ctx, channel.Type, config, message)
	s.record(channel, message, deliveryErr, 1, false)

	return deliveryErr
}

func (s *Service) subscribers(event string) ([]models.NotificationChannel, error) {
	var channels []models.NotificationChannel
	if err := facades.Orm().Query().Where("enabled", true).Find(&channels); err != nil {
		return nil, err
	}

	matching := make([]models.NotificationChannel, 0, len(channels))
	for _, channel := range channels {
		events := s.Events(&channel)
		if len(events) == 0 {
			matching = append(matching, channel)

			continue
		}

		for _, candidate := range events {
			if candidate == event {
				matching = append(matching, channel)

				break
			}
		}
	}

	return matching, nil
}

// deliver sends a message with retries and records the result.
func (s *Service) deliver(channel *models.NotificationChannel, message Message) {
	config, err := s.Config(channel)
	if err != nil {
		s.record(channel, message, err, 0, false)

		return
	}

	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		lastErr = Send(context.Background(), channel.Type, config, message)
		if lastErr == nil {
			s.record(channel, message, nil, attempt, false)

			return
		}

		if attempt < maxAttempts {
			time.Sleep(retryDelay * time.Duration(attempt))
		}
	}

	s.record(channel, message, lastErr, maxAttempts, true)
}

func (s *Service) record(channel *models.NotificationChannel, message Message, failure error, attempts int, retried bool) {
	delivery := &models.NotificationDelivery{
		ChannelID: channel.ID,
		Channel:   channel.Name,
		Event:     message.Event,
		Title:     message.Title,
		Body:      message.Body,
		Status:    models.NotificationStatusSent,
		Attempts:  attempts,
	}

	if failure != nil {
		delivery.Status = models.NotificationStatusFailed
		delivery.Error = failure.Error()
	}

	if err := facades.Orm().Query().Create(delivery); err != nil {
		return
	}

	if log := facades.Log(); log != nil && failure != nil {
		if retried {
			log.Warningf("ebox: notification %s to %s failed after %d attempts: %s", message.Event, channel.Name, attempts, failure)
		} else {
			log.Warningf("ebox: notification %s to %s failed: %s", message.Event, channel.Name, failure)
		}
	}
}

// History returns the newest deliveries.
func (s *Service) History(limit int) ([]models.NotificationDelivery, error) {
	if limit <= 0 || limit > historyRetain {
		limit = 50
	}

	var deliveries []models.NotificationDelivery
	if err := facades.Orm().Query().Order("id desc").Limit(limit).Find(&deliveries); err != nil {
		return nil, err
	}

	return deliveries, nil
}

// trimHistory keeps the delivery table small.
func (s *Service) trimHistory() {
	var stale []models.NotificationDelivery
	if err := facades.Orm().Query().Order("id desc").Offset(historyRetain).Find(&stale); err != nil || len(stale) == 0 {
		return
	}

	ids := make([]uint64, 0, len(stale))
	for _, delivery := range stale {
		ids = append(ids, delivery.ID)
	}

	_, _ = facades.Orm().Query().Where("id in ?", ids).Delete(&models.NotificationDelivery{})
}

// publish pushes the notification to connected browsers.
func (s *Service) publish(event, title, body string) {
	realtime.Default().Publish(realtime.Event{
		Topic: "notifications",
		Type:  "notification",
		Data: map[string]any{
			"event": event,
			"title": title,
			"body":  body,
			"at":    time.Now().UTC(),
		},
	})
}
