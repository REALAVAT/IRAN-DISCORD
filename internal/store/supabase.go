package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type GuildMeta struct {
	GuildID   string `json:"discord_guild_id"`
	InviteURL string `json:"invite_url"`
}

type GuildStats struct {
	GuildID       string    `json:"guild_id"`
	MembersTotal  *int      `json:"members_total,omitempty"`
	HumansOnline  *int      `json:"humans_online,omitempty"`
	BotPresent    bool      `json:"bot_present"`
	SyncedAt      time.Time `json:"synced_at"`
	RealUsers     *int      `json:"real_users,omitempty"`
	BotsTotal     *int      `json:"bots_total,omitempty"`
	BotsOnline    *int      `json:"bots_online,omitempty"`
	ChannelsTotal *int      `json:"channels_total,omitempty"`
}

type Store struct {
	baseURL string
	key     string
	client  *http.Client
}

func New(baseURL, serviceRoleKey string) *Store {
	return &Store{
		baseURL: strings.TrimRight(baseURL, "/"),
		key:     serviceRoleKey,
		client:  &http.Client{Timeout: 20 * time.Second},
	}
}

func (s *Store) ListListedGuilds(ctx context.Context) (map[string]string, error) {
	endpoint := fmt.Sprintf(
		"%s/rest/v1/discord_servers?select=discord_guild_id,invite_url&is_active=eq.true&discord_guild_id=not.is.null",
		s.baseURL,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	s.applyHeaders(req)

	res, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("supabase list guilds status %d", res.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return nil, err
	}

	var rows []GuildMeta
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}

	out := make(map[string]string, len(rows))
	for _, row := range rows {
		id := strings.TrimSpace(row.GuildID)
		if id == "" {
			continue
		}
		out[id] = strings.TrimSpace(row.InviteURL)
	}
	return out, nil
}

func (s *Store) UpsertStats(ctx context.Context, stats GuildStats) error {
	payload, err := json.Marshal([]GuildStats{stats})
	if err != nil {
		return err
	}

	endpoint := s.baseURL + "/rest/v1/guild_stats_cache?on_conflict=guild_id"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	s.applyHeaders(req)
	req.Header.Set("Prefer", "resolution=merge-duplicates,return=minimal")

	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return fmt.Errorf("supabase upsert status %d: %s", res.StatusCode, strings.TrimSpace(string(msg)))
	}
	return nil
}

func (s *Store) applyHeaders(req *http.Request) {
	req.Header.Set("apikey", s.key)
	req.Header.Set("Authorization", "Bearer "+s.key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}

func EncodeInviteCode(inviteURL string) string {
	code := inviteURL
	if u, err := url.Parse(inviteURL); err == nil && u.Path != "" {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		code = parts[len(parts)-1]
	}
	return code
}
