// Copyright 2019 Demian Harvill
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
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"io"
	"net"
	"os"
	"strconv"

	"github.com/gaterace/inventory/pkg/invauth"
	"github.com/gaterace/inventory/pkg/invservice"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"

	"github.com/kylelemons/go-gypsy/yaml"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	configPath := os.Getenv("INV_CONF")
	if configPath == "" {
		configPath = "conf.yaml"
	}

	config, err := yaml.ReadFile(configPath)
	if err != nil {
		fmt.Printf("configuration not found: " + configPath)
		os.Exit(1)
	}

	log_file, _ := config.Get("log_file")
	cert_file, _ := config.Get("cert_file")
	key_file, _ := config.Get("key_file")
	tls, _ := config.GetBool("tls")
	port, _ := config.GetInt("port")
	db_user, _ := config.Get("db_user")
	db_pwd, _ := config.Get("db_pwd")
	db_transport, _ := config.Get("db_transport")
	jwt_pub_file, _ := config.Get("jwt_pub_file")

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

	level.Info(logger).Log("db_user", db_user)
	level.Info(logger).Log("db_transport", db_transport)
	level.Info(logger).Log("jwt_pub_file", jwt_pub_file)


	if port == 0 {
		port = 50052
	}

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

	level.Info(logger).Log("msg", "starting grpc server")

	err = s.Serve(lis)
	if err != nil {
		level.Error(logger).Log("what", "Serve", "error", err)
	}

	level.Info(logger).Log("msg", "shutting down grpc server")

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
