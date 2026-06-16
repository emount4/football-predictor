package service

import (
	"errors"
	"football-predictor/internal/domain"
	"time"
)

type predictService struct {
	matchRepo   domain.MatchRepository
	predictRepo domain.PredictRepository
}

func NewPredictService(mRepo domain.MatchRepository, pRepo domain.PredictRepository) domain.PredictService {
	return &predictService{
		matchRepo:   mRepo,
		predictRepo: pRepo,
	}
}

func (s *predictService) MakePrediction(tgID int64, matchID int64, choice string) error {
	// Проверяем, существует ли вообще такой матч
	match, err := s.matchRepo.GetMatchByID(matchID)
	if err != nil {
		return err
	}
	if match == nil {
		return errors.New("match not found")
	}

	// Проверка на то, не прошел ли матч
	if time.Now().After(match.MatchTime) || match.Status != "SCHEDULED" {
		return errors.New("predictions are locked: match has already started")
	}

	// 3. Валидируем сам выбор
	if choice != "1" && choice != "X" && choice != "2" {
		return errors.New("invalid choice: must be '1', 'X' or '2'")
	}

	pred := &domain.Prediction{
		UserID:     tgID,
		MatchID:    matchID,
		UserChoice: choice,
	}

	return s.predictRepo.SavePrediction(pred)
}
