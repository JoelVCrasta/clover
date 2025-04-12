package main

import (
	"fmt"
	"strconv"
  "encoding/json"
)

func main() {
  bencode_string := "d8:announce35:udp://tracker.openbittorrent.com:8013:creation datei1327049827e4:infod6:lengthi20e4:name10:sample.txt12:piece lengthi65536e6:pieces70:<hex>5C C5 E6 52 BE 0D E6 F2 78 05 B3 04 64 FF 9B 00 F4 89 F0 C9</hex>7:privatei1eee"

	bencode_byte := []byte(bencode_string)

	fmt.Println("String: ", bencode_string)
	fmt.Println("Byte :", bencode_byte)

	result, newPos, err := recursiveDescent(bencode_byte, 0)
	if err != nil {
		fmt.Println("Parse error: ", err)
	}

  jsonResult, _ := json.MarshalIndent(result, "", "  ")
  fmt.Println("Parsed result: ", string(jsonResult))
	fmt.Println("New position: ", newPos)
}

// recursiveDescent
func recursiveDescent(buf []byte, pos int) (any, int, error) {
	if len(buf) == 0 || buf == nil {
		return nil, pos, fmt.Errorf("empty buffer")
	}

	if pos >= len(buf) {
		return nil, pos, fmt.Errorf("out of bounds")
	}

	switch buf[pos] {
	case 'i':
		return parseInt(buf, pos)

  case 'l':
    return parseList(buf, pos)

  case 'd':
    return parseDict(buf, pos)

	default:
		if buf[pos] >= '0' && buf[pos] <= '9' {
			return parseString(buf, pos)
		}
		return nil, pos, fmt.Errorf("invalid token at pos %d: %c", pos, buf[pos])
	}
}

// parseInt 
func parseInt(buf []byte, pos int) (int, int, error) {
	pos++
	start := pos
	length := len(buf)

	if pos >= length || buf[pos] == 'e' {
		return 0, pos, fmt.Errorf("empty integer at pos %d", pos)
	}

	if buf[pos] == '-' {
		pos++
		if pos >= length || (buf[pos] == 'e' || buf[pos] == '0') {
			return 0, pos, fmt.Errorf("invalid negative token at pos %d", pos)
		}
	}

	if buf[pos] == '0' && (pos+1 < length && buf[pos+1] != 'e') {
		return 0, pos, fmt.Errorf("invalid leading zero token at pos %d", pos)
	}

	for pos < length && buf[pos] >= '0' && buf[pos] <= '9' {
		pos++
	}

	if pos >= length || buf[pos] != 'e' {
		return 0, pos, fmt.Errorf("invalid terminate token at pos %d", pos)
	}

	parsedInt, err := strconv.Atoi(string(buf[start:pos]))
	if err != nil {
		return 0, pos, err
	}

	return parsedInt, pos + 1, nil
}

// parseString
func parseString(buf []byte, pos int) (string, int, error) {
	start := pos
	length := len(buf)

	for pos < length && buf[pos] >= '0' && buf[pos] <= '9' {
		pos++
	}

	if pos >= length || buf[pos] != ':' {
		return "", pos, fmt.Errorf("invalid length terminate token at pos %d", pos)
	}

	offset, err := strconv.Atoi(string(buf[start:pos]))
  if err != nil {
    return "", pos, err
  }

  pos++
  if (pos + offset) > length {
    return "", pos, fmt.Errorf("out of bounds for offset at pos %d", pos+offset)
  }

  parsedString := string(buf[pos:pos+offset])

	return parsedString, pos + offset, nil
}

// parseList
func parseList(buf []byte, pos int) ([]any, int, error) {
  pos++
  length := len(buf)
  arr := []any{}

  for pos < length && buf[pos] != 'e' {
    var data any
    var err error

    data, pos, err = recursiveDescent(buf, pos)
    if err != nil {
      return nil, pos, err
    }

    arr = append(arr, data)
  }

  if pos >= length || buf[pos] != 'e' {
		return nil, pos, fmt.Errorf("invalid list terminate token at pos %d", pos)
	}

  return arr, pos + 1, nil
}

// parseDict
func parseDict(buf []byte, pos int) (any, int, error) {
  pos++
  length := len(buf)
  dict := make(map[string]any)

  for pos < length && buf[pos] != 'e' {
    var key string
    var value any
    var err error

    key, pos, err = parseString(buf, pos)
    if err != nil {
      return nil, pos, err
    }

    if _, exists := dict[key]; exists {
      return nil, pos, fmt.Errorf("duplicate key '%s' at pos %d", key, pos)
    }

    value, pos, err = recursiveDescent(buf, pos)
    if err != nil {
      return nil, pos, err
    }

    dict[key] = value
  }

  if pos >= length || buf[pos] != 'e' {
    return nil, pos, fmt.Errorf("invalid dict terminate token at pos %d", pos)
  }
  
  return dict, pos + 1, nil 
}
