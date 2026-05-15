package services

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
	"sort"
	"strings"
	"sync"
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
	db      *gorm.DB
	rt      Broadcaster
	scoring ScoringService
	timerMu sync.Mutex
	timers  map[uint]*time.Timer
}

type RoomSettingsInput struct {
	Theme                string `json:"theme"`
	OpeningSentence      string `json:"openingSentence"`
	MaxUnitsPerTurn      int    `json:"maxUnitsPerTurn"`
	TotalRounds          int    `json:"totalRounds"`
	TurnTimeLimitSeconds int    `json:"turnTimeLimitSeconds"`
	DiceOrder            string `json:"diceOrder"`
}

type RoomSnapshot struct {
	Room          models.Room           `json:"room"`
	Contributions []models.Contribution `json:"contributions"`
	Results       []models.GameResult   `json:"results"`
	CurrentPlayer *models.RoomPlayer    `json:"currentPlayer"`
	NextPlayer    *models.RoomPlayer    `json:"nextPlayer"`
}

type GameArchiveDTO struct {
	RoomCode        string                `json:"roomCode"`
	Theme           string                `json:"theme"`
	OpeningSentence string                `json:"openingSentence"`
	FullStory       string                `json:"fullStory"`
	PlayerOrder     []ArchivePlayer       `json:"playerOrder"`
	Contributions   []ArchiveContribution `json:"contributions"`
	Results         []ArchiveResult       `json:"results"`
	FinishedAt      time.Time             `json:"finishedAt"`
}

type GameSummaryDTO struct {
	RoomCode      string    `json:"roomCode"`
	Theme         string    `json:"theme"`
	ScoreTotal    int       `json:"scoreTotal"`
	Rank          int       `json:"rank"`
	Contributions int       `json:"contributions"`
	FinishedAt    time.Time `json:"finishedAt"`
}

type ArchivePlayer struct {
	UserID     uint   `json:"userId"`
	Username   string `json:"username"`
	OrderIndex int    `json:"orderIndex"`
}

type ArchiveContribution struct {
	UserID      uint      `json:"userId"`
	Username    string    `json:"username"`
	RoundNumber int       `json:"roundNumber"`
	TurnIndex   int       `json:"turnIndex"`
	Text        string    `json:"text"`
	Units       int       `json:"units"`
	IsSkipped   bool      `json:"isSkipped"`
	ScoreTotal  int       `json:"scoreTotal"`
	CreatedAt   time.Time `json:"createdAt"`
}

type ArchiveResult struct {
	UserID        uint   `json:"userId"`
	Username      string `json:"username"`
	ScoreTotal    int    `json:"scoreTotal"`
	Contributions int    `json:"contributions"`
	Rank          int    `json:"rank"`
}

func NewRoomService(db *gorm.DB, rt Broadcaster, scoring ScoringService) *RoomService {
	if scoring == nil {
		scoring = &LocalScoringService{}
	}
	return &RoomService{db: db, rt: rt, scoring: scoring, timers: map[uint]*time.Timer{}}
}

func (s *RoomService) CreateRoom(hostID uint, password string, input RoomSettingsInput) (*RoomSnapshot, error) {
	if err := s.ensureUserActive(hostID); err != nil {
		return nil, err
	}
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
			MaxUnitsPerTurn: input.MaxUnitsPerTurn, TotalRounds: input.TotalRounds,
			TurnTimeLimitSeconds: normalizedTurnLimit(input.TurnTimeLimitSeconds), DiceOrder: input.DiceOrder,
		},
		Players: []models.RoomPlayer{{UserID: hostID, Ready: true, JoinedAt: now}},
	}
	if err := s.db.Create(&room).Error; err != nil {
		return nil, err
	}
	return s.Snapshot(code)
}

