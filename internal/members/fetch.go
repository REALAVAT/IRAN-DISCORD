package members

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
	botToken   string
}

func NewClient(timeout time.Duration, botToken string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		botToken:   botToken,
	}
}

func (c *Client) FetchTotalMembers(ctx context.Context, guildID, inviteURL string) (*int, error) {
	if inviteURL != "" {
		if count, err := c.fetchInviteMembers(ctx, inviteURL); err == nil && count != nil {
			return count, nil
		}
	}

	if guildID != "" {
		return c.fetchGuildWithCounts(ctx, guildID)
	}

	return nil, fmt.Errorf("no invite or guild id for member total")
}

func (c *Client) WidgetEnabled(ctx context.Context, guildID string) bool {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("https://discord.com/api/guilds/%s/widget.json", guildID),
		nil,
	)
	if err != nil {
		return false
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer res.Body.Close()
	return res.StatusCode == http.StatusOK
}

func (c *Client) fetchInviteMembers(ctx context.Context, inviteURL string) (*int, error) {
	code := inviteCode(inviteURL)
	if code == "" {
		return nil, fmt.Errorf("invalid invite url")
	}

	endpoint := fmt.Sprintf(
		"https://discord.com/api/v10/invites/%s?with_counts=true",
		url.PathEscape(code),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invite api status %d", res.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	var payload struct {
		ApproximateMemberCount *int `json:"approximate_member_count"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload.ApproximateMemberCount, nil
}

func (c *Client) fetchGuildWithCounts(ctx context.Context, guildID string) (*int, error) {
	endpoint := fmt.Sprintf("https://discord.com/api/v10/guilds/%s?with_counts=true", guildID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if c.botToken != "" {
		req.Header.Set("Authorization", "Bot "+c.botToken)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("guild with_counts status %d", res.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	var payload struct {
		ApproximateMemberCount *int `json:"approximate_member_count"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload.ApproximateMemberCount, nil
}

func inviteCode(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if u, err := url.Parse(raw); err == nil && u.Path != "" {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		return parts[len(parts)-1]
	}
	return raw
}
