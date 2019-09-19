package format

import (
	"sync"

	"github.com/studease/common/av"
)

// MediaStream is used as the base class for the creation of MediaStream objects
type MediaStream struct {
	mtx        sync.RWMutex
	trackIndex int
	tracks     []av.IMediaTrack
}

// Init this class
func (me *MediaStream) Init() *MediaStream {
	me.trackIndex = 0
	me.tracks = make([]av.IMediaTrack, 0)
	return me
}

// AddTrack creates a track with the codec given as argument
func (me *MediaStream) AddTrack(track av.IMediaTrack) {
	me.mtx.Lock()
	defer me.mtx.Unlock()

	track.SetID(me.trackIndex)

	me.trackIndex++
	me.tracks = append(me.tracks, track)
}

// RemoveTrack removes the track given as argument
func (me *MediaStream) RemoveTrack(track av.IMediaTrack) {
	me.mtx.Lock()
	defer me.mtx.Unlock()

	for i, t := range me.tracks {
		if t == track {
			me.tracks = append(me.tracks[:i], me.tracks[i+1:]...)
			track.Close()
			return
		}
	}
}

// AudioTrack returns the IMediaTrack objects found first that have their kind attribute set to "audio"
func (me *MediaStream) AudioTrack() av.IMediaTrack {
	me.mtx.RLock()
	defer me.mtx.RUnlock()

	for _, t := range me.tracks {
		if t.Kind() == av.KIND_AUDIO {
			return t
		}
	}

	return nil
}

// VideoTrack returns the IMediaTrack objects found first that have their kind attribute set to "video"
func (me *MediaStream) VideoTrack() av.IMediaTrack {
	me.mtx.RLock()
	defer me.mtx.RUnlock()

	for _, t := range me.tracks {
		if t.Kind() == av.KIND_VIDEO {
			return t
		}
	}

	return nil
}

// GetTracks returns a list of the IMediaTrack objects, regardless of the value of the kind attribute
func (me *MediaStream) GetTracks() []av.IMediaTrack {
	tracks := make([]av.IMediaTrack, 0)

	me.mtx.RLock()
	defer me.mtx.RUnlock()

	for _, t := range me.tracks {
		tracks = append(tracks, t)
	}

	return tracks
}

// GetTrackByID returns the track whose ID corresponds to the one given as argument
func (me *MediaStream) GetTrackByID(id int) av.IMediaTrack {
	me.mtx.RLock()
	defer me.mtx.RUnlock()

	for _, t := range me.tracks {
		if t.ID() == id {
			return t
		}
	}

	return nil
}

// Close closes all of the IMediaTrack objects stored in the MediaStream object
func (me *MediaStream) Close() error {
	me.mtx.Lock()
	defer me.mtx.Unlock()

	for _, t := range me.tracks {
		t.Close()
	}

	return nil
}
