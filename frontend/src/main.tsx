import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { AuthProvider, useAuth } from "./context/AuthContext";
import { I18nProvider, useI18n } from "./context/I18nContext";
import { LoginPage } from "./pages/LoginPage";
import { HomePage } from "./pages/HomePage";
import { AdminUsersPage } from "./pages/AdminUsersPage";
import { AccountPage } from "./pages/AccountPage";
import { RoomPage } from "./pages/RoomPage";
import { GameArchivePage } from "./pages/GameArchivePage";
import "./styles/app.css";

function RequireAuth({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth();
  const { t } = useI18n();
  if (loading) return <main className="shell">{t("common.loading")}</main>;
  if (!user) return <Navigate to="/login" replace />;
  return children;
}

function App() {
  return (
    <I18nProvider>
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/" element={<RequireAuth><HomePage /></RequireAuth>} />
            <Route path="/account" element={<RequireAuth><AccountPage /></RequireAuth>} />
            <Route path="/admin/users" element={<RequireAuth><AdminUsersPage /></RequireAuth>} />
            <Route path="/rooms/:code" element={<RequireAuth><RoomPage /></RequireAuth>} />
            <Route path="/games/:code" element={<RequireAuth><GameArchivePage /></RequireAuth>} />
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </I18nProvider>
  );
}

ReactDOM.createRoot(document.getElementById("root")!).render(<App />);
