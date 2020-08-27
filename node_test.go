package jsonquery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"sort"
	"strings"
	"testing"
)

func parseString(s string) (*Node, error) {
	return Parse(strings.NewReader(s))
}

func TestParseJsonNumberArray(t *testing.T) {
	s := `[1,2,3,4,5,6]`
	doc, err := parseString(s)
	if err != nil {
		t.Fatal(err)
	}
	// output like below:
	// <element>1</element>
	// <element>2</element>
	// ...
	// <element>6</element>
	if e, g := 6, len(doc.ChildNodes()); e != g {
		t.Fatalf("excepted %v but got %v", e, g)
	}
	var v []string
	for _, n := range doc.ChildNodes() {
		v = append(v, n.InnerText())
	}
	if got, expected := strings.Join(v, ","), "1,2,3,4,5,6"; got != expected {
		t.Fatalf("got %v but expected %v", got, expected)
	}
}

func TestParseJsonObject(t *testing.T) {
	s := `{
		"name":"John",
		"age":31,
		"city":"New York"
	}`
	doc, err := parseString(s)
	if err != nil {
		t.Fatal(err)
	}
	// output like below:
	// <name>John</name>
	// <age>31</age>
	// <city>New York</city>
	m := make(map[string]string)
	for _, n := range doc.ChildNodes() {
		m[n.Data] = n.InnerText()
	}
	expected := []struct {
		name, value string
	}{
		{"name", "John"},
		{"age", "31"},
		{"city", "New York"},
	}
	for _, v := range expected {
		if e, g := v.value, m[v.name]; e != g {
			t.Fatalf("expected %v=%v,but %v=%v", v.name, e, v.name, g)
		}
	}
}

func TestParseJsonObjectArray(t *testing.T) {
	s := `[
		{ "name":"Ford", "models":[ "Fiesta", "Focus", "Mustang" ] },
		{ "name":"BMW", "models":[ "320", "X3", "X5" ] },
        { "name":"Fiat", "models":[ "500", "Panda" ] }
	]`
	doc, err := parseString(s)
	if err != nil {
		t.Fatal(err)
	}
	/**
	<element>
		<name>Ford</name>
		<models>
			<element>Fiesta</element>
			<element>Focus</element>
			<element>Mustang</element>
		</models>
	</element>
	<element>
		<name>BMW</name>
		<models>
			<element>320</element>
			<element>X3</element>
			<element>X5</element>
		</models>
	</element>
	....
	*/
	if e, g := 3, len(doc.ChildNodes()); e != g {
		t.Fatalf("expected %v, but %v", e, g)
	}
	m := make(map[string][]string)
	for _, n := range doc.ChildNodes() {
		// Go to the next of the element list.
		var name string
		var models []string
		for _, e := range n.ChildNodes() {
			if e.Data == "name" {
				// a name node.
				name = e.InnerText()
			} else {
				// a models node.
				for _, k := range e.ChildNodes() {
					models = append(models, k.InnerText())
				}
			}
		}
		// Sort models list.
		sort.Strings(models)
		m[name] = models

	}
	expected := []struct {
		name, value string
	}{
		{"Ford", "Fiesta,Focus,Mustang"},
		{"BMW", "320,X3,X5"},
		{"Fiat", "500,Panda"},
	}
	for _, v := range expected {
		if e, g := v.value, strings.Join(m[v.name], ","); e != g {
			t.Fatalf("expected %v=%v,but %v=%v", v.name, e, v.name, g)
		}
	}
}

func TestParseJson(t *testing.T) {
	s := `{
		"name":"John",
		"age":30,
		"cars": [
			{ "name":"Ford", "models":[ "Fiesta", "Focus", "Mustang" ] },
			{ "name":"BMW", "models":[ "320", "X3", "X5" ] },
			{ "name":"Fiat", "models":[ "500", "Panda" ] }
		]
	 }`
	doc, err := parseString(s)
	if err != nil {
		t.Fatal(err)
	}
	n := doc.SelectElement("name")
	if n == nil {
		t.Fatal("n is nil")
	}
	if n.NextSibling != nil {
		t.Fatal("next sibling shoud be nil")
	}
	if e, g := "John", n.InnerText(); e != g {
		t.Fatalf("expected %v but %v", e, g)
	}
	cars := doc.SelectElement("cars")
	if e, g := 3, len(cars.ChildNodes()); e != g {
		t.Fatalf("expected %v but %v", e, g)
	}
}

