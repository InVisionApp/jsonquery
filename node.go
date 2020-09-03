package jsonquery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"sort"
	"strconv"
)

// A NodeType is the type of a Node.
type NodeType uint

type contentType string

const (
	// DocumentNode is a document object that, as the root of the document tree,
	// provides access to the entire XML document.
	DocumentNode NodeType = iota
	// ElementNode is an element.
	ElementNode
	// TextNode is the text content of a node.
	TextNode
)

const (
	interfaceType = contentType("interface")
	arrayType     = contentType("array")
	objectType    = contentType("object")

	stringType = contentType("string")
	boolType   = contentType("bool")
	nullType   = contentType("null")

	intType   = contentType("int")
	int8Type  = contentType("int8")
	int16Type = contentType("int16")
	int32Type = contentType("int32")
	int64Type = contentType("int64")

	uintType   = contentType("uint")
	uint8Type  = contentType("uint8")
	uint16Type = contentType("uint16")
	uint32Type = contentType("uint32")
	uint64Type = contentType("uint64")

	float32Type = contentType("float32")
	float64Type = contentType("float64")
)

var types = map[string]contentType{
	"bool":   boolType,
	"string": stringType,

	"int":   intType,
	"int8":  int8Type,
	"int16": int16Type,
	"int32": int32Type,
	"int64": int64Type,

	"uint":   uintType,
	"uint8":  uint8Type,
	"uint16": uint16Type,
	"uint32": uint32Type,
	"uint64": uint64Type,

	"float32": float32Type,
	"float64": float64Type,
}

// A Node consists of a NodeType and some Data (tag name for
// element nodes, content for text) and are part of a tree of Nodes.
type Node struct {
	Parent, PrevSibling, NextSibling, FirstChild, LastChild *Node

	Type NodeType
	Data string

	level       int
	contentType contentType
	idata       interface{}
	skipped     bool
}

// ChildNodes gets all child nodes of the node.
func (n *Node) ChildNodes() []*Node {
	var a []*Node
	for nn := n.FirstChild; nn != nil; nn = nn.NextSibling {
		a = append(a, nn)
	}
	return a
}

// InnerText gets the value of the node and all its child nodes.
func (n *Node) InnerText() string {
	var output func(*bytes.Buffer, *Node)
	output = func(buf *bytes.Buffer, n *Node) {
		if n.Type == TextNode {
			buf.WriteString(n.Data)
			return
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			output(buf, child)
		}
	}
	var buf bytes.Buffer
	output(&buf, n)
	return buf.String()
}

func (n *Node) InnerData() interface{} {
	switch n.contentType {
	case arrayType:
		arr := make([]interface{}, 0)
		for _, node := range n.ChildNodes() {
			if node.skipped {
				continue
			}

			arr = append(arr, node.InnerData())
		}
		return arr
	case objectType:
		obj := map[string]interface{}{}
		for _, node := range n.ChildNodes() {
			if node.skipped {
				continue
			}

			obj[node.Data] = node.InnerData()
		}
		return obj
	}

	if len(n.ChildNodes()) > 0 {
		return n.FirstChild.idata
	}

	return n.idata
}

func (n *Node) SetInnerData(idata interface{}) {
	if n.Type == ElementNode {
		n.ChildNodes()[0].SetInnerData(idata)
	} else if n.Type == TextNode {
		n.idata = idata
		if idata == nil {
			n.Parent.contentType = nullType
		} else {
			typeName := reflect.TypeOf(idata).Name()
			contentType, ok := types[typeName]
			if !ok {
				panic("SetInnerData does not support " + typeName + " type")
			}

			n.Parent.contentType = contentType
			n.Data = fmt.Sprintf("%v", idata)
		}
	}
}

func (n *Node) SetSkipped(skipped bool) {
	n.skipped = skipped
}

func (n *Node) Skipped() bool {
	return n.skipped
}

func (n *Node) GetParent(level int) *Node {
	if n.Parent.level == level {
		return n.Parent
	}

	return n.Parent.GetParent(level)
}

