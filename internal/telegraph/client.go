package telegraph

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
	Token   string
	BaseURL string
}

// New создает нового клиента. Мы передаем ему конфиг целиком.
func New(cfg *config.Config) *Client {
	return &Client{
		Token:   cfg.TelegraphToken,
		BaseURL: "https://api.telegra.ph",
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
		token, err = c.createAccount("MangaUploader")
		if err != nil {
			return "Ошибка создания аккаунта Telegraph: " + err.Error()
		}
		// Запоминаем токен в памяти клиента, чтобы не создавать аккаунт каждый раз
		c.Token = token
		fmt.Println("ВНИМАНИЕ: Создан новый временный аккаунт Telegraph")
	}

	content := c.imagesToNodes(imageUrls)

	contentJson, err := json.Marshal(content)
	if err != nil {
		return "Ошибка JSON: " + err.Error()
	}

	apiURL := c.BaseURL + "/createPage"
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

func (c *Client) createAccount(shortName string) (string, error) {
	apiURL := c.BaseURL + "/createAccount"
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

// EditPage редактирует существующую страницу
func (c *Client) EditPage(path string, title string, imageUrls []string, accessToken string) string {
	// Если токен не передан, берем из конфига (но лучше передавать тот, которым создавали)
	token := accessToken
	if token == "" {
		token = c.Token
	}

	content := c.imagesToNodes(imageUrls)

	contentJson, err := json.Marshal(content)
	if err != nil {
		return "Ошибка JSON: " + err.Error()
	}

	apiURL := c.BaseURL + "/editPage"
	data := url.Values{}
	data.Set("access_token", token)
	data.Set("path", path)
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

	// Успех, возвращаем URL (хотя он не меняется при редактировании)
	return tgResp.Result.Url
}

// imagesToNodes преобразует список URL-адресов изображений в узлы TelegraphNode
func (c *Client) imagesToNodes(imageUrls []string) []TelegraphNode {
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
	return content
}

type PageResponse struct {
	Ok     bool `json:"ok"`
	Result struct {
		Title   string        `json:"title"`
		Content []interface{} `json:"content"` // Content может быть строками или объектами
	} `json:"result"`
	Error string `json:"error"`
}

// GetPage получает заголовок и список изображений со страницы
func (c *Client) GetPage(path string) (string, []string, error) {
	apiURL := fmt.Sprintf("%s/getPage/%s?return_content=true", c.BaseURL, path)
	
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var pageResp PageResponse
	if err := json.Unmarshal(body, &pageResp); err != nil {
		return "", nil, err
	}

	if !pageResp.Ok {
		// Fix vet error: non-constant format string
		return "", nil, fmt.Errorf("%s", pageResp.Error)
	}

	var imageUrls []string
	for _, node := range pageResp.Result.Content {
		// Node может быть map[string]interface{} (если это тег)
		if nodeMap, ok := node.(map[string]interface{}); ok {
			if tag, ok := nodeMap["tag"].(string); ok && tag == "img" {
				if attrs, ok := nodeMap["attrs"].(map[string]interface{}); ok {
					if src, ok := attrs["src"].(string); ok {
						imageUrls = append(imageUrls, src)
					}
				}
			}
		}
	}

	return pageResp.Result.Title, imageUrls, nil
}
