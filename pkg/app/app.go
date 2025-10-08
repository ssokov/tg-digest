package app

import (
	"botsrv/pkg/botsrv"
	"context"
	"time"

	"botsrv/pkg/db"
	"botsrv/pkg/embedlog"

	"github.com/go-pg/pg/v10"
	"github.com/go-telegram/bot"
	"github.com/labstack/echo/v4"
	"github.com/vmkteam/zenrpc/v2"
)

type Config struct {
	Database *pg.Options
	Server   struct {
		Host      string
		Port      int
		IsDevel   bool
		EnableVFS bool
	}
	Bot botsrv.Config
}

type App struct {
	embedlog.Logger
	appName string
	cfg     Config
	db      db.DB
	dbc     *pg.DB
	echo    *echo.Echo
	vtsrv   zenrpc.Server

	b  *bot.Bot
	bm *botsrv.BotManager
}

func New(appName string, verbose bool, cfg Config, db db.DB, dbc *pg.DB) *App {
	a := &App{
		appName: appName,
		cfg:     cfg,
		db:      db,
		dbc:     dbc,
		echo:    echo.New(),
	}
	a.SetStdLoggers(verbose)
	a.echo.HideBanner = true
	a.echo.HidePort = true
	a.echo.IPExtractor = echo.ExtractIPFromRealIPHeader()

	a.bm = botsrv.NewBotManager(a.Logger, a.db)

	opts := []bot.Option{bot.WithAllowedUpdates(bot.AllowedUpdates{"message", "message_reaction", "message_reaction_count", "callback_query"}),
		bot.WithDefaultHandler(a.bm.DefaultHandler)}
	b, err := bot.New(cfg.Bot.Token, opts...)
	if err != nil {
		panic(err)
	}
	a.b = b

	return a
}

// Run is a function that runs application.
func (a *App) Run() error {
	a.registerMetrics()
	a.registerHandlers()
	a.registerDebugHandlers()
	a.registerAPIHandlers()

	a.bm.RegisterBotHandlers(a.b)
	go a.b.Start(context.TODO())
	return a.runHTTPServer(a.cfg.Server.Host, a.cfg.Server.Port)
}

// Shutdown is a function that gracefully stops HTTP server.
func (a *App) Shutdown(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := a.echo.Shutdown(ctx); err != nil {
		a.Errorf("shutting down server err=%q", err)
	}
}
