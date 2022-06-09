package server

import (
	"net/url"
	"testing"
)

func assertEqual(t *testing.T, a, b interface{}) {
	if a != b {
		t.Errorf("Not Equal. %d %d", a, b)
	}
}
func Test_parseRequestUri(t *testing.T) {
	//
	u, err := url.ParseRequestURI("http://127.0.0.1:8001/auth?username=7XolyNecRiVnM1yS0EB0&advisedUri=ssh://uptermd-gz.corvo.fun:2222&webPort=9922")
	if err != nil {
		t.Error(err)
	}
	assertEqual(t, u.Query().Get("username"), "7XolyNecRiVnM1yS0EB0")
	assertEqual(t, u.Query().Get("advisedUri"), "ssh://uptermd-gz.corvo.fun:2222")
	assertEqual(t, u.Query().Get("webPort"), "9922")
}

func Test_parseAdvisedUri(t *testing.T) {
	TEST_CASE := []struct {
		advisedUri string
		hostname   string
		port       string
	}{
		{advisedUri: "ssh://uptermd-gz.corvo.fun:2222", hostname: "uptermd-gz.corvo.fun", port: "2222"},
		{advisedUri: "ssh://uptermd-gz.corvo.fun", hostname: "uptermd-gz.corvo.fun", port: "22"},
	}

	for _, test := range TEST_CASE {
		hostname, port, err := parsedAdvisedUri(test.advisedUri)
		if err != nil {
			t.Error(err)
		}
		assertEqual(t, port, test.port)
		assertEqual(t, hostname, test.hostname)
	}
}

func Test_getHostForCookie(t *testing.T) {
	TEST_CASE := []struct {
		domain string
		host   string
	}{
		{
			domain: "my.uptermd-local.corvo.fun:8081",
			host:   "my.uptermd-local.corvo.fun",
		},
		{
			domain: "my.uptermd-local.corvo.fun",
			host:   "my.uptermd-local.corvo.fun",
		},
	}
	for _, test := range TEST_CASE {
		assertEqual(t, getHostForCookie(test.domain), test.host)
	}
}

func Test_getCookieKeyFromDomain(t *testing.T) {
	TEST_CASE := []struct {
		domain string
		port   string
		key    string
	}{
		{
			domain: "9923-954f8ae5c789df92af151052200bf971.uptermd-local.corvo.fun:8001",
			port:   "9923",
			key:    "954f8ae5c789df92af151052200bf971",
		},
		{
			domain: "3333-954f8ae5c789df92af151052200bf971.uptermd-local.corvo.fun:8001",
			port:   "3333",
			key:    "954f8ae5c789df92af151052200bf971",
		},
	}
	for _, test := range TEST_CASE {
		port, key := getCookieKeyFromDomain((test.domain))
		assertEqual(t, port, test.port)
		assertEqual(t, key, test.key)
	}

}
