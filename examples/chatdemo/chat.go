package chatdemo

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	chatdemov1 "github.com/go-go-golems/sessionstream/examples/chatdemo/gen/sessionstream/examples/chatdemo/v1"
	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	"google.golang.org/protobuf/proto"
)

const (
	CommandStartInference = "ChatStartInference"
	CommandStopInference  = "ChatStopInference"

	EventUserMessageAccepted = "ChatUserMessageAccepted"
	EventInferenceStarted    = "ChatInferenceStarted"
	EventTokensDelta         = "ChatTokensDelta"
	EventInferenceFinished   = "ChatInferenceFinished"
	EventInferenceStopped    = "ChatInferenceStopped"

	UIMessageAccepted = "ChatMessageAccepted"
	UIMessageStarted  = "ChatMessageStarted"
	UIMessageAppended = "ChatMessageAppended"
	UIMessageFinished = "ChatMessageFinished"
	UIMessageStopped  = "ChatMessageStopped"

	TimelineEntityChatMessage = "ChatMessage"
)

type Hooks struct {
	OnBackendEvent func(sessionID, eventName string, payload map[string]any)
}

type Option func(*Engine)

type Engine struct {
	mu         sync.Mutex
	nextID     int
	active     map[sessionstream.SessionId]*activeRun
	chunkDelay time.Duration
	hooks      Hooks
}

type activeRun struct {
	messageID string
	cancel    context.CancelFunc
	done      chan struct{}
}

type Service struct {
	hub    *sessionstream.Hub
	engine *Engine
}

func WithChunkDelay(delay time.Duration) Option {
	return func(e *Engine) {
		e.chunkDelay = delay
	}
}

func WithHooks(h Hooks) Option {
	return func(e *Engine) {
		e.hooks = h
	}
}

func NewEngine(opts ...Option) *Engine {
	engine := &Engine{
		active:     map[sessionstream.SessionId]*activeRun{},
		chunkDelay: 20 * time.Millisecond,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(engine)
		}
	}
	return engine
}

func RegisterSchemas(reg *sessionstream.SchemaRegistry) error {
	for _, err := range []error{
		reg.RegisterCommand(CommandStartInference, &chatdemov1.StartInferenceCommand{}),
		reg.RegisterCommand(CommandStopInference, &chatdemov1.StopInferenceCommand{}),
		reg.RegisterEvent(EventUserMessageAccepted, &chatdemov1.UserMessageAcceptedEvent{}),
		reg.RegisterEvent(EventInferenceStarted, &chatdemov1.InferenceStartedEvent{}),
		reg.RegisterEvent(EventTokensDelta, &chatdemov1.TokensDeltaEvent{}),
		reg.RegisterEvent(EventInferenceFinished, &chatdemov1.InferenceFinishedEvent{}),
		reg.RegisterEvent(EventInferenceStopped, &chatdemov1.InferenceStoppedEvent{}),
		reg.RegisterUIEvent(UIMessageAccepted, &chatdemov1.ChatMessageUpdate{}),
		reg.RegisterUIEvent(UIMessageStarted, &chatdemov1.ChatMessageUpdate{}),
		reg.RegisterUIEvent(UIMessageAppended, &chatdemov1.ChatMessageUpdate{}),
		reg.RegisterUIEvent(UIMessageFinished, &chatdemov1.ChatMessageUpdate{}),
		reg.RegisterUIEvent(UIMessageStopped, &chatdemov1.ChatMessageUpdate{}),
		reg.RegisterTimelineEntity(TimelineEntityChatMessage, &chatdemov1.ChatMessageEntity{}),
	} {
		if err != nil {
			return err
		}
	}
	return nil
}

func Install(hub *sessionstream.Hub, engine *Engine) error {
	if hub == nil {
		return fmt.Errorf("hub is nil")
	}
	if engine == nil {
		engine = NewEngine()
	}
	if err := hub.RegisterCommand(CommandStartInference, engine.handleStartInference); err != nil {
		return err
	}
	if err := hub.RegisterCommand(CommandStopInference, engine.handleStopInference); err != nil {
		return err
	}
	if err := hub.RegisterUIProjection(sessionstream.UIProjectionFunc(uiProjection)); err != nil {
		return err
	}
	if err := hub.RegisterTimelineProjection(sessionstream.TimelineProjectionFunc(timelineProjection)); err != nil {
		return err
	}
	return nil
}

