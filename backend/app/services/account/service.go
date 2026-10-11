package account

import (
	"errors"
	"strings"
	"time"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/services/crypto"
)

const SessionTTL = 30 * 24 * time.Hour

var (
	ErrInvalidCredentials = errors.New("account: invalid credentials")
	ErrSessionNotFound    = errors.New("account: session not found")
	ErrEmailTaken         = errors.New("account: email already in use")
)

type Service struct{}

func Default() *Service {
	return &Service{}
}

func (s *Service) Initialized() bool {
	count, err := facades.Orm().Query().Model(&models.User{}).Count()

	return err == nil && count > 0
}

func (s *Service) CreateUser(name, email, password string) (*models.User, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Name:         strings.TrimSpace(name),
		Email:        normalizeEmail(email),
		PasswordHash: hash,
	}

	if err := facades.Orm().Query().Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) UserByID(id uint64) (*models.User, error) {
	var user models.User
	if err := facades.Orm().Query().Where("id", id).FirstOrFail(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *Service) Authenticate(email, password string) (*models.User, error) {
	var user models.User
	if err := facades.Orm().Query().Where("email", normalizeEmail(email)).FirstOrFail(&user); err != nil {
		return nil, ErrInvalidCredentials
	}

	if !VerifyPassword(user.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}

	return &user, nil
}

func (s *Service) CreateSession(user *models.User, ip, userAgent string) (*models.Session, string, string, error) {
	token, err := crypto.RandomToken(32)
	if err != nil {
		return nil, "", "", err
	}

	csrf, err := crypto.RandomToken(32)
	if err != nil {
		return nil, "", "", err
	}

	now := time.Now()
	session := &models.Session{
		UserID:     user.ID,
		TokenHash:  crypto.HashToken(token),
		CSRFHash:   crypto.HashToken(csrf),
		IP:         ip,
		UserAgent:  truncate(userAgent, 512),
		LastSeenAt: now,
		ExpiresAt:  now.Add(SessionTTL),
	}

	if err := facades.Orm().Query().Create(session); err != nil {
		return nil, "", "", err
	}

	return session, token, csrf, nil
}

func (s *Service) ValidateSession(token string) (*models.Session, *models.User, error) {
	var session models.Session
	if err := facades.Orm().Query().Where("token_hash", crypto.HashToken(token)).FirstOrFail(&session); err != nil {
		return nil, nil, ErrSessionNotFound
	}

	if time.Now().After(session.ExpiresAt) {
		_, _ = facades.Orm().Query().Delete(&session)
		return nil, nil, ErrSessionNotFound
	}

	user, err := s.UserByID(session.UserID)
	if err != nil {
		return nil, nil, ErrSessionNotFound
	}

	s.touchSession(&session)

	return &session, user, nil
}

func (s *Service) touchSession(session *models.Session) {
	if time.Since(session.LastSeenAt) < 5*time.Minute {
		return
	}

	now := time.Now()
	session.LastSeenAt = now
	session.ExpiresAt = now.Add(SessionTTL)

	_ = facades.Orm().Query().Save(session)
}

func (s *Service) ListSessions(userID uint64) ([]models.Session, error) {
	var sessions []models.Session
	if err := facades.Orm().Query().
		Where("user_id", userID).
		OrderByDesc("last_seen_at").
		Find(&sessions); err != nil {
		return nil, err
	}

	return sessions, nil
}

func (s *Service) RevokeSession(userID, sessionID uint64) error {
	_, err := facades.Orm().Query().
		Where("id", sessionID).
		Where("user_id", userID).
		Delete(&models.Session{})

	return err
}

func (s *Service) RevokeOtherSessions(userID, keepSessionID uint64) error {
	_, err := facades.Orm().Query().
		Where("user_id", userID).
		Where("id != ?", keepSessionID).
		Delete(&models.Session{})

	return err
}

func (s *Service) CleanupExpired() (int64, error) {
	result, err := facades.Orm().Query().
		Where("expires_at < ?", time.Now()).
		Delete(&models.Session{})
	if err != nil {
		return 0, err
	}

	return result.RowsAffected, nil
}

func (s *Service) ChangePassword(user *models.User, current, next string) error {
	if !VerifyPassword(user.PasswordHash, current) {
		return ErrInvalidCredentials
	}

	hash, err := HashPassword(next)
	if err != nil {
		return err
	}

	user.PasswordHash = hash

	return facades.Orm().Query().Save(user)
}

func (s *Service) UpdateProfile(user *models.User, name, email string) error {
	normalized := normalizeEmail(email)

	if normalized != user.Email {
		count, err := facades.Orm().Query().
			Model(&models.User{}).
			Where("email", normalized).
			Where("id != ?", user.ID).
			Count()
		if err != nil {
			return err
		}
		if count > 0 {
			return ErrEmailTaken
		}
	}

	user.Name = strings.TrimSpace(name)
	user.Email = normalized

	return facades.Orm().Query().Save(user)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}

	return value[:max]
}
