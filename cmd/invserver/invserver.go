// Copyright 2019-2020 Demian Harvill
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Command line to run MServiceInventory GRPC server.
package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/gaterace/inventory/pkg/muxhandler"
	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/gaterace/inventory/pkg/invauth"
	"github.com/gaterace/inventory/pkg/invservice"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"

	"database/sql"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	cli := &cli{}

	cmd := &cobra.Command{
		Use:     "invserver",
		PreRunE: cli.setupConfig,
		RunE:    cli.run,
	}

	if err := setupFlags(cmd); err != nil {
		fmt.Println(err)
		os.Exit(1)

	}

	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}

type cli struct {
	cfg cfg
}

type cfg struct {
	InvConf     string
	LogFile     string
	CertFile    string
	KeyFile     string
	Tls         bool
	Port        int
	RestPort    int
	DbUser      string
	DbPwd       string
	DbTransport string
	JwtPubFile  string
	CorsOrigin  string
}

func setupFlags(cmd *cobra.Command) error {

	cmd.Flags().String("inv_conf", "conf.yaml", "Path to inventory config file.")
	cmd.Flags().String("log_file", "", "Path to log file.")
	cmd.Flags().String("cert_file", "", "Path to certificate file.")
	cmd.Flags().String("key_file", "", "Path to certificate key file.")
	cmd.Flags().Bool("tls", false, "Use tls for connection.")
	cmd.Flags().Int("port", 50055, "Port for RPC connections")
	cmd.Flags().Int("rest_port", 0, "Port for REST connections")
	cmd.Flags().String("db_user", "", "Database user name.")
	cmd.Flags().String("db_pwd", "", "Database user password.")
	cmd.Flags().String("db_transport", "", "Database transport string.")
	cmd.Flags().String("jwt_pub_file", "", "Path to JWT public certificate.")
	cmd.Flags().String("cors_origin", "", "Cross origin sites for REST.")

	return viper.BindPFlags(cmd.Flags())
}