func NewService(hub *sessionstream.Hub, engine *Engine) (*Service, error) {
	if hub == nil {
		return nil, fmt.Errorf("hub is nil")
	}
	if engine == nil {
		engine = NewEngine()
	}
	return &Service{hub: hub, engine: engine}, nil
}

func (s *Service) SubmitPrompt(ctx context.Context, sid sessionstream.SessionId, prompt string) error {
	if s == nil || s.hub == nil {
		return fmt.Errorf("chat demo service is not initialized")
	}
	prompt = strings.TrimSpace(prompt)
	if sid == "" {
		return fmt.Errorf("session id is empty")
	}
	if prompt == "" {
		return fmt.Errorf("prompt is empty")
	}
	return s.hub.Submit(ctx, sid, CommandStartInference, &chatdemov1.StartInferenceCommand{Prompt: prompt})
}

func (s *Service) Stop(ctx context.Context, sid sessionstream.SessionId) error {
	if s == nil || s.hub == nil {
		return fmt.Errorf("chat demo service is not initialized")
	}
	if sid == "" {
		return fmt.Errorf("session id is empty")
	}
	return s.hub.Submit(ctx, sid, CommandStopInference, &chatdemov1.StopInferenceCommand{})
}

func (s *Service) WaitIdle(ctx context.Context, sid sessionstream.SessionId) error {
	if s == nil || s.engine == nil {
		return fmt.Errorf("chat demo engine is not initialized")
	}
	return s.engine.WaitIdle(ctx, sid)
}

func (s *Service) Snapshot(ctx context.Context, sid sessionstream.SessionId) (sessionstream.Snapshot, error) {
	if s == nil || s.hub == nil {
		return sessionstream.Snapshot{}, fmt.Errorf("chat demo service is not initialized")
	}
	return s.hub.Snapshot(ctx, sid)
}

func (e *Engine) handleStartInference(ctx context.Context, cmd sessionstream.Command, _ *sessionstream.Session, pub sessionstream.EventPublisher) error {
	payload, ok := cmd.Payload.(*chatdemov1.StartInferenceCommand)
	if !ok {
		return fmt.Errorf("unexpected start inference payload %T", cmd.Payload)
	}
	prompt := strings.TrimSpace(payload.GetPrompt())
	if prompt == "" {
		prompt = "Explain sessionstream"
	}
	messageID := e.nextMessageID()
	userMessageID := messageID + "-user"
	userEvent := &chatdemov1.UserMessageAcceptedEvent{MessageId: userMessageID, Role: "user", Content: prompt, Streaming: false}
	if err := e.publish(ctx, cmd.SessionId, pub, EventUserMessageAccepted, userEvent, map[string]any{
		"messageId": userMessageID,
		"role":      "user",
		"content":   prompt,
		"streaming": false,
	}); err != nil {
		return err
	}
	runCtx, cancel := context.WithCancel(context.Background())
	run := &activeRun{messageID: messageID, cancel: cancel, done: make(chan struct{})}
	if previous := e.swapRun(cmd.SessionId, run); previous != nil {
		previous.cancel()
		<-previous.done
	}
	go e.runDemoInference(runCtx, cmd.SessionId, messageID, prompt, pub, run.done)
	return nil
}

func (e *Engine) handleStopInference(_ context.Context, cmd sessionstream.Command, _ *sessionstream.Session, _ sessionstream.EventPublisher) error {
	if current := e.currentRun(cmd.SessionId); current != nil {
		current.cancel()
	}
	return nil
}

