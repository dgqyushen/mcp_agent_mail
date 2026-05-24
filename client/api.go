package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"agent-mail/model"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	jwt        string
	sitePass   string
}

func New(baseURL, jwt, sitePass string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
		jwt:        jwt,
		sitePass:   sitePass,
	}
}

func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.jwt)
	req.Header.Set("x-lang", "en")
	if c.sitePass != "" {
		req.Header.Set("x-custom-auth", c.sitePass)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s %s: %w", method, path, err)
	}
	if resp.StatusCode == 429 {
		time.Sleep(3 * time.Second)
		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("retry %s %s: %w", method, path, err)
		}
	}
	return resp, nil
}

func (c *Client) GetSettings() (*model.SettingsResponse, error) {
	resp, err := c.doRequest("GET", "/api/settings", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get settings: %s %s", resp.Status, string(body))
	}
	var result model.SettingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) ListParsedMails(limit, offset int) (*model.PaginatedResult, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/parsed_mails?limit=%d&offset=%d", limit, offset), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list mails: %s %s", resp.Status, string(body))
	}
	var result model.PaginatedResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetParsedMail(id int) (*model.ParsedMail, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/parsed_mail/%d", id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get mail %d: %s %s", id, resp.Status, string(body))
	}
	var result model.ParsedMail
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DeleteMail(id int) error {
	resp, err := c.doRequest("DELETE", fmt.Sprintf("/api/mails/%d", id), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete mail %d: %s %s", id, resp.Status, string(body))
	}
	return nil
}

func (c *Client) ClearInbox() error {
	resp, err := c.doRequest("DELETE", "/api/clear_inbox", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("clear inbox: %s %s", resp.Status, string(body))
	}
	return nil
}

func (c *Client) SendMail(body *model.SendMailBody) error {
	data, _ := json.Marshal(body)
	resp, err := c.doRequest("POST", "/api/send_mail", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("send mail: %s %s", resp.Status, string(respBody))
	}
	return nil
}

func (c *Client) ListSendbox(limit, offset int) (*model.SendboxResult, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/sendbox?limit=%d&offset=%d", limit, offset), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list sendbox: %s %s", resp.Status, string(body))
	}
	var result model.SendboxResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DeleteSendbox(id int) error {
	resp, err := c.doRequest("DELETE", fmt.Sprintf("/api/sendbox/%d", id), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete sent %d: %s %s", id, resp.Status, string(body))
	}
	return nil
}

func (c *Client) ClearSentItems() error {
	resp, err := c.doRequest("DELETE", "/api/clear_sent_items", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("clear sent: %s %s", resp.Status, string(body))
	}
	return nil
}

func (c *Client) GetAutoReply() (*model.AutoReplyConfig, error) {
	resp, err := c.doRequest("GET", "/api/auto_reply", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get auto reply: %s %s", resp.Status, string(body))
	}
	var result model.AutoReplyConfig
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) SetAutoReply(cfg *model.AutoReplyConfig) error {
	data, _ := json.Marshal(map[string]*model.AutoReplyConfig{"auto_reply": cfg})
	resp, err := c.doRequest("POST", "/api/auto_reply", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("set auto reply: %s %s", resp.Status, string(body))
	}
	return nil
}

func (c *Client) GetWebhook() (*model.WebhookSettings, error) {
	resp, err := c.doRequest("GET", "/api/webhook/settings", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get webhook: %s %s", resp.Status, string(body))
	}
	var result model.WebhookSettings
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) SetWebhook(cfg *model.WebhookSettings) error {
	data, _ := json.Marshal(cfg)
	resp, err := c.doRequest("POST", "/api/webhook/settings", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("set webhook: %s %s", resp.Status, string(body))
	}
	return nil
}

func (c *Client) ListAttachments() (*model.AttachmentListResult, error) {
	resp, err := c.doRequest("GET", "/api/attachment/list", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list attachments: %s %s", resp.Status, string(body))
	}
	var result model.AttachmentListResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
