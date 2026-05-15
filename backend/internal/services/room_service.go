package services

import (
	"crypto/rand"
	"errors"
	"math/big"
	"sort"
	"strings"
	"time"

	"cuckoo/backend/internal/auth"
	"cuckoo/backend/internal/models"
	"cuckoo/backend/internal/utils"
	"gorm.io/gorm"
)

type Broadcaster interface {
	Broadcast(roomCode, eventType string, payload interface{})
}

type RoomService struct {
	db *gorm.DB
	rt Broadcaster
}

type RoomSettingsInput struct {
	Theme           string `json:"theme"`
	OpeningSentence string `json:"openingSentence"`
	MaxUnitsPerTurn int    `json:"maxUnitsPerTurn"`
	TotalRounds     int    `json:"totalRounds"`
	DiceOrder       string `json:"diceOrder"`
}

type RoomSnapshot struct {
	Room          models.Room           `json:"room"`
	Contributions []models.Contribution `json:"contributions"`
	Results       []models.GameResult   `json:"results"`
	CurrentPlayer *models.RoomPlayer    `json:"currentPlayer"`
	NextPlayer    *models.RoomPlayer    `json:"nextPlayer"`
}

func NewRoomService(db *gorm.DB, rt Broadcaster) *RoomService {
	return &RoomService{db: db, rt: rt}
}

func (s *RoomService) CreateRoom(hostID uint, password string, input RoomSettingsInput) (*RoomSnapshot, error) {
	if err := validateSettings(input); err != nil {
		return nil, err
	}
	code, err := s.generateCode()
	if err != nil {
		return nil, err
	}
	var passwordHash *string
	if strings.TrimSpace(password) != "" {
		hash, err := auth.HashPassword(password)
		if err != nil {
			return nil, err
		}
		passwordHash = &hash
	}
	now := time.Now()
	room := models.Room{
		Code: code, HostUserID: hostID, Status: models.RoomLobby, PasswordHash: passwordHash,
		Settings: models.RoomSettings{
			Theme: strings.TrimSpace(input.Theme), OpeningSentence: strings.TrimSpace(input.OpeningSentence),
			MaxUnitsPerTurn: input.MaxUnitsPerTurn, TotalRounds: input.TotalRounds, DiceOrder: input.DiceOrder,
		},
		Players: []models.RoomPlayer{{UserID: hostID, Ready: true, JoinedAt: now}},
	}
	if err := s.db.Create(&room).Error; err != nil {
		return nil, err
	}
	return s.Snapshot(code)
}

func (s *RoomService) JoinRoom(userID uint, code, password string) (*RoomSnapshot, error) {
	room, err := s.findRoom(code)
	if err != nil {
		return nil, err
	}
	if room.PasswordHash != nil && !auth.VerifyPassword(*room.PasswordHash, password) {
		return nil, errors.New("invalid room password")
	}
	var count int64
	if err := s.db.Model(&models.RoomPlayer{}).Where("room_id = ? AND user_id = ?", room.ID, userID).Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		if room.Status != models.RoomLobby {
			return nil, errors.New("room already started")
		}
		if err := s.db.Create(&models.RoomPlayer{RoomID: room.ID, UserID: userID, JoinedAt: time.Now()}).Error; err != nil {
			return nil, err
		}
		s.broadcast(code, "room.player_joined")
	}
	return s.Snapshot(code)
}

func (s *RoomService) Snapshot(code string) (*RoomSnapshot, error) {
	room, err := s.findRoom(code)
	if err != nil {
		return nil, err
	}
	var contributions []models.Contribution
	if err := s.db.Preload("User").Where("room_id = ?", room.ID).Order("round_number, turn_index").Find(&contributions).Error; err != nil {
		return nil, err
	}
	var results []models.GameResult
	if err := s.db.Preload("User").Where("room_id = ?", room.ID).Order("score_total desc").Find(&results).Error; err != nil {
		return nil, err
	}
	current, next := currentAndNext(room)
	return &RoomSnapshot{Room: *room, Contributions: contributions, Results: results, CurrentPlayer: current, NextPlayer: next}, nil
}

