package app

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"etrib5gc/logctx"
	"etrib5gc/mesh"
	"fmt"
	"github.com/reogac/utils"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

type Service interface {
	Run() error
	Stop()
	Init(*Context) error
}

type Context struct {
	context.Context
	*cli.Command
	ConfigData  []byte
	Cert        tls.Certificate
	CaPool      *x509.CertPool
	CertName    string
	meshOptions *mesh.Options //service mesh initialization options
}

func (ctx *Context) SetMeshOptions(opts *mesh.Options) {
	ctx.meshOptions = opts
}

type AppContext struct {
	name                         string
	service                      Service
	logFile                      *os.File
	agentFailure, serviceFailure bool
}

var commonFlags = []cli.Flag{
	&cli.StringFlag{
		Name:     "config",
		Aliases:  []string{"c"},
		Usage:    "Load configuration from `FILE`",
		Required: true,
	},
	&cli.StringFlag{
		Name:    "log",
		Aliases: []string{"l"},
		Usage:   "Output logs to `FILE`",
	},
	&cli.StringFlag{
		Name:    "logLevel",
		Aliases: []string{"v"},
		Usage:   "Log level: trace, debug, info, warn, error, fatal, panic",
	},
}

var nfFlags = []cli.Flag{
	&cli.StringFlag{
		Name:    "gw",
		Usage:   "Gateway name for certificate verification",
		Sources: cli.EnvVars("GATEWAY_NAME"),
	},
}

func RunNfApp(name, usage string, flags []cli.Flag, service Service) error {
	flags = append(flags, nfFlags...)
	return RunApp(name, usage, flags, service)
}

func RunApp(name, usage string, flags []cli.Flag, service Service) error {
	appCtx := &AppContext{
		service: service,
		name:    strings.ToLower(name),
	}

	cmd := cli.Command{
		Name:   name,
		Usage:  usage,
		Flags:  append(append(commonFlags, certFlags...), flags...),
		Action: appCtx.action,
	}

	return cmd.Run(context.Background(), os.Args)
}

func (ctx *AppContext) killWait(sigCh <-chan os.Signal, infoCh <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	select {
	case <-sigCh:
		fmt.Println("System interrupted")
	case <-infoCh:
	}

	//stop service
	if !ctx.serviceFailure {
		ctx.service.Stop()
	}

	//stop mesh agent
	if !ctx.agentFailure {
		mesh.Terminate()
	}
}

func (app *AppContext) action(cmdCtx context.Context, c *cli.Command) error {
	var err error
	//open log file
	logFileName := c.String("log")
	if len(logFileName) > 0 {
		if app.logFile, err = os.OpenFile(logFileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); err != nil {
			return utils.WrapError("Create log file", err)
		}
		defer app.logFile.Close()
	}

	//init logger
	logctx.Init(app.logFile, c.String("logLevel"), logrus.Fields{
		"app": app.name,
	})

	//read config
	configFileName := c.String("config")
	var buf []byte
	if buf, err = ioutil.ReadFile(configFileName); err != nil {
		return utils.WrapError("Read config file", err)
	}

	//init service
	ctx := &Context{
		Context:    cmdCtx,
		Command:    c,
		ConfigData: buf,
	}

	//load certificate
	if ctx.CertName, ctx.Cert, ctx.CaPool, err = loadCertificate(c); err != nil {
		logctx.WithFields(nil).Warnf("Fail to load certificate: %+v", err)
	}

	if err = app.service.Init(ctx); err != nil {
		return utils.WrapError("Service initialization", err)
	}

	//create go routine to wait for system interuption signals
	var wg sync.WaitGroup
	infoCh := make(chan struct{}, 1)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	wg.Add(1)
	defer wg.Wait()
	go app.killWait(sigCh, infoCh, &wg)

	if opts := ctx.meshOptions; opts != nil {
		opts.Cert = ctx.Cert
		opts.CaPool = ctx.CaPool
		opts.CertName = ctx.CertName
		opts.GatewayName = ctx.String("gw")
		//init mesh agent
		if err = mesh.Init(*ctx.meshOptions); err != nil {
			app.agentFailure = true
			infoCh <- struct{}{}
			return utils.WrapError("Init mesh agent", err)
		}
	}

	//run the service
	if err = app.service.Run(); err != nil {
		app.serviceFailure = true
		infoCh <- struct{}{}
		return utils.WrapError("Start service", err)
	}

	return nil
}
