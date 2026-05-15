import { FormEvent, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import { LanguageToggle } from "../components/LanguageToggle";
import { useAuth } from "../context/AuthContext";
import { useI18n } from "../context/I18nContext";

export function AccountPage() {
  const { user } = useAuth();
  const { t } = useI18n();
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  async function submit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setNotice("");
    try {
      await api.changePassword(currentPassword, newPassword);
      setCurrentPassword("");
      setNewPassword("");
      setNotice(t("account.passwordUpdated"));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Request failed");
    }
  }

  return (
    <main className="shell">
      <header className="topbar">
        <div><h1>{t("account.title")}</h1><p>{user?.username}</p></div>
        <div className="actions"><LanguageToggle /><Link className="button-link secondary" to="/">{t("common.back")}</Link></div>
      </header>
      {error && <p className="error">{error}</p>}
      {notice && <p className="notice">{notice}</p>}
      <section className="grid two">
        <form className="panel form" onSubmit={submit}>
          <h2>{t("account.changePassword")}</h2>
          <label>{t("account.currentPassword")}<input type="password" value={currentPassword} onChange={(e) => setCurrentPassword(e.target.value)} /></label>
          <label>{t("account.newPassword")}<input type="password" value={newPassword} minLength={8} placeholder={t("admin.passwordHint")} onChange={(e) => setNewPassword(e.target.value)} /></label>
          <button>{t("account.changePassword")}</button>
        </form>
      </section>
    </main>
  );
}
