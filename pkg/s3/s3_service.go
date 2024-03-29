package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	logger "github.com/phil-inc/plog-ng/pkg/core"
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

func (ps3 *S3Client) CreateBucket(ctx context.Context, bucket string) (*string, error) {
	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	}

	_, err := ps3.Client.CreateBucket(ctx, input)
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

func (ps3 *S3Client) ListBuckets(ctx context.Context) (*BucketLists, error) {
	input := &s3.ListBucketsInput{}

	result, err := ps3.Client.ListBuckets(ctx, input)
	if err != nil {
		return nil, err
	}

	res := BucketLists{result}
	return &res, nil
}

func (ps3 *S3Client) ListFiles(ctx context.Context, bucket, prefix, delimiter *string) (*FileLists, error) {

	input := &s3.ListObjectsV2Input{
		Bucket:    bucket,
		Prefix:    prefix,
		Delimiter: delimiter,
	}

	result, err := ps3.Client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, err
	}
	res := FileLists{result}
	return &res, nil
}

func (ps3 *S3Client) getFile(ctx context.Context, bucket *string, name *string) (*s3.GetObjectOutput, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(*bucket),
		Key:    aws.String(*name),
	}

	return ps3.Client.GetObject(ctx, input)
}

func (ps3 *S3Client) UploadFile(ctx context.Context, bucket string, filename string, file io.Reader) (*string, error) {
	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
		Body:   file,
	}

	uploader := manager.NewUploader(ps3.Client)

	result, err := uploader.Upload(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file, %v", err)
	}
	logger.Infof("File uploaded to s3: %s", result.Location)

	// Before returning the key, create an s3 key (URI)
	item := Item{
		Bucket: bucket,
		Key:    filename,
	}
	s3URI := item.URI()

	return &s3URI, nil
}

func (ps3 *S3Client) DownloadFile(ctx context.Context, bucket, filename string) ([]byte, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
	}

	downloader := manager.NewDownloader(ps3.Client)
	buffer := &manager.WriteAtBuffer{}

	numBytes, err := downloader.Download(ctx, buffer, input)
	if err != nil {
		return nil, fmt.Errorf("failed to download file, %v", err)
	}
	logger.Infof("File downloaded (%d bytes): %s\n", numBytes, filename)

	return buffer.Bytes(), nil
}

func (ps3 *S3Client) ReadFile(ctx context.Context, bucket *string, name *string) (string, error) {
	file, err := ps3.getFile(ctx, bucket, name)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(file.Body)
	content := buf.String()

	return content, nil
}

func (ps3 *S3Client) DownloadFileToPath(ctx context.Context, bucket, fileName, filePath string) error {
	b, err := ps3.DownloadFile(ctx, bucket, fileName)

	if err != nil {
		return err
	}

	err = os.WriteFile(filePath, b, 0644)
	return err
}

func (ps3 *S3Client) UploadFileFromPath(ctx context.Context, bucket, fileName, filePath string) (*string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q, %v", filePath, err)
	}

	defer f.Close()

	filename := path.Base(filePath)

	upload, err := ps3.UploadFile(ctx, bucket, filename, f)

	if err != nil {
		return nil, err
	}
	return upload, nil
}

func (ps3 *S3Client) UploadImage(ctx context.Context, fileName string, fileByte []byte, bucket string) (*string, error) {
	reader := bytes.NewReader(fileByte)

	upload, err := ps3.UploadFile(ctx, bucket, fileName, reader)
	if err != nil {
		return nil, err
	}

	return upload, nil
}

func (ps3 *S3Client) CreateBucketIfNotExist(c context.Context, bucketName string) (*string, error) {
	b, err := ps3.ListBuckets(c)
	if err != nil {
		return nil, err
	}
	for _, v := range b.Buckets {
		if *v.Name == bucketName {
			return nil, nil
		}
	}
	url, err := ps3.CreateBucket(c, bucketName)
	if err != nil {
		return nil, err
	}
	return url, nil
}

func (ps3 *S3Client) DoesBucketExist(c context.Context, bucketName string) bool {

	if !IsValidBucketName(bucketName) {
		return false
	}

	b, err := ps3.ListBuckets(c)
	if err != nil {
		return false
	}

	for _, v := range b.Buckets {
		if *v.Name == bucketName {
			return true
		}
	}

	return false
}

// Checks if bucket naming rule is followed. (Ref: https://docs.aws.amazon.com/AmazonS3/latest/userguide/bucketnamingrules.html)
func IsValidBucketName(bucketName string) bool {

	if len(bucketName) < 3 || len(bucketName) > 63 {
		return false
	}

	r := regexp.MustCompile(`^(([a-z0-9]|[a-z0-9][a-z0-9\-]*[a-z0-9])\.)*([a-z0-9]|[a-z0-9][a-z0-9\-]*[a-z0-9])$`)

	return r.MatchString(bucketName)

}

// DownloadFromS3 takes s3URI as parameter and returns []byte of file content
func (ps3 *S3Client) DownloadFromS3(ctx context.Context, s3URI string) ([]byte, error) {
	s3Item, err := ParseS3URI(s3URI)
	if err != nil {
		return nil, err
	}

	key := strings.TrimPrefix(s3Item.Key, "/")

	return ps3.DownloadFile(ctx, s3Item.Bucket, key)
}

// DownloadFromS3URIToPath takes s3URI as parameter and downloads the file to destination
func (ps3 *S3Client) DownloadFromS3URIToPath(ctx context.Context, s3URI, dest string) error {
	s3Item, err := ParseS3URI(s3URI)
	if err != nil {
		return err
	}

	key := strings.TrimPrefix(s3Item.Key, "/")

	return ps3.DownloadFileToPath(ctx, s3Item.Bucket, key, dest)
}

// List the files in the bucket using continuation token
func (ps3 *S3Client) ListFilesWithContinuationtoken(ctx context.Context, bucket, prefix, delimiter *string, continuationToken *string) (*FileLists, error) {

	input := &s3.ListObjectsV2Input{
		Bucket:            bucket,
		Prefix:            prefix,
		Delimiter:         delimiter,
		ContinuationToken: continuationToken,
	}

	result, err := ps3.Client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, err
	}
	res := FileLists{result}
	return &res, nil
}

// IsFile checks if the given key is s3 file
func (ps3 *S3Client) IsFile(ctx context.Context, bucket, key *string) bool {
	hoi := s3.HeadObjectInput{
		Bucket: bucket,
		Key:    key,
	}

	_, err := ps3.Client.HeadObject(ctx, &hoi)

	return err == nil
}