func (s *RoomService) SetReady(userID uint, code string, ready bool) (*RoomSnapshot, error) {
	room, err := s.findRoom(code)
	if err != nil {
		return nil, err
	}
	if room.Status != models.RoomLobby {
		return nil, errors.New("ready can only change in lobby")
	}
	if err := s.db.Model(&models.RoomPlayer{}).Where("room_id = ? AND user_id = ?", room.ID, userID).Update("ready", ready).Error; err != nil {
		return nil, err
	}
	s.broadcast(code, "room.player_ready_changed")
	return s.Snapshot(code)
}

func (s *RoomService) UpdateSettings(hostID uint, code string, input RoomSettingsInput) (*RoomSnapshot, error) {
	if err := validateSettings(input); err != nil {
		return nil, err
	}
	room, err := s.findRoom(code)
	if err != nil {
		return nil, err
	}
	if room.HostUserID != hostID {
		return nil, errors.New("host only")
	}
	if room.Status != models.RoomLobby {
		return nil, errors.New("settings are locked")
	}
	if err := s.db.Model(&models.RoomSettings{}).Where("room_id = ?", room.ID).Updates(map[string]interface{}{
		"theme": input.Theme, "opening_sentence": input.OpeningSentence,
		"max_units_per_turn": input.MaxUnitsPerTurn, "total_rounds": input.TotalRounds, "dice_order": input.DiceOrder,
	}).Error; err != nil {
		return nil, err
	}
	s.broadcast(code, "room.settings_updated")
	return s.Snapshot(code)
}

func (s *RoomService) StartRoll(hostID uint, code string) (*RoomSnapshot, error) {
	room, err := s.findRoom(code)
	if err != nil {
		return nil, err
	}
	if room.HostUserID != hostID {
		return nil, errors.New("host only")
	}
	if room.Status != models.RoomLobby {
		return nil, errors.New("room is not in lobby")
	}
	if len(room.Players) < 2 {
		return nil, errors.New("at least two players are required")
	}
	for _, p := range room.Players {
		if !p.Ready {
			return nil, errors.New("all players must be ready")
		}
	}
	if err := s.db.Model(&models.Room{}).Where("id = ?", room.ID).Update("status", models.RoomRolling).Error; err != nil {
		return nil, err
	}
	s.broadcast(code, "game.roll_started")
	return s.Snapshot(code)
}

func (s *RoomService) Roll(userID uint, code string) (*RoomSnapshot, error) {
	room, err := s.findRoom(code)
	if err != nil {
		return nil, err
	}
	if room.Status != models.RoomRolling {
		return nil, errors.New("room is not rolling")
	}
	player := findPlayer(room, userID)
	if player == nil {
		return nil, errors.New("not a room player")
	}
	roll, err := randomInt(100)
	if err != nil {
		return nil, err
	}
	value := roll + 1
	if err := s.db.Model(&models.RoomPlayer{}).Where("id = ?", player.ID).Updates(map[string]interface{}{"roll": value, "order_index": nil}).Error; err != nil {
		return nil, err
	}
	snap, err := s.resolveOrderIfReady(code)
	if err == nil {
		s.broadcast(code, "game.player_rolled")
	}
	return snap, err
}

func (s *RoomService) StartGame(hostID uint, code string) (*RoomSnapshot, error) {
	room, err := s.findRoom(code)
	if err != nil {
		return nil, err
	}
	if room.HostUserID != hostID {
		return nil, errors.New("host only")
	}
	if room.Status != models.RoomRolling {
		return nil, errors.New("room is not ready to start")
	}
	for _, p := range room.Players {
		if p.OrderIndex == nil {
			return nil, errors.New("turn order is not finalized")
		}
	}
	now := time.Now()
	if err := s.db.Model(&models.Room{}).Where("id = ?", room.ID).Updates(map[string]interface{}{
		"status": models.RoomActive, "current_round": 1, "current_index": 0, "turn_started_at": &now,
	}).Error; err != nil {
		return nil, err
	}
	s.broadcast(code, "game.started")
	return s.Snapshot(code)
}

