// Command validator runs validator.js exports over JSON Lines.
//
// Each input line is a JSON object {"id": ..., "fn": "<export name>", "args": [...]}. For each line the command
// writes one line: {"id": ..., "ok": <result>} when the call returns, or {"id": ..., "error": {"name": ...,
// "message": ...}} when it throws. Values that JSON cannot express are tagged objects: {"$undefined": true},
// {"$number": "NaN" | "Infinity" | "-Infinity" | "-0"}, {"$regexp": [source, flags]} and {"$date": <ms or null>}
// as an argument; a Date result is {"$date": "<toISOString text>"}, or {"$date": null} for an invalid date.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	validator "github.com/validatorjs/validator.js"
	"github.com/validatorjs/validator.js/internal/wire"
	"github.com/validatorjs/validator.js/js"
)

type request struct {
	ID   json.RawMessage `json:"id"`
	Fn   string          `json:"fn"`
	Args json.RawMessage `json:"args"`
}

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "validator:", err)
		os.Exit(1)
	}
}

func run(r io.Reader, w io.Writer) error {
	in := bufio.NewReader(r)
	out := bufio.NewWriter(w)
	for {
		line, err := in.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			if werr := handle(line, out); werr != nil {
				return werr
			}
		}
		if err == io.EOF {
			return out.Flush()
		}
		if err != nil {
			if ferr := out.Flush(); ferr != nil {
				return fmt.Errorf("%w (flushing output: %v)", err, ferr)
			}
			return fmt.Errorf("reading input: %w", err)
		}
	}
}

func handle(line []byte, out *bufio.Writer) error {
	var req request
	if err := json.Unmarshal(line, &req); err != nil {
		return writeError(out, nil, "InputError", err.Error())
	}
	args, err := wire.DecodeArgs(req.Args)
	if err != nil {
		return writeError(out, req.ID, "InputError", err.Error())
	}
	result, callErr := validator.Call(req.Fn, args)
	if callErr != nil {
		if jsErr, ok := callErr.(*js.Error); ok {
			return writeError(out, req.ID, jsErr.Name, jsErr.Message)
		}
		return writeError(out, req.ID, "InputError", callErr.Error())
	}
	var b bytes.Buffer
	b.WriteString(`{"id":`)
	b.Write(idOrNull(req.ID))
	b.WriteString(`,"ok":`)
	wire.Encode(&b, result)
	b.WriteString("}\n")
	_, err = out.Write(b.Bytes())
	return err
}

func idOrNull(id json.RawMessage) []byte {
	if len(id) == 0 {
		return []byte("null")
	}
	return id
}

func writeError(out *bufio.Writer, id json.RawMessage, name, message string) error {
	var b bytes.Buffer
	b.WriteString(`{"id":`)
	b.Write(idOrNull(id))
	b.WriteString(`,"error":{"name":`)
	wire.WriteString(&b, name)
	b.WriteString(`,"message":`)
	wire.WriteString(&b, message)
	b.WriteString("}}\n")
	_, err := out.Write(b.Bytes())
	return err
}