func (s *RoomService) JoinRoom(userID uint, code, password string) (*RoomSnapshot, error) {
	if err := s.ensureUserActive(userID); err != nil {
		return nil, err
	}
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
		"max_units_per_turn": input.MaxUnitsPerTurn, "total_rounds": input.TotalRounds,
		"turn_time_limit_seconds": normalizedTurnLimit(input.TurnTimeLimitSeconds), "dice_order": input.DiceOrder,
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
	s.scheduleTurnTimeout(room.ID, code, room.Settings.TurnTimeLimitSeconds, now)
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
		if now.Sub(*room.TurnStartedAt) > time.Duration(room.Settings.TurnTimeLimitSeconds)*time.Second {
			return nil, errors.New("turn timed out")
		}
	}
	score := s.scoring.Score(context.Background(), ScoringInput{
		Text: text, Units: units, MaxUnits: room.Settings.MaxUnitsPerTurn,
		TimeTakenMs: timeTaken, TimeLimitMs: room.Settings.TurnTimeLimitSeconds * 1000,
	})
	contribution := models.Contribution{
		RoomID: room.ID, UserID: userID, RoundNumber: room.CurrentRound, TurnIndex: room.CurrentIndex,
		Text: text, Units: units, TimeTakenMs: timeTaken, ScoreCompliance: score.Compliance, ScoreTime: score.Time,
		ScoreFluency: score.Fluency, ScoreTotal: score.Total,
	}
	if err := s.db.Create(&contribution).Error; err != nil {
		return nil, err
	}
	finished, err := s.advanceTurn(room, now)
	if err != nil {
		return nil, err
	}
	if finished {
		if err := s.finishRoom(room.ID); err != nil {
			return nil, err
		}
		s.cancelTurnTimeout(room.ID)
		s.broadcast(code, "game.ended")
		return s.Snapshot(code)
	}
	s.scheduleTurnTimeout(room.ID, code, room.Settings.TurnTimeLimitSeconds, now)
	s.broadcast(code, "game.contribution_added")
	s.broadcast(code, "game.turn_changed")
	return s.Snapshot(code)
}

func (s *RoomService) finishRoom(roomID uint) error {
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
		return s.archiveRoom(tx, roomID)
	})
}

func (s *RoomService) HandleTurnTimeout(code string) (*RoomSnapshot, error) {
	room, err := s.findRoom(code)
	if err != nil {
		return nil, err
	}
	if room.Status != models.RoomActive || room.TurnStartedAt == nil {
		return s.Snapshot(code)
	}
	now := time.Now()
	if now.Sub(*room.TurnStartedAt) < time.Duration(room.Settings.TurnTimeLimitSeconds)*time.Second {
		return s.Snapshot(code)
	}
	current, _ := currentAndNext(room)
	if current == nil {
		return s.Snapshot(code)
	}
	timeTaken := int(now.Sub(*room.TurnStartedAt).Milliseconds())
	var existing int64
	if err := s.db.Model(&models.Contribution{}).Where("room_id = ? AND round_number = ? AND turn_index = ?", room.ID, room.CurrentRound, room.CurrentIndex).Count(&existing).Error; err != nil {
		return nil, err
	}
	if existing > 0 {
		return s.Snapshot(code)
	}
	contribution := models.Contribution{
		RoomID: room.ID, UserID: current.UserID, RoundNumber: room.CurrentRound, TurnIndex: room.CurrentIndex,
		Text: "[system] turn skipped after timeout", Units: 0, TimeTakenMs: timeTaken, IsSkipped: true,
		ScoreCompliance: 0, ScoreTime: 0, ScoreFluency: 0, ScoreTotal: 0,
	}
	if err := s.db.Create(&contribution).Error; err != nil {
		return nil, err
	}
	finished, err := s.advanceTurn(room, now)
	if err != nil {
		return nil, err
	}
	s.broadcast(code, "game.turn_timeout")
	if finished {
		if err := s.finishRoom(room.ID); err != nil {
			return nil, err
		}
		s.cancelTurnTimeout(room.ID)
		s.broadcast(code, "game.ended")
		return s.Snapshot(code)
	}
	s.scheduleTurnTimeout(room.ID, code, room.Settings.TurnTimeLimitSeconds, now)
	s.broadcast(code, "game.turn_changed")
	return s.Snapshot(code)
}

