package network

import (
	"fmt"
	"io"
	"net"
	"os"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func getSFTPConnection(host, user, password, port string) (*sftp.Client, error) {
	var auths []ssh.AuthMethod

	// Use password authentication if provided
	if password != "" {
		auths = append(auths, ssh.Password(password))
	}

	// Initialize client configuration
	config := &ssh.ClientConfig{
		User: user,
		Auth: auths,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	address := fmt.Sprintf("%s:%s", host, port)

	// Connect to server
	connection, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return nil, err
	}

	defer connection.Close()

	// Create new SFTP client
	client, err := sftp.NewClient(connection)
	if err != nil {
		return nil, err
	}

	defer client.Close()

	return client, nil
}

// Upload file SFTP server
func UploadFileToSFTP(host, user, password, port, srcFile, dstFile string) (int64, error) {
	sc, err := getSFTPConnection(host, user, password, port)
	if err != nil {
		return 0, err
	}

	// Create destination file
	df, err := sc.Create(dstFile)
	if err != nil {
		return 0, err
	}

	// Create source file
	sf, err := os.Open(srcFile)
	if err != nil {
		return 0, nil
	}

	// Copy source file to destination file
	bytes, err := io.Copy(df, sf)
	if err != nil {
		return 0, err
	}

	return bytes, nil
}

func ReadFileFromSFTP(host, user, password, port, srcFile, dstFile string) (int64, error) {
	sc, err := getSFTPConnection(host, user, password, port)
	if err != nil {
		return 0, err
	}

	// Open source file in SFTP
	sf, err := sc.Open(srcFile)
	if err != nil {
		return 0, err
	}
	defer sf.Close()

	// Create destination file
	df, err := os.Create(dstFile)
	if err != nil {
		return 0, err
	}
	defer df.Close()

	// Copy source file from SFTP to destination file
	bytes, err := io.Copy(df, sf)
	if err != nil {
		return 0, err
	}

	return bytes, nil
}
