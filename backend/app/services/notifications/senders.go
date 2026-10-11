package notifications

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

const sendTimeout = 20 * time.Second

// Message is one notification to deliver.
type Message struct {
	Event string
	Title string
	Body  string
	Data  map[string]any
}

// sender delivers a message for one channel configuration.
type sender func(ctx context.Context, config map[string]string, message Message) error

var senders = map[string]sender{
	"email":    sendEmail,
	"discord":  sendWebhookJSON("content"),
	"slack":    sendWebhookJSON("text"),
	"telegram": sendTelegram,
	"webhook":  sendGenericWebhook,
}

// Send delivers a message through the given channel type.
func Send(ctx context.Context, channelType string, config map[string]string, message Message) error {
	dispatch, ok := senders[channelType]
	if !ok {
		return fmt.Errorf("notifications: unsupported channel type %q", channelType)
	}

	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	return dispatch(ctx, config, message)
}

func sendWebhookJSON(field string) sender {
	return func(ctx context.Context, config map[string]string, message Message) error {
		url := strings.TrimSpace(config["webhook_url"])
		if url == "" {
			return errors.New("notifications: the webhook url is missing")
		}

		payload := map[string]any{field: formatPlain(message)}
		if username := strings.TrimSpace(config["username"]); username != "" {
			payload["username"] = username
		}

		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		response, err := postJSON(ctx, url, body, nil)
		if err != nil {
			return err
		}
		defer func() { _ = response.Body.Close() }()

		if response.StatusCode >= 300 {
			return fmt.Errorf("notifications: the webhook answered with status %d", response.StatusCode)
		}

		return nil
	}
}

func sendTelegram(ctx context.Context, config map[string]string, message Message) error {
	token := strings.TrimSpace(config["bot_token"])
	chatID := strings.TrimSpace(config["chat_id"])

	if token == "" || chatID == "" {
		return errors.New("notifications: the bot token and chat id are required")
	}

	payload, err := json.Marshal(map[string]any{
		"chat_id":                  chatID,
		"text":                     formatPlain(message),
		"disable_web_page_preview": true,
	})
	if err != nil {
		return err
	}

	url := "https://api.telegram.org/bot" + token + "/sendMessage"

	response, err := postJSON(ctx, url, payload, nil)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(response.Body, 512))

		return fmt.Errorf("notifications: telegram answered with status %d: %s", response.StatusCode, strings.TrimSpace(string(detail)))
	}

	return nil
}

func sendGenericWebhook(ctx context.Context, config map[string]string, message Message) error {
	url := strings.TrimSpace(config["webhook_url"])
	if url == "" {
		return errors.New("notifications: the webhook url is missing")
	}

	body, err := json.Marshal(map[string]any{
		"event":   message.Event,
		"title":   message.Title,
		"message": message.Body,
		"data":    message.Data,
		"sent_at": time.Now().UTC(),
		"source":  "ebox",
	})
	if err != nil {
		return err
	}

	headers := map[string]string{}
	if secret := strings.TrimSpace(config["secret"]); secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		headers["X-Ebox-Signature"] = "sha256=" + hex.EncodeToString(mac.Sum(nil))
	}

	response, err := postJSON(ctx, url, body, headers)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode >= 300 {
		return fmt.Errorf("notifications: the webhook answered with status %d", response.StatusCode)
	}

	return nil
}

