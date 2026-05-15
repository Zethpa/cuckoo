import type { GameArchive, GameSummary, RoomSettings, RoomSnapshot, User } from "../types/game";

const API_BASE = import.meta.env.VITE_API_BASE ?? "/api";

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(options.headers ?? {}),
    },
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.error ?? "Request failed");
  }
  return data as T;
}

export const api = {
  login: (username: string, password: string) =>
    request<{ user: User }>("/auth/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    }),
  logout: () => request<{ ok: boolean }>("/auth/logout", { method: "POST" }),
  me: () => request<{ user: User }>("/me"),
  listUsers: () => request<{ users: User[] }>("/admin/users"),
  createUser: (username: string, role: User["role"]) =>
    request<{ users: User[]; initialPassword: string }>("/admin/users", {
      method: "POST",
      body: JSON.stringify({ username, role }),
    }),
  disableUser: (id: number) => request<{ users: User[] }>(`/admin/users/${id}`, { method: "DELETE" }),
  restoreUser: (id: number) => request<{ users: User[] }>(`/admin/users/${id}/restore`, { method: "POST" }),
  resetPassword: (id: number) =>
    request<{ users: User[]; initialPassword: string }>(`/admin/users/${id}/reset-password`, { method: "POST" }),
  changePassword: (currentPassword: string, newPassword: string) =>
    request<{ ok: boolean }>("/me/password", {
      method: "POST",
      body: JSON.stringify({ currentPassword, newPassword }),
    }),
  createRoom: (settings: RoomSettings, password: string) =>
    request<RoomSnapshot>("/rooms", {
      method: "POST",
      body: JSON.stringify({ settings, password }),
    }),
  joinRoom: (code: string, password: string) =>
    request<RoomSnapshot>("/rooms/join", {
      method: "POST",
      body: JSON.stringify({ code, password }),
    }),
  room: (code: string) => request<RoomSnapshot>(`/rooms/${code}`),
  myGames: (limit = 10) => request<{ games: GameSummary[] }>(`/me/games?limit=${limit}`),
  gameArchive: (code: string) => request<{ game: GameArchive }>(`/games/${code}`),
  ready: (code: string, ready: boolean) =>
    request<RoomSnapshot>(`/rooms/${code}/ready`, {
      method: "POST",
      body: JSON.stringify({ ready }),
    }),
  settings: (code: string, settings: RoomSettings) =>
    request<RoomSnapshot>(`/rooms/${code}/settings`, {
      method: "PATCH",
      body: JSON.stringify(settings),
    }),
  startRoll: (code: string) => request<RoomSnapshot>(`/rooms/${code}/start-roll`, { method: "POST" }),
  roll: (code: string) => request<RoomSnapshot>(`/rooms/${code}/roll`, { method: "POST" }),
  startGame: (code: string) => request<RoomSnapshot>(`/rooms/${code}/start-game`, { method: "POST" }),
  contribute: (code: string, text: string) =>
    request<RoomSnapshot>(`/rooms/${code}/contributions`, {
      method: "POST",
      body: JSON.stringify({ text }),
    }),
};

export function wsURL(code: string): string {
  const base = import.meta.env.VITE_WS_BASE ?? `${location.protocol === "https:" ? "wss" : "ws"}://${location.host}/api`;
  return `${base}/ws/rooms/${code}`;
}
