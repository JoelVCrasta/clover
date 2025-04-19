package parsing

import (
	"fmt"
	"strconv"
)

// Decode function takes a byte slice and returns the decoded value
func (t Torrent) Decode(buf []byte) (any, error) {
	result, pos, err := parseValue(buf, 0)
	if err != nil {
		return nil, err
	}
	if pos != len(buf) {
		return nil, fmt.Errorf("extra data at pos %d", pos)
	}

	return result, nil
}

func parseValue(buf []byte, pos int) (any, int, error) {
	if len(buf) == 0 || buf == nil {
		return nil, pos, fmt.Errorf("empty buffer")
	}

	if pos >= len(buf) {
		return nil, pos, fmt.Errorf("out of bounds at pos %d", pos)
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
		return nil, pos, fmt.Errorf("invalid token %c at pos %d", buf[pos], pos)
	}
}

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

func parseString(buf []byte, pos int) ([]byte, int, error) {
	start := pos
	length := len(buf)

	for pos < length && buf[pos] >= '0' && buf[pos] <= '9' {
		pos++
	}

	if pos >= length || buf[pos] != ':' {
		return nil, pos, fmt.Errorf("invalid length terminate token at pos %d", pos)
	}

	offset, err := strconv.Atoi(string(buf[start:pos]))
	if err != nil {
		return nil, pos, err
	}

	pos++
	if (pos + offset) > length {
		return nil, pos, fmt.Errorf("out of bounds for offset at pos %d", pos+offset)
	}

	parsedString := buf[pos : pos+offset]

	return parsedString, pos + offset, nil
}

func parseList(buf []byte, pos int) ([]any, int, error) {
	pos++
	length := len(buf)
	arr := []any{}

	for pos < length && buf[pos] != 'e' {
		var data any
		var err error

		data, pos, err = parseValue(buf, pos)
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

func parseDict(buf []byte, pos int) (any, int, error) {
	pos++
	length := len(buf)
	dict := make(map[string]any)

	for pos < length && buf[pos] != 'e' {
		var key []byte
		var value any
		var err error

		key, pos, err = parseString(buf, pos)
		if err != nil {
			return nil, pos, err
		}
		stringKey := string(key)

		if _, exists := dict[stringKey]; exists {
			return nil, pos, fmt.Errorf("duplicate key '%s' at pos %d", key, pos)
		}

		value, pos, err = parseValue(buf, pos)
		if err != nil {
			return nil, pos, err
		}

		dict[stringKey] = value
	}

	if pos >= length || buf[pos] != 'e' {
		return nil, pos, fmt.Errorf("invalid dict terminate token at pos %d", pos)
	}

	return dict, pos + 1, nil
}
