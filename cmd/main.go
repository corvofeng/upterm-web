package main

// Forward from local port 8001 to remote port 9999, by ssh tunnel

import (
	"net/http"

	server "upterm-web/server"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	flagListenAddress  = ""
	flagAdvisedDomain  = ""
	flagSubDomain      = ""
	flagSubDomainProto = ""
)

func main() {
	// http://my.uptermd-local.corvo.fun:8001/auth?username=7gAgeaDVTbGYuUvb5lIh&hostname=uptermd-gz.corvo.fun&port=2222&webport=9922
	// Setup SSH config (type *ssh.ClientConfig)
	var log = logrus.New().WithField("app", "upterm-web")
	var rootCmd = &cobra.Command{
		Use: "",
		Run: func(cmd *cobra.Command, args []string) {
			us := server.UptermWebServer{
				AdvisedDomain:  flagAdvisedDomain,
				SubDomain:      flagSubDomain,
				SubDomainProto: flagSubDomainProto,
				Logger:         log,
			}
			http.HandleFunc("/auth", us.Auth)
			http.HandleFunc("/", us.VSCodeConn)
			http.Handle("/.upterm/", http.StripPrefix("/.upterm/", http.FileServer(http.Dir("./static"))))
			log.Println("Listen on ", flagListenAddress)
			http.ListenAndServe(flagListenAddress, nil)
		},
	}

	rootCmd.PersistentFlags().StringVar(&flagListenAddress, "listen-address", "0.0.0.0:8001", "")
	rootCmd.PersistentFlags().StringVar(&flagAdvisedDomain, "advised-domain", "my.uptermd-local.corvo.fun:8001", "")
	rootCmd.PersistentFlags().StringVar(&flagSubDomain, "sub-domain", "uptermd-local.corvo.fun:8001", "")
	rootCmd.PersistentFlags().StringVar(&flagSubDomainProto, "sub-domain-proto", "http", "")

	rootCmd.Execute()
}
