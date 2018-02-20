# drog: a commandline tool for uploading files to google drive

A tool that uploads a file to google drive and converts it into corresponding google format when possible. 
See the table below:

**Source format**|**Target format**
:-----:|:-----:
CSV|Google spredsheet
ODS|Google spredsheet
XLS|Google spredsheet
XLSX|Google spredsheet
ODT|Google document
DOC|Google document
DOCX|Google document
TXT|Google document
HTML|Google document
ODP|Google presentation
PPT|Google presentation
PPTX|Google presentation

## Installation

```
go get github.com/kindlychung/drog
```

## Usage 

See `drog -h`

## Todo

* Upload stdin content, e.g. `xsel -b | drog -- --type txt`
* Upload webpage via URL, e.g `drog --url http://www.google.com`
    * In this case you might need to correct the image links in the webpage. You can use this tool: https://github.com/PuerkitoBio/goquery