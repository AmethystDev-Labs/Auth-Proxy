package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"authproxy/internal/config"
	"authproxy/internal/logging"
	"authproxy/internal/proxy"
	"authproxy/internal/server"
	"authproxy/internal/web"
	"go.uber.org/zap"
)

func main() {
	if err := run(os.Args[1:], os.LookupEnv, os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, lookup config.LookupEnv, stdout io.Writer) error {
	cfg, err := config.Load(args, lookup)
	if err != nil {
		var helpErr *config.HelpError
		if errors.As(err, &helpErr) {
			_, _ = fmt.Fprint(stdout, helpErr.Usage)
			return nil
		}
		return err
	}

	logger, err := logging.New(logging.Config{
		Level:     cfg.LogLevel,
		Format:    cfg.LogFormat,
		AddSource: cfg.LogAddSource,
	}, stdout)
	if err != nil {
		return err
	}
	defer logger.Sync()

	handler, err := server.New(server.Options{
		Config:    cfg,
		Logger:    logger,
		LoginPage: web.MustLoginPage(),
		Assets:    web.AssetsHandler(),
		HTTPProxy: proxy.NewHTTP(cfg.Upstream, cfg.SessionCookieName, logger),
		WSProxy:   proxy.NewWebSocket(cfg.Upstream, cfg.SessionCookieName, logger),
	})
	if err != nil {
		return err
	}

	logger.Info("main", "", fmt.Sprintf("starting authproxy listen=%q upstream=%q log_level=%q log_format=%q",
		cfg.ListenAddr,
		cfg.Upstream.String(),
		cfg.LogLevel,
		cfg.LogFormat,
	),
		zap.String("listen_addr", cfg.ListenAddr),
		zap.String("upstream", cfg.Upstream.String()),
		zap.String("log_level", cfg.LogLevel),
		zap.String("log_format", cfg.LogFormat),
	)

	return http.ListenAndServe(cfg.ListenAddr, logging.Middleware(logger, handler))
}
