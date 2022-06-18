package main

// Forward from local port 8001 to remote port 9999, by ssh tunnel

import (
	"net/http"
	"time"

	server "upterm-web/server"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	flagListenAddress = ""
	flagAdvisedUri    = ""
)

type httpFunc func(w http.ResponseWriter, r *http.Request)

func main() {
	// https://my.uptermd-local.corvo.fun/auth?payload=c3NoOi8vT0VxNkk2cERkR1VhNmZSejcyWXM6QDEyNy4wLjAuMToyMjIyP3dlYlBvcnQ9Mzg5MDc=
	var log = logrus.New().WithField("app", "upterm-web")

	fileServer := http.FileServer(http.Dir("./static"))
	noCacheMiddleware := func(fs http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-cache, no-store, no-transform, must-revalidate, private, max-age=0")
			w.Header().Set("Expires", time.Unix(0, 0).Format(time.RFC1123))
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("X-Accel-Expires", "0")
			fs.ServeHTTP(w, r)
		}
	}

	var rootCmd = &cobra.Command{
		Use: "",
		Run: func(cmd *cobra.Command, args []string) {
			us := server.InitUptermWebServer(flagAdvisedUri, log)
			http.HandleFunc("/auth", us.Auth)
			http.Handle("/.upterm/", http.StripPrefix("/.upterm/", noCacheMiddleware(fileServer)))
			http.HandleFunc("/", us.VSCodeConn)
			log.Println("Listen on ", flagListenAddress)
			http.ListenAndServe(flagListenAddress, nil)
		},
	}

	rootCmd.PersistentFlags().StringVar(&flagListenAddress, "listen-address", "127.0.0.1:8001", "")
	rootCmd.PersistentFlags().StringVar(&flagAdvisedUri, "advised-domain", "http://my.uptermd-local.corvo.fun:8001", "")

	rootCmd.Execute()
}
