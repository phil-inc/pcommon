## Network

This folder consists of all the functions related to upload and download of files to FTPS, SFTP and google sheets.

### FTPS Implementation

### SFTP Implementation

### Google Sheets Implementation

It uses the Google Sheets v4 Package [Version: v0.80.0](https://pkg.go.dev/google.golang.org/api@v0.80.0/sheets/v4?tab=versions) for Go which provides access to the Google Sheets API. It implements the basic functionality of google sheets such as create, read, export, clear, etc.
[Repository of Google APIs Client Library for Go](https://github.com/googleapis/google-api-go-client)

#### Prerequisites

* Google Service Account Credentials

#### Uses

To start working with this dependency, you need to retrieve the dependency in your Go project with the following command.

```
go get github.com/phil-inc/pcommon
```

Example Code:

```
package main

import (
	"github.com/phil-inc/pcommon/pkg/network/google_sheets"
)

func main() {
		var clearRanges []string
		gc := google_sheets.GoogleCreds{
            Type:            "type",
            ProjectID:       "project_id",
            PrivateKeyID:    "private_key_id",
            PrivateKey:      "private_key",
            ClientID:        "client_id",
            ClientEmail:     "client_email",
            AuthURI:         "auth_uri",
            TokenURI:        "token_uri",
            ProviderCertURI: "auth_provider_x509_cert_url",
            ClientCertURI:   "client_x509_cert_url",
		}
		
		clearRanges = append(clearRanges, "A1:A3", "B1:B5")
		
		err := gc.ClearDataOfGoogleSheetByIDAndRange("1tIjmeCgr9XRLOIihzvuZdAN57GPGieqBnbttbdkklDk", clearRanges)
		if err != nil {
        panic(err)
    }
}
```

### More Information

For some more information please read through our [Main README file](https://github.com/phil-inc/pcommon#readme).









