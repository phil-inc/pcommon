## pcommon AWS S3 Implementation
It uses the AWS SDK V2 for Go to create the S3 connection. It implements the basic functionality of S3 such as upload, download, listing, etc.

## Dependency

go get github.com/phil-inc/pcommon

## Prerequisites
* AWS Secret Key
* AWS Access Key
* AWS Region

## Uses

To start working with this dependency, you need to retrieve the dependency in your Go project with the following command.

```
go get github.com/phil-inc/pcommon
```

Example Code:

```
package main

import (
	"github.com/phil-inc/pcommon/pkg/s3"
)

func main() {
    client := s3.GetS3Client(assessKey, secretKey, region)
    buckets, err := client.ListBuckets()
    if err != nil {
        panic(err)
    }
    for _, bucket := range buckets.Buckets {
        fmt.Println(*bucket.Name + ": " + bucket.CreationDate.Format("2006-01-02 15:04:05 Monday"))
    }
}
```
