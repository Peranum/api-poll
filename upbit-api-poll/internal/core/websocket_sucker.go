package core

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/di"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/entity"
	"github.com/gorilla/websocket"
)

const (
	WebsocketSucketNewsChanSize = 128
	WebsocketSuckerPingInterval = 10 * time.Second
)

var ErrAlreadyStreaming = errors.New("already streaming")

type WebsocketSucker struct {
	connGuard sync.Mutex
	conn      *websocket.Conn

	streaming atomic.Bool

	deps di.Container
}

func NewWebsocketSucker(deps di.Container) *WebsocketSucker {
	sucker := &WebsocketSucker{
		deps: deps,
	}

	sucker.connectTillEstablished()

	return sucker
}

func (ws *WebsocketSucker) StreamNews(ctx context.Context) (<-chan entity.NewsTitle, error) {
	if !ws.streaming.CompareAndSwap(false, true) {
		return nil, ErrAlreadyStreaming
	}

	ws.deps.SendMessage(
		"🔍 <b>Websocket Sucker Started</b>\nURL: %s",
		ws.deps.Config.WebsocketSucker.URL,
	)

	newsChan := make(chan entity.NewsTitle, WebsocketSucketNewsChanSize)

	go func() {
		defer ws.streaming.Store(false)
		defer close(newsChan)

		wg := sync.WaitGroup{}

		wg.Add(1)

		go func() {
			defer wg.Done()
			ws.ping(ctx)
		}()

		ws.read(ctx, newsChan)

		wg.Wait()
	}()

	return newsChan, nil
}

func (ws *WebsocketSucker) read(ctx context.Context, newsChan chan entity.NewsTitle) {
	message := make(chan []byte)

	for {
		go func() {
			ws.connGuard.Lock()
			defer ws.connGuard.Unlock()

			for {
				_, receivedMessage, err := ws.conn.ReadMessage()
				if err != nil {
					ws.deps.Logger.Error("Failed to read message", "error", err)
					ws.connectTillEstablished()
					continue
				}

				message <- receivedMessage
				ws.deps.Logger.Debug("Received message", "message", string(receivedMessage))

				return
			}
		}()

		select {
		case <-ctx.Done():
			return

		case payload := <-message:
			suckerPayload := entity.SuckerPayload{}
			if err := suckerPayload.UnmarshalJSON(payload); err != nil {
				ws.deps.Logger.Error("Failed to unmarshal sucker payload", "error", err)
				continue
			}

			if suckerPayload.Exchange == "upbit" {
				newsChan <- suckerPayload.OriginalTitle

				ws.deps.SendMessage(
					"🚨 <b>New Listing Detected via WebSocket</b>\n\n"+
						"📰 <b>Title:</b> %s\n"+
						"🏢 <b>Exchange:</b> %s\n"+
						"🔗 <b>URL:</b> %s\n"+
						"⏰ <b>Time:</b> %s\n"+
						"🎯 <b>Tickers:</b> %s",
					suckerPayload.OriginalTitle,
					suckerPayload.Exchange,
					suckerPayload.URL,
					time.Unix(suckerPayload.Time, 0).Format("2006-01-02 15:04:05"),
					formatDetections(suckerPayload.Detections),
				)
			}

			ws.deps.Logger.Info("Received sucker payload", "payload", suckerPayload)
		}
	}
}

func (ws *WebsocketSucker) ping(ctx context.Context) {
	ticker := time.NewTicker(WebsocketSuckerPingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			func() {
				ws.connGuard.Lock()
				defer ws.connGuard.Unlock()

				if err := ws.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					ws.deps.Logger.Error("Failed to ping websocket", "error", err)
					ws.connectTillEstablished()
				}
			}()
		}
	}
}

func (ws *WebsocketSucker) connectTillEstablished() {
	delay := 1 * time.Second
	attempt := 1

	for {
		err := ws.connectToWS()
		if err == nil {
			return
		}

		ws.deps.Logger.Error("Failed to connect to websocket", "error", err, "attempt", attempt)
		attempt++

		time.Sleep(delay)
		delay = time.Duration(float64(delay) * math.E)
	}
}

func (ws *WebsocketSucker) connectToWS() error {
	headers := make(http.Header)
	headers.Add("Authorization", "Bearer "+ws.deps.Config.WebsocketSucker.APIKey)

	conn, _, err := websocket.DefaultDialer.Dial(ws.deps.Config.WebsocketSucker.URL, headers)
	if err != nil {
		return err
	}

	ws.conn = conn

	return nil
}

// Helper function to format detections
func formatDetections(detections []entity.Detection) string {
	if len(detections) == 0 {
		return "None"
	}

	tickers := make([]string, len(detections))
	for i, detection := range detections {
		tickers[i] = detection.Ticker
	}
	return strings.Join(tickers, ", ")
}
