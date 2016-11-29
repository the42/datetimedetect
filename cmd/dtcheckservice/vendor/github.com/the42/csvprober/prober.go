package csvprober

import (
	"bytes"
	"encoding/csv"
	"io"
	"math"
	"sort"
)

// You may change this prior to a call to NewProber() or change the returned struct
// prior to a call to Probe
var DefaultDelims = []rune{',', ';', '#', '|'}

// This many records will be tried to find an optimal CSV Reader definition
var ProbeRecords = 200

type statresults struct {
	Min, LQ, Median, UQ, Max int
	Mean, Stddev             float64
}

// This struct contains CSV heterogenity information about parsed CSV data
type CSVprobability struct {
	Parsedrecords int  // how many CSV records have been actually parsed?
	Delimiter     rune // What delimiter has been used?
	statresults        // statistical data concerning the attempts to parse CSV data
}

type CSVProbeResult struct {
	CSVprobability []CSVprobability
	ActualLines    int // How many lines did the inspected CSV data actually contain?
}

// sort interface
type csvprobabilityslice []CSVprobability

func (p csvprobabilityslice) Len() int      { return len(p) }
func (p csvprobabilityslice) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

// This function decides between two CSVprobability items which one of the two
// is more "compact" and more likely to be sane, well-formed CSV data. This is
// done by inspecting the Box and Whisker data on the number of records read.
func (p csvprobabilityslice) Less(i, j int) bool {
	// calculate the coefficient of variation http://en.wikipedia.org/wiki/Coefficient_of_variation
	cva := p[i].Stddev / p[i].Mean
	cvb := p[j].Stddev / p[j].Mean

	return cva < cvb
}

// destructively sort the data int-array and return Box and Whisker information
// http://en.wikipedia.org/wiki/Box_and_whisker_plot
// function will panic if len(data) == 0
func genstatdata(data []int) statresults {
	sort.IntSlice(data).Sort()

	res := statresults{}

	// calculate data for Box and Whisker
	res.Min = data[0]
	res.LQ = data[len(data)/4]
	res.Median = data[len(data)/2]
	res.UQ = data[len(data)/4*3]
	res.Max = data[len(data)-1]

	var sum, squaresum int
	for _, item := range data {
		sum += item
		squaresum += item * item
	}
	res.Mean = float64(sum) / float64(len(data))
	res.Stddev = math.Sqrt(float64(squaresum)/float64(len(data)) - res.Mean*res.Mean)
	return res
}

type CSVProber struct {
	RecordstoProbe int    // How many records should be inspected to gather statistical data?
	Delimiters     []rune // array of delimiting characters which should be tried when parsing CSV data
}

// This function accepts an io.Reader which will be used to read CSV data from.
// The returned CSVProbeResult contains statistical data about how uniform the
// CSV data is structured and will inform a CSV reader what data to keep and what
// to discard, as it might very likely be an ill-formed CSV data record.
func (p *CSVProber) Probe(r io.Reader) (*CSVProbeResult, error) {
	var prob []CSVprobability
	// keep the numer
	recordstoprobe := p.RecordstoProbe

	w := new(bytes.Buffer)
	for _, delim := range p.Delimiters {
		var numrecords []int

		// copy the reader to a secondary writter, otherwise it will get consumed
		// and we can not rewind the input stream
		csvreader := csv.NewReader(io.TeeReader(r, w))

		csvreader.Comma = delim
		csvreader.FieldsPerRecord = -1
		csvreader.LazyQuotes = true

		for i := 0; i < recordstoprobe; i++ {

			data, err := csvreader.Read()

			if err == nil {
				numrecords = append(numrecords, len(data))
			} else if err == io.EOF {
				recordstoprobe = i
				break
			} else if _, ok := err.(*csv.ParseError); !ok {
				// if the error is not a parse error, it might as well be a read error
				return nil, err
			}
		}

		// only append probability data, if at least more than one record could actually be read
		if len(numrecords) > 0 {
			prob = append(prob, CSVprobability{
				Parsedrecords: len(numrecords),
				statresults:   genstatdata(numrecords),
				Delimiter:     delim,
			})
		}

		// make the writer the new reader to be able to re-read the data
		r = w
	}

	// sort according to read quality likelihood. See Less
	sort.Sort(csvprobabilityslice(prob))

	// the number of actual records which were used to calculate the read quality statistics
	// might be smaller than the number of records which should be inspected (p.RecordstoProbe),
	// because the reader might simply contain not that many CSV records. Save the actually read
	// CSV records in CSVProbeResult.ActualLines.  If Min.records == Max.records == ActualLines,
	// a perfect match (perfectly structured CSV data) is found. Otherwise heuristics has to be
	// used as how to actually read and process the CSV data.
	return &CSVProbeResult{ActualLines: recordstoprobe, CSVprobability: prob}, nil
}

// NewProber will return a struct containing
func NewProber() *CSVProber {
	return &CSVProber{
		Delimiters:     DefaultDelims,
		RecordstoProbe: ProbeRecords,
	}
}
