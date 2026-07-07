package persist

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const stateVersion = 1

type GuildSnapshot struct {
	GuildID       string    `json:"guild_id"`
	Humans        int       `json:"humans"`
	Bots          int       `json:"bots"`
	HumansOnline  int       `json:"humans_online"`
	MembersLoaded bool      `json:"members_loaded"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type fileState struct {
	Version int                      `json:"version"`
	SavedAt time.Time                `json:"saved_at"`
	Guilds  map[string]GuildSnapshot `json:"guilds"`
}

type Store struct {
	path string
	mu   sync.RWMutex
	data fileState
}

func NewStore(dataDir string) (*Store, error) {
	if dataDir == "" {
		dataDir = "data"
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	s := &Store{
		path: filepath.Join(dataDir, "bot_state.json"),
		data: fileState{
			Version: stateVersion,
			Guilds:  make(map[string]GuildSnapshot),
		},
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read state: %w", err)
	}

	var loaded fileState
	if err := json.Unmarshal(raw, &loaded); err != nil {
		backup := s.path + ".corrupt-" + time.Now().UTC().Format("20060102-150405")
		if renameErr := os.Rename(s.path, backup); renameErr != nil {
			return fmt.Errorf("parse state: %w (could not backup corrupt file)", err)
		}
		slog.Warn("state file was corrupt — starting fresh", "err", err, "backup", backup)
		return nil
	}
	if loaded.Guilds == nil {
		loaded.Guilds = make(map[string]GuildSnapshot)
	}
	if loaded.Version == 0 {
		loaded.Version = stateVersion
	}

	s.mu.Lock()
	s.data = loaded
	s.mu.Unlock()
	return nil
}

func (s *Store) Save() error {
	s.mu.Lock()
	s.data.SavedAt = time.Now().UTC()
	payload, err := json.MarshalIndent(s.data, "", "  ")
	s.mu.Unlock()
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		return fmt.Errorf("write state temp: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("rename state: %w", err)
	}
	return nil
}

func (s *Store) UpsertGuild(snap GuildSnapshot) {
	if snap.GuildID == "" {
		return
	}
	snap.UpdatedAt = time.Now().UTC()

	s.mu.Lock()
	s.data.Guilds[snap.GuildID] = snap
	s.mu.Unlock()
}

func (s *Store) RemoveGuild(guildID string) {
	s.mu.Lock()
	delete(s.data.Guilds, guildID)
	s.mu.Unlock()
}

func (s *Store) Guild(guildID string) (GuildSnapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap, ok := s.data.Guilds[guildID]
	return snap, ok
}

func (s *Store) GlobalStats() (guilds, guildsReady, humansOnline int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	guilds = len(s.data.Guilds)
	for _, snap := range s.data.Guilds {
		if snap.MembersLoaded {
			guildsReady++
		}
		humansOnline += snap.HumansOnline
	}
	return guilds, guildsReady, humansOnline
}

func (s *Store) AllGuilds() map[string]GuildSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make(map[string]GuildSnapshot, len(s.data.Guilds))
	for id, snap := range s.data.Guilds {
		out[id] = snap
	}
	return out
}

func (s *Store) RunAutoSave(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			_ = s.Save()
			return
		case <-ticker.C:
			_ = s.Save()
		}
	}
}
