import { createContext, useContext, useEffect, useMemo, useState } from "react";

type Language = "en" | "zh";
type TranslationKey =
  | "action.create"
  | "action.createUser"
  | "action.join"
  | "action.login"
  | "action.logout"
  | "action.ready"
  | "action.roll"
  | "action.startDice"
  | "action.startGame"
  | "action.submit"
  | "action.unready"
  | "account.changePassword"
  | "account.currentPassword"
  | "account.newPassword"
  | "account.passwordUpdated"
  | "account.title"
  | "auth.loginFailed"
  | "auth.password"
  | "auth.username"
  | "admin.createUser"
  | "admin.empty"
  | "admin.passwordHint"
  | "admin.generatedPassword"
  | "admin.generatedPasswordHelp"
  | "admin.noPasswordInput"
  | "admin.role"
  | "admin.title"
  | "admin.users"
  | "admin.userCreated"
  | "admin.disable"
  | "admin.restore"
  | "admin.resetPassword"
  | "admin.disabled"
  | "admin.active"
  | "admin.passwordReset"
  | "account.recentGames"
  | "account.noGames"
  | "account.rank"
  | "game.title"
  | "game.finishedAt"
  | "common.loading"
  | "common.back"
  | "common.optional"
  | "common.status.active"
  | "common.status.finished"
  | "common.status.lobby"
  | "common.status.rolling"
  | "home.createRoom"
  | "home.admin"
  | "home.account"
  | "home.diceHighFirst"
  | "home.diceLowFirst"
  | "home.diceOrder"
  | "home.joinRoom"
  | "home.maxUnits"
  | "home.opening"
  | "home.passwordIfRequired"
  | "home.roomCode"
  | "home.roomPassword"
  | "home.rounds"
  | "home.theme"
  | "home.turnTimeLimit"
  | "language.toggle"
  | "room.actionFailed"
  | "room.current"
  | "room.d100"
  | "room.leaderboard"
  | "room.loading"
  | "room.next"
  | "room.players"
  | "room.ready"
  | "room.scoreTurns"
  | "room.story"
  | "room.waiting"
  | "room.waitingTurn"
  | "room.unitsHelp"
  | "room.timeLeft"
  | "room.writePlaceholder";