func (n *Node) JSON(skipped bool) (interface{}, error) {
	if n.InnerData() == nil {
		return nil, nil
	}

	switch n.contentType {
	case arrayType:
		arr := make([]interface{}, 0)
		for _, node := range n.ChildNodes() {
			if skipped && node.skipped {
				continue
			}

			value, err := node.JSON(skipped)
			if err != nil {
				return nil, err
			}
			arr = append(arr, value)
		}
		return arr, nil
	case objectType:
		obj := map[string]interface{}{}
		for _, node := range n.ChildNodes() {
			if skipped && node.skipped {
				continue
			}

			value, err := node.JSON(skipped)
			if err != nil {
				return nil, err
			}
			obj[node.Data] = value
		}
		return obj, nil
	case stringType:
		return n.InnerData(), nil
	case intType:
		return n.InnerData().(int), nil
	case int8Type:
		return n.InnerData().(int8), nil
	case int16Type:
		return n.InnerData().(int16), nil
	case int32Type:
		return n.InnerData().(int32), nil
	case int64Type:
		return n.InnerData().(int64), nil
	case uintType:
		return n.InnerData().(uint), nil
	case uint8Type:
		return n.InnerData().(uint8), nil
	case uint16Type:
		return n.InnerData().(uint16), nil
	case uint32Type:
		return n.InnerData().(uint32), nil
	case uint64Type:
		return n.InnerData().(uint64), nil
	case float32Type:
		return n.InnerData().(float32), nil
	case float64Type:
		return n.InnerData().(float64), nil
	case boolType:
		return strconv.ParseBool(n.InnerText())
	case nullType:
		return nil, nil
	default:
		return n.InnerData(), nil
	}
}

func (n *Node) toMap(skipped bool) (map[string]interface{}, error) {
	if n.contentType != objectType {
		return nil, fmt.Errorf("node is not object - %v", n.contentType)
	}

	v, jsonErr := n.JSON(skipped)
	if jsonErr != nil {
		return nil, jsonErr
	}

	return v.(map[string]interface{}), nil
}

func (n *Node) Maps(skipped bool) ([]map[string]interface{}, error) {
	if n.contentType != arrayType {
		return nil, fmt.Errorf("cannot convert Node to []map[string]interface{} - %v", n.contentType)
	}

	var records []map[string]interface{}
	for _, node := range n.ChildNodes() {
		if skipped && node.skipped {
			continue
		}

		v, jsonErr := node.toMap(skipped)
		if jsonErr != nil {
			return nil, jsonErr
		}

		records = append(records, v)
	}

	return records, nil
}

// SelectElement finds the first of child elements with the
// specified name.
func (n *Node) SelectElement(name string) *Node {
	for nn := n.FirstChild; nn != nil; nn = nn.NextSibling {
		if nn.Data == name {
			return nn
		}
	}
	return nil
}

// OutputXML prints the XML string.
func (n *Node) OutputXML() string {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0"?>`)
	for n := n.FirstChild; n != nil; n = n.NextSibling {
		outputXML(&buf, n)
	}
	return buf.String()
}

// LoadURL loads the JSON document from the specified URL.
func LoadURL(url string) (*Node, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return Parse(resp.Body)
}

// Parse JSON document.
func Parse(r io.Reader) (*Node, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return parse(b)
}

func ParseFromMaps(maps []map[string]interface{}) (*Node, error) {
	doc := &Node{Type: DocumentNode, contentType: arrayType}
	parseValue(maps, doc, 1)

	return doc, nil
}

