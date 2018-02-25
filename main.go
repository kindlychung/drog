package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	neturl "net/url"
	"os"
	"os/user"
	"path"
	"path/filepath"

	"github.com/PuerkitoBio/goquery"

	"github.com/gen2brain/dlgs"

	"google.golang.org/api/googleapi"

	"github.com/gen2brain/beeep"
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
	case ".pdf":
		sourceMime = "application/pdf"
	case ".png":
		sourceMime = "image/png"
	case ".jpg", ".jpeg":
		sourceMime = "image/jpeg"
	case ".html":
		sourceMime = "text/html"
	default:
	}
	return sourceMime, metaInfo
}

func validStdinType(ext string) bool {
	return ext == ".txt" || ext == ".html" || ext == ".csv"
}

// GetMetaInfo get mime type of source file and meta info on drive
func GetMetaInfo(path string, title string) (Mime, drive.File) {
	ext := filepath.Ext(path)
	// set to google doc by default
	return ExtToMeta(ext, title)
}

// PrintResult print drive upload result
func PrintResult(res *drive.File) {
	msg := fmt.Sprintf("Filename in drive: %s \nID in drive: %s \nMIME type: %s\n", res.Name, res.Id, res.MimeType)
	err := beeep.Notify("Upload succeeded", msg, "assets/information.png")
	if err != nil {
		panic(err)
	}
}

// Upload upload file
func (du *DriveUploader) Upload(path string, title string) {
	sourceMime, metaInfo := GetMetaInfo(path, title)
	f, err := os.Open(path)
	conv.BubbleErr(err, "Upload FAILED")
	res, err := du.service.Files.Create(&metaInfo).Media(f, googleapi.ContentType(sourceMime)).Do()
	conv.BubbleErr(err, "Upload FAILED")
	PrintResult(res)
}

// UploadFromReader upload from content of an io.Reader
func (du *DriveUploader) UploadFromReader(reader io.Reader, title string, ext string) {
	sourceMime, metaInfo := ExtToMeta(ext, title)
	res, err := du.service.Files.Create(&metaInfo).Media(reader, googleapi.ContentType(sourceMime)).Do()
	conv.BubbleErr(err, "Upload FAILED")
	PrintResult(res)
}

// UploadBytes upload a raw byte array
func (du *DriveUploader) UploadBytes(raw []byte, title string, ext string) {
	du.UploadFromReader(bytes.NewReader(raw), title, ext)
}

const drogMark = "  drogpost"

const helpMsg = `
drog: A commandline tool for uploading files to google drive

Usage: 
drog <path> <-ask|any text as tile>
echo "something" | drog -- <-ask|any text as title> <-ask|.csv|.html|.txt>
drog <--url|-u> <http://...> <title>
`

func main() {
	args := os.Args[1:]
	if len(args) < 2 || 3 < len(args) {
		log.Fatal(helpMsg)
	}
	if args[0] == "-h" || args[0] == "--help" {
		println(helpMsg)
		os.Exit(0)
	}
	du := NewUploader()
	if args[0] == "--" {
		if len(args) != 3 {
			log.Fatal(helpMsg)
		}
		title := args[1]
		if title == "-ask" {
			t, _, err := dlgs.Entry("Title of upload", "Please enter the title:", "any title")
			title = t
			conv.CheckErr(err)
		}
		title += drogMark
		ext := args[2]
		if ext == "-ask" {
			e, _, err := dlgs.Entry("Filetype of upload", "Please enter the file extension:", ".txt")
			ext = e
			conv.CheckErr(err)
		}
		if !validStdinType(ext) {
			fmt.Printf("invalide extension: %s", ext)
			log.Fatal(helpMsg)
		}
		reader := bufio.NewReader(os.Stdin)
		du.UploadFromReader(reader, title, ext)
		os.Exit(0)
	}
	if args[0] == "--url" || args[0] == "-u" {
		if len(args) != 3 {
			log.Fatal(helpMsg)
		}
		link := args[1]
		_, err := neturl.Parse(link)
		conv.CheckErr(err)
		httpDoc, err := goquery.NewDocument(link)
		conv.CheckErr(err)
		httpTitle := httpDoc.Find("title").Text()
		title := args[2]
		if title == "-ask" {
			title, _, err = dlgs.Entry("Title of upload", "Please enter the title:", httpTitle)
			conv.CheckErr(err)
		} else if title == "-onpage" {
			title = httpTitle
		}
		title += drogMark
		response, err := http.Get(link)
		du.UploadFromReader(bufio.NewReader(response.Body), title, ".html")
		os.Exit(0)
	}
	sourcePath := args[0]
	title := args[1]
	if title == "-ask" {
		t, _, err := dlgs.Entry("Title of upload", "Please enter the title:", "any title")
		title = t
		conv.CheckErr(err)
	}
	title += drogMark
	du.Upload(sourcePath, title)
}
