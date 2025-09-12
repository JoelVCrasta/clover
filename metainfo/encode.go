package metainfo

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
)

// Encode function takes a value and returns its bencoded representation as a byte slice
func (t Torrent) Encode(value any) ([]byte, error) {
	var buf bytes.Buffer

	err := encodeValue(&buf, value)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func encodeValue(buf *bytes.Buffer, value any) error {
	switch v := value.(type) {
	case int:
		return encodeInt(buf, v)
	case string:
		return encodeString(buf, []byte(v))
	case []byte:
		return encodeString(buf, v)
	case []any:
		return encodeList(buf, v)
	case map[string]any:
		return encodeDict(buf, v)
	default:
		return fmt.Errorf("unsupported type: %T", value)
	}
}

func encodeInt(buf *bytes.Buffer, value int) error {
	buf.WriteByte('i')
	buf.WriteString(strconv.Itoa(value))
	buf.WriteByte('e')
	return nil
}

func encodeString(buf *bytes.Buffer, value []byte) error {
	buf.WriteString(strconv.Itoa(len(value)))
	buf.WriteByte(':')
	buf.Write(value)
	return nil
}

func encodeList(buf *bytes.Buffer, list []any) error {
	buf.WriteByte('l')
	for _, item := range list {
		if err := encodeValue(buf, item); err != nil {
			return err
		}
	}
	buf.WriteByte('e')
	return nil
}

func encodeDict(buf *bytes.Buffer, dict map[string]any) error {
	buf.WriteByte('d')

	keys := make([]string, 0, len(dict))
	for k := range dict {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := dict[k]
		if err := encodeString(buf, []byte(k)); err != nil {
			return err
		}
		if err := encodeValue(buf, v); err != nil {
			return err
		}
	}

	buf.WriteByte('e')
	return nil
}