func TestLargeFloat(t *testing.T) {
	s := `{
		"large_number": 365823929453
	 }`
	doc, err := parseString(s)
	if err != nil {
		t.Fatal(err)
	}
	n := doc.SelectElement("large_number")
	if n.InnerText() != "365823929453" {
		t.Fatalf("expected %v but %v", "365823929453", n.InnerText())
	}
}

func TestJSON(t *testing.T) {
	files := []string{
		"basic.json",
		"screen_v3_01.json",
		"screen_v3_02.json",
		"screen_v3_03.json",
		"screen_v3_04.json",
	}

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			originalBytes, err := ioutil.ReadFile(path.Join("testdata", file))
			if err != nil {
				t.Fatal(err)
			}

			var iOriginalJSON interface{}
			err = json.NewDecoder(bytes.NewReader(originalBytes)).Decode(&iOriginalJSON)
			if err != nil {
				t.Fatal(err)
			}

			doc, err := Parse(bytes.NewReader(originalBytes))
			if err != nil {
				t.Fatal(err)
			}

			iDocJSON, err := doc.JSON(false)
			if err != nil {
				t.Fatal(err)
			}

			originalJSONBytes, err := json.Marshal(iOriginalJSON)
			if err != nil {
				t.Fatal(err)
			}
			docJSONBytes, err := json.Marshal(iDocJSON)
			if err != nil {
				t.Fatal(err)
			}
			if string(originalJSONBytes) != string(docJSONBytes) {
				t.Fatalf(
					"JSON from doc is different from original JSON \nOriginal: %s \nDoc:      %s",
					string(originalJSONBytes),
					string(docJSONBytes),
				)
			}
		})
	}
}

func TestFindAssetIDs(t *testing.T) {
	originalBytes, err := ioutil.ReadFile(path.Join("testdata", "screen_v3_01.json"))
	if err != nil {
		t.Fatal(err)
	}

	t.Run(`"*/asset_id"`, func(t *testing.T) {
		t.Run("Only one item in array", func(t *testing.T) {
			doc, err := Parse(bytes.NewReader(originalBytes))
			if err != nil {
				t.Fatal(err)
			}

			allNodes := Find(doc, "*/asset_id")
			nodes := unique(allNodes)

			if len(nodes) != 1 {
				t.Fatalf("Expected nodes to have only 1 iteam but got %v", len(nodes))
			}

			if nodes[0].Data != "asset_id" {
				t.Fatalf(`Expected Data to be "asset_id" but got %v`, nodes[0].Data)
			}

			if nodes[0].InnerData().(float64) != 0 {
				t.Fatalf("Expected InnerData() to be 0 but got %v", nodes[0].InnerData())
			}
		})

		t.Run("More than one item in array", func(t *testing.T) {
			doc, err := Parse(strings.NewReader(`[{"id":1,"asset_id":1,"layers":[{"asset_id":11}]},{"id":2,"asset_id":2}]`))
			if err != nil {
				t.Fatal(err)
			}

			allNodes := Find(doc, "*/asset_id")
			nodes := unique(allNodes)

			strAssetIDs := []string{"1", "2"}
			if len(nodes) != len(strAssetIDs) {
				t.Fatalf("Expected nodes to have %v items but got %v", len(strAssetIDs), len(nodes))
			}

			var assetIDs []string
			for _, node := range nodes {
				assetIDs = append(assetIDs, fmt.Sprintf("%v", node.InnerData()))
			}

			sort.Strings(strAssetIDs)
			sort.Strings(assetIDs)
			for i := range strAssetIDs {
				if strAssetIDs[i] != assetIDs[i] {
					t.Fatalf("Expect %v to equal %v", strAssetIDs[i], assetIDs[i])
				}
			}
		})
	})

	t.Run("//layers//exportOptions//asset_id", func(t *testing.T) {
		doc, err := Parse(bytes.NewReader(originalBytes))
		if err != nil {
			t.Fatal(err)
		}

		strAssetIDs := []string{"4632", "4629", "4627", "4631", "4630"}
		allNodes := Find(doc, "//layers//exportOptions//asset_id")
		nodes := unique(allNodes)

		var assetIDs []string
		if n := len(nodes); n != len(strAssetIDs) {
			t.Fatalf("Expected %v nodes but got %v", n, len(strAssetIDs))
		}
		for _, n := range nodes {
			if n.Data != "asset_id" {
				t.Fatalf("Expected asset_id but got %s", n.Data)
			}

			assetIDs = append(assetIDs, n.InnerText())
		}

		sort.Strings(strAssetIDs)
		sort.Strings(assetIDs)
		for i := range strAssetIDs {
			if strAssetIDs[i] != assetIDs[i] {
				t.Fatalf("Expected %+v to equal %+v", strAssetIDs[i], assetIDs[i])
			}
		}
	})
}

