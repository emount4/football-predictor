package worker

import (
	"football-predictor/internal/service"
	"log"
	"time"
)

type Worker struct {
	matchService *service.MatchService
}

func NewWorker(ms *service.MatchService) *Worker {
	return &Worker{matchService: ms}
}

func (w *Worker) Start() {

	go func() {
		resultTicker := time.NewTicker(30 * time.Minute)
		log.Println("Worker: Результаты матчей проверяются каждые 30 минут")

		if err := w.matchService.ProcessFinishedMatches(); err != nil {
			log.Printf("Worker Error (Results): %v", err)
		}

		for range resultTicker.C {
			if err := w.matchService.ProcessFinishedMatches(); err != nil {
				log.Printf("Worker Error (Results): %v", err)
			}
		}
	}()

	go func() {
		log.Println("Worker: Расписание матчей обновляется раз в сутки")

		// Сразу стянем матчи при старте, чтобы база не была пустой
		if err := w.matchService.FetchDailyMatches(); err != nil {
			log.Printf("Worker Error (Fetch): %v", err)
		}

		for {
			now := time.Now()
			// Вычисляем сколько времени осталось до следующих 2 часов ночи
			nextRun := time.Date(now.Year(), now.Month(), now.Day()+1, 2, 0, 0, 0, now.Location())
			time.Sleep(time.Until(nextRun))

			if err := w.matchService.FetchDailyMatches(); err != nil {
				log.Printf("Worker Error (Fetch): %v", err)
			}
		}
	}()
}
