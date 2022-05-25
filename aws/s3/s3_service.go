package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type BucketLists struct {
	*s3.ListBucketsOutput
}

type FileLists struct {
	*s3.ListObjectsV2Output
}

// Struct for storing the bucket and key for an item on s3.
type Item struct {
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
}

// Return a URI representation of an S3Item struct.
func (s3Key *Item) URI() string {
	path := path.Join(s3Key.Bucket, s3Key.Key)

	s3URI := fmt.Sprintf("s3://%s", path)

	return s3URI
}

// Parses an s3 URI for easy access to its different components.
// We store our s3 keys as URIs, to include all details necessary for retrieval.
func ParseS3URI(s3URI string) (*Item, error) {
	u, err := url.Parse(s3URI)
	if err != nil {
		return nil, err
	}

	item := Item{
		Bucket: u.Host,
		Key:    u.Path,
	}

	return &item, nil
}

func CreateBucket(client *s3.Client, bucket string) (*string, error) {
	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	}

	_, err := client.CreateBucket(context.Background(), input)
	if err != nil {
		return nil, err
	}

	// Before returning the key, create an s3 key (URI)
	item := Item{
		Bucket: bucket,
	}
	s3URI := item.URI()

	return &s3URI, nil
}

func ListBuckets(client *s3.Client) (*BucketLists, error) {
	input := &s3.ListBucketsInput{}

	result, err := client.ListBuckets(context.Background(), input)
	if err != nil {
		return nil, err
	}
	res := BucketLists{result}
	return &res, nil
}

func ListFiles(client *s3.Client, bucket *string) (*FileLists, error) {

	input := &s3.ListObjectsV2Input{
		Bucket: bucket,
	}

	result, err := client.ListObjectsV2(context.Background(), input)
	if err != nil {
		return nil, err
	}
	res := FileLists{result}
	return &res, nil
}

func getFile(client *s3.Client, bucket *string, name *string) (*s3.GetObjectOutput, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(*bucket),
		Key:    aws.String(*name),
	}

	return client.GetObject(context.Background(), input)
}

func UploadFile(client *s3.Client, bucket string, filename string, file io.Reader) (*string, error) {
	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
		Body:   file,
	}

	uploader := manager.NewUploader(client)

	_, err := uploader.Upload(context.Background(), input)
	if err != nil {
		return nil, err
	}
	// Before returning the key, create an s3 key (URI)
	item := Item{
		Bucket: bucket,
		Key:    filename,
	}
	s3URI := item.URI()

	return &s3URI, nil
}

func DownloadFile(client *s3.Client, bucket, filename string) ([]byte, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
	}

	downloader := manager.NewDownloader(client)
	buffer := &manager.WriteAtBuffer{}

	_, err := downloader.Download(context.TODO(), buffer, input)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func ReadFile(client *s3.Client, bucket *string, name *string) (string, error) {
	file, err := getFile(client, bucket, name)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(file.Body)
	content := buf.String()

	return content, nil
}

func DownloadFileToPath(client *s3.Client, bucket, fileName, filePath string) error {
	b, err := DownloadFile(client, bucket, fileName)

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filePath, b, 0644)
	return err
}

func UploadFileFromPath(client *s3.Client, bucket, fileName, filePath string) (*string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q, %v", filePath, err)
	}

	defer f.Close()

	filename := path.Base(filePath)

	upload, err := UploadFile(client, bucket, filename, f)

	if err != nil {
		return nil, err
	}
	return upload, nil
}

func UploadImage(client *s3.Client, fileName string, fileByte []byte, bucket string) (*string, error) {
	reader := bytes.NewReader(fileByte)

	upload, err := UploadFile(client, bucket, fileName, reader)
	if err != nil {
		return nil, err
	}

	return upload, nil
}
