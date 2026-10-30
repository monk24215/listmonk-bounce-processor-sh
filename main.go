package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"time"
)

type Config struct {
	POP3Host      string
	POP3Port      string
	POP3User      string
	POP3Pass      string
	ListmonkURL   string
	ListmonkUser  string
	ListmonkPass  string
	ScanInterval  time.Duration
	LogFile       string
}

type Logger struct {
	file *os.File
}

func (l *Logger) Log(level, msg string, data map[string]interface{}) {
	logEntry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"level":     level,
		"message":   msg,
	}
	for k, v := range data {
		logEntry[k] = v
	}
	jsonBytes, _ := json.Marshal(logEntry)
	l.file.Write(append(jsonBytes, '\n'))
	fmt.Printf("[%s] %s: %s\n", level, msg, jsonBytes)
}

func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}

func NewLogger(path string) (*Logger, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &Logger{file: f}, nil
}

func parseDSN(body string) (string, string) {
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.ToLower(line), "final-recipient:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				email := strings.TrimSpace(parts[len(parts)-1])
				email = strings.TrimPrefix(email, "rfc822;")
				return strings.TrimSpace(email), "hard"
			}
		}
		if strings.HasPrefix(strings.ToLower(line), "action:") {
			action := strings.TrimSpace(strings.Split(line, ":")[1])
			if strings.ToLower(action) == "failed" {
				return "", "hard"
			}
		}
	}
	return "", ""
}

func extractBounceEmail(msg string) (string, string) {
	reader := strings.NewReader(msg)
	m, err := mail.ReadMessage(reader)
	if err != nil {
		return "", ""
	}

	bodyBytes, _ := io.ReadAll(m.Body)
	body := string(bodyBytes)

	email, bounceType := parseDSN(body)
	if email != "" {
		return email, bounceType
	}

	if strings.Contains(body, "Undeliverable") || strings.Contains(body, "Failed") {
		if from := m.Header.Get("From"); from != "" {
			return from, "hard"
		}
	}

	return "", ""
}

func processBounceInListmonk(logger *Logger, cfg Config, email string, bounceType string) error {
	url := fmt.Sprintf("%s/api/bounces", cfg.ListmonkURL)

	payload := map[string]interface{}{
		"email": email,
		"type":  bounceType,
	}
	jsonPayload, _ := json.Marshal(payload)

	req := bytes.NewReader(jsonPayload)
	client := &http.Client{Timeout: 10 * time.Second}

	httpReq, _ := http.NewRequest("POST", url, req)
	httpReq.SetBasicAuth(cfg.ListmonkUser, cfg.ListmonkPass)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		logger.Log("ERROR", "Failed to call Listmonk API", map[string]interface{}{
			"email": email,
			"error": err.Error(),
		})
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Log("WARN", "Listmonk API returned non-success status", map[string]interface{}{
			"email":  email,
			"status": resp.StatusCode,
		})
	}

	logger.Log("INFO", "Bounce processed", map[string]interface{}{
		"email": email,
		"type":  bounceType,
	})
	return nil
}

type POP3Client struct {
	conn net.Conn
}

func (c *POP3Client) sendCmd(cmd string) (string, error) {
	if _, err := fmt.Fprintf(c.conn, "%s\r\n", cmd); err != nil {
		return "", err
	}
	buf := make([]byte, 4096)
	n, err := c.conn.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}
	return string(buf[:n]), nil
}

func dialPOP3(host string, port string, user string, pass string) (*POP3Client, error) {
	addr := fmt.Sprintf("%s:%s", host, port)
	conn, err := tls.Dial("tcp", addr, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		return nil, err
	}

	c := &POP3Client{conn: conn}

	resp, _ := c.sendCmd("")
	if !strings.HasPrefix(resp, "+OK") {
		return nil, fmt.Errorf("unexpected POP3 response: %s", resp)
	}

	if _, err := c.sendCmd(fmt.Sprintf("USER %s", user)); err != nil {
		return nil, err
	}

	if _, err := c.sendCmd(fmt.Sprintf("PASS %s", pass)); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *POP3Client) listMessages() ([]int, error) {
	resp, err := c.sendCmd("LIST")
	if err != nil {
		return nil, err
	}

	var ids []int
	lines := strings.Split(strings.TrimSpace(resp), "\n")
	
	for i, line := range lines {
		// Skip first line (status line like "+OK 5 messages")
		if i == 0 {
			continue
		}
		
		line = strings.TrimSpace(line)
		if line == "." || line == "" {
			break
		}
		
		fields := strings.Fields(line)
		if len(fields) > 0 {
			var id int
			fmt.Sscanf(fields[0], "%d", &id)
			if id > 0 {
				ids = append(ids, id)
			}
		}
	}
	return ids, nil
}

func (c *POP3Client) retrieveMessage(id int) (string, error) {
	resp, err := c.sendCmd(fmt.Sprintf("RETR %d", id))
	if err != nil {
		return "", err
	}
	return resp, nil
}

func (c *POP3Client) deleteMessage(id int) error {
	_, err := c.sendCmd(fmt.Sprintf("DELE %d", id))
	return err
}

func (c *POP3Client) close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func scanBounces(logger *Logger, cfg Config) {
	conn, err := dialPOP3(cfg.POP3Host, cfg.POP3Port, cfg.POP3User, cfg.POP3Pass)
	if err != nil {
		logger.Log("ERROR", "Failed to connect to POP3", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	defer conn.close()

	ids, err := conn.listMessages()
	if err != nil {
		logger.Log("ERROR", "Failed to list messages", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	logger.Log("INFO", "Scanning bounces", map[string]interface{}{
		"message_count": len(ids),
	})

	for _, id := range ids {
		msg, err := conn.retrieveMessage(id)
		if err != nil {
			logger.Log("WARN", "Failed to retrieve message", map[string]interface{}{
				"id":    id,
				"error": err.Error(),
			})
			continue
		}

		email, bounceType := extractBounceEmail(msg)

		if email != "" {
			processBounceInListmonk(logger, cfg, email, bounceType)
			conn.deleteMessage(id)
		}
	}
}

func main() {
	cfg := Config{
		POP3Host:     os.Getenv("POP3_HOST"),
		POP3Port:     os.Getenv("POP3_PORT"),
		POP3User:     os.Getenv("POP3_USER"),
		POP3Pass:     os.Getenv("POP3_PASS"),
		ListmonkURL:  os.Getenv("LISTMONK_URL"),
		ListmonkUser: os.Getenv("LISTMONK_USER"),
		ListmonkPass: os.Getenv("LISTMONK_PASS"),
		ScanInterval: 5 * time.Hour,
		LogFile:      "/var/log/bounce-processor.log",
	}

	logger, err := NewLogger(cfg.LogFile)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	logger.Log("INFO", "Bounce processor started", map[string]interface{}{
		"version": "1.0.0",
	})

	// Scan immediately on startup
	scanBounces(logger, cfg)

	// Then scan again every 5 hours
	ticker := time.NewTicker(cfg.ScanInterval)
	defer ticker.Stop()

	for range ticker.C {
		scanBounces(logger, cfg)
	}
}
