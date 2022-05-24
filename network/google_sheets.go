package network

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/phil-inc/pcommon/data"
	"github.com/phil-inc/pcommon/pconfig"
	"github.com/phil-inc/pcommon/util"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	drive "google.golang.org/api/drive/v2"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// GetClient get the client to work with g suite
func GetClient(ctx context.Context) (*http.Client, error) {
	data, err := loadGoogleCredCfgs(ctx)
	if err != nil {
		return nil, err
	}

	conf, err := google.JWTConfigFromJSON(data, "https://www.googleapis.com/auth/drive")
	if err != nil {
		return nil, err
	}

	client := conf.Client(ctx)
	if err != nil {
		return nil, err
	}

	return client, err
}

//ExportCSVToSheet takes the csvData to create a google sheet in the drive
func ExportCSVToSheet(ctx context.Context, namePrefix, csvData string) (string, error) {
	client, err := GetClient(ctx)
	if err != nil {
		return "", err
	}

	resp, err := CreateGoogleSheetInDrive(ctx, namePrefix)
	if err != nil {
		return "", err
	}

	srv, err := sheets.New(client)
	if err != nil {
		return "", err
	}

	var vr sheets.ValueRange

	r := csv.NewReader(strings.NewReader(csvData))
	rows, err := r.ReadAll()
	if err != nil {
		return "", err
	}

	for _, cols := range rows {

		var v []interface{}
		for _, col := range cols {
			v = append(v, col)
		}
		vr.Values = append(vr.Values, v)
	}

	//Update the sheet with the csv
	ssID := resp.Id
	writeRange := "sheet1"
	_, err = srv.Spreadsheets.Values.Update(ssID, writeRange, &vr).ValueInputOption("RAW").Do()
	if err != nil {
		return "", err
	}

	return resp.AlternateLink, nil
}

func CreateGoogleSheetInDrive(ctx context.Context, namePrefix string) (*drive.File, error) {
	client, err := GetClient(ctx)
	if err != nil {
		return nil, err
	}

	srv, err := drive.New(client)
	if err != nil {
		return nil, err
	}

	//Create a new sheet
	title := fmt.Sprintf("%s-sheet-%s", namePrefix, util.USFormatDate(util.NowPST()))
	fi := &drive.File{Title: title, Description: "description", MimeType: "application/vnd.google-apps.spreadsheet"}
	p := &drive.ParentReference{Id: pconfig.GetString("google.drive.folderId")}
	fi.Parents = []*drive.ParentReference{p}

	resp, err := srv.Files.Insert(fi).Do()
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type FileMetaData struct {
	ResourceKey       string            `json:"resourceKey"`
	LinkShareMetaData linkShareMetaData `json:"linkShareMetaData"`
}

type linkShareMetaData struct {
	SecurityUpdateEligible bool `json:"securityUpdateEligible"`
	SecurityUpdateEnabled  bool `json:"securityUpdateEnabled"`
}

type spreadSheetData struct {
	ID          string        `json:"spreadsheetId"`
	ValueRanges []valueRanges `json:"valueRanges"`
}

type valueRanges struct {
	Range          string     `json:"range"`
	MajorDimension string     `json:"majorDimension"`
	Values         [][]string `json:"values"`
}

//ReadDataFromGoogleSpreadSheet gets data from Google spreadsheet
func ReadDataFromGoogleSpreadSheet(sheetURL string) ([][]string, error) {
	reqHeaders := map[string]string{}

	body, err := HTTPGet(sheetURL, reqHeaders)
	if err != nil {
		return nil, err
	}

	response := new(spreadSheetData)
	if err := json.Unmarshal(body, response); err != nil {
		return nil, err
	}

	if len(response.ValueRanges) == 0 || len(response.ValueRanges[0].Values) == 0 || len(response.ValueRanges[0].Values[0]) == 0 {
		return nil, errors.New("incorrect data")
	}

	return response.ValueRanges[0].Values, nil
}

func ReadDataFromGoogleSpreadSheetByIDAndRange(ctx context.Context, sheetId, readRange string) ([][]interface{}, error) {
	client, err := GetClient(context.Background())
	if err != nil {
		return nil, err
	}

	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	resp, err := srv.Spreadsheets.Values.Get(sheetId, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve data from sheet: %v", err)
	}

	return resp.Values, nil
}

func ReadMetaDataFromGoogleSpreadSheetByID(sheetId string) (*FileMetaData, error) {
	client, err := GetClient(context.Background())
	if err != nil {
		return nil, err
	}

	metaDataUrl := fmt.Sprintf("https://www.googleapis.com/drive/v2/files/%s", sheetId)

	resp, err := client.Get(metaDataUrl)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve meta data from sheet: %v", err)
	}

	defer resp.Body.Close()
	rbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil
	}
	metaData := new(FileMetaData)
	if err := json.Unmarshal(rbody, metaData); err != nil {
		return nil, err
	}

	return metaData, nil
}

//ExportDataToGoogleSheetByIDAndRange takes the spreadsheet rows update the specified google sheet
func ExportDataToGoogleSheetByIDAndRange(sheetId, writeRange string, rows [][]interface{}) error {
	client, err := GetClient(oauth2.NoContext)
	if err != nil {
		return err
	}

	srv, err := sheets.New(client)
	if err != nil {
		return err
	}

	var vr sheets.ValueRange
	vr.Values = rows

	//Update the sheet with the csv
	ssID := sheetId
	_, err = srv.Spreadsheets.Values.Update(ssID, writeRange, &vr).ValueInputOption("RAW").Do()
	if err != nil {
		return err
	}

	return nil
}

