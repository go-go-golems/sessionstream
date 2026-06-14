package sessionstream

import "github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"

func TypeScriptModule() *spec.Module {
	return &spec.Module{Name: ModuleName, RawDTS: []string{
		"export type Ordinal = string;",
		"export interface ProtoMessage<TTypeName extends string = string> { readonly typeName: TTypeName; toJSON(): unknown; }",
		"export interface MessageNamespace<TMessage extends ProtoMessage = ProtoMessage> { readonly typeName: string; builder(): unknown; }",
		"export interface SchemaMap { commands?: Record<string, string | MessageNamespace>; events?: Record<string, string | MessageNamespace>; uiEvents?: Record<string, string | MessageNamespace>; entities?: Record<string, string | MessageNamespace>; }",
		"export interface Schemas { registerCommand(name: string, schema: string | MessageNamespace): this; registerEvent(name: string, schema: string | MessageNamespace): this; registerUIEvent(name: string, schema: string | MessageNamespace): this; registerTimelineEntity(kind: string, schema: string | MessageNamespace): this; }",
		"export interface Command { name: string; sessionId: string; payload: unknown; }",
		"export interface Event { name: string; sessionId: string; ordinal: Ordinal; payload: unknown; }",
		"export interface Session { id: string; metadata?: unknown; }",
		"export interface Publisher { publish(name: string, payload: unknown): Promise<void>; }",
		"export interface UIEvent { name: string; payload: unknown; }",
		"export interface TimelineEntity { kind: string; id: string; createdOrdinal?: Ordinal; lastEventOrdinal?: Ordinal; payload: unknown; tombstone?: boolean; }",
		"export interface TimelineView { ordinal(): Ordinal; get(kind: string, id: string): TimelineEntity | null; list(kind: string): TimelineEntity[]; }",
		"export interface Snapshot { sessionId: string; snapshotOrdinal: Ordinal; entities: TimelineEntity[]; }",
		"export interface Fanout { close(): void; readonly id?: string; }",
		"export interface Hub { submit(sessionId: string, name: string, payload: unknown): Promise<void>; snapshot(sessionId: string): Snapshot; command(name: string, handler: (cmd: Command, session: Session, publisher: Publisher) => void | Promise<void>): this; uiProjection(handler: (event: Event, session: Session, view: TimelineView) => UIEvent[] | Promise<UIEvent[]>): this; timelineProjection(handler: (event: Event, session: Session, view: TimelineView) => TimelineEntity[] | Promise<TimelineEntity[]>): this; run(): void; shutdown(): void; }",
		"export interface HubOptions { schemas?: Schemas; fanout?: Fanout; projectionPolicy?: 'fail' | 'advance'; }",
		"export interface WebSocketServer { connections(): Array<{ connectionId: string; subscriptions: string[] }>; }",
		"export function schemas(input?: SchemaMap): Schemas;",
		"export function hub(options?: HubOptions): Hub;",
		"export function eventEmitterFanout(emitter: unknown): Fanout;",
		"export const fanout: { eventEmitter(emitter: unknown): Fanout };",
		"export const webSocket: { server(hub: Hub): WebSocketServer };",
		"export const version: string;",
	}}
}
