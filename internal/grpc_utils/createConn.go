package grpc_utils

import (
	"ActQABot/conf"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"os"
	"path/filepath"
)

var NewGRPCConn = dialGRPC

func dialGRPC(host conf.Host) (grpc.ClientConnInterface, error) {
	var creds credentials.TransportCredentials
	if host.TlsCert != nil || host.ClientCert != nil {
		tlsCfg := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}

		if host.TlsCert != nil {
			certPath, err := filepath.Abs(*host.TlsCert)
			if err != nil {
				return nil, err
			}
			caCert, err := os.ReadFile(certPath)
			if err != nil {
				return nil, err
			}

			certPool := x509.NewCertPool()
			if !certPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to append CA cert from %s", certPath)
			}
			tlsCfg.RootCAs = certPool
		}

		if host.ClientCert != nil && host.ClientKey != nil {
			clientCertPath, err := filepath.Abs(*host.ClientCert)
			if err != nil {
				return nil, err
			}
			clientKeyPath, err := filepath.Abs(*host.ClientKey)
			if err != nil {
				return nil, err
			}

			cert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load client certificate: %w", err)
			}
			tlsCfg.Certificates = []tls.Certificate{cert}
		}

		if host.InsecureSkipVerify != nil && *host.InsecureSkipVerify {
			tlsCfg.InsecureSkipVerify = true
		}

		creds = credentials.NewTLS(tlsCfg)
	} else {
		creds = insecure.NewCredentials()
	}

	conn, err := grpc.NewClient(
		host.Address,
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %w", host.Address, err)
	}
	return conn, nil
}
