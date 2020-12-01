package main

import "encoding/base64"

// CellType represents an HBase cell
type CellType struct {
	Column string `json:"column"`
	Value  string `json:"$"`
	Time   uint64 `json:"timestamp"`
}

// EncCellType represents an HBase cell (base64 encoded)
type EncCellType struct {
	Column string `json:"column"`
	Value  string `json:"$"`
	Time   uint64 `json:"timestamp,omitempty"`
}

func (e *EncCellType) decode() (CellType, error) {
	dc, err := b642s(e.Column)
	if err != nil {
		return CellType{}, err
	}
	dv, err := b642s(e.Value)
	if err != nil {
		return CellType{}, err
	}
	return CellType{Column: dc, Value: dv, Time: e.Time}, nil
}

func (c *CellType) encode() EncCellType {
	return EncCellType{Column: s2b64(c.Column), Value: s2b64(c.Value), Time: c.Time}
}

// RowType represents an HBase row
type RowType struct {
	Key  string     `json:"key"`
	Cell []CellType `json:"Cell"`
}

// EncRowType represents an HBase row (base64 encoded)
type EncRowType struct {
	Key  string        `json:"key"`
	Cell []EncCellType `json:"Cell"`
}

func (e *EncRowType) decode() (RowType, error) {
	dk, err := b642s(e.Key)
	if err != nil {
		return RowType{}, err
	}
	r := RowType{Key: dk}
	for _, c := range e.Cell {
		d, err0 := c.decode()
		if err0 != nil {
			return RowType{}, err0
		}
		r.Cell = append(r.Cell, d)
	}
	return r, nil
}

func (r *RowType) encode() EncRowType {
	e := EncRowType{Key: s2b64(r.Key)}
	for _, c := range r.Cell {
		e.Cell = append(e.Cell, c.encode())
	}
	return e
}

// RowsType represents an HBase change set
type RowsType struct {
	Row []RowType `json:"Row"`
}

// EncRowsType represents an HBase change set (base64 encoded)
type EncRowsType struct {
	Row []EncRowType `json:"Row"`
}

func (es *EncRowsType) decode() (RowsType, error) {
	rs := RowsType{}
	for _, e := range es.Row {
		dr, err := e.decode()
		if err != nil {
			return RowsType{}, err
		}
		rs.Row = append(rs.Row, dr)
	}
	return rs, nil
}

func (rs *RowsType) encode() EncRowsType {
	es := EncRowsType{}
	for _, r := range rs.Row {
		es.Row = append(es.Row, r.encode())
	}
	return es
}

// base64 encoded string to string
func b642s(s string) (string, error) {
	d, err := base64.StdEncoding.DecodeString(s)
	return string(d[:]), err
}

// string to base64 encoded string
func s2b64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