func sendEmail(ctx context.Context, config map[string]string, message Message) error {
	host := strings.TrimSpace(config["host"])
	if host == "" {
		return errors.New("notifications: the smtp host is missing")
	}

	port := 587
	if value := strings.TrimSpace(config["port"]); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return errors.New("notifications: the smtp port is invalid")
		}
		port = parsed
	}

	from := strings.TrimSpace(config["from"])
	if from == "" {
		from = strings.TrimSpace(config["username"])
	}
	if from == "" {
		return errors.New("notifications: a sender address is required")
	}

	recipients := splitList(config["to"])
	if len(recipients) == 0 {
		return errors.New("notifications: at least one recipient is required")
	}

	username := strings.TrimSpace(config["username"])
	password := config["password"]

	address := net.JoinHostPort(host, strconv.Itoa(port))
	encryption := strings.ToLower(strings.TrimSpace(config["encryption"]))

	if encryption == "" {
		switch port {
		case 465:
			encryption = "tls"
		default:
			encryption = "starttls"
		}
	}

	body := buildEmail(from, recipients, message)

	dialer := &net.Dialer{Timeout: sendTimeout}
	var connection net.Conn
	var err error

	if encryption == "tls" {
		connection, err = tls.DialWithDialer(dialer, "tcp", address, &tls.Config{ServerName: host})
	} else {
		connection, err = dialer.DialContext(ctx, "tcp", address)
	}
	if err != nil {
		return fmt.Errorf("notifications: the smtp server could not be reached (%s)", err.Error())
	}
	defer func() { _ = connection.Close() }()

	client, err := smtp.NewClient(connection, host)
	if err != nil {
		return errors.New("notifications: the smtp handshake failed")
	}
	defer func() { _ = client.Close() }()

	if encryption == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: host}); err != nil {
				return errors.New("notifications: starttls failed")
			}
		}
	}

	if username != "" && password != "" {
		auth := smtp.PlainAuth("", username, password, host)
		if err := client.Auth(auth); err != nil {
			return errors.New("notifications: the smtp server rejected the credentials")
		}
	}

	if err := client.Mail(from); err != nil {
		return errors.New("notifications: the sender address was rejected")
	}

	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("notifications: the recipient %s was rejected", recipient)
		}
	}

	writer, err := client.Data()
	if err != nil {
		return errors.New("notifications: the message could not be sent")
	}

	if _, err := writer.Write(body); err != nil {
		_ = writer.Close()

		return errors.New("notifications: the message was cut off")
	}

	if err := writer.Close(); err != nil {
		return errors.New("notifications: the message was rejected")
	}

	return client.Quit()
}

func buildEmail(from string, recipients []string, message Message) []byte {
	subject := message.Title
	if subject == "" {
		subject = "ebox notification"
	}

	var builder strings.Builder
	builder.WriteString("From: " + from + "\r\n")
	builder.WriteString("To: " + strings.Join(recipients, ", ") + "\r\n")
	builder.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", subject) + "\r\n")
	builder.WriteString("Date: " + time.Now().UTC().Format(time.RFC1123Z) + "\r\n")
	builder.WriteString("MIME-Version: 1.0\r\n")
	builder.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	builder.WriteString("\r\n")
	builder.WriteString(message.Body)
	builder.WriteString("\r\n")

	return []byte(builder.String())
}

// formatPlain is the text form shared by chat channels.
func formatPlain(message Message) string {
	if message.Body == "" {
		return message.Title
	}

	if message.Title == "" {
		return message.Body
	}

	return message.Title + "\n\n" + message.Body
}

func postJSON(ctx context.Context, url string, body []byte, headers map[string]string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, errors.New("notifications: the url is invalid")
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "ebox-notifications")

	for key, value := range headers {
		request.Header.Set(key, value)
	}

	client := &http.Client{Timeout: sendTimeout}

	response, err := client.Do(request)
	if err != nil {
		return nil, errors.New("notifications: the endpoint could not be reached (" + err.Error() + ")")
	}

	return response, nil
}

func splitList(value string) []string {
	fields := strings.FieldsFunc(value, func(character rune) bool {
		return character == ',' || character == ';' || character == '\n' || character == ' '
	})

	items := make([]string, 0, len(fields))
	for _, field := range fields {
		if trimmed := strings.TrimSpace(field); trimmed != "" {
			items = append(items, trimmed)
		}
	}

	return items
}
