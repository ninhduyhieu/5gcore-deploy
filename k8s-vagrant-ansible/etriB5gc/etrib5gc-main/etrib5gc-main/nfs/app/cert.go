package app

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/reogac/utils"
	"github.com/urfave/cli/v3"
	"io/ioutil"
)

var certFlags = []cli.Flag{
	&cli.StringFlag{
		Name:  "cert",
		Usage: "Load certificate from `FILE`",
	},
	&cli.StringFlag{
		Name:    "key",
		Aliases: []string{"k"},
		Usage:   "Load private key from `FILE`",
	},
	&cli.StringFlag{
		Name:  "pem",
		Usage: "Load CA pem from `FILE`",
	},
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
