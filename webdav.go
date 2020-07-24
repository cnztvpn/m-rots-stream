package stream

import (
	"encoding/xml"
	"mime"
	"path"

	ds "github.com/m-rots/bernard/datastore"
)

type MultiStatus struct {
	XMLName   xml.Name   `xml:"D:multistatus"`
	Namespace string     `xml:"xmlns:D,attr"`
	Responses []Response `xml:"D:response"`
}

type Response struct {
	Href     string   `xml:"D:href"`
	Propstat Propstat `xml:"D:propstat"`
}

type Propstat struct {
	Prop   Prop   `xml:"D:prop"`
	Status string `xml:"D:status"`
}

type Prop struct {
	ContentLength uint64       `xml:"D:getcontentlength,omitempty"`
	ContentType   string       `xml:"D:getcontenttype,omitempty"`
	DisplayName   string       `xml:"D:displayname"`
	Etag          string       `xml:"D:getetag,omitempty"`
	ResourceType  ResourceType `xml:"D:resourcetype"`
}

type ResourceType struct {
	Collection string `xml:",innerxml"`
}

const (
	statusOK       = "HTTP/1.1 200 OK"
	statusLocation = "HTTP/1.1 301 Moved Permanently"
)

func createDavFolder(href string, name string) Response {
	return Response{
		Href: href,
		Propstat: Propstat{
			Prop: Prop{
				DisplayName: name,
				ResourceType: ResourceType{
					Collection: `<D:collection xmlns:D="DAV:"/>`,
				},
			},
			Status: statusOK,
		},
	}
}

func createDavFile(href string, f ds.File) Response {
	return Response{
		Href: href,
		Propstat: Propstat{
			Status: statusOK,
			Prop: Prop{
				ContentLength: uint64(f.Size),
				ContentType:   mime.TypeByExtension(path.Ext(f.Name)),
				DisplayName:   f.Name,
				Etag:          f.MD5,
			},
		},
	}
}
