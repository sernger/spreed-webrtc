/*
 * Spreed WebRTC.
 * Copyright (C) 2013-2014 struktur AG
 *
 * This file is part of Spreed WebRTC.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package main

import (
	"errors"
	"fmt"
	"github.com/gorilla/securecookie"
	"sync"
	"time"
)

var sessionNonces *securecookie.SecureCookie

type Session struct {
	Id        string
	Sid       string
	Ua        string
	UpdateRev uint64
	Status    interface{}
	Nonce     string
	Prio      int
	mutex     sync.RWMutex
	userid    string
	stamp     int64
}

func NewSession(id, sid string) *Session {

	return &Session{
		Id:    id,
		Sid:   sid,
		Prio:  100,
		stamp: time.Now().Unix(),
	}

}

func (s *Session) Update(update *SessionUpdate) uint64 {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, key := range update.Types {

		//fmt.Println("type update", key)
		switch key {
		case "Ua":
			s.Ua = update.Ua
		case "Status":
			s.Status = update.Status
		case "Prio":
			s.Prio = update.Prio
		}

	}

	s.UpdateRev++
	return s.UpdateRev

}

func (s *Session) Authorize(realm string, st *SessionToken) (string, error) {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.Id != st.Id || s.Sid != st.Sid {
		return "", errors.New("session id mismatch")
	}
	if s.userid != "" {
		return "", errors.New("session already authenticated")
	}

	// Create authentication nonce.
	var err error
	s.Nonce, err = sessionNonces.Encode(fmt.Sprintf("%s@%s", s.Sid, realm), st.Userid)

	return s.Nonce, err

}

func (s *Session) Authenticate(realm string, st *SessionToken, userid string) error {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.userid != "" {
		return errors.New("session already authenticated")
	}
	if userid == "" {
		if s.Nonce == "" || s.Nonce != st.Nonce {
			return errors.New("nonce validation failed")
		}
		err := sessionNonces.Decode(fmt.Sprintf("%s@%s", s.Sid, realm), st.Nonce, &userid)
		if err != nil {
			return err
		}
		if st.Userid != userid {
			return errors.New("user id mismatch")
		}
		s.Nonce = ""
	}

	s.userid = userid
	s.stamp = time.Now().Unix()
	s.UpdateRev++
	return nil

}

func (s *Session) Token() *SessionToken {

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return &SessionToken{Id: s.Id, Sid: s.Sid, Userid: s.userid}
}

func (s *Session) Data() *DataSession {

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return &DataSession{
		Id:     s.Id,
		Userid: s.userid,
		Ua:     s.Ua,
		Status: s.Status,
		Rev:    s.UpdateRev,
		Prio:   s.Prio,
		stamp:  s.stamp,
	}

}

func (s *Session) Userid() string {

	return s.userid

}

func (s *Session) DataSessionLeft(state string) *DataSession {

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return &DataSession{
		Type:   "Left",
		Id:     s.Id,
		Status: state,
	}

}

func (s *Session) DataSessionJoined() *DataSession {

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return &DataSession{
		Type:   "Joined",
		Id:     s.Id,
		Userid: s.userid,
		Ua:     s.Ua,
		Prio:   s.Prio,
	}

}

func (s *Session) DataSessionStatus() *DataSession {

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return &DataSession{
		Type:   "Status",
		Id:     s.Id,
		Userid: s.userid,
		Status: s.Status,
		Rev:    s.UpdateRev,
		Prio:   s.Prio,
	}

}

type SessionUpdate struct {
	Id     string
	Types  []string
	Roomid string
	Ua     string
	Prio   int
	Status interface{}
}

type SessionToken struct {
	Id     string // Public session id.
	Sid    string // Secret session id.
	Userid string // Public user id.
	Nonce  string `json:"Nonce,omitempty"` // User autentication nonce.
}

func init() {
	// Create nonce generator.
	sessionNonces = securecookie.New(securecookie.GenerateRandomKey(64), nil)
	sessionNonces.MaxAge(60)
}