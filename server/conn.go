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
	AdvisedUri     string
	ttlMap         *TTLMap
	subDomain      string
	subDomainProto string
	Logger         *logrus.Entry
}

var sLog = logrus.New().WithField("package", "server")

const portSplit = "-"

type SSHConnConfig struct {
	Username      string `json:"username"`
	Hostname      string `json:"hostname"`
	Port          string `json:"port"`
	WebPort       string `json:"webport"`
	VSCodeWebPort string `json:"vscode_webport"` // constant value to verify current port
}

type sshHandler struct {
	config           *ssh.ClientConfig
	serverAddrString string
	remoteAddrString string
}

func InitUptermWebServer(advisedDomain string, logger *logrus.Entry) *UptermWebServer {
	us := &UptermWebServer{
		AdvisedUri: advisedDomain,
		Logger:     logger,
	}
	u, err := url.Parse(advisedDomain)
	if err != nil {
		logger.Fatal("Can't parse advised domain: ", advisedDomain, err)
		return nil
	}
	us.subDomainProto = u.Scheme
	us.subDomain = strings.Join(strings.Split(u.Host, ".")[1:], ".")
	us.Logger.Infof("Infer to cookie domain: %s://%s", us.subDomainProto, us.subDomain)
	us.ttlMap = NewTTLMap(1024, 6*3600) // six hours for clean
	return us
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

func getCookieKeyFromDomain(domain string) (string, string) {
	data := strings.Split(domain, ".")
	portWithKey := strings.Split(data[0], portSplit)
	if len(portWithKey) > 1 {
		return portWithKey[0], portWithKey[1]
	} else {
		return "0", portWithKey[0]
	}
}

func (cfg *SSHConnConfig) encode() string {
	cfg_byte, _ := json.Marshal(cfg)
	return base64.StdEncoding.EncodeToString(cfg_byte)
}

func createSSHConfigFromCookie(req *http.Request) (*SSHConnConfig, error) {
	// cookie set by frontend js
	var cfg SSHConnConfig
	var rawString []byte
	var err error
	if data, err := req.Cookie("vscode"); err == nil {
		rawString, err = base64.StdEncoding.DecodeString(data.Value)
		if err != nil {
			return nil, err
		}
	}
	if len(rawString) == 0 {
		// ios home app try to find cfg in the get params
		data := req.URL.Query().Get("vscode")
		if rawString, err = base64.StdEncoding.DecodeString(data); err != nil {
			fmt.Println(rawString, err)
			return nil, err
		}
	}

	if err := json.Unmarshal(rawString, &cfg); err != nil {
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
	sDe, err := base64.StdEncoding.DecodeString(req.URL.Query().Get("payload"))
	if err != nil {
		w.Write([]byte(fmt.Sprintf("Could not parse payload: %s", err)))
		return
	}
	rawURL := string(sDe)
	u, err := url.Parse(rawURL)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("Could not parse url %s with %s", rawURL, err)))
		return
	}

	cfg := SSHConnConfig{
		Username:      u.User.Username(),
		Hostname:      u.Hostname(),
		Port:          u.Port(),
		WebPort:       u.Query().Get("webPort"),
		VSCodeWebPort: u.Query().Get("webPort"),
	}
	conn, err := cfg.checkSSHCoon()
	if err != nil {
		us.Logger.Errorf("ssh.Dial failed: %s", err)
		w.Write([]byte(fmt.Sprintf("Could not connect through %s with err %s", cfg, err)))
		return
	}
	defer conn.Close()

	domain := fmt.Sprintf("%s.%s", MD5(cfg.Username), us.subDomain)
	us.ttlMap.Put(MD5(cfg.Username), cfg)
	http.Redirect(w, req, fmt.Sprintf("%s://%s?upterm=true&page=loading&vscode=%s", us.subDomainProto, domain, cfg.encode()), http.StatusTemporaryRedirect)
}

func (us *UptermWebServer) VSCodeConn(w http.ResponseWriter, req *http.Request) {
	var cfg *SSHConnConfig
	var err error
	us.Logger.Info("Request: ", req.Host, req.URL.Path)
	if req.URL.Query().Get("upterm") == "true" {
		p := "./static/index.html"
		http.ServeFile(w, req, p)
		return
	}

	cfg, _ = createSSHConfigFromCookie(req)
	port, cacheKey := getCookieKeyFromDomain(req.Host)
	if cfg == nil { // try to find cfg in the ttl map
		if data := us.ttlMap.Get(cacheKey); data != nil {
			ssh_cfg := data.(SSHConnConfig)
			cfg = &ssh_cfg
			for {
				if port != cfg.VSCodeWebPort {
					// port for share
					cfg.WebPort = port
					break
				}

				cfg = nil
				us.Logger.Error("The vscode port could not accessed without a cookie")
				break
			}
		} else {
			us.Logger.Errorf("Can't find a valid cookie in: %s", req.Host)
		}
	}

	us.Logger.Info("Do request: ", req.Host, req.URL.Path)
	if cfg == nil {
		w.Write([]byte("Could not get valid valid cfg"))
		return
	}
	conn, err := cfg.checkSSHCoon()
	if err != nil {
		us.Logger.Error("Can't connect with err: ", err)
		w.Write([]byte(fmt.Sprintf("Could not connect throught %s error: %s ", req.Host, err)))
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
