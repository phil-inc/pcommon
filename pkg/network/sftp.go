package network

import (
	"fmt"
	"io"
	"net"
	"os"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func UploadFileToSFTP(host, user, password, port, srcFile, dstFile string) (int64, error) {
	// Initialize client configuration
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	address := fmt.Sprintf("%s:%s", host, port)

	return UploadFileToSFTPUsingConfig(config, address, srcFile, dstFile)
}

func UploadFileToSFTPWithAddress(user, password, address, srcFile, dstFile string) (int64, error) {
	// Initialize client configuration
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	return UploadFileToSFTPUsingConfig(config, address, srcFile, dstFile)
}

// Upload file to SFTP server
func UploadFileToSFTPUsingConfig(config *ssh.ClientConfig, address, srcFile, dstFile string) (int64, error) {
	// Connect to server
	connection, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return 0, err
	}

	defer connection.Close()

	// Create new SFTP client
	client, err := sftp.NewClient(connection)
	if err != nil {
		return 0, err
	}

	defer client.Close()

	// Create source file
	sf, err := os.Open(srcFile)
	if err != nil {
		return 0, nil
	}

	defer sf.Close()

	// Create destination file
	df, err := client.Create(dstFile)
	if err != nil {
		return 0, err
	}

	// Copy source file to destination file
	bytes, err := io.Copy(df, sf)
	if err != nil {
		return 0, err
	}

	return bytes, nil
}

// Upload file SFTP server
func UploadFileToSFTPUsingPrivateKey(user, privateKey, address, srcFile, dstFile string) (int64, error) {
	signer, _ := ssh.ParsePrivateKey([]byte(privateKey))
	config := &ssh.ClientConfig{
		User: user,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
	}

	// oysterpoint mavens requires ssh-rsa
	config.HostKeyAlgorithms = append(config.HostKeyAlgorithms, ssh.KeyAlgoRSA)

	return UploadFileToSFTPUsingConfig(config, address, srcFile, dstFile)
}

// Read file from SFTP server to source directory
func ReadFileFromSFTP(host, user, password, port, srcFile, dstFile string) (int64, error) {
	// Initialize client configuration
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	address := fmt.Sprintf("%s:%s", host, port)

	return ReadFileFromSFTPUsingConfig(config, address, srcFile, dstFile)
}

func ReadFileFromSFTPWithAddress(user, password, address, srcFile, dstFile string) (int64, error) {
	// Initialize client configuration
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	return ReadFileFromSFTPUsingConfig(config, address, srcFile, dstFile)
}

func ReadFileFromSFTPUsingConfig(config *ssh.ClientConfig, address, srcFile, dstFile string) (int64, error) {
	// Connect to server
	connection, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return 0, err
	}

	defer connection.Close()

	// Create new SFTP client
	client, err := sftp.NewClient(connection)
	if err != nil {
		return 0, err
	}

	defer client.Close()

	// Open source file in SFTP
	sf, err := client.Open(srcFile)
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

func ReadFileFromSFTPUsingPrivateKey(user, privateKey, address, srcFile, dstFile string) (int64, error) {
	signer, _ := ssh.ParsePrivateKey([]byte(privateKey))
	config := &ssh.ClientConfig{
		User: user,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
	}

	return ReadFileFromSFTPUsingConfig(config, address, srcFile, dstFile)
}

func ReadFileFromLegacySFTP(usr, password, address, srcFile, dstFile string) (int64, error) {
	config := &ssh.ClientConfig{
		User: usr,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
	}

	config.Ciphers = append(config.Ciphers, "3des-cbc")

	return ReadFileFromSFTPUsingConfig(config, address, srcFile, dstFile)
}
