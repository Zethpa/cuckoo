import { FormEvent, useState } from "react";
import { Navigate } from "react-router-dom";
import { LanguageToggle } from "../components/LanguageToggle";
import { useAuth } from "../context/AuthContext";
import { useI18n } from "../context/I18nContext";

export function LoginPage() {
  const { user, login } = useAuth();
  const { t } = useI18n();
  const [username, setUsername] = useState("admin");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  if (user) return <Navigate to="/" replace />;

  async function submit(e: FormEvent) {
    e.preventDefault();
    setError("");
    try {
      await login(username, password);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("auth.loginFailed"));
    }
  }

  return (
    <main className="auth-page">
      <div className="floating-action"><LanguageToggle /></div>
      <form className="panel auth-card" onSubmit={submit}>
        <h1>Cuckoo</h1>
        <label>{t("auth.username")}<input value={username} onChange={(e) => setUsername(e.target.value)} /></label>
        <label>{t("auth.password")}<input type="password" value={password} onChange={(e) => setPassword(e.target.value)} /></label>
        {error && <p className="error">{error}</p>}
        <button type="submit">{t("action.login")}</button>
      </form>
    </main>
  );
}
