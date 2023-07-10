package google_sheets

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	drive "google.golang.org/api/drive/v2"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// GetClient get the client to work with g suite
func (gc *GoogleCreds) GetClient(ctx context.Context) (*http.Client, error) {

	googleCredsJSON, err := json.Marshal(gc)
	if err != nil {
		return nil, err
	}

	googleCredsByte := []byte(googleCredsJSON)

	conf, err := google.JWTConfigFromJSON(googleCredsByte, GOOGLE_DRIVE_LINK)
	if err != nil {
		return nil, err
	}

	client := conf.Client(ctx)
	if err != nil {
		return nil, err
	}

	return client, err
}

// ExportCSVToSheet takes the csvData to create a google sheet in the drive
func (gc *GoogleCreds) ExportCSVToSheet(ctx context.Context, namePrefix, csvData string, driveFolderId string, sheetTitle string) (string, error) {
	client, err := gc.GetClient(ctx)
	if err != nil {
		return "", err
	}

	resp, err := gc.CreateGoogleSheetInDrive(ctx, namePrefix, driveFolderId, sheetTitle)
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
	writeRange := DEFAULT_WRITE_RANGE
	_, err = srv.Spreadsheets.Values.Update(ssID, writeRange, &vr).ValueInputOption("RAW").Do()
	if err != nil {
		return "", err
	}

	return resp.AlternateLink, nil
}

func (gc *GoogleCreds) CreateGoogleSheetInDrive(ctx context.Context, namePrefix string, driveFolderId string, title string) (*drive.File, error) {
	client, err := gc.GetClient(ctx)
	if err != nil {
		return nil, err
	}

	srv, err := drive.New(client)
	if err != nil {
		return nil, err
	}

	//Create a new sheet
	fi := &drive.File{Title: title, Description: SHEET_DESC, MimeType: SHEET_MIMETYPE}
	p := &drive.ParentReference{Id: driveFolderId}
	fi.Parents = []*drive.ParentReference{p}

	resp, err := srv.Files.Insert(fi).Do()
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (gc *GoogleCreds) ReadDataFromGoogleSpreadSheetByIDAndRange(ctx context.Context, sheetId, readRange string) ([][]interface{}, error) {
	client, err := gc.GetClient(context.Background())
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

func (gc *GoogleCreds) ReadMetaDataFromGoogleSpreadSheetByID(sheetId string) (*FileMetaData, error) {
	client, err := gc.GetClient(context.Background())
	if err != nil {
		return nil, err
	}

	metaDataUrl := fmt.Sprintf(GOOGLE_SHEETS_LINK, sheetId)

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

// ExportDataToGoogleSheetByIDAndRange takes the spreadsheet rows update the specified google sheet
func (gc *GoogleCreds) ExportDataToGoogleSheetByIDAndRange(sheetId, writeRange string, rows [][]interface{}) error {
	client, err := gc.GetClient(oauth2.NoContext)
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

// ExportDataToGoogleSheetByIDAndRange takes the spreadsheet rows update the specified google sheet and parse as user typed
func (gc *GoogleCreds) ExportDataToGoogleSheetByIDAndRangeParsedAsUserTyped(sheetId, writeRange string, rows [][]interface{}) error {
	client, err := gc.GetClient(oauth2.NoContext)
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

// ClearDataOfGoogleSheetByIDAndRange clears column data of the specified range
func (gc *GoogleCreds) ClearDataOfGoogleSheetByIDAndRange(sheetId string, clearRanges []string) error {
	client, err := gc.GetClient(oauth2.NoContext)
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

// GetModifiedDate gets the date when the google document was modified
func (gc *GoogleCreds) GetModifiedDate(fileID string) (*time.Time, error) {
	ctx := context.Background()
	client, err := gc.GetClient(ctx)
	if err != nil {
		return nil, err
	}

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	f, err := srv.Files.Get(fileID).Do()
	if err != nil {
		return nil, err
	}

	date, err := time.Parse(time.RFC3339, f.ModifiedDate)
	if err != nil {
		return nil, err
	}

	return &date, nil

}
