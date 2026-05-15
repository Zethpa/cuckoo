import { FormEvent, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api } from "../api/client";
import { LanguageToggle } from "../components/LanguageToggle";
import { useAuth } from "../context/AuthContext";
import { useI18n } from "../context/I18nContext";
import type { RoomSettings } from "../types/game";

const defaultSettings: RoomSettings = {
  theme: "A strange night at the train station",
  openingSentence: "The last train arrived with nobody inside except a singing suitcase.",
  maxUnitsPerTurn: 40,
  totalRounds: 3,
  diceOrder: "high_first",
};

export function HomePage() {
  const nav = useNavigate();
  const { user, logout } = useAuth();
  const { t } = useI18n();
  const [settings, setSettings] = useState(defaultSettings);
  const [roomPassword, setRoomPassword] = useState("");
  const [joinCode, setJoinCode] = useState("");
  const [joinPassword, setJoinPassword] = useState("");
  const [error, setError] = useState("");

  async function create(e: FormEvent) {
    e.preventDefault();
    setError("");
    try {
      const snap = await api.createRoom(settings, roomPassword);
      nav(`/rooms/${snap.room.code}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not create room");
    }
  }

  async function join(e: FormEvent) {
    e.preventDefault();
    setError("");
    try {
      const snap = await api.joinRoom(joinCode, joinPassword);
      nav(`/rooms/${snap.room.code}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not join room");
    }
  }

  return (
    <main className="shell">
      <header className="topbar">
        <div><h1>Cuckoo</h1><p>{user?.username}</p></div>
        <div className="actions">
          <LanguageToggle />
          <Link className="button-link secondary" to="/account">{t("home.account")}</Link>
          {user?.role === "admin" && <Link className="button-link secondary" to="/admin/users">{t("home.admin")}</Link>}
          <button className="secondary" onClick={logout}>{t("action.logout")}</button>
        </div>
      </header>
      {error && <p className="error">{error}</p>}
      <section className="grid two">
        <form className="panel form" onSubmit={create}>
          <h2>{t("home.createRoom")}</h2>
          <label>{t("home.theme")}<input value={settings.theme} maxLength={80} onChange={(e) => setSettings({ ...settings, theme: e.target.value })} /></label>
          <label>{t("home.opening")}<textarea value={settings.openingSentence} maxLength={300} onChange={(e) => setSettings({ ...settings, openingSentence: e.target.value })} /></label>
          <div className="row">
            <label>{t("home.maxUnits")}<input type="number" min={5} max={80} value={settings.maxUnitsPerTurn} onChange={(e) => setSettings({ ...settings, maxUnitsPerTurn: Number(e.target.value) })} /></label>
            <label>{t("home.rounds")}<input type="number" min={1} max={10} value={settings.totalRounds} onChange={(e) => setSettings({ ...settings, totalRounds: Number(e.target.value) })} /></label>
          </div>
          <label>{t("home.diceOrder")}<select value={settings.diceOrder} onChange={(e) => setSettings({ ...settings, diceOrder: e.target.value as RoomSettings["diceOrder"] })}><option value="high_first">{t("home.diceHighFirst")}</option><option value="low_first">{t("home.diceLowFirst")}</option></select></label>
          <label>{t("home.roomPassword")}<input value={roomPassword} onChange={(e) => setRoomPassword(e.target.value)} placeholder={t("common.optional")} /></label>
          <button>{t("action.create")}</button>
        </form>
        <form className="panel form" onSubmit={join}>
          <h2>{t("home.joinRoom")}</h2>
          <label>{t("home.roomCode")}<input value={joinCode} onChange={(e) => setJoinCode(e.target.value.toUpperCase())} /></label>
          <label>{t("auth.password")}<input value={joinPassword} onChange={(e) => setJoinPassword(e.target.value)} placeholder={t("home.passwordIfRequired")} /></label>
          <button>{t("action.join")}</button>
        </form>
      </section>
    </main>
  );
}
