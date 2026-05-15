import { FormEvent, useEffect, useState } from "react";
import { Link, Navigate } from "react-router-dom";
import { api } from "../api/client";
import { LanguageToggle } from "../components/LanguageToggle";
import { useAuth } from "../context/AuthContext";
import { useI18n } from "../context/I18nContext";
import type { User } from "../types/game";

export function AdminUsersPage() {
  const { user } = useAuth();
  const { t } = useI18n();
  const [users, setUsers] = useState<User[]>([]);
  const [username, setUsername] = useState("");
  const [role, setRole] = useState<User["role"]>("player");
  const [initialPassword, setInitialPassword] = useState("");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.listUsers()
      .then((res) => setUsers(res.users))
      .catch((err) => setError(err instanceof Error ? err.message : "Request failed"))
      .finally(() => setLoading(false));
  }, []);

  if (user?.role !== "admin") return <Navigate to="/" replace />;

  async function submit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setNotice("");
    try {
      const res = await api.createUser(username, role);
      setUsers(res.users);
      setUsername("");
      setRole("player");
      setInitialPassword(res.initialPassword);
      setNotice(t("admin.userCreated"));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Request failed");
    }
  }

  return (
    <main className="shell">
      <header className="topbar">
        <div><h1>{t("admin.title")}</h1><p>{user.username}</p></div>
        <div className="actions"><LanguageToggle /><Link className="button-link secondary" to="/">{t("common.back")}</Link></div>
      </header>
      {error && <p className="error">{error}</p>}
      {notice && <p className="notice">{notice}</p>}
      <section className="grid two">
        <form className="panel form" onSubmit={submit}>
          <h2>{t("admin.createUser")}</h2>
          <label>{t("auth.username")}<input value={username} onChange={(e) => setUsername(e.target.value)} /></label>
          <p className="hint">{t("admin.noPasswordInput")}</p>
          <label>{t("admin.role")}<select value={role} onChange={(e) => setRole(e.target.value as User["role"])}><option value="player">player</option><option value="admin">admin</option></select></label>
          <button>{t("action.createUser")}</button>
          {initialPassword && (
            <div className="generated-secret">
              <span>{t("admin.generatedPassword")}</span>
              <code>{initialPassword}</code>
              <small>{t("admin.generatedPasswordHelp")}</small>
            </div>
          )}
        </form>
        <section className="panel">
          <h2>{t("admin.users")}</h2>
          {loading && <p>{t("common.loading")}</p>}
          {!loading && users.length === 0 && <p>{t("admin.empty")}</p>}
          <ul className="users">
            {users.map((item) => <li key={item.id}><span>{item.username}</span><strong>{item.role}</strong></li>)}
          </ul>
        </section>
      </section>
    </main>
  );
}
