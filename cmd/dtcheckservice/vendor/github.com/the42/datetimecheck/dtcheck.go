// Copyright (c) 2016 Johann Höchtl
// See LICENSE for license

/*

Package datetimecheck returns information wheather a dataset contains datetime information

*/
package datetimecheck

import (
	"bytes"
	"encoding/csv"
	"io"
	"regexp"
	"strings"

	"github.com/rakyll/magicmime"
	"github.com/the42/csvprober"
)

// Check max as many bytes
const Checkupto = 8096

type DateTimeChecker struct {
	d *magicmime.Decoder
	l int
}

type Occurence struct {
	Line    int
	Offsets [][]int
	XPath   *string
}

type DateTimeCheckResponse struct {
	ContainsDT bool
	MimeType   *string
	CheckType  *string
	Read       int
	Occurence  []Occurence
}

var metadatadts = regexp.MustCompile("(?i)datum|zeit|datetime|timestamp")
var dataitemdt = regexp.MustCompile(`(?i)\d{1,2}\.\d{1,2}\.\d{4}|\d{4}-\d{1,2}|\d{2}:\d{2}|jän|jan|feb|märz|apr|mai|jun|jul|aug|sep|okt|nov|dez`)

func (d *DateTimeChecker) ContainsDateTimeBytes(b []byte, mt *string) (*DateTimeCheckResponse, error) {
	var dt string
	oc := &DateTimeCheckResponse{}

	if mt == nil || len(*mt) == 0 {
		// No mimetype given, autodetect
		r, err := d.d.TypeByBuffer(b)
		if err != nil {
			return nil, err
		}
		dt = r
		oc.MimeType = &r
	} else {
		dt = *mt
	}
	// detect file type to check accordingly
	dt = strings.ToLower(dt)
	if strings.Contains(dt, "csv") || strings.Contains(dt, "text") {
		dt = "csv"
	}

	oc.CheckType = &dt

	switch dt {
	case "csv":

		//TODO(jh): Check if probe consumes the reader
		//  We are cloning it, might be unneccessary
		b1 := bytes.NewBuffer(b)
		b2 := bytes.NewBuffer(b)

		csvprober := csvprober.NewProber()
		csvproberes, err := csvprober.Probe(b1)
		if err != nil {
			return nil, err
		}
		csvreader := csv.NewReader(b2)
		csvreader.Comma = csvproberes.CSVprobability[0].Delimiter
		csvreader.LazyQuotes = true

		var re *regexp.Regexp

		var i int
		for ; ; i++ {
			record, err := csvreader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			// check if header or data elements contains a notion of a datetime
			for _, v := range record {
				if i == 0 {
					re = metadatadts
				} else {
					re = dataitemdt
				}
				if pos := re.FindAllStringIndex(v, -1); len(pos) > 0 {
					oc.Occurence = append(oc.Occurence, Occurence{Line: i + 1, Offsets: pos})
				}
			}
		}
		oc.Read = i

	}

	if len(oc.Occurence) > 0 {
		oc.ContainsDT = true
	}
	return oc, nil
}

func (d *DateTimeChecker) ContainsDateTimeStream(s io.Reader, mt *string) (*DateTimeCheckResponse, error) {
	var buf = make([]byte, d.l)
	io.ReadFull(s, buf)
	return d.ContainsDateTimeBytes(buf, mt)
}

func ContainsDateTimeBytes(b []byte, mt *string) (*DateTimeCheckResponse, error) {

	d, err := NewDateTimeChecker(0)
	if err != nil {
		return nil, err
	}
	defer d.Close()
	return d.ContainsDateTimeBytes(b, mt)
}

func ContainsDatetimeReader(r io.Reader, mt *string) (*DateTimeCheckResponse, error) {

	d, err := NewDateTimeChecker(0)
	if err != nil {
		return nil, err
	}
	defer d.Close()
	var buf = make([]byte, d.l)
	io.ReadFull(r, buf)

	return d.ContainsDateTimeBytes(buf, mt)
}

func NewDateTimeChecker(limit int) (*DateTimeChecker, error) {

	d, err := magicmime.NewDecoder(magicmime.Flag(magicmime.MAGIC_MIME))
	if err != nil {
		return nil, err
	}
	dc := &DateTimeChecker{d: d, l: limit}
	return dc, nil
}

func (d *DateTimeChecker) Close() {
	d.d.Close()
}
