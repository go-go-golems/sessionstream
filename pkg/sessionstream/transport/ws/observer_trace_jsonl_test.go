package ws

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestObserverJSONLTraceSink(t *testing.T) {
	var modelBuffer, intervalBuffer bytes.Buffer
	sink, err := NewObserverJSONLTraceSink(&modelBuffer, &intervalBuffer)
	if err != nil {
		t.Fatal(err)
	}
	model := ObserverModelEvent{
		SchemaVersion: 1,
		RunID:         "run",
		DispatcherID:  "dispatcher",
		Sequence:      1,
		OperationID:   "op-1",
		Operation:     "submit",
		Action:        "submit_accepted",
		ItemID:        7,
		Updates:       map[string]any{"queue_len": 1},
	}
	interval := ObserverIntervalEvent{
		SchemaVersion: 1,
		RunID:         "run",
		DispatcherID:  "dispatcher",
		Sequence:      1,
		OperationID:   "op-1",
		Operation:     "submit",
		Phase:         "invoke",
	}
	sink.OnObserverModelEvent(model)
	sink.OnObserverIntervalEvent(interval)
	if err := sink.Err(); err != nil {
		t.Fatal(err)
	}
	var gotModel ObserverModelEvent
	if err := json.NewDecoder(&modelBuffer).Decode(&gotModel); err != nil {
		t.Fatal(err)
	}
	if gotModel.OperationID != model.OperationID || gotModel.Updates["queue_len"] != float64(1) {
		t.Fatalf("model JSONL = %+v", gotModel)
	}
	var gotInterval ObserverIntervalEvent
	if err := json.NewDecoder(&intervalBuffer).Decode(&gotInterval); err != nil {
		t.Fatal(err)
	}
	if gotInterval.OperationID != interval.OperationID || gotInterval.Phase != interval.Phase {
		t.Fatalf("interval JSONL = %+v", gotInterval)
	}
}

func TestObserverJSONLTraceSinkRejectsNilWriters(t *testing.T) {
	var buffer bytes.Buffer
	if _, err := NewObserverJSONLTraceSink(nil, &buffer); err == nil {
		t.Fatal("nil model writer accepted")
	}
	if _, err := NewObserverJSONLTraceSink(&buffer, nil); err == nil {
		t.Fatal("nil interval writer accepted")
	}
}
