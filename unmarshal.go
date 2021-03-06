package colly

import (
	"errors"
	"reflect"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Unmarshal is a shorthand for colly.UnmarshalHTML
func (h *HTMLElement) Unmarshal(v interface{}) error {
	return UnmarshalHTML(v, h.DOM)
}

// UnmarshalHTML declaratively extracts text or attributes to a struct from
// HTML response using struct tags composed of css selectors.
// Allowed struct tags:
//  - "selector" (required): CSS (goquery) selector of the desired data
//  - "attr" (optional): Selects the matching element's attribute's value.
//     Leave it blank or omit to get the text of the element.
//
// Example struct declaration:
//
//   type Nested struct {
//   	String  string   `selector:"div > p"`
//      Classes []string `selector:"li" attr:"class"`
//   	Struct  *Nested  `selector:"div > div"`
//   }
//
// Supported types: struct, *struct, string, []string
func UnmarshalHTML(v interface{}, s *goquery.Selection) error {
	rv := reflect.ValueOf(v)

	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("Invalid type or nil-pointer")
	}

	sv := rv.Elem()
	st := reflect.TypeOf(v).Elem()

	for i := 0; i < sv.NumField(); i++ {
		attrV := sv.Field(i)
		if !attrV.CanAddr() || !attrV.CanSet() {
			continue
		}
		if err := unmarshalAttr(s, attrV, st.Field(i)); err != nil {
			return err
		}
	}
	return nil
}

func unmarshalAttr(s *goquery.Selection, attrV reflect.Value, attrT reflect.StructField) error {
	selector := attrT.Tag.Get("selector")
	htmlAttr := attrT.Tag.Get("attr")
	// TODO support more types
	switch attrV.Kind() {
	case reflect.Slice:
		if err := unmarshalSlice(s, selector, htmlAttr, attrV); err != nil {
			return err
		}
	case reflect.String:
		val := getDOMValue(s.Find(selector), htmlAttr)
		attrV.Set(reflect.Indirect(reflect.ValueOf(val)))
	case reflect.Struct:
		if err := unmarshalStruct(s, selector, attrV); err != nil {
			return err
		}
	case reflect.Ptr:
		if err := unmarshalPtr(s, selector, attrV); err != nil {
			return err
		}
	default:
		return errors.New("Invalid type: " + attrV.String())
	}
	return nil
}

func unmarshalStruct(s *goquery.Selection, selector string, attrV reflect.Value) error {
	newS := s
	if selector != "" {
		newS = newS.Find(selector)
	}
	if newS.Nodes == nil {
		return nil
	}
	v := reflect.New(attrV.Type())
	err := UnmarshalHTML(v.Interface(), newS)
	if err != nil {
		return err
	}
	attrV.Set(reflect.Indirect(v))
	return nil
}

func unmarshalPtr(s *goquery.Selection, selector string, attrV reflect.Value) error {
	newS := s
	if selector != "" {
		newS = newS.Find(selector)
	}
	if newS.Nodes == nil {
		return nil
	}
	e := attrV.Type().Elem()
	if e.Kind() != reflect.Struct {
		return errors.New("Invalid slice type")
	}
	v := reflect.New(e)
	err := UnmarshalHTML(v.Interface(), newS)
	if err != nil {
		return err
	}
	attrV.Set(v)
	return nil
}

func unmarshalSlice(s *goquery.Selection, selector, htmlAttr string, attrV reflect.Value) error {
	if attrV.Pointer() == 0 {
		v := reflect.MakeSlice(attrV.Type(), 0, 0)
		attrV.Set(v)
	}
	switch attrV.Type().Elem().Kind() {
	case reflect.String:
		s.Find(selector).Each(func(_ int, s *goquery.Selection) {
			val := getDOMValue(s, htmlAttr)
			attrV.Set(reflect.Append(attrV, reflect.Indirect(reflect.ValueOf(val))))
		})
	default:
		return errors.New("Invalid slice type")
	}
	return nil
}

func getDOMValue(s *goquery.Selection, attr string) string {
	if attr == "" {
		return strings.TrimSpace(s.First().Text())
	}
	attrV, _ := s.Attr(attr)
	return attrV
}
