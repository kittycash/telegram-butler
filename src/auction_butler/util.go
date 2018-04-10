package auction_butler

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	ErrNoBidFound       = errors.New("no bid found in message")
	ErrUnableToParseBid = errors.New("unable to parse bid")
)

func niceDuration(d time.Duration) string {
	if d < 0 {
		return d.String()
	}

	var hours, minutes, seconds int
	seconds = int(d.Seconds())
	hours, seconds = seconds/3600, seconds%3600
	minutes, seconds = seconds/60, seconds%60

	if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%dh%dm", hours, minutes)
		} else {
			return fmt.Sprintf("%dh", hours)
		}
	} else {
		if minutes > 0 {
			if seconds > 0 {
				return fmt.Sprintf("%dm%ds", minutes, seconds)
			} else {
				return fmt.Sprintf("%dm", minutes)
			}
		} else {
			return fmt.Sprintf("%ds", seconds)
		}
	}
}

func appendField(fields []string, name, format string, args ...interface{}) []string {
	value := fmt.Sprintf(format, args...)
	return append(fields, fmt.Sprintf("*%s*: %s", strings.Title(name), value))
}

func parseDuration(args string) (time.Duration, error) {
	hours, err := strconv.ParseFloat(args, 64)
	if err == nil {
		return time.Second * time.Duration(hours*3600), nil
	}

	return time.ParseDuration(args)
}

func SplitToString(a []int, sep string) string {
	if len(a) == 0 {
		return ""
	}

	b := make([]string, len(a))
	for i, v := range a {
		b[i] = strconv.Itoa(v)
	}
	return strings.Join(b, sep)
}

func findBid(bidStr string) (*Bid, error) {
	r := regexp.MustCompile(`(\d+((?:\.|,)\d*)?|(?:\.|,)\d+)\s*(BTC|SKY)?`)
	matches := r.FindStringSubmatch(bidStr)
	if len(matches) == 0 {
		return nil, ErrNoBidFound
	}

	bid := parseBid(matches[1], matches[3])
	if bid == nil {
		return nil, ErrUnableToParseBid
	}

	return bid, nil
}

func parseBid(bid string, foundBidType string) *Bid {
	var bidValue float64
	var bidType string

	bid = strings.Replace(bid, ",", ".", 1)
	bidValue, err := strconv.ParseFloat(bid, 64)
	if err != nil {
		return nil
	}

	if foundBidType != "" {
		bidType = foundBidType
	} else {
		if bidValue > 5 {
			bidType = "SKY"
		} else {
			bidType = "BTC"
		}
	}

	if bidType == "BTC" {
		bidValue = toFixed(bidValue, 2)
	} else {
		bidValue = toFixed(bidValue, 0)
	}

	return &Bid{
		Value:    bidValue,
		CoinType: bidType,
	}
}

func niceTime(time time.Time) string {
	//18:00 UTC 24.03
	return fmt.Sprintf("%v:%v %v %v.%v", time.Hour(), time.Minute(), time.Location().String(), time.Day(), time.Month())

}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num * output)) / output
}

