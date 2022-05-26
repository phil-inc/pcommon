package google_sheets

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
