package s3

import (
	"context"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func ListFiles(client *s3.Client, bucket *string) (*s3.ListObjectsV2Output, error) {

	input := &s3.ListObjectsV2Input{
		Bucket: bucket,
	}

	return client.ListObjectsV2(context.Background(), input)
}

func GetFile(client *s3.Client, bucket *string, prefix string, name string) (*s3.GetObjectAclOutput, error) {
	key := filepath.Join(prefix, name)

	input := &s3.GetObjectAclInput{
		Bucket: aws.String(*bucket),
		Key:    aws.String(key),
	}

	return client.GetObjectAcl(context.Background(), input)
}
