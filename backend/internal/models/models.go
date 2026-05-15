package models

import "time"

const (
	RoleAdmin  = "admin"
	RolePlayer = "player"

	RoomLobby    = "lobby"
	RoomRolling  = "rolling"
	RoomActive   = "active"
	RoomFinished = "finished"

	DiceHighFirst = "high_first"
	DiceLowFirst  = "low_first"
)

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:80;not null" json:"username"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Role         string    `gorm:"size:24;not null;default:player" json:"role"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Room struct {
	ID            uint         `gorm:"primaryKey" json:"id"`
	Code          string       `gorm:"uniqueIndex;size:12;not null" json:"code"`
	HostUserID    uint         `gorm:"not null" json:"hostUserId"`
	Host          User         `gorm:"foreignKey:HostUserID" json:"host"`
	Status        string       `gorm:"size:24;not null;default:lobby" json:"status"`
	PasswordHash  *string      `json:"-"`
	CurrentRound  int          `gorm:"not null;default:0" json:"currentRound"`
	CurrentIndex  int          `gorm:"not null;default:0" json:"currentIndex"`
	TurnStartedAt *time.Time   `json:"turnStartedAt"`
	Settings      RoomSettings `json:"settings"`
	Players       []RoomPlayer `json:"players"`
	CreatedAt     time.Time    `json:"createdAt"`
	UpdatedAt     time.Time    `json:"updatedAt"`
}

type RoomSettings struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	RoomID          uint      `gorm:"uniqueIndex;not null" json:"roomId"`
	Theme           string    `gorm:"size:80;not null" json:"theme"`
	OpeningSentence string    `gorm:"size:300;not null" json:"openingSentence"`
	MaxUnitsPerTurn int       `gorm:"not null" json:"maxUnitsPerTurn"`
	TotalRounds     int       `gorm:"not null" json:"totalRounds"`
	DiceOrder       string    `gorm:"size:24;not null;default:high_first" json:"diceOrder"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type RoomPlayer struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	RoomID     uint      `gorm:"uniqueIndex:idx_room_user;not null" json:"roomId"`
	UserID     uint      `gorm:"uniqueIndex:idx_room_user;not null" json:"userId"`
	User       User      `json:"user"`
	Ready      bool      `gorm:"not null;default:false" json:"ready"`
	Roll       *int      `json:"roll"`
	OrderIndex *int      `json:"orderIndex"`
	JoinedAt   time.Time `json:"joinedAt"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type Turn struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	RoomID      uint       `gorm:"index;not null" json:"roomId"`
	UserID      uint       `gorm:"not null" json:"userId"`
	RoundNumber int        `gorm:"not null" json:"roundNumber"`
	TurnIndex   int        `gorm:"not null" json:"turnIndex"`
	StartedAt   time.Time  `json:"startedAt"`
	CompletedAt *time.Time `json:"completedAt"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type Contribution struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	RoomID          uint      `gorm:"index;not null" json:"roomId"`
	UserID          uint      `gorm:"not null" json:"userId"`
	User            User      `json:"user"`
	RoundNumber     int       `gorm:"not null" json:"roundNumber"`
	TurnIndex       int       `gorm:"not null" json:"turnIndex"`
	Text            string    `gorm:"type:text;not null" json:"text"`
	Units           int       `gorm:"not null" json:"units"`
	TimeTakenMs     int       `gorm:"not null" json:"timeTakenMs"`
	ScoreCompliance int       `gorm:"not null" json:"scoreCompliance"`
	ScoreTime       int       `gorm:"not null" json:"scoreTime"`
	ScoreFluency    int       `gorm:"not null" json:"scoreFluency"`
	ScoreTotal      int       `gorm:"not null" json:"scoreTotal"`
	CreatedAt       time.Time `json:"createdAt"`
}

type GameResult struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	RoomID        uint      `gorm:"uniqueIndex:idx_room_result_user;not null" json:"roomId"`
	UserID        uint      `gorm:"uniqueIndex:idx_room_result_user;not null" json:"userId"`
	User          User      `json:"user"`
	ScoreTotal    int       `gorm:"not null" json:"scoreTotal"`
	Contributions int       `gorm:"not null" json:"contributions"`
	CreatedAt     time.Time `json:"createdAt"`
}
