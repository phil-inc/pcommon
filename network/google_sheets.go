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
	"github.com/phil-inc/pcommon/util"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	drive "google.golang.org/api/drive/v2"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// GetClient get the client to work with g suite
func GetClient(ctx context.Context, googleCreds data.GoogleCreds) (*http.Client, error) {

	cs := util.ToJSON(googleCreds)
	googleCredsByte := []byte(cs)

	conf, err := google.JWTConfigFromJSON(googleCredsByte, "https://www.googleapis.com/auth/drive")
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
func ExportCSVToSheet(ctx context.Context, namePrefix, csvData string, googleCreds data.GoogleCreds, driveFolderId string) (string, error) {
	client, err := GetClient(ctx, googleCreds)
	if err != nil {
		return "", err
	}

	resp, err := CreateGoogleSheetInDrive(ctx, namePrefix, googleCreds, driveFolderId)
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

func CreateGoogleSheetInDrive(ctx context.Context, namePrefix string, googleCreds data.GoogleCreds, driveFolderId string) (*drive.File, error) {
	client, err := GetClient(ctx, googleCreds)
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
	p := &drive.ParentReference{Id: driveFolderId}
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

func ReadDataFromGoogleSpreadSheetByIDAndRange(ctx context.Context, sheetId, readRange string, googleCreds data.GoogleCreds) ([][]interface{}, error) {
	client, err := GetClient(context.Background(), googleCreds)
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

func ReadMetaDataFromGoogleSpreadSheetByID(sheetId string, googleCreds data.GoogleCreds) (*FileMetaData, error) {
	client, err := GetClient(context.Background(), googleCreds)
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
func ExportDataToGoogleSheetByIDAndRange(sheetId, writeRange string, rows [][]interface{}, googleCreds data.GoogleCreds) error {
	client, err := GetClient(oauth2.NoContext, googleCreds)
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

//ExportDataToGoogleSheetByIDAndRange takes the spreadsheet rows update the specified google sheet and parse as user typed
func ExportDataToGoogleSheetByIDAndRangeParsedAsUserTyped(sheetId, writeRange string, rows [][]interface{}, googleCreds data.GoogleCreds) error {
	client, err := GetClient(oauth2.NoContext, googleCreds)
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
func ClearDataOfGoogleSheetByIDAndRange(sheetId string, clearRanges []string, googleCreds data.GoogleCreds) error {
	client, err := GetClient(oauth2.NoContext, googleCreds)
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
