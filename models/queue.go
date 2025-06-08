package models

import "sync"

type SongQueue struct {
	Queue []*SongInfo
	Loop  bool

	mu sync.Mutex
}

// Add appends a song to the playback queue
func (sq *SongQueue) Add(song *SongInfo) {
	sq.mu.Lock()
	defer sq.mu.Unlock()
	sq.Queue = append(sq.Queue, song)
}

// Current returns the currently playing song (index 0)
func (sq *SongQueue) Current() *SongInfo {
	sq.mu.Lock()
	defer sq.mu.Unlock()
	if len(sq.Queue) == 0 {
		return nil
	}
	return sq.Queue[0]
}

// Next removes the current song and returns the new current song
func (sq *SongQueue) Next() *SongInfo {
	sq.mu.Lock()
	defer sq.mu.Unlock()
	if len(sq.Queue) == 0 {
		return nil
	}
	if sq.Loop {
		return sq.Queue[0]
	}
	sq.Queue = sq.Queue[1:]
	if len(sq.Queue) > 0 {
		return sq.Queue[0]
	}
	return nil
}

// Remove deletes a song at a given index
func (sq *SongQueue) Remove(index int) {
	sq.mu.Lock()
	defer sq.mu.Unlock()
	if index >= 0 && index < len(sq.Queue) {
		sq.Queue = append(sq.Queue[:index], sq.Queue[index+1:]...)
	}
}
