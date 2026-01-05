package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type SendMessageRequest struct {
	ChatID              string `json:"chat_id"`
	Text                string `json:"text"`
	ParseMode           string `json:"parse_mode"`
	ScheduleDate        int64  `json:"schedule_date,omitempty"`
}

func SendScheduledMessage(token string, chatID string, text string, scheduleTime time.Time) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	
	reqBody := SendMessageRequest{
		ChatID:       chatID,
		Text:         text,
		ParseMode:    "HTML",
	}

	if !scheduleTime.IsZero() {
		reqBody.ScheduleDate = scheduleTime.Unix()
	}

	jsonData, _ := json.Marshal(reqBody)
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ошибка Telegram API: %d", resp.StatusCode)
	}
	return nil
}