const translations: Record<Language, Record<TranslationKey, string>> = {
  en: {
    "action.create": "Create",
    "action.createUser": "Create user",
    "action.join": "Join",
    "action.login": "Login",
    "action.logout": "Logout",
    "action.ready": "Ready",
    "action.roll": "Roll d100",
    "action.startDice": "Start Dice",
    "action.startGame": "Start Game",
    "action.submit": "Submit",
    "action.unready": "Unready",
    "account.changePassword": "Change password",
    "account.currentPassword": "Current password",
    "account.newPassword": "New password",
    "account.passwordUpdated": "Password updated",
    "account.title": "Account",
    "auth.loginFailed": "Login failed",
    "auth.password": "Password",
    "auth.username": "Username",
    "admin.createUser": "Create User",
    "admin.empty": "No users yet",
    "admin.passwordHint": "At least 8 characters",
    "admin.generatedPassword": "Initial password",
    "admin.generatedPasswordHelp": "Share this password once. It is not stored in plaintext.",
    "admin.noPasswordInput": "Initial password is generated automatically from the username and server secret.",
    "admin.role": "Role",
    "admin.title": "Developer Admin",
    "admin.users": "Users",
    "admin.userCreated": "User created",
    "admin.disable": "Disable",
    "admin.restore": "Restore",
    "admin.resetPassword": "Reset password",
    "admin.disabled": "disabled",
    "admin.active": "active",
    "admin.passwordReset": "Password reset",
    "account.recentGames": "Recent games",
    "account.noGames": "No finished games yet",
    "account.rank": "Rank",
    "game.title": "Game Archive",
    "game.finishedAt": "Finished",
    "common.back": "Back",
    "common.loading": "Loading...",
    "common.optional": "Optional",
    "common.status.active": "active",
    "common.status.finished": "finished",
    "common.status.lobby": "lobby",
    "common.status.rolling": "rolling",
    "home.createRoom": "Create Room",
    "home.admin": "Admin",
    "home.account": "Account",
    "home.diceHighFirst": "High roll first",
    "home.diceLowFirst": "Low roll first",
    "home.diceOrder": "Dice order",
    "home.joinRoom": "Join Room",
    "home.maxUnits": "Max units",
    "home.opening": "Opening",
    "home.passwordIfRequired": "If required",
    "home.roomCode": "Room code",
    "home.roomPassword": "Room password",
    "home.rounds": "Rounds",
    "home.theme": "Theme",
    "home.turnTimeLimit": "Turn time limit (seconds)",
    "language.toggle": "中文",
    "room.actionFailed": "Action failed",
    "room.current": "Current",
    "room.d100": "d100",
    "room.leaderboard": "Leaderboard",
    "room.loading": "Loading room...",
    "room.next": "Next",
    "room.players": "Players",
    "room.ready": "ready",
    "room.scoreTurns": "turns",
    "room.story": "Story",
    "room.waiting": "waiting",
    "room.waitingTurn": "Waiting for your turn",
    "room.unitsHelp": "CJK characters count one each; English words and number runs count one each.",
    "room.timeLeft": "Time left",
    "room.writePlaceholder": "Continue the story...",
  },
  zh: {
    "action.create": "创建",
    "action.createUser": "创建用户",
    "action.join": "加入",
    "action.login": "登录",
    "action.logout": "退出登录",
    "action.ready": "准备",
    "action.roll": "掷 d100",
    "action.startDice": "开始掷骰",
    "action.startGame": "开始游戏",
    "action.submit": "提交",
    "action.unready": "取消准备",
    "account.changePassword": "修改密码",
    "account.currentPassword": "当前密码",
    "account.newPassword": "新密码",
    "account.passwordUpdated": "密码已更新",
    "account.title": "账号",
    "auth.loginFailed": "登录失败",
    "auth.password": "密码",
    "auth.username": "用户名",
    "admin.createUser": "创建用户",
    "admin.empty": "暂无用户",
    "admin.passwordHint": "至少 8 个字符",
    "admin.generatedPassword": "初始密码",
    "admin.generatedPasswordHelp": "请只展示/传递一次；系统不会明文存储该密码。",
    "admin.noPasswordInput": "初始密码会由用户名和服务器密钥自动生成。",
    "admin.role": "角色",
    "admin.title": "开发者后台",
    "admin.users": "用户列表",
    "admin.userCreated": "用户已创建",
    "admin.disable": "禁用",
    "admin.restore": "恢复",
    "admin.resetPassword": "重置密码",
    "admin.disabled": "已禁用",
    "admin.active": "可用",
    "admin.passwordReset": "密码已重置",
    "account.recentGames": "最近对局",
    "account.noGames": "暂无已结束对局",
    "account.rank": "排名",
    "game.title": "对局归档",
    "game.finishedAt": "结束时间",
    "common.back": "返回",
    "common.loading": "加载中...",
    "common.optional": "可选",
    "common.status.active": "游戏中",
    "common.status.finished": "已结束",
    "common.status.lobby": "大厅",
    "common.status.rolling": "掷骰中",
    "home.createRoom": "创建房间",
    "home.admin": "后台",
    "home.account": "账号",
    "home.diceHighFirst": "点数大先写",
    "home.diceLowFirst": "点数小先写",
    "home.diceOrder": "骰子顺序",
    "home.joinRoom": "加入房间",
    "home.maxUnits": "每回合上限",
    "home.opening": "开场句",
    "home.passwordIfRequired": "如需要",
    "home.roomCode": "房间码",
    "home.roomPassword": "房间密码",
    "home.rounds": "总轮数",
    "home.theme": "主题",
    "home.turnTimeLimit": "每回合限时（秒）",
    "language.toggle": "EN",
    "room.actionFailed": "操作失败",
    "room.current": "当前",
    "room.d100": "百面骰",
    "room.leaderboard": "排行榜",
    "room.loading": "房间加载中...",
    "room.next": "下一位",
    "room.players": "玩家",
    "room.ready": "已准备",
    "room.scoreTurns": "次接龙",
    "room.story": "故事",
    "room.waiting": "等待中",
    "room.waitingTurn": "等待你的回合",
    "room.unitsHelp": "中日韩字符每字 1 unit；英文连续词和数字连续串各算 1 unit。",
    "room.timeLeft": "剩余时间",
    "room.writePlaceholder": "继续这个故事...",
  },
};

type I18nState = {
  language: Language;
  setLanguage: (language: Language) => void;
  toggleLanguage: () => void;
  t: (key: TranslationKey) => string;
};

const I18nContext = createContext<I18nState | null>(null);

export function I18nProvider({ children }: { children: React.ReactNode }) {
  const [language, setLanguage] = useState<Language>(() => {
    const stored = localStorage.getItem("cuckoo.language");
    return stored === "zh" || stored === "en" ? stored : "en";
  });

  useEffect(() => {
    localStorage.setItem("cuckoo.language", language);
    document.documentElement.lang = language === "zh" ? "zh-CN" : "en";
  }, [language]);

  const value = useMemo<I18nState>(() => ({
    language,
    setLanguage,
    toggleLanguage: () => setLanguage((current) => current === "en" ? "zh" : "en"),
    t: (key) => translations[language][key],
  }), [language]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n() {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useI18n must be used inside I18nProvider");
  return ctx;
}
