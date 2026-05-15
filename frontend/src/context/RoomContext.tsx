import { createContext, useContext, useEffect, useMemo, useRef, useState } from "react";
import { api, wsURL } from "../api/client";
import type { DraftUpdate, RoomEvent, RoomSnapshot } from "../types/game";

type RoomState = {
  snapshot: RoomSnapshot | null;
  error: string | null;
  drafts: Record<number, string>;
  setSnapshot: (snapshot: RoomSnapshot) => void;
  sendDraft: (text: string) => void;
};

const RoomContext = createContext<RoomState | null>(null);

export function RoomProvider({ code, children }: { code: string; children: React.ReactNode }) {
  const [snapshot, setSnapshot] = useState<RoomSnapshot | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [drafts, setDrafts] = useState<Record<number, string>>({});
  const socketRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    let closed = false;
    api.room(code).then(setSnapshot).catch((err) => setError(err.message));
    const ws = new WebSocket(wsURL(code));
    socketRef.current = ws;
    ws.onmessage = (msg) => {
      const event = JSON.parse(msg.data) as RoomEvent;
      if (closed) return;
      if (event.type === "game.draft_updated") {
        const draft = event.payload as DraftUpdate;
        setDrafts((current) => ({ ...current, [draft.userId]: draft.text }));
        return;
      }
      if ("room" in event.payload && event.payload.room) {
        setSnapshot(event.payload);
        if (event.type === "game.contribution_added" || event.type === "game.turn_changed" || event.type === "game.turn_timeout") {
          setDrafts({});
        }
      }
    };
    ws.onerror = () => setError("WebSocket connection failed");
    return () => {
      closed = true;
      socketRef.current = null;
      ws.close();
    };
  }, [code]);

  const sendDraft = (text: string) => {
    const ws = socketRef.current;
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: "game.draft_update", payload: { text } }));
    }
  };

  const value = useMemo(() => ({ snapshot, error, drafts, setSnapshot, sendDraft }), [snapshot, error, drafts]);
  return <RoomContext.Provider value={value}>{children}</RoomContext.Provider>;
}

export function useRoom() {
  const ctx = useContext(RoomContext);
  if (!ctx) throw new Error("useRoom must be used inside RoomProvider");
  return ctx;
}
