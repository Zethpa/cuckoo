import { FormEvent, useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { api } from "../api/client";
import { LanguageToggle } from "../components/LanguageToggle";
import { RoomProvider, useRoom } from "../context/RoomContext";
import { useAuth } from "../context/AuthContext";
import { useI18n } from "../context/I18nContext";
import type { RoomSnapshot } from "../types/game";
import { countStoryUnits } from "../utils/wordCount";

export function RoomPage() {
  const { code = "" } = useParams();
  return <RoomProvider code={code}><RoomView code={code} /></RoomProvider>;
}

function RoomView({ code }: { code: string }) {
  const { user } = useAuth();
  const { t } = useI18n();
  const { snapshot, error, drafts, setSnapshot, sendDraft } = useRoom();
  const [text, setText] = useState("");
  const [actionError, setActionError] = useState("");
  const [now, setNow] = useState(Date.now());

  useEffect(() => {
    const interval = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(interval);
  }, []);
  if (error) return <main className="shell"><p className="error">{error}</p></main>;
  if (!snapshot) return <main className="shell">{t("room.loading")}</main>;

  const { room, contributions, currentPlayer, nextPlayer, results } = snapshot;
  const me = room.players.find((p) => p.userId === user?.id);
  const isHost = room.hostUserId === user?.id;
  const isTurn = currentPlayer?.userId === user?.id;
  const units = countStoryUnits(text);
  const timeLeft = room.turnStartedAt
    ? Math.max(0, room.settings.turnTimeLimitSeconds - Math.floor((now - new Date(room.turnStartedAt).getTime()) / 1000))
    : room.settings.turnTimeLimitSeconds;

  async function run(fn: () => Promise<RoomSnapshot>) {
    setActionError("");
    try {
      setSnapshot(await fn());
    } catch (err) {
      setActionError(err instanceof Error ? err.message : t("room.actionFailed"));
    }
  }

  async function submit(e: FormEvent) {
    e.preventDefault();
    await run(() => api.contribute(code, text));
    sendDraft("");
    setText("");
  }

  function updateText(value: string) {
    setText(value);
    if (isTurn) sendDraft(value);
  }

  return (
    <main className="shell">
      <header className="topbar">
        <div><h1>Room {room.code}</h1><p>{room.settings.theme}</p></div>
        <div className="actions"><LanguageToggle /><span className={`badge ${room.status}`}>{t(`common.status.${room.status}`)}</span></div>
      </header>
      {actionError && <p className="error">{actionError}</p>}
      <section className="grid layout">
        <aside className="panel">
          <h2>{t("room.players")}</h2>
          <ul className="players">
            {room.players.map((p) => (
              <li key={p.id}>
                <span>{p.orderIndex !== null ? `${p.orderIndex + 1}. ` : ""}{p.user.username}</span>
                <span>{p.roll ? `${t("room.d100")}: ${p.roll}` : p.ready ? t("room.ready") : t("room.waiting")}</span>
              </li>
            ))}
          </ul>
          {room.status === "lobby" && me && !isHost && <button onClick={() => run(() => api.ready(code, !me.ready))}>{me.ready ? t("action.unready") : t("action.ready")}</button>}
          {room.status === "lobby" && isHost && <button onClick={() => run(() => api.startRoll(code))}>{t("action.startDice")}</button>}
          {room.status === "rolling" && me?.roll === null && <button onClick={() => run(() => api.roll(code))}>{t("action.roll")}</button>}
          {room.status === "rolling" && isHost && room.players.every((p) => p.orderIndex !== null) && <button onClick={() => run(() => api.startGame(code))}>{t("action.startGame")}</button>}
        </aside>
        <section className="panel story">
          <h2>{t("room.story")}</h2>
          <p className="opening">{room.settings.openingSentence}</p>
          {contributions.map((c) => <article className={c.isSkipped ? "skipped" : ""} key={c.id}><strong>{c.user.username}</strong><p>{c.isSkipped ? t("room.waitingTurn") : c.text}</p></article>)}
          {room.status === "active" && currentPlayer && drafts[currentPlayer.userId] && currentPlayer.userId !== user?.id && (
            <article className="draft-preview"><strong>{currentPlayer.user.username}</strong><p>{drafts[currentPlayer.userId]}<span className="cursor" /></p></article>
          )}
          {room.status === "active" && <div className="turn">{t("room.current")}: {currentPlayer?.user.username} · {t("room.next")}: {nextPlayer?.user.username} · {t("room.timeLeft")}: {timeLeft}s</div>}
          {room.status === "active" && (
            <form onSubmit={submit} className="composer">
              <textarea disabled={!isTurn || timeLeft === 0} value={text} onChange={(e) => updateText(e.target.value)} placeholder={isTurn ? t("room.writePlaceholder") : t("room.waitingTurn")} />
              <div className="composer-bar">
                <span className={units > room.settings.maxUnitsPerTurn ? "over" : ""}>{units}/{room.settings.maxUnitsPerTurn} · {t("room.unitsHelp")}</span>
                <button disabled={!isTurn || timeLeft === 0 || units === 0 || units > room.settings.maxUnitsPerTurn}>{t("action.submit")}</button>
              </div>
            </form>
          )}
          {room.status === "finished" && <Leaderboard results={results} />}
        </section>
      </section>
    </main>
  );
}

function Leaderboard({ results }: { results: Array<{ id: number; user: { username: string }; scoreTotal: number; contributions: number }> }) {
  const { t } = useI18n();
  return (
    <section className="leaderboard">
      <h2>{t("room.leaderboard")}</h2>
      {results.map((r, i) => <div className="score" key={r.id}><span>{i + 1}. {r.user.username}</span><strong>{r.scoreTotal}</strong><small>{r.contributions} {t("room.scoreTurns")}</small></div>)}
    </section>
  );
}
