// respondToHeartbeat echoes the nonce from an application-level server ping.
// It returns true when frame was a heartbeat so callers may choose whether to
// render it alongside domain frames.
export function respondToHeartbeat(socket, frame) {
  if (!frame?.ping || socket?.readyState !== WebSocket.OPEN) return false;
  socket.send(JSON.stringify({ pong: { nonce: frame.ping.nonce || "" } }));
  return true;
}
