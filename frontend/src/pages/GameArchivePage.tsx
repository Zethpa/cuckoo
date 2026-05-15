import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { api } from "../api/client";
import { LanguageToggle } from "../components/LanguageToggle";
import { useI18n } from "../context/I18nContext";
import type { GameArchive } from "../types/game";

export function GameArchivePage() {
  const { code = "" } = useParams();
  const { t } = useI18n();
  const [game, setGame] = useState<GameArchive | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api.gameArchive(code)
      .then((res) => setGame(res.game))
      .catch((err) => setError(err instanceof Error ? err.message : "Request failed"));
  }, [code]);

  if (error) return <main className="shell"><p className="error">{error}</p></main>;
  if (!game) return <main className="shell">{t("common.loading")}</main>;

  return (
    <main className="shell">
      <header className="topbar">
        <div><h1>{t("game.title")} {game.roomCode}</h1><p>{game.theme}</p></div>
        <div className="actions"><LanguageToggle /><Link className="button-link secondary" to="/account">{t("common.back")}</Link></div>
      </header>
      <section className="grid two">
        <section className="panel story">
          <h2>{t("room.story")}</h2>
          <p className="opening">{game.openingSentence}</p>
          {game.contributions.map((c) => <article key={`${c.roundNumber}-${c.turnIndex}`} className={c.isSkipped ? "skipped" : ""}><strong>{c.username}</strong><p>{c.text}</p></article>)}
        </section>
        <section className="panel">
          <h2>{t("room.leaderboard")}</h2>
          <p>{t("game.finishedAt")}: {new Date(game.finishedAt).toLocaleString()}</p>
          {game.results.map((r) => <div className="score" key={r.userId}><span>{r.rank}. {r.username}</span><strong>{r.scoreTotal}</strong><small>{r.contributions} {t("room.scoreTurns")}</small></div>)}
        </section>
      </section>
    </main>
  );
}
