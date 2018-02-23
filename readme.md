![drog-icon](https://github.com/kindlychung/drog/blob/master/icon/drog.png)
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

```
Usage:
drog <path> <title>
echo "something" | drog -- <title> <.csv|.html|.txt>
drog <--url|-u> <http://...> <title>
```
