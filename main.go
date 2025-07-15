package main

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Cookie helper functions
func getCookieValue(r *http.Request, name string) (string, bool) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", false
	}
	return cookie.Value, true
}

type SessionRequestType int

const (
	SessionCreate SessionRequestType = iota
	SessionDestroy
	SessionQuery
	SessionCounting
	SessionPing
	SessionCountOnline
	SessionLogSessionIn
)

type SessionRequest struct {
	Type    SessionRequestType
	Payload interface{}
	RespCh  chan *SessionManagerResponse
}

type SessionBody struct {
	RemoteAddr      string
	LastContactTime int64
	SessionNumber   int
}

type SessionManagerResponse struct {
	Error error
	Data  interface{}
}

type SessionManager struct {
	closeCh      chan struct{}
	closed       bool
	sessionReqCh chan *SessionRequest
	privateStore map[string]*SessionBody
}

type SessionCreatePayload struct {
	RemoteAddr string
}

type SessionLogSessionInPayload struct {
	SessionId  string
	RemoteAddr string
}

const onlineTimeout = 10000

func (sm *SessionManager) countOnline() int {
	now := time.Now().UnixMilli()
	count := 0
	for _, sessionBody := range sm.privateStore {
		if now-sessionBody.LastContactTime < onlineTimeout {
			count++
		}
	}
	return count
}

func (sm *SessionManager) createSession(sessionId string, remoteAddr string) {
	sessionBody := new(SessionBody)
	sessionBody.RemoteAddr = remoteAddr
	sessionBody.SessionNumber = len(sm.privateStore)
	sessionBody.LastContactTime = time.Now().UnixMilli()
	sm.privateStore[sessionId] = sessionBody
}

func NewSessionManager() *SessionManager {
	sm := new(SessionManager)
	sm.closeCh = make(chan struct{})
	sm.sessionReqCh = make(chan *SessionRequest)
	sm.closed = false
	sm.privateStore = make(map[string]*SessionBody)
	go func() {
		for {
			select {
			case <-sm.closeCh:
				log.Println("SessionManager closed")
				return
			case sessionReq := <-sm.sessionReqCh:
				switch sessionReq.Type {
				case SessionCreate:
					if payload, ok := sessionReq.Payload.(*SessionCreatePayload); ok {
						sessionId := uuid.New().String()
						sm.createSession(sessionId, payload.RemoteAddr)
						sessionReq.RespCh <- &SessionManagerResponse{
							Error: nil,
							Data:  sessionId,
						}
					} else {
						sessionReq.RespCh <- &SessionManagerResponse{
							Error: errors.New("invalid session create payload"),
							Data:  nil,
						}
					}

				case SessionDestroy:
					if sessionId, ok := sessionReq.Payload.(string); ok {
						if sm.privateStore[sessionId] != nil {
							delete(sm.privateStore, sessionId)
						}
						sessionReq.RespCh <- &SessionManagerResponse{
							Error: nil,
							Data:  nil,
						}
					} else {
						sessionReq.RespCh <- &SessionManagerResponse{
							Error: errors.New("invalid session id"),
							Data:  nil,
						}
					}
				case SessionQuery:
					if sessionId, ok := sessionReq.Payload.(string); ok {
						sessionReq.RespCh <- &SessionManagerResponse{
							Error: nil,
							Data:  sm.privateStore[sessionId],
						}
					} else {
						sessionReq.RespCh <- &SessionManagerResponse{
							Error: errors.New("invalid session id"),
							Data:  nil,
						}
					}
				case SessionCounting:
					sessionReq.RespCh <- &SessionManagerResponse{
						Error: nil,
						Data:  len(sm.privateStore),
					}
				case SessionPing:
					if sessionId, ok := sessionReq.Payload.(string); ok {
						if sessionBody, ok := sm.privateStore[sessionId]; ok {
							sessionBody.LastContactTime = time.Now().UnixMilli()
							sessionReq.RespCh <- &SessionManagerResponse{
								Error: nil,
								Data:  nil,
							}
						} else {
							sessionReq.RespCh <- &SessionManagerResponse{
								Error: errors.New("session not found"),
								Data:  nil,
							}
						}

					} else {
						sessionReq.RespCh <- &SessionManagerResponse{
							Error: errors.New("invalid session id"),
							Data:  nil,
						}
					}
				case SessionCountOnline:
					sessionReq.RespCh <- &SessionManagerResponse{
						Error: nil,
						Data:  sm.countOnline(),
					}
				case SessionLogSessionIn:
					if payload, ok := sessionReq.Payload.(*SessionLogSessionInPayload); ok {
						if sessionBody, ok := sm.privateStore[payload.SessionId]; ok {
							sessionBody.RemoteAddr = payload.RemoteAddr
							sessionBody.LastContactTime = time.Now().UnixMilli()
							sessionReq.RespCh <- &SessionManagerResponse{
								Error: nil,
								Data:  nil,
							}
						} else {
							sm.createSession(payload.SessionId, payload.RemoteAddr)
							sessionReq.RespCh <- &SessionManagerResponse{
								Error: nil,
								Data:  nil,
							}
						}
					} else {
						sessionReq.RespCh <- &SessionManagerResponse{
							Error: errors.New("invalid session log session in payload"),
							Data:  nil,
						}
					}
				}
			}
		}
	}()

	return sm
}

func (sm *SessionManager) CreateSession(payload *SessionCreatePayload) (string, error) {
	req := new(SessionRequest)
	req.Payload = payload
	req.RespCh = make(chan *SessionManagerResponse)
	req.Type = SessionCreate

	go func() {
		sm.sessionReqCh <- req
	}()

	smResp := <-req.RespCh
	if smResp.Error != nil {
		return "", smResp.Error
	}

	return smResp.Data.(string), nil
}

