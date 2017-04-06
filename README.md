# Hooker
![Hooker](http://s4.pikabu.ru/post_img/2015/01/26/1/1422226538_2049899097.png)

> Tool for easy s3 sftp file management

## Installation
```bash
go get github.com/m1ome/hooker
```

## Usage
```bash
➔ hooker --help
Usage of hooker:
  -check int
    	Interval in seconds of file check (default 180)
  -clear
    	Clear file after send (default true)
  -dir string
    	Directory we should look for a new files (default "/Users/w1n2k/Work/Golang/src/github.com/m1ome/hooker")
  -interval int
    	Time in seconds to sleep between checks (default 60)
  -pattern string
    	Pattern we look files in directory (default ".xml")
  -token string
    	Auth token for API
  -url string
    	URL of reports API (default "http://localhost:3000/")
  -v	Verbose output
```

## Request

Request is gzipped data, containing:
* **Name** - filename
* **Comment** - type of file
Inside of gzip XML minified data contained