func (s *RoomService) SubmitContribution(userID uint, code, text string) (*RoomSnapshot, error) {
	room, err := s.findRoom(code)
	if err != nil {
		return nil, err
	}
	if room.Status != models.RoomActive {
		return nil, errors.New("game is not active")
	}
	current, _ := currentAndNext(room)
	if current == nil || current.UserID != userID {
		return nil, errors.New("not your turn")
	}
	text = strings.TrimSpace(text)
	units := utils.CountStoryUnits(text)
	if units == 0 {
		return nil, errors.New("contribution cannot be empty")
	}
	if units > room.Settings.MaxUnitsPerTurn {
		return nil, errors.New("contribution exceeds word limit")
	}
	now := time.Now()
	timeTaken := 0
	if room.TurnStartedAt != nil {
		timeTaken = int(now.Sub(*room.TurnStartedAt).Milliseconds())
	}
	scoreTime := 30 - min(timeTaken/2000, 30)
	contribution := models.Contribution{
		RoomID: room.ID, UserID: userID, RoundNumber: room.CurrentRound, TurnIndex: room.CurrentIndex,
		Text: text, Units: units, TimeTakenMs: timeTaken, ScoreCompliance: 50, ScoreTime: scoreTime,
		ScoreFluency: 20, ScoreTotal: 50 + scoreTime + 20,
	}
	if err := s.db.Create(&contribution).Error; err != nil {
		return nil, err
	}
	nextIndex := room.CurrentIndex + 1
	nextRound := room.CurrentRound
	if nextIndex >= len(room.Players) {
		nextIndex = 0
		nextRound++
	}
	if nextRound > room.Settings.TotalRounds {
		if err := s.finishRoom(room.ID, code); err != nil {
			return nil, err
		}
		s.broadcast(code, "game.ended")
		return s.Snapshot(code)
	}
	if err := s.db.Model(&models.Room{}).Where("id = ?", room.ID).Updates(map[string]interface{}{
		"current_round": nextRound, "current_index": nextIndex, "turn_started_at": &now,
	}).Error; err != nil {
		return nil, err
	}
	s.broadcast(code, "game.contribution_added")
	s.broadcast(code, "game.turn_changed")
	return s.Snapshot(code)
}

func (s *RoomService) finishRoom(roomID uint, code string) error {
	var rows []struct {
		UserID uint
		Score  int
		Count  int
	}
	if err := s.db.Model(&models.Contribution{}).Select("user_id, SUM(score_total) as score, COUNT(*) as count").Where("room_id = ?", roomID).Group("user_id").Scan(&rows).Error; err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Room{}).Where("id = ?", roomID).Updates(map[string]interface{}{"status": models.RoomFinished}).Error; err != nil {
			return err
		}
		for _, row := range rows {
			result := models.GameResult{RoomID: roomID, UserID: row.UserID, ScoreTotal: row.Score, Contributions: row.Count}
			if err := tx.Where("room_id = ? AND user_id = ?", roomID, row.UserID).Assign(result).FirstOrCreate(&result).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *RoomService) resolveOrderIfReady(code string) (*RoomSnapshot, error) {
	room, err := s.findRoom(code)
	if err != nil {
		return nil, err
	}
	rolls := map[int]int{}
	allRolled := true
	for _, p := range room.Players {
		if p.Roll == nil {
			allRolled = false
			continue
		}
		rolls[*p.Roll]++
	}
	if !allRolled {
		return s.Snapshot(code)
	}
	hasTie := false
	for _, count := range rolls {
		if count > 1 {
			hasTie = true
		}
	}
	if hasTie {
		for _, p := range room.Players {
			if p.Roll != nil && rolls[*p.Roll] > 1 {
				if err := s.db.Model(&models.RoomPlayer{}).Where("id = ?", p.ID).Updates(map[string]interface{}{"roll": nil, "order_index": nil}).Error; err != nil {
					return nil, err
				}
			}
		}
		s.broadcast(code, "game.roll_tie_required")
		return s.Snapshot(code)
	}
	sort.Slice(room.Players, func(i, j int) bool {
		if room.Settings.DiceOrder == models.DiceLowFirst {
			return *room.Players[i].Roll < *room.Players[j].Roll
		}
		return *room.Players[i].Roll > *room.Players[j].Roll
	})
	for i, p := range room.Players {
		if err := s.db.Model(&models.RoomPlayer{}).Where("id = ?", p.ID).Update("order_index", i).Error; err != nil {
			return nil, err
		}
	}
	s.broadcast(code, "game.order_finalized")
	return s.Snapshot(code)
}

