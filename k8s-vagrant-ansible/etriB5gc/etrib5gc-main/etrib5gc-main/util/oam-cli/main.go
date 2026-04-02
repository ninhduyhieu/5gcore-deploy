package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/reogac/utils"
	"github.com/reogac/utils/oam/client"
	"github.com/urfave/cli/v3"
	"io/ioutil"
	"os"
)

var flags = []cli.Flag{
	&cli.StringFlag{
		Name:  "cert",
		Usage: "Load certificate from `FILE`",
	},
	&cli.StringFlag{
		Name:  "pem",
		Usage: "Load CA pem from `FILE`",
	},
	&cli.StringFlag{
		Name:    "key",
		Aliases: []string{"k"},
		Usage:   "Load private key from `FILE`",
	},
}

func main() {
	cmd := cli.Command{
		Name:   "oam-cli",
		Usage:  "OAM client",
		Flags:  flags,
		Action: action,
	}

	cmd.Run(context.Background(), os.Args)
}

func loadCertificate(ctx *cli.Command) (subject string, cert tls.Certificate, caPool *x509.CertPool, err error) {
	certFile := ctx.String("cert")
	keyFile := ctx.String("key")
	if cert, err = tls.LoadX509KeyPair(certFile, keyFile); err != nil {
		err = utils.WrapError("Load certificate", err)
		return
	}
	var x509Cert *x509.Certificate
	if x509Cert, err = x509.ParseCertificate(cert.Certificate[0]); err != nil {
		err = utils.WrapError("Parse certificate", err)
		return
	}
	subject = x509Cert.Subject.CommonName

	pemFile := ctx.String("pem")
	var pem []byte
	if pem, err = ioutil.ReadFile(pemFile); err != nil {
		err = utils.WrapError("Load CA root", err)
		return
	}

	caPool = x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(pem) {
		err = fmt.Errorf("Failed to add CA certificate to pool")
	}
	return
}

func action(ctx context.Context, c *cli.Command) error {
	//load certificate
	if _, cert, caPool, err := loadCertificate(c); err != nil {
		fmt.Printf("Fail to load certificate and root CA pem: %+v\n", err)
		client.NewClient(nil, nil).Run()
	} else {
		client.NewClient(&cert, caPool).Run()
	}
	return nil
}
