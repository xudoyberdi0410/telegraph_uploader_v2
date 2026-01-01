package telegraph // <--- Лучше назвать пакет так же, как папку

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"telegraph_uploader_v2/internal/config"
)

// Client хранит настройки для работы с Telegraph
type Client struct {
	Token string
}

// New создает нового клиента. Мы передаем ему конфиг целиком.
func New(cfg *config.Config) *Client {
	return &Client{
		Token: cfg.TelegraphToken,
	}
}

// Структуры для парсинга JSON
type TelegraphResponse struct {
	Ok     bool `json:"ok"`
	Result struct {
		Url string `json:"url"`
	} `json:"result"`
	Error string `json:"error"`
}

type TelegraphNode struct {
	Tag   string            `json:"tag"`
	Attrs map[string]string `json:"attrs"`
}

// CreatePage теперь метод структуры Client (c *Client)
// Мы переименовали CreateTelegraphPage -> CreatePage, так как пакет уже называется telegraph
func (c *Client) CreatePage(title string, imageUrls []string) string {
	// Используем токен из структуры
	token := c.Token

	// Если токена нет в конфиге, создаем временный аккаунт
	if token == "" {
		var err error
		token, err = createTelegraphAccount("MangaUploader")
		if err != nil {
			return "Ошибка создания аккаунта Telegraph: " + err.Error()
		}
		// Запоминаем токен в памяти клиента, чтобы не создавать аккаунт каждый раз
		c.Token = token 
		fmt.Println("ВНИМАНИЕ: Создан новый временный токен Telegraph:", token)
	}

	// Формируем контент
	var content []TelegraphNode
	for _, link := range imageUrls {
		node := TelegraphNode{
			Tag: "img",
			Attrs: map[string]string{
				"src": link,
			},
		}
		content = append(content, node)
	}

	contentJson, err := json.Marshal(content)
	if err != nil {
		return "Ошибка JSON: " + err.Error()
	}

	apiURL := "https://api.telegra.ph/createPage"
	data := url.Values{}
	data.Set("access_token", token)
	data.Set("title", title)
	data.Set("content", string(contentJson))
	data.Set("return_content", "false")

	resp, err := http.PostForm(apiURL, data)
	if err != nil {
		return "Ошибка сети Telegraph: " + err.Error()
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	
	var tgResp TelegraphResponse
	if err := json.Unmarshal(body, &tgResp); err != nil {
		return "Ошибка ответа API: " + string(body)
	}

	if !tgResp.Ok {
		return "Telegraph API Error: " + tgResp.Error
	}

	return tgResp.Result.Url
}

// Вспомогательная функция (private)
func createTelegraphAccount(shortName string) (string, error) {
	apiURL := "https://api.telegra.ph/createAccount"
	data := url.Values{}
	data.Set("short_name", shortName)
	data.Set("author_name", "MangaBot")

	resp, err := http.PostForm(apiURL, data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	
	type AccountResp struct {
		Ok     bool `json:"ok"`
		Result struct {
			AccessToken string `json:"access_token"`
		} `json:"result"`
	}
	var acc AccountResp
	json.Unmarshal(body, &acc)
	
	if !acc.Ok {
		return "", fmt.Errorf("не удалось создать аккаунт")
	}
	return acc.Result.AccessToken, nil
}