func parseValue(x interface{}, top *Node, level int) {
	addNode := func(n *Node) {
		if n.level == top.level {
			top.NextSibling = n
			n.PrevSibling = top
			n.Parent = top.Parent
			if top.Parent != nil {
				top.Parent.LastChild = n
			}
		} else if n.level > top.level {
			n.Parent = top
			if top.FirstChild == nil {
				top.FirstChild = n
				top.LastChild = n
			} else {
				t := top.LastChild
				t.NextSibling = n
				n.PrevSibling = t
				top.LastChild = n
			}
		}
	}

	addTextNodeFromInteger := func(v interface{}) {
		s := fmt.Sprintf("%v", v)
		n := &Node{Data: s, Type: TextNode, level: level, idata: v}
		addNode(n)
	}

	addTextNodeFromFloat := func(v float64) {
		s := strconv.FormatFloat(v, 'f', -1, 64)
		n := &Node{Data: s, Type: TextNode, level: level, idata: v}
		addNode(n)
	}

	// Handle nil value
	if x == nil {
		top.contentType = nullType
		n := &Node{Data: "", Type: TextNode, level: level, idata: x}
		addNode(n)

		return
	}

	// Handle slice
	if reflect.TypeOf(x).Kind() == reflect.Slice {
		top.contentType = arrayType

		index := 0
		value := reflect.ValueOf(x)
		for index < value.Len() {
			n := &Node{Type: ElementNode, level: level}
			addNode(n)
			parseValue(value.Index(index).Interface(), n, level+1)
			index++
		}

		return
	}

	// Handle basic types
	switch v := x.(type) {
	case map[string]interface{}:
		// The Go’s map iteration order is random.
		// (https://blog.golang.org/go-maps-in-action#Iteration-order)
		var keys []string
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		top.contentType = objectType
		for _, key := range keys {
			n := &Node{Data: key, Type: ElementNode, level: level}
			addNode(n)
			parseValue(v[key], n, level+1)
		}
	case string:
		top.contentType = stringType
		n := &Node{Data: v, Type: TextNode, level: level, idata: v}
		addNode(n)
	case int:
		top.contentType = intType
		addTextNodeFromInteger(v)
	case int8:
		top.contentType = int8Type
		addTextNodeFromInteger(v)
	case int16:
		top.contentType = int16Type
		addTextNodeFromInteger(v)
	case int32:
		top.contentType = int32Type
		addTextNodeFromInteger(v)
	case int64:
		top.contentType = int64Type
		addTextNodeFromInteger(v)
	case uint:
		top.contentType = uintType
		addTextNodeFromInteger(v)
	case uint8:
		top.contentType = uint8Type
		addTextNodeFromInteger(v)
	case uint16:
		top.contentType = uint16Type
		addTextNodeFromInteger(v)
	case uint32:
		top.contentType = uint32Type
		addTextNodeFromInteger(v)
	case uint64:
		top.contentType = uint64Type
		addTextNodeFromInteger(v)
	case float32:
		top.contentType = float32Type
		addTextNodeFromFloat(float64(v))
	case float64:
		top.contentType = float64Type
		addTextNodeFromFloat(v)
	case bool:
		top.contentType = boolType
		s := strconv.FormatBool(v)
		n := &Node{Data: s, Type: TextNode, level: level, idata: v}
		addNode(n)
	default:
		top.contentType = interfaceType
		s := fmt.Sprintf("%v", v)
		n := &Node{Data: s, Type: TextNode, level: level, idata: v}
		addNode(n)
	}
}

func parse(b []byte) (*Node, error) {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}

	doc := &Node{Type: DocumentNode}
	switch v.(type) {
	case []interface{}:
		doc.contentType = arrayType
	case map[string]interface{}:
		doc.contentType = objectType
	}

	parseValue(v, doc, 1)
	return doc, nil
}

func outputXML(buf *bytes.Buffer, n *Node) {
	switch n.Type {
	case ElementNode:
		if n.Data == "" {
			buf.WriteString("<element>")
		} else {
			buf.WriteString("<" + n.Data + ">")
		}
	case TextNode:
		buf.WriteString(n.Data)
		return
	}

	for child := n.FirstChild; child != nil; child = child.NextSibling {
		outputXML(buf, child)
	}
	if n.Data == "" {
		buf.WriteString("</element>")
	} else {
		buf.WriteString("</" + n.Data + ">")
	}
}
