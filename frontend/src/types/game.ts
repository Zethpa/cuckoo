export type User = {
  id: number;
  username: string;
  role: "admin" | "player";
  isDisabled: boolean;
  disabledAt?: string | null;
  createdAt?: string;
  updatedAt?: string;
};

export type RoomSettings = {
  theme: string;
  openingSentence: string;
  maxUnitsPerTurn: number;
  totalRounds: number;
  turnTimeLimitSeconds: number;
  diceOrder: "high_first" | "low_first";
};

export type RoomPlayer = {
  id: number;
  roomId: number;
  userId: number;
  user: User;
  ready: boolean;
  roll: number | null;
  orderIndex: number | null;
  joinedAt: string;
};

export type Room = {
  id: number;
  code: string;
  hostUserId: number;
  host: User;
  status: "lobby" | "rolling" | "active" | "finished";
  currentRound: number;
  currentIndex: number;
  turnStartedAt: string | null;
  settings: RoomSettings;
  players: RoomPlayer[];
};

export type Contribution = {
  id: number;
  roomId: number;
  userId: number;
  user: User;
  roundNumber: number;
  turnIndex: number;
  text: string;
  units: number;
  timeTakenMs: number;
  isSkipped: boolean;
  scoreTotal: number;
  createdAt: string;
};

export type GameResult = {
  id: number;
  roomId: number;
  userId: number;
  user: User;
  scoreTotal: number;
  contributions: number;
};

export type RoomSnapshot = {
  room: Room;
  contributions: Contribution[];
  results: GameResult[];
  currentPlayer: RoomPlayer | null;
  nextPlayer: RoomPlayer | null;
};

export type RoomEvent = {
  type: string;
  roomCode: string;
  payload: RoomSnapshot | DraftUpdate;
  sentAt: string;
};

export type DraftUpdate = {
  userId: number;
  text: string;
};

export type GameSummary = {
  roomCode: string;
  theme: string;
  scoreTotal: number;
  rank: number;
  contributions: number;
  finishedAt: string;
};

export type GameArchive = {
  roomCode: string;
  theme: string;
  openingSentence: string;
  fullStory: string;
  playerOrder: Array<{ userId: number; username: string; orderIndex: number }>;
  contributions: Array<{
    userId: number;
    username: string;
    roundNumber: number;
    turnIndex: number;
    text: string;
    units: number;
    isSkipped: boolean;
    scoreTotal: number;
    createdAt: string;
  }>;
  results: Array<{ userId: number; username: string; scoreTotal: number; contributions: number; rank: number }>;
  finishedAt: string;
};
