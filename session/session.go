package session

import "github.com/opencola/opencola/provider"

type Session struct {
	Messages []provider.Message
}

func New() *Session {
	return &Session{
		Messages: make([]provider.Message, 0),
	}
}

func (s *Session) AddMessage(role, content string) {
	s.Messages = append(s.Messages, provider.Message{
		Role:    role,
		Content: content,
	})
}

func (s *Session) Clear() {
	s.Messages = make([]provider.Message, 0)
}
