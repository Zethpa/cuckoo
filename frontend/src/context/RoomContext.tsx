import { createContext, useContext, useEffect, useMemo, useState } from "react";
import { api, wsURL } from "../api/client";
import type { RoomEvent, RoomSnapshot } from "../types/game";

type RoomState = {
  snapshot: RoomSnapshot | null;
  error: string | null;
  setSnapshot: (snapshot: RoomSnapshot) => void;
};

const RoomContext = createContext<RoomState | null>(null);

export function RoomProvider({ code, children }: { code: string; children: React.ReactNode }) {
  const [snapshot, setSnapshot] = useState<RoomSnapshot | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let closed = false;
    api.room(code).then(setSnapshot).catch((err) => setError(err.message));
    const ws = new WebSocket(wsURL(code));
    ws.onmessage = (msg) => {
      const event = JSON.parse(msg.data) as RoomEvent;
      if (!closed && event.payload?.room) setSnapshot(event.payload);
    };
    ws.onerror = () => setError("WebSocket connection failed");
    return () => {
      closed = true;
      ws.close();
    };
  }, [code]);

  const value = useMemo(() => ({ snapshot, error, setSnapshot }), [snapshot, error]);
  return <RoomContext.Provider value={value}>{children}</RoomContext.Provider>;
}

export function useRoom() {
  const ctx = useContext(RoomContext);
  if (!ctx) throw new Error("useRoom must be used inside RoomProvider");
  return ctx;
}