// loading from platfromConfig only
func loadGoogleCredCfgs(ctx context.Context) ([]byte, error) {
	var gcred data.GoogleCreds

	gcred.AuthURI = pconfig.GetString("googleCreds.auth_uri")
	gcred.ClientCertURI = pconfig.GetString("googleCreds.client_x509_cert_url")
	gcred.ClientEmail = pconfig.GetString("googleCreds.client_email")
	gcred.ClientID = pconfig.GetString("googleCreds.client_id")
	gcred.PrivateKey = pconfig.GetString("googleCreds.private_key")
	gcred.ProjectID = pconfig.GetString("google.projectId")
	gcred.PrivateKeyID = pconfig.GetString("google.privateKeyId")
	gcred.ProviderCertURI = pconfig.GetString("googleCreds.auth_provider_x509_cert_url")
	gcred.TokenURI = pconfig.GetString("googleCreds.token_uri")
	gcred.Type = pconfig.GetString("googleCreds.type")

	cs := util.ToJSON(gcred)
	return []byte(cs), nil
}

//ExportDataToGoogleSheetByIDAndRange takes the spreadsheet rows update the specified google sheet and parse as user typed
func ExportDataToGoogleSheetByIDAndRangeParsedAsUserTyped(sheetId, writeRange string, rows [][]interface{}) error {
	client, err := GetClient(oauth2.NoContext)
	if err != nil {
		return err
	}

	srv, err := sheets.New(client)
	if err != nil {
		return err
	}

	var vr sheets.ValueRange
	vr.Values = rows

	//Update the sheet with the csv
	ssID := sheetId
	_, err = srv.Spreadsheets.Values.Update(ssID, writeRange, &vr).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		return err
	}

	return nil
}

//ClearDataOfGoogleSheetByIDAndRange clears column data of the specified range
func ClearDataOfGoogleSheetByIDAndRange(sheetId string, clearRanges []string) error {
	client, err := GetClient(oauth2.NoContext)
	if err != nil {
		return err
	}

	srv, err := sheets.New(client)
	if err != nil {
		return err
	}

	var cr sheets.BatchClearValuesRequest
	cr.Ranges = clearRanges
	ssID := sheetId

	_, err = srv.Spreadsheets.Values.BatchClear(ssID, &cr).Do()
	if err != nil {
		return err
	}

	return nil
}

// // after
// // Retrieve a token, saves the token, then returns the generated client.
// func geetClient(config *oauth2.Config) *http.Client {
// 	// The file token.json stores the user's access and refresh tokens, and is
// 	// created automatically when the authorization flow completes for the first
// 	// time.
// 	tokFile := "token.json"
// 	tok, err := tokenFromFile(tokFile)
// 	if err != nil {
// 		tok = getTokenFromWeb(config)
// 		saveToken(tokFile, tok)
// 	}
// 	return config.Client(context.Background(), tok)
// }

// // Request a token from the web, then returns the retrieved token.
// func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
// 	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
// 	fmt.Printf("Go to the following link in your browser then type the "+
// 		"authorization code: \n%v\n", authURL)

// 	var authCode string
// 	if _, err := fmt.Scan(&authCode); err != nil {
// 		log.Fatalf("Unable to read authorization code: %v", err)
// 	}

// 	tok, err := config.Exchange(context.TODO(), authCode)
// 	if err != nil {
// 		log.Fatalf("Unable to retrieve token from web: %v", err)
// 	}
// 	return tok
// }

// // Retrieves a token from a local file.
// func tokenFromFile(file string) (*oauth2.Token, error) {
// 	f, err := os.Open(file)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer f.Close()
// 	tok := &oauth2.Token{}
// 	err = json.NewDecoder(f).Decode(tok)
// 	return tok, err
// }

// // Saves a token to a file path.
// func saveToken(path string, token *oauth2.Token) {
// 	fmt.Printf("Saving credential file to: %s\n", path)
// 	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
// 	if err != nil {
// 		log.Fatalf("Unable to cache oauth token: %v", err)
// 	}
// 	defer f.Close()
// 	json.NewEncoder(f).Encode(token)
// }

// func main() {
// 	ctx := context.Background()
// 	b, err := ioutil.ReadFile("credentials.json")
// 	if err != nil {
// 		log.Fatalf("Unable to read client secret file: %v", err)
// 	}

// 	// If modifying these scopes, delete your previously saved token.json.
// 	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets.readonly")
// 	if err != nil {
// 		log.Fatalf("Unable to parse client secret file to config: %v", err)
// 	}
// 	client := getClient(config)

// 	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
// 	if err != nil {
// 		log.Fatalf("Unable to retrieve Sheets client: %v", err)
// 	}

// 	// Prints the names and majors of students in a sample spreadsheet:
// 	// https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms/edit
// 	spreadsheetId := "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms"
// 	readRange := "Class Data!A2:E"
// 	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
// 	if err != nil {
// 		log.Fatalf("Unable to retrieve data from sheet: %v", err)
// 	}

// 	if len(resp.Values) == 0 {
// 		fmt.Println("No data found.")
// 	} else {
// 		fmt.Println("Name, Major:")
// 		for _, row := range resp.Values {
// 			// Print columns A and E, which correspond to indices 0 and 4.
// 			fmt.Printf("%s, %s\n", row[0], row[4])
// 		}
// 	}
// }