func (e *Engine) runDemoInference(ctx context.Context, sid sessionstream.SessionId, messageID, prompt string, pub sessionstream.EventPublisher, done chan struct{}) {
	defer close(done)
	defer e.clearRun(sid, messageID)

	started := &chatdemov1.InferenceStartedEvent{MessageId: messageID, Prompt: prompt, Role: "assistant", Content: "", Status: "streaming", Streaming: true}
	if err := e.publish(ctx, sid, pub, EventInferenceStarted, started, map[string]any{"messageId": messageID, "prompt": prompt, "role": "assistant", "content": "", "status": "streaming", "streaming": true}); err != nil {
		return
	}

	answer := renderAnswer(prompt)
	chunks := chunkText(answer, 10)
	accumulated := ""
	for _, chunk := range chunks {
		select {
		case <-ctx.Done():
			stopped := &chatdemov1.InferenceStoppedEvent{MessageId: messageID, Role: "assistant", Text: accumulated, Content: accumulated, Status: "stopped", Streaming: false}
			_ = e.publish(context.Background(), sid, pub, EventInferenceStopped, stopped, map[string]any{"messageId": messageID, "role": "assistant", "text": accumulated, "content": accumulated, "status": "stopped", "streaming": false})
			return
		case <-time.After(e.chunkDelay):
		}
		accumulated += chunk
		delta := &chatdemov1.TokensDeltaEvent{MessageId: messageID, Role: "assistant", Chunk: chunk, Text: accumulated, Content: accumulated, Status: "streaming", Streaming: true}
		if err := e.publish(context.Background(), sid, pub, EventTokensDelta, delta, map[string]any{"messageId": messageID, "role": "assistant", "chunk": chunk, "text": accumulated, "content": accumulated, "status": "streaming", "streaming": true}); err != nil {
			return
		}
	}
	finished := &chatdemov1.InferenceFinishedEvent{MessageId: messageID, Role: "assistant", Text: accumulated, Content: accumulated, Status: "finished", Streaming: false}
	_ = e.publish(context.Background(), sid, pub, EventInferenceFinished, finished, map[string]any{"messageId": messageID, "role": "assistant", "text": accumulated, "content": accumulated, "status": "finished", "streaming": false})
}

func (e *Engine) publish(ctx context.Context, sid sessionstream.SessionId, pub sessionstream.EventPublisher, name string, payload proto.Message, hookPayload map[string]any) error {
	if e.hooks.OnBackendEvent != nil {
		e.hooks.OnBackendEvent(string(sid), name, cloneMap(hookPayload))
	}
	return pub.Publish(ctx, sessionstream.Event{Name: name, SessionId: sid, Payload: payload})
}

