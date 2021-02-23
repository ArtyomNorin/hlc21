// Code generated by easyjson for marshaling/unmarshaling. DO NOT EDIT.

package client

import (
	json "encoding/json"
	easyjson "github.com/mailru/easyjson"
	jlexer "github.com/mailru/easyjson/jlexer"
	jwriter "github.com/mailru/easyjson/jwriter"
)

// suppress unused package warning
var (
	_ *json.RawMessage
	_ *jlexer.Lexer
	_ *jwriter.Writer
	_ easyjson.Marshaler
)

func easyjsonD2b7633eDecodeHlc21Client(in *jlexer.Lexer, out *License) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "id":
			out.ID = int(in.Int())
		case "digAllowed":
			out.DigAllowed = int(in.Int())
		case "digUsed":
			out.DigUsed = int(in.Int())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjsonD2b7633eEncodeHlc21Client(out *jwriter.Writer, in License) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"id\":"
		out.RawString(prefix[1:])
		out.Int(int(in.ID))
	}
	{
		const prefix string = ",\"digAllowed\":"
		out.RawString(prefix)
		out.Int(int(in.DigAllowed))
	}
	{
		const prefix string = ",\"digUsed\":"
		out.RawString(prefix)
		out.Int(int(in.DigUsed))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v License) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonD2b7633eEncodeHlc21Client(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v License) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonD2b7633eEncodeHlc21Client(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *License) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonD2b7633eDecodeHlc21Client(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *License) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonD2b7633eDecodeHlc21Client(l, v)
}
func easyjsonD2b7633eDecodeHlc21Client1(in *jlexer.Lexer, out *ExploreAreaOut) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "area":
			(out.Area).UnmarshalEasyJSON(in)
		case "amount":
			out.Amount = int(in.Int())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjsonD2b7633eEncodeHlc21Client1(out *jwriter.Writer, in ExploreAreaOut) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"area\":"
		out.RawString(prefix[1:])
		(in.Area).MarshalEasyJSON(out)
	}
	{
		const prefix string = ",\"amount\":"
		out.RawString(prefix)
		out.Int(int(in.Amount))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v ExploreAreaOut) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonD2b7633eEncodeHlc21Client1(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v ExploreAreaOut) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonD2b7633eEncodeHlc21Client1(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *ExploreAreaOut) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonD2b7633eDecodeHlc21Client1(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *ExploreAreaOut) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonD2b7633eDecodeHlc21Client1(l, v)
}
func easyjsonD2b7633eDecodeHlc21Client2(in *jlexer.Lexer, out *ExploreAreaIn) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "posX":
			out.PosX = int(in.Int())
		case "posY":
			out.PosY = int(in.Int())
		case "sizeX":
			out.SizeX = int(in.Int())
		case "sizeY":
			out.SizeY = int(in.Int())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjsonD2b7633eEncodeHlc21Client2(out *jwriter.Writer, in ExploreAreaIn) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"posX\":"
		out.RawString(prefix[1:])
		out.Int(int(in.PosX))
	}
	{
		const prefix string = ",\"posY\":"
		out.RawString(prefix)
		out.Int(int(in.PosY))
	}
	{
		const prefix string = ",\"sizeX\":"
		out.RawString(prefix)
		out.Int(int(in.SizeX))
	}
	{
		const prefix string = ",\"sizeY\":"
		out.RawString(prefix)
		out.Int(int(in.SizeY))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v ExploreAreaIn) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonD2b7633eEncodeHlc21Client2(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v ExploreAreaIn) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonD2b7633eEncodeHlc21Client2(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *ExploreAreaIn) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonD2b7633eDecodeHlc21Client2(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *ExploreAreaIn) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonD2b7633eDecodeHlc21Client2(l, v)
}
func easyjsonD2b7633eDecodeHlc21Client3(in *jlexer.Lexer, out *DigIn) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "licenseID":
			out.LicenseID = int(in.Int())
		case "posX":
			out.PosX = int(in.Int())
		case "posY":
			out.PosY = int(in.Int())
		case "depth":
			out.Depth = int(in.Int())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjsonD2b7633eEncodeHlc21Client3(out *jwriter.Writer, in DigIn) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"licenseID\":"
		out.RawString(prefix[1:])
		out.Int(int(in.LicenseID))
	}
	{
		const prefix string = ",\"posX\":"
		out.RawString(prefix)
		out.Int(int(in.PosX))
	}
	{
		const prefix string = ",\"posY\":"
		out.RawString(prefix)
		out.Int(int(in.PosY))
	}
	{
		const prefix string = ",\"depth\":"
		out.RawString(prefix)
		out.Int(int(in.Depth))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v DigIn) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonD2b7633eEncodeHlc21Client3(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v DigIn) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonD2b7633eEncodeHlc21Client3(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *DigIn) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonD2b7633eDecodeHlc21Client3(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *DigIn) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonD2b7633eDecodeHlc21Client3(l, v)
}
