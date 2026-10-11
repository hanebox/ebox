package controllers

import (
	"net/http"
	"strings"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/models"
	"ebox/app/services/notifications"
	"ebox/app/support/httpx"
)

type NotificationController struct{}

func NewNotificationController() *NotificationController {
	return &NotificationController{}
}

type notificationChannelResponse struct {
	models.NotificationChannel
	Config   map[string]string `json:"config"`
	Subscribed []string        `json:"subscribed_events"`
}

func (c *NotificationController) Index(ctx contractshttp.Context) contractshttp.Response {
	channels, err := notifications.Default().List()
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to load notification channels.")
	}

	responses := make([]notificationChannelResponse, 0, len(channels))
	for _, channel := range channels {
		responses = append(responses, notificationChannelResponse{
			NotificationChannel: channel,
			Config:              notifications.Default().PublicConfig(&channel),
			Subscribed:          notifications.Default().Events(&channel),
		})
	}

	return httpx.Ok(ctx, responses)
}

func (c *NotificationController) Events(ctx contractshttp.Context) contractshttp.Response {
	return httpx.Ok(ctx, contractshttp.Json{"events": models.NotificationEvents()})
}

func (c *NotificationController) History(ctx contractshttp.Context) contractshttp.Response {
	deliveries, err := notifications.Default().History(ctx.Request().InputInt("limit", 50))
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to load deliveries.")
	}

	return httpx.Ok(ctx, deliveries)
}

func (c *NotificationController) Store(ctx contractshttp.Context) contractshttp.Response {
	input, failure := notificationInput(ctx)
	if failure != nil {
		return failure
	}

	channel, err := notifications.Default().Create(input)
	if err != nil {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, err.Error())
	}

	return httpx.Created(ctx, notificationChannelResponse{
		NotificationChannel: *channel,
		Config:              notifications.Default().PublicConfig(channel),
		Subscribed:          notifications.Default().Events(channel),
	})
}

func (c *NotificationController) Update(ctx contractshttp.Context) contractshttp.Response {
	channel, err := notifications.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Channel not found.")
	}

	input, failure := notificationInput(ctx)
	if failure != nil {
		return failure
	}

	if err := notifications.Default().Update(channel, input); err != nil {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, err.Error())
	}

	return httpx.Ok(ctx, notificationChannelResponse{
		NotificationChannel: *channel,
		Config:              notifications.Default().PublicConfig(channel),
		Subscribed:          notifications.Default().Events(channel),
	})
}

func (c *NotificationController) Destroy(ctx contractshttp.Context) contractshttp.Response {
	channel, err := notifications.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Channel not found.")
	}

	if err := notifications.Default().Delete(channel); err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to delete the channel.")
	}

	return httpx.Ok(ctx, contractshttp.Json{"message": "Channel deleted."})
}

// Toggle enables or disables a channel.
func (c *NotificationController) Toggle(ctx contractshttp.Context) contractshttp.Response {
	channel, err := notifications.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Channel not found.")
	}

	enabled := ctx.Request().InputBool("enabled", !channel.Enabled)
	if err := notifications.Default().SetEnabled(channel, enabled); err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to update the channel.")
	}

	return httpx.Ok(ctx, contractshttp.Json{"enabled": enabled})
}

// Test sends a message through the channel.
func (c *NotificationController) Test(ctx contractshttp.Context) contractshttp.Response {
	channel, err := notifications.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Channel not found.")
	}

	if err := notifications.Default().Test(ctx.Request().Origin().Context(), channel); err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, err.Error())
	}

	return httpx.Ok(ctx, contractshttp.Json{"message": "Test notification sent to " + channel.Name + "."})
}

// Send dispatches a custom message to every subscribed channel.
func (c *NotificationController) Send(ctx contractshttp.Context) contractshttp.Response {
	title := strings.TrimSpace(ctx.Request().Input("title"))
	body := strings.TrimSpace(ctx.Request().Input("body"))

	if title == "" {
		return httpx.ValidationErrors(ctx, validationErrors{"title": {"A title is required."}})
	}

	notifications.Default().Notify("manual", title, body, nil)

	return httpx.Ok(ctx, contractshttp.Json{"message": "Notification queued."})
}

func notificationInput(ctx contractshttp.Context) (notifications.Input, contractshttp.Response) {
	input := notifications.Input{
		Name:    strings.TrimSpace(ctx.Request().Input("name")),
		Type:    ctx.Request().Input("type"),
		Enabled: ctx.Request().InputBool("enabled", true),
		Config:  map[string]string{},
		Events:  []string{},
	}

	for key, value := range ctx.Request().InputMap("config") {
		if text, ok := value.(string); ok {
			input.Config[key] = strings.TrimSpace(text)
		}
	}

	for _, event := range ctx.Request().InputArray("events") {
		if trimmed := strings.TrimSpace(event); trimmed != "" {
			input.Events = append(input.Events, trimmed)
		}
	}

	if input.Name == "" {
		return input, httpx.ValidationErrors(ctx, validationErrors{"name": {"The name is required."}})
	}

	if input.Type == "" {
		return input, httpx.ValidationErrors(ctx, validationErrors{"type": {"Select a channel type."}})
	}

	return input, nil
}
