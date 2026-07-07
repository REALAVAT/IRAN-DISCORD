package membersync

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/iran-discord-services/stats/internal/guild"
)

type Queue struct {
	session *discordgo.Session
	cache   *guild.Cache
	delay   time.Duration

	mu      sync.Mutex
	pending []string
	queued  map[string]struct{}
	notify  chan struct{}
}

func NewQueue(session *discordgo.Session, cache *guild.Cache, delay time.Duration) *Queue {
	if delay <= 0 {
		delay = 5 * time.Second
	}
	return &Queue{
		session: session,
		cache:   cache,
		delay:   delay,
		queued:  make(map[string]struct{}),
		notify:  make(chan struct{}, 1),
	}
}

func (q *Queue) Enqueue(guildID string) {
	if guildID == "" {
		return
	}

	q.mu.Lock()
	if _, ok := q.queued[guildID]; ok {
		q.mu.Unlock()
		return
	}
	q.queued[guildID] = struct{}{}
	q.pending = append(q.pending, guildID)
	q.mu.Unlock()

	select {
	case q.notify <- struct{}{}:
	default:
	}
}

func (q *Queue) EnqueueAll(guildIDs []string) {
	for _, id := range guildIDs {
		q.Enqueue(id)
	}
}

func (q *Queue) Run(ctx context.Context) {
	for {
		guildID, ok := q.pop()
		if !ok {
			select {
			case <-ctx.Done():
				return
			case <-q.notify:
				continue
			}
		}

		q.request(guildID)

		select {
		case <-ctx.Done():
			return
		case <-time.After(q.delay):
		}
	}
}

func (q *Queue) pop() (string, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.pending) == 0 {
		return "", false
	}
	id := q.pending[0]
	q.pending = q.pending[1:]
	delete(q.queued, id)
	return id, true
}

func (q *Queue) request(guildID string) {
	if err := q.session.RequestGuildMembers(guildID, "", 0, "", true); err != nil {
		slog.Warn("member sync request failed", "guild", guildID, "err", err)
		return
	}
	slog.Debug("member sync requested", "guild", guildID, "with_presences", true)
}
