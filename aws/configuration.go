package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func GetS3Client(assessKey, secretKey, region string) *s3.Client {

	opts := s3.Options{
		Region:      *aws.String(region),
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(assessKey, secretKey, "")),
	}

	// Create an Amazon S3 service client
	client := s3.New(opts)

	return client
}