func (s *RoomService) ListUserGames(userID uint, limit int) ([]GameSummaryDTO, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	var rows []struct {
		RoomCode      string
		Theme         string
		ScoreTotal    int
		Contributions int
		FinishedAt    time.Time
		ResultsJSON   string
	}
	err := s.db.Table("game_results").
		Select("game_archives.room_code, game_archives.theme, game_results.score_total, game_results.contributions, game_archives.finished_at, game_archives.results_json").
		Joins("JOIN game_archives ON game_archives.room_id = game_results.room_id").
		Where("game_results.user_id = ?", userID).
		Order("game_archives.finished_at DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	games := make([]GameSummaryDTO, 0, len(rows))
	for _, row := range rows {
		rank := 0
		var results []ArchiveResult
		_ = json.Unmarshal([]byte(row.ResultsJSON), &results)
		for _, result := range results {
			if result.UserID == userID {
				rank = result.Rank
				break
			}
		}
		games = append(games, GameSummaryDTO{
			RoomCode: row.RoomCode, Theme: row.Theme, ScoreTotal: row.ScoreTotal,
			Rank: rank, Contributions: row.Contributions, FinishedAt: row.FinishedAt,
		})
	}
	return games, nil
}

func (s *RoomService) GameArchive(code string) (*GameArchiveDTO, error) {
	var archive models.GameArchive
	if err := s.db.Where("room_code = ?", strings.ToUpper(code)).First(&archive).Error; err != nil {
		return nil, err
	}
	dto := GameArchiveDTO{
		RoomCode: archive.RoomCode, Theme: archive.Theme, OpeningSentence: archive.OpeningSentence,
		FullStory: archive.FullStory, FinishedAt: archive.FinishedAt,
	}
	_ = json.Unmarshal([]byte(archive.PlayerOrderJSON), &dto.PlayerOrder)
	_ = json.Unmarshal([]byte(archive.ContributionsJSON), &dto.Contributions)
	_ = json.Unmarshal([]byte(archive.ResultsJSON), &dto.Results)
	return &dto, nil
}

func (s *RoomService) advanceTurn(room *models.Room, startedAt time.Time) (bool, error) {
	nextIndex := room.CurrentIndex + 1
	nextRound := room.CurrentRound
	if nextIndex >= len(room.Players) {
		nextIndex = 0
		nextRound++
	}
	if nextRound > room.Settings.TotalRounds {
		return true, nil
	}
	err := s.db.Model(&models.Room{}).Where("id = ? AND status = ?", room.ID, models.RoomActive).Updates(map[string]interface{}{
		"current_round": nextRound, "current_index": nextIndex, "turn_started_at": &startedAt,
	}).Error
	return false, err
}

func (s *RoomService) archiveRoom(tx *gorm.DB, roomID uint) error {
	var existing int64
	if err := tx.Model(&models.GameArchive{}).Where("room_id = ?", roomID).Count(&existing).Error; err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}
	var room models.Room
	if err := tx.Preload("Settings").Preload("Players.User").Where("id = ?", roomID).First(&room).Error; err != nil {
		return err
	}
	sort.Slice(room.Players, func(i, j int) bool {
		if room.Players[i].OrderIndex == nil {
			return false
		}
		if room.Players[j].OrderIndex == nil {
			return true
		}
		return *room.Players[i].OrderIndex < *room.Players[j].OrderIndex
	})
	var contributions []models.Contribution
	if err := tx.Preload("User").Where("room_id = ?", roomID).Order("round_number, turn_index").Find(&contributions).Error; err != nil {
		return err
	}
	var results []models.GameResult
	if err := tx.Preload("User").Where("room_id = ?", roomID).Order("score_total desc, contributions desc, user_id asc").Find(&results).Error; err != nil {
		return err
	}
	playerOrder := make([]ArchivePlayer, 0, len(room.Players))
	for i, player := range room.Players {
		orderIndex := i
		if player.OrderIndex != nil {
			orderIndex = *player.OrderIndex
		}
		playerOrder = append(playerOrder, ArchivePlayer{UserID: player.UserID, Username: player.User.Username, OrderIndex: orderIndex})
	}
	archiveContributions := make([]ArchiveContribution, 0, len(contributions))
	storyParts := []string{room.Settings.OpeningSentence}
	for _, c := range contributions {
		archiveContributions = append(archiveContributions, ArchiveContribution{
			UserID: c.UserID, Username: c.User.Username, RoundNumber: c.RoundNumber, TurnIndex: c.TurnIndex,
			Text: c.Text, Units: c.Units, IsSkipped: c.IsSkipped, ScoreTotal: c.ScoreTotal, CreatedAt: c.CreatedAt,
		})
		if !c.IsSkipped && strings.TrimSpace(c.Text) != "" {
			storyParts = append(storyParts, c.Text)
		}
	}
	archiveResults := make([]ArchiveResult, 0, len(results))
	for i, result := range results {
		archiveResults = append(archiveResults, ArchiveResult{
			UserID: result.UserID, Username: result.User.Username, ScoreTotal: result.ScoreTotal,
			Contributions: result.Contributions, Rank: i + 1,
		})
	}
	playerOrderJSON, err := json.Marshal(playerOrder)
	if err != nil {
		return err
	}
	contributionsJSON, err := json.Marshal(archiveContributions)
	if err != nil {
		return err
	}
	resultsJSON, err := json.Marshal(archiveResults)
	if err != nil {
		return err
	}
	archive := models.GameArchive{
		RoomID: room.ID, RoomCode: room.Code, Theme: room.Settings.Theme, OpeningSentence: room.Settings.OpeningSentence,
		FullStory: strings.Join(storyParts, "\n\n"), PlayerOrderJSON: string(playerOrderJSON),
		ContributionsJSON: string(contributionsJSON), ResultsJSON: string(resultsJSON), FinishedAt: time.Now(),
	}
	return tx.Create(&archive).Error
}

