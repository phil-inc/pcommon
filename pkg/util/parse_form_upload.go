package util

import (
	"bufio"
	"errors"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/gabriel-vasile/mimetype"
)

type UploadedFile struct {
	FileName    string
	FileContent []byte
}

func ParseImageUpload(r *http.Request) ([]*UploadedFile, error) {
	files, err := ParseFormRequest(r)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if !IsImageUpload(file.FileContent) {
			return nil, errors.New("invalid mimetype")
		}
	}
	return files, nil
}

func ParseFormRequest(r *http.Request) ([]*UploadedFile, error) {
	// parse request
	const _24K = (1 << 10) * 24 //read in 24K bytes chunk
	if err := r.ParseMultipartForm(_24K); nil != err {
		return nil, errors.New("invalid file data: can't parse multipart form")
	}

	uploadedFiles := make([]*UploadedFile, 0)
	for _, fheaders := range r.MultipartForm.File {
		for _, hdr := range fheaders {
			fileContent, err := processFile(hdr)
			if err != nil {
				return nil, err
			}

			file := &UploadedFile{FileName: hdr.Filename, FileContent: fileContent}
			uploadedFiles = append(uploadedFiles, file)
		}
		return uploadedFiles, nil
	}
	return uploadedFiles, nil
}

var validMimeTypes = []string{
	"image/jpeg",
	"image/jp2",
	"image/gif",
	"image/png",
	"image/bmp",
	"image/webp",
	"image/heic",
	"image/heif",
}

func processFile(hdr *multipart.FileHeader) ([]byte, error) {
	file, err := hdr.Open()
	if err != nil {
		return nil, errors.New("invalid file data: no headers")
	}
	reader := bufio.NewReader(file)
	defer func() {
		file.Close()
	}()

	fileContent, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return fileContent, nil
}

func IsImageUpload(fileContent []byte) bool {
	// only allow images to be uploaded
	mime := mimetype.Detect(fileContent)
	validMimeType := false
	for _, mimeType := range validMimeTypes {
		if mime.Is(mimeType) {
			validMimeType = true
			break
		}
	}
	return validMimeType
}
