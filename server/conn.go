package server

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

type UptermWebServer struct {
	ListenAddr     string
	AdvisedDomain  string
	SubDomain      string
	SubDomainProto string
	Logger         *logrus.Entry
}

var sLog = logrus.New().WithField("package", "server")

type SSHConnConfig struct {
	Username string `json:"username"`
	Hostname string `json:"hostname"`
	Port     string `json:"port"`
	WebPort  string `json:"webport"`
}

type sshHandler struct {
	config           *ssh.ClientConfig
	serverAddrString string
	remoteAddrString string
}

// MD5 hashes using md5 algorithm
func MD5(text string) string {
	data := []byte(text)
	return fmt.Sprintf("%x", md5.Sum(data))
}
func parsedAdvisedUri(advisedUri string) (string, string, error) {
	u, err := url.Parse(advisedUri)
	if err != nil {
		return "", "", err
	}
	if u.Port() == "" {
		return u.Hostname(), "22", nil
	} else {
		return u.Hostname(), u.Port(), nil
	}
}

func getHostForCookie(domain string) string {
	data := strings.Split(domain, ":")
	if len(data) > 0 {
		return data[0]
	}
	sLog.Errorf("Could not get host from %s", domain)
	return domain
}

func (cfg *SSHConnConfig) createCookie(w http.ResponseWriter, cookieDomain, realDomain string) {
	cfg_byte, _ := json.Marshal(cfg)
	cookie := http.Cookie{
		Name:     MD5(realDomain),
		Value:    base64.StdEncoding.EncodeToString(cfg_byte),
		Domain:   getHostForCookie(cookieDomain),
		Path:     "/",
		MaxAge:   10 * 24 * 3600,
		SameSite: http.SameSiteLaxMode,
	}
	sLog.Debug("Create cookie for %s %+v", realDomain, cookie)
	http.SetCookie(w, &cookie)
}

func createSSHConfig(req *http.Request) (*SSHConnConfig, error) {
	data, err := req.Cookie(MD5(req.Host))
	sLog.Debug("Want get cookie ", MD5(req.Host))
	if err != nil {
		return nil, err
	}
	var cfg SSHConnConfig
	val, err := base64.StdEncoding.DecodeString(data.Value)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(val, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// check if ssh connection is alive
func (cfg *SSHConnConfig) checkSSHCoon() (net.Conn, error) {
	addr := fmt.Sprintf("%s:%s", cfg.Hostname, cfg.Port)

	sshConn, err := ssh.Dial("tcp", addr, &ssh.ClientConfig{
		User: cfg.Username,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	})

	if err != nil {
		sLog.Printf("ssh.Dial ssh server failed: %s", err)
		return nil, err
	}
	conn, err := sshConn.Dial("tcp", fmt.Sprintf("localhost:%s", cfg.WebPort))
	if err != nil {
		sLog.Printf("ssh.Dial vscode web failed: %s", err)
		return nil, err
	}
	return conn, nil
}

func (us *UptermWebServer) Auth(w http.ResponseWriter, req *http.Request) {
	hostname, port, err := parsedAdvisedUri(req.URL.Query().Get("advisedUri"))
	if err != nil {
		w.Write([]byte(fmt.Sprintf("Could not parse %s", err)))
		return
	}
	cfg := SSHConnConfig{
		Username: req.URL.Query().Get("username"),
		Hostname: hostname,
		Port:     port,
		WebPort:  req.URL.Query().Get("webPort"),
	}
	conn, err := cfg.checkSSHCoon()
	if err != nil {
		us.Logger.Errorf("ssh.Dial failed: %s", err)
		w.Write([]byte(fmt.Sprintf("Could not connect through %s with err %s", cfg, err)))
		return
	}
	defer conn.Close()

	domain := fmt.Sprintf("%s-%s.%s", cfg.WebPort, MD5(cfg.Username), us.SubDomain)
	cfg.createCookie(w, us.SubDomain, domain)
	http.Redirect(w, req, fmt.Sprintf("%s://%s/.upterm/loading", us.SubDomainProto, domain), http.StatusTemporaryRedirect)
}

func (us *UptermWebServer) VSCodeConn(w http.ResponseWriter, req *http.Request) {
	cfg, err := createSSHConfig(req)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("Could not parse config from cookie %s ", err)))
		return
	}
	conn, err := cfg.checkSSHCoon()
	if err != nil {
		us.Logger.Error("Can't connect with err: ", err)
		w.Write([]byte(fmt.Sprintf("Could not connect throught %s with error: %s ", cfg, err)))
		return
	}

	client, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		us.Logger.Error("hijacker failed with err: ", err)
		w.Write([]byte(fmt.Sprintf("hijacker failed: %s", err)))
		return
	}

	// copy http header to the socket, bootstrap.
	data := req.Clone(req.Context())
	data.Write(conn)

	go func() {
		io.Copy(client, conn)
		client.Close()
		conn.Close()
	}()
	go func() {
		io.Copy(conn, client)
		client.Close()
		conn.Close()
	}()
}
