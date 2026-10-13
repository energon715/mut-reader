package readers

import (
	"encoding/xml"
	"errors"
	"io"
)

// DecodeXML bounds depth, node count and directives before struct decoding.
// Charset conversion is shared with the FB2 streaming reader.
func DecodeXML(data []byte, value any) error {
	if len(data) > MaxEntryBytes {
		return errors.New("XML size limit exceeded")
	}
	d := decoder(data)
	depth := 0
	count := 0
	for {
		token, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		count++
		if count > 500000 {
			return errors.New("XML token limit exceeded")
		}
		switch token.(type) {
		case xml.Directive:
			return errors.New("XML directives are not supported")
		case xml.StartElement:
			depth++
			if depth > 128 {
				return errors.New("XML depth limit exceeded")
			}
		case xml.EndElement:
			depth--
		}
	}
	return decoder(data).Decode(value)
}
