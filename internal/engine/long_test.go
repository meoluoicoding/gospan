package engine

import (
	"strings"
	"testing"

	"github.com/snzpet/gospan/internal/config"
)

// TestLongMultiWordTrigger verifies that a long, multi-word Vietnamese phrase
// works as a trigger even when preceded by other typed text.
func TestLongMultiWordTrigger(t *testing.T) {
	trigger := "kính gửi anh chị em đồng nghiệp thân mến"
	e := newTestEngine(t, Config{}, rule(trigger, "Xin chào"))

	res, count := e.FeedString("nội dung " + trigger)
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if res.Match.Trigger != trigger {
		t.Errorf("trigger = %q, want %q", res.Match.Trigger, trigger)
	}
}

// TestLongReplacementPreserved verifies a multi-kilobyte replacement survives
// expansion byte-for-byte.
func TestLongReplacementPreserved(t *testing.T) {
	long := strings.Repeat("Đây là một câu tiếng Việt dài để kiểm tra. ", 300)
	e := newTestEngine(t, Config{}, rule(";long", long))

	res, count := e.FeedString(";long")
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if res.Match.Replacement != long {
		t.Fatalf("replacement altered: got %d runes, want %d", len([]rune(res.Match.Replacement)), len([]rune(long)))
	}
	if len([]rune(res.Match.Replacement)) < 10000 {
		t.Fatalf("expected a long replacement, got %d runes", len([]rune(res.Match.Replacement)))
	}
}

// TestMultilineReplacement verifies block-scalar replacements keep their newlines.
func TestMultilineReplacement(t *testing.T) {
	ml := "Trân trọng,\nKhoa\nSĐT: 0123456789\n"
	e := newTestEngine(t, Config{}, rule(";sig", ml))

	res, count := e.FeedString(";sig")
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if res.Match.Replacement != ml {
		t.Errorf("multiline replacement = %q, want %q", res.Match.Replacement, ml)
	}
	if strings.Count(res.Match.Replacement, "\n") != 3 {
		t.Errorf("expected 3 newlines, got %d", strings.Count(res.Match.Replacement, "\n"))
	}
}

// TestLongWordWithoutTrigger does not expand and does not corrupt the buffer.
func TestLongWordWithoutTrigger(t *testing.T) {
	e := newTestEngine(t, Config{}, rule(";dc", "được"))
	word := strings.Repeat("a", 200)
	if _, count := e.FeedString(word); count != 0 {
		t.Fatal("a long word with no trigger must not expand")
	}
	// The engine must still work afterwards.
	if _, count := e.FeedString(";dc"); count != 1 {
		t.Fatal("engine should still expand after processing a long word")
	}
}

// TestTriggerLongerThanBuffer documents the buffer-size interaction: a trigger
// that cannot fit in the buffer can never match.
func TestTriggerLongerThanBuffer(t *testing.T) {
	trigger := strings.Repeat("a", 300)

	tooSmall := newTestEngine(t, Config{BufferSize: 64}, rule(trigger, "x"))
	if _, count := tooSmall.FeedString(trigger); count != 0 {
		t.Fatal("trigger longer than the buffer must not match")
	}

	bigEnough := newTestEngine(t, Config{BufferSize: 512}, rule(trigger, "x"))
	if _, count := bigEnough.FeedString(trigger); count != 1 {
		t.Fatal("trigger that fits in the buffer should match")
	}
}

// TestManyLongTriggers exercises a large dictionary with long triggers.
func TestManyLongTriggers(t *testing.T) {
	matches := make([]config.MatchConfig, 0, 2000)
	for i := 0; i < 2000; i++ {
		matches = append(matches, rule(
			"chèn câu số "+itoa(i)+" vào văn bản",
			"nội dung thay thế cho câu số "+itoa(i),
		))
	}
	e := newTestEngine(t, Config{BufferSize: 256}, matches...)

	res, count := e.FeedString("mở đầu chèn câu số 1234 vào văn bản")
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if res.Match.Replacement != "nội dung thay thế cho câu số 1234" {
		t.Errorf("replacement = %q", res.Match.Replacement)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
