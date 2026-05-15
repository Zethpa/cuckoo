package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"strings"
	"time"

	"cuckoo/backend/internal/auth"
	"cuckoo/backend/internal/config"
	"cuckoo/backend/internal/models"
	"gorm.io/gorm"
)

type AuthService struct {
	db  *gorm.DB
	cfg config.Config
}

func NewAuthService(db *gorm.DB, cfg config.Config) *AuthService {
	return &AuthService{db: db, cfg: cfg}
}

func (s *AuthService) SeedAdmin() error {
	if s.cfg.AdminUsername == "" || s.cfg.AdminPassword == "" {
		return nil
	}
	var count int64
	if err := s.db.Model(&models.User{}).Where("username = ?", s.cfg.AdminUsername).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return s.AddUser(s.cfg.AdminUsername, s.cfg.AdminPassword, models.RoleAdmin)
}

func (s *AuthService) AddUser(username, password, role string) error {
	username = strings.TrimSpace(username)
	if username == "" || len(password) < 8 {
		return errors.New("username is required and password must be at least 8 characters")
	}
	if role == "" {
		role = models.RolePlayer
	}
	if role != models.RoleAdmin && role != models.RolePlayer {
		return errors.New("role must be admin or player")
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	return s.db.Create(&models.User{Username: username, PasswordHash: hash, Role: role}).Error
}

func (s *AuthService) AddUserWithGeneratedPassword(username, role string) (string, error) {
	password := s.GenerateInitialPassword(username)
	return password, s.AddUser(username, password, role)
}

func (s *AuthService) GenerateInitialPassword(username string) string {
	normalized := strings.ToLower(strings.TrimSpace(username))
	mac := hmac.New(sha256.New, []byte(s.cfg.JWTSecret))
	_, _ = mac.Write([]byte("cuckoo-initial-password:" + normalized))
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(mac.Sum(nil))
	return "ck-" + strings.ToLower(encoded[:4]) + "-" + strings.ToLower(encoded[4:8]) + "-" + strings.ToLower(encoded[8:12])
}

func (s *AuthService) ChangePassword(userID uint, currentPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return errors.New("new password must be at least 8 characters")
	}
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return err
	}
	if !auth.VerifyPassword(user.PasswordHash, currentPassword) {
		return errors.New("current password is incorrect")
	}
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("password_hash", hash).Error
}

func (s *AuthService) ListUsers() ([]models.User, error) {
	var users []models.User
	if err := s.db.Order("created_at desc").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (s *AuthService) Login(username, password string) (*models.User, string, error) {
	var user models.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, "", errors.New("invalid credentials")
	}
	if user.IsDisabled {
		return nil, "", errors.New("account disabled")
	}
	if !auth.VerifyPassword(user.PasswordHash, password) {
		return nil, "", errors.New("invalid credentials")
	}
	token, err := auth.SignToken(s.cfg, user.ID, user.Username, user.Role)
	return &user, token, err
}

func (s *AuthService) FindUser(id uint) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *AuthService) DisableUser(actorID, targetID uint) error {
	if actorID == targetID {
		return errors.New("admin cannot disable self")
	}
	var target models.User
	if err := s.db.First(&target, targetID).Error; err != nil {
		return err
	}
	if target.IsDisabled {
		return nil
	}
	if target.Role == models.RoleAdmin {
		ok, err := s.hasAnotherActiveAdmin(targetID)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("at least one active admin is required")
		}
	}
	now := time.Now()
	return s.db.Model(&models.User{}).Where("id = ?", targetID).Updates(map[string]interface{}{"is_disabled": true, "disabled_at": &now}).Error
}

func (s *AuthService) RestoreUser(targetID uint) error {
	return s.db.Model(&models.User{}).Where("id = ?", targetID).Updates(map[string]interface{}{"is_disabled": false, "disabled_at": nil}).Error
}

func (s *AuthService) ResetPassword(targetID uint) (string, error) {
	var user models.User
	if err := s.db.First(&user, targetID).Error; err != nil {
		return "", err
	}
	password := s.GenerateInitialPassword(user.Username + "-" + time.Now().UTC().Format(time.RFC3339Nano))
	hash, err := auth.HashPassword(password)
	if err != nil {
		return "", err
	}
	return password, s.db.Model(&models.User{}).Where("id = ?", targetID).Update("password_hash", hash).Error
}

func (s *AuthService) IsActiveUser(userID uint) error {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return err
	}
	if user.IsDisabled {
		return errors.New("account disabled")
	}
	return nil
}

func (s *AuthService) hasAnotherActiveAdmin(userID uint) (bool, error) {
	var count int64
	err := s.db.Model(&models.User{}).Where("id <> ? AND role = ? AND is_disabled = ?", userID, models.RoleAdmin, false).Count(&count).Error
	return count > 0, err
}