func (s *RoomService) scheduleTurnTimeout(roomID uint, code string, seconds int, startedAt time.Time) {
	if seconds <= 0 {
		return
	}
	s.cancelTurnTimeout(roomID)
	delay := time.Until(startedAt.Add(time.Duration(seconds) * time.Second))
	if delay < 0 {
		delay = 0
	}
	timer := time.AfterFunc(delay+200*time.Millisecond, func() {
		_, _ = s.HandleTurnTimeout(code)
	})
	s.timerMu.Lock()
	s.timers[roomID] = timer
	s.timerMu.Unlock()
}

func (s *RoomService) cancelTurnTimeout(roomID uint) {
	s.timerMu.Lock()
	defer s.timerMu.Unlock()
	if timer := s.timers[roomID]; timer != nil {
		timer.Stop()
		delete(s.timers, roomID)
	}
}

func (s *RoomService) ensureUserActive(userID uint) error {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return err
	}
	if user.IsDisabled {
		return errors.New("account disabled")
	}
	return nil
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
	turnLimit := normalizedTurnLimit(input.TurnTimeLimitSeconds)
	if turnLimit < 30 || turnLimit > 600 {
		return errors.New("turn time limit must be 30-600 seconds")
	}
	if input.DiceOrder == "" {
		return errors.New("dice order is required")
	}
	if input.DiceOrder != models.DiceHighFirst && input.DiceOrder != models.DiceLowFirst {
		return errors.New("invalid dice order")
	}
	return nil
}

func normalizedTurnLimit(seconds int) int {
	if seconds == 0 {
		return 120
	}
	return seconds
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
