package s3

import (
	"bytes"
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func CreateBucket(client *s3.Client, bucket string) (*s3.CreateBucketOutput, error) {
	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	}

	return client.CreateBucket(context.Background(), input)
}

func ListBuckets(client *s3.Client) (*s3.ListBucketsOutput, error) {
	input := &s3.ListBucketsInput{}

	return client.ListBuckets(context.Background(), input)
}

func ListFiles(client *s3.Client, bucket *string) (*s3.ListObjectsV2Output, error) {

	input := &s3.ListObjectsV2Input{
		Bucket: bucket,
	}

	return client.ListObjectsV2(context.Background(), input)
}

func GetFileAcl(client *s3.Client, bucket *string, name string) (*s3.GetObjectAclOutput, error) {
	key := name

	input := &s3.GetObjectAclInput{
		Bucket: aws.String(*bucket),
		Key:    aws.String(key),
	}

	return client.GetObjectAcl(context.Background(), input)
}

func GetFile(client *s3.Client, bucket *string, name *string) (*s3.GetObjectOutput, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(*bucket),
		Key:    aws.String(*name),
	}

	return client.GetObject(context.Background(), input)
}

func UploadFile(client *s3.Client, bucket string, filename string, file io.Reader) (*manager.UploadOutput, error) {
	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
		Body:   file,
	}

	uploader := manager.NewUploader(client)

	return uploader.Upload(context.Background(), input)
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
	file, err := GetFile(client, bucket, name)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(file.Body)
	content := buf.String()

	return content, nil
}
