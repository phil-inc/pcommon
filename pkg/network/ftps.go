package network

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/jlaffaye/ftp"
)

func getFTPSConnection(hostname, username, password, port string) (*ftp.ServerConn, error) {
	cfg := tls.Config{
		InsecureSkipVerify: true,
		ServerName:         hostname,
		ClientSessionCache: tls.NewLRUClientSessionCache(32),
	}

	fullHost := fmt.Sprintf("%s:%s", hostname, port)

	conn, err := ftp.Dial(fullHost, ftp.DialWithExplicitTLS(&cfg))
	if err != nil {
		return nil, err
	}

	err = conn.Login(username, password)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func UploadFileToFTPS(hostname, username, password, dstFile, port, content string) error {
	conn, err := getFTPSConnection(hostname, username, password, port)
	if err != nil {
		return err
	}

	defer conn.Quit()

	err = conn.Stor(dstFile, strings.NewReader(content))
	if err != nil {
		return err
	}

	return nil
}

func ReadFileFromFTPS(hostname, username, password, srcFile, port string) (string, error) {
	conn, err := getFTPSConnection(hostname, username, password, port)
	if err != nil {
		return "", err
	}

	defer conn.Quit()

	res, err := conn.Retr(srcFile)
	if err != nil {
		return "", err
	}

	defer res.Close()

	b := new(bytes.Buffer)
	b.ReadFrom(res)

	return b.String(), nil
}
