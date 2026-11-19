package sf

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"

	"github.com/spf13/cobra"

	"github.com/ForceCLI/force/lib"
)

// apiError converts an error returned by the Salesforce API into an Error
// named after the API error code, as sf reports it.
func apiError(err error) error {
	if err == nil {
		return nil
	}
	var forceErrors lib.ForceErrors
	if errors.As(err, &forceErrors) && len(forceErrors) > 0 {
		return namedAPIError(forceErrors[0].ErrorCode, forceErrors[0].Message)
	}
	// lib reports a response it cannot parse as "unhandled error: <status>
	// <body>".  sf names these errors ERROR_HTTP_<status>.
	if match := unhandledErrorPattern.FindStringSubmatch(err.Error()); match != nil {
		e := namedAPIError("ERROR_HTTP_"+match[1], match[2]+"\n")
		e.Label = "1"
		return e
	}
	return err
}

var unhandledErrorPattern = regexp.MustCompile(`(?s)^unhandled error: (\d+) (.*)$`)

func namedAPIError(code, message string) *Error {
	data := newOrderedMap()
	data.Set("errorCode", code)
	data.Set("message", message)
	return &Error{Name: code, Message: message, ExitCode: 1, Data: data}
}

// parseOrderedJSON decodes JSON and keeps the key order of objects.  Objects
// become *orderedMap, arrays []interface{} and numbers json.Number.
func parseOrderedJSON(data []byte) (interface{}, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	value, err := decodeOrdered(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, fmt.Errorf("unexpected data after JSON value")
	}
	return value, nil
}

func decodeOrdered(dec *json.Decoder) (interface{}, error) {
	token, err := dec.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return token, nil
	}
	switch delim {
	case '{':
		object := newOrderedMap()
		for dec.More() {
			key, err := dec.Token()
			if err != nil {
				return nil, err
			}
			name, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("object key is not a string")
			}
			value, err := decodeOrdered(dec)
			if err != nil {
				return nil, err
			}
			object.Set(name, value)
		}
		if _, err := dec.Token(); err != nil {
			return nil, err
		}
		return object, nil
	case '[':
		array := []interface{}{}
		for dec.More() {
			value, err := decodeOrdered(dec)
			if err != nil {
				return nil, err
			}
			array = append(array, value)
		}
		if _, err := dec.Token(); err != nil {
			return nil, err
		}
		return array, nil
	}
	return nil, fmt.Errorf("unexpected delimiter %v", delim)
}

// startSpinner prints the progress line sf prints on stderr for a command
// that waits on the org: "<message>... " when the work starts and a status
// such as "done" when it ends.  Nothing is printed with --json.
func startSpinner(cmd *cobra.Command, message string) func(status string) {
	if jsonRequested(cmd) {
		return func(string) {}
	}
	errOut := cmd.ErrOrStderr()
	fmt.Fprintf(errOut, "%s... ", message)
	stopped := false
	return func(status string) {
		if stopped {
			return
		}
		stopped = true
		fmt.Fprintln(errOut, status)
	}
}

// displayValue formats a JSON value for a table cell or a "key: value"
// line the way sf does.
func displayValue(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return "null"
	case string:
		return v
	case json.Number:
		return v.String()
	case bool:
		if v {
			return "true"
		}
		return "false"
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(encoded)
}
