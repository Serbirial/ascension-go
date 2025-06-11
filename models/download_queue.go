package models

import (
	"fmt"
	"sync"
	"time"
)

// DownloadQueueRequest represents a download request for a guild
type DownloadQueueRequest struct {
	URL     string
	GuildID string
	Context *Context
	Done    chan bool
}

// DownloadQueue handles download requests per guild and manages queues
type DownloadQueue struct {
	SongQueues  map[string]*SongQueue
	Queue       []*DownloadQueueRequest
	mu          sync.Mutex
	startedOnce sync.Once
}

// Add queues a download request with the interaction context
func (dq *DownloadQueue) Add(ctx *Context, url string, guildID string) chan bool {
	done := make(chan bool, 1) // buffered to avoid blocking

	dq.mu.Lock()
	dq.Queue = append(dq.Queue, &DownloadQueueRequest{
		URL:     url,
		GuildID: guildID,
		Context: ctx,
		Done:    done,
	})
	dq.mu.Unlock()

	return done
}

// StartDownloader runs the background downloader thread
func (dq *DownloadQueue) StartDownloader() {
	dq.startedOnce.Do(func() {
		go dq.downloaderLoop()
	})
}

// GetOrCreateSongQueue gets or creates the queue for a guild
func (dq *DownloadQueue) GetOrCreateSongQueue(guildID string) *SongQueue {
	dq.mu.Lock()
	defer dq.mu.Unlock()

	sq, exists := dq.SongQueues[guildID]
	if !exists {
		sq = &SongQueue{
			Queue: []*SongInfo{},
		}
		dq.SongQueues[guildID] = sq
	}
	return sq
}

// downloaderLoop runs forever, processing download requests
func (dq *DownloadQueue) downloaderLoop() {
	for {
		dq.mu.Lock()
		if len(dq.Queue) == 0 {
			dq.mu.Unlock()
			time.Sleep(250 * time.Millisecond)
			continue
		}

		req := dq.Queue[0]
		dq.Queue = dq.Queue[1:]
		dq.mu.Unlock()

		ctx := req.Context

		var songInfo *SongInfo
		var err error

		if ctx.Client.DetachedDownloader {
			songInfo, err = ctx.Client.SendDownloadDetached(req.URL)
			if err != nil {
				fmt.Println("Detached download error:", err)
				ctx.Send("Download Server had an error while downloading.")
				req.Done <- false

				close(req.Done)

				continue
			}
		} else {
			songInfo, err = ctx.Client.SendDownloadToWS(req.URL, req.GuildID)
			if err != nil {
				fmt.Println("WS download error:", err)
				ctx.Send("Music Server had an error while downloading.")
				req.Done <- false

				close(req.Done)

				continue
			}
		}

		if songInfo != nil {
			sq := dq.GetOrCreateSongQueue(req.GuildID)
			sq.Add(songInfo)

			ctx.Send(fmt.Sprintf("Queued: **%s** by **%s**", songInfo.Title, songInfo.Uploader))
			// Send feedback to user who requested the download
			if req.Done != nil {
				req.Done <- true
				close(req.Done)
			}
			continue
		} else {
			if songInfo == nil {
				ctx.Send("Downloader returned `nil`, check logs for errors.")
				req.Done <- false

				continue
			}
		}
	}
}
