package s3

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	Client *s3.Client
}

func GetS3Client(assessKey, secretKey, region string) *S3Client {

	opts := s3.Options{
		Region:      *aws.String(region),
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(assessKey, secretKey, "")),
	}

	// Create an Amazon S3 service client
	s3Client := s3.New(opts)

	client := &S3Client{
		Client: s3Client,
	}

	return client
}
