package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"telegraph_uploader_v2/internal/database"
	"telegraph_uploader_v2/internal/repository"
	"telegraph_uploader_v2/internal/telegram"
	"telegraph_uploader_v2/internal/telegraph"
)

type PublicationService struct {
	tgClient    *telegraph.Client
	telegram    *telegram.Client
	historyRepo repository.HistoryRepository
	titleRepo   repository.TitleRepository
}

func NewPublicationService(tg *telegraph.Client, telegram *telegram.Client, history repository.HistoryRepository, titles repository.TitleRepository) *PublicationService {
	return &PublicationService{
		tgClient:    tg,
		telegram:    telegram,
		historyRepo: history,
		titleRepo:   titles,
	}
}

type PageResult struct {
	URL       string
	HistoryID uint
}

func (s *PublicationService) CreatePage(title string, images []string, titleID int) (PageResult, error) {
	url := s.tgClient.CreatePage(title, images)

	if len(url) < 4 || url[:4] != "http" {
		return PageResult{}, fmt.Errorf("telegraph error: %s", url)
	}

	var tID *uint
	if titleID > 0 {
		u := uint(titleID)
		tID = &u
	}
	// TODO: Pass tgphToken if needed, or get from somewhere.
	// In App it was `a.config.TelegraphToken`.
	// But `tgClient` has the token. `s.tgClient.Token`.

	id, err := s.historyRepo.Add(title, url, len(images), s.tgClient.Token, tID)

	return PageResult{URL: url, HistoryID: id}, err
}

func (s *PublicationService) EditPage(path string, title string, images []string, token string) string {
	return s.tgClient.EditPage(path, title, images, token)
}

func (s *PublicationService) GetPage(pageUrl string) (string, []string, error) {
	parts := strings.Split(pageUrl, "/")
	path := parts[len(parts)-1]
	return s.tgClient.GetPage(path)
}

func applyVariables(content string, variables []database.TitleVariable) string {
	replacerArgs := make([]string, 0, len(variables)*2)
	for _, v := range variables {
		if v.Key != "" {
			replacerArgs = append(replacerArgs, "{{"+v.Key+"}}", v.Value)
		}
	}
	if len(replacerArgs) > 0 {
		return strings.NewReplacer(replacerArgs...).Replace(content)
	}
	return content
}

func (s *PublicationService) PublishPost(ctx context.Context, historyID uint, channelID int64, accessHash int64, content string, scheduledTime time.Time) error {
	// 1. Get History Item
	item, err := s.historyRepo.GetByID(historyID)
	if err != nil {
		return err
	}

	// 2. Prepare Content
	content = strings.ReplaceAll(content, "{{Title}}", item.Title)
	content = strings.ReplaceAll(content, "{{Link}}", item.Url)

	// Custom Variables
	if item.TitleID != nil && *item.TitleID > 0 {
		title, err := s.titleRepo.GetByID(*item.TitleID)
		if err == nil {
			content = applyVariables(content, title.Variables)
		}
	}

	// 3. Schedule
	return s.telegram.ScheduleMessageByID(ctx, channelID, accessHash, content, scheduledTime)
}
