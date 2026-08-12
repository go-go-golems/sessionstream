package ws

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// ObserverJSONLTraceSink writes model and interval streams independently. It
// serializes concurrent callbacks and flushes each JSON object through the
// supplied writers; caller-owned writers remain open.
type ObserverJSONLTraceSink struct {
	mu              sync.Mutex
	modelEncoder    *json.Encoder
	intervalEncoder *json.Encoder
	err             error
}

func NewObserverJSONLTraceSink(modelWriter, intervalWriter io.Writer) (*ObserverJSONLTraceSink, error) {
	if modelWriter == nil {
		return nil, fmt.Errorf("observer model trace writer is nil")
	}
	if intervalWriter == nil {
		return nil, fmt.Errorf("observer interval trace writer is nil")
	}
	return &ObserverJSONLTraceSink{
		modelEncoder:    json.NewEncoder(modelWriter),
		intervalEncoder: json.NewEncoder(intervalWriter),
	}, nil
}

func (s *ObserverJSONLTraceSink) OnObserverModelEvent(event ObserverModelEvent) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err == nil {
		s.err = s.modelEncoder.Encode(event)
	}
}

func (s *ObserverJSONLTraceSink) OnObserverIntervalEvent(event ObserverIntervalEvent) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err == nil {
		s.err = s.intervalEncoder.Encode(event)
	}
}

// Err returns the first encoding error observed by either stream.
func (s *ObserverJSONLTraceSink) Err() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}
