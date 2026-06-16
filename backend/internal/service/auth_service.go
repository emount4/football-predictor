package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"football-predictor/internal/domain"
	"football-predictor/internal/repository"
	"net/url"
	"sort"
	"strings"
)

type AuthService struct {
	repo     *repository.Repository
	botToken string
}

func NewAuthService(r *repository.Repository, botToken string) *AuthService {
	return &AuthService{
		repo:     r,
		botToken: botToken,
	}
}

// TGUser описывает структуру пользователя внутри initData
type TGUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	PhotoURL  string `json:"photo_url"` // Добавили поле в парсинг из Telegram
}

func (s *AuthService) ValidateAndGetUser(initDataStr string) (*domain.User, error) {
	if err := s.verifyTelegramInitData(initDataStr); err != nil {
		return nil, fmt.Errorf("auth failed: %w", err)
	}

	values, err := url.ParseQuery(initDataStr)
	if err != nil {
		return nil, err
	}

	userJSON := values.Get("user")
	if userJSON == "" {
		return nil, errors.New("user data missing in initData")
	}

	var tgUser TGUser
	if err := json.Unmarshal([]byte(userJSON), &tgUser); err != nil {
		return nil, err
	}

	displayName := tgUser.FirstName
	if tgUser.LastName != "" {
		displayName += " " + tgUser.LastName
	}

	user, err := s.repo.GetByTgID(tgUser.ID)
	if err != nil {
		return nil, err
	}

	if user == nil {
		// Регистрация нового
		user = &domain.User{
			TgID:        tgUser.ID,
			Username:    tgUser.Username,
			DisplayName: displayName,
			PhotoURL:    tgUser.PhotoURL, // Сохраняем url
			TotalPoints: 0,
		}
		if err := s.repo.Create(user); err != nil {
			return nil, err
		}
	} else {
		// Если юзер уже есть, обновляем его профиль (вдруг поменял аватарку или имя)
		if user.PhotoURL != tgUser.PhotoURL || user.DisplayName != displayName || user.Username != tgUser.Username {
			_ = s.repo.UpdateProfile(tgUser.ID, tgUser.Username, displayName, tgUser.PhotoURL)
			user.PhotoURL = tgUser.PhotoURL
			user.DisplayName = displayName
			user.Username = tgUser.Username
		}
	}

	return user, nil
}

// Внутренний метод валидации хэша Telegram
func (s *AuthService) verifyTelegramInitData(initDataStr string) error {
	values, err := url.ParseQuery(initDataStr)
	if err != nil {
		return err
	}

	tgHash := values.Get("hash")
	if tgHash == "" {
		return errors.New("hash missing")
	}

	// Собираем все параметры, кроме самого hash, сортируем по алфавиту
	var keys []string
	for k := range values {
		if k != "hash" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var dataCheckArr []string
	for _, k := range keys {
		dataCheckArr = append(dataCheckArr, fmt.Sprintf("%s=%s", k, values.Get(k)))
	}
	dataCheckString := strings.Join(dataCheckArr, "\n")

	// Считаем секретный ключ на основе токена бота
	mac := hmac.New(sha256.New, []byte("WebAppData"))
	mac.Write([]byte(s.botToken))
	secretKey := mac.Sum(nil)

	// Считаем финальный хэш от нашей строки параметров
	mac2 := hmac.New(sha256.New, secretKey)
	mac2.Write([]byte(dataCheckString))
	calcHash := hex.EncodeToString(mac2.Sum(nil))

	// Сравниваем то, что прислал фронтенд, с тем, что вычислили мы
	if calcHash != tgHash {
		return errors.New("data integrity compromised (invalid hash)")
	}

	return nil
}
