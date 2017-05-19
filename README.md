# Hooker
> Tool for easy s3 sftp file management

![Hooker](http://s4.pikabu.ru/post_img/2015/01/26/1/1422226538_2049899097.png)

## Installation
```bash
go get github.com/m1ome/hooker
```

## Usage
```bash
Usage of hooker:
  -check int
        Interval in seconds of file check (default 180)
  -clear
        Clear file after send (default true)
  -dir string
        Directory we should look for a new files (default "/Users/w1n2k/Work/Golang/src/github.com/m1ome/hooker")
  -interval int
        Time in seconds to sleep between checks (default 60)
  -listen string
        Server listen address (default ":8080")
  -out string
        Directory we should place zip files into (default "/Users/w1n2k/Work/Golang/src/github.com/m1ome/hooker")
  -patterns string
        Patterns we look files in directory (seperated by: ,) (default ".xml, .xlsx")
  -sep string
        Pattern separator (default ",")
  -timeout int
        Timeout waiting request from API (default 180)
  -token string
        Auth token for API
  -url string
        URL of reports API (default "http://localhost:3000/")
  -v    Verbose output
  -zip
        Zip file (default true)
```

## Request [POST]

**Body:** gzipped data

**Headers:**
```
X-Access-Token: <TOKEN_HERE>
X-File-Name: GPS-CPSbalexp20170325.xml
Content-Encoding: gzip
```

## Information request [GET]
## Path: `/`
## Response: 
```json
{
    "dir_files":[
        ".DS_Store",
        ".gitkeep",
        "GPS-CPSbalexp20170316 3.xml"
    ],
    "working_files":[
        "GPS-CPSbalexp20170316 3.xml"
    ]
}
```