func TestSetInnerDataAndInnerData(t *testing.T) {
	b, err := ioutil.ReadFile(path.Join("testdata", "records.json"))
	if err != nil {
		t.Fatal(err)
	}

	doc, err := Parse(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}

	strIDs := []string{"100", "200", "300"}

	nodes := Find(doc, "//userID")
	if len(strIDs) != len(nodes) {
		t.Fatalf("Expected nodes to have %d items, but got only %d", len(strIDs), len(nodes))
	}

	for i, node := range nodes {
		strID := strIDs[i]
		node.SetInnerData(strIDs[i])

		if strID != node.InnerData() {
			t.Fatalf("Expected %s but got %s", strID, node.InnerData())
		}
	}

	idoc, err := doc.JSON(true)
	if err != nil {
		t.Fatal(err)
	}

	records := idoc.([]interface{})
	if len(records) != 3 {
		t.Fatalf("Expected records to have 3 items, but got %v", len(records))
	}

	for i, irecord := range records {
		record := irecord.(map[string]interface{})

		if strIDs[i] != fmt.Sprintf("%v", record["userID"]) {
			t.Fatalf("Expected %v to equal %v", strIDs[i], fmt.Sprintf("%v", record["userID"]))
		}
	}
}

func TestSetSkippedAndSkipped(t *testing.T) {
	b, err := ioutil.ReadFile(path.Join("testdata", "records.json"))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("record", func(t *testing.T) {
		doc, err := Parse(bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}

		if len(doc.ChildNodes()) != 3 {
			t.Fatalf("Expected doc to have 3 children but got only %d", len(doc.ChildNodes()))
		}

		doc.ChildNodes()[0].SetSkipped(true)
		if !doc.ChildNodes()[0].Skipped() {
			t.Fatalf("Expect record skipped to be true")
		}

		idoc, err := doc.JSON(true)
		if err != nil {
			t.Fatal(err)
		}

		irecords := idoc.([]interface{})
		if len(irecords) != 2 {
			t.Fatalf("Expected records to have 2 items, but got only %d", len(irecords))
		}
		for _, irecord := range irecords {
			record := irecord.(map[string]interface{})

			if "1" == fmt.Sprintf("%v", record["id"]) {
				t.Fatalf("record with id 1 should be skipped")
			}
		}
	})

	t.Run("key/value", func(t *testing.T) {
		doc, err := Parse(bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}

		nodes := Find(doc, "*/userID")
		if len(nodes) != 3 {
			t.Fatalf("Expected nodes to have 3 items, but got only %d", len(nodes))
		}

		nodes[0].SetSkipped(true)
		i, err := doc.JSON(true)
		if err != nil {
			t.Fatal(err)
		}

		b, err := json.Marshal(i)
		if err != nil {
			t.Fatal(err)
		}

		var records []struct {
			ID     *float64 `json:"id,omitempty"`
			UserID *float64 `json:"userID,omitempty"`
			RoleID *float64 `json:"roleID,omitempty"`
		}
		err = json.Unmarshal(b, &records)
		if err != nil {
			t.Fatal(err)
		}

		if len(records) != 3 {
			t.Fatalf("Expected records to have 3 items, but got only %d", len(records))
		}
		if *records[0].ID != 1 {
			t.Fatalf("Expected id to be 1, but got %v", *records[0].ID)
		}
		if *records[0].RoleID != 3 {
			t.Fatalf("Expected roleID to be 3, but got %v", *records[0].RoleID)
		}
		if records[0].UserID != nil {
			t.Fatalf("Expected userID to be nil, but got %v", *records[0].UserID)
		}
	})
}

func unique(allNodes []*Node) []*Node {
	var tmpMap = make(map[*Node]bool)
	for _, node := range allNodes {
		tmpMap[node] = true
	}

	var nodes []*Node
	for node := range tmpMap {
		nodes = append(nodes, node)
	}

	return nodes
}
