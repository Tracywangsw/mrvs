package visitno

import (
	cryptorand "crypto/rand"
	"fmt"
	"io"
	"math/big"
	"time"
)

// Generate 生成形如 V20261105-482913 的业务单号。
// r 为 nil 时使用加密安全随机源；传入自定义 reader 便于测试。
func Generate(now time.Time, r io.Reader) (string, error) {
	if r == nil {
		r = cryptorand.Reader
	}
	n, err := randInt(r, 1000000)
	if err != nil {
		return "", fmt.Errorf("生成拜访单号失败: %w", err)
	}
	return fmt.Sprintf("V%s-%06d", now.UTC().Format("20060102"), n), nil
}

// Valid 判断字符串是否符合拜访单号格式。
func Valid(s string) bool {
	if len(s) != len("V20060101-000000") {
		return false
	}
	if s[0] != 'V' || s[9] != '-' {
		return false
	}
	for i := 1; i < 9; i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	for i := 10; i < 16; i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func randInt(r io.Reader, max int64) (int64, error) {
	n, err := cryptorand.Int(r, big.NewInt(max))
	if err != nil {
		return 0, err
	}
	return n.Int64(), nil
}
