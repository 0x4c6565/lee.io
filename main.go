package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/0x4c6565/lee.io/pkg/server"
	"github.com/0x4c6565/lee.io/pkg/tool"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Info().Msg("lee.io Starting")

	config, err := InitConfig()
	if err != nil {
		log.Fatal().Msgf("failed to initialise config: %s", err)
	}

	if config.Debug {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
		log.Logger = log.With().Caller().Stack().Logger()
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-stop
		log.Info().Msg("Caught signal, shutting down..")
		cancel()
	}()

	conn, err := sqlx.Connect("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", config.DB.User, config.DB.Password, config.DB.Host, config.DB.Port, config.DB.DB))
	if err != nil {
		log.Fatal().Msgf("failed to connect to database: %s", err)
	}

	serverOpts := server.ServerOptions{
		Initialise: config.Initialise,
	}

	server := server.NewServer(serverOpts).WithTools(
		tool.NewWhois(),
		tool.NewIP(),
		tool.NewPort(),
		tool.NewSelfSigned(),
		tool.NewKeypair(),
		tool.NewSubnet(),
		tool.NewMAC(tool.NewMACOUIMySQLRepository(conn)),
		tool.NewBGP(tool.NewBGPRouteMySQLRepository(conn)),
		tool.NewUUID(),
		tool.NewGeoIP(tool.NewGeoIP2FileSystemReader(config.GeoIP.DatabasePath)),
		tool.NewPassword(),
		tool.NewSSLDecode(),
		tool.NewEUI64(),
		tool.NewSSL(),
		tool.NewProjectName(),
	)

	err = server.Start(ctx)
	if err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("Server failed")
	}

	log.Info().Msg("lee.io shutdown")
}