func (e *Engine) WaitIdle(ctx context.Context, sid sessionstream.SessionId) error {
	for {
		run := e.currentRun(sid)
		if run == nil {
			return nil
		}
		select {
		case <-run.done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (e *Engine) nextMessageID() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.nextID++
	return fmt.Sprintf("chat-msg-%d", e.nextID)
}

func (e *Engine) swapRun(sid sessionstream.SessionId, run *activeRun) *activeRun {
	e.mu.Lock()
	defer e.mu.Unlock()
	prev := e.active[sid]
	e.active[sid] = run
	return prev
}

func (e *Engine) currentRun(sid sessionstream.SessionId) *activeRun {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.active[sid]
}

func (e *Engine) clearRun(sid sessionstream.SessionId, messageID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	current := e.active[sid]
	if current != nil && current.messageID == messageID {
		delete(e.active, sid)
	}
}

func uiProjection(_ context.Context, ev sessionstream.Event, _ *sessionstream.Session, _ sessionstream.TimelineView) ([]sessionstream.UIEvent, error) {
	update, ok := chatUpdateFromEvent(ev)
	if !ok {
		return nil, nil
	}
	update.Ordinal = fmt.Sprintf("%d", ev.Ordinal)
	switch ev.Name {
	case EventUserMessageAccepted:
		return []sessionstream.UIEvent{{Name: UIMessageAccepted, Payload: update}}, nil
	case EventInferenceStarted:
		return []sessionstream.UIEvent{{Name: UIMessageStarted, Payload: update}}, nil
	case EventTokensDelta:
		return []sessionstream.UIEvent{{Name: UIMessageAppended, Payload: update}}, nil
	case EventInferenceFinished:
		return []sessionstream.UIEvent{{Name: UIMessageFinished, Payload: update}}, nil
	case EventInferenceStopped:
		return []sessionstream.UIEvent{{Name: UIMessageStopped, Payload: update}}, nil
	default:
		return nil, nil
	}
}

func timelineProjection(_ context.Context, ev sessionstream.Event, _ *sessionstream.Session, view sessionstream.TimelineView) ([]sessionstream.TimelineEntity, error) {
	messageID := eventMessageID(ev.Payload)
	if messageID == "" {
		return nil, nil
	}
	entity := currentChatEntity(view, TimelineEntityChatMessage, messageID)
	entity.MessageId = messageID
	switch payload := ev.Payload.(type) {
	case *chatdemov1.UserMessageAcceptedEvent:
		entity.Role = "user"
		entity.Content = payload.GetContent()
		entity.Text = payload.GetContent()
		entity.Streaming = false
	case *chatdemov1.InferenceStartedEvent:
		entity.Prompt = payload.GetPrompt()
		entity.Role = "assistant"
		entity.Content = ""
		entity.Text = ""
		entity.Status = "streaming"
		entity.Streaming = true
	case *chatdemov1.TokensDeltaEvent:
		entity.Role = "assistant"
		entity.Content = payload.GetContent()
		entity.Text = payload.GetText()
		entity.Status = "streaming"
		entity.Streaming = true
	case *chatdemov1.InferenceFinishedEvent:
		entity.Role = "assistant"
		entity.Content = payload.GetContent()
		entity.Text = payload.GetText()
		entity.Status = "finished"
		entity.Streaming = false
	case *chatdemov1.InferenceStoppedEvent:
		entity.Role = "assistant"
		entity.Content = payload.GetContent()
		entity.Text = payload.GetText()
		entity.Status = "stopped"
		entity.Streaming = false
	default:
		return nil, nil
	}
	return []sessionstream.TimelineEntity{{Kind: TimelineEntityChatMessage, Id: messageID, Payload: entity}}, nil
}

func chatUpdateFromEvent(ev sessionstream.Event) (*chatdemov1.ChatMessageUpdate, bool) {
	switch payload := ev.Payload.(type) {
	case *chatdemov1.UserMessageAcceptedEvent:
		return &chatdemov1.ChatMessageUpdate{MessageId: payload.GetMessageId(), Role: payload.GetRole(), Content: payload.GetContent(), Text: payload.GetContent(), Streaming: payload.GetStreaming()}, true
	case *chatdemov1.InferenceStartedEvent:
		return &chatdemov1.ChatMessageUpdate{MessageId: payload.GetMessageId(), Role: payload.GetRole(), Prompt: payload.GetPrompt(), Content: payload.GetContent(), Status: payload.GetStatus(), Streaming: payload.GetStreaming()}, true
	case *chatdemov1.TokensDeltaEvent:
		return &chatdemov1.ChatMessageUpdate{MessageId: payload.GetMessageId(), Role: payload.GetRole(), Chunk: payload.GetChunk(), Text: payload.GetText(), Content: payload.GetContent(), Status: payload.GetStatus(), Streaming: payload.GetStreaming()}, true
	case *chatdemov1.InferenceFinishedEvent:
		return &chatdemov1.ChatMessageUpdate{MessageId: payload.GetMessageId(), Role: payload.GetRole(), Text: payload.GetText(), Content: payload.GetContent(), Status: payload.GetStatus(), Streaming: payload.GetStreaming()}, true
	case *chatdemov1.InferenceStoppedEvent:
		return &chatdemov1.ChatMessageUpdate{MessageId: payload.GetMessageId(), Role: payload.GetRole(), Text: payload.GetText(), Content: payload.GetContent(), Status: payload.GetStatus(), Streaming: payload.GetStreaming()}, true
	default:
		return nil, false
	}
}

func eventMessageID(msg proto.Message) string {
	switch payload := msg.(type) {
	case *chatdemov1.UserMessageAcceptedEvent:
		return payload.GetMessageId()
	case *chatdemov1.InferenceStartedEvent:
		return payload.GetMessageId()
	case *chatdemov1.TokensDeltaEvent:
		return payload.GetMessageId()
	case *chatdemov1.InferenceFinishedEvent:
		return payload.GetMessageId()
	case *chatdemov1.InferenceStoppedEvent:
		return payload.GetMessageId()
	default:
		return ""
	}
}

func currentChatEntity(view sessionstream.TimelineView, kind, id string) *chatdemov1.ChatMessageEntity {
	entity, ok := view.Get(kind, id)
	if !ok || entity.Payload == nil {
		return &chatdemov1.ChatMessageEntity{}
	}
	if pb, ok := entity.Payload.(*chatdemov1.ChatMessageEntity); ok {
		return proto.Clone(pb).(*chatdemov1.ChatMessageEntity)
	}
	return &chatdemov1.ChatMessageEntity{}
}

func renderAnswer(prompt string) string {
	return "Answer: " + prompt
}

func chunkText(text string, size int) []string {
	if size <= 0 || len(text) <= size {
		return []string{text}
	}
	out := make([]string, 0, (len(text)+size-1)/size)
	for len(text) > 0 {
		if len(text) <= size {
			out = append(out, text)
			break
		}
		out = append(out, text[:size])
		text = text[size:]
	}
	return out
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