func (sm *SessionManager) DestroySession(sessionId string) error {
	req := new(SessionRequest)
	req.Payload = sessionId
	req.RespCh = make(chan *SessionManagerResponse)
	req.Type = SessionDestroy

	go func() {
		sm.sessionReqCh <- req
	}()

	smResp := <-req.RespCh
	if smResp.Error != nil {
		return smResp.Error
	}

	return nil
}

func (sm *SessionManager) QuerySession(sessionId string) (*SessionBody, error) {
	req := new(SessionRequest)
	req.Payload = sessionId
	req.RespCh = make(chan *SessionManagerResponse)
	req.Type = SessionQuery

	go func() {
		sm.sessionReqCh <- req
	}()

	smResp := <-req.RespCh
	if smResp.Error != nil {
		return nil, smResp.Error
	}

	return smResp.Data.(*SessionBody), nil
}

func (sm *SessionManager) CountTotalSession() (int, error) {
	req := new(SessionRequest)
	req.Payload = nil
	req.RespCh = make(chan *SessionManagerResponse)
	req.Type = SessionCounting

	go func() {
		sm.sessionReqCh <- req
	}()

	smResp := <-req.RespCh
	if smResp.Error != nil {
		return 0, smResp.Error
	}

	return smResp.Data.(int), nil
}

func (sm *SessionManager) PingSession(sessionId string) error {
	req := new(SessionRequest)
	req.Payload = sessionId
	req.RespCh = make(chan *SessionManagerResponse)
	req.Type = SessionPing

	go func() {
		sm.sessionReqCh <- req
	}()

	smResp := <-req.RespCh
	if smResp.Error != nil {
		return smResp.Error
	}

	return nil
}

func (sm *SessionManager) CountOnlineSession() (int, error) {
	req := new(SessionRequest)
	req.Payload = nil
	req.RespCh = make(chan *SessionManagerResponse)
	req.Type = SessionCountOnline

	go func() {
		sm.sessionReqCh <- req
	}()

	smResp := <-req.RespCh
	if smResp.Error != nil {
		return 0, smResp.Error
	}

	return smResp.Data.(int), nil
}

func (sm *SessionManager) LogSessionIn(sessionId string, remoteAddr string) error {
	req := new(SessionRequest)
	req.Payload = &SessionLogSessionInPayload{
		SessionId:  sessionId,
		RemoteAddr: remoteAddr,
	}
	req.RespCh = make(chan *SessionManagerResponse)
	req.Type = SessionLogSessionIn

	go func() {
		sm.sessionReqCh <- req
	}()

	smResp := <-req.RespCh
	if smResp.Error != nil {
		return smResp.Error
	}

	return nil
}

func (sm *SessionManager) Close() {
	if sm.closed {
		return
	}
	sm.closed = true
	close(sm.closeCh)
}

type EchoMessageExtension = map[string]string

type EchoMessageBody struct {
	SendAt     *int64               `json:"sendAt,omitempty"`
	ReceivedAt *int64               `json:"receivedAt,omitempty"`
	Seq        int32                `json:"seq"`
	Extension  EchoMessageExtension `json:"extension,omitempty"`
}

type EchoMessageType int

const (
	EchoMessageTypePing EchoMessageType = iota // 0
	EchoMessageTypePong                        // 1
)

type EchoMessage struct {
	Type EchoMessageType `json:"type"`
	Data EchoMessageBody `json:"data"`
}

func main() {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool {
		return true
	}}

	// Initialize session manager
	sessionManager := NewSessionManager()
	defer sessionManager.Close()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		var ok bool
		var err error
		var sessionId string
		var extraHeaders http.Header = nil

		remoteAddr := r.RemoteAddr
		if sessionId, ok = getCookieValue(r, "sessionId"); ok {
			sessionManager.LogSessionIn(sessionId, remoteAddr)
			log.Printf("Logged in session: %s for %s", sessionId, remoteAddr)
		} else {
			sessionCreatePayload := new(SessionCreatePayload)
			sessionCreatePayload.RemoteAddr = remoteAddr
			sessionId, err = sessionManager.CreateSession(sessionCreatePayload)
			if err != nil {
				panic(err)
			}

			log.Printf("Created session: %s for %s", sessionId, remoteAddr)

			extraHeaders = http.Header{}
			extraHeaders.Add("Set-Cookie", "sessionId="+sessionId)
		}

		conn, err := upgrader.Upgrade(w, r, extraHeaders)

		if err != nil {
			log.Println(err)
			return
		}
		defer conn.Close()
		for {
			echoMsg := new(EchoMessage)
			if err := conn.ReadJSON(echoMsg); err != nil {
				log.Println(err)
				return
			}
			err := sessionManager.PingSession(sessionId)
			if err != nil {
				panic(err)
			}
			echoMsg.Type = EchoMessageTypePong
			echoMsg.Data.ReceivedAt = new(int64)
			*echoMsg.Data.ReceivedAt = time.Now().UnixMilli()
			extraEchoData := make(map[string]string)
			var onlineCount int = 0
			onlineCount, err = sessionManager.CountOnlineSession()
			if err != nil {
				panic(err)
			}
			extraEchoData["onlineCount"] = strconv.Itoa(onlineCount)
			sessionBody, err := sessionManager.QuerySession(sessionId)
			if err != nil {
				panic(err)
			}
			extraEchoData["sessionNumber"] = strconv.Itoa(sessionBody.SessionNumber)
			echoMsg.Data.Extension = extraEchoData
			if err := conn.WriteJSON(echoMsg); err != nil {
				log.Println(err)
				return
			}

		}
	})

	http.ListenAndServe(":8082", nil)
}
