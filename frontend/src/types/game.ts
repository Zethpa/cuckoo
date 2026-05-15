export type User = {
  id: number;
  username: string;
  role: "admin" | "player";
  createdAt?: string;
  updatedAt?: string;
};

export type RoomSettings = {
  theme: string;
  openingSentence: string;
  maxUnitsPerTurn: number;
  totalRounds: number;
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
  payload: RoomSnapshot;
  sentAt: string;
};
