package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path"
	"path/filepath"

	"google.golang.org/api/googleapi"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"

	conv "bitbucket.org/kindlychung/convenience"
)

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
	cacheFile, err := tokenCacheFile()
	if err != nil {
		log.Fatalf("Unable to get path to cached credential file. %v", err)
	}
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(cacheFile, tok)
	}
	return config.Client(ctx, tok)
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir,
		url.QueryEscape("drive-go-quickstart.json")), err
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// DriveUploader upload service wrapper
type DriveUploader struct {
	service *drive.Service
}

// NewUploader constructor
func NewUploader() DriveUploader {
	user, err := user.Current()
	conv.CheckErr(err)
	configPath := path.Join(user.HomeDir, ".google_drive_client_secret.json")
	ctx := context.Background()
	b, err := ioutil.ReadFile(configPath)
	conv.CheckErr(err)
	// If modifying these scopes, delete your previously saved credentials
	// at ~/.credentials/drive-go-quickstart.json
	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	conv.CheckErr(err)
	client := getClient(ctx, config)
	service, err := drive.New(client)
	conv.CheckErr(err)
	return DriveUploader{service: service}
}

// Mime alias for string
type Mime = string

// ExtToMeta get meta info from extension name
func ExtToMeta(ext string, title string) (Mime, drive.File) {
	metaInfo := drive.File{
		Name:     title,
		MimeType: "application/vnd.google-apps.document",
	}
	// plain text by default
	sourceMime := "text/plain"
	switch ext {
	case ".csv":
		metaInfo.MimeType = "application/vnd.google-apps.spreadsheet"
		sourceMime = "text/csv"
	case ".xls", ".xlt", ".xla":
		metaInfo.MimeType = "application/vnd.google-apps.spreadsheet"
		sourceMime = "application/vnd.ms-excel"
	case ".xlsx":
		metaInfo.MimeType = "application/vnd.google-apps.spreadsheet"
		sourceMime = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".ods":
		metaInfo.MimeType = "application/vnd.google-apps.spreadsheet"
		sourceMime = "application/vnd.oasis.opendocument.spreadsheet"
	case ".odg":
		metaInfo.MimeType = "application/vnd.google-apps.drawing"
		sourceMime = "application/vnd.oasis.opendocument.graphics"
	case ".ppt", ".pot", ".pps", ".ppa":
		metaInfo.MimeType = "application/vnd.google-apps.presentation"
		sourceMime = "application/vnd.ms-powerpoint"
	case ".pptx":
		metaInfo.MimeType = "application/vnd.google-apps.presentation"
		sourceMime = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".odp":
		metaInfo.MimeType = "application/vnd.google-apps.presentation"
		sourceMime = "application/vnd.oasis.opendocument.presentation"
	case ".doc", ".dot":
		sourceMime = "application/msword"
	case ".docx":
		sourceMime = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".odt":
		sourceMime = "application/vnd.oasis.opendocument.text"
	case ".html":
		sourceMime = "text/html"
	default:
	}
	return sourceMime, metaInfo
}

// GetMetaInfo get mime type of source file and meta info on drive
func GetMetaInfo(path string, title string) (Mime, drive.File) {
	ext := filepath.Ext(path)
	println("Extension: ", ext) //##debug
	// set to google doc by default
	return ExtToMeta(ext, title)
}

// PrintResult print drive upload result
func PrintResult(res *drive.File) {
	fmt.Printf("Upload succeeded. \nFilename in drive: %s \nID in drive: %s \nMIME type: %s\n", res.Name, res.Id, res.MimeType)
}

// Upload upload file
func (du *DriveUploader) Upload(path string, title string) {
	sourceMime, metaInfo := GetMetaInfo(path, title)
	f, err := os.Open(path)
	conv.CheckErr(err)
	res, err := du.service.Files.Create(&metaInfo).Media(f, googleapi.ContentType(sourceMime)).Do()
	conv.CheckErr(err)
	PrintResult(res)
}

// UploadBytes upload a raw byte array
func (du *DriveUploader) UploadBytes(raw []byte, title string, ext string) {
	sourceMime, metaInfo := ExtToMeta(ext, title)
	res, err := du.service.Files.Create(&metaInfo).Media(bytes.NewReader(raw), googleapi.ContentType(sourceMime)).Do()
	conv.CheckErr(err)
	fmt.Printf("Upload succeeded. \nFilename in drive: %s \nID in drive: %s \nMIME type: %s\n", res.Name, res.Id, res.MimeType)
	PrintResult(res)
}

const helpMsg = `
drog: A commandline tool for uploading files to google drive

Usage: 
drog <path> <title>
`

func main() {
	args := os.Args[1:]
	if args[0] == "-h" || args[0] == "--help" {
		println(helpMsg)
		os.Exit(0)
	}
	du := NewUploader()
	if args[0] == "--" {
		// read from stdin, not implemented yet
		os.Exit(0)
	}
	sourcePath := args[0]
	title := args[1]
	du.Upload(sourcePath, title)
}