func (s *RoomService) findRoom(code string) (*models.Room, error) {
	var room models.Room
	if err := s.db.Preload("Host").Preload("Settings").Preload("Players.User").Where("code = ?", strings.ToUpper(code)).First(&room).Error; err != nil {
		return nil, err
	}
	sort.Slice(room.Players, func(i, j int) bool {
		if room.Players[i].OrderIndex == nil && room.Players[j].OrderIndex == nil {
			return room.Players[i].JoinedAt.Before(room.Players[j].JoinedAt)
		}
		if room.Players[i].OrderIndex == nil {
			return false
		}
		if room.Players[j].OrderIndex == nil {
			return true
		}
		return *room.Players[i].OrderIndex < *room.Players[j].OrderIndex
	})
	return &room, nil
}

func currentAndNext(room *models.Room) (*models.RoomPlayer, *models.RoomPlayer) {
	if room.Status != models.RoomActive || len(room.Players) == 0 {
		return nil, nil
	}
	current := room.Players[room.CurrentIndex%len(room.Players)]
	next := room.Players[(room.CurrentIndex+1)%len(room.Players)]
	return &current, &next
}

func findPlayer(room *models.Room, userID uint) *models.RoomPlayer {
	for i := range room.Players {
		if room.Players[i].UserID == userID {
			return &room.Players[i]
		}
	}
	return nil
}

func validateSettings(input RoomSettingsInput) error {
	input.Theme = strings.TrimSpace(input.Theme)
	input.OpeningSentence = strings.TrimSpace(input.OpeningSentence)
	if len(input.Theme) < 1 || len(input.Theme) > 80 {
		return errors.New("theme must be 1-80 characters")
	}
	if len(input.OpeningSentence) < 1 || len(input.OpeningSentence) > 300 {
		return errors.New("opening sentence must be 1-300 characters")
	}
	if input.MaxUnitsPerTurn < 5 || input.MaxUnitsPerTurn > 80 {
		return errors.New("max units per turn must be 5-80")
	}
	if input.TotalRounds < 1 || input.TotalRounds > 10 {
		return errors.New("total rounds must be 1-10")
	}
	if input.DiceOrder == "" {
		return errors.New("dice order is required")
	}
	if input.DiceOrder != models.DiceHighFirst && input.DiceOrder != models.DiceLowFirst {
		return errors.New("invalid dice order")
	}
	return nil
}

func (s *RoomService) generateCode() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	for attempts := 0; attempts < 20; attempts++ {
		var b strings.Builder
		for i := 0; i < 6; i++ {
			n, err := randomInt(len(alphabet))
			if err != nil {
				return "", err
			}
			b.WriteByte(alphabet[n])
		}
		code := b.String()
		var count int64
		if err := s.db.Model(&models.Room{}).Where("code = ?", code).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return code, nil
		}
	}
	return "", errors.New("could not generate room code")
}

func randomInt(max int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}

func (s *RoomService) broadcast(code, eventType string) {
	if s.rt == nil {
		return
	}
	snap, err := s.Snapshot(code)
	if err == nil {
		s.rt.Broadcast(code, eventType, snap)
	}
}