func (c *cli) setupConfig(cmd *cobra.Command, args []string) error {
	var err error

	viper.AutomaticEnv()

	configFile := viper.GetString("inv_conf")
	if err != nil {
		return err
	}
	viper.SetConfigFile(configFile)

	if err = viper.ReadInConfig(); err != nil {
		// it's ok if config file doesn't exist
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	c.cfg.LogFile = viper.GetString("log_file")
	c.cfg.CertFile = viper.GetString("cert_file")
	c.cfg.KeyFile = viper.GetString("key_file")
	c.cfg.Tls = viper.GetBool("tls")
	c.cfg.Port = viper.GetInt("port")
	c.cfg.RestPort = viper.GetInt("rest-port")
	c.cfg.DbUser = viper.GetString("db_user")
	c.cfg.DbPwd = viper.GetString("db_pwd")
	c.cfg.DbTransport = viper.GetString("db_transport")
	c.cfg.JwtPubFile = viper.GetString("jwt_pub_file")
	c.cfg.CorsOrigin = viper.GetString("cors_origin")

	return nil
}

func (c *cli) run(cmd *cobra.Command, args []string) error {
	// TODO

	log_file := c.cfg.LogFile
	cert_file := c.cfg.CertFile
	key_file := c.cfg.KeyFile
	tls := c.cfg.Tls
	port := c.cfg.Port
	rest_port := c.cfg.RestPort
	db_user := c.cfg.DbUser
	db_pwd := c.cfg.DbPwd
	db_transport := c.cfg.DbTransport
	jwt_pub_file := c.cfg.JwtPubFile
	cors_origin := c.cfg.CorsOrigin

	var logWriter io.Writer

	if log_file == "" {
		logWriter = os.Stderr
	} else {
		logfile, _ := os.OpenFile(log_file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		defer logfile.Close()
		logWriter = logfile
	}
	logger := log.NewLogfmtLogger(log.NewSyncWriter(logWriter))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)

	level.Info(logger).Log("log_file", log_file)
	level.Info(logger).Log("cert_file", cert_file)
	level.Info(logger).Log("key_file", key_file)
	level.Info(logger).Log("tls", tls)
	level.Info(logger).Log("port", port)
	level.Info(logger).Log("rest_port", rest_port)

	level.Info(logger).Log("db_user", db_user)
	level.Info(logger).Log("db_transport", db_transport)
	level.Info(logger).Log("jwt_pub_file", jwt_pub_file)
	level.Info(logger).Log("cors_origin", cors_origin)

	listen_port := ":" + strconv.Itoa(int(port))
	// fmt.Println(listen_port)

	lis, err := net.Listen("tcp", listen_port)
	if err != nil {
		level.Error(logger).Log("what", "net.listen", "error", err)
		os.Exit(1)
	}

	var opts []grpc.ServerOption
	if tls {
		creds, err := credentials.NewServerTLSFromFile(cert_file, key_file)
		if err != nil {
			grpclog.Fatalf("Failed to generate credentials %v", err)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}

	s := grpc.NewServer(opts...)

	invService := invservice.NewInvService()

	sqlDb, err := SetupDatabaseConnections(db_user, db_pwd, db_transport)
	if err != nil {
		level.Error(logger).Log("what", "SetupDatabaseConnections", "error", err)
		os.Exit(1)
	}

	invService.SetLogger(logger)
	invService.SetDatabaseConnection(sqlDb)

	// wire up the authorization middleware

	invAuth := invauth.NewInvAuth(invService)

	invAuth.SetLogger(logger)

	invAuth.SetPublicKey(jwt_pub_file)
	invAuth.SetDatabaseConnection(sqlDb)
	err = invAuth.NewApiServer(s)
	if err != nil {
		level.Error(logger).Log("what", "NewApiServer", "error", err)
		os.Exit(1)
	}

	var srv *http.Server

	if rest_port > 0 {
		r := mux.NewRouter()
		mh := muxhandler.NewMuxHandler(invAuth, r)
		mh.AddRoutes()

		var handler http.Handler
		if cors_origin != "" {
			origins := strings.Split(cors_origin, ",")
			c := cors.New(cors.Options{
				AllowedOrigins:   origins,
				AllowCredentials: true,
				AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
				AllowedHeaders:   []string{"*"},
				// Debug: true,
			})
			level.Info(logger).Log("msg", "using cors")
			handler = c.Handler(r)
		} else {
			handler = r
		}

		addrString := fmt.Sprintf(":%d", rest_port)
		srv = &http.Server{
			Addr:         addrString,
			WriteTimeout: time.Second * 15,
			ReadTimeout:  time.Second * 15,
			Handler:      handler, // Pass our instance of gorilla/mux in.
		}

		go func() {
			level.Info(logger).Log("msg", "starting http server")
			if tls {
				err = srv.ListenAndServeTLS(cert_file, key_file)
			} else {
				err = srv.ListenAndServe()
			}
			if err != nil {
				level.Error(logger).Log("what", "ListenAndServe", "error", err)
			}
		}()
	}

	go func() {
		level.Info(logger).Log("msg", "starting grpc server")

		err = s.Serve(lis)
		if err != nil {
			level.Error(logger).Log("what", "Serve", "error", err)
		}
	}()

	ch := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(ch, os.Interrupt)

	// Block until we receive our signal.
	<-ch

	s.GracefulStop()
	level.Info(logger).Log("msg", "shutting down grpc server")

	if srv != nil {
		// Create a deadline to wait for.
		wait, _ := time.ParseDuration("15s")

		ctx, cancel := context.WithTimeout(context.Background(), wait)
		defer cancel()
		// Doesn't block if no connections, but will otherwise wait
		// until the timeout deadline.
		srv.Shutdown(ctx)

		level.Info(logger).Log("msg", "shutting down http server")
	}

	// os.Exit(0)

	return nil
}

func SetupDatabaseConnections(db_user string, db_pwd string, db_transport string) (*sql.DB, error) {
	var sqlDb *sql.DB
	endpoint := db_user + ":" + db_pwd + "@" + db_transport + "/inventory"

	var err error
	sqlDb, err = sql.Open("mysql", endpoint)
	if err == nil {
		err = sqlDb.Ping()
		if err != nil {
			sqlDb = nil
		}

	}

	return sqlDb, err
}
